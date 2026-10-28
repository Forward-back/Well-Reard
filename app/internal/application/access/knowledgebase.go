// Package access resolves resource permissions independently of HTTP routing.
// Tenant roles, API-key capabilities, and resource ownership guards remain
// separate checks; a KB grant does not replace them.
package access

import (
	"context"
	"errors"
	"fmt"

	"gaze/internal/types"
)

// Resource access errors retain distinct adapter response mappings.
var (
	ErrUnauthorized       = errors.New("kb_access: unauthorized")
	ErrNotFound           = errors.New("kb_access: not found")
	ErrForbidden          = errors.New("kb_access: forbidden")
	ErrInvalidAgentSource = errors.New("kb_access: invalid agent source")
)

// KBRequest keeps the authenticated caller separate from the resource tenant.
// AgentSourceTenantID is parsed only when the explicit-agent fallback is used.
type KBRequest struct {
	Caller              types.Caller
	AgentID             string
	AgentSourceTenantID string
}

// KBAccess binds a resolved resource permission to its original caller.
type KBAccess struct {
	KnowledgeBase     *types.KnowledgeBase
	Caller            types.Caller
	EffectiveTenantID uint64
	Permission        types.OrgMemberRole
	// A read resolution must not mint a write grant even for the KB owner.
	operationPermission types.OrgMemberRole
}

// Context scopes resource operations without changing the authenticated caller.
func (a *KBAccess) Context(ctx context.Context) context.Context {
	if a == nil || a.KnowledgeBase == nil {
		return ctx
	}
	scoped := a.WithGrant(ctx)
	if types.CallerFromContext(scoped) != a.Caller.Normalize() {
		return ctx
	}
	return types.WithExecutionTenant(scoped, a.EffectiveTenantID)
}

// WithGrant carries this authorization without changing execution scope.
func (a *KBAccess) WithGrant(ctx context.Context) context.Context {
	if a == nil || a.KnowledgeBase == nil {
		return ctx
	}
	caller := a.Caller.Normalize()
	if existing, ok := ctx.Value(types.CallerContextKey).(types.Caller); ok && existing != caller {
		return ctx // A grant cannot replace an already authenticated caller.
	}
	grants, _ := ctx.Value(types.KBGrantsContextKey).([]kbGrant)
	permission := a.Permission
	if a.operationPermission.IsValid() && permission.HasPermission(a.operationPermission) {
		permission = a.operationPermission
	}
	grants = append(
		append([]kbGrant(nil), grants...),
		kbGrant{caller: caller, kbID: a.KnowledgeBase.ID, tenantID: a.EffectiveTenantID, permission: permission},
	)
	return context.WithValue(types.WithCaller(ctx, caller), types.KBGrantsContextKey, grants)
}

// KBShareLookup resolves organization sharing with the caller role cap.
type KBShareLookup interface {
	CheckTenantKBPermission(context.Context, string, uint64, types.TenantRole) (types.OrgMemberRole, bool, error)
}

// DepartmentKBShareLookup resolves the materialized department share scope of
// a knowledge base. It is the same-tenant counterpart of KBShareLookup.
type DepartmentKBShareLookup interface {
	IsKnowledgeBaseSharedToDepartment(ctx context.Context, tenantID uint64, kbID, departmentID string) (bool, error)
}

// departmentPermission maps a department role to the KB permission ladder.
// Company administrators without a department role are handled separately:
// they get read-through only, never a business write grant.
func departmentPermission(role types.DepartmentRole) types.OrgMemberRole {
	switch role {
	case types.DepartmentRoleManager:
		return types.OrgRoleAdmin
	case types.DepartmentRoleKnowledgeAdmin:
		return types.OrgRoleEditor
	case types.DepartmentRoleEditor:
		return types.OrgRoleEditor
	case types.DepartmentRoleViewer:
		return types.OrgRoleViewer
	default:
		return ""
	}
}

// sameTenantPermission resolves the caller's business permission inside one
// company. A KB owned by another department is readable only when that
// department is in the KB's share scope.
func sameTenantPermission(
	ctx context.Context,
	caller types.Caller,
	kb *types.KnowledgeBase,
	departmentShares DepartmentKBShareLookup,
) (types.OrgMemberRole, error) {
	if _, ok := types.TenantAPIKeyScopeFromContext(ctx); ok {
		// API keys are company-level machine principals. Their capability
		// gates and KB allow-list are enforced by the route/API-key layer,
		// not by a human's department.
		return types.OrgRoleAdmin, nil
	}
	if caller.DepartmentID == "" || kb.DepartmentID == "" {
		return "", nil
	}
	if caller.DepartmentID == kb.DepartmentID {
		return departmentPermission(caller.DepartmentRole), nil
	}
	if departmentShares == nil {
		return "", nil
	}
	shared, err := departmentShares.IsKnowledgeBaseSharedToDepartment(
		ctx, kb.TenantID, kb.ID, caller.DepartmentID,
	)
	if err != nil {
		return "", err
	}
	if shared {
		return types.OrgRoleViewer, nil
	}
	return "", nil
}

