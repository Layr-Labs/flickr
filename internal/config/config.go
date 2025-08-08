package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ContextKey is the key used to store context in cli.Context
type contextKey string

const (
	ConfigKey         contextKey = "config"
	ContextKey        contextKey = "context"
	LoggerKey         contextKey = "logger"
	ContractClientKey contextKey = "contractClient"
)

// Config represents the CLI configuration
type Config struct {
	CurrentContext string              `json:"currentContext,omitempty"`
	Contexts       map[string]*Context `json:"contexts,omitempty"`
}

// Context represents a configuration context
type Context struct {
	// Core settings
	AVSAddress       string `json:"avsAddress,omitempty"`
	OperatorSetID    uint32 `json:"operatorSetId,omitempty"`
	ReleaseManager   string `json:"releaseManager,omitempty"`
	RPCURL           string `json:"rpcUrl,omitempty"`
	
	// Optional settings
	Name             string            `json:"name,omitempty"`
	EnvironmentVars  map[string]string `json:"environmentVars,omitempty"`
	
	// ECDSA Signer configuration (mutually exclusive)
	ECDSAPrivateKey    string `json:"ecdsaPrivateKey,omitempty"`    // Hex-encoded private key
	KeystorePath       string `json:"keystorePath,omitempty"`       // Path to keystore file
	KeystorePassword   string `json:"keystorePassword,omitempty"`   // Keystore password
}

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".flickr")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(configDir, "config.json"), nil
}

// LoadConfig loads the configuration from disk
func LoadConfig() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty config if file doesn't exist
			return &Config{
				Contexts: make(map[string]*Context),
			}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if cfg.Contexts == nil {
		cfg.Contexts = make(map[string]*Context)
	}

	return &cfg, nil
}

// SaveConfig saves the configuration to disk
func SaveConfig(cfg *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetCurrentContext returns the current context
func GetCurrentContext() (*Context, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	if cfg.CurrentContext == "" {
		return nil, fmt.Errorf("no current context set")
	}

	ctx, exists := cfg.Contexts[cfg.CurrentContext]
	if !exists {
		return nil, fmt.Errorf("current context '%s' not found", cfg.CurrentContext)
	}

	return ctx, nil
}

// ToMap converts context to a map for display
func (c *Context) ToMap() map[string]interface{} {
	m := make(map[string]interface{})
	
	if c.AVSAddress != "" {
		m["avs-address"] = c.AVSAddress
	}
	if c.OperatorSetID != 0 {
		m["operator-set-id"] = c.OperatorSetID
	}
	if c.ReleaseManager != "" {
		m["release-manager"] = c.ReleaseManager
	}
	if c.RPCURL != "" {
		m["rpc-url"] = c.RPCURL
	}
	if c.Name != "" {
		m["name"] = c.Name
	}
	if len(c.EnvironmentVars) > 0 {
		m["environment-vars"] = c.EnvironmentVars
	}
	
	// Add signer info
	if c.ECDSAPrivateKey != "" {
		m["ecdsa-private-key"] = c.ECDSAPrivateKey
	}
	if c.KeystorePath != "" {
		m["keystore-path"] = c.KeystorePath
		if c.KeystorePassword != "" {
			m["keystore-password"] = c.KeystorePassword
		}
	}
	
	return m
}