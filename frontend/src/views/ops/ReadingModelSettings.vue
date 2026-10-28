<template>
  <div class="reading-model-settings">
    <div class="reading-model-settings__head">
      <h3>{{ $t('ops.readingModel.title') }}</h3>
      <p>{{ $t('ops.readingModel.description') }}</p>
    </div>
    <t-select
      v-model="selectedModelId"
      :options="modelOptions"
      :loading="loading"
      :placeholder="$t('ops.readingModel.placeholder')"
      filterable
      clearable
      @change="save"
    />
    <p v-if="!loading && !modelOptions.length" class="reading-model-settings__hint">
      {{ $t('ops.readingModel.noModels') }}
    </p>
    <p v-else class="reading-model-settings__hint">
      {{ $t('ops.readingModel.hint') }}
    </p>
    <p v-if="saved" class="reading-model-settings__saved">{{ $t('ops.readingModel.saved') }}</p>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { listModels } from '@/api/model'
import { getReadingConfig, updateReadingConfig } from '@/api/reading'
import { modelSupportsThinking } from '@/utils/thinkingControl'

const loading = ref(false)
const selectedModelId = ref('')
const modelOptions = ref<Array<{ value: string; label: string }>>([])
const saved = ref(false)

async function load() {
  loading.value = true
  try {
    const [models, config] = await Promise.all([
      listModels('KnowledgeQA'),
      getReadingConfig(),
    ])
    modelOptions.value = (models || [])
      .filter((model: any) => modelSupportsThinking({
        type: String(model.type || ''),
        source: String(model.source || ''),
        name: String(model.name || ''),
        parameters: model.parameters || {},
      }))
      .map((model: any) => ({
        value: String(model.id || ''),
        label: `${model.display_name || model.name || String(model.id || '')} · 思考模型`,
      })).filter((option) => option.value)
    selectedModelId.value = String(config?.data?.reading_model_id || '')
  } catch (err: any) {
    MessagePlugin.error(err?.message || String(err))
  } finally {
    loading.value = false
  }
}

async function save() {
  saved.value = false
  try {
    const response = await updateReadingConfig(selectedModelId.value)
    selectedModelId.value = String(response?.data?.reading_model_id || '')
    saved.value = true
  } catch (err: any) {
    MessagePlugin.error(err?.message || String(err))
  }
}

onMounted(load)
</script>

<style scoped lang="less">
.reading-model-settings {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-width: 620px;
}

.reading-model-settings__head h3 {
  margin: 0 0 5px;
  font-size: 15px;
  font-weight: 600;
}

.reading-model-settings__head p,
.reading-model-settings__hint {
  margin: 0;
  font-size: 12px;
  line-height: 1.5;
  color: var(--td-text-color-secondary);
}

.reading-model-settings__saved {
  margin: 0;
  font-size: 12px;
  color: var(--td-success-color);
}
</style>
