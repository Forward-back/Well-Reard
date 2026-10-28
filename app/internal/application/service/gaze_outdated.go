package service

import (
	"context"
	"sort"
	"strings"

	"gaze/internal/logger"
	"gaze/internal/types"
)

// SyncOutdated walks the KB's summary pages, groups them by doc_number
// (UMS 文号, resolved through each page's source knowledge), and marks every
// page except the newest in each group as outdated with superseded_by_slug
// pointing at the current page (§2.15 对齐: Wiki 版本过期同步, 文号驱动).
//
// Pages without a resolvable doc_number, and pages whose source knowledge is
// missing, never participate — a blank doc number means "version chain
// unknown", not "outdated".
func (s *gazePageService) SyncOutdated(ctx context.Context, kbID string) (int, error) {
	tenantID := types.MustTenantIDFromContext(ctx)

	pages, err := s.repo.ListByType(ctx, kbID, types.GazePageTypeSummary)
	if err != nil {
		return 0, err
	}
	if len(pages) < 2 {
		return 0, nil
	}

	// Resolve doc_number per page via its source knowledge IDs.
	knowledgeIDs := make([]string, 0)
	for _, p := range pages {
		for _, ref := range p.SourceRefs {
			if id, _, ok := strings.Cut(ref, "|"); ok && id != "" {
				knowledgeIDs = append(knowledgeIDs, id)
			}
		}
	}
	docNumberByKnowledge := map[string]string{}
	if len(knowledgeIDs) > 0 {
		knowledges, err := s.knowledgeRepo.GetKnowledgeBatch(ctx, tenantID, knowledgeIDs)
		if err != nil {
			logger.Warnf(ctx, "gaze outdated: load knowledge batch failed: %v", err)
			return 0, err
		}
		for _, k := range knowledges {
			if k.DocNumber != "" {
				docNumberByKnowledge[k.ID] = k.DocNumber
			}
		}
	}

	// Group pages by doc_number.
	type chain struct {
		docNumber string
		pages     []*types.GazePage
	}
	chainsByNumber := map[string]*chain{}
	chainOrder := make([]string, 0)
	for _, p := range pages {
		number := ""
		for _, ref := range p.SourceRefs {
			if id, _, ok := strings.Cut(ref, "|"); ok {
				if n, found := docNumberByKnowledge[id]; found && n != "" {
					number = n
					break
				}
			}
		}
		if number == "" {
			continue
		}
		c, ok := chainsByNumber[number]
		if !ok {
			c = &chain{docNumber: number}
			chainsByNumber[number] = c
			chainOrder = append(chainOrder, number)
		}
		c.pages = append(c.pages, p)
	}

	changed := 0
	for _, number := range chainOrder {
		c := chainsByNumber[number]
		if len(c.pages) < 2 {
			continue
		}
		// Newest page (latest UpdatedAt, ties broken by CreatedAt, then slug
		// for determinism) stays current; every other page is outdated.
		sort.SliceStable(c.pages, func(i, j int) bool {
			a, b := c.pages[i], c.pages[j]
			if !a.UpdatedAt.Equal(b.UpdatedAt) {
				return a.UpdatedAt.After(b.UpdatedAt)
			}
			if !a.CreatedAt.Equal(b.CreatedAt) {
				return a.CreatedAt.After(b.CreatedAt)
			}
			return a.Slug < b.Slug
		})
		current := c.pages[0]
		for _, old := range c.pages[1:] {
			if old.Outdated && old.SupersededBySlug == current.Slug {
				continue
			}
			old.Outdated = true
			old.SupersededBySlug = current.Slug
			if err := s.repo.UpdateMeta(ctx, old); err != nil {
				logger.Warnf(ctx, "gaze outdated: mark page %s (doc_number=%s) failed: %v",
					old.Slug, number, err)
				continue
			}
			changed++
			logger.Infof(ctx, "gaze outdated: page %s superseded by %s (doc_number=%s)",
				old.Slug, current.Slug, number)
		}
		// The current page may have been outdated by an earlier run and then
		// received newer content since; clear the flag.
		if current.Outdated || current.SupersededBySlug != "" {
			current.Outdated = false
			current.SupersededBySlug = ""
			if err := s.repo.UpdateMeta(ctx, current); err != nil {
				logger.Warnf(ctx, "gaze outdated: clear current page %s failed: %v", current.Slug, err)
			} else {
				changed++
			}
		}
	}

	return changed, nil
}
