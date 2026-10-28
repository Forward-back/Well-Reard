<template>
  <div class="kb-acl-settings">
    <div class="section-header">
      <h2>{{ $t('kbAcl.title') }}</h2>
      <p class="section-description">{{ $t('kbAcl.description') }}</p>
    </div>

    <div v-if="loading" class="acl-loading">
      <t-loading size="small" />
    </div>

    <template v-else>
      <div class="acl-form">
        <t-select v-model="form.resourceType" class="acl-field" :options="resourceTypeOptions" />
        <t-select v-if="form.resourceType === 'folder'" v-model="form.resourceId" class="acl-field acl-field--folder"
          :options="folderOptions" :placeholder="$t('kbAcl.folderPlaceholder')" />
        <t-select v-else v-model="form.resourceId" class="acl-field acl-field--folder" :options="documentOptions"
          :placeholder="$t('kbAcl.filePlaceholder')" filterable />
        <t-select v-model="form.effect" class="acl-field" :options="effectOptions" />
        <t-select v-model="form.subjectType" class="acl-field" :options="subjectTypeOptions" />
        <t-select v-if="form.subjectType === 'department'" v-model="form.subjectId" class="acl-field"
          :options="departmentOptions" :placeholder="$t('kbAcl.departmentPlaceholder')" />
        <t-select v-else-if="form.subjectType === 'user'" v-model="form.subjectId" class="acl-field"
          :options="memberOptions" :placeholder="$t('kbAcl.userPlaceholder')" filterable />
        <t-button theme="primary" :loading="saving" :disabled="!canSubmit" @click="createRule">
          {{ $t('kbAcl.add') }}
        </t-button>
      </div>

      <div v-if="rules.length === 0" class="acl-empty">{{ $t('kbAcl.empty') }}</div>
      <div v-else class="acl-list">
        <div v-for="rule in rules" :key="rule.id" class="acl-row">
          <div class="acl-row-main">
            <t-tag size="small" :theme="rule.effect === 'deny' ? 'danger' : 'success'" variant="light">
              {{ rule.effect === 'deny' ? $t('kbAcl.deny') : $t('kbAcl.allow') }}
            </t-tag>
            <span class="acl-resource">{{ resourceLabel(rule) }}</span>
            <span class="acl-subject">{{ subjectLabel(rule) }}</span>
          </div>
          <t-button size="small" variant="text" theme="danger" @click="removeRule(rule)">
            {{ $t('common.delete') }}
          </t-button>
        </div>
      </div>
      <p class="acl-hint">{{ $t('kbAcl.hint') }}</p>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { listDepartments, type Department } from '@/api/department'
import { listMembers, type TenantMember } from '@/api/tenant/members'
import {
  createKBAclRule,
  deleteKBAclRule,
  listKBAcl,
  listKnowledgeFiles,
  listKnowledgeFolders,
  type KBAclEffect,
  type KBAclRule,
  type KBAclSubjectType,
  type KnowledgeFolderNode,
} from '@/api/knowledge-base'

const props = withDefaults(defineProps<{
  kbId: string
  canManage?: boolean
}>(), {
  canManage: false,
})

const { t } = useI18n()
const authStore = useAuthStore()
const loading = ref(false)
const saving = ref(false)
const rules = ref<KBAclRule[]>([])
const folders = ref<KnowledgeFolderNode[]>([])
const documents = ref<any[]>([])
const departments = ref<Department[]>([])
const members = ref<TenantMember[]>([])

const form = reactive<{
  resourceType: 'folder' | 'file'
  resourceId: string
  effect: KBAclEffect
  subjectType: KBAclSubjectType
  subjectId: string
}>({
  resourceType: 'folder',
  resourceId: '',
  effect: 'deny',
  subjectType: 'all',
  subjectId: '',
})

const isManager = computed(() => authStore.hasDepartmentRole('manager'))
const resourceTypeOptions = computed(() => [
  { value: 'folder', label: t('kbAcl.resourceFolder') },
  { value: 'file', label: t('kbAcl.resourceFile') },
])
const effectOptions = computed(() => [
  { value: 'deny', label: t('kbAcl.deny') },
  ...(isManager.value && form.resourceType !== 'file' ? [{ value: 'allow', label: t('kbAcl.allow') }] : []),
])
const subjectTypeOptions = computed(() => [
  { value: 'all', label: t('kbAcl.subjectAll') },
  { value: 'department', label: t('kbAcl.subjectDepartment') },
  { value: 'user', label: t('kbAcl.subjectUser') },
])
const departmentOptions = computed(() => departments.value.map(d => ({ value: d.id, label: d.name })))
const memberOptions = computed(() => members.value.map(m => ({
  value: m.user_id,
  label: m.username ? `${m.username} (${m.email})` : m.email,
})))
const documentOptions = computed(() => documents.value.map(d => ({
  value: d.id,
  label: d.file_name || d.title || d.id,
})))
const canSubmit = computed(() => (
  props.canManage && !!form.resourceId && (form.subjectType === 'all' ||
    (form.subjectType === 'department' && !!form.subjectId) ||
    (form.subjectType === 'user' && !!form.subjectId))
))

