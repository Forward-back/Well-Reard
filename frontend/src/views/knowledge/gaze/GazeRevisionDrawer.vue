<template>
  <SettingDrawer v-model:visible="drawerVisible" class="gaze-revision-drawer" :title="drawerTitle" icon="history"
    width="760px" :min-width="560" :max-width="1280" storage-key="setting-drawer:width:gaze-revision-history"
    hide-footer>
    <div class="gaze-rev-layout">
      <!-- Version list -->
      <aside class="gaze-rev-list">
        <div class="gaze-rev-list-items">
          <div v-if="currentPage" class="gaze-rev-item"
            :class="{ 'gaze-rev-item--active': selectedVersion === currentPage.version }"
            @click="selectCurrent">
            <div class="gaze-rev-item-primary">
              <span class="gaze-rev-version">v{{ currentPage.version }}</span>
              <span class="gaze-rev-current-label">{{ t('knowledgeEditor.gazeBrowser.revisionCurrent') }}</span>
            </div>
            <div class="gaze-rev-item-secondary">
              <span>{{ sourceLabel(currentPage.last_edit_source) }}</span>
              <span class="gaze-rev-time">{{ formatShortTime(currentPage.updated_at) }}</span>
            </div>
          </div>

          <div v-for="rev in revisions" :key="rev.id" class="gaze-rev-item"
            :class="{ 'gaze-rev-item--active': selectedVersion === rev.version }" @click="selectRevision(rev)">
            <div class="gaze-rev-item-primary">
              <span class="gaze-rev-version">v{{ rev.version }}</span>
            </div>
            <div class="gaze-rev-item-secondary">
              <span>{{ sourceLabel(rev.edit_source) }}</span>
              <span class="gaze-rev-time">{{ formatShortTime(rev.edited_at) }}</span>
            </div>
          </div>

          <div v-if="revisions.length < total" class="gaze-rev-load-more">
            <t-button size="small" variant="outline" theme="default" :loading="loadingList" block @click="loadMore">
              {{ t('knowledgeEditor.gazeBrowser.loadMoreShort') }}
            </t-button>
          </div>
        </div>
        <div v-if="!loadingList && revisions.length === 0" class="gaze-rev-empty">
          {{ t('knowledgeEditor.gazeBrowser.revisionEmpty') }}
        </div>
      </aside>

      <!-- Detail pane -->
      <div class="gaze-rev-detail">
        <template v-if="selectedVersion !== null && canShowDiff">
          <div class="gaze-rev-detail-head">
            <div class="gaze-rev-detail-context">
              <div class="gaze-rev-detail-range">{{ versionRangeLabel }}</div>
              <div v-if="contextHint" class="gaze-rev-detail-sub">{{ contextHint }}</div>
            </div>
            <div class="gaze-rev-detail-controls">
              <div v-if="viewModeOptions.length > 1" class="gaze-rev-view-switch" role="tablist"
                :aria-label="t('knowledgeEditor.gazeBrowser.revisionViewModeLabel')">
                <button v-for="option in viewModeOptions" :key="option.value" type="button"
                  class="gaze-rev-view-switch-btn" role="tab" :aria-selected="viewMode === option.value"
                  :class="{ active: viewMode === option.value }" @click="viewMode = option.value">
                  {{ option.label }}
                </button>
              </div>
              <t-popconfirm v-if="canEdit && selectedRevision"
                :content="t('knowledgeEditor.gazeBrowser.revertConfirm', { ver: selectedRevision.version })"
                @confirm="doRevert">
                <t-button size="small" variant="text" theme="warning" :loading="reverting">
                  <template #icon><t-icon name="rollback" /></template>
                  {{ t('knowledgeEditor.gazeBrowser.revertBtn') }}
                </t-button>
              </t-popconfirm>
            </div>
          </div>

          <div v-if="loadingDetail || diffLoading" class="gaze-rev-detail-loading">
            <t-loading size="small" />
            <span>{{ t('knowledgeEditor.gazeBrowser.loading') }}</span>
          </div>

          <div v-else-if="viewMode !== 'raw'" class="gaze-rev-diff">
            <div v-if="diffSections.length === 0" class="gaze-rev-empty-diff">
              {{ t('knowledgeEditor.gazeBrowser.revisionDiffEmpty') }}
            </div>
            <template v-for="section in diffSections" :key="section.field">
              <div class="gaze-rev-diff-block">
                <div class="gaze-rev-diff-block-label">{{ revisionDiffFieldLabel(section.field) }}</div>
                <pre class="gaze-rev-diff-block-body"><span v-for="(line, idx) in section.lines"
                  :key="`${section.field}-${idx}`"
                  :class="['gaze-rev-diff-line', `gaze-rev-diff-line--${line.type}`]">{{ diffPrefix(line.type) }}{{ line.text }}
