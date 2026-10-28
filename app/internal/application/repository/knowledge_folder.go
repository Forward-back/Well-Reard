package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"gaze/internal/types"
	"gaze/internal/types/interfaces"
)

var (
	ErrFolderNotFound      = errors.New("knowledge folder not found")
	ErrFolderNotEmpty      = errors.New("knowledge folder is not empty")
	ErrFolderInvalidMove   = errors.New("invalid knowledge folder move")
	ErrFolderAlreadyExists = errors.New("knowledge folder already exists")
)

func isDuplicateFolderError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique constraint")
}

type knowledgeFolderRepository struct {
	db *gorm.DB
}

// NewKnowledgeFolderRepository wires the real folder store.
func NewKnowledgeFolderRepository(db *gorm.DB) interfaces.KnowledgeFolderRepository {
	return &knowledgeFolderRepository{db: db}
}

func (r *knowledgeFolderRepository) EnsureRoot(
	ctx context.Context, tenantID uint64, kbID, createdBy string,
) (*types.KnowledgeFolder, error) {
	root, err := r.GetByPath(ctx, tenantID, kbID, "")
	if err != nil || root != nil {
		return root, err
	}
	root = &types.KnowledgeFolder{
		ID:              "folder-root-" + kbID,
		TenantID:        tenantID,
		KnowledgeBaseID: kbID,
		ParentID:        "",
		Name:            "",
		Path:            "",
		Depth:           0,
		IsRoot:          true,
		CreatedBy:       createdBy,
	}
	if err := r.Create(ctx, root); err != nil {
		// Concurrent lazy creation: return the winner.
		if existing, getErr := r.GetByPath(ctx, tenantID, kbID, ""); getErr == nil && existing != nil {
			return existing, nil
		}
		return nil, err
	}
	return root, nil
}

func (r *knowledgeFolderRepository) GetByID(
	ctx context.Context, tenantID uint64, kbID, id string,
) (*types.KnowledgeFolder, error) {
	var folder types.KnowledgeFolder
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND kb_id = ? AND id = ?", tenantID, kbID, id).
		First(&folder).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &folder, nil
}

func (r *knowledgeFolderRepository) GetByPath(
	ctx context.Context, tenantID uint64, kbID, path string,
) (*types.KnowledgeFolder, error) {
	var folder types.KnowledgeFolder
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND kb_id = ? AND path = ?", tenantID, kbID, path).
		First(&folder).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &folder, nil
}

func (r *knowledgeFolderRepository) ListByKB(
	ctx context.Context, tenantID uint64, kbID string,
) ([]*types.KnowledgeFolder, error) {
	var folders []*types.KnowledgeFolder
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND kb_id = ?", tenantID, kbID).
		Order("depth ASC, path ASC").
		Find(&folders).Error; err != nil {
		return nil, err
	}
	return folders, nil
}

func (r *knowledgeFolderRepository) Create(ctx context.Context, folder *types.KnowledgeFolder) error {
	if folder == nil {
		return ErrFolderNotFound
	}
	folder.Name = strings.TrimSpace(folder.Name)
	if folder.ID == "" {
		folder.ID = uuid.New().String()
	}
	now := time.Now()
	if folder.CreatedAt.IsZero() {
		folder.CreatedAt = now
	}
	folder.UpdatedAt = now
	if folder.IsRoot {
		folder.ParentID = ""
		folder.Path = ""
		folder.Depth = 0
		folder.Name = ""
		return r.db.WithContext(ctx).Create(folder).Error
	}
	if folder.Name == "" {
		return ErrFolderNotFound
	}
	parent, err := r.GetByID(ctx, folder.TenantID, folder.KnowledgeBaseID, folder.ParentID)
	if err != nil {
		return err
	}
	if parent == nil {
		return ErrFolderNotFound
	}
	folder.Path = parent.Path
	if folder.Path != "" {
		folder.Path += "/"
	}
	folder.Path += folder.Name
	folder.Depth = parent.Depth + 1
	if err := r.db.WithContext(ctx).Create(folder).Error; err != nil {
		if isDuplicateFolderError(err) {
			return ErrFolderAlreadyExists
		}
		return err
	}
	return nil
}

// subtreePrefixes computes the old and new path of a folder and its
// descendants' replacement segment.
func (r *knowledgeFolderRepository) subtree(
	ctx context.Context, tx *gorm.DB, tenantID uint64, kbID, folderID string,
) (*types.KnowledgeFolder, error) {
	var folder types.KnowledgeFolder
	err := tx.WithContext(ctx).
		Where("tenant_id = ? AND kb_id = ? AND id = ?", tenantID, kbID, folderID).
		First(&folder).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrFolderNotFound
	}
	if err != nil {
		return nil, err
	}
	if folder.IsRoot {
		return nil, ErrFolderInvalidMove
	}
	return &folder, nil
}

