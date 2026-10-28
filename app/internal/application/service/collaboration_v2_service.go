package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"gaze/internal/application/repository"
	werrors "gaze/internal/errors"
	"gaze/internal/types"
	"github.com/google/uuid"
)

func (s *collaborationService) ListDocCRChunkLocks(
	ctx context.Context, kbID string, knowledgeID string,
) ([]*types.DocCRChunkLock, error) {
	if _, err := collaborationActor(ctx); err != nil {
		return nil, err
	}
	kb, err := s.getKB(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if err := ensureDocRead(ctx, kb); err != nil {
		return nil, err
	}
	if _, err := s.loadKnowledgeForKB(ctx, kb, knowledgeID); err != nil {
		return nil, err
	}
	return s.repo.ListActiveDocCRChunkLocks(ctx, kb.TenantID, kb.ID, knowledgeID)
}

func (s *collaborationService) ListChunkCitationStats(
	ctx context.Context, kbID string, chunkIDs []string,
) ([]*types.ChunkCitationStat, error) {
	if _, err := collaborationActor(ctx); err != nil {
		return nil, err
	}
	kb, err := s.getKB(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if err := ensureDocRead(ctx, kb); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(chunkIDs))
	ids := make([]string, 0, len(chunkIDs))
	for _, id := range chunkIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
		if len(ids) >= 500 {
			break
		}
	}
	if s.aclService != nil && len(ids) > 0 {
		chunks := s.chunkMap(ctx, kb.TenantID, ids)
		allowedKnowledge := make(map[string]bool)
		filtered := make([]string, 0, len(ids))
		for _, id := range ids {
			chunk, ok := chunks[id]
			if !ok {
				continue
			}
			allowed, cached := allowedKnowledge[chunk.KnowledgeID]
			if !cached {
				knowledge, err := s.knowledgeRepo.GetKnowledgeByIDOnly(ctx, chunk.KnowledgeID)
				if err != nil || knowledge == nil || knowledge.KnowledgeBaseID != kb.ID || knowledge.TenantID != kb.TenantID {
					allowed = false
				} else {
					allowed, _, err = s.aclService.Evaluate(
						ctx, kb.ID, string(types.KBAclResourceFile), knowledge.ID, knowledge.FolderID,
					)
					allowed = err == nil && allowed
				}
				allowedKnowledge[chunk.KnowledgeID] = allowed
			}
			if allowed {
				filtered = append(filtered, id)
			}
		}
		ids = filtered
	}
	return s.repo.ListChunkCitationStats(ctx, kb.TenantID, ids)
}

