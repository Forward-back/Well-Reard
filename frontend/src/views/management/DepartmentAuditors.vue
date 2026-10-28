<template>
  <div class="auditors-page">
    <header class="page-head">
      <div>
        <h1>{{ $t('management.auditors.title') }}</h1>
        <p>{{ $t('management.auditors.description', { dept: departmentName || '-' }) }}</p>
      </div>
    </header>

    <div v-if="!departmentId" class="page-empty">{{ $t('management.auditors.noDepartment') }}</div>
    <div v-else class="table-card">
      <div v-if="!members.length" class="page-empty">{{ $t('management.auditors.empty') }}</div>
      <div v-for="member in members" v-else :key="member.user_id" class="auditor-row">
        <span class="member-main">
          <strong>{{ member.username || member.email }}</strong>
          <small>{{ member.email }}</small>
        </span>
        <span class="member-role">{{ member.department_role || 'viewer' }}</span>
        <t-switch :value="auditorIds.has(member.user_id)" :loading="savingId === member.user_id"
          @change="(value: string | number | boolean) => toggle(member, Boolean(value))" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import {
  addDepartmentCRAuditor,
  listDepartmentCRAuditors,
  listDepartmentMembers,
  listDepartments,
  removeDepartmentCRAuditor,
  type Department,
} from '@/api/department'
import type { TenantMember } from '@/api/tenant/members'

const { t } = useI18n()
const authStore = useAuthStore()
const members = ref<TenantMember[]>([])
const auditorIds = ref(new Set<string>())
const departments = ref<Department[]>([])
const savingId = ref('')

const departmentId = computed(() => authStore.currentDepartmentId || '')
const departmentName = computed(() => departments.value.find((dept) => dept.id === departmentId.value)?.name || '')
const tenantID = () => Number(authStore.effectiveTenantId || 0)

const load = async () => {
  if (!tenantID() || !departmentId.value) return
  try {
    const [deptRes, memberRes, auditorRes] = await Promise.all([
      listDepartments(tenantID()),
      listDepartmentMembers(tenantID(), departmentId.value),
      listDepartmentCRAuditors(tenantID(), departmentId.value),
    ])
    departments.value = deptRes.data || []
    members.value = memberRes.data || []
    auditorIds.value = new Set(auditorRes.data?.user_ids || [])
  } catch {
    members.value = []
    auditorIds.value = new Set()
  }
}

const toggle = async (member: TenantMember, enabled: boolean) => {
  savingId.value = member.user_id
  try {
    if (enabled) {
      await addDepartmentCRAuditor(tenantID(), departmentId.value, member.user_id)
      auditorIds.value = new Set([...auditorIds.value, member.user_id])
      MessagePlugin.success(t('management.auditors.addSuccess'))
    } else {
      await removeDepartmentCRAuditor(tenantID(), departmentId.value, member.user_id)
      const next = new Set(auditorIds.value)
      next.delete(member.user_id)
      auditorIds.value = next
      MessagePlugin.success(t('management.auditors.removeSuccess'))
    }
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('management.auditors.updateFailed'))
  } finally {
    savingId.value = ''
  }
}

onMounted(load)
</script>

<style lang="less" scoped>
.auditors-page {
  width: 100%;
  max-width: 980px;
  margin: 0 auto;
  padding: 24px 24px 48px;
  box-sizing: border-box;
}

.page-head {
  margin-bottom: 16px;

  h1 { margin: 0 0 5px; font-size: 20px; font-weight: 600; }
  p { margin: 0; color: var(--td-text-color-secondary); font-size: 12px; line-height: 18px; }
}

.table-card {
  padding: 4px 16px;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: 10px;
}

.auditor-row {
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

.member-role {
  width: 120px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.page-empty {
  padding: 32px;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  text-align: center;
}
</style>
