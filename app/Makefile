.PHONY: help build run test fmt lint deps docs install-swagger build-prod anydoc-lib build-anydoc clean migrate-up migrate-down migrate-version migrate-create migrate-force migrate-goto download_spatial

BINARY_NAME=Gaze
MAIN_PATH=./cmd/server

help:
	@echo "Gaze 后端 Makefile"
	@echo ""
	@echo "  build            构建后端二进制"
	@echo "  run              构建并运行后端"
	@echo "  test             运行 Go 测试"
	@echo "  fmt              格式化 Go 代码"
	@echo "  lint             运行 golangci-lint"
	@echo "  deps             下载 Go 依赖"
	@echo "  docs             生成 Swagger"
	@echo "  install-swagger  安装 swag"
	@echo "  build-prod       生产构建"
	@echo "  anydoc-lib       构建 anydoc 静态库（Rust）"
	@echo "  build-anydoc     构建带 anydoc 的应用"
	@echo "  migrate-up/down/version/create/force/goto  数据库迁移"
	@echo ""
	@echo "全栈编排、镜像、前端和 Lite 打包请在仓库根 Makefile 执行。"

build:
	go build -o $(BINARY_NAME) $(MAIN_PATH)

run: build
	./$(BINARY_NAME)

test:
	go test -v ./...

fmt:
	go fmt ./...

lint:
	golangci-lint run

deps:
	go mod download

docs:
	@echo "生成 Swagger API 文档..."
	swag init -g $(MAIN_PATH)/main.go -o ./docs --parseDependency --parseInternal
	@echo "文档已生成到 ./docs 目录"

install-swagger:
	go install github.com/swaggo/swag/cmd/swag@latest

anydoc-lib:
	./scripts/build-anydoc-lib.sh

build-anydoc: anydoc-lib
	go build -tags anydoc -o $(BINARY_NAME) $(MAIN_PATH)

build-prod:
	VERSION=$$(git describe --tags --abbrev=0 2>/dev/null || echo "$${VERSION:-unknown}"); \
	COMMIT_ID=$${COMMIT_ID:-unknown}; \
	CGO_ENABLED=1 \
	CGO_CFLAGS="-Wno-deprecated-declarations" \
	CGO_LDFLAGS="$$(if [ "$$(uname)" = 'Darwin' ]; then echo '-Wl,-no_warn_duplicate_libraries'; fi)" \
	BUILD_TIME=$${BUILD_TIME:-unknown}; \
	GO_VERSION=$${GO_VERSION:-unknown}; \
	LDFLAGS="-X 'gaze/internal/handler.Version=$$VERSION' -X 'gaze/internal/handler.Edition=standard' -X 'gaze/internal/handler.CommitID=$$COMMIT_ID' -X 'gaze/internal/handler.BuildTime=$$BUILD_TIME' -X 'gaze/internal/handler.GoVersion=$$GO_VERSION' -X 'google.golang.org/protobuf/reflect/protoregistry.conflictPolicy=warn'"; \
	go build -tags "$(GO_BUILD_TAGS)" -ldflags="-w -s $$LDFLAGS" -o $(BINARY_NAME) $(MAIN_PATH)

migrate-up:
	./scripts/migrate.sh up

migrate-down:
	./scripts/migrate.sh down

migrate-version:
	./scripts/migrate.sh version

migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "Error: migration name is required"; \
		echo "Usage: make migrate-create name=your_migration_name"; \
		exit 1; \
	fi
	./scripts/migrate.sh create $(name)

migrate-force:
	@if [ -z "$(version)" ]; then \
		echo "Error: version is required"; \
		echo "Usage: make migrate-force version=4"; \
		exit 1; \
	fi
	./scripts/migrate.sh force $(version)

migrate-goto:
	@if [ -z "$(version)" ]; then \
		echo "Error: version is required"; \
		echo "Usage: make migrate-goto version=3"; \
		exit 1; \
	fi
	./scripts/migrate.sh goto $(version)

download_spatial:
	go run cmd/download/duckdb/duckdb.go

clean:
	go clean
	rm -f $(BINARY_NAME)
