package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gaze/internal/handler"
	"gaze/internal/types"
)

// RegisterChunkerDebugRoutes wires the read-only chunker preview endpoint
// used by the KB editor's debug panel. Stateless — uses no service deps.
//
// Viewer+ floor: the endpoint surfaces inside the tenant UI, so any
// authenticated tenant member can call it; revoked accounts whose JWT
// has not yet expired are kept out by the role check, matching the
// rest of the RBAC matrix in this file.
func RegisterChunkerDebugRoutes(r *gin.RouterGroup, g *rbacGuards) {
	g.apiKeyRoute(r, http.MethodPost, "/chunker/preview", apiKeyRetrieve(apiKeyIngest(apiKeyFullAccess())), g.Viewer(), handler.PreviewChunking)
}

// RegisterChunkRoutes 注册分块相关的路由
//
// Mutating routes addressed via :knowledge_id inherit per-KB ownership
// from the owning knowledge entry's KB (PR 5, #1303); the chain hop is
// shared with RegisterKnowledgeRoutes via OwnedChunkKBOrAdmin so the
// same "creator-of-the-KB OR Admin+" rule applies to chunk edits.
func RegisterChunkRoutes(r *gin.RouterGroup, handler *handler.ChunkHandler, g *rbacGuards) {
	// 分块路由组。Scoped API key 需要 ingest 能力写内容，retrieve 能力读内容；
	// 两者仍受 KB 白名单约束。
	chunks := g.apiKeyGroup(r.Group("/chunks"), apiKeyIngest(apiKeyFullAccess()))
	chunkRead := chunks.With(apiKeyRetrieve(apiKeyFullAccess()))
	{
		// 获取分块列表 — Viewer+ 且对父 KB 有 read 权限（own / shared / via shared agent）
		chunkRead.GET("/:knowledge_id", g.Viewer(), g.KBAccessReadFromKnowledgeIDParam("knowledge_id"), handler.ListKnowledgeChunks)
		// 通过chunk_id获取单个chunk（不需要knowledge_id） — Viewer+ 且对父 KB 有 read 权限
		chunkRead.GET("/by-id/:id", g.Viewer(), g.KBAccessReadFromChunkIDParam("id"), handler.GetChunkByIDOnly)
		chunkRead.GET("/:knowledge_id/:id/revisions", g.Viewer(), g.KBAccessReadFromKnowledgeIDParam("knowledge_id"), handler.ListChunkRevisions)
		// 删除分块 — KB owner OR Admin+，且对父 KB 有 write 权限
		chunks.DELETE("/:knowledge_id/:id", g.OwnedChunkKBOrAdmin(), g.KBAccessWriteFromKnowledgeIDParam("knowledge_id"), handler.DeleteChunk)
		// 删除知识下的所有分块 — KB owner OR Admin+，且对父 KB 有 write 权限
		chunks.DELETE("/:knowledge_id", g.OwnedChunkKBOrAdmin(), g.KBAccessWriteFromKnowledgeIDParam("knowledge_id"), handler.DeleteChunksByKnowledgeID)
		// 更新分块信息 — KB owner OR Admin+，且对父 KB 有 write 权限
		chunks.PUT("/:knowledge_id/:id", g.OwnedChunkKBOrAdmin(), g.KBAccessWriteFromKnowledgeIDParam("knowledge_id"), handler.UpdateChunk)
		chunks.POST("/:knowledge_id/:id/revert", g.OwnedChunkKBOrAdmin(), g.KBAccessWriteFromKnowledgeIDParam("knowledge_id"), handler.RevertChunk)
		// 删除单个生成的问题（通过分块 id） — 与其它 chunk mutation 一致：
		// KB owner OR Admin+。早期这里因为链路 (chunk_id -> knowledge_id ->
		// kb -> creator_id) 还没接通，被临时降级成 Contributor，导致一个
		// 「能编辑所有 chunk 的同样规则在这条路由上反而更宽松」的不一致。
		// 现在通过 KBCreatorLookupFromChunkIDParam 把那一跳补上，统一矩阵。
		chunks.DELETE("/by-id/:id/questions", g.OwnedChunkKBOrAdminFromChunkID(), g.KBAccessWriteFromChunkIDParam("id"), handler.DeleteGeneratedQuestion)
		chunks.PUT("/by-id/:id/questions", g.OwnedChunkKBOrAdminFromChunkID(), g.KBAccessWriteFromChunkIDParam("id"), handler.UpsertGeneratedQuestion)
		chunks.POST("/by-id/:id/questions/regenerate", g.OwnedChunkKBOrAdminFromChunkID(), g.KBAccessWriteFromChunkIDParam("id"), handler.RegenerateGeneratedQuestions)
	}
}

