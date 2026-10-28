<template>
  <div class="collab-v2 collab-workbench">
    <div class="collab-workbench-toolbar">
      <div class="ftabs">
        <button
          v-for="tab in statusTabs"
          :key="tab.value"
          type="button"
          class="ftab"
          :class="{ on: statusFilter === tab.value }"
          @click="statusFilter = tab.value"
        >
          {{ tab.label }}<span v-if="counts[tab.key] != null" class="ftab-count">{{ counts[tab.key] }}</span>
        </button>
      </div>
      <div class="collab-workbench-actions">
        <label class="mine-toggle">
          <input v-model="mineOnly" type="checkbox" />
          <span>{{ t('collaboration.mine') }}</span>
        </label>
        <select v-model="typeFilter" class="type-select" :aria-label="t('collaboration.typeAll')">
          <option value="all">{{ t('collaboration.typeAll') }}</option>
          <option value="issue">{{ t('collaboration.typeIssue') }}</option>
          <option value="cr">{{ t('collaboration.typeCR') }}</option>
        </select>
        <template v-if="knowledgeId">
          <button type="button" class="cv-btn" @click="createIssueVisible = true">
            <t-icon name="chat-bubble-help" size="14px" />
            {{ t('collaboration.createIssue') }}
          </button>
          <button v-if="canEdit" type="button" class="cv-btn cv-btn-primary" @click="openNewCR">
            <t-icon name="git-branch" size="14px" />
            {{ t('collaboration.createCR') }}
          </button>
        </template>
      </div>
    </div>

    <div v-if="loading" class="cv-empty">{{ t('common.loading') }}</div>
    <div v-else-if="loadError" class="cv-empty">
      {{ loadError }}
      <button type="button" class="cv-link" @click="loadList">{{ t('common.retry') }}</button>
    </div>
    <div v-else-if="!items.length" class="empty-card">
      <p class="empty-title">{{ t('collaboration.empty') }}</p>
      <p class="empty-hint">{{ t('collaboration.emptyHint') }}</p>
    </div>
    <div v-else class="rows">
      <button
        v-for="item in items"
        :key="item.type + '-' + item.id"
        type="button"
        class="row"
        @click="openItem(item)"
      >
        <span class="cv-type-icon" :class="item.type === 'cr' ? 'is-cr' : 'is-issue'">
          <t-icon :name="item.type === 'cr' ? 'git-branch' : 'chat-bubble-help'" size="14px" />
        </span>
        <span class="row-main">
          <span class="row-title-line">
            <span class="row-seq">#{{ item.seq_id }}</span>
            <span class="row-title">{{ item.title || t('collaboration.untitled') }}</span>
            <span
              v-if="item.type === 'cr'"
              class="cv-badge"
              :class="badgeClass(item)"
            >{{ badgeLabel(item) }}</span>
            <span v-else class="cv-badge" :class="item.status === 'closed' ? 'is-closed' : 'is-open'">
              {{ item.status === 'closed' ? t('collaboration.issueClosed') : t('collaboration.issueOpen') }}
            </span>
            <span v-if="item.type === 'issue' && item.anchor_orphaned" class="cv-badge is-action">{{ t('collaboration.chunkOrphaned') }}</span>
          </span>
          <span class="row-meta">
            <span>{{ item.type === 'cr' ? t('collaboration.typeCR') : t('collaboration.typeIssue') }}</span>
            <span v-if="!knowledgeId && item.knowledge_title" class="row-doc">
              <t-icon name="file" size="12px" />
              {{ item.knowledge_title }}
            </span>
            <span>{{ item.created_by_name || item.created_by }}</span>
            <span v-if="item.type === 'cr' && item.item_count">
              {{ t('collaboration.crChangedChunks', { count: item.item_count }) }}
            </span>
            <span v-if="item.comments_count">
              <t-icon name="chat" size="12px" />
              {{ item.comments_count }}
            </span>
          </span>
        </span>
        <span class="row-when">{{ relativeTime(item.updated_at || item.created_at) }}</span>
      </button>
    </div>

    <div v-if="total > pageSize" class="pagination">
      <t-pagination
        v-model="page"
        :total="total"
        :page-size="pageSize"
        :show-page-size="false"
        size="small"
        @current-change="loadList"
      />
    </div>

    <IssueCreateDialog
      v-model:visible="createIssueVisible"
      :kb-id="kbId"
      :knowledge-id="knowledgeId"
      :knowledge-title="knowledgeTitle"
      @created="onCreated"
    />
    <CollaborationDetailDrawer
      v-model:visible="issueDrawerVisible"
      :kb-id="kbId"
      type="issue"
      :item-id="issueDrawerId"
      :can-manage="canManage"
      :can-edit="canEdit"
      @changed="loadList"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { listDocCollaborations, type CollaborationItem, type DocCRStatus } from '@/api/collaboration'
