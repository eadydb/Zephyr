package transport

import (
	"context"
)

// TransportAdapter provides a unified interface for all MCP transport protocols
// This is the core abstraction that allows protocol-agnostic MCP server implementation
type TransportAdapter interface {
	// Start begins the transport protocol listening and handling
	Start(ctx context.Context) error

	// Stop gracefully shuts down the transport
	Stop() error

	// Name returns the protocol name (stdio, sse, http)
	Name() string

	// IsHealthy returns true if the transport is functioning properly
	IsHealthy() bool
}

// TransportConfig holds configuration for any transport protocol
type TransportConfig struct {
	Protocol string                 `yaml:"protocol"`
	Options  map[string]interface{} `yaml:"options"`
}

// TransportFactory creates transport adapters based on configuration
type TransportFactory interface {
	// CreateTransport creates a new transport adapter for the specified protocol
	CreateTransport(config TransportConfig) (TransportAdapter, error)

	// SupportedProtocols returns list of supported protocol names
	SupportedProtocols() []string
}
