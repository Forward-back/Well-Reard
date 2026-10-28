<template>
  <div class="skill-catalog-settings">
    <div class="section-header">
      <div>
        <h2>{{ $t('skillCatalog.title') }}</h2>
        <p class="section-description">{{ $t('skillCatalog.description') }}</p>
      </div>
      <t-button v-if="canManageSkills" theme="primary" @click="openAdd">{{ $t('skillCatalog.add') }}</t-button>
    </div>

    <div v-if="loading" class="skill-loading">
      <t-loading size="small" />
    </div>

    <template v-else>
      <div v-if="catalog.length === 0" class="skill-empty">{{ $t('skillCatalog.empty') }}</div>
      <div v-else class="skill-list">
        <article v-for="item in catalog" :key="item.id" class="skill-card"
          :class="{ 'skill-card--disabled': item.enabled === false }">
          <div class="skill-card-main">
            <div class="skill-card-heading">
              <t-icon name="book" class="skill-icon" />
              <span class="skill-name">{{ item.name }}</span>
              <t-tag v-if="item.department_id" size="small" variant="light">
                {{ departmentName(item.department_id) }}
              </t-tag>
              <t-tag v-if="item.version" size="small" variant="light">{{ item.version }}</t-tag>
              <t-tag size="small" :theme="item.enabled === false ? 'default' : 'success'" variant="light">
                {{ item.enabled === false ? $t('skillCatalog.disabled') : $t('skillCatalog.enabled') }}
              </t-tag>
            </div>
            <p v-if="item.description" class="skill-description">{{ item.description }}</p>
          </div>
          <div v-if="canEditSkill(item)" class="skill-card-actions">
            <t-button v-if="canManageSkills" size="small" variant="text" @click="openGrants(item)">
              {{ $t('skillCatalog.grants') }}
            </t-button>
            <t-button size="small" variant="text" @click="toggleEnabled(item)">
              {{ item.enabled === false ? $t('skillCatalog.enable') : $t('skillCatalog.disable') }}
            </t-button>
            <t-button v-if="canManageSkills" size="small" variant="text" theme="danger" @click="removeSkill(item)">
              {{ $t('common.delete') }}
            </t-button>
          </div>
        </article>
      </div>
    </template>

    <t-dialog v-model:visible="addVisible" :header="$t('skillCatalog.addTitle')" :confirm-btn="null"
      width="560px">
      <div class="skill-add-form">
        <p class="skill-add-hint">{{ $t('skillCatalog.addHint') }}</p>
        <input ref="fileInputRef" type="file" accept=".zip,.md" class="skill-file-input"
          @change="onFileSelected" />
        <t-button variant="outline" :loading="registering" @click="chooseFile">
          {{ $t('skillCatalog.chooseFile') }}
        </t-button>
        <span v-if="pendingFile" class="skill-file-name">{{ pendingFile.name }}</span>
        <t-divider>{{ $t('skillCatalog.orSource') }}</t-divider>
        <t-textarea v-model="source" :autosize="{ minRows: 3, maxRows: 6 }"
          :placeholder="$t('skillCatalog.sourcePlaceholder')" />
        <div class="skill-dialog-actions">
          <t-button theme="primary" :loading="registering" @click="registerSkill">
            {{ $t('skillCatalog.register') }}
          </t-button>
        </div>
      </div>
    </t-dialog>

    <t-dialog v-model:visible="grantsVisible" :header="grantsTitle" :confirm-btn="null" width="600px">
      <div class="skill-grant-form">
        <t-select v-model="grantSubjectType" :options="subjectTypeOptions" class="skill-grant-field" />
        <t-select v-if="grantSubjectType === 'user'" v-model="grantSubjectId" :options="memberOptions"
          filterable class="skill-grant-field" :placeholder="$t('skillCatalog.userPlaceholder')" />
        <t-select v-else v-model="grantSubjectId" :options="departmentOptions"
          class="skill-grant-field" :placeholder="$t('skillCatalog.departmentPlaceholder')" />
        <t-button theme="primary" :loading="grantSaving" :disabled="!grantSubjectId" @click="addGrant">
          {{ $t('skillCatalog.addGrant') }}
        </t-button>
      </div>

      <div v-if="grants.length === 0" class="skill-empty">{{ $t('skillCatalog.noGrants') }}</div>
      <div v-else class="skill-grant-list">
        <div v-for="grant in grants" :key="grant.id" class="skill-grant-row">
          <span>{{ grantLabel(grant) }}</span>
          <t-button size="small" variant="text" theme="danger" @click="removeGrant(grant)">
            {{ $t('common.delete') }}
          </t-button>
        </div>
      </div>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { MessagePlugin } from 'tdesign-vue-next'
