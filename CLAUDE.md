# CLAUDE.md — AI Agent 操作指南

本文件面向在本仓库工作的 AI Agent。项目术语、角色定义、能力矩阵不在本文件重复，统一读[术语与约定](docs/01-项目总览/术语与约定.md)与[角色生态位规定](docs/02-需求与设计/角色与权限/角色生态位规定.md)。

## 1. 仓库形态

WellRead 是公司内部单租户、本地部署的企业知识库系统，根目录为 monorepo：

| 目录 | 职责 |
|---|---|
| `app/` | Go 后端、docreader、CLI、Go SDK、Swagger、迁移 |
| `frontend/` | Vue 3 SPA；能力契约从 `app/contracts/capabilities.json` 读取 |
| `rag/` | Flask 检索/生成侧车；语料/模型/索引在 `data/rag/`（gitignore） |
| `mcp/` | MCP 外部接入层 |
| `deployments/` | 六服务 compose、hotreload overlay、systemd、env example |
| `scripts/` | 跨服务构建/运维脚本 |
| `docs/` | 01 总览、02 需求与设计、03 进度、04 历史归档、05 演示、images |

本地运行数据在仓库内 `data/rag/`（gitignore），不进入 git、不进入镜像。

## 2. 开工前阅读顺序

1. [当前进度](docs/03-进度与交接/当前进度.md)：当前阶段、待办、验证基线、风险。
2. [系统总览](docs/01-项目总览/系统总览.md)：服务生态、权限、问答/入库/协作链路。
3. 与任务对应的 `docs/02-需求与设计/` 文档。
4. [能力与路由矩阵](docs/02-需求与设计/角色与权限/能力与路由矩阵.md) 与[共享空间与ACL](docs/02-需求与设计/角色与权限/共享空间与ACL.md)：涉及权限、菜单、KB 时必读。
5. 历史快照只用于追溯，不作为当前实现依据。

## 3. 硬性规则

- **后端是安全边界**：前端隐藏不是授权；部门、KB、ACL、CR、API Key 最终由后端资源守卫判定。
- **能力码单一事实源**：`app/contracts/capabilities.json`；页面/菜单/路由从 caps 派生，不写角色 if-else。
- **不越角色**：company_admin 不进入 KB/问答/精读/共享空间；system_admin 不进入业务；knowledge_admin 不做共享/ACL/删 KB；editor 不删文档/建库；viewer 不提交 CR。
- **检索必须带 `acl_scope`**：缺 scope fail-closed；不提供未过滤兜底。
- **不记录员工原始问题**：检索审计只保留 query hash 与聚合；治理台不展示原始 query。
- **密钥不落库/不落文档**：`.env`、模型 Key、测试口令不提交；平台 API Key 只显示掩码。
- **RAG 数据不进入 git/镜像**：不把 `Data/`、`models/`、`faiss_index.bin` 加进 git；本地数据放 `data/rag/`，不要删除。
- **文档来源**：产品/需求文档只从 app 业务侧与当前产品文档获取；`rag/test/`、`data/rag/Data` 是独立测试数据，不能当作业务需求或演示数据。
- **只推个人仓库**：origin = `git@github.com:Forward-back/Well-Reard.git`（SSH；网页 `https://github.com/Forward-back/Well-Reard`）；不要添加或推送 Tencent/WeKnora 上游；提交前确认不含 `.env`、密钥、`data/rag/`、`backups/`。
- **高风险动作先确认**：删除/移动大量文件、重置数据库、重建数据卷、轮换凭据、修改部署与 CI，先向用户说明影响。

## 4. 常用命令

```bash
make up / down / restart / ps / logs
make dev / dev-logs / dev-stop
make test-app / test-frontend / test-rag / test-mcp / test
make images / package-lite
make migrate-up / migrate-down
make health
```

- 前端开发：`cd frontend && npm run dev`，`/api` 代理到 6152。
- RAG 语法：`cd rag && python -m compileall -q src`。
- MCP 测试：`cd mcp && uv sync --extra test && uv run pytest -q`。
- 本机 Windows 限制：`go build ./...` 可能被 duckdb 静态链接阻断；`internal/types` 测试可能因 gojieba CGO 崩溃。记录为环境限制，不要为此改业务代码。

## 5. 路径约定

- compose 项目名固定 `app`，沿用原命名卷（`app_postgres-data` 等）。
- 构建上下文：app/docreader `../app`；frontend `../frontend`；rag `../rag`。
- 前端镜像使用 named context `contracts=app/contracts`；`frontend/Dockerfile` 把契约复制到 `/app/contracts/capabilities.json`。
- RAG 挂载：`../data/rag:/app/external`（相对于 `deployments/`）；`RAG_DATA_DIR=/app/external/Data`、`LOCAL_MODELS_DIR=/app/external/models`；FAISS 索引写 `dirname(DATA_DIR)/faiss_index.bin`。
- 根 `scripts/` 只放跨服务脚本；Go 构建脚本在 `app/scripts/`；前端脚本在 `frontend/scripts/`。

## 6. 文档规范

- 文档状态只用：`当前`、`优化中`、`重构中`、`规划`、`历史`；日期只写在文档头。
- 当前文档名不带日期、Phase、批次；阶段记录放 `docs/03-进度与交接/阶段记录/`。
- `docs/` 下不放 README；根 `README.md` 是唯一文档总入口。
- 每份文档一个主题；产品/功能文档写“为什么用、怎么用、业务规则、界面意图、对比、差距、验收”，不复制 API 字段/数据库表结构。
- 角色矩阵只维护一份，其他文档引用[角色生态位规定](docs/02-需求与设计/角色与权限/角色生态位规定.md)。
- 历史归档保留原貌；需要时加“历史快照/旧路径对照”，不按当前口径重写。
- 文档与代码冲突时：以当前代码/运行结果为准，把差异写入对应文档的“当前实现与差距”，不要按旧文档直接改代码。

## 7. 改动流程

1. 读当前进度与对应 02 文档，确认任务属于当前阶段。
2. 涉及权限/路由：先改能力契约与资源守卫，再改页面；补角色矩阵与 403 测试。
3. 涉及检索/ACL：保证四路召回与引用复核都带 scope；补 fail-closed 与唯一 token 验收。
4. 涉及精读/技能：精读固定 `grill-with-docs`，技能只用于普通问答；两者不共享可执行工具。
5. 涉及文档：更新对应功能文档的“当前实现与差距”，需要时更新阶段记录，不新建重复文档。
6. 验证后记录命令、结果与限制；未验证的不要写成已验证。

## 8. 相关入口

- [系统总览](docs/01-项目总览/系统总览.md)
- [术语与约定](docs/01-项目总览/术语与约定.md)
- [角色生态位规定](docs/02-需求与设计/角色与权限/角色生态位规定.md)
- [当前进度](docs/03-进度与交接/当前进度.md)
- [部署手册](deployments/README.md)
- [RAG 文档](rag/README.md)、[MCP 文档](mcp/README.md)
- [app/README](app/README.md)、[frontend/README](frontend/README.md)
