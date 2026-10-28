package interfaces

import (
	"context"

	"gaze/internal/types"
)

// KnowledgeFolderRepository persists the real knowledge-folder tree. Paths are
// derived from parent/name; callers never supply an arbitrary path on writes.
type KnowledgeFolderRepository interface {
	EnsureRoot(ctx context.Context, tenantID uint64, kbID, createdBy string) (*types.KnowledgeFolder, error)
	GetByID(ctx context.Context, tenantID uint64, kbID, id string) (*types.KnowledgeFolder, error)
	GetByPath(ctx context.Context, tenantID uint64, kbID, path string) (*types.KnowledgeFolder, error)
	ListByKB(ctx context.Context, tenantID uint64, kbID string) ([]*types.KnowledgeFolder, error)
	Create(ctx context.Context, folder *types.KnowledgeFolder) error
	RenameSubtree(ctx context.Context, tenantID uint64, kbID, folderID, name string) (*types.KnowledgeFolder, error)
	MoveSubtree(ctx context.Context, tenantID uint64, kbID, folderID, newParentID string) (*types.KnowledgeFolder, error)
	Delete(ctx context.Context, tenantID uint64, kbID, folderID string) error
	CountDocumentsByFolder(ctx context.Context, tenantID uint64, kbID string) (map[string]int64, error)
	UpdateKnowledgeFolderIDs(ctx context.Context, tenantID uint64, kbID string, knowledgeIDs []string, folder *types.KnowledgeFolder) (int64, error)
	EnsurePath(ctx context.Context, tenantID uint64, kbID, path, createdBy string) (*types.KnowledgeFolder, error)
}