</span></pre>
              </div>
            </template>
          </div>

          <div v-else-if="selectedRevision" class="gaze-rev-raw">
            <pre class="gaze-rev-raw-body">{{ rawRevisionText }}</pre>
          </div>
        </template>

        <div v-else class="gaze-rev-detail-hint">{{ t('knowledgeEditor.gazeBrowser.revisionSelectHint') }}</div>
      </div>
    </div>
  </SettingDrawer>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { MessagePlugin } from 'tdesign-vue-next'
import SettingDrawer from '@/components/settings/SettingDrawer.vue'
import {
  listGazeRevisions,
  getGazeRevision,
  revertGazePage,
  type GazePage,
  type GazePageRevision,
} from '@/api/gaze'
import { diffGazeRevision, type GazeRevisionDiffField, type GazeRevisionSnapshot } from '@/utils/gazeRevisionDiff'

type ViewMode = 'incremental' | 'cumulative' | 'raw'

interface DiffPair {
  fromVersion: number
  toVersion: number
  from: GazeRevisionSnapshot
  to: GazeRevisionSnapshot
}

const props = defineProps<{
  visible: boolean
  kbId: string
  slug: string
  currentPage: GazePage | null
  canEdit?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:visible', visible: boolean): void
  (e: 'reverted', page: GazePage): void
}>()

const { t } = useI18n()

const drawerVisible = computed({
  get: () => props.visible,
  set: (val) => emit('update:visible', val),
})

const drawerTitle = computed(() =>
  t('knowledgeEditor.gazeBrowser.historyTitle', { title: props.currentPage?.title || props.slug }),
)

const PAGE_SIZE = 50

const revisions = ref<GazePageRevision[]>([])
const total = ref(0)
const loadingList = ref(false)

const selectedVersion = ref<number | null>(null)
const selectedRevision = ref<GazePageRevision | null>(null)
const detailContent = ref('')
const loadingDetail = ref(false)
const viewMode = ref<ViewMode>('incremental')
const reverting = ref(false)
const diffLoading = ref(false)
const diffPair = ref<DiffPair | null>(null)

const snapshotCache = new Map<number, GazeRevisionSnapshot>()

const currentVersion = computed(() => props.currentPage?.version ?? null)

const isCurrentSelected = computed(() =>
  currentVersion.value !== null && selectedVersion.value === currentVersion.value,
)

const canShowDiff = computed(() => {
  if (selectedVersion.value === null || !props.currentPage) return false
  if (viewMode.value === 'raw') return true
  if (viewMode.value === 'cumulative') {
    return !isCurrentSelected.value && selectedVersion.value! < currentVersion.value!
  }
  // incremental: v1 diffs from empty; later versions diff from the previous one
  return selectedVersion.value! >= 1
})

const viewModeOptions = computed(() => {
  if (isCurrentSelected.value) return []
  const ver = selectedVersion.value!
  const cur = currentVersion.value!
  const options: Array<{ value: ViewMode; label: string }> = []
  options.push({ value: 'incremental', label: t('knowledgeEditor.gazeBrowser.revisionDiffIncremental') })
  if (ver < cur) {
    options.push({ value: 'cumulative', label: t('knowledgeEditor.gazeBrowser.revisionDiffCumulative') })
  }
  options.push({ value: 'raw', label: t('knowledgeEditor.gazeBrowser.revisionRaw') })
  return options
})

const versionRangeLabel = computed(() => {
  if (viewMode.value === 'raw') {
    return selectedRevision.value ? `v${selectedRevision.value.version}` : ''
  }
  if (!diffPair.value) return ''
  if (diffPair.value.fromVersion < 1) {
    return t('knowledgeEditor.gazeBrowser.revisionInitialRange', { ver: diffPair.value.toVersion })
  }
  return `v${diffPair.value.fromVersion} → v${diffPair.value.toVersion}`
})

const contextHint = computed(() => {
  if (viewMode.value === 'raw' && selectedRevision.value) {
    return [
      sourceLabel(selectedRevision.value.edit_source),
      formatShortTime(selectedRevision.value.edited_at),
    ].filter(Boolean).join(' · ')
  }
  if (viewMode.value === 'incremental') {
    if ((isCurrentSelected.value && (currentVersion.value ?? 0) <= 1)
      || selectedVersion.value === 1) {
      return t('knowledgeEditor.gazeBrowser.revisionInitialCreationHint')
    }
    return isCurrentSelected.value
      ? t('knowledgeEditor.gazeBrowser.revisionLatestChangeHint')
      : t('knowledgeEditor.gazeBrowser.revisionIncrementalHint', { ver: selectedVersion.value ?? 0 })
  }
  if (viewMode.value === 'cumulative') {
    return t('knowledgeEditor.gazeBrowser.revisionCumulativeHint')
  }
  return ''
})