func (r *knowledgeFolderRepository) RenameSubtree(
	ctx context.Context, tenantID uint64, kbID, folderID, name string,
) (*types.KnowledgeFolder, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrFolderNotFound
	}
	var out *types.KnowledgeFolder
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		folder, err := r.subtree(ctx, tx, tenantID, kbID, folderID)
		if err != nil {
			return err
		}
		oldPath := folder.Path
		newPath := name
		if folder.ParentID != "" {
			parent, err := r.GetByID(ctx, tenantID, kbID, folder.ParentID)
			if err != nil {
				return err
			}
			if parent == nil {
				return ErrFolderNotFound
			}
			newPath = parent.Path
			if newPath != "" {
				newPath += "/"
			}
			newPath += name
		}
		if oldPath == newPath && folder.Name == name {
			out = folder
			return nil
		}
		if err := tx.Model(&types.KnowledgeFolder{}).
			Where("tenant_id = ? AND kb_id = ? AND id = ?", tenantID, kbID, folderID).
			Updates(map[string]any{"name": name, "path": newPath, "updated_at": time.Now()}).Error; err != nil {
			return err
		}
		if err := tx.Model(&types.KnowledgeFolder{}).
			Where("tenant_id = ? AND kb_id = ? AND path LIKE ?", tenantID, kbID, oldPath+"/%").
			Updates(map[string]any{
				"path":       gorm.Expr("? || substr(path, ?)", newPath, utf8.RuneCountInString(oldPath)+1),
				"updated_at": time.Now(),
			}).Error; err != nil {
			return err
		}
		if err := tx.Model(&types.Knowledge{}).
			Where("tenant_id = ? AND knowledge_base_id = ? AND (folder_path = ? OR folder_path LIKE ?)",
				tenantID, kbID, oldPath, oldPath+"/%").
			Updates(map[string]any{
				"folder_path": gorm.Expr("? || substr(folder_path, ?)", newPath, utf8.RuneCountInString(oldPath)+1),
				"updated_at":  time.Now(),
			}).Error; err != nil {
			return err
		}
		folder.Name = name
		folder.Path = newPath
		folder.UpdatedAt = time.Now()
		out = folder
		return nil
	})
	if isDuplicateFolderError(err) {
		return nil, ErrFolderAlreadyExists
	}
	return out, err
}

func (r *knowledgeFolderRepository) MoveSubtree(
	ctx context.Context, tenantID uint64, kbID, folderID, newParentID string,
) (*types.KnowledgeFolder, error) {
	var out *types.KnowledgeFolder
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		folder, err := r.subtree(ctx, tx, tenantID, kbID, folderID)
		if err != nil {
			return err
		}
		if newParentID == folder.ID {
			return ErrFolderInvalidMove
		}
		if newParentID == "" {
			return ErrFolderInvalidMove
		}
		parent, err := r.GetByID(ctx, tenantID, kbID, newParentID)
		if err != nil {
			return err
		}
		if parent == nil {
			return ErrFolderNotFound
		}
		if parent.Path == folder.Path || strings.HasPrefix(parent.Path, folder.Path+"/") {
			return ErrFolderInvalidMove
		}
		oldPath := folder.Path
		newPath := parent.Path
		if newPath != "" {
			newPath += "/"
		}
		newPath += folder.Name
		depthDelta := parent.Depth + 1 - folder.Depth
		if oldPath == newPath && folder.ParentID == newParentID {
			out = folder
			return nil
		}
		if err := tx.Model(&types.KnowledgeFolder{}).
			Where("tenant_id = ? AND kb_id = ? AND id = ?", tenantID, kbID, folderID).
			Updates(map[string]any{
				"parent_id":  newParentID,
				"path":       newPath,
				"depth":      parent.Depth + 1,
				"updated_at": time.Now(),
			}).Error; err != nil {
			return err
		}
		if err := tx.Model(&types.KnowledgeFolder{}).
			Where("tenant_id = ? AND kb_id = ? AND path LIKE ?", tenantID, kbID, oldPath+"/%").
			Updates(map[string]any{
				"path":       gorm.Expr("? || substr(path, ?)", newPath, utf8.RuneCountInString(oldPath)+1),
				"depth":      gorm.Expr("depth + ?", depthDelta),
				"updated_at": time.Now(),
			}).Error; err != nil {
			return err
		}
		if err := tx.Model(&types.Knowledge{}).
			Where("tenant_id = ? AND knowledge_base_id = ? AND (folder_path = ? OR folder_path LIKE ?)",
				tenantID, kbID, oldPath, oldPath+"/%").
			Updates(map[string]any{
				"folder_path": gorm.Expr("? || substr(folder_path, ?)", newPath, utf8.RuneCountInString(oldPath)+1),
				"updated_at":  time.Now(),
			}).Error; err != nil {
			return err
		}
		folder.ParentID = newParentID
		folder.Path = newPath
		folder.Depth = parent.Depth + 1
		folder.UpdatedAt = time.Now()
		out = folder
		return bumpTenantACLEpoch(tx, tenantID)
	})
	if isDuplicateFolderError(err) {
		return nil, ErrFolderAlreadyExists
	}
	return out, err
}

