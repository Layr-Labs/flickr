package context

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/yourorg/flickr/internal/config"
	"github.com/yourorg/flickr/internal/middleware"
	"go.uber.org/zap"
)

func setCommand() *cli.Command {
	return &cli.Command{
		Name:  "set",
		Usage: "Set context properties",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "avs-address",
				Usage: "Set the AVS contract address",
			},
			&cli.Uint64Flag{
				Name:  "operator-set-id",
				Usage: "Set the operator set ID",
			},
			&cli.StringFlag{
				Name:  "rpc-url",
				Usage: "Set the Ethereum RPC URL",
			},
			&cli.StringFlag{
				Name:  "release-manager",
				Usage: "Set the release manager contract address",
			},
			&cli.StringFlag{
				Name:  "name",
				Usage: "Set the container name prefix",
			},
			&cli.StringSliceFlag{
				Name:  "env",
				Usage: "Set environment variables (KEY=VALUE)",
			},
			&cli.StringFlag{
				Name:  "ecdsa-private-key",
				Usage: "Set ECDSA private key (hex encoded)",
			},
			&cli.StringFlag{
				Name:  "keystore-path",
				Usage: "Set path to keystore file",
			},
			&cli.StringFlag{
				Name:  "keystore-password",
				Usage: "Set keystore password",
			},
		},
		Action: contextSetAction,
	}
}

func contextSetAction(c *cli.Context) error {
	log := middleware.GetLogger(c)

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.CurrentContext == "" {
		return fmt.Errorf("no current context set")
	}

	ctx, exists := cfg.Contexts[cfg.CurrentContext]
	if !exists {
		return fmt.Errorf("current context '%s' not found", cfg.CurrentContext)
	}

	updated := false

	if addr := c.String("avs-address"); addr != "" {
		ctx.AVSAddress = addr
		updated = true
		log.Info("Updated AVS address", zap.String("address", addr))
	}

	if id := c.Uint64("operator-set-id"); c.IsSet("operator-set-id") {
		ctx.OperatorSetID = uint32(id)
		updated = true
		log.Info("Updated operator set ID", zap.Uint32("id", uint32(id)))
	}

	if url := c.String("rpc-url"); url != "" {
		ctx.RPCURL = url
		updated = true
		log.Info("Updated RPC URL", zap.String("url", url))
	}

	if addr := c.String("release-manager"); addr != "" {
		ctx.ReleaseManager = addr
		updated = true
		log.Info("Updated release manager address", zap.String("address", addr))
	}

	if name := c.String("name"); name != "" {
		ctx.Name = name
		updated = true
		log.Info("Updated container name prefix", zap.String("name", name))
	}

	// Handle environment variables
	envFlags := c.StringSlice("env")
	if len(envFlags) > 0 {
		if ctx.EnvironmentVars == nil {
			ctx.EnvironmentVars = make(map[string]string)
		}

		for _, env := range envFlags {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid env format: %s (expected KEY=VALUE)", env)
			}

			key := parts[0]
			value := parts[1]
			ctx.EnvironmentVars[key] = value
			log.Info("Set environment variable", zap.String("key", key))
			updated = true
		}
	}

	// Handle signer configuration (mutually exclusive)
	if privateKey := c.String("ecdsa-private-key"); privateKey != "" {
		// Setting private key clears keystore settings
		ctx.ECDSAPrivateKey = privateKey
		ctx.KeystorePath = ""
		ctx.KeystorePassword = ""
		updated = true
		log.Info("Updated ECDSA private key")
	}

	if keystorePath := c.String("keystore-path"); keystorePath != "" {
		// Setting keystore clears private key
		ctx.KeystorePath = keystorePath
		ctx.ECDSAPrivateKey = ""
		updated = true
		log.Info("Updated keystore path", zap.String("path", keystorePath))
	}

	if keystorePassword := c.String("keystore-password"); keystorePassword != "" {
		if ctx.KeystorePath == "" {
			return fmt.Errorf("keystore-password requires keystore-path to be set")
		}
		ctx.KeystorePassword = keystorePassword
		updated = true
		log.Info("Updated keystore password")
	}

	if !updated {
		return fmt.Errorf("no values provided to update")
	}

	if err := config.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Context '%s' updated\n", cfg.CurrentContext)
	return nil
}