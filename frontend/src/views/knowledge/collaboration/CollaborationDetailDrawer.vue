<template>
  <t-drawer :visible="visible" :size="isCompact ? '100%' : '760px'" :z-index="2350" attach="body" :footer="false"
    :close-btn="true" class="collab-detail-drawer" @close="handleClose">
    <template #header>
      <div class="collab-detail-heading">
        <div class="collab-detail-kicker">
          <t-icon :name="type === 'cr' ? 'git-branch' : 'chat-bubble-help'" size="16px" />
          <span>{{ type === 'cr' ? $t('collaboration.typeCR') : $t('collaboration.typeIssue') }}</span>
          <span v-if="record?.seq_id" class="collab-detail-seq">#{{ record.seq_id }}</span>
        </div>
        <div class="collab-detail-title">{{ record?.title || $t('collaboration.detail') }}</div>
        <div v-if="record" class="collab-detail-tags">
          <t-tag size="small" variant="light" :theme="statusTheme(record.status)">{{ statusLabel(record.status) }}</t-tag>
          <t-tag v-if="record.anchor_orphaned" size="small" variant="light" theme="warning">
            {{ $t('collaboration.chunkOrphaned') }}
          </t-tag>
          <span class="collab-detail-doc" :title="record.knowledge_title || ''">
            <t-icon name="file" size="13px" />
            {{ record.knowledge_title || record.knowledge_id }}
          </span>
        </div>
      </div>
    </template>

    <div class="collab-detail-body">
      <div v-if="loading" class="collab-detail-state">
        <t-loading size="small" />
        <span>{{ $t('common.loading') }}</span>
      </div>
      <div v-else-if="loadError" class="collab-detail-state collab-detail-state--error">
        <t-icon name="error-circle" size="20px" />
        <span>{{ loadError }}</span>
        <t-button variant="text" size="small" @click="loadDetail">{{ $t('common.retry') }}</t-button>
      </div>

      <template v-else-if="record">
        <!-- Issue -->
        <template v-if="type === 'issue'">
          <section class="collab-detail-section">
            <div class="collab-meta-grid">
              <div class="collab-meta-item">
                <span class="collab-meta-label">{{ $t('collaboration.createdBy', { name: '' }).replace(/[：:\\s]*$/, '') }}</span>
                <span class="collab-meta-value">{{ userName(issue?.created_by_name, issue?.created_by) }}</span>
              </div>
              <div class="collab-meta-item">
                <span class="collab-meta-label">{{ $t('collaboration.createdAt') }}</span>
                <span class="collab-meta-value">{{ formatTime(issue?.created_at) }}</span>
              </div>
              <div class="collab-meta-item">
                <span class="collab-meta-label">{{ $t('collaboration.chunk') }}</span>
                <span class="collab-meta-value">
                  <template v-if="issue?.chunk_id && !issue?.anchor_orphaned">
                    {{ $t('collaboration.chunkShort', { index: (issue.chunk_index || 0) + 1 }) }}
                  </template>
                  <template v-else>{{ $t('collaboration.documentLevel') }}</template>
                </span>
              </div>
              <div v-if="issue?.closed_at" class="collab-meta-item">
                <span class="collab-meta-label">{{ $t('collaboration.statusClosed') }}</span>
                <span class="collab-meta-value">{{ formatTime(issue.closed_at) }}</span>
              </div>
            </div>

            <div v-if="issue?.chunk_excerpt" class="collab-context-card">
              <div class="collab-context-head">
                <t-icon name="file" size="14px" />
                <span>{{ $t('collaboration.fromChunk', { index: (issue.chunk_index || 0) + 1 }) }}</span>
              </div>
              <p class="collab-context-excerpt">{{ issue.chunk_excerpt }}</p>
            </div>

            <div v-if="issue?.anchor_orphaned" class="collab-banner collab-banner--warning">
              <t-icon name="info-circle" size="14px" />
              <span>{{ $t('collaboration.orphanedHint') }}</span>
            </div>

            <div class="collab-body-text">{{ issue?.body || $t('common.noData') }}</div>

            <div v-if="canManage" class="collab-detail-actions">
              <t-button v-if="issue?.status === 'open'" size="small" variant="outline" @click="toggleIssueStatus('closed')">
                <template #icon><t-icon name="check-circle" size="15px" /></template>
                {{ $t('collaboration.closeIssue') }}
              </t-button>
              <t-button v-else size="small" variant="outline" @click="toggleIssueStatus('open')">
                <template #icon><t-icon name="rollback" size="15px" /></template>
                {{ $t('collaboration.reopenIssue') }}
              </t-button>
              <t-popconfirm theme="warning" :content="$t('collaboration.deleteIssueConfirm')" @confirm="removeIssue">
                <t-button size="small" variant="outline" theme="danger">
                  <template #icon><t-icon name="delete" size="15px" /></template>
                  {{ $t('collaboration.deleteIssue') }}
                </t-button>
              </t-popconfirm>
            </div>
          </section>
        </template>

        <!-- Change request -->
        <template v-else>
          <section class="collab-detail-section">
            <div class="collab-meta-grid">
              <div class="collab-meta-item">
                <span class="collab-meta-label">{{ $t('collaboration.createdBy', { name: '' }).replace(/[：:\\s]*$/, '') }}</span>
                <span class="collab-meta-value">{{ userName(cr?.created_by_name, cr?.created_by) }}</span>
              </div>
              <div class="collab-meta-item">
                <span class="collab-meta-label">{{ $t('collaboration.createdAt') }}</span>
                <span class="collab-meta-value">{{ formatTime(cr?.created_at) }}</span>
              </div>
              <div class="collab-meta-item">
                <span class="collab-meta-label">{{ $t('collaboration.crItems') }}</span>
                <span class="collab-meta-value">{{ cr?.items?.length || 0 }}</span>
              </div>
              <div v-if="cr?.reviewed_at" class="collab-meta-item">
                <span class="collab-meta-label">{{ $t('collaboration.reviewedBy') }}</span>
                <span class="collab-meta-value">
                  {{ userName(cr?.reviewed_by_name, cr?.reviewed_by) }} · {{ formatTime(cr.reviewed_at) }}
                </span>
              </div>
            </div>

            <div v-if="cr?.description" class="collab-body-text collab-body-text--muted">{{ cr.description }}</div>

            <div v-if="cr?.anchor_orphaned" class="collab-banner collab-banner--warning">
              <t-icon name="info-circle" size="14px" />
              <span>{{ $t('collaboration.orphanedHint') }}</span>
            </div>
            <div v-else-if="hasConflict" class="collab-banner collab-banner--danger">
              <t-icon name="error-circle" size="14px" />
              <span>{{ $t('collaboration.conflictHint') }}</span>
            </div>
          </section>

          <section class="collab-detail-section">
            <h4 class="collab-detail-section-title">{{ $t('collaboration.crItems') }}</h4>
            <div class="collab-diff-list">
              <article v-for="item in cr?.items || []" :key="item.id" class="collab-diff-card">
                <header class="collab-diff-head">
                  <div class="collab-diff-head-left">
                    <span class="collab-diff-index">
                      {{ item.chunk_index !== undefined
                        ? $t('collaboration.itemN', { index: item.chunk_index + 1 })
                        : $t('collaboration.chunk') }}
                    </span>
                    <t-tag v-if="item.conflict" size="small" variant="light" theme="danger">
                      {{ $t('collaboration.conflict') }}
                    </t-tag>
                    <t-tag v-if="item.anchor_orphaned" size="small" variant="light" theme="warning">
                      {{ $t('collaboration.chunkOrphaned') }}
                    </t-tag>
                  </div>
                  <div class="collab-diff-legend">
                    <span class="collab-diff-legend-item collab-diff-legend-item--del">
                      <i />{{ $t('collaboration.originalContent') }}
                    </span>
                    <span class="collab-diff-legend-item collab-diff-legend-item--add">
                      <i />{{ $t('collaboration.suggestedContent') }}
                    </span>
                  </div>
                </header>
                <pre v-if="diffRows(item).length" class="collab-diff-body"><span
                  v-for="(line, lineIndex) in diffRows(item)" :key="lineIndex"
                  :class="['collab-diff-line', `collab-diff-line--${line.type}`]">{{ diffPrefix(line.type) }}{{ line.text || ' ' }}
