<template>
  <div class="governance-page">
    <div class="governance-header">
      <div>
        <h1>{{ isCompany ? $t('governance.companyTitle') : $t('governance.deptTitle') }}</h1>
        <p>{{ isCompany ? $t('governance.companyDesc') : $t('governance.deptDesc') }}</p>
      </div>
      <div v-if="isCompany" class="governance-filter">
        <t-select v-model="departmentId" :options="departmentOptions" clearable
          :placeholder="$t('governance.allDepartments')" @change="load" />
      </div>
    </div>

    <div v-if="loading" class="governance-loading"><t-loading size="small" /></div>
    <template v-else-if="stats">
      <div class="governance-todo" :class="{ 'is-allok': !todoItems.length }">
        <span class="governance-todo-title">
          <t-icon :name="todoItems.length ? 'notification' : 'check-circle'" size="16px" />
          {{ todoItems.length ? $t('governance.todo.title') : $t('governance.todo.allOk') }}
        </span>
        <span v-for="item in todoItems" :key="item.key" class="governance-todo-item">
          {{ item.label }}
        </span>
        <span v-if="!todoItems.length" class="governance-todo-desc">{{ $t('governance.todo.allOkDesc') }}</span>
      </div>

      <div v-if="!isCompany" class="governance-summary">
        <t-icon name="chart-bubble" size="17px" />
        <span>{{ summaryText }}</span>
      </div>

      <div class="governance-cards">
        <div v-for="card in cards" :key="card.key" class="governance-card" :class="{ 'is-warn': card.tone === 'warn', 'is-error': card.tone === 'error' }">
          <span class="governance-card-label">{{ card.label }}</span>
          <span class="governance-card-value" :class="`is-${card.tone || 'default'}`">{{ card.value }}</span>
          <span class="governance-card-sub">{{ card.sub }}</span>
        </div>
      </div>

      <section v-if="!isCompany && (stats.kb_health || []).length" class="governance-section">
        <h2>{{ $t('governance.kbHealth.title') }}</h2>
        <t-table row-key="id" :data="stats.kb_health || []" :columns="kbHealthColumns"
          size="small" :bordered="false" />
      </section>

      <div v-if="!isCompany" class="governance-insights">
        <section class="governance-section">
          <h2>{{ $t('governance.readingQuality.title') }}</h2>
          <div class="insight-row">
            <span>{{ $t('governance.readingQuality.runs') }}</span>
            <strong>{{ stats.reading_run_total }} / {{ stats.reading_finished_total }} / {{ stats.reading_stopped_total + stats.reading_failed_total }}</strong>
          </div>
          <div class="insight-row">
            <span>{{ $t('governance.readingQuality.questions') }}</span>
            <strong>{{ stats.reading_question_total }} · {{ Number(stats.reading_auto_answer_rate || 0).toFixed(1) }}%</strong>
          </div>
          <div class="insight-row">
            <span>{{ $t('governance.readingQuality.conclusions') }}</span>
            <strong>{{ stats.reading_confirmed_total }} / {{ stats.reading_conclusion_total }}</strong>
          </div>
          <div class="insight-row">
            <span>{{ $t('governance.readingQuality.contradictions') }}</span>
            <strong>{{ stats.reading_open_contradiction_total }} / {{ stats.reading_contradiction_total }}</strong>
          </div>
          <div class="insight-row">
            <span>{{ $t('governance.readingQuality.terms') }}</span>
            <strong>{{ stats.reading_term_total }}</strong>
          </div>
          <div class="insight-row">
            <span>{{ $t('governance.readingQuality.averages') }}</span>
            <strong>{{ Number(stats.reading_avg_steps || 0).toFixed(1) }} · {{ Math.round(stats.reading_avg_tokens || 0) }} · {{ formatDuration(stats.reading_avg_duration_ms || 0) }}</strong>
          </div>
          <p v-if="stats.reading_legacy_conclusion_total > 0" class="insight-hint">
            {{ $t('governance.readingQuality.legacyHint', { count: stats.reading_legacy_conclusion_total }) }}
          </p>
        </section>
        <section class="governance-section">
          <h2>{{ $t('governance.collab.title') }}</h2>
          <div class="insight-row">
            <span>{{ $t('governance.collab.pendingCr') }}</span>
            <strong>{{ stats.pending_cr_total }}</strong>
          </div>
          <div class="insight-row">
            <span>{{ $t('governance.collab.openIssues') }}</span>
            <strong>{{ stats.open_issue_total }}</strong>
          </div>
          <div class="insight-row">
            <span>{{ $t('governance.collab.avgCycle') }}</span>
            <strong>{{ formatDays(stats.cr_avg_cycle_days) }}</strong>
          </div>
        </section>
        <section class="governance-section">
          <h2>{{ $t('governance.auditSummary.title') }}</h2>
          <div class="insight-row">
            <span>{{ $t('governance.auditSummary.anomalies') }}</span>
            <strong :class="{ 'is-error': auditAnomalies > 0 }">{{ auditAnomalies }}</strong>
          </div>
          <p class="insight-hint">{{ $t('governance.auditSummary.hint') }}</p>
        </section>
      </div>

      <section v-if="isCompany && (stats.departments || []).length" class="governance-section">
        <h2>{{ $t('governance.departments') }}</h2>
        <t-table row-key="department_id" :data="stats.departments || []" :columns="departmentColumns"
          size="small" :bordered="false" />
      </section>

      <section v-if="isCompany" class="governance-section">
        <h2>{{ $t('governance.recentDocuments') }}</h2>
        <t-table row-key="id" :data="stats.recent_documents || []" :columns="documentColumns"
          size="small" :bordered="false" />
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import { listDepartments, type Department } from '@/api/department'
import { useAuthStore } from '@/stores/auth'
import {
  getCompanyDashboard,
  getDeptDashboard,
  getGovernanceAudit,
  type GovernanceAuditEntry,
  type GovernanceStats,
} from '@/api/governance'

