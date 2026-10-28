package router

import (
	"github.com/gin-gonic/gin"

	"gaze/internal/capabilities"
	"gaze/internal/handler"
	"gaze/internal/middleware"
)

// RegisterReadingRoutes mounts the interactive document reading API. These
// endpoints are human, session-scoped, read-only over KB content, and are not
// exposed to API keys (the API-key gate default-denies undeclared routes).
func RegisterReadingRoutes(r *gin.RouterGroup, h *handler.ReadingHandler, g *rbacGuards) {
	if h == nil {
		return
	}
	reading := r.Group("/reading", g.Viewer())
	{
		reading.POST("/runs", h.Create)
		reading.GET("/runs/active", h.GetActive)
		reading.GET("/runs/:id", h.Get)
		reading.POST("/runs/:id/step", h.Step)
		reading.POST("/runs/:id/feedback", h.Feedback)
		reading.POST("/runs/:id/contradictions/:contradictionId/resolve", h.ResolveContradiction)
		reading.GET("/runs/:id/report", h.GetReport)
		reading.PUT("/runs/:id/report", h.SaveReport)
		reading.DELETE("/runs/:id/report", h.DeleteReport)
		reading.GET("/runs/:id/documents", h.GetDocuments)
		reading.POST("/runs/:id/add-documents", h.AddDocuments)
		reading.POST("/runs/:id/stop", h.Stop)
		reading.POST("/runs/:id/resume", h.Resume)
		reading.GET("/runs/:id/export", h.Export)
	}
	// The legacy migration is a system-level maintenance action. It must live
	// outside the Viewer group: system_admin holds no workspace business role.
	adminReading := r.Group("/reading", g.AdminOrSystemAdmin())
	adminReading.POST("/admin/migrate-legacy",
		middleware.RequireCapability(capabilities.SettingsWorkspaceManage),
		h.MigrateLegacy,
	)
}
