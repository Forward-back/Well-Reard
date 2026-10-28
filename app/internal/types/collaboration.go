package types

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Collaboration statuses and target kinds (需求一 Git 式文档维护).
const (
	DocIssueStatusOpen   = "open"
	DocIssueStatusClosed = "closed"

	DocCRStatusDraft            = "draft"
	DocCRStatusPendingReview    = "pending_review"
	DocCRStatusChangesRequested = "changes_requested"
	DocCRStatusMerged           = "merged"
	DocCRStatusClosed           = "closed"

	// Deprecated aliases keep older call sites compiling while the five-state
	// machine replaces the original open/accepted/rejected values.
	DocCRStatusOpen     = DocCRStatusPendingReview
	DocCRStatusAccepted = DocCRStatusMerged
	DocCRStatusRejected = DocCRStatusChangesRequested

	DocCommentTargetIssue = "issue"
	DocCommentTargetCR    = "cr"

	DocCRReviewAccept         = "accept"
	DocCRReviewReject         = "reject"
	DocCRReviewRequestChanges = "request_changes"
	DocCRReviewApprove        = "approve"

	DocCRReviewerPending          = "pending"
	DocCRReviewerApproved         = "approved"
	DocCRReviewerChangesRequested = "changes_requested"

	DocCollaborationTypeIssue = "issue"
	DocCollaborationTypeCR    = "cr"

	DocVersionSourceUpload     = "upload"
	DocVersionSourceManualEdit = "manual_edit"
	DocVersionSourceCRMerge    = "cr_merge"
	DocVersionSourceRollback   = "rollback"

	DocIssueTypeDataError = "data_error"
	DocIssueTypeOutdated  = "outdated"
	DocIssueTypeUnclear   = "unclear"
	DocIssueTypeFormat    = "format"
	DocIssueTypeOther     = "other"
)

// Engine sync states stored on knowledges.
const (
	EngineSyncStatusSynced  = "synced"
	EngineSyncStatusPending = "pending"
	EngineSyncStatusFailed  = "failed"
)

