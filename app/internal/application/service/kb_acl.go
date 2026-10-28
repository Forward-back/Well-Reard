package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"gaze/internal/application/access"
	"gaze/internal/application/repository"
	"gaze/internal/types"
	"gaze/internal/types/interfaces"
)

var (
	ErrACLForbidden     = errors.New("acl change requires department manager")
	ErrACLInvalidRule   = errors.New("invalid acl rule")
	ErrACLFileAllow     = errors.New("file acl rules can only deny")
	ErrACLRuleNotFound  = errors.New("acl rule not found")
	ErrACLAlreadyExists = errors.New("acl rule already exists")
	ErrACLDepartment    = errors.New("department acl subject not found")
)

type kbAclService struct {
	repo          interfaces.KBAclRepository
	kbService     interfaces.KnowledgeBaseService
	folderRepo    interfaces.KnowledgeFolderRepository
	knowledgeRepo interfaces.KnowledgeRepository
	deptRepo      interfaces.DepartmentRepository
	audit         interfaces.AuditLogService
}

func NewKBAclService(
	repo interfaces.KBAclRepository,
	kbService interfaces.KnowledgeBaseService,
	folderRepo interfaces.KnowledgeFolderRepository,
	knowledgeRepo interfaces.KnowledgeRepository,
	deptRepo interfaces.DepartmentRepository,
	audit interfaces.AuditLogService,
) interfaces.KBAclService {
	return &kbAclService{
		repo:          repo,
		kbService:     kbService,
		folderRepo:    folderRepo,
		knowledgeRepo: knowledgeRepo,
		deptRepo:      deptRepo,
		audit:         audit,
	}
}

type aclKB struct {
	kb *types.KnowledgeBase
}