const rawRevisionText = computed(() => {
  if (!selectedRevision.value) return detailContent.value
  const parts: string[] = []
  if (selectedRevision.value.title) parts.push(selectedRevision.value.title)
  if (selectedRevision.value.summary) {
    if (parts.length) parts.push('')
    parts.push(selectedRevision.value.summary)
  }
  if (detailContent.value) {
    if (parts.length) parts.push('')
    parts.push(detailContent.value)
  }
  return parts.join('\n')
})

const diffSections = computed(() => {
  if (!diffPair.value || viewMode.value === 'raw') return []
  return diffGazeRevision(diffPair.value.from, diffPair.value.to)
})

watch(
  () => [props.visible, props.slug] as const,
  ([visible]) => {
    if (visible && props.slug) {
      resetAndLoad()
    }
  },
)

watch(
  () => viewModeOptions.value,
  (options) => {
    if (options.length === 0) return
    if (!options.some((option) => option.value === viewMode.value)) {
      viewMode.value = options[0].value
    }
  },
)

watch(
  () => [selectedVersion.value, viewMode.value, props.currentPage?.version, props.currentPage?.content,
    props.currentPage?.title, props.currentPage?.summary] as const,
  () => {
    if (props.visible && viewMode.value !== 'raw') {
      void loadDiffPair()
    }
  },
)

function snapshotFromPage(page: GazePage): GazeRevisionSnapshot {
  return {
    title: page.title || '',
    summary: page.summary || '',
    content: page.content || '',
  }
}

function snapshotFromRevisionData(data: GazePageRevision, content: string): GazeRevisionSnapshot {
  return {
    title: data.title || '',
    summary: data.summary || '',
    content,
  }
}

async function loadVersionSnapshot(version: number): Promise<GazeRevisionSnapshot> {
  if (!props.currentPage) {
    return { title: '', summary: '', content: '' }
  }
  if (version === props.currentPage.version) {
    return snapshotFromPage(props.currentPage)
  }
  const cached = snapshotCache.get(version)
  if (cached) return cached
  const res = await getGazeRevision(props.kbId, props.slug, version)
  const data = (res as any).data || (res as any)
  const snap = snapshotFromRevisionData(data, data.content || '')
  snapshotCache.set(version, snap)
  return snap
}

let diffRequestSeq = 0

async function loadDiffPair() {
  const seq = ++diffRequestSeq
  if (!props.currentPage || selectedVersion.value === null || !canShowDiff.value) {
    diffPair.value = null
    diffLoading.value = false
    return
  }

  const currentVer = props.currentPage.version
  let fromVer = 0
  let toVer = 0

  if (viewMode.value === 'incremental') {
    toVer = isCurrentSelected.value ? currentVer : selectedVersion.value!
    fromVer = toVer - 1
  } else {
    fromVer = selectedVersion.value!
    toVer = currentVer
  }

  if (fromVer < 0 || toVer < 1 || fromVer >= toVer) {
    diffPair.value = null
    diffLoading.value = false
    return
  }

  diffLoading.value = true
  try {
    const from = fromVer < 1
      ? { title: '', summary: '', content: '' }
      : await loadVersionSnapshot(fromVer)
    if (seq !== diffRequestSeq) return
    const to = await loadVersionSnapshot(toVer)
    if (seq !== diffRequestSeq) return
    diffPair.value = { fromVersion: fromVer, toVersion: toVer, from, to }
  } catch (e: any) {
    if (seq !== diffRequestSeq) return
    diffPair.value = null
    MessagePlugin.error(e?.message || t('knowledgeEditor.gazeBrowser.revisionLoadFailed'))
  } finally {
    if (seq === diffRequestSeq) diffLoading.value = false
  }
}

function resetAndLoad() {
  detailRequestSeq++
  diffRequestSeq++
  snapshotCache.clear()
  revisions.value = []
  total.value = 0
  selectedVersion.value = props.currentPage?.version ?? null
  selectedRevision.value = null
  detailContent.value = ''
  loadingDetail.value = false
  diffPair.value = null
  diffLoading.value = false
  viewMode.value = 'incremental'
  loadList(0)
  void loadDiffPair()
}

