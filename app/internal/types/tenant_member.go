package types

import (
	"time"

	"gorm.io/gorm"
)

// TenantRole represents a user's role inside a single tenant.
//
// Tenant roles govern intra-tenant authority (who can create/edit/delete
// resources, manage tenant settings, etc.) and are orthogonal to the
// OrgMemberRole defined in organization.go, which governs cross-tenant
// sharing. A user may therefore carry different TenantRole values in
// different tenants (one TenantMember row per (user, tenant) pair).
type TenantRole string

const (
	// TenantRoleCompanyAdmin is the merged company-level administrator role:
	// manages company members/departments/platform configuration and may read
	// all non-denied company content. It does not imply business (KB) write
	// authority; that comes from the department role.
	TenantRoleCompanyAdmin TenantRole = "company_admin"
	// TenantRoleMember is a regular company member. Business authority is
	// determined by TenantMember.DepartmentRole.
	TenantRoleMember TenantRole = "member"
	// TenantRoleOps is an internal, non-assignable role for a tenantless
	// system administrator receiving the implicit single-tenant context for
	// technical configuration. It ranks below every business role, so it
	// never satisfies Viewer+ business guards.
	TenantRoleOps TenantRole = "ops"

	// Deprecated aliases kept for compile compatibility while the RBAC surface
	// is migrated. New code must use CompanyAdmin / Member plus DepartmentRole.
	TenantRoleOwner       = TenantRoleCompanyAdmin
	TenantRoleAdmin       = TenantRoleCompanyAdmin
	TenantRoleContributor = TenantRoleMember
	TenantRoleViewer      = TenantRoleMember
)

// tenantRoleLevel maps each role to a numeric level used for hierarchy
// comparisons. Company admin outranks members; business authority is checked
// through DepartmentRole, not this ladder.
var tenantRoleLevel = map[TenantRole]int{
	TenantRoleCompanyAdmin: 30,
	TenantRoleMember:       10,
	TenantRoleOps:          0,
}

// IsValid reports whether r is one of the defined tenant roles.
func (r TenantRole) IsValid() bool {
	_, ok := tenantRoleLevel[r]
	return ok
}

// IsAssignable reports whether the role may be granted through member or
// invitation management. TenantRoleOps is internal-only.
func (r TenantRole) IsAssignable() bool {
	return r == TenantRoleCompanyAdmin || r == TenantRoleMember
}

// Level returns the numeric privilege level of the role. Unknown roles
// return 0, which is strictly less than any defined role.
func (r TenantRole) Level() int {
	return tenantRoleLevel[r]
}

// HasPermission reports whether r is at least as privileged as required.
// Used by RequireRole-style middleware to gate endpoints.
func (r TenantRole) HasPermission(required TenantRole) bool {
	return r.Level() >= required.Level()
}

// TenantMemberStatus enumerates the lifecycle states of a membership row.
type TenantMemberStatus string

const (
	// TenantMemberStatusActive is the normal membership state; the user
	// can authenticate into the tenant and is subject to their role.
	TenantMemberStatusActive TenantMemberStatus = "active"
	// TenantMemberStatusInvited represents a pending invitation that has
	// not yet been accepted. The auth middleware treats this as "not a
	// member" until the status flips to active.
	TenantMemberStatusInvited TenantMemberStatus = "invited"
	// TenantMemberStatusSuspended is an admin-revoked membership. The
	// row is preserved for audit trail but the user cannot authenticate
	// into the tenant.
	TenantMemberStatusSuspended TenantMemberStatus = "suspended"
)

// TenantMember represents the (user, tenant) membership record that
// carries the user's TenantRole for that specific tenant.
//
// A user has one TenantMember row per tenant they belong to. The home
// tenant recorded on User.TenantID is always one of these rows; additional
// rows are created when an admin adds the user to another tenant.
type TenantMember struct {
	// Surrogate primary key.
	ID uint64 `json:"id" gorm:"primaryKey;autoIncrement"`
	// UserID references users.id. Together with TenantID forms the logical
	// key enforced by the partial unique index uniq_user_tenant.
	UserID string `json:"user_id" gorm:"type:varchar(36);not null;index"`
	// TenantID references tenants.id.
	TenantID uint64 `json:"tenant_id" gorm:"not null;index"`
	// Role held by the user inside this tenant.
	Role TenantRole `json:"role" gorm:"type:varchar(20);not null;default:'member'"`
	// DepartmentID is the user's single department inside the company.
	// Nullable only for company administrators (platform staff may sit
	// outside the business department structure).
	DepartmentID *string `json:"department_id,omitempty" gorm:"type:varchar(36);index"`
	// DepartmentRole carries business authority inside DepartmentID.
	DepartmentRole DepartmentRole `json:"department_role,omitempty" gorm:"type:varchar(20)"`
	// DepartmentName is a read-time projection, never persisted.
	DepartmentName string `json:"department_name,omitempty" gorm:"-"`
	// Status controls whether this membership is honoured by the auth
	// middleware; see TenantMemberStatus constants.
	Status TenantMemberStatus `json:"status" gorm:"type:varchar(20);not null;default:'active'"`
	// InvitedBy records the user ID of the admin who created this row via
	// an invitation flow. Nil for rows created by self-service registration.
	InvitedBy *string `json:"invited_by,omitempty" gorm:"type:varchar(36)"`
	// JoinedAt is when the membership became active.
	JoinedAt  time.Time      `json:"joined_at"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

// TableName binds TenantMember to the tenant_members table.
func (TenantMember) TableName() string {
	return "tenant_members"
}

// Membership is the login-response-friendly projection of a TenantMember
// joined with tenant name. Returned as part of LoginResponse so the
// frontend can render a tenant switcher and gate UI by role.
type Membership struct {
	TenantID       uint64         `json:"tenant_id"`
	TenantName     string         `json:"tenant_name"`
	Role           TenantRole     `json:"role"`
	DepartmentID   *string        `json:"department_id,omitempty"`
	DepartmentRole DepartmentRole `json:"department_role,omitempty"`
	// Caps is the role-derived capability set for this membership. Computed
	// server-side from contracts/capabilities.json; UI-only, never an
	// authorization source.
	Caps []string `json:"caps"`
}

// TenantMemberResponse is the API projection of a TenantMember row joined
// with the human-facing user fields the management UI needs (email,
// username, avatar). It is intentionally NOT the GORM model: returning
// the model directly would leak DeletedAt/UpdatedAt and lock the DB
// schema into the public API. Use this for `/tenants/:id/members` only.
type TenantMemberResponse struct {
	UserID         string             `json:"user_id"`
	Email          string             `json:"email"`
	Username       string             `json:"username"`
	Avatar         string             `json:"avatar,omitempty"`
	Role           TenantRole         `json:"role"`
	DepartmentID   *string            `json:"department_id,omitempty"`
	DepartmentRole DepartmentRole     `json:"department_role,omitempty"`
	DepartmentName string             `json:"department_name,omitempty"`
	Status         TenantMemberStatus `json:"status"`
	InvitedBy      *string            `json:"invited_by,omitempty"`
	JoinedAt       time.Time          `json:"joined_at"`
}
