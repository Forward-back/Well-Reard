package interfaces

import (
	"context"

	"gaze/internal/types"
)

// GazePageService defines the gaze page service interface.
// Provides high-level operations for gaze page CRUD, link management,
// and chunk synchronization.
type GazePageService interface {
	// CreatePage creates a new gaze page, parses outbound links, updates
	// bidirectional link references, and syncs to chunks for retrieval.
	CreatePage(ctx context.Context, page *types.GazePage) (*types.GazePage, error)

	// UpdatePage updates an existing gaze page, re-parses links, and updates
	// bidirectional references. The `version` field is incremented only when
	// a user-visible content field (title, content, summary, page_type,
	// status) actually differs from the stored value — bookkeeping-only
	// writes (e.g. refreshing source_refs after a same-content re-ingest) are
	// persisted without bumping the version.
	UpdatePage(ctx context.Context, page *types.GazePage) (*types.GazePage, error)

	// UpdatePageMeta updates only metadata (status, source_refs, etc.) without
	// incrementing version or re-parsing links. Use for publish/archive/source
	// ref changes driven by internal reconciliation.
	UpdatePageMeta(ctx context.Context, page *types.GazePage) error

	// UpdateAutoLinkedContent persists content changes that come from
	// machine-only link decoration — cross-link injection and dead-link
	// cleanup — without bumping `version`. Out-links are re-parsed from the
	// new body and bidirectional in-link references on target pages are
	// refreshed. Conceptually the body still represents the same revision
	// the user saw before, just with gaze-link markup added or stripped, so
	// surfacing a version bump to end users is misleading.
	UpdateAutoLinkedContent(ctx context.Context, page *types.GazePage) error

	// GetPageBySlug retrieves a gaze page by its slug within a knowledge base.
	GetPageBySlug(ctx context.Context, kbID string, slug string) (*types.GazePage, error)

	// RepairContentLinks rewrites dead [[slug]] references in content to their
	// most likely live target (rewrite-only — never strips). Used by the agent
	// write path to auto-correct LLM-mangled slugs (especially UUID-based
	// summary slugs) before persistence. Returns the possibly-updated content
	// and whether any rewrite happened. Best-effort: callers may ignore errors.
	RepairContentLinks(ctx context.Context, kbID, selfSlug, content string) (string, bool, error)

	// GetPageByID retrieves a gaze page by its unique ID.
	GetPageByID(ctx context.Context, id string) (*types.GazePage, error)

	// ListPages lists gaze pages with optional filtering and pagination.
	ListPages(ctx context.Context, req *types.GazePageListRequest) (*types.GazePageListResponse, error)

	// DeletePage soft-deletes a gaze page and removes its chunk sync.
	DeletePage(ctx context.Context, kbID string, slug string) error

	// GetIndex returns the index page for a knowledge base.
	// Creates a default one if it doesn't exist.
	//
	// The index row now carries only the LLM-generated intro in its content
	// column — the heavy directory listing was moved out of gaze_pages.
	// Callers that want a renderable directory should use GetIndexView,
	// which assembles {intro, groups} structured output on demand.
	GetIndex(ctx context.Context, kbID string) (*types.GazePage, error)

	// GetIndexView returns the structured index response — intro (from the
	// index row) plus a paginated window of directory entries per requested
	// page_type. `types` restricts which page types to include (empty =
	// all), `limit` bounds the per-type window size, `cursor` is an opaque
	// offset string resumed from a previous response.
	//
	// Keeping the directory as JSON rather than a multi-MB markdown blob
	// means a 40k-page KB no longer pays O(N) transport + rendering cost
	// on every index open. See /gaze/index handler + GazeBrowser.vue for
	// the consumer.
	GetIndexView(ctx context.Context, kbID string, pageTypes []string, limit int, cursor string) (*types.GazeIndexResponse, error)

	// GetGraph returns the link graph data for visualization. The caller
	// supplies a GazeGraphRequest describing the desired slice of the graph
	// (overview top-N or ego neighborhood around a center slug). Callers
	// that need the full graph (e.g. gaze lint) can set Limit <= 0 to
	// disable the node cap; the HTTP handler always clamps Limit to a
	// safe range before invoking the service.
	GetGraph(ctx context.Context, req *types.GazeGraphRequest) (*types.GazeGraphData, error)

	// GetStats returns aggregate statistics about the gaze.
	GetStats(ctx context.Context, kbID string) (*types.GazeStats, error)

	// RebuildLinks re-parses all pages and rebuilds bidirectional link references.
	RebuildLinks(ctx context.Context, kbID string) error

	// InjectCrossLinks scans specified pages and injects [[gaze-links]] for mentions
	// of other gaze page titles/aliases in the content.
	InjectCrossLinks(ctx context.Context, kbID string, affectedSlugs []string)

	// RebuildIndexPage regenerates the index page.
	RebuildIndexPage(ctx context.Context, kbID string) error

	// ListAllPages retrieves all non-archived gaze pages in a knowledge base without pagination.
	// Used for index rebuild, graph generation, cross-link injection, etc.
	ListAllPages(ctx context.Context, kbID string) ([]*types.GazePage, error)

	// ListByType returns every gaze page of the given type. Used by
	// callers that need the full set for a specific type (e.g. intro
	// regeneration needs summary pages). For directory rendering, use
	// GetIndexView instead — it paginates via ListByTypeLight.
	ListByType(ctx context.Context, kbID string, pageType string) ([]*types.GazePage, error)

	// ListPagesBySourceRef retrieves all gaze pages whose source_refs reference
	// the given knowledge ID. Used by delete/ingest reconciliation paths that
	// need to find pages touched by a specific document at read time (rather
	// than relying on a caller-provided stale snapshot).
	ListPagesBySourceRef(ctx context.Context, kbID string, knowledgeID string) ([]*types.GazePage, error)

	// ListSlugsBySourceRef returns just the slugs of pages that cite the
	// given knowledge id. Cheaper than ListPagesBySourceRef when callers
	// only need the slug set (e.g. gaze ingest's "before" snapshot).
	ListSlugsBySourceRef(ctx context.Context, kbID string, knowledgeID string) ([]string, error)

	// ListBySlugs is the lazy fetcher used by the gaze ingest batch
	// context: returns lightweight projections (no content / source_refs)
	// for the requested slugs in a single IN query. Used in place of
	// the pre-batch ListAllPages dump.
	ListBySlugs(ctx context.Context, kbID string, slugs []string) (map[string]*types.GazePageLite, error)

	// ListSummariesByKnowledgeIDs returns summary-page content keyed by
	// the knowledge id that authored it. Used by the retract / reparse
	// branches of reduceSlugUpdates for "what was this doc's
	// contribution?" framing.
	ListSummariesByKnowledgeIDs(ctx context.Context, kbID string, kids []string) (map[string]string, error)

	// ExistsSlugs reports which of the given slugs are live (non-archived).
	// Used by cleanDeadLinks to validate out-link targets cheaply.
	ExistsSlugs(ctx context.Context, kbID string, slugs []string) (map[string]bool, error)

	// ListAllSlugs returns every non-archived slug in the KB. Used by
	// lint to compute the live-slug set for broken-link detection.
	ListAllSlugs(ctx context.Context, kbID string) ([]string, error)

	// ListPagesCursor walks the KB in id-asc order with an opaque
	// cursor. Used by lint to stream the page set without holding it
	// all in memory at once.
	ListPagesCursor(ctx context.Context, kbID string, cursor string, limit int) ([]*types.GazePage, string, error)

	// ListByTypeRecent returns the most-recently-updated pages of the
	// given type, projected to slug/title/summary, capped at `limit`.
	// Used by rebuildIndexPage's first-time generation path.
	ListByTypeRecent(ctx context.Context, kbID string, pageType string, limit int) ([]types.GazeIndexEntry, error)

	// FindSimilarPages performs a pg_trgm similarity search over
	// page titles. Used by the dedup pre-filter to surface candidate
	// merge targets server-side.
	FindSimilarPages(ctx context.Context, kbID string, query string, pageTypes []string, limit int) ([]*types.GazePageLite, error)

	// FindPagesByNormalizedTitle returns non-archived pages of pageType whose
	// display title matches identity after the same whitespace/case fold used
	// by gaze ingest identity claims. Used so exact same-title pages are found
	// even when they miss the trigram top-K.
	FindPagesByNormalizedTitle(ctx context.Context, kbID, pageType, identity string) ([]*types.GazePageLite, error)

	// FindPagesByNormalizedTitles is the batched form of
	// FindPagesByNormalizedTitle. identities are already whitespace-stripped
	// and lowercased; empty entries are ignored.
	FindPagesByNormalizedTitles(ctx context.Context, kbID, pageType string, identities []string) ([]*types.GazePageLite, error)

	// ListDistinctCategoryPaths returns the existing gaze folder paths (split
	// into segments), capped at maxPaths. Used by gaze ingest's taxonomy
	// planner as the pool of folders to reuse.
	ListDistinctCategoryPaths(ctx context.Context, kbID string, maxPaths int) ([][]string, error)

	// --- Folder tree (gaze_folders) ---

	// ListChildFolders returns the direct child folders of parentID ("" =
	// root) relevant to the given page types, each enriched with a recursive
	// page count (the folder's whole subtree, restricted to pageTypes) and
	// whether it has relevant sub-folders. A folder is shown when its subtree
	// holds at least one page of pageTypes, or when it is entirely empty (a
	// user-created container with no pages of any type). When pageTypes is
	// empty every type counts.
	ListChildFolders(
		ctx context.Context, kbID string, parentID string, pageTypes []string,
	) ([]types.GazeFolderNode, error)
	// GetFolder retrieves a single folder by id.
	GetFolder(ctx context.Context, kbID string, id string) (*types.GazeFolder, error)
	// CreateFolder creates a new (initially empty) folder under parentID and
	// returns it. Fails if a sibling with the same name already exists.
	CreateFolder(ctx context.Context, kbID string, tenantID uint64, parentID string, name string) (*types.GazeFolder, error)
	// RenameOrMoveFolder renames and/or reparents a folder, recomputing the
	// materialized path/depth of the whole subtree and the cached paths of
	// every page underneath. Returns the updated folder.
	RenameOrMoveFolder(ctx context.Context, kbID string, id string, newName string, newParentID string, moveParent bool) (*types.GazeFolder, error)
	// DeleteFolder removes an empty folder. Fails if it still contains pages
	// or child folders (the UI must move or delete contents first).
	DeleteFolder(ctx context.Context, kbID string, id string) error
	// PruneEmptyFolderChains deletes candidate folders that are still empty,
	// walking upward through newly-empty ancestors. Candidates are supplied by
	// document retract cleanup so unrelated user-created empty folders remain.
	PruneEmptyFolderChains(ctx context.Context, kbID string, folderIDs []string) ([]string, error)
	// FindOrCreateFolderPath resolves a category path (e.g. ["AI","RAG"]) to a
	// folder id, creating any missing intermediate folders. Returns the leaf
	// folder id and the canonical (cleaned) path. An empty/blank path resolves
	// to the root ("").
	FindOrCreateFolderPath(ctx context.Context, kbID string, tenantID uint64, path []string) (string, []string, error)
	// MovePage relocates a page into folderID ("" = root), recomputing its
	// cached category path. Returns the updated page.
	MovePage(ctx context.Context, kbID string, slug string, folderID string) (*types.GazePage, error)

	// CountByType returns page counts grouped by type for a knowledge
	// base. Re-exposed at the service layer so the index intro
	// generation path can frame the LLM prompt with "showing N of M"
	// metadata when the recent-N projection doesn't cover the full
	// summary set.
	CountByType(ctx context.Context, kbID string) (map[string]int64, error)

	// SearchPages performs full-text search over gaze pages.
	SearchPages(ctx context.Context, kbID string, query string, limit int) ([]*types.GazePage, error)

	// ListRevisions returns the stored historical snapshots for a page
	// (newest first, content omitted) plus the total snapshot count and the
	// page's current version. The current version itself has no revision
	// row — it lives in gaze_pages.
	ListRevisions(ctx context.Context, kbID string, slug string, limit int, offset int) (*types.GazePageRevisionListResponse, error)

	// GetRevision returns one historical snapshot (content included).
	GetRevision(ctx context.Context, kbID string, slug string, version int) (*types.GazePageRevision, error)

	// RevertPageToVersion rolls the page back to the content of the given
	// stored revision by applying it as a regular edit: the pre-revert state
	// is snapshotted, version advances, and links are re-parsed. The edit is
	// attributed to GazeEditSourceRevert.
	RevertPageToVersion(ctx context.Context, kbID string, slug string, version int) (*types.GazePage, error)

	// SyncOutdated walks the KB's summary pages, groups them by doc_number
	// (UMS 文号, resolved through each page's source knowledge), and marks
	// every page except the newest in each group as outdated with
	// superseded_by_slug pointing at the current page. Returns the number of
	// pages whose outdated state changed.
	SyncOutdated(ctx context.Context, kbID string) (int, error)
}

