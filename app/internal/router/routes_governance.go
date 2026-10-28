package router

import (
	"github.com/gin-gonic/gin"

	"gaze/internal/capabilities"
	"gaze/internal/handler"
	"gaze/internal/middleware"
)

// RegisterGovernanceRoutes mounts read-only management dashboards and audit
// search. Each route applies a coarse capability gate; the service derives the
// data scope from the caller's membership.
func RegisterGovernanceRoutes(r *gin.RouterGroup, h *handler.GovernanceHandler, g *rbacGuards) {
	if h == nil {
		return
	}
	governance := r.Group("/governance", g.Viewer())
	{
		governance.GET("/dashboard/dept", middleware.RequireCapability(capabilities.StatsDeptRead), h.DeptDashboard)
		governance.GET("/dashboard/company", middleware.RequireCapability(capabilities.StatsCompanyRead), h.CompanyDashboard)
		governance.GET("/audit",
			middleware.RequireAnyCapability(capabilities.AuditDeptRead, capabilities.AuditCompanyRead),
			h.Audit)
	}
}
