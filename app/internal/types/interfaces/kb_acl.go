package interfaces

import (
	"context"

	"gaze/internal/types"
)

type KBAclRepository interface {
	ListByKB(ctx context.Context, tenantID uint64, kbID string) ([]*types.KBAclRule, error)
	Create(ctx context.Context, rule *types.KBAclRule) error
	// CreateWithAudit creates the rule, bumps the tenant ACL epoch, and writes
	// the audit row in one transaction. A nil entry skips the audit write.
	CreateWithAudit(ctx context.Context, rule *types.KBAclRule, entry *types.AuditLog) error
	Delete(ctx context.Context, tenantID uint64, kbID, ruleID string) error
	// DeleteWithAudit deletes the rule, bumps the tenant ACL epoch, and writes
	// the audit row in one transaction. A nil entry skips the audit write.
	DeleteWithAudit(ctx context.Context, tenantID uint64, kbID, ruleID string, entry *types.AuditLog) error
	CountByDepartment(ctx context.Context, tenantID uint64, departmentID string) (int64, error)
	GetEpoch(ctx context.Context, tenantID uint64) (int64, error)
	BumpEpoch(ctx context.Context, tenantID uint64) (int64, error)
}
