<template>
  <div class="dept-members-page">
    <header class="page-head">
      <div>
        <h1>{{ $t('management.members.title') }}</h1>
        <p>{{ $t('management.members.description', { dept: departmentName || '-' }) }}</p>
      </div>
      <t-button theme="primary" :disabled="!departmentId" @click="openInvite">
        <template #icon><t-icon name="user-add" /></template>
        {{ $t('management.members.invite') }}
      </t-button>
    </header>

    <div v-if="!departmentId" class="page-empty">{{ $t('management.members.noDepartment') }}</div>
    <template v-else>
      <t-tabs v-model="tab">
        <t-tab-panel value="members" :label="$t('management.members.tabs.members', { count: members.length })" />
        <t-tab-panel value="invitations" :label="$t('management.members.tabs.invitations', { count: invitations.length })" />
      </t-tabs>

      <div v-if="tab === 'members'" class="table-card">
        <div v-if="!members.length" class="page-empty">{{ $t('management.members.emptyMembers') }}</div>
        <div v-for="member in members" v-else :key="member.user_id" class="member-row">
          <span class="member-main">
            <strong>{{ member.username || member.email }}</strong>
            <small>{{ member.email }}</small>
          </span>
          <t-select class="role-select" :value="member.department_role || 'viewer'" :options="roleOptions"
            size="small" @change="(value: string | number) => changeRole(member, String(value))" />
          <span class="member-status">{{ member.status }}</span>
          <t-button size="small" variant="text" theme="danger" @click="removeMember(member)">
            {{ $t('management.members.remove') }}
          </t-button>
        </div>
      </div>

      <div v-else class="table-card">
        <div v-if="!invitations.length" class="page-empty">{{ $t('management.members.emptyInvitations') }}</div>
        <div v-for="invitation in invitations" v-else :key="invitation.id" class="member-row">
          <span class="member-main">
            <strong>{{ invitation.invitee_name || invitation.invitee_email || '-' }}</strong>
            <small>{{ invitation.invitee_email || '' }}</small>
          </span>
          <span class="member-role">{{ roleLabel(invitation.department_role) }}</span>
          <span class="member-status">{{ invitation.status }}</span>
          <t-button size="small" variant="text" theme="danger" @click="revoke(invitation)">
            {{ $t('management.members.revoke') }}
          </t-button>
        </div>
      </div>
    </template>

    <t-dialog v-model:visible="inviteVisible" :header="$t('management.members.inviteTitle')"
      :confirm-btn="{ content: $t('management.members.inviteConfirm'), loading: inviting }"
      :cancel-btn="$t('management.members.cancel')" @confirm="submitInvite">
      <div class="invite-form">
        <t-input v-model="inviteEmail" :placeholder="$t('management.members.emailPlaceholder')">
          <template #prefix-icon><t-icon name="mail" /></template>
        </t-input>
        <t-select v-model="inviteRole" :options="roleOptions" :placeholder="$t('management.members.rolePlaceholder')" />
      </div>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import {
  assignDepartmentMember,
  listDepartmentMembers,
  listDepartments,
  removeDepartmentMember,
  type Department,
} from '@/api/department'
import { createInvitation, listTenantInvitations, revokeInvitation, type TenantInvitation } from '@/api/tenant/invitations'
import type { DepartmentRole, TenantMember } from '@/api/tenant/members'

const { t } = useI18n()
const authStore = useAuthStore()
const tab = ref('members')
const members = ref<TenantMember[]>([])
const invitations = ref<TenantInvitation[]>([])
const departments = ref<Department[]>([])
const inviteVisible = ref(false)
const inviteEmail = ref('')
const inviteRole = ref<DepartmentRole>('viewer')
const inviting = ref(false)

const departmentId = computed(() => authStore.currentDepartmentId || '')
const departmentName = computed(() => departments.value.find((dept) => dept.id === departmentId.value)?.name || '')
const roleOptions = computed(() => [
  { value: 'manager', label: 'manager' },
  { value: 'knowledge_admin', label: 'knowledge_admin' },
  { value: 'editor', label: 'editor' },
  { value: 'viewer', label: 'viewer' },
])

