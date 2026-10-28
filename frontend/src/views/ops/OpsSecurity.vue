<template>
  <div class="ops-page">
    <header class="ops-page-head">
      <h2>{{ $t('ops.security.title') }}</h2>
      <p>{{ $t('ops.security.description') }}</p>
    </header>
    <t-tabs v-model="tab" class="ops-tabs">
      <t-tab-panel value="keys" :label="$t('ops.security.tabs.keys')">
        <div class="ops-section-card"><PlatformAPIKeys /></div>
      </t-tab-panel>
      <t-tab-panel value="audit" :label="$t('ops.security.tabs.audit')">
        <div class="ops-section-card"><SystemAuditLog /></div>
      </t-tab-panel>
      <t-tab-panel value="network" :label="$t('ops.security.tabs.network')">
        <div class="ops-section-card"><SystemSettings :sections="['security']" hide-header /></div>
      </t-tab-panel>
    </t-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import PlatformAPIKeys from '@/views/system/PlatformAPIKeys.vue'
import SystemAuditLog from '@/views/system/SystemAuditLog.vue'
import SystemSettings from '@/views/system/SystemSettings.vue'

const tab = ref('keys')
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
