# WellRead MCP Server

把 WellRead 知识库 API 暴露为 Model Context Protocol 工具，供 Claude Desktop、Cursor、KiloCode、VS Code Copilot 等 MCP 客户端调用。服务本身不存业务数据，所有读写都转发到 WellRead 后端。

## 快速开始

```bash
cd mcp
uv sync --extra test
export WELLREAD_BASE_URL="http://localhost:8080/api/v1"
export WELLREAD_API_KEY="<租户 API Key>"
uv run run_server.py
```

API Key 由 company_admin 创建。只做检索/问答用 `read` preset；需要摄入内容用 `write`；完整管理用 `admin` 或 full-access。详细步骤见 [安装与运行](docs/安装与运行.md)。

## 代码地图

| 文件 | 作用 |
|---|---|
| `gaze_mcp_server.py` | MCP 客户端封装、31 个工具、stdio/SSE/HTTP 传输 |
| `main.py` | 统一 CLI 入口，支持 `--transport`、`--check-only` |
| `run_server.py` | stdio 启动包装 |
| `run.py` | 最小便捷启动脚本 |
| `upload_paths.py` | 本地上传文件路径校验 |
| `tests/` | pytest/unittest 测试 |
| `scripts/check_imports.py` | 手工检查 MCP SDK 导入 |
| `pyproject.toml` / `setup.py` | 打包与 console entry points |
| `requirements.txt` / `uv.lock` | 运行依赖 |
| `Dockerfile` | HTTP 传输容器镜像 |

## 文档地图

| 文档 | 内容 |
|---|---|
| [安装与运行](docs/安装与运行.md) | 前置条件、API Key、环境变量、stdio/HTTP/SSE、Docker、故障排查 |
| [客户端接入](docs/客户端接入.md) | Claude Desktop、Cursor、KiloCode、远程 HTTP 的配置模板 |
| [工具参考](docs/工具参考.md) | 31 个工具的参数、用途、权限与错误语义 |
| [调用示例](docs/调用示例.md) | 检索、问答、摄入、更新、Agent、Gaze 的典型流程 |
| [架构与实现](docs/架构与实现.md) | 模块结构、请求链路、传输、鉴权、并发、安全、测试与限制 |

## 边界

- MCP 使用租户 API Key，不继承员工角色；部门共享空间等依赖部门上下文的接口可能返回 403。
- API Key 的权限由后端路由和 capability 决定；MCP 只做转发，不额外提权。
- 删除 KB、删除文档、删除分块、创建模型等工具具有破坏性，不要给不可信客户端 full-access Key。
- 网络传输必须设置 `MCP_SERVER_AUTH_TOKEN`；本机单用户优先 stdio。

## 测试

```bash
cd mcp
uv run pytest -q
uv run python scripts/check_imports.py
```