import { useAuthStore } from '@/stores/auth'
import { listDepartmentMembers, listDepartments, type Department } from '@/api/department'
import { listMembers, type TenantMember } from '@/api/tenant/members'
import {
  createSkillGrant,
  deleteSkillCatalog,
  deleteSkillGrant,
  listSkillCatalog,
  listSkillGrants,
  registerSkillCatalogFromFile,
  registerSkillCatalogFromSource,
  setSkillCatalogEnabled,
  type SkillCatalogItem,
  type SkillEditorGrant,
  type SkillGrantSubjectType,
} from '@/api/skill'

const { t } = useI18n()
const authStore = useAuthStore()
const loading = ref(false)
const catalog = ref<SkillCatalogItem[]>([])
const addVisible = ref(false)
const registering = ref(false)
const pendingFile = ref<File | null>(null)
const source = ref('')
const fileInputRef = ref<HTMLInputElement | null>(null)

const grantsVisible = ref(false)
const grantsSkill = ref<SkillCatalogItem | null>(null)
const grants = ref<SkillEditorGrant[]>([])
const grantSubjectType = ref<SkillGrantSubjectType>('user')
const grantSubjectId = ref('')
const grantSaving = ref(false)
const members = ref<TenantMember[]>([])
const departments = ref<Department[]>([])

const grantsTitle = computed(() => t('skillCatalog.grantsTitle', {
  name: grantsSkill.value?.name || '',
}))
const subjectTypeOptions = computed(() => [
  { value: 'user', label: t('skillCatalog.subjectUser') },
  { value: 'department', label: t('skillCatalog.subjectDepartment') },
])
const memberOptions = computed(() => members.value.map((member) => ({
  value: member.user_id,
  label: member.username ? `${member.username} (${member.email})` : member.email,
})))
const canManageSkills = computed(() => authStore.hasCaps(['settings:skills_manage']))

const departmentOptions = computed(() => {
  const current = grantsSkill.value?.department_id || authStore.currentDepartmentId
  return departments.value
    .filter((dept) => !current || dept.id === current)
    .map((dept) => ({ value: dept.id, label: dept.name }))
})

const departmentName = (id: string) =>
  departments.value.find((dept) => dept.id === id)?.name || id

const canEditSkill = (item: SkillCatalogItem) => {
  if (item.can_edit === true) return true
  if (!canManageSkills.value) return false
  return !item.department_id || item.department_id === authStore.currentDepartmentId
}

const load = async () => {
  loading.value = true
  try {
    const response = await listSkillCatalog()
    catalog.value = Array.isArray(response.data) ? response.data : []
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('skillCatalog.loadFailed'))
  } finally {
    loading.value = false
  }
}

const loadGrantSubjects = async () => {
  const tenantID = Number(authStore.effectiveTenantId || 0)
  if (!tenantID) return
  try {
    const departmentID = grantsSkill.value?.department_id || authStore.currentDepartmentId
    const [memberRes, deptRes] = await Promise.all([
      departmentID
        ? listDepartmentMembers(tenantID, departmentID)
        : listMembers(tenantID, { page: 1, page_size: 100 }),
      listDepartments(tenantID),
    ])
    const memberData = memberRes.data
    members.value = Array.isArray(memberData) ? memberData : (memberData?.members || [])
    departments.value = Array.isArray(deptRes.data) ? deptRes.data : []
  } catch {
    members.value = []
    departments.value = []
  }
}

const openAdd = () => {
  pendingFile.value = null
  source.value = ''
  addVisible.value = true
}

const chooseFile = () => fileInputRef.value?.click()

const onFileSelected = (event: Event) => {
  const input = event.target as HTMLInputElement
  pendingFile.value = input.files?.[0] || null
}

const registerSkill = async () => {
  if (!pendingFile.value && !source.value.trim()) {
    MessagePlugin.warning(t('skillCatalog.addRequired'))
    return
  }
  registering.value = true
  try {
    if (pendingFile.value) {
      await registerSkillCatalogFromFile(pendingFile.value)
    } else {
      await registerSkillCatalogFromSource(source.value.trim())
    }
    MessagePlugin.success(t('skillCatalog.addSuccess'))
    addVisible.value = false
    await load()
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('skillCatalog.addFailed'))
  } finally {
    registering.value = false
  }
}

