package handler

import (
	stderrors "errors"
	"net/http"
	"strconv"
	"strings"

	"gaze/internal/errors"
	"gaze/internal/logger"
	"gaze/internal/types"
	"gaze/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

// CollaborationHandler exposes the Git-style document collaboration API.
// Routes already resolved KB access; the service enforces the finer role
// matrix (viewer/editor/admin) with the caller taken from ctx.
type CollaborationHandler struct {
	service interfaces.CollaborationService
}

// NewCollaborationHandler creates the collaboration handler.
func NewCollaborationHandler(service interfaces.CollaborationService) *CollaborationHandler {
	return &CollaborationHandler{service: service}
}

func (h *CollaborationHandler) fail(c *gin.Context, err error) {
	logger.Warnf(c.Request.Context(), "collaboration: %v", err)
	var appErr *errors.AppError
	if stderrors.As(err, &appErr) {
		_ = c.Error(appErr)
		return
	}
	_ = c.Error(errors.NewInternalServerError(err.Error()))
}

func collabPage(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	return page, pageSize
}

// collabKBID reads the KB path param. Collaboration routes share gin's
// existing "/knowledge-bases/:id" wildcard name because gin rejects two
// different wildcard names at the same tree position.
func collabKBID(c *gin.Context) string {
	if id := c.Param("kb_id"); id != "" {
		return id
	}
	return c.Param("id")
}

// ListIssues godoc
// @Summary      列出文档问题
// @Tags         文档协作
// @Router       /knowledge-bases/{kb_id}/issues [get]
func (h *CollaborationHandler) ListIssues(c *gin.Context) {
	req := &types.DocIssueListRequest{
		KnowledgeBaseID: collabKBID(c),
		KnowledgeID:     strings.TrimSpace(c.Query("knowledge_id")),
		ChunkID:         strings.TrimSpace(c.Query("chunk_id")),
		Status:          strings.TrimSpace(c.Query("status")),
		CreatedBy:       strings.TrimSpace(c.Query("created_by")),
	}
	req.Page, req.PageSize = collabPage(c)
	result, err := h.service.ListDocIssues(c.Request.Context(), req)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// CreateIssue godoc
// @Summary      提交文档问题
// @Tags         文档协作
// @Router       /knowledge-bases/{kb_id}/issues [post]
func (h *CollaborationHandler) CreateIssue(c *gin.Context) {
	var req types.CreateDocIssueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	issue, err := h.service.CreateDocIssue(c.Request.Context(), collabKBID(c), &req)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": issue})
}

// GetIssue godoc
// @Summary      获取文档问题详情
// @Tags         文档协作
// @Router       /knowledge-bases/{kb_id}/issues/{issue_id} [get]
func (h *CollaborationHandler) GetIssue(c *gin.Context) {
	issue, err := h.service.GetDocIssue(c.Request.Context(), collabKBID(c), c.Param("issue_id"))
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": issue})
}

// UpdateIssue godoc
// @Summary      更新/关闭/重开文档问题
// @Tags         文档协作
// @Router       /knowledge-bases/{kb_id}/issues/{issue_id} [patch]
func (h *CollaborationHandler) UpdateIssue(c *gin.Context) {
	var req types.UpdateDocIssueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	issue, err := h.service.UpdateDocIssue(c.Request.Context(), collabKBID(c), c.Param("issue_id"), &req)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": issue})
}

// DeleteIssue godoc
// @Summary      删除文档问题（软删）
// @Tags         文档协作
// @Router       /knowledge-bases/{kb_id}/issues/{issue_id} [delete]
func (h *CollaborationHandler) DeleteIssue(c *gin.Context) {
	if err := h.service.DeleteDocIssue(c.Request.Context(), collabKBID(c), c.Param("issue_id")); err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ListChangeRequests godoc
// @Summary      列出变更请求
// @Tags         文档协作
// @Router       /knowledge-bases/{kb_id}/change-requests [get]
func (h *CollaborationHandler) ListChangeRequests(c *gin.Context) {
	req := &types.DocCRListRequest{
		KnowledgeBaseID: collabKBID(c),
		KnowledgeID:     strings.TrimSpace(c.Query("knowledge_id")),
		Status:          strings.TrimSpace(c.Query("status")),
		CreatedBy:       strings.TrimSpace(c.Query("created_by")),
	}
	req.Page, req.PageSize = collabPage(c)
	result, err := h.service.ListDocChangeRequests(c.Request.Context(), req)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// CreateChangeRequest godoc
// @Summary      提交变更请求
// @Tags         文档协作
// @Router       /knowledge-bases/{kb_id}/change-requests [post]
func (h *CollaborationHandler) CreateChangeRequest(c *gin.Context) {
	var req types.CreateDocCRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	cr, err := h.service.CreateDocChangeRequest(c.Request.Context(), collabKBID(c), &req)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": cr})
}

// GetChangeRequest godoc
// @Summary      获取变更请求详情
// @Tags         文档协作
// @Router       /knowledge-bases/{kb_id}/change-requests/{cr_id} [get]
func (h *CollaborationHandler) GetChangeRequest(c *gin.Context) {
	cr, err := h.service.GetDocChangeRequest(c.Request.Context(), collabKBID(c), c.Param("cr_id"))
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": cr})
}

// ResubmitChangeRequest godoc
// @Summary      重新提交已驳回的变更请求
// @Tags         文档协作
// @Router       /knowledge-bases/{kb_id}/change-requests/{cr_id}/resubmit [post]
func (h *CollaborationHandler) ResubmitChangeRequest(c *gin.Context) {
	var req types.CreateDocCRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	cr, err := h.service.ResubmitDocChangeRequest(c.Request.Context(), collabKBID(c), c.Param("cr_id"), &req)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": cr})
}

// ReviewChangeRequest godoc
// @Summary      接受或驳回变更请求
// @Tags         文档协作
// @Router       /knowledge-bases/{kb_id}/change-requests/{cr_id}/review [post]
func (h *CollaborationHandler) ReviewChangeRequest(c *gin.Context) {
	var req types.ReviewDocCRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	cr, err := h.service.ReviewDocChangeRequest(c.Request.Context(), collabKBID(c), c.Param("cr_id"), &req)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": cr})
}

// ListIssueComments godoc
// @Summary      列出问题评论
// @Tags         文档协作
// @Router       /knowledge-bases/{kb_id}/issues/{issue_id}/comments [get]
func (h *CollaborationHandler) ListIssueComments(c *gin.Context) {
	h.listComments(c, types.DocCommentTargetIssue, c.Param("issue_id"))
}

// CreateIssueComment godoc
// @Summary      添加问题评论
// @Tags         文档协作
// @Router       /knowledge-bases/{kb_id}/issues/{issue_id}/comments [post]
func (h *CollaborationHandler) CreateIssueComment(c *gin.Context) {
	h.createComment(c, types.DocCommentTargetIssue, c.Param("issue_id"))
}

// ListCRComments godoc
// @Summary      列出变更请求评论
// @Tags         文档协作
// @Router       /knowledge-bases/{kb_id}/change-requests/{cr_id}/comments [get]
func (h *CollaborationHandler) ListCRComments(c *gin.Context) {
	h.listComments(c, types.DocCommentTargetCR, c.Param("cr_id"))
}

// CreateCRComment godoc
// @Summary      添加变更请求评论
// @Tags         文档协作
// @Router       /knowledge-bases/{kb_id}/change-requests/{cr_id}/comments [post]
func (h *CollaborationHandler) CreateCRComment(c *gin.Context) {
	h.createComment(c, types.DocCommentTargetCR, c.Param("cr_id"))
}

func (h *CollaborationHandler) listComments(c *gin.Context, targetType, targetID string) {
	comments, err := h.service.ListDocComments(c.Request.Context(), collabKBID(c), targetType, targetID)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": comments})
}

type createCommentBody struct {
	Body string `json:"body" binding:"required"`
}

func (h *CollaborationHandler) createComment(c *gin.Context, targetType, targetID string) {
	var body createCommentBody
	if err := c.ShouldBindJSON(&body); err != nil {
		_ = c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	comment, err := h.service.CreateDocComment(c.Request.Context(), collabKBID(c), &types.CreateDocCommentRequest{
		TargetType: targetType,
		TargetID:   targetID,
		Body:       body.Body,
	})
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": comment})
}

// DeleteComment godoc
// @Summary      删除评论（软删）
// @Tags         文档协作
// @Router       /knowledge-bases/{kb_id}/comments/{comment_id} [delete]
func (h *CollaborationHandler) DeleteComment(c *gin.Context) {
	if err := h.service.DeleteDocComment(c.Request.Context(), collabKBID(c), c.Param("comment_id")); err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ListCollaborations godoc
// @Summary      统一列出问题与变更请求
// @Tags         文档协作
// @Router       /knowledge-bases/{kb_id}/collaborations [get]
func (h *CollaborationHandler) ListCollaborations(c *gin.Context) {
	req := &types.DocCollaborationListRequest{
		KnowledgeBaseID: collabKBID(c),
		KnowledgeID:     strings.TrimSpace(c.Query("knowledge_id")),
		Type:            strings.TrimSpace(c.Query("type")),
		Status:          strings.TrimSpace(c.Query("status")),
		CreatedBy:       strings.TrimSpace(c.Query("created_by")),
	}
	req.Page, req.PageSize = collabPage(c)
	result, err := h.service.ListDocCollaborations(c.Request.Context(), req)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}
