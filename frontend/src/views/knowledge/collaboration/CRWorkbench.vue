<template>
  <div class="collab-v2 workbench-page">
    <div class="cv-topbar">
      <button type="button" class="cv-back" @click="backToDoc">‹ {{ t('collaboration.backToDoc') }}</button>
      <span class="cv-topbar-title">{{ t('collaboration.crWorkbenchTitle') }}</span>
      <span class="cv-snapshot" v-if="baseVersionLabel">{{ t('collaboration.basedOnSnapshot', { version: baseVersionLabel }) }}</span>
      <span class="cv-topbar-note">{{ draftNote }}</span>
      <div class="panel-toggles">
        <button
          type="button"
          class="panel-toggle"
          :class="{ 'is-collapsed': leftCollapsed }"
          :title="leftCollapsed ? t('collaboration.expandChunkPanel') : t('collaboration.collapseChunkPanel')"
          :aria-label="leftCollapsed ? t('collaboration.expandChunkPanel') : t('collaboration.collapseChunkPanel')"
          @click="togglePanel('left')"
        >
          <svg viewBox="0 0 20 20" width="18" height="18" fill="none" xmlns="http://www.w3.org/2000/svg">
            <rect x="1.5" y="1.5" width="17" height="17" rx="3" stroke="currentColor" stroke-width="1.2" />
            <line x1="7.5" y1="1.5" x2="7.5" y2="18.5" stroke="currentColor" stroke-width="1.2" />
            <line x1="4" y1="7.5" x2="4" y2="12.5" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" />
          </svg>
        </button>
        <button
          type="button"
          class="panel-toggle is-right"
          :class="{ 'is-collapsed': rightCollapsed }"
          :title="rightCollapsed ? t('collaboration.expandFormPanel') : t('collaboration.collapseFormPanel')"
          :aria-label="rightCollapsed ? t('collaboration.expandFormPanel') : t('collaboration.collapseFormPanel')"
          @click="togglePanel('right')"
        >
          <svg viewBox="0 0 20 20" width="18" height="18" fill="none" xmlns="http://www.w3.org/2000/svg">
            <rect x="1.5" y="1.5" width="17" height="17" rx="3" stroke="currentColor" stroke-width="1.2" />
            <line x1="12.5" y1="1.5" x2="12.5" y2="18.5" stroke="currentColor" stroke-width="1.2" />
            <line x1="16" y1="7.5" x2="16" y2="12.5" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" />
          </svg>
        </button>
      </div>
      <button type="button" class="cv-btn" :disabled="submitting || isEditingChanges" @click="saveDraft(true)">
        {{ t('collaboration.saveDraft') }}
      </button>
    </div>

    <div v-if="loading" class="cv-empty">{{ t('common.loading') }}</div>
    <div v-else-if="loadError" class="cv-empty">
      {{ loadError }}
      <button type="button" class="cv-link" @click="loadPage">{{ t('common.retry') }}</button>
    </div>
    <div v-else class="cv-cols workbench-cols">
      <!-- left: chunk picker -->
      <aside class="workbench-left" :class="{ 'is-collapsed': leftCollapsed }" :inert="leftCollapsed || undefined">
        <div class="left-head">
          <b>{{ t('collaboration.involvedChunks') }}</b>
          <span class="cv-muted">{{ allChunks.length }} {{ t('collaboration.chunksUnit') }}</span>
        </div>
        <div class="chunk-search">
          <t-icon name="search" size="14px" />
          <input v-model="chunkSearch" :placeholder="t('collaboration.searchChunks')" />
        </div>
        <div class="chunk-list">
          <button
            v-for="chunk in filteredChunks"
            :key="chunk.id"
            type="button"
            class="chunk-option"
            :class="{ 'is-selected': selectedChunkIds.includes(chunk.id) }"
            @click="toggleChunk(chunk)"
          >
            <span class="chunk-check" :class="{ on: selectedChunkIds.includes(chunk.id) }">
              <t-icon v-if="selectedChunkIds.includes(chunk.id)" name="check" size="12px" />
            </span>
            <span class="chunk-option-body">
              <span class="chunk-option-title">
                #{{ chunk.index + 1 }}
                <span v-if="suggestionChanged(chunk)" class="chunk-tag is-edit">{{ t('collaboration.edited') }}</span>
                <span v-if="lockFor(chunk.id)" class="chunk-tag is-lock" :title="lockFor(chunk.id)?.title">
                  ⚠ CR#{{ lockFor(chunk.id)?.seq_id }} {{ t('collaboration.inProgress') }}
                </span>
                <span v-if="citationStats[chunk.id]?.last_30_days" class="chunk-tag is-cite">
                  {{ t('collaboration.citationStat', { count: citationStats[chunk.id]?.last_30_days }) }}
                </span>
              </span>
              <span class="chunk-option-excerpt">{{ excerpt(chunk.content) }}</span>
            </span>
          </button>
        </div>
        <div class="left-foot">
          <span>{{ t('collaboration.selectedCount', { selected: selectedChunkIds.length, total: allChunks.length }) }}</span>
          <button type="button" class="cv-link" @click="clearSelection">{{ t('collaboration.clear') }}</button>
        </div>
      </aside>

      <!-- center: compare editor -->
      <main class="workbench-center">
        <div v-if="!selectedChunks.length" class="cv-empty padded">
          {{ t('collaboration.selectChunksHint') }}
        </div>
        <article v-for="chunk in selectedChunks" :key="chunk.id" class="cv-card chunk-editor">
          <div class="cv-card-head">
            <span class="chunk-id">#{{ chunk.index + 1 }}</span>
            <span class="cv-seg">
              <button type="button" :class="{ 'is-active': editorMode[chunk.id] !== 'diff' }" @click="editorMode[chunk.id] = 'edit'">
                {{ t('collaboration.editMode') }}
              </button>
              <button type="button" :class="{ 'is-active': editorMode[chunk.id] === 'diff' }" @click="editorMode[chunk.id] = 'diff'">
                {{ t('collaboration.previewMode') }}
              </button>
            </span>
            <span class="chunk-stats">
              <span class="cv-add">+{{ editorStats(chunk).added }}</span>
              <span class="cv-del">−{{ editorStats(chunk).removed }}</span>
              <span class="cv-muted">· {{ editorStats(chunk).similarity }}%</span>
            </span>
          </div>
          <div v-if="lockFor(chunk.id)" class="lock-banner">
            ⚠ {{ t('collaboration.chunkLockHint', { seq: lockFor(chunk.id)?.seq_id, title: lockFor(chunk.id)?.title }) }}
          </div>
          <SplitDiff v-if="editorMode[chunk.id] === 'diff'" :before="chunk.content" :after="suggestionValue(chunk)" />
          <div v-else class="editor-split">
            <section class="editor-pane">
              <header class="editor-pane-head">
                <span><t-icon name="file" size="12px" /> {{ t('collaboration.originalContent') }}</span>
                <span class="editor-pane-badge">{{ t('collaboration.readOnly') }}</span>
              </header>
              <div class="editor-surface" :ref="(el) => setEditorSurface(chunk.id, el)" @scroll.passive="syncEditorScroll(chunk.id, 'original', $event)">
                <div class="editor-gutter">
                  <span v-for="line in lineCount(chunk.content)" :key="line">{{ line }}</span>
                </div>
                <pre class="editor-code">{{ chunk.content || t('collaboration.noItemContent') }}</pre>
              </div>
            </section>
            <section class="editor-pane">
              <header class="editor-pane-head">
                <span><t-icon name="file" size="12px" /> {{ t('collaboration.suggestedContent') }}</span>
                <span class="editor-pane-badge is-editable">{{ t('collaboration.editable') }}</span>
              </header>
              <div class="editor-surface is-editing">
                <div class="editor-gutter" :ref="(el) => setEditorGutter(chunk.id, el)">
                  <span v-for="line in lineCount(suggestionValue(chunk))" :key="line">{{ line }}</span>
                </div>
                <textarea
                  v-model="suggestions[chunk.id]"
                  class="editor-code editor-textarea"
                  wrap="off"
                  :ref="(el) => setEditorTextarea(chunk.id, el)"
                  :placeholder="t('collaboration.suggestedContentPlaceholder')"
                  :disabled="submitting"
                  spellcheck="false"
                  @scroll.passive="syncEditorScroll(chunk.id, 'suggested', $event)"
                />
              </div>
            </section>
            <div class="editor-status">
              <span>{{ t('collaboration.editorLines', { count: lineCount(suggestionValue(chunk)) }) }}</span>
              <span>UTF-8 · Markdown</span>
            </div>
          </div>
        </article>
      </main>

      <!-- right: submission form -->
      <aside class="workbench-right" :class="{ 'is-collapsed': rightCollapsed }" :inert="rightCollapsed || undefined">
        <section class="form-section">
          <label class="form-label"><b>{{ t('collaboration.issueTitle') }}</b></label>
          <input v-model="title" class="cv-input" :placeholder="t('collaboration.crTitlePlaceholder')" maxlength="255" />
        </section>
        <section class="form-section">
          <label class="form-label"><b>{{ t('collaboration.crDescription') }}</b><span class="cv-muted">Markdown</span></label>
          <textarea v-model="description" class="cv-textarea" rows="4" :placeholder="t('collaboration.crDescriptionPlaceholder')" />
        </section>
        <section class="form-section">
          <label class="form-label"><b>{{ t('collaboration.tags') }}</b></label>
          <input v-model="tagsText" class="cv-input" :placeholder="t('collaboration.tagsPlaceholder')" />
        </section>
        <section class="form-section">
          <div class="form-label">
            <b>{{ t('collaboration.linkedIssuesSection') }}</b>
            <button type="button" class="cv-link" @click="issuePickerVisible = !issuePickerVisible">＋ {{ t('collaboration.addLinkedIssue') }}</button>
          </div>
          <div v-if="selectedIssues.length" class="linked-issues">
            <span v-for="issue in selectedIssues" :key="issue.id" class="linked-issue-chip">
              <t-icon name="chat-bubble-help" size="12px" /> #{{ issue.seq_id }} {{ issue.title }}
              <button type="button" class="chip-remove" @click="removeLinkedIssue(issue.id)">×</button>
            </span>
          </div>
          <div v-else class="cv-muted small">{{ t('collaboration.noLinkedIssue') }}</div>
          <div v-if="issuePickerVisible" class="picker-panel">
            <div v-if="!openIssues.length" class="cv-muted small">{{ t('collaboration.noOpenIssue') }}</div>
            <label v-for="issue in openIssues" :key="issue.id" class="picker-row">
              <input type="checkbox" :checked="linkedIssueIds.includes(issue.id)" @change="toggleLinkedIssue(issue.id)" />
              <span>#{{ issue.seq_id }} {{ issue.title }}</span>
            </label>
          </div>
        </section>
        <section class="form-section">
          <div class="form-label">
            <b>{{ t('collaboration.reviewers') }}</b>
            <button type="button" class="cv-link" @click="reviewerPickerVisible = !reviewerPickerVisible">＋</button>
          </div>
          <div class="reviewer-row">
            <span v-for="member in selectedReviewers" :key="member.user_id" class="cv-avatar" :title="member.username || member.email">
              {{ userInitial(member.username, member.email) }}
            </span>
            <span v-if="!selectedReviewers.length" class="cv-muted small">{{ t('collaboration.noReviewer') }}</span>
          </div>
          <div v-if="reviewerPickerVisible" class="picker-panel">
            <label v-for="member in members" :key="member.user_id" class="picker-row">
              <input type="checkbox" :checked="reviewerIds.includes(member.user_id)" @change="toggleReviewer(member.user_id)" />
              <span>{{ member.username || member.email }} <span class="cv-muted">· {{ member.role }}</span></span>
            </label>
          </div>
        </section>
        <section class="form-section">
          <div class="form-label"><b>{{ t('collaboration.selfCheck') }}</b></div>
          <label class="check-row"><input v-model="checks.source" type="checkbox" /> {{ t('collaboration.checkSource') }}</label>
          <label class="check-row"><input v-model="checks.basis" type="checkbox" /> {{ t('collaboration.checkBasis') }}</label>
          <label class="check-row"><input v-model="checks.conflict" type="checkbox" /> {{ t('collaboration.checkConflict') }}</label>
        </section>
        <section class="form-section">
          <div class="impact-box">
            {{ t('collaboration.impactMerge', { count: changedItems.length, version: nextVersionLabel }) }}
            <span v-if="citedMonthly" class="cv-muted"> · {{ t('collaboration.citedMonthly', { count: citedMonthly }) }}</span>
            <span class="cv-muted">{{ t('collaboration.impactSync') }}</span>
          </div>
        </section>
        <section class="form-section">
          <button type="button" class="cv-btn cv-btn-primary cv-btn-block" :disabled="!canSubmit" @click="submitCR">
            {{ submitting ? t('collaboration.submitting') : t('collaboration.submitCR') }}
          </button>
          <div v-if="!canSubmit" class="form-hint">{{ t('collaboration.submitHint') }}</div>
          <button type="button" class="cv-btn cv-btn-block secondary-block" :disabled="submitting || isEditingChanges" @click="saveDraft(true)">
            {{ t('collaboration.saveDraft') }}
          </button>
        </section>
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  createDocChangeRequest,
  getDocChangeRequest,
  getDocIssue,
  listDocChangeRequests,
  listDocCRChunkLocks,
  listChunkCitationStats,
  listDocIssues,
  listDocVersions,
  resubmitDocChangeRequest,
  submitDocCRDraft,
  updateDocCRDraft,
  type ChunkCitationStat,
  type DocCRChunkLock,
  type DocChangeRequest,
  type DocCRItemPayload,
  type DocIssue,
} from '@/api/collaboration'
import { getKnowledgeDetails, listKnowledgeChunks } from '@/api/knowledge-base'
import { fetchAllTenantMembers, type TenantMember } from '@/api/tenant/members'
import { useAuthStore } from '@/stores/auth'
import SplitDiff from './SplitDiff.vue'
import { diffStats } from './diff'
import { userInitial } from './status'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

