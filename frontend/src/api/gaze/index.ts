import { get, post, put, del } from "../../utils/request";

// encodeSlugPath encodes each segment of a hierarchical gaze slug (e.g.
// "foo/bar baz?") so the URL is safe while preserving the "/" separators
// between segments. Using encodeURIComponent on the whole slug would also
// escape the "/" and break hierarchical routing on the backend.
function encodeSlugPath(slug: string): string {
  return slug.split("/").map(encodeURIComponent).join("/");
}

// Gaze Page Types
export interface GazePage {
  id: string;
  tenant_id: number;
  knowledge_base_id: string;
  slug: string;
  title: string;
  page_type: string;
  status: string;
  content: string;
  summary: string;
  aliases: string[];
  parent_slug?: string;
  category_path?: string[];
  gaze_path?: string;
  depth?: number;
  sort_order?: number;
  source_refs: string[];
  in_links: string[];
  out_links: string[];
  page_metadata: Record<string, any>;
  version: number;
  // Author kind of the current version: 'pipeline' | 'agent' | 'user' |
  // 'revert'. Empty/missing on legacy rows (treat as 'pipeline').
  last_edit_source?: string;
  last_editor_id?: string;
  created_at: string;
  updated_at: string;
}

export interface GazePageListResponse {
  pages: GazePage[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface GazeFolder {
  id: string;
  tenant_id: number;
  knowledge_base_id: string;
  parent_id: string;
  name: string;
  path: string;
  depth: number;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export interface GazeFolderNode extends GazeFolder {
  page_count: number;
  has_children: boolean;
}

export interface GazeFolderListResponse {
  parent_id: string;
  folders: GazeFolderNode[];
}

export interface GazeGraphMeta {
  mode: 'overview' | 'ego' | string;
  total: number;
  returned: number;
  truncated: boolean;
  center?: string;
  depth?: number;
  familiar_count?: number;
}

export interface GazeGraphData {
  nodes: { slug: string; title: string; page_type: string; link_count: number; familiar?: boolean }[];
  edges: { source: string; target: string }[];
  meta: GazeGraphMeta;
}

export interface GazeStats {
  total_pages: number;
  pages_by_type: Record<string, number>;
  total_links: number;
  orphan_count: number;
  recent_updates: GazePage[];
  pending_tasks: number;
  pending_issues: number;
  is_active: boolean;
}

export interface GazePageIssue {
  id: string;
  tenant_id: number;
  knowledge_base_id: string;
  slug: string;
  issue_type: string;
  description: string;
  suspected_knowledge_ids: string[];
  status: string;
  reported_by: string;
  created_at: string;
  updated_at: string;
}

// Gaze API Functions
export function listGazePages(kbId: string, params?: {
  page_type?: string;
  status?: string;
  query?: string;
  category_path?: string;
  category_depth?: number;
  page?: number;
  page_size?: number;
  sort_by?: string;
  sort_order?: string;
}) {
  const query = new URLSearchParams();
  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== '') {
        query.set(key, String(value));
      }
    });
  }
  const qs = query.toString();
  return get(`/api/v1/knowledgebase/${kbId}/gaze/pages${qs ? '?' + qs : ''}`);
}

// listGazeFolders returns the direct child folders of parentId ("" = root),
// each enriched with a recursive page_count and a has_children flag so the tree
// can render expand affordances and empty folders without a second request.
// pageTypes scopes the view to a sidebar tab: only folders whose subtree holds
// a page of those types (or are entirely empty) come back, and page_count is
// counted within those types.
export function listGazeFolders(kbId: string, parentId = '', pageTypes = '') {
  const query = new URLSearchParams();
  if (parentId) query.set('parent_id', parentId);
  if (pageTypes) query.set('page_types', pageTypes);
  const qs = query.toString();
  return get(`/api/v1/knowledgebase/${kbId}/gaze/folders${qs ? '?' + qs : ''}`);
}

