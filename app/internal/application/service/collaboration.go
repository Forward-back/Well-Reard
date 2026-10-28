package service

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"gaze/internal/application/access"
	"gaze/internal/application/repository"
	werrors "gaze/internal/errors"
	"gaze/internal/logger"
	"gaze/internal/types"
	"gaze/internal/types/interfaces"
	"github.com/google/uuid"
)

const (
	docIssueTitleMaxLen = 255
	docCommentMaxLen    = 5000
	docChunkExcerptLen  = 120
)

// collaborationService implements Collaborative document maintenance.
type collaborationService struct {
	repo           interfaces.CollaborationRepository
	kbService      interfaces.KnowledgeBaseService
	knowledgeRepo  interfaces.KnowledgeRepository
	chunkRepo      interfaces.ChunkRepository
	userRepo       interfaces.UserRepository
	versions       interfaces.DocumentVersionService
	aclService     interfaces.KBAclService
	departmentRepo interfaces.DepartmentRepository
	audit          interfaces.AuditLogService
}

// NewCollaborationService creates the collaboration service.
func NewCollaborationService(
	repo interfaces.CollaborationRepository,
	kbService interfaces.KnowledgeBaseService,
	knowledgeRepo interfaces.KnowledgeRepository,
	chunkRepo interfaces.ChunkRepository,
	userRepo interfaces.UserRepository,
	versions interfaces.DocumentVersionService,
	aclService interfaces.KBAclService,
	departmentRepo interfaces.DepartmentRepository,
	audit interfaces.AuditLogService,
) interfaces.CollaborationService {
	return &collaborationService{
		repo:           repo,
		kbService:      kbService,
		knowledgeRepo:  knowledgeRepo,
		chunkRepo:      chunkRepo,
		userRepo:       userRepo,
		versions:       versions,
		aclService:     aclService,
		departmentRepo: departmentRepo,
		audit:          audit,
	}
}

func collaborationActor(ctx context.Context) (types.Caller, error) {
	caller := types.CallerFromContext(ctx).Normalize()
	if caller.TenantID == 0 || caller.UserID == "" {
		return caller, werrors.NewForbiddenError("需要登录后操作")
	}
	return caller, nil
}

func (s *collaborationService) getKB(ctx context.Context, kbID string) (*types.KnowledgeBase, error) {
	kb, err := s.kbService.GetKnowledgeBaseByID(ctx, kbID)
	if err != nil || kb == nil {
		return nil, werrors.NewNotFoundError("知识库不存在")
	}
	return kb, nil
}

func ensureDocRead(ctx context.Context, kb *types.KnowledgeBase) error {
	if kb == nil {
		return werrors.NewForbiddenError("无权访问该知识库")
	}
	if access.HasKBGrant(ctx, kb.ID, kb.TenantID, types.OrgRoleViewer) {
		return nil
	}
	caller := types.CallerFromContext(ctx).Normalize()
	if kb.DepartmentID != "" && caller.DepartmentID == kb.DepartmentID {
		return nil
	}
	return werrors.NewForbiddenError("无权访问该知识库")
}

func ensureDocEditor(ctx context.Context, kb *types.KnowledgeBase) error {
	if kb == nil {
		return werrors.NewForbiddenError("仅编辑及以上角色可提交变更请求")
	}
	if access.HasKBGrant(ctx, kb.ID, kb.TenantID, types.OrgRoleEditor) {
		return nil
	}
	caller := types.CallerFromContext(ctx).Normalize()
	if kb.DepartmentID != "" && caller.DepartmentID == kb.DepartmentID &&
		caller.DepartmentRole.HasPermission(types.DepartmentRoleEditor) {
		return nil
	}
	return werrors.NewForbiddenError("仅编辑及以上角色可提交变更请求")
}

func canModerateDoc(ctx context.Context, kb *types.KnowledgeBase) bool {
	if kb == nil {
		return false
	}
	if access.HasKBGrant(ctx, kb.ID, kb.TenantID, types.OrgRoleAdmin) {
		return true
	}
	caller := types.CallerFromContext(ctx).Normalize()
	return kb.DepartmentID != "" && caller.DepartmentID == kb.DepartmentID &&
		caller.DepartmentRole == types.DepartmentRoleManager
}

func (s *collaborationService) aclScope(ctx context.Context, kb *types.KnowledgeBase) (*types.KBAclScope, error) {
	if s.aclService == nil || kb == nil {
		return nil, nil
	}
	return s.aclService.Scope(ctx, kb.ID)
}

func (s *collaborationService) ensureKnowledgeACL(
	ctx context.Context, kb *types.KnowledgeBase, knowledge *types.Knowledge,
) error {
	if s.aclService == nil || kb == nil || knowledge == nil {
		return nil
	}
	allowed, _, err := s.aclService.Evaluate(
		ctx, kb.ID, string(types.KBAclResourceFile), knowledge.ID, knowledge.FolderID,
	)
	if err != nil {
		return err
	}
	if !allowed {
		return werrors.NewForbiddenError("无权访问该文档")
	}
	return nil
}

func (s *collaborationService) ensureKnowledgeACLByID(
	ctx context.Context, kb *types.KnowledgeBase, knowledgeID string,
) error {
	if s.aclService == nil || kb == nil || strings.TrimSpace(knowledgeID) == "" {
		return nil
	}
	knowledge, err := s.knowledgeRepo.GetKnowledgeByIDOnly(ctx, knowledgeID)
	if err != nil || knowledge == nil || knowledge.KnowledgeBaseID != kb.ID || knowledge.TenantID != kb.TenantID {
		return werrors.NewNotFoundError("文档不存在")
	}
	return s.ensureKnowledgeACL(ctx, kb, knowledge)
}

func (s *collaborationService) loadKnowledgeForKB(
	ctx context.Context, kb *types.KnowledgeBase, knowledgeID string,
) (*types.Knowledge, error) {
	if strings.TrimSpace(knowledgeID) == "" {
		return nil, werrors.NewValidationError("缺少文档 ID")
	}
	knowledge, err := s.knowledgeRepo.GetKnowledgeByIDOnly(ctx, knowledgeID)
	if err != nil || knowledge == nil || knowledge.KnowledgeBaseID != kb.ID || knowledge.TenantID != kb.TenantID {
		return nil, werrors.NewNotFoundError("文档不存在")
	}
	if err := s.ensureKnowledgeACL(ctx, kb, knowledge); err != nil {
		return nil, err
	}
	return knowledge, nil
}

func normalizeDocTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", werrors.NewValidationError("标题不能为空")
	}
	if len([]rune(title)) > docIssueTitleMaxLen {
		return "", werrors.NewValidationError("标题过长")
	}
	return title, nil
}

func normalizeIssueType(value string) string {
	switch strings.TrimSpace(value) {
	case types.DocIssueTypeDataError:
		return types.DocIssueTypeDataError
	case types.DocIssueTypeOutdated:
		return types.DocIssueTypeOutdated
	case types.DocIssueTypeUnclear:
		return types.DocIssueTypeUnclear
	case types.DocIssueTypeFormat:
		return types.DocIssueTypeFormat
	default:
		return types.DocIssueTypeOther
	}
}

func maxInt(value, floor int) int {
	if value < floor {
		return floor
	}
	return value
}

func normalizeCRTags(tags []string) types.StringList {
	out := make(types.StringList, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if len([]rune(tag)) > 32 {
			tag = string([]rune(tag)[:32])
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
		if len(out) >= 10 {
			break
		}
	}
	return out
}

func chunkExcerpt(content string) string {
	content = strings.Join(strings.Fields(content), " ")
	runes := []rune(content)
	if len(runes) > docChunkExcerptLen {
		return string(runes[:docChunkExcerptLen]) + "…"
	}
	return content
}

func (s *collaborationService) userNameMap(ctx context.Context, ids []string) map[string]string {
	out := make(map[string]string, len(ids))
	seen := make(map[string]struct{}, len(ids))
	unique := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return out
	}
	users, err := s.userRepo.GetUsersByIDs(ctx, unique)
	if err != nil {
		logger.Warnf(ctx, "collaboration: resolve user names failed: %v", err)
		return out
	}
	for id, user := range users {
		if user != nil && user.Username != "" {
			out[id] = user.Username
		}
	}
	return out
}