interface ChunkOption {
  id: string
  index: number
  content: string
}

function panelStorageKey(side: 'left' | 'right') {
  return `gaze.collab.workbench.${side}Collapsed`
}

function readPanelState(side: 'left' | 'right') {
  try {
    return localStorage.getItem(panelStorageKey(side)) === '1'
  } catch {
    return false
  }
}

function togglePanel(side: 'left' | 'right') {
  const next = side === 'left' ? !leftCollapsed.value : !rightCollapsed.value
  if (side === 'left') leftCollapsed.value = next
  else rightCollapsed.value = next
  try {
    localStorage.setItem(panelStorageKey(side), next ? '1' : '0')
  } catch { /* storage unavailable */ }
}

const kbId = computed(() => String(route.params.kbId || ''))
const docId = computed(() => String(route.params.docId || ''))

const loading = ref(true)
const loadError = ref('')
const submitting = ref(false)
const leftCollapsed = ref(readPanelState('left'))
const rightCollapsed = ref(readPanelState('right'))
const docTitle = ref('')
const allChunks = ref<ChunkOption[]>([])
const selectedChunkIds = ref<string[]>([])
const suggestions = reactive<Record<string, string>>({})
const anchorTexts = reactive<Record<string, string>>({})
const editorMode = reactive<Record<string, 'edit' | 'diff'>>({})
const chunkSearch = ref('')
const title = ref('')
const description = ref('')
const tagsText = ref('')
const reviewerIds = ref<string[]>([])
const members = ref<TenantMember[]>([])
const openIssues = ref<DocIssue[]>([])
const chunkLocks = ref<DocCRChunkLock[]>([])
const citationStats = ref<Record<string, ChunkCitationStat>>({})
const linkedIssueIds = ref<string[]>([])
const issuePickerVisible = ref(false)
const reviewerPickerVisible = ref(false)
const checks = reactive({ source: false, basis: false, conflict: false })
const draftId = ref('')
const editingCRId = ref('')
const editingStatus = ref('')
const draftNote = ref('')
const baseVersionLabel = ref('')
const nextVersionLabel = ref('V1.0')
const hydrating = ref(true)
let autosaveTimer: ReturnType<typeof setTimeout> | null = null
let autosaveGeneration = 0

