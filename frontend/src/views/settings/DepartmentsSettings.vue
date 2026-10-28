<template>
  <div class="departments-settings">
    <div class="section-header">
      <h2>{{ $t('departments.title') }}</h2>
      <p class="section-description">{{ $t('departments.description') }}</p>
    </div>

    <div class="settings-actions">
      <t-button v-if="isCompanyAdmin" theme="primary" @click="openCreate">
        <template #icon><t-icon name="add" /></template>
        {{ $t('departments.create') }}
      </t-button>
    </div>

    <t-table row-key="id" :data="departments" :columns="columns" :loading="loading" hover stripe>
      <template #name="{ row }">
        <div class="dept-name-cell">
          <span class="dept-name">{{ row.name }}</span>
          <t-tag v-if="row.is_default" size="small" variant="light">{{ $t('departments.defaultBadge') }}</t-tag>
        </div>
      </template>
      <template #member_count="{ row }">{{ row.member_count ?? 0 }}</template>
      <template #knowledge_base_count="{ row }">{{ row.knowledge_base_count ?? 0 }}</template>
      <template #operations="{ row }">
        <div class="row-actions">
          <t-button size="small" variant="text" theme="primary" @click="openMembers(row)">
            {{ $t('departments.members') }}
          </t-button>
          <t-button v-if="isCompanyAdmin" size="small" variant="text" @click="openEdit(row)">
            {{ $t('common.edit') }}
          </t-button>
          <t-popconfirm
            v-if="isCompanyAdmin && !row.is_default"
            :content="$t('departments.deleteConfirm', { name: row.name })"
            :confirm-btn="{ content: $t('common.confirm'), theme: 'danger' }"
            :cancel-btn="{ content: $t('common.cancel') }"
            @confirm="removeDepartment(row)"
          >
            <t-button size="small" variant="text" theme="danger">
              {{ $t('common.delete') }}
            </t-button>
          </t-popconfirm>
        </div>
      </template>
    </t-table>

    <t-dialog
      v-model:visible="editorVisible"
      :header="editingId ? $t('departments.editTitle') : $t('departments.createTitle')"
      :confirm-btn="{ content: $t('common.save'), theme: 'primary', loading: saving }"
      :cancel-btn="{ content: $t('common.cancel') }"
      @confirm="saveDepartment"
    >
      <t-form label-align="top">
        <t-form-item :label="$t('departments.nameLabel')">
          <t-input v-model="form.name" :maxlength="128" />
        </t-form-item>
        <t-form-item :label="$t('departments.descriptionLabel')">
          <t-textarea v-model="form.description" :autosize="{ minRows: 2, maxRows: 4 }" />
        </t-form-item>
      </t-form>
    </t-dialog>

    <t-dialog
      v-model:visible="membersVisible"
      :header="$t('departments.membersTitle', { name: activeDepartment?.name || '' })"
      :footer="false"
      width="720px"
    >
      <t-table row-key="user_id" :data="members" :columns="memberColumns" :loading="membersLoading" size="small">
        <template #member="{ row }">
          <div class="member-cell">
            <span class="member-name">{{ row.username || row.email }}</span>
            <span v-if="row.username && row.email" class="member-email">{{ row.email }}</span>
          </div>
        </template>
        <template #department_role="{ row }">
          <t-select
            v-if="canManageActiveMembers"
            :value="row.department_role"
            size="small"
            :options="roleOptions"
            @change="(val: string) => changeMemberRole(row, val)"
          />
          <t-tag v-else size="small" variant="light">{{ roleLabel(row.department_role) }}</t-tag>
        </template>
      </t-table>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import {
  assignDepartmentMember,
  createDepartment,
  deleteDepartment,
  listDepartmentMembers,
  listDepartments,
  updateDepartment,
  type Department,
} from '@/api/department'
import type { DepartmentRole, TenantMember } from '@/api/tenant/members'

const { t } = useI18n()
const authStore = useAuthStore()

const departments = ref<Department[]>([])
const loading = ref(false)
const saving = ref(false)
const editorVisible = ref(false)
const editingId = ref('')
const form = reactive({ name: '', description: '' })

const membersVisible = ref(false)
const membersLoading = ref(false)
const activeDepartment = ref<Department | null>(null)
const members = ref<TenantMember[]>([])

const isCompanyAdmin = computed(() => authStore.isCompanyAdmin)
const canManageActiveMembers = computed(() => {
  if (isCompanyAdmin.value) return true
  return !!activeDepartment.value &&
    activeDepartment.value.id === authStore.currentDepartmentId &&
    authStore.hasDepartmentRole('manager')
})

