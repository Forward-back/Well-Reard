<template>
  <div class="collab-v2 version-history">
    <div class="version-toolbar">
      <div class="version-title">
        <span class="version-name">{{ docTitle || t('collaboration.versionHistory') }}</span>
        <span v-if="currentVersion" class="cv-badge is-open">{{ currentVersion.version_label }} {{ t('collaboration.current') }}</span>
      </div>
      <div class="version-actions">
        <button type="button" class="cv-btn" @click="downloadCurrent">
          <t-icon name="download" size="14px" /> {{ t('collaboration.download') }}
        </button>
        <button type="button" class="cv-btn" :disabled="loading" @click="loadPage">
          <t-icon name="refresh" size="14px" /> {{ t('collaboration.refresh') }}
        </button>
      </div>
    </div>

    <div v-if="loading" class="cv-empty">{{ t('common.loading') }}</div>
    <div v-else-if="loadError" class="cv-empty">{{ loadError }}</div>
    <template v-else>
      <div class="timeline-line">
        <div v-for="node in timelineNodes" :key="node.key" class="timeline-node">
          <span class="node-dot" :class="node.dotClass" />
          <b>{{ node.label }}</b>
          <span v-if="node.badge" class="node-badge">{{ node.badge }}</span>
          <span class="node-date">{{ node.date }}</span>
        </div>
      </div>

      <div class="compare-bar">
        <select v-model="fromVersionId" class="version-select" @change="loadCompare">
          <option v-for="version in versions" :key="version.id" :value="version.id">{{ version.version_label }}</option>
        </select>
        <span class="compare-arrow">⇄</span>
        <select v-model="toVersionId" class="version-select" @change="loadCompare">
          <option v-for="version in versions" :key="version.id" :value="version.id">{{ version.version_label }}</option>
        </select>
        <span class="compare-stats">
          <span class="cv-add">+{{ stats.added }}</span>
          <span class="cv-del">−{{ stats.removed }}</span>
          <span class="cv-muted">{{ t('collaboration.chunksUnit') }}</span>
        </span>
        <button type="button" class="cv-btn" @click="swapCompare">⇄ {{ t('collaboration.swapDirection') }}</button>
        <div class="cv-seg push-right">
          <button type="button" :class="{ 'is-active': diffMode === 'unified' }" @click="diffMode = 'unified'">Unified</button>
          <button type="button" :class="{ 'is-active': diffMode === 'split' }" @click="diffMode = 'split'">Split</button>
        </div>
      </div>

      <div v-if="!diffs.length" class="cv-empty">{{ t('collaboration.noVersionDiff') }}</div>
      <div v-else class="diff-list">
        <article v-for="diff in diffs" :key="diff.chunkId" class="cv-card diff-item">
          <div class="cv-card-head">
            <span class="chunk-id">#{{ diff.chunkIndex + 1 }}</span>
            <span class="cv-muted small">
              {{ t('collaboration.similarity', { value: diff.similarity + '%' }) }} ·
              <span class="cv-add">+{{ diff.added }}</span>
              <span class="cv-del">−{{ diff.removed }}</span>
            </span>
          </div>
          <DiffBlock :before="diff.before" :after="diff.after" :mode="diffMode" />
        </article>
      </div>

      <div class="version-list">
        <div class="list-head">{{ t('collaboration.allVersions') }}</div>
        <div v-for="version in versions" :key="version.id" class="version-row">
          <span class="version-pill" :class="{ 'is-new': version.is_current }">{{ version.version_label }}</span>
          <div class="version-row-main">
            <b>{{ version.title || versionSourceLabel(version.source_type) }}</b>
            <small>{{ versionSourceLabel(version.source_type) }} · {{ version.created_by_name || version.created_by }}</small>
          </div>
          <span class="version-row-stats">{{ version.chunk_count }} {{ t('collaboration.chunksUnit') }}</span>
          <span class="version-row-time">{{ formatDateTime(version.created_at) }}</span>
          <button type="button" class="cv-icon-btn" :title="t('collaboration.download')" @click="downloadVersion(version)">
            <t-icon name="download" size="13px" />
          </button>
          <button
            v-if="canRollback && !version.is_current"
            type="button"
            class="cv-icon-btn is-warning"
            :title="t('collaboration.rollback')"
            @click="askRollback(version)"
          >
            <t-icon name="rollback" size="13px" />
          </button>
        </div>
      </div>
    </template>

    <t-dialog v-model:visible="rollbackVisible" :header="t('collaboration.rollback')" width="500px" :footer="false">
      <div class="collab-v2 dialog-body">
        <p class="cv-muted">
          {{ t('collaboration.rollbackHint', { version: rollbackTarget?.version_label || '' }) }}
        </p>
        <textarea v-model="rollbackComment" class="cv-textarea" rows="3" :placeholder="t('collaboration.rollbackCommentPlaceholder')" />
        <div class="dialog-actions">
          <button type="button" class="cv-btn" @click="rollbackVisible = false">{{ t('common.cancel') }}</button>
          <button type="button" class="cv-btn cv-btn-primary" :disabled="rollingBack" @click="confirmRollback">
            {{ t('collaboration.rollback') }}
          </button>
        </div>
      </div>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  getDocVersion,
  listDocChangeRequests,
  listDocVersions,
  rollbackDocVersion,
  type DocChangeRequest,
  type DocVersion,
  type DocVersionChunk,
} from '@/api/collaboration'
import { useAuthStore } from '@/stores/auth'
import DiffBlock from './DiffBlock.vue'
import { diffMarkdownVersions } from './diff'
import { formatDateTime, CR_STATUS_LABELS } from './status'

