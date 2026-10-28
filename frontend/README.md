# WellRead Frontend

WellRead 的 Vue 3 单页应用：知识库、问答、精读、共享空间、治理台、管理工作台和运维台。前端只负责展示与交互，权限最终由后端能力契约和资源守卫决定。

## 技术栈

- Vue 3 + TypeScript + Vite 7
- TDesign Vue Next
- Pinia（状态）、Vue Router（路由）
- vue-i18n（多语言）
- Markdown/Mermaid/KaTeX、Office 预览、xlsx 导入等展示能力

## 目录地图

| 路径 | 作用 |
|---|---|
| `src/api/` | 按业务域拆分的后端 API 封装 |
| `src/components/` | 通用组件（菜单、聊天、设置、模型选择等） |
| `src/composables/` | 可复用组合式逻辑 |
| `src/config/` | 能力契约、路由访问、设置分区、部署能力、管理卡片 |
| `src/directives/` | 全局指令 |
| `src/hooks/` | 页面级 hooks |
| `src/i18n/` | 语言包、审计动作文案、语言审计脚本 |
| `src/router/` | 路由表、路由守卫、403 |
| `src/stores/` | Pinia 状态（auth、ui、menu、chat 资源等） |
| `src/styles/` | 全局样式与主题扩展 |
| `src/types/` | 前后端共享类型/视图类型 |
| `src/utils/` | 请求、格式化、Markdown、引用等工具 |
| `src/views/` | 页面：auth/chat/knowledge/reading/shared-space/governance/management/ops/settings |
| `public/` | 静态资源；登录页架构图在 `public/diagrams/` |
| `design/` | 设计源（图标、字体），不进入生产构建 |
| `scripts/` | 前端专项脚本（构建、图标矢量化、PR 校验） |

## 本地开发

```bash
cd frontend
npm ci
npm run dev
```

- 开发地址：`http://localhost:5173`
- `/api`、`/files` 默认代理到 `http://localhost:6152`
- 可通过 `VITE_DEV_PROXY_TARGET` 或 `FRONTEND_BACKEND_URL` 改代理目标

## 构建与检查

```bash
npm run type-check       # vue-tsc
npm test                 # tsx --test
npm run scan-i18n-gaps   # 扫描缺失文案
npm run build            # 生产构建到 dist/
npm run preview          # 本地预览生产构建，4173
```

`npm run build-with-types` 并行跑 type-check 和 build，适合交付前使用。

## 能力契约与路由

- 单一事实源：`../app/contracts/capabilities.json`。
- `src/config/capabilities.ts` 直接导入该文件；Vite 允许读取仓库根下的 `app/contracts`。
- 路由通过 `meta.requiredCaps` / `meta.requiredAnyCaps` 粗筛；不满足进入 `/403`。
- 菜单、设置分区、管理卡片都由能力推导，不在页面里写“角色 if-else”。
- 后端资源守卫（部门、KB、ACL、CR、API Key）才是最终边界；前端隐藏不是安全措施。

路由命名空间：

- `/login`、`/register`、`/join`、`/403`：认证与错误页。
- `/platform/*`：公司工作台（知识库、问答、精读、共享空间、治理、管理、设置）。
- `/ops/*`：系统管理员运维工作台。
- `/knowledgeBase`：兼容入口。

## 数据与请求

- `src/api/` 按域封装；统一请求工具在 `src/utils/request`。
- 登录态、当前空间、角色/能力在 `src/stores/auth`；切换空间后菜单与路由重新按 caps 计算。
- 聊天/SSE、知识库、协作、治理、设置分别有独立 API 模块，不在页面里直接拼 URL。
- 公共能力变更优先改 `contracts/capabilities.json` 和 `src/config/*`，再改页面。

## i18n

- 主语言包：`src/i18n/locales/zh-CN.ts`。
- 新增文案用 `npm run scan-i18n-gaps` 定位缺失项；不要把中文文案硬编码在新增的通用组件里，业务页已有历史文案按现有模式处理。

## 资产与设计源

- 运行时资源：`src/assets/`（字体、图片、主题）。
- 设计源：`design/ui-source/`、`design/fonts/`、`design/brand/`；不进入镜像，映射见 `design/README.md`。
- 手绘图标矢量化：`scripts/vectorize_icon.py`；生成结果在聊天检索图标组件中。
- 新增 UI 组件应模仿同页同型组件，不另起一套视觉语言。

## 登录页与架构图

- 页面：`src/views/auth/Login.vue`。
- 左侧展示区为 Archify 生成的交互式架构图：`public/diagrams/wellread-architecture.html`。
- iframe 使用 `?embed=1&theme=light`；`Login.vue` 加载后移除 `data-embed`，注入登录页样式隐藏 Viewer 外壳，并调用 `Archify.motionGovernor.resume()` 开启流动与节点详情。
- 架构图为暖色 editorial、中文标签，重点展示身份与角色、共享与 ACL、协作与版本。
- 图源与校验产物在仓库根 `.archify/architecture-wellread-login-20261026/`；重新生成后需验证节点详情、流动动画和徽标隐藏。

## Docker 与 nginx

```bash
docker build \
  --build-context contracts=../app/contracts \
  -f Dockerfile \
  -t gaze-ui:latest .
```

- 构建阶段需要 `contracts` named context，产物不包含 `design/`。
- 运行阶段是 nginx，`nginx-api-proxy.conf` 关闭代理缓冲，适配 SSE。
- 容器端口 80；`MAX_FILE_SIZE_MB` 控制上传限制。

## 约定

- 页面可见性从 caps 派生；不要在前端自行扩大后端返回的数据范围。
- 新增路由必须声明 `requiredCaps` 或 `requiredAnyCaps`。
- API 返回的敏感字段不在前端日志/控制台打印。
- 大列表/长文档优先复用现有虚拟滚动、分页和 Markdown 渲染管线。
- 业务角色和页面边界以根目录 `docs/02-需求与设计/角色生态位规定.md` 为准。

## 相关文档

- 文档总入口：`../README.md`
- 角色与页面权限：`../docs/02-需求与设计/角色与权限/角色生态位规定.md`
- 能力与路由：`../docs/02-需求与设计/角色与权限/能力与路由矩阵.md`
- 界面规范：`../docs/02-需求与设计/界面与原型/界面规范.md`
- 设计源说明：`design/README.md`
