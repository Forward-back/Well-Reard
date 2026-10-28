<template>
  <div class="ops-page">
    <header class="ops-page-head">
      <div>
        <h2>{{ $t('ops.accounts.title') }}</h2>
        <p>{{ $t('ops.accounts.description') }}</p>
      </div>
      <CreateUserDialog v-model:visible="createVisible" @announced="onAnnounced">
        <t-button theme="primary">
          <template #icon><t-icon name="user-add" /></template>
          {{ $t('ops.accounts.create') }}
        </t-button>
      </CreateUserDialog>
    </header>

    <div class="ops-section-card">
      <div class="ops-account-toolbar">
        <t-input v-model="query" class="ops-account-search" clearable
          :placeholder="$t('ops.accounts.searchPlaceholder')" @enter="load">
          <template #prefix-icon><t-icon name="search" /></template>
        </t-input>
        <t-button variant="outline" :loading="loading" @click="load">
          {{ $t('ops.accounts.search') }}
        </t-button>
        <span class="ops-account-count">{{ $t('ops.accounts.count', { count: users.length }) }}</span>
      </div>

      <div v-if="loading && !users.length" class="ops-account-empty">
        <t-loading size="small" />
      </div>
      <div v-else-if="!users.length" class="ops-account-empty">
        {{ $t('ops.accounts.empty') }}
      </div>
      <div v-else class="ops-account-table">
        <div class="ops-account-row ops-account-row--head">
          <span>{{ $t('ops.accounts.columns.user') }}</span>
          <span>{{ $t('ops.accounts.columns.status') }}</span>
          <span>{{ $t('ops.accounts.columns.role') }}</span>
          <span>{{ $t('ops.accounts.columns.created') }}</span>
          <span>{{ $t('ops.accounts.columns.actions') }}</span>
        </div>
        <div v-for="user in users" :key="user.id" class="ops-account-row">
          <span class="ops-account-user">
            <strong>{{ user.username }}</strong>
            <small>{{ user.email }}</small>
          </span>
          <span>
            <t-tag :theme="user.is_active ? 'success' : 'default'" variant="light" size="small">
              {{ user.is_active ? $t('ops.accounts.active') : $t('ops.accounts.inactive') }}
            </t-tag>
          </span>
          <span>
            <t-tag v-if="user.is_system_admin" theme="warning" variant="light" size="small">
              {{ $t('ops.accounts.systemAdmin') }}
            </t-tag>
            <span v-else class="ops-account-muted">{{ $t('ops.accounts.member') }}</span>
          </span>
          <span class="ops-account-muted">{{ formatDate(user.created_at) }}</span>
          <span class="ops-account-actions">
            <t-button size="small" variant="text" @click="openReset(user.email)">
              {{ $t('ops.accounts.resetPassword') }}
            </t-button>
            <t-button size="small" variant="text" :theme="user.is_system_admin ? 'danger' : 'warning'"
              @click="toggleSystemAdmin(user)">
              {{ user.is_system_admin ? $t('ops.accounts.revokeAdmin') : $t('ops.accounts.promoteAdmin') }}
            </t-button>
          </span>
        </div>
      </div>
    </div>

    <div class="ops-section-card">
      <header class="ops-panel-head">
        <div>
          <h3>{{ $t('ops.accounts.accessRules.title') }}</h3>
          <p>{{ $t('ops.accounts.accessRules.description') }}</p>
        </div>
      </header>
      <SystemSettings :sections="['access']" hide-header hide-account-actions />
    </div>

    <t-dialog v-model:visible="resetVisible" :header="$t('ops.accounts.resetTitle')"
      :confirm-btn="{ content: $t('ops.accounts.resetConfirm'), theme: 'danger', loading: resetSaving }"
      :cancel-btn="$t('ops.accounts.cancel')" @confirm="submitReset">
      <div class="ops-reset-form">
        <p class="ops-reset-hint">{{ $t('ops.accounts.resetHint', { email: resetEmail }) }}</p>
        <t-input v-model="resetPassword" type="password" autocomplete="new-password"
          :placeholder="$t('ops.accounts.newPasswordPlaceholder')">
          <template #prefix-icon><t-icon name="lock-on" /></template>
        </t-input>
        <t-input v-model="resetConfirm" type="password" autocomplete="new-password"
          :placeholder="$t('ops.accounts.confirmPasswordPlaceholder')">
          <template #prefix-icon><t-icon name="lock-on" /></template>
        </t-input>
      </div>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import CreateUserDialog from '@/views/system/CreateUserDialog.vue'
import SystemSettings from '@/views/system/SystemSettings.vue'
import {
  listSystemUsers,
  promoteUserToSystemAdmin,
  resetUserPassword,
  revokeSystemAdmin,
  type SystemUserSummary,
} from '@/api/system'

