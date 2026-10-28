package interfaces

import (
	"context"

	"gaze/internal/types"
)

// GovernanceRepository provides read-only aggregate queries for dashboards
// and department-scoped audit reads.
type GovernanceRepository interface {
	Stats(ctx context.Context, tenantID uint64, departmentID string, company bool) (*types.GovernanceStats, error)
	Audit(ctx context.Context, tenantID uint64, departmentID string, afterID uint64, limit int) ([]*types.GovernanceAuditEntry, error)
}

// GovernanceService applies capability-derived scope on top of the repository.
type GovernanceService interface {
	DeptStats(ctx context.Context, requestedDepartmentID string) (*types.GovernanceStats, error)
	CompanyStats(ctx context.Context, departmentID string) (*types.GovernanceStats, error)
	Audit(ctx context.Context, departmentID string, afterID uint64, limit int) ([]*types.GovernanceAuditEntry, error)
}