func (s *collaborationService) ListDocAnchorComments(
	ctx context.Context, kbID string, knowledgeID string,
) ([]*types.DocAnchorComment, error) {
	if _, err := collaborationActor(ctx); err != nil {
		return nil, err
	}
	kb, err := s.getKB(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if err := ensureDocRead(ctx, kb); err != nil {
		return nil, err
	}
	if _, err := s.loadKnowledgeForKB(ctx, kb, knowledgeID); err != nil {
		return nil, err
	}
	comments, err := s.repo.ListDocAnchorComments(ctx, kb.TenantID, knowledgeID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(comments))
	for _, comment := range comments {
		ids = append(ids, comment.CreatedBy)
	}
	names := s.userNameMap(ctx, ids)
	for _, comment := range comments {
		comment.CreatedByName = names[comment.CreatedBy]
	}
	return comments, nil
}

func (s *collaborationService) CreateDocAnchorComment(
	ctx context.Context, kbID string, req *types.CreateDocAnchorCommentRequest,
) (*types.DocAnchorComment, error) {
	caller, err := collaborationActor(ctx)
	if err != nil {
		return nil, err
	}
	kb, err := s.getKB(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if err := ensureDocRead(ctx, kb); err != nil {
		return nil, err
	}
	knowledge, err := s.loadKnowledgeForKB(ctx, kb, req.KnowledgeID)
	if err != nil {
		return nil, err
	}
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return nil, werrors.NewValidationError("评论内容不能为空")
	}
	if len([]rune(body)) > docCommentMaxLen {
		return nil, werrors.NewValidationError("评论过长")
	}
	chunkID := strings.TrimSpace(req.ChunkID)
	if chunkID != "" {
		chunk, chunkErr := s.chunkRepo.GetChunkByID(ctx, kb.TenantID, chunkID)
		if chunkErr != nil || chunk == nil || chunk.KnowledgeID != knowledge.ID {
			return nil, werrors.NewValidationError("评论锚点无效")
		}
	}
	now := time.Now()
	comment := &types.DocAnchorComment{
		ID:              uuid.NewString(),
		TenantID:        kb.TenantID,
		KnowledgeBaseID: kb.ID,
		KnowledgeID:     knowledge.ID,
		ChunkID:         chunkID,
		ParentID:        strings.TrimSpace(req.ParentID),
		AnchorText:      strings.TrimSpace(req.AnchorText),
		AnchorStart:     maxInt(req.AnchorStart, 0),
		AnchorEnd:       maxInt(req.AnchorEnd, 0),
		AnchorLineStart: maxInt(req.AnchorLineStart, 0),
		AnchorLineEnd:   maxInt(req.AnchorLineEnd, 0),
		Body:            body,
		CreatedBy:       caller.UserID,
		CreatedAt:       now,
	}
	if comment.ParentID != "" {
		parent, parentErr := s.repo.GetDocAnchorComment(ctx, kb.TenantID, comment.ParentID)
		if parentErr != nil || parent.KnowledgeID != knowledge.ID {
			return nil, werrors.NewValidationError("回复的评论不存在")
		}
		if comment.ChunkID == "" {
			comment.ChunkID = parent.ChunkID
		}
		if comment.AnchorText == "" {
			comment.AnchorText = parent.AnchorText
			comment.AnchorStart = parent.AnchorStart
			comment.AnchorEnd = parent.AnchorEnd
			comment.AnchorLineStart = parent.AnchorLineStart
			comment.AnchorLineEnd = parent.AnchorLineEnd
		}
	}
	if err := s.repo.CreateDocAnchorComment(ctx, comment); err != nil {
		return nil, err
	}
	names := s.userNameMap(ctx, []string{caller.UserID})
	comment.CreatedByName = names[caller.UserID]
	return comment, nil
}

func (s *collaborationService) UpdateDocAnchorComment(
	ctx context.Context, kbID string, commentID string, req *types.UpdateDocAnchorCommentRequest,
) (*types.DocAnchorComment, error) {
	caller, err := collaborationActor(ctx)
	if err != nil {
		return nil, err
	}
	kb, err := s.getKB(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if err := ensureDocRead(ctx, kb); err != nil {
		return nil, err
	}
	comment, err := s.repo.GetDocAnchorComment(ctx, kb.TenantID, commentID)
	if err != nil {
		if errors.Is(err, repository.ErrDocAnchorNotFound) {
			return nil, werrors.NewNotFoundError("评论不存在")
		}
		return nil, err
	}
	if comment.KnowledgeBaseID != kb.ID {
		return nil, werrors.NewNotFoundError("评论不存在")
	}
	if err := s.ensureKnowledgeACLByID(ctx, kb, comment.KnowledgeID); err != nil {
		return nil, err
	}
	moderator := canModerateDoc(ctx, kb)
	if comment.CreatedBy != caller.UserID && !moderator {
		return nil, werrors.NewForbiddenError("无权修改该评论")
	}
	if req.Body != nil {
		body := strings.TrimSpace(*req.Body)
		if body == "" {
			return nil, werrors.NewValidationError("评论内容不能为空")
		}
		if len([]rune(body)) > docCommentMaxLen {
			return nil, werrors.NewValidationError("评论过长")
		}
		comment.Body = body
	}
	if req.Resolved != nil {
		if *req.Resolved {
			now := time.Now()
			comment.ResolvedAt = &now
			comment.ResolvedBy = caller.UserID
		} else {
			comment.ResolvedAt = nil
			comment.ResolvedBy = ""
		}
	}
	if err := s.repo.UpdateDocAnchorComment(ctx, comment); err != nil {
		return nil, err
	}
	names := s.userNameMap(ctx, []string{comment.CreatedBy})
	comment.CreatedByName = names[comment.CreatedBy]
	return comment, nil
}

func (s *collaborationService) DeleteDocAnchorComment(
	ctx context.Context, kbID string, commentID string,
) error {
	caller, err := collaborationActor(ctx)
	if err != nil {
		return err
	}
	kb, err := s.getKB(ctx, kbID)
	if err != nil {
		return err
	}
	if err := ensureDocRead(ctx, kb); err != nil {
		return err
	}
	comment, err := s.repo.GetDocAnchorComment(ctx, kb.TenantID, commentID)
	if err != nil {
		if errors.Is(err, repository.ErrDocAnchorNotFound) {
			return werrors.NewNotFoundError("评论不存在")
		}
		return err
	}
	if comment.KnowledgeBaseID != kb.ID {
		return werrors.NewNotFoundError("评论不存在")
	}
	if err := s.ensureKnowledgeACLByID(ctx, kb, comment.KnowledgeID); err != nil {
		return err
	}
	if comment.CreatedBy != caller.UserID && !canModerateDoc(ctx, kb) {
		return werrors.NewForbiddenError("无权删除该评论")
	}
	now := time.Now()
	comment.DeletedAt = &now
	comment.DeletedBy = caller.UserID
	return s.repo.UpdateDocAnchorComment(ctx, comment)
}

func (s *collaborationService) ListDocVersions(
	ctx context.Context, kbID string, knowledgeID string,
) ([]*types.DocVersion, error) {
	if _, err := collaborationActor(ctx); err != nil {
		return nil, err
	}
	kb, err := s.getKB(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if err := ensureDocRead(ctx, kb); err != nil {
		return nil, err
	}
	if _, err := s.loadKnowledgeForKB(ctx, kb, knowledgeID); err != nil {
		return nil, err
	}
	if s.versions == nil {
		return nil, werrors.NewInternalServerError("版本服务未配置")
	}
	versions, err := s.versions.ListVersions(ctx, kb.TenantID, kb.ID, knowledgeID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(versions))
	for _, version := range versions {
		ids = append(ids, version.CreatedBy)
	}
	names := s.userNameMap(ctx, ids)
	for _, version := range versions {
		version.CreatedByName = names[version.CreatedBy]
	}
	return versions, nil
}

func (s *collaborationService) GetDocVersion(
	ctx context.Context, kbID string, knowledgeID string, versionID string,
) (*types.DocVersion, []*types.DocVersionChunk, error) {
	if _, err := collaborationActor(ctx); err != nil {
		return nil, nil, err
	}
	kb, err := s.getKB(ctx, kbID)
	if err != nil {
		return nil, nil, err
	}
	if err := ensureDocRead(ctx, kb); err != nil {
		return nil, nil, err
	}
	if _, err := s.loadKnowledgeForKB(ctx, kb, knowledgeID); err != nil {
		return nil, nil, err
	}
	if s.versions == nil {
		return nil, nil, werrors.NewInternalServerError("版本服务未配置")
	}
	version, chunks, err := s.versions.GetVersion(ctx, kb.TenantID, kb.ID, knowledgeID, versionID)
	if err != nil {
		if errors.Is(err, repository.ErrDocVersionNotFound) {
			return nil, nil, werrors.NewNotFoundError("版本不存在")
		}
		return nil, nil, err
	}
	return version, chunks, nil
}

func (s *collaborationService) CompareDocVersions(
	ctx context.Context, kbID string, knowledgeID string, from string, to string,
) (*types.DocVersionCompareResponse, error) {
	if _, err := collaborationActor(ctx); err != nil {
		return nil, err
	}
	kb, err := s.getKB(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if err := ensureDocRead(ctx, kb); err != nil {
		return nil, err
	}
	if _, err := s.loadKnowledgeForKB(ctx, kb, knowledgeID); err != nil {
		return nil, err
	}
	if s.versions == nil {
		return nil, werrors.NewInternalServerError("版本服务未配置")
	}
	return s.versions.CompareVersions(ctx, kb.TenantID, kb.ID, knowledgeID, from, to)
}

func (s *collaborationService) RollbackDocVersion(
	ctx context.Context, kbID string, knowledgeID string, versionID string, req *types.RollbackDocVersionRequest,
) (*types.DocVersion, error) {
	caller, err := collaborationActor(ctx)
	if err != nil {
		return nil, err
	}
	kb, err := s.getKB(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if !canModerateDoc(ctx, kb) {
		return nil, werrors.NewForbiddenError("仅管理员可回滚版本")
	}
	if _, err := s.loadKnowledgeForKB(ctx, kb, knowledgeID); err != nil {
		return nil, err
	}
	if s.versions == nil {
		return nil, werrors.NewInternalServerError("版本服务未配置")
	}
	comment := ""
	if req != nil {
		comment = strings.TrimSpace(req.Comment)
	}
	version, err := s.versions.Rollback(ctx, kb.TenantID, kb.ID, knowledgeID, versionID, caller.UserID, comment)
	if err != nil {
		if errors.Is(err, repository.ErrDocVersionNotFound) {
			return nil, werrors.NewNotFoundError("版本不存在")
		}
		return nil, err
	}
	return version, nil
}
