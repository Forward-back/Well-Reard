package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"gaze/internal/application/repository"
	"gaze/internal/types"
	"gaze/internal/types/interfaces"
)

var (
	errGazePageNotFoundInScope = errors.New("gaze page not found in current scope")
	errGazePageAmbiguous       = errors.New("gaze page exists in multiple knowledge bases")
)

// GazeRouteResolver stores request-local provenance for gaze slugs. It is
// server-side routing state, not a model-handle registry: only KBs already in
// the current GazeScope can ever be returned.
type GazeRouteResolver struct {
	mu     sync.RWMutex
	bySlug map[string]map[string]struct{}
}

func NewGazeRouteResolver() *GazeRouteResolver {
	return &GazeRouteResolver{bySlug: make(map[string]map[string]struct{})}
}

func (r *GazeRouteResolver) remember(slug, kbID string) {
	if r == nil || slug == "" || kbID == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	owners := r.bySlug[slug]
	if owners == nil {
		owners = make(map[string]struct{})
		r.bySlug[slug] = owners
	}
	owners[kbID] = struct{}{}
}

func (r *GazeRouteResolver) forget(slug, kbID string) {
	if r == nil || slug == "" || kbID == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	owners := r.bySlug[slug]
	delete(owners, kbID)
	if len(owners) == 0 {
		delete(r.bySlug, slug)
	}
}

func (r *GazeRouteResolver) rememberPage(page *types.GazePage, kbID string) {
	if r == nil || page == nil || kbID == "" {
		return
	}
	r.remember(page.Slug, kbID)
	for _, slug := range page.OutLinks {
		r.remember(slug, kbID)
	}
	for _, slug := range page.InLinks {
		r.remember(slug, kbID)
	}
}

// scopesForSlug returns cached owners that are still present in the current
// server-owned scope. An empty result means the caller should search all
// scopes. Scope order and per-KB document/tag filters are preserved.
func (r *GazeRouteResolver) scopesForSlug(slug string, scopes []GazeScope) []GazeScope {
	if r == nil || slug == "" || len(scopes) == 0 {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	owners := r.bySlug[slug]
	if len(owners) == 0 {
		return nil
	}
	matched := make([]GazeScope, 0, len(owners))
	for _, scope := range scopes {
		if _, ok := owners[scope.KnowledgeBaseID]; ok {
			matched = append(matched, scope)
		}
	}
	return matched
}

func scopesOutsideKBs(scopes []GazeScope, excluded []GazeScope) []GazeScope {
	if len(excluded) == 0 {
		return scopes
	}
	excludedKBs := make(map[string]struct{}, len(excluded))
	for _, scope := range excluded {
		excludedKBs[scope.KnowledgeBaseID] = struct{}{}
	}
	remaining := make([]GazeScope, 0, len(scopes))
	for _, scope := range scopes {
		if _, ok := excludedKBs[scope.KnowledgeBaseID]; !ok {
			remaining = append(remaining, scope)
		}
	}
	return remaining
}

// resolveUniqueGazePage is the shared mutation/issue routing boundary. It
// checks every allowed KB (cached provenance only affects order) and refuses
// ambiguous slugs instead of silently mutating the first KB.
func resolveUniqueGazePage(
	ctx context.Context,
	service interfaces.GazePageService,
	slug string,
	kbIDs []string,
	routes *GazeRouteResolver,
) (*types.GazePage, string, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, "", fmt.Errorf("slug is required")
	}
	scopes := NewGazeScopesFromKBIDs(kbIDs)
	preferred := routes.scopesForSlug(slug, scopes)
	ordered := append(append([]GazeScope(nil), preferred...), scopesOutsideKBs(scopes, preferred)...)
	type hit struct {
		page *types.GazePage
		kbID string
	}
	var hits []hit
	for _, scope := range ordered {
		page, err := service.GetPageBySlug(ctx, scope.KnowledgeBaseID, slug)
		if err != nil {
			if errors.Is(err, repository.ErrGazePageNotFound) {
				continue
			}
			return nil, "", fmt.Errorf(
				"failed to resolve gaze page %s in knowledge base %s: %w",
				slug, scope.KnowledgeBaseID, err,
			)
		}
		if page == nil {
			continue
		}
		if page.KnowledgeBaseID != "" && page.KnowledgeBaseID != scope.KnowledgeBaseID {
			return nil, "", fmt.Errorf(
				"gaze page %s returned knowledge base %s while resolving allowed scope %s",
				slug, page.KnowledgeBaseID, scope.KnowledgeBaseID,
			)
		}
		kbID := scope.KnowledgeBaseID
		hits = append(hits, hit{page: page, kbID: kbID})
		routes.rememberPage(page, kbID)
	}
	switch len(hits) {
	case 0:
		return nil, "", fmt.Errorf("%w: %s", errGazePageNotFoundInScope, slug)
	case 1:
		return hits[0].page, hits[0].kbID, nil
	default:
		owners := make([]string, 0, len(hits))
		for _, item := range hits {
			owners = append(owners, item.kbID)
		}
		return nil, "", fmt.Errorf("%w: slug %s belongs to %s", errGazePageAmbiguous, slug, strings.Join(owners, ", "))
	}
}

