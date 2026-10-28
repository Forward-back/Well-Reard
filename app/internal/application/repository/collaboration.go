package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"gaze/internal/types"
	"gaze/internal/types/interfaces"
	"gorm.io/gorm"
)

var (
	ErrDocIssueNotFound   = errors.New("document issue not found")
	ErrDocCRNotFound      = errors.New("document change request not found")
	ErrDocCommentNotFound = errors.New("document comment not found")
	ErrDocCRNotOpen       = errors.New("document change request is not open")
	ErrDocCRRevisionStale = errors.New("document change request chunk revision conflict")
)

// collaborationRepository implements CollaborationRepository.
type collaborationRepository struct {
	db *gorm.DB
}

// NewCollaborationRepository creates the collaboration repository.
func NewCollaborationRepository(db *gorm.DB) interfaces.CollaborationRepository {
	return &collaborationRepository{db: db}
}

func clampDocPage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func (r *collaborationRepository) CreateDocIssue(ctx context.Context, issue *types.DocIssue) error {
	return r.db.WithContext(ctx).Create(issue).Error
}

func (r *collaborationRepository) GetDocIssue(
	ctx context.Context, tenantID uint64, kbID string, id string,
) (*types.DocIssue, error) {
	var issue types.DocIssue
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND knowledge_base_id = ? AND id = ? AND deleted_at IS NULL", tenantID, kbID, id).
		First(&issue).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDocIssueNotFound
		}
		return nil, err
	}
	return &issue, nil
}

