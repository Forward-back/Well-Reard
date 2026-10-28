package interfaces

import (
	"context"

	"gaze/internal/types"
)

// DepartmentRepository persists departments and the department half of
// tenant_members. It deliberately does not enforce business invariants; the
// service layer owns those.
type DepartmentRepository interface {
	Create(ctx context.Context, dept *types.Department, previousDepartmentCount int64) error
	Get(ctx context.Context, tenantID uint64, id string) (*types.Department, error)
	List(ctx context.Context, tenantID uint64) ([]*types.Department, error)
	Update(ctx context.Context, dept *types.Department) error
	SoftDelete(ctx context.Context, tenantID uint64, id string) error
	EnsureDefault(ctx context.Context, tenantID uint64) (*types.Department, error)

	CountDepartments(ctx context.Context, tenantID uint64) (int64, error)
	CountMembers(ctx context.Context, tenantID uint64, departmentID string) (int64, error)
	CountKnowledgeBases(ctx context.Context, tenantID uint64, departmentID string) (int64, error)
	CountShareReferences(ctx context.Context, tenantID uint64, departmentID string) (int64, error)
	CountManagers(ctx context.Context, tenantID uint64, departmentID string) (int64, error)

	ListMembers(ctx context.Context, tenantID uint64, departmentID string) ([]*types.TenantMember, error)
	UpdateMemberAssignment(ctx context.Context, tenantID uint64, userID string, departmentID string, role types.DepartmentRole) error
	RemoveMemberAssignment(ctx context.Context, tenantID uint64, userID string, departmentID string) error
	ListCRAuditors(ctx context.Context, tenantID uint64, departmentID string) ([]*types.DepartmentCRAuditor, error)
	IsCRAuditor(ctx context.Context, tenantID uint64, departmentID, userID string) (bool, error)
	AddCRAuditor(ctx context.Context, row *types.DepartmentCRAuditor) error
	RemoveCRAuditor(ctx context.Context, tenantID uint64, departmentID, userID string) error
}

// DepartmentService is the business layer for department spaces.
type DepartmentService interface {
	EnsureDefault(ctx context.Context, tenantID uint64) (*types.Department, error)
	List(ctx context.Context, tenantID uint64) ([]*types.Department, error)
	Create(ctx context.Context, tenantID uint64, name, description string) (*types.Department, error)
	Update(ctx context.Context, tenantID uint64, id, name, description string) (*types.Department, error)
	Delete(ctx context.Context, tenantID uint64, id string) error
	ListMembers(ctx context.Context, tenantID uint64, id string) ([]*types.TenantMember, error)
	AssignMember(ctx context.Context, tenantID uint64, departmentID, userID string, role types.DepartmentRole) error
	RemoveMember(ctx context.Context, tenantID uint64, departmentID, userID string) error
	ListCRAuditors(ctx context.Context, tenantID uint64, departmentID string) ([]*types.DepartmentCRAuditor, error)
	AddCRAuditor(ctx context.Context, tenantID uint64, departmentID, userID string) error
	RemoveCRAuditor(ctx context.Context, tenantID uint64, departmentID, userID string) error
	IsCRAuditor(ctx context.Context, tenantID uint64, departmentID, userID string) (bool, error)
}
