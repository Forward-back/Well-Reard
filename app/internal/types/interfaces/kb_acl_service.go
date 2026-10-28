package interfaces

import (
	"context"

	"gaze/internal/types"
)

type KBAclService interface {
	List(ctx context.Context, kbID string) ([]*types.KBAclRule, int64, error)
	Create(ctx context.Context, kbID string, req types.KBAclRuleRequest) (*types.KBAclRule, error)
	Delete(ctx context.Context, kbID, ruleID string) error
	Epoch(ctx context.Context, kbID string) (int64, error)
	Evaluate(ctx context.Context, kbID, resourceType, resourceID, folderID string) (bool, string, error)
	Scope(ctx context.Context, kbID string) (*types.KBAclScope, error)
	// RetrievalScope builds the engine-facing scope for one or more KBs.
	// No ACL rules anywhere -> mode=all; otherwise mode=allow with the union
	// of readable document IDs (including every document of rule-free KBs).
	RetrievalScope(ctx context.Context, kbIDs []string) (*types.ACLScope, error)
}
