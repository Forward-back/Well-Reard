package interfaces

import (
	"context"

	"gaze/internal/types"
)

// KnowledgeFolderService exposes the real knowledge-folder tree.
type KnowledgeFolderService interface {
	List(ctx context.Context, kbID string) (*types.KnowledgeFolderTree, error)
	Create(ctx context.Context, kbID, parentID, name string) (*types.KnowledgeFolder, error)
	Rename(ctx context.Context, kbID, folderID, name string) (*types.KnowledgeFolder, error)
	Move(ctx context.Context, kbID, folderID, newParentID string) (*types.KnowledgeFolder, error)
	Delete(ctx context.Context, kbID, folderID string) error
	EnsurePath(ctx context.Context, kbID, path, createdBy string) (*types.KnowledgeFolder, error)
	Get(ctx context.Context, kbID, folderID string) (*types.KnowledgeFolder, error)
	SubtreeIDs(ctx context.Context, kbID, folderID string, recursive bool) ([]string, error)
}
