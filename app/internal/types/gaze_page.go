package types

import (
	"database/sql/driver"
	"encoding/json"
	"strings"
	"time"

	"gorm.io/gorm"
)

// GazeCategoryMaxDepth is the hard cap on how many folder levels a gaze page's
// category_path may keep. The ingest prompts intentionally ask the model for at
// most 2 levels to keep the directory tree shallow; this storage cap is one
// level deeper as a defensive bound so a slightly over-eager model (or a
// reconcile remap) cannot create an unbounded breadcrumb. It is the single
// source of truth shared by the service, repository, and taxonomy layers so
// stored paths and queried paths are always cleaned identically.
const GazeCategoryMaxDepth = 3

var gazeCategorySeparatorReplacer = strings.NewReplacer("／", "/", "｜", "/", "|", "/")

// CleanGazeCategoryPart normalizes a single raw category label that may itself
// carry embedded separators, wrapping quotes/brackets, or page-type noise, and
// returns the cleaned sub-labels (type labels such as "entity"/"实体" dropped).
func CleanGazeCategoryPart(part string) []string {
	part = strings.TrimSpace(part)
	if part == "" {
		return nil
	}
	part = gazeCategorySeparatorReplacer.Replace(part)
	rawParts := strings.Split(part, "/")
	cleaned := make([]string, 0, len(rawParts))
	for _, raw := range rawParts {
		label := strings.TrimSpace(raw)
		label = strings.Trim(label, `"'“”‘’[]（）()`)
		label = strings.TrimSpace(label)
		if label == "" || isGazeTypeCategoryLabel(label) {
			continue
		}
		cleaned = append(cleaned, label)
	}
	return cleaned
}

// CleanGazeCategoryPath cleans, deduplicates, and caps a full category path at
// GazeCategoryMaxDepth. Centralizing this guarantees that the path a page is
// stored with and the path a list/filter query is matched against go through
// the exact same normalization, so directory filters cannot silently drift.
func CleanGazeCategoryPath(parts []string) []string {
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		for _, label := range CleanGazeCategoryPart(part) {
			if containsGazeString(cleaned, label) {
				continue
			}
			cleaned = append(cleaned, label)
			if len(cleaned) >= GazeCategoryMaxDepth {
				return cleaned
			}
		}
	}
	return cleaned
}

// TrimGazeFolderSegments keeps folder path segments verbatim, dropping only
// blank entries. Folder names are validated on creation (no separators), and
// the folder tree is the source of truth for a page's placement, so unlike
// CleanGazeCategoryPath no page-type filtering, deduplication, or depth cap
// applies: a user may legitimately name a folder "概念" or "Concepts".
func TrimGazeFolderSegments(parts []string) []string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// GazeFolderPathSegments splits a materialized folder path ("AI/RAG") into its
// literal segments. An empty/blank path yields nil (the gaze root).
func GazeFolderPathSegments(path string) []string {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	return TrimGazeFolderSegments(strings.Split(path, "/"))
}

// SplitGazePageTypes parses a page_type value that may carry several
// comma-separated types (e.g. "entity,concept") into a deduplicated slice,
// dropping blanks. An empty/whitespace-only input yields nil ("no filter").
// Shared by the handler (query parsing) and repository (List filter) so the
// two layers split identically.
func SplitGazePageTypes(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, 4)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
	}
	return out
}

func isGazeTypeCategoryLabel(label string) bool {
	normalized := strings.ToLower(strings.TrimSpace(label))
	normalized = strings.TrimSuffix(normalized, "s")
	switch normalized {
	case "entity", "实体", "實體", "concept", "概念", "summary", "摘要", "gaze", "页面", "頁面":
		return true
	default:
		return false
	}
}

func containsGazeString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// GazePageType constants define the types of gaze pages
const (
	// GazePageTypeSummary represents a document summary page
	GazePageTypeSummary = "summary"
	// GazePageTypeEntity represents an entity page (person, organization, place, etc.)
	GazePageTypeEntity = "entity"
	// GazePageTypeConcept represents a concept/topic page
	GazePageTypeConcept = "concept"
	// GazePageTypeIndex represents the gaze index page (index.md)
	GazePageTypeIndex = "index"
	// GazePageTypeSynthesis represents a synthesis/analysis page.
	// NOT auto-created by ingest — Agent creates these via gaze_write_page tool
	// when it generates cross-document analysis, trends, or insights during conversations.
	GazePageTypeSynthesis = "synthesis"
	// GazePageTypeComparison represents a comparison page.
	// NOT auto-created by ingest — Agent creates these via gaze_write_page tool
	// when the user asks to compare entities, concepts, or approaches.
	GazePageTypeComparison = "comparison"
)

