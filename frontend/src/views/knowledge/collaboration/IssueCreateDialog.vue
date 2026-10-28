<template>
  <t-dialog
    :visible="visible"
    :footer="false"
    :header="false"
    width="580px"
    dialog-class-name="collab-issue-dialog"
    :close-on-overlay-click="false"
    destroy-on-close
    @close="handleClose"
  >
    <div class="collab-v2 issue-dialog">
      <div class="issue-head">
        <span class="cv-type-icon is-issue"><t-icon name="chat-bubble-help" size="14px" /></span>
        <div class="issue-head-main">
          <div class="issue-title-row">
            <b>{{ t('collaboration.createIssue') }}</b>
            <span v-if="knowledgeTitle" class="issue-doc">{{ knowledgeTitle }}</span>
          </div>
          <span class="issue-sub">{{ anchorLabel }}</span>
        </div>
        <button type="button" class="cv-icon-btn" :aria-label="t('common.close')" @click="handleClose">
          <t-icon name="close" size="15px" />
        </button>
      </div>

      <div class="issue-body">
        <div v-if="anchorExcerpt" class="quote-block">
          <div class="quote-head">
            <span class="quote-index">#{{ (chunkIndex || 0) + 1 }}</span>
            <span>{{ t('collaboration.quoteOriginal') }}</span>
          </div>
          <p class="quote-text">{{ anchorExcerpt }}</p>
        </div>

        <label class="field">
          <span class="field-label">{{ t('collaboration.issueTitle') }}</span>
          <input
            ref="titleInput"
            v-model="title"
            class="cv-input"
            :placeholder="t('collaboration.issueTitlePlaceholder')"
            maxlength="255"
            :disabled="submitting"
          />
        </label>

        <div class="field">
          <span class="field-label">{{ t('collaboration.issueType') }}</span>
          <div class="type-chips">
            <button
              v-for="type in ISSUE_TYPES"
              :key="type"
              type="button"
              class="type-chip"
              :class="{ on: issueType === type }"
              :disabled="submitting"
              @click="issueType = type"
            >
              {{ ISSUE_TYPE_LABELS[type] }}
            </button>
          </div>
        </div>

        <label class="field">
          <span class="field-label">{{ t('collaboration.issueBody') }}</span>
          <textarea
            v-model="body"
            class="cv-textarea"
            :placeholder="t('collaboration.issueBodyPlaceholder')"
            rows="5"
            :disabled="submitting"
          />
        </label>
      </div>

      <div class="issue-footer">
        <button type="button" class="cv-link convert-link" :disabled="submitting" @click="convertToCR">
          <t-icon name="swap" size="14px" />
          {{ t('collaboration.convertToCR') }}
        </button>
        <span class="footer-actions">
          <button type="button" class="cv-btn" :disabled="submitting" @click="handleClose">
            {{ t('common.cancel') }}
          </button>
          <button
            type="button"
            class="cv-btn cv-btn-primary"
            :disabled="submitting || !title.trim()"
            @click="submit(false)"
          >
            {{ submitting ? t('collaboration.submitting') : t('collaboration.submitIssue') }}
          </button>
        </span>
      </div>
    </div>
  </t-dialog>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { createDocIssue, type DocIssue, type DocIssueType } from '@/api/collaboration'
import { ISSUE_TYPES, ISSUE_TYPE_LABELS } from './status'

const { t } = useI18n()
const router = useRouter()

const props = withDefaults(defineProps<{
  visible: boolean
  kbId: string
  knowledgeId: string
  knowledgeTitle?: string
  chunkId?: string
  chunkIndex?: number
  chunkExcerpt?: string
  anchorText?: string
  anchorStart?: number
  anchorEnd?: number
  anchorLineStart?: number
  anchorLineEnd?: number
  defaultTitle?: string
  defaultBody?: string
}>(), {
  knowledgeTitle: '',
  chunkId: '',
  chunkIndex: 0,
  chunkExcerpt: '',
  anchorText: '',
  anchorStart: 0,
  anchorEnd: 0,
  anchorLineStart: 0,
  anchorLineEnd: 0,
  defaultTitle: '',
  defaultBody: '',
})

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'created', issue: DocIssue): void
}>()

