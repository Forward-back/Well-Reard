<template>
  <div class="ops-page">
    <section class="ops-todo" :class="{ 'is-allok': !alerts.length }">
      <span class="ops-todo-title">
        <t-icon :name="alerts.length ? 'notification' : 'check-circle'" size="17px" />
        {{ alerts.length ? $t('ops.overview.todo.title') : $t('ops.overview.todo.allOk') }}
      </span>
      <template v-if="alerts.length">
        <button v-for="alert in alerts.slice(0, 3)" :key="alert.id" type="button" class="ops-todo-item"
          @click="go(alert.actionPath)">
          {{ alert.title }}
          <span class="ops-priority" :class="`is-${alert.priority.toLowerCase()}`">{{ alert.priority }}</span>
        </button>
      </template>
      <span v-else class="ops-todo-desc">{{ $t('ops.overview.todo.allOkDesc') }}</span>
      <button type="button" class="ops-todo-go" @click="load">
        {{ $t('ops.overview.todo.refresh') }}
        <t-icon name="refresh" size="13px" />
      </button>
    </section>

    <section class="ops-health-grid">
      <article v-for="card in healthCards" :key="card.key" class="ops-health-card"
        :class="{ 'is-bad': card.tone === 'error' }">
        <div class="ops-health-head">
          <span class="ops-health-icon" :class="`is-${card.tone}`">
            <t-icon :name="card.icon" size="18px" />
          </span>
          <span class="ops-health-label">{{ card.label }}</span>
          <span class="ops-health-pill" :class="`is-${card.tone}`">{{ card.status }}</span>
        </div>
        <div class="ops-health-value">{{ card.value }}</div>
        <div class="ops-health-desc">{{ card.description }}</div>
        <div class="ops-health-probe">
          <span>{{ card.dependency }}</span>
          <span>{{ checkedAtText }}</span>
        </div>
      </article>
    </section>

    <section class="ops-overview-grid">
      <div class="ops-overview-main">
        <article class="ops-panel">
          <header class="ops-panel-head">
            <div>
              <h2>{{ $t('ops.overview.queueChart.title') }}</h2>
              <p>{{ $t('ops.overview.queueChart.description') }}</p>
            </div>
            <span class="ops-refresh-tag">{{ $t('ops.overview.refresh10s') }}</span>
          </header>
          <div class="ops-queue-grid">
            <div class="ops-queue-cell" :class="{ 'is-hot': queuePending >= 10, 'is-warn': queuePending > 0 && queuePending < 10 }">
              <strong>{{ queuePending }}</strong>
              <span>{{ $t('ops.overview.queueChart.pending') }}</span>
            </div>
            <div class="ops-queue-cell">
              <strong>{{ queueActive }}</strong>
              <span>{{ $t('ops.overview.queueChart.active') }}</span>
            </div>
            <div class="ops-queue-cell" :class="{ 'is-warn': queueRetry > 0 }">
              <strong>{{ queueRetry }}</strong>
              <span>{{ $t('ops.overview.queueChart.retry') }}</span>
            </div>
            <div class="ops-queue-cell is-idle">
              <strong>{{ queueArchived }}</strong>
              <span>{{ $t('ops.overview.queueChart.archived') }}</span>
            </div>
          </div>
          <div class="ops-trend-label">{{ $t('ops.overview.queueChart.trend') }}</div>
          <div class="ops-trend" role="img" :aria-label="$t('ops.overview.queueChart.trendAria')">
            <span v-for="(value, index) in queueTrend" :key="index" class="ops-trend-bar"
              :style="{ height: trendHeight(value) }" />
          </div>
          <div class="ops-queue-meta">
            <span>{{ $t('ops.overview.queueChart.paused', { count: pausedQueues }) }}</span>
            <span>{{ $t('ops.overview.queueChart.archivedToday', { count: queueArchived }) }}</span>
            <span>{{ $t('ops.overview.queueChart.retryMeta', { count: queueRetry }) }}</span>
          </div>
        </article>

        <article class="ops-panel">
          <header class="ops-panel-head">
            <div>
              <h2>{{ $t('ops.overview.resource.title') }}</h2>
              <p>{{ $t('ops.overview.resource.description') }}</p>
            </div>
            <span class="ops-refresh-tag">{{ $t('ops.overview.refresh10s') }}</span>
          </header>
          <div class="ops-resource-list">
            <div v-for="row in resourceRows" :key="row.key" class="ops-resource-row">
              <span class="ops-resource-name">{{ row.label }}</span>
              <div class="ops-resource-track">
                <div class="ops-resource-fill" :class="`is-${row.tone}`" :style="{ width: `${row.percent}%` }" />
                <span class="ops-resource-threshold" :style="{ left: `${row.threshold}%` }" />
              </div>
              <span class="ops-resource-pct" :class="`is-${row.tone}`">{{ row.percentText }}</span>
              <span class="ops-resource-detail">{{ row.detail }}</span>
            </div>
          </div>
          <p class="ops-resource-hint">{{ $t('ops.overview.resource.thresholdHint') }}</p>
        </article>

        <article class="ops-panel">
          <header class="ops-panel-head">
            <div>
              <h2>{{ $t('ops.overview.modelChart.title') }}</h2>
              <p>{{ $t('ops.overview.modelChart.description') }}</p>
            </div>
            <button type="button" class="ops-text-link" @click="go('/ops/models')">
              {{ $t('ops.overview.modelChart.manage') }}
            </button>
          </header>
          <div v-if="modelChannels.length" class="ops-channel-list">
            <div v-for="channel in modelChannels" :key="channel.key" class="ops-channel-row">
              <span class="ops-channel-dot" />
              <span class="ops-channel-name">{{ channel.name }}</span>
              <span class="ops-channel-meta">{{ $t('ops.overview.modelChart.configured', { count: channel.count }) }}</span>
              <span class="ops-channel-types">{{ channel.types }}</span>
              <button type="button" class="ops-channel-test" @click="go('/ops/models')">
                {{ $t('ops.overview.modelChart.test') }}
              </button>
            </div>
          </div>
          <div v-else class="ops-empty-chart">
            <t-icon name="control-platform" size="28px" />
            <span>{{ $t('ops.overview.modelChart.empty') }}</span>
          </div>
        </article>
      </div>

      <aside class="ops-overview-side">
        <article class="ops-panel">
          <header class="ops-panel-head">
            <div>
              <h2>{{ $t('ops.overview.alerts.title') }}</h2>
              <p>{{ $t('ops.overview.alerts.description') }}</p>
            </div>
            <span class="ops-refresh-tag">{{ $t('ops.overview.refresh10s') }}</span>
          </header>
          <div v-if="alerts.length" class="ops-alert-list">
            <button v-for="alert in alerts" :key="alert.id" type="button" class="ops-alert-item"
              @click="go(alert.actionPath)">
              <span class="ops-priority" :class="`is-${alert.priority.toLowerCase()}`">{{ alert.priority }}</span>
              <span class="ops-alert-copy">
                <span class="ops-alert-title">{{ alert.title }}</span>
                <span class="ops-alert-detail">{{ alert.detail }}</span>
              </span>
              <t-icon name="chevron-right" size="14px" class="ops-alert-arrow" />
            </button>
          </div>
          <div v-else class="ops-allok-panel">
            <t-icon name="check-circle" size="28px" />
            <span>{{ $t('ops.overview.alerts.empty') }}</span>
          </div>
        </article>

        <article class="ops-panel">
          <header class="ops-panel-head">
            <div>
              <h2>{{ $t('ops.overview.recentAudit.title') }}</h2>
              <p>{{ $t('ops.overview.recentAudit.description') }}</p>
            </div>
            <button type="button" class="ops-text-link" @click="go('/ops/security')">
              {{ $t('ops.overview.viewAll') }}
            </button>
          </header>
          <ul v-if="recentAudit.length" class="ops-audit-list">
            <li v-for="entry in recentAudit" :key="entry.id">
              <span class="ops-audit-dot" :class="{ 'is-error': isAuditError(entry) }" />
              <div class="ops-audit-copy">
                <div class="ops-audit-line">
                  <span class="ops-audit-actor">{{ actorLabel(entry) }}</span>
                  <span class="ops-audit-action">{{ auditActionLabel(entry.action) }}</span>
                  <span class="ops-audit-time">{{ formatTime(entry.created_at) }}</span>
                </div>
                <div class="ops-audit-detail">{{ auditDetail(entry) || $t('ops.overview.recentAudit.noDetail') }}</div>
              </div>
            </li>
          </ul>
          <div v-else class="ops-empty-chart ops-empty-chart--compact">
            <t-icon name="history" size="24px" />
            <span>{{ $t('ops.overview.recentAudit.empty') }}</span>
          </div>
          <p class="ops-audit-foot">{{ $t('ops.overview.recentAudit.scopeHint') }}</p>
        </article>
      </aside>
    </section>

    <section class="ops-quick-grid">
      <button v-for="action in quickActions" :key="action.key" type="button" class="ops-quick-action"
        @click="runQuickAction(action)">
        <span class="ops-quick-icon"><t-icon :name="action.icon" size="18px" /></span>
        <span class="ops-quick-copy">
          <span class="ops-quick-title">{{ $t(action.titleKey) }}</span>
          <span class="ops-quick-desc">{{ action.description }}</span>
        </span>
        <t-icon name="arrow-right" size="14px" class="ops-quick-arrow" />
      </button>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import {
  getSystemInfo,
  getParserEngines,
  getRuntimeQueues,
  listSystemAuditLog,
  listSystemUsers,
  type QueueStat,
  type SystemInfo,
} from '@/api/system'
import { listModels, type ModelConfig } from '@/api/model'
import { listStorageBackends } from '@/api/storage-backend'
import type { AuditLog } from '@/api/tenant/audit-log'

