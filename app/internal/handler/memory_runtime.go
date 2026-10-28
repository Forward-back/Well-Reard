package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "gaze/internal/errors"
	"gaze/internal/types/interfaces"
)

// MemoryRuntimeHandler keeps the single memory mutation used by chat: the
// "forget this item" action on a turn's used-memory row.
type MemoryRuntimeHandler struct {
	service interfaces.MemoryService
}

func NewMemoryRuntimeHandler(service interfaces.MemoryService) *MemoryRuntimeHandler {
	return &MemoryRuntimeHandler{service: service}
}

func (h *MemoryRuntimeHandler) DeleteItem(c *gin.Context) {
	if err := h.service.DeleteItem(c.Request.Context(), c.Param("id")); err != nil {
		_ = c.Error(apperrors.NewInternalServerError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