const title = ref('')
const body = ref('')
const issueType = ref<DocIssueType>('other')
const submitting = ref(false)
const titleInput = ref<HTMLInputElement | null>(null)

const anchorExcerpt = computed(() => props.anchorText || props.chunkExcerpt || '')
const anchorLabel = computed(() => {
  if (props.chunkId) {
    return t('collaboration.anchorAttached', { index: (props.chunkIndex || 0) + 1 })
  }
  return t('collaboration.documentAnchor')
})

watch(() => props.visible, (visible) => {
  if (!visible) return
  title.value = props.defaultTitle || ''
  body.value = props.defaultBody || ''
  issueType.value = 'other'
  submitting.value = false
  void nextTick(() => titleInput.value?.focus())
})

const handleClose = () => {
  if (submitting.value) return
  emit('update:visible', false)
}

async function createIssue(): Promise<DocIssue | null> {
  if (!title.value.trim()) {
    MessagePlugin.warning(t('collaboration.titleRequired'))
    return null
  }
  submitting.value = true
  try {
    const result: any = await createDocIssue(props.kbId, {
      knowledge_id: props.knowledgeId,
      chunk_id: props.chunkId || undefined,
      issue_type: issueType.value,
      anchor_text: props.anchorText || '',
      anchor_start: props.anchorStart || 0,
      anchor_end: props.anchorEnd || 0,
      anchor_line_start: props.anchorLineStart || 0,
      anchor_line_end: props.anchorLineEnd || 0,
      title: title.value.trim(),
      body: body.value.trim(),
    })
    if (result?.success === false) {
      MessagePlugin.error(result?.message || t('collaboration.createIssueFailed'))
      return null
    }
    const issue = (result?.data || result) as DocIssue
    MessagePlugin.success(t('collaboration.createIssueSuccess'))
    emit('created', issue)
    return issue
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('collaboration.createIssueFailed'))
    return null
  } finally {
    submitting.value = false
  }
}

async function submit(convert: boolean) {
  const issue = await createIssue()
  if (!issue) return
  if (!convert) {
    emit('update:visible', false)
    return
  }
  emit('update:visible', false)
  router.push({
    path: `/platform/knowledge-bases/${props.kbId}/doc/${props.knowledgeId}/cr/new`,
    query: {
      issue: issue.id,
      chunk: props.chunkId || undefined,
      title: title.value.trim(),
      desc: body.value.trim(),
    },
  })
}

async function convertToCR() {
  await submit(true)
}
</script>

<style scoped lang="less">
@import '@/styles/collaboration.css';

.issue-dialog {
  --cv-ink: #111827;
  border-radius: 12px;
  overflow: hidden;
}

.issue-head {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 14px 16px;
  background: var(--cv-bg);
  border-bottom: 1px solid var(--cv-line);
}

.issue-head-main {
  flex: 1;
  min-width: 0;
}

.issue-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.issue-title-row b {
  font-size: 14px;
}

.issue-doc {
  color: var(--cv-faint);
  font-size: 11px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.issue-sub {
  display: block;
  margin-top: 2px;
  color: var(--cv-faint);
  font-size: 11px;
}

.issue-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px 16px;
}

.quote-block {
  border: 1px dashed var(--cv-line);
  border-radius: 9px;
  padding: 9px 12px;
  background: var(--cv-bg);
}

.quote-head {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--cv-faint);
  font-size: 11px;
}

.quote-index {
  color: var(--cv-brand-strong);
  font-weight: 650;
}

.quote-text {
  margin: 4px 0 0;
  color: var(--cv-sub);
  font-size: 12px;
  line-height: 1.65;
  max-height: 84px;
  overflow: auto;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field-label {
  color: var(--cv-sub);
  font-size: 12px;
}

.type-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.type-chip {
  border: 1px solid var(--cv-line);
  border-radius: 99px;
  padding: 2px 12px;
  background: #fff;
  color: var(--cv-sub);
  font: inherit;
  font-size: 12px;
  cursor: pointer;
}

.type-chip.on {
  border-color: var(--cv-brand-border);
  background: var(--cv-brand-bg);
  color: var(--cv-brand-strong);
  font-weight: 650;
}

.issue-footer {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 11px 16px;
  border-top: 1px solid var(--cv-line);
}

.convert-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
}

.footer-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}
</style>