// DocIssue is a chunk- or document-level issue raised by a KB viewer.
type DocIssue struct {
	ID              string     `json:"id"                 gorm:"type:varchar(36);primaryKey"`
	SeqID           int64      `json:"seq_id"             gorm:"type:bigint;uniqueIndex;autoIncrement"`
	TenantID        uint64     `json:"tenant_id"`
	KnowledgeBaseID string     `json:"knowledge_base_id"  gorm:"type:varchar(36);index"`
	KnowledgeID     string     `json:"knowledge_id"       gorm:"type:varchar(36);index"`
	ChunkID         string     `json:"chunk_id"           gorm:"type:varchar(36);index"`
	IssueType       string     `json:"issue_type"         gorm:"type:varchar(20);not null;default:'other'"`
	AnchorText      string     `json:"anchor_text"`
	AnchorStart     int        `json:"anchor_start"`
	AnchorEnd       int        `json:"anchor_end"`
	AnchorLineStart int        `json:"anchor_line_start"`
	AnchorLineEnd   int        `json:"anchor_line_end"`
	Title           string     `json:"title"              gorm:"type:varchar(255);not null"`
	Body            string     `json:"body"`
	Status          string     `json:"status"             gorm:"type:varchar(20);not null;default:'open'"`
	AnchorOrphaned  bool       `json:"anchor_orphaned"    gorm:"not null;default:false"`
	CreatedBy       string     `json:"created_by"         gorm:"type:varchar(36);not null"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ClosedBy        string     `json:"closed_by"          gorm:"type:varchar(36)"`
	ClosedAt        *time.Time `json:"closed_at"`
	DeletedBy       string     `json:"deleted_by"         gorm:"type:varchar(36)"`
	DeletedAt       *time.Time `json:"deleted_at"`

	CreatedByName   string `json:"created_by_name,omitempty"    gorm:"-"`
	KnowledgeTitle  string `json:"knowledge_title,omitempty"    gorm:"-"`
	ChunkExcerpt    string `json:"chunk_excerpt,omitempty"      gorm:"-"`
	ChunkIndex      int    `json:"chunk_index,omitempty"        gorm:"-"`
	CommentsCount   int64  `json:"comments_count,omitempty"     gorm:"-"`
	RelationshipTag string `json:"relationship,omitempty"       gorm:"-"`
}

// DocChangeRequest bundles one or more whole-chunk suggestions for a single
// document. All items land together or not at all.
type DocChangeRequest struct {
	ID              string     `json:"id"                 gorm:"type:varchar(36);primaryKey"`
	SeqID           int64      `json:"seq_id"             gorm:"type:bigint;uniqueIndex;autoIncrement"`
	TenantID        uint64     `json:"tenant_id"`
	KnowledgeBaseID string     `json:"knowledge_base_id"  gorm:"type:varchar(36);index"`
	KnowledgeID     string     `json:"knowledge_id"       gorm:"type:varchar(36);index"`
	Title           string     `json:"title"              gorm:"type:varchar(255);not null"`
	Description     string     `json:"description"`
	Status          string     `json:"status"             gorm:"type:varchar(24);not null;default:'pending_review'"`
	BaseVersionID   string     `json:"base_version_id"    gorm:"type:varchar(36)"`
	MergedVersionID string     `json:"merged_version_id"  gorm:"type:varchar(36)"`
	MergedAt        *time.Time `json:"merged_at"`
	ClosedBy        string     `json:"closed_by"          gorm:"type:varchar(36)"`
	ClosedAt        *time.Time `json:"closed_at"`
	Tags            StringList `json:"tags"               gorm:"type:jsonb"`
	DraftUpdatedAt  *time.Time `json:"draft_updated_at"`
	AnchorOrphaned  bool       `json:"anchor_orphaned"    gorm:"not null;default:false"`
	CreatedBy       string     `json:"created_by"         gorm:"type:varchar(36);not null"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ReviewedBy      string     `json:"reviewed_by"        gorm:"type:varchar(36)"`
	CanReview       bool       `json:"can_review"         gorm:"-"`
	ReviewedAt      *time.Time `json:"reviewed_at"`
	ReviewComment   string     `json:"review_comment"`

	CreatedByName   string           `json:"created_by_name,omitempty" gorm:"-"`
	ReviewedByName  string           `json:"reviewed_by_name,omitempty" gorm:"-"`
	KnowledgeTitle  string           `json:"knowledge_title,omitempty" gorm:"-"`
	ItemCount       int64            `json:"item_count,omitempty"      gorm:"-"`
	Items           []*DocCRItem     `json:"items,omitempty"           gorm:"-"`
	Reviews         []*DocCRReview   `json:"reviews,omitempty"         gorm:"-"`
	Reviewers       []*DocCRReviewer `json:"reviewers,omitempty"       gorm:"-"`
	LinkedIssues    []*DocIssue      `json:"linked_issues,omitempty"   gorm:"-"`
	RelationshipTag string           `json:"relationship,omitempty"    gorm:"-"`
}

// DocCRItem is one whole-chunk suggestion inside a CR. original_content is a
// server-side snapshot taken at submission time.
type DocCRItem struct {
	ID               string     `json:"id"          gorm:"type:varchar(36);primaryKey"`
	TenantID         uint64     `json:"tenant_id"`
	CRID             string     `json:"cr_id"       gorm:"type:varchar(36);index"`
	ChunkID          string     `json:"chunk_id"    gorm:"type:varchar(36);index"`
	OriginalContent  string     `json:"original_content"`
	SuggestedContent string     `json:"suggested_content"`
	Seq              int        `json:"seq"         gorm:"not null;default:0"`
	OriginalRevision int        `json:"original_revision"`
	ViewedBy         StringList `json:"viewed_by"     gorm:"type:jsonb"`
	AnchorText       string     `json:"anchor_text"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	ChunkIndex     int    `json:"chunk_index,omitempty"     gorm:"-"`
	AnchorOrphaned bool   `json:"anchor_orphaned,omitempty" gorm:"-"`
	Conflict       bool   `json:"conflict,omitempty"        gorm:"-"`
	CurrentContent string `json:"current_content,omitempty" gorm:"-"`
}

// DocCRReview records every accept/reject decision so resubmitted CRs keep
// their previous review comments.
type DocCRReview struct {
	ID         string    `json:"id"         gorm:"type:varchar(36);primaryKey"`
	TenantID   uint64    `json:"tenant_id"`
	CRID       string    `json:"cr_id"      gorm:"type:varchar(36);index"`
	Action     string    `json:"action"     gorm:"type:varchar(20);not null"`
	Comment    string    `json:"comment"`
	ReviewedBy string    `json:"reviewed_by" gorm:"type:varchar(36);not null"`
	CreatedAt  time.Time `json:"created_at"`

	ReviewedByName string `json:"reviewed_by_name,omitempty" gorm:"-"`
}

// DocCRReviewer is one assigned reviewer and their current decision for a CR.
type DocCRReviewer struct {
	ID        string     `json:"id"         gorm:"type:varchar(36);primaryKey"`
	TenantID  uint64     `json:"tenant_id"`
	CRID      string     `json:"cr_id"      gorm:"type:varchar(36);index"`
	UserID    string     `json:"user_id"    gorm:"type:varchar(36);index"`
	Status    string     `json:"status"     gorm:"type:varchar(24);not null;default:'pending'"`
	Comment   string     `json:"comment"`
	DecidedAt *time.Time `json:"decided_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`

	UserName string `json:"user_name,omitempty" gorm:"-"`
}

