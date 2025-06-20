# Zephyr MCP Server

[![Go Version](https://img.shields.io/badge/go-1.24.4-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Docker Pulls](https://img.shields.io/badge/docker-available-blue.svg)](https://hub.docker.com)

Zephyr 是一个高性能的 MCP (Model Context Protocol) 服务器，采用 Go 语言编写。它支持多种传输协议，提供插件架构以扩展自定义工具功能，专为快速、模块化开发而设计。

## ✨ 特性

- 🚀 **多协议传输支持** - STDIO、SSE、HTTP 多种传输方式
- 🔌 **插件架构** - 动态插件系统，支持自定义工具扩展
- ⚙️ **灵活配置** - 支持 YAML 文件、环境变量、命令行参数配置
- 📊 **监控指标** - 内置监控和健康检查端点
- 🐳 **容器化部署** - 完整的 Docker 和 Docker Compose 支持
- 🛡️ **安全特性** - 速率限制、超时控制、权限管理
- 📝 **完整日志** - 结构化日志输出，支持多种格式
- 🔄 **优雅关闭** - 资源管理和优雅关闭机制

## 🛠️ 内置工具

- **systeminfo** - 系统信息查询（操作系统、架构、内存、运行时详情）
- **currenttime** - 当前时间获取（支持时区配置）
- **fileops** - 文件操作工具（读取、写入、列表、状态检查）

## 📋 系统要求

- Go 1.24.4 或更高版本
- Docker（可选，用于容器化部署）
- Linux/macOS/Windows 系统支持


## 📚 API 文档

### MCP 协议支持

Zephyr 完全兼容 MCP (Model Context Protocol) 规范：

- **工具调用** - 支持工具注册和调用
- **资源管理** - 文件和数据资源访问
- **消息传递** - 双向消息通信
- **会话管理** - 客户端会话管理

### 传输协议

#### STDIO 传输

```bash
# 使用 STDIO 协议
zephyr serve --transport stdio
```

#### SSE 传输

```bash
# 启动 SSE 服务器
zephyr serve --transport sse --port 26841

# 客户端连接
curl -N -H "Accept: text/event-stream" http://localhost:26841/sse
```

#### HTTP 传输

```bash
# 启动 HTTP 服务器
zephyr serve --transport http --port 26842

# 发送请求
curl -X POST http://localhost:26842/mcp \
  -H "Content-Type: application/json" \
  -d '{"method": "tools/list", "params": {}}'
```