// SharedAgentLookup verifies access to a particular shared agent.
type SharedAgentLookup interface {
	GetSharedAgentForTenant(context.Context, uint64, types.TenantRole, string, ...uint64) (*types.CustomAgent, error)
}

// AgentShareLookup also supports read access through any reachable agent.
type AgentShareLookup interface {
	SharedAgentLookup
	TenantCanAccessKBViaSomeSharedAgent(context.Context, uint64, types.TenantRole, *types.KnowledgeBase) (bool, error)
}

// ResolveKB authorizes a server-loaded KB (or a document's persisted KB/tenant
// reference). Its tenant is authoritative; resolving the owner from a share
// record again would add a lookup and a second source of truth. It never treats
// an effective resource tenant as the caller.
//
// Same-tenant access is department-scoped: a KB owned by the caller's
// department follows the department role ladder; a KB from another department
// is readable only when the department share row exists and is read-only.
// Company administrators without a department role get read-through only.
// API keys remain company-level machine principals.
//
// Share lookup errors do not grant access; an independent agent grant may still
// allow reading. An explicit agent selector never falls back to another agent.
func ResolveKB(ctx context.Context, request KBRequest, kb *types.KnowledgeBase, required types.OrgMemberRole,
	shares KBShareLookup, agents AgentShareLookup, departmentShares ...DepartmentKBShareLookup,
) (*KBAccess, error) {
	if request.Caller.TenantID == 0 {
		return nil, ErrUnauthorized
	}
	if kb == nil || kb.ID == "" {
		return nil, ErrNotFound
	}
	if err := types.AuthorizeTenantAPIKeyKnowledgeBases(ctx, kb.ID); err != nil {
		return nil, err
	}
	request.Caller = request.Caller.Normalize()
	if caller, ok := ctx.Value(types.CallerContextKey).(types.Caller); ok && caller != request.Caller {
		return nil, ErrForbidden
	}
	grant := func(permission types.OrgMemberRole) (*KBAccess, error) {
		return &KBAccess{
			KnowledgeBase: kb, Caller: request.Caller,
			EffectiveTenantID: kb.TenantID, Permission: permission, operationPermission: required,
		}, nil
	}
	if kb.TenantID == request.Caller.TenantID {
		var deptShares DepartmentKBShareLookup
		if len(departmentShares) > 0 {
			deptShares = departmentShares[0]
		}
		permission, err := sameTenantPermission(ctx, request.Caller, kb, deptShares)
		if err != nil {
			return nil, err
		}
		if permission != "" && permission.HasPermission(required) {
			return grant(permission)
		}
		// Platform administrators may inspect any company KB, but that is a
		// read-through right: it never satisfies an Editor/Admin business
		// mutation. A company admin who also holds a department role already
		// got that stronger grant above.
		if required == types.OrgRoleViewer &&
			request.Caller.Role.HasPermission(types.TenantRoleCompanyAdmin) {
			return grant(types.OrgRoleViewer)
		}
		return nil, ErrForbidden
	}
	if shares != nil {
		permissions := NewKBSharePermissions(ctx, shares, request.Caller.TenantID, request.Caller.Role)
		permission, shared, err := permissions.Permission(kb.ID)
		if err == nil && shared && permission.HasPermission(required) {
			return grant(permission)
		}
	}
	if required != types.OrgRoleViewer || agents == nil {
		return nil, ErrForbidden
	}
	if request.AgentID != "" {
		source, err := types.ParseAgentSourceTenantID(request.AgentSourceTenantID)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidAgentSource, err)
		}
		agent, err := agents.GetSharedAgentForTenant(
			ctx,
			request.Caller.TenantID,
			request.Caller.Role,
			request.AgentID,
			source,
		)
		if err == nil && types.SharedAgentIncludesKB(agent, kb) {
			return grant(types.OrgRoleViewer)
		}
	} else {
		allowed,
			err := agents.TenantCanAccessKBViaSomeSharedAgent(ctx,
			request.Caller.TenantID,
			request.Caller.Role,
			kb)
		if err == nil && allowed {
			return grant(types.OrgRoleViewer)
		}
	}
	return nil, ErrForbidden
}