const { t } = useI18n()
const route = useRoute()
const authStore = useAuthStore()
const props = defineProps<{ scope?: 'company' | 'dept' }>()
const isCompany = computed(() => props.scope ? props.scope === 'company' : route.name === 'governanceCompany')
const stats = ref<GovernanceStats | null>(null)
const loading = ref(false)
const departmentId = ref('')
const departments = ref<Department[]>([])
const auditAnomalies = ref(0)

const departmentOptions = computed(() => departments.value.map((dept) => ({
  value: dept.id,
  label: dept.name,
})))

const cards = computed(() => {
  const value = stats.value
  if (!value) return []
  return [
    {
      key: 'assets',
      label: t('governance.cards.assets'),
      value: value.document_total,
      sub: t('governance.cards.assetsSub', { kb: value.kb_total, out: value.shared_out_total }),
      tone: 'default',
    },
    {
      key: 'health',
      label: t('governance.cards.health'),
      value: value.failed_total > 0
        ? t('governance.cards.healthNeedFix')
        : value.processing_total > 0
          ? t('governance.cards.healthProcessing')
          : t('governance.cards.healthOk'),
      sub: t('governance.cards.healthSub', { processing: value.processing_total, failed: value.failed_total }),
      tone: value.failed_total > 0 ? 'error' : value.processing_total > 0 ? 'warn' : 'success',
    },
    {
      key: 'search',
      label: t('governance.cards.search'),
      value: value.search_total,
      sub: t('governance.cards.searchSub', { rate: value.search_hit_rate.toFixed(1) }),
      tone: 'default',
    },
    {
      key: 'nohit',
      label: t('governance.cards.noHit'),
      value: value.no_hit_total,
      sub: value.search_total > 0
        ? t('governance.cards.noHitSub', { rate: ((value.no_hit_total / value.search_total) * 100).toFixed(1) })
        : t('governance.cards.noHitEmpty'),
      tone: value.no_hit_total > 0 ? 'warn' : 'success',
    },
    {
      key: 'collab',
      label: t('governance.cards.collab'),
      value: formatDays(value.cr_avg_cycle_days),
      sub: t('governance.cards.collabSub', { pending: value.pending_cr_total, issues: value.open_issue_total }),
      tone: value.pending_cr_total > 0 || value.open_issue_total > 0 ? 'warn' : 'success',
    },
    {
      key: 'reading',
      label: t('governance.cards.reading'),
      value: value.reading_run_total,
      sub: t('governance.cards.readingSub', {
        finished: value.reading_finished_total,
        confirmed: value.reading_confirmed_total,
      }),
      tone: 'default',
    },
    {
      key: 'readingQuality',
      label: t('governance.cards.readingQuality'),
      value: `${Number(value.reading_auto_answer_rate || 0).toFixed(1)}%`,
      sub: t('governance.cards.readingQualitySub', {
        questions: value.reading_question_total,
        conclusions: value.reading_confirmed_total,
        contradictions: value.reading_open_contradiction_total,
      }),
      tone: value.reading_open_contradiction_total > 0 ? 'warn' : 'success',
    },
    {
      key: 'audit',
      label: t('governance.cards.auditAnomaly'),
      value: auditAnomalies.value,
      sub: t('governance.cards.auditSub'),
      tone: auditAnomalies.value > 0 ? 'error' : 'success',
    },
  ]
})

