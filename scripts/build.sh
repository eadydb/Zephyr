#!/bin/bash

# Zephyr MCP Server 构建脚本
# 负责构建应用、插件和Docker镜像

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
IMAGE_NAME="zephyr"
IMAGE_TAG="latest"

# 默认平台配置
DEFAULT_HOST_PLATFORM=""
DEFAULT_TARGET_PLATFORM=""

# 检测主机平台
detect_host_platform() {
    case "$(uname -s)" in
        Linux*)     DEFAULT_HOST_PLATFORM=linux;;
        Darwin*)    DEFAULT_HOST_PLATFORM=macos;;
        *)          echo "不支持的主机平台: $(uname -s)"; exit 1;;
    esac
    echo "检测到主机平台: $DEFAULT_HOST_PLATFORM"
}

# 设置目标平台
set_target_platform() {
    local target_platform="${1:-}"
    
    if [ -n "$target_platform" ]; then
        # 手动指定平台 - 只支持 linux 和当前平台
        case "$target_platform" in
            "linux"|"linux/amd64")
                TARGET_PLATFORM="linux/amd64"
                TARGET_OS="linux"
                TARGET_ARCH="amd64"
                USE_CROSS_BUILD=true
                ;;
            *)
                echo "不支持的目标平台: $target_platform"
                echo "支持的平台: linux (仅在明确指定时)"
                exit 1
                ;;
        esac
        echo "指定目标平台: $TARGET_PLATFORM"
    else
        # 默认使用当前系统平台
        case "$DEFAULT_HOST_PLATFORM" in
            linux)
                TARGET_PLATFORM="linux/amd64"
                TARGET_OS="linux"
                TARGET_ARCH="amd64"
                USE_CROSS_BUILD=false
                ;;
            macos)
                if [ "$(uname -m)" = "arm64" ]; then
                    TARGET_PLATFORM="darwin/arm64"
                    TARGET_OS="darwin"
                    TARGET_ARCH="arm64"
                else
                    TARGET_PLATFORM="darwin/amd64"
                    TARGET_OS="darwin"
                    TARGET_ARCH="amd64"
                fi
                USE_CROSS_BUILD=false
                ;;
        esac
        echo "使用当前系统平台: $TARGET_PLATFORM"
    fi
}

# 检查Docker Buildx支持
check_buildx() {
    if ! docker buildx version >/dev/null 2>&1; then
        echo "警告: Docker Buildx 不可用，将使用标准构建"
        return 1
    fi
    
    # 确保启用了多平台构建器
    if ! docker buildx ls | grep -q "docker-container"; then
        echo "创建多平台构建器..."
        docker buildx create --name multiplatform --driver docker-container --use 2>/dev/null || true
    fi
    
    return 0
}

# 构建插件
build_plugins() {
    echo "构建插件..."
    cd "$PROJECT_ROOT"
    
    if [ -f "scripts/build-plugins.sh" ]; then
        chmod +x scripts/build-plugins.sh
        scripts/build-plugins.sh build
    else
        echo "错误: 插件构建脚本不存在"
        exit 1
    fi
}

# 构建主应用 (本地二进制)
build_app() {
    echo "构建主应用..."
    cd "$PROJECT_ROOT"
    
    # 确保bin目录存在
    mkdir -p bin
    
    # 构建二进制文件
    echo "构建目标: GOOS=$TARGET_OS GOARCH=$TARGET_ARCH"
    
    if [ "$TARGET_OS" = "linux" ]; then
        CGO_ENABLED=0 GOOS="$TARGET_OS" GOARCH="$TARGET_ARCH" go build -a -installsuffix cgo -o bin/zephyr cmd/zephyr/main.go
    else
        GOOS="$TARGET_OS" GOARCH="$TARGET_ARCH" go build -o bin/zephyr cmd/zephyr/main.go
    fi
    
    echo "✅ 应用构建完成: bin/zephyr (平台: $TARGET_PLATFORM)"
}

# 构建Docker镜像
build_image() {
    echo "构建Docker镜像..."
    cd "$PROJECT_ROOT"
    
    # 只有在跨平台构建Linux时才使用buildx
    if [ "$USE_CROSS_BUILD" = true ] && [ "$TARGET_OS" = "linux" ]; then
        echo "跨平台构建Linux镜像"
        if check_buildx; then
            echo "使用 Docker Buildx 进行跨平台构建"
            docker buildx build \
                --platform "$TARGET_PLATFORM" \
                --load \
                -t "zephyr-mcp-server:$IMAGE_TAG" \
                .
        else
            echo "错误: 跨平台构建需要Docker Buildx支持"
            exit 1
        fi
    else
        # 使用标准构建（当前平台）
        echo "使用标准Docker构建"
        docker build -t "zephyr-mcp-server:$IMAGE_TAG" .
    fi
    
    echo "✅ Docker镜像构建完成: zephyr-mcp-server:$IMAGE_TAG (平台: $TARGET_PLATFORM)"
    
    # 显示镜像信息
    echo "镜像信息:"
    docker images "zephyr-mcp-server:$IMAGE_TAG" --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"
}

