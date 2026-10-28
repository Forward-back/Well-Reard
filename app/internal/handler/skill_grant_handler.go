package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gaze/internal/errors"
	"gaze/internal/types"
	"gaze/internal/types/interfaces"
)

// SkillGrantHandler exposes per-skill editor grants.
type SkillGrantHandler struct {
	service interfaces.SkillEditorGrantService
}

func NewSkillGrantHandler(service interfaces.SkillEditorGrantService) *SkillGrantHandler {
	return &SkillGrantHandler{service: service}
}

func (h *SkillGrantHandler) List(c *gin.Context) {
	grants, err := h.service.List(c.Request.Context(), c.Param("skill_id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": grants})
}

func (h *SkillGrantHandler) Create(c *gin.Context) {
	var req types.SkillEditorGrantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	grant, err := h.service.Create(c.Request.Context(), c.Param("skill_id"), req)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": grant})
}

func (h *SkillGrantHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("skill_id"), c.Param("grant_id")); err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
