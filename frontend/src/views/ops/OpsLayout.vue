<template>
  <div class="ops-shell">
    <aside class="ops-sidebar">
      <div class="ops-brand">
        <span class="ops-brand-name">WellRead</span>
        <span class="ops-brand-badge">{{ $t('ops.badge') }}</span>
        <span class="ops-env-badge" :class="{ 'is-dev': isDevBuild }">{{ environmentLabel }}</span>
      </div>

      <nav class="ops-nav">
        <button v-for="item in navItems" :key="item.key" type="button"
          :class="['ops-nav-item', { 'is-active': isActive(item) }]" @click="go(item.path)">
          <t-icon :name="item.icon" size="18px" class="ops-nav-icon" />
          <span>{{ $t(item.labelKey) }}</span>
        </button>
      </nav>

      <div class="ops-user-wrap" ref="userMenuRef">
        <button type="button" class="ops-user" @click="toggleUserMenu">
          <span class="ops-avatar">{{ userInitial }}</span>
          <span class="ops-user-meta">
            <span class="ops-user-name">{{ userName || $t('ops.user.fallback') }}</span>
            <span class="ops-user-role">{{ $t('ops.user.role') }}</span>
          </span>
          <t-icon :name="userMenuOpen ? 'chevron-up' : 'chevron-down'" size="14px" />
        </button>

        <Transition name="ops-dropdown">
          <div v-if="userMenuOpen" class="ops-user-menu" @click.stop>
            <button type="button" class="ops-user-menu-item" @click="openSettings">
              <t-icon name="setting" size="16px" />
              <span>{{ $t('ops.user.settings') }}</span>
            </button>
            <button v-if="canSwitchCompany" type="button" class="ops-user-menu-item" @click="switchCompany">
              <t-icon name="swap" size="16px" />
              <span>{{ $t('ops.user.switchCompany') }}</span>
            </button>
            <div class="ops-user-menu-divider" />
            <button type="button" class="ops-user-menu-item is-danger" @click="logout">
              <t-icon name="logout" size="16px" />
              <span>{{ $t('ops.user.logout') }}</span>
            </button>
          </div>
        </Transition>
      </div>
    </aside>

    <section class="ops-main">
      <header class="ops-topbar">
        <div class="ops-topbar-title">
          <h1>{{ pageTitle }}</h1>
          <span class="ops-topbar-subtitle">{{ $t('ops.topbar.subtitle') }}</span>
        </div>
        <div class="ops-topbar-status">
          <span class="ops-status-dot" :class="{ 'is-warn': migrationFailed }" />
          <span class="ops-status-text">
            {{ migrationFailed ? $t('ops.topbar.migrationFailed') : $t('ops.topbar.running') }}
          </span>
          <span v-if="version" class="ops-topbar-chip">v{{ version }}</span>
          <span class="ops-topbar-chip">{{ $t('ops.topbar.uptime', { value: uptimeText }) }}</span>
          <span class="ops-topbar-chip">{{ $t('ops.topbar.healthCheck') }}</span>
          <span class="ops-topbar-refresh">{{ $t('ops.topbar.updatedAt', { time: lastRefresh }) }}</span>
          <t-button size="small" variant="outline" @click="refresh">
            <template #icon><t-icon name="refresh" /></template>
            {{ $t('ops.topbar.refresh') }}
          </t-button>
        </div>
      </header>

      <main class="ops-content">
        <RouterView />
      </main>
    </section>

    <Settings mode="ops" />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import Settings from '@/views/settings/Settings.vue'
import { useAuthStore } from '@/stores/auth'
import { useUIStore } from '@/stores/ui'
import { logout as logoutApi } from '@/api/auth'
import { getSystemInfo, type SystemInfo } from '@/api/system'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const uiStore = useUIStore()
const { t } = useI18n()

const navItems = [
  { key: 'overview', path: '/ops/overview', labelKey: 'ops.nav.overview', icon: 'chart-bar' },
  { key: 'models', path: '/ops/models', labelKey: 'ops.nav.models', icon: 'control-platform' },
  { key: 'integrations', path: '/ops/integrations', labelKey: 'ops.nav.integrations', icon: 'secured' },
  { key: 'runtime', path: '/ops/runtime', labelKey: 'ops.nav.runtime', icon: 'queue' },
  { key: 'security', path: '/ops/security', labelKey: 'ops.nav.security', icon: 'lock-on' },
  { key: 'accounts', path: '/ops/accounts', labelKey: 'ops.nav.accounts', icon: 'usergroup' },
  { key: 'settings', path: '/ops/settings', labelKey: 'ops.nav.settings', icon: 'setting' },
]

