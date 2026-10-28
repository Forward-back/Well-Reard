package service

import (
	"context"
	"strings"
	"time"

	werrors "gaze/internal/errors"
	"gaze/internal/types"
)

// SetKnowledgeLifecycle flips a document between draft / effective / expired.
// Deprecation is a governed action: only the department manager may do it, it
// writes an audit record, and the new status is re-pushed to the retrieval
// engine so QA stops citing a retired document.
func (s *knowledgeService) SetKnowledgeLifecycle(
	ctx context.Context, knowledgeID string, req *types.KnowledgeLifecycleRequest,
) (*types.Knowledge, error) {
	if req == nil {
		return nil, werrors.NewBadRequestError("缺少生命周期参数")
	}
	status := strings.ToLower(strings.TrimSpace(req.Status))
	switch status {
	case "draft", "effective", "expired":
	default:
		return nil, werrors.NewValidationError("status 只能是 draft、effective 或 expired")
	}
	record, kb, err := loadKnowledgeWrite(ctx, s.repo, s.kbService, knowledgeID)
	if err != nil {
		return nil, err
	}
	if !canModerateDoc(ctx, kb) {
		return nil, werrors.NewForbiddenError("仅部门负责人可变更文档生命周期")
	}
	now := time.Now()
	updates := map[string]interface{}{
		"lifecycle_status": status,
		"updated_at":       now,
	}
	if v := strings.TrimSpace(req.EffectiveDate); v != "" {
		if len(v) > 50 {
			return nil, werrors.NewValidationError("生效日期过长")
		}
		updates["effective_date"] = v
	}
	if v := strings.TrimSpace(req.ExpiryDate); v != "" {
		if len(v) > 50 {
			return nil, werrors.NewValidationError("失效日期过长")
		}
		updates["expiry_date"] = v
	}
	if v := strings.TrimSpace(req.SupersededBy); v != "" {
		if len(v) > 100 {
			return nil, werrors.NewValidationError("替代文号过长")
		}
		updates["superseded_by"] = v
	}
	before := record.LifecycleStatus
	if err := s.repo.UpdateKnowledgeColumns(ctx, knowledgeID, updates); err != nil {
		return nil, err
	}
	record.LifecycleStatus = status
	record.UpdatedAt = now
	if v, ok := updates["effective_date"].(string); ok {
		record.EffectiveDate = v
	}
	if v, ok := updates["expiry_date"].(string); ok {
		record.ExpiryDate = v
	}
	if v, ok := updates["superseded_by"].(string); ok {
		record.SupersededBy = v
	}
	recordKBActivity(ctx, s.audit, kb.TenantID, kb.ID,
		types.AuditActionKnowledgeLifecycleChanged, "knowledge", knowledgeID,
		types.AuditOutcomeAccepted, map[string]any{
			"from":           before,
			"to":             status,
			"reason":         strings.TrimSpace(req.Reason),
			"effective_date": record.EffectiveDate,
			"expiry_date":    record.ExpiryDate,
			"superseded_by":  record.SupersededBy,
		})
	// 状态变化必须尽快反映到检索侧；失败时保留 pending 标记等重推。
	s.engineSync.ScheduleKnowledgeSync(ctx, record.TenantID, knowledgeID)
	return record, nil
}
