<template>
  <div class="governance-home">
    <div class="governance-tabs">
      <t-tabs v-model="tab" @change="onTabChange">
        <t-tab-panel value="dashboard" :label="$t('governance.tabs.dashboard')" />
        <t-tab-panel v-if="canAudit" value="audit" :label="$t('governance.tabs.audit')" />
      </t-tabs>
    </div>
    <div v-show="tab === 'dashboard'">
      <GovernanceDashboard :scope="scope" />
    </div>
    <div v-if="tab === 'audit' && canAudit">
      <GovernanceAudit />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import GovernanceDashboard from './GovernanceDashboard.vue'
import GovernanceAudit from './GovernanceAudit.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const scope = computed<'company' | 'dept'>(() =>
  authStore.hasCompanyCaps(['stats:company_read']) ? 'company' : 'dept',
)
const canAudit = computed(() =>
  authStore.hasCompanyCaps(['audit:company_read']) || authStore.hasCompanyCaps(['audit:dept_read']),
)
const initialTab = route.query.tab === 'audit' && canAudit.value ? 'audit' : 'dashboard'
const tab = ref(initialTab)

watch(canAudit, (allowed) => {
  if (!allowed && tab.value === 'audit') tab.value = 'dashboard'
})

const onTabChange = (value: string | number) => {
  void router.replace({ path: '/platform/governance', query: { tab: String(value) } })
}
</script>

<style lang="less" scoped>
.governance-home {
  flex: 1;
  min-height: 0;
  width: 100%;
  overflow-y: auto;
}

.governance-tabs {
  max-width: 1180px;
  margin: 0 auto;
  padding: 14px 24px 0;
}
</style>
