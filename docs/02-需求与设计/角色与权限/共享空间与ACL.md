# 共享空间与 ACL

> 状态：当前
> 最后更新：2026-10-26
> 目标读者：产品、后端、前端、测试
> 关联：[角色生态位规定](角色生态位规定.md)、[能力与路由矩阵](能力与路由矩阵.md)、[知识库与文档管理](../功能与交互/知识库与文档管理.md)、[协作与版本控制](../功能与交互/协作与版本控制.md)

> 实施状态摘要：部门/角色、共享范围、访问判定、邀请部门字段、复制/副本共享行、共享空间/部门管理 UI 已落地；真实文件夹已完成（folder_id 查询/移动/拖拽/边界测试）；库内 ACL 全量 enforcement/UI 已完成；检索 ACL 核心已落地（acl_scope、FAISS/BM25/元数据分支过滤与引用复核、retrieval audit）。剩余：FAISS 原生 IDSelector、Text2SQL 线程级过滤、检索审计批量/保留期、显式“全部部门”标记、YLSW 账号种子。

> 2026-10-26 角色收口：company_admin 不再进入 KB/问答/共享空间，也不再兼任部门角色；KB 共享/发布/ACL 全部归 manager，knowledge_admin 不参与 ACL。

## 1. 背景与范围

公司内部知识平台采用“公司 → 部门 → 知识库 → 文件夹/文件”的组织结构。每个部门一个空间，可建立若干知识库；知识库可定向共享给其他部门，在对方“共享空间”中形成只读入口；库内文件夹/文件支持继承收窄与 deny 例外；检索必须在召回前完成权限过滤，任何越权内容不得进入大模型上下文。

本期范围：

- 部门实体与成员归属。
- 公司/部门两层角色矩阵与鉴权。
- 知识库归属部门、部门共享范围、共享空间只读入口。
- 真实文件夹实体，替换现有 `knowledges.folder_path`。
- 库内文件夹/文件 ACL（allow 收窄 + deny 例外）。
- 检索全路径 ACL（FAISS / BM25 / 元数据 / text-to-SQL）、fail-closed、复核、缓存失效与审计。
- 消费者门户 `visibility=public` 的独立 ACL 语义。
- 存量数据与角色迁移、测试账号闭环与验收。

非目标（V1 不做）：

- 部门层级/树形部门、用户组。
- 跨公司共享、外部组织协作。
- 字段级脱敏、属性基加密、文档分支。
- 独立策略引擎/OPA。
- 文件级 allow 放宽（文件只继承 + deny）。

## 2. 术语映射

| 术语 | 本系统实现 |
|---|---|
| 公司 | `tenant` |
| 部门空间 | 新增 `departments` 实体，隶属 tenant |
| 知识库 | `knowledge_bases`，新增 `department_id` |
| 共享范围 | 知识库可被哪些部门检索的部门集合，存 `kb_department_shares` |
| 共享空间 | 接收部门的只读入口投影，不复制文件；按共享范围计算 |
| 库内公开 | 文件夹 allow 主体为“全公司”，但被祖先 allow 与 KB 共享范围双重限制 |
| deny 例外 | 在继承允许集合上显式排除某用户、某部门或全公司 |
| 权限主体 | 用户、部门、全公司（allow 与 deny 共用主体类型；适用位置见 §6） |

## 3. 组织与角色模型

### 3.1 组织

- 每个用户属于且仅属于一个部门；公司管理员不持有部门角色。
- 部门归属生效于公司内所有业务权限；跨部门共享不改变用户在对方部门的角色。
- `tenant_members` 增加 `department_id`、`department_role`。
- 公司角色收敛为 `company_admin | member`；部门角色为 `manager | knowledge_admin | editor | viewer`。
- 公司管理员不兼任部门角色；即使历史数据存在，也不参与能力合并。

### 3.2 角色矩阵

