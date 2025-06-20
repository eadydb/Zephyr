package transport

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"sync"

	"github.com/mark3labs/mcp-go/server"
)

// STDIOAdapter implements TransportAdapter for STDIO transport
type STDIOAdapter struct {
	mcpServer   *server.MCPServer
	stdioServer *server.StdioServer
	mu          sync.RWMutex
	running     bool
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewSTDIOAdapter creates a new STDIO transport adapter
func NewSTDIOAdapter(mcpServer *server.MCPServer) *STDIOAdapter {
	stdioServer := server.NewStdioServer(mcpServer)

	// Configure error logging to stderr
	stdioServer.SetErrorLogger(log.New(os.Stderr, "[MCP-STDIO] ", log.LstdFlags))

	return &STDIOAdapter{
		mcpServer:   mcpServer,
		stdioServer: stdioServer,
	}
}

// Start begins the STDIO transport communication
func (s *STDIOAdapter) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("STDIO transport already running")
	}

	// Create cancellable context
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.running = true

	// Start STDIO server in background
	go func() {
		defer func() {
			s.mu.Lock()
			s.running = false
			s.mu.Unlock()
		}()

		err := s.stdioServer.Listen(s.ctx, os.Stdin, os.Stdout)
		if err != nil && err != context.Canceled {
			slog.Error("STDIO transport error", "error", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the STDIO transport
func (s *STDIOAdapter) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	if s.cancel != nil {
		s.cancel()
	}

	s.running = false
	return nil
}

// Name returns the transport protocol name
func (s *STDIOAdapter) Name() string {
	return "stdio"
}

// IsHealthy returns true if the transport is functioning properly
func (s *STDIOAdapter) IsHealthy() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.running && s.ctx != nil && s.ctx.Err() == nil
}
