<template>
  <div class="forbidden-page">
    <div class="forbidden-card">
      <div class="forbidden-code">403</div>
      <h1 class="forbidden-title">{{ $t('forbidden.title') }}</h1>
      <p class="forbidden-desc">{{ $t('forbidden.description') }}</p>
      <div v-if="requiredCaps.length" class="forbidden-caps">
        <span class="forbidden-caps-label">{{ $t('forbidden.requiredCaps') }}</span>
        <t-tag v-for="cap in requiredCaps" :key="cap" theme="default" variant="light">
          {{ capabilityLabel(cap) }}
        </t-tag>
      </div>
      <div class="forbidden-actions">
        <t-button theme="primary" @click="goHome">{{ $t('forbidden.backHome') }}</t-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { CAPABILITY_LABELS } from '@/config/capabilities'

const route = useRoute()
const router = useRouter()

const requiredCaps = computed(() => {
  const raw = route.query.required
  const value = Array.isArray(raw) ? raw.join(',') : String(raw || '')
  return value.split(',').map((item) => item.trim()).filter(Boolean)
})

const capabilityLabel = (cap: string) => CAPABILITY_LABELS[cap] || cap

const goHome = () => {
  void router.replace('/platform/knowledge-bases')
}
</script>

<style lang="less" scoped>
.forbidden-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 24px;
  background: var(--td-bg-color-page);
}

.forbidden-card {
  width: 100%;
  max-width: 520px;
  padding: 40px;
  text-align: center;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: 12px;
}

.forbidden-code {
  font-size: 56px;
  font-weight: 700;
  line-height: 1;
  color: var(--td-brand-color);
}

.forbidden-title {
  margin: 16px 0 8px;
  font-size: 22px;
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
  margin-top: 20px;
}

.forbidden-caps-label {
  font-size: 13px;
  color: var(--td-text-color-secondary);
}

.forbidden-actions {
  margin-top: 28px;
}
</style>
