package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"gaze/internal/types/interfaces"
)

// GovernanceHandler exposes read-only management dashboards and audit search.
type GovernanceHandler struct {
	service interfaces.GovernanceService
}

func NewGovernanceHandler(service interfaces.GovernanceService) *GovernanceHandler {
	return &GovernanceHandler{service: service}
}

func (h *GovernanceHandler) DeptDashboard(c *gin.Context) {
	stats, err := h.service.DeptStats(c.Request.Context(), c.Query("department_id"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}

func (h *GovernanceHandler) CompanyDashboard(c *gin.Context) {
	stats, err := h.service.CompanyStats(c.Request.Context(), c.Query("department_id"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}

func (h *GovernanceHandler) Audit(c *gin.Context) {
	afterID, _ := strconv.ParseUint(c.DefaultQuery("after_id", "0"), 10, 64)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	rows, err := h.service.Audit(c.Request.Context(), c.Query("department_id"), afterID, limit)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rows})
}