// DocCRIssueLink explicitly links a CR to issues it fixes.
type DocCRIssueLink struct {
	ID        string    `json:"id"         gorm:"type:varchar(36);primaryKey"`
	TenantID  uint64    `json:"tenant_id"`
	CRID      string    `json:"cr_id"      gorm:"type:varchar(36);index"`
	IssueID   string    `json:"issue_id"   gorm:"type:varchar(36);index"`
	CreatedAt time.Time `json:"created_at"`
}

func (DocCRIssueLink) TableName() string { return "doc_cr_issues" }

// DocCRChunkLock reports an in-flight CR touching a chunk.
type DocCRChunkLock struct {
	ChunkID string `json:"chunk_id"`
	CRID    string `json:"cr_id"`
	SeqID   int64  `json:"seq_id"`
	Title   string `json:"title"`
	Status  string `json:"status"`
}

// ChunkCitationStat is the answer-reference count for an app chunk.
type ChunkCitationStat struct {
	ChunkID    string `json:"chunk_id"`
	Total      int64  `json:"total"`
	Last30Days int64  `json:"last_30_days" gorm:"column:last_30_days"`
}

// DocVersion is one immutable document snapshot. Version labels are V1.{seq}.
type DocVersion struct {
	ID              string    `json:"id"               gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64    `json:"tenant_id"`
	KnowledgeBaseID string    `json:"knowledge_base_id" gorm:"type:varchar(36);index"`
	KnowledgeID     string    `json:"knowledge_id"     gorm:"type:varchar(36);index"`
	VersionSeq      int       `json:"version_seq"`
	VersionLabel    string    `json:"version_label"    gorm:"type:varchar(16)"`
	SourceType      string    `json:"source_type"      gorm:"type:varchar(24);not null"`
	SourceID        string    `json:"source_id"        gorm:"type:varchar(36)"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	ChunkCount      int       `json:"chunk_count"`
	CreatedBy       string    `json:"created_by"       gorm:"type:varchar(36)"`
	CreatedAt       time.Time `json:"created_at"`
	IsCurrent       bool      `json:"is_current"       gorm:"-"`

	CreatedByName string `json:"created_by_name,omitempty" gorm:"-"`
}

// DocVersionChunk is one chunk snapshot inside a DocVersion.
type DocVersionChunk struct {
	ID         string `json:"id"          gorm:"type:varchar(36);primaryKey"`
	TenantID   uint64 `json:"tenant_id"`
	VersionID  string `json:"version_id"  gorm:"type:varchar(36);index"`
	ChunkID    string `json:"chunk_id"    gorm:"type:varchar(36);index"`
	ChunkIndex int    `json:"chunk_index"`
	Content    string `json:"content"`
	IsEnabled  bool   `json:"is_enabled"`
}