</span></pre>
                <div v-else class="collab-diff-empty">{{ $t('collaboration.noItemContent') }}</div>
                <div v-if="item.conflict && item.current_content !== undefined" class="collab-current-content">
                  <div class="collab-current-content-title">{{ $t('collaboration.currentContent') }}</div>
                  <pre>{{ item.current_content || $t('collaboration.noItemContent') }}</pre>
                </div>
              </article>
            </div>
          </section>

          <section v-if="(cr?.linked_issues || []).length" class="collab-detail-section">
            <h4 class="collab-detail-section-title">{{ $t('collaboration.linkedIssuesSection') }}</h4>
            <div class="collab-linked-issue-list">
              <div v-for="issueRef in cr?.linked_issues || []" :key="issueRef.id" class="collab-linked-issue">
                <t-icon name="chat-bubble-help" size="14px" />
                <span class="collab-linked-issue-seq">#{{ issueRef.seq_id }}</span>
                <span class="collab-linked-issue-title">{{ issueRef.title }}</span>
                <t-tag size="small" variant="light" :theme="statusTheme(issueRef.status)">{{ statusLabel(issueRef.status) }}</t-tag>
              </div>
            </div>
          </section>

          <section v-if="(cr?.reviews || []).length" class="collab-detail-section">
            <h4 class="collab-detail-section-title">{{ $t('collaboration.acceptanceHistory') }}</h4>
            <div class="collab-review-history">
              <div v-for="review in cr?.reviews || []" :key="review.id" class="collab-review-history-item">
                <div class="collab-review-history-head">
                  <t-tag size="small" variant="light" :theme="review.action === 'accept' ? 'success' : 'danger'">
                    {{ review.action === 'accept' ? $t('collaboration.reviewAccept') : $t('collaboration.reviewReject') }}
                  </t-tag>
                  <span class="collab-review-history-user">{{ userName(review.reviewed_by_name, review.reviewed_by) }}</span>
                  <span class="collab-review-history-time">{{ formatTime(review.created_at) }}</span>
                </div>
                <p v-if="review.comment" class="collab-review-history-comment">{{ review.comment }}</p>
              </div>
            </div>
          </section>

          <section v-if="reviewMode" class="collab-detail-section collab-review-form">
            <h4 class="collab-detail-section-title">
              {{ reviewMode === 'approve' ? $t('collaboration.approveCR') : $t('collaboration.requestChanges') }}
            </h4>
            <p class="collab-review-hint">
              {{ reviewMode === 'approve' ? $t('collaboration.approveHint') : $t('collaboration.requestChangesHint') }}
            </p>
            <t-textarea v-model="reviewComment" :placeholder="$t('collaboration.reviewCommentPlaceholder')"
              :autosize="{ minRows: 3, maxRows: 8 }" :disabled="reviewSubmitting" />
            <div class="collab-review-actions">
              <t-button variant="outline" size="small" :disabled="reviewSubmitting" @click="cancelReview">
                {{ $t('common.cancel') }}
              </t-button>
              <t-button size="small" :theme="reviewMode === 'approve' ? 'primary' : 'danger'" :loading="reviewSubmitting"
                @click="submitReview">
                {{ $t('collaboration.reviewConfirm') }}
              </t-button>
            </div>
          </section>

          <section v-if="resubmitMode" class="collab-detail-section collab-resubmit-form">
            <h4 class="collab-detail-section-title">{{ $t('collaboration.resubmit') }}</h4>
            <p class="collab-review-hint">{{ $t('collaboration.resubmitHint') }}</p>
            <div class="collab-form-field">
              <label>{{ $t('collaboration.issueTitle') }}</label>
              <t-input v-model="resubmitTitle" :disabled="resubmitSubmitting" :maxlength="255" />
            </div>
            <div class="collab-form-field">
              <label>{{ $t('collaboration.crDescription') }}</label>
              <t-textarea v-model="resubmitDescription" :autosize="{ minRows: 2, maxRows: 5 }"
                :disabled="resubmitSubmitting" />
            </div>
            <div v-for="item in cr?.items || []" :key="'resubmit-' + item.id" class="collab-form-field">
              <label>
                {{ item.chunk_index !== undefined
                  ? $t('collaboration.itemN', { index: item.chunk_index + 1 })
                  : $t('collaboration.chunk') }} · {{ $t('collaboration.suggestedContent') }}
              </label>
              <t-textarea v-model="resubmitSuggestions[item.chunk_id]" :autosize="{ minRows: 5, maxRows: 14 }"
                :disabled="resubmitSubmitting" />
            </div>
            <div class="collab-review-actions">
              <t-button variant="outline" size="small" :disabled="resubmitSubmitting" @click="cancelResubmit">
                {{ $t('common.cancel') }}
              </t-button>
              <t-button theme="primary" size="small" :loading="resubmitSubmitting" @click="submitResubmit">
                {{ $t('collaboration.resubmit') }}
              </t-button>
            </div>
          </section>

          <div v-if="!reviewMode && !resubmitMode" class="collab-detail-actions">
            <template v-if="canManage && cr?.status === 'pending_review'">
              <t-button size="small" theme="primary" :disabled="!canAccept" @click="startReview('approve')">
                <template #icon><t-icon name="check" size="15px" /></template>
                {{ $t('collaboration.approve') }}
              </t-button>
              <t-button size="small" variant="outline" theme="danger" @click="startReview('request_changes')">
                {{ $t('collaboration.requestChanges') }}
              </t-button>
            </template>
            <t-button v-if="canResubmit" size="small" variant="outline" @click="startResubmit">
              <template #icon><t-icon name="refresh" size="15px" /></template>
              {{ $t('collaboration.resubmit') }}
            </t-button>
          </div>
        </template>

        <section class="collab-detail-section collab-comments-section">
          <div class="collab-comments-head">
            <h4 class="collab-detail-section-title">
              {{ $t('collaboration.comments', { count: comments.length }) }}
            </h4>
          </div>
          <div v-if="comments.length" class="collab-comment-list">
            <div v-for="comment in comments" :key="comment.id" class="collab-comment"
              :class="{ 'is-deleted': comment.deleted }">
              <template v-if="comment.deleted">
                <div class="collab-comment-deleted">
                  <t-icon name="chat-bubble" size="14px" />
                  <span>{{ $t('collaboration.commentDeleted') }}</span>
                </div>
              </template>
              <template v-else>
                <div class="collab-comment-head">
                  <span class="collab-comment-avatar"><t-icon name="user-circle" size="16px" /></span>
                  <span class="collab-comment-user">{{ userName(comment.created_by_name, comment.created_by) }}</span>
                  <span class="collab-comment-time">{{ formatTime(comment.created_at) }}</span>
                  <t-popconfirm v-if="canDeleteComment(comment)" theme="warning"
                    :content="$t('collaboration.deleteCommentConfirm')" @confirm="removeComment(comment)">
                    <t-button class="collab-comment-delete" size="small" variant="text" shape="square"
                      :title="$t('collaboration.deleteComment')">
                      <template #icon><t-icon name="delete" size="14px" /></template>
                    </t-button>
                  </t-popconfirm>
                </div>
                <div class="collab-comment-body">{{ comment.body }}</div>
              </template>
            </div>
          </div>
          <div v-else class="collab-empty-hint">{{ $t('collaboration.noComments') }}</div>

          <div class="collab-comment-composer">
            <t-textarea v-model="commentDraft" :placeholder="$t('collaboration.commentPlaceholder')"
              :autosize="{ minRows: 3, maxRows: 8 }" :disabled="commentSubmitting" />
            <div class="collab-comment-composer-actions">
              <t-button size="small" theme="primary" :loading="commentSubmitting"
                :disabled="!commentDraft.trim()" @click="submitComment">
                {{ $t('collaboration.commentSubmit') }}
              </t-button>
            </div>
          </div>
        </section>
      </template>
    </div>
  </t-drawer>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import {
  createCRComment,
  createIssueComment,
  deleteDocComment,
  deleteDocIssue,
  getDocChangeRequest,
  getDocIssue,
  listCRComments,
  listIssueComments,
  resubmitDocChangeRequest,
  reviewDocChangeRequest,
  updateDocIssue,
  type DocCRItem,
  type DocComment,
  type DocChangeRequest,
  type DocIssue,
} from '@/api/collaboration'
import { useAuthStore } from '@/stores/auth'
import { diffGazeLines, type GazeDiffLine } from '@/utils/gazeLineDiff'
import { CR_STATUS_LABELS } from './status'