const { t } = useI18n()
const users = ref<SystemUserSummary[]>([])
const query = ref('')
const loading = ref(false)
const createVisible = ref(false)
const resetVisible = ref(false)
const resetEmail = ref('')
const resetPassword = ref('')
const resetConfirm = ref('')
const resetSaving = ref(false)

const load = async () => {
  loading.value = true
  try {
    const res = await listSystemUsers({ query: query.value.trim(), limit: 50 })
    users.value = res.users || []
  } catch {
    MessagePlugin.error(t('ops.accounts.loadFailed'))
  } finally {
    loading.value = false
  }
}

const formatDate = (value?: string) => {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleDateString()
}

const onAnnounced = (message: string) => {
  if (message) MessagePlugin.success(message)
  void load()
}

const openReset = (email: string) => {
  resetEmail.value = email
  resetPassword.value = ''
  resetConfirm.value = ''
  resetVisible.value = true
}

const submitReset = async () => {
  if (!resetPassword.value || resetPassword.value !== resetConfirm.value) {
    MessagePlugin.warning(t('ops.accounts.passwordMismatch'))
    return
  }
  resetSaving.value = true
  try {
    await resetUserPassword({ email: resetEmail.value, new_password: resetPassword.value })
    MessagePlugin.success(t('ops.accounts.resetSuccess'))
    resetVisible.value = false
  } catch {
    MessagePlugin.error(t('ops.accounts.resetFailed'))
  } finally {
    resetSaving.value = false
  }
}

const toggleSystemAdmin = (user: SystemUserSummary) => {
  const revoke = user.is_system_admin
  const dialog = DialogPlugin.confirm({
    header: revoke ? t('ops.accounts.revokeTitle') : t('ops.accounts.promoteTitle'),
    body: revoke
      ? t('ops.accounts.revokeBody', { user: user.email })
      : t('ops.accounts.promoteBody', { user: user.email }),
    confirmBtn: {
      content: revoke ? t('ops.accounts.revokeAdmin') : t('ops.accounts.promoteAdmin'),
      theme: revoke ? 'danger' : 'warning',
    },
    cancelBtn: t('ops.accounts.cancel'),
    onConfirm: async () => {
      try {
        if (revoke) await revokeSystemAdmin(user.id)
        else await promoteUserToSystemAdmin({ user_id: user.id })
        MessagePlugin.success(revoke ? t('ops.accounts.revokeSuccess') : t('ops.accounts.promoteSuccess'))
        await load()
      } catch {
        MessagePlugin.error(t('ops.accounts.roleChangeFailed'))
      }
      dialog.destroy()
    },
  })
}

onMounted(() => {
  void load()
})
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
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
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

.ops-account-toolbar {
  display: flex;
  align-items: center;
  gap: 9px;
  margin-bottom: 14px;
}

.ops-account-search {
  max-width: 330px;
}

.ops-account-count {
  margin-left: auto;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.ops-account-table {
  border-top: 1px solid var(--td-component-stroke);
}

.ops-account-row {
  display: grid;
  grid-template-columns: minmax(220px, 1.6fr) 90px 110px 120px minmax(220px, 1fr);
  align-items: center;
  gap: 12px;
  padding: 11px 0;
  border-bottom: 1px solid var(--td-component-stroke);
  font-size: 12px;

  &--head {
    padding: 9px 0;
    color: var(--td-text-color-placeholder);
    font-size: 11px;
  }

  &:last-child {
    border-bottom: 0;
  }
}

.ops-account-user {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;

  strong {
    overflow: hidden;
    color: var(--td-text-color-primary);
    font-weight: 500;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  small {
    overflow: hidden;
    color: var(--td-text-color-placeholder);
    font-size: 11px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.ops-account-muted {
  color: var(--td-text-color-placeholder);
}

.ops-account-actions {
  display: flex;
  justify-content: flex-end;
  gap: 2px;
}

.ops-account-empty {
  display: flex;
  justify-content: center;
  padding: 36px 0;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.ops-panel-head {
  margin-bottom: 8px;

  h3 { margin: 0 0 3px; font-size: 14px; font-weight: 600; }
  p { margin: 0; color: var(--td-text-color-placeholder); font-size: 11px; }
}

.ops-reset-form {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.ops-reset-hint {
  margin: 0 0 2px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 18px;
}

@media (max-width: 900px) {
  .ops-account-row {
    grid-template-columns: minmax(180px, 1fr) 80px 100px;

    > :nth-child(4),
    > :nth-child(5) {
      grid-column: span 3;
    }

    &--head > :nth-child(4),
    &--head > :nth-child(5) {
      display: none;
    }
  }

  .ops-account-actions {
    justify-content: flex-start;
  }
}
</style>
