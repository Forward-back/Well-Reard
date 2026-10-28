package router

import (
	"github.com/gin-gonic/gin"

	"gaze/internal/capabilities"
	"gaze/internal/handler"
	"gaze/internal/middleware"
)

// RegisterDepartmentRoutes mounts the department-space API under
// /tenants/:id/departments. Structural mutations are company-admin operations;
// member role assignment is additionally delegated to department managers and
// re-checked inside the service.
func RegisterDepartmentRoutes(r *gin.RouterGroup, h *handler.DepartmentHandler, g *rbacGuards) {
	tenantByID := r.Group("/tenants/:id", g.PathTenantMatch())
	{
		tenantByID.GET("/departments", g.Viewer(), h.ListDepartments)
		tenantByID.POST("/departments", g.Admin(), middleware.RequireCapability(capabilities.DepartmentManage), h.CreateDepartment)
		tenantByID.PUT("/departments/:dept_id", g.Admin(), middleware.RequireCapability(capabilities.DepartmentManage), h.UpdateDepartment)
		tenantByID.DELETE("/departments/:dept_id", g.Admin(), middleware.RequireCapability(capabilities.DepartmentManage), h.DeleteDepartment)
		tenantByID.GET("/departments/:dept_id/members", g.Viewer(), h.ListDepartmentMembers)
		tenantByID.PUT("/departments/:dept_id/members/:user_id", g.Viewer(), middleware.RequireCapability(capabilities.MemberManage), h.AssignDepartmentMember)
		tenantByID.DELETE("/departments/:dept_id/members/:user_id", g.Viewer(), middleware.RequireCapability(capabilities.MemberManage), h.RemoveDepartmentMember)
		tenantByID.GET("/departments/:dept_id/cr-auditors", g.Viewer(), middleware.RequireCapability(capabilities.CRAuditManage), h.ListDepartmentCRAuditors)
		tenantByID.POST("/departments/:dept_id/cr-auditors", g.Viewer(), middleware.RequireCapability(capabilities.CRAuditManage), h.AddDepartmentCRAuditor)
		tenantByID.DELETE("/departments/:dept_id/cr-auditors/:user_id", g.Viewer(), middleware.RequireCapability(capabilities.CRAuditManage), h.RemoveDepartmentCRAuditor)
	}
}