const { t } = useI18n()
const authStore = useAuthStore()

const props = withDefaults(defineProps<{
  visible: boolean
  kbId: string
  type: 'issue' | 'cr' | ''
  itemId: string
  canManage?: boolean
  canEdit?: boolean
}>(), {
  canManage: false,
  canEdit: false,
})

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'changed'): void
}>()

const isCompact = computed(() => typeof window !== 'undefined' && window.innerWidth < 900)
const loading = ref(false)
const loadError = ref('')
const issue = ref<DocIssue | null>(null)
const cr = ref<DocChangeRequest | null>(null)
const comments = ref<DocComment[]>([])
const commentDraft = ref('')
const commentSubmitting = ref(false)

const reviewMode = ref<'' | 'approve' | 'request_changes'>('')
const reviewComment = ref('')
const closeIssueIds = ref<string[]>([])
const reviewSubmitting = ref(false)

const resubmitMode = ref(false)
const resubmitTitle = ref('')
const resubmitDescription = ref('')
const resubmitSuggestions = ref<Record<string, string>>({})
const resubmitSubmitting = ref(false)

let loadGeneration = 0

const record = computed(() => (props.type === 'cr' ? cr.value : issue.value))
const hasConflict = computed(() => (cr.value?.items || []).some((item) => item.conflict))
const canAccept = computed(() => !cr.value?.anchor_orphaned && !hasConflict.value)
const canResubmit = computed(() =>
  cr.value?.status === 'changes_requested'
  && !!authStore.currentUserId
  && cr.value?.created_by === authStore.currentUserId
  && props.canEdit,
)