const filteredChunks = computed(() => {
  const query = chunkSearch.value.trim().toLowerCase()
  if (!query) return allChunks.value
  return allChunks.value.filter((chunk) =>
    String(chunk.index + 1).includes(query.replace(/^#/, ''))
    || (chunk.content || '').toLowerCase().includes(query),
  )
})

const selectedChunks = computed(() =>
  selectedChunkIds.value
    .map((id) => allChunks.value.find((chunk) => chunk.id === id))
    .filter(Boolean) as ChunkOption[],
)

const selectedReviewers = computed(() =>
  members.value.filter((member) => reviewerIds.value.includes(member.user_id)),
)

const selectedIssues = computed(() =>
  openIssues.value.filter((issue) => linkedIssueIds.value.includes(issue.id)),
)

const changedItems = computed<DocCRItemPayload[]>(() =>
  selectedChunks.value
    .map((chunk) => ({
      chunk_id: chunk.id,
      suggested_content: suggestions[chunk.id] ?? chunk.content,
      anchor_text: anchorTexts[chunk.id] || '',
    }))
    .filter((item) => item.suggested_content !== (allChunks.value.find((chunk) => chunk.id === item.chunk_id)?.content || '')),
)
const citedMonthly = computed(() =>
  changedItems.value.reduce((sum, item) => sum + (citationStats.value[item.chunk_id]?.last_30_days || 0), 0),
)

const canSubmit = computed(() =>
  title.value.trim().length > 0
  && changedItems.value.length > 0
  && checks.source
  && checks.basis,
)
const isEditingChanges = computed(() => editingCRId.value !== '' && editingStatus.value === 'changes_requested')

function suggestionChanged(chunk: ChunkOption) {
  return (suggestions[chunk.id] ?? chunk.content) !== chunk.content
}

function lockFor(chunkId: string) {
  return chunkLocks.value.find((lock) => lock.chunk_id === chunkId && lock.cr_id !== editingCRId.value)
}

function suggestionValue(chunk: ChunkOption) {
  return suggestions[chunk.id] ?? chunk.content
}

const editorGutters = new Map<string, HTMLElement>()
const editorSurfaces = new Map<string, HTMLElement>()
const editorTextareas = new Map<string, HTMLTextAreaElement>()

function setEditorGutter(chunkId: string, el: any) {
  const element = el instanceof HTMLElement ? el : null
  if (element) editorGutters.set(chunkId, element)
  else editorGutters.delete(chunkId)
}

function setEditorSurface(chunkId: string, el: any) {
  const element = el instanceof HTMLElement ? el : null
  if (element) editorSurfaces.set(chunkId, element)
  else editorSurfaces.delete(chunkId)
}

function setEditorTextarea(chunkId: string, el: any) {
  const element = el instanceof HTMLTextAreaElement ? el : null
  if (element) editorTextareas.set(chunkId, element)
  else editorTextareas.delete(chunkId)
}

function syncEditorScroll(chunkId: string, source: 'original' | 'suggested', event: Event) {
  const surface = editorSurfaces.get(chunkId)
  const textarea = editorTextareas.get(chunkId)
  const target = event.target as HTMLElement | null
  if (!target || !surface || !textarea) return
  const scrollTop = target.scrollTop
  const counterpart = source === 'original' ? textarea : surface
  if (Math.abs(counterpart.scrollTop - scrollTop) > 1) {
    counterpart.scrollTop = scrollTop
  }
  const gutter = editorGutters.get(chunkId)
  if (gutter) gutter.scrollTop = scrollTop
}

function lineCount(value: string) {
  return Math.max(1, String(value ?? '').split('\n').length)
}

function editorStats(chunk: ChunkOption) {
  return diffStats(chunk.content, suggestionValue(chunk))
}

function excerpt(content: string) {
  const text = String(content || '').replace(/\s+/g, ' ').trim()
  return text.length > 68 ? `${text.slice(0, 68)}…` : text
}

function toggleChunk(chunk: ChunkOption) {
  const index = selectedChunkIds.value.indexOf(chunk.id)
  if (index >= 0) {
    selectedChunkIds.value.splice(index, 1)
  } else {
    selectedChunkIds.value.push(chunk.id)
    if (suggestions[chunk.id] === undefined) suggestions[chunk.id] = chunk.content
    if (editorMode[chunk.id] === undefined) editorMode[chunk.id] = 'edit'
  }
}

function clearSelection() {
  selectedChunkIds.value = []
}

function toggleReviewer(userId: string) {
  const index = reviewerIds.value.indexOf(userId)
  if (index >= 0) reviewerIds.value.splice(index, 1)
  else reviewerIds.value.push(userId)
}

function toggleLinkedIssue(issueId: string) {
  const index = linkedIssueIds.value.indexOf(issueId)
  if (index >= 0) linkedIssueIds.value.splice(index, 1)
  else linkedIssueIds.value.push(issueId)
}

function removeLinkedIssue(issueId: string) {
  linkedIssueIds.value = linkedIssueIds.value.filter((id) => id !== issueId)
}

function parseTags() {
  return tagsText.value.split(/[,，]/).map((tag) => tag.trim()).filter(Boolean)
}

async function loadAllChunks() {
  const first: any = await listKnowledgeChunks(docId.value, 1, 100)
  if (first?.success === false) throw new Error(first?.message || 'load chunks failed')
  const total = Number(first?.total || 0)
  const pages = Math.max(1, Math.ceil(total / 100))
  const all: any[] = [...(first?.data || [])]
  if (pages > 1) {
    const rest = await Promise.all(
      Array.from({ length: pages - 1 }, (_, i) => listKnowledgeChunks(docId.value, i + 2, 100)),
    )
    for (const page of rest as any[]) {
      if (page?.success !== false) all.push(...(page?.data || []))
    }
  }
  return all
    .filter((chunk) => chunk && chunk.id)
    .map((chunk, position): ChunkOption => ({
      id: String(chunk.id),
      index: typeof chunk.chunk_index === 'number' ? chunk.chunk_index : position,
      content: String(chunk.content || ''),
    }))
    .sort((left, right) => left.index - right.index)
}

async function loadPage() {
  loading.value = true
  loadError.value = ''
  hydrating.value = true
  try {
    const [chunks, detail, issueList, versions, locks] = await Promise.all([
      loadAllChunks(),
      getKnowledgeDetails(docId.value).catch(() => null),
      listDocIssues(kbId.value, { knowledge_id: docId.value, page_size: 100, status: 'open' }).catch(() => null),
      listDocVersions(kbId.value, docId.value).catch(() => null),
      listDocCRChunkLocks(kbId.value, docId.value).catch(() => null),
    ])
    allChunks.value = chunks
    docTitle.value = (detail as any)?.data?.title || (detail as any)?.title || ''
    openIssues.value = (issueList as any)?.data?.items || []
    chunkLocks.value = (locks as any)?.data || []
    try {
      const statResult: any = await listChunkCitationStats(kbId.value, chunks.map((chunk) => chunk.id))
      const mapped: Record<string, ChunkCitationStat> = {}
      for (const stat of statResult?.data || []) mapped[stat.chunk_id] = stat
      citationStats.value = mapped
    } catch { /* stats are best-effort */ }
    const versionList = (versions as any)?.data || []
    if (versionList.length) {
      baseVersionLabel.value = versionList[0].version_label
      nextVersionLabel.value = `V1.${Number(versionList[0].version_seq || 0) + 1}`
    }
    const tenantId = Number(authStore.tenant?.id || 0)
    if (tenantId) {
      members.value = await fetchAllTenantMembers(tenantId).catch(() => [])
    }

    const presetIds = String(route.query.chunks || route.query.chunk || '')
      .split(',').map((id) => id.trim()).filter(Boolean)
    const draftParam = String(route.query.draft || '')
    const crParam = String(route.query.cr || '')
    const issueParam = String(route.query.issue || '')
    let hydratedFromCR = false
    if (draftParam || crParam) {
      const crResult: any = await getDocChangeRequest(kbId.value, draftParam || crParam)
      const existing: DocChangeRequest | undefined = crResult?.data
      if (existing && (existing.status === 'draft' || existing.status === 'changes_requested')) {
        hydrateDraft(existing)
        const base = versionList.find((version: any) => version.id === existing.base_version_id)
        if (base) baseVersionLabel.value = base.version_label
        hydratedFromCR = true
      }
    }
    if (!hydratedFromCR) {
      if (title.value === '' && route.query.title) title.value = String(route.query.title)
      if (description.value === '' && route.query.desc) description.value = String(route.query.desc)
      for (const id of presetIds) {
        const chunk = allChunks.value.find((item) => item.id === id)
        if (!chunk) continue
        if (!selectedChunkIds.value.includes(id)) selectedChunkIds.value.push(id)
        if (suggestions[id] === undefined) suggestions[id] = chunk.content
        if (editorMode[id] === undefined) editorMode[id] = 'edit'
      }
      if (route.query.anchor) {
        for (const id of presetIds) anchorTexts[id] = String(route.query.anchor)
      }
      if (issueParam) {
        try {
          const issueResult: any = await getDocIssue(kbId.value, issueParam)
          const issue: DocIssue | undefined = issueResult?.data
          if (issue) {
            if (!openIssues.value.some((item) => item.id === issue.id)) openIssues.value.unshift(issue)
            if (!linkedIssueIds.value.includes(issue.id)) linkedIssueIds.value.push(issue.id)
          }
        } catch { /* ignore missing issue */ }
      }
    }
    if (!hydratedFromCR) {
      // Try resuming the creator's latest draft for this document.
      const drafts: any = await listDocChangeRequests(kbId.value, {
        knowledge_id: docId.value,
        status: 'draft',
        created_by: authStore.currentUserId,
        page_size: 20,
      }).catch(() => null)
      const firstDraft = drafts?.data?.items?.[0]
      if (firstDraft && !draftId.value && selectedChunkIds.value.length === 0 && !title.value.trim()) {
        const detailResult: any = await getDocChangeRequest(kbId.value, firstDraft.id).catch(() => null)
        if (detailResult?.data) hydrateDraft(detailResult.data)
      }
    }
  } catch (error: any) {
    loadError.value = error?.message || t('collaboration.loadFailed')
  } finally {
    loading.value = false
    window.setTimeout(() => { hydrating.value = false }, 50)
  }
}

function hydrateDraft(draft: DocChangeRequest) {
  editingCRId.value = draft.id
  editingStatus.value = draft.status
  if (draft.status === 'draft') draftId.value = draft.id
  title.value = draft.title || ''
  description.value = draft.description || ''
  tagsText.value = (draft.tags || []).join('，')
  baseVersionLabel.value = ''
  reviewerIds.value = (draft.reviewers || []).map((reviewer) => reviewer.user_id)
  linkedIssueIds.value = (draft.linked_issues || []).map((issue) => issue.id)
  for (const issue of draft.linked_issues || []) {
    if (!openIssues.value.some((item) => item.id === issue.id)) openIssues.value.push(issue)
  }
  selectedChunkIds.value = []
  for (const item of draft.items || []) {
    selectedChunkIds.value.push(item.chunk_id)
    suggestions[item.chunk_id] = item.suggested_content
    anchorTexts[item.chunk_id] = item.anchor_text || ''
    editorMode[item.chunk_id] = 'edit'
  }
  draftNote.value = draft.status === 'draft'
    ? t('collaboration.draftRestored')
    : t('collaboration.editingChangesRequested')
}

function payload(asDraft = false) {
  const items = asDraft
    ? selectedChunks.value.map((chunk) => ({
        chunk_id: chunk.id,
        suggested_content: suggestionValue(chunk),
        anchor_text: anchorTexts[chunk.id] || '',
      }))
    : changedItems.value
  return {
    knowledge_id: docId.value,
    title: title.value.trim(),
    description: description.value.trim(),
    items,
    reviewer_ids: reviewerIds.value,
    tags: parseTags(),
    linked_issue_ids: linkedIssueIds.value,
    base_version_id: '',
    as_draft: asDraft,
  }
}

async function ensureDraft() {
  if (draftId.value) return draftId.value
  const result: any = await createDocChangeRequest(kbId.value, { ...payload(true), as_draft: true })
  if (result?.success === false) throw new Error(result?.message || t('collaboration.saveDraftFailed'))
  const draft = result?.data
  if (draft?.id) draftId.value = draft.id
  draftNote.value = t('collaboration.draftSavedAt', { time: new Date().toLocaleTimeString() })
  return draftId.value
}

async function saveDraft(showToast = false) {
  if (submitting.value) return
  if (isEditingChanges.value) {
    if (showToast) MessagePlugin.info(t('collaboration.resubmitNoDraft'))
    return
  }
  try {
    draftNote.value = t('collaboration.savingDraft')
    const id = await ensureDraft()
    if (id) {
      await updateDocCRDraft(kbId.value, id, payload(true))
      draftNote.value = t('collaboration.draftSavedAt', { time: new Date().toLocaleTimeString() })
    }
    if (showToast) MessagePlugin.success(t('collaboration.draftSaved'))
  } catch (error: any) {
    draftNote.value = t('collaboration.saveDraftFailed')
    if (showToast) MessagePlugin.error(error?.message || t('collaboration.saveDraftFailed'))
  }
}

function scheduleAutosave() {
  if (hydrating.value || isEditingChanges.value) return
  const generation = ++autosaveGeneration
  draftNote.value = t('collaboration.savingDraft')
  if (autosaveTimer) clearTimeout(autosaveTimer)
  autosaveTimer = setTimeout(() => {
    if (generation !== autosaveGeneration) return
    void saveDraft(false)
  }, 1200)
}

watch(
  () => JSON.stringify({
    title: title.value,
    description: description.value,
    tags: tagsText.value,
    reviewers: reviewerIds.value,
    issues: linkedIssueIds.value,
    chunks: selectedChunkIds.value,
    suggestions: selectedChunkIds.value.map((id) => suggestions[id] ?? ''),
  }),
  () => scheduleAutosave(),
)

async function submitCR() {
  if (!canSubmit.value || submitting.value) return
  submitting.value = true
  try {
    const body = { ...payload(false), as_draft: false }
    let result: any
    if (isEditingChanges.value) {
      result = await resubmitDocChangeRequest(kbId.value, editingCRId.value, body)
    } else if (draftId.value) {
      result = await submitDocCRDraft(kbId.value, draftId.value, body)
    } else {
      result = await createDocChangeRequest(kbId.value, body)
    }
    if (result?.success === false) {
      MessagePlugin.error(result?.message || t('collaboration.createCRFailed'))
      return
    }
    const cr = result?.data
    MessagePlugin.success(t('collaboration.createCRSuccess'))
    if (cr?.id) {
      router.replace({
        path: `/platform/knowledge-bases/${kbId.value}/doc/${docId.value}/cr/${cr.id}`,
      })
    } else {
      backToDoc()
    }
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('collaboration.createCRFailed'))
  } finally {
    submitting.value = false
  }
}

function backToDoc() {
  router.push({
    path: `/platform/knowledge-bases/${kbId.value}`,
    query: {
      tab: 'documents',
      knowledge_id: docId.value,
      doc_tab: 'collab',
    },
  })
}

onMounted(loadPage)
</script>

<style scoped lang="less">
@import '@/styles/collaboration.css';

.workbench-page {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  border-radius: 12px;
  overflow: hidden;
}

.chunk-list,
.workbench-center,
.workbench-right {
  scrollbar-width: none;
  -ms-overflow-style: none;
}

.chunk-list::-webkit-scrollbar,
.workbench-center::-webkit-scrollbar,
.workbench-right::-webkit-scrollbar {
  width: 0;
  height: 0;
  display: none;
}

.editor-surface {
  scrollbar-width: none;
  scrollbar-color: #cbd5e1 transparent;
}

.editor-textarea {
  scrollbar-width: thin;
  scrollbar-color: #cbd5e1 transparent;
}

.editor-surface::-webkit-scrollbar {
  width: 0;
  height: 0;
  display: none;
}

.editor-textarea::-webkit-scrollbar {
  width: 8px;
  height: 0;
}

.editor-textarea::-webkit-scrollbar:horizontal {
  display: none;
}

.editor-textarea::-webkit-scrollbar-corner {
  display: none;
}

.editor-surface::-webkit-scrollbar-track,
.editor-textarea::-webkit-scrollbar-track {
  background: #f6f8fa;
}

.editor-surface::-webkit-scrollbar-thumb,
.editor-textarea::-webkit-scrollbar-thumb {
  border-radius: 4px;
  background: #cbd5e1;
}

.editor-surface::-webkit-scrollbar-thumb:hover,
.editor-textarea::-webkit-scrollbar-thumb:hover {
  background: #9ca3af;
}

.workbench-cols {
  display: flex;
  flex: 1 1 0;
  min-height: 0;
  height: 0;
  overflow: hidden;
}

.panel-toggles {
  display: flex;
  align-items: center;
  gap: 2px;
}

.panel-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--cv-sub);
  cursor: pointer;
  transition: color 0.15s ease, background 0.15s ease;
}

