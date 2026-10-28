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
	ErrACLRuleExists   = errors.New("acl rule already exists")
	ErrACLRuleNotFound = errors.New("acl rule not found")
)

type kbAclRepository struct {
	db *gorm.DB
}

func NewKBAclRepository(db *gorm.DB) interfaces.KBAclRepository {
	return &kbAclRepository{db: db}
}

func (r *kbAclRepository) ListByKB(ctx context.Context, tenantID uint64, kbID string) ([]*types.KBAclRule, error) {
	var rules []*types.KBAclRule
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND kb_id = ?", tenantID, kbID).
		Order("created_at ASC").
		Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

func (r *kbAclRepository) Create(ctx context.Context, rule *types.KBAclRule) error {
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = time.Now()
	}
	if err := r.db.WithContext(ctx).Create(rule).Error; err != nil {
		msg := strings.ToLower(err.Error())
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique constraint") {
			return ErrACLRuleExists
		}
		return err
	}
	return nil
}

func (r *kbAclRepository) CreateWithAudit(
	ctx context.Context, rule *types.KBAclRule, entry *types.AuditLog,
) error {
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = time.Now()
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(rule).Error; err != nil {
			msg := strings.ToLower(err.Error())
			if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique constraint") {
				return ErrACLRuleExists
			}
			return err
		}
		if err := bumpTenantACLEpoch(tx, rule.TenantID); err != nil {
			return err
		}
		if entry != nil {
			entry.TargetID = rule.ID
			if err := tx.Create(entry).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *kbAclRepository) Delete(ctx context.Context, tenantID uint64, kbID, ruleID string) error {
	res := r.db.WithContext(ctx).
		Where("tenant_id = ? AND kb_id = ? AND id = ?", tenantID, kbID, ruleID).
		Delete(&types.KBAclRule{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrACLRuleNotFound
	}
	return nil
}

func (r *kbAclRepository) DeleteWithAudit(
	ctx context.Context, tenantID uint64, kbID, ruleID string, entry *types.AuditLog,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("tenant_id = ? AND kb_id = ? AND id = ?", tenantID, kbID, ruleID).
			Delete(&types.KBAclRule{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrACLRuleNotFound
		}
		if err := bumpTenantACLEpoch(tx, tenantID); err != nil {
			return err
		}
		if entry != nil {
			if err := tx.Create(entry).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *kbAclRepository) CountByDepartment(ctx context.Context, tenantID uint64, departmentID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&types.KBAclRule{}).
		Where("tenant_id = ? AND subject_type = ? AND subject_id = ?",
			tenantID, types.KBAclSubjectDepartment, departmentID).
		Count(&count).Error
	return count, err
}

func (r *kbAclRepository) GetEpoch(ctx context.Context, tenantID uint64) (int64, error) {
	var row types.TenantACLEpoch
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return row.Epoch, nil
}

func bumpTenantACLEpoch(tx *gorm.DB, tenantID uint64) error {
	return tx.Exec(
		`INSERT INTO tenant_acl_epochs (tenant_id, epoch, updated_at)
		 VALUES (?, 1, CURRENT_TIMESTAMP)
		 ON CONFLICT (tenant_id) DO UPDATE
		 SET epoch = tenant_acl_epochs.epoch + 1, updated_at = CURRENT_TIMESTAMP`,
		tenantID,
	).Error
}

func (r *kbAclRepository) BumpEpoch(ctx context.Context, tenantID uint64) (int64, error) {
	var epoch int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := bumpTenantACLEpoch(tx, tenantID); err != nil {
			return err
		}
		return tx.Model(&types.TenantACLEpoch{}).
			Where("tenant_id = ?", tenantID).
			Pluck("epoch", &epoch).Error
	})
	return epoch, err
}
