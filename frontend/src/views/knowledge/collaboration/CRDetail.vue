<template>
  <div class="collab-v2 cr-detail-page">
    <div class="cv-topbar">
      <button type="button" class="cv-back" @click="backToDoc">‹ {{ t('collaboration.backToDoc') }}</button>
      <span class="cv-topbar-title">{{ t('collaboration.typeCR') }} #{{ cr?.seq_id || '' }}</span>
      <button type="button" class="cv-icon-btn" :title="t('common.refresh')" @click="loadPage">
        <t-icon name="refresh" size="14px" />
      </button>
    </div>

    <div v-if="loading" class="cv-empty">{{ t('common.loading') }}</div>
    <div v-else-if="loadError" class="cv-empty">{{ loadError }}</div>
    <template v-else-if="cr">
      <header class="detail-header">
        <div class="detail-heading">
          <div class="detail-title">
            {{ cr.title || t('collaboration.untitled') }}
            <span class="cv-badge" :class="CR_STATUS_BADGE[cr.status]">{{ CR_STATUS_LABELS[cr.status] }}</span>
            <span class="detail-seq">#{{ cr.seq_id }}</span>
          </div>
          <div class="detail-meta">
            <b>{{ cr.created_by_name || cr.created_by }}</b>
            {{ t('collaboration.crWantsMerge', { count: cr.items?.length || 0 }) }}
            <span class="version-pill">{{ baseVersionLabel }}</span>
            →
            <span class="version-pill is-new">{{ targetVersionLabel }}</span>
            <span class="cv-muted">· {{ formatDateTime(cr.created_at) }}</span>
            <span v-if="approvedCount" class="approved-text">· {{ t('collaboration.approvedCount', { approved: approvedCount, total: reviewers.length }) }}</span>
          </div>
        </div>
      </header>

      <div class="cv-tabs">
        <button type="button" class="cv-tab" :class="{ 'is-active': tab === 'conversation' }" @click="tab = 'conversation'">
          {{ t('collaboration.tabConversation') }}<span class="cv-tab-count">{{ conversation.length }}</span>
        </button>
        <button type="button" class="cv-tab" :class="{ 'is-active': tab === 'changes' }" @click="tab = 'changes'">
          {{ t('collaboration.tabChanges') }}<span class="cv-tab-count">{{ cr.items?.length || 0 }}</span>
        </button>
        <button type="button" class="cv-tab" :class="{ 'is-active': tab === 'versions' }" @click="tab = 'versions'">
          {{ t('collaboration.tabVersions') }}
        </button>
        <button type="button" class="cv-tab" :class="{ 'is-active': tab === 'checks' }" @click="tab = 'checks'">
          {{ t('collaboration.tabChecks') }}<span class="cv-tab-count">{{ warningCount }}</span>
        </button>
      </div>

      <div class="detail-body">
        <main class="detail-main">
          <template v-if="tab === 'conversation'">
            <div class="cv-card pad conversation-card">
              <span class="cv-avatar">{{ userInitial(cr.created_by_name, cr.created_by) }}</span>
              <div class="conversation-main">
                <div class="conversation-meta"><b>{{ cr.created_by_name || cr.created_by }}</b> {{ t('collaboration.createdThisCR') }} · {{ formatDateTime(cr.created_at) }}</div>
                <p class="conversation-text">{{ cr.description || t('collaboration.noDescription') }}</p>
              </div>
            </div>
            <div class="timeline">
              <div v-for="event in conversation" :key="event.key" class="timeline-event" :class="{ 'is-ok': event.kind === 'review' && event.action === 'accept' }">
                <div class="timeline-content">
                  <div class="timeline-title">
                    <b>{{ event.actor }}</b>
                    <span v-if="event.kind === 'review'">
                      {{ event.action === 'accept' ? t('collaboration.reviewedApproved') : t('collaboration.reviewedChanges') }}
                    </span>
                    <span v-else>{{ t('collaboration.commented') }}</span>
                    <span class="cv-muted">· {{ formatDateTime(event.time) }}</span>
                  </div>
                  <div v-if="event.body" class="timeline-body">{{ event.body }}</div>
                  <button
                    v-if="event.kind === 'comment' && canDeleteComment(event)"
                    type="button"
                    class="cv-link timeline-delete"
                    @click="deleteComment(event.id)"
                  >{{ t('common.remove') }}</button>
                </div>
              </div>
            </div>
            <div class="comment-composer">
              <textarea v-model="commentDraft" class="cv-textarea" rows="3" :placeholder="t('collaboration.commentPlaceholder')" />
              <div class="composer-actions">
                <button type="button" class="cv-btn cv-btn-primary" :disabled="!commentDraft.trim() || commentSubmitting" @click="postComment">
                  {{ t('collaboration.commentSubmit') }}
                </button>
              </div>
            </div>
          </template>

          <template v-else-if="tab === 'changes'">
            <div class="diff-toolbar">
              <div class="cv-seg">
                <button type="button" :class="{ 'is-active': diffMode === 'unified' }" @click="diffMode = 'unified'">Unified</button>
                <button type="button" :class="{ 'is-active': diffMode === 'split' }" @click="diffMode = 'split'">Split</button>
              </div>
              <span class="cv-muted">{{ t('collaboration.changesTotal', { added: totalAdded, removed: totalRemoved, count: cr.items?.length || 0 }) }}</span>
            </div>
            <article v-for="item in cr.items || []" :key="item.id" class="cv-card diff-card">
              <div class="cv-card-head">
                <span class="chunk-id">#{{ (item.chunk_index ?? 0) + 1 }}</span>
                <span class="cv-muted small">{{ t('collaboration.similarity', { value: itemSimilarity(item) }) }} · {{ t('collaboration.changeCount', { count: itemStats(item).added + itemStats(item).removed }) }}</span>
                <span v-if="item.anchor_orphaned" class="cv-badge is-action">{{ t('collaboration.chunkOrphaned') }}</span>
                <span v-else-if="item.conflict" class="cv-badge is-action">{{ t('collaboration.conflict') }}</span>
                <label class="seen-check">
                  <input type="checkbox" :checked="isSeen(item)" :disabled="!canMarkViewed" @change="markSeen(item)" />
                  {{ t('collaboration.seen') }}
                </label>
              </div>
              <DiffBlock :before="item.original_content" :after="item.suggested_content" :mode="diffMode" />
              <div v-if="item.conflict && item.current_content !== undefined" class="current-content">
                <div class="current-content-title">{{ t('collaboration.currentContent') }}</div>
                <pre>{{ item.current_content || t('collaboration.noItemContent') }}</pre>
              </div>
            </article>
          </template>

          <template v-else-if="tab === 'versions'">
            <div class="version-summary">
              <span class="version-pill">{{ baseVersionLabel }}</span>
              <span>→</span>
              <span class="version-pill is-new">{{ targetVersionLabel }}</span>
              <button type="button" class="cv-btn" @click="openVersionTab">{{ t('collaboration.openVersionHistory') }}</button>
            </div>
            <div class="cv-card">
              <div v-for="version in versions" :key="version.id" class="version-row">
                <span class="version-pill" :class="{ 'is-new': version.is_current }">{{ version.version_label }}</span>
                <span class="version-row-title">{{ version.title || versionSourceLabel(version.source_type) }}</span>
                <span class="cv-muted small">{{ versionSourceLabel(version.source_type) }} · {{ version.chunk_count }} {{ t('collaboration.chunksUnit') }}</span>
                <span class="version-row-time">{{ formatDateTime(version.created_at) }}</span>
              </div>
            </div>
          </template>

          <template v-else>
            <div class="check-list">
              <div v-for="item in checks" :key="item.key" class="check-item" :class="{ 'is-warning': !item.ok }">
                <span class="check-dot" />
                <div>
                  <div class="check-title">{{ item.label }}</div>
                  <div class="check-hint">{{ item.hint }}</div>
                </div>
              </div>
            </div>
          </template>
        </main>

        <aside class="detail-side">
          <section class="cv-card side-card">
            <div class="side-head">
              <span>{{ t('collaboration.reviewers') }}</span>
              <button v-if="canManageReviewers" type="button" class="cv-link" @click="openReviewerPicker">＋</button>
            </div>
            <div class="side-body reviewer-list">
              <div v-for="reviewer in reviewers" :key="reviewer.id" class="reviewer-row">
                <span class="cv-avatar">{{ userInitial(reviewer.user_name, reviewer.user_id) }}</span>
                <span class="reviewer-name">{{ reviewer.user_name || reviewer.user_id }}</span>
                <span class="reviewer-status" :class="reviewer.status">{{ REVIEWER_STATUS_LABELS[reviewer.status] }}</span>
              </div>
              <div v-if="!reviewers.length" class="cv-muted small">{{ t('collaboration.noReviewer') }}</div>
            </div>
          </section>
          <section class="cv-card side-card">
            <div class="side-head">{{ t('collaboration.version') }}</div>
            <div class="side-body version-pair">
              <span class="version-pill">{{ baseVersionLabel }}</span>
              <span>→</span>
              <span class="version-pill is-new">{{ targetVersionLabel }}</span>
            </div>
          </section>
          <section class="cv-card side-card">
            <div class="side-head">{{ t('collaboration.changeStats') }}</div>
            <div class="side-body change-stats">
              <span class="cv-add">+{{ totalAdded }} {{ t('collaboration.lines') }}</span>
              <span class="cv-del">−{{ totalRemoved }} {{ t('collaboration.lines') }}</span>
              <span class="cv-muted">{{ cr.items?.length || 0 }} {{ t('collaboration.chunksUnit') }}</span>
            </div>
          </section>
          <section v-if="cr.tags?.length" class="cv-card side-card">
            <div class="side-head">{{ t('collaboration.tags') }}</div>
            <div class="side-body tags-row">
              <span v-for="tag in cr.tags" :key="tag" class="cv-chip">{{ tag }}</span>
            </div>
          </section>

          <template v-if="cr.status === 'pending_review'">
            <button v-if="canMerge" type="button" class="cv-btn cv-btn-primary cv-btn-block" @click="openMergeDialog">
              {{ t('collaboration.mergeCR') }}
            </button>
            <div class="side-hint">{{ t('collaboration.mergeHint', { version: targetVersionLabel }) }}</div>
            <div v-if="needsApprovalToMerge" class="side-hint is-warning">{{ t('collaboration.mergeApprovalRequired') }}</div>
            <div v-if="canReview" class="review-actions">
              <button type="button" class="cv-btn cv-btn-block" :disabled="reviewSubmitting" @click="openReview('approve')">
                {{ t('collaboration.approveCR') }}
              </button>
              <button type="button" class="cv-btn cv-btn-danger cv-btn-block" :disabled="reviewSubmitting" @click="openReview('request_changes')">
                {{ t('collaboration.requestChanges') }}
              </button>
            </div>
            <button v-if="canClose" type="button" class="cv-btn cv-btn-danger cv-btn-block ghost-block" @click="closeCR">
              {{ t('collaboration.closeCR') }}
            </button>
          </template>
          <template v-else-if="cr.status === 'changes_requested'">
            <button v-if="isCreator" type="button" class="cv-btn cv-btn-primary cv-btn-block" @click="editAgain">
              {{ t('collaboration.editAndResubmit') }}
            </button>
            <div class="side-hint">{{ t('collaboration.changesRequestedHint') }}</div>
          </template>
          <template v-else-if="cr.status === 'merged'">
            <div class="merged-note">{{ t('collaboration.mergedInto', { version: targetVersionLabel }) }}</div>
          </template>
        </aside>
      </div>
    </template>

    <t-dialog v-model:visible="reviewDialogVisible" :header="reviewMode === 'approve' ? t('collaboration.approveCR') : t('collaboration.requestChanges')" width="520px" :footer="false">
      <div class="collab-v2 dialog-body">
        <p class="cv-muted">{{ reviewMode === 'approve' ? t('collaboration.approveHint') : t('collaboration.requestChangesHint') }}</p>
        <textarea v-model="reviewComment" class="cv-textarea" rows="4" :placeholder="t('collaboration.reviewCommentPlaceholder')" />
        <div class="dialog-actions">
          <button type="button" class="cv-btn" @click="reviewDialogVisible = false">{{ t('common.cancel') }}</button>
          <button type="button" class="cv-btn cv-btn-primary" :disabled="reviewSubmitting" @click="submitReview">{{ t('collaboration.reviewConfirm') }}</button>
        </div>
      </div>
    </t-dialog>

    <t-dialog v-model:visible="mergeDialogVisible" :header="t('collaboration.mergeCR')" width="560px" :footer="false">
      <div class="collab-v2 dialog-body">
        <p class="cv-muted">{{ t('collaboration.mergeDialogHint', { version: targetVersionLabel }) }}</p>
        <div v-if="openLinkedIssues.length" class="linked-close-list">
          <div class="cv-section-label"><b>{{ t('collaboration.linkedIssues') }}</b></div>
          <label v-for="issue in openLinkedIssues" :key="issue.id" class="picker-row">
            <input v-model="closeIssueIds" type="checkbox" :value="issue.id" />
            <span>#{{ issue.seq_id }} {{ issue.title }}</span>
          </label>
        </div>
        <textarea v-model="mergeComment" class="cv-textarea" rows="3" :placeholder="t('collaboration.mergeCommentPlaceholder')" />
        <div class="dialog-actions">
          <button type="button" class="cv-btn" @click="mergeDialogVisible = false">{{ t('common.cancel') }}</button>
          <button type="button" class="cv-btn cv-btn-primary" :disabled="merging" @click="confirmMerge">{{ t('collaboration.mergeCR') }}</button>
        </div>
      </div>
    </t-dialog>

    <t-dialog v-model:visible="reviewerDialogVisible" :header="t('collaboration.reviewers')" width="520px" :footer="false">
      <div class="collab-v2 dialog-body">
        <label v-for="member in members" :key="member.user_id" class="picker-row">
          <input v-model="reviewerSelection" type="checkbox" :value="member.user_id" />
          <span>{{ member.username || member.email }} <span class="cv-muted">· {{ member.role }}</span></span>
        </label>
        <div class="dialog-actions">
          <button type="button" class="cv-btn" @click="reviewerDialogVisible = false">{{ t('common.cancel') }}</button>
          <button type="button" class="cv-btn cv-btn-primary" :disabled="reviewerSaving" @click="saveReviewers">{{ t('common.save') }}</button>
        </div>
      </div>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  closeDocChangeRequest,
  createCRComment,
  deleteDocComment,
  getDocChangeRequest,
  listCRComments,
  listDocVersions,
  markDocCRViewed,
  mergeDocChangeRequest,
  reviewDocChangeRequest,
  updateDocCRReviewers,
  type DocCRItem,
  type DocCRReviewer,
  type DocChangeRequest,
  type DocComment,
  type DocVersion,
} from '@/api/collaboration'
import { fetchAllTenantMembers, type TenantMember } from '@/api/tenant/members'
import { useAuthStore } from '@/stores/auth'
import DiffBlock from './DiffBlock.vue'
import { diffStats } from './diff'
import {
  CR_STATUS_BADGE,
  CR_STATUS_LABELS,
  REVIEWER_STATUS_LABELS,
  formatDateTime,
  userInitial,
} from './status'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const kbId = computed(() => String(route.params.kbId || ''))
const docId = computed(() => String(route.params.docId || ''))
const crId = computed(() => String(route.params.crId || ''))