import { useAuthStore } from '@/stores/auth'
import IssueCreateDialog from './IssueCreateDialog.vue'
import CollaborationDetailDrawer from './CollaborationDetailDrawer.vue'
import { CR_STATUS_BADGE, CR_STATUS_LABELS, relativeTime } from './status'

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()

const props = withDefaults(defineProps<{
  kbId: string
  knowledgeId?: string
  knowledgeTitle?: string
  canEdit?: boolean
  canManage?: boolean
}>(), {
  knowledgeId: '',
  knowledgeTitle: '',
  canEdit: false,
  canManage: false,
})

const typeFilter = ref<'all' | 'issue' | 'cr'>('all')
const statusFilter = ref('')
const mineOnly = ref(false)
const items = ref<CollaborationItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(false)
const loadError = ref('')
const counts = ref<Record<string, number | null>>({})
const createIssueVisible = ref(false)
const issueDrawerVisible = ref(false)
const issueDrawerId = ref('')

const statusTabs = computed(() => [
  { value: '', key: 'all', label: t('collaboration.allRecords') },
  { value: 'draft', key: 'draft', label: CR_STATUS_LABELS.draft },
  { value: 'pending_review', key: 'pending_review', label: CR_STATUS_LABELS.pending_review },
  { value: 'changes_requested', key: 'changes_requested', label: CR_STATUS_LABELS.changes_requested },
  { value: 'merged', key: 'merged', label: CR_STATUS_LABELS.merged },
  { value: 'closed', key: 'closed', label: CR_STATUS_LABELS.closed },
])

const currentUserId = computed(() => authStore.currentUserId || '')

function badgeClass(item: CollaborationItem) {
  return CR_STATUS_BADGE[item.status as DocCRStatus] || 'is-closed'
}

function badgeLabel(item: CollaborationItem) {
  const status = item.status as DocCRStatus
  if (status === 'changes_requested' && item.created_by === currentUserId.value) {
    return t('collaboration.needUpdate')
  }
  return CR_STATUS_LABELS[status] || item.status
}

let loadGeneration = 0

const query = (status: string, pageNo: number, size: number) => ({
  knowledge_id: props.knowledgeId || undefined,
  type: typeFilter.value === 'all' ? undefined : typeFilter.value,
  status: status || undefined,
  created_by: mineOnly.value ? currentUserId.value || undefined : undefined,
  page: pageNo,
  page_size: size,
})

const loadList = async () => {
  if (!props.kbId) return
  const generation = ++loadGeneration
  loading.value = true
  loadError.value = ''
  try {
    const result: any = await listDocCollaborations(props.kbId, query(statusFilter.value, page.value, pageSize))
    if (generation !== loadGeneration) return
    if (result?.success === false || !result?.data) {
      loadError.value = result?.message || t('collaboration.loadFailed')
      items.value = []
      total.value = 0
      return
    }
    items.value = result.data.items || []
    total.value = result.data.total || 0
  } catch (error: any) {
    if (generation !== loadGeneration) return
    loadError.value = error?.message || t('collaboration.loadFailed')
    items.value = []
    total.value = 0
  } finally {
    if (generation === loadGeneration) loading.value = false
  }
}

