<template>
  <div class="governance-page">
    <div class="governance-header">
      <div>
        <h1>{{ $t('governance.auditTitle') }}</h1>
        <p>{{ $t('governance.auditDesc') }}</p>
      </div>
      <t-button variant="outline" :loading="loading" @click="load">{{ $t('governance.refresh') }}</t-button>
    </div>

    <div v-if="anomalyCount > 0" class="audit-anomaly">
      <t-icon name="error-circle" size="18px" />
      <span>{{ $t('governance.audit.anomaly', { count: anomalyCount }) }}</span>
    </div>

    <div class="audit-toolbar">
      <t-input v-model="query" clearable class="audit-search" :placeholder="$t('governance.audit.searchPlaceholder')">
        <template #prefix-icon><t-icon name="search" /></template>
      </t-input>
      <t-select v-model="outcomeFilter" class="audit-filter" :options="outcomeOptions" />
      <t-button variant="outline" size="small" :disabled="!filteredRows.length" @click="exportCSV">
        {{ $t('governance.audit.export') }}
      </t-button>
      <span class="audit-count">{{ $t('governance.audit.count', { count: filteredRows.length }) }}</span>
    </div>

    <div class="governance-section">
      <t-table row-key="id" :data="pagedRows" :columns="columns" size="small" :bordered="false"
        :loading="loading" />
      <div v-if="!loading && filteredRows.length === 0" class="governance-empty">
        {{ $t('governance.auditEmpty') }}
      </div>
      <t-pagination
        v-if="filteredRows.length > pageSize"
        v-model="page"
        v-model:page-size="pageSize"
        :total="filteredRows.length"
        :page-size-options="[20, 50, 100]"
        class="audit-pagination"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, h, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { MessagePlugin } from 'tdesign-vue-next'
import { getGovernanceAudit, type GovernanceAuditEntry } from '@/api/governance'
import { listMembers, type TenantMember } from '@/api/tenant/members'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const authStore = useAuthStore()
const rows = ref<GovernanceAuditEntry[]>([])
const members = ref<TenantMember[]>([])
const loading = ref(false)
const query = ref('')
const outcomeFilter = ref('all')
const page = ref(1)
const pageSize = ref(20)

const ACTION_LABELS: Record<string, string> = {
  'rbac.member_added': '添加成员',
  'rbac.member_removed': '移除成员',
  'rbac.member_role_changed': '调整成员角色',
  'rbac.member_left': '成员退出',
  'rbac.access_denied': '访问被拒绝',
  'rbac.invitation_sent': '发送邀请',
  'rbac.invitation_accepted': '接受邀请',
  'rbac.invitation_declined': '拒绝邀请',
  'rbac.invitation_revoked': '撤销邀请',
  'rbac.invitation_expired': '邀请过期',
  'kb.created': '创建知识库',
  'kb.updated': '更新知识库',
  'kb.deleted': '删除知识库',
  'kb.duplicated': '复制知识库',
  'folder.created': '创建文件夹',
  'folder.updated': '更新文件夹',
  'folder.deleted': '删除文件夹',
  'reading.run_created': '开始精读',
  'reading.section_read': '阅读精读章节',
  'reading.run_finished': '结束精读',
  'reading.run_stopped': '停止精读',
  'reading.run_resumed': '继续精读',
  'reading.run_exported': '导出精读报告',
  'reading.feedback_received': '提交精读反馈',
  'collaboration.cr_reviewed': '审批变更请求',
  'skill.editor_grant_created': '授予技能编辑',
  'skill.editor_grant_deleted': '撤销技能编辑',
}

const OUTCOME_LABELS: Record<string, string> = {
  success: '成功',
  failed: '失败',
  denied: '拒绝',
  canceled: '取消',
  partial: '部分成功',
  accepted: '已接受',
}

const actorName = (id: string) => {
  const member = members.value.find((item) => item.user_id === id)
  return member?.username || member?.email || (id ? id.slice(0, 8) : '-')
}

const outcomeTheme = (outcome: string) => {
  if (outcome === 'failed' || outcome === 'denied') return 'danger'
  if (outcome === 'canceled' || outcome === 'partial') return 'warning'
  return 'success'
}