const { t } = useI18n()
const authStore = useAuthStore()

const props = withDefaults(defineProps<{
  kbId: string
  knowledgeId: string
  docTitle?: string
  active?: boolean
}>(), {
  docTitle: '',
  active: true,
})

const emit = defineEmits<{ (e: 'changed'): void }>()

const loading = ref(false)
const loadError = ref('')
const versions = ref<DocVersion[]>([])
const activeCRs = ref<DocChangeRequest[]>([])
const fromVersionId = ref('')
const toVersionId = ref('')
const diffMode = ref<'unified' | 'split'>('unified')
const diffs = ref<ReturnType<typeof diffMarkdownVersions>>([])
const rollbackVisible = ref(false)
const rollbackTarget = ref<DocVersion | null>(null)
const rollbackComment = ref('')
const rollingBack = ref(false)

const canRollback = computed(() => authStore.hasRole('admin') || authStore.hasRole('owner'))
const currentVersion = computed(() => versions.value.find((version) => version.is_current))
const stats = computed(() => ({
  added: diffs.value.reduce((sum, diff) => sum + diff.added, 0),
  removed: diffs.value.reduce((sum, diff) => sum + diff.removed, 0),
}))

const timelineNodes = computed(() => {
  const ordered = [...versions.value].reverse()
  const nodes: Array<{ key: string; label: string; date: string; dotClass: string; badge?: string }> = ordered.map((version) => ({
    key: version.id,
    label: version.version_label,
    date: `${formatDateTime(version.created_at).slice(0, 5)} · ${versionSourceLabel(version.source_type)}${version.source_id ? ` #${version.source_id.slice(0, 4)}` : ''}`,
    dotClass: version.is_current ? 'is-current' : 'is-past',
    badge: version.is_current ? t('collaboration.current') : '',
  }))
  for (const cr of activeCRs.value) {
    nodes.push({
      key: `cr-${cr.id}`,
      label: CR_STATUS_LABELS[cr.status] || t('collaboration.draftNode'),
      date: `#${cr.seq_id} ${cr.title || ''}`,
      dotClass: 'is-draft',
    })
  }
  return nodes
})

function versionSourceLabel(source: string) {
  const map: Record<string, string> = {
    upload: t('collaboration.versionSourceUpload'),
    manual_edit: t('collaboration.versionSourceManual'),
    cr_merge: t('collaboration.versionSourceMerge'),
    rollback: t('collaboration.versionSourceRollback'),
  }
  return map[source] || source
}

async function loadPage() {
  if (!props.kbId || !props.knowledgeId) return
  loading.value = true
  loadError.value = ''
  try {
    const [versionResult, crResult] = await Promise.all([
      listDocVersions(props.kbId, props.knowledgeId),
      listDocChangeRequests(props.kbId, {
        knowledge_id: props.knowledgeId,
        page_size: 100,
      }).catch(() => null),
    ])
    versions.value = (versionResult as any)?.data || []
    const crs: DocChangeRequest[] = (crResult as any)?.data?.items || []
    activeCRs.value = crs.filter((cr) =>
      cr.status === 'draft' || cr.status === 'pending_review' || cr.status === 'changes_requested',
    )
    if (versions.value.length) {
      if (!toVersionId.value || !versions.value.some((version) => version.id === toVersionId.value)) {
        toVersionId.value = versions.value[0].id
      }
      if (!fromVersionId.value || !versions.value.some((version) => version.id === fromVersionId.value)) {
        fromVersionId.value = versions.value[1]?.id || versions.value[0].id
      }
      await loadCompare()
    }
  } catch (error: any) {
    loadError.value = error?.message || t('collaboration.loadFailed')
  } finally {
    loading.value = false
  }
}

async function loadCompare() {
  if (!fromVersionId.value || !toVersionId.value) {
    diffs.value = []
    return
  }
  try {
    const fromResult: any = await getDocVersion(props.kbId, props.knowledgeId, fromVersionId.value)
    const toResult: any = await getDocVersion(props.kbId, props.knowledgeId, toVersionId.value)
    const fromChunks: DocVersionChunk[] = fromResult?.chunks || fromResult?.data?.chunks || []
    const toChunks: DocVersionChunk[] = toResult?.chunks || toResult?.data?.chunks || []
    diffs.value = diffMarkdownVersions(fromChunks as any, toChunks as any)
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('collaboration.loadFailed'))
  }
}

function swapCompare() {
  const from = fromVersionId.value
  fromVersionId.value = toVersionId.value
  toVersionId.value = from
  void loadCompare()
}

