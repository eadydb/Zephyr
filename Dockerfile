# Build stage - use Debian for consistent libc
FROM golang:1.23-bookworm AS builder

# Install build dependencies including GCC for CGO plugin building
RUN apt-get update && apt-get install -y \
    git \
    ca-certificates \
    make \
    bash \
    gcc \
    libc6-dev \
    binutils \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Set Go toolchain to auto to allow newer versions
ENV GOTOOLCHAIN=auto

# Copy go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build plugins first (requires CGO for plugin buildmode)
ENV CGO_ENABLED=1
RUN chmod +x scripts/build-plugins.sh && ./scripts/build-plugins.sh build

# Build the application (KEEP CGO ENABLED for plugin loading support)
# Remove static compilation flags to support dynamic plugin loading
ENV CGO_ENABLED=1
RUN GOOS=linux go build -o zephyr cmd/zephyr/main.go

# Runtime stage - use Debian for better CGO compatibility
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    tzdata \
    wget \
    libc6 \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN groupadd -g 1001 zephyr && \
    useradd -u 1001 -g zephyr -s /bin/sh -m zephyr

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/zephyr .
COPY --from=builder /app/scripts/config.yaml ./config.yaml

# Copy plugins
COPY --from=builder /app/plugins plugins/

# Create logs directory and set proper permissions
RUN mkdir -p logs && \
    chown -R zephyr:zephyr /app && \
    chmod -R 755 /app/plugins

# Switch to non-root user
USER zephyr

# Environment variables for container runtime
ENV ZEPHYR_LOG_LEVEL=info
ENV ZEPHYR_LOG_FORMAT=json

# Expose ports based on config.yaml
# SSE transport: 26841
# HTTP transport: 26842  
# Monitoring: 26843
EXPOSE 26841 26842 26843

# Health check using monitoring endpoint
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:26843/health || exit 1

# Default command - start server with config file
CMD ["./zephyr", "serve", "--config", "config.yaml"] 