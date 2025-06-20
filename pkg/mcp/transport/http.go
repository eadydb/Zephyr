package transport

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

// HTTPAdapter implements TransportAdapter for StreamableHTTP transport
type HTTPAdapter struct {
	mcpServer        *server.MCPServer
	streamableServer *server.StreamableHTTPServer
	httpServer       *http.Server
	config           HTTPConfig
	mu               sync.RWMutex
	running          bool
}

// HTTPConfig holds HTTP-specific configuration
type HTTPConfig struct {
	Host    string
	Port    int
	Timeout time.Duration
}

// NewHTTPAdapter creates a new StreamableHTTP transport adapter
func NewHTTPAdapter(mcpServer *server.MCPServer, config HTTPConfig) *HTTPAdapter {
	// Create StreamableHTTP server with configuration
	streamableServer := server.NewStreamableHTTPServer(mcpServer,
		server.WithEndpointPath("/mcp"),
	)

	return &HTTPAdapter{
		mcpServer:        mcpServer,
		streamableServer: streamableServer,
		config:           config,
	}
}

// Start begins the StreamableHTTP transport server
func (h *HTTPAdapter) Start(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		return fmt.Errorf("HTTP transport already running")
	}

	// Create HTTP server
	mux := http.NewServeMux()

	// Mount the streamable HTTP handler
	mux.Handle("/mcp", h.streamableServer)

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Add CORS support for web clients
	mux.HandleFunc("/", h.corsMiddleware(http.NotFoundHandler()).ServeHTTP)

	addr := fmt.Sprintf("%s:%d", h.config.Host, h.config.Port)
	h.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  h.config.Timeout,
		WriteTimeout: h.config.Timeout,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in background
	go func() {
		defer func() {
			h.mu.Lock()
			h.running = false
			h.mu.Unlock()
		}()

		slog.Info("Starting StreamableHTTP server", "address", addr)
		if err := h.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	h.running = true
	return nil
}

// Stop gracefully shuts down the HTTP transport
func (h *HTTPAdapter) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.running || h.httpServer == nil {
		return nil
	}

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := h.httpServer.Shutdown(shutdownCtx)
	h.running = false
	return err
}

// Name returns the transport protocol name
func (h *HTTPAdapter) Name() string {
	return "http"
}

// IsHealthy returns true if the transport is functioning properly
func (h *HTTPAdapter) IsHealthy() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.running && h.httpServer != nil
}

// corsMiddleware adds CORS headers for HTTP transport
func (h *HTTPAdapter) corsMiddleware(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Cache-Control")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Type")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler.ServeHTTP(w, r)
	}
}
