package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the complete application configuration
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Transport  TransportConfig  `yaml:"transport"`
	Monitoring MonitoringConfig `yaml:"monitoring"`
	Plugins    PluginsConfig    `yaml:"plugins"`
	Logging    LoggingConfig    `yaml:"logging"`
	Security   SecurityConfig   `yaml:"security"`
}

// ServerConfig holds server-level configuration
type ServerConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Debug   bool   `yaml:"debug"`
}

// TransportConfig holds transport protocol configuration
type TransportConfig struct {
	Protocol string      `yaml:"protocol"`
	STDIO    STDIOConfig `yaml:"stdio"`
	SSE      SSEConfig   `yaml:"sse"`
	HTTP     HTTPConfig  `yaml:"http"`
}

// STDIOConfig holds STDIO transport configuration
type STDIOConfig struct {
	BufferSize int `yaml:"buffer_size"`
}

// SSEConfig holds Server-Sent Events configuration
type SSEConfig struct {
	Port        int    `yaml:"port"`
	Host        string `yaml:"host"`
	CORSEnabled bool   `yaml:"cors_enabled"`
}

// HTTPConfig holds HTTP transport configuration
type HTTPConfig struct {
	Port    int           `yaml:"port"`
	Host    string        `yaml:"host"`
	Timeout time.Duration `yaml:"timeout"`
}

// PluginsConfig holds plugin system configuration
type PluginsConfig struct {
	Discovery DiscoveryConfig       `yaml:"discovery"`
	Tools     map[string]ToolConfig `yaml:"tools"`
}

// DiscoveryConfig holds plugin discovery configuration
type DiscoveryConfig struct {
	Enabled      bool          `yaml:"enabled"`
	Directories  []string      `yaml:"directories"`
	ScanInterval time.Duration `yaml:"scan_interval"`
}

// ToolConfig holds individual tool configuration
type ToolConfig struct {
	Enabled  bool                   `yaml:"enabled"`
	Settings map[string]interface{} `yaml:"settings,inline"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
	File   string `yaml:"file"`
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	RateLimit RateLimitConfig `yaml:"rate_limit"`
	Timeout   TimeoutConfig   `yaml:"timeout"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerMinute int  `yaml:"requests_per_minute"`
}

// TimeoutConfig holds timeout configuration
type TimeoutConfig struct {
	Request  time.Duration `yaml:"request"`
	Shutdown time.Duration `yaml:"shutdown"`
}

// MonitoringConfig configures monitoring and metrics
type MonitoringConfig struct {
	Enabled        bool            `yaml:"enabled"`
	Port           int             `yaml:"port"`
	Host           string          `yaml:"host"`
	Endpoints      EndpointsConfig `yaml:"endpoints"`
	UpdateInterval string          `yaml:"update_interval"`
}

// EndpointsConfig configures monitoring endpoints
type EndpointsConfig struct {
	Metrics string `yaml:"metrics"`
	Health  string `yaml:"health"`
}

// Load loads configuration from file with environment variable overrides
func Load(configPath string) (*Config, error) {
	// Start with defaults
	config := defaultConfig()

	// Load from file if exists
	if configPath != "" {
		if err := loadFromFile(config, configPath); err != nil {
			return nil, fmt.Errorf("failed to load config from file: %w", err)
		}
	}

	// Apply environment variable overrides
	applyEnvOverrides(config)

	// Validate configuration
	if err := validate(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// defaultConfig returns configuration with sensible defaults
func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Name:    "zephyr-mcp-server",
			Version: "1.0.0",
			Debug:   false,
		},
		Transport: TransportConfig{
			Protocol: "stdio",
			STDIO: STDIOConfig{
				BufferSize: 4096,
			},
			SSE: SSEConfig{
				Port:        26841,
				Host:        "localhost",
				CORSEnabled: true,
			},
			HTTP: HTTPConfig{
				Port:    26842,
				Host:    "localhost",
				Timeout: 30 * time.Second,
			},
		},
		Plugins: PluginsConfig{
			Discovery: DiscoveryConfig{
				Enabled:      true,
				Directories:  []string{"./plugins"},
				ScanInterval: 60 * time.Second,
			},
			Tools: map[string]ToolConfig{
				"systeminfo": {Enabled: true},
				"currenttime": {
					Enabled: true,
					Settings: map[string]interface{}{
						"timezone": "UTC",
					},
				},
			},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
		Security: SecurityConfig{
			RateLimit: RateLimitConfig{
				Enabled:           true,
				RequestsPerMinute: 100,
			},
			Timeout: TimeoutConfig{
				Request:  10 * time.Second,
				Shutdown: 30 * time.Second,
			},
		},
		Monitoring: MonitoringConfig{
			Enabled:        true,
			Port:           26843,
			Host:           "localhost",
			Endpoints:      EndpointsConfig{Metrics: "/metrics", Health: "/health"},
			UpdateInterval: "1m",
		},
	}
}

// loadFromFile loads configuration from YAML file
func loadFromFile(config *Config, path string) error {
	if !filepath.IsAbs(path) {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		path = filepath.Join(wd, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, config)
}

// applyEnvOverrides applies environment variable overrides
func applyEnvOverrides(config *Config) {
	// Server configuration
	if val := os.Getenv("ZEPHYR_SERVER_DEBUG"); val != "" {
		config.Server.Debug = strings.ToLower(val) == "true"
	}

	// Transport configuration
	if val := os.Getenv("ZEPHYR_TRANSPORT_PROTOCOL"); val != "" {
		config.Transport.Protocol = val
	}
	if val := os.Getenv("ZEPHYR_TRANSPORT_SSE_PORT"); val != "" {
		if port := parseIntEnv(val); port > 0 {
			config.Transport.SSE.Port = port
		}
	}
	if val := os.Getenv("ZEPHYR_TRANSPORT_HTTP_PORT"); val != "" {
		if port := parseIntEnv(val); port > 0 {
			config.Transport.HTTP.Port = port
		}
	}

	// Logging configuration
	if val := os.Getenv("ZEPHYR_LOGGING_LEVEL"); val != "" {
		config.Logging.Level = val
	}
}