const roleOptions = computed(() => [
  { value: 'manager', label: t('departments.roles.manager') },
  { value: 'knowledge_admin', label: t('departments.roles.knowledgeAdmin') },
  { value: 'editor', label: t('departments.roles.editor') },
  { value: 'viewer', label: t('departments.roles.viewer') },
])

const columns = computed(() => [
  { colKey: 'name', title: t('departments.columns.name'), ellipsis: true },
  { colKey: 'description', title: t('departments.columns.description'), ellipsis: true },
  { colKey: 'member_count', title: t('departments.columns.members'), width: 100 },
  { colKey: 'knowledge_base_count', title: t('departments.columns.knowledgeBases'), width: 120 },
  { colKey: 'operations', title: t('departments.columns.operations'), width: 220 },
])

const memberColumns = computed(() => [
  { colKey: 'member', title: t('departments.columns.member'), ellipsis: true },
  { colKey: 'department_role', title: t('departments.columns.departmentRole'), width: 180 },
])

const tenantId = computed(() => Number(authStore.effectiveTenantId || 0))

const load = async () => {
  if (!tenantId.value) return
  loading.value = true
  try {
    const res = await listDepartments(tenantId.value)
    departments.value = Array.isArray(res.data) ? res.data : []
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('departments.loadFailed'))
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  editingId.value = ''
  form.name = ''
  form.description = ''
  editorVisible.value = true
}

const openEdit = (dept: Department) => {
  editingId.value = dept.id
  form.name = dept.name
  form.description = dept.description || ''
  editorVisible.value = true
}

const saveDepartment = async () => {
  if (!form.name.trim()) {
    MessagePlugin.warning(t('departments.nameRequired'))
    return
  }
  saving.value = true
  try {
    const body = { name: form.name.trim(), description: form.description.trim() }
    const res = editingId.value
      ? await updateDepartment(tenantId.value, editingId.value, body)
      : await createDepartment(tenantId.value, body)
    if (res.success === false) throw new Error(res.message || t('departments.saveFailed'))
    MessagePlugin.success(editingId.value ? t('departments.updateSuccess') : t('departments.createSuccess'))
    editorVisible.value = false
    await load()
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('departments.saveFailed'))
  } finally {
    saving.value = false
  }
}

const removeDepartment = async (dept: Department) => {
  try {
    const res = await deleteDepartment(tenantId.value, dept.id)
    if (res.success === false) throw new Error(res.message || t('departments.deleteFailed'))
    MessagePlugin.success(t('departments.deleteSuccess'))
    await load()
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('departments.deleteFailed'))
  }
}

const openMembers = async (dept: Department) => {
  activeDepartment.value = dept
  membersVisible.value = true
  membersLoading.value = true
  try {
    const res = await listDepartmentMembers(tenantId.value, dept.id)
    members.value = Array.isArray(res.data) ? res.data : []
  } catch (err: any) {
    members.value = []
    MessagePlugin.error(err?.message || t('departments.membersLoadFailed'))
  } finally {
    membersLoading.value = false
  }
}

const roleLabel = (role?: DepartmentRole) => {
  return roleOptions.value.find(opt => opt.value === role)?.label || role || ''
}

const changeMemberRole = async (member: TenantMember, role: string) => {
  if (!activeDepartment.value || !role) return
  try {
    const res = await assignDepartmentMember(
      tenantId.value,
      activeDepartment.value.id,
      member.user_id,
      role as DepartmentRole,
    )
    if (res.success === false) throw new Error(res.message || t('departments.roleSaveFailed'))
    member.department_role = role as DepartmentRole
    MessagePlugin.success(t('departments.roleSaveSuccess'))
    await load()
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('departments.roleSaveFailed'))
  }
}

onMounted(load)
</script>

<style lang="less" scoped>
.departments-settings {
  width: 100%;
}

.section-header {
  margin-bottom: 20px;

  h2 {
    margin: 0 0 8px;
    font-size: 20px;
    font-weight: 600;
    color: var(--td-text-color-primary);
  }

  .section-description {
    margin: 0;
    font-size: 14px;
    line-height: 1.5;
    color: var(--td-text-color-secondary);
  }
}

.settings-actions {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 16px;
}

.dept-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.dept-name {
  font-weight: 500;
  color: var(--td-text-color-primary);
}

.row-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.member-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.member-name {
  font-weight: 500;
  color: var(--td-text-color-primary);
}

.member-email {
  font-size: 12px;
  color: var(--td-text-color-secondary);
}
</style>
