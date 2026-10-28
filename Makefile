SHELL := /bin/bash

COMPOSE := docker compose -f deployments/docker-compose.yml
DEV_COMPOSE := docker compose -f deployments/docker-compose.yml -f deployments/docker-compose.hotreload.yml

.PHONY: help up down restart ps logs dev dev-logs dev-stop test test-app test-frontend test-rag test-mcp build-app build-frontend images images-app images-docreader images-frontend package-lite migrate-up migrate-down health

help:
	@echo "WellRead / Gaze 根 Makefile"
	@echo ""
	@echo "全栈:"
	@echo "  up                启动全栈"
	@echo "  down              停止全栈"
	@echo "  restart           重启全栈"
	@echo "  ps                查看容器状态"
	@echo "  logs              查看全栈日志"
	@echo "  dev               启动热重载后端"
	@echo "  dev-logs          查看热重载后端日志"
	@echo ""
	@echo "构建与测试:"
	@echo "  build-app         构建 Go 后端"
	@echo "  build-frontend    构建前端"
	@echo "  test-app          Go 构建与测试"
	@echo "  test-frontend     前端 type-check / test / build"
	@echo "  test-rag          RAG 语法检查与 compose config"
	@echo "  test-mcp          MCP 测试"
	@echo "  images            构建全部镜像"
	@echo "  package-lite      打包 Lite 发行包"
	@echo ""
	@echo "数据库:"
	@echo "  migrate-up        执行迁移"

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

restart:
	$(COMPOSE) restart

ps:
	$(COMPOSE) ps

logs:
	$(COMPOSE) logs -f

dev:
	$(DEV_COMPOSE) up -d --build app

dev-logs:
	$(DEV_COMPOSE) logs -f app

dev-stop:
	$(DEV_COMPOSE) down

build-app:
	$(MAKE) -C app build

build-frontend:
	cd frontend && npm run build

test-app:
	cd app && go build ./... && go vet ./...
	cd app && go test ./internal/... ./contracts/...

test-frontend:
	cd frontend && npm run type-check
	cd frontend && npm test
	cd frontend && npm run build

test-rag:
	cd rag && python -m compileall -q src
	cd rag && docker compose -f docker-compose.yml config >/dev/null

test-mcp:
	cd mcp && uv sync --extra test && uv run pytest -q

test: test-app test-frontend test-rag test-mcp

images:
	./scripts/build_images.sh

images-app:
	./scripts/build_images.sh --app

images-docreader:
	./scripts/build_images.sh --docreader

images-frontend:
	./scripts/build_images.sh --frontend

package-lite:
	./scripts/package-lite.sh

migrate-up:
	cd app && ./scripts/migrate.sh up

migrate-down:
	cd app && ./scripts/migrate.sh down

health:
	@curl -f http://localhost:6152/health || curl -f http://localhost:8080/health
