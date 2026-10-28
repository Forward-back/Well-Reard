import type { DocCRStatus, DocCRReviewerStatus, DocIssueStatus, DocIssueType } from '@/api/collaboration'

export const CR_STATUS_LABELS: Record<DocCRStatus, string> = {
  draft: '草稿',
  pending_review: '待审阅',
  changes_requested: '修改中',
  merged: '已合并',
  closed: '已关闭',
}

export const CR_STATUS_BADGE: Record<DocCRStatus, string> = {
  draft: 'is-draft',
  pending_review: 'is-open',
  changes_requested: 'is-action',
  merged: 'is-merged',
  closed: 'is-closed',
}

export const ISSUE_STATUS_LABELS: Record<DocIssueStatus, string> = {
  open: '开放',
  closed: '已关闭',
}

export const ISSUE_TYPE_LABELS: Record<DocIssueType, string> = {
  data_error: '数据错误',
  outdated: '事实过期',
  unclear: '表述不清',
  format: '格式问题',
  other: '其他',
}

export const ISSUE_TYPES: DocIssueType[] = ['data_error', 'outdated', 'unclear', 'format', 'other']

export const REVIEWER_STATUS_LABELS: Record<DocCRReviewerStatus, string> = {
  pending: '待审',
  approved: '已批准',
  changes_requested: '要求修改',
}

export function relativeTime(value?: string | null): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  const diff = Date.now() - date.getTime()
  const minute = 60 * 1000
  const hour = 60 * minute
  const day = 24 * hour
  if (diff < minute) return '刚刚'
  if (diff < hour) return `${Math.floor(diff / minute)} 分钟前`
  if (diff < day) return `${Math.floor(diff / hour)} 小时前`
  if (diff < 7 * day) return `${Math.floor(diff / day)} 天前`
  return date.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' })
}

export function formatDateTime(value?: string | null): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function userInitial(name?: string, id?: string): string {
  const text = (name || id || '?').trim()
  return text.slice(0, 1).toUpperCase()
}