// RegisterKnowledgeRoutes 注册知识相关的路由
//
// Per-KB ownership applies on the per-:id mutating routes (PR 5,
// #1303): the URL :id is a knowledge id, OwnedKnowledgeKBOrAdmin
// walks it back to KB.CreatorID so a Contributor who owns the KB can
// edit/delete any of its documents while a non-owner Contributor gets
// 403. KB-scoped upload routes (`/knowledge-bases/:id/knowledge/...`)
// reuse OwnedKBOrAdmin because the URL :id is the KB id directly.
// Body-scoped batch operations have a Contributor route gate and resolve
// their KB ownership plus Editor operation grant inside the handler.
func RegisterKnowledgeRoutes(r *gin.RouterGroup, handler *handler.KnowledgeHandler, g *rbacGuards) {
	// 知识库下的知识路由组（URL :id is the KB id）。Scoped API key 需要
	// ingest 能力才能写内容，且仍受 KB 范围限制；清空 KB 只允许 full-access key。
	kb := g.apiKeyGroup(r.Group("/knowledge-bases/:id/knowledge"), apiKeyIngest(apiKeyFullAccess()))
	kbRead := kb.With(apiKeyRetrieve(apiKeyFullAccess()))
	{
		kb.POST("/file", g.OwnedKBOrAdmin(), g.KBAccessWrite("id"), handler.CreateKnowledgeFromFile)
		kb.POST("/url", g.OwnedKBOrAdmin(), g.KBAccessWrite("id"), handler.CreateKnowledgeFromURL)
		kb.POST("/manual", g.OwnedKBOrAdmin(), g.KBAccessWrite("id"), handler.CreateManualKnowledge)
		kbRead.GET("", g.Viewer(), g.KBAccessRead("id"), handler.ListKnowledge)
		kbRead.GET("/folders", g.Viewer(), g.KBAccessRead("id"), handler.ListKnowledgeFolders)
		kb.POST("/folders", g.DepartmentRole(types.DepartmentRoleKnowledgeAdmin), g.KBAccessWrite("id"), handler.CreateKnowledgeFolder)
		kb.PUT("/folders/:folder_id", g.DepartmentRole(types.DepartmentRoleKnowledgeAdmin), g.KBAccessWrite("id"), handler.UpdateKnowledgeFolder)
		kb.DELETE("/folders/:folder_id", g.DepartmentRole(types.DepartmentRoleKnowledgeAdmin), g.KBAccessWrite("id"), handler.DeleteKnowledgeFolder)
		// In-KB ACL: manager/knowledge_admin only; mutations additionally
		// require an Editor operation grant on the owning KB.
		acl := g.apiKeyGroup(r.Group("/knowledge-bases/:id/acl"), apiKeyFullAccess())
		acl.GET("", g.DepartmentRole(types.DepartmentRoleManager), g.KBAccessRead("id"), handler.ListKBAcl)
		acl.GET("/epoch", g.DepartmentRole(types.DepartmentRoleManager), g.KBAccessRead("id"), handler.GetKBAclEpoch)
		acl.GET("/effective", g.DepartmentRole(types.DepartmentRoleManager), g.KBAccessRead("id"), handler.GetKBAclEffective)
		acl.POST("", g.DepartmentRole(types.DepartmentRoleManager), g.KBAccessWrite("id"), handler.CreateKBAcl)
		acl.DELETE("/:rule_id", g.DepartmentRole(types.DepartmentRoleManager), g.KBAccessWrite("id"), handler.DeleteKBAcl)
		// Clearing all contents under a KB is a destructive op; gate
		// behind Admin instead of Contributor.
		kb.With(apiKeyFullAccess()).DELETE("", g.Admin(), g.KBAccessWrite("id"), handler.ClearKnowledgeBaseContents)
	}

	// 知识路由组（URL :id is a knowledge id; the guard walks it to the parent KB）
	kgrp := r.Group("/knowledge")
	k := g.apiKeyGroup(kgrp, apiKeyIngest(apiKeyFullAccess()))
	kRead := k.With(apiKeyRetrieve(apiKeyFullAccess()))
	{
		// Cross-knowledge endpoints (no :id) can't be gated on a single
		// KB via the URL — they accept a kb_id (or source/target KB) in the
		// body and the handler fans out the access check itself. /batch and
		// /search are read routes (retrieve). /move, /batch-delete,
		// /batch-reparse and /tags are content writes that each bound
		// themselves to a single (or source+target) KB and enforce the API
		// key's KB allow-list downstream — MoveKnowledge via
		// requireTenantAPIKeyKnowledgeBases(source,target); the batch ops via
		// validateKnowledgeBaseAccessWithKBID (requireTenantAPIKeyKnowledgeBase)
		// plus a per-item "belongs to the authorized KB" check in the handler
		// and service. They are therefore declared for API keys under the
		// ingest capability, matching their single-document siblings
		// (k.DELETE("/:id"), k.POST("/:id/reparse"), k.PUT("/:id")).
		kRead.GET("/batch", g.Viewer(), handler.GetKnowledgeBatch)
		kRead.GET("/metadata-fields", g.Viewer(), handler.ListMetadataFieldDefs)
		kRead.GET("/:id", g.Viewer(), g.KBAccessReadFromKnowledgeIDParam("id"), handler.GetKnowledge)
		kRead.GET("/:id/stages", g.Viewer(), g.KBAccessReadFromKnowledgeIDParam("id"), handler.GetKnowledgeSpans)
		kRead.GET("/:id/spans", g.Viewer(), g.KBAccessReadFromKnowledgeIDParam("id"), handler.GetKnowledgeSpans)
		k.DELETE("/:id", g.OwnedKnowledgeKBOrAdmin(), g.KBAccessWriteFromKnowledgeIDParam("id"), handler.DeleteKnowledge)
		k.PUT("/:id", g.OwnedKnowledgeKBOrAdmin(), g.KBAccessWriteFromKnowledgeIDParam("id"), handler.UpdateKnowledge)
		k.POST("/:id/regenerate-summary", g.OwnedKnowledgeKBOrAdmin(), g.KBAccessWriteFromKnowledgeIDParam("id"), handler.RegenerateKnowledgeSummary)
		k.PUT("/manual/:id", g.OwnedKnowledgeKBOrAdmin(), g.KBAccessWriteFromKnowledgeIDParam("id"), handler.UpdateManualKnowledge)
		k.POST("/:id/reparse", g.OwnedKnowledgeKBOrAdmin(), g.KBAccessWriteFromKnowledgeIDParam("id"), handler.ReparseKnowledge)
		k.POST("/:id/cancel-parse", g.OwnedKnowledgeKBOrAdmin(), g.KBAccessWriteFromKnowledgeIDParam("id"), handler.CancelKnowledgeParse)
		// Downloading exposes the original source file, so it has a stricter
		// boundary than viewing parsed content or previewing it: tenant Viewers
		// cannot download from their own workspace, and org-shared Viewer access
		// cannot download from the source workspace. API keys still follow the
		// retrieve capability declared by kRead; role guards intentionally defer
		// machine-principal authorization to the API-key gate.
		kRead.GET("/:id/download", g.Contributor(), g.KBAccessWriteFromKnowledgeIDParam("id"), handler.DownloadKnowledgeFile)
		kRead.GET("/:id/preview", g.Viewer(), g.KBAccessReadFromKnowledgeIDParam("id"), handler.PreviewKnowledgeFile)
		k.PUT("/image/:id/:chunk_id", g.OwnedKnowledgeKBOrAdmin(), g.KBAccessWriteFromKnowledgeIDParam("id"), handler.UpdateImageInfo)
		kRead.GET("/search", g.Viewer(), handler.SearchKnowledge)
		kRead.GET("/move/progress/:task_id", g.Viewer(), handler.GetKnowledgeMoveProgress)
		// Batch / cross-KB content writes: JWT Contributor+, or an API key
		// with the ingest capability (or full access). Each handler binds the
		// operation to a single (move: source+target) KB and rejects any KB
		// or knowledge id outside the key's allow-list, so a scoped ingest key
		// can only touch KBs it is already permitted to write.
		k.PUT("/tags", g.Contributor(), handler.UpdateKnowledgeTagBatch)
		k.POST("/batch-reparse", g.Contributor(), handler.BatchReparseKnowledge)
		k.POST("/batch-delete", g.Contributor(), handler.BatchDeleteKnowledge)
		k.POST("/folder", g.Contributor(), handler.MoveKnowledgeToFolder)
		k.POST("/move", g.Contributor(), handler.MoveKnowledge)
	}
}

