package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"gaze/internal/application/repository"
	"gaze/internal/capabilities"
	werrors "gaze/internal/errors"
	"gaze/internal/types"
	"gaze/internal/types/interfaces"
)

type skillEditorGrantService struct {
	repo    interfaces.SkillEditorGrantRepository
	skills  repository.TenantSkillRepository
	members interfaces.TenantMemberService
	depts   interfaces.DepartmentRepository
	audit   interfaces.AuditLogService
}

func NewSkillEditorGrantService(
	repo interfaces.SkillEditorGrantRepository,
	skills repository.TenantSkillRepository,
	members interfaces.TenantMemberService,
	depts interfaces.DepartmentRepository,
	audit interfaces.AuditLogService,
) interfaces.SkillEditorGrantService {
	return &skillEditorGrantService{repo: repo, skills: skills, members: members, depts: depts, audit: audit}
}

func (s *skillEditorGrantService) requireManage(ctx context.Context) (types.Caller, error) {
	caller := types.CallerFromContext(ctx).Normalize()
	if caller.TenantID == 0 || caller.UserID == "" {
		return caller, werrors.NewForbiddenError("需要登录后操作")
	}
	if types.IsSystemAdminFromContext(ctx) {
		return caller, nil
	}
	caps := capabilities.MustDefault().CapsFor(string(caller.Role), string(caller.DepartmentRole), false)
	if !capabilities.Has(caps, capabilities.SettingsSkillsManage) {
		return caller, werrors.NewForbiddenError("缺少技能管理能力")
	}
	return caller, nil
}

func (s *skillEditorGrantService) ensureSkill(ctx context.Context, tenantID uint64, skillID string) error {
	skillID = strings.TrimSpace(skillID)
	if skillID == "" {
		return werrors.NewValidationError("缺少技能 ID")
	}
	if s.skills != nil {
		cat, err := s.skills.GetCatalog(ctx, tenantID, skillID)
		if err != nil {
			return err
		}
		if cat == nil {
			return werrors.NewNotFoundError("技能不存在")
		}
		if !canManageDepartmentSkill(ctx, cat) {
			return werrors.NewForbiddenError("只能维护本部门的技能")
		}
		return nil
	}
	exists, err := s.repo.SkillExists(ctx, tenantID, skillID)
	if err != nil {
		return err
	}
	if !exists {
		return werrors.NewNotFoundError("技能不存在")
	}
	return nil
}

func (s *skillEditorGrantService) List(
	ctx context.Context, skillID string,
) ([]*types.SkillEditorGrant, error) {
	caller, err := s.requireManage(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.ensureSkill(ctx, caller.TenantID, skillID); err != nil {
		return nil, err
	}
	return s.repo.ListBySkill(ctx, caller.TenantID, strings.TrimSpace(skillID))
}

func (s *skillEditorGrantService) Create(
	ctx context.Context, skillID string, req types.SkillEditorGrantRequest,
) (*types.SkillEditorGrant, error) {
	caller, err := s.requireManage(ctx)
	if err != nil {
		return nil, err
	}
	skillID = strings.TrimSpace(skillID)
	if err := s.ensureSkill(ctx, caller.TenantID, skillID); err != nil {
		return nil, err
	}
	subjectID := strings.TrimSpace(req.SubjectID)
	switch req.SubjectType {
	case types.SkillGrantSubjectUser:
		if subjectID == "" || s.members == nil {
			return nil, werrors.NewValidationError("授权用户不能为空")
		}
		member, err := s.members.GetMembership(ctx, subjectID, caller.TenantID)
		if err != nil {
			return nil, err
		}
		if member == nil || member.Status != types.TenantMemberStatusActive {
			return nil, werrors.NewValidationError("授权用户不属于当前空间")
		}
		if member.DepartmentID == nil || *member.DepartmentID != caller.DepartmentID {
			return nil, werrors.NewValidationError("只能授权给本部门成员")
		}
	case types.SkillGrantSubjectDepartment:
		if subjectID == "" || s.depts == nil {
			return nil, werrors.NewValidationError("授权部门不能为空")
		}
		dept, err := s.depts.Get(ctx, caller.TenantID, subjectID)
		if err != nil {
			return nil, err
		}
		if dept == nil {
			return nil, werrors.NewValidationError("授权部门不存在")
		}
		if subjectID != caller.DepartmentID {
			return nil, werrors.NewValidationError("只能授权给本部门")
		}
	default:
		return nil, werrors.NewValidationError("无效的授权主体")
	}
	grant := &types.SkillEditorGrant{
		TenantID:    caller.TenantID,
		SkillID:     skillID,
		SubjectType: req.SubjectType,
		SubjectID:   subjectID,
		CreatedBy:   caller.UserID,
	}
	if err := s.repo.Create(ctx, grant); err != nil {
		if errors.Is(err, repository.ErrSkillGrantExists) {
			return nil, werrors.NewValidationError("该授权已存在")
		}
		return nil, err
	}
	s.logAudit(ctx, caller, types.AuditActionSkillEditorGrantCreated, grant.ID, map[string]any{
		"skill_id":     skillID,
		"subject_type": string(grant.SubjectType),
		"subject_id":   grant.SubjectID,
	})
	return grant, nil
}

func (s *skillEditorGrantService) Delete(ctx context.Context, skillID, grantID string) error {
	caller, err := s.requireManage(ctx)
	if err != nil {
		return err
	}
	if err := s.ensureSkill(ctx, caller.TenantID, skillID); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, caller.TenantID, strings.TrimSpace(skillID), strings.TrimSpace(grantID)); err != nil {
		if errors.Is(err, repository.ErrSkillGrantNotFound) {
			return werrors.NewNotFoundError("授权不存在")
		}
		return err
	}
	s.logAudit(ctx, caller, types.AuditActionSkillEditorGrantDeleted, grantID, map[string]any{
		"skill_id": strings.TrimSpace(skillID),
	})
	return nil
}