const statusLabel = (status?: string) => {
  switch (status) {
    case 'open': return t('collaboration.statusOpen')
    case 'closed': return t('collaboration.statusClosed')
    case 'draft': return CR_STATUS_LABELS.draft
    case 'pending_review': return CR_STATUS_LABELS.pending_review
    case 'changes_requested': return CR_STATUS_LABELS.changes_requested
    case 'merged': return CR_STATUS_LABELS.merged
    default: return status || ''
  }
}

const statusTheme = (status?: string) => {
  switch (status) {
    case 'open':
    case 'pending_review':
    case 'merged': return 'success'
    case 'changes_requested': return 'warning'
    case 'draft': return 'default'
    case 'closed': return 'default'
    default: return 'default'
  }
}

const userName = (name?: string, id?: string) => name || id || t('collaboration.unknownUser')

const formatTime = (value?: string | null) => {
  if (!value) return '—'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

const diffRows = (item: DocCRItem): GazeDiffLine[] =>
  diffGazeLines(item.original_content || '', item.suggested_content || '')

const diffPrefix = (type: GazeDiffLine['type']) => {
  if (type === 'add') return '+ '
  if (type === 'del') return '- '
  return '  '
}

const loadComments = async () => {
  if (!props.itemId) return
  const result: any = props.type === 'cr'
    ? await listCRComments(props.kbId, props.itemId)
    : await listIssueComments(props.kbId, props.itemId)
  if (result?.success === false) return
  comments.value = result?.data || []
}

const loadDetail = async () => {
  if (!props.visible || !props.itemId || !props.type) return
  const generation = ++loadGeneration
  loading.value = true
  loadError.value = ''
  issue.value = null
  cr.value = null
  comments.value = []
  reviewMode.value = ''
  resubmitMode.value = false
  try {
    if (props.type === 'cr') {
      const result: any = await getDocChangeRequest(props.kbId, props.itemId)
      if (generation !== loadGeneration) return
      if (result?.success === false || !result?.data) {
        loadError.value = result?.message || t('collaboration.loadFailed')
        return
      }
      cr.value = result.data
    } else {
      const result: any = await getDocIssue(props.kbId, props.itemId)
      if (generation !== loadGeneration) return
      if (result?.success === false || !result?.data) {
        loadError.value = result?.message || t('collaboration.loadFailed')
        return
      }
      issue.value = result.data
    }
    await loadComments()
  } catch (error: any) {
    if (generation === loadGeneration) {
      loadError.value = error?.message || t('collaboration.loadFailed')
    }
  } finally {
    if (generation === loadGeneration) loading.value = false
  }
}

watch(
  () => [props.visible, props.type, props.itemId],
  () => { loadDetail() },
  { immediate: true },
)

const handleClose = () => {
  emit('update:visible', false)
}

const toggleIssueStatus = async (status: 'open' | 'closed') => {
  if (!issue.value) return
  try {
    const result: any = await updateDocIssue(props.kbId, issue.value.id, { status })
    if (result?.success === false) {
      MessagePlugin.error(result?.message || t('collaboration.statusChangeFailed'))
      return
    }
    if (result?.data) issue.value = result.data
    MessagePlugin.success(t('collaboration.statusChanged'))
    emit('changed')
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('collaboration.statusChangeFailed'))
  }
}

