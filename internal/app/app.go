package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/eadydb/zephyr/internal/config"
	"github.com/eadydb/zephyr/internal/registry"
	"github.com/eadydb/zephyr/pkg/mcp/server"
	"github.com/eadydb/zephyr/pkg/mcp/transport"
	"github.com/eadydb/zephyr/pkg/plugin"
)

// App represents the main application
type App struct {
	name    string
	version string
	config  *config.Config
	logger  *slog.Logger

	// Core components
	metrics       *server.MetricsCollector
	registry      plugin.ToolRegistry
	pluginManager *plugin.PluginManager
	mcpServer     *server.Server
	transport     transport.TransportAdapter

	// Configuration management
	configPath    string
	configWatcher *config.Watcher

	// Runtime context
	ctx    context.Context
	cancel context.CancelFunc
}

// AppOptions holds optional configuration for the app
type AppOptions struct {
	ConfigPath      string
	LogLevel        string
	LogFormat       string
	EnableHotReload bool
}

// New creates a new application instance
func New(name, version string, opts *AppOptions) (*App, error) {
	app := &App{
		name:    name,
		version: version,
	}

	if err := app.initialize(opts); err != nil {
		return nil, fmt.Errorf("failed to initialize app: %w", err)
	}

	return app, nil
}

// initialize sets up all application components
func (a *App) initialize(opts *AppOptions) error {
	// Setup context
	a.ctx, a.cancel = context.WithCancel(context.Background())

	// Setup logging
	if err := a.setupLogging(opts); err != nil {
		return fmt.Errorf("failed to setup logging: %w", err)
	}

	// Load configuration
	if err := a.loadConfig(opts); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Setup configuration hot reload if enabled
	if opts != nil && opts.EnableHotReload {
		if err := a.setupConfigWatcher(); err != nil {
			a.logger.Warn("Failed to setup config hot reload", "error", err)
		}
	}

	// Initialize core components
	if err := a.initializeComponents(); err != nil {
		return fmt.Errorf("failed to initialize components: %w", err)
	}

	return nil
}

// setupLogging configures structured logging
func (a *App) setupLogging(opts *AppOptions) error {
	logLevel := slog.LevelInfo
	if opts != nil && opts.LogLevel != "" {
		switch opts.LogLevel {
		case "debug":
			logLevel = slog.LevelDebug
		case "info":
			logLevel = slog.LevelInfo
		case "warn":
			logLevel = slog.LevelWarn
		case "error":
			logLevel = slog.LevelError
		}
	}

	var handler slog.Handler
	if opts != nil && opts.LogFormat == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: logLevel,
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: logLevel,
		})
	}

	a.logger = slog.New(handler)
	slog.SetDefault(a.logger)

	return nil
}

// loadConfig loads application configuration
func (a *App) loadConfig(opts *AppOptions) error {
	configPath := "config.yaml"
	if opts != nil && opts.ConfigPath != "" {
		configPath = opts.ConfigPath
	}

	a.configPath = configPath

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	a.config = cfg
	return nil
}

