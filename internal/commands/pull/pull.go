package pull

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
	"github.com/yourorg/flickr/internal/config"
	"github.com/yourorg/flickr/internal/eth"
	"github.com/yourorg/flickr/internal/middleware"
	"github.com/yourorg/flickr/internal/ref"
	"go.uber.org/zap"
)

// Command returns the pull command
func Command() *cli.Command {
	return &cli.Command{
		Name:  "pull",
		Usage: "Pull Docker images for a release from the chain",
		Description: `Fetches release information from the on-chain ReleaseManager and pulls 
all Docker images associated with the release. Can pull the latest release or a specific 
release ID.`,
		Flags: []cli.Flag{
			&cli.Uint64Flag{
				Name:  "release-id",
				Usage: "Specific release ID to pull (defaults to latest)",
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
			&cli.BoolFlag{
				Name:  "all",
				Usage: "Pull all artifacts (default pulls only the first)",
			},
		},
		Action: pullAction,
	}
}

func pullAction(c *cli.Context) error {
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

	log.Info("Using configuration",
		zap.String("avs", avsAddress),
		zap.Uint32("operatorSet", operatorSetID),
		zap.String("releaseManager", rmAddr.Hex()),
		zap.String("rpcURL", rpcURL))

	// Parse addresses
	avs := common.HexToAddress(avsAddress)

	// Create Ethereum client
	rmClient, err := eth.NewClient(rpcURL, rmAddr)
	if err != nil {
		return fmt.Errorf("failed to create Ethereum client: %w", err)
	}
	defer rmClient.Close()

	// Fetch release
	ctx := context.Background()
	
	// First check if there are any releases
	total, err := rmClient.GetTotalReleases(ctx, avs, operatorSetID)
	if err != nil {
		return fmt.Errorf("failed to check releases: %w", err)
	}
	
	if total.Int64() == 0 {
		return fmt.Errorf(`no releases found for this operator set

To push a release, run:
  flickr push --image <your-image>

Current configuration:
  AVS: %s
  Operator Set: %d`, avs.Hex(), operatorSetID)
	}

	var (
		release   eth.Release
		releaseID uint64
	)

	if c.IsSet("release-id") {
		// Fetch specific release
		releaseID = c.Uint64("release-id")
		if releaseID >= uint64(total.Int64()) {
			return fmt.Errorf("release ID %d does not exist (total releases: %d)", releaseID, total.Int64())
		}
		release, err = rmClient.GetRelease(ctx, avs, operatorSetID, releaseID)
		if err != nil {
			return fmt.Errorf("failed to get release %d: %w", releaseID, err)
		}
		log.Info("Fetched release", zap.Uint64("releaseID", releaseID))
	} else {
		// Fetch latest release
		release, releaseID, err = rmClient.GetLatestRelease(ctx, avs, operatorSetID)
		if err != nil {
			// Provide better error message for common issues
			if strings.Contains(err.Error(), "arithmetic underflow") || strings.Contains(err.Error(), "NoReleases") {
				return fmt.Errorf(`no releases available for this operator set

To push a release, run:
  flickr push --image <your-image>`)
			}
			return fmt.Errorf("failed to get latest release: %w", err)
		}
		log.Info("Fetched latest release", zap.Uint64("releaseID", releaseID))
	}

	// Validate release has artifacts
	if len(release.Artifacts) == 0 {
		return fmt.Errorf("no artifacts in release %d", releaseID)
	}

	// Determine which artifacts to pull
	artifactsToPull := release.Artifacts
	if !c.Bool("all") && len(release.Artifacts) > 1 {
		// Only pull the first artifact by default
		artifactsToPull = release.Artifacts[:1]
		log.Info("Pulling first artifact only (use --all for all artifacts)",
			zap.Int("totalArtifacts", len(release.Artifacts)))
	}

	// Pull each artifact
	pulledImages := make([]string, 0, len(artifactsToPull))
	for i, artifact := range artifactsToPull {
		// Convert digest to string format
		digest := ref.Digest32ToSha256String(artifact.Digest32)

		// Build pullable reference
		reference, err := ref.BuildReference(artifact.Registry, digest)
		if err != nil {
			return fmt.Errorf("failed to build reference for artifact %d: %w", i, err)
		}

		log.Info("Pulling Docker image",
			zap.Int("artifact", i+1),
			zap.Int("total", len(artifactsToPull)),
			zap.String("reference", reference))

		// Docker pull
		cmd := exec.Command("docker", "pull", reference)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to pull image %s: %v\n%s", reference, err, string(output))
		}

		pulledImages = append(pulledImages, reference)
		log.Info("Successfully pulled image", zap.String("reference", reference))
	}

	// Print summary
	fmt.Printf("\nSuccessfully pulled release %d\n", releaseID)
	fmt.Printf("AVS: %s\n", avs.Hex())
	fmt.Printf("Operator Set: %d\n", operatorSetID)
	fmt.Printf("Upgrade By Time: %d\n", release.UpgradeByTime)
	fmt.Printf("\nPulled Images:\n")
	for _, img := range pulledImages {
		fmt.Printf("  - %s\n", img)
	}

	if !c.Bool("all") && len(release.Artifacts) > 1 {
		fmt.Printf("\nNote: Only pulled first artifact. Use --all to pull all %d artifacts.\n", len(release.Artifacts))
	}

	return nil
}