const loading = ref(true)
const loadError = ref('')
const cr = ref<DocChangeRequest | null>(null)
const comments = ref<DocComment[]>([])
const versions = ref<DocVersion[]>([])
const members = ref<TenantMember[]>([])
const tab = ref<'conversation' | 'changes' | 'versions' | 'checks'>('changes')
const diffMode = ref<'unified' | 'split'>('unified')
const commentDraft = ref('')
const commentSubmitting = ref(false)
const reviewDialogVisible = ref(false)
const reviewMode = ref<'approve' | 'request_changes'>('approve')
const reviewComment = ref('')
const reviewSubmitting = ref(false)
const mergeDialogVisible = ref(false)
const mergeComment = ref('')
const closeIssueIds = ref<string[]>([])
const merging = ref(false)
const reviewerDialogVisible = ref(false)
const reviewerSelection = ref<string[]>([])
const reviewerSaving = ref(false)

const currentUserId = computed(() => authStore.currentUserId || '')
const canManage = computed(() => authStore.currentDepartmentRole === 'manager' || authStore.hasRole('admin'))
const isCreator = computed(() => cr.value?.created_by === currentUserId.value)
const reviewers = computed(() => cr.value?.reviewers || [])
const myReviewer = computed<DocCRReviewer | undefined>(() =>
  reviewers.value.find((reviewer) => reviewer.user_id === currentUserId.value),
)
const approvedCount = computed(() => reviewers.value.filter((reviewer) => reviewer.status === 'approved').length)
const approvedByOtherCount = computed(() =>
  reviewers.value.filter((reviewer) => reviewer.status === 'approved' && reviewer.user_id !== cr.value?.created_by).length,
)
const canReview = computed(() =>
  cr.value?.status === 'pending_review'
  && Boolean(cr.value?.can_review)
  && (!myReviewer.value || myReviewer.value.status === 'pending'),
)
const canMerge = computed(() =>
  cr.value?.status === 'pending_review'
  && !cr.value?.anchor_orphaned
  && !(cr.value?.items || []).some((item) => item.conflict || item.anchor_orphaned)
  && approvedByOtherCount.value > 0
  && canManage.value,
)
const needsApprovalToMerge = computed(() =>
  canManage.value
  && cr.value?.status === 'pending_review'
  && !cr.value?.anchor_orphaned
  && !(cr.value?.items || []).some((item) => item.conflict || item.anchor_orphaned)
  && approvedByOtherCount.value === 0,
)
const canClose = computed(() => canManage.value || isCreator.value)
const canManageReviewers = computed(() =>
  (canManage.value || isCreator.value)
  && cr.value?.status !== 'merged'
  && cr.value?.status !== 'closed',
)
const canMarkViewed = computed(() => Boolean(myReviewer.value) || canManage.value)
const openLinkedIssues = computed(() => (cr.value?.linked_issues || []).filter((issue) => issue.status === 'open'))