| 能力 | company_admin | manager（本部门） | knowledge_admin（本部门） | editor（本部门） | viewer（本部门） | 跨部门共享入 |
|---|---|---|---|---|---|---|
| 公司成员/部门治理 | 是 | 否 | 否 | 否 | 否 | 否 |
| 调整本部门成员角色 | 否（不兼任部门角色） | 是 | 否 | 否 | 否 | 否 |
| 跨部门调动/移除公司成员 | 是 | 否 | 否 | 否 | 否 | 否 |
| 创建 KB | 否（不进入 KB） | 是 | 是 | 否 | 否 | 否 |
| 删除 KB | 否 | 是 | 否 | 否 | 否 | 否 |
| 修改 KB 共享范围 | 否 | 是 | 否 | 否 | 否 | 否 |
| 库内 ACL 管理 | 否 | 是（本部门 KB 内 allow/deny） | 否 | 否 | 否 | 否 |
| 文件夹结构（建/改名/移动/删） | 否 | 是 | 是 | 否 | 否 | 否 |
| 文档上传/新建 | 否 | 是 | 是 | 是 | 否 | 否 |
| 删除文档 | 否 | 是（任意） | 是（任意） | 否 | 否 | 否 |
| 内容编辑/CR | 否（不进入业务） | 是 | 是 | 是 | 否 | 否 |
| 检索/阅读本部门 KB | 否（无业务入口） | 是 | 是 | 是 | 是 | 是 |
| 检索/阅读共享入 KB | 否（无业务入口） | 只读 | 只读 | 只读 | 只读 | 只读 |
| issue/评论 | 否 | 是 | 是 | 是 | 是 | 是 |
| 下载/导出 | 否 | 是 | 是 | 是 | 是 | 否 |
| 转授共享 | 否 | 否 | 否 | 否 | 否 | 否 |

补充约束：

- 公司管理员不进入 KB 正文；只读范围仅限治理聚合、审计事件元数据与技能目录。不因公司治理权获得 KB 阅读、编辑、共享、ACL 或建库能力。
- 有知识库的部门必须至少保留 1 名 manager；公司至少保留 1 名 company_admin。
- 部门 manager 只能调整本部门已有成员的部门角色，不能跨部门调动，也不能移除公司成员。
- 共享入其他部门的成员：只读 + issue/评论；禁止编辑、提 CR、下载、管理 ACL、转授。

### 3.3 成员/部门管理

- company_admin：创建/重命名/删除部门；邀请/移除公司成员；跨部门调动；分配部门角色；可任命/更换 manager。
- 部门删除：存在成员、知识库、共享范围引用或 ACL 引用时拒绝删除；先清理或转移后再删。
- 新建成员：member 必须指定部门与部门角色；company_admin 不设部门角色。
- 部门新建：同一事务内物化“覆盖全部旧部门”的知识库共享范围，把新部门补入这些 KB。

## 4. 知识库归属与共享范围

### 4.1 数据模型

- `knowledge_bases.department_id`：所属部门，非空。创建时等于创建人部门；公司管理员不创建 KB。
- `kb_department_shares(id, tenant_id, kb_id, department_id, created_by, created_at)`：共享范围部门集合；所属部门始终保留一行，不允许移除。
- 不使用独立布尔“公开/内部”字段表达内部共享；`visibility` 仅保留消费者门户发布语义（§7.4）。

### 4.2 默认与全局公开

- KB 创建时共享范围 = `{所属部门}`。
- 选择“全部部门”：物化当前公司全部部门为共享行。
- 新建部门时：若某 KB 的共享行数量等于新建前的部门总数（即覆盖全部门），在同一事务把新部门补入；否则不补。
- 普通成员的主知识库列表仅展示本部门 KB；共享入 KB 只在共享空间展示。
- 公司管理员不进入知识库列表，不提供部门切换式 KB 只读视图。
- 共享空间入口以 KB 共享范围为准：共享范围包含本部门即展示入口；进入后按 ACL 裁剪文件夹与文档，无可见内容时显示空态。

## 5. 真实文件夹实体

### 5.1 模型

`knowledge_folders(id, tenant_id, kb_id, parent_id, name, path, depth, is_root, created_by, created_at, updated_at, deleted_at)`：

- 每个 KB 一个系统根目录（`is_root=true`），不可删除/改名/移动。
- `parent_id` 构成树；`path` 为计算/维护的展示路径，用于列表展示与兼容；ACL 永远以 `folder_id` 为准。
- 唯一约束：同一父级下名称唯一；每 KB 一个 root。
- `knowledges.folder_id` 非空，根目录文档挂 root folder。
- 目标状态不再使用 `knowledges.folder_path`；迁移期保留列供回滚，代码全部切 folder_id，验收后清理迁移删除。

### 5.2 操作语义

- 新建：manager/knowledge_admin；指定 parent + name。
- 重命名/移动：manager/knowledge_admin；移动禁止目标为自身或后代；事务内更新子树 path/depth。
- 删除：仅空文件夹可删；含子文件夹或文档返回冲突；删除写审计。
- 文档移动：editor+ 可移动文档，但只能移动到本部门 KB 的 folder；移动不改变文件级 deny。
- 旧 `folder_path` 请求参数在一个版本内翻译为 folder_id；响应返回 folder_id + 计算 path。对外文档与 Go client 同步更新。

### 5.3 迁移