const removeIssue = async () => {
  if (!issue.value) return
  try {
    const result: any = await deleteDocIssue(props.kbId, issue.value.id)
    if (result?.success === false) {
      MessagePlugin.error(result?.message || t('collaboration.deleteFailed'))
      return
    }
    MessagePlugin.success(t('collaboration.deleteSuccess'))
    emit('changed')
    emit('update:visible', false)
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('collaboration.deleteFailed'))
  }
}

const startReview = (mode: 'approve' | 'request_changes') => {
  reviewMode.value = mode
  reviewComment.value = ''
  closeIssueIds.value = []
}

const cancelReview = () => {
  reviewMode.value = ''
  reviewComment.value = ''
  closeIssueIds.value = []
}

const toggleCloseIssue = (id: string, checked: boolean) => {
  if (checked) {
    if (!closeIssueIds.value.includes(id)) closeIssueIds.value = [...closeIssueIds.value, id]
    return
  }
  closeIssueIds.value = closeIssueIds.value.filter((value) => value !== id)
}

const submitReview = async () => {
  if (!cr.value || !reviewMode.value) return
  if (reviewMode.value === 'request_changes' && !reviewComment.value.trim()) {
    MessagePlugin.warning(t('collaboration.rejectCommentRequired'))
    return
  }
  reviewSubmitting.value = true
  try {
    const result: any = await reviewDocChangeRequest(props.kbId, cr.value.id, {
      action: reviewMode.value,
      comment: reviewComment.value.trim(),
    })
    if (result?.success === false) {
      MessagePlugin.error(result?.message || t('collaboration.reviewFailed'))
      return
    }
    if (result?.data) cr.value = result.data
    MessagePlugin.success(t('collaboration.reviewSaved'))
    cancelReview()
    emit('changed')
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('collaboration.reviewFailed'))
  } finally {
    reviewSubmitting.value = false
  }
}