.panel-toggle:hover {
  color: var(--cv-brand-strong);
  background: var(--cv-brand-bg);
}

.panel-toggle.is-collapsed {
  color: var(--cv-brand-strong);
  background: var(--cv-brand-bg);
}

.panel-toggle svg {
  pointer-events: none;
}

.workbench-left {
  flex: none;
  width: 230px;
  min-width: 0;
  border-right: 1px solid var(--cv-line);
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
  opacity: 1;
  transform: translateX(0);
  transition:
    width 260ms cubic-bezier(0.22, 0.61, 0.36, 1),
    border-color 200ms ease,
    opacity 180ms ease,
    transform 260ms cubic-bezier(0.22, 0.61, 0.36, 1);
}

.workbench-left.is-collapsed {
  width: 0;
  border-right-color: transparent;
  opacity: 0;
  transform: translateX(-12px);
  pointer-events: none;
}

.left-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px 8px;
  font-size: 12px;
  color: var(--cv-sub);
}

.left-head b {
  color: var(--cv-ink);
}

.chunk-search {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 2px 12px 8px;
  padding: 5px 10px;
  border-radius: 8px;
  background: var(--cv-chip);
  color: var(--cv-faint);
  font-size: 12px;
}

.chunk-search input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: none;
  background: transparent;
  font: inherit;
  color: var(--cv-ink);
}