const columns = computed(() => [
  {
    colKey: 'created_at',
    title: t('governance.auditTable.time'),
    width: 170,
    cell: (_h: unknown, ctx: { row: GovernanceAuditEntry }) => formatTime(ctx.row.created_at),
  },
  {
    colKey: 'action',
    title: t('governance.auditTable.action'),
    cell: (_h: unknown, ctx: { row: GovernanceAuditEntry }) => ACTION_LABELS[ctx.row.action] || ctx.row.action,
  },
  {
    colKey: 'actor_user_id',
    title: t('governance.auditTable.actor'),
    cell: (_h: unknown, ctx: { row: GovernanceAuditEntry }) => actorName(ctx.row.actor_user_id),
  },
  {
    colKey: 'target_type',
    title: t('governance.auditTable.target'),
    ellipsis: true,
    cell: (_h: unknown, ctx: { row: GovernanceAuditEntry }) => {
      const target = [ctx.row.target_type, ctx.row.target_id ? ctx.row.target_id.slice(0, 12) : ''].filter(Boolean).join(' · ')
      return target || ctx.row.scope_type || '-'
    },
  },
  {
    colKey: 'outcome',
    title: t('governance.auditTable.outcome'),
    width: 100,
    cell: (_h: unknown, ctx: { row: GovernanceAuditEntry }) => h('span', {
      class: ['audit-outcome', `is-${outcomeTheme(ctx.row.outcome)}`],
    }, OUTCOME_LABELS[ctx.row.outcome] || ctx.row.outcome),
  },
])

const outcomeOptions = computed(() => [
  { value: 'all', label: t('governance.audit.allOutcomes') },
  { value: 'success', label: OUTCOME_LABELS.success },
  { value: 'denied', label: OUTCOME_LABELS.denied },
  { value: 'failed', label: OUTCOME_LABELS.failed },
])

const filteredRows = computed(() => {
  const q = query.value.trim().toLowerCase()
  return rows.value.filter((row) => {
    if (outcomeFilter.value !== 'all' && row.outcome !== outcomeFilter.value) return false
    if (!q) return true
    return [actorName(row.actor_user_id), ACTION_LABELS[row.action] || row.action, row.target_type, row.target_id, row.scope_id]
      .some((value) => String(value || '').toLowerCase().includes(q))
  })
})

const pagedRows = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return filteredRows.value.slice(start, start + pageSize.value)
})

watch([query, outcomeFilter, pageSize], () => {
  page.value = 1
})

function csvCell(value: unknown) {
  const text = String(value ?? '')
  return `"${text.replace(/"/g, '""')}"`
}

function exportCSV() {
  const header = [
    t('governance.auditTable.time'),
    t('governance.auditTable.action'),
    t('governance.auditTable.actor'),
    t('governance.auditTable.target'),
    t('governance.auditTable.outcome'),
  ]
  const lines = [header.map(csvCell).join(',')]
  for (const row of filteredRows.value) {
    lines.push([
      formatTime(row.created_at),
      ACTION_LABELS[row.action] || row.action,
      actorName(row.actor_user_id),
      [row.target_type, row.target_id].filter(Boolean).join(' · '),
      OUTCOME_LABELS[row.outcome] || row.outcome,
    ].map(csvCell).join(','))
  }
  const blob = new Blob(['﻿' + lines.join('\n')], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `wellread-audit-${new Date().toISOString().slice(0, 10)}.csv`
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
}

const anomalyCount = computed(() =>
  rows.value.filter((row) => ['failed', 'denied', 'canceled'].includes(row.outcome)).length,
)

const formatTime = (value?: string) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

const loadMembers = async () => {
  const tenantID = Number(authStore.effectiveTenantId || 0)
  if (!tenantID) return
  try {
    const response = await listMembers(tenantID, { page: 1, page_size: 100 })
    members.value = response.data?.members || []
  } catch {
    members.value = []
  }
}

const load = async () => {
  loading.value = true
  try {
    const response = await getGovernanceAudit({ limit: 100 })
    rows.value = Array.isArray(response.data) ? response.data : []
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('governance.loadFailed'))
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await loadMembers()
  await load()
})
</script>

<style lang="less" scoped>
.governance-page {
  max-width: 1180px;
  margin: 0 auto;
  padding: 28px 24px 48px;
}

.governance-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;

  h1 {
    margin: 0 0 6px;
    font-size: 22px;
    font-weight: 600;
    color: var(--td-text-color-primary);
  }

  p {
    margin: 0;
    font-size: 13px;
    color: var(--td-text-color-secondary);
  }
}

.audit-anomaly {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  padding: 10px 14px;
  border: 1px solid #fca5a5;
  border-radius: 9px;
  color: #b91c1c;
  background: #fef2f2;
  font-size: 12px;
}

.audit-toolbar {
  display: flex;
  align-items: center;
  gap: 9px;
  margin-bottom: 12px;
}

.audit-search {
  max-width: 300px;
}

.audit-filter {
  width: 130px;
}

.audit-count {
  margin-left: auto;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.governance-section {
  margin-top: 0;
  padding: 16px;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: 10px;
}

.governance-empty {
  padding: 24px;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  text-align: center;
}

.audit-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 14px;
}

:deep(.audit-outcome) {
  display: inline-block;
  padding: 1px 7px;
  border-radius: 4px;
  font-size: 11px;

  &.is-success { color: #15803d; background: #ecfdf5; }
  &.is-warning { color: #b45309; background: #fffbeb; }
  &.is-danger { color: #b91c1c; background: #fef2f2; }
}
</style>
