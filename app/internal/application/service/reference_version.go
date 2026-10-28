package service

import (
	"context"

	"gaze/internal/logger"
	"gaze/internal/types"
	"gaze/internal/types/interfaces"
)

// AttachReferenceVersions stamps each citation with the document version that
// was current when the answer was produced. Answers are persisted with these
// fields, so a later edit or rollback can always be told apart from what the
// answer actually relied on. Lookups are failures-open: a missing version
// service or lookup error only leaves the fields empty.
func AttachReferenceVersions(
	ctx context.Context,
	versions interfaces.DocumentVersionService,
	tenantID uint64,
	refs types.References,
) {
	if versions == nil || tenantID == 0 || len(refs) == 0 {
		return
	}
	cache := make(map[string]*types.DocVersion)
	for _, ref := range refs {
		if ref == nil || ref.KnowledgeID == "" || ref.VersionID != "" {
			continue
		}
		version, ok := cache[ref.KnowledgeID]
		if !ok {
			latest, err := versions.LatestVersion(ctx, tenantID, ref.KnowledgeID)
			if err != nil {
				logger.Warnf(ctx, "citation version lookup failed for knowledge %s: %v", ref.KnowledgeID, err)
			}
			version = latest
			cache[ref.KnowledgeID] = version
		}
		if version == nil {
			continue
		}
		ref.VersionID = version.ID
		ref.VersionLabel = version.VersionLabel
	}
}
