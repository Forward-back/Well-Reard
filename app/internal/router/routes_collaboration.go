package router

import (
	"gaze/internal/handler"
	"github.com/gin-gonic/gin"
)

// RegisterCollaborationRoutes wires the Git-style document maintenance API
// (需求一). Every route resolves KB access first; the service then applies the
// finer role matrix because viewers may open issues and comment while only
// editors may submit CRs and only admins may review/close/delete.
//
// The path wildcard is ":id" (not ":kb_id") to match the existing
// /knowledge-bases/:id route tree — gin panics on two wildcard names at the
// same position.
func RegisterCollaborationRoutes(
	r *gin.RouterGroup,
	h *handler.CollaborationHandler,
	es *handler.EngineSyncHandler,
	g *rbacGuards,
) {
	if h == nil {
		return
	}
	collab := g.apiKeyGroup(r.Group("/knowledge-bases/:id"), apiKeyIngest(apiKeyFullAccess()))
	collabRead := collab.With(apiKeyRetrieve(apiKeyFullAccess()))
	{
		// Manual YLSW re-push after accepted CRs. KB access is resolved here;
		// the handler additionally requires tenant/organization admin.
		if es != nil {
			collab.POST("/engine-sync", g.KBAccessRead("id"), es.TriggerKBSync)
		}
		collabRead.GET("/issues", g.KBAccessRead("id"), h.ListIssues)
		collab.POST("/issues", g.KBAccessRead("id"), h.CreateIssue)
		collabRead.GET("/issues/:issue_id", g.KBAccessRead("id"), h.GetIssue)
		collab.PATCH("/issues/:issue_id", g.KBAccessRead("id"), h.UpdateIssue)
		collab.DELETE("/issues/:issue_id", g.KBAccessRead("id"), h.DeleteIssue)
		collabRead.GET("/issues/:issue_id/comments", g.KBAccessRead("id"), h.ListIssueComments)
		collab.POST("/issues/:issue_id/comments", g.KBAccessRead("id"), h.CreateIssueComment)

		collabRead.GET("/change-requests", g.KBAccessRead("id"), h.ListChangeRequests)
		collab.POST("/change-requests", g.KBAccessRead("id"), h.CreateChangeRequest)
		collabRead.GET("/change-requests/:cr_id", g.KBAccessRead("id"), h.GetChangeRequest)
		collab.POST("/change-requests/:cr_id/resubmit", g.KBAccessRead("id"), h.ResubmitChangeRequest)
		collab.POST("/change-requests/:cr_id/review", g.KBAccessRead("id"), h.ReviewChangeRequest)
		collab.POST("/change-requests/:cr_id/merge", g.KBAccessRead("id"), h.MergeChangeRequest)
		collab.POST("/change-requests/:cr_id/close", g.KBAccessRead("id"), h.CloseChangeRequest)
		collab.PATCH("/change-requests/:cr_id/draft", g.KBAccessRead("id"), h.UpdateChangeRequestDraft)
		collab.POST("/change-requests/:cr_id/submit", g.KBAccessRead("id"), h.SubmitChangeRequestDraft)
		collab.PUT("/change-requests/:cr_id/reviewers", g.KBAccessRead("id"), h.UpdateChangeRequestReviewers)
		collab.POST("/change-requests/:cr_id/viewed", g.KBAccessRead("id"), h.MarkChangeRequestViewed)
		collabRead.GET("/change-requests/:cr_id/comments", g.KBAccessRead("id"), h.ListCRComments)
		collab.POST("/change-requests/:cr_id/comments", g.KBAccessRead("id"), h.CreateCRComment)

		collabRead.GET("/documents/:knowledge_id/anchor-comments", g.KBAccessRead("id"), h.ListAnchorComments)
		collabRead.GET("/documents/:knowledge_id/chunk-locks", g.KBAccessRead("id"), h.ListChunkLocks)
		collabRead.GET("/citation-stats", g.KBAccessRead("id"), h.ListChunkCitationStats)
		collab.POST("/documents/:knowledge_id/anchor-comments", g.KBAccessRead("id"), h.CreateAnchorComment)
		collab.PATCH("/anchor-comments/:comment_id", g.KBAccessRead("id"), h.UpdateAnchorComment)
		collab.DELETE("/anchor-comments/:comment_id", g.KBAccessRead("id"), h.DeleteAnchorComment)

		collabRead.GET("/documents/:knowledge_id/versions", g.KBAccessRead("id"), h.ListVersions)
		collabRead.GET("/documents/:knowledge_id/versions/compare", g.KBAccessRead("id"), h.CompareVersions)
		collabRead.GET("/documents/:knowledge_id/versions/:version_id", g.KBAccessRead("id"), h.GetVersion)
		collab.POST("/documents/:knowledge_id/versions/:version_id/rollback", g.KBAccessRead("id"), h.RollbackVersion)

		collab.DELETE("/comments/:comment_id", g.KBAccessRead("id"), h.DeleteComment)
		collabRead.GET("/collaborations", g.KBAccessRead("id"), h.ListCollaborations)
	}
}
