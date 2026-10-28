<template>
  <div class="management-page">
    <header class="management-header">
      <h1>{{ $t('management.title') }}</h1>
      <p>{{ $t('management.description') }}</p>
    </header>

    <div class="management-todo" :class="{ 'is-allok': pendingInvitations === 0 }">
      <span class="management-todo-title">
        <t-icon :name="pendingInvitations ? 'notification' : 'check-circle'" size="16px" />
        {{ pendingInvitations ? $t('management.todo.pending', { count: pendingInvitations }) : $t('management.todo.allOk') }}
      </span>
      <button type="button" class="management-todo-go" @click="openMembersQuick">
        {{ $t('management.todo.openMembers') }}
      </button>
    </div>

    <div v-if="cards.length" class="management-layout">
      <div>
        <h2 class="management-group-title">{{ $t('management.group.organization') }}</h2>
        <div class="management-grid">
          <button v-for="card in cards" :key="card.key" type="button" class="management-card"
            @click="openCard(card)">
            <span class="management-card-icon">
              <t-icon :name="card.icon" size="22px" />
            </span>
            <span class="management-card-copy">
              <span class="management-card-title">{{ $t(card.titleKey) }}</span>
              <span class="management-card-desc">{{ $t(card.descriptionKey) }}</span>
            </span>
            <t-icon name="arrow-right" class="management-card-arrow" />
          </button>
        </div>
      </div>

      <aside class="management-side">
        <h2 class="management-group-title">{{ $t('management.group.actions') }}</h2>
        <button type="button" class="management-quick" @click="openMembersQuick">
          <t-icon name="user-add" size="18px" />
          <span>{{ $t('management.quick.invite') }}</span>
        </button>
        <button v-if="authStore.isCompanyAdmin" type="button" class="management-quick" @click="openSection('departments')">
          <t-icon name="system-sum" size="18px" />
          <span>{{ $t('management.quick.department') }}</span>
        </button>
        <button type="button" class="management-quick" @click="router.push({ path: '/platform/governance', query: { tab: 'audit' } })">
          <t-icon name="history" size="18px" />
          <span>{{ $t('management.quick.audit') }}</span>
        </button>
      </aside>
    </div>

    <ForbiddenState v-else :required-caps="[...MANAGEMENT_ENTRY_CAPS]" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import ForbiddenState from '@/components/ForbiddenState.vue'
import { useAuthStore } from '@/stores/auth'
import { listTenantInvitations } from '@/api/tenant/invitations'
import {
  MANAGEMENT_ENTRY_CAPS,
  visibleManagementCards,
  type ManagementCardDefinition,
} from '@/config/managementCards'

const router = useRouter()
const authStore = useAuthStore()
const pendingInvitations = ref(0)

const cards = computed(() => visibleManagementCards(authStore.companyCaps))

const openSection = (section: string) => {
  router.push({ path: '/platform/settings', query: { section } })
}

const openMembersQuick = () => {
  if (authStore.isCompanyAdmin) {
    openSection('members')
    return
  }
  router.push('/platform/management/members')
}

const loadPendingInvitations = async () => {
  const tenantID = Number(authStore.effectiveTenantId || 0)
  if (!tenantID || !authStore.hasCompanyCaps(['member:manage'])) return
  try {
    const res = await listTenantInvitations(tenantID, { page: 1, page_size: 20 })
    pendingInvitations.value = Number(res.data?.total || 0)
  } catch {
    pendingInvitations.value = 0
  }
}

onMounted(() => {
  void loadPendingInvitations()
})

const openCard = (card: ManagementCardDefinition) => {
  if (card.key === 'members' && !authStore.isCompanyAdmin) {
    router.push('/platform/management/members')
    return
  }
  if (card.target.kind === 'governance') {
    router.push(authStore.hasCompanyCaps(['stats:company_read'])
      ? '/platform/governance/company'
      : '/platform/governance/dept')
    return
  }
  if (card.target.kind === 'audit') {
    router.push('/platform/governance/audit')
    return
  }
  if (card.target.kind === 'auditors') {
    router.push('/platform/management/auditors')
    return
  }
  if (card.target.kind === 'kbSettings') {
    router.push('/platform/management/knowledge-bases')
    return
  }
  router.push({ path: '/platform/settings', query: { section: card.target.section } })
}
</script>

<style lang="less" scoped>
.management-page {
  width: 100%;
  max-width: 1180px;
  margin: 0 auto;
  padding: 28px 24px 48px;
  box-sizing: border-box;
}

.management-header {
  margin-bottom: 22px;

  h1 {
    margin: 0 0 6px;
    font-size: 22px;
    font-weight: 600;
    color: var(--td-text-color-primary);
  }

  p {
    margin: 0;
    font-size: 13px;
    color: var(--td-text-color-secondary);
  }
}

.management-todo {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 15px;
  margin-bottom: 18px;
  border: 1px solid #fed7aa;
  border-radius: 10px;
  background: #fff7ed;

  &.is-allok {
    border-color: #bbf7d0;
    background: #f0fdf4;
  }
}

.management-todo-title {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  color: #9a3412;
  font-size: 13px;
  font-weight: 600;

  .is-allok & { color: #15803d; }
}

.management-todo-go {
  margin-left: auto;
  padding: 0;
  border: 0;
  color: var(--td-brand-color);
  background: transparent;
  font-size: 12px;
  cursor: pointer;
}

.management-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.8fr) minmax(230px, 0.8fr);
  gap: 18px;
  align-items: start;
}

.management-group-title {
  margin: 0 0 10px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.04em;
}

.management-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
}

.management-side {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.management-quick {
  display: flex;
  align-items: center;
  gap: 9px;
  width: 100%;
  padding: 11px 13px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 9px;
  color: var(--td-text-color-primary);
  background: var(--td-bg-color-container);
  font-size: 12px;
  text-align: left;
  cursor: pointer;

  &:hover {
    color: var(--td-brand-color);
    border-color: var(--td-brand-color);
    background: var(--td-brand-color-light);
  }
}

@media (max-width: 900px) {
  .management-layout {
    grid-template-columns: minmax(0, 1fr);
  }
}

.management-card {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  padding: 18px;
  text-align: left;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: 10px;
  cursor: pointer;
  transition: border-color 0.2s ease, box-shadow 0.2s ease, transform 0.2s ease;

  &:hover {
    border-color: var(--td-brand-color-light-active);
    box-shadow: var(--td-shadow-1);
    transform: translateY(-1px);

    .management-card-arrow {
      color: var(--td-brand-color);
      transform: translateX(2px);
    }
  }

  &:focus-visible {
    outline: 2px solid var(--td-brand-color);
    outline-offset: 2px;
  }
}

.management-card-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 40px;
  width: 40px;
  height: 40px;
  border-radius: 10px;
  color: var(--td-brand-color);
  background: var(--td-brand-color-light);
}

.management-card-copy {
  display: flex;
  flex: 1;
  min-width: 0;
  flex-direction: column;
  gap: 4px;
}

.management-card-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--td-text-color-primary);
}

.management-card-desc {
  font-size: 12px;
  line-height: 18px;
  color: var(--td-text-color-secondary);
}

.management-card-arrow {
  flex-shrink: 0;
  color: var(--td-text-color-placeholder);
  transition: color 0.2s ease, transform 0.2s ease;
}
</style>
