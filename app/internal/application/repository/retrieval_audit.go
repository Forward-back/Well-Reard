package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"gaze/internal/types"
)

type RetrievalAuditRepository interface {
	Create(ctx context.Context, entry *types.RetrievalAuditLog) error
	CreateBatch(ctx context.Context, entries []*types.RetrievalAuditLog) error
	DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type retrievalAuditRepository struct{ db *gorm.DB }

func NewRetrievalAuditRepository(db *gorm.DB) RetrievalAuditRepository {
	return &retrievalAuditRepository{db: db}
}

func (r *retrievalAuditRepository) Create(ctx context.Context, entry *types.RetrievalAuditLog) error {
	return r.db.WithContext(ctx).Create(entry).Error
}

func (r *retrievalAuditRepository) CreateBatch(ctx context.Context, entries []*types.RetrievalAuditLog) error {
	if len(entries) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&entries).Error
}

func (r *retrievalAuditRepository) DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("created_at < ?", cutoff).
		Delete(&types.RetrievalAuditLog{})
	return result.RowsAffected, result.Error
}