async function loadList(offset: number) {
  loadingList.value = true
  try {
    const res = await listGazeRevisions(props.kbId, props.slug, { limit: PAGE_SIZE, offset })
    const data = (res as any).data || (res as any)
    const items: GazePageRevision[] = data.revisions || []
    if (offset === 0) {
      revisions.value = items
    } else {
      // Snapshots are created while the user pages through, which shifts the
      // newest-first window. Drop versions we already hold so an overlapping
      // page cannot produce duplicate rows (and duplicate :key values).
      const seen = new Set(revisions.value.map((r) => r.version))
      revisions.value = [...revisions.value, ...items.filter((r) => !seen.has(r.version))]
    }
    total.value = data.total ?? revisions.value.length
  } catch (e: any) {
    MessagePlugin.error(e?.message || t('knowledgeEditor.gazeBrowser.revisionLoadFailed'))
  } finally {
    loadingList.value = false
  }
}

function loadMore() {
  if (loadingList.value) return
  loadList(revisions.value.length)
}

function selectCurrent() {
  detailRequestSeq++
  selectedVersion.value = props.currentPage?.version ?? null
  selectedRevision.value = null
  detailContent.value = ''
  loadingDetail.value = false
  viewMode.value = 'incremental'
}

// Monotonic token guarding the detail fetch: clicking through the list fires
// overlapping requests, and a slow earlier one must not overwrite the body of
// the revision the user is actually looking at.
let detailRequestSeq = 0

async function selectRevision(rev: GazePageRevision) {
  const seq = ++detailRequestSeq
  selectedVersion.value = rev.version
  selectedRevision.value = rev
  detailContent.value = ''
  viewMode.value = 'incremental'
  loadingDetail.value = true
  try {
    const res = await getGazeRevision(props.kbId, props.slug, rev.version)
    if (seq !== detailRequestSeq) return
    const data = (res as any).data || (res as any)
    selectedRevision.value = { ...rev, ...data }
    detailContent.value = data.content || ''
    snapshotCache.set(rev.version, snapshotFromRevisionData(data, data.content || ''))
  } catch (e: any) {
    if (seq !== detailRequestSeq) return
    MessagePlugin.error(e?.message || t('knowledgeEditor.gazeBrowser.revisionLoadFailed'))
  } finally {
    if (seq === detailRequestSeq) loadingDetail.value = false
  }
}

async function doRevert() {
  if (!selectedRevision.value) return
  reverting.value = true
  try {
    const res = await revertGazePage(props.kbId, props.slug, selectedRevision.value.version)
    const updated = ((res as any).data || (res as any)) as GazePage
    MessagePlugin.success(t('knowledgeEditor.gazeBrowser.revertSuccess', { ver: selectedRevision.value.version }))
    emit('reverted', updated)
    // Stay open: reload so the just-created snapshot of the pre-revert
    // version shows up and the "current" entry reflects the new version.
    resetAndLoad()
  } catch (e: any) {
    MessagePlugin.error(e?.message || t('knowledgeEditor.gazeBrowser.revertFailed'))
  } finally {
    reverting.value = false
  }
}

function diffPrefix(type: 'same' | 'add' | 'del'): string {
  return type === 'add' ? '+ ' : type === 'del' ? '- ' : '  '
}

function revisionDiffFieldLabel(field: GazeRevisionDiffField): string {
  switch (field) {
    case 'title':
      return t('knowledgeEditor.gazeBrowser.revisionDiffTitle')
    case 'summary':
      return t('knowledgeEditor.gazeBrowser.revisionDiffSummary')
    default:
      return t('knowledgeEditor.gazeBrowser.revisionDiffContent')
  }
}

function sourceLabel(source?: string): string {
  switch (source) {
    case 'user':
      return t('knowledgeEditor.gazeBrowser.editSourceUser')
    case 'agent':
      return t('knowledgeEditor.gazeBrowser.editSourceAgent')
    case 'revert':
      return t('knowledgeEditor.gazeBrowser.editSourceRevert')
    default:
      return t('knowledgeEditor.gazeBrowser.editSourcePipeline')
  }
}

