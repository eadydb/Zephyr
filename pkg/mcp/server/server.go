package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/eadydb/zephyr/pkg/plugin"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps the MCP server with tool registry
type Server struct {
	mcpServer *server.MCPServer
	registry  plugin.ToolRegistry
	metrics   *MetricsCollector
	name      string
	version   string
}

// New creates a new MCP server instance
func New(name, version string, registry plugin.ToolRegistry) *Server {
	return &Server{
		name:     name,
		version:  version,
		registry: registry,
		metrics:  NewMetricsCollector(), // Create default metrics collector
	}
}

// NewWithMetrics creates a new MCP server instance with custom metrics collector
func NewWithMetrics(name, version string, registry plugin.ToolRegistry, metrics *MetricsCollector) *Server {
	return &Server{
		name:     name,
		version:  version,
		registry: registry,
		metrics:  metrics,
	}
}

// Start starts the MCP server
func (s *Server) Start() error {
	slog.Info("Starting MCP server", "name", s.name, "version", s.version)

	// Create new MCP server
	s.mcpServer = server.NewMCPServer(s.name, s.version)

	// Register tools with MCP server
	if err := s.registerTools(); err != nil {
		return fmt.Errorf("failed to register tools: %w", err)
	}

	slog.Info("MCP server started successfully")
	return nil
}

// Stop stops the MCP server
func (s *Server) Stop() error {
	slog.Info("Stopping MCP server...")

	// Shutdown registry
	if s.registry != nil {
		if err := s.registry.Shutdown(); err != nil {
			slog.Error("Error shutting down registry", "error", err)
		}
	}

	slog.Info("MCP server stopped")
	return nil
}

// GetMCPServer returns the underlying MCP server
func (s *Server) GetMCPServer() *server.MCPServer {
	return s.mcpServer
}

// GetMetrics returns the metrics collector
func (s *Server) GetMetrics() *MetricsCollector {
	return s.metrics
}

// registerTools registers all tools from the registry with the MCP server
func (s *Server) registerTools() error {
	if s.registry == nil {
		slog.Info("No registry provided, skipping tool registration")
		return nil
	}

	// Discover tools
	if err := s.registry.DiscoverTools(); err != nil {
		return fmt.Errorf("failed to discover tools: %w", err)
	}

	// Get all tools from registry
	tools := s.registry.ListTools()
	toolNames := make([]string, 0, len(tools))

	// Register each tool with MCP server
	for _, tool := range tools {
		if err := s.registerTool(tool); err != nil {
			slog.Warn("Failed to register tool", "name", tool.Name(), "error", err)
			continue
		}
		toolNames = append(toolNames, tool.Name())
	}

	slog.Info("Registered tools", "count", len(toolNames), "tools", toolNames)
	return nil
}

// registerTool registers a single tool with the MCP server
func (s *Server) registerTool(tool plugin.MCPToolPlugin) error {
	toolDef := tool.MCPToolDefinition()

	// Create MCP tool handler with metrics instrumentation
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()
		toolName := tool.Name()

		// Convert arguments to map using the helper method
		input := request.GetArguments()

		// Execute the tool
		result, err := tool.Execute(ctx, input)
		duration := time.Since(startTime)

		// Record metrics
		if s.metrics != nil {
			s.metrics.RecordRequest(duration, toolName, err != nil)
		}

		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Error executing tool %s: %v", toolName, err)),
				},
				IsError: true,
			}, nil
		}

		// Format result as text content
		resultText := ""
		switch v := result.(type) {
		case string:
			resultText = v
		case map[string]interface{}, []interface{}:
			// For complex data, format as JSON
			if jsonBytes, err := json.Marshal(v); err == nil {
				resultText = string(jsonBytes)
			} else {
				resultText = fmt.Sprintf("%+v", v)
			}
		default:
			resultText = fmt.Sprintf("%v", v)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent(resultText),
			},
		}, nil
	}

	// Create MCP tool definition with proper schema type
	mcpTool := mcp.Tool{
		Name:        toolDef.Name,
		Description: toolDef.Description,
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: toolDef.InputSchema,
		},
	}

	// If the tool has properties with required fields, extract them
	if props, ok := toolDef.InputSchema["properties"].(map[string]interface{}); ok {
		mcpTool.InputSchema.Properties = props
		if required, ok := toolDef.InputSchema["required"].([]string); ok {
			mcpTool.InputSchema.Required = required
		} else if required, ok := toolDef.InputSchema["required"].([]interface{}); ok {
			// Convert []interface{} to []string
			stringRequired := make([]string, len(required))
			for i, v := range required {
				if s, ok := v.(string); ok {
					stringRequired[i] = s
				}
			}
			mcpTool.InputSchema.Required = stringRequired
		}
	} else {
		// If no properties, use the whole schema as properties
		mcpTool.InputSchema.Properties = toolDef.InputSchema
	}

	// Register with MCP server
	s.mcpServer.AddTool(mcpTool, handler)

	return nil
}
