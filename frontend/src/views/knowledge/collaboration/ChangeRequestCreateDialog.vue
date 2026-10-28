<template>
  <t-dialog :visible="visible" :footer="false" width="720px" dialog-class-name="collab-cr-dialog"
    :close-on-overlay-click="false" destroy-on-close @close="handleClose">
    <template #header>
      <div class="collab-dialog-heading">
        <div class="collab-dialog-heading-row">
          <t-icon name="git-branch" size="16px" class="collab-dialog-heading-icon" aria-hidden="true" />
          <span class="collab-dialog-title">{{ $t('collaboration.createCR') }}</span>
        </div>
        <p v-if="knowledgeTitle" class="collab-dialog-document" :title="knowledgeTitle">{{ knowledgeTitle }}</p>
      </div>
    </template>

    <div class="collab-dialog-body collab-cr-body">
      <div class="collab-form-field">
        <label>{{ $t('collaboration.issueTitle') }}</label>
        <t-input v-model="title" :placeholder="$t('collaboration.crTitlePlaceholder')" :maxlength="255"
          :disabled="submitting" autofocus />
      </div>

      <div class="collab-form-field">
        <label>{{ $t('collaboration.crDescription') }}</label>
        <t-textarea v-model="description" :placeholder="$t('collaboration.crDescriptionPlaceholder')"
          :autosize="{ minRows: 2, maxRows: 5 }" :disabled="submitting" />
      </div>

      <div class="collab-form-field">
        <div class="collab-form-label-row">
          <label>{{ $t('collaboration.crItems') }}</label>
          <span class="collab-form-counter">{{ $t('collaboration.crChunkCount', { count: selectedIds.length }) }}</span>
        </div>
        <div class="collab-chunk-toolbar">
          <t-input v-model="chunkFilter" size="small" clearable :disabled="submitting"
            :placeholder="$t('collaboration.crSearchPlaceholder')" />
          <span class="collab-chunk-load-hint">
            <t-loading v-if="chunkLoading" size="small" />
            {{ chunkLoading ? $t('collaboration.crLoadingChunks') : $t('collaboration.crLoadedChunks', { count: allChunks.length }) }}
          </span>
        </div>
        <div v-if="chunkLoadError" class="collab-empty-hint collab-chunk-load-error">
          {{ $t('collaboration.crChunkLoadFailed') }}
          <t-button variant="text" size="small" :disabled="chunkLoading" @click="loadAllChunks">
            {{ $t('common.retry') }}
          </t-button>
        </div>
        <div v-if="!allChunks.length && !chunkLoading" class="collab-empty-hint">{{ $t('collaboration.crNoChunks') }}</div>
        <div v-else-if="!filteredChunks.length" class="collab-empty-hint">{{ $t('collaboration.crNoMatch') }}</div>
        <div v-else class="collab-chunk-select-list">
          <label v-for="chunk in filteredChunks" :key="chunk.id" class="collab-chunk-select-row"
            :class="{ 'is-selected': isSelected(chunk.id) }">
            <t-checkbox :checked="isSelected(chunk.id)" :disabled="submitting"
              @change="(checked: boolean) => toggleChunk(chunk, checked)" />
            <span class="collab-chunk-select-body">
              <span class="collab-chunk-select-head">
                <span class="collab-chunk-badge">{{ $t('collaboration.chunkShort', { index: (chunk.index || 0) + 1 }) }}</span>
                <span v-if="chunk.is_enabled === false" class="collab-chunk-disabled">
                  {{ $t('knowledgeBase.disabledStatus') }}
                </span>
              </span>
              <span class="collab-chunk-select-excerpt">{{ excerpt(chunk.content) }}</span>
            </span>
          </label>
        </div>
      </div>

      <div v-for="chunk in selectedChunks" :key="'edit-' + chunk.id" class="collab-form-field collab-suggestion-field">
        <div class="collab-form-label-row">
          <label>{{ $t('collaboration.itemN', { index: (chunk.index || 0) + 1 }) }} · {{ $t('collaboration.crSuggested') }}</label>
          <t-button variant="text" size="small" :disabled="submitting" @click="resetSuggestion(chunk)">
            {{ $t('collaboration.crResetSuggestion') }}
          </t-button>
        </div>
        <t-textarea v-model="suggested[chunk.id]" :placeholder="$t('collaboration.crSuggestedPlaceholder')"
          :autosize="{ minRows: 5, maxRows: 14 }" :disabled="submitting" />
      </div>
    </div>

    <div class="collab-dialog-footer">
      <t-button variant="outline" size="small" :disabled="submitting" @click="handleClose">
        {{ $t('common.cancel') }}
      </t-button>
      <t-button theme="primary" size="small" :loading="submitting" :disabled="!canSubmit" @click="submit">
        {{ submitting ? $t('collaboration.submitting') : $t('collaboration.submit') }}
      </t-button>
    </div>
  </t-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import { createDocChangeRequest } from '@/api/collaboration'
