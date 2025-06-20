package main

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/eadydb/zephyr/pkg/plugin"
)

// Plugin is the exported plugin instance
var Plugin plugin.DynamicPlugin = &SystemInfoPlugin{}

// SystemInfoPlugin implements the DynamicPlugin interface
type SystemInfoPlugin struct {
	initialized bool
}

// NewPlugin is the factory function that will be called by the plugin loader
func NewPlugin() plugin.DynamicPlugin {
	return &SystemInfoPlugin{}
}

// Name returns the plugin name
func (p *SystemInfoPlugin) Name() string {
	return "systeminfo"
}

// Version returns the plugin version
func (p *SystemInfoPlugin) Version() string {
	return "1.0.0"
}

// Description returns the plugin description
func (p *SystemInfoPlugin) Description() string {
	return "Provides system information including OS, architecture, memory, and runtime details"
}

// Initialize initializes the plugin
func (p *SystemInfoPlugin) Initialize() error {
	if p.initialized {
		return fmt.Errorf("plugin already initialized")
	}
	p.initialized = true
	return nil
}

// Shutdown cleans up the plugin
func (p *SystemInfoPlugin) Shutdown() error {
	p.initialized = false
	return nil
}

// MCPToolDefinition returns the MCP tool definition
func (p *SystemInfoPlugin) MCPToolDefinition() plugin.MCPTool {
	return plugin.MCPTool{
		Name:        "systeminfo",
		Description: "Get system information including OS, architecture, memory usage, and Go runtime details",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"detailed": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether to include detailed memory statistics",
					"default":     true,
				},
			},
		},
	}
}

// InputSchema returns the input schema for the tool
func (p *SystemInfoPlugin) InputSchema() map[string]interface{} {
	return p.MCPToolDefinition().InputSchema
}

// Execute executes the tool with the given arguments
func (p *SystemInfoPlugin) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	if !p.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	// Parse detailed flag
	detailed := true
	if detailedArg, exists := args["detailed"]; exists {
		if d, ok := detailedArg.(bool); ok {
			detailed = d
		}
	}

	// Get basic system info
	info := map[string]interface{}{
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
		"cpus":       runtime.NumCPU(),
		"go_version": runtime.Version(),
		"goroutines": runtime.NumGoroutine(),
	}

	// Add detailed memory info if requested
	if detailed {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		info["memory"] = map[string]interface{}{
			"alloc":        memStats.Alloc,
			"total_alloc":  memStats.TotalAlloc,
			"sys":          memStats.Sys,
			"heap_alloc":   memStats.HeapAlloc,
			"heap_sys":     memStats.HeapSys,
			"heap_idle":    memStats.HeapIdle,
			"heap_inuse":   memStats.HeapInuse,
			"heap_objects": memStats.HeapObjects,
			"gc_cycles":    memStats.NumGC,
			"gc_pause_ns":  memStats.PauseNs,
		}
	}

	// Return as JSON string for consistent output
	jsonBytes, err := json.Marshal(info)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal system info: %w", err)
	}

	return string(jsonBytes), nil
}

// main function is required for plugin compilation but won't be used
func main() {
	// This is a plugin, main() won't be called
}