- 按现有 `knowledges.folder_path` 的每一级前缀建树；文档挂到最深一级对应 folder。
- 根路径空字符串文档挂 root folder。
- 迁移脚本需校验：每 KB 仅一个 root、无孤儿 parent、path 与树一致、文档 folder_id 非空。

## 6. 库内 ACL

### 6.1 数据模型

`kb_acl_rules(id, tenant_id, kb_id, resource_type, resource_id, effect, subject_type, subject_id, created_by, created_at)`：

- `resource_type`: `folder` | `file`；`resource_id`: folder_id 或 knowledge_id。
- `effect`: `allow` | `deny`。
- `subject_type`: `user` | `department` | `all`；`subject_id` 对 all 为空。
- 唯一规则元组：同一资源、效果、主体只能有一条。
- 文件只允许 deny；文件 allow 直接拒绝（错误码明确）。
- 库内 ACL 仅 manager 可设置；knowledge_admin 不参与 ACL。
- `tenant_acl_epochs(tenant_id PK, epoch)`：任一共享/ACL/文件夹/成员/KB 部门变更在同一事务自增，用于缓存失效与检索 scope 版本。

### 6.2 继承算法

对用户 U 访问文件夹 F：

1. 从 KB 根到 F 逐层收集 allow 规则。某一层存在 allow 规则时，U 必须满足该层任一 allow 主体；所有层都必须通过（严格交集）。
2. `all` allow 表示“KB 共享范围内的所有成员”，不是全公司；仍受祖先 allow 限制。
3. 若 F 或任一祖先存在命中 U / U 部门 / 全公司的 deny，直接拒绝；deny 优先于所有 allow。
4. 文件：继承所在文件夹结果；文件自身 deny 命中即拒绝。文件没有 allow。
5. 库内公开 = 文件夹 allow `all`；其有效集合 = 祖先交集 ∩ KB 共享范围，不能突破父级收窄。
6. company_admin 不参与 KB 内容访问判定；治理台只消费聚合与审计元数据。未来如需合规审阅正文，应单独设计只读审阅角色，不恢复公司管理员穿透。
7. 跨部门共享入用户：仅按上述规则得到只读结果；不因本部门角色获得额外权限。

### 6.3 管理权限与审计

- 共享范围：仅本部门 manager（company_admin 无业务共享权）。
- ACL：全部归 manager；knowledge_admin 无 ACL 权限。
- 所有 ACL/共享/部门变更必须同步写 `audit_logs`，含操作人、时间、资源、变更前后主体集合；审计写入失败则事务回滚、变更失败。
- 被 deny 的文件：管理员在权限页可见标题/路径/规则，不可见正文；可撤销 deny。

### 6.4 示例

KB 共享范围 `{研发部, 市场部, 财务部}`：

- 文件夹 `/研发内部` allow 研发部：市场/财务用户不可见；公司管理员无业务入口，不参与 KB 可见性。
- 其子文件夹 `/研发内部/公开` allow 全公司：有效仍为 `{研发部}`（祖先交集），不会恢复市场/财务。
- `/项目Alpha` allow 用户 `mkt.viewer`：仅该用户通过（其部门在 KB 范围内）。
- 文件 `all-deny.md` deny 全公司：所有人（含 manager/创建者）正文不可见；manager 仍可在 ACL 管理页看到元数据并撤销规则。

## 7. 检索权限执行

### 7.1 计算与传输

- App 侧 ACL 服务为唯一权威，计算：
  - 可准入 KB：本部门 KB ∪ 共享范围含本部门的 KB。company_admin 不进入 KB 检索。
  - 可读 document_id 集合：展开文件夹继承、allow 交集、deny 例外。
- 请求携带 `acl_scope`：
  - `mode=all`：准入 KB 内全部文档可读（无收窄/deny 时）。
  - `mode=allow`：携带允许 document_id 列表（允许集小于排除集时）。
  - `mode=deny`：携带排除 document_id 列表。
  - `scope_version`：tenant ACL epoch。
- App 按 `(tenant, user, kb)` 缓存可读集，命中需校验 epoch；ACL/共享/文件夹/成员/KB 部门变更事务内 bump epoch。
- ACL 计算失败、超时、无部门（member）时：不调用引擎，返回错误或空结果。

### 7.2 YLSW 全路径执行

