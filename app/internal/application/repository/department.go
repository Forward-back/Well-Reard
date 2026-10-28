package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"gaze/internal/types"
	"gaze/internal/types/interfaces"
)

type departmentRepository struct {
	db *gorm.DB
}

// NewDepartmentRepository wires the department store.
func NewDepartmentRepository(db *gorm.DB) interfaces.DepartmentRepository {
	return &departmentRepository{db: db}
}

func normalizeDepartmentName(name string) string {
	return strings.TrimSpace(name)
}

func (r *departmentRepository) Create(ctx context.Context, dept *types.Department, previousDepartmentCount int64) error {
	if dept.ID == "" {
		dept.ID = uuid.New().String()
	}
	dept.Name = normalizeDepartmentName(dept.Name)
	if dept.Name == "" {
		return errors.New("department name is required")
	}
	if dept.CreatedAt.IsZero() {
		dept.CreatedAt = time.Now()
	}
	dept.UpdatedAt = time.Now()

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(dept).Error; err != nil {
			return err
		}
		// A KB whose materialized share set covered every department before
		// this insert is company-wide; the new department must inherit it.
		//
		// In a one-department company every KB trivially has share count 1,
		// which is not evidence of an explicit “all departments” choice.
		// Require at least two existing departments so adding the first new
		// department cannot silently publish every historical KB to it.
		if previousDepartmentCount <= 1 {
			return nil
		}
		var kbIDs []string
		if err := tx.Model(&types.KnowledgeBase{}).
			Where("tenant_id = ? AND department_id IS NOT NULL AND deleted_at IS NULL", dept.TenantID).
			Where("(SELECT COUNT(*) FROM kb_department_shares s WHERE s.kb_id = knowledge_bases.id) = ?", previousDepartmentCount).
			Pluck("id", &kbIDs).Error; err != nil {
			return err
		}
		for _, kbID := range kbIDs {
			share := &types.KBDepartmentShare{
				ID:              uuid.New().String(),
				TenantID:        dept.TenantID,
				KnowledgeBaseID: kbID,
				DepartmentID:    dept.ID,
				CreatedBy:       "",
				CreatedAt:       time.Now(),
			}
			if err := tx.Create(share).Error; err != nil {
				if isDuplicateDepartmentShare(err) {
					continue
				}
				return err
			}
		}
		return bumpTenantACLEpoch(tx, dept.TenantID)
	})
}

func isDuplicateDepartmentShare(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique constraint")
}

func (r *departmentRepository) Get(ctx context.Context, tenantID uint64, id string) (*types.Department, error) {
	var dept types.Department
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		First(&dept).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &dept, nil
}

func (r *departmentRepository) List(ctx context.Context, tenantID uint64) ([]*types.Department, error) {
	var depts []*types.Department
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("is_default DESC, name ASC").
		Find(&depts).Error; err != nil {
		return nil, err
	}
	return depts, nil
}

func (r *departmentRepository) Update(ctx context.Context, dept *types.Department) error {
	dept.Name = normalizeDepartmentName(dept.Name)
	if dept.Name == "" {
		return errors.New("department name is required")
	}
	dept.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).
		Model(&types.Department{}).
		Where("tenant_id = ? AND id = ?", dept.TenantID, dept.ID).
		Updates(map[string]any{
			"name":        dept.Name,
			"description": dept.Description,
			"updated_at":  dept.UpdatedAt,
		}).Error
}

func (r *departmentRepository) SoftDelete(ctx context.Context, tenantID uint64, id string) error {
	res := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		Delete(&types.Department{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *departmentRepository) EnsureDefault(ctx context.Context, tenantID uint64) (*types.Department, error) {
	id := types.DefaultDepartmentID(tenantID)
	if existing, err := r.Get(ctx, tenantID, id); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}
	dept := &types.Department{
		ID:          id,
		TenantID:    tenantID,
		Name:        "默认部门",
		Description: "系统自动创建的默认部门",
		IsDefault:   true,
	}
	if err := r.Create(ctx, dept, 0); err != nil {
		// Concurrent lazy creation: return the winner.
		if existing, getErr := r.Get(ctx, tenantID, id); getErr == nil && existing != nil {
			return existing, nil
		}
		return nil, err
	}
	return dept, nil
}

func (r *departmentRepository) CountDepartments(ctx context.Context, tenantID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&types.Department{}).
		Where("tenant_id = ?", tenantID).Count(&count).Error
	return count, err
}

