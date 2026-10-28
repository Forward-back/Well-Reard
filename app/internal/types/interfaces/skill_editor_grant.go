package interfaces

import (
	"context"

	"gaze/internal/types"
)

// SkillEditorGrantRepository persists per-skill editor grants.
type SkillEditorGrantRepository interface {
	ListBySkill(ctx context.Context, tenantID uint64, skillID string) ([]*types.SkillEditorGrant, error)
	Create(ctx context.Context, grant *types.SkillEditorGrant) error
	Delete(ctx context.Context, tenantID uint64, skillID, grantID string) error
	HasGrant(ctx context.Context, tenantID uint64, skillID string, subjectType types.SkillGrantSubjectType, subjectID string) (bool, error)
	SkillExists(ctx context.Context, tenantID uint64, skillID string) (bool, error)
}

// SkillEditorGrantService manages grants and answers editability checks.
type SkillEditorGrantService interface {
	List(ctx context.Context, skillID string) ([]*types.SkillEditorGrant, error)
	Create(ctx context.Context, skillID string, req types.SkillEditorGrantRequest) (*types.SkillEditorGrant, error)
	Delete(ctx context.Context, skillID, grantID string) error
	CanEdit(ctx context.Context, skillID string) (bool, error)
}
