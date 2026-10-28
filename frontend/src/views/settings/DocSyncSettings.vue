<template>
  <div class="doc-sync-settings">
    <div class="section-header">
      <h2>{{ $t('docSync.title') }}</h2>
      <p class="section-description">{{ $t('docSync.description') }}</p>
    </div>

    <div class="intro">
      <t-icon name="info-circle" class="intro-icon" />
      <div>
        <p class="intro-title">{{ $t('docSync.dailyEnabled') }}</p>
        <p class="intro-desc">{{ $t('docSync.manualHint') }}</p>
      </div>
    </div>

    <div class="settings-group">
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ $t('docSync.dailyEnabled') }}</label>
          <p class="desc">{{ $t('docSync.dailyEnabledHint') }}</p>
        </div>
        <div class="setting-control">
          <t-switch v-model="enabled" :disabled="!canEdit || loading || saving" />
        </div>
      </div>
    </div>

    <div v-if="canEdit" class="settings-actions">
      <t-button theme="primary" :loading="saving" :disabled="loading" @click="save">
        {{ $t('common.save') }}
      </t-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { getTenantEngineSyncConfig, updateTenantEngineSyncConfig } from '@/api/collaboration'

const { t } = useI18n()
const authStore = useAuthStore()

const canEdit = computed(() => authStore.hasCaps(['settings:workspace_manage']))
const enabled = ref(false)
const loading = ref(false)
const saving = ref(false)

const loadConfig = async () => {
  loading.value = true
  try {
    const result: any = await getTenantEngineSyncConfig()
    if (result?.success === false) {
      MessagePlugin.error(result?.message || t('docSync.loadFailed'))
      return
    }
    const data = result?.data ?? result
    enabled.value = data?.enabled === true
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('docSync.loadFailed'))
  } finally {
    loading.value = false
  }
}

const save = async () => {
  saving.value = true
  try {
    const result: any = await updateTenantEngineSyncConfig({ enabled: enabled.value })
    if (result?.success === false) {
      MessagePlugin.error(result?.message || t('docSync.saveFailed'))
      return
    }
    MessagePlugin.success(t('docSync.saveSuccess'))
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('docSync.saveFailed'))
  } finally {
    saving.value = false
  }
}

onMounted(loadConfig)
</script>

<style lang="less" scoped>
.doc-sync-settings {
  width: 100%;
}

.section-header {
  margin-bottom: 24px;

  h2 {
    font-size: 20px;
    font-weight: 600;
    color: var(--td-text-color-primary);
    margin: 0 0 8px 0;
  }

  .section-description {
    font-size: 14px;
    color: var(--td-text-color-secondary);
    margin: 0;
    line-height: 1.5;
  }
}

.intro {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 14px 16px;
  margin-bottom: 8px;
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);
}

.intro-icon {
  color: var(--td-brand-color);
  margin-top: 2px;
  flex-shrink: 0;
}

.intro-title {
  margin: 0 0 4px 0;
  font-size: 14px;
  font-weight: 500;
  color: var(--td-text-color-primary);
}

.intro-desc {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: var(--td-text-color-secondary);
}

.settings-group {
  display: flex;
  flex-direction: column;
}

.setting-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
  padding: 16px 0;
  border-bottom: 1px solid var(--td-component-stroke);

  &:last-child {
    border-bottom: none;
  }
}

.setting-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;

  label {
    font-size: 14px;
    font-weight: 500;
    color: var(--td-text-color-primary);
  }

  .desc {
    margin: 0;
    font-size: 13px;
    line-height: 1.6;
    color: var(--td-text-color-secondary);
  }
}

.setting-control {
  flex-shrink: 0;
  padding-top: 2px;
}

.settings-actions {
  display: flex;
  justify-content: flex-end;
  padding-top: 16px;
}
</style>
