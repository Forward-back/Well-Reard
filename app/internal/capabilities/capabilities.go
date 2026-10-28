// Package capabilities loads and validates the shared capability contract and
// derives a caller's capability set from company and department roles.
//
// Capabilities are a projection of existing roles, never an authorization
// source of their own: resource guards (department shares, KB grants, ACL)
// remain the security boundary.
package capabilities

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"gaze/contracts"
)

// Capability keys. Every key in contracts/capabilities.json must appear here;
// the loader rejects unknown keys so the Go whitelist stays authoritative.
const (
	KBRead                     = "kb:read"
	KBCreate                   = "kb:create"
	KBContentEdit              = "kb:content_edit"
	KBSettingsManage           = "kb:settings_manage"
	KBShareManage              = "kb:share_manage"
	KBAclManage                = "kb:acl_manage"
	FolderManage               = "folder:manage"
	SharedSpaceView            = "shared_space:view"
	MemberView                 = "member:view"
	MemberManage               = "member:manage"
	DepartmentManage           = "department:manage"
	SettingsSelfManage         = "settings:self_manage"
	SettingsModelsManage       = "settings:models_manage"
	SettingsInfraManage        = "settings:infrastructure_manage"
	SettingsIntegrationsManage = "settings:integrations_manage"
	SettingsSkillsManage       = "settings:skills_manage"
	SettingsWorkspaceManage    = "settings:workspace_manage"
	SettingsSystemManage       = "settings:system_manage"
	SkillsView                 = "skills:view"
	CRAuditManage              = "cr:audit_manage"
	StatsDeptRead              = "stats:dept_read"
	StatsCompanyRead           = "stats:company_read"
	AuditDeptRead              = "audit:dept_read"
	AuditCompanyRead           = "audit:company_read"
)

// Role keys used in the contract. Company roles are company_admin/member;
// department roles are manager/knowledge_admin/editor/viewer. system_admin is
// a platform principal and is never merged into membership caps.
const (
	RoleCompanyAdmin   = "company_admin"
	RoleMember         = "member"
	RoleManager        = "manager"
	RoleKnowledgeAdmin = "knowledge_admin"
	RoleEditor         = "editor"
	RoleViewer         = "viewer"
	RoleSystemAdmin    = "system_admin"
)

var allowedKeys = map[string]struct{}{
	KBRead: {}, KBCreate: {}, KBContentEdit: {}, KBSettingsManage: {},
	CRAuditManage: {},
	KBShareManage: {}, KBAclManage: {}, FolderManage: {},
	SharedSpaceView: {}, MemberView: {}, MemberManage: {}, DepartmentManage: {},
	SettingsSelfManage: {}, SettingsModelsManage: {}, SettingsInfraManage: {},
	SettingsIntegrationsManage: {}, SettingsSkillsManage: {},
	SettingsWorkspaceManage: {}, SettingsSystemManage: {},
	SkillsView: {}, StatsDeptRead: {}, StatsCompanyRead: {},
	AuditDeptRead: {}, AuditCompanyRead: {},
}

var allowedRoles = map[string]struct{}{
	RoleCompanyAdmin: {}, RoleMember: {}, RoleManager: {}, RoleKnowledgeAdmin: {},
	RoleEditor: {}, RoleViewer: {}, RoleSystemAdmin: {},
}

// Contract is the parsed capability contract.
type Contract struct {
	Keys  []string            `json:"keys"`
	Roles map[string][]string `json:"roles"`

	keyIndex map[string]int
}

// Load parses and validates a capability contract.
func Load(data []byte) (*Contract, error) {
	var raw Contract
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("capabilities: parse contract: %w", err)
	}
	if err := raw.validate(); err != nil {
		return nil, err
	}
	raw.keyIndex = make(map[string]int, len(raw.Keys))
	for i, key := range raw.Keys {
		raw.keyIndex[key] = i
	}
	return &raw, nil
}

