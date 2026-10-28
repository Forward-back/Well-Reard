<template>
  <div class="kb-dept-share-panel">
    <div class="share-panel-header">
      <div class="share-panel-titlewrap">
        <h2 class="share-panel-title">{{ $t('kbShare.title') }}</h2>
      </div>
      <p class="share-panel-desc">{{ $t('kbShare.description') }}</p>
    </div>

    <div v-if="loading" class="share-loading">
      <t-loading size="small" />
    </div>

    <template v-else>
      <div class="share-row">
        <span class="share-label">{{ $t('kbShare.owningDepartment') }}</span>
        <t-tag size="small" variant="light" theme="primary">{{ owningDepartmentName }}</t-tag>
      </div>

      <div class="share-row share-row--scope">
        <span class="share-label">{{ $t('kbShare.sharedTo') }}</span>
        <div v-if="canEdit" class="share-scope-editor">
          <t-checkbox-group v-model="selectedIDs" class="share-checkboxes">
            <t-checkbox
              v-for="dept in shareableDepartments"
              :key="dept.id"
              :value="dept.id"
              :label="dept.name"
            />
          </t-checkbox-group>
          <div class="share-scope-actions">
            <t-button size="small" variant="text" theme="primary" @click="selectAll">
              {{ $t('kbShare.selectAll') }}
            </t-button>
            <t-button size="small" theme="primary" :loading="saving" @click="save">
              {{ $t('common.save') }}
            </t-button>
          </div>
        </div>
        <div v-else class="share-scope-readonly">
          <t-tag v-for="id in sharedIDs" :key="id" size="small" variant="light">
            {{ departmentName(id) }}
          </t-tag>
        </div>
      </div>

      <p v-if="canEdit" class="share-hint">{{ $t('kbShare.managerHint') }}</p>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { listDepartments, type Department } from '@/api/department'
import { getKBDepartmentShares, setKBDepartmentShares } from '@/api/knowledge-base'

const props = withDefaults(defineProps<{
  kbId: string
  canShare?: boolean
}>(), {
  canShare: false,
})

const { t } = useI18n()
const authStore = useAuthStore()

const loading = ref(false)
const saving = ref(false)
const departments = ref<Department[]>([])
const owningDepartmentId = ref('')
const selectedIDs = ref<string[]>([])
const sharedIDs = computed(() => selectedIDs.value.filter(id => id !== owningDepartmentId.value))

const owningDepartmentName = computed(() => departmentName(owningDepartmentId.value))
const shareableDepartments = computed(() => departments.value.filter(d => d.id !== owningDepartmentId.value))
const canEdit = computed(() => (
  props.canShare &&
  authStore.hasDepartmentRole('manager') &&
  owningDepartmentId.value === authStore.currentDepartmentId
))

const departmentName = (id: string) => {
  return departments.value.find(dept => dept.id === id)?.name || id
}

const load = async () => {
  if (!props.kbId) return
  loading.value = true
  try {
    const tenantId = Number(authStore.effectiveTenantId || 0)
    const [deptRes, shareRes] = await Promise.all([
      tenantId ? listDepartments(tenantId) : Promise.resolve({ success: true, data: [] }),
      getKBDepartmentShares(props.kbId) as Promise<any>,
    ])
    departments.value = Array.isArray(deptRes?.data) ? deptRes.data : []
    const data = shareRes?.data || {}
    owningDepartmentId.value = data.owning_department_id || ''
    selectedIDs.value = Array.isArray(data.department_ids) ? [...data.department_ids] : []
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('kbShare.loadFailed'))
  } finally {
    loading.value = false
  }
}

const selectAll = () => {
  selectedIDs.value = [owningDepartmentId.value, ...shareableDepartments.value.map(d => d.id)]
}

const save = async () => {
  saving.value = true
  try {
    const res: any = await setKBDepartmentShares(
      props.kbId,
      selectedIDs.value.filter(id => id !== owningDepartmentId.value),
    )
    if (res?.success === false) throw new Error(res?.message || t('kbShare.saveFailed'))
    const data = res?.data || {}
    selectedIDs.value = Array.isArray(data.department_ids) ? [...data.department_ids] : selectedIDs.value
    MessagePlugin.success(t('kbShare.saveSuccess'))
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('kbShare.saveFailed'))
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<style lang="less" scoped>
.kb-dept-share-panel {
  width: 100%;
}

.share-panel-header {
  margin-bottom: 20px;
}

.share-panel-title {
  margin: 0 0 6px;
  font-size: 16px;
  font-weight: 600;
  color: var(--td-text-color-primary);
}

.share-panel-desc {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: var(--td-text-color-secondary);
}

.share-loading {
  display: flex;
  justify-content: center;
  padding: 24px;
}

.share-row {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  padding: 14px 0;
  border-bottom: 1px solid var(--td-component-stroke);

  &:last-of-type {
    border-bottom: none;
  }
}

.share-label {
  flex: 0 0 96px;
  padding-top: 4px;
  font-size: 13px;
  color: var(--td-text-color-secondary);
}

.share-scope-editor {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.share-checkboxes {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 8px 16px;
}

.share-scope-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.share-scope-readonly {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding-top: 2px;
}

.share-hint {
  margin: 12px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--td-text-color-placeholder);
}
</style>