// CanEdit applies the resource-level grant on top of the capability check.
// Default editors are manager/knowledge_admin/company_admin; editor and viewer
// need an explicit user or department grant for this skill.
func (s *skillEditorGrantService) CanEdit(ctx context.Context, skillID string) (bool, error) {
	caller := types.CallerFromContext(ctx).Normalize()
	if caller.TenantID == 0 || caller.UserID == "" {
		return false, werrors.NewForbiddenError("需要登录后操作")
	}
	skillID = strings.TrimSpace(skillID)
	if skillID == "" {
		return false, werrors.NewValidationError("缺少技能 ID")
	}
	if s.repo == nil {
		return false, nil
	}
	var departmentSkill *types.TenantSkillCatalogEntity
	if s.skills != nil {
		cat, err := s.skills.GetCatalog(ctx, caller.TenantID, skillID)
		if err != nil {
			return false, err
		}
		if cat == nil {
			return false, werrors.NewNotFoundError("技能不存在")
		}
		if !canViewDepartmentSkill(ctx, cat) {
			return false, nil
		}
		departmentSkill = cat
	} else {
		exists, err := s.repo.SkillExists(ctx, caller.TenantID, skillID)
		if err != nil {
			return false, err
		}
		if !exists {
			return false, werrors.NewNotFoundError("技能不存在")
		}
	}
	if types.IsSystemAdminFromContext(ctx) {
		if departmentSkill == nil || canManageDepartmentSkill(ctx, departmentSkill) {
			return true, nil
		}
		return false, nil
	}
	caps := capabilities.MustDefault().CapsFor(string(caller.Role), string(caller.DepartmentRole), false)
	if capabilities.Has(caps, capabilities.SettingsSkillsManage) {
		if departmentSkill == nil || canManageDepartmentSkill(ctx, departmentSkill) {
			return true, nil
		}
		return false, nil
	}
	userGrant, err := s.repo.HasGrant(ctx, caller.TenantID, skillID, types.SkillGrantSubjectUser, caller.UserID)
	if err != nil {
		return false, err
	}
	if userGrant {
		return true, nil
	}
	if caller.DepartmentID != "" {
		deptGrant, err := s.repo.HasGrant(
			ctx, caller.TenantID, skillID, types.SkillGrantSubjectDepartment, caller.DepartmentID,
		)
		if err != nil {
			return false, err
		}
		if deptGrant {
			return true, nil
		}
	}
	return false, nil
}

func (s *skillEditorGrantService) logAudit(
	ctx context.Context, caller types.Caller, action types.AuditAction, targetID string, details map[string]any,
) {
	if s.audit == nil {
		return
	}
	raw, _ := json.Marshal(details)
	_ = s.audit.Log(ctx, &types.AuditLog{
		TenantID:    caller.TenantID,
		ActorUserID: caller.UserID,
		Action:      action,
		TargetType:  "skill_editor_grant",
		TargetID:    targetID,
		Outcome:     types.AuditOutcomeSuccess,
		Details:     types.JSON(raw),
	})
}