func (r *knowledgeFolderRepository) Delete(
	ctx context.Context, tenantID uint64, kbID, folderID string,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		folder, err := r.subtree(ctx, tx, tenantID, kbID, folderID)
		if err != nil {
			return err
		}
		var childCount int64
		if err := tx.Model(&types.KnowledgeFolder{}).
			Where("tenant_id = ? AND kb_id = ? AND parent_id = ?", tenantID, kbID, folder.ID).
			Count(&childCount).Error; err != nil {
			return err
		}
		var docCount int64
		if err := tx.Model(&types.Knowledge{}).
			Where("tenant_id = ? AND knowledge_base_id = ? AND folder_id = ? AND parse_status <> ?",
				tenantID, kbID, folder.ID, types.ParseStatusDeleting).
			Count(&docCount).Error; err != nil {
			return err
		}
		if childCount > 0 || docCount > 0 {
			return ErrFolderNotEmpty
		}
		if err := tx.Where("tenant_id = ? AND kb_id = ? AND id = ?", tenantID, kbID, folder.ID).
			Delete(&types.KnowledgeFolder{}).Error; err != nil {
			return err
		}
		return bumpTenantACLEpoch(tx, tenantID)
	})
}

func (r *knowledgeFolderRepository) CountDocumentsByFolder(
	ctx context.Context, tenantID uint64, kbID string,
) (map[string]int64, error) {
	var rows []*types.KnowledgeFolderCount
	if err := r.db.WithContext(ctx).
		Model(&types.Knowledge{}).
		Select("folder_id AS folder_id, COUNT(*) AS count").
		Where("tenant_id = ? AND knowledge_base_id = ? AND parse_status <> ?",
			tenantID, kbID, types.ParseStatusDeleting).
		Group("folder_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		if row != nil {
			out[row.FolderID] = row.Count
		}
	}
	return out, nil
}

func (r *knowledgeFolderRepository) UpdateKnowledgeFolderIDs(
	ctx context.Context, tenantID uint64, kbID string, knowledgeIDs []string, folder *types.KnowledgeFolder,
) (int64, error) {
	if len(knowledgeIDs) == 0 {
		return 0, nil
	}
	if folder == nil {
		return 0, ErrFolderNotFound
	}
	var affected int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.
			Model(&types.Knowledge{}).
			Where("tenant_id = ? AND knowledge_base_id = ? AND id IN ?", tenantID, kbID, knowledgeIDs).
			Updates(map[string]any{
				"folder_id":   folder.ID,
				"folder_path": folder.Path,
				"updated_at":  time.Now(),
			})
		if res.Error != nil {
			return res.Error
		}
		affected = res.RowsAffected
		if affected == 0 {
			return nil
		}
		return bumpTenantACLEpoch(tx, tenantID)
	})
	return affected, err
}

func (r *knowledgeFolderRepository) EnsurePath(
	ctx context.Context, tenantID uint64, kbID, rawPath, createdBy string,
) (*types.KnowledgeFolder, error) {
	path := types.NormalizeKnowledgeFolderPath(rawPath)
	root, err := r.EnsureRoot(ctx, tenantID, kbID, createdBy)
	if err != nil {
		return nil, err
	}
	if path == "" {
		return root, nil
	}
	parent := root
	for _, segment := range strings.Split(path, "/") {
		currentPath := parent.Path
		if currentPath != "" {
			currentPath += "/"
		}
		currentPath += segment
		next, err := r.GetByPath(ctx, tenantID, kbID, currentPath)
		if err != nil {
			return nil, err
		}
		if next == nil {
			next = &types.KnowledgeFolder{
				TenantID:        tenantID,
				KnowledgeBaseID: kbID,
				ParentID:        parent.ID,
				Name:            segment,
				CreatedBy:       createdBy,
			}
			if err := r.Create(ctx, next); err != nil {
				// Concurrent create: read the winner.
				next, err = r.GetByPath(ctx, tenantID, kbID, currentPath)
				if err != nil {
					return nil, err
				}
				if next == nil {
					return nil, ErrFolderNotFound
				}
			}
		}
		parent = next
	}
	return parent, nil
}
