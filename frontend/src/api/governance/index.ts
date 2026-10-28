import { get } from '@/utils/request'

export interface GovernanceDepartmentStat {
  department_id: string
  department_name: string
  kb_total: number
  document_total: number
  member_total: number
}

export interface GovernanceRecentDocument {
  id: string
  title: string
  file_name: string
  knowledge_base_id: string
  knowledge_base: string
  parse_status: string
  updated_at: string
}

export interface GovernanceKBHealth {
  id: string
  name: string
  document_total: number
  updated_at: string
  search_total: number
  external_search_total: number
  shared_out_total: number
}

export interface GovernanceStats {
  scope: 'dept' | 'company'
  department_id?: string
  department_name?: string
  kb_total: number
  document_total: number
  processing_total: number
  failed_total: number
  member_total: number
  session_total: number
  message_total: number
  manager_total: number
  knowledge_admin_total: number
  editor_total: number
  viewer_total: number
  shared_in_total: number
  shared_out_total: number
  departments?: GovernanceDepartmentStat[]
  recent_documents: GovernanceRecentDocument[]
  generated_at: string
  period_days: number
  search_total: number
  no_hit_total: number
  search_hit_rate: number
  pending_cr_total: number
  open_issue_total: number
  cr_avg_cycle_days: number
  reading_run_total: number
  reading_finished_total: number
  reading_stopped_total: number
  reading_failed_total: number
  reading_question_total: number
  reading_auto_answered_total: number
  reading_auto_answer_rate: number
  reading_conclusion_total: number
  reading_confirmed_total: number
  reading_legacy_conclusion_total: number
  reading_term_total: number
  reading_contradiction_total: number
  reading_open_contradiction_total: number
  reading_avg_steps: number
  reading_avg_tokens: number
  reading_avg_duration_ms: number
  shared_external_searches: number
  kb_health?: GovernanceKBHealth[]
}

export interface GovernanceAuditEntry {
  id: number
  actor_user_id: string
  action: string
  target_type: string
  target_id: string
  outcome: string
  scope_type: string
  scope_id: string
  created_at: string
}

export function getDeptDashboard(departmentId?: string) {
  return get<{ data: GovernanceStats }>('/api/v1/governance/dashboard/dept', {
    params: departmentId ? { department_id: departmentId } : {},
  })
}

export function getCompanyDashboard(departmentId?: string) {
  return get<{ data: GovernanceStats }>('/api/v1/governance/dashboard/company', {
    params: departmentId ? { department_id: departmentId } : {},
  })
}

export function getGovernanceAudit(params: {
  department_id?: string
  after_id?: number
  limit?: number
} = {}) {
  return get<{ data: GovernanceAuditEntry[] }>('/api/v1/governance/audit', { params })
}
