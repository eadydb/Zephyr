package transport

import (
	"fmt"
	"time"

	"github.com/eadydb/zephyr/internal/config"
	"github.com/mark3labs/mcp-go/server"
)

// Factory implements TransportFactory interface
type Factory struct {
	mcpServer *server.MCPServer
}

// NewFactory creates a new transport factory instance
func NewFactory() TransportFactory {
	return &Factory{}
}

// SetMCPServer sets the MCP server instance for the factory
func (f *Factory) SetMCPServer(mcpServer *server.MCPServer) {
	f.mcpServer = mcpServer
}

// CreateTransport creates a transport adapter based on the configuration
func (f *Factory) CreateTransport(transportConfig TransportConfig) (TransportAdapter, error) {
	// For factory usage, we need an MCP server instance
	// This will be set by the caller before calling CreateTransport
	if f.mcpServer == nil {
		return nil, fmt.Errorf("MCP server not set in factory")
	}

	return CreateTransportFromConfig(transportConfig, f.mcpServer)
}

// SupportedProtocols returns the list of supported transport protocols
func (f *Factory) SupportedProtocols() []string {
	return []string{"stdio", "sse", "http"}
}

// CreateTransportFromFullConfig creates a transport adapter from full application config
func CreateTransportFromFullConfig(cfg *config.Config, mcpServer *server.MCPServer) (TransportAdapter, error) {
	return CreateTransport(cfg.Transport.Protocol, mcpServer, &cfg.Transport)
}

// CreateTransportFromConfig is a convenience function that creates a transport
// adapter directly from TransportConfig (for compatibility with adapter.go interface)
func CreateTransportFromConfig(transportConfig TransportConfig, mcpServer *server.MCPServer) (TransportAdapter, error) {
	protocol := transportConfig.Protocol

	switch protocol {
	case "stdio":
		return NewSTDIOAdapter(mcpServer), nil

	case "sse":
		// Extract SSE options from generic options map
		options := transportConfig.Options
		sseConfig := SSEConfig{
			Host:        getStringOption(options, "host", "localhost"),
			Port:        getIntOption(options, "port", 26841),
			CORSEnabled: getBoolOption(options, "cors_enabled", true),
		}
		return NewSSEAdapter(mcpServer, sseConfig), nil

	case "http":
		// Extract HTTP options from generic options map
		options := transportConfig.Options
		httpConfig := HTTPConfig{
			Host:    getStringOption(options, "host", "localhost"),
			Port:    getIntOption(options, "port", 26842),
			Timeout: getDurationOption(options, "timeout", 30*time.Second),
		}
		return NewHTTPAdapter(mcpServer, httpConfig), nil

	default:
		return nil, fmt.Errorf("unsupported transport protocol: %s", protocol)
	}
}

// Helper functions to extract typed values from options map
func getStringOption(options map[string]interface{}, key, defaultValue string) string {
	if val, ok := options[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

func getIntOption(options map[string]interface{}, key string, defaultValue int) int {
	if val, ok := options[key]; ok {
		if num, ok := val.(int); ok {
			return num
		}
		if num, ok := val.(float64); ok {
			return int(num)
		}
	}
	return defaultValue
}

func getBoolOption(options map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := options[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultValue
}

func getDurationOption(options map[string]interface{}, key string, defaultValue time.Duration) time.Duration {
	if val, ok := options[key]; ok {
		if str, ok := val.(string); ok {
			if duration, err := time.ParseDuration(str); err == nil {
				return duration
			}
		}
	}
	return defaultValue
}

// CreateTransport creates a transport adapter based on the protocol
func CreateTransport(protocol string, mcpServer *server.MCPServer, cfg *config.TransportConfig) (TransportAdapter, error) {
	switch protocol {
	case "stdio":
		return NewSTDIOAdapter(mcpServer), nil
	case "sse":
		sseConfig := SSEConfig{
			Host:        cfg.SSE.Host,
			Port:        cfg.SSE.Port,
			CORSEnabled: cfg.SSE.CORSEnabled,
		}
		return NewSSEAdapter(mcpServer, sseConfig), nil
	case "http":
		httpConfig := HTTPConfig{
			Host:    cfg.HTTP.Host,
			Port:    cfg.HTTP.Port,
			Timeout: cfg.HTTP.Timeout,
		}
		return NewHTTPAdapter(mcpServer, httpConfig), nil
	default:
		return nil, fmt.Errorf("unsupported transport protocol: %s", protocol)
	}
}