func (r *collaborationRepository) ListDocIssues(
	ctx context.Context, req *types.DocIssueListRequest,
) ([]*types.DocIssue, int64, error) {
	page, pageSize := clampDocPage(req.Page, req.PageSize)
	base := r.db.WithContext(ctx).Model(&types.DocIssue{}).
		Where("tenant_id = ? AND knowledge_base_id = ? AND deleted_at IS NULL", req.TenantID, req.KnowledgeBaseID)
	if req.ACLActive {
		if len(req.AllowedKnowledgeIDs) == 0 {
			base = base.Where("1 = 0")
		} else {
			base = base.Where("knowledge_id IN ?", req.AllowedKnowledgeIDs)
		}
	}
	if req.KnowledgeID != "" {
		base = base.Where("knowledge_id = ?", req.KnowledgeID)
	}
	if req.ChunkID != "" {
		base = base.Where("chunk_id = ?", req.ChunkID)
	}
	if req.Status != "" {
		base = base.Where("status = ?", req.Status)
	}
	if req.CreatedBy != "" {
		base = base.Where("created_by = ?", req.CreatedBy)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var issues []*types.DocIssue
	if err := base.
		Order("updated_at DESC").
		Order("seq_id DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&issues).Error; err != nil {
		return nil, 0, err
	}
	return issues, total, nil
}

func (r *collaborationRepository) UpdateDocIssue(ctx context.Context, issue *types.DocIssue) error {
	result := r.db.WithContext(ctx).
		Model(&types.DocIssue{}).
		Where("tenant_id = ? AND knowledge_base_id = ? AND id = ? AND deleted_at IS NULL",
			issue.TenantID, issue.KnowledgeBaseID, issue.ID).
		Updates(map[string]interface{}{
			"title":             issue.Title,
			"body":              issue.Body,
			"issue_type":        issue.IssueType,
			"anchor_text":       issue.AnchorText,
			"anchor_start":      issue.AnchorStart,
			"anchor_end":        issue.AnchorEnd,
			"anchor_line_start": issue.AnchorLineStart,
			"anchor_line_end":   issue.AnchorLineEnd,
			"status":            issue.Status,
			"closed_by":         issue.ClosedBy,
			"closed_at":         issue.ClosedAt,
			"deleted_by":        issue.DeletedBy,
			"deleted_at":        issue.DeletedAt,
			"updated_at":        issue.UpdatedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrDocIssueNotFound
	}
	return nil
}

func (r *collaborationRepository) ListOpenDocIssuesByChunkIDs(
	ctx context.Context, tenantID uint64, kbID string, chunkIDs []string,
) ([]*types.DocIssue, error) {
	if len(chunkIDs) == 0 {
		return nil, nil
	}
	var issues []*types.DocIssue
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND knowledge_base_id = ? AND status = ? AND deleted_at IS NULL AND chunk_id IN ?",
			tenantID, kbID, types.DocIssueStatusOpen, chunkIDs).
		Order("seq_id ASC").
		Find(&issues).Error
	return issues, err
}

func (r *collaborationRepository) MarkDocIssuesOrphaned(
	ctx context.Context, tenantID uint64, knowledgeID string, chunkIDs []string,
) error {
	if knowledgeID == "" {
		return nil
	}
	q := r.db.WithContext(ctx).Model(&types.DocIssue{}).
		Where("tenant_id = ? AND knowledge_id = ? AND status = ? AND deleted_at IS NULL AND anchor_orphaned = ?",
			tenantID, knowledgeID, types.DocIssueStatusOpen, false)
	if len(chunkIDs) > 0 {
		q = q.Where("chunk_id IN ?", chunkIDs)
	} else {
		q = q.Where("chunk_id <> ''")
	}
	return q.Update("anchor_orphaned", true).Error
}

func (r *collaborationRepository) CreateDocChangeRequest(
	ctx context.Context, cr *types.DocChangeRequest,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(cr).Error; err != nil {
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

func (r *collaborationRepository) GetDocChangeRequest(
	ctx context.Context, tenantID uint64, kbID string, id string,
) (*types.DocChangeRequest, error) {
	var cr types.DocChangeRequest
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND knowledge_base_id = ? AND id = ?", tenantID, kbID, id).
		First(&cr).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDocCRNotFound
		}
		return nil, err
	}
	var items []*types.DocCRItem
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND cr_id = ?", tenantID, id).
		Order("seq ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	var reviews []*types.DocCRReview
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND cr_id = ?", tenantID, id).
		Order("created_at ASC").
		Find(&reviews).Error; err != nil {
		return nil, err
	}
	cr.Items = items
	cr.Reviews = reviews
	cr.ItemCount = int64(len(items))
	return &cr, nil
}

func (r *collaborationRepository) ListDocChangeRequests(
	ctx context.Context, req *types.DocCRListRequest,
) ([]*types.DocChangeRequest, int64, error) {
	page, pageSize := clampDocPage(req.Page, req.PageSize)
	base := r.db.WithContext(ctx).Model(&types.DocChangeRequest{}).
		Where("tenant_id = ? AND knowledge_base_id = ?", req.TenantID, req.KnowledgeBaseID)
	if req.ACLActive {
		if len(req.AllowedKnowledgeIDs) == 0 {
			base = base.Where("1 = 0")
		} else {
			base = base.Where("knowledge_id IN ?", req.AllowedKnowledgeIDs)
		}
	}
	if req.KnowledgeID != "" {
		base = base.Where("knowledge_id = ?", req.KnowledgeID)
	}
	if req.Status != "" {
		base = base.Where("status = ?", req.Status)
	}
	if req.CreatedBy != "" {
		base = base.Where("created_by = ?", req.CreatedBy)
	}
	if req.VisibilityUserID != "" {
		base = base.Where("(status <> ? OR created_by = ?)", types.DocCRStatusDraft, req.VisibilityUserID)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var crs []*types.DocChangeRequest
	if err := base.
		Order("updated_at DESC").
		Order("seq_id DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&crs).Error; err != nil {
		return nil, 0, err
	}
	return crs, total, nil
}

func (r *collaborationRepository) ResubmitDocCR(ctx context.Context, cr *types.DocChangeRequest) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&types.DocChangeRequest{}).
			Where("tenant_id = ? AND knowledge_base_id = ? AND id = ? AND status = ?",
				cr.TenantID, cr.KnowledgeBaseID, cr.ID, types.DocCRStatusChangesRequested).
			Updates(map[string]interface{}{
				"title":            cr.Title,
				"description":      cr.Description,
				"status":           types.DocCRStatusPendingReview,
				"base_version_id":  cr.BaseVersionID,
				"tags":             cr.Tags,
				"draft_updated_at": nil,
				"anchor_orphaned":  cr.AnchorOrphaned,
				"updated_at":       cr.UpdatedAt,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
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

func (r *collaborationRepository) ApplyDocCRReview(ctx context.Context, payload *types.DocCRReviewTx) error {
	if payload == nil || payload.CR == nil || payload.Review == nil {
		return ErrDocCRNotFound
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := payload.Now
		if now.IsZero() {
			now = time.Now()
		}

		for _, chunk := range payload.Chunks {
			expected := chunk.ContentRevision - 1
			result := tx.Model(&types.Chunk{}).
				Where("id = ? AND tenant_id = ? AND content_revision = ?", chunk.ID, chunk.TenantID, expected).
				Updates(map[string]interface{}{
					"content":          chunk.Content,
					"content_revision": chunk.ContentRevision,
					"last_editor_id":   chunk.LastEditorID,
					"updated_at":       chunk.UpdatedAt,
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return ErrDocCRRevisionStale
			}
		}

		for _, revision := range payload.Revisions {
			if err := tx.Create(revision).Error; err != nil {
				return err
			}
		}

		crResult := tx.Model(&types.DocChangeRequest{}).
			Where("tenant_id = ? AND id = ? AND status = ?", payload.CR.TenantID, payload.CR.ID, types.DocCRStatusOpen).
			Updates(map[string]interface{}{
				"status":         payload.CR.Status,
				"reviewed_by":    payload.CR.ReviewedBy,
				"reviewed_at":    payload.CR.ReviewedAt,
				"review_comment": payload.CR.ReviewComment,
				"updated_at":     now,
			})
		if crResult.Error != nil {
			return crResult.Error
		}
		if crResult.RowsAffected != 1 {
			return ErrDocCRNotOpen
		}

		if err := tx.Create(payload.Review).Error; err != nil {
			return err
		}

		if len(payload.CloseIssueIDs) > 0 {
			if err := tx.Model(&types.DocIssue{}).
				Where("tenant_id = ? AND id IN ? AND status = ? AND deleted_at IS NULL",
					payload.CR.TenantID, payload.CloseIssueIDs, types.DocIssueStatusOpen).
				Updates(map[string]interface{}{
					"status":     types.DocIssueStatusClosed,
					"closed_by":  payload.CR.ReviewedBy,
					"closed_at":  now,
					"updated_at": now,
				}).Error; err != nil {
				return err
			}
		}

		if payload.MarkEngineSyncPending && payload.KnowledgeID != "" {
			if err := tx.Model(&types.Knowledge{}).
				Where("tenant_id = ? AND id = ?", payload.CR.TenantID, payload.KnowledgeID).
				Updates(map[string]interface{}{
					"engine_sync_status":       types.EngineSyncStatusPending,
					"engine_sync_requested_at": now,
					"engine_sync_completed_at": nil,
					"engine_sync_error":        "",
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *collaborationRepository) ListOpenDocCRsByChunkIDs(
	ctx context.Context, tenantID uint64, kbID string, chunkIDs []string,
) ([]*types.DocChangeRequest, error) {
	if len(chunkIDs) == 0 {
		return nil, nil
	}
	var crs []*types.DocChangeRequest
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND knowledge_base_id = ? AND status = ?", tenantID, kbID, types.DocCRStatusOpen).
		Where("id IN (SELECT cr_id FROM doc_cr_items WHERE tenant_id = ? AND chunk_id IN ?)", tenantID, chunkIDs).
		Order("seq_id ASC").
		Find(&crs).Error
	return crs, err
}

func (r *collaborationRepository) MarkDocCRsOrphaned(
	ctx context.Context, tenantID uint64, knowledgeID string, chunkIDs []string,
) error {
	if knowledgeID == "" {
		return nil
	}
	q := r.db.WithContext(ctx).Model(&types.DocChangeRequest{}).
		Where("tenant_id = ? AND knowledge_id = ? AND status = ? AND anchor_orphaned = ?",
			tenantID, knowledgeID, types.DocCRStatusOpen, false)
	if len(chunkIDs) > 0 {
		q = q.Where("id IN (SELECT cr_id FROM doc_cr_items WHERE tenant_id = ? AND chunk_id IN ?)",
			tenantID, chunkIDs)
	}
	return q.Update("anchor_orphaned", true).Error
}

func (r *collaborationRepository) CreateDocComment(ctx context.Context, comment *types.DocComment) error {
	return r.db.WithContext(ctx).Create(comment).Error
}

func (r *collaborationRepository) GetDocComment(
	ctx context.Context, tenantID uint64, kbID string, id string,
) (*types.DocComment, error) {
	var comment types.DocComment
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND knowledge_base_id = ? AND id = ?", tenantID, kbID, id).
		First(&comment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDocCommentNotFound
		}
		return nil, err
	}
	return &comment, nil
}

func (r *collaborationRepository) ListDocComments(
	ctx context.Context, tenantID uint64, kbID string, targetType string, targetID string,
) ([]*types.DocComment, error) {
	var comments []*types.DocComment
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND knowledge_base_id = ? AND target_type = ? AND target_id = ?",
			tenantID, kbID, targetType, targetID).
		Order("created_at ASC").
		Find(&comments).Error
	return comments, err
}

func (r *collaborationRepository) UpdateDocComment(ctx context.Context, comment *types.DocComment) error {
	result := r.db.WithContext(ctx).
		Model(&types.DocComment{}).
		Where("tenant_id = ? AND knowledge_base_id = ? AND id = ?", comment.TenantID, comment.KnowledgeBaseID, comment.ID).
		Updates(map[string]interface{}{
			"body":       comment.Body,
			"deleted_by": comment.DeletedBy,
			"deleted_at": comment.DeletedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrDocCommentNotFound
	}
	return nil
}

func (r *collaborationRepository) CountDocCommentsByTargets(
	ctx context.Context, tenantID uint64, kbID string, targetType string, targetIDs []string,
) (map[string]int64, error) {
	out := make(map[string]int64, len(targetIDs))
	if len(targetIDs) == 0 {
		return out, nil
	}
	type row struct {
		TargetID string
		Cnt      int64
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Model(&types.DocComment{}).
		Select("target_id, COUNT(*) AS cnt").
		Where("tenant_id = ? AND knowledge_base_id = ? AND target_type = ? AND target_id IN ? AND deleted_at IS NULL",
			tenantID, kbID, targetType, targetIDs).
		Group("target_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, item := range rows {
		out[item.TargetID] = item.Cnt
	}
	return out, nil
}

// collaborationRow is the scan target for the unified aggregate query. It
// mirrors CollaborationItem but without gorm:"-" runtime-only tags so the
// aliased count columns actually land.
type collaborationRow struct {
	Type            string     `gorm:"column:type"`
	ID              string     `gorm:"column:id"`
	SeqID           int64      `gorm:"column:seq_id"`
	Title           string     `gorm:"column:title"`
	Status          string     `gorm:"column:status"`
	KnowledgeID     string     `gorm:"column:knowledge_id"`
	KnowledgeBaseID string     `gorm:"column:knowledge_base_id"`
	KnowledgeTitle  string     `gorm:"column:knowledge_title"`
	ChunkID         string     `gorm:"column:chunk_id"`
	CreatedBy       string     `gorm:"column:created_by"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
	AnchorOrphaned  bool       `gorm:"column:anchor_orphaned"`
	CommentsCount   int64      `gorm:"column:comments_count"`
	ItemCount       int64      `gorm:"column:item_count"`
	ReviewedAt      *time.Time `gorm:"column:reviewed_at"`
}

func (r *collaborationRepository) ListCollaborations(
	ctx context.Context, req *types.DocCollaborationListRequest,
) ([]*types.CollaborationItem, int64, error) {
	page, pageSize := clampDocPage(req.Page, req.PageSize)

	buildIssueBranch := req.Type == "" || req.Type == types.DocCollaborationTypeIssue
	buildCRBranch := req.Type == "" || req.Type == types.DocCollaborationTypeCR
	if !buildIssueBranch && !buildCRBranch {
		return nil, 0, nil
	}

	common := func(alias string) (string, []interface{}) {
		where := []string{alias + "tenant_id = ?", alias + "knowledge_base_id = ?"}
		args := []interface{}{req.TenantID, req.KnowledgeBaseID}
		if req.KnowledgeID != "" {
			where = append(where, alias+"knowledge_id = ?")
			args = append(args, req.KnowledgeID)
		}
		if req.Status != "" {
			where = append(where, alias+"status = ?")
			args = append(args, req.Status)
		}
		if req.CreatedBy != "" {
			where = append(where, alias+"created_by = ?")
			args = append(args, req.CreatedBy)
		}
		if req.ACLActive {
			if len(req.AllowedKnowledgeIDs) == 0 {
				where = append(where, "1 = 0")
			} else {
				placeholders := strings.TrimRight(strings.Repeat("?,", len(req.AllowedKnowledgeIDs)), ",")
				where = append(where, alias+"knowledge_id IN ("+placeholders+")")
				for _, id := range req.AllowedKnowledgeIDs {
					args = append(args, id)
				}
			}
		}
		return strings.Join(where, " AND "), args
	}

	branches := make([]string, 0, 2)
	args := make([]interface{}, 0, 16)
	if buildIssueBranch {
		where, branchArgs := common("i.")
		where += " AND i.deleted_at IS NULL"
		branches = append(branches, `
SELECT 'issue' AS type, i.id AS id, i.seq_id AS seq_id, i.title AS title, i.status AS status,
       i.knowledge_id AS knowledge_id, i.knowledge_base_id AS knowledge_base_id,
       COALESCE(k.title, '') AS knowledge_title, COALESCE(i.chunk_id, '') AS chunk_id,
       i.created_by AS created_by, i.created_at AS created_at, i.updated_at AS updated_at,
       i.anchor_orphaned AS anchor_orphaned,
       (SELECT COUNT(*) FROM doc_comments c
         WHERE c.tenant_id = i.tenant_id AND c.target_type = 'issue'
           AND c.target_id = i.id AND c.deleted_at IS NULL) AS comments_count,
       0 AS item_count, NULL AS reviewed_at
FROM doc_issues i
LEFT JOIN knowledges k ON k.id = i.knowledge_id
WHERE `+where)
		args = append(args, branchArgs...)
	}
	if buildCRBranch {
		where, branchArgs := common("r.")
		if req.VisibilityUserID != "" {
			where += " AND (r.status <> 'draft' OR r.created_by = ?)"
			branchArgs = append(branchArgs, req.VisibilityUserID)
		}
		branches = append(branches, `
SELECT 'cr' AS type, r.id AS id, r.seq_id AS seq_id, r.title AS title, r.status AS status,
       r.knowledge_id AS knowledge_id, r.knowledge_base_id AS knowledge_base_id,
       COALESCE(k.title, '') AS knowledge_title, '' AS chunk_id,
       r.created_by AS created_by, r.created_at AS created_at, r.updated_at AS updated_at,
       r.anchor_orphaned AS anchor_orphaned,
       (SELECT COUNT(*) FROM doc_comments c
         WHERE c.tenant_id = r.tenant_id AND c.target_type = 'cr'
           AND c.target_id = r.id AND c.deleted_at IS NULL) AS comments_count,
       (SELECT COUNT(*) FROM doc_cr_items ci WHERE ci.cr_id = r.id) AS item_count,
       r.reviewed_at AS reviewed_at
FROM doc_change_requests r
LEFT JOIN knowledges k ON k.id = r.knowledge_id
WHERE `+where)
		args = append(args, branchArgs...)
	}

	union := strings.Join(branches, " UNION ALL ")

	var total int64
	if err := r.db.WithContext(ctx).Raw("SELECT COUNT(*) FROM ("+union+") t", args...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, nil
	}

	var rows []collaborationRow
	query := "SELECT * FROM (" + union + ") t ORDER BY updated_at DESC, seq_id DESC LIMIT ? OFFSET ?"
	queryArgs := append(append([]interface{}{}, args...), pageSize, (page-1)*pageSize)
	if err := r.db.WithContext(ctx).Raw(query, queryArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	items := make([]*types.CollaborationItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, &types.CollaborationItem{
			Type:            row.Type,
			ID:              row.ID,
			SeqID:           row.SeqID,
			Title:           row.Title,
			Status:          row.Status,
			KnowledgeID:     row.KnowledgeID,
			KnowledgeBaseID: row.KnowledgeBaseID,
			KnowledgeTitle:  row.KnowledgeTitle,
			ChunkID:         row.ChunkID,
			CreatedBy:       row.CreatedBy,
			CreatedAt:       row.CreatedAt,
			UpdatedAt:       row.UpdatedAt,
			AnchorOrphaned:  row.AnchorOrphaned,
			CommentsCount:   row.CommentsCount,
			ItemCount:       row.ItemCount,
			ReviewedAt:      row.ReviewedAt,
		})
	}
	return items, total, nil
}