// GazePageRepository defines the gaze page data persistence interface.
type GazePageRepository interface {
	// Create inserts a new gaze page record.
	Create(ctx context.Context, page *types.GazePage) error

	// Update rewrites a gaze page record with optimistic locking and
	// unconditionally increments `version`. Callers are responsible for
	// deciding whether the edit is user-visible — the service layer uses
	// UpdateMeta for bookkeeping-only writes instead.
	Update(ctx context.Context, page *types.GazePage) error

	// UpdateMeta updates bookkeeping / provenance fields (in/out links,
	// status, source_refs, chunk_refs, page_metadata, updated_at) without
	// touching `version`. Safe for link maintenance, re-ingest with an
	// unchanged body, and status-only transitions.
	UpdateMeta(ctx context.Context, page *types.GazePage) error

	// UpdateAutoLinkedContent rewrites `content`, `out_links` and
	// `updated_at` in place while leaving `version` untouched. Intended for
	// machine-only link markup changes (cross-link injection / dead-link
	// cleanup) so the first-ingest page doesn't jump to v2 just because the
	// post-processor added a `[[...]]` wrapper.
	UpdateAutoLinkedContent(ctx context.Context, page *types.GazePage) error

	// GetByID retrieves a gaze page by its unique ID.
	GetByID(ctx context.Context, id string) (*types.GazePage, error)

	// GetBySlug retrieves a gaze page by slug within a knowledge base.
	GetBySlug(ctx context.Context, kbID string, slug string) (*types.GazePage, error)

	// List retrieves gaze pages with filtering and pagination.
	List(ctx context.Context, req *types.GazePageListRequest) ([]*types.GazePage, int64, error)

	// ListByType retrieves all gaze pages of a given type within a knowledge base.
	ListByType(ctx context.Context, kbID string, pageType string) ([]*types.GazePage, error)

	// ListByTypeLight returns a paginated window of lightweight entries
	// (slug/title/summary only) for the given page_type plus the total
	// non-archived count. Used by the structured index API so reads do not
	// have to materialize TEXT content for every gaze_pages row.
	ListByTypeLight(ctx context.Context, kbID string, pageType string, limit int, offset int) ([]types.GazeIndexEntry, int64, error)

	// ListBySourceRef retrieves all gaze pages that reference a given source knowledge ID.
	ListBySourceRef(ctx context.Context, kbID string, sourceKnowledgeID string) ([]*types.GazePage, error)

	// ListSlugsBySourceRef returns just the slugs of pages that reference
	// the given knowledge id. Same predicate as ListBySourceRef but
	// projected to a single column for the gaze ingest pipeline's
	// "before" snapshot path.
	ListSlugsBySourceRef(ctx context.Context, kbID string, sourceKnowledgeID string) ([]string, error)

	// ListBySlugs returns lightweight projections (slug/title/page_type/
	// status/aliases/out_links) for the given slugs in one IN query.
	// Used by the lazy fetcher path in gaze ingest's Map/Reduce phases.
	ListBySlugs(ctx context.Context, kbID string, slugs []string) (map[string]*types.GazePageLite, error)

	// ListSummariesByKnowledgeIDs returns summary-page content keyed by
	// the knowledge id that authored it. Used by the retract branch of
	// reduceSlugUpdates for "what was this doc's contribution?" framing.
	ListSummariesByKnowledgeIDs(ctx context.Context, kbID string, kids []string) (map[string]string, error)

	// ExistsSlugs reports which of the given slugs are live (non-
	// archived). Used by cleanDeadLinks to validate out-link targets
	// without loading the referenced pages' content.
	ExistsSlugs(ctx context.Context, kbID string, slugs []string) (map[string]bool, error)

	// ListAllSlugs returns every non-archived slug in the KB. Used by
	// lint to compute the live-slug set without paying for ListAll's
	// full row materialization.
	ListAllSlugs(ctx context.Context, kbID string) ([]string, error)

	// ListPagesCursor returns up to `limit` pages ordered by id ASC,
	// paginated by an opaque numeric cursor. Used by lint to stream
	// the KB without loading every page at once.
	ListPagesCursor(ctx context.Context, kbID string, cursor string, limit int) ([]*types.GazePage, string, error)

	// ListByTypeRecent returns the most-recently-updated pages of the
	// given type, projected to slug/title/summary, capped at `limit`.
	// Used by rebuildIndexPage's first-time generation path so the
	// LLM prompt size is bounded on large KBs.
	ListByTypeRecent(ctx context.Context, kbID string, pageType string, limit int) ([]types.GazeIndexEntry, error)

	// FindSimilarPages returns the top-k pages whose lowercase title
	// is most similar to the query under pg_trgm. `pageTypes` empty
	// defaults to entity+concept. Used by the dedup pre-filter to
	// surface candidate merge targets server-side.
	FindSimilarPages(ctx context.Context, kbID string, query string, pageTypes []string, limit int) ([]*types.GazePageLite, error)

	// FindPagesByNormalizedTitle returns non-archived pages of pageType whose
	// whitespace-stripped, lowercased title equals identity.
	FindPagesByNormalizedTitle(ctx context.Context, kbID, pageType, identity string) ([]*types.GazePageLite, error)

	// FindPagesByNormalizedTitles is the batched form of
	// FindPagesByNormalizedTitle.
	FindPagesByNormalizedTitles(ctx context.Context, kbID, pageType string, identities []string) ([]*types.GazePageLite, error)

	// ListDistinctCategoryPaths returns the materialized paths of existing
	// gaze folders (split into segments), capped at maxPaths. Used by the
	// gaze ingest taxonomy planner as the pool of folders to reuse.
	ListDistinctCategoryPaths(ctx context.Context, kbID string, maxPaths int) ([][]string, error)

	// --- Folder tree (gaze_folders) ---

	// CreateFolder inserts a new directory node.
	CreateFolder(ctx context.Context, folder *types.GazeFolder) error
	// GetFolderByID retrieves a folder by id within a knowledge base.
	GetFolderByID(ctx context.Context, kbID string, id string) (*types.GazeFolder, error)
	// GetChildFolderByName returns the live child folder of parentID with the
	// given name, or ErrGazeFolderNotFound. Used by find-or-create.
	GetChildFolderByName(ctx context.Context, kbID string, parentID string, name string) (*types.GazeFolder, error)
	// ListChildFolders returns the direct child folders of parentID, ordered
	// by sort_order then name.
	ListChildFolders(ctx context.Context, kbID string, parentID string) ([]*types.GazeFolder, error)
	// ListAllFolders returns every folder in the knowledge base.
	ListAllFolders(ctx context.Context, kbID string) ([]*types.GazeFolder, error)
	// UpdateFolder rewrites a folder's mutable fields (name, parent_id, path,
	// depth, sort_order, updated_at).
	UpdateFolder(ctx context.Context, folder *types.GazeFolder) error
	// DeleteFolder soft-deletes a folder by id.
	DeleteFolder(ctx context.Context, kbID string, id string) error
	// CountPagesInFolder returns the number of live (non-archived) pages
	// directly in the folder (folder_id = id).
	CountPagesInFolder(ctx context.Context, kbID string, folderID string) (int64, error)
	// CountPagesByFolder returns live (non-archived) page counts grouped by
	// folder_id. When pageTypes is non-empty the count is restricted to those
	// page types; otherwise all types are counted. Pages at the gaze root carry
	// the empty-string key.
	CountPagesByFolder(ctx context.Context, kbID string, pageTypes []string) (map[string]int64, error)
	// ListPagesByFolderIDs returns all pages whose folder_id is in the set.
	// Used to recompute cached paths when a folder subtree is moved/renamed.
	ListPagesByFolderIDs(ctx context.Context, kbID string, folderIDs []string) ([]*types.GazePage, error)

	// ListAll retrieves all non-archived gaze pages in a knowledge base (for link rebuilding, graph generation).
	ListAll(ctx context.Context, kbID string) ([]*types.GazePage, error)

	// ListRecentForSuggestions returns recent user-visible gaze pages under the given
	// knowledge bases, used to produce fallback suggested questions for Gaze-only KBs
	// that do not have AI-generated document questions or recommended FAQ entries.
	// Excludes the index page and archived pages. Returns up to `limit` rows sorted
	// by updated_at descending.
	ListRecentForSuggestions(ctx context.Context, tenantID uint64, kbIDs []string, limit int) ([]*types.GazePage, error)

	// Delete soft-deletes a gaze page by knowledge base ID and slug.
	Delete(ctx context.Context, kbID string, slug string) error

	// DeleteByID soft-deletes a gaze page by ID.
	DeleteByID(ctx context.Context, id string) error

	// Search performs full-text search on gaze pages within a knowledge base.
	Search(ctx context.Context, kbID string, query string, limit int) ([]*types.GazePage, error)

	// CountByType returns page counts grouped by type for a knowledge base.
	CountByType(ctx context.Context, kbID string) (map[string]int64, error)

	// CountOrphans returns the number of pages with no inbound links.
	CountOrphans(ctx context.Context, kbID string) (int64, error)

	// UpdateWithRevision snapshots the version being superseded and applies
	// the page update in a single transaction, so history can never contain
	// a snapshot of a version that is still current. Inserting an already-
	// present (page_id, version) pair is a silent no-op.
	UpdateWithRevision(ctx context.Context, page *types.GazePage, rev *types.GazePageRevision) error

	// ListRevisions returns snapshots for a page newest-first (content
	// column omitted) plus the total snapshot count.
	ListRevisions(ctx context.Context, kbID string, pageID string, limit int, offset int) ([]*types.GazePageRevision, int64, error)

	// GetRevision returns one snapshot with content.
	GetRevision(ctx context.Context, kbID string, pageID string, version int) (*types.GazePageRevision, error)

	// PruneRevisions bounds per-page history storage using the two-tier
	// retention described by the request.
	PruneRevisions(ctx context.Context, req types.GazeRevisionPruneRequest) error

	// DeleteRevisionsByPage hard-deletes a page's entire snapshot history.
	DeleteRevisionsByPage(ctx context.Context, pageID string) error
}
