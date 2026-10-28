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

var (
	ErrSkillGrantExists   = errors.New("skill editor grant already exists")
	ErrSkillGrantNotFound = errors.New("skill editor grant not found")
)

type skillEditorGrantRepository struct {
	db *gorm.DB
}

func NewSkillEditorGrantRepository(db *gorm.DB) interfaces.SkillEditorGrantRepository {
	return &skillEditorGrantRepository{db: db}
}

func (r *skillEditorGrantRepository) ListBySkill(
	ctx context.Context, tenantID uint64, skillID string,
) ([]*types.SkillEditorGrant, error) {
	var grants []*types.SkillEditorGrant
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND skill_id = ?", tenantID, skillID).
		Order("created_at ASC").
		Find(&grants).Error; err != nil {
		return nil, err
	}
	return grants, nil
}

func (r *skillEditorGrantRepository) Create(ctx context.Context, grant *types.SkillEditorGrant) error {
	if grant.ID == "" {
		grant.ID = uuid.NewString()
	}
	if grant.CreatedAt.IsZero() {
		grant.CreatedAt = time.Now()
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(grant).Error; err != nil {
			msg := strings.ToLower(err.Error())
			if errors.Is(err, gorm.ErrDuplicatedKey) ||
				strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique constraint") {
				return ErrSkillGrantExists
			}
			return err
		}
		return bumpTenantACLEpoch(tx, grant.TenantID)
	})
	return err
}

func (r *skillEditorGrantRepository) Delete(
	ctx context.Context, tenantID uint64, skillID, grantID string,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("tenant_id = ? AND skill_id = ? AND id = ?", tenantID, skillID, grantID).
			Delete(&types.SkillEditorGrant{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrSkillGrantNotFound
		}
		return bumpTenantACLEpoch(tx, tenantID)
	})
}

func (r *skillEditorGrantRepository) HasGrant(
	ctx context.Context, tenantID uint64, skillID string,
	subjectType types.SkillGrantSubjectType, subjectID string,
) (bool, error) {
	if strings.TrimSpace(subjectID) == "" {
		return false, nil
	}
	var count int64
	err := r.db.WithContext(ctx).Model(&types.SkillEditorGrant{}).
		Where("tenant_id = ? AND skill_id = ? AND subject_type = ? AND subject_id = ?",
			tenantID, skillID, subjectType, subjectID).
		Count(&count).Error
	return count > 0, err
}

func (r *skillEditorGrantRepository) SkillExists(
	ctx context.Context, tenantID uint64, skillID string,
) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&types.TenantSkillCatalogEntity{}).
		Where("tenant_id = ? AND id = ?", tenantID, skillID).
		Count(&count).Error
	return count > 0, err
}