type AlertPriority = 'P1' | 'P2' | 'P3'
type OpsAlert = {
  id: string
  priority: AlertPriority
  title: string
  detail: string
  actionPath: string
}
type QuickAction = {
  key: string
  icon: string
  titleKey: string
  description: string
  path?: string
  action?: 'refresh'
}

const router = useRouter()
const authStore = useAuthStore()
const { t } = useI18n()

const info = ref<SystemInfo | null>(null)
const parserConnected = ref(false)
const parserCount = ref(0)
const storageAvailable = ref(false)
const storageBackendName = ref('')
const modelConfigs = ref<ModelConfig[]>([])
const queues = ref<QueueStat[]>([])
const queueAvailable = ref(true)
const recentAudit = ref<AuditLog[]>([])
const actorEmails = ref<Record<string, string>>({})
const queueTrend = ref<number[]>(Array.from({ length: 12 }, () => 0))
const checkedAt = ref('')
let timer: ReturnType<typeof setInterval> | undefined

const queuePending = computed(() => queues.value.reduce((sum, q) => sum + (q.pending || 0), 0))
const queueActive = computed(() => queues.value.reduce((sum, q) => sum + (q.active || 0), 0))
const queueRetry = computed(() => queues.value.reduce((sum, q) => sum + (q.retry || 0), 0))
const queueArchived = computed(() => queues.value.reduce((sum, q) => sum + (q.archived || 0), 0))
const pausedQueues = computed(() => queues.value.filter((q) => q.paused).length)
const modelCount = computed(() => modelConfigs.value.length)
const storageUsed = computed(() => Number(authStore.tenant?.storage_used || 0))
const storageQuota = computed(() => Number(authStore.tenant?.storage_quota || 0))
const storageRatio = computed(() => storageQuota.value > 0 ? Math.min(1, storageUsed.value / storageQuota.value) : 0)
const storagePercent = computed(() => toDisplayPercent(storageRatio.value * 100))