// DocAnchorComment is a document-selection comment with an anchored excerpt.
// parent_id non-empty makes it a reply in the thread.
type DocAnchorComment struct {
	ID              string     `json:"id"                gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64     `json:"tenant_id"`
	KnowledgeBaseID string     `json:"knowledge_base_id" gorm:"type:varchar(36);index"`
	KnowledgeID     string     `json:"knowledge_id"      gorm:"type:varchar(36);index"`
	ChunkID         string     `json:"chunk_id"          gorm:"type:varchar(36);index"`
	ParentID        string     `json:"parent_id"         gorm:"type:varchar(36);index"`
	AnchorText      string     `json:"anchor_text"`
	AnchorStart     int        `json:"anchor_start"`
	AnchorEnd       int        `json:"anchor_end"`
	AnchorLineStart int        `json:"anchor_line_start"`
	AnchorLineEnd   int        `json:"anchor_line_end"`
	Body            string     `json:"body"`
	ResolvedAt      *time.Time `json:"resolved_at"`
	ResolvedBy      string     `json:"resolved_by"       gorm:"type:varchar(36)"`
	CreatedBy       string     `json:"created_by"        gorm:"type:varchar(36);not null"`
	CreatedAt       time.Time  `json:"created_at"`
	DeletedBy       string     `json:"deleted_by"        gorm:"type:varchar(36)"`
	DeletedAt       *time.Time `json:"deleted_at"`

	CreatedByName string `json:"created_by_name,omitempty" gorm:"-"`
}

// StringList stores a JSON string array in Postgres jsonb / SQLite text.
type StringList []string

func (s StringList) Value() (driver.Value, error) { return json.Marshal([]string(s)) }

func (s *StringList) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return nil
	}
	if len(b) == 0 {
		*s = nil
		return nil
	}
	var out []string
	if err := json.Unmarshal(b, &out); err != nil {
		return err
	}
	*s = out
	return nil
}

// DocComment is a flat comment thread entry on an issue or a CR.
type DocComment struct {
	ID              string     `json:"id"                gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64     `json:"tenant_id"`
	KnowledgeBaseID string     `json:"knowledge_base_id" gorm:"type:varchar(36);index"`
	TargetType      string     `json:"target_type"       gorm:"type:varchar(16);not null"`
	TargetID        string     `json:"target_id"         gorm:"type:varchar(36);not null"`
	Body            string     `json:"body"`
	CreatedBy       string     `json:"created_by"        gorm:"type:varchar(36);not null"`
	CreatedAt       time.Time  `json:"created_at"`
	DeletedBy       string     `json:"deleted_by"        gorm:"type:varchar(36)"`
	DeletedAt       *time.Time `json:"deleted_at"`

	CreatedByName string `json:"created_by_name,omitempty" gorm:"-"`
	Deleted       bool   `json:"deleted,omitempty"         gorm:"-"`
}

// CollaborationItem is the unified issue/CR row returned by the KB aggregate
// endpoint and the future workbench.
type CollaborationItem struct {
	Type            string     `json:"type"`
	ID              string     `json:"id"`
	SeqID           int64      `json:"seq_id"`
	Title           string     `json:"title"`
	Status          string     `json:"status"`
	KnowledgeID     string     `json:"knowledge_id"`
	KnowledgeBaseID string     `json:"knowledge_base_id"`
	KnowledgeTitle  string     `json:"knowledge_title,omitempty"`
	ChunkID         string     `json:"chunk_id,omitempty"`
	CreatedBy       string     `json:"created_by"`
	CreatedByName   string     `json:"created_by_name,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	AnchorOrphaned  bool       `json:"anchor_orphaned"`
	CommentsCount   int64      `json:"comments_count"`
	ItemCount       int64      `json:"item_count,omitempty"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	RelationshipTag string     `json:"relationship,omitempty"`
}

// EngineSyncConfig is the tenant-level daily engine resync switch.
type EngineSyncConfig struct {
	Enabled bool `json:"enabled"`
}

// DocIssueListRequest filters the KB-scoped issue list.
type DocIssueListRequest struct {
	TenantID            uint64
	KnowledgeBaseID     string
	KnowledgeID         string
	ChunkID             string
	Status              string
	CreatedBy           string
	Page                int
	PageSize            int
	ACLActive           bool
	AllowedKnowledgeIDs []string
}

// DocCRListRequest filters the KB-scoped change-request list.
type DocCRListRequest struct {
	TenantID            uint64
	KnowledgeBaseID     string
	KnowledgeID         string
	Status              string
	CreatedBy           string
	VisibilityUserID    string
	Page                int
	PageSize            int
	ACLActive           bool
	AllowedKnowledgeIDs []string
}

