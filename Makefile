# Zephyr MCP Server Makefile

APP_NAME = zephyr
IMAGE_NAME = zephyr
CONTAINER_NAME = zephyr

# 默认目标
.PHONY: all
all: build

# 构建应用和插件
.PHONY: build
build:
	@echo "构建应用和插件..."
	@scripts/build.sh app $(if $(PLATFORM),--platform=$(PLATFORM))
	@echo "✅ 构建完成"

# 构建Docker镜像
.PHONY: image
image:
	@echo "构建Docker镜像..."
	@scripts/build.sh image $(if $(PLATFORM),--platform=$(PLATFORM))
	@echo "✅ 镜像构建完成"

# 构建所有组件
.PHONY: build-all
build-all:
	@echo "构建所有组件..."
	@scripts/build.sh all $(if $(PLATFORM),--platform=$(PLATFORM))
	@echo "✅ 所有组件构建完成"

# 构建Linux镜像（跨平台）
.PHONY: image-linux
image-linux:
	@echo "构建Linux镜像..."
	@scripts/build.sh image --platform=linux
	@echo "✅ Linux镜像构建完成"

# 构建ARM64 Linux镜像
.PHONY: image-linux-arm64
image-linux-arm64:
	@echo "构建ARM64 Linux镜像..."
	@scripts/build.sh image --platform=linux/arm64
	@echo "✅ ARM64 Linux镜像构建完成"

# 使用docker-compose部署
.PHONY: deploy
deploy:
	@echo "部署服务..."
	@scripts/deploy.sh start
	@echo "✅ 部署完成"

# 启动服务
.PHONY: start
start:
	@scripts/deploy.sh start

# 停止服务
.PHONY: stop
stop:
	@scripts/deploy.sh stop

# 重启服务
.PHONY: restart
restart:
	@scripts/deploy.sh restart

# 查看服务状态
.PHONY: status
status:
	@scripts/deploy.sh status

# 查看服务详情
.PHONY: info
info:
	@scripts/deploy.sh info

# 查看日志
.PHONY: logs
logs:
	@scripts/deploy.sh logs -f

# 清理构建文件
.PHONY: clean
clean:
	@scripts/build.sh clean

# 清理部署资源
.PHONY: clean-deploy
clean-deploy:
	@scripts/deploy.sh cleanup

# 完全清理（构建+部署）
.PHONY: clean-all
clean-all:
	@scripts/deploy.sh cleanup all
	@scripts/build.sh clean all

# 运行测试
.PHONY: test
test:
	@scripts/build.sh test

# 本地运行（开发模式）
.PHONY: dev
dev: build
	@echo "启动开发模式..."
	@./bin/$(APP_NAME) serve --config scripts/config.yaml

# 显示帮助
.PHONY: help
help:
	@echo "Zephyr MCP Server Makefile"
	@echo ""
	@echo "构建命令:"
	@echo "  build        构建应用和插件"
	@echo "  image        构建Docker镜像"
	@echo "  build-all    构建所有组件"
	@echo "  image-linux  构建Linux镜像（跨平台）"
	@echo "  image-linux-arm64  构建ARM64 Linux镜像"
	@echo ""
	@echo "部署命令:"
	@echo "  deploy       部署服务"
	@echo "  start        启动服务"
	@echo "  stop         停止服务"
	@echo "  restart      重启服务"
	@echo "  status       查看服务状态"
	@echo "  info         查看服务详情"
	@echo "  logs         查看服务日志"
	@echo ""
	@echo "其他命令:"
	@echo "  test         运行测试"
	@echo "  dev          本地开发模式运行"
	@echo "  clean        清理构建文件"
	@echo "  clean-deploy 清理部署资源"
	@echo "  clean-all    清理所有资源"
	@echo "  help         显示此帮助信息"
	@echo ""
	@echo "平台选项:"
	@echo "  使用 PLATFORM=平台 来指定目标平台"
	@echo "  例如: make image PLATFORM=linux"
	@echo "  例如: make build-all PLATFORM=linux/arm64" 