package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"sync"
	"time"
)

// DynamicPlugin represents a dynamically loaded plugin
type DynamicPlugin interface {
	// Plugin identification
	Name() string
	Version() string
	Description() string

	// Plugin lifecycle
	Initialize() error
	Shutdown() error

	// MCP tool interface
	MCPToolDefinition() MCPTool
	Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
	InputSchema() map[string]interface{}
}

// PluginMetadata contains plugin metadata from plugin.json
type PluginMetadata struct {
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Description  string                 `json:"description"`
	Author       string                 `json:"author"`
	APIVersion   string                 `json:"api_version"`
	EntryPoint   string                 `json:"entry_point"`
	Dependencies []string               `json:"dependencies"`
	Permissions  []string               `json:"permissions"`
	ConfigSchema map[string]interface{} `json:"config_schema"`
}

// LoadedPlugin represents a loaded plugin with its metadata and instance
type LoadedPlugin struct {
	Metadata  PluginMetadata
	Plugin    DynamicPlugin
	Handle    *plugin.Plugin
	LoadedAt  time.Time
	Directory string
	Enabled   bool
}

// PluginManager manages dynamic loading and lifecycle of plugins
type PluginManager struct {
	mu          sync.RWMutex
	plugins     map[string]*LoadedPlugin // name -> plugin
	pluginPaths map[string]string        // name -> directory path
	registry    ToolRegistry             // existing registry for integration
	baseDir     string                   // plugins base directory
	discovered  map[string]PluginMetadata
	loaded      map[string]*DynamicPluginAdapter
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(baseDir string, registry ToolRegistry) *PluginManager {
	return &PluginManager{
		plugins:     make(map[string]*LoadedPlugin),
		pluginPaths: make(map[string]string),
		registry:    registry,
		baseDir:     baseDir,
		discovered:  make(map[string]PluginMetadata),
		loaded:      make(map[string]*DynamicPluginAdapter),
	}
}

// DiscoverPlugins scans the plugins directory for available plugins
func (pm *PluginManager) DiscoverPlugins() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Create base directory if it doesn't exist
	if err := os.MkdirAll(pm.baseDir, 0o755); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}

	// Scan for plugin directories
	entries, err := os.ReadDir(pm.baseDir)
	if err != nil {
		return fmt.Errorf("failed to read plugins directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginDir := filepath.Join(pm.baseDir, entry.Name())
		metadataPath := filepath.Join(pluginDir, "plugin.json")

		// Check if plugin.json exists
		if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
			continue
		}

		// Load metadata
		metadata, err := pm.loadMetadata(metadataPath)
		if err != nil {
			slog.Warn("Failed to load metadata for plugin", "plugin", entry.Name(), "error", err)
			continue
		}

		pm.pluginPaths[metadata.Name] = pluginDir
		pm.discovered[metadata.Name] = metadata
		slog.Info("Discovered plugin", "name", metadata.Name, "version", metadata.Version, "path", pluginDir)
	}

	return nil
}

// LoadPlugin loads a specific plugin by name
func (pm *PluginManager) LoadPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pluginInfo, exists := pm.discovered[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Check if already loaded
	if pm.loaded[name] != nil {
		return fmt.Errorf("plugin %s already loaded", name)
	}

	// Get plugin directory path
	pluginDir, exists := pm.pluginPaths[name]
	if !exists {
		return fmt.Errorf("plugin directory for %s not found", name)
	}

	// Open the plugin file
	p, err := plugin.Open(filepath.Join(pluginDir, name+".so"))
	if err != nil {
		return fmt.Errorf("failed to open plugin %s: %v", name, err)
	}

	// Look up the DynamicPlugin symbol
	sym, err := p.Lookup("Plugin")
	if err != nil {
		return fmt.Errorf("failed to find Plugin symbol in %s: %v", name, err)
	}

	// Try to assert as pointer to DynamicPlugin first
	var dynamicPlugin DynamicPlugin
	if pluginPtr, ok := sym.(*DynamicPlugin); ok && pluginPtr != nil {
		dynamicPlugin = *pluginPtr
	} else if directPlugin, ok := sym.(DynamicPlugin); ok {
		dynamicPlugin = directPlugin
	} else {
		return fmt.Errorf("plugin %s does not implement DynamicPlugin interface (got %T)", name, sym)
	}

	// Initialize the plugin
	if err := dynamicPlugin.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize plugin %s: %v", name, err)
	}

	// Create adapter and register with registry
	adapter := &DynamicPluginAdapter{
		plugin:   dynamicPlugin,
		metadata: pluginInfo,
	}

	// Register with tool registry if provided
	if pm.registry != nil {
		if err := pm.registry.RegisterTool(adapter); err != nil {
			// Clean up: shutdown the plugin since registration failed
			dynamicPlugin.Shutdown()
			return fmt.Errorf("failed to register plugin %s with registry: %v", name, err)
		}
		slog.Info("Registered MCP tool", "name", name, "version", pluginInfo.Version, "description", pluginInfo.Description)
	}

	// Store the loaded plugin
	pm.loaded[name] = adapter
	slog.Info("Successfully loaded plugin", "name", name, "version", pluginInfo.Version)

	return nil
}

