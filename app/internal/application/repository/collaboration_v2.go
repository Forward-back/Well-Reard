package repository

import (
	"context"
	"errors"
	"time"

	"gaze/internal/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func newID() string { return uuid.NewString() }

var (
	ErrDocVersionNotFound = errors.New("document version not found")
	ErrDocAnchorNotFound  = errors.New("document anchor comment not found")
)

func (r *collaborationRepository) ReplaceDocCRReviewers(
	ctx context.Context, tenantID uint64, crID string, reviewers []*types.DocCRReviewer,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND cr_id = ?", tenantID, crID).
			Delete(&types.DocCRReviewer{}).Error; err != nil {
			return err
		}
		if len(reviewers) == 0 {
			return nil
		}
		return tx.Create(reviewers).Error
	})
}

func (r *collaborationRepository) ListDocCRReviewers(
	ctx context.Context, tenantID uint64, crID string,
) ([]*types.DocCRReviewer, error) {
	var reviewers []*types.DocCRReviewer
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND cr_id = ?", tenantID, crID).
		Order("created_at ASC").
		Find(&reviewers).Error
	return reviewers, err
}

func (r *collaborationRepository) UpdateDocCRReviewer(
	ctx context.Context, reviewer *types.DocCRReviewer,
) error {
	return r.db.WithContext(ctx).
		Model(&types.DocCRReviewer{}).
		Where("id = ? AND tenant_id = ?", reviewer.ID, reviewer.TenantID).
		Updates(map[string]interface{}{
			"status":     reviewer.Status,
			"comment":    reviewer.Comment,
			"decided_at": reviewer.DecidedAt,
			"updated_at": reviewer.UpdatedAt,
		}).Error
}

func (r *collaborationRepository) CreateDocCRReviewHistory(
	ctx context.Context, review *types.DocCRReview,
) error {
	return r.db.WithContext(ctx).Create(review).Error
}

func (r *collaborationRepository) ReplaceDocCRLinkedIssues(
	ctx context.Context, tenantID uint64, crID string, issueIDs []string,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND cr_id = ?", tenantID, crID).
			Delete(&types.DocCRIssueLink{}).Error; err != nil {
			return err
		}
		seen := map[string]struct{}{}
		links := make([]*types.DocCRIssueLink, 0, len(issueIDs))
		for _, issueID := range issueIDs {
			if issueID == "" {
				continue
			}
			if _, ok := seen[issueID]; ok {
				continue
			}
			seen[issueID] = struct{}{}
			links = append(links, &types.DocCRIssueLink{
				ID:        newID(),
				TenantID:  tenantID,
				CRID:      crID,
				IssueID:   issueID,
				CreatedAt: time.Now(),
			})
		}
		if len(links) == 0 {
			return nil
		}
		return tx.Create(&links).Error
	})
}

func (r *collaborationRepository) ListDocCRLinkedIssues(
	ctx context.Context, tenantID uint64, crID string,
) ([]*types.DocIssue, error) {
	var links []*types.DocCRIssueLink
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND cr_id = ?", tenantID, crID).
		Find(&links).Error; err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return nil, nil
	}
	ids := make([]string, 0, len(links))
	for _, link := range links {
		ids = append(ids, link.IssueID)
	}
	var issues []*types.DocIssue
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id IN ? AND deleted_at IS NULL", tenantID, ids).
		Order("seq_id ASC").
		Find(&issues).Error; err != nil {
		return nil, err
	}
	return issues, nil
}