// DocCollaborationListRequest filters the unified issue/CR aggregate.
type DocCollaborationListRequest struct {
	TenantID            uint64
	KnowledgeBaseID     string
	KnowledgeID         string
	Type                string
	Status              string
	CreatedBy           string
	VisibilityUserID    string
	Page                int
	PageSize            int
	ACLActive           bool
	AllowedKnowledgeIDs []string
}

// CreateDocIssueRequest is the viewer-submitted issue payload.
type CreateDocIssueRequest struct {
	KnowledgeID     string `json:"knowledge_id" binding:"required"`
	ChunkID         string `json:"chunk_id"`
	IssueType       string `json:"issue_type"`
	AnchorText      string `json:"anchor_text"`
	AnchorStart     int    `json:"anchor_start"`
	AnchorEnd       int    `json:"anchor_end"`
	AnchorLineStart int    `json:"anchor_line_start"`
	AnchorLineEnd   int    `json:"anchor_line_end"`
	Title           string `json:"title"        binding:"required,max=255"`
	Body            string `json:"body"`
}

// UpdateDocIssueRequest patches an issue; status may only be open/closed.
type UpdateDocIssueRequest struct {
	Title     *string `json:"title"`
	Body      *string `json:"body"`
	Status    *string `json:"status"`
	IssueType *string `json:"issue_type"`
}

// DocCRItemInput is one whole-chunk suggestion supplied at CR creation.
type DocCRItemInput struct {
	ChunkID          string `json:"chunk_id"          binding:"required"`
	SuggestedContent string `json:"suggested_content"`
	AnchorText       string `json:"anchor_text"`
}

// CreateDocCRRequest submits a single-document CR for one or more chunks.
type CreateDocCRRequest struct {
	KnowledgeID    string           `json:"knowledge_id" binding:"required"`
	Title          string           `json:"title"        binding:"max=255"`
	Description    string           `json:"description"`
	Items          []DocCRItemInput `json:"items"`
	ReviewerIDs    []string         `json:"reviewer_ids"`
	Tags           []string         `json:"tags"`
	LinkedIssueIDs []string         `json:"linked_issue_ids"`
	BaseVersionID  string           `json:"base_version_id"`
	AsDraft        bool             `json:"as_draft"`
}

// UpdateDocCRDraftRequest autosaves a draft CR.
type UpdateDocCRDraftRequest struct {
	Title          string           `json:"title"`
	Description    string           `json:"description"`
	Items          []DocCRItemInput `json:"items"`
	ReviewerIDs    []string         `json:"reviewer_ids"`
	Tags           []string         `json:"tags"`
	LinkedIssueIDs []string         `json:"linked_issue_ids"`
	BaseVersionID  string           `json:"base_version_id"`
}

// SubmitDocCRDraftRequest finalizes a draft into pending_review.
type SubmitDocCRDraftRequest struct {
	CreateDocCRRequest
}

// ReviewDocCRRequest carries one reviewer's decision.
type ReviewDocCRRequest struct {
	Action  string `json:"action"          binding:"required,oneof=approve request_changes reject"`
	Comment string `json:"comment"`
}

// MergeDocCRRequest merges an approved CR and optionally closes linked issues.
type MergeDocCRRequest struct {
	Comment       string   `json:"comment"`
	CloseIssueIDs []string `json:"close_issue_ids"`
}

// UpdateDocCRReviewersRequest replaces the reviewer assignment. The tenant
// owner/admin may do this at any time; the creator may set it at creation.
type UpdateDocCRReviewersRequest struct {
	ReviewerIDs []string `json:"reviewer_ids" binding:"required"`
}

// CreateDocAnchorCommentRequest creates a selection comment or a reply.
type CreateDocAnchorCommentRequest struct {
	KnowledgeID     string `json:"knowledge_id" binding:"required"`
	ChunkID         string `json:"chunk_id"`
	ParentID        string `json:"parent_id"`
	AnchorText      string `json:"anchor_text"`
	AnchorStart     int    `json:"anchor_start"`
	AnchorEnd       int    `json:"anchor_end"`
	AnchorLineStart int    `json:"anchor_line_start"`
	AnchorLineEnd   int    `json:"anchor_line_end"`
	Body            string `json:"body" binding:"required"`
}

// UpdateDocAnchorCommentRequest resolves or edits a selection comment.
type UpdateDocAnchorCommentRequest struct {
	Body     *string `json:"body"`
	Resolved *bool   `json:"resolved"`
}

