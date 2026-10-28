package handler

import (
	"net/http"
	"strings"

	"gaze/internal/errors"
	"gaze/internal/types"
	"github.com/gin-gonic/gin"
)

func (h *CollaborationHandler) MergeChangeRequest(c *gin.Context) {
	var req types.MergeDocCRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	cr, err := h.service.MergeDocChangeRequest(c.Request.Context(), collabKBID(c), c.Param("cr_id"), &req)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": cr})
}

func (h *CollaborationHandler) CloseChangeRequest(c *gin.Context) {
	var req struct {
		Comment string `json:"comment"`
	}
	_ = c.ShouldBindJSON(&req)
	cr, err := h.service.CloseDocChangeRequest(c.Request.Context(), collabKBID(c), c.Param("cr_id"), req.Comment)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": cr})
}

func (h *CollaborationHandler) UpdateChangeRequestDraft(c *gin.Context) {
	var req types.UpdateDocCRDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	cr, err := h.service.UpdateDocCRDraft(c.Request.Context(), collabKBID(c), c.Param("cr_id"), &req)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": cr})
}

func (h *CollaborationHandler) SubmitChangeRequestDraft(c *gin.Context) {
	var req types.SubmitDocCRDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	cr, err := h.service.SubmitDocCRDraft(c.Request.Context(), collabKBID(c), c.Param("cr_id"), &req)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": cr})
}

func (h *CollaborationHandler) UpdateChangeRequestReviewers(c *gin.Context) {
	var req types.UpdateDocCRReviewersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	cr, err := h.service.UpdateDocCRReviewers(c.Request.Context(), collabKBID(c), c.Param("cr_id"), &req)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": cr})
}

func (h *CollaborationHandler) MarkChangeRequestViewed(c *gin.Context) {
	if err := h.service.MarkDocCRViewed(c.Request.Context(), collabKBID(c), c.Param("cr_id")); err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *CollaborationHandler) ListChunkLocks(c *gin.Context) {
	locks, err := h.service.ListDocCRChunkLocks(c.Request.Context(), collabKBID(c), c.Param("knowledge_id"))
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": locks})
}

func (h *CollaborationHandler) ListChunkCitationStats(c *gin.Context) {
	raw := strings.TrimSpace(c.Query("chunk_ids"))
	ids := make([]string, 0)
	if raw != "" {
		ids = strings.Split(raw, ",")
	}
	stats, err := h.service.ListChunkCitationStats(c.Request.Context(), collabKBID(c), ids)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}

func (h *CollaborationHandler) ListAnchorComments(c *gin.Context) {
	comments, err := h.service.ListDocAnchorComments(c.Request.Context(), collabKBID(c), c.Param("knowledge_id"))
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": comments})
}

func (h *CollaborationHandler) CreateAnchorComment(c *gin.Context) {
	var req types.CreateDocAnchorCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	comment, err := h.service.CreateDocAnchorComment(c.Request.Context(), collabKBID(c), &req)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": comment})
}

func (h *CollaborationHandler) UpdateAnchorComment(c *gin.Context) {
	var req types.UpdateDocAnchorCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	comment, err := h.service.UpdateDocAnchorComment(c.Request.Context(), collabKBID(c), c.Param("comment_id"), &req)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": comment})
}

func (h *CollaborationHandler) DeleteAnchorComment(c *gin.Context) {
	if err := h.service.DeleteDocAnchorComment(c.Request.Context(), collabKBID(c), c.Param("comment_id")); err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *CollaborationHandler) ListVersions(c *gin.Context) {
	versions, err := h.service.ListDocVersions(c.Request.Context(), collabKBID(c), c.Param("knowledge_id"))
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": versions})
}

func (h *CollaborationHandler) GetVersion(c *gin.Context) {
	version, chunks, err := h.service.GetDocVersion(c.Request.Context(), collabKBID(c), c.Param("knowledge_id"), c.Param("version_id"))
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": version, "chunks": chunks})
}

func (h *CollaborationHandler) CompareVersions(c *gin.Context) {
	from := strings.TrimSpace(c.Query("from"))
	to := strings.TrimSpace(c.Query("to"))
	result, err := h.service.CompareDocVersions(c.Request.Context(), collabKBID(c), c.Param("knowledge_id"), from, to)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func (h *CollaborationHandler) RollbackVersion(c *gin.Context) {
	var req types.RollbackDocVersionRequest
	_ = c.ShouldBindJSON(&req)
	version, err := h.service.RollbackDocVersion(c.Request.Context(), collabKBID(c), c.Param("knowledge_id"), c.Param("version_id"), &req)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": version})
}