const baseVersionLabel = computed(() => {
  if (!cr.value) return 'V1.0'
  const match = versions.value.find((version) => version.id === cr.value?.base_version_id)
  return match?.version_label || versions.value[versions.value.length - 1]?.version_label || 'V1.0'
})
const targetVersionLabel = computed(() => {
  if (!cr.value) return 'V1.1'
  const merged = versions.value.find((version) => version.id === cr.value?.merged_version_id)
  if (merged) return merged.version_label
  const latest = versions.value[0]
  const next = latest ? Number(latest.version_seq || 0) + 1 : 1
  return `V1.${next}`
})

const conversation = computed(() => {
  if (!cr.value) return []
  const events: Array<{
    key: string
    kind: 'review' | 'comment'
    id: string
    actor: string
    action?: string
    body?: string
    time: string
    createdBy?: string
  }> = []
  for (const review of cr.value.reviews || []) {
    events.push({
      key: `review-${review.id}`,
      kind: 'review',
      id: review.id,
      actor: review.reviewed_by_name || review.reviewed_by,
      action: review.action,
      body: review.comment,
      time: review.created_at,
    })
  }
  for (const comment of comments.value) {
    if (comment.deleted) continue
    events.push({
      key: `comment-${comment.id}`,
      kind: 'comment',
      id: comment.id,
      actor: comment.created_by_name || comment.created_by,
      body: comment.body,
      time: comment.created_at,
      createdBy: comment.created_by,
    })
  }
  return events.sort((left, right) => new Date(left.time).getTime() - new Date(right.time).getTime())
})

