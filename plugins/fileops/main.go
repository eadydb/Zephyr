package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eadydb/zephyr/pkg/plugin"
)

// Plugin is the exported plugin instance
var Plugin plugin.DynamicPlugin = &FileOpsPlugin{}

// FileOpsPlugin implements the DynamicPlugin interface
type FileOpsPlugin struct {
	initialized bool
	maxFileSize int64 // Maximum file size to read (in bytes)
}

// NewPlugin is the factory function that will be called by the plugin loader
func NewPlugin() plugin.DynamicPlugin {
	return &FileOpsPlugin{
		maxFileSize: 10 * 1024 * 1024, // 10MB default limit
	}
}

// Name returns the plugin name
func (p *FileOpsPlugin) Name() string {
	return "fileops"
}

// Version returns the plugin version
func (p *FileOpsPlugin) Version() string {
	return "1.0.0"
}

// Description returns the plugin description
func (p *FileOpsPlugin) Description() string {
	return "Provides file system operations including read, write, list, and metadata operations"
}

// Initialize initializes the plugin
func (p *FileOpsPlugin) Initialize() error {
	if p.initialized {
		return fmt.Errorf("plugin already initialized")
	}
	p.initialized = true
	return nil
}

// Shutdown cleans up the plugin
func (p *FileOpsPlugin) Shutdown() error {
	p.initialized = false
	return nil
}

// MCPToolDefinition returns the MCP tool definition
func (p *FileOpsPlugin) MCPToolDefinition() plugin.MCPTool {
	return plugin.MCPTool{
		Name:        "fileops",
		Description: "File system operations: read, write, list, stat, exists",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"operation": map[string]interface{}{
					"type":        "string",
					"description": "File operation: 'read', 'write', 'list', 'stat', 'exists'",
					"enum":        []string{"read", "write", "list", "stat", "exists"},
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "File or directory path",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Content to write (for write operation)",
				},
				"encoding": map[string]interface{}{
					"type":        "string",
					"description": "Encoding for content: 'utf8' or 'base64'",
					"default":     "utf8",
				},
				"create_dirs": map[string]interface{}{
					"type":        "boolean",
					"description": "Create parent directories if they don't exist (for write operation)",
					"default":     false,
				},
			},
			"required": []string{"operation", "path"},
		},
	}
}

// InputSchema returns the input schema for the tool
func (p *FileOpsPlugin) InputSchema() map[string]interface{} {
	return p.MCPToolDefinition().InputSchema
}

// Execute executes the tool with the given arguments
func (p *FileOpsPlugin) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	if !p.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	// Parse operation
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation parameter is required and must be a string")
	}

	// Parse path
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter is required and must be a string")
	}

	// Validate and clean path
	cleanPath, err := p.validatePath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Execute operation
	switch operation {
	case "read":
		return p.readFile(cleanPath, args)
	case "write":
		return p.writeFile(cleanPath, args)
	case "list":
		return p.listDirectory(cleanPath)
	case "stat":
		return p.statFile(cleanPath)
	case "exists":
		return p.fileExists(cleanPath)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// validatePath validates and cleans the file path
func (p *FileOpsPlugin) validatePath(path string) (string, error) {
	// Clean the path
	cleanPath := filepath.Clean(path)

	// Check for directory traversal attempts
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("directory traversal not allowed")
	}

	// Convert to absolute path for consistency
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	return absPath, nil
}

// readFile reads a file and returns its content
func (p *FileOpsPlugin) readFile(path string, args map[string]interface{}) (interface{}, error) {
	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Check if it's a file
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file: %s", path)
	}

	// Check file size
	if info.Size() > p.maxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes (max: %d bytes)", info.Size(), p.maxFileSize)
	}

	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse encoding
	encoding := "utf8"
	if enc, exists := args["encoding"]; exists {
		if e, ok := enc.(string); ok {
			encoding = e
		}
	}

	// Prepare result
	result := map[string]interface{}{
		"operation": "read",
		"path":      path,
		"size":      info.Size(),
		"encoding":  encoding,
	}

	// Encode content based on requested encoding
	switch encoding {
	case "utf8":
		result["content"] = string(content)
	case "base64":
		result["content"] = base64.StdEncoding.EncodeToString(content)
	default:
		return nil, fmt.Errorf("unsupported encoding: %s", encoding)
	}

	return p.jsonResponse(result)
}

