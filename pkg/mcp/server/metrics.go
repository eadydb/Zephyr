package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"
)

// MetricsCollector handles server metrics collection
type MetricsCollector struct {
	mu sync.RWMutex

	// Server metrics
	startTime     time.Time
	requestCount  int64
	errorCount    int64
	toolCallCount map[string]int64

	// Performance metrics
	avgResponseTime time.Duration
	responseTimes   []time.Duration
	maxResponseTime time.Duration

	// System metrics
	memoryStats runtime.MemStats
	goroutines  int
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		startTime:     time.Now(),
		toolCallCount: make(map[string]int64),
		responseTimes: make([]time.Duration, 0, 1000), // Keep last 1000 response times
	}
}

// RecordRequest records a request with its response time
func (m *MetricsCollector) RecordRequest(duration time.Duration, toolName string, isError bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requestCount++
	if isError {
		m.errorCount++
	}

	if toolName != "" {
		m.toolCallCount[toolName]++
	}

	// Update response times
	m.responseTimes = append(m.responseTimes, duration)
	if len(m.responseTimes) > 1000 {
		m.responseTimes = m.responseTimes[1:] // Keep only last 1000
	}

	// Update max response time
	if duration > m.maxResponseTime {
		m.maxResponseTime = duration
	}

	// Calculate average response time
	var total time.Duration
	for _, rt := range m.responseTimes {
		total += rt
	}
	m.avgResponseTime = total / time.Duration(len(m.responseTimes))
}

// UpdateSystemMetrics updates system-level metrics
func (m *MetricsCollector) UpdateSystemMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(&m.memoryStats)
	m.goroutines = runtime.NumGoroutine()
}

// GetMetrics returns current metrics as a map
func (m *MetricsCollector) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uptime := time.Since(m.startTime)

	// Calculate error rate safely
	errorRate := 0.0
	requestsPerSec := 0.0

	if m.requestCount > 0 {
		errorRate = float64(m.errorCount) / float64(m.requestCount)
		requestsPerSec = float64(m.requestCount) / uptime.Seconds()
	}

	metrics := map[string]interface{}{
		"server": map[string]interface{}{
			"uptime_seconds":   uptime.Seconds(),
			"start_time":       m.startTime.Format(time.RFC3339),
			"request_count":    m.requestCount,
			"error_count":      m.errorCount,
			"error_rate":       errorRate,
			"requests_per_sec": requestsPerSec,
		},
		"performance": map[string]interface{}{
			"avg_response_time_ms": m.avgResponseTime.Milliseconds(),
			"max_response_time_ms": m.maxResponseTime.Milliseconds(),
			"total_requests":       len(m.responseTimes),
		},
		"tools": m.toolCallCount,
		"system": map[string]interface{}{
			"goroutines":      m.goroutines,
			"memory_alloc":    m.memoryStats.Alloc,
			"memory_sys":      m.memoryStats.Sys,
			"memory_heap":     m.memoryStats.HeapAlloc,
			"memory_heap_sys": m.memoryStats.HeapSys,
			"gc_cycles":       m.memoryStats.NumGC,
		},
	}

	return metrics
}

// ServeHTTP implements http.Handler for metrics endpoint
func (m *MetricsCollector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Update system metrics before serving
	m.UpdateSystemMetrics()

	w.Header().Set("Content-Type", "application/json")

	metrics := m.GetMetrics()
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		http.Error(w, "Failed to encode metrics", http.StatusInternalServerError)
		return
	}
}

// HealthCheck provides a simple health check endpoint
func (m *MetricsCollector) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.mu.RLock()
	uptime := time.Since(m.startTime)
	requestCount := m.requestCount
	errorCount := m.errorCount
	m.mu.RUnlock()

	// Simple health criteria
	healthy := true
	status := "healthy"

	// Check error rate (unhealthy if > 50% errors in last 100 requests)
	if requestCount > 100 && float64(errorCount)/float64(requestCount) > 0.5 {
		healthy = false
		status = "unhealthy - high error rate"
	}

	// Check if server has been running for at least 10 seconds
	if uptime < 10*time.Second {
		status = "starting"
	}

	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"status":    status,
		"healthy":   healthy,
		"uptime":    uptime.String(),
		"version":   "1.0.0",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	statusCode := http.StatusOK
	if !healthy {
		statusCode = http.StatusServiceUnavailable
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// StartMetricsServer starts an HTTP server for metrics and health endpoints
func (m *MetricsCollector) StartMetricsServer(ctx context.Context, addr string) error {
	mux := http.NewServeMux()

	// Existing endpoints
	mux.HandleFunc("/health", m.HealthCheck)
	mux.HandleFunc("/metrics", m.ServeHTTP)

	// New plugin management endpoints
	mux.HandleFunc("/plugins", m.pluginListHandler)
	mux.HandleFunc("/plugins/", m.pluginDetailHandler)
	mux.HandleFunc("/plugins/reload", m.pluginReloadHandler)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Metrics server error", "error", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}

// pluginListHandler returns the list of all plugins
func (mc *MetricsCollector) pluginListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// This would need to be injected from the plugin manager
	// For now, return empty list
	response := map[string]interface{}{
		"plugins": []map[string]interface{}{},
		"count":   0,
	}

	json.NewEncoder(w).Encode(response)
}

// pluginDetailHandler returns details about a specific plugin
func (mc *MetricsCollector) pluginDetailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract plugin name from URL path
	path := strings.TrimPrefix(r.URL.Path, "/plugins/")
	if path == "" {
		http.Error(w, "Plugin name required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// This would need plugin manager integration
	response := map[string]interface{}{
		"error": "Plugin not found: " + path,
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(response)
}

// pluginReloadHandler handles plugin reload requests
func (mc *MetricsCollector) pluginReloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		PluginName string `json:"plugin_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// This would need plugin manager integration
	response := map[string]interface{}{
		"success": false,
		"error":   "Plugin reload not implemented yet",
		"plugin":  request.PluginName,
	}

	json.NewEncoder(w).Encode(response)
}