const totalAdded = computed(() => (cr.value?.items || []).reduce((sum, item) => sum + itemStats(item).added, 0))
const totalRemoved = computed(() => (cr.value?.items || []).reduce((sum, item) => sum + itemStats(item).removed, 0))

const checks = computed(() => {
  const items = cr.value?.items || []
  const conflict = items.some((item) => item.conflict)
  const orphaned = items.some((item) => item.anchor_orphaned) || Boolean(cr.value?.anchor_orphaned)
  const unchanged = items.some((item) => item.suggested_content === item.original_content)
  return [
    {
      key: 'anchor',
      ok: !orphaned,
      label: t('collaboration.checkAnchor'),
      hint: orphaned ? t('collaboration.checkAnchorFailed') : t('collaboration.checkPassed'),
    },
    {
      key: 'conflict',
      ok: !conflict,
      label: t('collaboration.checkConflictState'),
      hint: conflict ? t('collaboration.checkConflictFailed') : t('collaboration.checkPassed'),
    },
    {
      key: 'suggestion',
      ok: !unchanged,
      label: t('collaboration.checkSuggestion'),
      hint: unchanged ? t('collaboration.checkSuggestionFailed') : t('collaboration.checkPassed'),
    },
    {
      key: 'reviewer',
      ok: approvedByOtherCount.value > 0,
      label: t('collaboration.checkReviewer'),
      hint: approvedByOtherCount.value > 0
        ? t('collaboration.approvedCount', { approved: approvedCount.value, total: reviewers.value.length })
        : t('collaboration.checkReviewerFailed'),
    },
  ]
})
const warningCount = computed(() => checks.value.filter((item) => !item.ok).length)