func (s *collaborationService) knowledgeTitleMap(ctx context.Context, ids []string) map[string]string {
	out := make(map[string]string, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := out[id]; ok {
			continue
		}
		knowledge, err := s.knowledgeRepo.GetKnowledgeByIDOnly(ctx, id)
		if err == nil && knowledge != nil {
			out[id] = knowledge.Title
		}
	}
	return out
}

func (s *collaborationService) chunkMap(
	ctx context.Context, tenantID uint64, ids []string,
) map[string]*types.Chunk {
	out := make(map[string]*types.Chunk, len(ids))
	if len(ids) == 0 {
		return out
	}
	chunks, err := s.chunkRepo.ListChunksByID(ctx, tenantID, ids)
	if err != nil {
		logger.Warnf(ctx, "collaboration: load chunks failed: %v", err)
		return out
	}
	for _, chunk := range chunks {
		out[chunk.ID] = chunk
	}
	return out
}

func (s *collaborationService) enrichIssues(ctx context.Context, issues []*types.DocIssue) {
	if len(issues) == 0 {
		return
	}
	tenantID := issues[0].TenantID
	userIDs := make([]string, 0, len(issues))
	knowledgeIDs := make([]string, 0, len(issues))
	chunkIDs := make([]string, 0, len(issues))
	issueIDs := make([]string, 0, len(issues))
	for _, issue := range issues {
		userIDs = append(userIDs, issue.CreatedBy)
		knowledgeIDs = append(knowledgeIDs, issue.KnowledgeID)
		issueIDs = append(issueIDs, issue.ID)
		if issue.ChunkID != "" {
			chunkIDs = append(chunkIDs, issue.ChunkID)
		}
	}
	names := s.userNameMap(ctx, userIDs)
	titles := s.knowledgeTitleMap(ctx, knowledgeIDs)
	chunks := s.chunkMap(ctx, tenantID, chunkIDs)
	counts, err := s.repo.CountDocCommentsByTargets(
		ctx, tenantID, issues[0].KnowledgeBaseID, types.DocCommentTargetIssue, issueIDs,
	)
	if err != nil {
		logger.Warnf(ctx, "collaboration: count issue comments failed: %v", err)
		counts = map[string]int64{}
	}

	missingByKnowledge := map[string][]string{}
	for _, issue := range issues {
		issue.CreatedByName = names[issue.CreatedBy]
		issue.KnowledgeTitle = titles[issue.KnowledgeID]
		issue.CommentsCount = counts[issue.ID]
		if issue.ChunkID != "" {
			chunk, ok := chunks[issue.ChunkID]
			if !ok {
				issue.AnchorOrphaned = true
				missingByKnowledge[issue.KnowledgeID] = append(missingByKnowledge[issue.KnowledgeID], issue.ChunkID)
				continue
			}
			issue.ChunkIndex = chunk.ChunkIndex
			issue.ChunkExcerpt = chunkExcerpt(chunk.Content)
		}
	}
	for knowledgeID, missing := range missingByKnowledge {
		if err := s.repo.MarkDocIssuesOrphaned(ctx, tenantID, knowledgeID, missing); err != nil {
			logger.Warnf(ctx, "collaboration: mark issue anchors orphaned failed: %v", err)
		}
	}
}

func (s *collaborationService) enrichCR(ctx context.Context, kb *types.KnowledgeBase, cr *types.DocChangeRequest) {
	userIDs := []string{cr.CreatedBy, cr.ReviewedBy}
	reviewIDs := make([]string, 0, len(cr.Reviews))
	for _, review := range cr.Reviews {
		reviewIDs = append(reviewIDs, review.ReviewedBy)
	}
	names := s.userNameMap(ctx, append(userIDs, reviewIDs...))
	titles := s.knowledgeTitleMap(ctx, []string{cr.KnowledgeID})
	cr.CreatedByName = names[cr.CreatedBy]
	cr.ReviewedByName = names[cr.ReviewedBy]
	cr.KnowledgeTitle = titles[cr.KnowledgeID]
	for _, review := range cr.Reviews {
		review.ReviewedByName = names[review.ReviewedBy]
	}
	if reviewers, err := s.repo.ListDocCRReviewers(ctx, cr.TenantID, cr.ID); err != nil {
		logger.Warnf(ctx, "collaboration: list CR reviewers failed: %v", err)
	} else {
		ids := make([]string, 0, len(reviewers))
		for _, reviewer := range reviewers {
			ids = append(ids, reviewer.UserID)
		}
		reviewerNames := s.userNameMap(ctx, ids)
		for _, reviewer := range reviewers {
			reviewer.UserName = reviewerNames[reviewer.UserID]
		}
		cr.Reviewers = reviewers
	}
	cr.ItemCount = int64(len(cr.Items))
	if len(cr.Items) == 0 {
		if linked, err := s.repo.ListDocCRLinkedIssues(ctx, cr.TenantID, cr.ID); err == nil && len(linked) > 0 {
			s.enrichIssues(ctx, linked)
			cr.LinkedIssues = linked
		}
		return
	}

	chunkIDs := make([]string, 0, len(cr.Items))
	for _, item := range cr.Items {
		chunkIDs = append(chunkIDs, item.ChunkID)
	}
	chunks := s.chunkMap(ctx, cr.TenantID, chunkIDs)
	missing := make([]string, 0)
	for _, item := range cr.Items {
		chunk, ok := chunks[item.ChunkID]
		if !ok {
			item.AnchorOrphaned = true
			missing = append(missing, item.ChunkID)
			continue
		}
		item.ChunkIndex = chunk.ChunkIndex
		item.CurrentContent = chunk.Content
		// Conflict is only actionable before merge; after merging the current
		// content differs from the snapshot by design.
		if cr.Status != types.DocCRStatusMerged && cr.Status != types.DocCRStatusClosed {
			item.Conflict = chunk.Content != item.OriginalContent
		}
	}
	if len(missing) > 0 {
		cr.AnchorOrphaned = true
		if err := s.repo.MarkDocCRsOrphaned(ctx, cr.TenantID, cr.KnowledgeID, missing); err != nil {
			logger.Warnf(ctx, "collaboration: mark CR anchors orphaned failed: %v", err)
		}
	}

	linkedByID := make(map[string]*types.DocIssue)
	order := make([]string, 0)
	addIssue := func(issue *types.DocIssue) {
		if issue == nil {
			return
		}
		if _, ok := linkedByID[issue.ID]; !ok {
			order = append(order, issue.ID)
		}
		linkedByID[issue.ID] = issue
	}
	if explicit, err := s.repo.ListDocCRLinkedIssues(ctx, cr.TenantID, cr.ID); err != nil {
		logger.Warnf(ctx, "collaboration: list explicit linked issues failed: %v", err)
	} else {
		s.enrichIssues(ctx, explicit)
		for _, issue := range explicit {
			addIssue(issue)
		}
	}
	if issues, err := s.repo.ListOpenDocIssuesByChunkIDs(ctx, cr.TenantID, kb.ID, chunkIDs); err != nil {
		logger.Warnf(ctx, "collaboration: list linked issues failed: %v", err)
	} else {
		s.enrichIssues(ctx, issues)
		for _, issue := range issues {
			addIssue(issue)
		}
	}
	if len(order) > 0 {
		linked := make([]*types.DocIssue, 0, len(order))
		for _, id := range order {
			linked = append(linked, linkedByID[id])
		}
		cr.LinkedIssues = linked
	}
}

