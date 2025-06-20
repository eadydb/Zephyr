package plugin

import (
	"context"
)

// MCPTool represents an MCP tool definition for the protocol
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// MCPToolPlugin defines the interface for MCP tool plugins
// This extends our existing plugin system to support MCP-specific functionality
type MCPToolPlugin interface {
	// Basic plugin interface
	Name() string
	Description() string
	Version() string

	// MCP-specific methods
	MCPToolDefinition() MCPTool
	Execute(ctx context.Context, input map[string]interface{}) (interface{}, error)
	InputSchema() map[string]interface{}

	// Lifecycle methods
	Initialize() error
	Cleanup() error
}

// ToolRegistry manages MCP tool plugins
type ToolRegistry interface {
	// RegisterTool adds a tool to the registry
	RegisterTool(tool MCPToolPlugin) error

	// UnregisterTool removes a tool from the registry
	UnregisterTool(name string) error

	// GetTool retrieves a tool by name
	GetTool(name string) (MCPToolPlugin, error)

	// ListTools returns all registered tools
	ListTools() []MCPToolPlugin

	// DiscoverTools scans for available tools
	DiscoverTools() error

	// Lifecycle
	Shutdown() error
}

// PluginAdapter bridges existing plugins to MCP tools
type PluginAdapter interface {
	// CanAdapt checks if a plugin can be adapted to MCP tool
	CanAdapt(plugin interface{}) bool

	// Adapt converts a plugin to MCPToolPlugin
	Adapt(plugin interface{}) (MCPToolPlugin, error)
}
