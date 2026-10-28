package service

import (
	"context"
	"testing"

	"gaze/internal/types"
)

func TestDepartmentSkillScope(t *testing.T) {
	managerA := types.WithCaller(context.Background(), types.Caller{
		TenantID: 1, Role: types.TenantRoleViewer, DepartmentID: "dept-a",
	})
	companyAdmin := types.WithCaller(context.Background(), types.Caller{
		TenantID: 1, Role: types.TenantRoleCompanyAdmin, DepartmentID: "dept-a",
	})
	skillA := &types.TenantSkillCatalogEntity{DepartmentID: "dept-a"}
	skillB := &types.TenantSkillCatalogEntity{DepartmentID: "dept-b"}
	legacy := &types.TenantSkillCatalogEntity{DepartmentID: ""}

	if !canViewDepartmentSkill(managerA, skillA) {
		t.Fatal("department manager should view own skill")
	}
	if canViewDepartmentSkill(managerA, skillB) {
		t.Fatal("department manager must not view another department's skill")
	}
	if !canViewDepartmentSkill(managerA, legacy) {
		t.Fatal("legacy tenant-wide skills stay visible")
	}
	if !canViewDepartmentSkill(companyAdmin, skillB) {
		t.Fatal("company admin should read every department")
	}
	if !canManageDepartmentSkill(managerA, skillA) {
		t.Fatal("department manager should edit own skill")
	}
	if canManageDepartmentSkill(managerA, skillB) {
		t.Fatal("department manager must not edit another department's skill")
	}
	if canManageDepartmentSkill(companyAdmin, skillB) {
		t.Fatal("company admin cross-department access is read-only")
	}
	if !canManageDepartmentSkill(context.Background(), skillB) {
		t.Fatal("system/service context without a department keeps legacy access")
	}
}