function itemStats(item: DocCRItem) {
  return diffStats(item.original_content, item.suggested_content)
}

function itemSimilarity(item: DocCRItem) {
  return `${itemStats(item).similarity}%`
}

function isSeen(item: DocCRItem) {
  return (item.viewed_by || []).includes(currentUserId.value)
}

function canDeleteComment(event: any) {
  return event.kind === 'comment' && (event.createdBy === currentUserId.value || canManage.value)
}

async function loadPage() {
  loading.value = true
  loadError.value = ''
  try {
    const [detailResult, commentResult, versionResult] = await Promise.all([
      getDocChangeRequest(kbId.value, crId.value),
      listCRComments(kbId.value, crId.value).catch(() => null),
      listDocVersions(kbId.value, docId.value).catch(() => null),
    ])
    if (detailResult?.success === false || !detailResult?.data) {
      loadError.value = detailResult?.message || t('collaboration.loadFailed')
      return
    }
    cr.value = detailResult.data
    comments.value = (commentResult as any)?.data || []
    versions.value = (versionResult as any)?.data || []
    const tenantId = Number(authStore.tenant?.id || 0)
    if (tenantId) members.value = await fetchAllTenantMembers(tenantId).catch(() => [])
    const hashTab = String(route.hash || '').replace('#', '')
    if (hashTab === 'conversation' || hashTab === 'changes' || hashTab === 'versions' || hashTab === 'checks') {
      tab.value = hashTab
    }
  } catch (error: any) {
    loadError.value = error?.message || t('collaboration.loadFailed')
  } finally {
    loading.value = false
  }
}

