package access

import (
	"context"
	"errors"
	"testing"

	"gaze/internal/types"
	"github.com/stretchr/testify/require"
)

type departmentShareLookup struct {
	shared bool
	err    error
	calls  int
}

func (s *departmentShareLookup) IsKnowledgeBaseSharedToDepartment(
	context.Context, uint64, string, string,
) (bool, error) {
	s.calls++
	return s.shared, s.err
}

func TestResolveKBDepartmentScope(t *testing.T) {
	for _, tt := range []struct {
		name      string
		caller    types.Caller
		kbDept    string
		lookup    *departmentShareLookup
		required  types.OrgMemberRole
		apiKey    bool
		want      types.OrgMemberRole
		wantErr   error
		wantCalls int
	}{
		{
			name:     "viewer reads own department",
			caller:   types.Caller{TenantID: 1, Role: types.TenantRoleMember, DepartmentID: "d1", DepartmentRole: types.DepartmentRoleViewer},
			kbDept:   "d1",
			required: types.OrgRoleViewer,
			want:     types.OrgRoleViewer,
		},
		{
			name:     "viewer cannot write own department",
			caller:   types.Caller{TenantID: 1, Role: types.TenantRoleMember, DepartmentID: "d1", DepartmentRole: types.DepartmentRoleViewer},
			kbDept:   "d1",
			required: types.OrgRoleEditor,
			wantErr:  ErrForbidden,
		},
		{
			name:     "editor writes own department",
			caller:   types.Caller{TenantID: 1, Role: types.TenantRoleMember, DepartmentID: "d1", DepartmentRole: types.DepartmentRoleEditor},
			kbDept:   "d1",
			required: types.OrgRoleEditor,
			want:     types.OrgRoleEditor,
		},
		{
			name:     "knowledge admin writes own department",
			caller:   types.Caller{TenantID: 1, Role: types.TenantRoleMember, DepartmentID: "d1", DepartmentRole: types.DepartmentRoleKnowledgeAdmin},
			kbDept:   "d1",
			required: types.OrgRoleEditor,
			want:     types.OrgRoleEditor,
		},
		{
			name:     "manager writes own department",
			caller:   types.Caller{TenantID: 1, Role: types.TenantRoleMember, DepartmentID: "d1", DepartmentRole: types.DepartmentRoleManager},
			kbDept:   "d1",
			required: types.OrgRoleEditor,
			want:     types.OrgRoleAdmin,
		},
		{
			name:     "company admin reads another department",
			caller:   types.Caller{TenantID: 1, Role: types.TenantRoleCompanyAdmin},
			kbDept:   "d2",
			required: types.OrgRoleViewer,
			want:     types.OrgRoleViewer,
		},
		{
			name:     "company admin cannot write another department",
			caller:   types.Caller{TenantID: 1, Role: types.TenantRoleCompanyAdmin},
			kbDept:   "d2",
			required: types.OrgRoleEditor,
			wantErr:  ErrForbidden,
		},
		{
			name:     "company admin with manager role writes own department",
			caller:   types.Caller{TenantID: 1, Role: types.TenantRoleCompanyAdmin, DepartmentID: "d1", DepartmentRole: types.DepartmentRoleManager},
			kbDept:   "d1",
			required: types.OrgRoleEditor,
			want:     types.OrgRoleAdmin,
		},
		{
			name:      "shared-in reader",
			caller:    types.Caller{TenantID: 1, Role: types.TenantRoleMember, DepartmentID: "d1", DepartmentRole: types.DepartmentRoleViewer},
			kbDept:    "d2",
			lookup:    &departmentShareLookup{shared: true},
			required:  types.OrgRoleViewer,
			want:      types.OrgRoleViewer,
			wantCalls: 1,
		},
		{
			name:      "shared-in is read only",
			caller:    types.Caller{TenantID: 1, Role: types.TenantRoleMember, DepartmentID: "d1", DepartmentRole: types.DepartmentRoleEditor},
			kbDept:    "d2",
			lookup:    &departmentShareLookup{shared: true},
			required:  types.OrgRoleEditor,
			wantErr:   ErrForbidden,
			wantCalls: 1,
		},
		{
			name:      "other department without share denied",
			caller:    types.Caller{TenantID: 1, Role: types.TenantRoleMember, DepartmentID: "d1", DepartmentRole: types.DepartmentRoleManager},
			kbDept:    "d2",
			lookup:    &departmentShareLookup{},
			required:  types.OrgRoleViewer,
			wantErr:   ErrForbidden,
			wantCalls: 1,
		},
		{
			name:      "share lookup error fails closed",
			caller:    types.Caller{TenantID: 1, Role: types.TenantRoleMember, DepartmentID: "d1", DepartmentRole: types.DepartmentRoleViewer},
			kbDept:    "d2",
			lookup:    &departmentShareLookup{err: errors.New("database unavailable")},
			required:  types.OrgRoleViewer,
			wantCalls: 1,
		},
		{
			name:     "member without department denied",
			caller:   types.Caller{TenantID: 1, Role: types.TenantRoleMember},
			kbDept:   "d1",
			required: types.OrgRoleViewer,
			wantErr:  ErrForbidden,
		},
		{
			name:     "company admin reads legacy KB without department",
			caller:   types.Caller{TenantID: 1, Role: types.TenantRoleCompanyAdmin},
			required: types.OrgRoleViewer,
			want:     types.OrgRoleViewer,
		},
		{
			name:     "api key is company level",
			caller:   types.Caller{TenantID: 1, Role: types.TenantRoleMember},
			kbDept:   "d1",
			apiKey:   true,
			required: types.OrgRoleEditor,
			want:     types.OrgRoleAdmin,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			kb := &types.KnowledgeBase{ID: "kb", TenantID: 1, DepartmentID: tt.kbDept}
			ctx := context.Background()
			if tt.apiKey {
				ctx = types.WithTenantAPIKeyScope(ctx, types.TenantAPIKeyScope{
					KnowledgeBaseIDs: types.StringArray{"kb"},
				})
			}
			grant, err := ResolveKB(
				ctx,
				KBRequest{Caller: tt.caller},
				kb,
				tt.required,
				nil,
				nil,
				tt.lookup,
			)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, grant)
			} else if tt.lookup != nil && tt.lookup.err != nil {
				require.Error(t, err)
				require.Nil(t, grant)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, grant.Permission)
			}
			if tt.lookup != nil {
				require.Equal(t, tt.wantCalls, tt.lookup.calls)
			}
		})
	}
}

