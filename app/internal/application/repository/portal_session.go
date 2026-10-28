package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"gaze/internal/types"
)

// PortalSessionRepository persists consumer-portal conversations (§2.13 P3).
type PortalSessionRepository struct {
	db *gorm.DB
}

// NewPortalSessionRepository creates a new portal session repository.
func NewPortalSessionRepository(db *gorm.DB) *PortalSessionRepository {
	return &PortalSessionRepository{db: db}
}

// CreateSession inserts a new portal session.
func (r *PortalSessionRepository) CreateSession(ctx context.Context, s *types.PortalSession) error {
	return r.db.WithContext(ctx).Create(s).Error
}

// GetSessionByID loads one portal session.
func (r *PortalSessionRepository) GetSessionByID(ctx context.Context, id string) (*types.PortalSession, error) {
	var s types.PortalSession
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// ListSessionsByIdentity lists sessions for one de-identified consumer key,
// newest first (portal history view).
func (r *PortalSessionRepository) ListSessionsByIdentity(
	ctx context.Context, identityKey string, limit int,
) ([]*types.PortalSession, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var out []*types.PortalSession
	if err := r.db.WithContext(ctx).
		Where("identity_key = ?", identityKey).
		Order("created_at DESC").
		Limit(limit).
		Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// AppendMessage inserts one portal message and bumps the session counters.
func (r *PortalSessionRepository) AppendMessage(ctx context.Context, m *types.PortalMessage) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(m).Error; err != nil {
			return err
		}
		now := time.Now()
		return tx.Model(&types.PortalSession{}).
			Where("id = ?", m.SessionID).
			Updates(map[string]interface{}{
				"message_count":   gorm.Expr("message_count + 1"),
				"last_message_at": now,
				"updated_at":      now,
			}).Error
	})
}

// ListMessages returns all messages of a session in order.
func (r *PortalSessionRepository) ListMessages(ctx context.Context, sessionID string) ([]*types.PortalMessage, error) {
	var out []*types.PortalMessage
	if err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("id ASC").
		Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// ListSessionsForOps lists recent portal sessions for the operations page
// (Admin+ only). limit caps the result; no identity expansion happens here —
// identity_key stays de-identified.
func (r *PortalSessionRepository) ListSessionsForOps(
	ctx context.Context, offset, limit int,
) ([]*types.PortalSession, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var out []*types.PortalSession
	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// AnonymizeSessionsByIdentity detaches all sessions of one consumer key
// from their identity (§2.13 P3-7 logout decoupling): history is kept for
// operations but no longer linkable to the account.
func (r *PortalSessionRepository) AnonymizeSessionsByIdentity(
	ctx context.Context, identityKey string,
) (int64, error) {
	res := r.db.WithContext(ctx).
		Model(&types.PortalSession{}).
		Where("identity_key = ?", identityKey).
		Update("identity_key", "")
	return res.RowsAffected, res.Error
}

// DeleteSessionsOlderThan hard-deletes portal sessions (and their messages
// via FK cascade) whose last activity predates cutoff (§2.13 P3-7 retention).
func (r *PortalSessionRepository) DeleteSessionsOlderThan(
	ctx context.Context, cutoff time.Time,
) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("COALESCE(last_message_at, created_at) < ?", cutoff).
		Delete(&types.PortalSession{})
	return res.RowsAffected, res.Error
}
