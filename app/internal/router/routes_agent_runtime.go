package router

import (
	"github.com/gin-gonic/gin"

	"gaze/internal/handler"
)

// RegisterAgentRuntimeRoutes exposes read-only agent discovery for external
// integrations. Authoring routes remain unregistered.
func RegisterAgentRuntimeRoutes(r *gin.RouterGroup, h *handler.AgentRuntimeHandler, g *rbacGuards) {
	agents := g.apiKeyGroup(r.Group("/agents"), apiKeyReadAgents(apiKeyChat(apiKeyFullAccess())))
	{
		agents.GET("", g.ViewerOrSystemAdmin(), h.ListAgents)
		agents.GET("/:id", g.Viewer(), h.GetAgent)
		agents.GET("/:id/suggested-questions", g.Viewer(), h.SuggestedQuestions)
	}
}