// writeFile writes content to a file
func (p *FileOpsPlugin) writeFile(path string, args map[string]interface{}) (interface{}, error) {
	// Parse content
	content, ok := args["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content parameter is required for write operation")
	}

	// Parse encoding
	encoding := "utf8"
	if enc, exists := args["encoding"]; exists {
		if e, ok := enc.(string); ok {
			encoding = e
		}
	}

	// Parse create_dirs flag
	createDirs := false
	if cd, exists := args["create_dirs"]; exists {
		if c, ok := cd.(bool); ok {
			createDirs = c
		}
	}

	// Decode content based on encoding
	var data []byte
	var err error
	switch encoding {
	case "utf8":
		data = []byte(content)
	case "base64":
		data, err = base64.StdEncoding.DecodeString(content)
		if err != nil {
			return nil, fmt.Errorf("invalid base64 content: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported encoding: %s", encoding)
	}

	// Create parent directories if requested
	if createDirs {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create directories: %w", err)
		}
	}

	// Write file
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	result := map[string]interface{}{
		"operation":   "write",
		"path":        path,
		"size":        len(data),
		"encoding":    encoding,
		"create_dirs": createDirs,
	}

	return p.jsonResponse(result)
}

// listDirectory lists directory contents
func (p *FileOpsPlugin) listDirectory(path string) (interface{}, error) {
	// Check if directory exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("directory not found: %s", path)
		}
		return nil, fmt.Errorf("failed to stat directory: %w", err)
	}

	// Check if it's a directory
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", path)
	}

	// Read directory
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// Build result
	var files []map[string]interface{}
	for _, entry := range entries {
		fileInfo, err := entry.Info()
		if err != nil {
			continue // Skip entries with errors
		}

		files = append(files, map[string]interface{}{
			"name":    entry.Name(),
			"type":    p.getFileType(entry),
			"size":    fileInfo.Size(),
			"mode":    fileInfo.Mode().String(),
			"modtime": fileInfo.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	result := map[string]interface{}{
		"operation": "list",
		"path":      path,
		"count":     len(files),
		"files":     files,
	}

	return p.jsonResponse(result)
}

// statFile gets file/directory metadata
func (p *FileOpsPlugin) statFile(path string) (interface{}, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	result := map[string]interface{}{
		"operation": "stat",
		"path":      path,
		"name":      info.Name(),
		"size":      info.Size(),
		"mode":      info.Mode().String(),
		"modtime":   info.ModTime().Format("2006-01-02 15:04:05"),
		"is_dir":    info.IsDir(),
	}

	return p.jsonResponse(result)
}

// fileExists checks if a file/directory exists
func (p *FileOpsPlugin) fileExists(path string) (interface{}, error) {
	_, err := os.Stat(path)
	exists := err == nil

	result := map[string]interface{}{
		"operation": "exists",
		"path":      path,
		"exists":    exists,
	}

	if err != nil && !os.IsNotExist(err) {
		result["error"] = err.Error()
	}

	return p.jsonResponse(result)
}

// getFileType determines the file type from directory entry
func (p *FileOpsPlugin) getFileType(entry os.DirEntry) string {
	if entry.IsDir() {
		return "directory"
	}
	return "file"
}

// jsonResponse converts result to JSON string
func (p *FileOpsPlugin) jsonResponse(result map[string]interface{}) (interface{}, error) {
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(jsonBytes), nil
}

// main function is required for plugin compilation but won't be used
func main() {
	// This is a plugin, main() won't be called
}