const applicationTone = computed(() => info.value?.db_migration_error ? 'error' : 'success')
const parserTone = computed(() => parserConnected.value ? 'success' : 'error')
const storageTone = computed(() => storageAvailable.value ? 'success' : 'warn')
const modelTone = computed(() => modelCount.value > 0 ? 'success' : 'error')
const queueTone = computed(() => {
  if (!queueAvailable.value) return 'warn'
  if (queueRetry.value > 0 || queuePending.value >= 10) return 'warn'
  return 'success'
})

const checkedAtText = computed(() => checkedAt.value
  ? t('ops.overview.health.checkedAt', { time: checkedAt.value })
  : t('ops.overview.health.checking'))

const healthCards = computed(() => [
  {
    key: 'app', icon: 'server', tone: applicationTone.value,
    label: t('ops.overview.health.app.title'),
    status: info.value?.db_migration_error ? t('ops.overview.health.error') : t('ops.overview.health.ok'),
    value: versionLabel.value,
    description: info.value?.db_migration_error
      ? t('ops.overview.health.app.migrationFailed')
      : t('ops.overview.health.app.description', { uptime: formatUptime(info.value?.uptime_seconds) }),
    dependency: t('ops.overview.health.app.dependency'),
  },
  {
    key: 'parser', icon: 'file-search', tone: parserTone.value,
    label: t('ops.overview.health.parser.title'),
    status: parserConnected.value ? t('ops.overview.health.ok') : t('ops.overview.health.error'),
    value: parserConnected.value ? t('ops.overview.health.parser.connected') : t('ops.overview.health.parser.disconnected'),
    description: t('ops.overview.health.parser.description', { count: parserCount.value }),
    dependency: t('ops.overview.health.parser.dependency'),
  },
  {
    key: 'storage', icon: 'cloud', tone: storageTone.value,
    label: t('ops.overview.health.storage.title'),
    status: storageAvailable.value ? t('ops.overview.health.ok') : t('ops.overview.health.warn'),
    value: storageBackendName.value || t('ops.overview.health.storage.none'),
    description: storageQuota.value > 0
      ? t('ops.overview.health.storage.usage', { used: formatBytes(storageUsed.value), quota: formatBytes(storageQuota.value) })
      : t('ops.overview.health.storage.description'),
    dependency: t('ops.overview.health.storage.dependency'),
  },
  {
    key: 'model', icon: 'control-platform', tone: modelTone.value,
    label: t('ops.overview.health.model.title'),
    status: modelCount.value > 0 ? t('ops.overview.health.ok') : t('ops.overview.health.error'),
    value: t('ops.overview.health.model.count', { count: modelCount.value }),
    description: t('ops.overview.health.model.description', { engine: info.value?.vector_store_engine || '-' }),
    dependency: t('ops.overview.health.model.dependency'),
  },
  {
    key: 'queue', icon: 'queue', tone: queueTone.value,
    label: t('ops.overview.health.queue.title'),
    status: !queueAvailable.value ? t('ops.overview.health.unavailable') : queueTone.value === 'success' ? t('ops.overview.health.ok') : t('ops.overview.health.warn'),
    value: t('ops.overview.health.queue.count', { pending: queuePending.value, retry: queueRetry.value }),
    description: t('ops.overview.health.queue.description', { archived: queueArchived.value }),
    dependency: t('ops.overview.health.queue.dependency'),
  },
])