func (c *Contract) validate() error {
	if len(c.Keys) == 0 {
		return fmt.Errorf("capabilities: contract has no keys")
	}
	seen := make(map[string]struct{}, len(c.Keys))
	for _, key := range c.Keys {
		if _, ok := allowedKeys[key]; !ok {
			return fmt.Errorf("capabilities: unknown key %q", key)
		}
		if _, dup := seen[key]; dup {
			return fmt.Errorf("capabilities: duplicate key %q", key)
		}
		seen[key] = struct{}{}
	}
	for key := range allowedKeys {
		if _, ok := seen[key]; !ok {
			return fmt.Errorf("capabilities: whitelisted key %q missing from contract", key)
		}
	}
	if len(c.Roles) == 0 {
		return fmt.Errorf("capabilities: contract has no roles")
	}
	for role, caps := range c.Roles {
		if _, ok := allowedRoles[role]; !ok {
			return fmt.Errorf("capabilities: unknown role %q", role)
		}
		roleSeen := make(map[string]struct{}, len(caps))
		for _, cap := range caps {
			if _, ok := allowedKeys[cap]; !ok {
				return fmt.Errorf("capabilities: role %q references unknown key %q", role, cap)
			}
			if _, dup := roleSeen[cap]; dup {
				return fmt.Errorf("capabilities: role %q repeats key %q", role, cap)
			}
			roleSeen[cap] = struct{}{}
		}
	}
	for role := range allowedRoles {
		if _, ok := c.Roles[role]; !ok {
			return fmt.Errorf("capabilities: role %q missing from contract", role)
		}
	}
	return nil
}

var (
	defaultOnce     sync.Once
	defaultContract *Contract
	defaultErr      error
)

// Default returns the process-wide embedded contract. It is loaded once on
// first use; an invalid contract is a startup/programming error.
func Default() (*Contract, error) {
	defaultOnce.Do(func() {
		defaultContract, defaultErr = Load(contracts.CapabilitiesJSON)
	})
	return defaultContract, defaultErr
}

// MustDefault returns the embedded contract or panics. Use during startup
// wiring where an invalid contract should fail fast.
func MustDefault() *Contract {
	c, err := Default()
	if err != nil {
		panic(err)
	}
	return c
}

// normalizeCompanyRole maps legacy tenant-role aliases onto the two company
// roles. Unknown roles yield no base capabilities.
func normalizeCompanyRole(role string) string {
	switch role {
	case RoleCompanyAdmin, "owner", "admin":
		return RoleCompanyAdmin
	case RoleMember, "contributor", "viewer":
		return RoleMember
	default:
		return ""
	}
}

// CapsFor returns the deduplicated capability set for a membership. The
// result follows contract key order so responses stay stable. A system admin
// may opt in to the platform role, but callers must keep that separate from
// membership caps.
func (c *Contract) CapsFor(companyRole, departmentRole string, systemAdmin bool) []string {
	roles := make([]string, 0, 3)
	normalized := normalizeCompanyRole(companyRole)
	if normalized != "" {
		roles = append(roles, normalized)
	}
	// Company administrators are governance-only: they never inherit a
	// department business role, even if a stale membership assignment exists.
	if normalized != RoleCompanyAdmin {
		if _, ok := allowedRoles[departmentRole]; ok && departmentRole != RoleCompanyAdmin &&
			departmentRole != RoleMember && departmentRole != RoleSystemAdmin {
			roles = append(roles, departmentRole)
		}
	}
	if systemAdmin {
		roles = append(roles, RoleSystemAdmin)
	}
	granted := make(map[string]struct{})
	for _, role := range roles {
		for _, cap := range c.Roles[role] {
			granted[cap] = struct{}{}
		}
	}
	out := make([]string, 0, len(granted))
	for _, key := range c.Keys {
		if _, ok := granted[key]; ok {
			out = append(out, key)
		}
	}
	return out
}

// Has reports whether key is in caps.
func Has(caps []string, key string) bool {
	for _, cap := range caps {
		if cap == key {
			return true
		}
	}
	return false
}

// Keys returns the full contract key list (copy-safe).
func (c *Contract) KeysList() []string {
	out := make([]string, len(c.Keys))
	copy(out, c.Keys)
	return out
}

// RoleNames returns the contract role names in stable order.
func (c *Contract) RoleNames() []string {
	out := make([]string, 0, len(c.Roles))
	for role := range c.Roles {
		out = append(out, role)
	}
	sort.Strings(out)
	return out
}