// createGazeFolder creates a new empty folder under parentId ("" = root).
export function createGazeFolder(kbId: string, parentId: string, name: string) {
  return post(`/api/v1/knowledgebase/${kbId}/gaze/folders`, { parent_id: parentId, name });
}

// updateGazeFolder renames and/or reparents a folder. Pass move_parent: true
// (and parent_id) to reparent; omit it for a pure rename.
export function updateGazeFolder(
  kbId: string,
  folderId: string,
  data: { name?: string; parent_id?: string; move_parent?: boolean },
) {
  return put(`/api/v1/knowledgebase/${kbId}/gaze/folders/${folderId}`, data);
}

// deleteGazeFolder removes an empty folder (no pages, no sub-folders).
export function deleteGazeFolder(kbId: string, folderId: string) {
  return del(`/api/v1/knowledgebase/${kbId}/gaze/folders/${folderId}`);
}

// moveGazePage relocates a page into folderId ("" = root). The slug is sent in
// the body because gaze slugs are hierarchical.
export function moveGazePage(kbId: string, slug: string, folderId: string) {
  return put(`/api/v1/knowledgebase/${kbId}/gaze/move-page`, { slug, folder_id: folderId });
}

export function createGazePage(kbId: string, data: Partial<GazePage>) {
  return post(`/api/v1/knowledgebase/${kbId}/gaze/pages`, data);
}

export function getGazePage(kbId: string, slug: string) {
  return get(`/api/v1/knowledgebase/${kbId}/gaze/pages/${encodeSlugPath(slug)}`);
}

// GazePageUpdatePayload is a partial update: absent fields keep their stored
// value. `version` is the optimistic-lock guard — send the version the page
// had when the user started editing; the backend answers 409 (with
// `current_version` in the body) when someone else edited in between.
export interface GazePageUpdatePayload {
  title?: string;
  content?: string;
  summary?: string;
  page_type?: string;
  status?: string;
  aliases?: string[];
  version?: number;
}

export function updateGazePage(kbId: string, slug: string, data: GazePageUpdatePayload) {
  return put(`/api/v1/knowledgebase/${kbId}/gaze/pages/${encodeSlugPath(slug)}`, data);
}

export function deleteGazePage(kbId: string, slug: string) {
  return del(`/api/v1/knowledgebase/${kbId}/gaze/pages/${encodeSlugPath(slug)}`);
}

// GazePageRevision is one immutable snapshot of a superseded page version.
// `content` is only populated when fetching a single revision.
export interface GazePageRevision {
  id: string;
  tenant_id: number;
  knowledge_base_id: string;
  page_id: string;
  slug: string;
  version: number;
  title: string;
  page_type: string;
  status: string;
  content?: string;
  summary: string;
  aliases: string[];
  edit_source: string;
  editor_id: string;
  edited_at: string;
  created_at: string;
}

export interface GazeRevisionListResponse {
  revisions: GazePageRevision[];
  total: number;
  current_version: number;
}

// listGazeRevisions returns the page's historical snapshots newest-first
// (content omitted) plus the current version number. The current version has
// no revision row — it is the page itself.
export function listGazeRevisions(kbId: string, slug: string, params?: { limit?: number; offset?: number }) {
  const query = new URLSearchParams();
  if (params?.limit !== undefined) query.set('limit', String(params.limit));
  if (params?.offset !== undefined) query.set('offset', String(params.offset));
  const qs = query.toString();
  return get(`/api/v1/knowledgebase/${kbId}/gaze/revisions/${encodeSlugPath(slug)}${qs ? '?' + qs : ''}`);
}

// getGazeRevision returns one snapshot with full content.
export function getGazeRevision(kbId: string, slug: string, version: number) {
  return get(`/api/v1/knowledgebase/${kbId}/gaze/revisions/${encodeSlugPath(slug)}?version=${version}`);
}

