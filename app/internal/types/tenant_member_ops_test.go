package types

import "testing"

func TestTenantRoleOpsIsInternalOnly(t *testing.T) {
	if !TenantRoleOps.IsValid() {
		t.Fatal("ops role must be a valid context role")
	}
	if TenantRoleOps.IsAssignable() {
		t.Fatal("ops role must not be assignable through member/invitation management")
	}
	if TenantRoleOps.HasPermission(TenantRoleViewer) {
		t.Fatal("ops role must rank below every business role")
	}
	if !TenantRoleCompanyAdmin.IsAssignable() || !TenantRoleMember.IsAssignable() {
		t.Fatal("company admin and member must remain assignable")
	}
}
