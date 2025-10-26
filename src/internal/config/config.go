package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Upstreams []UpstreamConfig `yaml:"upstreams"`
	Logging   LoggingConfig   `yaml:"logging"`
	Token     TokenConfig     `yaml:"token"`
}

// ServerConfig holds server settings
type ServerConfig struct {
	Address      string   `yaml:"address"`
	Port         int      `yaml:"port"`
	ReadTimeout  int      `yaml:"read_timeout"`   // seconds
	WriteTimeout int      `yaml:"write_timeout"`  // seconds
	IdleTimeout  int      `yaml:"idle_timeout"`   // seconds
	AllowedPaths []string `yaml:"allowed_paths"`  // allowed path patterns (e.g., /run_sse, /apps/*)
}

// UpstreamConfig defines an upstream service
type UpstreamConfig struct {
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	Audience string `yaml:"audience"`
	Timeout  int    `yaml:"timeout"` // seconds
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Format string `yaml:"format"` // json, text
}

// TokenConfig holds token management settings
type TokenConfig struct {
	RefreshBeforeExpiry int  `yaml:"refresh_before_expiry"` // minutes
	EnableCache         bool `yaml:"enable_cache"`
}

// GetAddress returns the full server address
func (s *ServerConfig) GetAddress() string {
	return fmt.Sprintf("%s:%d", s.Address, s.Port)
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Server.Port)
	}

	if len(c.Upstreams) == 0 {
		return fmt.Errorf("no upstreams configured")
	}

	for i, upstream := range c.Upstreams {
		if upstream.Name == "" {
			return fmt.Errorf("upstream[%d]: name is required", i)
		}
		if upstream.URL == "" {
			return fmt.Errorf("upstream[%d]: url is required", i)
		}
		if upstream.Audience == "" {
			return fmt.Errorf("upstream[%d]: audience is required", i)
		}
	}

	return nil
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Set defaults
	if config.Server.Address == "" {
		config.Server.Address = "0.0.0.0"
	}
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.Server.ReadTimeout == 0 {
		config.Server.ReadTimeout = 30
	}
	if config.Server.WriteTimeout == 0 {
		config.Server.WriteTimeout = 30
	}
	if config.Server.IdleTimeout == 0 {
		config.Server.IdleTimeout = 120
	}
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	if config.Logging.Format == "" {
		config.Logging.Format = "text"
	}
	if config.Token.RefreshBeforeExpiry == 0 {
		config.Token.RefreshBeforeExpiry = 5 // 5 minutes
	}
	config.Token.EnableCache = true // Always enable cache

	// Set default timeouts for upstreams
	for i := range config.Upstreams {
		if config.Upstreams[i].Timeout == 0 {
			config.Upstreams[i].Timeout = 30
		}
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}
