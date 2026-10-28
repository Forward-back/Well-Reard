package handler

import (
	stderrors "errors"
	"io"
	"net/http"

	"gaze/internal/application/service"
	"gaze/internal/errors"
	"gaze/internal/logger"
	"github.com/gin-gonic/gin"
)

// EngineSyncHandler exposes the manual YLSW re-push endpoint (需求一):
// CR acceptance only flags documents as pending; an admin presses the button
// to rebuild the engine's copy from the current app-side chunks.
type EngineSyncHandler struct {
	service *service.EngineSyncService
}

// NewEngineSyncHandler creates the engine sync handler.
func NewEngineSyncHandler(service *service.EngineSyncService) *EngineSyncHandler {
	return &EngineSyncHandler{service: service}
}

func (h *EngineSyncHandler) fail(c *gin.Context, err error) {
	logger.Warnf(c.Request.Context(), "engine sync: %v", err)
	var appErr *errors.AppError
	if stderrors.As(err, &appErr) {
		_ = c.Error(appErr)
		return
	}
	_ = c.Error(errors.NewInternalServerError(err.Error()))
}

type engineSyncBody struct {
	KnowledgeIDs []string `json:"knowledge_ids"`
}

// TriggerKBSync godoc
// @Summary      触发文档重推 YLSW
// @Description  管理员手动把应用侧分块内容重推到检索引擎；knowledge_ids 为空时重推所有 pending/failed 文档
// @Tags         文档协作
// @Router       /knowledge-bases/{kb_id}/engine-sync [post]
func (h *EngineSyncHandler) TriggerKBSync(c *gin.Context) {
	if h.service == nil {
		h.fail(c, errors.NewInternalServerError("engine sync service unavailable"))
		return
	}
	var body engineSyncBody
	if err := c.ShouldBindJSON(&body); err != nil && !stderrors.Is(err, io.EOF) {
		_ = c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	queued, err := h.service.TriggerKBSync(c.Request.Context(), c.Param("id"), body.KnowledgeIDs)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": gin.H{"queued": queued}})
}
