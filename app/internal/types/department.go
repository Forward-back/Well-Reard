package types

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// DepartmentRole is a user's role inside one department. It is orthogonal to
// the company role on TenantMember (company_admin | member): platform
// administration comes from the company role, business authority from the
// department role.
type DepartmentRole string

const (
	// DepartmentRoleManager owns a department space: share scope, all ACL
	// rules (including all-company deny), content, and department roles.
	DepartmentRoleManager DepartmentRole = "manager"
	// DepartmentRoleKnowledgeAdmin maintains department KB content and
	// folder/file ACL (user/department deny), but cannot change the
	// cross-department share scope or set all-company deny.
	DepartmentRoleKnowledgeAdmin DepartmentRole = "knowledge_admin"
	// DepartmentRoleEditor uploads/edits content and participates in CRs.
	DepartmentRoleEditor DepartmentRole = "editor"
	// DepartmentRoleViewer reads department KBs and shared-in KBs, and may
	// file issues/comments.
	DepartmentRoleViewer DepartmentRole = "viewer"
)

var departmentRoleLevel = map[DepartmentRole]int{
	DepartmentRoleManager:        40,
	DepartmentRoleKnowledgeAdmin: 30,
	DepartmentRoleEditor:         20,
	DepartmentRoleViewer:         10,
}

// IsValid reports whether r is a defined department role.
func (r DepartmentRole) IsValid() bool {
	_, ok := departmentRoleLevel[r]
	return ok
}

// Level returns the privilege level; unknown roles are below every defined role.
func (r DepartmentRole) Level() int {
	return departmentRoleLevel[r]
}

// HasPermission reports whether r is at least as privileged as required.
func (r DepartmentRole) HasPermission(required DepartmentRole) bool {
	return r.Level() >= required.Level()
}

// Department is a company's business department/space.
type Department struct {
	ID          string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID    uint64         `json:"tenant_id" gorm:"not null;index"`
	Name        string         `json:"name" gorm:"type:varchar(128);not null"`
	Description string         `json:"description" gorm:"type:text;not null;default:''"`
	IsDefault   bool           `json:"is_default" gorm:"not null;default:false"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Counts are computed by the service; never persisted.
	MemberCount        int64 `json:"member_count" gorm:"-"`
	KnowledgeBaseCount int64 `json:"knowledge_base_count" gorm:"-"`
}

// TableName binds Department to departments.
func (Department) TableName() string { return "departments" }

// DefaultDepartmentID is deterministic so migrations and lazy backfills can
// refer to a tenant's default department without an extra lookup.
func DefaultDepartmentID(tenantID uint64) string {
	return fmt.Sprintf("dept-default-%d", tenantID)
}

// DepartmentName looks safe for empty names in API responses.
func (d Department) DisplayName() string {
	if name := strings.TrimSpace(d.Name); name != "" {
		return name
	}
	return d.ID
}

// KBDepartmentShare is one row of a knowledge base's materialized department
// share scope. The owning department always has a row.
type KBDepartmentShare struct {
	ID             string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID       uint64    `json:"tenant_id" gorm:"not null;index"`
	KnowledgeBaseID string   `json:"kb_id" gorm:"column:kb_id;type:varchar(36);not null;uniqueIndex:idx_kb_department_shares_unique,priority:1"`
	DepartmentID   string    `json:"department_id" gorm:"type:varchar(36);not null;uniqueIndex:idx_kb_department_shares_unique,priority:2"`
	CreatedBy      string    `json:"created_by" gorm:"type:varchar(36);not null;default:''"`
	CreatedAt      time.Time `json:"created_at"`
}

// TableName binds KBDepartmentShare to kb_department_shares.
func (KBDepartmentShare) TableName() string { return "kb_department_shares" }

// DepartmentAssignment is the department half of a membership.
type DepartmentAssignment struct {
	DepartmentID   string
	DepartmentRole DepartmentRole
}

// DepartmentCRAuditor grants a department member permission to review
// (approve / request changes) change requests across all KBs in the
// department. It never grants merge or close authority.
type DepartmentCRAuditor struct {
	TenantID     uint64    `json:"tenant_id" gorm:"primaryKey"`
	DepartmentID string    `json:"department_id" gorm:"type:varchar(36);primaryKey"`
	UserID       string    `json:"user_id" gorm:"type:varchar(36);primaryKey"`
	GrantedBy    string    `json:"granted_by" gorm:"type:varchar(36);not null"`
	CreatedAt    time.Time `json:"created_at"`
}
