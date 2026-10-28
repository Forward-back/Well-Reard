package router

import (
	"github.com/gin-gonic/gin"

	"gaze/internal/handler"
	"gaze/internal/middleware"
)

// RegisterPortalRoutes registers the consumer portal surface (§2.13 P2/P3).
// Portal routes deliberately sit OUTSIDE the internal Auth/API-key
// middleware chain: they run on portal tokens issued by this service.
//
// Endpoints:
//   - POST /portal/anonymous-token — anonymous token issuance (no auth)
//   - GET  /portal/kbs             — public KB listing (portal token)
//   - POST /portal/chat            — portal QA, SSE stream (portal token)
//   - GET  /portal/sessions        — session history (portal token)
//   - GET  /portal/sessions/:id/messages — one session's messages (portal token)
//   - POST /portal/logout          — consumer logout (anonymizes history)
//
// All routes share the per-IP rate limiter (§2.13 P3-3); anonymous-token
// is the unauthenticated spam surface and gets it too.
func RegisterPortalRoutes(r *gin.Engine, portalHandler *handler.PortalHandler) {
	if portalHandler == nil {
		return
	}
	portal := r.Group("/portal", middleware.PortalIPRateLimit())
	{
		portal.POST("/anonymous-token", portalHandler.IssueAnonymousToken)
		portal.GET("/kbs", middleware.PortalAuthRequired(), portalHandler.ListPublicKnowledgeBases)
		portal.POST("/chat", middleware.PortalAuthRequired(), portalHandler.Chat)
		portal.GET("/sessions", middleware.PortalAuthRequired(), portalHandler.Sessions)
		portal.GET("/sessions/:id/messages", middleware.PortalAuthRequired(), portalHandler.SessionMessages)
		portal.POST("/logout", middleware.PortalAuthRequired(), portalHandler.Logout)
	}
}

// RegisterPortalOpsRoutes mounts the operations surface for portal data on
// the authenticated internal API: Admin+ only (§2.13 — de-identified view).
func RegisterPortalOpsRoutes(rg *gin.RouterGroup, portalHandler *handler.PortalHandler, g *rbacGuards) {
	if portalHandler == nil {
		return
	}
	ops := rg.Group("/portal-analytics")
	{
		ops.GET("/sessions", g.Admin(), portalHandler.ListPortalSessionsForOps)
		ops.GET("/sessions/:id/messages", g.Admin(), portalHandler.ListPortalSessionMessages)
	}
}
