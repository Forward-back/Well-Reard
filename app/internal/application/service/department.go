package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"gaze/internal/types"
	"gaze/internal/types/interfaces"
)

var (
	ErrDepartmentNotFound         = errors.New("department not found")
	ErrDepartmentNameRequired     = errors.New("department name is required")
	ErrDepartmentNonEmpty         = errors.New("department still has members, knowledge bases or share references")
	ErrDefaultDepartmentProtected = errors.New("the default department cannot be deleted")
	ErrDepartmentForbidden        = errors.New("insufficient department permission")
	ErrLastDepartmentManager      = errors.New("department with knowledge bases must keep at least one manager")
	ErrInvalidDepartmentRole      = errors.New("invalid department role")
	ErrDepartmentMemberNotFound   = errors.New("department member not found")
)

type departmentService struct {
	repo       interfaces.DepartmentRepository
	memberRepo interfaces.TenantMemberRepository
	audit      interfaces.AuditLogService
}

// NewDepartmentService wires the department business layer.
func NewDepartmentService(
	repo interfaces.DepartmentRepository,
	memberRepo interfaces.TenantMemberRepository,
	audit interfaces.AuditLogService,
) interfaces.DepartmentService {
	return &departmentService{repo: repo, memberRepo: memberRepo, audit: audit}
}

func isCompanyAdmin(ctx context.Context) bool {
	return types.TenantRoleFromContext(ctx).HasPermission(types.TenantRoleCompanyAdmin)
}

func (s *departmentService) emitAudit(ctx context.Context, tenantID uint64, targetID string, action types.AuditAction, details map[string]any) {
	if s.audit == nil {
		return
	}
	raw, _ := json.Marshal(details)
	uid, _ := types.UserIDFromContext(ctx)
	_ = s.audit.Log(ctx, &types.AuditLog{
		TenantID:    tenantID,
		ActorUserID: uid,
		ActorRole:   string(types.TenantRoleFromContext(ctx)),
		Action:      action,
		TargetType:  "department",
		TargetID:    targetID,
		Outcome:     types.AuditOutcomeSuccess,
		Details:     types.JSON(raw),
	})
}

func (s *departmentService) EnsureDefault(ctx context.Context, tenantID uint64) (*types.Department, error) {
	return s.repo.EnsureDefault(ctx, tenantID)
}

