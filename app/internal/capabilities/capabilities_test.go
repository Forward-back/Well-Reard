package capabilities

import (
	"encoding/json"
	"strings"
	"testing"

	"gaze/contracts"
)

func TestEmbeddedContractIsValid(t *testing.T) {
	c, err := Load(contracts.CapabilitiesJSON)
	if err != nil {
		t.Fatalf("embedded contract invalid: %v", err)
	}
	if len(c.Keys) != len(allowedKeys) {
		t.Fatalf("keys = %d, want %d", len(c.Keys), len(allowedKeys))
	}
	for _, role := range []string{
		RoleCompanyAdmin, RoleMember, RoleManager, RoleKnowledgeAdmin,
		RoleEditor, RoleViewer, RoleSystemAdmin,
	} {
		if _, ok := c.Roles[role]; !ok {
			t.Fatalf("missing role %q", role)
		}
	}
}

func TestContractRejectsUnknownKey(t *testing.T) {
	raw := map[string]any{
		"keys":  []string{KBRead, "nemesis:everything"},
		"roles": map[string][]string{RoleMember: {KBRead}},
	}
	data, _ := json.Marshal(raw)
	if _, err := Load(data); err == nil || !strings.Contains(err.Error(), "unknown key") {
		t.Fatalf("expected unknown key error, got %v", err)
	}
}

func TestContractRejectsUnknownRoleCap(t *testing.T) {
	raw := map[string]any{
		"keys": allKeys(),
		"roles": map[string][]string{
			RoleCompanyAdmin: {}, RoleMember: {KBRead}, RoleManager: {},
			RoleKnowledgeAdmin: {}, RoleEditor: {}, RoleViewer: {},
			RoleSystemAdmin: {"kb:teleport"},
		},
	}
	data, _ := json.Marshal(raw)
	if _, err := Load(data); err == nil || !strings.Contains(err.Error(), "unknown key") {
		t.Fatalf("expected unknown key error, got %v", err)
	}
}

func TestCapsForRoleMatrix(t *testing.T) {
	c := MustDefault()
	tests := []struct {
		name       string
		company    string
		department string
		want       []string
		reject     []string
	}{
		{
			name:    "company admin is governance read-only for business",
			company: RoleCompanyAdmin, department: RoleViewer,
			want:   []string{MemberManage, DepartmentManage, StatsCompanyRead, AuditCompanyRead},
			reject: []string{KBRead, SharedSpaceView, KBCreate, KBContentEdit, KBShareManage, KBAclManage, FolderManage, SettingsModelsManage, SettingsInfraManage, SettingsIntegrationsManage, SettingsWorkspaceManage, SettingsSkillsManage},
		},
		{
			name:    "manager carries department business caps",
			company: RoleMember, department: RoleManager,
			want: []string{MemberManage, KBCreate, KBContentEdit, KBShareManage, KBAclManage, FolderManage, AuditDeptRead},
		},
		{
			name:    "knowledge admin cannot share or audit",
			company: RoleMember, department: RoleKnowledgeAdmin,
			want:   []string{KBCreate, KBContentEdit, StatsDeptRead},
			reject: []string{KBAclManage, KBShareManage, AuditDeptRead},
		},
		{
			name:    "editor edits only",
			company: RoleMember, department: RoleEditor,
			want:   []string{KBRead, KBContentEdit, SkillsView},
			reject: []string{KBCreate, KBAclManage, StatsDeptRead},
		},
		{
			name:    "viewer reads shared space",
			company: RoleMember, department: RoleViewer,
			want:   []string{KBRead, SharedSpaceView, SettingsSelfManage},
			reject: []string{KBContentEdit, SkillsView},
		},
		{
			name:    "system admin is separate from membership caps",
			company: RoleMember, department: RoleViewer, want: []string{KBRead},
			reject: []string{SettingsSystemManage},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := c.CapsFor(tc.company, tc.department, false)
			for _, key := range tc.want {
				if !Has(got, key) {
					t.Errorf("missing %q in %v", key, got)
				}
			}
			for _, key := range tc.reject {
				if Has(got, key) {
					t.Errorf("unexpected %q in %v", key, got)
				}
			}
		})
	}
	systemCaps := c.CapsFor(RoleSystemAdmin, "", true)
	for _, key := range []string{
		SettingsSystemManage,
		SettingsModelsManage,
		SettingsInfraManage,
		SettingsIntegrationsManage,
		SettingsWorkspaceManage,
	} {
		if !Has(systemCaps, key) {
			t.Fatalf("system admin caps missing %s: %v", key, systemCaps)
		}
	}
	if Has(c.CapsFor(RoleCompanyAdmin, "", false), SettingsSystemManage) {
		t.Fatalf("company admin must not receive settings:system_manage")
	}
}

func TestLegacyRoleAliases(t *testing.T) {
	c := MustDefault()
	if got := c.CapsFor("owner", "", false); !Has(got, MemberManage) {
		t.Fatalf("owner alias should map to company_admin: %v", got)
	}
	if got := c.CapsFor("contributor", RoleEditor, false); !Has(got, KBContentEdit) {
		t.Fatalf("contributor alias should map to member and keep editor caps: %v", got)
	}
}

func allKeys() []string {
	out := make([]string, 0, len(allowedKeys))
	for key := range allowedKeys {
		out = append(out, key)
	}
	return out
}
