📌 CREATIVE PHASE START: Configuration Schema Design
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

## 1️⃣ PROBLEM
**Description**: Design a flexible configuration schema for MCP server supporting multiple transport protocols, plugin discovery, and runtime configuration

**Requirements**:
- Support configuration for 3 transport protocols (STDIO, SSE, HTTP)
- Plugin discovery and loading configuration
- Server-level settings (logging, timeouts, security)
- Environment variable override support
- Development vs production configuration profiles
- Hot-reload capability for development
- Validation and sensible defaults

**Constraints**:
- Use YAML as primary configuration format
- Leverage existing zephyr configuration patterns
- Keep configuration hierarchy simple and intuitive
- Support both file-based and environment-based config
- Validate configuration at startup

## 2️⃣ OPTIONS

**Option A: Flat Configuration** - Single-level configuration with prefixed keys
**Option B: Hierarchical YAML** - Nested configuration sections by component
**Option C: Profile-Based Config** - Environment-specific configuration profiles

## 3️⃣ ANALYSIS

| Criterion | Flat Configuration | Hierarchical YAML | Profile-Based |
|-----------|-------------------|-------------------|---------------|
| **Simplicity** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ |
| **Organization** | ⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| **Environment Support** | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Maintainability** | ⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Validation** | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| **Developer Experience** | ⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |

**Key Insights**:
- Flat configuration becomes unwieldy with multiple components
- Hierarchical YAML provides best organization and readability
- Profile-Based offers excellent environment management but adds complexity
- YAML validation requires structured schema
- Environment overrides work well with hierarchical keys (SERVER_TRANSPORT_PROTOCOL)

## 4️⃣ DECISION
**Selected**: Option B: Hierarchical YAML with Environment Overrides
**Rationale**: Best balance of organization, simplicity, and developer experience with clear component separation

## 5️⃣ IMPLEMENTATION NOTES

### Configuration Structure
```yaml
# config.yaml
server:
  name: "zephyr-mcp-server"
  version: "1.0.0"
  debug: false
  
transport:
  protocol: "stdio"  # stdio|sse|http
  stdio:
    buffer_size: 4096
  sse:
    port: 26841
    host: "localhost"
    cors_enabled: true
  http:
    port: 26842
    host: "localhost"
    timeout: "30s"
    
plugins:
  discovery:
    enabled: true
    directories: ["./plugins", "/usr/local/lib/zephyr/plugins"]
    scan_interval: "60s"
  tools:
    systeminfo:
      enabled: true
    currenttime:
      enabled: true
      timezone: "UTC"
      
logging:
  level: "info"  # debug|info|warn|error
  format: "json"  # json|text
  output: "stdout"  # stdout|stderr|file
  file: "/var/log/zephyr-mcp.log"
  
security:
  rate_limit:
    enabled: true
    requests_per_minute: 100
  timeout:
    request: "10s"
    shutdown: "30s"
```

### Environment Variable Mapping
- `ZEPHYR_SERVER_DEBUG` → `server.debug`
- `ZEPHYR_TRANSPORT_PROTOCOL` → `transport.protocol`
- `ZEPHYR_TRANSPORT_SSE_PORT` → `transport.sse.port`
- `ZEPHYR_PLUGINS_DISCOVERY_ENABLED` → `plugins.discovery.enabled`
- `ZEPHYR_LOGGING_LEVEL` → `logging.level`

### Go Configuration Structs
```go
type Config struct {
    Server    ServerConfig    `yaml:"server"`
    Transport TransportConfig `yaml:"transport"`
    Plugins   PluginsConfig   `yaml:"plugins"`
    Logging   LoggingConfig   `yaml:"logging"`
    Security  SecurityConfig  `yaml:"security"`
}

type ServerConfig struct {
    Name    string `yaml:"name"`
    Version string `yaml:"version"`
    Debug   bool   `yaml:"debug"`
}

type TransportConfig struct {
    Protocol string      `yaml:"protocol"`
    STDIO    STDIOConfig  `yaml:"stdio"`
    SSE      SSEConfig    `yaml:"sse"`
    HTTP     HTTPConfig   `yaml:"http"`
}
```

### File Structure
- `internal/config/config.go` - Configuration types and loading
- `internal/config/validation.go` - Configuration validation
- `internal/config/env.go` - Environment variable handling
- `config.yaml` - Default configuration file
- `config.dev.yaml` - Development overrides

### Configuration Loading Pattern
1. **Load defaults**: Built-in sensible defaults
2. **Load file**: Read config.yaml from multiple search paths
3. **Environment overrides**: Apply environment variable overrides
4. **Validation**: Validate final configuration
5. **Hot reload**: Watch for file changes in development mode

### Validation Rules
- Transport protocol must be one of: stdio, sse, http
- Port numbers must be valid (1-65535)
- Timeout values must be valid durations
- Plugin directories must exist and be readable
- Log level must be valid
- Required fields validation

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
�� CREATIVE PHASE END 