async function downloadVersion(version: DocVersion) {
  try {
    const result: any = await getDocVersion(props.kbId, props.knowledgeId, version.id)
    const chunks: DocVersionChunk[] = result?.chunks || result?.data?.chunks || []
    downloadText(`${props.docTitle || 'document'}-${version.version_label}.md`, chunks.map((chunk) => chunk.content).join('\n\n'))
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('collaboration.downloadFailed'))
  }
}

function downloadCurrent() {
  const current = currentVersion.value
  if (current) void downloadVersion(current)
}

function downloadText(fileName: string, content: string) {
  const blob = new Blob([content], { type: 'text/markdown;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = fileName
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
}

function askRollback(version: DocVersion) {
  rollbackTarget.value = version
  rollbackComment.value = ''
  rollbackVisible.value = true
}

async function confirmRollback() {
  if (!rollbackTarget.value) return
  rollingBack.value = true
  try {
    const result: any = await rollbackDocVersion(props.kbId, props.knowledgeId, rollbackTarget.value.id, rollbackComment.value.trim())
    if (result?.success === false) {
      MessagePlugin.error(result?.message || t('collaboration.rollbackFailed'))
      return
    }
    rollbackVisible.value = false
    MessagePlugin.success(t('collaboration.rollbackSuccess'))
    emit('changed')
    await loadPage()
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('collaboration.rollbackFailed'))
  } finally {
    rollingBack.value = false
  }
}

watch(
  [() => props.kbId, () => props.knowledgeId, () => props.active],
  ([kbId, knowledgeId, active]) => {
    if (active && kbId && knowledgeId) void loadPage()
  },
  { immediate: true },
)
</script>

<style scoped lang="less">
@import '@/styles/collaboration.css';

.version-history {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.version-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.version-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.version-name {
  font-size: 15px;
  font-weight: 650;
}

.version-actions {
  display: flex;
  gap: 6px;
}

.timeline-line {
  display: flex;
  align-items: flex-start;
  gap: 0;
  padding: 14px 8px 4px;
  overflow-x: auto;
}

.timeline-node {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 3px;
  min-width: 118px;
  position: relative;
}

.timeline-node + .timeline-node::before {
  content: '';
  position: absolute;
  left: -50%;
  top: 8px;
  width: 100%;
  height: 2px;
  background: var(--cv-line);
}

.node-dot {
  width: 16px;
  height: 16px;
  border-radius: 99px;
  border: 3px solid #fff;
  box-shadow: 0 0 0 1px var(--cv-line);
  background: #d1d5db;
  z-index: 1;
}

.node-dot.is-current {
  background: var(--cv-brand);
  box-shadow: 0 0 0 1px var(--cv-brand);
}

.node-dot.is-draft {
  background: #fff;
  border: 2px dashed var(--cv-faint);
}

.timeline-node b {
  font-size: 12px;
  color: var(--cv-ink);
}

.node-badge {
  border-radius: 99px;
  padding: 0 8px;
  background: var(--cv-brand-bg);
  color: var(--cv-brand-strong);
  font-size: 10px;
  font-weight: 600;
}

.node-date {
  color: var(--cv-faint);
  font-size: 10px;
  text-align: center;
  line-height: 1.4;
}

.compare-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  border: 1px solid var(--cv-line);
  border-radius: 10px;
  padding: 9px 12px;
  background: var(--cv-bg);
}

.version-select {
  border: 1px solid var(--cv-line);
  border-radius: 8px;
  padding: 4px 12px;
  background: #fff;
  font: inherit;
  font-size: 12px;
  font-weight: 600;
  color: var(--cv-ink);
}

.compare-arrow {
  color: var(--cv-faint);
}

.compare-stats {
  font-variant-numeric: tabular-nums;
  font-size: 12px;
}

.push-right {
  margin-left: auto;
}

.diff-item {
  margin-bottom: 12px;
  overflow: hidden;
}

.chunk-id {
  border-radius: 6px;
  padding: 0 9px;
  background: var(--cv-brand-bg);
  color: var(--cv-brand-strong);
  font-size: 12px;
  font-weight: 650;
}

.small {
  font-size: 11px;
}

.version-list {
  border: 1px solid var(--cv-line);
  border-radius: 10px;
  overflow: hidden;
}

.list-head {
  padding: 8px 14px;
  background: var(--cv-bg);
  color: var(--cv-sub);
  font-size: 12px;
}

.version-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  border-top: 1px solid var(--cv-line);
  font-size: 12px;
}

.version-row:hover {
  background: var(--cv-bg);
}

.version-pill {
  border: 1px solid var(--cv-line);
  border-radius: 7px;
  padding: 0 10px;
  font-size: 12px;
  font-weight: 650;
  flex: none;
}

.version-pill.is-new {
  background: var(--cv-brand-bg);
  border-color: var(--cv-brand-border);
  color: var(--cv-brand-strong);
}

.version-row-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.version-row-main b {
  color: var(--cv-ink);
}

.version-row-main small {
  color: var(--cv-faint);
  font-size: 10px;
}

.version-row-stats,
.version-row-time {
  color: var(--cv-sub);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.dialog-body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
