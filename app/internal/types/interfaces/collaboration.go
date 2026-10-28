package interfaces

import (
	"context"

	"gaze/internal/types"
)

// CollaborationRepository persists the Git-style document collaboration
// entities (issues, change requests, reviews, comments) plus the unified
// aggregate listing used by the KB collaboration page and workbench.
type CollaborationRepository interface {
	// CreateDocIssue inserts an issue row.
	CreateDocIssue(ctx context.Context, issue *types.DocIssue) error

	// GetDocIssue loads one live (non-deleted) issue narrowed to a KB.
	GetDocIssue(ctx context.Context, tenantID uint64, kbID string, id string) (*types.DocIssue, error)

	// ListDocIssues returns a page of issues plus the total count.
	ListDocIssues(ctx context.Context, req *types.DocIssueListRequest) ([]*types.DocIssue, int64, error)

	// UpdateDocIssue rewrites title, body and status.
	UpdateDocIssue(ctx context.Context, issue *types.DocIssue) error

	// ListOpenDocIssuesByChunkIDs returns open issues anchored to any of the
	// given chunks in the KB (accept-CR confirmation dialog).
	ListOpenDocIssuesByChunkIDs(ctx context.Context, tenantID uint64, kbID string, chunkIDs []string) ([]*types.DocIssue, error)

	// MarkDocIssuesOrphaned flips anchor_orphaned for issues anchored to the
	// chunks of a re-parsed document (or to the document itself when
	// chunkIDs is empty).
	MarkDocIssuesOrphaned(ctx context.Context, tenantID uint64, knowledgeID string, chunkIDs []string) error

	// CreateDocChangeRequest inserts a CR together with its items.
	CreateDocChangeRequest(ctx context.Context, cr *types.DocChangeRequest) error

	// GetDocChangeRequest loads one CR with items and review history.
	GetDocChangeRequest(ctx context.Context, tenantID uint64, kbID string, id string) (*types.DocChangeRequest, error)

	// ListDocChangeRequests returns a page of CRs plus the total count.
	ListDocChangeRequests(ctx context.Context, req *types.DocCRListRequest) ([]*types.DocChangeRequest, int64, error)

	// ResubmitDocCR reopens a rejected CR with a fresh item set, keeping the
	// previous review history intact.
	ResubmitDocCR(ctx context.Context, cr *types.DocChangeRequest) error

	// ApplyDocCRReview atomically writes an accept/reject decision: optional
	// chunk body updates guarded by content_revision, chunk revisions, the CR
	// row, the review-history row, linked issue closures and the document's
	// engine-sync marker.
	ApplyDocCRReview(ctx context.Context, tx *types.DocCRReviewTx) error

	// ListOpenDocCRsByChunkIDs returns open CRs whose items touch any of the
	// given chunks (anchor-orphan scan).
	ListOpenDocCRsByChunkIDs(ctx context.Context, tenantID uint64, kbID string, chunkIDs []string) ([]*types.DocChangeRequest, error)

	// MarkDocCRsOrphaned flips anchor_orphaned for open CRs anchored to the
	// chunks of a re-parsed document.
	MarkDocCRsOrphaned(ctx context.Context, tenantID uint64, knowledgeID string, chunkIDs []string) error

	// CreateDocComment inserts a comment.
	CreateDocComment(ctx context.Context, comment *types.DocComment) error

	// GetDocComment loads one comment narrowed to a KB, including deleted rows.
	GetDocComment(ctx context.Context, tenantID uint64, kbID string, id string) (*types.DocComment, error)

	// ListDocComments returns a target's comments oldest-first. Deleted rows
	// are returned as placeholders with Deleted=true.
	ListDocComments(ctx context.Context, tenantID uint64, kbID string, targetType string, targetID string) ([]*types.DocComment, error)

	// UpdateDocComment rewrites the comment body or its soft-delete markers.
	UpdateDocComment(ctx context.Context, comment *types.DocComment) error

	// CountDocCommentsByTargets returns live comment counts keyed by target ID.
	CountDocCommentsByTargets(ctx context.Context, tenantID uint64, kbID string, targetType string, targetIDs []string) (map[string]int64, error)

	// ListCollaborations returns the unified issue/CR aggregate page.
	ListCollaborations(ctx context.Context, req *types.DocCollaborationListRequest) ([]*types.CollaborationItem, int64, error)

	// ---- v2: reviewers / drafts / links ----
	ReplaceDocCRReviewers(ctx context.Context, tenantID uint64, crID string, reviewers []*types.DocCRReviewer) error
	ListDocCRReviewers(ctx context.Context, tenantID uint64, crID string) ([]*types.DocCRReviewer, error)
	UpdateDocCRReviewer(ctx context.Context, reviewer *types.DocCRReviewer) error
	CreateDocCRReviewHistory(ctx context.Context, review *types.DocCRReview) error
	ReplaceDocCRLinkedIssues(ctx context.Context, tenantID uint64, crID string, issueIDs []string) error
	ListDocCRLinkedIssues(ctx context.Context, tenantID uint64, crID string) ([]*types.DocIssue, error)
	UpdateDocCR(ctx context.Context, cr *types.DocChangeRequest) error
	UpdateDocCRDraft(ctx context.Context, cr *types.DocChangeRequest) error
	ListActiveDocCRChunkLocks(ctx context.Context, tenantID uint64, kbID, knowledgeID string) ([]*types.DocCRChunkLock, error)
	ListChunkCitationStats(ctx context.Context, tenantID uint64, chunkIDs []string) ([]*types.ChunkCitationStat, error)
	MergeDocCR(ctx context.Context, tx *types.DocCRMergeTx) error
	MarkDocCRItemViewed(ctx context.Context, tenantID uint64, crID string, userID string) error

	// ---- v2: versions ----
	CreateDocVersion(ctx context.Context, version *types.DocVersion, chunks []*types.DocVersionChunk) error
	GetLatestDocVersion(ctx context.Context, tenantID uint64, knowledgeID string) (*types.DocVersion, error)
	ListDocVersions(ctx context.Context, tenantID uint64, knowledgeID string) ([]*types.DocVersion, error)
	GetDocVersion(ctx context.Context, tenantID uint64, knowledgeID string, versionID string) (*types.DocVersion, []*types.DocVersionChunk, error)
	ApplyDocVersionRollback(ctx context.Context, tx *types.DocVersionRollbackTx) error

	// ---- v2: selection comments ----
	CreateDocAnchorComment(ctx context.Context, comment *types.DocAnchorComment) error
	ListDocAnchorComments(ctx context.Context, tenantID uint64, knowledgeID string) ([]*types.DocAnchorComment, error)
	GetDocAnchorComment(ctx context.Context, tenantID uint64, id string) (*types.DocAnchorComment, error)
	UpdateDocAnchorComment(ctx context.Context, comment *types.DocAnchorComment) error
}

// DocumentVersionService snapshots documents and performs immutable rollback.
type DocumentVersionService interface {
	EnsureInitialVersion(ctx context.Context, tenantID uint64, kbID, knowledgeID, actorID string) (*types.DocVersion, error)
	CreateVersion(ctx context.Context, tenantID uint64, kbID, knowledgeID, sourceType, sourceID, title, description, actorID string) (*types.DocVersion, error)
	ListVersions(ctx context.Context, tenantID uint64, kbID, knowledgeID string) ([]*types.DocVersion, error)
	GetVersion(ctx context.Context, tenantID uint64, kbID, knowledgeID, versionID string) (*types.DocVersion, []*types.DocVersionChunk, error)
	CompareVersions(ctx context.Context, tenantID uint64, kbID, knowledgeID, fromVersionID, toVersionID string) (*types.DocVersionCompareResponse, error)
	Rollback(ctx context.Context, tenantID uint64, kbID, knowledgeID, versionID, actorID, comment string) (*types.DocVersion, error)
	RecordManualEdit(ctx context.Context, tenantID uint64, kbID, knowledgeID, actorID string) error
}