.chunk-list {
  flex: 1;
  overflow: auto;
  min-height: 0;
}

.chunk-option {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  width: 100%;
  padding: 8px 14px;
  border: 0;
  border-top: 1px solid var(--cv-line);
  background: #fff;
  text-align: left;
  font: inherit;
  cursor: pointer;
}

.chunk-option:hover {
  background: var(--cv-bg);
}

.chunk-option.is-selected {
  background: var(--cv-brand-bg);
}

.chunk-check {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 15px;
  height: 15px;
  margin-top: 2px;
  border: 1.5px solid #cbd5e1;
  border-radius: 4px;
  color: #fff;
}

.chunk-check.on {
  background: var(--cv-brand);
  border-color: var(--cv-brand);
}

.chunk-option-body {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
}

.chunk-option-title {
  display: flex;
  align-items: center;
  gap: 5px;
  color: var(--cv-ink);
  font-size: 12px;
  font-weight: 650;
}

.chunk-tag {
  border-radius: 4px;
  padding: 0 5px;
  font-size: 10px;
  font-weight: 500;
}

.chunk-tag.is-edit {
  background: var(--cv-brand-bg);
  color: var(--cv-brand-strong);
}

.chunk-tag.is-lock {
  background: var(--cv-amber-bg);
  color: var(--cv-amber);
}