const startResubmit = () => {
  if (!cr.value) return
  resubmitMode.value = true
  resubmitTitle.value = cr.value.title
  resubmitDescription.value = cr.value.description || ''
  const suggestions: Record<string, string> = {}
  for (const item of cr.value.items || []) {
    suggestions[item.chunk_id] = item.suggested_content
  }
  resubmitSuggestions.value = suggestions
}

const cancelResubmit = () => {
  resubmitMode.value = false
}

const submitResubmit = async () => {
  if (!cr.value) return
  if (!resubmitTitle.value.trim()) {
    MessagePlugin.warning(t('collaboration.titleRequired'))
    return
  }
  const items: Array<{ chunk_id: string; suggested_content: string }> = []
  for (const item of cr.value.items || []) {
    const text = resubmitSuggestions.value[item.chunk_id] || ''
    if (!text.trim()) {
      MessagePlugin.warning(t('collaboration.crSuggestionRequired', { index: (item.chunk_index || 0) + 1 }))
      return
    }
    const current = item.current_content !== undefined ? item.current_content : item.original_content
    if (text === current) {
      MessagePlugin.warning(t('collaboration.crSuggestionUnchanged', { index: (item.chunk_index || 0) + 1 }))
      return
    }
    items.push({ chunk_id: item.chunk_id, suggested_content: text })
  }
  resubmitSubmitting.value = true
  try {
    const result: any = await resubmitDocChangeRequest(props.kbId, cr.value.id, {
      knowledge_id: cr.value.knowledge_id,
      title: resubmitTitle.value.trim(),
      description: resubmitDescription.value.trim(),
      items,
    })
    if (result?.success === false) {
      MessagePlugin.error(result?.message || t('collaboration.resubmitFailed'))
      return
    }
    if (result?.data) cr.value = result.data
    MessagePlugin.success(t('collaboration.resubmitSuccess'))
    resubmitMode.value = false
    emit('changed')
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('collaboration.resubmitFailed'))
  } finally {
    resubmitSubmitting.value = false
  }
}