const pageTitles: Record<string, string> = {
  opsOverview: 'ops.nav.overview',
  opsModels: 'ops.nav.models',
  opsIntegrations: 'ops.nav.integrations',
  opsRuntime: 'ops.nav.runtime',
  opsSecurity: 'ops.nav.security',
  opsAccounts: 'ops.nav.accounts',
  opsSettings: 'ops.nav.settings',
}

const pageTitle = computed(() => {
  const key = pageTitles[String(route.name || '')]
  return key ? t(key) : t('ops.title')
})

const systemInfo = ref<SystemInfo | null>(null)
const lastRefresh = ref('')
const userMenuOpen = ref(false)
const userMenuRef = ref<HTMLElement | null>(null)
let refreshTimer: ReturnType<typeof setInterval> | undefined

const version = computed(() => {
  const raw = (systemInfo.value?.version || '').trim()
  return raw && raw !== 'unknown' ? raw : 'dev'
})
const isDevBuild = computed(() => version.value === 'dev')
const environmentLabel = computed(() => isDevBuild.value ? 'DEV' : 'PROD')
const migrationFailed = computed(() => Boolean(systemInfo.value?.db_migration_error))
const uptimeText = computed(() => {
  const seconds = systemInfo.value?.uptime_seconds
  if (!seconds || seconds < 0) return '-'
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (days > 0) return t('ops.topbar.uptimeDays', { days, hours })
  if (hours > 0) return t('ops.topbar.uptimeHours', { hours, minutes })
  return t('ops.topbar.uptimeMinutes', { minutes })
})
const userName = computed(() => authStore.user?.username || '')
const userInitial = computed(() => (userName.value || 'S').charAt(0).toUpperCase())
const canSwitchCompany = computed(() => (authStore.memberships || []).length > 0)

const isActive = (item: { path: string }) => route.path === item.path || route.path.startsWith(`${item.path}/`)

const go = (path: string) => {
  userMenuOpen.value = false
  if (route.path !== path) router.push(path)
}

const toggleUserMenu = () => {
  userMenuOpen.value = !userMenuOpen.value
}

const openSettings = () => {
  userMenuOpen.value = false
  uiStore.openSettings('general')
}

const switchCompany = () => {
  userMenuOpen.value = false
  router.push('/platform/knowledge-bases')
}

const logout = async () => {
  userMenuOpen.value = false
  try {
    await logoutApi()
  } catch {
    // Local cleanup below is still correct when the session is already gone.
  }
  authStore.logout()
  MessagePlugin.success(t('ops.user.logoutSuccess'))
  router.push('/login')
}

const refresh = () => {
  window.dispatchEvent(new CustomEvent('ops:refresh'))
}

const loadStatus = async () => {
  try {
    const res = await getSystemInfo()
    systemInfo.value = res.data || null
    lastRefresh.value = new Date().toLocaleTimeString()
  } catch {
    // Keep the last known status; page-level data still renders.
  }
}

const handleClickOutside = (event: MouseEvent) => {
  if (userMenuRef.value && !userMenuRef.value.contains(event.target as Node)) {
    userMenuOpen.value = false
  }
}

onMounted(() => {
  void loadStatus()
  refreshTimer = setInterval(() => { void loadStatus() }, 60000)
  document.addEventListener('click', handleClickOutside)
})

onBeforeUnmount(() => {
  if (refreshTimer) clearInterval(refreshTimer)
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style lang="less" scoped>
.ops-shell {
  display: flex;
  width: 100%;
  height: 100%;
  min-height: 100vh;
  background: var(--td-bg-color-page);
  color: var(--td-text-color-primary);
}

.ops-sidebar {
  display: flex;
  flex: 0 0 232px;
  flex-direction: column;
  width: 232px;
  padding: 18px 12px 12px;
  box-sizing: border-box;
  background: var(--td-bg-color-sidebar);
  border-right: 1px solid var(--td-component-stroke);
}

.ops-brand {
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding: 0 10px 20px;

  &-name {
    font-family: "WellReadSerif", Georgia, serif;
    font-size: 24px;
    color: var(--td-text-color-primary);
  }

  &-badge {
    padding: 2px 6px;
    border-radius: 4px;
    font-size: 10px;
    font-weight: 600;
    color: var(--td-brand-color);
    background: var(--td-brand-color-light);
  }
}

.ops-env-badge {
  padding: 2px 6px;
  border-radius: 4px;
  color: var(--td-text-color-secondary);
  background: var(--td-bg-color-secondarycontainer);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.04em;

  &.is-dev {
    color: #92400e;
    background: #fef3c7;
  }
}

.ops-nav {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 3px;
  min-height: 0;
}

.ops-nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 9px 10px;
  border: 0;
  border-radius: 6px;
  color: var(--td-text-color-secondary);
  background: transparent;
  font-family: var(--app-font-family);
  font-size: 14px;
  text-align: left;
  cursor: pointer;
  transition: background-color 0.16s ease, color 0.16s ease;

  &:hover {
    color: var(--td-text-color-primary);
    background: var(--td-bg-color-container-hover);
  }

  &.is-active {
    color: var(--td-brand-color);
    background: var(--td-brand-color-light);
    font-weight: 600;
  }
}

.ops-nav-icon {
  flex-shrink: 0;
}

.ops-user-wrap {
  position: relative;
  padding-top: 12px;
  border-top: 1px solid var(--td-component-stroke);
}

.ops-user {
  display: flex;
  align-items: center;
  gap: 9px;
  width: 100%;
  padding: 7px 8px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--td-text-color-primary);
  text-align: left;
  cursor: pointer;

  &:hover {
    background: var(--td-bg-color-container-hover);
  }
}

