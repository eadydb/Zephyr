package config

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ReloadCallback is called when configuration is reloaded
type ReloadCallback func(*Config) error

// Watcher monitors configuration file changes and triggers reloads
type Watcher struct {
	configPath string
	fsWatcher  *fsnotify.Watcher
	callbacks  []ReloadCallback
	logger     *slog.Logger

	// State management
	mu      sync.RWMutex
	config  *Config
	running bool
	stopCh  chan struct{}

	// Debouncing
	debounceDelay time.Duration
	lastReload    time.Time
}

// WatcherOptions holds configuration for the watcher
type WatcherOptions struct {
	DebounceDelay time.Duration
	Logger        *slog.Logger
}

// NewWatcher creates a new configuration file watcher
func NewWatcher(configPath string, opts *WatcherOptions) (*Watcher, error) {
	if opts == nil {
		opts = &WatcherOptions{}
	}

	// Set defaults
	if opts.DebounceDelay == 0 {
		opts.DebounceDelay = 1 * time.Second
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Load initial configuration
	config, err := Load(configPath)
	if err != nil {
		fsWatcher.Close()
		return nil, fmt.Errorf("failed to load initial configuration: %w", err)
	}

	w := &Watcher{
		configPath:    configPath,
		fsWatcher:     fsWatcher,
		callbacks:     make([]ReloadCallback, 0),
		logger:        opts.Logger,
		config:        config,
		stopCh:        make(chan struct{}),
		debounceDelay: opts.DebounceDelay,
	}

	return w, nil
}

// AddCallback registers a callback to be called when configuration reloads
func (w *Watcher) AddCallback(callback ReloadCallback) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.callbacks = append(w.callbacks, callback)
}

// GetConfig returns the current configuration (thread-safe)
func (w *Watcher) GetConfig() *Config {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.config
}

// Start begins watching the configuration file for changes
func (w *Watcher) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return fmt.Errorf("watcher is already running")
	}

	// Add the config file to the watcher
	absPath, err := filepath.Abs(w.configPath)
	if err != nil {
		w.mu.Unlock()
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if err := w.fsWatcher.Add(absPath); err != nil {
		w.mu.Unlock()
		return fmt.Errorf("failed to watch config file: %w", err)
	}

	w.running = true
	w.mu.Unlock()

	w.logger.Info("Started configuration file watcher", "file", absPath)

	// Start the watching goroutine
	go w.watchLoop(ctx)

	return nil
}

// Stop stops watching the configuration file
func (w *Watcher) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return nil
	}

	close(w.stopCh)
	w.running = false

	if err := w.fsWatcher.Close(); err != nil {
		return fmt.Errorf("failed to close file watcher: %w", err)
	}

	w.logger.Info("Stopped configuration file watcher")
	return nil
}

// ReloadNow manually triggers a configuration reload
func (w *Watcher) ReloadNow() error {
	w.logger.Info("Manual configuration reload triggered")
	return w.reloadConfig()
}

// watchLoop is the main event loop for file watching
func (w *Watcher) watchLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			w.logger.Debug("Configuration watcher stopped due to context cancellation")
			return

		case <-w.stopCh:
			w.logger.Debug("Configuration watcher stopped")
			return

		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				w.logger.Error("File watcher events channel closed")
				return
			}

			w.handleFileEvent(event)

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				w.logger.Error("File watcher errors channel closed")
				return
			}

			w.logger.Error("File watcher error", "error", err)
		}
	}
}

// handleFileEvent processes file system events
func (w *Watcher) handleFileEvent(event fsnotify.Event) {
	w.logger.Debug("File system event", "event", event.String())

	// Only handle write and create events
	if event.Op&fsnotify.Write != fsnotify.Write && event.Op&fsnotify.Create != fsnotify.Create {
		return
	}

	// Check if this is our config file
	eventPath, err := filepath.Abs(event.Name)
	if err != nil {
		w.logger.Error("Failed to resolve event file path", "error", err)
		return
	}

	configPath, err := filepath.Abs(w.configPath)
	if err != nil {
		w.logger.Error("Failed to resolve config file path", "error", err)
		return
	}

	if eventPath != configPath {
		return
	}

	// Debounce rapid file changes
	w.mu.RLock()
	if time.Since(w.lastReload) < w.debounceDelay {
		w.logger.Debug("Skipping config reload due to debouncing")
		w.mu.RUnlock()
		return
	}
	w.mu.RUnlock()

	// Trigger reload
	if err := w.reloadConfig(); err != nil {
		w.logger.Error("Failed to reload configuration", "error", err)
	}
}

// reloadConfig performs the actual configuration reload
func (w *Watcher) reloadConfig() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.logger.Info("Reloading configuration", "file", w.configPath)

	// Load new configuration
	newConfig, err := Load(w.configPath)
	if err != nil {
		return fmt.Errorf("failed to load new configuration: %w", err)
	}

	// Update current config
	// Note: We keep a reference to the old config for potential future rollback functionality
	w.config = newConfig
	w.lastReload = time.Now()

	// Call all registered callbacks
	var callbackErrors []error
	for i, callback := range w.callbacks {
		if err := callback(newConfig); err != nil {
			w.logger.Error("Configuration reload callback failed",
				"callback_index", i, "error", err)
			callbackErrors = append(callbackErrors, err)
		}
	}

	// If any callback failed, consider rolling back
	if len(callbackErrors) > 0 {
		w.logger.Warn("Some configuration reload callbacks failed, keeping new config but logging errors",
			"failed_callbacks", len(callbackErrors),
			"total_callbacks", len(w.callbacks))

		// Note: We don't rollback automatically as partial success might be acceptable
		// The calling application can decide what to do based on callback errors
	}

	w.logger.Info("Configuration reloaded successfully",
		"callbacks_executed", len(w.callbacks),
		"callback_errors", len(callbackErrors))

	return nil
}

// IsRunning returns whether the watcher is currently running
func (w *Watcher) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}
