package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/lazyclaw/lazyclaw/internal/models"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Instances   []models.InstanceProfile `yaml:"instances"`
	UI          UIConfig                 `yaml:"ui"`
	Security    SecurityConfig           `yaml:"security"`
	OpenClawCLI string                   `yaml:"openclaw_cli,omitempty"` // Path to openclaw binary
}

// UIConfig holds UI-related settings
type UIConfig struct {
	Theme        string `yaml:"theme"`
	RefreshMs    int    `yaml:"refresh_ms"`
	LogTailLines int    `yaml:"log_tail_lines"`
}

// SecurityConfig holds security-related settings
type SecurityConfig struct {
	DefaultScopes    []string `yaml:"default_scopes"`
	AllowWriteScopes bool     `yaml:"allow_write_scopes"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Instances: []models.InstanceProfile{},
		UI: UIConfig{
			Theme:        "auto",
			RefreshMs:    1000,
			LogTailLines: 500,
		},
		Security: SecurityConfig{
			DefaultScopes:    []string{"operator.read"},
			AllowWriteScopes: false,
		},
	}
}

// ConfigDir returns the configuration directory path
func ConfigDir() (string, error) {
	// Check XDG_CONFIG_HOME first
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "lazyclaw"), nil
}

// ConfigPath returns the full path to the config file
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yml"), nil
}

// Load loads the configuration from disk
// Returns the config, whether this is a first run (no config exists), and any error
func Load() (*Config, bool, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, false, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// First run - return default config
			return DefaultConfig(), true, nil
		}
		return nil, false, err
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, false, err
	}

	return cfg, false, nil
}

// Save writes the configuration to disk
func Save(cfg *Config) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path, err := ConfigPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	// Write atomically: write to temp file, then rename
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}

// AddInstance adds a new instance to the configuration
func (c *Config) AddInstance(instance models.InstanceProfile) {
	c.Instances = append(c.Instances, instance)
}

// GetInstance returns an instance by name
func (c *Config) GetInstance(name string) *models.InstanceProfile {
	for i := range c.Instances {
		if c.Instances[i].Name == name {
			return &c.Instances[i]
		}
	}
	return nil
}