const toggleEnabled = async (item: SkillCatalogItem) => {
  try {
    await setSkillCatalogEnabled(item.id, item.enabled === false)
    await load()
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('skillCatalog.updateFailed'))
  }
}

const removeSkill = async (item: SkillCatalogItem) => {
  try {
    await deleteSkillCatalog(item.id)
    MessagePlugin.success(t('skillCatalog.deleteSuccess'))
    await load()
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('skillCatalog.deleteFailed'))
  }
}

const openGrants = async (item: SkillCatalogItem) => {
  grantsSkill.value = item
  grantSubjectId.value = ''
  grantsVisible.value = true
  await Promise.all([loadGrants(item.id), loadGrantSubjects()])
}

const loadGrants = async (skillID: string) => {
  try {
    const response = await listSkillGrants(skillID)
    grants.value = Array.isArray(response.data) ? response.data : []
  } catch (err: any) {
    grants.value = []
    MessagePlugin.error(err?.message || t('skillCatalog.grantsLoadFailed'))
  }
}

const grantLabel = (grant: SkillEditorGrant) => {
  if (grant.subject_type === 'user') {
    return `${t('skillCatalog.subjectUser')}：${memberOptions.value.find((o) => o.value === grant.subject_id)?.label || grant.subject_id}`
  }
  return `${t('skillCatalog.subjectDepartment')}：${departmentOptions.value.find((o) => o.value === grant.subject_id)?.label || grant.subject_id}`
}

const addGrant = async () => {
  if (!grantsSkill.value) return
  grantSaving.value = true
  try {
    await createSkillGrant(grantsSkill.value.id, {
      subject_type: grantSubjectType.value,
      subject_id: grantSubjectId.value,
    })
    grantSubjectId.value = ''
    await loadGrants(grantsSkill.value.id)
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('skillCatalog.grantFailed'))
  } finally {
    grantSaving.value = false
  }
}

const removeGrant = async (grant: SkillEditorGrant) => {
  if (!grantsSkill.value) return
  try {
    await deleteSkillGrant(grantsSkill.value.id, grant.id)
    await loadGrants(grantsSkill.value.id)
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('skillCatalog.grantFailed'))
  }
}

onMounted(() => {
  void load()
  void loadGrantSubjects()
})
</script>

<style lang="less" scoped>
.skill-catalog-settings {
  width: 100%;
}

.section-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;

  h2 {
    margin: 0 0 8px;
    font-size: 20px;
    font-weight: 600;
    color: var(--td-text-color-primary);
  }
}

.section-description {
  margin: 0;
  font-size: 14px;
  line-height: 1.5;
  color: var(--td-text-color-secondary);
}

.skill-loading {
  display: flex;
  justify-content: center;
  padding: 24px;
}

.skill-empty {
  padding: 20px 0;
  font-size: 13px;
  color: var(--td-text-color-placeholder);
}

.skill-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.skill-card {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;

  &--disabled {
    opacity: 0.65;
  }
}

.skill-card-main {
  min-width: 0;
}

.skill-card-heading {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.skill-icon {
  color: var(--td-brand-color);
}

.skill-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--td-text-color-primary);
}

.skill-description {
  margin: 6px 0 0;
  font-size: 13px;
  line-height: 1.6;
  color: var(--td-text-color-secondary);
}

.skill-card-actions {
  display: flex;
  flex-shrink: 0;
  gap: 2px;
}

.skill-add-form,
.skill-grant-form {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
}

.skill-add-form {
  flex-direction: column;
  align-items: stretch;
}

.skill-add-hint {
  margin: 0;
  font-size: 13px;
  color: var(--td-text-color-secondary);
}

.skill-file-input {
  display: none;
}

.skill-file-name {
  font-size: 12px;
  color: var(--td-text-color-placeholder);
}

.skill-dialog-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 8px;
}

.skill-grant-field {
  min-width: 160px;
  flex: 1;
}

.skill-grant-list {
  margin-top: 16px;
  border-top: 1px solid var(--td-component-stroke);
}

.skill-grant-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid var(--td-component-stroke);
  font-size: 13px;
}
</style>