// revertGazePage rolls the page back to a stored revision. Applied as a
// regular edit: the pre-revert state is snapshotted and version advances,
// so a revert is itself revertable.
export function revertGazePage(kbId: string, slug: string, version: number) {
  return post(`/api/v1/knowledgebase/${kbId}/gaze/revert`, { slug, version });
}

export interface GazeIndexEntryDTO {
  slug: string;
  title: string;
  summary: string;
  parent_slug?: string;
  category_path?: string[];
  gaze_path?: string;
  depth?: number;
  sort_order?: number;
}

export interface GazeIndexGroup {
  type: string;
  total: number;
  items: GazeIndexEntryDTO[];
  next_cursor?: string;
}

export interface GazeIndexResponse {
  intro: string;
  version: number;
  groups: GazeIndexGroup[];
}

// getGazeIndex fetches the structured index view for a gaze KB. The
// backend replaced the legacy "markdown blob of intro + directory" with
// { intro, groups } so a 40k-page gaze no longer round-trips multiple
// megabytes on every index open. Pass `types` to restrict which
// page_type buckets come back; `limit` bounds the per-group window;
// `cursor` resumes from a previous response.
export function getGazeIndex(
  kbId: string,
  params?: { types?: string[]; limit?: number; cursor?: string },
) {
  const query = new URLSearchParams();
  if (params) {
    if (params.types && params.types.length > 0) query.set('types', params.types.join(','));
    if (params.limit !== undefined) query.set('limit', String(params.limit));
    if (params.cursor) query.set('cursor', params.cursor);
  }
  const qs = query.toString();
  const suffix = qs ? `?${qs}` : '';
  return get(`/api/v1/knowledgebase/${kbId}/gaze/index${suffix}`);
}

export interface GazeGraphQueryParams {
  mode?: 'overview' | 'ego';
  center?: string;
  depth?: number;
  types?: string[];
  limit?: number;
}

// getGazeGraph fetches a slice of the gaze link graph. Without params the
// backend returns the top-500 most-connected pages (overview mode). Pass
// `mode: 'ego', center: <slug>` to drill into a specific page's neighborhood.
// For knowledge bases with tens of thousands of pages the overview cap is
// what prevents the browser from choking on a 30MB payload / 100k SVG nodes.
export function getGazeGraph(kbId: string, params?: GazeGraphQueryParams) {
  const query = new URLSearchParams();
  if (params) {
    if (params.mode) query.set('mode', params.mode);
    if (params.center) query.set('center', params.center);
    if (params.depth !== undefined) query.set('depth', String(params.depth));
    if (params.limit !== undefined) query.set('limit', String(params.limit));
    if (params.types && params.types.length > 0) {
      query.set('types', params.types.join(','));
    }
  }
  const qs = query.toString();
  return get(`/api/v1/knowledgebase/${kbId}/gaze/graph${qs ? '?' + qs : ''}`);
}

export function getGazeStats(kbId: string) {
  return get(`/api/v1/knowledgebase/${kbId}/gaze/stats`);
}

export function searchGazePages(kbId: string, q: string, limit?: number) {
  const params = new URLSearchParams({ q });
  if (limit) params.set('limit', String(limit));
  return get(`/api/v1/knowledgebase/${kbId}/gaze/search?${params.toString()}`);
}

export function listGazeIssues(kbId: string, slug?: string, status?: string) {
  const params = new URLSearchParams();
  if (slug) params.set('slug', slug);
  if (status) params.set('status', status);
  return get(`/api/v1/knowledgebase/${kbId}/gaze/issues?${params.toString()}`);
}

export function updateGazeIssueStatus(kbId: string, issueId: string, status: string) {
  return put(`/api/v1/knowledgebase/${kbId}/gaze/issues/${issueId}/status`, { status });
}

export function rebuildGazeLinks(kbId: string) {
  return post(`/api/v1/knowledgebase/${kbId}/gaze/rebuild-links`, {});
}