watch(() => form.resourceType, (type) => {
  form.resourceId = ''
  if (type === 'file') form.effect = 'deny'
})

const folderOptions = computed(() => {
  const out: Array<{ value: string; label: string }> = []
  const walk = (nodes: KnowledgeFolderNode[], depth: number) => {
    for (const node of nodes) {
      if (node.id) out.push({ value: node.id, label: `${'　'.repeat(depth)}${node.name}` })
      walk(node.children || [], depth + 1)
    }
  }
  walk(folders.value, 0)
  return out
})

const folderName = (id: string) => folderOptions.value.find(o => o.value === id)?.label.trim() || id
const departmentName = (id: string) => departments.value.find(d => d.id === id)?.name || id
const memberName = (id: string) => memberOptions.value.find(o => o.value === id)?.label || id

const documentName = (id: string) => documentOptions.value.find(o => o.value === id)?.label || id
const resourceLabel = (rule: KBAclRule) => rule.resource_type === 'folder' ? folderName(rule.resource_id) : documentName(rule.resource_id)
const subjectLabel = (rule: KBAclRule) => {
  if (rule.subject_type === 'all') return t('kbAcl.subjectAll')
  if (rule.subject_type === 'department') return `${t('kbAcl.subjectDepartment')}: ${departmentName(rule.subject_id)}`
  return `${t('kbAcl.subjectUser')}: ${memberName(rule.subject_id)}`
}

const load = async () => {
  if (!props.kbId) return
  loading.value = true
  const tenantId = Number(authStore.effectiveTenantId || 0)
  try {
    const [aclRes, folderRes, deptRes, memberRes, docRes] = await Promise.all([
      listKBAcl(props.kbId) as Promise<any>,
      listKnowledgeFolders(props.kbId) as Promise<any>,
      tenantId ? listDepartments(tenantId) : Promise.resolve({ success: true, data: [] }),
      tenantId ? listMembers(tenantId, { page: 1, page_size: 100 }) : Promise.resolve({ success: true, data: { members: [] } }),
      listKnowledgeFiles(props.kbId, { page: 1, page_size: 100 }),
    ])
    rules.value = Array.isArray(aclRes?.data?.rules) ? aclRes.data.rules : []
    folders.value = Array.isArray(folderRes?.data?.folders) ? folderRes.data.folders : []
    departments.value = Array.isArray(deptRes?.data) ? deptRes.data : []
    members.value = Array.isArray(memberRes?.data?.members) ? memberRes.data.members : []
    documents.value = Array.isArray(docRes?.data) ? docRes.data : []
    if (!form.resourceId && folders.value.length) {
      form.resourceId = folders.value[0].id || ''
    }
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('kbAcl.loadFailed'))
  } finally {
    loading.value = false
  }
}

const createRule = async () => {
  if (!props.canManage || !canSubmit.value) return
  saving.value = true
  try {
    const res: any = await createKBAclRule(props.kbId, {
      resource_type: form.resourceType,
      resource_id: form.resourceId,
      effect: form.resourceType === 'file' ? 'deny' : form.effect,
      subject_type: form.subjectType,
      subject_id: form.subjectType === 'all' ? '' : form.subjectId,
    })
    if (res?.success === false) throw new Error(res?.message || t('kbAcl.createFailed'))
    MessagePlugin.success(t('kbAcl.createSuccess'))
    form.subjectId = ''
    await load()
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('kbAcl.createFailed'))
  } finally {
    saving.value = false
  }
}

const removeRule = async (rule: KBAclRule) => {
  try {
    const res: any = await deleteKBAclRule(props.kbId, rule.id)
    if (res?.success === false) throw new Error(res?.message || t('kbAcl.deleteFailed'))
    MessagePlugin.success(t('kbAcl.deleteSuccess'))
    await load()
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('kbAcl.deleteFailed'))
  }
}

onMounted(load)
</script>

<style lang="less" scoped>
.kb-acl-settings {
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

.acl-loading {
  display: flex;
  justify-content: center;
  padding: 24px;
}

.acl-form {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  margin-bottom: 16px;
}

.acl-field {
  width: 160px;

  &--folder {
    width: 220px;
  }
}

.acl-empty {
  padding: 20px 0;
  font-size: 13px;
  color: var(--td-text-color-placeholder);
}

.acl-list {
  display: flex;
  flex-direction: column;
  border-top: 1px solid var(--td-component-stroke);
}

.acl-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 0;
  border-bottom: 1px solid var(--td-component-stroke);
}

.acl-row-main {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.acl-resource {
  font-size: 13px;
  color: var(--td-text-color-primary);
}

.acl-subject {
  font-size: 12px;
  color: var(--td-text-color-secondary);
}

.acl-hint {
  margin: 12px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--td-text-color-placeholder);
}
</style>
