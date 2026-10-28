<template>
  <div class="kb-settings-page">
    <header class="page-head">
      <div>
        <h1>{{ $t('management.kbSettings.title') }}</h1>
        <p>{{ $t('management.kbSettings.description', { dept: departmentName || '-' }) }}</p>
      </div>
    </header>

    <div v-if="loading" class="page-empty"><t-loading size="small" /></div>
    <div v-else-if="!kbs.length" class="page-empty">{{ $t('management.kbSettings.empty') }}</div>
    <div v-else class="kb-grid">
      <article v-for="kb in kbs" :key="kb.id" class="kb-card">
        <div class="kb-card-main">
          <span class="kb-card-icon"><t-icon name="book" size="20px" /></span>
          <span class="kb-card-copy">
            <strong>{{ kb.name }}</strong>
            <small>{{ kb.description || $t('management.kbSettings.docs', { count: kb.document_count || 0 }) }}</small>
          </span>
        </div>
        <t-button size="small" variant="outline" @click="openSettings(kb.id)">
          {{ $t('management.kbSettings.open') }}
        </t-button>
      </article>
    </div>

    <KnowledgeBaseEditorModal :visible="uiStore.showKBEditorModal" :mode="uiStore.kbEditorMode"
      :kb-id="uiStore.currentKBId || undefined" :initial-type="uiStore.kbEditorType"
      @update:visible="(val) => val ? null : uiStore.closeKBEditor()" @success="load" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { listKnowledgeBases } from '@/api/knowledge-base'
import { listDepartments, type Department } from '@/api/department'
import { useAuthStore } from '@/stores/auth'
import { useUIStore } from '@/stores/ui'
import KnowledgeBaseEditorModal from '@/views/knowledge/KnowledgeBaseEditorModal.vue'

const { t } = useI18n()
const authStore = useAuthStore()
const uiStore = useUIStore()
const kbs = ref<any[]>([])
const departments = ref<Department[]>([])
const loading = ref(false)

const departmentName = computed(() =>
  departments.value.find((dept) => dept.id === authStore.currentDepartmentId)?.name || '',
)

const load = async () => {
  loading.value = true
  try {
    const tenantID = Number(authStore.effectiveTenantId || 0)
    const [kbRes, deptRes] = await Promise.all([
      listKnowledgeBases(),
      tenantID ? listDepartments(tenantID) : Promise.resolve({ data: [] }),
    ])
    kbs.value = Array.isArray((kbRes as any)?.data) ? (kbRes as any).data : []
    departments.value = Array.isArray((deptRes as any)?.data) ? (deptRes as any).data : []
  } catch {
    kbs.value = []
  } finally {
    loading.value = false
  }
}

const openSettings = (kbId: string) => {
  uiStore.openKBSettings(kbId)
}

onMounted(load)
</script>

<style lang="less" scoped>
.kb-settings-page {
  width: 100%;
  max-width: 1040px;
  margin: 0 auto;
  padding: 24px 24px 48px;
  box-sizing: border-box;
}

.page-head {
  margin-bottom: 16px;

  h1 { margin: 0 0 5px; font-size: 20px; font-weight: 600; }
  p { margin: 0; color: var(--td-text-color-secondary); font-size: 12px; }
}

.kb-grid {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.kb-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 16px;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: 10px;
}

.kb-card-main {
  display: flex;
  flex: 1;
  min-width: 0;
  align-items: center;
  gap: 11px;
}

.kb-card-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 36px;
  width: 36px;
  height: 36px;
  border-radius: 9px;
  color: var(--td-brand-color);
  background: var(--td-brand-color-light);
}

.kb-card-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 3px;

  strong { overflow: hidden; color: var(--td-text-color-primary); font-size: 13px; text-overflow: ellipsis; white-space: nowrap; }
  small { overflow: hidden; color: var(--td-text-color-placeholder); font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }
}

.page-empty {
  padding: 36px;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  text-align: center;
}
</style>