// setupConfigWatcher initializes the configuration file watcher
func (a *App) setupConfigWatcher() error {
	watcher, err := config.NewWatcher(a.configPath, &config.WatcherOptions{
		Logger: a.logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create config watcher: %w", err)
	}

	// Register reload callback
	watcher.AddCallback(a.onConfigReload)

	a.configWatcher = watcher
	a.logger.Info("Configuration hot reload enabled", "config_file", a.configPath)

	return nil
}

// onConfigReload is called when configuration is reloaded
func (a *App) onConfigReload(newConfig *config.Config) error {
	a.logger.Info("Processing configuration reload")

	// Update app config reference
	a.config = newConfig

	// TODO: Implement selective component updates based on config changes
	// For now, we just log the reload and update the config reference
	// In the future, we could:
	// 1. Compare old vs new config to determine what changed
	// 2. Selectively update only affected components
	// 3. Handle cases where certain changes require restart

	a.logger.Info("Configuration reload completed successfully")
	return nil
}

// initializeComponents initializes all application components
func (a *App) initializeComponents() error {
	a.logger.Info("Initializing application components",
		"name", a.name,
		"version", a.version)

	// Create metrics collector
	a.metrics = server.NewMetricsCollector()

	// Create registry
	a.registry = registry.NewRegistry(&a.config.Plugins)

	// Create and setup plugin manager
	a.pluginManager = plugin.NewPluginManager("./plugins", a.registry)
	if err := a.setupPlugins(); err != nil {
		return fmt.Errorf("failed to setup plugins: %w", err)
	}

	// Create MCP server
	a.mcpServer = server.NewWithMetrics(a.name, a.version, a.registry, a.metrics)
	if err := a.mcpServer.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	// Create transport
	transportAdapter, err := transport.CreateTransportFromFullConfig(a.config, a.mcpServer.GetMCPServer())
	if err != nil {
		return fmt.Errorf("failed to create transport: %w", err)
	}
	a.transport = transportAdapter

	return nil
}

// setupPlugins handles plugin discovery and loading
func (a *App) setupPlugins() error {
	a.logger.Info("Starting plugin discovery", "directories", []string{"./plugins"})

	if err := a.pluginManager.DiscoverPlugins(); err != nil {
		a.logger.Error("Failed to discover plugins", "error", err)
		return err
	}

	if err := a.pluginManager.LoadAllPlugins(); err != nil {
		a.logger.Warn("Some plugins failed to load", "error", err)
	}

	// Log plugin status
	pluginStatus := a.pluginManager.ListPlugins()
	var loadedPlugins []string
	for name, status := range pluginStatus {
		if status.Loaded {
			loadedPlugins = append(loadedPlugins, name)
		}
	}

	a.logger.Info("Plugin discovery completed",
		"tool_count", len(loadedPlugins),
		"tools", loadedPlugins)

	return nil
}

// Run starts the application and blocks until shutdown
func (a *App) Run() error {
	a.logger.Info("Starting application", "name", a.name, "version", a.version)

	// Start configuration watcher if enabled
	if a.configWatcher != nil {
		if err := a.configWatcher.Start(a.ctx); err != nil {
			a.logger.Warn("Failed to start config watcher", "error", err)
		}
	}

	// Start monitoring server if enabled
	if a.config.Monitoring.Enabled {
		go a.startMonitoring()
	}

	// Start transport
	if err := a.transport.Start(a.ctx); err != nil {
		return fmt.Errorf("failed to start transport: %w", err)
	}

	// Setup graceful shutdown
	return a.waitForShutdown()
}

// startMonitoring starts the monitoring server
func (a *App) startMonitoring() {
	monitoringAddr := fmt.Sprintf("%s:%d", a.config.Monitoring.Host, a.config.Monitoring.Port)
	a.logger.Info("Starting monitoring server", "address", monitoringAddr)

	if err := a.metrics.StartMetricsServer(a.ctx, monitoringAddr); err != nil {
		a.logger.Error("Monitoring server error", "error", err)
	}
}

// waitForShutdown waits for shutdown signal and performs graceful shutdown
func (a *App) waitForShutdown() error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	a.logger.Info("Application is running. Press Ctrl+C to stop.")

	// Wait for shutdown signal
	sig := <-sigChan
	a.logger.Info("Received shutdown signal", "signal", sig)

	return a.Shutdown()
}

// Shutdown performs graceful shutdown of all components
func (a *App) Shutdown() error {
	a.logger.Info("Shutting down application...")

	var shutdownErrors []error

	// Cancel context for monitoring and other goroutines
	a.cancel()

	// Stop configuration watcher
	if a.configWatcher != nil {
		if err := a.configWatcher.Stop(); err != nil {
			a.logger.Error("Error stopping config watcher", "error", err)
			shutdownErrors = append(shutdownErrors, err)
		}
	}

	// Stop transport
	if a.transport != nil {
		if err := a.transport.Stop(); err != nil {
			a.logger.Error("Error stopping transport", "error", err)
			shutdownErrors = append(shutdownErrors, err)
		}
	}

	// Unload all plugins gracefully
	if a.pluginManager != nil {
		pluginStatus := a.pluginManager.ListPlugins()
		for name := range pluginStatus {
			if err := a.pluginManager.UnloadPlugin(name); err != nil {
				a.logger.Error("Error unloading plugin", "plugin", name, "error", err)
				shutdownErrors = append(shutdownErrors, err)
			}
		}
	}

	// Stop MCP server
	if a.mcpServer != nil {
		if err := a.mcpServer.Stop(); err != nil {
			a.logger.Error("Error stopping MCP server", "error", err)
			shutdownErrors = append(shutdownErrors, err)
		}
	}

	if len(shutdownErrors) > 0 {
		a.logger.Error("Shutdown completed with errors", "error_count", len(shutdownErrors))
		return fmt.Errorf("shutdown had %d errors", len(shutdownErrors))
	}

	a.logger.Info("Shutdown complete")
	return nil
}

// ReloadConfig manually triggers a configuration reload
func (a *App) ReloadConfig() error {
	if a.configWatcher == nil {
		return fmt.Errorf("configuration hot reload is not enabled")
	}

	return a.configWatcher.ReloadNow()
}

// GetConfig returns the application configuration
func (a *App) GetConfig() *config.Config {
	if a.configWatcher != nil {
		return a.configWatcher.GetConfig()
	}
	return a.config
}

// GetLogger returns the application logger
func (a *App) GetLogger() *slog.Logger {
	return a.logger
}