watch(tab, (value) => {
  router.replace({ hash: value })
})

async function postComment() {
  if (!commentDraft.value.trim()) return
  commentSubmitting.value = true
  try {
    const result: any = await createCRComment(kbId.value, crId.value, commentDraft.value.trim())
    if (result?.success === false) {
      MessagePlugin.error(result?.message || t('collaboration.commentFailed'))
      return
    }
    commentDraft.value = ''
    await loadPage()
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('collaboration.commentFailed'))
  } finally {
    commentSubmitting.value = false
  }
}

async function deleteComment(commentId: string) {
  try {
    await deleteDocComment(kbId.value, commentId)
    await loadPage()
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('collaboration.deleteFailed'))
  }
}

async function markSeen(item: DocCRItem) {
  if (!currentUserId.value) return
  try {
    await markDocCRViewed(kbId.value, crId.value)
    item.viewed_by = [...(item.viewed_by || []), currentUserId.value]
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('collaboration.markViewedFailed'))
  }
}

function openReview(mode: 'approve' | 'request_changes') {
  reviewMode.value = mode
  reviewComment.value = ''
  reviewDialogVisible.value = true
}

async function submitReview() {
  reviewSubmitting.value = true
  try {
    const result: any = await reviewDocChangeRequest(kbId.value, crId.value, {
      action: reviewMode.value,
      comment: reviewComment.value.trim(),
    })
    if (result?.success === false) {
      MessagePlugin.error(result?.message || t('collaboration.reviewFailed'))
      return
    }
    reviewDialogVisible.value = false
    MessagePlugin.success(t('collaboration.reviewSaved'))
    await loadPage()
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('collaboration.reviewFailed'))
  } finally {
    reviewSubmitting.value = false
  }
}

function openMergeDialog() {
  mergeComment.value = ''
  closeIssueIds.value = openLinkedIssues.value.map((issue) => issue.id)
  mergeDialogVisible.value = true
}

async function confirmMerge() {
  merging.value = true
  try {
    const result: any = await mergeDocChangeRequest(kbId.value, crId.value, {
      comment: mergeComment.value.trim(),
      close_issue_ids: closeIssueIds.value,
    })
    if (result?.success === false) {
      MessagePlugin.error(result?.message || t('collaboration.mergeFailed'))
      return
    }
    mergeDialogVisible.value = false
    MessagePlugin.success(t('collaboration.mergeSuccess'))
    await loadPage()
    tab.value = 'changes'
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('collaboration.mergeFailed'))
  } finally {
    merging.value = false
  }
}

