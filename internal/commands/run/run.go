package run

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
	"github.com/yourorg/flickr/internal/config"
	"github.com/yourorg/flickr/internal/controller"
	"github.com/yourorg/flickr/internal/docker"
	"github.com/yourorg/flickr/internal/eth"
	"github.com/yourorg/flickr/internal/middleware"
	"go.uber.org/zap"
)

// Command returns the run command
func Command() *cli.Command {
	return &cli.Command{
		Name:  "run",
		Usage: "Run an AVS release in Docker",
		Description: `Fetches release information from the on-chain ReleaseManager and runs the 
specified release as a Docker container with AVS context. Can run the latest release or a 
specific release ID.`,
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
			&cli.Uint64Flag{
				Name:  "release-id",
				Usage: "Specific release ID (defaults to latest)",
			},
			&cli.StringFlag{
				Name:  "name",
				Usage: "Container name",
			},
			&cli.BoolFlag{
				Name:    "detach",
				Aliases: []string{"d"},
				Usage:   "Run container in background",
			},
			&cli.StringSliceFlag{
				Name:    "env",
				Aliases: []string{"e"},
				Usage:   "Environment variables (KEY=VALUE)",
			},
			&cli.StringSliceFlag{
				Name:  "cmd",
				Usage: "Command to run in the container",
			},
		},
		Action: runAction,
	}
}

func runAction(c *cli.Context) error {
	log := middleware.GetLogger(c)
	
	// Get context
	currentCtx, err := middleware.GetCurrentContext(c)
	if err != nil {
		// Context might be empty for help commands
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

	// Parse optional release ID
	var relID *uint64
	if c.IsSet("release-id") {
		id := c.Uint64("release-id")
		relID = &id
	}

	// Parse environment variables
	envMap := make(map[string]string)
	
	// Start with context environment variables
	for k, v := range currentCtx.EnvironmentVars {
		envMap[k] = v
	}
	
	// Override with command-line environment variables
	for _, env := range c.StringSlice("env") {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid env format: %s (expected KEY=VALUE)", env)
		}
		envMap[parts[0]] = parts[1]
	}

	// Get container name (from flag or context)
	containerName := c.String("name")
	if containerName == "" && currentCtx.Name != "" {
		containerName = fmt.Sprintf("%s-%d", currentCtx.Name, time.Now().Unix())
	}

	// Create Ethereum client
	rmClient, err := eth.NewClient(rpcURL, rmAddr)
	if err != nil {
		return fmt.Errorf("failed to create Ethereum client: %w", err)
	}
	defer rmClient.Close()

	// Create Docker runner
	dockerRunner := docker.New()

	// Create controller
	ctrl := controller.New(rmClient, dockerRunner)

	// Prepare config
	cfg := controller.RunConfig{
		AVS:            avs,
		OperatorSetID:  operatorSetID,
		ReleaseID:      relID,
		ReleaseManager: rmAddr,
		RPCURL:         rpcURL,
		Name:           containerName,
		Detached:       c.Bool("detach"),
		Env:            envMap,
		Cmd:            c.StringSlice("cmd"),
	}

	// Execute
	ctx := context.Background()
	if err := ctrl.Execute(ctx, cfg); err != nil {
		return err
	}

	if c.Bool("detach") {
		fmt.Println("Container started in detached mode")
		if containerName != "" {
			fmt.Printf("Container name: %s\n", containerName)
		}
	} else {
		fmt.Println("Container execution completed")
	}

	return nil
}