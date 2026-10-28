# Gaze / WellRead App

WellRead 的 Go 后端：HTTP API、领域与应用服务、权限资源守卫、文档解析客户端、CLI、Go SDK、数据库迁移和 Swagger。前端、RAG、MCP、部署和项目文档在仓库根目录。

本 README 是 `app/` 的导航入口：先看模块边界和目录，再按需要进入子系统文档。具体业务规则以根目录 `docs/` 为准。

## 1. 模块边界

| 归属 | 内容 |
|---|---|
| `app/` | API、会话、知识库、文档、部门/RBAC/ACL、治理、设置、CLI、SDK、docreader 客户端 |
| `frontend/` | Vue 3 SPA；能力契约从 `app/contracts` 读取 |
| `rag/` | 检索与生成侧车；app 通过 `/chat/stream`、`/search`、`/ingest` 调用 |
| `mcp/` | 外部 MCP 接入层 |
| `deployments/` | 全栈 compose、环境变量、systemd |

`app/` 不负责前端渲染、RAG 索引构建或外部部署编排。

## 2. 目录地图

```text
app/
├── cmd/server/              # HTTP 服务入口
├── internal/
│   ├── agent/               # Agent 运行时（外部接入保留）
│   ├── application/         # 应用服务与仓储
│   ├── capabilities/        # 能力契约加载
│   ├── datasource/          # 外部数据源同步框架
│   ├── event/               # 进程内事件总线
│   ├── handler/             # HTTP handlers
│   ├── middleware/          # 认证、RBAC、KB/ACL 资源守卫
│   ├── modelcontext/        # 模型上下文句柄/引用编解码
│   ├── router/              # 路由与守卫矩阵
│   └── types/               # 领域类型与接口
├── cli/                     # WellRead CLI（独立 Go module）
├── client/                  # Go SDK
├── contracts/               # capabilities.json（前后端共享能力契约）
├── migrations/              # PostgreSQL / SQLite 迁移
├── docreader/               # 文档解析 gRPC 服务
├── config/                  # 业务配置与提示词
├── docker/                  # app/docreader 镜像
├── scripts/                 # Go/构建/迁移/anydoc/e2e 脚本
├── docs/                    # Swagger 生成物（不是人工文档）
├── licenses/                # 分发许可
└── third_party/             # anydoc Go/Rust 绑定
```

## 3. 子系统文档

| 子系统 | 文档 | 说明 |
|---|---|---|
| CLI | [cli/使用说明.md](cli/使用说明.md) | 安装、profile、KB/文档、搜索、问答、JSON/退出码 |
| Go SDK | [client/使用说明.md](client/使用说明.md) | 客户端初始化、KB/文档、会话、流式问答、模型与分块 |
| DocReader | [docreader/服务说明.md](docreader/服务说明.md) | 解析服务、gRPC、环境变量、PDF/表格/OCR |
| 数据源同步 | [internal/datasource/数据源同步框架.md](internal/datasource/数据源同步框架.md) | Connector 架构、任务流、数据模型、API |
| 连接器开发 | [internal/datasource/连接器开发指南.md](internal/datasource/连接器开发指南.md) | 新平台接入步骤、OAuth、分页、增量、删除同步 |
| 事件总线 | [internal/event/事件系统.md](internal/event/事件系统.md) | 进程内事件、处理器、中间件、集成方式 |
| 事件示例 | [internal/event/事件系统使用示例.md](internal/event/事件系统使用示例.md) | 在查询/检索/生成链路中埋点 |
| 模型上下文 | [internal/modelcontext/模型上下文边界.md](internal/modelcontext/模型上下文边界.md) | 模型句柄、引用、MCP 路由、解码边界 |
| 云镜像脚本 | [scripts/cloud-image/云镜像部署.md](scripts/cloud-image/云镜像部署.md) | 旧云镜像脚本说明；当前主部署见 `deployments/` |

CLI 的 Agent 契约、退出码和 JSON envelope 以 [cli/AGENTS.md](cli/AGENTS.md) 为准；`使用说明.md` 只做人类入口。

## 4. 常用命令

```bash
cd app
make build             # 构建后端
make test              # Go 测试
make docs              # 重新生成 Swagger
make anydoc-lib        # 构建 anydoc 静态库
make build-anydoc      # 带 anydoc 引擎构建
```

全栈启动、前端、镜像和 Lite 打包在仓库根：

```bash
make up
make dev
make test-app
make images
```

## 5. 配置

- 全栈环境变量：`../deployments/.env`，示例 `../deployments/.env.example`。
- app 业务配置：`config/config.yaml`（compose 绑定挂载到容器）。
- Lite：`.env.lite.example`。
- 密钥、模型凭据、DB 密码不入库；由 env 注入。

## 6. 能力契约

`contracts/capabilities.json` 是公司角色/部门角色到能力码的唯一事实源：

- 后端 `internal/capabilities` 加载并校验。
- 前端 `frontend/src/config/capabilities.ts` 直接导入。
- 路由 meta 使用 `requiredCaps` / `requiredAnyCaps`；页面只是投影，最终由后端资源守卫判定。
- 新增能力必须同时更新 JSON、Go 白名单、前端类型/测试和相关路由。

## 7. 数据库与迁移

- PostgreSQL 迁移：`migrations/versioned/`。
- SQLite/Lite 迁移：`migrations/sqlite/`。
- 命令：`scripts/migrate.sh up|down|version|create|force|goto`。
- `AUTO_MIGRATE=true` 时 app 启动自动执行迁移；生产升级前先备份。

## 8. API 与 Swagger

- API 前缀：`/api/v1`。
- Swagger UI：启动后 `/swagger/index.html`。
- `docs/docs.go`、`docs/swagger.json`、`docs/swagger.yaml` 都是 `swag init` 生成物，**不要手改**。
- `docs/swagger_contract_test.go` 是契约测试，校验生成物包含 `/knowledge-search`、不残留 `/sessions/search` 旧路由、`DELETE /models/{id}` 记录 400。
- 改 handler 注释后：`make docs && go test ./docs/`。

## 9. 测试与调试

```bash
cd app
go build ./internal/... ./contracts/...
go test ./internal/... ./contracts/...
go vet ./...
```

Windows 本机可能因 duckdb/gjieba CGO 链接/初始化失败；以 Linux 容器构建和 CI 为准。RAG 侧 e2e 见 `../rag/docs/测试与评测.md`。

## 10. 相关项目文档

- 文档索引：`../docs/README.md`
- 角色权限：`../docs/02-需求与设计/角色生态位规定.md`
- 能力/路由：`../docs/02-需求与设计/能力_角色_页面路由需求.md`
- 部署：`../deployments/README.md`
- 前端：`../frontend/README.md`
- RAG：`../rag/README.md`

## License

见 `LICENSE` 与 `THIRD_PARTY_NOTICES.md`。