import { listKnowledgeChunks } from '@/api/knowledge-base'

export interface CollaborationChunkOption {
  id: string
  index?: number
  content: string
  is_enabled?: boolean
}

const { t } = useI18n()

const props = withDefaults(defineProps<{
  visible: boolean
  kbId: string
  knowledgeId: string
  knowledgeTitle?: string
  chunks: CollaborationChunkOption[]
  initialChunkId?: string
}>(), {
  knowledgeTitle: '',
  chunks: () => [],
  initialChunkId: '',
})

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'created', cr: any): void
}>()

const title = ref('')
const description = ref('')
const selectedIds = ref<string[]>([])
const suggested = ref<Record<string, string>>({})
const submitting = ref(false)
const allChunks = ref<CollaborationChunkOption[]>([])
const chunkFilter = ref('')
const chunkLoading = ref(false)
const chunkLoadError = ref('')
let loadGeneration = 0

const normalizeChunk = (raw: any): CollaborationChunkOption => ({
  id: raw.id,
  index: typeof raw.chunk_index === 'number' ? raw.chunk_index : raw.index,
  content: raw.content || '',
  is_enabled: raw.is_enabled,
})

const chunkMap = computed(() => {
  const map = new Map<string, CollaborationChunkOption>()
  allChunks.value.forEach((chunk) => map.set(chunk.id, chunk))
  return map
})

const filteredChunks = computed(() => {
  const query = chunkFilter.value.trim().toLowerCase()
  if (!query) return allChunks.value
  const numeric = query.replace(/^#/, '')
  return allChunks.value.filter((chunk) =>
    String((chunk.index || 0) + 1).includes(numeric)
    || (chunk.content || '').toLowerCase().includes(query),
  )
})

const loadAllChunks = async () => {
  const generation = ++loadGeneration
  chunkLoading.value = true
  chunkLoadError.value = ''
  try {
    const first: any = await listKnowledgeChunks(props.knowledgeId, 1, 100)
    if (generation !== loadGeneration) return
    if (first?.success === false) {
      throw new Error(first?.message || t('collaboration.crChunkLoadFailed'))
    }
    const firstBatch = (first?.data || []).map(normalizeChunk)
    const total = Number(first?.total || firstBatch.length)
    const pageCount = Math.ceil(total / 100)
    const restPages = await Promise.all(
      Array.from({ length: Math.max(0, pageCount - 1) }, (_, i) => props.knowledgeId
        ? listKnowledgeChunks(props.knowledgeId, i + 2, 100)
        : Promise.resolve(null)),
    )
    if (generation !== loadGeneration) return
    const merged = new Map<string, CollaborationChunkOption>()
    for (const chunk of allChunks.value) merged.set(chunk.id, chunk)
    for (const chunk of firstBatch) merged.set(chunk.id, chunk)
    for (const page of restPages as any[]) {
      if (!page || page.success === false) continue
      for (const raw of page.data || []) {
        const chunk = normalizeChunk(raw)
        if (chunk.id) merged.set(chunk.id, chunk)
      }
    }
    allChunks.value = [...merged.values()].sort((a, b) => (a.index || 0) - (b.index || 0))
  } catch (error: any) {
    if (generation !== loadGeneration) return
    chunkLoadError.value = error?.message || t('collaboration.crChunkLoadFailed')
  } finally {
    if (generation === loadGeneration) chunkLoading.value = false
  }
}

const selectedChunks = computed(() =>
  selectedIds.value.map((id) => chunkMap.value.get(id)).filter(Boolean) as CollaborationChunkOption[],
)

const canSubmit = computed(() => {
  if (!title.value.trim() || selectedIds.value.length === 0) return false
  return selectedChunks.value.every((chunk) => (suggested.value[chunk.id] || '').trim() !== '')
})

watch(() => props.visible, (visible) => {
  if (!visible) {
    loadGeneration++
    chunkLoading.value = false
    return
  }
  title.value = ''
  description.value = ''
  selectedIds.value = []
  suggested.value = {}
  submitting.value = false
  chunkFilter.value = ''
  chunkLoadError.value = ''
  allChunks.value = props.chunks.map(normalizeChunk)
  if (props.initialChunkId) {
    const chunk = allChunks.value.find((item) => item.id === props.initialChunkId)
    if (chunk) {
      selectedIds.value = [chunk.id]
      suggested.value[chunk.id] = chunk.content
    }
  }
  void loadAllChunks()
}, { immediate: true })

const isSelected = (id: string) => selectedIds.value.includes(id)

const toggleChunk = (chunk: CollaborationChunkOption, checked: boolean) => {
  if (checked) {
    if (!selectedIds.value.includes(chunk.id)) {
      selectedIds.value = [...selectedIds.value, chunk.id]
    }
    if (suggested.value[chunk.id] === undefined) {
      suggested.value[chunk.id] = chunk.content
    }
    return
  }
  selectedIds.value = selectedIds.value.filter((id) => id !== chunk.id)
}

const resetSuggestion = (chunk: CollaborationChunkOption) => {
  suggested.value[chunk.id] = chunk.content
}

const excerpt = (content: string) => {
  const text = (content || '').replace(/\s+/g, ' ').trim()
  return text.length > 96 ? `${text.slice(0, 96)}...` : text
}

const handleClose = () => {
  if (submitting.value) return
  emit('update:visible', false)
}

const submit = async () => {
  if (!title.value.trim()) {
    MessagePlugin.warning(t('collaboration.titleRequired'))
    return
  }
  if (selectedIds.value.length === 0) {
    MessagePlugin.warning(t('collaboration.crSelectChunk'))
    return
  }
  const items: Array<{ chunk_id: string; suggested_content: string }> = []
  for (const chunk of selectedChunks.value) {
    const text = suggested.value[chunk.id] || ''
    if (!text.trim()) {
      MessagePlugin.warning(t('collaboration.crSuggestionRequired', { index: (chunk.index || 0) + 1 }))
      return
    }
    if (text === chunk.content) {
      MessagePlugin.warning(t('collaboration.crSuggestionUnchanged', { index: (chunk.index || 0) + 1 }))
      return
    }
    items.push({ chunk_id: chunk.id, suggested_content: text })
  }

  submitting.value = true
  try {
    const result: any = await createDocChangeRequest(props.kbId, {
      knowledge_id: props.knowledgeId,
      title: title.value.trim(),
      description: description.value.trim(),
      items,
    })
    if (result?.success === false) {
      MessagePlugin.error(result?.message || t('collaboration.createCRFailed'))
      return
    }
    MessagePlugin.success(t('collaboration.createCRSuccess'))
    emit('created', result?.data)
    emit('update:visible', false)
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('collaboration.createCRFailed'))
  } finally {
    submitting.value = false
  }
}
</script>