func (r *departmentRepository) CountMembers(ctx context.Context, tenantID uint64, departmentID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&types.TenantMember{}).
		Where("tenant_id = ? AND department_id = ? AND status = ?", tenantID, departmentID, types.TenantMemberStatusActive).
		Count(&count).Error
	return count, err
}

func (r *departmentRepository) CountKnowledgeBases(ctx context.Context, tenantID uint64, departmentID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&types.KnowledgeBase{}).
		Where("tenant_id = ? AND department_id = ? AND is_temporary = ?", tenantID, departmentID, false).
		Count(&count).Error
	return count, err
}

func (r *departmentRepository) CountShareReferences(ctx context.Context, tenantID uint64, departmentID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&types.KBDepartmentShare{}).
		Where("tenant_id = ? AND department_id = ?", tenantID, departmentID).
		Count(&count).Error
	return count, err
}

func (r *departmentRepository) CountManagers(ctx context.Context, tenantID uint64, departmentID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&types.TenantMember{}).
		Where("tenant_id = ? AND department_id = ? AND department_role = ? AND status = ?",
			tenantID, departmentID, types.DepartmentRoleManager, types.TenantMemberStatusActive).
		Count(&count).Error
	return count, err
}

func (r *departmentRepository) ListMembers(ctx context.Context, tenantID uint64, departmentID string) ([]*types.TenantMember, error) {
	var members []*types.TenantMember
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND department_id = ? AND status = ?", tenantID, departmentID, types.TenantMemberStatusActive).
		Order("joined_at ASC, id ASC").
		Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}

func (r *departmentRepository) RemoveMemberAssignment(
	ctx context.Context,
	tenantID uint64,
	userID string,
	departmentID string,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.
			Model(&types.TenantMember{}).
			Where("tenant_id = ? AND user_id = ? AND department_id = ?", tenantID, userID, departmentID).
			Updates(map[string]any{
				"department_id":   nil,
				"department_role": nil,
				"updated_at":      time.Now(),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (r *departmentRepository) UpdateMemberAssignment(
	ctx context.Context,
	tenantID uint64,
	userID string,
	departmentID string,
	role types.DepartmentRole,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.
			Model(&types.TenantMember{}).
			Where("tenant_id = ? AND user_id = ?", tenantID, userID).
			Updates(map[string]any{
				"department_id":   departmentID,
				"department_role": role,
				"updated_at":      time.Now(),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return bumpTenantACLEpoch(tx, tenantID)
	})
}

func (r *departmentRepository) ListCRAuditors(ctx context.Context, tenantID uint64, departmentID string) ([]*types.DepartmentCRAuditor, error) {
	var rows []*types.DepartmentCRAuditor
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND department_id = ?", tenantID, departmentID).
		Order("created_at ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *departmentRepository) IsCRAuditor(ctx context.Context, tenantID uint64, departmentID, userID string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&types.DepartmentCRAuditor{}).
		Where("tenant_id = ? AND department_id = ? AND user_id = ?", tenantID, departmentID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *departmentRepository) AddCRAuditor(ctx context.Context, row *types.DepartmentCRAuditor) error {
	return r.db.WithContext(ctx).Where(
		"tenant_id = ? AND department_id = ? AND user_id = ?",
		row.TenantID, row.DepartmentID, row.UserID,
	).FirstOrCreate(row).Error
}

func (r *departmentRepository) RemoveCRAuditor(ctx context.Context, tenantID uint64, departmentID, userID string) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND department_id = ? AND user_id = ?", tenantID, departmentID, userID).
		Delete(&types.DepartmentCRAuditor{}).Error
}
