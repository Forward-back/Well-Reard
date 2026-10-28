package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gaze/internal/capabilities"
	"gaze/internal/handler"
	"gaze/internal/middleware"
)

// Models are tenant-wide infrastructure (LLM credentials, embeddings,
// rerankers); Viewer+ for reads, Admin+ for any mutation. Credential
// subresource writes are also Admin+ since secrets are tenant-scoped.
func RegisterModelRoutes(
	r *gin.RouterGroup,
	handler *handler.ModelHandler,
	credHandler *handler.ModelCredentialsHandler,
	g *rbacGuards,
) {
	// 模型路由组。空间级基础设施：仅完全访问（Owner）API key 可访问。
	models := g.apiKeyGroup(r.Group("/models"), apiKeyManageModels(apiKeyFullAccess()))
	{
		// 获取模型厂商列表 — Viewer+
		models.GET("/providers", g.ViewerOrSystemAdmin(), handler.ListModelProviders)
		// 创建模型 — Admin+ / settings:models_manage
		models.POST("", g.AdminOrSystemAdmin(), middleware.RequireCapability(capabilities.SettingsModelsManage), handler.CreateModel)
		// 获取模型列表 — Viewer+
		models.GET("", g.ViewerOrSystemAdmin(), handler.ListModels)
		// 调试已保存模型会发起真实上游调用并产生费用 — Admin+
		models.POST("/:id/debug", g.AdminOrSystemAdmin(), middleware.RequireCapability(capabilities.SettingsModelsManage), handler.DebugModel)
		// 获取单个模型 — Viewer+
		models.GET("/:id", g.ViewerOrSystemAdmin(), handler.GetModel)
		// 更新模型 — Admin+；内置模型仍由服务层额外限定为 SystemAdmin。
		models.PUT("/:id", g.AdminOrSystemAdmin(), middleware.RequireCapability(capabilities.SettingsModelsManage), handler.UpdateModel)
		// 删除模型 — Admin+
		models.DELETE("/:id", g.AdminOrSystemAdmin(), middleware.RequireCapability(capabilities.SettingsModelsManage), handler.DeleteModel)
		// Per-field credential subresource (see internal/handler/model_credentials.go) — Admin+
		models.PUT("/:id/credentials", g.AdminOrSystemAdmin(), middleware.RequireCapability(capabilities.SettingsModelsManage), credHandler.Put)
		models.DELETE("/:id/credentials/:field", g.AdminOrSystemAdmin(), middleware.RequireCapability(capabilities.SettingsModelsManage), credHandler.DeleteField)
	}
}

func RegisterInitializationRoutes(r *gin.RouterGroup, handler *handler.InitializationHandler, g *rbacGuards) {
	// 初始化接口
	// GetCurrentConfigByKB 是只读，Viewer+ 即可（KB 受限 key 可读其范围内的 KB）。
	g.apiKeyRoute(r, http.MethodGet, "/initialization/config/:kbId",
		apiKeyRetrieve(apiKeyFullAccess()), g.Viewer(), g.KBAccessRead("kbId"), handler.GetCurrentConfigByKB)
	// InitializeByKB / UpdateKBConfig 都是改 KB 的核心模型/storage 配置 —
	// 跟 PUT /knowledge-bases/:id 同等敏感，挂同款 OwnedKB 矩阵 + KBAccessWrite
	//（API-key 主体短路 Owned* 守卫，KB allow-list 只能靠 KBAccess 兜底）。
	g.apiKeyRoute(r, http.MethodPost, "/initialization/initialize/:kbId",
		apiKeyManageKnowledgeBases(apiKeyFullAccess()), g.OwnedKBOrAdminFromKbIDParam(), g.KBAccessWrite("kbId"), handler.InitializeByKB)
	g.apiKeyRoute(r, http.MethodPut, "/initialization/config/:kbId",
		apiKeyManageKnowledgeBases(apiKeyFullAccess()), g.KBAccessWrite("kbId"), middleware.RequireCapability(capabilities.KBSettingsManage), handler.UpdateKBConfig)

	// Ollama / 远程 API / 抽取等系统级检测/下载操作。这些不绑某个 KB，
	// 会改空间级模型配置或拉远端模型；JWT 侧只读探测 Viewer+、变更 Admin+。
	// 对 API key 均为空间级：full-access key 可用，scoped key 需要 manage_models。
	g.apiKeyRoute(r, http.MethodGet, "/initialization/ollama/status", apiKeyManageModels(apiKeyFullAccess()), g.ViewerOrSystemAdmin(), handler.CheckOllamaStatus)
	g.apiKeyRoute(r, http.MethodGet, "/initialization/ollama/models", apiKeyManageModels(apiKeyFullAccess()), g.ViewerOrSystemAdmin(), handler.ListOllamaModels)
	g.apiKeyRoute(r, http.MethodPost, "/initialization/ollama/models/check", apiKeyManageModels(apiKeyFullAccess()), g.AdminOrSystemAdmin(), handler.CheckOllamaModels)
	g.apiKeyRoute(r, http.MethodPost, "/initialization/ollama/models/download", apiKeyManageModels(apiKeyFullAccess()), g.AdminOrSystemAdmin(), handler.DownloadOllamaModel)
	g.apiKeyRoute(r, http.MethodGet, "/initialization/ollama/download/progress/:taskId", apiKeyManageModels(apiKeyFullAccess()), g.ViewerOrSystemAdmin(), handler.GetDownloadProgress)
	g.apiKeyRoute(r, http.MethodGet, "/initialization/ollama/download/tasks", apiKeyManageModels(apiKeyFullAccess()), g.ViewerOrSystemAdmin(), handler.ListDownloadTasks)

	// 远程API相关接口
	g.apiKeyRoute(r, http.MethodPost, "/initialization/remote/check", apiKeyManageModels(apiKeyFullAccess()), g.AdminOrSystemAdmin(), handler.CheckRemoteModel)
	g.apiKeyRoute(r, http.MethodPost, "/initialization/embedding/test", apiKeyManageModels(apiKeyFullAccess()), g.AdminOrSystemAdmin(), handler.TestEmbeddingModel)
	g.apiKeyRoute(r, http.MethodPost, "/initialization/rerank/check", apiKeyManageModels(apiKeyFullAccess()), g.AdminOrSystemAdmin(), handler.CheckRerankModel)
	g.apiKeyRoute(r, http.MethodPost, "/initialization/asr/check", apiKeyManageModels(apiKeyFullAccess()), g.AdminOrSystemAdmin(), handler.CheckASRModel)
	g.apiKeyRoute(r, http.MethodPost, "/initialization/multimodal/test", apiKeyManageModels(apiKeyFullAccess()), g.AdminOrSystemAdmin(), handler.TestMultimodalFunction)

	g.apiKeyRoute(r, http.MethodPost, "/initialization/extract/text-relation", apiKeyManageModels(apiKeyFullAccess()), g.AdminOrSystemAdmin(), handler.ExtractTextRelations)
	g.apiKeyRoute(r, http.MethodPost, "/initialization/extract/fabri-tag", apiKeyManageModels(apiKeyFullAccess()), g.AdminOrSystemAdmin(), handler.FabriTag)
	g.apiKeyRoute(r, http.MethodPost, "/initialization/extract/fabri-text", apiKeyManageModels(apiKeyFullAccess()), g.AdminOrSystemAdmin(), handler.FabriText)
}