const summaryText = computed(() => {
  const value = stats.value
  if (!value) return ''
  return t('governance.summary', {
    docs: value.document_total,
    searches: value.search_total,
    nohit: value.no_hit_total,
    cycle: formatDays(value.cr_avg_cycle_days),
    issues: value.open_issue_total,
    external: value.shared_external_searches,
  })
})

const formatDays = (days: number) => {
  if (!days || days <= 0) return '-'
  return `${days.toFixed(1)} ${t('governance.days')}`
}

const formatDuration = (ms: number) => {
  if (!ms || ms <= 0) return '-'
  const seconds = ms / 1000
  if (seconds < 60) return `${seconds.toFixed(1)}s`
  return `${(seconds / 60).toFixed(1)}min`
}

const formatDate = (value?: string) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleDateString()
}

const kbHealthColumns = computed(() => [
  { colKey: 'name', title: t('governance.kbHealth.name'), ellipsis: true },
  { colKey: 'document_total', title: t('governance.kbHealth.docs') },
  {
    colKey: 'updated_at',
    title: t('governance.kbHealth.updated'),
    cell: (_h: unknown, ctx: { row: { updated_at?: string } }) => formatDate(ctx.row.updated_at),
  },
  { colKey: 'search_total', title: t('governance.kbHealth.searches') },
  { colKey: 'external_search_total', title: t('governance.kbHealth.external') },
  { colKey: 'shared_out_total', title: t('governance.kbHealth.sharedOut') },
])

const todoItems = computed(() => {
  const value = stats.value
  if (!value) return []
  const items: Array<{ key: string; label: string }> = []
  if (value.failed_total > 0) items.push({ key: 'failed', label: t('governance.todo.failed', { count: value.failed_total }) })
  if (value.processing_total > 0) items.push({ key: 'processing', label: t('governance.todo.processing', { count: value.processing_total }) })
  if (value.pending_cr_total > 0) items.push({ key: 'pendingCr', label: t('governance.todo.pendingCr', { count: value.pending_cr_total }) })
  if (value.open_issue_total > 0) items.push({ key: 'openIssues', label: t('governance.todo.openIssues', { count: value.open_issue_total }) })
  if (auditAnomalies.value > 0) items.push({ key: 'audit', label: t('governance.todo.audit', { count: auditAnomalies.value }) })
  return items
})

const departmentColumns = computed(() => [
  { colKey: 'department_name', title: t('governance.table.department') },
  { colKey: 'kb_total', title: t('governance.table.kb') },
  { colKey: 'document_total', title: t('governance.table.doc') },
  { colKey: 'member_total', title: t('governance.table.member') },
])

const formatTimestamp = (value?: string): string => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

const documentColumns = computed(() => [
  { colKey: 'title', title: t('governance.table.title'), ellipsis: true },
  { colKey: 'knowledge_base', title: t('governance.table.kb'), ellipsis: true },
  { colKey: 'parse_status', title: t('governance.table.status') },
  {
    colKey: 'updated_at',
    title: t('governance.table.updated'),
    width: 180,
    cell: (_h: unknown, ctx: { row: { updated_at?: string } }) => formatTimestamp(ctx.row.updated_at),
  },
])