func (s *collaborationService) ListDocIssues(
	ctx context.Context, req *types.DocIssueListRequest,
) (*types.DocIssueListResponse, error) {
	if _, err := collaborationActor(ctx); err != nil {
		return nil, err
	}
	kb, err := s.getKB(ctx, req.KnowledgeBaseID)
	if err != nil {
		return nil, err
	}
	if err := ensureDocRead(ctx, kb); err != nil {
		return nil, err
	}
	req.TenantID = kb.TenantID
	if scope, err := s.aclScope(ctx, kb); err != nil {
		return nil, err
	} else if scope != nil && scope.Active {
		req.ACLActive = true
		req.AllowedKnowledgeIDs = scope.AllowedKnowledgeIDs
	}
	issues, total, err := s.repo.ListDocIssues(ctx, req)
	if err != nil {
		return nil, err
	}
	s.enrichIssues(ctx, issues)
	page, pageSize := normalizeDocPagination(req.Page, req.PageSize)
	return &types.DocIssueListResponse{Items: issues, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *collaborationService) CreateDocIssue(
	ctx context.Context, kbID string, req *types.CreateDocIssueRequest,
) (*types.DocIssue, error) {
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
	title, err := normalizeDocTitle(req.Title)
	if err != nil {
		return nil, err
	}
	issueType := normalizeIssueType(req.IssueType)
	chunkID := strings.TrimSpace(req.ChunkID)
	if chunkID != "" {
		chunk, err := s.chunkRepo.GetChunkByID(ctx, kb.TenantID, chunkID)
		if err != nil || chunk == nil || chunk.KnowledgeID != knowledge.ID || chunk.KnowledgeBaseID != kb.ID {
			return nil, werrors.NewValidationError("分块锚点无效")
		}
	}
	now := time.Now()
	issue := &types.DocIssue{
		ID:              uuid.NewString(),
		TenantID:        kb.TenantID,
		KnowledgeBaseID: kb.ID,
		KnowledgeID:     knowledge.ID,
		ChunkID:         chunkID,
		IssueType:       issueType,
		AnchorText:      strings.TrimSpace(req.AnchorText),
		AnchorStart:     maxInt(req.AnchorStart, 0),
		AnchorEnd:       maxInt(req.AnchorEnd, 0),
		AnchorLineStart: maxInt(req.AnchorLineStart, 0),
		AnchorLineEnd:   maxInt(req.AnchorLineEnd, 0),
		Title:           title,
		Body:            strings.TrimSpace(req.Body),
		Status:          types.DocIssueStatusOpen,
		CreatedBy:       caller.UserID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.repo.CreateDocIssue(ctx, issue); err != nil {
		return nil, err
	}
	s.enrichIssues(ctx, []*types.DocIssue{issue})
	return issue, nil
}

func (s *collaborationService) GetDocIssue(
	ctx context.Context, kbID string, issueID string,
) (*types.DocIssue, error) {
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
	issue, err := s.repo.GetDocIssue(ctx, kb.TenantID, kb.ID, issueID)
	if err != nil {
		if errors.Is(err, repository.ErrDocIssueNotFound) {
			return nil, werrors.NewNotFoundError("问题不存在")
		}
		return nil, err
	}
	if err := s.ensureKnowledgeACLByID(ctx, kb, issue.KnowledgeID); err != nil {
		return nil, err
	}
	s.enrichIssues(ctx, []*types.DocIssue{issue})
	return issue, nil
}

func (s *collaborationService) UpdateDocIssue(
	ctx context.Context, kbID string, issueID string, req *types.UpdateDocIssueRequest,
) (*types.DocIssue, error) {
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
	issue, err := s.repo.GetDocIssue(ctx, kb.TenantID, kb.ID, issueID)
	if err != nil {
		if errors.Is(err, repository.ErrDocIssueNotFound) {
			return nil, werrors.NewNotFoundError("问题不存在")
		}
		return nil, err
	}
	if err := s.ensureKnowledgeACLByID(ctx, kb, issue.KnowledgeID); err != nil {
		return nil, err
	}
	moderator := canModerateDoc(ctx, kb)
	owner := issue.CreatedBy == caller.UserID
	if !moderator && !owner {
		return nil, werrors.NewForbiddenError("无权修改该问题")
	}
	if req.Title != nil {
		title, err := normalizeDocTitle(*req.Title)
		if err != nil {
			return nil, err
		}
		issue.Title = title
	}
	if req.Body != nil {
		issue.Body = strings.TrimSpace(*req.Body)
	}
	if req.IssueType != nil {
		issue.IssueType = normalizeIssueType(*req.IssueType)
	}
	if req.Status != nil {
		if !moderator {
			return nil, werrors.NewForbiddenError("仅管理员可关闭或重开问题")
		}
		status := strings.TrimSpace(*req.Status)
		if status != types.DocIssueStatusOpen && status != types.DocIssueStatusClosed {
			return nil, werrors.NewValidationError("无效的问题状态")
		}
		issue.Status = status
		if status == types.DocIssueStatusClosed {
			now := time.Now()
			issue.ClosedBy = caller.UserID
			issue.ClosedAt = &now
		} else {
			issue.ClosedBy = ""
			issue.ClosedAt = nil
		}
	}
	issue.UpdatedAt = time.Now()
	if err := s.repo.UpdateDocIssue(ctx, issue); err != nil {
		return nil, err
	}
	s.enrichIssues(ctx, []*types.DocIssue{issue})
	return issue, nil
}

func (s *collaborationService) DeleteDocIssue(
	ctx context.Context, kbID string, issueID string,
) error {
	caller, err := collaborationActor(ctx)
	if err != nil {
		return err
	}
	kb, err := s.getKB(ctx, kbID)
	if err != nil {
		return err
	}
	if !canModerateDoc(ctx, kb) {
		return werrors.NewForbiddenError("仅管理员可删除问题")
	}
	issue, err := s.repo.GetDocIssue(ctx, kb.TenantID, kb.ID, issueID)
	if err != nil {
		if errors.Is(err, repository.ErrDocIssueNotFound) {
			return werrors.NewNotFoundError("问题不存在")
		}
		return err
	}
	if err := s.ensureKnowledgeACLByID(ctx, kb, issue.KnowledgeID); err != nil {
		return err
	}
	now := time.Now()
	issue.DeletedBy = caller.UserID
	issue.DeletedAt = &now
	issue.UpdatedAt = now
	return s.repo.UpdateDocIssue(ctx, issue)
}

func (s *collaborationService) normalizeCRItems(
	ctx context.Context, kb *types.KnowledgeBase, knowledge *types.Knowledge, inputs []types.DocCRItemInput,
) ([]*types.DocCRItem, error) {
	if len(inputs) == 0 {
		return nil, werrors.NewValidationError("至少需要一个分块建议")
	}
	seen := make(map[string]struct{}, len(inputs))
	chunkIDs := make([]string, 0, len(inputs))
	for _, input := range inputs {
		chunkID := strings.TrimSpace(input.ChunkID)
		if chunkID == "" {
			return nil, werrors.NewValidationError("分块 ID 不能为空")
		}
		if _, ok := seen[chunkID]; ok {
			return nil, werrors.NewValidationError("同一个分块只能提交一条建议")
		}
		seen[chunkID] = struct{}{}
		chunkIDs = append(chunkIDs, chunkID)
	}
	chunks := s.chunkMap(ctx, kb.TenantID, chunkIDs)
	now := time.Now()
	items := make([]*types.DocCRItem, 0, len(inputs))
	for index, input := range inputs {
		chunkID := strings.TrimSpace(input.ChunkID)
		chunk, ok := chunks[chunkID]
		if !ok || chunk.KnowledgeID != knowledge.ID || chunk.KnowledgeBaseID != kb.ID {
			return nil, werrors.NewValidationError("分块锚点无效或已失效")
		}
		suggested := input.SuggestedContent
		if strings.TrimSpace(suggested) == "" {
			return nil, werrors.NewValidationError("建议内容不能为空")
		}
		if suggested == chunk.Content {
			return nil, werrors.NewValidationError("建议内容与当前内容一致")
		}
		items = append(items, &types.DocCRItem{
			ID:               uuid.NewString(),
			TenantID:         kb.TenantID,
			ChunkID:          chunkID,
			OriginalContent:  chunk.Content,
			SuggestedContent: suggested,
			Seq:              index,
			CreatedAt:        now,
		})
	}
	return items, nil
}

func (s *collaborationService) buildCRReviewers(
	ctx context.Context, kb *types.KnowledgeBase, crID string, userIDs []string, now time.Time,
) ([]*types.DocCRReviewer, error) {
	seen := make(map[string]struct{}, len(userIDs))
	unique := make([]string, 0, len(userIDs))
	for _, id := range userIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return nil, nil
	}
	users, err := s.userRepo.GetUsersByIDs(ctx, unique)
	if err != nil {
		return nil, err
	}
	out := make([]*types.DocCRReviewer, 0, len(unique))
	for _, id := range unique {
		if users[id] == nil {
			return nil, werrors.NewValidationError("审阅者不存在")
		}
		out = append(out, &types.DocCRReviewer{
			ID:        uuid.NewString(),
			TenantID:  kb.TenantID,
			CRID:      crID,
			UserID:    id,
			Status:    types.DocCRReviewerPending,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
	return out, nil
}

func (s *collaborationService) normalizeCRDraftItems(
	ctx context.Context, kb *types.KnowledgeBase, knowledge *types.Knowledge, inputs []types.DocCRItemInput,
) ([]*types.DocCRItem, error) {
	seen := make(map[string]struct{}, len(inputs))
	chunkIDs := make([]string, 0, len(inputs))
	for _, input := range inputs {
		chunkID := strings.TrimSpace(input.ChunkID)
		if chunkID == "" {
			continue
		}
		if _, ok := seen[chunkID]; ok {
			continue
		}
		seen[chunkID] = struct{}{}
		chunkIDs = append(chunkIDs, chunkID)
	}
	chunks := s.chunkMap(ctx, kb.TenantID, chunkIDs)
	now := time.Now()
	items := make([]*types.DocCRItem, 0, len(inputs))
	for index, input := range inputs {
		chunkID := strings.TrimSpace(input.ChunkID)
		if chunkID == "" {
			continue
		}
		chunk, ok := chunks[chunkID]
		if !ok || chunk.KnowledgeID != knowledge.ID || chunk.KnowledgeBaseID != kb.ID {
			continue
		}
		suggested := input.SuggestedContent
		if strings.TrimSpace(suggested) == "" {
			suggested = chunk.Content
		}
		items = append(items, &types.DocCRItem{
			ID:               uuid.NewString(),
			TenantID:         kb.TenantID,
			ChunkID:          chunkID,
			OriginalContent:  chunk.Content,
			OriginalRevision: chunk.ContentRevision,
			SuggestedContent: suggested,
			AnchorText:       strings.TrimSpace(input.AnchorText),
			Seq:              index,
			ViewedBy:         types.StringList{},
			CreatedAt:        now,
			UpdatedAt:        now,
		})
	}
	return items, nil
}

func (s *collaborationService) resolveCRLinkedIssueIDs(
	ctx context.Context, kb *types.KnowledgeBase, knowledgeID string, chunkIDs []string, explicit []string,
) ([]string, error) {
	allowed := make(map[string]struct{})
	if len(explicit) > 0 {
		for _, id := range explicit {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			issue, err := s.repo.GetDocIssue(ctx, kb.TenantID, kb.ID, id)
			if err != nil || issue.KnowledgeID != knowledgeID {
				return nil, werrors.NewValidationError("关联问题不存在")
			}
			allowed[id] = struct{}{}
		}
	}
	issues, err := s.repo.ListOpenDocIssuesByChunkIDs(ctx, kb.TenantID, kb.ID, chunkIDs)
	if err != nil {
		return nil, err
	}
	for _, issue := range issues {
		allowed[issue.ID] = struct{}{}
	}
	out := make([]string, 0, len(allowed))
	for id := range allowed {
		out = append(out, id)
	}
	return out, nil
}

func (s *collaborationService) CreateDocChangeRequest(
	ctx context.Context, kbID string, req *types.CreateDocCRRequest,
) (*types.DocChangeRequest, error) {
	caller, err := collaborationActor(ctx)
	if err != nil {
		return nil, err
	}
	kb, err := s.getKB(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if err := ensureDocEditor(ctx, kb); err != nil {
		return nil, err
	}
	knowledge, err := s.loadKnowledgeForKB(ctx, kb, req.KnowledgeID)
	if err != nil {
		return nil, err
	}
	status := types.DocCRStatusPendingReview
	if req.AsDraft {
		status = types.DocCRStatusDraft
	}
	title := strings.TrimSpace(req.Title)
	if title != "" || !req.AsDraft {
		title, err = normalizeDocTitle(title)
		if err != nil {
			return nil, err
		}
	}
	var items []*types.DocCRItem
	if req.AsDraft {
		items, err = s.normalizeCRDraftItems(ctx, kb, knowledge, req.Items)
	} else {
		items, err = s.normalizeCRItems(ctx, kb, knowledge, req.Items)
	}
	if err != nil {
		return nil, err
	}
	if !req.AsDraft && len(items) == 0 {
		return nil, werrors.NewValidationError("至少需要一个分块建议")
	}

	baseVersionID := strings.TrimSpace(req.BaseVersionID)
	if s.versions != nil {
		if latest, versionErr := s.versions.EnsureInitialVersion(ctx, kb.TenantID, kb.ID, knowledge.ID, caller.UserID); versionErr == nil && latest != nil {
			if baseVersionID == "" {
				baseVersionID = latest.ID
			}
		}
	}

	now := time.Now()
	cr := &types.DocChangeRequest{
		ID:              uuid.NewString(),
		TenantID:        kb.TenantID,
		KnowledgeBaseID: kb.ID,
		KnowledgeID:     knowledge.ID,
		Title:           title,
		Description:     strings.TrimSpace(req.Description),
		Status:          status,
		Tags:            normalizeCRTags(req.Tags),
		BaseVersionID:   baseVersionID,
		CreatedBy:       caller.UserID,
		CreatedAt:       now,
		UpdatedAt:       now,
		Items:           items,
	}
	if req.AsDraft {
		cr.DraftUpdatedAt = &now
	}
	if err := s.repo.CreateDocChangeRequest(ctx, cr); err != nil {
		return nil, err
	}
	reviewers, err := s.buildCRReviewers(ctx, kb, cr.ID, req.ReviewerIDs, now)
	if err != nil {
		return nil, err
	}
	if len(reviewers) > 0 {
		if err := s.repo.ReplaceDocCRReviewers(ctx, kb.TenantID, cr.ID, reviewers); err != nil {
			return nil, err
		}
	}
	chunkIDs := make([]string, 0, len(items))
	for _, item := range items {
		chunkIDs = append(chunkIDs, item.ChunkID)
	}
	linkedIDs, err := s.resolveCRLinkedIssueIDs(ctx, kb, knowledge.ID, chunkIDs, req.LinkedIssueIDs)
	if err != nil {
		return nil, err
	}
	if len(linkedIDs) > 0 {
		if err := s.repo.ReplaceDocCRLinkedIssues(ctx, kb.TenantID, cr.ID, linkedIDs); err != nil {
			return nil, err
		}
	}
	return s.GetDocChangeRequest(ctx, kb.ID, cr.ID)
}

func (s *collaborationService) ListDocChangeRequests(
	ctx context.Context, req *types.DocCRListRequest,
) (*types.DocCRListResponse, error) {
	caller, err := collaborationActor(ctx)
	if err != nil {
		return nil, err
	}
	kb, err := s.getKB(ctx, req.KnowledgeBaseID)
	if err != nil {
		return nil, err
	}
	if err := ensureDocRead(ctx, kb); err != nil {
		return nil, err
	}
	req.TenantID = kb.TenantID
	req.VisibilityUserID = caller.UserID
	if scope, err := s.aclScope(ctx, kb); err != nil {
		return nil, err
	} else if scope != nil && scope.Active {
		req.ACLActive = true
		req.AllowedKnowledgeIDs = scope.AllowedKnowledgeIDs
	}
	crs, total, err := s.repo.ListDocChangeRequests(ctx, req)
	if err != nil {
		return nil, err
	}
	names := s.userNameMap(ctx, collectCRUserIDs(crs))
	for _, cr := range crs {
		cr.CreatedByName = names[cr.CreatedBy]
		cr.ReviewedByName = names[cr.ReviewedBy]
	}
	page, pageSize := normalizeDocPagination(req.Page, req.PageSize)
	return &types.DocCRListResponse{Items: crs, Total: total, Page: page, PageSize: pageSize}, nil
}

func collectCRUserIDs(crs []*types.DocChangeRequest) []string {
	ids := make([]string, 0, len(crs)*2)
	for _, cr := range crs {
		ids = append(ids, cr.CreatedBy, cr.ReviewedBy)
	}
	return ids
}

func (s *collaborationService) GetDocChangeRequest(
	ctx context.Context, kbID string, crID string,
) (*types.DocChangeRequest, error) {
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
	cr, err := s.repo.GetDocChangeRequest(ctx, kb.TenantID, kb.ID, crID)
	if err != nil {
		if errors.Is(err, repository.ErrDocCRNotFound) {
			return nil, werrors.NewNotFoundError("变更请求不存在")
		}
		return nil, err
	}
	if err := s.ensureKnowledgeACLByID(ctx, kb, cr.KnowledgeID); err != nil {
		return nil, err
	}
	s.enrichCR(ctx, kb, cr)
	cr.CanReview = s.canReviewCR(ctx, kb, cr, types.CallerFromContext(ctx).Normalize())
	return cr, nil
}

func (s *collaborationService) canReviewCR(ctx context.Context, kb *types.KnowledgeBase, cr *types.DocChangeRequest, caller types.Caller) bool {
	if cr == nil || cr.Status != types.DocCRStatusPendingReview {
		return false
	}
	moderator := canModerateDoc(ctx, kb)
	assigned := false
	if reviewers, err := s.repo.ListDocCRReviewers(ctx, kb.TenantID, cr.ID); err == nil {
		for _, reviewer := range reviewers {
			if reviewer.UserID == caller.UserID {
				assigned = true
				break
			}
		}
	}
	auditor := false
	if s.departmentRepo != nil && kb.DepartmentID != "" {
		auditor, _ = s.departmentRepo.IsCRAuditor(ctx, kb.TenantID, kb.DepartmentID, caller.UserID)
	}
	if cr.CreatedBy == caller.UserID {
		return false
	}
	return moderator || assigned || auditor
}

func hasApprovalFromOther(reviewers []*types.DocCRReviewer, submitterID string) bool {
	for _, reviewer := range reviewers {
		if reviewer == nil {
			continue
		}
		if reviewer.UserID == submitterID {
			continue
		}
		if reviewer.Status == types.DocCRReviewerApproved {
			return true
		}
	}
	return false
}

func (s *collaborationService) ResubmitDocChangeRequest(
	ctx context.Context, kbID string, crID string, req *types.CreateDocCRRequest,
) (*types.DocChangeRequest, error) {
	caller, err := collaborationActor(ctx)
	if err != nil {
		return nil, err
	}
	kb, err := s.getKB(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if err := ensureDocEditor(ctx, kb); err != nil {
		return nil, err
	}
	cr, err := s.repo.GetDocChangeRequest(ctx, kb.TenantID, kb.ID, crID)
	if err != nil {
		if errors.Is(err, repository.ErrDocCRNotFound) {
			return nil, werrors.NewNotFoundError("变更请求不存在")
		}
		return nil, err
	}
	if cr.CreatedBy != caller.UserID {
		return nil, werrors.NewForbiddenError("仅提交者可重新提交该变更请求")
	}
	if cr.Status != types.DocCRStatusRejected {
		return nil, werrors.NewValidationError("仅已驳回的变更请求可重新提交")
	}
	knowledge, err := s.loadKnowledgeForKB(ctx, kb, cr.KnowledgeID)
	if err != nil {
		return nil, err
	}
	if req.KnowledgeID != "" && req.KnowledgeID != cr.KnowledgeID {
		return nil, werrors.NewValidationError("变更请求不可更换文档")
	}
	title, err := normalizeDocTitle(req.Title)
	if err != nil {
		return nil, err
	}
	items, err := s.normalizeCRItems(ctx, kb, knowledge, req.Items)
	if err != nil {
		return nil, err
	}
	cr.Title = title
	cr.Description = strings.TrimSpace(req.Description)
	cr.Tags = normalizeCRTags(req.Tags)
	cr.UpdatedAt = time.Now()
	cr.Items = items
	if s.versions != nil {
		if latest, versionErr := s.versions.EnsureInitialVersion(ctx, kb.TenantID, kb.ID, cr.KnowledgeID, caller.UserID); versionErr == nil && latest != nil {
			cr.BaseVersionID = latest.ID
		}
	}
	if err := s.repo.ResubmitDocCR(ctx, cr); err != nil {
		if errors.Is(err, repository.ErrDocCRNotOpen) {
			return nil, werrors.NewValidationError("变更请求状态已变化，请刷新后重试")
		}
		return nil, err
	}
	// Re-submission invalidates every prior approval; reviewers are rebuilt
	// (creator may reassign) and decisions start at pending again.
	if len(req.ReviewerIDs) > 0 {
		reviewers, buildErr := s.buildCRReviewers(ctx, kb, cr.ID, req.ReviewerIDs, cr.UpdatedAt)
		if buildErr != nil {
			return nil, buildErr
		}
		if err := s.repo.ReplaceDocCRReviewers(ctx, kb.TenantID, cr.ID, reviewers); err != nil {
			return nil, err
		}
	} else if existing, listErr := s.repo.ListDocCRReviewers(ctx, kb.TenantID, cr.ID); listErr == nil {
		for _, reviewer := range existing {
			reviewer.Status = types.DocCRReviewerPending
			reviewer.Comment = ""
			reviewer.DecidedAt = nil
			reviewer.UpdatedAt = cr.UpdatedAt
			if err := s.repo.UpdateDocCRReviewer(ctx, reviewer); err != nil {
				return nil, err
			}
		}
	}
	chunkIDs := make([]string, 0, len(items))
	for _, item := range items {
		chunkIDs = append(chunkIDs, item.ChunkID)
	}
	linkedIDs, linkErr := s.resolveCRLinkedIssueIDs(ctx, kb, cr.KnowledgeID, chunkIDs, req.LinkedIssueIDs)
	if linkErr != nil {
		return nil, linkErr
	}
	if len(linkedIDs) > 0 {
		if err := s.repo.ReplaceDocCRLinkedIssues(ctx, kb.TenantID, cr.ID, linkedIDs); err != nil {
			return nil, err
		}
	}
	return s.GetDocChangeRequest(ctx, kb.ID, cr.ID)
}

func (s *collaborationService) ReviewDocChangeRequest(
	ctx context.Context, kbID string, crID string, req *types.ReviewDocCRRequest,
) (*types.DocChangeRequest, error) {
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
	cr, err := s.repo.GetDocChangeRequest(ctx, kb.TenantID, kb.ID, crID)
	if err != nil {
		if errors.Is(err, repository.ErrDocCRNotFound) {
			return nil, werrors.NewNotFoundError("变更请求不存在")
		}
		return nil, err
	}
	if err := s.ensureKnowledgeACLByID(ctx, kb, cr.KnowledgeID); err != nil {
		return nil, err
	}
	if cr.Status != types.DocCRStatusPendingReview {
		return nil, werrors.NewValidationError("该变更请求当前不可审阅")
	}
	action := strings.TrimSpace(req.Action)
	comment := strings.TrimSpace(req.Comment)
	if action == "reject" {
		action = types.DocCRReviewRequestChanges
	}
	if action != types.DocCRReviewApprove && action != types.DocCRReviewRequestChanges {
		return nil, werrors.NewValidationError("无效的审批动作")
	}
	if action == types.DocCRReviewRequestChanges && comment == "" {
		return nil, werrors.NewValidationError("要求修改时必须填写审批意见")
	}

	reviewers, err := s.repo.ListDocCRReviewers(ctx, kb.TenantID, cr.ID)
	if err != nil {
		return nil, err
	}
	assigned := false
	var mine *types.DocCRReviewer
	for _, reviewer := range reviewers {
		if reviewer.UserID == caller.UserID {
			assigned = true
			mine = reviewer
			break
		}
	}
	moderator := canModerateDoc(ctx, kb)
	auditor := false
	if s.departmentRepo != nil && kb.DepartmentID != "" {
		auditor, _ = s.departmentRepo.IsCRAuditor(ctx, kb.TenantID, kb.DepartmentID, caller.UserID)
	}
	if !assigned && !moderator && !auditor {
		return nil, werrors.NewForbiddenError("仅指定审阅者、部门审计员或管理员可审批")
	}
	if cr.CreatedBy == caller.UserID {
		return nil, werrors.NewForbiddenError("不能审批自己提交的变更请求")
	}
	now := time.Now()
	if mine == nil {
		mine = &types.DocCRReviewer{
			ID:        uuid.NewString(),
			TenantID:  cr.TenantID,
			CRID:      cr.ID,
			UserID:    caller.UserID,
			CreatedAt: now,
		}
		reviewers = append(reviewers, mine)
	}
	if action == types.DocCRReviewApprove {
		mine.Status = types.DocCRReviewerApproved
	} else {
		mine.Status = types.DocCRReviewerChangesRequested
	}
	mine.Comment = comment
	mine.DecidedAt = &now
	mine.UpdatedAt = now
	if err := s.repo.ReplaceDocCRReviewers(ctx, kb.TenantID, cr.ID, reviewers); err != nil {
		return nil, err
	}
	historyAction := types.DocCRReviewAccept
	if action == types.DocCRReviewRequestChanges {
		historyAction = types.DocCRReviewReject
	}
	if err := s.repo.CreateDocCRReviewHistory(ctx, &types.DocCRReview{
		ID:         uuid.NewString(),
		TenantID:   cr.TenantID,
		CRID:       cr.ID,
		Action:     historyAction,
		Comment:    comment,
		ReviewedBy: caller.UserID,
		CreatedAt:  now,
	}); err != nil {
		return nil, err
	}
	if action == types.DocCRReviewRequestChanges {
		cr.Status = types.DocCRStatusChangesRequested
		cr.ReviewedBy = caller.UserID
		cr.ReviewedAt = &now
		cr.ReviewComment = comment
		cr.UpdatedAt = now
		if err := s.repo.UpdateDocCR(ctx, cr); err != nil {
			return nil, err
		}
	}
	if s.audit != nil {
		details, _ := json.Marshal(map[string]any{
			"decision":          action,
			"knowledge_base_id": kb.ID,
			"knowledge_id":      cr.KnowledgeID,
		})
		if err := s.audit.Log(ctx, &types.AuditLog{
			TenantID:    cr.TenantID,
			ActorUserID: caller.UserID,
			Action:      types.AuditActionDocCRReviewed,
			TargetType:  "doc_change_request",
			TargetID:    cr.ID,
			Outcome:     types.AuditOutcomeSuccess,
			Details:     types.JSON(details),
		}); err != nil {
			logger.Warnf(ctx, "collaboration: audit CR review %s failed: %v", cr.ID, err)
		}
	}
	return s.GetDocChangeRequest(ctx, kb.ID, cr.ID)
}

func (s *collaborationService) MergeDocChangeRequest(
	ctx context.Context, kbID string, crID string, req *types.MergeDocCRRequest,
) (*types.DocChangeRequest, error) {
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
	cr, err := s.repo.GetDocChangeRequest(ctx, kb.TenantID, kb.ID, crID)
	if err != nil {
		if errors.Is(err, repository.ErrDocCRNotFound) {
			return nil, werrors.NewNotFoundError("变更请求不存在")
		}
		return nil, err
	}
	if err := s.ensureKnowledgeACLByID(ctx, kb, cr.KnowledgeID); err != nil {
		return nil, err
	}
	if cr.Status != types.DocCRStatusPendingReview {
		return nil, werrors.NewValidationError("仅待审阅的变更请求可合并")
	}
	if !canModerateDoc(ctx, kb) {
		return nil, werrors.NewForbiddenError("仅部门负责人可合并变更请求")
	}
	reviewers, err := s.repo.ListDocCRReviewers(ctx, kb.TenantID, cr.ID)
	if err != nil {
		return nil, err
	}
	if !hasApprovalFromOther(reviewers, cr.CreatedBy) {
		return nil, werrors.NewValidationError("合并前需要至少一名其他审阅者或部门审计员批准")
	}
	if cr.AnchorOrphaned {
		return nil, werrors.NewValidationError("分块锚点已失效，无法合并")
	}
	now := time.Now()
	chunks, revisions, allowedIssueIDs, err := s.prepareCRAcceptance(ctx, kb, cr, caller.UserID, now)
	if err != nil {
		return nil, err
	}
	chunkIDs := make([]string, 0, len(cr.Items))
	for _, item := range cr.Items {
		chunkIDs = append(chunkIDs, item.ChunkID)
	}
	allowedIssueIDs = s.expandAllowedIssueIDs(ctx, kb, cr, allowedIssueIDs, chunkIDs)
	closeIDs, err := filterCRCloseIssueIDs(req.CloseIssueIDs, allowedIssueIDs)
	if err != nil {
		return nil, err
	}
	version, err := s.newCRMergeVersion(ctx, kb, cr, caller.UserID, now)
	if err != nil {
		return nil, err
	}
	cr.Status = types.DocCRStatusMerged
	cr.ReviewedBy = caller.UserID
	cr.ReviewedAt = &now
	cr.ReviewComment = strings.TrimSpace(req.Comment)
	cr.UpdatedAt = now
	if version != nil {
		cr.MergedVersionID = version.ID
		cr.MergedAt = &now
	}
	tx := &types.DocCRMergeTx{
		CR:                    cr,
		Chunks:                chunks,
		Revisions:             revisions,
		CloseIssueIDs:         closeIDs,
		KnowledgeID:           cr.KnowledgeID,
		Version:               version,
		MarkEngineSyncPending: true,
		Now:                   now,
	}
	if err := s.repo.MergeDocCR(ctx, tx); err != nil {
		if errors.Is(err, repository.ErrDocCRRevisionStale) {
			return nil, werrors.NewValidationError("分块内容已被他人修改，请刷新后重新提交")
		}
		if errors.Is(err, repository.ErrDocCRNotOpen) {
			return nil, werrors.NewValidationError("该变更请求状态已变化")
		}
		return nil, err
	}
	return s.GetDocChangeRequest(ctx, kb.ID, cr.ID)
}

func (s *collaborationService) CloseDocChangeRequest(
	ctx context.Context, kbID string, crID string, comment string,
) (*types.DocChangeRequest, error) {
	caller, err := collaborationActor(ctx)
	if err != nil {
		return nil, err
	}
	kb, err := s.getKB(ctx, kbID)
	if err != nil {
		return nil, err
	}
	cr, err := s.repo.GetDocChangeRequest(ctx, kb.TenantID, kb.ID, crID)
	if err != nil {
		if errors.Is(err, repository.ErrDocCRNotFound) {
			return nil, werrors.NewNotFoundError("变更请求不存在")
		}
		return nil, err
	}
	if err := s.ensureKnowledgeACLByID(ctx, kb, cr.KnowledgeID); err != nil {
		return nil, err
	}
	if !canModerateDoc(ctx, kb) && cr.CreatedBy != caller.UserID {
		return nil, werrors.NewForbiddenError("仅提交者或管理员可关闭变更请求")
	}
	if cr.Status == types.DocCRStatusMerged || cr.Status == types.DocCRStatusClosed {
		return nil, werrors.NewValidationError("该变更请求已结束")
	}
	now := time.Now()
	cr.Status = types.DocCRStatusClosed
	cr.ClosedBy = caller.UserID
	cr.ClosedAt = &now
	cr.ReviewComment = strings.TrimSpace(comment)
	cr.UpdatedAt = now
	if err := s.repo.UpdateDocCR(ctx, cr); err != nil {
		return nil, err
	}
	return s.GetDocChangeRequest(ctx, kb.ID, cr.ID)
}

func (s *collaborationService) UpdateDocCRDraft(
	ctx context.Context, kbID string, crID string, req *types.UpdateDocCRDraftRequest,
) (*types.DocChangeRequest, error) {
	caller, err := collaborationActor(ctx)
	if err != nil {
		return nil, err
	}
	kb, err := s.getKB(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if err := ensureDocEditor(ctx, kb); err != nil {
		return nil, err
	}
	cr, err := s.repo.GetDocChangeRequest(ctx, kb.TenantID, kb.ID, crID)
	if err != nil {
		if errors.Is(err, repository.ErrDocCRNotFound) {
			return nil, werrors.NewNotFoundError("变更请求不存在")
		}
		return nil, err
	}
	if cr.CreatedBy != caller.UserID || cr.Status != types.DocCRStatusDraft {
		return nil, werrors.NewForbiddenError("仅草稿创建者可保存草稿")
	}
	knowledge, err := s.loadKnowledgeForKB(ctx, kb, cr.KnowledgeID)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(req.Title)
	if title != "" {
		title, err = normalizeDocTitle(title)
		if err != nil {
			return nil, err
		}
	}
	items, err := s.normalizeCRDraftItems(ctx, kb, knowledge, req.Items)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	cr.Title = title
	cr.Description = strings.TrimSpace(req.Description)
	cr.Tags = normalizeCRTags(req.Tags)
	cr.Items = items
	cr.DraftUpdatedAt = &now
	cr.UpdatedAt = now
	if s.versions != nil {
		if latest, versionErr := s.versions.EnsureInitialVersion(ctx, kb.TenantID, kb.ID, cr.KnowledgeID, caller.UserID); versionErr == nil && latest != nil {
			cr.BaseVersionID = latest.ID
		}
	}
	if err := s.repo.UpdateDocCRDraft(ctx, cr); err != nil {
		return nil, err
	}
	if len(req.ReviewerIDs) > 0 {
		reviewers, buildErr := s.buildCRReviewers(ctx, kb, cr.ID, req.ReviewerIDs, now)
		if buildErr != nil {
			return nil, buildErr
		}
		if err := s.repo.ReplaceDocCRReviewers(ctx, kb.TenantID, cr.ID, reviewers); err != nil {
			return nil, err
		}
	}
	chunkIDs := make([]string, 0, len(items))
	for _, item := range items {
		chunkIDs = append(chunkIDs, item.ChunkID)
	}
	linkedIDs, linkErr := s.resolveCRLinkedIssueIDs(ctx, kb, cr.KnowledgeID, chunkIDs, req.LinkedIssueIDs)
	if linkErr != nil {
		return nil, linkErr
	}
	if len(linkedIDs) > 0 {
		if err := s.repo.ReplaceDocCRLinkedIssues(ctx, kb.TenantID, cr.ID, linkedIDs); err != nil {
			return nil, err
		}
	}
	return s.GetDocChangeRequest(ctx, kb.ID, cr.ID)
}

func (s *collaborationService) SubmitDocCRDraft(
	ctx context.Context, kbID string, crID string, req *types.SubmitDocCRDraftRequest,
) (*types.DocChangeRequest, error) {
	caller, err := collaborationActor(ctx)
	if err != nil {
		return nil, err
	}
	kb, err := s.getKB(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if err := ensureDocEditor(ctx, kb); err != nil {
		return nil, err
	}
	cr, err := s.repo.GetDocChangeRequest(ctx, kb.TenantID, kb.ID, crID)
	if err != nil {
		if errors.Is(err, repository.ErrDocCRNotFound) {
			return nil, werrors.NewNotFoundError("变更请求不存在")
		}
		return nil, err
	}
	if cr.CreatedBy != caller.UserID || cr.Status != types.DocCRStatusDraft {
		return nil, werrors.NewForbiddenError("仅草稿创建者可提交草稿")
	}
	knowledge, err := s.loadKnowledgeForKB(ctx, kb, cr.KnowledgeID)
	if err != nil {
		return nil, err
	}
	title, err := normalizeDocTitle(req.Title)
	if err != nil {
		return nil, err
	}
	items, err := s.normalizeCRItems(ctx, kb, knowledge, req.Items)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	cr.Title = title
	cr.Description = strings.TrimSpace(req.Description)
	cr.Tags = normalizeCRTags(req.Tags)
	cr.Items = items
	cr.Status = types.DocCRStatusPendingReview
	cr.DraftUpdatedAt = nil
	cr.UpdatedAt = now
	if s.versions != nil {
		if latest, versionErr := s.versions.EnsureInitialVersion(ctx, kb.TenantID, kb.ID, cr.KnowledgeID, caller.UserID); versionErr == nil && latest != nil {
			cr.BaseVersionID = latest.ID
		}
	}
	if err := s.repo.UpdateDocCRDraft(ctx, cr); err != nil {
		return nil, err
	}
	if len(req.ReviewerIDs) > 0 {
		reviewers, buildErr := s.buildCRReviewers(ctx, kb, cr.ID, req.ReviewerIDs, now)
		if buildErr != nil {
			return nil, buildErr
		}
		if err := s.repo.ReplaceDocCRReviewers(ctx, kb.TenantID, cr.ID, reviewers); err != nil {
			return nil, err
		}
	} else if existing, listErr := s.repo.ListDocCRReviewers(ctx, kb.TenantID, cr.ID); listErr == nil {
		for _, reviewer := range existing {
			reviewer.Status = types.DocCRReviewerPending
			reviewer.Comment = ""
			reviewer.DecidedAt = nil
			reviewer.UpdatedAt = now
			if err := s.repo.UpdateDocCRReviewer(ctx, reviewer); err != nil {
				return nil, err
			}
		}
	}
	chunkIDs := make([]string, 0, len(items))
	for _, item := range items {
		chunkIDs = append(chunkIDs, item.ChunkID)
	}
	linkedIDs, linkErr := s.resolveCRLinkedIssueIDs(ctx, kb, cr.KnowledgeID, chunkIDs, req.LinkedIssueIDs)
	if linkErr != nil {
		return nil, linkErr
	}
	if len(linkedIDs) > 0 {
		if err := s.repo.ReplaceDocCRLinkedIssues(ctx, kb.TenantID, cr.ID, linkedIDs); err != nil {
			return nil, err
		}
	}
	return s.GetDocChangeRequest(ctx, kb.ID, cr.ID)
}

func (s *collaborationService) UpdateDocCRReviewers(
	ctx context.Context, kbID string, crID string, req *types.UpdateDocCRReviewersRequest,
) (*types.DocChangeRequest, error) {
	caller, err := collaborationActor(ctx)
	if err != nil {
		return nil, err
	}
	kb, err := s.getKB(ctx, kbID)
	if err != nil {
		return nil, err
	}
	cr, err := s.repo.GetDocChangeRequest(ctx, kb.TenantID, kb.ID, crID)
	if err != nil {
		if errors.Is(err, repository.ErrDocCRNotFound) {
			return nil, werrors.NewNotFoundError("变更请求不存在")
		}
		return nil, err
	}
	if err := s.ensureKnowledgeACLByID(ctx, kb, cr.KnowledgeID); err != nil {
		return nil, err
	}
	if !canModerateDoc(ctx, kb) && cr.CreatedBy != caller.UserID {
		return nil, werrors.NewForbiddenError("仅空间管理员或提交者可调整审阅者")
	}
	if cr.Status == types.DocCRStatusMerged || cr.Status == types.DocCRStatusClosed {
		return nil, werrors.NewValidationError("已结束的变更请求不可调整审阅者")
	}
	existing, _ := s.repo.ListDocCRReviewers(ctx, kb.TenantID, cr.ID)
	byUser := make(map[string]*types.DocCRReviewer, len(existing))
	for _, reviewer := range existing {
		byUser[reviewer.UserID] = reviewer
	}
	now := time.Now()
	reviewers, err := s.buildCRReviewers(ctx, kb, cr.ID, req.ReviewerIDs, now)
	if err != nil {
		return nil, err
	}
	for _, reviewer := range reviewers {
		if prior, ok := byUser[reviewer.UserID]; ok {
			reviewer.Status = prior.Status
			reviewer.Comment = prior.Comment
			reviewer.DecidedAt = prior.DecidedAt
			reviewer.CreatedAt = prior.CreatedAt
		}
	}
	if err := s.repo.ReplaceDocCRReviewers(ctx, kb.TenantID, cr.ID, reviewers); err != nil {
		return nil, err
	}
	cr.UpdatedAt = now
	_ = s.repo.UpdateDocCR(ctx, cr)
	return s.GetDocChangeRequest(ctx, kb.ID, cr.ID)
}

func (s *collaborationService) MarkDocCRViewed(
	ctx context.Context, kbID string, crID string,
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
	cr, err := s.repo.GetDocChangeRequest(ctx, kb.TenantID, kb.ID, crID)
	if err != nil {
		return err
	}
	if err := s.ensureKnowledgeACLByID(ctx, kb, cr.KnowledgeID); err != nil {
		return err
	}
	return s.repo.MarkDocCRItemViewed(ctx, kb.TenantID, cr.ID, caller.UserID)
}

func (s *collaborationService) expandAllowedIssueIDs(
	ctx context.Context, kb *types.KnowledgeBase, cr *types.DocChangeRequest,
	allowed map[string]struct{}, chunkIDs []string,
) map[string]struct{} {
	out := allowed
	if out == nil {
		out = make(map[string]struct{})
	}
	for _, issue := range cr.LinkedIssues {
		if issue != nil && issue.Status == types.DocIssueStatusOpen {
			out[issue.ID] = struct{}{}
		}
	}
	if explicit, err := s.repo.ListDocCRLinkedIssues(ctx, kb.TenantID, cr.ID); err == nil {
		for _, issue := range explicit {
			if issue != nil && issue.Status == types.DocIssueStatusOpen {
				out[issue.ID] = struct{}{}
			}
		}
	}
	return out
}

func (s *collaborationService) newCRMergeVersion(
	ctx context.Context, kb *types.KnowledgeBase, cr *types.DocChangeRequest, actorID string, now time.Time,
) (*types.DocVersion, error) {
	if s.versions == nil {
		return nil, nil
	}
	latest, err := s.versions.EnsureInitialVersion(ctx, kb.TenantID, kb.ID, cr.KnowledgeID, actorID)
	if err != nil {
		return nil, err
	}
	seq := 0
	if latest != nil {
		seq = latest.VersionSeq + 1
	}
	return &types.DocVersion{
		ID:              uuid.NewString(),
		TenantID:        kb.TenantID,
		KnowledgeBaseID: kb.ID,
		KnowledgeID:     cr.KnowledgeID,
		VersionSeq:      seq,
		VersionLabel:    "V1." + strconv.Itoa(seq),
		SourceType:      types.DocVersionSourceCRMerge,
		SourceID:        cr.ID,
		Title:           cr.Title,
		Description:     cr.Description,
		CreatedBy:       actorID,
		CreatedAt:       now,
	}, nil
}

func (s *collaborationService) prepareCRAcceptance(
	ctx context.Context, kb *types.KnowledgeBase, cr *types.DocChangeRequest, editorID string, now time.Time,
) ([]*types.Chunk, []*types.ChunkRevision, map[string]struct{}, error) {
	chunkIDs := make([]string, 0, len(cr.Items))
	for _, item := range cr.Items {
		chunkIDs = append(chunkIDs, item.ChunkID)
	}
	chunkMap := s.chunkMap(ctx, kb.TenantID, chunkIDs)
	chunks := make([]*types.Chunk, 0, len(cr.Items))
	revisions := make([]*types.ChunkRevision, 0, len(cr.Items))
	allowedIssueIDs := make(map[string]struct{})
	for _, item := range cr.Items {
		chunk, ok := chunkMap[item.ChunkID]
		if !ok {
			return nil, nil, nil, werrors.NewValidationError("分块锚点已失效，无法接受，请驳回后重新提交")
		}
		if chunk.Content != item.OriginalContent {
			return nil, nil, nil, werrors.NewValidationError("分块内容已被修改，请驳回后重新提交")
		}
		revisions = append(revisions, &types.ChunkRevision{
			ID:              uuid.NewString(),
			TenantID:        chunk.TenantID,
			KnowledgeBaseID: chunk.KnowledgeBaseID,
			KnowledgeID:     chunk.KnowledgeID,
			ChunkID:         chunk.ID,
			Revision:        chunk.ContentRevision,
			Content:         chunk.Content,
			IsEnabled:       chunk.IsEnabled,
			EditorID:        chunk.LastEditorID,
			EditSource:      "user",
			EditedAt:        chunk.UpdatedAt,
			CreatedAt:       now,
		})
		chunk.Content = item.SuggestedContent
		chunk.ContentRevision++
		chunk.LastEditorID = editorID
		chunk.UpdatedAt = now
		chunks = append(chunks, chunk)
	}
	issues, err := s.repo.ListOpenDocIssuesByChunkIDs(ctx, kb.TenantID, kb.ID, chunkIDs)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, issue := range issues {
		allowedIssueIDs[issue.ID] = struct{}{}
	}
	return chunks, revisions, allowedIssueIDs, nil
}

func filterCRCloseIssueIDs(requested []string, allowed map[string]struct{}) ([]string, error) {
	if len(requested) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{}, len(requested))
	out := make([]string, 0, len(requested))
	for _, id := range requested {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := allowed[id]; !ok {
			return nil, werrors.NewValidationError("包含不属于该变更请求关联分块的问题")
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

func (s *collaborationService) ensureCommentTarget(
	ctx context.Context, kb *types.KnowledgeBase, targetType string, targetID string,
) error {
	switch targetType {
	case types.DocCommentTargetIssue:
		issue, err := s.repo.GetDocIssue(ctx, kb.TenantID, kb.ID, targetID)
		if err != nil {
			if errors.Is(err, repository.ErrDocIssueNotFound) {
				return werrors.NewNotFoundError("问题不存在")
			}
			return err
		}
		return s.ensureKnowledgeACLByID(ctx, kb, issue.KnowledgeID)
	case types.DocCommentTargetCR:
		cr, err := s.repo.GetDocChangeRequest(ctx, kb.TenantID, kb.ID, targetID)
		if err != nil {
			if errors.Is(err, repository.ErrDocCRNotFound) {
				return werrors.NewNotFoundError("变更请求不存在")
			}
			return err
		}
		return s.ensureKnowledgeACLByID(ctx, kb, cr.KnowledgeID)
	default:
		return werrors.NewValidationError("无效的评论目标")
	}
}

func (s *collaborationService) ListDocComments(
	ctx context.Context, kbID string, targetType string, targetID string,
) ([]*types.DocComment, error) {
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
	if err := s.ensureCommentTarget(ctx, kb, targetType, targetID); err != nil {
		return nil, err
	}
	comments, err := s.repo.ListDocComments(ctx, kb.TenantID, kb.ID, targetType, targetID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(comments))
	for _, comment := range comments {
		ids = append(ids, comment.CreatedBy)
	}
	names := s.userNameMap(ctx, ids)
	for _, comment := range comments {
		if comment.DeletedAt != nil {
			comment.Deleted = true
			comment.Body = ""
			comment.CreatedByName = ""
			continue
		}
		comment.CreatedByName = names[comment.CreatedBy]
	}
	return comments, nil
}

func (s *collaborationService) CreateDocComment(
	ctx context.Context, kbID string, req *types.CreateDocCommentRequest,
) (*types.DocComment, error) {
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
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return nil, werrors.NewValidationError("评论内容不能为空")
	}
	if len([]rune(body)) > docCommentMaxLen {
		return nil, werrors.NewValidationError("评论内容过长")
	}
	if err := s.ensureCommentTarget(ctx, kb, req.TargetType, req.TargetID); err != nil {
		return nil, err
	}
	comment := &types.DocComment{
		ID:              uuid.NewString(),
		TenantID:        kb.TenantID,
		KnowledgeBaseID: kb.ID,
		TargetType:      req.TargetType,
		TargetID:        req.TargetID,
		Body:            body,
		CreatedBy:       caller.UserID,
		CreatedAt:       time.Now(),
	}
	if err := s.repo.CreateDocComment(ctx, comment); err != nil {
		return nil, err
	}
	comment.CreatedByName = s.userNameMap(ctx, []string{comment.CreatedBy})[comment.CreatedBy]
	return comment, nil
}

func (s *collaborationService) DeleteDocComment(
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
	comment, err := s.repo.GetDocComment(ctx, kb.TenantID, kb.ID, commentID)
	if err != nil {
		if errors.Is(err, repository.ErrDocCommentNotFound) {
			return werrors.NewNotFoundError("评论不存在")
		}
		return err
	}
	if comment.DeletedAt != nil {
		return nil
	}
	if comment.CreatedBy != caller.UserID && !canModerateDoc(ctx, kb) {
		return werrors.NewForbiddenError("无权删除该评论")
	}
	now := time.Now()
	comment.DeletedBy = caller.UserID
	comment.DeletedAt = &now
	return s.repo.UpdateDocComment(ctx, comment)
}

func (s *collaborationService) ListDocCollaborations(
	ctx context.Context, req *types.DocCollaborationListRequest,
) (*types.DocCollaborationListResponse, error) {
	caller, err := collaborationActor(ctx)
	if err != nil {
		return nil, err
	}
	kb, err := s.getKB(ctx, req.KnowledgeBaseID)
	if err != nil {
		return nil, err
	}
	if err := ensureDocRead(ctx, kb); err != nil {
		return nil, err
	}
	req.TenantID = kb.TenantID
	req.VisibilityUserID = caller.UserID
	if scope, err := s.aclScope(ctx, kb); err != nil {
		return nil, err
	} else if scope != nil && scope.Active {
		req.ACLActive = true
		req.AllowedKnowledgeIDs = scope.AllowedKnowledgeIDs
	}
	items, total, err := s.repo.ListCollaborations(ctx, req)
	if err != nil {
		return nil, err
	}
	userIDs := make([]string, 0, len(items))
	chunkIDs := make([]string, 0, len(items))
	for _, item := range items {
		userIDs = append(userIDs, item.CreatedBy)
		if item.ChunkID != "" {
			chunkIDs = append(chunkIDs, item.ChunkID)
		}
	}
	names := s.userNameMap(ctx, userIDs)
	chunks := s.chunkMap(ctx, kb.TenantID, chunkIDs)
	for _, item := range items {
		item.CreatedByName = names[item.CreatedBy]
		if item.ChunkID != "" {
			if _, ok := chunks[item.ChunkID]; !ok {
				item.AnchorOrphaned = true
			}
		}
	}
	page, pageSize := normalizeDocPagination(req.Page, req.PageSize)
	return &types.DocCollaborationListResponse{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func normalizeDocPagination(page, pageSize int) (int, int) {
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
