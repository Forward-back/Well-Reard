<template>
  <div class="ops-page">
    <header class="ops-page-head">
      <h2>{{ $t('ops.settings.title') }}</h2>
      <p>{{ $t('ops.settings.description') }}</p>
    </header>
    <div class="ops-section-card">
      <SystemSettings :sections="['tenant', 'other']" hide-header />
    </div>
    <div class="ops-section-card">
      <t-tabs v-model="workspaceTab">
        <t-tab-panel value="chat" :label="$t('ops.settings.tabs.chat')">
          <ChatHistorySettings />
        </t-tab-panel>
        <t-tab-panel value="sync" :label="$t('ops.settings.tabs.sync')">
          <DocSyncSettings />
        </t-tab-panel>
        <t-tab-panel value="retrieval" :label="$t('ops.settings.tabs.retrieval')">
          <RetrievalSettings />
        </t-tab-panel>
      </t-tabs>
    </div>
    <div class="ops-section-card">
      <SystemInfo />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import SystemSettings from '@/views/system/SystemSettings.vue'
import SystemInfo from '@/views/settings/SystemInfo.vue'
import ChatHistorySettings from '@/views/settings/ChatHistorySettings.vue'
import DocSyncSettings from '@/views/settings/DocSyncSettings.vue'
import RetrievalSettings from '@/views/settings/RetrievalSettings.vue'

const workspaceTab = ref('chat')
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

  & + & {
    margin-top: 16px;
  }
}
</style>