.ops-avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 28px;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  color: var(--td-brand-color);
  background: var(--td-brand-color-light);
  font-size: 12px;
  font-weight: 700;
}

.ops-user-meta {
  display: flex;
  flex: 1;
  min-width: 0;
  flex-direction: column;
  gap: 1px;
}

.ops-user-name {
  overflow: hidden;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-user-role {
  color: var(--td-text-color-placeholder);
  font-size: 11px;
}

.ops-user-menu {
  position: absolute;
  right: 0;
  bottom: calc(100% + 8px);
  left: 0;
  z-index: 20;
  padding: 6px;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  box-shadow: var(--td-shadow-2);
}

.ops-user-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px 9px;
  border: 0;
  border-radius: 5px;
  color: var(--td-text-color-primary);
  background: transparent;
  font-size: 13px;
  text-align: left;
  cursor: pointer;

  &:hover {
    background: var(--td-bg-color-container-hover);
  }

  &.is-danger {
    color: var(--td-error-color);
  }
}

.ops-user-menu-divider {
  height: 1px;
  margin: 5px 4px;
  background: var(--td-component-stroke);
}

.ops-dropdown-enter-active,
.ops-dropdown-leave-active {
  transition: opacity 0.12s ease, transform 0.12s ease;
}

.ops-dropdown-enter-from,
.ops-dropdown-leave-to {
  opacity: 0;
  transform: translateY(4px);
}

.ops-main {
  display: flex;
  flex: 1;
  min-width: 0;
  flex-direction: column;
}

.ops-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  min-height: 66px;
  padding: 0 28px;
  background: var(--td-bg-color-container);
  border-bottom: 1px solid var(--td-component-stroke);
}

.ops-topbar-title {
  display: flex;
  align-items: baseline;
  gap: 10px;
  min-width: 0;

  h1 {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
    white-space: nowrap;
  }
}

.ops-topbar-subtitle {
  overflow: hidden;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-topbar-status {
  display: flex;
  align-items: center;
  gap: 9px;
  flex-wrap: wrap;
  justify-content: flex-end;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.ops-status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--td-success-color);

  &.is-warn {
    background: var(--td-error-color);
  }
}

.ops-status-text {
  margin-right: 3px;
  color: var(--td-text-color-primary);
}

.ops-topbar-chip {
  padding: 3px 7px;
  border-radius: 4px;
  color: var(--td-text-color-secondary);
  background: var(--td-bg-color-secondarycontainer);
}

.ops-topbar-refresh {
  color: var(--td-text-color-placeholder);
}

.ops-content {
  flex: 1;
  min-height: 0;
  overflow: auto;
}

@media (max-width: 900px) {
  .ops-sidebar {
    flex-basis: 64px;
    width: 64px;
    padding-right: 6px;
    padding-left: 6px;
  }

  .ops-brand-name,
  .ops-brand-badge,
  .ops-nav-item span,
  .ops-user-meta {
    display: none;
  }

  .ops-brand {
    justify-content: center;
    padding-right: 0;
    padding-left: 0;
  }

  .ops-nav-item {
    justify-content: center;
  }

  .ops-user {
    justify-content: center;
  }

  .ops-topbar {
    align-items: flex-start;
    flex-direction: column;
    gap: 8px;
    padding: 12px 18px;
  }

  .ops-topbar-status {
    justify-content: flex-start;
  }
}
</style>