async function closeCR() {
  try {
    const result: any = await closeDocChangeRequest(kbId.value, crId.value)
    if (result?.success === false) {
      MessagePlugin.error(result?.message || t('collaboration.closeFailed'))
      return
    }
    MessagePlugin.success(t('collaboration.closed'))
    await loadPage()
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('collaboration.closeFailed'))
  }
}

function openReviewerPicker() {
  reviewerSelection.value = reviewers.value.map((reviewer) => reviewer.user_id)
  reviewerDialogVisible.value = true
}

async function saveReviewers() {
  reviewerSaving.value = true
  try {
    const result: any = await updateDocCRReviewers(kbId.value, crId.value, reviewerSelection.value)
    if (result?.success === false) {
      MessagePlugin.error(result?.message || t('collaboration.saveFailed'))
      return
    }
    reviewerDialogVisible.value = false
    await loadPage()
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('collaboration.saveFailed'))
  } finally {
    reviewerSaving.value = false
  }
}

function editAgain() {
  router.push({
    path: `/platform/knowledge-bases/${kbId.value}/doc/${docId.value}/cr/new`,
    query: { cr: crId.value },
  })
}

function openVersionTab() {
  router.push({
    path: `/platform/knowledge-bases/${kbId.value}`,
    query: {
      tab: 'documents',
      knowledge_id: docId.value,
      doc_tab: 'versions',
    },
  })
}

function versionSourceLabel(source: string) {
  const map: Record<string, string> = {
    upload: t('collaboration.versionSourceUpload'),
    manual_edit: t('collaboration.versionSourceManual'),
    cr_merge: t('collaboration.versionSourceMerge'),
    rollback: t('collaboration.versionSourceRollback'),
  }
  return map[source] || source
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

function canDeleteCommentByCreator(createdBy?: string) {
  return createdBy === currentUserId.value || canManage.value
}

onMounted(loadPage)
</script>

<style scoped lang="less">
@import '@/styles/collaboration.css';

.cr-detail-page {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  border-radius: 12px;
  overflow: hidden;
}

.cr-detail-page > .cv-topbar,
.cr-detail-page > .detail-header,
.cr-detail-page > .cv-tabs {
  flex: none;
}

.detail-header {
  padding: 16px 22px 12px;
  border-bottom: 1px solid var(--cv-line);
}

.detail-title {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  font-size: 17px;
  font-weight: 700;
}

.detail-seq {
  color: var(--cv-faint);
  font-size: 13px;
  font-weight: 400;
}

.detail-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  margin-top: 4px;
  color: var(--cv-sub);
  font-size: 12px;
}

.version-pill {
  border: 1px solid var(--cv-line);
  border-radius: 7px;
  padding: 0 9px;
  color: var(--cv-ink);
  font-size: 12px;
  font-weight: 650;
}

.version-pill.is-new {
  background: var(--cv-brand-bg);
  border-color: var(--cv-brand-border);
  color: var(--cv-brand-strong);
}

.approved-text {
  color: var(--cv-brand-strong);
}

.detail-body {
  display: flex;
  flex: 1 1 0;
  min-height: 0;
  gap: 20px;
  padding: 18px 22px;
  align-items: stretch;
  overflow: hidden;
}

.detail-main {
  flex: 1;
  min-width: 0;
  min-height: 0;
  overflow-y: auto;
  scrollbar-width: none;
  -ms-overflow-style: none;
}

.detail-main::-webkit-scrollbar {
  width: 0;
  height: 0;
  display: none;
}

.detail-side {
  flex: none;
  width: 252px;
  min-height: 0;
  overflow-y: auto;
  scrollbar-width: none;
  -ms-overflow-style: none;
}

.detail-side::-webkit-scrollbar {
  width: 0;
  height: 0;
  display: none;
}

.pad {
  padding: 12px 16px;
}

.conversation-card {
  display: flex;
  gap: 10px;
  align-items: flex-start;
}

.conversation-main {
  min-width: 0;
}

.conversation-meta {
  color: var(--cv-sub);
  font-size: 12px;
}

.conversation-meta b {
  color: var(--cv-ink);
}

.conversation-text {
  margin: 4px 0 0;
  color: var(--cv-ink);
  font-size: 13px;
  white-space: pre-wrap;
}

.timeline {
  margin: 12px 0 0 12px;
  padding-left: 14px;
  border-left: 2px solid var(--cv-line);
}