const loadDepartments = async () => {
  const tenantID = Number(authStore.effectiveTenantId || 0)
  if (!tenantID || !isCompany.value) return
  try {
    const response = await listDepartments(tenantID)
    departments.value = Array.isArray(response.data) ? response.data : []
  } catch {
    departments.value = []
  }
}

const load = async () => {
  loading.value = true
  try {
    const response = isCompany.value
      ? await getCompanyDashboard(departmentId.value || undefined)
      : await getDeptDashboard()
    stats.value = response.data
    auditAnomalies.value = 0
    if (authStore.hasCompanyCaps(['audit:company_read']) || authStore.hasCompanyCaps(['audit:dept_read'])) {
      try {
        const audit = await getGovernanceAudit({
          department_id: departmentId.value || undefined,
          limit: 100,
        })
        const rows: GovernanceAuditEntry[] = Array.isArray(audit.data) ? audit.data : []
        auditAnomalies.value = rows.filter((row) =>
          ['failed', 'denied', 'canceled'].includes(row.outcome),
        ).length
      } catch {
        auditAnomalies.value = 0
      }
    }
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('governance.loadFailed'))
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await loadDepartments()
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

.governance-filter {
  width: 220px;
}

.governance-loading {
  display: flex;
  justify-content: center;
  padding: 40px;
}

.governance-todo {
  display: flex;
  align-items: center;
  gap: 9px;
  flex-wrap: wrap;
  padding: 10px 15px;
  margin-bottom: 14px;
  border: 1px solid #fed7aa;
  border-radius: 10px;
  background: #fff7ed;

  &.is-allok {
    border-color: #bbf7d0;
    background: #f0fdf4;
  }
}

.governance-todo-title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: #9a3412;
  font-size: 13px;
  font-weight: 600;

  .is-allok & { color: #15803d; }
}

.governance-todo-item {
  padding: 3px 9px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.8);
  color: #7c2d12;
  font-size: 12px;
}

.governance-todo-desc {
  color: #15803d;
  font-size: 12px;
}

.governance-summary {
  display: flex;
  align-items: flex-start;
  gap: 9px;
  margin-bottom: 14px;
  padding: 11px 15px;
  border: 1px solid #bbf7d0;
  border-radius: 10px;
  color: #14532d;
  background: linear-gradient(90deg, #f0fdf4, #fff);
  font-size: 12px;
  line-height: 19px;

  .t-icon { flex-shrink: 0; color: #16a34a; margin-top: 1px; }
}

.governance-insights {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
  margin-top: 14px;
}

.insight-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 9px 0;
  border-bottom: 1px solid var(--td-component-stroke);
  color: var(--td-text-color-secondary);
  font-size: 12px;

  &:last-child { border-bottom: 0; }
  strong { color: var(--td-text-color-primary); font-size: 16px; }
  strong.is-error { color: var(--td-error-color); }
}

.insight-hint {
  margin: 8px 0 0;
  color: var(--td-text-color-placeholder);
  font-size: 10px;
  line-height: 16px;
}

.governance-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 12px;
  margin-bottom: 22px;
}

.governance-card {
  display: flex;
  flex-direction: column;
  gap: 5px;
  padding: 15px 16px;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: 10px;

  &.is-warn {
    border-color: #fcd34d;
    background: #fffbeb;
  }

  &.is-error {
    border-color: #fca5a5;
    background: #fef2f2;
  }
}

.governance-card-label {
  font-size: 12px;
  color: var(--td-text-color-secondary);
}

.governance-card-value {
  font-size: 24px;
  font-weight: 600;
  color: var(--td-text-color-primary);

  &.is-warn { color: #b45309; }
  &.is-error { color: #b91c1c; }
  &.is-success { color: var(--td-success-color); }
}

.governance-card-sub {
  overflow: hidden;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.governance-section {
  margin-top: 18px;
  padding: 16px;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: 10px;

  h2 {
    margin: 0 0 12px;
    font-size: 15px;
    font-weight: 600;
  }
}
</style>
