package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"gaze/internal/errors"
	"gaze/internal/types"
	"gaze/internal/types/interfaces"
)

// ReadingHandler exposes the interactive document reading API.
type ReadingHandler struct {
	service interfaces.ReadingRunService
}

func NewReadingHandler(service interfaces.ReadingRunService) *ReadingHandler {
	return &ReadingHandler{service: service}
}

func (h *ReadingHandler) Create(c *gin.Context) {
	var req types.CreateReadingRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	run, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": run})
}

func (h *ReadingHandler) Get(c *gin.Context) {
	run, err := h.service.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": run})
}

func (h *ReadingHandler) GetActive(c *gin.Context) {
	run, err := h.service.GetActive(c.Request.Context(), c.Query("session_id"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": run})
}

func (h *ReadingHandler) Step(c *gin.Context) {
	run, err := h.service.Step(c.Request.Context(), c.Param("id"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": run})
}

func (h *ReadingHandler) Feedback(c *gin.Context) {
	var req types.ReadingFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	run, err := h.service.Feedback(c.Request.Context(), c.Param("id"), &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": run})
}

func (h *ReadingHandler) ResolveContradiction(c *gin.Context) {
	var req types.ReadingContradictionResolutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	run, err := h.service.ResolveContradiction(
		c.Request.Context(), c.Param("id"), c.Param("contradictionId"), &req,
	)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": run})
}

func (h *ReadingHandler) GetReport(c *gin.Context) {
	report, err := h.service.GetReport(c.Request.Context(), c.Param("id"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": report})
}

func (h *ReadingHandler) GetDocuments(c *gin.Context) {
	documents, err := h.service.GetDocuments(c.Request.Context(), c.Param("id"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": documents})
}

func (h *ReadingHandler) SaveReport(c *gin.Context) {
	var req types.SaveReadingReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	report, err := h.service.SaveReport(c.Request.Context(), c.Param("id"), &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": report})
}

func (h *ReadingHandler) DeleteReport(c *gin.Context) {
	baseVersion := -1
	if raw := c.Query("base_version"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			_ = c.Error(errors.NewBadRequestError("base_version 必须是整数"))
			return
		}
		baseVersion = parsed
	}
	report, err := h.service.DeleteReport(c.Request.Context(), c.Param("id"), baseVersion)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": report})
}

func (h *ReadingHandler) MigrateLegacy(c *gin.Context) {
	limit := 500
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			_ = c.Error(errors.NewBadRequestError("limit 必须是正整数"))
			return
		}
		limit = parsed
	}
	migrated, err := h.service.MigrateLegacyFinishedRuns(c.Request.Context(), limit)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"migrated": migrated}})
}

func (h *ReadingHandler) AddDocuments(c *gin.Context) {
	var req types.AddReadingDocumentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	run, err := h.service.AddDocuments(c.Request.Context(), c.Param("id"), &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": run})
}

func (h *ReadingHandler) Stop(c *gin.Context) {
	run, err := h.service.Stop(c.Request.Context(), c.Param("id"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": run})
}

func (h *ReadingHandler) Resume(c *gin.Context) {
	run, err := h.service.Resume(c.Request.Context(), c.Param("id"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": run})
}

func (h *ReadingHandler) Export(c *gin.Context) {
	format := c.DefaultQuery("format", "markdown")
	report, err := h.service.Export(c.Request.Context(), c.Param("id"), format)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": report})
}
