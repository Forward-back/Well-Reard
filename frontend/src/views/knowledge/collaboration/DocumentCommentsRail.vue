<template>
  <aside class="collab-v2 comments-rail">
    <div class="rail-head">
      <b>{{ t('collaboration.anchorComments') }}</b>
      <span class="cv-muted">{{ comments.length }}</span>
      <button type="button" class="cv-icon-btn" :aria-label="t('common.close')" @click="emit('close')">
        <t-icon name="close" size="14px" />
      </button>
    </div>

    <div v-if="composerAnchor" class="rail-composer">
      <div class="composer-anchor">
        <span class="anchor-chip">#{{ (composerAnchor.chunkIndex || 0) + 1 }}</span>
        <span class="anchor-text">{{ composerAnchor.anchorText || t('collaboration.selectionAnchor') }}</span>
      </div>
      <textarea v-model="draft" class="cv-textarea" rows="3" :placeholder="t('collaboration.anchorCommentPlaceholder')" />
      <div class="composer-buttons">
        <button type="button" class="cv-btn" @click="cancelComposer">{{ t('common.cancel') }}</button>
        <button type="button" class="cv-btn cv-btn-primary" :disabled="!draft.trim() || submitting" @click="submitComposer">
          {{ t('collaboration.commentSubmit') }}
        </button>
      </div>
    </div>

    <div v-if="loading" class="cv-empty">{{ t('common.loading') }}</div>
    <div v-else-if="!threads.length" class="cv-empty">{{ t('collaboration.noAnchorComments') }}</div>
    <div v-else class="rail-list">
      <article v-for="thread in threads" :key="thread.root.id" class="rail-thread" :class="{ 'is-resolved': thread.root.resolved_at }">
        <div class="thread-anchor" role="button" tabindex="0" @click="jump(thread.root)">
          <span class="anchor-chip">#{{ (thread.root.anchor_line_start || 1) }}</span>
          <span class="anchor-text">{{ thread.root.anchor_text || t('collaboration.selectionAnchor') }}</span>
        </div>
        <div class="thread-comment">
          <span class="cv-avatar">{{ userInitial(thread.root.created_by_name, thread.root.created_by) }}</span>
          <div class="thread-main">
            <div class="thread-meta">
              <b>{{ thread.root.created_by_name || thread.root.created_by }}</b>
              <span>{{ formatDateTime(thread.root.created_at) }}</span>
              <span v-if="thread.root.resolved_at" class="resolved-tag">{{ t('collaboration.resolved') }}</span>
            </div>
            <p class="thread-body">{{ thread.root.body }}</p>
            <div class="thread-actions">
              <button type="button" class="cv-link" @click="startReply(thread.root)">{{ t('collaboration.reply') }}</button>
              <button v-if="canManage(thread.root)" type="button" class="cv-link" @click="toggleResolved(thread.root)">
                {{ thread.root.resolved_at ? t('collaboration.reopenComment') : t('collaboration.resolveComment') }}
              </button>
              <button v-if="canManage(thread.root)" type="button" class="cv-link danger" @click="remove(thread.root)">
                {{ t('common.remove') }}
              </button>
            </div>
          </div>
        </div>
        <div v-for="reply in thread.replies" :key="reply.id" class="thread-comment reply">
          <span class="cv-avatar">{{ userInitial(reply.created_by_name, reply.created_by) }}</span>
          <div class="thread-main">
            <div class="thread-meta">
              <b>{{ reply.created_by_name || reply.created_by }}</b>
              <span>{{ formatDateTime(reply.created_at) }}</span>
            </div>
            <p class="thread-body">{{ reply.body }}</p>
          </div>
        </div>
        <div v-if="replyTo === thread.root.id" class="reply-composer">
          <textarea v-model="replyDraft" class="cv-textarea" rows="2" :placeholder="t('collaboration.replyPlaceholder')" />
          <div class="composer-buttons">
            <button type="button" class="cv-btn" @click="replyTo = ''">{{ t('common.cancel') }}</button>
            <button type="button" class="cv-btn cv-btn-primary" :disabled="!replyDraft.trim() || submitting" @click="submitReply(thread.root)">
              {{ t('collaboration.commentSubmit') }}
            </button>
          </div>
        </div>
      </article>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  createAnchorComment,
  deleteAnchorComment,
  listAnchorComments,
  updateAnchorComment,
  type DocAnchorComment,
} from '@/api/collaboration'
import { useAuthStore } from '@/stores/auth'
import { formatDateTime, userInitial } from './status'

export interface CommentAnchor {
  chunkId?: string
  chunkIndex?: number
  anchorText?: string
  anchorStart?: number
  anchorEnd?: number
  anchorLineStart?: number
  anchorLineEnd?: number
}

const { t } = useI18n()
const authStore = useAuthStore()

const props = withDefaults(defineProps<{
  kbId: string
  knowledgeId: string
}>(), {})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'jump-to-chunk', chunkId: string): void
  (e: 'count-change', count: number): void
}>()

const loading = ref(false)
const comments = ref<DocAnchorComment[]>([])
const composerAnchor = ref<CommentAnchor | null>(null)
const draft = ref('')
const submitting = ref(false)
const replyTo = ref('')
const replyDraft = ref('')

const canModerate = computed(() => authStore.hasRole('admin') || authStore.hasRole('owner'))

const threads = computed(() => {
  const roots = comments.value.filter((comment) => !comment.parent_id)
  return roots.map((root) => ({
    root,
    replies: comments.value
      .filter((comment) => comment.parent_id === root.id)
      .sort((left, right) => new Date(left.created_at).getTime() - new Date(right.created_at).getTime()),
  }))
})

function canManage(comment: DocAnchorComment) {
  return canModerate.value || comment.created_by === authStore.currentUserId
}

