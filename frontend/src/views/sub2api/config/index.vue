<template>
  <div class="sub2api-config" v-loading="store.configLoading">
    <header class="page-head">
      <div>
        <h1>{{ t('sub2api.common.connectionConfig') }}</h1>
        <p>{{ t('sub2api.admin.configSubtitle') }}</p>
      </div>
      <el-tag :type="store.statusType(snapshot?.status)" effect="light" round>
        {{ store.statusText(snapshot?.status) }}
      </el-tag>
    </header>

    <el-card shadow="never" class="form-card">
      <el-form :model="form" label-width="130px" label-position="right" :disabled="!canEdit">
        <el-form-item :label="t('sub2api.admin.autoSync')">
          <el-switch v-model="form.syncEnabled" />
        </el-form-item>
        <el-form-item :label="t('sub2api.admin.baseUrl')">
          <el-input v-model="form.baseUrl" placeholder="https://your-sub2api.example.com" />
        </el-form-item>
        <el-form-item :label="t('sub2api.admin.adminApiKey')">
          <el-input v-model="form.adminApiKey" type="password" show-password placeholder="sk-..." />
        </el-form-item>
        <el-form-item :label="t('sub2api.admin.consoleUrl')">
          <el-input v-model="form.consoleUrl" placeholder="https://your-sub2api.example.com" />
          <span class="form-hint">{{ t('sub2api.admin.consoleUrlHint') }}</span>
        </el-form-item>
        <el-form-item :label="t('sub2api.admin.syncInterval')">
          <el-input-number v-model="form.syncIntervalMinutes" :min="1" :max="1440" />
        </el-form-item>
        <el-form-item :label="t('sub2api.admin.historyDays')">
          <el-input-number v-model="form.historyDays" :min="1" :max="365" />
        </el-form-item>
        <el-form-item :label="t('sub2api.admin.pageSize')">
          <el-input-number v-model="form.pageSize" :min="50" :max="1000" :step="50" />
        </el-form-item>
        <el-form-item v-if="canEdit">
          <el-button :loading="store.testing" @click="onTest">{{
            t('sub2api.common.testConnection')
          }}</el-button>
          <el-button type="primary" :loading="store.saving" @click="onSave">{{
            t('sub2api.common.saveConfig')
          }}</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { PERM } from '@/config/permission'
import { useButtonPermission } from '@/composables/useButtonPermission'
import { useSub2APIStore } from '@/stores/sub2api'

defineOptions({ name: 'Sub2APIConfig' })

const store = useSub2APIStore()
const { snapshot, configForm: form } = storeToRefs(store)
const { t } = useI18n()
const canEdit = useButtonPermission([PERM.SUB2API_EDIT], [])

const onTest = async () => {
  if (!canEdit.value) return
  const result = await store.testConfig()
  if (!result) return
  if (result.connected) ElMessage.success(result.message || t('sub2api.common.connected'))
  else ElMessage.error(result.message || t('sub2api.common.disconnected'))
}

const onSave = async () => {
  if (!canEdit.value) return
  const ok = await store.saveConfig()
  if (ok) ElMessage.success(t('sub2api.common.saved'))
}

onMounted(() => {
  store.loadAdmin()
})
</script>

<style scoped lang="scss">
.sub2api-config {
  padding: 16px;
}

.page-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 16px;

  h1 {
    margin: 0;
    font-size: 20px;
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  p {
    margin: 6px 0 0;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }
}

.form-card {
  max-width: 720px;
  border-radius: 12px;
}

.form-hint {
  margin-left: 10px;
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}

@media (max-width: 640px) {
  .sub2api-config {
    padding: 10px;
  }

  .form-card {
    max-width: none;
  }

  .form-card :deep(.el-form-item) {
    display: block;
    margin-bottom: 13px;
  }

  .form-card :deep(.el-form-item__label) {
    display: block;
    width: auto !important;
    margin-bottom: 5px;
    text-align: left;
  }

  .form-card :deep(.el-form-item__content) {
    width: 100%;
    margin-left: 0 !important;
  }

  .form-card :deep(.el-input),
  .form-card :deep(.el-input-number) {
    width: 100%;
  }

  .form-hint {
    display: block;
    width: 100%;
    margin: 4px 0 0;
  }
}
</style>