<style lang="less" scoped>
.collab-dialog-heading {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.collab-dialog-heading-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.collab-dialog-heading-icon {
  color: var(--td-brand-color);
}

.collab-dialog-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--td-text-color-primary);
}

.collab-dialog-document {
  margin: 0;
  padding-left: 24px;
  font-size: 12px;
  color: var(--td-text-color-placeholder);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.collab-dialog-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding-top: 4px;
}

.collab-cr-body {
  max-height: 62vh;
  overflow-y: auto;
  padding-right: 4px;
}

.collab-form-field {
  display: flex;
  flex-direction: column;
  gap: 6px;

  label {
    font-size: 13px;
    font-weight: 500;
    color: var(--td-text-color-primary);
  }
}

.collab-form-label-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.collab-form-counter {
  font-size: 12px;
  color: var(--td-text-color-placeholder);
}

.collab-chunk-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;

  :deep(.t-input) {
    flex: 1;
  }
}

.collab-chunk-load-hint {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
  font-size: 12px;
  color: var(--td-text-color-placeholder);
}

.collab-chunk-load-error {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  border-style: solid;
  border-color: var(--td-warning-color-3);
}

.collab-empty-hint {
  padding: 10px 12px;
  border-radius: 6px;
  border: 1px dashed var(--td-component-stroke);
  font-size: 12px;
  color: var(--td-text-color-placeholder);
}

.collab-chunk-select-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 220px;
  overflow-y: auto;
  padding: 2px;
}

.collab-chunk-select-row {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 8px 10px;
  border-radius: 6px;
  border: 1px solid var(--td-component-stroke);
  background: var(--td-bg-color-container);
  cursor: pointer;
  transition: border-color 0.15s ease, background-color 0.15s ease;

  &:hover {
    border-color: var(--td-brand-color-light-active);
    background: var(--td-bg-color-container-hover);
  }

  &.is-selected {
    border-color: var(--td-brand-color);
    background: var(--td-brand-color-light);
  }
}

.collab-chunk-select-body {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}

.collab-chunk-select-head {
  display: flex;
  align-items: center;
  gap: 6px;
}

.collab-chunk-badge {
  font-size: 12px;
  font-weight: 600;
  color: var(--td-brand-color);
}

.collab-chunk-disabled {
  font-size: 11px;
  color: var(--td-text-color-placeholder);
}

.collab-chunk-select-excerpt {
  font-size: 12px;
  line-height: 1.5;
  color: var(--td-text-color-secondary);
  word-break: break-word;
}

.collab-suggestion-field {
  padding: 12px;
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
}

.collab-dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 20px;
  padding-top: 12px;
  border-top: 1px solid var(--td-component-stroke);
}
</style>