.timeline-event {
  position: relative;
  padding: 8px 0;
}

.timeline-event::before {
  content: '';
  position: absolute;
  left: -21px;
  top: 13px;
  width: 10px;
  height: 10px;
  border-radius: 99px;
  background: #fff;
  border: 2px solid var(--cv-faint);
}

.timeline-event.is-ok::before {
  border-color: var(--cv-brand);
  background: var(--cv-brand);
}

.timeline-title {
  color: var(--cv-faint);
  font-size: 12px;
}

.timeline-title b {
  color: var(--cv-ink);
}

.timeline-body {
  margin-top: 4px;
  padding: 8px 14px;
  border: 1px solid var(--cv-line);
  border-radius: 10px;
  color: var(--cv-sub);
  font-size: 12px;
  white-space: pre-wrap;
}

.timeline-delete {
  margin-top: 3px;
  font-size: 11px;
}

.comment-composer {
  margin-top: 16px;
}

.composer-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 8px;
}

.diff-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 10px;
}

.diff-card {
  margin-bottom: 14px;
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

.seen-check {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  margin-left: auto;
  color: var(--cv-sub);
  font-size: 12px;
  cursor: pointer;
}

.seen-check input {
  accent-color: var(--cv-brand);
}

.current-content {
  border-top: 1px solid var(--cv-line);
  padding: 10px 14px;
  background: var(--cv-bg);
}

.current-content-title {
  color: var(--cv-sub);
  font-size: 11px;
  margin-bottom: 4px;
}

.current-content pre {
  margin: 0;
  white-space: pre-wrap;
  font: 12px/1.7 ui-monospace, Menlo, Consolas, monospace;
}

.version-summary {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}

.version-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  border-top: 1px solid var(--cv-line);
  font-size: 12px;
}

.version-row:first-child {
  border-top: 0;
}

.version-row-title {
  flex: 1;
  color: var(--cv-ink);
  font-weight: 550;
}

.version-row-time {
  color: var(--cv-faint);
  font-variant-numeric: tabular-nums;
}

.check-list {
  border: 1px solid var(--cv-line);
  border-radius: 10px;
  overflow: hidden;
}

.check-item {
  display: flex;
  gap: 10px;
  padding: 11px 14px;
  border-top: 1px solid var(--cv-line);
}

.check-item:first-child {
  border-top: 0;
}

.check-dot {
  width: 8px;
  height: 8px;
  margin-top: 6px;
  border-radius: 99px;
  background: var(--cv-brand);
  flex: none;
}

.check-item.is-warning .check-dot {
  background: var(--cv-amber);
}

.check-title {
  color: var(--cv-ink);
  font-size: 13px;
  font-weight: 600;
}

.check-hint {
  color: var(--cv-faint);
  font-size: 11px;
}

.side-card {
  margin-bottom: 10px;
}

.side-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 14px;
  border-bottom: 1px solid var(--cv-line);
  background: var(--cv-bg);
  color: var(--cv-sub);
  font-size: 12px;
  border-radius: 10px 10px 0 0;
}

.side-body {
  padding: 10px 14px;
}

.reviewer-list {
  display: flex;
  flex-direction: column;
  gap: 7px;
}

.reviewer-row {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: 12px;
}

.reviewer-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.reviewer-status {
  color: var(--cv-faint);
  font-size: 11px;
}

.reviewer-status.approved {
  color: var(--cv-brand-strong);
}

.reviewer-status.changes_requested {
  color: var(--cv-amber);
}

.version-pair {
  display: flex;
  align-items: center;
  gap: 8px;
}

.change-stats {
  display: flex;
  gap: 10px;
  font-variant-numeric: tabular-nums;
  font-size: 12px;
}

.tags-row {
  display: flex;
  gap: 5px;
  flex-wrap: wrap;
}

.side-hint {
  margin-top: 5px;
  text-align: center;
  color: var(--cv-faint);
  font-size: 11px;
}

.side-hint.is-warning {
  color: var(--cv-amber);
}

.review-actions {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 10px;
}

.ghost-block {
  margin-top: 10px;
  border-color: transparent;
}

.merged-note {
  padding: 10px 12px;
  border-radius: 9px;
  background: var(--cv-purple-bg);
  color: var(--cv-purple);
  font-size: 12px;
  text-align: center;
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

.linked-close-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
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
</style>