func (s *departmentService) List(ctx context.Context, tenantID uint64) ([]*types.Department, error) {
	depts, err := s.repo.List(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	for _, dept := range depts {
		if count, err := s.repo.CountMembers(ctx, tenantID, dept.ID); err == nil {
			dept.MemberCount = count
		}
		if count, err := s.repo.CountKnowledgeBases(ctx, tenantID, dept.ID); err == nil {
			dept.KnowledgeBaseCount = count
		}
	}
	return depts, nil
}

func (s *departmentService) Create(ctx context.Context, tenantID uint64, name, description string) (*types.Department, error) {
	if !isCompanyAdmin(ctx) {
		return nil, ErrDepartmentForbidden
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrDepartmentNameRequired
	}
	previous, err := s.repo.CountDepartments(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	dept := &types.Department{
		TenantID:    tenantID,
		Name:        name,
		Description: strings.TrimSpace(description),
	}
	if err := s.repo.Create(ctx, dept, previous); err != nil {
		return nil, err
	}
	s.emitAudit(ctx, tenantID, dept.ID, types.AuditActionDepartmentCreated, map[string]any{
		"name": dept.Name,
	})
	return dept, nil
}

func (s *departmentService) Update(ctx context.Context, tenantID uint64, id, name, description string) (*types.Department, error) {
	if !isCompanyAdmin(ctx) {
		return nil, ErrDepartmentForbidden
	}
	dept, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if dept == nil {
		return nil, ErrDepartmentNotFound
	}
	oldName := dept.Name
	dept.Name = strings.TrimSpace(name)
	dept.Description = strings.TrimSpace(description)
	if dept.Name == "" {
		return nil, ErrDepartmentNameRequired
	}
	if err := s.repo.Update(ctx, dept); err != nil {
		return nil, err
	}
	s.emitAudit(ctx, tenantID, dept.ID, types.AuditActionDepartmentUpdated, map[string]any{
		"old_name": oldName,
		"new_name": dept.Name,
	})
	return dept, nil
}

func (s *departmentService) Delete(ctx context.Context, tenantID uint64, id string) error {
	if !isCompanyAdmin(ctx) {
		return ErrDepartmentForbidden
	}
	dept, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if dept == nil {
		return ErrDepartmentNotFound
	}
	if dept.IsDefault {
		return ErrDefaultDepartmentProtected
	}
	members, err := s.repo.CountMembers(ctx, tenantID, id)
	if err != nil {
		return err
	}
	kbs, err := s.repo.CountKnowledgeBases(ctx, tenantID, id)
	if err != nil {
		return err
	}
	shares, err := s.repo.CountShareReferences(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if members > 0 || kbs > 0 || shares > 0 {
		return ErrDepartmentNonEmpty
	}
	if err := s.repo.SoftDelete(ctx, tenantID, id); err != nil {
		return err
	}
	s.emitAudit(ctx, tenantID, id, types.AuditActionDepartmentDeleted, map[string]any{
		"name": dept.Name,
	})
	return nil
}

func (s *departmentService) ListMembers(ctx context.Context, tenantID uint64, id string) ([]*types.TenantMember, error) {
	dept, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if dept == nil {
		return nil, ErrDepartmentNotFound
	}
	caller := types.CallerFromContext(ctx)
	allowed := isCompanyAdmin(ctx) ||
		(caller.DepartmentID == id && caller.DepartmentRole == types.DepartmentRoleManager)
	if !allowed {
		return nil, ErrDepartmentForbidden
	}
	return s.repo.ListMembers(ctx, tenantID, id)
}

func (s *departmentService) AssignMember(
	ctx context.Context,
	tenantID uint64,
	departmentID string,
	userID string,
	role types.DepartmentRole,
) error {
	if !role.IsValid() {
		return ErrInvalidDepartmentRole
	}
	dept, err := s.repo.Get(ctx, tenantID, departmentID)
	if err != nil {
		return err
	}
	if dept == nil {
		return ErrDepartmentNotFound
	}
	member, err := s.memberRepo.Get(ctx, userID, tenantID)
	if err != nil {
		return err
	}
	if member == nil {
		return ErrDepartmentMemberNotFound
	}

	caller := types.CallerFromContext(ctx)
	companyAdmin := isCompanyAdmin(ctx)
	isDeptManager := caller.DepartmentID == departmentID && caller.DepartmentRole == types.DepartmentRoleManager
	if !companyAdmin && !isDeptManager {
		return ErrDepartmentForbidden
	}
	if !companyAdmin {
		// Department managers cannot pull users from another department or
		// change their company-level membership; they only assign roles
		// inside their own department.
		if member.DepartmentID == nil || *member.DepartmentID != departmentID {
			return ErrDepartmentForbidden
		}
	}

	if member.DepartmentRole == types.DepartmentRoleManager && role != types.DepartmentRoleManager {
		kbs, err := s.repo.CountKnowledgeBases(ctx, tenantID, departmentID)
		if err != nil {
			return err
		}
		managers, err := s.repo.CountManagers(ctx, tenantID, departmentID)
		if err != nil {
			return err
		}
		if kbs > 0 && managers <= 1 {
			return ErrLastDepartmentManager
		}
	}

	if err := s.repo.UpdateMemberAssignment(ctx, tenantID, userID, departmentID, role); err != nil {
		return err
	}
	s.emitAudit(ctx, tenantID, departmentID, types.AuditActionDepartmentMemberAssigned, map[string]any{
		"user_id":    userID,
		"old_role":   string(member.DepartmentRole),
		"new_role":   string(role),
		"department": departmentID,
	})
	return nil
}

func (s *departmentService) RemoveMember(ctx context.Context, tenantID uint64, departmentID, userID string) error {
	dept, err := s.repo.Get(ctx, tenantID, departmentID)
	if err != nil {
		return err
	}
	if dept == nil {
		return ErrDepartmentNotFound
	}
	member, err := s.memberRepo.Get(ctx, userID, tenantID)
	if err != nil {
		return err
	}
	if member == nil || member.DepartmentID == nil || *member.DepartmentID != departmentID {
		return ErrDepartmentMemberNotFound
	}
	caller := types.CallerFromContext(ctx)
	companyAdmin := isCompanyAdmin(ctx)
	isDeptManager := caller.DepartmentID == departmentID && caller.DepartmentRole == types.DepartmentRoleManager
	if !companyAdmin && !isDeptManager {
		return ErrDepartmentForbidden
	}
	if member.DepartmentRole == types.DepartmentRoleManager {
		kbs, err := s.repo.CountKnowledgeBases(ctx, tenantID, departmentID)
		if err != nil {
			return err
		}
		managers, err := s.repo.CountManagers(ctx, tenantID, departmentID)
		if err != nil {
			return err
		}
		if kbs > 0 && managers <= 1 {
			return ErrLastDepartmentManager
		}
	}
	if err := s.repo.RemoveMemberAssignment(ctx, tenantID, userID, departmentID); err != nil {
		return err
	}
	s.emitAudit(ctx, tenantID, departmentID, types.AuditActionDepartmentMemberRemoved, map[string]any{
		"user_id":    userID,
		"old_role":   string(member.DepartmentRole),
		"department": departmentID,
	})
	return nil
}

func (s *departmentService) ListCRAuditors(ctx context.Context, tenantID uint64, departmentID string) ([]*types.DepartmentCRAuditor, error) {
	if err := s.requireDepartmentManager(ctx, tenantID, departmentID); err != nil {
		return nil, err
	}
	return s.repo.ListCRAuditors(ctx, tenantID, departmentID)
}

func (s *departmentService) AddCRAuditor(ctx context.Context, tenantID uint64, departmentID, userID string) error {
	if err := s.requireDepartmentManager(ctx, tenantID, departmentID); err != nil {
		return err
	}
	member, err := s.memberRepo.Get(ctx, userID, tenantID)
	if err != nil {
		return err
	}
	if member == nil || member.Status != types.TenantMemberStatusActive ||
		member.DepartmentID == nil || *member.DepartmentID != departmentID {
		return ErrDepartmentMemberNotFound
	}
	caller := types.CallerFromContext(ctx)
	if err := s.repo.AddCRAuditor(ctx, &types.DepartmentCRAuditor{
		TenantID:     tenantID,
		DepartmentID: departmentID,
		UserID:       userID,
		GrantedBy:    caller.UserID,
	}); err != nil {
		return err
	}
	s.emitAudit(ctx, tenantID, departmentID, types.AuditActionDepartmentCRAuditorAdded, map[string]any{
		"user_id":    userID,
		"department": departmentID,
	})
	return nil
}

func (s *departmentService) RemoveCRAuditor(ctx context.Context, tenantID uint64, departmentID, userID string) error {
	if err := s.requireDepartmentManager(ctx, tenantID, departmentID); err != nil {
		return err
	}
	if err := s.repo.RemoveCRAuditor(ctx, tenantID, departmentID, userID); err != nil {
		return err
	}
	s.emitAudit(ctx, tenantID, departmentID, types.AuditActionDepartmentCRAuditorRemoved, map[string]any{
		"user_id":    userID,
		"department": departmentID,
	})
	return nil
}

func (s *departmentService) IsCRAuditor(ctx context.Context, tenantID uint64, departmentID, userID string) (bool, error) {
	return s.repo.IsCRAuditor(ctx, tenantID, departmentID, userID)
}

func (s *departmentService) requireDepartmentManager(ctx context.Context, tenantID uint64, departmentID string) error {
	caller := types.CallerFromContext(ctx)
	if caller.TenantID != tenantID || caller.DepartmentID != departmentID ||
		caller.DepartmentRole != types.DepartmentRoleManager {
		return ErrDepartmentForbidden
	}
	return nil
}
