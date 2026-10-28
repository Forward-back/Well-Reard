<template>
  <div class="ops-page">
    <header class="ops-page-head">
      <h2>{{ $t('ops.runtime.title') }}</h2>
      <p>{{ $t('ops.runtime.description') }}</p>
    </header>
    <div class="ops-section-card ops-section-card--flush">
      <RuntimeQueues />
    </div>
    <div class="ops-section-card ops-runtime-config">
      <SystemSettings :sections="['runtime']" hide-header />
    </div>
  </div>
</template>

<script setup lang="ts">
import RuntimeQueues from '@/views/system/RuntimeQueues.vue'
import SystemSettings from '@/views/system/SystemSettings.vue'
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

  &--flush {
    padding: 0;
    overflow: hidden;
  }

  & + & {
    margin-top: 16px;
  }
}

.ops-runtime-config {
  margin-top: 16px;
}
</style>