// GazePageStatus constants
const (
	// GazePageStatusDraft indicates the page is a draft
	GazePageStatusDraft = "draft"
	// GazePageStatusPublished indicates the page is published and visible
	GazePageStatusPublished = "published"
	// GazePageStatusArchived indicates the page is archived
	GazePageStatusArchived = "archived"
)

// IsValidGazePageType reports whether pageType is one of the known page
// types. Unknown values would silently disappear from the type-filtered
// listings the browser is built on, so write paths reject them.
func IsValidGazePageType(pageType string) bool {
	switch pageType {
	case GazePageTypeSummary, GazePageTypeEntity, GazePageTypeConcept,
		GazePageTypeIndex, GazePageTypeSynthesis, GazePageTypeComparison:
		return true
	default:
		return false
	}
}

// IsValidGazePageStatus reports whether status is one of the known statuses.
func IsValidGazePageStatus(status string) bool {
	switch status {
	case GazePageStatusDraft, GazePageStatusPublished, GazePageStatusArchived:
		return true
	default:
		return false
	}
}

// GazePage represents a single gaze page in a gaze knowledge base.
// Gaze pages are LLM-generated, interlinked markdown documents that form
// a persistent, compounding knowledge artifact.
type GazePage struct {
	// Unique identifier (UUID)
	ID string `json:"id" gorm:"type:varchar(36);primaryKey"`
	// Workspace ID for multi-workspace isolation
	TenantID uint64 `json:"tenant_id" gorm:"index"`
	// Knowledge base this page belongs to
	KnowledgeBaseID string `json:"knowledge_base_id" gorm:"type:varchar(36);index"`
	// URL-friendly slug for addressing, e.g. "entity/acme-corp", "concept/rag"
	// Unique within a knowledge base
	Slug string `json:"slug" gorm:"type:varchar(255);uniqueIndex:idx_kb_slug"`
	// Human-readable title
	Title string `json:"title" gorm:"type:varchar(512)"`
	// Page type: summary, entity, concept, index, synthesis, comparison
	PageType string `json:"page_type" gorm:"type:varchar(32);index"`
	// Page status: draft, published, archived
	Status string `json:"status" gorm:"type:varchar(32);default:'published'"`
	// Full markdown content
	Content string `json:"content" gorm:"type:text"`
	// One-line summary for index listing
	Summary string `json:"summary" gorm:"type:text"`
	// Alternate names, abbreviations, acronyms or translated names
	Aliases StringArray `json:"aliases" gorm:"type:json"`
	// ParentSlug optionally points at the gaze page that should act as this
	// page's semantic parent in the directory tree. The parent may be empty
	// when the page is grouped only by FolderID.
	ParentSlug string `json:"parent_slug,omitempty" gorm:"type:varchar(255);index"`
	// FolderID is the single source of truth for where this page sits in the
	// directory tree — a reference to gaze_folders.id ("" = gaze root). The
	// CategoryPath / GazePath / Depth fields below are denormalized caches
	// recomputed from this folder's chain on every write so list/index/search
	// queries don't have to join gaze_folders.
	FolderID string `json:"folder_id,omitempty" gorm:"column:folder_id;type:varchar(36);index;default:''"`
	// CategoryPath is the directory breadcrumb that groups this page in the
	// gaze browser, e.g. ["AI", "LLM 应用", "RAG"]. Derived cache of the
	// folder chain identified by FolderID.
	CategoryPath StringArray `json:"category_path,omitempty" gorm:"type:json"`
	// GazePath is a normalized, sortable path derived from page_type,
	// category_path, and title. It keeps large directory listings cheap to sort.
	GazePath string `json:"gaze_path,omitempty" gorm:"type:varchar(1024);index"`
	// Depth is len(CategoryPath), cached for filtering / display.
	Depth int `json:"depth,omitempty" gorm:"default:0;index"`
	// SortOrder allows generated or manually edited pages to control sibling
	// ordering before falling back to title.
	SortOrder int `json:"sort_order,omitempty" gorm:"default:0;index"`
	// References to source knowledge IDs that contributed to this page.
	// Format matches the legacy "<knowledge_id>|<doc_title>" convention used
	// across the ingest pipeline, so retract / display code can split on `|`
	// to recover the title. Document-level granularity.
	SourceRefs StringArray `json:"source_refs" gorm:"type:json"`
	// ChunkRefs records the specific source-document chunks this page was
	// built from — one UUID per cited chunk. Populated during ingest from
	// the chunk-citation pass; refreshed wholesale whenever the page is
	// re-materialized. Empty for summary pages (they are document-level
	// synopses and don't carry chunk-level citations). Use this when you
	// need to surface the underlying evidence for a gaze page, or to
	// retract citations when a source document is deleted.
	ChunkRefs StringArray `json:"chunk_refs" gorm:"type:json"`
	// Slugs of pages that link TO this page (backlinks)
	InLinks StringArray `json:"in_links" gorm:"type:json"`
	// Slugs of pages this page links to (outbound links)
	OutLinks StringArray `json:"out_links" gorm:"type:json"`
	// Arbitrary metadata (tags, categories, dates, etc.)
	PageMetadata JSON `json:"page_metadata" gorm:"column:page_metadata;type:json"`
	// Version number. Incremented only when a user-visible content field
	// (title, content, summary, page_type, status) actually changes; pure
	// bookkeeping writes (link maintenance, same-content re-ingest, status
	// sync from background jobs) leave it untouched so it can be used as a
	// real "the page was edited" signal.
	Version int `json:"version" gorm:"default:1"`
	// LastEditSource records who authored the CURRENT version: pipeline |
	// agent | user | revert. Empty for legacy rows (treated as pipeline).
	// When the version is superseded this value travels into the revision
	// snapshot, so each historical version keeps its own author kind.
	LastEditSource string `json:"last_edit_source,omitempty" gorm:"type:varchar(16);default:''"`
	// LastEditorID is the user id of the caller that produced the current
	// version (empty for background pipeline writes).
	LastEditorID string `json:"last_editor_id,omitempty" gorm:"type:varchar(64);default:''"`
	// Outdated marks a page superseded by a newer page sharing the same
	// doc_number (UMS 文号). Only summary pages participate in the version
	// chain; see gazePageService.SyncOutdated.
	Outdated bool `json:"outdated" gorm:"column:outdated;default:false"`
	// SupersededBySlug points at the slug of the page that replaces this one
	// ("" when current). Paired with Outdated for the 「已过期」 UI label.
	SupersededBySlug string `json:"superseded_by_slug,omitempty" gorm:"column:superseded_by_slug;type:varchar(255);default:''"`
	// Creation time
	CreatedAt time.Time `json:"created_at"`
	// Last update time
	UpdatedAt time.Time `json:"updated_at"`
	// Soft delete
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

// TableName specifies the database table name
func (GazePage) TableName() string {
	return "gaze_pages"
}

// Gaze edit source constants describe who authored a page version. Stored in
// GazePage.LastEditSource (current version) and GazePageRevision.EditSource
// (historical versions).
const (
	// GazeEditSourcePipeline marks versions written by the gaze ingest
	// pipeline (also the fallback for legacy rows with an empty source).
	GazeEditSourcePipeline = "pipeline"
	// GazeEditSourceAgent marks versions written through the agent gaze
	// tools (gaze_write_page / gaze_replace_text / ...).
	GazeEditSourceAgent = "agent"
	// GazeEditSourceUser marks versions written by a human through the
	// gaze editor UI / REST API.
	GazeEditSourceUser = "user"
	// GazeEditSourceRevert marks versions produced by rolling the page
	// back to an earlier revision.
	GazeEditSourceRevert = "revert"
)

// NormalizeGazeEditSource maps unknown / empty values to
// GazeEditSourcePipeline so legacy rows and forgotten call sites degrade to
// the historical behavior ("the machine wrote this").
func NormalizeGazeEditSource(source string) string {
	switch source {
	case GazeEditSourceAgent, GazeEditSourceUser, GazeEditSourceRevert, GazeEditSourcePipeline:
		return source
	default:
		return GazeEditSourcePipeline
	}
}

// GazePageRevision is one immutable snapshot of a superseded page version.
// The current version lives only in gaze_pages; when an edit replaces
// version V the pre-edit state is inserted here as (page_id, V) inside the
// same transaction, so every historical version stays diffable and
// revertable. Rows are pruned per GazeRevisionPruneRequest to bound storage
// on hot pipeline pages.
type GazePageRevision struct {
	ID              string      `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64      `json:"tenant_id" gorm:"index"`
	KnowledgeBaseID string      `json:"knowledge_base_id" gorm:"type:varchar(36);index:idx_gaze_page_revisions_kb_slug"`
	PageID          string      `json:"page_id" gorm:"type:varchar(36);uniqueIndex:idx_gaze_page_revisions_page_version"`
	Slug            string      `json:"slug" gorm:"type:varchar(255);index:idx_gaze_page_revisions_kb_slug"`
	Version         int         `json:"version" gorm:"uniqueIndex:idx_gaze_page_revisions_page_version"`
	Title           string      `json:"title" gorm:"type:varchar(512)"`
	PageType        string      `json:"page_type" gorm:"type:varchar(32)"`
	Status          string      `json:"status" gorm:"type:varchar(32)"`
	Content         string      `json:"content,omitempty" gorm:"type:text"`
	Summary         string      `json:"summary" gorm:"type:text"`
	Aliases         StringArray `json:"aliases" gorm:"type:json"`
	// Author of THIS version (same semantics as GazePage.LastEditSource).
	EditSource string `json:"edit_source" gorm:"type:varchar(16);default:''"`
	EditorID   string `json:"editor_id" gorm:"type:varchar(64);default:''"`
	// When this version was authored (the page's updated_at while current).
	EditedAt  time.Time `json:"edited_at"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName specifies the database table name
func (GazePageRevision) TableName() string {
	return "gaze_page_revisions"
}

// Snapshot retention is two-tiered, because the two kinds of history have
// very different value. A hot hub page is rewritten by the ingest pipeline on
// every related batch, so machine snapshots would otherwise crowd out the
// handful of human/agent edits this feature exists to protect.
const (
	// GazeMaxRevisionsPerPage is how many recent versions are kept for
	// prunable (machine-authored) snapshots.
	GazeMaxRevisionsPerPage = 50
	// GazeMaxRevisionsHardCap is the absolute per-page ceiling, applied
	// regardless of author, so a page edited only by hand still has bounded
	// storage.
	GazeMaxRevisionsHardCap = 200
)

// GazePrunableEditSources lists the edit sources whose snapshots may be
// dropped by the soft cap. Everything else (user / agent / revert) survives
// until the hard cap. Legacy rows carry an empty source, hence "".
var GazePrunableEditSources = []string{"", GazeEditSourcePipeline}

// GazeRevisionPruneRequest describes the two-tier retention applied to one
// page's snapshot history. A version threshold <= 0 disables that tier.
type GazeRevisionPruneRequest struct {
	PageID string
	// KeepFromVersion: snapshots below this version are dropped, but only
	// when their edit source is listed in PrunableSources.
	KeepFromVersion int
	PrunableSources []string
	// HardKeepFromVersion: snapshots below this version are always dropped,
	// whatever their author.
	HardKeepFromVersion int
}

// GazePageUpdateRequest is the partial-update payload for
// PUT /gaze/pages/*slug. All content fields are optional pointers — absent
// fields keep their stored value, so a client can change just the body
// without re-sending (and risking clobbering) title/status/aliases.
type GazePageUpdateRequest struct {
	Title    *string      `json:"title,omitempty"`
	Content  *string      `json:"content,omitempty"`
	Summary  *string      `json:"summary,omitempty"`
	PageType *string      `json:"page_type,omitempty"`
	Status   *string      `json:"status,omitempty"`
	Aliases  *StringArray `json:"aliases,omitempty"`
	// Version is the optimistic-lock guard: when > 0 the update is rejected
	// with a conflict if the stored version differs (someone else edited the
	// page since the client loaded it). 0 skips the check (legacy clients).
	Version int `json:"version,omitempty"`
}

// GazePageRevisionListResponse is the payload for GET /gaze/revisions/*slug.
// Revisions hold the superseded versions (content omitted in list mode);
// the current version is described by CurrentVersion + the page row itself,
// which the frontend already has.
type GazePageRevisionListResponse struct {
	Revisions      []*GazePageRevision `json:"revisions"`
	Total          int64               `json:"total"`
	CurrentVersion int                 `json:"current_version"`
}

// GazePageRevertRequest rolls a page back to one of its stored revisions.
// Slug travels in the body (not the path) for the same reason as
// GazePageMoveRequest: hierarchical slugs collide with gin's catch-all.
type GazePageRevertRequest struct {
	Slug    string `json:"slug" binding:"required"`
	Version int    `json:"version" binding:"required"`
}

// GazeFolderRootID is the sentinel parent/folder id meaning "the gaze root"
// (a page or folder directly under the top level, with no parent folder).
const GazeFolderRootID = ""

// GazeFolder is a first-class directory node in the gaze browser. Folders
// exist independently of pages — an empty folder persists so users can lay
// out a skeleton and file pages into it later. The tree is an adjacency list
// (ParentID, "" = root); Path is the materialized "/"-joined name chain kept
// purely for cheap display/sort. A gaze page's placement is GazePage.FolderID.
type GazeFolder struct {
	ID              string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64         `json:"tenant_id" gorm:"index"`
	KnowledgeBaseID string         `json:"knowledge_base_id" gorm:"type:varchar(36);index"`
	ParentID        string         `json:"parent_id" gorm:"column:parent_id;type:varchar(36);index;default:''"`
	Name            string         `json:"name" gorm:"type:varchar(255)"`
	Path            string         `json:"path" gorm:"type:varchar(1024)"`
	Depth           int            `json:"depth" gorm:"default:0"`
	SortOrder       int            `json:"sort_order" gorm:"default:0"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

// TableName specifies the database table name
func (GazeFolder) TableName() string {
	return "gaze_folders"
}

// GazeFolderNode is one directory node returned to the browser, enriched with
// the live page count directly under it and whether it has child folders so
// the UI can render an expand affordance without a second round-trip.
type GazeFolderNode struct {
	GazeFolder
	PageCount   int64 `json:"page_count"`
	HasChildren bool  `json:"has_children"`
}

// GazeFolderListResponse is the payload for listing the direct children of a
// folder (parent_id="" = root level).
type GazeFolderListResponse struct {
	ParentID string           `json:"parent_id"`
	Folders  []GazeFolderNode `json:"folders"`
}

// GazeFolderCreateRequest creates a new (initially empty) folder under ParentID.
type GazeFolderCreateRequest struct {
	ParentID string `json:"parent_id"`
	Name     string `json:"name"`
}

// GazeFolderUpdateRequest renames and/or reparents a folder. ParentID is
// applied only when MoveParent is true so a pure rename doesn't have to
// re-send the (possibly root "") parent and risk an accidental move.
type GazeFolderUpdateRequest struct {
	Name       string `json:"name,omitempty"`
	ParentID   string `json:"parent_id,omitempty"`
	MoveParent bool   `json:"move_parent,omitempty"`
}

// GazePageMoveRequest relocates the page identified by Slug into FolderID
// ("" = root). Slug is carried in the body (not the path) because gaze slugs
// are hierarchical ("entity/acme") and would collide with gin's catch-all.
type GazePageMoveRequest struct {
	Slug     string `json:"slug" binding:"required"`
	FolderID string `json:"folder_id"`
}

// GazeConfig stores gaze-specific configuration for a knowledge base.
// Applicable to document-type knowledge bases with gaze feature enabled.
// Whether the gaze feature is turned on is controlled by IndexingStrategy.GazeEnabled;
// this struct only carries gaze-specific tunables.
type GazeConfig struct {
	// SynthesisModelID is the LLM model ID used for gaze page generation and updates
	SynthesisModelID string `yaml:"synthesis_model_id" json:"synthesis_model_id"`
	// ContentInstructions controls tone, structure and emphasis for generated
	// summary/entity/index prose. Citation and merge rules remain system-owned.
	ContentInstructions string `yaml:"content_instructions,omitempty" json:"content_instructions,omitempty"`
}

// Value implements the driver.Valuer interface
func (c GazeConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan implements the sql.Scanner interface
func (c *GazeConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, c)
}

// GazePageListRequest represents a request to list gaze pages with filtering
type GazePageListRequest struct {
	KnowledgeBaseID string      `json:"knowledge_base_id"`
	PageType        string      `json:"page_type,omitempty"`      // filter by type
	Status          string      `json:"status,omitempty"`         // filter by status
	Query           string      `json:"query,omitempty"`          // full-text search
	FolderID        *string     `json:"folder_id,omitempty"`      // exact folder placement ("" = root)
	CategoryPath    StringArray `json:"category_path,omitempty"`  // exact directory path
	CategoryDepth   *int        `json:"category_depth,omitempty"` // exact directory depth, including 0 for root
	Page            int         `json:"page,omitempty"`           // pagination page (1-based)
	PageSize        int         `json:"page_size,omitempty"`      // pagination size
	SortBy          string      `json:"sort_by,omitempty"`        // "updated_at", "created_at", "title"
	SortOrder       string      `json:"sort_order,omitempty"`     // "asc" or "desc"
}

// GazePageListResponse represents a paginated list of gaze pages
type GazePageListResponse struct {
	Pages      []*GazePage `json:"pages"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// GazeGraphMode enumerates the graph query modes exposed to the API.
const (
	// GazeGraphModeOverview returns the top-N most-connected pages as an
	// overview of the knowledge base. Intended for the first graph open.
	GazeGraphModeOverview = "overview"
	// GazeGraphModeEgo returns the neighborhood around a center page up to a
	// configurable depth. Intended for drill-down interactions.
	GazeGraphModeEgo = "ego"
)

// GazeGraphRequest is the service-layer input for graph queries. It is
// populated by the HTTP handler from query params and passed down to the
// service, which is responsible for enforcing mode-specific semantics.
//
// Limit policy: a non-positive `Limit` means "no cap" and is reserved for
// internal callers (e.g. gaze lint) that need the full graph. The HTTP
// handler always clamps `Limit` into a safe range before calling the
// service so external traffic can never request an uncapped graph.
type GazeGraphRequest struct {
	KnowledgeBaseID string
	Mode            string   // "overview" (default) | "ego"
	Center          string   // ego mode center slug (required when Mode == "ego")
	Depth           int      // ego mode BFS depth, >= 1
	Types           []string // optional page_type filter; empty = no filter
	Limit           int      // max nodes to return; <= 0 means uncapped
	// FamiliarKnowledgeIDs are documents this person keeps drawing answers
	// from. Pages whose source_refs intersect the set are marked Familiar so
	// the existing Gaze graph can light them up without cloning a second graph.
	FamiliarKnowledgeIDs []string
}

// GazeGraphData represents the link graph structure for visualization.
type GazeGraphData struct {
	Nodes []GazeGraphNode `json:"nodes"`
	Edges []GazeGraphEdge `json:"edges"`
	Meta  GazeGraphMeta   `json:"meta"`
}

// GazeGraphMeta describes how the returned subgraph relates to the full
// knowledge base graph. The frontend uses `Truncated` to decide whether to
// surface a "showing X of Y" hint and to enable ego-expansion UI.
type GazeGraphMeta struct {
	Mode      string `json:"mode"`
	Total     int    `json:"total"`            // total node count in the KB before filtering/limit
	Returned  int    `json:"returned"`         // number of nodes actually returned
	Truncated bool   `json:"truncated"`        // true when Returned < Total (after filters)
	Center    string `json:"center,omitempty"` // populated in ego mode
	Depth     int    `json:"depth,omitempty"`  // populated in ego mode
	// FamiliarCount is how many returned nodes are lit up for this person.
	FamiliarCount int `json:"familiar_count,omitempty"`
}

// GazeGraphNode represents a node in the gaze link graph
type GazeGraphNode struct {
	Slug     string `json:"slug"`
	Title    string `json:"title"`
	PageType string `json:"page_type"`
	// Number of inbound + outbound links
	LinkCount int `json:"link_count"`
	// Familiar is true when this page was built from a document this person
	// keeps citing in answers. It is a personal overlay, not a property of
	// the page: two people looking at the same gaze see different highlights.
	Familiar bool `json:"familiar,omitempty"`
}

// GazeGraphEdge represents a directed edge in the gaze link graph
type GazeGraphEdge struct {
	Source string `json:"source"` // source slug
	Target string `json:"target"` // target slug
}

// GazeStats provides aggregate statistics about the gaze
type GazeStats struct {
	TotalPages    int64            `json:"total_pages"`
	PagesByType   map[string]int64 `json:"pages_by_type"`
	TotalLinks    int64            `json:"total_links"`
	OrphanCount   int64            `json:"orphan_count"`   // pages with no inbound links
	RecentUpdates []*GazePage      `json:"recent_updates"` // last N updated pages
	PendingTasks  int64            `json:"pending_tasks"`  // number of documents waiting to be ingested
	IsActive      bool             `json:"is_active"`      // whether gaze ingestion is currently running
}

// GazeIndexEntry is a single row in the structured gaze index response.
// Only the columns needed to render a clickable directory entry are
// carried — the backend projects SELECT slug, title, summary so a 40k-
// page KB does not pay for TEXT content transport on every index open.
type GazeIndexEntry struct {
	Slug         string      `json:"slug"`
	Title        string      `json:"title"`
	Summary      string      `json:"summary"`
	ParentSlug   string      `json:"parent_slug,omitempty"`
	CategoryPath StringArray `json:"category_path,omitempty"`
	GazePath     string      `json:"gaze_path,omitempty"`
	Depth        int         `json:"depth,omitempty"`
	SortOrder    int         `json:"sort_order,omitempty"`
}

// GazeIndexGroup bundles the entries for one page_type into a page-sized
// slice. `Total` is the full count across the KB for the type; `Items`
// holds the current paginated window starting at `NextOffset - len(Items)`.
// An empty NextCursor means the window is already at the end of the type.
type GazeIndexGroup struct {
	Type       string           `json:"type"`
	Total      int64            `json:"total"`
	Items      []GazeIndexEntry `json:"items"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

// GazeIndexResponse is what GET /gaze/index returns. The heavy directory
// markdown that used to sit in gaze_pages.content is gone — only the LLM-
// generated intro survives there. Everything else is assembled on demand
// from the index repo's light-column projection, keeping index reads
// O(page_size) regardless of KB size.
type GazeIndexResponse struct {
	Intro   string           `json:"intro"`
	Version int              `json:"version"`
	Groups  []GazeIndexGroup `json:"groups"`
}

// GazePageLite is a slim projection of GazePage carrying only the fields
// the gaze ingest pipeline reaches for during Map / Reduce. It exists so
// per-batch fetcher queries don't have to load the full multi-MB content
// column for every page they want a title or out-link from.
//
// Use cases:
//
//   - SlugTitleFetcher: resolve slug -> title for cross-link injection.
//   - cleanDeadLinks: read out_links + status without pulling content.
//   - dedup pre-filter: title + aliases + page_type for the trgm /
//     surface-similarity comparisons.
//
// Aliases is included because dedup and cross-link injection both treat
// the alias surface forms as first-class match targets; OutLinks is
// included so dead-link cleanup can determine which pages reference a
// given dead slug without a second query.
type GazePageLite struct {
	Slug     string      `json:"slug"`
	Title    string      `json:"title"`
	PageType string      `json:"page_type"`
	Status   string      `json:"status"`
	Aliases  StringArray `json:"aliases,omitempty"`
	OutLinks StringArray `json:"out_links,omitempty"`
}

// GazeSourceKnowledgeID extracts the knowledge id from a source_refs entry,
// stored as "uuid" or "uuid|title".
func GazeSourceKnowledgeID(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if i := strings.IndexByte(ref, '|'); i > 0 {
		return strings.TrimSpace(ref[:i])
	}
	return ref
}

// SourceKnowledgeIDs returns the document ids this page was built from.
func (p *GazePage) SourceKnowledgeIDs() []string {
	if p == nil || len(p.SourceRefs) == 0 {
		return nil
	}
	ids := make([]string, 0, len(p.SourceRefs))
	for _, ref := range p.SourceRefs {
		if id := GazeSourceKnowledgeID(ref); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// BuiltFrom reports whether any of this page's sources is in the given set.
func (p *GazePage) BuiltFrom(knowledgeIDs map[string]struct{}) bool {
	if p == nil || len(knowledgeIDs) == 0 {
		return false
	}
	for _, id := range p.SourceKnowledgeIDs() {
		if _, ok := knowledgeIDs[id]; ok {
			return true
		}
	}
	return false
}
