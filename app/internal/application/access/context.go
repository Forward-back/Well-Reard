package access

import (
	"context"

	"gaze/internal/types"
)

// Context grants contain values, not mutable KB objects. Deriving a context
// copies the grant slice so parallel search branches cannot widen each other.
type kbGrant struct {
	caller     types.Caller
	kbID       string
	tenantID   uint64
	permission types.OrgMemberRole
	task       bool
}

type agentGrant struct {
	caller types.Caller
	scope  types.SharedAgentKBScope
}

// WithSharedAgent scopes execution after the caller has authorized this exact
// agent. Only its configured KBs receive read access; other source-tenant KBs
// and the source tenant's organization memberships are not inherited.
func WithSharedAgent(ctx context.Context, agent *types.CustomAgent) context.Context {
	grant := agentGrant{caller: types.CallerFromContext(ctx), scope: types.NewSharedAgentKBScope(agent)}
	ctx = context.WithValue(ctx, types.SharedAgentGrantContextKey, grant)
	if agent == nil {
		return ctx
	}
	return types.WithExecutionTenant(ctx, agent.TenantID)
}

// HasKBGrant checks only previously resolved resource access. It never infers
// ownership from the execution tenant, and always reapplies API-key scope.
func HasKBGrant(ctx context.Context, kbID string, tenantID uint64, required types.OrgMemberRole) bool {
	if !required.IsValid() || types.AuthorizeTenantAPIKeyKnowledgeBases(ctx, kbID) != nil {
		return false
	}
	caller := types.CallerFromContext(ctx)
	if kbID == "" || tenantID == 0 {
		return false
	}
	grants, _ := ctx.Value(types.KBGrantsContextKey).([]kbGrant)
	for _, grant := range grants {
		if (caller.TenantID != 0 || grant.task) && grant.caller == caller && grant.kbID == kbID &&
			grant.tenantID == tenantID &&
			grant.permission.HasPermission(required) {
			return true
		}
	}
	grant, ok := ctx.Value(types.SharedAgentGrantContextKey).(agentGrant)
	return ok && caller.TenantID != 0 && required == types.OrgRoleViewer && grant.caller == caller &&
		grant.scope.Allows(kbID, tenantID)
}

// KBPermissions combines caller ownership, exact context grants, department
// scope and organization shares for one service operation. Pass nil shares
// when that entry point does not permit organization expansion (for example, a
// caller without a user).
type KBPermissions struct {
	ctx        context.Context
	caller     types.Caller
	shares     *KBSharePermissions
	deptShares DepartmentKBShareLookup
}

// NewKBPermissions resolves reads for one caller and operation.
func NewKBPermissions(
	ctx context.Context,
	shares KBShareLookup,
	departmentShares ...DepartmentKBShareLookup,
) *KBPermissions {
	caller := types.CallerFromContext(ctx)
	var deptShares DepartmentKBShareLookup
	if len(departmentShares) > 0 {
		deptShares = departmentShares[0]
	}
	return &KBPermissions{
		ctx:        ctx,
		caller:     caller,
		shares:     NewKBSharePermissions(ctx, shares, caller.TenantID, caller.Role),
		deptShares: deptShares,
	}
}

// CheckKB combines exact grants, department scope and organization sharing.
// Same-tenant reads must use this method: tenant equality alone never grants
// access, because departments are the business boundary inside one company.
func (p *KBPermissions) CheckKB(kb *types.KnowledgeBase, required types.OrgMemberRole) (bool, error) {
	if kb == nil || kb.ID == "" || kb.TenantID == 0 || !required.IsValid() {
		return false, nil
	}
	if err := types.AuthorizeTenantAPIKeyKnowledgeBases(p.ctx, kb.ID); err != nil {
		return false, err
	}
	if HasKBGrant(p.ctx, kb.ID, kb.TenantID, required) {
		return true, nil
	}
	if p.caller.TenantID == 0 {
		return false, nil
	}
	if p.caller.TenantID == kb.TenantID {
		permission, err := sameTenantPermission(p.ctx, p.caller, kb, p.deptShares)
		if err != nil {
			return false, err
		}
		if permission != "" && permission.HasPermission(required) {
			return true, nil
		}
		return required == types.OrgRoleViewer &&
			p.caller.Role.HasPermission(types.TenantRoleCompanyAdmin), nil
	}
	return p.shares.Check(kb.ID, required)
}

// Check is the legacy ID-only form. It cannot resolve a KB's department, so
// it retains the old tenant-equality read behavior for retrieval callers that
// have not yet been moved to CheckKB. Do not use it for route or mutation
// authorization; those paths must use CheckKB / ResolveKB.
func (p *KBPermissions) Check(kbID string, ownerTenantID uint64, required types.OrgMemberRole) (bool, error) {
	if kbID == "" || ownerTenantID == 0 || !required.IsValid() {
		return false, nil
	}
	if err := types.AuthorizeTenantAPIKeyKnowledgeBases(p.ctx, kbID); err != nil {
		return false, err
	}
	if (p.caller.TenantID == ownerTenantID && required == types.OrgRoleViewer) ||
		HasKBGrant(p.ctx, kbID, ownerTenantID, required) {
		return true, nil
	}
	if p.caller.TenantID == 0 || p.caller.TenantID == ownerTenantID {
		return false, nil
	}
	return p.shares.Check(kbID, required)
}