.chunk-tag.is-cite {
  background: var(--cv-purple-bg);
  color: var(--cv-purple);
}

.lock-banner {
  margin: 8px 14px 0;
  border-radius: 7px;
  padding: 7px 10px;
  background: var(--cv-amber-bg);
  color: var(--cv-amber);
  font-size: 11px;
}

.chunk-option-excerpt {
  color: var(--cv-faint);
  font-size: 11px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.left-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 14px;
  border-top: 1px solid var(--cv-line);
  color: var(--cv-sub);
  font-size: 12px;
}

.workbench-center {
  flex: 1;
  min-width: 0;
  min-height: 0;
  height: 100%;
  overflow-y: auto;
  padding: 14px 16px;
  background: var(--cv-bg);
}

.padded {
  padding-top: 40px;
}

.chunk-editor {
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

.chunk-stats {
  margin-left: auto;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.editor-split {
  position: relative;
  display: grid;
  grid-template-columns: 1fr 1fr;
  padding-bottom: 22px;
  border-top: 1px solid var(--cv-line);
  background: #fff;
}

.editor-pane {
  min-width: 0;
  border-right: 1px solid var(--cv-line);
  background: #fff;
}

.editor-pane:last-of-type {
  border-right: 0;
}

.editor-pane-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  height: 30px;
  padding: 0 12px;
  border-bottom: 1px solid var(--cv-line);
  background: #f6f8fa;
  color: #4b5563;
  font-size: 11px;
  transition: color 0.15s ease, border-color 0.15s ease;
}

.editor-pane-head > span:first-child {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.editor-pane:focus-within .editor-pane-head {
  color: var(--cv-brand-strong);
  border-bottom-color: var(--cv-brand-border);
}

.editor-pane-badge {
  border-radius: 99px;
  padding: 0 7px;
  background: #eef1f4;
  color: #6b7280;
  font-size: 10px;
}

.editor-pane-badge.is-editable {
  background: var(--cv-brand-bg);
  color: var(--cv-brand-strong);
}

.editor-surface {
  display: flex;
  align-items: flex-start;
  height: 280px;
  overflow: auto;
  background: #fff;
  scrollbar-gutter: stable;
}

.editor-surface > pre.editor-code {
  flex: 0 0 auto;
  min-width: 100%;
  min-height: 100%;
}

.editor-surface.is-editing {
  overflow: hidden;
}

.editor-surface.is-editing .editor-gutter {
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

.editor-gutter {
  position: sticky;
  left: 0;
  z-index: 1;
  flex: none;
  width: 42px;
  padding: 8px 8px 8px 0;
  min-height: 100%;
  overflow: hidden;
  text-align: right;
  user-select: none;
  color: #9ca3af;
  background: #f6f8fa;
  font: 12px/20px 'Cascadia Code', 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
}

.editor-gutter span {
  display: block;
  height: 20px;
}

.editor-code {
  flex: 1;
  min-width: 0;
  margin: 0;
  padding: 8px 14px;
  color: #1f2937;
  background: #fff;
  font: 12px/20px 'Cascadia Code', 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
  white-space: pre;
  word-break: normal;
  tab-size: 2;
}

.editor-textarea {
  height: 100%;
  resize: none;
  border: 0;
  outline: 0;
  overflow: auto;
  scrollbar-gutter: stable;
}

.editor-textarea:focus {
  box-shadow: inset 0 0 0 1px var(--cv-brand-border);
}

.editor-status {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  height: 22px;
  padding: 0 12px;
  border-top: 1px solid var(--cv-line);
  background: #f6f8fa;
  color: #6b7280;
  font-size: 10px;
}

.workbench-right {
  flex: none;
  width: 286px;
  min-width: 0;
  min-height: 0;
  height: 100%;
  border-left: 1px solid var(--cv-line);
  overflow-y: auto;
  background: #fff;
  opacity: 1;
  transform: translateX(0);
  transition:
    width 260ms cubic-bezier(0.22, 0.61, 0.36, 1),
    border-color 200ms ease,
    opacity 180ms ease,
    transform 260ms cubic-bezier(0.22, 0.61, 0.36, 1);
}

.workbench-right.is-collapsed {
  width: 0;
  border-left-color: transparent;
  opacity: 0;
  transform: translateX(12px);
  pointer-events: none;
  overflow: hidden;
}

.form-section {
  padding: 12px 16px;
  border-top: 1px solid var(--cv-line);
}

.form-section:first-child {
  border-top: 0;
}

.form-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 6px;
  color: var(--cv-sub);
  font-size: 12px;
}

.form-label b {
  color: var(--cv-ink);
}

.small {
  font-size: 11px;
}

.linked-issues {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}

.linked-issue-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  border-radius: 6px;
  padding: 1px 8px;
  background: var(--cv-chip);
  color: var(--cv-sub);
  font-size: 11px;
}

.chip-remove {
  border: 0;
  background: none;
  color: var(--cv-faint);
  cursor: pointer;
  padding: 0;
  font-size: 13px;
}

.reviewer-row {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.picker-panel {
  margin-top: 8px;
  border: 1px solid var(--cv-line);
  border-radius: 8px;
  max-height: 180px;
  overflow: auto;
  padding: 4px 8px;
  background: #fff;
}

.picker-row {
  display: flex;
  align-items: flex-start;
  gap: 7px;
  padding: 5px 0;
  color: var(--cv-sub);
  font-size: 12px;
  cursor: pointer;
}

.picker-row input {
  margin-top: 2px;
  accent-color: var(--cv-brand);
}

.check-row {
  display: flex;
  align-items: flex-start;
  gap: 7px;
  padding: 4px 0;
  color: var(--cv-sub);
  font-size: 12px;
  cursor: pointer;
}

.check-row input {
  margin-top: 2px;
  accent-color: var(--cv-brand);
}

.impact-box {
  border: 1px dashed var(--cv-line);
  border-radius: 9px;
  padding: 9px 12px;
  background: var(--cv-bg);
  color: var(--cv-sub);
  font-size: 11.5px;
  line-height: 1.8;
}

.form-hint {
  margin-top: 5px;
  text-align: center;
  color: var(--cv-amber);
  font-size: 11px;
}

.secondary-block {
  margin-top: 8px;
}
</style>
