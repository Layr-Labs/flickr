package context

import (
	"fmt"

	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
	"github.com/yourorg/flickr/internal/config"
	"github.com/yourorg/flickr/internal/middleware"
	"github.com/yourorg/flickr/internal/signer"
	"go.uber.org/zap"
)

func listCommand() *cli.Command {
	return &cli.Command{
		Name:   "list",
		Usage:  "List all contexts",
		Action: contextListAction,
	}
}

func contextListAction(c *cli.Context) error {
	log := middleware.GetLogger(c)

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	log.Info("Listing contexts", zap.Int("count", len(cfg.Contexts)))

	if len(cfg.Contexts) == 0 {
		fmt.Println("No contexts configured")
		fmt.Println("\nTo create a context, run:")
		fmt.Println("  flickr context create --name default --use")
		return nil
	}

	// Create table
	table := tablewriter.NewWriter(c.App.Writer)
	table.Header("CURRENT", "NAME", "AVS ADDRESS", "OPERATOR SET", "RPC URL", "SIGNER")

	for name, ctx := range cfg.Contexts {
		current := ""
		if name == cfg.CurrentContext {
			current = "*"
		}

		avsAddr := ctx.AVSAddress
		if avsAddr == "" {
			avsAddr = "-"
		}

		rpcURL := ctx.RPCURL
		if rpcURL == "" {
			rpcURL = "-"
		}

		operatorSet := "-"
		if ctx.OperatorSetID != 0 {
			operatorSet = fmt.Sprintf("%d", ctx.OperatorSetID)
		}

		// Get signer address if configured
		signerAddr := "-"
		if sig, err := signer.FromContext(ctx); err == nil {
			signerAddr = sig.Address().Hex()
		}

		table.Append([]string{
			current,
			name,
			avsAddr,
			operatorSet,
			rpcURL,
			signerAddr,
		})
	}

	table.Render()
	return nil
}