function formatShortTime(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  const now = new Date()
  if (d.toDateString() === now.toDateString()) {
    return d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
  }
  return d.toLocaleString(undefined, {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

</script>

<style scoped>
.gaze-rev-layout {
  display: flex;
  flex: 1;
  min-height: 0;
  height: 100%;
  align-items: stretch;
}

.gaze-rev-list {
  width: 220px;
  flex-shrink: 0;
  min-height: 0;
  overflow: hidden;
  padding: 12px 0;
  border-right: 1px solid var(--td-component-stroke);
  display: flex;
  flex-direction: column;
  background: var(--td-bg-color-container);
}

.gaze-rev-list-items {
  flex: 0 0 auto;
  overflow-y: auto;
  min-height: 0;
  padding: 0 8px 0 14px;
}

.gaze-rev-item {
  border: none;
  border-radius: 6px;
  padding: 6px 10px 6px 14px;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}

.gaze-rev-item:hover,
.gaze-rev-item--active {
  background: var(--td-bg-color-container-hover);
}

.gaze-rev-item--active .gaze-rev-version,
.gaze-rev-item--active .gaze-rev-current-label {
  color: var(--td-brand-color);
}

.gaze-rev-item-primary {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.gaze-rev-version {
  font-weight: 400;
  font-size: 14px;
  line-height: 20px;
  font-family: var(--td-font-family-mono, monospace);
  color: var(--td-text-color-primary);
  transition: color 0.15s ease;
}

.gaze-rev-current-label {
  font-size: 12px;
  line-height: 16px;
  color: var(--td-text-color-placeholder);
  transition: color 0.15s ease;
}

.gaze-rev-item-secondary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: 2px;
  min-width: 0;
  font-size: 11px;
  line-height: 16px;
  color: var(--td-text-color-placeholder);
}

.gaze-rev-time {
  font-size: 11px;
  color: var(--td-text-color-placeholder);
  white-space: nowrap;
  flex-shrink: 0;
  font-variant-numeric: tabular-nums;
}

.gaze-rev-load-more {
  padding: 8px 0 4px;
}

.gaze-rev-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px 14px;
  text-align: center;
  font-size: 12px;
  line-height: 1.5;
  color: var(--td-text-color-placeholder);
}

.gaze-rev-detail {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 14px 18px 18px;
}

.gaze-rev-detail-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.gaze-rev-detail-context {
  min-width: 0;
  flex: 1;
}

.gaze-rev-detail-range {
  font-family: var(--td-font-family-mono, monospace);
  font-size: 15px;
  font-weight: 600;
  line-height: 1.4;
  color: var(--td-text-color-primary);
}

.gaze-rev-detail-sub {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--td-text-color-placeholder);
}

.gaze-rev-detail-controls {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.gaze-rev-view-switch {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 2px;
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);
}

.gaze-rev-view-switch-btn {
  padding: 5px 10px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 1.4;
  white-space: nowrap;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}

.gaze-rev-view-switch-btn:hover {
  color: var(--td-text-color-primary);
}

.gaze-rev-view-switch-btn.active {
  color: var(--td-brand-color);
  background: var(--td-bg-color-container);
  font-weight: 500;
}

.gaze-rev-detail-hint {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--td-text-color-placeholder);
  padding: 24px;
  text-align: center;
  font-size: 13px;
  line-height: 1.5;
}

.gaze-rev-detail-loading {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--td-text-color-placeholder);
  font-size: 13px;
}

.gaze-rev-diff {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.gaze-rev-empty-diff {
  padding: 24px 0;
  text-align: center;
  font-size: 13px;
  color: var(--td-text-color-placeholder);
}

.gaze-rev-diff-block {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.gaze-rev-diff-block-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--td-text-color-placeholder);
}

.gaze-rev-diff-block-body {
  overflow: auto;
  margin: 0;
  padding: 10px 12px;
  background: var(--td-bg-color-secondarycontainer);
  border-radius: 8px;
  font-size: 12px;
  line-height: 1.7;
  font-family: var(--td-font-family-mono, monospace);
  white-space: pre-wrap;
  word-break: break-word;
}

.gaze-rev-raw {
  flex: 1;
  min-height: 0;
  overflow: auto;
}

.gaze-rev-raw-body {
  margin: 0;
  padding: 12px 14px;
  background: var(--td-bg-color-secondarycontainer);
  border-radius: 8px;
  font-size: 13px;
  line-height: 1.7;
  font-family: var(--td-font-family-mono, monospace);
  white-space: pre-wrap;
  word-break: break-word;
}

.gaze-rev-diff-line {
  display: block;
}

.gaze-rev-diff-line--add {
  background: rgba(7, 192, 95, 0.08);
  color: var(--td-text-color-primary);
}

.gaze-rev-diff-line--del {
  background: rgba(213, 73, 65, 0.06);
  color: var(--td-text-color-secondary);
}
</style>

<style lang="less">
.gaze-revision-drawer {
  .t-drawer__content-wrapper,
  .t-drawer__content {
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .t-drawer__body {
    flex: 1;
    min-height: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .setting-drawer__body {
    flex: 1;
    min-height: 0;
    padding: 0;
    gap: 0;
    animation: none;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }
}
</style>
