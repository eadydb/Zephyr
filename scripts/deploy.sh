#!/bin/bash

# Zephyr MCP Server 部署脚本
# 专门负责使用 docker-compose 进行部署

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
IMAGE_NAME="zephyr-mcp-server"
IMAGE_TAG="latest"
COMPOSE_SERVICE="zephyr-mcp"

# 检查镜像是否存在
check_image() {
    if ! docker images -q "$IMAGE_NAME:$IMAGE_TAG" | grep -q .; then
        echo "❌ Docker镜像不存在: $IMAGE_NAME:$IMAGE_TAG"
        echo "请先运行构建脚本: ./scripts/build.sh"
        exit 1
    fi
    echo "✅ 找到Docker镜像: $IMAGE_NAME:$IMAGE_TAG"
}

# 检查docker-compose配置
check_compose() {
    cd "$PROJECT_ROOT"
    if [ ! -f "docker-compose.yml" ]; then
        echo "❌ docker-compose.yml 文件不存在"
        exit 1
    fi
    
    # 验证配置文件
    if ! docker-compose config >/dev/null 2>&1; then
        echo "❌ docker-compose.yml 配置文件有错误"
        docker-compose config
        exit 1
    fi
    echo "✅ docker-compose 配置验证通过"
}

# 启动服务
start_services() {
    echo "启动服务..."
    cd "$PROJECT_ROOT"
    
    docker-compose up -d
    
    echo "✅ 服务启动完成"
    echo "服务名称: $COMPOSE_SERVICE"
    echo "HTTP端口: 26842"
    echo "SSE端口: 26841" 
    echo "API端口: 26842"
    echo "监控端口: 26843"
}

# 停止服务
stop_services() {
    echo "停止服务..."
    cd "$PROJECT_ROOT"
    
    docker-compose down
    
    echo "✅ 服务已停止"
}

# 重启服务
restart_services() {
    echo "重启服务..."
    cd "$PROJECT_ROOT"
    
    docker-compose restart
    
    echo "✅ 服务已重启"
}

# 检查服务状态
check_status() {
    echo "检查服务状态..."
    cd "$PROJECT_ROOT"
    
    # 显示容器状态
    docker-compose ps
    
    # 检查健康状态
    echo ""
    echo "检查服务健康状态..."
    sleep 3
    
    if curl -s http://localhost:26843/health >/dev/null 2>&1; then
        echo "✅ 服务健康检查通过"
        echo "健康检查地址: http://localhost:26843/health"
        echo "监控指标地址: http://localhost:26843/metrics"
    else
        echo "⚠️  服务健康检查失败，请查看日志"
    fi
}

# 查看日志
show_logs() {
    echo "显示服务日志..."
    cd "$PROJECT_ROOT"
    
    if [ "$1" = "-f" ]; then
        docker-compose logs -f
    else
        docker-compose logs --tail 100
    fi
}

# 拉取最新镜像
pull_images() {
    echo "拉取最新镜像..."
    cd "$PROJECT_ROOT"
    
    # 如果镜像不是本地构建，则可以拉取
    # docker-compose pull
    echo "本地构建镜像，无需拉取"
}

# 查看服务详情
show_info() {
    echo "服务详情:"
    cd "$PROJECT_ROOT"
    
    echo ""
    echo "=== 服务状态 ==="
    docker-compose ps
    
    echo ""
    echo "=== 端口映射 ==="
    docker-compose port zephyr-mcp 26841 2>/dev/null && echo "SSE: http://localhost:$(docker-compose port zephyr-mcp 26841 | cut -d: -f2)"
    docker-compose port zephyr-mcp 26842 2>/dev/null && echo "HTTP: http://localhost:$(docker-compose port zephyr-mcp 26842 | cut -d: -f2)"
    docker-compose port zephyr-mcp 26843 2>/dev/null && echo "监控: http://localhost:$(docker-compose port zephyr-mcp 26843 | cut -d: -f2)"
    
    echo ""
    echo "=== 资源使用 ==="
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}" | grep zephyr-mcp || echo "服务未运行"
}

# 清理资源
cleanup_services() {
    echo "清理服务资源..."
    cd "$PROJECT_ROOT"
    
    # 停止并删除容器、网络
    docker-compose down -v
    
    # 删除悬挂的镜像（可选）
    if [ "${1:-}" = "all" ]; then
        echo "删除相关镜像..."
        docker-compose down -v --rmi all 2>/dev/null || true
        echo "✅ 已删除所有相关资源"
    fi
    
    echo "✅ 清理完成"
}

# 显示帮助信息
show_help() {
    echo "Zephyr MCP Server 部署脚本"
    echo "专门负责使用 docker-compose 进行部署"
    echo ""
    echo "用法: $0 [命令]"
    echo ""
    echo "命令:"
    echo "  start        启动服务"
    echo "  stop         停止服务"
    echo "  restart      重启服务"
    echo "  status       检查服务状态"
    echo "  logs         查看服务日志"
    echo "  logs -f      实时查看服务日志"
    echo "  info         显示服务详细信息"
    echo "  pull         拉取最新镜像"
    echo "  cleanup      清理服务资源"
    echo "  cleanup all  清理所有资源（包括镜像）"
    echo "  help         显示此帮助信息"
    echo ""
    echo "部署前提:"
    echo "  请先运行构建脚本: ./scripts/build.sh"
    echo ""
    echo "示例:"
    echo "  $0 start     # 启动服务"
    echo "  $0 status    # 检查状态"
    echo "  $0 logs -f   # 实时查看日志"
}

# 主执行逻辑
main() {
    case "${1:-start}" in
        "start")
            check_image
            check_compose
            start_services
            check_status
            ;;
        "stop")
            stop_services
            ;;
        "restart")
            check_image
            check_compose
            restart_services
            check_status
            ;;
        "status")
            check_status
            ;;
        "logs")
            show_logs "${2:-}"
            ;;
        "info")
            show_info
            ;;
        "pull")
            pull_images
            ;;
        "cleanup")
            cleanup_services "${2:-}"
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            echo "未知命令: $1"
            echo "使用 '$0 help' 查看帮助信息"
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@" 