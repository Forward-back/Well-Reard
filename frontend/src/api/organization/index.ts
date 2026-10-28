import { get } from '@/utils/request'

// Shared-space read models. Organization management was removed with its
// backend; only the department-shared KB projection and legacy space-scoped
// list types remain for existing list/detail views.

export interface SharedKnowledgeBase {
  knowledge_base: {
    id: string
    name: string
    description: string
    type: string
    knowledge_count?: number
    chunk_count?: number
    [key: string]: unknown
  }
  share_id: string
  organization_id: string
  org_name: string
  permission: 'admin' | 'editor' | 'viewer'
  source_tenant_id: number
  shared_at: string
}

/** When set, this KB is visible in the space via a shared agent (read-only, no direct KB share) */
export interface SourceFromAgentInfo {
  agent_id: string
  agent_name: string
  /** "all" | "selected" | "none" — for showing agent's KB strategy in the drawer */
  kb_selection_mode?: string
}

/** Legacy space-scoped list item kept for the remaining list-view type surface. */
export type OrganizationSharedKnowledgeBaseItem = SharedKnowledgeBase & {
  is_mine: boolean
  source_from_agent?: SourceFromAgentInfo
}

export interface SharedAgentInfo {
  agent: { id: string; name: string; description?: string; [key: string]: any }
  share_id: string
  organization_id: string
  org_name: string
  permission: string
  source_tenant_id: number
  shared_at: string
  shared_by_user_id?: string
  shared_by_username?: string
  web_search_ready: boolean
  disabled_by_me?: boolean
}

export interface ApiResponse<T> {
  success: boolean
  data?: T
  message?: string
}

/** Department-shared KB projection from GET /shared-space/knowledge-bases. */
export async function listSharedKnowledgeBases(): Promise<ApiResponse<SharedKnowledgeBase[]>> {
  try {
    const response: any = await get('/api/v1/shared-space/knowledge-bases')
    const rows = Array.isArray(response?.data) ? response.data : []
    const data: SharedKnowledgeBase[] = rows.map((kb: any) => ({
      knowledge_base: kb,
      share_id: kb.share_id || kb.id,
      organization_id: kb.shared_from_dept_id || kb.department_id || '',
      org_name: kb.shared_from_dept_name || '',
      permission: kb.my_permission || 'viewer',
      source_tenant_id: kb.tenant_id,
      shared_at: kb.shared_at || kb.updated_at || '',
    }))
    return { ...response, data }
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to list shared knowledge bases' }
  }
}

/** Legacy organization space list; the backend route is retired, so it returns empty. */
export async function listOrganizationSharedKnowledgeBases(_orgId: string): Promise<ApiResponse<OrganizationSharedKnowledgeBaseItem[]>> {
  return { success: true, data: [] }
}
