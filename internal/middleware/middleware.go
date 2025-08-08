package middleware

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/yourorg/flickr/internal/config"
	"github.com/yourorg/flickr/internal/logger"
	"go.uber.org/zap"
)

// ChainBeforeFuncs chains multiple BeforeFuncs together
func ChainBeforeFuncs(funcs ...cli.BeforeFunc) cli.BeforeFunc {
	return func(c *cli.Context) error {
		for _, fn := range funcs {
			if err := fn(c); err != nil {
				return err
			}
		}
		return nil
	}
}

// ConfigBeforeFunc loads configuration and context
func ConfigBeforeFunc(c *cli.Context) error {
	// Initialize logger early
	verbose := c.Bool("verbose")
	logger.InitGlobalLoggerWithWriter(verbose, c.App.Writer)
	l := logger.GetLogger()

	// Check if user is requesting help
	for _, arg := range os.Args {
		if arg == "--help" || arg == "-h" || arg == "help" {
			// Set empty context for help display
			c.Context = context.WithValue(c.Context, config.ConfigKey, &config.Config{})
			c.Context = context.WithValue(c.Context, config.ContextKey, &config.Context{})
			return nil
		}
	}

	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		l.Error("Failed to load config", zap.Error(err))
		// Create empty config if it doesn't exist
		cfg = &config.Config{
			Contexts: make(map[string]*config.Context),
		}
	}

	// Get current context
	var currentCtx *config.Context
	if cfg.CurrentContext != "" {
		if ctx, exists := cfg.Contexts[cfg.CurrentContext]; exists {
			currentCtx = ctx
		}
	}

	// Allow context commands to run without a current context
	if currentCtx == nil && !isContextCommand(c) && !isHelpCommand(c) {
		// For non-context commands, we need a context
		l.Error("No context configured")
		fmt.Fprintf(os.Stderr, "\nError: No context configured\n\n")
		fmt.Fprintf(os.Stderr, "To create a context, run:\n")
		fmt.Fprintf(os.Stderr, "  flickr context create --name default --use\n\n")
		fmt.Fprintf(os.Stderr, "To list available contexts:\n")
		fmt.Fprintf(os.Stderr, "  flickr context list\n\n")
		return fmt.Errorf("no context configured")
	}

	// If no context for context commands, create empty one
	if currentCtx == nil {
		currentCtx = &config.Context{}
	}

	// Store in context
	c.Context = context.WithValue(c.Context, config.ConfigKey, cfg)
	c.Context = context.WithValue(c.Context, config.ContextKey, currentCtx)
	c.Context = context.WithValue(c.Context, config.LoggerKey, l)

	return nil
}

// LoggerBeforeFunc initializes the logger
func LoggerBeforeFunc(c *cli.Context) error {
	verbose := c.Bool("verbose")
	logger.InitGlobalLoggerWithWriter(verbose, c.App.Writer)
	l := logger.GetLogger()
	c.Context = context.WithValue(c.Context, config.LoggerKey, l)
	return nil
}

// GetLogger retrieves the logger from context
func GetLogger(c *cli.Context) logger.Logger {
	if l, ok := c.Context.Value(config.LoggerKey).(logger.Logger); ok {
		return l
	}
	
	// Create a new logger if not found
	verbose := c.Bool("verbose")
	return logger.NewLoggerWithWriter(verbose, c.App.Writer)
}

// GetConfig retrieves the config from context
func GetConfig(c *cli.Context) (*config.Config, error) {
	cfg, ok := c.Context.Value(config.ConfigKey).(*config.Config)
	if !ok || cfg == nil {
		return nil, fmt.Errorf("config not initialized")
	}
	return cfg, nil
}

// GetCurrentContext retrieves the current context from context
func GetCurrentContext(c *cli.Context) (*config.Context, error) {
	ctx, ok := c.Context.Value(config.ContextKey).(*config.Context)
	if !ok || ctx == nil {
		return nil, fmt.Errorf("context not initialized")
	}
	return ctx, nil
}

// ExitErrHandler handles errors on exit
func ExitErrHandler(c *cli.Context, err error) {
	if err == nil {
		return
	}

	// Try to get logger from context, or create a new one
	var log logger.Logger
	if c != nil {
		log = GetLogger(c)
	} else {
		logger.InitGlobalLogger(false)
		log = logger.GetLogger()
	}

	// Log the error with appropriate context
	if c != nil && c.Command != nil {
		log.Error("Command execution failed",
			zap.String("command", c.Command.Name),
			zap.Error(err))
	} else {
		log.Error("Command execution failed", zap.Error(err))
	}
}

func isContextCommand(c *cli.Context) bool {
	if c.NArg() == 0 {
		return false
	}
	cmd := c.Args().Get(0)
	return cmd == "context"
}

func isHelpCommand(c *cli.Context) bool {
	if c.NArg() == 0 {
		return false
	}
	cmd := c.Args().Get(0)
	return cmd == "help" || cmd == "version"
}