const loadCounts = async () => {
  if (!props.kbId) return
  const statuses = ['', 'draft', 'pending_review', 'changes_requested', 'merged', 'closed']
  const entries = await Promise.all(
    statuses.map(async (status) => {
      try {
        const result: any = await listDocCollaborations(props.kbId, query(status, 1, 1))
        return [status || 'all', result?.data?.total ?? null] as const
      } catch {
        return [status || 'all', null] as const
      }
    }),
  )
  counts.value = Object.fromEntries(entries)
}

watch([typeFilter, mineOnly, () => props.knowledgeId], () => {
  page.value = 1
  void loadCounts()
  void loadList()
}, { immediate: true })

watch(statusFilter, () => {
  page.value = 1
  void loadList()
})

function openNewCR() {
  if (!props.knowledgeId) return
  router.push({
    path: `/platform/knowledge-bases/${props.kbId}/doc/${props.knowledgeId}/cr/new`,
  })
}

function openItem(item: CollaborationItem) {
  if (item.type === 'cr') {
    router.push({
      path: `/platform/knowledge-bases/${props.kbId}/doc/${item.knowledge_id}/cr/${item.id}`,
    })
    return
  }
  issueDrawerId.value = item.id
  issueDrawerVisible.value = true
}

function onCreated() {
  void loadCounts()
  void loadList()
}

defineExpose({ reload: () => { void loadCounts(); void loadList() } })
</script>

<style scoped lang="less">
@import '@/styles/collaboration.css';

.collab-workbench {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 0;
}

.collab-workbench-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.ftabs {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}

.ftab {
  border: 0;
  background: transparent;
  border-radius: 99px;
  padding: 3px 12px;
  color: var(--cv-sub);
  font: inherit;
  font-size: 12px;
  cursor: pointer;
}

.ftab:hover {
  color: var(--cv-ink);
  background: var(--cv-bg);
}

.ftab.on {
  background: var(--cv-ink);
  color: #fff;
  font-weight: 650;
}

.ftab-count {
  margin-left: 5px;
  opacity: 0.7;
  font-variant-numeric: tabular-nums;
}

.collab-workbench-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.mine-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--cv-sub);
  font-size: 12px;
  user-select: none;
  cursor: pointer;
}

.mine-toggle input {
  accent-color: var(--cv-brand);
}

.type-select {
  border: 1px solid var(--cv-line);
  border-radius: 8px;
  padding: 4px 8px;
  background: #fff;
  color: var(--cv-sub);
  font: inherit;
  font-size: 12px;
  outline: none;
}

.rows {
  border: 1px solid var(--cv-line);
  border-radius: 10px;
  overflow: hidden;
}

.row {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  width: 100%;
  padding: 11px 14px;
  border: 0;
  border-top: 1px solid var(--cv-line);
  background: #fff;
  text-align: left;
  font: inherit;
  cursor: pointer;
}

.row:first-child {
  border-top: 0;
}

.row:hover {
  background: var(--cv-bg);
}

.row-main {
  display: flex;
  flex-direction: column;
  gap: 3px;
  flex: 1;
  min-width: 0;
}

.row-title-line {
  display: flex;
  align-items: center;
  gap: 7px;
  min-width: 0;
}

.row-seq {
  color: var(--cv-faint);
  font-size: 11px;
  flex: none;
}

.row-title {
  color: var(--cv-ink);
  font-size: 13px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.row-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  color: var(--cv-faint);
  font-size: 11px;
}

.row-meta span {
  display: inline-flex;
  align-items: center;
  gap: 3px;
}

.row-doc {
  max-width: 240px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.row-when {
  color: var(--cv-faint);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
  flex: none;
}

.empty-card {
  padding: 30px 16px;
  border: 1px solid var(--cv-line);
  border-radius: 10px;
  text-align: center;
}

.empty-title {
  margin: 0;
  color: var(--cv-ink);
  font-weight: 600;
}

.empty-hint {
  margin: 4px 0 0;
  color: var(--cv-faint);
  font-size: 12px;
}

.pagination {
  display: flex;
  justify-content: flex-end;
}
</style>