// RegisterKnowledgeBaseRoutes 注册知识库相关的路由
func RegisterKnowledgeBaseRoutes(r *gin.RouterGroup, handler *handler.KnowledgeBaseHandler, g *rbacGuards) {
	// 知识库路由组。API-key 可达性按能力分两档，全部通过 apiKeyGroup 声明，
	// 不要再用裸 kbgrp.Handle 注册（那会绕过网关、对所有 key 静默默认拒绝）：
	//
	//   1. 读取（list/detail/search/progress/move-targets）—— retrieve OR full-access（kb）
	//   2. KB 生命周期管理（create/copy/duplicate/update/delete）
	//      —— manage_kbs OR full-access（kbManagement）
	//
	// 第 2 档整条 KB 生命周期共用同一策略：manage_kbs 是「管理知识库」capability，
	// 建/拷/改/删都是它的分内事。KB 的 allow-list 仍在下游生效——copy/duplicate/
	// update/delete 的目标 KB 会被 allow-list 兜住；create 无源可约束，限定 allow-list
	// 的 key 建出的新 KB 落在其 allow-list 之外（同租户、无越权，只是建完自己管不到），
	// 空 allow-list 的 key 则是全租户 KB 管理、新建天然在范围内。KB 内容写入（文档/
	// 分块/FAQ/Tag/Gaze）由对应子路由的 ingest 能力控制，不在本组。
	kbgrp := r.Group("/knowledge-bases")
	kb := g.apiKeyGroup(kbgrp, apiKeyRetrieve(apiKeyFullAccess()))
	kbManagement := kb.With(apiKeyManageKnowledgeBases(apiKeyFullAccess()))
	{
		// 共享空间：共享入本部门的只读 KB 投影。
		r.GET("/shared-space/knowledge-bases", g.Viewer(), handler.ListSharedKnowledgeBases)
		// 创建知识库 — 本部门 manager / knowledge_admin；company_admin 仅在兼任部门角色时可用。
		kbManagement.POST("", g.DepartmentRole(types.DepartmentRoleKnowledgeAdmin), handler.CreateKnowledgeBase)
		// 获取知识库列表 — Viewer+ for JWT callers; retrieve-capable API keys pass via the gate.
		kb.GET("", g.ViewerOrSystemAdmin(), handler.ListKnowledgeBases)
		// 获取知识库详情 — Viewer+ 且对 KB 有 read 权限
		kb.GET("/:id", g.Viewer(), g.KBAccessRead("id"), handler.GetKnowledgeBase)
		// 更新/删除知识库 — 两层正交鉴权，缺一不可：
		//   OwnedKBOrAdmin  管「租户内」归属：非创建者的 Contributor 改不了
		//                   同事的 KB（跨租户 KB 在此走 lookup=NotFound → 交给
		//                   下游处理，不在此拦）。
		//   KBAccessWrite   管「跨租户」访问级：自有 KB 或被组织共享(editor)。
		// handler 内再按 permission/所有者租户做最终判定 —— 尤其 DeleteKnowledgeBase
		// 以调用者「自身」租户(c.Keys，未被 KBAccess 改写)校验 kb.TenantID，
		// 把删除锁死为「所有者租户 + Admin」，共享 editor 无法删除源 KB。
		// Department manager/knowledge_admin write access is enforced by
		// KBAccessWrite (own-department role ladder). OwnedKBOrAdmin used to
		// run first and blocked them even though the matrix grants edit.
		kbManagement.PUT("/:id", g.KBAccessWrite("id"), handler.UpdateKnowledgeBase)
		kbManagement.DELETE("/:id", g.KBAccessWrite("id"), handler.DeleteKnowledgeBase)
		// 消费者门户发布标记 — JWT Admin+ 专属(Owner 权限内);API key 走 manage_kbs。
		// 发布是对外暴露动作,不做 OwnedKBOrAdmin:即使 KB 创建者(Contributor)也无权对外发布(§2.13)。
		kbManagement.PUT("/:id/visibility", g.KBAccessWrite("id"), handler.SetKnowledgeBaseVisibility)
		// 部门共享范围：读取对 KB 有 read 权限即可；修改仅所属部门 manager。
		kb.GET("/:id/shares", g.Viewer(), g.KBAccessRead("id"), handler.GetKBDepartmentShares)
		kbManagement.PUT("/:id/shares", g.DepartmentRole(types.DepartmentRoleManager), g.KBAccessRead("id"), handler.SetKBDepartmentShares)
		// 置顶/取消置顶知识库 — 创建者本人 OR Admin+ 且对 KB 有 write 权限
		// Pin state is now per-(user, kb) (migration 000050). Anyone with
		// at least Viewer-level read access to the KB — including users
		// who reached it via a shared agent — may pin it for themselves;
		// no edit permission is required. The OwnedKBOrAdmin guard was
		// removed accordingly. The route still requires KB read access
		// so callers can't poke at KBs they can't see.
		kb.PUT("/:id/pin", g.Viewer(), g.KBAccessRead("id"), handler.TogglePinKnowledgeBase)
		// 混合搜索 — Viewer+ 且对 KB 有 read 权限 (read-only)
		// POST is preferred; GET with JSON body is kept for backward compatibility (#1727).
		kb.POST("/:id/hybrid-search", g.Viewer(), g.KBAccessRead("id"), handler.HybridSearch)
		kb.GET("/:id/hybrid-search", g.Viewer(), g.KBAccessRead("id"), handler.HybridSearch)
		// 拷贝知识库 — 产出新 KB，与 create 同档：JWT Contributor+，API key 需 manage_kbs 或 full-access。
		// 源 KB 通过 body 里的 source_id 传入（非 :id 路径参数），无法套用基于路径参数
		// 的 KBAccessRead，故源/目标 KB 的租户归属与 allow-list 校验在 handler 内完成
		// （requireTenantAPIKeyKnowledgeBases 会把 source_id/target_id 兜进 allow-list）。
		// 副本归调用者所有，不需要原 KB 的所有权。
		kbManagement.POST("/copy", g.DepartmentRole(types.DepartmentRoleKnowledgeAdmin), handler.CopyKnowledgeBase)
		// 创建知识库副本 — 产出新 KB，与 create 同档：JWT Contributor+，API key 需 manage_kbs 或 full-access；
		// 且对源 KB 有 read 权限（KBAccessRead 会对限定 key 兜住源 KB）。只创建新的 KB 设置记录，不复制内容/索引/分享。
		kbManagement.POST("/:id/duplicate", g.DepartmentRole(types.DepartmentRoleKnowledgeAdmin), g.KBAccessRead("id"), handler.DuplicateKnowledgeBase)
		// 获取知识库复制进度 — Viewer+；只读。manage_kbs（发起 copy 的 key）或
		// retrieve 均可轮询；任务按租户隔离（requireTaskProgressTenant），key 只能
		// 查本租户任务。
		kb.With(apiKeyRetrieve(apiKeyManageKnowledgeBases(apiKeyFullAccess()))).
			GET("/copy/progress/:task_id", g.Viewer(), handler.GetKBCloneProgress)
		// 获取可移动目标知识库列表 — Viewer+ 且对 KB 有 read 权限
		kb.GET("/:id/move-targets", g.Viewer(), g.KBAccessRead("id"), handler.ListMoveTargets)
	}
}

