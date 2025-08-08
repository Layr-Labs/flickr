package metadata

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
	"github.com/yourorg/flickr/internal/config"
	"github.com/yourorg/flickr/internal/eth"
	"github.com/yourorg/flickr/internal/middleware"
	"github.com/yourorg/flickr/internal/signer"
	"go.uber.org/zap"
)

// Command returns the metadata command
func Command() *cli.Command {
	return &cli.Command{
		Name:  "metadata",
		Usage: "Manage metadata URIs for operator sets",
		Subcommands: []*cli.Command{
			setCommand(),
			getCommand(),
		},
	}
}

func setCommand() *cli.Command {
	return &cli.Command{
		Name:  "set",
		Usage: "Set metadata URI for an operator set",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "uri",
				Usage:    "Metadata URI (e.g., https://example.com/metadata.json)",
				Required: true,
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
				Value: 200000,
			},
		},
		Action: setAction,
	}
}

func getCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get metadata URI for an operator set",
		Flags: []cli.Flag{
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
		},
		Action: getAction,
	}
}

func setAction(c *cli.Context) error {
	log := middleware.GetLogger(c)

	// Get context
	currentCtx, err := middleware.GetCurrentContext(c)
	if err != nil {
		currentCtx = &config.Context{}
	}

	// Get configuration from flags or context
	avsAddress, operatorSetID, rpcURL, rmAddr, err := getConfig(c, currentCtx)
	if err != nil {
		return err
	}

	// Get signer from context
	sig, err := signer.FromContext(currentCtx)
	if err != nil {
		return fmt.Errorf("no signer configured: %w", err)
	}

	log.Info("Setting metadata URI",
		zap.String("avs", avsAddress),
		zap.Uint32("operatorSet", operatorSetID),
		zap.String("releaseManager", rmAddr.Hex()),
		zap.String("signer", sig.Address().Hex()))

	// Parse addresses
	avs := common.HexToAddress(avsAddress)

	// Get URI from flag
	uri := c.String("uri")
	if uri == "" {
		return fmt.Errorf("--uri is required")
	}

	// Create Ethereum client with signer
	rmClient, err := eth.NewClientWithSigner(rpcURL, rmAddr, sig)
	if err != nil {
		return fmt.Errorf("failed to create Ethereum client: %w", err)
	}
	defer rmClient.Close()

	// Publish metadata URI
	ctx := context.Background()
	tx, err := rmClient.PublishMetadataURI(ctx, avs, operatorSetID, uri, c.Uint64("gas-limit"))
	if err != nil {
		return fmt.Errorf("failed to publish metadata URI: %w", err)
	}

	log.Info("Transaction submitted",
		zap.String("txHash", tx.Hash().Hex()),
		zap.String("from", sig.Address().Hex()),
		zap.String("to", rmAddr.Hex()))

	fmt.Printf("Metadata URI set successfully!\n")
	fmt.Printf("Transaction: %s\n", tx.Hash().Hex())
	fmt.Printf("AVS: %s\n", avs.Hex())
	fmt.Printf("Operator Set: %d\n", operatorSetID)
	fmt.Printf("URI: %s\n", uri)

	return nil
}

func getAction(c *cli.Context) error {
	log := middleware.GetLogger(c)

	// Get context
	currentCtx, err := middleware.GetCurrentContext(c)
	if err != nil {
		currentCtx = &config.Context{}
	}

	// Get configuration from flags or context
	avsAddress, operatorSetID, rpcURL, rmAddr, err := getConfig(c, currentCtx)
	if err != nil {
		return err
	}

	log.Info("Getting metadata URI",
		zap.String("avs", avsAddress),
		zap.Uint32("operatorSet", operatorSetID),
		zap.String("releaseManager", rmAddr.Hex()))

	// Parse addresses
	avs := common.HexToAddress(avsAddress)

	// Create Ethereum client
	rmClient, err := eth.NewClient(rpcURL, rmAddr)
	if err != nil {
		return fmt.Errorf("failed to create Ethereum client: %w", err)
	}
	defer rmClient.Close()

	// Get metadata URI
	ctx := context.Background()
	uri, err := rmClient.GetMetadataURI(ctx, avs, operatorSetID)
	if err != nil {
		return fmt.Errorf("failed to get metadata URI: %w", err)
	}

	if uri == "" {
		fmt.Printf("No metadata URI set for AVS %s, Operator Set %d\n", avs.Hex(), operatorSetID)
		fmt.Printf("\nTo set a metadata URI, run:\n")
		fmt.Printf("  flickr metadata set --uri \"https://your-metadata-uri.json\"\n")
	} else {
		fmt.Printf("Metadata URI: %s\n", uri)
		fmt.Printf("AVS: %s\n", avs.Hex())
		fmt.Printf("Operator Set: %d\n", operatorSetID)
	}

	return nil
}

// getConfig extracts configuration from flags or context
func getConfig(c *cli.Context, currentCtx *config.Context) (string, uint32, string, common.Address, error) {
	// Get AVS address (from flag or context)
	avsAddress := c.String("avs")
	if avsAddress == "" {
		avsAddress = currentCtx.AVSAddress
	}
	if avsAddress == "" {
		return "", 0, "", common.Address{}, fmt.Errorf("--avs is required (or set in context with 'flickr context set --avs-address')")
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
		return "", 0, "", common.Address{}, fmt.Errorf("--rpc-url is required (or set in context with 'flickr context set --rpc-url')")
	}

	// Get release manager address (from flag, context, or chain default)
	releaseManager := c.String("release-manager")
	if releaseManager == "" {
		releaseManager = currentCtx.ReleaseManager
	}

	// Get the actual address (may use chain defaults)
	rmAddr, err := eth.GetReleaseManagerAddress(rpcURL, releaseManager)
	if err != nil {
		return "", 0, "", common.Address{}, fmt.Errorf("failed to get ReleaseManager address: %w", err)
	}

	return avsAddress, operatorSetID, rpcURL, rmAddr, nil
}