func (r *collaborationRepository) UpdateDocCR(ctx context.Context, cr *types.DocChangeRequest) error {
	result := r.db.WithContext(ctx).
		Model(&types.DocChangeRequest{}).
		Where("tenant_id = ? AND knowledge_base_id = ? AND id = ?", cr.TenantID, cr.KnowledgeBaseID, cr.ID).
		Updates(map[string]interface{}{
			"title":             cr.Title,
			"description":       cr.Description,
			"status":            cr.Status,
			"base_version_id":   cr.BaseVersionID,
			"merged_version_id": cr.MergedVersionID,
			"merged_at":         cr.MergedAt,
			"closed_by":         cr.ClosedBy,
			"closed_at":         cr.ClosedAt,
			"tags":              cr.Tags,
			"draft_updated_at":  cr.DraftUpdatedAt,
			"anchor_orphaned":   cr.AnchorOrphaned,
			"updated_at":        cr.UpdatedAt,
			"reviewed_by":       cr.ReviewedBy,
			"reviewed_at":       cr.ReviewedAt,
			"review_comment":    cr.ReviewComment,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrDocCRNotFound
	}
	return nil
}

func (r *collaborationRepository) UpdateDocCRDraft(ctx context.Context, cr *types.DocChangeRequest) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&types.DocChangeRequest{}).
			Where("tenant_id = ? AND knowledge_base_id = ? AND id = ? AND status IN ?",
				cr.TenantID, cr.KnowledgeBaseID, cr.ID,
				[]string{types.DocCRStatusDraft, types.DocCRStatusPendingReview, types.DocCRStatusChangesRequested}).
			Updates(map[string]interface{}{
				"title":            cr.Title,
				"description":      cr.Description,
				"status":           cr.Status,
				"base_version_id":  cr.BaseVersionID,
				"tags":             cr.Tags,
				"draft_updated_at": cr.DraftUpdatedAt,
				"updated_at":       cr.UpdatedAt,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrDocCRNotOpen
		}
		if err := tx.Where("tenant_id = ? AND cr_id = ?", cr.TenantID, cr.ID).
			Delete(&types.DocCRItem{}).Error; err != nil {
			return err
		}
		for _, item := range cr.Items {
			item.CRID = cr.ID
			item.TenantID = cr.TenantID
			if err := tx.Create(item).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *collaborationRepository) ListActiveDocCRChunkLocks(
	ctx context.Context, tenantID uint64, kbID, knowledgeID string,
) ([]*types.DocCRChunkLock, error) {
	var locks []*types.DocCRChunkLock
	err := r.db.WithContext(ctx).Raw(`
SELECT ci.chunk_id AS chunk_id, r.id AS cr_id, r.seq_id AS seq_id,
       r.title AS title, r.status AS status
FROM doc_cr_items ci
JOIN doc_change_requests r ON r.id = ci.cr_id
WHERE r.tenant_id = ? AND r.knowledge_base_id = ? AND r.knowledge_id = ?
  AND r.status IN ?
ORDER BY r.seq_id DESC`, tenantID, kbID, knowledgeID,
		[]string{types.DocCRStatusPendingReview, types.DocCRStatusChangesRequested},
	).Scan(&locks).Error
	return locks, err
}

func (r *collaborationRepository) ListChunkCitationStats(
	ctx context.Context, tenantID uint64, chunkIDs []string,
) ([]*types.ChunkCitationStat, error) {
	if len(chunkIDs) == 0 || r.db.Dialector.Name() != "postgres" {
		return nil, nil
	}
	var stats []*types.ChunkCitationStat
	err := r.db.WithContext(ctx).Raw(`
SELECT ref->>'id' AS chunk_id,
       COUNT(*) AS total,
       COUNT(*) FILTER (WHERE m.created_at >= NOW() - INTERVAL '30 days') AS last_30_days
FROM messages m
JOIN sessions s ON s.id = m.session_id
CROSS JOIN LATERAL jsonb_array_elements(m.knowledge_references) ref
WHERE s.tenant_id = ? AND m.role = 'assistant'
  AND m.deleted_at IS NULL AND ref->>'id' IN ?
GROUP BY ref->>'id'`, tenantID, chunkIDs).Scan(&stats).Error
	return stats, err
}

func (r *collaborationRepository) MergeDocCR(ctx context.Context, payload *types.DocCRMergeTx) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := payload.Now
		cr := payload.CR

		for _, chunk := range payload.Chunks {
			expected := chunk.ContentRevision - 1
			res := tx.Model(&types.Chunk{}).
				Where("id = ? AND tenant_id = ? AND content_revision = ? AND deleted_at IS NULL",
					chunk.ID, chunk.TenantID, expected).
				Updates(map[string]interface{}{
					"content":          chunk.Content,
					"content_revision": chunk.ContentRevision,
					"last_editor_id":   chunk.LastEditorID,
					"updated_at":       now,
				})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return ErrDocCRRevisionStale
			}
		}
		if len(payload.Revisions) > 0 {
			if err := tx.Create(payload.Revisions).Error; err != nil {
				return err
			}
		}

		res := tx.Model(&types.DocChangeRequest{}).
			Where("tenant_id = ? AND knowledge_base_id = ? AND id = ? AND status IN ?",
				cr.TenantID, cr.KnowledgeBaseID, cr.ID,
				[]string{types.DocCRStatusPendingReview, types.DocCRStatusChangesRequested}).
			Updates(map[string]interface{}{
				"status":            types.DocCRStatusMerged,
				"merged_version_id": cr.MergedVersionID,
				"merged_at":         cr.MergedAt,
				"reviewed_by":       cr.ReviewedBy,
				"reviewed_at":       cr.ReviewedAt,
				"review_comment":    cr.ReviewComment,
				"updated_at":        now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrDocCRNotOpen
		}

		if len(payload.CloseIssueIDs) > 0 {
			if err := tx.Model(&types.DocIssue{}).
				Where("tenant_id = ? AND id IN ? AND status = ?", cr.TenantID, payload.CloseIssueIDs, types.DocIssueStatusOpen).
				Updates(map[string]interface{}{
					"status":     types.DocIssueStatusClosed,
					"closed_by":  cr.ReviewedBy,
					"closed_at":  now,
					"updated_at": now,
				}).Error; err != nil {
				return err
			}
		}

		if payload.Version != nil {
			if err := createSnapshotTx(tx, payload.Version); err != nil {
				return err
			}
		}

		if payload.MarkEngineSyncPending {
			if err := tx.Model(&types.Knowledge{}).
				Where("tenant_id = ? AND id = ?", cr.TenantID, payload.KnowledgeID).
				Updates(map[string]interface{}{
					"engine_sync_status":       types.EngineSyncStatusPending,
					"engine_sync_requested_at": now,
					"engine_sync_error":        "",
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *collaborationRepository) MarkDocCRItemViewed(
	ctx context.Context, tenantID uint64, crID string, userID string,
) error {
	var items []*types.DocCRItem
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND cr_id = ?", tenantID, crID).
		Find(&items).Error; err != nil {
		return err
	}
	for _, item := range items {
		found := false
		for _, id := range item.ViewedBy {
			if id == userID {
				found = true
				break
			}
		}
		if found {
			continue
		}
		item.ViewedBy = append(item.ViewedBy, userID)
		if err := r.db.WithContext(ctx).Model(&types.DocCRItem{}).
			Where("id = ?", item.ID).
			Update("viewed_by", item.ViewedBy).Error; err != nil {
			return err
		}
	}
	return nil
}

// ---- versions ----

func (r *collaborationRepository) CreateDocVersion(
	ctx context.Context, version *types.DocVersion, chunks []*types.DocVersionChunk,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(version).Error; err != nil {
			return err
		}
		if len(chunks) > 0 {
			return tx.CreateInBatches(chunks, 100).Error
		}
		return nil
	})
}

func (r *collaborationRepository) GetLatestDocVersion(
	ctx context.Context, tenantID uint64, knowledgeID string,
) (*types.DocVersion, error) {
	var version types.DocVersion
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND knowledge_id = ?", tenantID, knowledgeID).
		Order("version_seq DESC").
		First(&version).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDocVersionNotFound
		}
		return nil, err
	}
	return &version, nil
}

