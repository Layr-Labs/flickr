package push

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
	"github.com/yourorg/flickr/internal/config"
	"github.com/yourorg/flickr/internal/eth"
	"github.com/yourorg/flickr/internal/middleware"
	"github.com/yourorg/flickr/internal/ref"
	"github.com/yourorg/flickr/internal/signer"
	"go.uber.org/zap"
)

// Command returns the push command
func Command() *cli.Command {
	return &cli.Command{
		Name:  "push",
		Usage: "Push a Docker image to registry and create on-chain release",
		Description: `Pushes a Docker image to a registry and creates an on-chain release 
in the ReleaseManager contract for the configured AVS and operator set.`,
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:     "image",
				Usage:    "Docker image(s) to push (e.g., myregistry.io/myimage:tag)",
				Required: true,
			},
			&cli.Uint64Flag{
				Name:  "upgrade-by-time",
				Usage: "Unix timestamp for upgrade deadline (defaults to 30 days from now)",
			},
			&cli.StringFlag{
				Name:  "registry",
				Usage: "Override registry URL (uses image registry by default)",
			},
			&cli.BoolFlag{
				Name:  "skip-docker-push",
				Usage: "Skip Docker push (assumes image already in registry)",
			},
			&cli.StringFlag{
				Name:  "avs",
				Usage: "AVS contract address (uses context if not provided)",
			},
			&cli.Uint64Flag{
				Name:  "operator-set",
				Usage: "Operator set ID (uses context if not provided)",
			},
			&cli.StringFlag{
				Name:  "release-manager",
				Usage: "ReleaseManager contract address (uses chain default if not provided)",
			},
			&cli.StringFlag{
				Name:  "rpc-url",
				Usage: "Ethereum RPC URL (uses context if not provided)",
			},
			&cli.Uint64Flag{
				Name:  "gas-limit",
				Usage: "Gas limit for transaction",
				Value: 500000,
			},
		},
		Action: pushAction,
	}
}

func pushAction(c *cli.Context) error {
	log := middleware.GetLogger(c)

	// Get context
	currentCtx, err := middleware.GetCurrentContext(c)
	if err != nil {
		currentCtx = &config.Context{}
	}

	// Get AVS address (from flag or context)
	avsAddress := c.String("avs")
	if avsAddress == "" {
		avsAddress = currentCtx.AVSAddress
	}
	if avsAddress == "" {
		return fmt.Errorf("--avs is required (or set in context with 'flickr context set --avs-address')")
	}

	// Get operator set ID (from flag or context)
	operatorSetID := uint32(c.Uint64("operator-set"))
	if operatorSetID == 0 && c.IsSet("operator-set") {
		// Explicitly set to 0
	} else if operatorSetID == 0 {
		operatorSetID = currentCtx.OperatorSetID
	}

	// Get RPC URL (from flag or context)
	rpcURL := c.String("rpc-url")
	if rpcURL == "" {
		rpcURL = currentCtx.RPCURL
	}
	if rpcURL == "" {
		return fmt.Errorf("--rpc-url is required (or set in context with 'flickr context set --rpc-url')")
	}

	// Get release manager address (from flag, context, or chain default)
	releaseManager := c.String("release-manager")
	if releaseManager == "" {
		releaseManager = currentCtx.ReleaseManager
	}

	// Get the actual address (may use chain defaults)
	rmAddr, err := eth.GetReleaseManagerAddress(rpcURL, releaseManager)
	if err != nil {
		return fmt.Errorf("failed to get ReleaseManager address: %w", err)
	}

	// Get signer from context
	sig, err := signer.FromContext(currentCtx)
	if err != nil {
		return fmt.Errorf("no signer configured: %w", err)
	}

	log.Info("Using configuration",
		zap.String("avs", avsAddress),
		zap.Uint32("operatorSet", operatorSetID),
		zap.String("releaseManager", rmAddr.Hex()),
		zap.String("rpcURL", rpcURL),
		zap.String("signer", sig.Address().Hex()))

	// Parse addresses
	avs := common.HexToAddress(avsAddress)

	// Get images to push
	images := c.StringSlice("image")
	if len(images) == 0 {
		return fmt.Errorf("at least one --image is required")
	}

	// Process artifacts
	artifacts := make([]eth.Artifact, 0, len(images))
	
	for _, image := range images {
		log.Info("Processing image", zap.String("image", image))

		// Push Docker image unless skipped
		if !c.Bool("skip-docker-push") {
			log.Info("Pushing Docker image", zap.String("image", image))
			cmd := exec.Command("docker", "push", image)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to push image %s: %v\n%s", image, err, string(output))
			}
			log.Info("Docker push successful", zap.String("image", image))
		}

		// Get digest from the image
		digest, registry, err := getImageDigest(image)
		if err != nil {
			return fmt.Errorf("failed to get digest for %s: %w", image, err)
		}

		// Override registry if specified
		if c.String("registry") != "" {
			registry = c.String("registry")
		}

		// Convert digest to [32]byte
		var digest32 [32]byte
		copy(digest32[:], digest)

		artifacts = append(artifacts, eth.Artifact{
			Registry: registry,
			Digest32: digest32,
		})

		log.Info("Prepared artifact",
			zap.String("registry", registry),
			zap.String("digest", ref.Digest32ToSha256String(digest32)))
	}

	// Get upgrade-by-time (default to 30 days from now)
	upgradeByTime := uint32(c.Uint64("upgrade-by-time"))
	if upgradeByTime == 0 {
		upgradeByTime = uint32(time.Now().Add(30 * 24 * time.Hour).Unix())
	}

	// Create Ethereum client with signer
	rmClient, err := eth.NewClientWithSigner(rpcURL, rmAddr, sig)
	if err != nil {
		return fmt.Errorf("failed to create Ethereum client: %w", err)
	}
	defer rmClient.Close()

	// Check if metadata URI is set
	ctx := context.Background()
	metadataURI, err := rmClient.GetMetadataURI(ctx, avs, operatorSetID)
	if err != nil {
		return fmt.Errorf("failed to check metadata URI: %w", err)
	}

	if metadataURI == "" {
		return fmt.Errorf(`no metadata URI set for this operator set

Please set a metadata URI first with:
  flickr metadata set --uri "https://your-metadata-uri.json"

Current configuration:
  AVS: %s
  Operator Set: %d`, avs.Hex(), operatorSetID)
	}

	log.Info("Metadata URI verified", zap.String("uri", metadataURI))

	// Push release on-chain
	log.Info("Pushing release on-chain",
		zap.Int("artifactCount", len(artifacts)),
		zap.Uint32("upgradeByTime", upgradeByTime))
	tx, err := rmClient.PushRelease(ctx, avs, operatorSetID, artifacts, upgradeByTime, c.Uint64("gas-limit"))
	if err != nil {
		return fmt.Errorf("failed to push release: %w", err)
	}

	log.Info("Transaction submitted",
		zap.String("txHash", tx.Hash().Hex()),
		zap.String("from", sig.Address().Hex()),
		zap.String("to", rmAddr.Hex()))

	fmt.Printf("Release pushed successfully!\n")
	fmt.Printf("Transaction: %s\n", tx.Hash().Hex())
	fmt.Printf("AVS: %s\n", avs.Hex())
	fmt.Printf("Operator Set: %d\n", operatorSetID)
	fmt.Printf("Artifacts: %d\n", len(artifacts))

	return nil
}