const versionLabel = computed(() => {
  const raw = (info.value?.version || '').trim()
  return raw && raw !== 'unknown' ? `v${raw}` : 'dev'
})

const modelChannels = computed(() => {
  const groups = new Map<string, { key: string; name: string; count: number; types: Set<string> }>()
  for (const model of modelConfigs.value) {
    const provider = model.parameters?.provider || (model.source === 'local' ? 'Ollama' : 'Remote')
    const key = String(provider).toLowerCase()
    const group = groups.get(key) || { key, name: provider, count: 0, types: new Set<string>() }
    group.count += 1
    group.types.add(model.type)
    groups.set(key, group)
  }
  return Array.from(groups.values()).map((group) => ({
    ...group,
    types: Array.from(group.types).join(' / '),
  }))
})

const resourceRows = computed(() => {
  const rows = [
    {
      key: 'storage',
      label: t('ops.overview.resource.storage'),
      percent: storagePercent.value,
      threshold: 80,
      tone: storagePercent.value >= 80 ? 'error' : storagePercent.value >= 60 ? 'warn' : 'success',
      percentText: `${storagePercent.value}%`,
      detail: storageQuota.value > 0
        ? t('ops.overview.resource.storageDetail', { used: formatBytes(storageUsed.value), quota: formatBytes(storageQuota.value) })
        : t('ops.overview.resource.noQuota'),
    },
    {
      key: 'queue',
      label: t('ops.overview.resource.queue'),
      percent: Math.min(100, Math.round((queuePending.value / 20) * 100)),
      threshold: 50,
      tone: queuePending.value >= 10 ? 'error' : queuePending.value > 0 ? 'warn' : 'success',
      percentText: `${queuePending.value}`,
      detail: t('ops.overview.resource.queueDetail', { retry: queueRetry.value }),
    },
    {
      key: 'archive',
      label: t('ops.overview.resource.archive'),
      percent: Math.min(100, Math.round((queueArchived.value / 1000) * 100)),
      threshold: 50,
      tone: queueArchived.value >= 500 ? 'warn' : 'success',
      percentText: `${queueArchived.value}`,
      detail: t('ops.overview.resource.archiveDetail'),
    },
  ]
  return rows
})

