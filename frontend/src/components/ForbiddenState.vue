<template>
  <div class="forbidden-state">
    <div class="forbidden-icon">
      <t-icon name="lock-on" size="40px" />
    </div>
    <h2 class="forbidden-title">{{ $t('forbidden.title') }}</h2>
    <p class="forbidden-desc">{{ $t('forbidden.description') }}</p>
    <div v-if="requiredCaps.length" class="forbidden-caps">
      <span class="forbidden-caps-label">{{ $t('forbidden.requiredCaps') }}</span>
      <t-tag v-for="cap in requiredCaps" :key="cap" theme="default" variant="light">
        {{ capabilityLabel(cap) }}
      </t-tag>
    </div>
  </div>
</template>

<script setup lang="ts">
import { CAPABILITY_LABELS } from '@/config/capabilities'

defineProps<{ requiredCaps: string[] }>()

const capabilityLabel = (cap: string) => CAPABILITY_LABELS[cap] || cap
</script>

<style lang="less" scoped>
.forbidden-state {
  padding: 40px 24px;
  text-align: center;
}

.forbidden-icon {
  color: var(--td-text-color-placeholder);
}

.forbidden-title {
  margin: 12px 0 8px;
  font-size: 20px;
  font-weight: 600;
  color: var(--td-text-color-primary);
}

.forbidden-desc {
  margin: 0;
  font-size: 14px;
  line-height: 1.6;
  color: var(--td-text-color-secondary);
}

.forbidden-caps {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  justify-content: center;
  margin-top: 16px;
}

.forbidden-caps-label {
  font-size: 13px;
  color: var(--td-text-color-secondary);
}
</style>