const canDeleteComment = (comment: DocComment) =>
  props.canManage || (!!authStore.currentUserId && comment.created_by === authStore.currentUserId)

const removeComment = async (comment: DocComment) => {
  try {
    const result: any = await deleteDocComment(props.kbId, comment.id)
    if (result?.success === false) {
      MessagePlugin.error(result?.message || t('collaboration.deleteFailed'))
      return
    }
    comment.deleted = true
    comment.body = ''
    MessagePlugin.success(t('collaboration.deleteSuccess'))
    emit('changed')
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('collaboration.deleteFailed'))
  }
}

const submitComment = async () => {
  if (!commentDraft.value.trim() || !props.itemId) return
  commentSubmitting.value = true
  try {
    const body = commentDraft.value.trim()
    const result: any = props.type === 'cr'
      ? await createCRComment(props.kbId, props.itemId, body)
      : await createIssueComment(props.kbId, props.itemId, body)
    if (result?.success === false) {
      MessagePlugin.error(result?.message || t('collaboration.commentFailed'))
      return
    }
    if (result?.data) comments.value = [...comments.value, result.data]
    commentDraft.value = ''
    MessagePlugin.success(t('collaboration.commentSuccess'))
    emit('changed')
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('collaboration.commentFailed'))
  } finally {
    commentSubmitting.value = false
  }
}
</script>

<style lang="less" scoped>
.collab-detail-heading {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.collab-detail-kicker {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--td-text-color-placeholder);
}

.collab-detail-seq {
  font-weight: 600;
  color: var(--td-text-color-secondary);
}

.collab-detail-title {
  font-size: 16px;
  font-weight: 600;
  line-height: 1.4;
  color: var(--td-text-color-primary);
  word-break: break-word;
}

.collab-detail-tags {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}

.collab-detail-doc {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  font-size: 12px;
  color: var(--td-text-color-placeholder);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.collab-detail-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding-bottom: 24px;
}

.collab-detail-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 64px 0;
  font-size: 13px;
  color: var(--td-text-color-placeholder);

  &--error {
    flex-direction: column;
    color: var(--td-text-color-secondary);
  }
}

.collab-detail-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
  border-radius: 8px;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
}

.collab-detail-section-title {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--td-text-color-primary);
}

.collab-meta-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 28px;
}

.collab-meta-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.collab-meta-label {
  font-size: 11px;
  color: var(--td-text-color-placeholder);
}

.collab-meta-value {
  font-size: 12px;
  color: var(--td-text-color-secondary);
}

.collab-context-card {
  padding: 10px 12px;
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
}

.collab-context-head {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 500;
  color: var(--td-text-color-secondary);
}

.collab-context-excerpt {
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--td-text-color-secondary);
  white-space: pre-wrap;
  word-break: break-word;
}

.collab-body-text {
  font-size: 13px;
  line-height: 1.75;
  color: var(--td-text-color-primary);
  white-space: pre-wrap;
  word-break: break-word;

  &--muted {
    color: var(--td-text-color-secondary);
  }
}

.collab-banner {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  padding: 8px 10px;
  border-radius: 6px;
  font-size: 12px;
  line-height: 1.6;

  &--warning {
    background: var(--td-warning-color-1);
    color: var(--td-warning-color-7);
  }

  &--danger {
    background: var(--td-error-color-1);
    color: var(--td-error-color);
  }
}

.collab-detail-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  flex-wrap: wrap;
  padding-top: 4px;
}

.collab-diff-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.collab-diff-card {
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  overflow: hidden;
}

.collab-diff-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  padding: 8px 12px;
  background: var(--td-bg-color-secondarycontainer);
}

.collab-diff-head-left {
  display: flex;
  align-items: center;
  gap: 6px;
}

.collab-diff-index {
  font-size: 12px;
  font-weight: 600;
  color: var(--td-text-color-primary);
}

.collab-diff-legend {
  display: flex;
  align-items: center;
  gap: 10px;
}

.collab-diff-legend-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: var(--td-text-color-placeholder);

  i {
    width: 10px;
    height: 10px;
    border-radius: 2px;
  }

  &--del i { background: var(--td-error-color-3); }
  &--add i { background: var(--td-success-color-3); }
}