# 验证构建结果
verify_build() {
    echo "验证构建结果..."
    
    # 检查二进制文件
    if [ -f "$PROJECT_ROOT/bin/zephyr" ]; then
        echo "✅ 二进制文件: $(ls -lh "$PROJECT_ROOT/bin/zephyr" | awk '{print $5}')"
    else
        echo "❌ 二进制文件未找到"
    fi
    
    # 检查插件
    echo "插件状态:"
    cd "$PROJECT_ROOT"
    scripts/build-plugins.sh list
    
    # 检查Docker镜像
    if docker images -q "zephyr-mcp-server:$IMAGE_TAG" | grep -q .; then
        echo "✅ Docker镜像已准备就绪"
    else
        echo "❌ Docker镜像未找到"
    fi
}

# 清理构建文件
clean_build() {
    echo "清理构建文件..."
    cd "$PROJECT_ROOT"
    
    # 清理二进制文件
    rm -f bin/zephyr
    
    # 清理插件
    scripts/build-plugins.sh clean 2>/dev/null || true
    
    # 清理Docker镜像（可选）
    if [ "${1:-}" = "all" ]; then
        docker rmi "zephyr-mcp-server:$IMAGE_TAG" 2>/dev/null || true
        echo "✅ 已删除Docker镜像"
    fi
    
    echo "✅ 清理完成"
}

# 运行测试
run_tests() {
    echo "运行测试..."
    cd "$PROJECT_ROOT"
    
    # Go测试
    echo "运行Go测试..."
    go test ./... || true
    
    # 插件测试
    echo "运行插件测试..."
    scripts/build-plugins.sh test || true
    
    echo "✅ 测试完成"
}

# 显示帮助信息
show_help() {
    echo "Zephyr MCP Server 构建脚本"
    echo ""
    echo "用法: $0 [命令] [--platform=平台]"
    echo ""
    echo "命令:"
    echo "  plugins      只构建插件"
    echo "  app          只构建应用"
    echo "  image        只构建Docker镜像"
    echo "  all          构建所有组件 (默认)"
    echo "  test         运行测试"
    echo "  verify       验证构建结果"
    echo "  clean        清理构建文件"
    echo "  clean all    清理所有资源（包括镜像）"
    echo "  help         显示此帮助信息"
    echo ""
    echo "平台选项:"
    echo "  --platform=PLATFORM  指定目标平台"
    echo ""
    echo "支持的平台:"
    echo "  默认: 当前系统平台 (自动检测)"
    echo "  linux: 仅在明确指定时构建Linux平台"
    echo ""
    echo "示例:"
    echo "  $0                           # 使用当前系统平台构建"
    echo "  $0 all                       # 构建所有组件"
    echo "  $0 image --platform=linux   # 构建Linux镜像"
    echo "  $0 verify                    # 验证构建结果"
    echo ""
    echo "跨平台构建:"
    echo "  仅支持在macOS上构建Linux镜像"
    echo "  需要Docker Buildx支持，首次使用会自动创建多平台构建器"
}

# 解析命令行参数
parse_args() {
    local target_platform=""
    local command=""
    local first_arg=true
    
    # 解析参数
    for arg in "$@"; do
        case $arg in
            --platform=*)
                target_platform="${arg#*=}"
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            -*)
                echo "未知参数: $arg"
                show_help
                exit 1
                ;;
            *)
                if [ "$first_arg" = true ]; then
                    command="$arg"
                    first_arg=false
                fi
                ;;
        esac
    done
    
    # 设置默认命令
    if [ -z "$command" ]; then
        command="all"
    fi
    
    # 返回解析结果
    echo "$command|$target_platform"
}

# 主执行逻辑
main() {
    # 解析命令行参数
    local parse_result
    parse_result=$(parse_args "$@")
    local command="${parse_result%|*}"
    local target_platform="${parse_result#*|}"
    
    # 检测主机平台
    detect_host_platform
    
    # 设置目标平台
    set_target_platform "$target_platform"
    
    case "$command" in
        "plugins")
            build_plugins
            ;;
        "app")
            build_app
            ;;
        "image")
            # 构建镜像前确保插件已构建
            build_plugins
            build_image
            ;;
        "all")
            build_plugins
            build_app
            build_image
            verify_build
            ;;
        "test")
            run_tests
            ;;
        "verify")
            verify_build
            ;;
        "clean")
            clean_build "${2:-}"
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