const alerts = computed<OpsAlert[]>(() => {
  const items: OpsAlert[] = []
  if (info.value?.db_migration_error) {
    items.push({
      id: 'migration', priority: 'P1',
      title: t('ops.overview.alerts.migration.title'),
      detail: t('ops.overview.alerts.migration.detail'),
      actionPath: '/ops/settings',
    })
  }
  if (!parserConnected.value) {
    items.push({
      id: 'parser', priority: 'P1',
      title: t('ops.overview.alerts.parser.title'),
      detail: t('ops.overview.alerts.parser.detail'),
      actionPath: '/ops/models',
    })
  }
  if (modelCount.value === 0) {
    items.push({
      id: 'model', priority: 'P1',
      title: t('ops.overview.alerts.model.title'),
      detail: t('ops.overview.alerts.model.detail'),
      actionPath: '/ops/models',
    })
  }
  if (!queueAvailable.value) {
    items.push({
      id: 'queue-unavailable', priority: 'P1',
      title: t('ops.overview.alerts.queueUnavailable.title'),
      detail: t('ops.overview.alerts.queueUnavailable.detail'),
      actionPath: '/ops/runtime',
    })
  }
  if (queueRetry.value > 0) {
    items.push({
      id: 'retry', priority: 'P2',
      title: t('ops.overview.alerts.retry.title', { count: queueRetry.value }),
      detail: t('ops.overview.alerts.retry.detail'),
      actionPath: '/ops/runtime',
    })
  }
  if (queuePending.value >= 10) {
    items.push({
      id: 'backlog', priority: 'P2',
      title: t('ops.overview.alerts.backlog.title', { count: queuePending.value }),
      detail: t('ops.overview.alerts.backlog.detail'),
      actionPath: '/ops/runtime',
    })
  }
  if (storagePercent.value >= 80) {
    items.push({
      id: 'storage', priority: 'P2',
      title: t('ops.overview.alerts.storage.title', { percent: storagePercent.value }),
      detail: t('ops.overview.alerts.storage.detail'),
      actionPath: '/ops/settings',
    })
  } else if (storagePercent.value >= 60) {
    items.push({
      id: 'storage-watch', priority: 'P3',
      title: t('ops.overview.alerts.storageWatch.title', { percent: storagePercent.value }),
      detail: t('ops.overview.alerts.storageWatch.detail'),
      actionPath: '/ops/settings',
    })
  }
  if (queueArchived.value >= 500) {
    items.push({
      id: 'archive', priority: 'P3',
      title: t('ops.overview.alerts.archive.title', { count: queueArchived.value }),
      detail: t('ops.overview.alerts.archive.detail'),
      actionPath: '/ops/runtime',
    })
  }
  const order: Record<AlertPriority, number> = { P1: 0, P2: 1, P3: 2 }
  return items.sort((a, b) => order[a.priority] - order[b.priority])
})

const quickActions: QuickAction[] = [
  {
    key: 'refresh',
    icon: 'refresh',
    titleKey: 'ops.overview.quickActions.refresh',
    description: t('ops.overview.quickActions.refreshDesc'),
    action: 'refresh',
  },
  {
    key: 'runtime',
    icon: 'queue',
    titleKey: 'ops.overview.quickActions.runtime',
    description: t('ops.overview.quickActions.runtimeDesc'),
    path: '/ops/runtime',
  },
  {
    key: 'models',
    icon: 'control-platform',
    titleKey: 'ops.overview.quickActions.models',
    description: t('ops.overview.quickActions.modelsDesc'),
    path: '/ops/models',
  },
  {
    key: 'security',
    icon: 'lock-on',
    titleKey: 'ops.overview.quickActions.security',
    description: t('ops.overview.quickActions.securityDesc'),
    path: '/ops/security',
  },
]

const AUDIT_LABELS: Record<string, string> = {
  'system.setting_changed': '修改系统设置',
  'system.admin_promoted': '授予系统管理员',
  'system.admin_revoked': '撤销系统管理员',
  'system.user_password_reset': '重置用户密码',
  'system.user_created': '创建用户',
  'system.api_key_created': '创建平台 API Key',
  'system.api_key_revoked': '撤销平台 API Key',
  'system.queue_task_retried': '重试队列任务',
  'system.queue_task_deleted': '删除队列任务',
  'system.queue_task_run_now': '立即执行队列任务',
  'system.queue_task_cancelled': '取消队列任务',
  'system.queue_archived_purged': '清理归档任务',
}

const auditActionLabel = (action: string) => AUDIT_LABELS[action] || action
const actorLabel = (entry: AuditLog) => actorEmails.value[entry.actor_user_id] || entry.actor_role || entry.actor_user_id
const isAuditError = (entry: AuditLog) => entry.outcome === 'failed' || entry.outcome === 'denied' || entry.outcome === 'canceled'

const auditDetail = (entry: AuditLog) => {
  if (!entry.details) return ''
  if (typeof entry.details === 'string') return entry.details
  const details = entry.details as Record<string, unknown>
  const parts: string[] = []
  for (const key of ['target_email', 'target_username', 'key', 'queue', 'purged', 'task_type', 'old_role', 'new_role']) {
    if (details[key] !== undefined && details[key] !== null && details[key] !== '') {
      parts.push(`${key}: ${String(details[key])}`)
    }
  }
  if (parts.length) return parts.join(' · ')
  return Object.entries(details).slice(0, 3).map(([key, value]) => `${key}: ${String(value)}`).join(' · ')
}

const toDisplayPercent = (value: number) => {
  if (value > 0 && value < 1) return Number(value.toFixed(1))
  return Math.round(value)
}

const trendHeight = (value: number) => {
  const max = Math.max(...queueTrend.value, 1)
  return `${Math.max(4, Math.round((value / max) * 100))}%`
}

const formatTime = (value?: string) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