// DocVersionCompareResponse returns the two snapshots for client-side diffing.
type DocVersionCompareResponse struct {
	From       *DocVersion        `json:"from"`
	To         *DocVersion        `json:"to"`
	FromChunks []*DocVersionChunk `json:"from_chunks"`
	ToChunks   []*DocVersionChunk `json:"to_chunks"`
}

// RollbackDocVersionRequest optionally records why a rollback happened.
type RollbackDocVersionRequest struct {
	Comment string `json:"comment"`
}

// CreateDocCommentRequest adds a flat comment to an issue or CR.
type CreateDocCommentRequest struct {
	TargetType string `json:"target_type" binding:"required,oneof=issue cr"`
	TargetID   string `json:"target_id"   binding:"required"`
	Body       string `json:"body"        binding:"required"`
}

// DocCRReviewTx is the repository-level atomic payload for a CR decision.
// Chunks/Revisions are populated only on accept; every chunk write is guarded
// by the revision the reviewer saw so a concurrent edit cannot be clobbered.
type DocCRReviewTx struct {
	CR                    *DocChangeRequest
	Review                *DocCRReview
	Chunks                []*Chunk
	Revisions             []*ChunkRevision
	CloseIssueIDs         []string
	KnowledgeID           string
	MarkEngineSyncPending bool
	Now                   time.Time
}

// DocCRChunkSnapshot lets the service pair a CR item with its live chunk.
type DocCRChunkSnapshot struct {
	ChunkID         string
	Content         string
	ContentRevision int
}

// DocCRMergeTx atomically applies an approved CR, closes linked issues and
// snapshots the resulting document version.
type DocCRMergeTx struct {
	CR                    *DocChangeRequest
	Chunks                []*Chunk
	Revisions             []*ChunkRevision
	CloseIssueIDs         []string
	KnowledgeID           string
	Version               *DocVersion
	MarkEngineSyncPending bool
	Now                   time.Time
}

// DocVersionRollbackTx atomically restores a snapshot and records the new
// rollback version.
type DocVersionRollbackTx struct {
	Version               *DocVersion
	KnowledgeID           string
	Chunks                []*Chunk
	NewChunks             []*Chunk
	Revisions             []*ChunkRevision
	DisableChunkIDs       []string
	MarkEngineSyncPending bool
	Now                   time.Time
}

// DocIssueListResponse is the paginated issue list envelope.
type DocIssueListResponse struct {
	Items    []*DocIssue `json:"items"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// DocCRListResponse is the paginated CR list envelope.
type DocCRListResponse struct {
	Items    []*DocChangeRequest `json:"items"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}

// DocCollaborationListResponse is the unified aggregate list envelope.
type DocCollaborationListResponse struct {
	Items    []*CollaborationItem `json:"items"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}

func (c EngineSyncConfig) Value() (driver.Value, error) { return json.Marshal(c) }

func (c *EngineSyncConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return nil
	}
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, c)
}

// BeforeCreate fills SeqID for SQLite, which has no autoIncrement on
// non-primary-key columns.
func (i *DocIssue) BeforeCreate(tx *gorm.DB) error {
	if tx.Dialector.Name() != "sqlite" || i.SeqID != 0 {
		return nil
	}
	var maxSeqID *int64
	if err := tx.Unscoped().Model(&DocIssue{}).Select("MAX(seq_id)").Scan(&maxSeqID).Error; err != nil {
		return err
	}
	if maxSeqID != nil {
		i.SeqID = *maxSeqID + 1
	} else {
		i.SeqID = 1
	}
	return nil
}

// BeforeCreate fills SeqID for SQLite (see DocIssue.BeforeCreate).
func (c *DocChangeRequest) BeforeCreate(tx *gorm.DB) error {
	if tx.Dialector.Name() != "sqlite" || c.SeqID != 0 {
		return nil
	}
	var maxSeqID *int64
	if err := tx.Unscoped().Model(&DocChangeRequest{}).Select("MAX(seq_id)").Scan(&maxSeqID).Error; err != nil {
		return err
	}
	if maxSeqID != nil {
		c.SeqID = *maxSeqID + 1
	} else {
		c.SeqID = 1
	}
	return nil
}
