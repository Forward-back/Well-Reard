package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"gaze/internal/application/service"
	apperrors "gaze/internal/errors"
	"gaze/internal/types"
	"gaze/internal/types/interfaces"
)

// DepartmentHandler exposes company department spaces. Route guards enforce
// company-admin for structural mutations; the service re-checks permissions
// and all invariants.
type DepartmentHandler struct {
	service     interfaces.DepartmentService
	userService interfaces.UserService
}

func NewDepartmentHandler(service interfaces.DepartmentService, userService interfaces.UserService) *DepartmentHandler {
	return &DepartmentHandler{service: service, userService: userService}
}

type createDepartmentRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type updateDepartmentRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type assignDepartmentMemberRequest struct {
	DepartmentRole types.DepartmentRole `json:"department_role" binding:"required"`
}

func (h *DepartmentHandler) writeDepartmentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrDepartmentNotFound):
		c.Error(apperrors.NewNotFoundError(err.Error()))
	case errors.Is(err, service.ErrDepartmentNameRequired), errors.Is(err, service.ErrInvalidDepartmentRole):
		c.Error(apperrors.NewValidationError(err.Error()))
	case errors.Is(err, service.ErrDepartmentForbidden):
		c.Error(apperrors.NewForbiddenError(err.Error()))
	case errors.Is(err, service.ErrDepartmentNonEmpty), errors.Is(err, service.ErrDefaultDepartmentProtected),
		errors.Is(err, service.ErrLastDepartmentManager):
		c.Error(apperrors.NewConflictError(err.Error()))
	case errors.Is(err, service.ErrDepartmentMemberNotFound):
		c.Error(apperrors.NewNotFoundError(err.Error()))
	default:
		c.Error(apperrors.NewInternalServerError(err.Error()))
	}
}

func (h *DepartmentHandler) ListDepartments(c *gin.Context) {
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	depts, err := h.service.List(c.Request.Context(), tenantID)
	if err != nil {
		h.writeDepartmentError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": depts})
}

func (h *DepartmentHandler) CreateDepartment(c *gin.Context) {
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	var req createDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("invalid request body").WithDetails(err.Error()))
		return
	}
	dept, err := h.service.Create(c.Request.Context(), tenantID, req.Name, req.Description)
	if err != nil {
		h.writeDepartmentError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": dept})
}

func (h *DepartmentHandler) UpdateDepartment(c *gin.Context) {
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	var req updateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("invalid request body").WithDetails(err.Error()))
		return
	}
	dept, err := h.service.Update(c.Request.Context(), tenantID, strings.TrimSpace(c.Param("dept_id")), req.Name, req.Description)
	if err != nil {
		h.writeDepartmentError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": dept})
}

func (h *DepartmentHandler) DeleteDepartment(c *gin.Context) {
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), tenantID, strings.TrimSpace(c.Param("dept_id"))); err != nil {
		h.writeDepartmentError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *DepartmentHandler) ListDepartmentMembers(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	members, err := h.service.ListMembers(ctx, tenantID, strings.TrimSpace(c.Param("dept_id")))
	if err != nil {
		h.writeDepartmentError(c, err)
		return
	}
	ids := make([]string, 0, len(members))
	for _, member := range members {
		ids = append(ids, member.UserID)
	}
	usersByID := map[string]*types.User{}
	if users, err := h.userService.GetUsersByIDs(ctx, ids); err == nil {
		usersByID = users
	}
	resp := make([]types.TenantMemberResponse, 0, len(members))
	for _, member := range members {
		row := types.TenantMemberResponse{
			UserID:         member.UserID,
			Role:           member.Role,
			DepartmentID:   member.DepartmentID,
			DepartmentRole: member.DepartmentRole,
			Status:         member.Status,
			InvitedBy:      member.InvitedBy,
			JoinedAt:       member.JoinedAt,
		}
		if user, ok := usersByID[member.UserID]; ok && user != nil {
			row.Email = user.Email
			row.Username = user.Username
			row.Avatar = user.Avatar
		}
		resp = append(resp, row)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

func (h *DepartmentHandler) AssignDepartmentMember(c *gin.Context) {
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	var req assignDepartmentMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("invalid request body").WithDetails(err.Error()))
		return
	}
	err := h.service.AssignMember(
		c.Request.Context(),
		tenantID,
		strings.TrimSpace(c.Param("dept_id")),
		strings.TrimSpace(c.Param("user_id")),
		req.DepartmentRole,
	)
	if err != nil {
		h.writeDepartmentError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *DepartmentHandler) RemoveDepartmentMember(c *gin.Context) {
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	err := h.service.RemoveMember(
		c.Request.Context(),
		tenantID,
		strings.TrimSpace(c.Param("dept_id")),
		strings.TrimSpace(c.Param("user_id")),
	)
	if err != nil {
		h.writeDepartmentError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

type crAuditorRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

func (h *DepartmentHandler) ListDepartmentCRAuditors(c *gin.Context) {
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	rows, err := h.service.ListCRAuditors(
		c.Request.Context(), tenantID, strings.TrimSpace(c.Param("dept_id")),
	)
	if err != nil {
		h.writeDepartmentError(c, err)
		return
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		if row != nil {
			ids = append(ids, row.UserID)
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"user_ids": ids}})
}

func (h *DepartmentHandler) AddDepartmentCRAuditor(c *gin.Context) {
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	var req crAuditorRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.UserID) == "" {
		c.Error(apperrors.NewValidationError("user_id is required"))
		return
	}
	if err := h.service.AddCRAuditor(
		c.Request.Context(), tenantID, strings.TrimSpace(c.Param("dept_id")), strings.TrimSpace(req.UserID),
	); err != nil {
		h.writeDepartmentError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *DepartmentHandler) RemoveDepartmentCRAuditor(c *gin.Context) {
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	if err := h.service.RemoveCRAuditor(
		c.Request.Context(), tenantID, strings.TrimSpace(c.Param("dept_id")), strings.TrimSpace(c.Param("user_id")),
	); err != nil {
		h.writeDepartmentError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
