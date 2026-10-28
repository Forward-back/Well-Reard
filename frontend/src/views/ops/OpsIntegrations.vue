<template>
  <div class="ops-page">
    <header class="ops-page-head">
      <h2>{{ $t('ops.integrations.title') }}</h2>
      <p>{{ $t('ops.integrations.description') }}</p>
    </header>
    <t-tabs v-model="tab" class="ops-tabs">
      <t-tab-panel v-for="item in tabs" :key="item.key" :value="item.key" :label="$t(`integrations.tabs.${item.key}`)">
        <div class="ops-section-card">
          <IntegrationSettingsSection :tab="item.key" />
        </div>
      </t-tab-panel>
    </t-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import IntegrationSettingsSection from '@/views/integrations/IntegrationSettingsSection.vue'
import { INTEGRATION_PREVIEW_ITEMS } from '@/config/integrations'

const tabs = INTEGRATION_PREVIEW_ITEMS
const tab = ref(tabs[0]?.key || 'api')
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