// getImageDigest gets the digest and registry from a Docker image
func getImageDigest(image string) ([]byte, string, error) {
	// Get the digest using docker inspect
	cmd := exec.Command("docker", "inspect", "--format", "{{.RepoDigests}}", image)
	output, err := cmd.Output()
	if err != nil {
		return nil, "", fmt.Errorf("failed to inspect image: %w", err)
	}

	// Parse the output to get the digest
	// Format is like: [registry.io/image@sha256:abcd1234...]
	digestStr := strings.TrimSpace(string(output))
	digestStr = strings.Trim(digestStr, "[]")
	
	if digestStr == "" {
		return nil, "", fmt.Errorf("no digest found for image (may need to pull first)")
	}

	// Handle multiple digests (space-separated)
	digests := strings.Fields(digestStr)
	if len(digests) == 0 {
		return nil, "", fmt.Errorf("no digest found for image")
	}

	// Use the first digest
	firstDigest := digests[0]

	// Split by @ to separate registry/image from digest
	parts := strings.Split(firstDigest, "@")
	if len(parts) != 2 {
		return nil, "", fmt.Errorf("unexpected digest format: %s", firstDigest)
	}

	registryImage := parts[0]
	digestPart := parts[1]

	// Extract registry from the full image name
	registryParts := strings.Split(registryImage, "/")
	registry := ""
	if len(registryParts) > 1 && (strings.Contains(registryParts[0], ".") || strings.Contains(registryParts[0], ":")) {
		registry = registryParts[0]
	}

	// Parse the sha256:... part
	if !strings.HasPrefix(digestPart, "sha256:") {
		return nil, "", fmt.Errorf("unexpected digest format: %s", digestPart)
	}

	hexDigest := strings.TrimPrefix(digestPart, "sha256:")
	
	// Convert hex string to bytes
	digest := make([]byte, 32)
	for i := 0; i < 32; i++ {
		fmt.Sscanf(hexDigest[i*2:i*2+2], "%02x", &digest[i])
	}

	return digest, registry, nil
}