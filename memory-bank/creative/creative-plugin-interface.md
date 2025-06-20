📌 CREATIVE PHASE START: Plugin Interface Design
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

## 1️⃣ PROBLEM
**Description**: Design clean plugin interfaces for MCP tools that integrate with existing zephyr plugin system while supporting MCP-specific requirements

**Requirements**:
- Extend existing zephyr plugin architecture for MCP tools
- Support MCP tool metadata (name, description, input schema)
- Enable runtime tool registration and discovery
- Support both built-in and external plugins
- Clean integration with mcp-go tool interfaces
- Tool lifecycle management (load, execute, unload)

**Constraints**:
- Must be backward compatible with existing plugin system
- Follow Go plugin patterns and interfaces
- Keep plugin interface simple and focused
- Support JSON schema validation for tool inputs
- Enable plugin hot-reloading for development

## 2️⃣ OPTIONS

**Option A: Interface Extension** - Extend existing plugin interfaces with MCP-specific methods
**Option B: Composition Pattern** - Compose MCP tool interface with existing plugin interface
**Option C: Registry-Based System** - Separate MCP tool registry with plugin discovery hooks

## 3️⃣ ANALYSIS

| Criterion | Interface Extension | Composition Pattern | Registry-Based |
|-----------|-------------------|-------------------|----------------|
| **Backward Compatibility** | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Simplicity** | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ |
| **Separation of Concerns** | ⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Flexibility** | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Testing** | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **mcp-go Integration** | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |

**Key Insights**:
- Interface Extension is simple but breaks existing plugin compatibility
- Composition Pattern provides good balance of compatibility and functionality
- Registry-Based offers best separation and flexibility but adds complexity
- Registry pattern aligns well with MCP tool discovery requirements
- Composition allows gradual migration of existing plugins

## 4️⃣ DECISION
**Selected**: Option C: Registry-Based System with Composition
**Rationale**: Best separation of concerns, future-proof design, and excellent mcp-go integration while maintaining backward compatibility

## 5️⃣ IMPLEMENTATION NOTES

### Core Plugin Components
- **MCPToolPlugin Interface**: MCP-specific tool interface 
- **PluginRegistry**: Manages both traditional and MCP plugins
- **ToolRegistry**: Specialized registry for MCP tools
- **Plugin Adapter**: Bridges existing plugins to MCP tools where possible

### Key Interfaces
```go
// MCP Tool Plugin Interface
type MCPToolPlugin interface {
    // Basic plugin interface
    Name() string
    Description() string
    Version() string
    
    // MCP-specific methods
    MCPToolDefinition() MCPTool
    Execute(ctx context.Context, input map[string]interface{}) (interface{}, error)
    InputSchema() map[string]interface{}
    
    // Lifecycle methods
    Initialize() error
    Cleanup() error
}

// Tool Registry Interface
type ToolRegistry interface {
    RegisterTool(tool MCPToolPlugin) error
    UnregisterTool(name string) error
    GetTool(name string) (MCPToolPlugin, error)
    ListTools() []MCPToolPlugin
    DiscoverTools() error
}

// Bridge for existing plugins
type PluginAdapter interface {
    CanAdapt(plugin interface{}) bool
    Adapt(plugin interface{}) (MCPToolPlugin, error)
}
```

### File Structure
- `pkg/plugin/mcp.go` - MCP plugin interfaces and types
- `internal/registry/tool_registry.go` - MCP tool registry implementation  
- `internal/registry/plugin_adapter.go` - Adapter for existing plugins
- `plugins/systeminfo/systeminfo.go` - System info tool implementation
- `plugins/currenttime/currenttime.go` - Current time tool implementation

### Integration Pattern
1. **Registration**: Tools register via ToolRegistry during startup
2. **Discovery**: Registry scans plugin directories for MCP tools
3. **Adaptation**: Existing plugins adapted to MCP interface if possible
4. **Execution**: MCP server calls tools via unified interface
5. **Lifecycle**: Registry manages tool lifecycle and health

### Plugin Development Pattern
```go
// Example tool implementation
type SystemInfoTool struct{}

func (s *SystemInfoTool) Name() string { return "systeminfo" }
func (s *SystemInfoTool) Description() string { return "Get system information" }
func (s *SystemInfoTool) MCPToolDefinition() MCPTool { /* return mcp tool def */ }
func (s *SystemInfoTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
    // Implementation
}
func (s *SystemInfoTool) InputSchema() map[string]interface{} { /* return JSON schema */ }
```

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
�� CREATIVE PHASE END 