// RegisterKnowledgeBaseActivityRoutes exposes the read-only per-KB activity
// feed. It intentionally stays JWT-only: audit history is a sensitive owner
// surface and no existing workspace API-key capability grants audit access.
func RegisterKnowledgeBaseActivityRoutes(r *gin.RouterGroup, auditHandler *handler.AuditLogHandler, g *rbacGuards) {
	if auditHandler == nil {
		return
	}
	r.GET("/knowledge-bases/:id/activity",
		g.OwnedKBOrAdmin(), g.KBAccessRead("id"), auditHandler.ListKnowledgeBaseActivity)
}

// RegisterKnowledgeTagRoutes 注册知识库标签相关路由。
//
// Tags are KB metadata: Viewer reads, Contributor writes. Per-KB
// ownership granularity for tags is out of scope for PR 2; this is
// purely role-based.
func RegisterKnowledgeTagRoutes(r *gin.RouterGroup, tagHandler *handler.TagHandler, g *rbacGuards) {
	if tagHandler == nil {
		return
	}
	// Tags 是 KB 的子资源 — 创建/编辑/删除标签会改变 KB 内容的检索分类
	// 行为，应该与 KB 主体的"creator OR Admin+"矩阵一致，避免一个无
	// 关 Contributor 在他人 KB 里乱建/删标签影响 KB owner 的内容组织。
	kbTags := g.apiKeyGroup(r.Group("/knowledge-bases/:id/tags"), apiKeyIngest(apiKeyFullAccess()))
	kbTagsRead := kbTags.With(apiKeyRetrieve(apiKeyFullAccess()))
	{
		// KBAccessRead/Write resolve own/shared/agent-visible access and
		// rewrite the request's tenant context to the effective tenant
		// for the duration of the handler — so the handler no longer
		// needs its own effectiveCtxForKB helper.
		kbTagsRead.GET("", g.Viewer(), g.KBAccessRead("id"), tagHandler.ListTags)
		kbTags.POST("", g.OwnedKBOrAdmin(), g.KBAccessWrite("id"), tagHandler.CreateTag)
		kbTags.PUT("/:tag_id", g.OwnedKBOrAdmin(), g.KBAccessWrite("id"), tagHandler.UpdateTag)
		kbTags.DELETE("/:tag_id", g.OwnedKBOrAdmin(), g.KBAccessWrite("id"), tagHandler.DeleteTag)
	}
}