func (r *collaborationRepository) ListDocVersions(
	ctx context.Context, tenantID uint64, knowledgeID string,
) ([]*types.DocVersion, error) {
	var versions []*types.DocVersion
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND knowledge_id = ?", tenantID, knowledgeID).
		Order("version_seq DESC").
		Find(&versions).Error
	return versions, err
}

func (r *collaborationRepository) GetDocVersion(
	ctx context.Context, tenantID uint64, knowledgeID string, versionID string,
) (*types.DocVersion, []*types.DocVersionChunk, error) {
	var version types.DocVersion
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND knowledge_id = ? AND id = ?", tenantID, knowledgeID, versionID).
		First(&version).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrDocVersionNotFound
		}
		return nil, nil, err
	}
	var chunks []*types.DocVersionChunk
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND version_id = ?", tenantID, versionID).
		Order("chunk_index ASC").
		Find(&chunks).Error; err != nil {
		return nil, nil, err
	}
	return &version, chunks, nil
}

func (r *collaborationRepository) ApplyDocVersionRollback(
	ctx context.Context, payload *types.DocVersionRollbackTx,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := payload.Now
		for _, chunk := range payload.Chunks {
			updates := map[string]interface{}{
				"content":          chunk.Content,
				"is_enabled":       chunk.IsEnabled,
				"content_revision": chunk.ContentRevision,
				"last_editor_id":   chunk.LastEditorID,
				"updated_at":       now,
				"deleted_at":       nil,
			}
			if err := tx.Model(&types.Chunk{}).
				Where("id = ? AND tenant_id = ?", chunk.ID, chunk.TenantID).
				Updates(updates).Error; err != nil {
				return err
			}
		}
		if len(payload.NewChunks) > 0 {
			for _, chunk := range payload.NewChunks {
				updates := map[string]interface{}{
					"knowledge_base_id": chunk.KnowledgeBaseID,
					"knowledge_id":      chunk.KnowledgeID,
					"content":           chunk.Content,
					"source_content":    chunk.SourceContent,
					"chunk_index":       chunk.ChunkIndex,
					"is_enabled":        chunk.IsEnabled,
					"chunk_type":        chunk.ChunkType,
					"index_status":      chunk.IndexStatus,
					"content_revision":  chunk.ContentRevision,
					"last_editor_id":    chunk.LastEditorID,
					"updated_at":        now,
					"deleted_at":        nil,
				}
				res := tx.Unscoped().Model(&types.Chunk{}).
					Where("id = ? AND tenant_id = ?", chunk.ID, chunk.TenantID).
					Updates(updates)
				if res.Error != nil {
					return res.Error
				}
				if res.RowsAffected == 0 {
					if err := tx.Create(chunk).Error; err != nil {
						return err
					}
				}
			}
		}
		if len(payload.Revisions) > 0 {
			if err := tx.Create(payload.Revisions).Error; err != nil {
				return err
			}
		}
		if len(payload.DisableChunkIDs) > 0 {
			if err := tx.Model(&types.Chunk{}).
				Where("tenant_id = ? AND knowledge_id = ? AND id IN ? AND deleted_at IS NULL",
					payload.Version.TenantID, payload.KnowledgeID, payload.DisableChunkIDs).
				Update("deleted_at", now).Error; err != nil {
				return err
			}
		}
		if payload.Version != nil {
			if err := createSnapshotTx(tx, payload.Version); err != nil {
				return err
			}
		}
		if payload.MarkEngineSyncPending {
			if err := tx.Model(&types.Knowledge{}).
				Where("tenant_id = ? AND id = ?", payload.Version.TenantID, payload.KnowledgeID).
				Updates(map[string]interface{}{
					"engine_sync_status":       types.EngineSyncStatusPending,
					"engine_sync_requested_at": now,
					"engine_sync_error":        "",
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func createSnapshotTx(tx *gorm.DB, version *types.DocVersion) error {
	var chunks []*types.Chunk
	if err := tx.Where("tenant_id = ? AND knowledge_id = ? AND deleted_at IS NULL",
		version.TenantID, version.KnowledgeID).
		Order("chunk_index ASC").
		Find(&chunks).Error; err != nil {
		return err
	}
	version.ChunkCount = len(chunks)
	if err := tx.Create(version).Error; err != nil {
		return err
	}
	snaps := make([]*types.DocVersionChunk, 0, len(chunks))
	for _, chunk := range chunks {
		snaps = append(snaps, &types.DocVersionChunk{
			ID:         newID(),
			TenantID:   version.TenantID,
			VersionID:  version.ID,
			ChunkID:    chunk.ID,
			ChunkIndex: chunk.ChunkIndex,
			Content:    chunk.Content,
			IsEnabled:  chunk.IsEnabled,
		})
	}
	if len(snaps) == 0 {
		return nil
	}
	return tx.CreateInBatches(snaps, 100).Error
}

// ---- selection comments ----

func (r *collaborationRepository) CreateDocAnchorComment(
	ctx context.Context, comment *types.DocAnchorComment,
) error {
	return r.db.WithContext(ctx).Create(comment).Error
}

func (r *collaborationRepository) ListDocAnchorComments(
	ctx context.Context, tenantID uint64, knowledgeID string,
) ([]*types.DocAnchorComment, error) {
	var comments []*types.DocAnchorComment
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND knowledge_id = ? AND deleted_at IS NULL", tenantID, knowledgeID).
		Order("created_at ASC").
		Find(&comments).Error
	return comments, err
}

func (r *collaborationRepository) GetDocAnchorComment(
	ctx context.Context, tenantID uint64, id string,
) (*types.DocAnchorComment, error) {
	var comment types.DocAnchorComment
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ? AND deleted_at IS NULL", tenantID, id).
		First(&comment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDocAnchorNotFound
		}
		return nil, err
	}
	return &comment, nil
}

func (r *collaborationRepository) UpdateDocAnchorComment(
	ctx context.Context, comment *types.DocAnchorComment,
) error {
	return r.db.WithContext(ctx).Model(&types.DocAnchorComment{}).
		Where("tenant_id = ? AND id = ?", comment.TenantID, comment.ID).
		Updates(map[string]interface{}{
			"body":        comment.Body,
			"resolved_at": comment.ResolvedAt,
			"resolved_by": comment.ResolvedBy,
			"deleted_at":  comment.DeletedAt,
			"deleted_by":  comment.DeletedBy,
		}).Error
}