const formatUptime = (seconds?: number) => {
  if (!seconds || seconds < 0) return '-'
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${Math.max(1, minutes)}m`
}

const formatBytes = (value: number) => {
  if (!value) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let index = 0
  let amount = value
  while (amount >= 1024 && index < units.length - 1) {
    amount /= 1024
    index += 1
  }
  return `${amount.toFixed(amount >= 10 || index === 0 ? 0 : 1)} ${units[index]}`
}

const load = async () => {
  const [infoResult, parserResult, modelsResult, queueResult, auditResult, userResult] = await Promise.allSettled([
    getSystemInfo(),
    getParserEngines(),
    listModels(),
    getRuntimeQueues(),
    listSystemAuditLog({ limit: 6 }),
    listSystemUsers({ limit: 100 }),
  ])
  const [storageResult] = await Promise.allSettled([listStorageBackends()])

  if (infoResult.status === 'fulfilled') info.value = infoResult.value.data || null
  if (parserResult.status === 'fulfilled') {
    const payload = parserResult.value as unknown as { data?: unknown[]; connected?: boolean }
    parserCount.value = Array.isArray(payload.data) ? payload.data.length : 0
    parserConnected.value = Boolean(payload.connected)
  }
  if (storageResult.status === 'fulfilled') {
    const backends = storageResult.value.data || []
    const defaultID = storageResult.value.default_storage_backend_id || null
    const active = backends.find((b) => b.id === defaultID)
      || backends.find((b) => b.status === 'active')
      || backends[0]
    storageAvailable.value = Boolean(active)
    storageBackendName.value = active?.name || ''
  }
  if (modelsResult.status === 'fulfilled') modelConfigs.value = modelsResult.value || []
  if (queueResult.status === 'fulfilled') {
    queueAvailable.value = queueResult.value.available
    queues.value = queueResult.value.queues || []
    const currentLoad = queuePending.value + queueActive.value + queueRetry.value
    queueTrend.value = [...queueTrend.value.slice(1), currentLoad]
  }
  if (auditResult.status === 'fulfilled') recentAudit.value = auditResult.value.data || []
  if (userResult.status === 'fulfilled') {
    const map: Record<string, string> = {}
    for (const user of userResult.value.users || []) map[user.id] = user.email || user.username
    actorEmails.value = map
  }
  checkedAt.value = new Date().toLocaleTimeString()
}

const go = (path: string) => {
  router.push(path)
}

const runQuickAction = (action: QuickAction) => {
  if (action.action === 'refresh') {
    void load()
    return
  }
  if (action.path) go(action.path)
}

let refreshHandler: (() => void) | undefined

onMounted(() => {
  void load()
  timer = setInterval(() => { void load() }, 10000)
  refreshHandler = () => { void load() }
  window.addEventListener('ops:refresh', refreshHandler)
})

onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
  if (refreshHandler) window.removeEventListener('ops:refresh', refreshHandler)
})
</script>

<style lang="less" scoped>
.ops-page {
  width: 100%;
  max-width: 1380px;
  margin: 0 auto;
  padding: 20px 28px 44px;
  box-sizing: border-box;
}

.ops-todo {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  padding: 11px 16px;
  margin-bottom: 14px;
  background: #fff7ed;
  border: 1px solid #fed7aa;
  border-radius: 10px;

  &.is-allok {
    background: #f0fdf4;
    border-color: #bbf7d0;
  }
}

.ops-todo-title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: #9a3412;
  font-size: 13px;
  font-weight: 600;

  .is-allok & { color: #15803d; }
}

.ops-todo-desc {
  color: #15803d;
  font-size: 12px;
}

.ops-todo-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  border: 0;
  border-radius: 7px;
  background: rgba(255, 255, 255, 0.8);
  color: #7c2d12;
  font-size: 12px;
  cursor: pointer;

  &:hover { background: #fff; }
}

.ops-todo-go {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-left: auto;
  padding: 0;
  border: 0;
  background: transparent;
  color: #c2410c;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;

  .is-allok & { color: #15803d; }
}

.ops-priority {
  flex-shrink: 0;
  padding: 1px 5px;
  border-radius: 3px;
  color: #fff;
  font-size: 10px;
  font-weight: 600;

  &.is-p1 { background: #dc2626; }
  &.is-p2 { background: #f59e0b; }
  &.is-p3 { background: #94a3b8; }
}

.ops-health-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}

.ops-health-card,
.ops-panel {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: 10px;
}

.ops-health-card {
  padding: 14px 15px 11px;

  &.is-bad {
    border-color: #fca5a5;
    background: #fef2f2;
  }
}

.ops-health-head {
  display: flex;
  align-items: center;
  gap: 7px;
  min-width: 0;
}

.ops-health-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 26px;
  width: 26px;
  height: 26px;
  border-radius: 7px;

  &.is-success { color: var(--td-success-color); background: rgba(7, 192, 95, 0.1); }
  &.is-warn { color: var(--td-warning-color); background: rgba(237, 123, 47, 0.1); }
  &.is-error { color: var(--td-error-color); background: rgba(213, 73, 65, 0.1); }
}

.ops-health-label {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-health-pill {
  flex-shrink: 0;
  padding: 2px 6px;
  border-radius: 10px;
  font-size: 10px;

  &.is-success { color: var(--td-success-color); background: rgba(7, 192, 95, 0.1); }
  &.is-warn { color: var(--td-warning-color); background: rgba(237, 123, 47, 0.1); }
  &.is-error { color: var(--td-error-color); background: rgba(213, 73, 65, 0.1); }
}

.ops-health-value {
  margin-top: 11px;
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-size: 18px;
  font-weight: 600;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-health-desc {
  margin-top: 3px;
  overflow: hidden;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  line-height: 16px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-health-probe {
  display: flex;
  justify-content: space-between;
  gap: 6px;
  margin-top: 8px;
  padding-top: 7px;
  border-top: 1px dashed var(--td-component-stroke);
  color: var(--td-text-color-placeholder);
  font-size: 10px;
  white-space: nowrap;
}

.ops-overview-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.55fr) minmax(310px, 1fr);
  gap: 14px;
  align-items: start;
}

.ops-overview-main,
.ops-overview-side {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
}

.ops-panel {
  padding: 16px 18px;
}

.ops-panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 13px;

  h2 {
    margin: 0 0 3px;
    font-size: 14px;
    font-weight: 600;
  }

  p {
    margin: 0;
    color: var(--td-text-color-placeholder);
    font-size: 11px;
    line-height: 16px;
  }
}

.ops-refresh-tag {
  flex-shrink: 0;
  padding: 2px 7px;
  border-radius: 4px;
  color: var(--td-text-color-placeholder);
  background: var(--td-bg-color-secondarycontainer);
  font-size: 10px;
  white-space: nowrap;
}

.ops-queue-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 9px;
}

.ops-queue-cell {
  display: flex;
  align-items: center;
  flex-direction: column;
  gap: 2px;
  padding: 12px 8px;
  border-radius: 9px;
  background: var(--td-bg-color-secondarycontainer);

  strong {
    color: var(--td-success-color);
    font-size: 22px;
    font-weight: 600;
    font-variant-numeric: tabular-nums;
  }

  span {
    color: var(--td-text-color-secondary);
    font-size: 11px;
  }

  &.is-warn strong { color: var(--td-warning-color); }
  &.is-hot strong { color: var(--td-error-color); }
  &.is-idle strong { color: var(--td-text-color-placeholder); }
}

.ops-trend-label {
  margin-top: 12px;
  color: var(--td-text-color-placeholder);
  font-size: 10px;
}

.ops-trend {
  display: flex;
  align-items: flex-end;
  gap: 3px;
  height: 36px;
  margin-top: 5px;
}

.ops-trend-bar {
  flex: 1;
  min-height: 4px;
  border-radius: 2px 2px 0 0;
  background: #d1fae5;
}

.ops-queue-meta {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  margin-top: 7px;
  padding-top: 8px;
  border-top: 1px solid var(--td-component-stroke);
  color: var(--td-text-color-placeholder);
  font-size: 10px;
}

.ops-resource-list {
  display: flex;
  flex-direction: column;
}

.ops-resource-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 0;
  border-bottom: 1px solid var(--td-component-stroke);

  &:last-child { border-bottom: 0; }
}

.ops-resource-name {
  flex: 0 0 58px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.ops-resource-track {
  position: relative;
  flex: 1;
  height: 8px;
  border-radius: 4px;
  background: var(--td-bg-color-secondarycontainer);
}

.ops-resource-fill {
  height: 100%;
  border-radius: inherit;
  background: var(--td-success-color);

  &.is-warn { background: var(--td-warning-color); }
  &.is-error { background: var(--td-error-color); }
}

.ops-resource-threshold {
  position: absolute;
  top: -3px;
  width: 2px;
  height: 14px;
  border-radius: 1px;
  background: #ef4444;
}

.ops-resource-pct {
  flex: 0 0 38px;
  color: var(--td-success-color);
  font-size: 12px;
  font-weight: 600;
  text-align: right;

  &.is-warn { color: var(--td-warning-color); }
  &.is-error { color: var(--td-error-color); }
}

.ops-resource-detail {
  flex: 0 0 116px;
  overflow: hidden;
  color: var(--td-text-color-placeholder);
  font-size: 10px;
  text-align: right;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-resource-hint {
  margin: 7px 0 0;
  color: var(--td-text-color-placeholder);
  font-size: 10px;
}

.ops-channel-list {
  display: flex;
  flex-direction: column;
}

.ops-channel-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 9px 0;
  border-bottom: 1px solid var(--td-component-stroke);
  font-size: 12px;

  &:last-child { border-bottom: 0; }
}

.ops-channel-dot {
  flex: 0 0 6px;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--td-success-color);
}

.ops-channel-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  color: var(--td-text-color-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-channel-meta {
  flex-shrink: 0;
  color: var(--td-text-color-secondary);
  font-size: 11px;
}

.ops-channel-types {
  flex: 0 0 118px;
  overflow: hidden;
  color: var(--td-text-color-placeholder);
  font-size: 10px;
  text-align: right;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-channel-test {
  flex-shrink: 0;
  padding: 3px 8px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 5px;
  color: var(--td-text-color-secondary);
  background: transparent;
  font-size: 10px;
  cursor: pointer;

  &:hover { color: var(--td-brand-color); border-color: var(--td-brand-color); }
}

.ops-alert-list {
  display: flex;
  flex-direction: column;
}

.ops-alert-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  width: 100%;
  padding: 9px 0;
  border: 0;
  border-bottom: 1px solid var(--td-component-stroke);
  background: transparent;
  text-align: left;
  cursor: pointer;

  &:last-child { border-bottom: 0; }
  &:hover .ops-alert-title { color: var(--td-brand-color); }
}

.ops-alert-copy {
  display: flex;
  flex: 1;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.ops-alert-title {
  color: var(--td-text-color-primary);
  font-size: 12px;
  line-height: 17px;
  transition: color 0.15s ease;
}

.ops-alert-detail {
  color: var(--td-text-color-placeholder);
  font-size: 10px;
  line-height: 15px;
}

.ops-alert-arrow {
  margin-top: 2px;
  color: var(--td-text-color-placeholder);
}

.ops-allok-panel {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 88px;
  color: var(--td-success-color);
  font-size: 12px;
}

.ops-audit-list {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;

  li {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    padding: 9px 0;
    border-bottom: 1px solid var(--td-component-stroke);

    &:last-child { border-bottom: 0; padding-bottom: 0; }
  }
}

.ops-audit-dot {
  flex: 0 0 6px;
  width: 6px;
  height: 6px;
  margin-top: 5px;
  border-radius: 50%;
  background: var(--td-success-color);

  &.is-error { background: var(--td-error-color); }
}

.ops-audit-copy {
  display: flex;
  flex: 1;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.ops-audit-line {
  display: flex;
  align-items: baseline;
  gap: 6px;
  min-width: 0;
  font-size: 11px;
}

.ops-audit-actor {
  flex: 0 0 auto;
  max-width: 110px;
  overflow: hidden;
  color: var(--td-brand-color);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-audit-action {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  color: var(--td-text-color-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-audit-time {
  flex-shrink: 0;
  color: var(--td-text-color-placeholder);
  font-size: 10px;
}

.ops-audit-detail {
  overflow: hidden;
  color: var(--td-text-color-placeholder);
  font-size: 10px;
  line-height: 15px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-audit-foot {
  margin: 10px 0 0;
  padding-top: 8px;
  border-top: 1px dashed var(--td-component-stroke);
  color: var(--td-text-color-placeholder);
  font-size: 10px;
  line-height: 15px;
}

.ops-quick-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  margin-top: 14px;
}

.ops-quick-action {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 13px 14px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 10px;
  background: var(--td-bg-color-container);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s ease, background 0.15s ease;

  &:hover {
    border-color: var(--td-brand-color);
    background: var(--td-brand-color-light);
  }
}

.ops-quick-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 32px;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  color: var(--td-brand-color);
  background: var(--td-brand-color-light);
}

.ops-quick-copy {
  display: flex;
  flex: 1;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.ops-quick-title {
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 600;
}

.ops-quick-desc {
  overflow: hidden;
  color: var(--td-text-color-placeholder);
  font-size: 10px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-quick-arrow {
  flex-shrink: 0;
  color: var(--td-text-color-placeholder);
}

.ops-text-link {
  flex-shrink: 0;
  padding: 0;
  border: 0;
  color: var(--td-brand-color);
  background: transparent;
  font-size: 11px;
  cursor: pointer;
}

.ops-empty-chart {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 90px;
  color: var(--td-text-color-placeholder);
  font-size: 12px;

  &--compact { min-height: 96px; }
}

@media (max-width: 1180px) {
  .ops-health-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .ops-overview-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .ops-overview-side {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 760px) {
  .ops-page {
    padding: 16px 14px 32px;
  }

  .ops-health-grid,
  .ops-quick-grid,
  .ops-overview-side {
    grid-template-columns: minmax(0, 1fr);
  }

  .ops-queue-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .ops-resource-detail {
    display: none;
  }
}
</style>
