package router

import (
	"github.com/gin-gonic/gin"

	"gaze/internal/capabilities"
	"gaze/internal/handler"
	"gaze/internal/middleware"
	"gaze/internal/types/interfaces"
)

// RegisterSkillRoutes registers skill routes.
//
// PR 2 currently only exposes a read-only `ListSkills`; gated to
// Viewer+. Future skill upload / enable endpoints must use Admin+ since
// skills run sandboxed code on tenant resources.
func RegisterSkillRoutes(
	r *gin.RouterGroup,
	skillHandler *handler.SkillHandler,
	grants interfaces.SkillEditorGrantService,
	g *rbacGuards,
) {
	skills := r.Group("/skills")
	{
		// Usable skills for @ mention / chat — Viewer+ (selection is not edit).
		skills.GET("", g.Viewer(), skillHandler.ListSkills)
		// Enabled skills for the QA skill picker — every member may select.
		skills.GET("/selectable", g.Viewer(), skillHandler.ListSelectableCatalog)
		// Catalog listing is a skill-management surface.
		skills.GET("/catalog", g.Viewer(), middleware.RequireCapability(capabilities.SkillsView), skillHandler.ListCatalog)
	}
	// Catalog writes are tenant governance actions. Scoped API keys cannot
	// hold them; per-skill file reads/deletes additionally require editability.
	catalogWrite := g.apiKeyGroup(r.Group("/skills/catalog"), apiKeyFullAccess())
	{
		catalogWrite.POST("", g.Viewer(), middleware.RequireCapability(capabilities.SettingsSkillsManage), skillHandler.RegisterCatalog)
		catalogWrite.POST("/:id/install", g.Viewer(), middleware.RequireCapability(capabilities.SettingsSkillsManage), skillHandler.InstallCatalog)
		catalogWrite.GET("/:id/files", g.Viewer(), middleware.RequireSkillEdit(grants), skillHandler.ListCatalogFiles)
		catalogWrite.GET("/:id/files/content", g.Viewer(), middleware.RequireSkillEdit(grants), skillHandler.GetCatalogFile)
		catalogWrite.PUT("/:id/enabled", g.Viewer(), middleware.RequireSkillEdit(grants), skillHandler.SetCatalogEnabled)
		catalogWrite.PUT("/:id", g.Viewer(), middleware.RequireSkillEdit(grants), skillHandler.UpdateCatalog)
		catalogWrite.DELETE("/:id", g.Viewer(), middleware.RequireSkillEdit(grants), skillHandler.DeleteCatalog)
	}
}

// RegisterSkillGrantRoutes mounts per-skill editor grants. These are human
// governance endpoints, so they stay outside the API-key route table (keys are
// default-denied) and require the skill-management capability.
func RegisterSkillGrantRoutes(r *gin.RouterGroup, h *handler.SkillGrantHandler, g *rbacGuards) {
	if h == nil {
		return
	}
	grants := r.Group("/skills/:skill_id/grants")
	{
		grants.GET("", g.Viewer(), middleware.RequireCapability(capabilities.SettingsSkillsManage), h.List)
		grants.POST("", g.Viewer(), middleware.RequireCapability(capabilities.SettingsSkillsManage), h.Create)
		grants.DELETE("/:grant_id", g.Viewer(), middleware.RequireCapability(capabilities.SettingsSkillsManage), h.Delete)
	}
}