// RegisterGazePageRoutes registers gaze page related routes.
//
// Gaze pages are KB content (gaze mode): reads are Viewer+ and gated by
// KBAccessRead (own / org-shared / via shared agent), matching FAQ /
// chunk / tag read routes. Content mutations (create/update/delete) and
// maintenance actions (rebuild-links, auto-fix, change issue status)
// honour per-KB ownership via OwnedGazeKBOrAdmin (PR 5, #1303): the URL
// :kb_id resolves directly to the owning KB so a Contributor who owns
// the KB can manage its gaze, while a non-owner Contributor gets 403.
func RegisterGazePageRoutes(r *gin.RouterGroup, gazeHandler *handler.GazePageHandler, g *rbacGuards) {
	gaze := g.apiKeyGroup(r.Group("/knowledgebase/:kb_id/gaze"), apiKeyIngest(apiKeyFullAccess()))
	gazeRead := gaze.With(apiKeyRetrieve(apiKeyFullAccess()))
	{
		// Page CRUD
		gazeRead.GET("/pages", g.Viewer(), g.KBAccessRead("kb_id"), gazeHandler.ListPages)
		gaze.POST("/pages", g.OwnedGazeKBOrAdmin(), g.KBAccessWrite("kb_id"), gazeHandler.CreatePage)
		gaze.PUT("/move-page", g.OwnedGazeKBOrAdmin(), g.KBAccessWrite("kb_id"), gazeHandler.MovePage)
		gazeRead.GET("/pages/*slug", g.Viewer(), g.KBAccessRead("kb_id"), gazeHandler.GetPage)
		gaze.PUT("/pages/*slug", g.OwnedGazeKBOrAdmin(), g.KBAccessWrite("kb_id"), gazeHandler.UpdatePage)
		gaze.DELETE("/pages/*slug", g.OwnedGazeKBOrAdmin(), g.KBAccessWrite("kb_id"), gazeHandler.DeletePage)

		// Revision history (slug is a catch-all like /pages; revert carries
		// the slug in the body for the same reason move-page does)
		gazeRead.GET("/revisions/*slug", g.Viewer(), g.KBAccessRead("kb_id"), gazeHandler.ListRevisions)
		gaze.POST("/revert", g.OwnedGazeKBOrAdmin(), g.KBAccessWrite("kb_id"), gazeHandler.RevertPage)

		// Folder tree (directory nodes)
		gazeRead.GET("/folders", g.Viewer(), g.KBAccessRead("kb_id"), gazeHandler.ListFolders)
		gaze.POST("/folders", g.OwnedGazeKBOrAdmin(), g.KBAccessWrite("kb_id"), gazeHandler.CreateFolder)
		gaze.PUT("/folders/:folder_id", g.OwnedGazeKBOrAdmin(), g.KBAccessWrite("kb_id"), gazeHandler.UpdateFolder)
		gaze.DELETE("/folders/:folder_id", g.OwnedGazeKBOrAdmin(), g.KBAccessWrite("kb_id"), gazeHandler.DeleteFolder)

		// Special pages
		gazeRead.GET("/index", g.Viewer(), g.KBAccessRead("kb_id"), gazeHandler.GetIndex)

		// Graph and stats
		gazeRead.GET("/graph", g.Viewer(), g.KBAccessRead("kb_id"), gazeHandler.GetGraph)
		gazeRead.GET("/stats", g.Viewer(), g.KBAccessRead("kb_id"), gazeHandler.GetStats)

		// Search and maintenance
		gazeRead.GET("/search", g.Viewer(), g.KBAccessRead("kb_id"), gazeHandler.SearchPages)
		gaze.POST("/rebuild-links", g.OwnedGazeKBOrAdmin(), g.KBAccessWrite("kb_id"), gazeHandler.RebuildLinks)
		gaze.POST("/outdated-sync", g.OwnedGazeKBOrAdmin(), g.KBAccessWrite("kb_id"), gazeHandler.SyncOutdated)
	}
}