func (s *kbAclService) loadKB(ctx context.Context, kbID string) (*aclKB, error) {
	kb, err := s.kbService.GetKnowledgeBaseByID(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if kb == nil || kb.ID != kbID {
		return nil, repository.ErrFolderNotFound
	}
	return &aclKB{kb: kb}, nil
}

func (s *kbAclService) authorize(ctx context.Context, kbID string) (*types.KnowledgeBase, error) {
	loaded, err := s.loadKB(ctx, kbID)
	if err != nil {
		return nil, err
	}
	kb := loaded.kb
	if err := access.RequireKBWrite(ctx, kb); err != nil {
		return nil, err
	}
	caller := types.CallerFromContext(ctx)
	if caller.DepartmentID == "" || caller.DepartmentID != kb.DepartmentID {
		return nil, ErrACLForbidden
	}
	if caller.DepartmentRole != types.DepartmentRoleManager &&
		caller.DepartmentRole != types.DepartmentRoleKnowledgeAdmin {
		return nil, ErrACLForbidden
	}
	return kb, nil
}

func (s *kbAclService) List(ctx context.Context, kbID string) ([]*types.KBAclRule, int64, error) {
	loaded, err := s.loadKB(ctx, kbID)
	if err != nil {
		return nil, 0, err
	}
	rules, err := s.repo.ListByKB(ctx, loaded.kb.TenantID, kbID)
	if err != nil {
		return nil, 0, err
	}
	epoch, err := s.repo.GetEpoch(ctx, loaded.kb.TenantID)
	if err != nil {
		return nil, 0, err
	}
	return rules, epoch, nil
}

func (s *kbAclService) validateRule(ctx context.Context, kb *types.KnowledgeBase, req types.KBAclRuleRequest) (*types.KBAclRule, error) {
	req.ResourceID = strings.TrimSpace(req.ResourceID)
	req.SubjectID = strings.TrimSpace(req.SubjectID)
	if req.ResourceID == "" || !req.Effect.Valid() || !req.SubjectType.Valid() {
		return nil, ErrACLInvalidRule
	}
	switch req.ResourceType {
	case types.KBAclResourceFolder:
		folder, err := s.folderRepo.GetByID(ctx, kb.TenantID, kb.ID, req.ResourceID)
		if err != nil {
			return nil, err
		}
		if folder == nil {
			return nil, ErrACLInvalidRule
		}
	case types.KBAclResourceFile:
		if s.knowledgeRepo != nil {
			knowledge, err := s.knowledgeRepo.GetKnowledgeByIDOnly(ctx, req.ResourceID)
			if err != nil || knowledge == nil || knowledge.KnowledgeBaseID != kb.ID {
				return nil, ErrACLInvalidRule
			}
		}
		if req.Effect == types.KBAclEffectAllow {
			return nil, ErrACLFileAllow
		}
	default:
		return nil, ErrACLInvalidRule
	}
	switch req.SubjectType {
	case types.KBAclSubjectUser:
		if req.SubjectID == "" {
			return nil, ErrACLInvalidRule
		}
	case types.KBAclSubjectDepartment:
		if req.SubjectID == "" || s.deptRepo == nil {
			return nil, ErrACLDepartment
		}
		dept, err := s.deptRepo.Get(ctx, kb.TenantID, req.SubjectID)
		if err != nil {
			return nil, err
		}
		if dept == nil {
			return nil, ErrACLDepartment
		}
	case types.KBAclSubjectAll:
		req.SubjectID = ""
	default:
		return nil, ErrACLInvalidRule
	}
	caller := types.CallerFromContext(ctx)
	if caller.DepartmentRole != types.DepartmentRoleManager {
		// Knowledge admins may only add user/department deny rules.
		if req.Effect != types.KBAclEffectDeny ||
			(req.SubjectType != types.KBAclSubjectUser && req.SubjectType != types.KBAclSubjectDepartment) {
			return nil, ErrACLForbidden
		}
	}
	createdBy, _ := types.UserIDFromContext(ctx)
	return &types.KBAclRule{
		TenantID:        kb.TenantID,
		KnowledgeBaseID: kb.ID,
		ResourceType:    req.ResourceType,
		ResourceID:      req.ResourceID,
		Effect:          req.Effect,
		SubjectType:     req.SubjectType,
		SubjectID:       req.SubjectID,
		CreatedBy:       createdBy,
	}, nil
}

func (s *kbAclService) Create(ctx context.Context, kbID string, req types.KBAclRuleRequest) (*types.KBAclRule, error) {
	kb, err := s.authorize(ctx, kbID)
	if err != nil {
		return nil, err
	}
	rule, err := s.validateRule(ctx, kb, req)
	if err != nil {
		return nil, err
	}
	var entry *types.AuditLog
	if s.audit != nil {
		raw, _ := json.Marshal(map[string]any{
			"kb_id":        kb.ID,
			"resource":     string(rule.ResourceType),
			"resource_id":  rule.ResourceID,
			"effect":       string(rule.Effect),
			"subject_type": string(rule.SubjectType),
			"subject_id":   rule.SubjectID,
		})
		entry = &types.AuditLog{
			TenantID: kb.TenantID, ActorUserID: rule.CreatedBy,
			Action:     types.AuditAction("acl.rule_created"),
			TargetType: "kb_acl_rule", TargetID: rule.ID,
			Outcome: types.AuditOutcomeSuccess, Details: types.JSON(raw),
		}
	}
	if err := s.repo.CreateWithAudit(ctx, rule, entry); err != nil {
		if errors.Is(err, repository.ErrACLRuleExists) {
			return nil, ErrACLAlreadyExists
		}
		return nil, err
	}
	return rule, nil
}

func (s *kbAclService) Delete(ctx context.Context, kbID, ruleID string) error {
	kb, err := s.authorize(ctx, kbID)
	if err != nil {
		return err
	}
	var entry *types.AuditLog
	if s.audit != nil {
		caller := types.CallerFromContext(ctx)
		entry = &types.AuditLog{
			TenantID:    kb.TenantID,
			ActorUserID: caller.UserID,
			Action:      types.AuditAction("acl.rule_deleted"),
			TargetType:  "kb_acl_rule",
			TargetID:    ruleID,
			Outcome:     types.AuditOutcomeSuccess,
		}
	}
	if err := s.repo.DeleteWithAudit(ctx, kb.TenantID, kb.ID, ruleID, entry); err != nil {
		if errors.Is(err, repository.ErrACLRuleNotFound) {
			return ErrACLRuleNotFound
		}
		return err
	}
	return nil
}

func (s *kbAclService) Epoch(ctx context.Context, kbID string) (int64, error) {
	loaded, err := s.loadKB(ctx, kbID)
	if err != nil {
		return 0, err
	}
	return s.repo.GetEpoch(ctx, loaded.kb.TenantID)
}

func aclSubjectMatches(caller types.Caller, rule *types.KBAclRule) bool {
	if rule == nil {
		return false
	}
	switch rule.SubjectType {
	case types.KBAclSubjectAll:
		return true
	case types.KBAclSubjectUser:
		return rule.SubjectID != "" && rule.SubjectID == caller.UserID
	case types.KBAclSubjectDepartment:
		return rule.SubjectID != "" && rule.SubjectID == caller.DepartmentID
	default:
		return false
	}
}

func aclFolderChain(ctx context.Context, repo interfaces.KnowledgeFolderRepository, kb *types.KnowledgeBase, folderID string) ([]*types.KnowledgeFolder, error) {
	root, err := repo.EnsureRoot(ctx, kb.TenantID, kb.ID, "")
	if err != nil {
		return nil, err
	}
	selected := root
	if strings.TrimSpace(folderID) != "" {
		selected, err = repo.GetByID(ctx, kb.TenantID, kb.ID, strings.TrimSpace(folderID))
		if err != nil {
			return nil, err
		}
		if selected == nil {
			return nil, ErrKnowledgeFolderNotFound
		}
	}
	chain := []*types.KnowledgeFolder{}
	for current := selected; current != nil; {
		chain = append(chain, current)
		if current.ParentID == "" || current.IsRoot {
			break
		}
		current, err = repo.GetByID(ctx, kb.TenantID, kb.ID, current.ParentID)
		if err != nil {
			return nil, err
		}
	}
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain, nil
}

func evaluateFolderRules(
	caller types.Caller,
	rules []*types.KBAclRule,
	chain []*types.KnowledgeFolder,
) (bool, string) {
	byFolder := map[string][]*types.KBAclRule{}
	for _, rule := range rules {
		if rule != nil && rule.ResourceType == types.KBAclResourceFolder {
			byFolder[rule.ResourceID] = append(byFolder[rule.ResourceID], rule)
		}
	}
	for _, folder := range chain {
		hasAllow := false
		allowMatched := false
		for _, rule := range byFolder[folder.ID] {
			if rule.Effect == types.KBAclEffectDeny && aclSubjectMatches(caller, rule) {
				return false, "deny:" + folder.ID
			}
			if rule.Effect == types.KBAclEffectAllow {
				hasAllow = true
				if aclSubjectMatches(caller, rule) {
					allowMatched = true
				}
			}
		}
		if hasAllow && !caller.Role.HasPermission(types.TenantRoleCompanyAdmin) && !allowMatched {
			return false, "allow:" + folder.ID
		}
	}
	return true, ""
}

// RetrievalScope builds the engine-facing scope. Empty kbIDs fail closed
// (allow no documents) rather than expanding to the full corpus.
func (s *kbAclService) RetrievalScope(ctx context.Context, kbIDs []string) (*types.ACLScope, error) {
	caller := types.CallerFromContext(ctx).Normalize()
	epoch, err := s.repo.GetEpoch(ctx, caller.TenantID)
	if err != nil {
		return nil, err
	}
	if len(kbIDs) == 0 {
		return &types.ACLScope{Mode: types.ACLScopeAllow, DocIDs: []string{}, Version: epoch}, nil
	}
	anyActive := false
	docIDs := make(map[string]struct{})
	seenKB := make(map[string]struct{}, len(kbIDs))
	for _, kbID := range kbIDs {
		kbID = strings.TrimSpace(kbID)
		if kbID == "" {
			continue
		}
		if _, ok := seenKB[kbID]; ok {
			continue
		}
		seenKB[kbID] = struct{}{}
		loaded, err := s.loadKB(ctx, kbID)
		if err != nil {
			return nil, err
		}
		scope, err := s.Scope(ctx, kbID)
		if err != nil {
			return nil, err
		}
		if scope.Active {
			anyActive = true
			for _, id := range scope.AllowedKnowledgeIDs {
				docIDs[id] = struct{}{}
			}
			continue
		}
		// A rule-free KB is fully readable, but the combined scope may still
		// need an explicit allow list because another KB in the request has
		// rules. Enumerate its documents now.
		docs, err := s.knowledgeRepo.ListKnowledgeByKnowledgeBaseID(ctx, loaded.kb.TenantID, kbID)
		if err != nil {
			return nil, err
		}
		for _, doc := range docs {
			if doc != nil {
				docIDs[doc.ID] = struct{}{}
			}
		}
	}
	if !anyActive {
		return &types.ACLScope{Mode: types.ACLScopeAll, Version: epoch}, nil
	}
	allowed := make([]string, 0, len(docIDs))
	for id := range docIDs {
		allowed = append(allowed, id)
	}
	return &types.ACLScope{Mode: types.ACLScopeAllow, DocIDs: allowed, Version: epoch}, nil
}

// Evaluate computes the effective ACL decision for one folder/file. KB/department
// access is resolved by the caller's route guard; this method only evaluates the
// in-KB allow/deny chain.
func (s *kbAclService) Evaluate(
	ctx context.Context, kbID, resourceType, resourceID, folderID string,
) (bool, string, error) {
	loaded, err := s.loadKB(ctx, kbID)
	if err != nil {
		return false, "", err
	}
	kb := loaded.kb
	caller := types.CallerFromContext(ctx).Normalize()
	if _, ok := types.TenantAPIKeyScopeFromContext(ctx); ok {
		// API keys are company-level machine principals: bypass folder allow,
		// but never bypass deny.
		caller.Role = types.TenantRoleCompanyAdmin
	}
	rules, err := s.repo.ListByKB(ctx, kb.TenantID, kb.ID)
	if err != nil {
		return false, "", err
	}
	chain, err := aclFolderChain(ctx, s.folderRepo, kb, folderID)
	if err != nil {
		return false, "", err
	}
	allowed, reason := evaluateFolderRules(caller, rules, chain)
	if !allowed {
		return false, reason, nil
	}
	if types.KBAclResourceType(resourceType) == types.KBAclResourceFile {
		for _, rule := range rules {
			if rule != nil && rule.ResourceType == types.KBAclResourceFile &&
				rule.ResourceID == resourceID && rule.Effect == types.KBAclEffectDeny &&
				aclSubjectMatches(caller, rule) {
				return false, "deny:file:" + resourceID, nil
			}
		}
	}
	return true, "", nil
}

// Scope computes the caller's effective read scope for a KB. Active=false
// means no ACL rules apply.
func (s *kbAclService) Scope(ctx context.Context, kbID string) (*types.KBAclScope, error) {
	loaded, err := s.loadKB(ctx, kbID)
	if err != nil {
		return nil, err
	}
	kb := loaded.kb
	rules, err := s.repo.ListByKB(ctx, kb.TenantID, kb.ID)
	if err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		return &types.KBAclScope{Active: false}, nil
	}
	caller := types.CallerFromContext(ctx).Normalize()
	if _, ok := types.TenantAPIKeyScopeFromContext(ctx); ok {
		caller.Role = types.TenantRoleCompanyAdmin
	}
	folders, err := s.folderRepo.ListByKB(ctx, kb.TenantID, kb.ID)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*types.KnowledgeFolder, len(folders))
	rootID := ""
	for _, folder := range folders {
		if folder == nil {
			continue
		}
		byID[folder.ID] = folder
		if folder.IsRoot {
			rootID = folder.ID
		}
	}
	folderAllowed := map[string]bool{}
	var evalFolder func(id string) bool
	evalFolder = func(id string) bool {
		if allowed, ok := folderAllowed[id]; ok {
			return allowed
		}
		chain, err := aclFolderChain(ctx, s.folderRepo, kb, id)
		if err != nil {
			folderAllowed[id] = false
			return false
		}
		allowed, _ := evaluateFolderRules(caller, rules, chain)
		folderAllowed[id] = allowed
		return allowed
	}
	allowedFolders := make([]string, 0, len(folders))
	for _, folder := range folders {
		if folder != nil && evalFolder(folder.ID) {
			allowedFolders = append(allowedFolders, folder.ID)
		}
	}
	documents, err := s.knowledgeRepo.ListKnowledgeByKnowledgeBaseID(ctx, kb.TenantID, kb.ID)
	if err != nil {
		return nil, err
	}
	allowedDocs := make([]string, 0, len(documents))
	counts := make(map[string]int64)
	for _, doc := range documents {
		if doc == nil || !evalFolder(doc.FolderID) {
			continue
		}
		denied := false
		for _, rule := range rules {
			if rule != nil && rule.ResourceType == types.KBAclResourceFile &&
				rule.ResourceID == doc.ID && rule.Effect == types.KBAclEffectDeny &&
				aclSubjectMatches(caller, rule) {
				denied = true
				break
			}
		}
		if denied {
			continue
		}
		allowedDocs = append(allowedDocs, doc.ID)
		counts[doc.FolderID]++
	}
	return &types.KBAclScope{
		Active:               true,
		AllowedKnowledgeIDs:  allowedDocs,
		AllowedFolderIDs:     allowedFolders,
		RootFolderID:         rootID,
		FolderDocumentCounts: counts,
	}, nil
}