- `/chat`、`/chat/stream`、`/search` 必须解析 `acl_scope`；缺失或非法 fail-closed（拒绝/空结果）。
- FAISS：用 `SearchParametersHNSW(sel=IDSelectorBatch(...))` 在 ANN 遍历中过滤；若运行版本不支持 selector，则超额召回后按允许向量集掩码，并记录降级日志。
- BM25：使用现有 `allowed_chunk_ids`。
- 元数据检索：SQL JOIN `documents.source_doc_id`，只查允许文档。
- text-to-SQL：thread-local 注入 `tables_structured.document_id IN (...)`；SQL 结果进入上下文前复核。
- planner_multi / 多 query 变体：每个召回分支先应用 ACL，再融合；禁止先生成中间答案再过滤。
- 融合、rerank、邻居/父上下文扩展后，组装上下文与引用前做一次 Top-K 文档级复核；任何未通过项丢弃。
- 结果为空时返回“无权限/无结果”，不得回退到未过滤检索。

### 7.3 审计

- 检索审计写 `retrieval_audit_logs`：tenant、user、principal_kind（member/company_admin/portal）、session、query 摘要、可准入 KB、ACL scope 摘要、命中 document_id、判定结果、错误。
- 异步批量写 + 失败重试；默认保留 180 天，可配置；不阻断检索。
- 权限/共享/deny 变更审计仍走 `audit_logs` 同步链路（§6.3）。

### 7.4 消费者门户

- `visibility=public` 仅 manager 可设置；设置前置条件：KB 共享范围 = 全部部门。
- 匿名 principal = 全公司主体：
  - 仅 `allow all` 的文件夹通过；部门级/用户级 allow 不生效。
  - `deny all` 隐藏；`deny` 部门/用户对匿名不适用（匿名不属于任何部门/用户）。
  - KB 必须 public 且共享全部门才进入门户候选。
- 门户检索单独记录 portal 审计，不复用内部用户身份。

## 8. 共享空间 UI

- 左侧“共享空间”菜单对所有成员可见，不再依赖 admin/org capability；路由改为部门共享页，旧组织页面下线。
- 共享空间列表：共享入本部门的 KB，按来源部门分组；显示只读标识、KB 标题、来源部门、更新时间。
- 进入共享 KB：只读详情/检索/文档浏览；隐藏编辑、上传、删除、CR、共享、ACL、下载入口；保留 issue/评论。
- 无可见文档时展示空态，但 KB 入口仍按共享范围展示。
- 主知识库列表显示“当前部门”；不提供 company_admin 的跨部门 KB 只读切换。
- KB 设置内“共享范围”改为部门多选：所属部门锁定，支持“全部部门”；仅 manager 可编辑。
- 库内 ACL UI：文件夹树/文件操作菜单进入权限面板，展示 allow/deny 规则与有效权限预览；全公司 deny 仅 manager 可见。

## 9. 数据模型与迁移

新增/变更表：

- `departments`（tenant_id, name, is_default, ...）。
- `tenant_members`：+ department_id, department_role；role 收敛 company_admin/member。
- `knowledge_bases`：+ department_id。
- `kb_department_shares`。
- `knowledge_folders`；`knowledges`：+ folder_id（目标删除 folder_path）。
- `kb_acl_rules`；`tenant_acl_epochs`。
- `retrieval_audit_logs`。

迁移编号（按当前最大值顺延）：

- PG `000107_departments`、`000108_invitation_department`、`000109_knowledge_folders`、`000110_kb_acl`、`000111_retrieval_audit`。
- SQLite `000019_departments`、`000020_invitation_department`、`000021_knowledge_folders`、`000022_kb_acl`、`000023_retrieval_audit`。

存量迁移：

- 每租户创建“默认部门”；所有现有成员归入；所有现有 KB 归入默认部门。
- 现有 KB 共享范围 = `{默认部门}`；`visibility=public` 的 KB 额外物化全部门。
- 旧 owner/admin → company_admin，并兼任默认部门 manager。
- 旧 contributor 若创建过 KB → knowledge_admin；否则 editor。
- 旧 viewer → viewer。
- 旧组织/跨租户共享（`kb_shares`、organization 路由/UI）不再读写，保持停用；不迁移旧共享数据。
- 文件夹按 §5.3 回填；验收通过后清理 `folder_path`。

## 10. 测试账号与验收闭环

公司：YLSW（tenant）。部门：研发部、市场部、财务部。以下为共享空间专项回归账号，日常六角色测试账号见[角色生态位规定](角色生态位规定.md) §1.1。

| 账号 | 公司角色 | 部门/部门角色 |
|---|---|---|
| `admin@test` | company_admin | 无部门角色 |
| `dev.manager@test` | member | 研发部 / manager |
| `dev.knowledge@test` | member | 研发部 / knowledge_admin |
| `dev.editor@test` | member | 研发部 / editor |
| `dev.viewer@test` | member | 研发部 / viewer |
| `mkt.manager@test` | member | 市场部 / manager |
| `mkt.viewer@test` | member | 市场部 / viewer |
| `fin.viewer@test` | member | 财务部 / viewer |