// UnloadPlugin unloads a specific plugin by name
func (pm *PluginManager) UnloadPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	loadedPlugin, exists := pm.loaded[name]
	if !exists {
		return fmt.Errorf("plugin %s not loaded", name)
	}

	// Unregister from tool registry first
	if pm.registry != nil {
		if err := pm.registry.UnregisterTool(name); err != nil {
			slog.Warn("Failed to unregister plugin from registry", "plugin", name, "error", err)
		} else {
			slog.Debug("Plugin unregistered from registry", "plugin", name)
		}
	}

	// Shutdown the plugin
	if err := loadedPlugin.plugin.Shutdown(); err != nil {
		return fmt.Errorf("failed to shutdown plugin %s: %v", name, err)
	}

	// Remove from loaded plugins
	delete(pm.loaded, name)
	slog.Info("Successfully unloaded plugin", "plugin", name)

	return nil
}

// ReloadPlugin reloads a plugin (unload then load)
func (pm *PluginManager) ReloadPlugin(name string) error {
	// Check if plugin is loaded
	pm.mu.RLock()
	_, isLoaded := pm.plugins[name]
	pm.mu.RUnlock()

	if isLoaded {
		if err := pm.UnloadPlugin(name); err != nil {
			return fmt.Errorf("failed to unload plugin for reload: %w", err)
		}
	}

	return pm.LoadPlugin(name)
}

// ListPlugins returns information about all discovered and loaded plugins
func (pm *PluginManager) ListPlugins() map[string]PluginStatus {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make(map[string]PluginStatus)

	// Add all discovered plugins
	for name, path := range pm.pluginPaths {
		status := PluginStatus{
			Name:       name,
			Directory:  path,
			Discovered: true,
			Loaded:     false,
		}

		if loadedPlugin, exists := pm.plugins[name]; exists {
			status.Loaded = true
			status.Enabled = loadedPlugin.Enabled
			status.LoadedAt = loadedPlugin.LoadedAt
			status.Version = loadedPlugin.Metadata.Version
			status.Description = loadedPlugin.Metadata.Description
		}

		result[name] = status
	}

	return result
}

// GetPlugin returns a loaded plugin by name
func (pm *PluginManager) GetPlugin(name string) (*LoadedPlugin, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, exists := pm.plugins[name]
	return plugin, exists
}

// LoadAllPlugins loads all discovered plugins
func (pm *PluginManager) LoadAllPlugins() error {
	var errors []string

	for name := range pm.discovered {
		if err := pm.LoadPlugin(name); err != nil {
			errors = append(errors, fmt.Sprintf("plugin %s: %v", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to load some plugins: %s", strings.Join(errors, "; "))
	}

	return nil
}

// PluginStatus represents the status of a plugin
type PluginStatus struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	Directory   string    `json:"directory"`
	Discovered  bool      `json:"discovered"`
	Loaded      bool      `json:"loaded"`
	Enabled     bool      `json:"enabled"`
	LoadedAt    time.Time `json:"loaded_at,omitempty"`
}

// loadMetadata loads plugin metadata from plugin.json
func (pm *PluginManager) loadMetadata(path string) (PluginMetadata, error) {
	var metadata PluginMetadata

	data, err := os.ReadFile(path)
	if err != nil {
		return metadata, err
	}

	if err := json.Unmarshal(data, &metadata); err != nil {
		return metadata, err
	}

	// Validate required fields
	if metadata.Name == "" {
		return metadata, fmt.Errorf("plugin name is required")
	}
	if metadata.Version == "" {
		return metadata, fmt.Errorf("plugin version is required")
	}
	if metadata.EntryPoint == "" {
		return metadata, fmt.Errorf("plugin entry_point is required")
	}

	return metadata, nil
}

// DynamicPluginAdapter adapts DynamicPlugin to MCPToolPlugin interface
type DynamicPluginAdapter struct {
	plugin   DynamicPlugin
	metadata PluginMetadata
}

func (dpa *DynamicPluginAdapter) Name() string {
	return dpa.plugin.Name()
}

func (dpa *DynamicPluginAdapter) Version() string {
	return dpa.plugin.Version()
}

func (dpa *DynamicPluginAdapter) Description() string {
	return dpa.plugin.Description()
}

func (dpa *DynamicPluginAdapter) MCPToolDefinition() MCPTool {
	return dpa.plugin.MCPToolDefinition()
}

func (dpa *DynamicPluginAdapter) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return dpa.plugin.Execute(ctx, args)
}

func (dpa *DynamicPluginAdapter) InputSchema() map[string]interface{} {
	return dpa.plugin.InputSchema()
}

func (dpa *DynamicPluginAdapter) Initialize() error {
	// Plugin is already initialized during loading, so this is a no-op
	return nil
}

func (dpa *DynamicPluginAdapter) Cleanup() error {
	return dpa.plugin.Shutdown()
}
