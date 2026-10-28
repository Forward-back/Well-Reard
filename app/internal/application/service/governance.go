package service

import (
	"context"
	"strings"

	"gaze/internal/capabilities"
	werrors "gaze/internal/errors"
	"gaze/internal/types"
	"gaze/internal/types/interfaces"
)

type governanceService struct {
	repo     interfaces.GovernanceRepository
	deptRepo interfaces.DepartmentRepository
}

func NewGovernanceService(
	repo interfaces.GovernanceRepository,
	deptRepo interfaces.DepartmentRepository,
) interfaces.GovernanceService {
	return &governanceService{repo: repo, deptRepo: deptRepo}
}

func governanceCaller(ctx context.Context) (types.Caller, []string, error) {
	caller := types.CallerFromContext(ctx).Normalize()
	if caller.TenantID == 0 || caller.UserID == "" {
		return caller, nil, werrors.NewForbiddenError("需要登录后操作")
	}
	caps := capabilities.MustDefault().CapsFor(
		string(caller.Role), string(caller.DepartmentRole), false,
	)
	return caller, caps, nil
}

func (s *governanceService) DeptStats(
	ctx context.Context, requestedDepartmentID string,
) (*types.GovernanceStats, error) {
	caller, caps, err := governanceCaller(ctx)
	if err != nil {
		return nil, err
	}
	if !capabilities.Has(caps, capabilities.StatsDeptRead) {
		return nil, werrors.NewForbiddenError("缺少部门统计能力")
	}
	if caller.DepartmentID == "" {
		return nil, werrors.NewForbiddenError("当前账号未归属部门")
	}
	requested := strings.TrimSpace(requestedDepartmentID)
	if requested != "" && requested != caller.DepartmentID {
		return nil, werrors.NewForbiddenError("不能查看其他部门统计")
	}
	return s.repo.Stats(ctx, caller.TenantID, caller.DepartmentID, false)
}

func (s *governanceService) CompanyStats(
	ctx context.Context, departmentID string,
) (*types.GovernanceStats, error) {
	caller, caps, err := governanceCaller(ctx)
	if err != nil {
		return nil, err
	}
	if !capabilities.Has(caps, capabilities.StatsCompanyRead) {
		return nil, werrors.NewForbiddenError("缺少公司统计能力")
	}
	departmentID = strings.TrimSpace(departmentID)
	if departmentID != "" {
		dept, err := s.deptRepo.Get(ctx, caller.TenantID, departmentID)
		if err != nil {
			return nil, err
		}
		if dept == nil {
			return nil, werrors.NewNotFoundError("部门不存在")
		}
	}
	return s.repo.Stats(ctx, caller.TenantID, departmentID, true)
}

func (s *governanceService) Audit(
	ctx context.Context, departmentID string, afterID uint64, limit int,
) ([]*types.GovernanceAuditEntry, error) {
	caller, caps, err := governanceCaller(ctx)
	if err != nil {
		return nil, err
	}
	requested := strings.TrimSpace(departmentID)
	if capabilities.Has(caps, capabilities.AuditCompanyRead) {
		if requested != "" {
			dept, err := s.deptRepo.Get(ctx, caller.TenantID, requested)
			if err != nil {
				return nil, err
			}
			if dept == nil {
				return nil, werrors.NewNotFoundError("部门不存在")
			}
		}
		return s.repo.Audit(ctx, caller.TenantID, requested, afterID, limit)
	}
	if capabilities.Has(caps, capabilities.AuditDeptRead) {
		if caller.DepartmentID == "" {
			return nil, werrors.NewForbiddenError("当前账号未归属部门")
		}
		if requested != "" && requested != caller.DepartmentID {
			return nil, werrors.NewForbiddenError("不能查看其他部门审计")
		}
		return s.repo.Audit(ctx, caller.TenantID, caller.DepartmentID, afterID, limit)
	}
	return nil, werrors.NewForbiddenError("缺少审计能力")
}