知识库与 ACL：

- `KB-R`（研发部），共享 `{研发部, 市场部}`：
  - `/公共资料` allow all（KB 范围内）
  - `/研发内部` allow 研发部
  - `/项目Alpha` allow 研发部 + 用户 `mkt.viewer`
  - `/受限` allow 研发部；内含 `all-deny.md` deny 全公司
- `KB-R2`（研发部），仅共享研发部。
- `KB-M`（市场部），共享 `{市场部, 研发部}`。
- `KB-F`（财务部），仅共享财务部。

验收断言：

1. 登录/菜单/按钮差异符合 §3.2；company_admin 无 KB/问答/共享空间入口，直接访问返回 403。
2. `dev.manager` 可改 KB-R 共享范围、ACL（含全公司 deny）；`dev.knowledge` 不可改共享范围、不可设全公司 deny。
3. `mkt.viewer` 共享空间可见 KB-R；可见 `/公共资料`、`/项目Alpha`；不可见 `/研发内部`、`/受限`。
4. `mkt.manager` 不可见 `/项目Alpha`（用户级 allow 只给 mkt.viewer）；不可见 `/研发内部`。
5. `fin.viewer` 看不到 KB-R/KB-M；`dev.viewer` 看不到 KB-F。
6. 唯一 token 检索：`mkt.viewer` 搜不到 `/研发内部` 与 all-deny 文件；`dev.manager` 也搜不到 all-deny 正文；company_admin 无检索入口；SQL/元数据/向量/BM25 四路径一致。
7. 共享范围移除市场部后，mkt 的共享空间入口与检索结果在约定延迟内消失；重新加入后恢复。
8. all-deny 文件对 manager 只显示元数据与规则，可撤销；撤销后按 allow 恢复可见；company_admin 无业务入口。
9. 共享入 KB 无编辑/CR/下载入口；issue/评论可用。
10. `role_company_admin` 不能进入 KB/问答/共享空间；治理台只返回聚合与审计元数据；不能调用共享/ACL。
11. 门户匿名：public + 全部门共享的 KB 中，仅 allow all 文件夹可见；all-deny 不可见；部门/用户 allow 不生效。
12. 审计：ACL/共享变更可还原操作人、时间、前后主体集合；检索审计可还原用户、可准入 KB、过滤摘要、命中 document_id 与判定。

## 11. 实施阶段

1. **地基**：部门/角色迁移、KB 归属、共享范围、成员/部门 API、RBAC 守卫。
2. **文件夹**：真实文件夹实体、folder_id 全链路替换、文件夹 CRUD、前端树/上传/移动。
3. **ACL**：规则表、继承评估器、ACL/共享 API、缓存 epoch、变更审计、单测。
4. **能力路由**：能力契约、membership caps、路由 `required_caps`、统一 403、设置分区能力化、内部 Agent 面裁剪、技能权限基础。详见[能力与路由矩阵](能力与路由矩阵.md)。
5. **检索**：acl_scope、YLSW 全路径过滤、fail-closed、复核、门户 ACL、检索审计。
6. **技能/精读**：技能问答注入、ReadingRun 交互式文档精读。详见[技能与问答注入](../功能与交互/技能与问答注入.md)、[交互式精读](../功能与交互/交互式精读.md)。
7. **UI/验收**：共享空间页、KB 共享/ACL UI、设置委派 UI、治理台、精读陪读 UI、种子账号、E2E 闭环。详见[治理台与审计](../功能与交互/治理台与审计.md)、[部门管理工作台](../功能与交互/部门管理工作台.md)。

## 12. 风险与开放项

- `folder_path` 全量替换回归面大：迁移需 dry-run/回填校验，保留 down 与一版兼容参数。
- FAISS IDSelector 依赖版本；增加能力检测与超额召回降级，降级必须记录。
- ACL epoch 漏 bump 会导致缓存越权：所有变更统一走事务 helper，代码评审重点检查。
- text-to-SQL 过滤遗漏：所有结构化表必须带 document_id；SQL 路径统一入口注入，E2E 用唯一 token 验证。
- 部门删除与 ACL/共享引用：本期采用“有引用即拒绝删除”，避免悬挂规则。
- 公司管理员不进入 KB 正文；治理数据只到聚合与审计事件元数据。后续如需合规审阅正文，应单独设计只读审阅角色与审计，而不是恢复公司管理员穿透。
- 检索审计为异步：需监控写入失败与积压，保留 180 天可配置。