func TestKBPermissionsCheckKBUsesDepartmentScope(t *testing.T) {
	kb := &types.KnowledgeBase{ID: "kb", TenantID: 1, DepartmentID: "d1"}
	ctx := types.WithCaller(context.Background(), types.Caller{
		TenantID: 1, UserID: "u", Role: types.TenantRoleMember,
		DepartmentID: "d1", DepartmentRole: types.DepartmentRoleEditor,
	})
	permissions := NewKBPermissions(ctx, nil)

	allowed, err := permissions.CheckKB(kb, types.OrgRoleEditor)
	require.NoError(t, err)
	require.True(t, allowed)

	allowed, err = permissions.CheckKB(kb, types.OrgRoleAdmin)
	require.NoError(t, err)
	require.False(t, allowed)

	adminCtx := types.WithCaller(context.Background(), types.Caller{
		TenantID: 1, UserID: "admin", Role: types.TenantRoleCompanyAdmin,
	})
	adminPermissions := NewKBPermissions(adminCtx, nil)
	allowed, err = adminPermissions.CheckKB(kb, types.OrgRoleViewer)
	require.NoError(t, err)
	require.True(t, allowed)
	allowed, err = adminPermissions.CheckKB(kb, types.OrgRoleEditor)
	require.NoError(t, err)
	require.False(t, allowed)
}
