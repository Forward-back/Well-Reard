<template>
  <div class="ops-page">
    <header class="ops-page-head">
      <h2>{{ $t('ops.models.title') }}</h2>
      <p>{{ $t('ops.models.description') }}</p>
    </header>
    <t-tabs v-model="tab" class="ops-tabs">
      <t-tab-panel value="models" :label="$t('ops.models.tabs.models')">
        <div class="ops-section-card"><ModelSettings /></div>
      </t-tab-panel>
      <t-tab-panel value="reading" :label="$t('ops.models.tabs.reading')">
        <div class="ops-section-card"><ReadingModelSettings /></div>
      </t-tab-panel>
      <t-tab-panel value="ollama" :label="$t('ops.models.tabs.ollama')">
        <div class="ops-section-card"><OllamaSettings /></div>
      </t-tab-panel>
      <t-tab-panel value="parser" :label="$t('ops.models.tabs.parser')">
        <div class="ops-section-card"><ParserEngineSettings /></div>
      </t-tab-panel>
      <t-tab-panel value="storage" :label="$t('ops.models.tabs.storage')">
        <div class="ops-section-card"><StorageBackendSettings /></div>
      </t-tab-panel>
    </t-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import ModelSettings from '@/views/settings/ModelSettings.vue'
import ReadingModelSettings from '@/views/ops/ReadingModelSettings.vue'
import OllamaSettings from '@/views/settings/OllamaSettings.vue'
import ParserEngineSettings from '@/views/settings/ParserEngineSettings.vue'
import StorageBackendSettings from '@/views/settings/StorageBackendSettings.vue'

const tab = ref('models')
</script>

<style lang="less" scoped>
.ops-page {
  width: 100%;
  max-width: 1220px;
  margin: 0 auto;
  padding: 24px 28px 48px;
  box-sizing: border-box;
}

.ops-page-head {
  margin-bottom: 14px;

  h2 { margin: 0 0 5px; font-size: 18px; font-weight: 600; }
  p { margin: 0; color: var(--td-text-color-secondary); font-size: 12px; }
}

.ops-section-card {
  padding: 20px 22px;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: 10px;
}
</style>
