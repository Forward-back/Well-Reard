package interfaces

import (
	"context"

	"gaze/internal/types"
)

// CollaborationService implements Git-style document maintenance: chunk- and
// document-level issues, multi-chunk change requests, review history and
// comment threads. Permissions are enforced here (route guards only resolve KB
// access); the caller is always taken from ctx.
type CollaborationService interface {
	ListDocIssues(ctx context.Context, req *types.DocIssueListRequest) (*types.DocIssueListResponse, error)
	CreateDocIssue(ctx context.Context, kbID string, req *types.CreateDocIssueRequest) (*types.DocIssue, error)
	GetDocIssue(ctx context.Context, kbID string, issueID string) (*types.DocIssue, error)
	UpdateDocIssue(ctx context.Context, kbID string, issueID string, req *types.UpdateDocIssueRequest) (*types.DocIssue, error)
	DeleteDocIssue(ctx context.Context, kbID string, issueID string) error

	ListDocChangeRequests(ctx context.Context, req *types.DocCRListRequest) (*types.DocCRListResponse, error)
	GetDocChangeRequest(ctx context.Context, kbID string, crID string) (*types.DocChangeRequest, error)
	CreateDocChangeRequest(ctx context.Context, kbID string, req *types.CreateDocCRRequest) (*types.DocChangeRequest, error)
	ResubmitDocChangeRequest(ctx context.Context, kbID string, crID string, req *types.CreateDocCRRequest) (*types.DocChangeRequest, error)
	ReviewDocChangeRequest(ctx context.Context, kbID string, crID string, req *types.ReviewDocCRRequest) (*types.DocChangeRequest, error)
	MergeDocChangeRequest(ctx context.Context, kbID string, crID string, req *types.MergeDocCRRequest) (*types.DocChangeRequest, error)
	CloseDocChangeRequest(ctx context.Context, kbID string, crID string, comment string) (*types.DocChangeRequest, error)
	UpdateDocCRDraft(ctx context.Context, kbID string, crID string, req *types.UpdateDocCRDraftRequest) (*types.DocChangeRequest, error)
	SubmitDocCRDraft(ctx context.Context, kbID string, crID string, req *types.SubmitDocCRDraftRequest) (*types.DocChangeRequest, error)
	UpdateDocCRReviewers(ctx context.Context, kbID string, crID string, req *types.UpdateDocCRReviewersRequest) (*types.DocChangeRequest, error)
	MarkDocCRViewed(ctx context.Context, kbID string, crID string) error
	ListDocCRChunkLocks(ctx context.Context, kbID string, knowledgeID string) ([]*types.DocCRChunkLock, error)
	ListChunkCitationStats(ctx context.Context, kbID string, chunkIDs []string) ([]*types.ChunkCitationStat, error)

	ListDocAnchorComments(ctx context.Context, kbID string, knowledgeID string) ([]*types.DocAnchorComment, error)
	CreateDocAnchorComment(ctx context.Context, kbID string, req *types.CreateDocAnchorCommentRequest) (*types.DocAnchorComment, error)
	UpdateDocAnchorComment(ctx context.Context, kbID string, commentID string, req *types.UpdateDocAnchorCommentRequest) (*types.DocAnchorComment, error)
	DeleteDocAnchorComment(ctx context.Context, kbID string, commentID string) error

	ListDocVersions(ctx context.Context, kbID string, knowledgeID string) ([]*types.DocVersion, error)
	GetDocVersion(ctx context.Context, kbID string, knowledgeID string, versionID string) (*types.DocVersion, []*types.DocVersionChunk, error)
	CompareDocVersions(ctx context.Context, kbID string, knowledgeID string, from string, to string) (*types.DocVersionCompareResponse, error)
	RollbackDocVersion(ctx context.Context, kbID string, knowledgeID string, versionID string, req *types.RollbackDocVersionRequest) (*types.DocVersion, error)

	ListDocComments(ctx context.Context, kbID string, targetType string, targetID string) ([]*types.DocComment, error)
	CreateDocComment(ctx context.Context, kbID string, req *types.CreateDocCommentRequest) (*types.DocComment, error)
	DeleteDocComment(ctx context.Context, kbID string, commentID string) error

	ListDocCollaborations(ctx context.Context, req *types.DocCollaborationListRequest) (*types.DocCollaborationListResponse, error)
}
