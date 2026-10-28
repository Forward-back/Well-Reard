<template>
  <div class="shared-space-page">
    <div class="shared-space-header">
      <div>
        <h2>{{ $t('sharedSpace.title') }}</h2>
        <p class="subtitle">{{ $t('sharedSpace.description') }}</p>
      </div>
      <t-button variant="outline" :loading="loading" @click="load">
        <template #icon><t-icon name="refresh" /></template>
        {{ $t('common.refresh') }}
      </t-button>
    </div>

    <div v-if="loading && groups.length === 0" class="shared-space-loading">
      <t-loading size="medium" />
    </div>

    <t-empty v-else-if="groups.length === 0" :description="$t('sharedSpace.empty')" />

    <section v-for="group in groups" :key="group.departmentId" class="dept-group">
      <div class="dept-group-header">
        <span class="dept-group-name">{{ group.departmentName }}</span>
        <span class="dept-group-count">{{ $t('sharedSpace.count', { count: group.items.length }) }}</span>
      </div>
      <div class="kb-grid">
        <article v-for="kb in group.items" :key="kb.id" class="shared-kb-card" @click="openKB(kb)">
          <div class="kb-card-top">
            <div class="kb-icon">
              <t-icon name="folder" size="20px" />
            </div>
            <t-tag size="small" variant="light" theme="default">{{ $t('sharedSpace.readonly') }}</t-tag>
          </div>
          <h3 class="kb-title">{{ kb.name }}</h3>
          <p class="kb-description">{{ kb.description || $t('sharedSpace.noDescription') }}</p>
          <div class="kb-meta">
            <span>{{ $t('sharedSpace.knowledgeCount', { count: kb.knowledge_count ?? 0 }) }}</span>
            <span v-if="kb.updated_at">{{ formatStringDate(new Date(kb.updated_at)) }}</span>
          </div>
        </article>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { listDepartments, type Department } from '@/api/department'
import { listSharedSpaceKnowledgeBases } from '@/api/knowledge-base'
import { useAuthStore } from '@/stores/auth'
import { formatStringDate } from '@/utils/index'

interface SharedKB {
  id: string
  name: string
  description?: string
  department_id?: string
  shared_from_dept_id?: string
  knowledge_count?: number
  updated_at?: string
  is_shared_in?: boolean
  my_permission?: string
}

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const loading = ref(false)
const sharedKbs = ref<SharedKB[]>([])
const departments = ref<Department[]>([])

const departmentName = (id?: string) => {
  if (!id) return t('sharedSpace.unknownDepartment')
  return departments.value.find(dept => dept.id === id)?.name || t('sharedSpace.unknownDepartment')
}

const groups = computed(() => {
  const byDept = new Map<string, SharedKB[]>()
  for (const kb of sharedKbs.value) {
    const id = kb.shared_from_dept_id || kb.department_id || ''
    if (!byDept.has(id)) byDept.set(id, [])
    byDept.get(id)!.push(kb)
  }
  return Array.from(byDept.entries()).map(([departmentId, items]) => ({
    departmentId,
    departmentName: departmentName(departmentId),
    items,
  }))
})

const load = async () => {
  loading.value = true
  try {
    const tenantId = Number(authStore.effectiveTenantId || 0)
    const [sharedRes, deptRes] = await Promise.all([
      listSharedSpaceKnowledgeBases() as Promise<any>,
      tenantId ? listDepartments(tenantId) : Promise.resolve({ success: true, data: [] }),
    ])
    sharedKbs.value = Array.isArray(sharedRes?.data) ? sharedRes.data : []
    departments.value = Array.isArray(deptRes?.data) ? deptRes.data : []
  } finally {
    loading.value = false
  }
}

const openKB = (kb: SharedKB) => {
  router.push(`/platform/knowledge-bases/${kb.id}`)
}

onMounted(load)
</script>

<style lang="less" scoped>
.shared-space-page {
  width: 100%;
  height: 100%;
  overflow-y: auto;
  padding: 24px 32px 40px;
  box-sizing: border-box;
}

.shared-space-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 24px;

  h2 {
    margin: 0 0 8px;
    font-size: 24px;
    font-weight: 600;
    color: var(--td-text-color-primary);
  }

  .subtitle {
    margin: 0;
    font-size: 14px;
    line-height: 1.6;
    color: var(--td-text-color-secondary);
  }
}

.shared-space-loading {
  display: flex;
  justify-content: center;
  padding: 80px 0;
}

.dept-group {
  margin-bottom: 28px;
}

.dept-group-header {
  display: flex;
  align-items: baseline;
  gap: 10px;
  margin-bottom: 12px;
}

.dept-group-name {
  font-size: 16px;
  font-weight: 600;
  color: var(--td-text-color-primary);
}

.dept-group-count {
  font-size: 12px;
  color: var(--td-text-color-placeholder);
}

.kb-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 16px;
}

.shared-kb-card {
  padding: 18px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 10px;
  background: var(--td-bg-color-container);
  cursor: pointer;
  transition: border-color 0.2s, box-shadow 0.2s;

  &:hover {
    border-color: var(--td-brand-color);
    box-shadow: var(--td-shadow-1);
  }
}

.kb-card-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.kb-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: var(--td-brand-color-light);
  color: var(--td-brand-color);
}

.kb-title {
  margin: 0 0 6px;
  font-size: 15px;
  font-weight: 600;
  color: var(--td-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.kb-description {
  margin: 0;
  min-height: 38px;
  font-size: 13px;
  line-height: 1.5;
  color: var(--td-text-color-secondary);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.kb-meta {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  margin-top: 14px;
  font-size: 12px;
  color: var(--td-text-color-placeholder);
}
</style>