// resolveGazeCreateKB selects a creation target only when server-side context
// is unambiguous: one cached provenance owner or one Gaze KB in scope.
func resolveGazeCreateKB(
	slug string,
	kbIDs []string,
	routes *GazeRouteResolver,
	serverHints ...string,
) (string, error) {
	scopes := NewGazeScopesFromKBIDs(kbIDs)
	preferred := routes.scopesForSlug(strings.TrimSpace(slug), scopes)
	allowed := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		allowed[scope.KnowledgeBaseID] = struct{}{}
	}
	candidates := make([]string, 0, len(preferred)+len(serverHints))
	for _, scope := range preferred {
		candidates = append(candidates, scope.KnowledgeBaseID)
	}
	for _, kbID := range serverHints {
		if _, ok := allowed[kbID]; ok {
			candidates = append(candidates, kbID)
		}
	}
	candidates = dedupNonEmptyStrings(candidates)
	if len(candidates) == 1 {
		return candidates[0], nil
	}
	if len(candidates) > 1 {
		return "", fmt.Errorf(
			"cannot choose a knowledge base for new gaze page %s: server provenance conflicts across %s",
			slug, strings.Join(candidates, ", "),
		)
	}
	if len(scopes) == 1 {
		return scopes[0].KnowledgeBaseID, nil
	}
	return "", fmt.Errorf("cannot choose a knowledge base for new gaze page %s from %d allowed scopes", slug, len(scopes))
}

func gazeKnowledgeBasesForSourceRefs(
	ctx context.Context,
	refs []string,
	knowledgeService interfaces.KnowledgeService,
	allowedKBIDs []string,
) ([]string, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	if knowledgeService == nil {
		return nil, fmt.Errorf("knowledge service is unavailable")
	}
	allowed := make(map[string]struct{}, len(allowedKBIDs))
	for _, kbID := range dedupNonEmptyStrings(allowedKBIDs) {
		allowed[kbID] = struct{}{}
	}
	var kbIDs []string
	for _, ref := range refs {
		knowledgeID := strings.TrimSpace(strings.SplitN(ref, "|", 2)[0])
		if knowledgeID == "" {
			continue
		}
		knowledge, err := knowledgeService.GetKnowledgeByIDOnly(ctx, knowledgeID)
		if err != nil || knowledge == nil {
			if err == nil {
				err = fmt.Errorf("empty result")
			}
			return nil, fmt.Errorf("failed to resolve source document %s: %w", knowledgeID, err)
		}
		if _, ok := allowed[knowledge.KnowledgeBaseID]; !ok {
			return nil, fmt.Errorf(
				"source document %s belongs to non-Gaze or unauthorized knowledge base %s",
				knowledgeID, knowledge.KnowledgeBaseID,
			)
		}
		kbIDs = append(kbIDs, knowledge.KnowledgeBaseID)
	}
	return dedupNonEmptyStrings(kbIDs), nil
}

func firstGazeRoute(routes []*GazeRouteResolver) *GazeRouteResolver {
	if len(routes) > 0 && routes[0] != nil {
		return routes[0]
	}
	return NewGazeRouteResolver()
}