.collab-diff-body {
  margin: 0;
  padding: 8px 0;
  max-height: 360px;
  overflow: auto;
  font-family: var(--td-font-family-mono, monospace);
  font-size: 12px;
  line-height: 1.6;
  background: var(--td-bg-color-container);
  white-space: pre-wrap;
  word-break: break-word;
}

.collab-diff-line {
  display: block;
  padding: 0 12px;

  &--del {
    background: var(--td-error-color-1);
    color: var(--td-error-color-7);
  }

  &--add {
    background: var(--td-success-color-1);
    color: var(--td-success-color-7);
  }

  &--same {
    color: var(--td-text-color-secondary);
  }
}

.collab-diff-empty {
  padding: 12px;
  font-size: 12px;
  color: var(--td-text-color-placeholder);
}

.collab-current-content {
  border-top: 1px dashed var(--td-component-stroke);
  padding: 8px 12px 12px;
  background: var(--td-bg-color-container);

  pre {
    margin: 4px 0 0;
    max-height: 200px;
    overflow: auto;
    font-size: 12px;
    line-height: 1.6;
    white-space: pre-wrap;
    word-break: break-word;
    color: var(--td-text-color-secondary);
  }
}

.collab-current-content-title {
  font-size: 11px;
  font-weight: 600;
  color: var(--td-text-color-placeholder);
}

.collab-linked-issue-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.collab-linked-issue {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  font-size: 12px;
  color: var(--td-text-color-secondary);
}

.collab-linked-issue-seq {
  font-weight: 600;
}

.collab-linked-issue-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.collab-review-history {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.collab-review-history-item {
  padding: 8px 10px;
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
}

.collab-review-history-head {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--td-text-color-secondary);
}

.collab-review-history-user {
  font-weight: 500;
}

.collab-review-history-time {
  color: var(--td-text-color-placeholder);
}

.collab-review-history-comment {
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--td-text-color-secondary);
  white-space: pre-wrap;
  word-break: break-word;
}

.collab-review-form,
.collab-resubmit-form {
  border-color: var(--td-brand-color-light-active);
  background: var(--td-bg-color-container);
}

.collab-review-hint {
  margin: 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--td-text-color-placeholder);
}

.collab-linked-check-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.collab-linked-check-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--td-text-color-primary);
}

.collab-linked-check {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  padding: 6px 8px;
  border-radius: 6px;
  border: 1px solid var(--td-component-stroke);
  cursor: pointer;
}

.collab-linked-check-seq {
  font-size: 12px;
  font-weight: 600;
  color: var(--td-text-color-secondary);
}

.collab-linked-check-title {
  font-size: 12px;
  font-weight: 400;
  color: var(--td-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.collab-review-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.collab-form-field {
  display: flex;
  flex-direction: column;
  gap: 6px;

  label {
    font-size: 12px;
    font-weight: 500;
    color: var(--td-text-color-primary);
  }
}

.collab-empty-hint {
  padding: 10px 12px;
  border-radius: 6px;
  border: 1px dashed var(--td-component-stroke);
  font-size: 12px;
  color: var(--td-text-color-placeholder);
}

.collab-comments-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.collab-comment-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.collab-comment {
  padding: 10px 12px;
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);

  &.is-deleted {
    padding: 8px 12px;
    background: transparent;
    border: 1px dashed var(--td-component-stroke);
  }
}

.collab-comment-head {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--td-text-color-secondary);
}

.collab-comment-avatar {
  display: inline-flex;
  color: var(--td-brand-color);
}

.collab-comment-user {
  font-weight: 500;
}

.collab-comment-time {
  color: var(--td-text-color-placeholder);
}

.collab-comment-delete {
  margin-left: auto;
}

.collab-comment-body {
  margin-top: 6px;
  font-size: 13px;
  line-height: 1.65;
  color: var(--td-text-color-primary);
  white-space: pre-wrap;
  word-break: break-word;
}

.collab-comment-deleted {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-style: italic;
  color: var(--td-text-color-placeholder);
}

.collab-comment-composer {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-top: 4px;
}

.collab-comment-composer-actions {
  display: flex;
  justify-content: flex-end;
}
</style>
