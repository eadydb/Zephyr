package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/eadydb/zephyr/pkg/plugin"
)

// Plugin is the exported plugin instance
var Plugin plugin.DynamicPlugin = &CurrentTimePlugin{}

// CurrentTimePlugin implements the DynamicPlugin interface
type CurrentTimePlugin struct {
	initialized bool
}

// NewPlugin is the factory function that will be called by the plugin loader
func NewPlugin() plugin.DynamicPlugin {
	return &CurrentTimePlugin{}
}

// Name returns the plugin name
func (p *CurrentTimePlugin) Name() string {
	return "currenttime"
}

// Version returns the plugin version
func (p *CurrentTimePlugin) Version() string {
	return "1.0.0"
}

// Description returns the plugin description
func (p *CurrentTimePlugin) Description() string {
	return "Provides current time information in various formats and timezones"
}

// Initialize initializes the plugin
func (p *CurrentTimePlugin) Initialize() error {
	if p.initialized {
		return fmt.Errorf("plugin already initialized")
	}
	p.initialized = true
	return nil
}

// Shutdown cleans up the plugin
func (p *CurrentTimePlugin) Shutdown() error {
	p.initialized = false
	return nil
}

// MCPToolDefinition returns the MCP tool definition
func (p *CurrentTimePlugin) MCPToolDefinition() plugin.MCPTool {
	return plugin.MCPTool{
		Name:        "currenttime",
		Description: "Get current time in various formats and timezones",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"timezone": map[string]interface{}{
					"type":        "string",
					"description": "Timezone (e.g., 'UTC', 'America/New_York', 'Asia/Tokyo')",
					"default":     "UTC",
				},
				"format": map[string]interface{}{
					"type":        "string",
					"description": "Time format ('rfc3339', 'unix', 'kitchen', 'stamp')",
					"default":     "rfc3339",
				},
				"include_utc": map[string]interface{}{
					"type":        "boolean",
					"description": "Include UTC time in response",
					"default":     true,
				},
			},
		},
	}
}

// InputSchema returns the input schema for the tool
func (p *CurrentTimePlugin) InputSchema() map[string]interface{} {
	return p.MCPToolDefinition().InputSchema
}

// Execute executes the tool with the given arguments
func (p *CurrentTimePlugin) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	if !p.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	// Parse arguments
	timezone := "UTC"
	format := "rfc3339"
	includeUTC := true

	if tz, exists := args["timezone"]; exists {
		if t, ok := tz.(string); ok {
			timezone = t
		}
	}

	if fmt, exists := args["format"]; exists {
		if f, ok := fmt.(string); ok {
			format = f
		}
	}

	if inc, exists := args["include_utc"]; exists {
		if i, ok := inc.(bool); ok {
			includeUTC = i
		}
	}

	// Get current time
	now := time.Now()

	// Load timezone
	var loc *time.Location
	var err error
	if timezone == "UTC" {
		loc = time.UTC
	} else {
		loc, err = time.LoadLocation(timezone)
		if err != nil {
			return nil, fmt.Errorf("invalid timezone %s: %w", timezone, err)
		}
	}

	// Convert to target timezone
	localTime := now.In(loc)

	// Format time based on requested format
	var formattedTime string
	switch format {
	case "rfc3339":
		formattedTime = localTime.Format(time.RFC3339)
	case "unix":
		formattedTime = fmt.Sprintf("%d", localTime.Unix())
	case "kitchen":
		formattedTime = localTime.Format(time.Kitchen)
	case "stamp":
		formattedTime = localTime.Format(time.Stamp)
	default:
		// Try as custom format
		formattedTime = localTime.Format(format)
	}

	// Build response
	result := map[string]interface{}{
		"timezone":       timezone,
		"time":           formattedTime,
		"format":         format,
		"unix_timestamp": localTime.Unix(),
	}

	// Include UTC time if requested
	if includeUTC && timezone != "UTC" {
		utcTime := now.UTC()
		var utcFormatted string
		switch format {
		case "rfc3339":
			utcFormatted = utcTime.Format(time.RFC3339)
		case "unix":
			utcFormatted = fmt.Sprintf("%d", utcTime.Unix())
		case "kitchen":
			utcFormatted = utcTime.Format(time.Kitchen)
		case "stamp":
			utcFormatted = utcTime.Format(time.Stamp)
		default:
			utcFormatted = utcTime.Format(format)
		}

		result["utc"] = map[string]interface{}{
			"time":           utcFormatted,
			"unix_timestamp": utcTime.Unix(),
		}
	}

	// Return as JSON string for consistent output
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal time info: %w", err)
	}

	return string(jsonBytes), nil
}

// main function is required for plugin compilation but won't be used
func main() {
	// This is a plugin, main() won't be called
}