// RegisterStorageBackendRoutes manages concrete object/file storage instances.
func RegisterStorageBackendRoutes(r *gin.RouterGroup, h *handler.StorageBackendHandler, g *rbacGuards) {
	backends := g.apiKeyGroup(r.Group("/storage-backends"), apiKeyManageStorageBackends(apiKeyFullAccess()))
	{
		backends.GET("/types", g.ViewerOrSystemAdmin(), h.Types)
		backends.POST("/test", g.AdminOrSystemAdmin(), middleware.RequireCapability(capabilities.SettingsInfraManage), h.TestRaw)
		backends.POST("", g.AdminOrSystemAdmin(), middleware.RequireCapability(capabilities.SettingsInfraManage), h.Create)
		backends.GET("", g.ViewerOrSystemAdmin(), h.List)
		backends.GET("/:id", g.ViewerOrSystemAdmin(), h.Get)
		backends.PUT("/:id", g.AdminOrSystemAdmin(), middleware.RequireCapability(capabilities.SettingsInfraManage), h.Update)
		backends.DELETE("/:id", g.AdminOrSystemAdmin(), middleware.RequireCapability(capabilities.SettingsInfraManage), h.Delete)
		backends.POST("/:id/test", g.AdminOrSystemAdmin(), middleware.RequireCapability(capabilities.SettingsInfraManage), h.TestByID)
		backends.PUT("/:id/default", g.AdminOrSystemAdmin(), middleware.RequireCapability(capabilities.SettingsInfraManage), h.SetDefault)
	}
}

// RegisterDataSourceRoutes 注册数据源相关的路由
//
// Data sources hold external service credentials (Feishu/Notion/Yuque)
// and trigger sync jobs that mutate KB content tenant-wide. Reads are
// Viewer+; everything else (CRUD, validation, sync control, credential
// subresource) is Admin+.
func RegisterDataSourceRoutes(
	r *gin.RouterGroup,
	handler *handler.DataSourceHandler,
	credHandler *handler.DataSourceCredentialsHandler,
	g *rbacGuards,
) {
	// Data source routes
	ds := g.apiKeyGroup(r.Group("/datasource"), apiKeyManageDataSources(apiKeyFullAccess()))
	{
		// Get available connector types — Viewer+
		ds.GET("/types", g.Viewer(), handler.GetAvailableConnectors)

		// Validate credentials without persistence (for "Test Connection" button) — Admin+
		ds.POST("/validate-credentials", g.Admin(), handler.ValidateCredentials)

		// CRUD operations
		ds.POST("", g.Admin(), handler.CreateDataSource)
		ds.GET("", g.Viewer(), handler.ListDataSources)
		ds.GET("/:id", g.Viewer(), handler.GetDataSource)
		ds.PUT("/:id", g.Admin(), handler.UpdateDataSource)
		ds.DELETE("/:id", g.Admin(), handler.DeleteDataSource)

		// Credential subresource. Single logical field "credentials" because
		// connector credentials are a per-connector atomic map (see
		// internal/handler/datasource_credentials.go). — Admin+
		ds.PUT("/:id/credentials", g.Admin(), credHandler.Put)
		ds.DELETE("/:id/credentials/:field", g.Admin(), credHandler.DeleteField)

		// Connection and resource management — Admin+
		ds.POST("/:id/validate", g.Admin(), handler.ValidateConnection)
		ds.GET("/:id/resources", g.Admin(), handler.ListAvailableResources)
		ds.POST("/:id/resource-ancestors", g.Admin(), handler.ResolveResourceAncestors)

		// Sync management — Admin+
		ds.POST("/:id/sync", g.Admin(), handler.ManualSync)
		ds.POST("/:id/pause", g.Admin(), handler.PauseDataSource)
		ds.POST("/:id/resume", g.Admin(), handler.ResumeDataSource)

		// Sync logs — Viewer+ (read-only audit trail)
		ds.GET("/:id/logs", g.Viewer(), handler.GetSyncLogs)
		ds.GET("/logs/:log_id", g.Viewer(), handler.GetSyncLog)
	}
}