const roleLabel = (role?: DepartmentRole) => role || 'viewer'
const tenantID = () => Number(authStore.effectiveTenantId || 0)

const loadDepartments = async () => {
  if (!tenantID()) return
  try {
    const res = await listDepartments(tenantID())
    departments.value = res.data || []
  } catch {
    departments.value = []
  }
}

const loadMembers = async () => {
  if (!tenantID() || !departmentId.value) return
  try {
    const res = await listDepartmentMembers(tenantID(), departmentId.value)
    members.value = res.data || []
  } catch {
    members.value = []
  }
}

const loadInvitations = async () => {
  if (!tenantID()) return
  try {
    const res = await listTenantInvitations(tenantID(), { page: 1, page_size: 100 })
    invitations.value = (res.data?.invitations || []).filter(
      (inv) => inv.department_id === departmentId.value,
    )
  } catch {
    invitations.value = []
  }
}

const refresh = async () => {
  await Promise.all([loadMembers(), loadInvitations()])
}

const openInvite = () => {
  inviteEmail.value = ''
  inviteRole.value = 'viewer'
  inviteVisible.value = true
}

const submitInvite = async () => {
  if (!inviteEmail.value.trim() || !departmentId.value) return
  inviting.value = true
  try {
    await createInvitation(tenantID(), {
      email: inviteEmail.value.trim(),
      role: 'member',
      department_id: departmentId.value,
      department_role: inviteRole.value,
    })
    MessagePlugin.success(t('management.members.inviteSuccess'))
    inviteVisible.value = false
    await refresh()
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('management.members.inviteFailed'))
  } finally {
    inviting.value = false
  }
}

const changeRole = async (member: TenantMember, role: string) => {
  try {
    await assignDepartmentMember(tenantID(), departmentId.value, member.user_id, role as DepartmentRole)
    MessagePlugin.success(t('management.members.roleUpdated'))
    await loadMembers()
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('management.members.roleUpdateFailed'))
  }
}

const removeMember = (member: TenantMember) => {
  const dialog = DialogPlugin.confirm({
    header: t('management.members.removeTitle'),
    body: t('management.members.removeBody', { name: member.username || member.email }),
    confirmBtn: { content: t('management.members.remove'), theme: 'danger' },
    cancelBtn: t('management.members.cancel'),
    onConfirm: async () => {
      try {
        await removeDepartmentMember(tenantID(), departmentId.value, member.user_id)
        MessagePlugin.success(t('management.members.removeSuccess'))
        await refresh()
      } catch (err: any) {
        MessagePlugin.error(err?.message || t('management.members.removeFailed'))
      }
      dialog.destroy()
    },
  })
}

const revoke = async (invitation: TenantInvitation) => {
  try {
    await revokeInvitation(tenantID(), invitation.id)
    MessagePlugin.success(t('management.members.revokeSuccess'))
    await loadInvitations()
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('management.members.revokeFailed'))
  }
}

onMounted(async () => {
  await loadDepartments()
  await refresh()
})
</script>

<style lang="less" scoped>
.dept-members-page {
  width: 100%;
  max-width: 1080px;
  margin: 0 auto;
  padding: 24px 24px 48px;
  box-sizing: border-box;
}

.page-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;

  h1 { margin: 0 0 5px; font-size: 20px; font-weight: 600; }
  p { margin: 0; color: var(--td-text-color-secondary); font-size: 12px; }
}

.table-card {
  margin-top: 12px;
  padding: 4px 16px;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: 10px;
}

.member-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 0;
  border-bottom: 1px solid var(--td-component-stroke);

  &:last-child { border-bottom: 0; }
}

.member-main {
  display: flex;
  flex: 1;
  min-width: 0;
  flex-direction: column;
  gap: 2px;

  strong { overflow: hidden; font-size: 13px; text-overflow: ellipsis; white-space: nowrap; }
  small { overflow: hidden; color: var(--td-text-color-placeholder); font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }
}

.role-select {
  width: 150px;
}

.member-role,
.member-status {
  width: 90px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.member-status {
  width: 70px;
  color: var(--td-text-color-placeholder);
}

.page-empty {
  padding: 32px;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  text-align: center;
}

.invite-form {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
</style>
