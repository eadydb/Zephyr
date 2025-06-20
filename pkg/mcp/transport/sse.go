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

// SSEAdapter implements TransportAdapter for Server-Sent Events transport
type SSEAdapter struct {
	mcpServer  *server.MCPServer
	sseServer  *server.SSEServer
	httpServer *http.Server
	config     SSEConfig
	mu         sync.RWMutex
	running    bool
}

// SSEConfig holds SSE-specific configuration
type SSEConfig struct {
	Host        string
	Port        int
	CORSEnabled bool
}

// NewSSEAdapter creates a new SSE transport adapter
func NewSSEAdapter(mcpServer *server.MCPServer, config SSEConfig) *SSEAdapter {
	// Create SSE server with configuration
	sseServer := server.NewSSEServer(mcpServer,
		server.WithSSEEndpoint("/sse"),
		server.WithMessageEndpoint("/message"),
		server.WithKeepAlive(true),
	)

	return &SSEAdapter{
		mcpServer: mcpServer,
		sseServer: sseServer,
		config:    config,
	}
}

// Start begins the SSE transport server
func (s *SSEAdapter) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("SSE transport already running")
	}

	// Create HTTP server
	mux := http.NewServeMux()

	// Configure CORS if enabled
	if s.config.CORSEnabled {
		mux.HandleFunc("/sse", s.corsMiddleware(s.sseServer.SSEHandler()))
		mux.HandleFunc("/message", s.corsMiddleware(s.sseServer.MessageHandler()))
	} else {
		mux.Handle("/sse", s.sseServer.SSEHandler())
		mux.Handle("/message", s.sseServer.MessageHandler())
	}

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start server in background
	go func() {
		defer func() {
			s.mu.Lock()
			s.running = false
			s.mu.Unlock()
		}()

		slog.Info("Starting SSE server", "address", addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("SSE server error", "error", err)
		}
	}()

	s.running = true
	return nil
}

// Stop gracefully shuts down the SSE transport
func (s *SSEAdapter) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running || s.httpServer == nil {
		return nil
	}

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := s.httpServer.Shutdown(shutdownCtx)
	s.running = false
	return err
}

// Name returns the transport protocol name
func (s *SSEAdapter) Name() string {
	return "sse"
}

// IsHealthy returns true if the transport is functioning properly
func (s *SSEAdapter) IsHealthy() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.running && s.httpServer != nil
}

// corsMiddleware adds CORS headers for SSE transport
func (s *SSEAdapter) corsMiddleware(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Cache-Control")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Type")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler.ServeHTTP(w, r)
	}
}