async function load() {
  if (!props.kbId || !props.knowledgeId) return
  loading.value = true
  try {
    const result: any = await listAnchorComments(props.kbId, props.knowledgeId)
    if (result?.success === false) {
      MessagePlugin.error(result?.message || t('collaboration.loadFailed'))
      return
    }
    comments.value = result?.data || []
    emit('count-change', comments.value.filter((comment) => !comment.deleted_at).length)
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('collaboration.loadFailed'))
  } finally {
    loading.value = false
  }
}

function openComposer(anchor: CommentAnchor) {
  composerAnchor.value = anchor
  draft.value = ''
}

function cancelComposer() {
  composerAnchor.value = null
  draft.value = ''
}

async function submitComposer() {
  const anchor = composerAnchor.value
  if (!anchor || !draft.value.trim()) return
  submitting.value = true
  try {
    const result: any = await createAnchorComment(props.kbId, props.knowledgeId, {
      knowledge_id: props.knowledgeId,
      chunk_id: anchor.chunkId || undefined,
      anchor_text: anchor.anchorText || '',
      anchor_start: anchor.anchorStart || 0,
      anchor_end: anchor.anchorEnd || 0,
      anchor_line_start: anchor.anchorLineStart || 0,
      anchor_line_end: anchor.anchorLineEnd || 0,
      body: draft.value.trim(),
    })
    if (result?.success === false) {
      MessagePlugin.error(result?.message || t('collaboration.commentFailed'))
      return
    }
    cancelComposer()
    await load()
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('collaboration.commentFailed'))
  } finally {
    submitting.value = false
  }
}

function startReply(comment: DocAnchorComment) {
  replyTo.value = comment.id
  replyDraft.value = ''
}

async function submitReply(comment: DocAnchorComment) {
  if (!replyDraft.value.trim()) return
  submitting.value = true
  try {
    const result: any = await createAnchorComment(props.kbId, props.knowledgeId, {
      knowledge_id: props.knowledgeId,
      parent_id: comment.id,
      body: replyDraft.value.trim(),
    })
    if (result?.success === false) {
      MessagePlugin.error(result?.message || t('collaboration.commentFailed'))
      return
    }
    replyTo.value = ''
    replyDraft.value = ''
    await load()
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('collaboration.commentFailed'))
  } finally {
    submitting.value = false
  }
}

async function toggleResolved(comment: DocAnchorComment) {
  try {
    const result: any = await updateAnchorComment(props.kbId, comment.id, { resolved: !comment.resolved_at })
    if (result?.success === false) {
      MessagePlugin.error(result?.message || t('collaboration.saveFailed'))
      return
    }
    await load()
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('collaboration.saveFailed'))
  }
}

async function remove(comment: DocAnchorComment) {
  try {
    const result: any = await deleteAnchorComment(props.kbId, comment.id)
    if (result?.success === false) {
      MessagePlugin.error(result?.message || t('collaboration.deleteFailed'))
      return
    }
    await load()
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('collaboration.deleteFailed'))
  }
}

function jump(comment: DocAnchorComment) {
  if (comment.chunk_id) emit('jump-to-chunk', comment.chunk_id)
}

watch([() => props.kbId, () => props.knowledgeId], ([kbId, knowledgeId]) => {
  if (kbId && knowledgeId) void load()
}, { immediate: true })

defineExpose({ openComposer, reload: load })
</script>

<style scoped lang="less">
@import '@/styles/collaboration.css';

.comments-rail {
  display: flex;
  flex-direction: column;
  width: 300px;
  flex: none;
  border-left: 1px solid var(--cv-line);
  background: #fff;
  min-height: 0;
  height: 100%;
}

.rail-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--cv-line);
  color: var(--cv-sub);
  font-size: 12px;
}

.rail-head b {
  color: var(--cv-ink);
}

.rail-head .cv-icon-btn {
  margin-left: auto;
}

.rail-composer {
  padding: 10px 12px;
  border-bottom: 1px solid var(--cv-line);
  background: var(--cv-bg);
}

.composer-anchor,
.thread-anchor {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  margin-bottom: 6px;
  color: var(--cv-faint);
  font-size: 11px;
}

.thread-anchor {
  cursor: pointer;
}

.anchor-chip {
  flex: none;
  border-radius: 4px;
  padding: 0 5px;
  background: var(--cv-brand-bg);
  color: var(--cv-brand-strong);
  font-weight: 650;
}

.anchor-text {
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.composer-buttons {
  display: flex;
  justify-content: flex-end;
  gap: 6px;
  margin-top: 6px;
}

.rail-list {
  flex: 1;
  overflow: auto;
  min-height: 0;
}

.rail-thread {
  padding: 10px 12px;
  border-top: 1px solid var(--cv-line);
}

.rail-thread:first-child {
  border-top: 0;
}

.rail-thread.is-resolved {
  opacity: 0.62;
}

.thread-comment {
  display: flex;
  gap: 7px;
  margin-top: 4px;
}

.thread-comment.reply {
  margin-left: 22px;
}

.thread-main {
  min-width: 0;
}

.thread-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--cv-faint);
  font-size: 10px;
}

.thread-meta b {
  color: var(--cv-ink);
  font-size: 11px;
}

.resolved-tag {
  border-radius: 99px;
  padding: 0 6px;
  background: var(--cv-chip);
  color: var(--cv-sub);
}

.thread-body {
  margin: 3px 0 0;
  color: var(--cv-ink);
  font-size: 12px;
  white-space: pre-wrap;
}

.thread-actions {
  display: flex;
  gap: 10px;
  margin-top: 4px;
  font-size: 11px;
}

.thread-actions .danger {
  color: var(--cv-red);
}

.reply-composer {
  margin: 6px 0 0 22px;
}
</style>
