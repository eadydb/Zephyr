package config

import (
	"fmt"
	"strconv"
)

// validate performs configuration validation
func validate(config *Config) error {
	// Validate transport protocol
	validProtocols := map[string]bool{
		"stdio": true,
		"sse":   true,
		"http":  true,
	}

	if !validProtocols[config.Transport.Protocol] {
		return fmt.Errorf("invalid transport protocol: %s (must be one of: stdio, sse, http)", config.Transport.Protocol)
	}

	// Validate port numbers
	if config.Transport.SSE.Port < 1 || config.Transport.SSE.Port > 65535 {
		return fmt.Errorf("invalid SSE port: %d (must be 1-65535)", config.Transport.SSE.Port)
	}

	if config.Transport.HTTP.Port < 1 || config.Transport.HTTP.Port > 65535 {
		return fmt.Errorf("invalid HTTP port: %d (must be 1-65535)", config.Transport.HTTP.Port)
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if !validLogLevels[config.Logging.Level] {
		return fmt.Errorf("invalid log level: %s (must be one of: debug, info, warn, error)", config.Logging.Level)
	}

	// Validate timeouts are positive
	if config.Security.Timeout.Request <= 0 {
		return fmt.Errorf("request timeout must be positive")
	}

	if config.Security.Timeout.Shutdown <= 0 {
		return fmt.Errorf("shutdown timeout must be positive")
	}

	return nil
}

// Enhanced parseIntEnv with proper error handling
func parseIntEnv(val string) int {
	if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
		return parsed
	}
	return 0
}
