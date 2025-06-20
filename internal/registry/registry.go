package registry

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/eadydb/zephyr/internal/config"
	mcpplugin "github.com/eadydb/zephyr/pkg/plugin"
)

// Registry implements ToolRegistry interface for managing MCP tool plugins
type Registry struct {
	config    *config.PluginsConfig
	tools     map[string]mcpplugin.MCPToolPlugin
	toolsLock sync.RWMutex

	// Discovery state
	discoveryEnabled bool
	scanInterval     time.Duration
	directories      []string
	stopDiscovery    chan struct{}
	discoveryRunning bool
	discoveryMutex   sync.Mutex
}

// NewRegistry creates a new tool registry instance
func NewRegistry(cfg *config.PluginsConfig) mcpplugin.ToolRegistry {
	return &Registry{
		config:           cfg,
		tools:            make(map[string]mcpplugin.MCPToolPlugin),
		discoveryEnabled: cfg.Discovery.Enabled,
		scanInterval:     cfg.Discovery.ScanInterval,
		directories:      cfg.Discovery.Directories,
		stopDiscovery:    make(chan struct{}),
	}
}

// RegisterTool registers a new MCP tool plugin
func (r *Registry) RegisterTool(tool mcpplugin.MCPToolPlugin) error {
	if tool == nil {
		return fmt.Errorf("tool cannot be nil")
	}

	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	r.toolsLock.Lock()
	defer r.toolsLock.Unlock()

	// Check if tool already exists
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool already registered: %s", name)
	}

	// Initialize the tool
	if err := tool.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize tool %s: %w", name, err)
	}

	r.tools[name] = tool
	slog.Info("Registered MCP tool", "name", name, "version", tool.Version(), "description", tool.Description())

	return nil
}

// UnregisterTool unregisters an MCP tool plugin
func (r *Registry) UnregisterTool(name string) error {
	r.toolsLock.Lock()
	defer r.toolsLock.Unlock()

	tool, exists := r.tools[name]
	if !exists {
		return fmt.Errorf("tool not found: %s", name)
	}

	// Cleanup the tool
	if err := tool.Cleanup(); err != nil {
		slog.Warn("Error cleaning up tool", "name", name, "error", err)
	}

	delete(r.tools, name)
	slog.Info("Unregistered MCP tool", "name", name)

	return nil
}

// GetTool retrieves an MCP tool plugin by name
func (r *Registry) GetTool(name string) (mcpplugin.MCPToolPlugin, error) {
	r.toolsLock.RLock()
	defer r.toolsLock.RUnlock()

	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	return tool, nil
}

// ListTools returns all registered MCP tool plugins
func (r *Registry) ListTools() []mcpplugin.MCPToolPlugin {
	r.toolsLock.RLock()
	defer r.toolsLock.RUnlock()

	tools := make([]mcpplugin.MCPToolPlugin, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}

	return tools
}

// DiscoverTools performs plugin discovery - now a no-op since we use dynamic plugin manager
func (r *Registry) DiscoverTools() error {
	slog.Info("Static tool discovery disabled - using dynamic plugin manager")
	return nil
}

// StartPeriodicDiscovery starts background plugin discovery
func (r *Registry) StartPeriodicDiscovery() error {
	r.discoveryMutex.Lock()
	defer r.discoveryMutex.Unlock()

	if !r.discoveryEnabled {
		return nil
	}

	if r.discoveryRunning {
		return fmt.Errorf("periodic discovery already running")
	}

	r.discoveryRunning = true
	go r.periodicDiscoveryLoop()

	slog.Info("Started periodic plugin discovery", "interval", r.scanInterval)
	return nil
}

// StopPeriodicDiscovery stops background plugin discovery
func (r *Registry) StopPeriodicDiscovery() error {
	r.discoveryMutex.Lock()
	defer r.discoveryMutex.Unlock()

	if !r.discoveryRunning {
		return nil
	}

	close(r.stopDiscovery)
	r.discoveryRunning = false
	r.stopDiscovery = make(chan struct{})

	slog.Info("Stopped periodic plugin discovery")
	return nil
}

// Shutdown gracefully shuts down the registry
func (r *Registry) Shutdown() error {
	// Stop periodic discovery
	if err := r.StopPeriodicDiscovery(); err != nil {
		slog.Error("Error stopping periodic discovery", "error", err)
	}

	// Cleanup all tools
	r.toolsLock.Lock()
	defer r.toolsLock.Unlock()

	for name, tool := range r.tools {
		if err := tool.Cleanup(); err != nil {
			slog.Error("Error cleaning up tool", "name", name, "error", err)
		}
	}

	r.tools = make(map[string]mcpplugin.MCPToolPlugin)
	slog.Info("Registry shutdown complete")

	return nil
}

// periodicDiscoveryLoop runs the periodic discovery in background
func (r *Registry) periodicDiscoveryLoop() {
	ticker := time.NewTicker(r.scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := r.DiscoverTools(); err != nil {
				slog.Error("Error during periodic discovery", "error", err)
			}
		case <-r.stopDiscovery:
			return
		}
	}
}

// getToolNames returns list of registered tool names
func (r *Registry) getToolNames() []string {
	r.toolsLock.RLock()
	defer r.toolsLock.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}
