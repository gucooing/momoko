<!-- 登录设备 -->
<template>
  <BaseCard>
    <template #header>
      <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div class="space-y-1">
          <h1 class="text-2xl font-semibold">{{ t('user.loginDevices') }}</h1>
        </div>
        <div class="flex items-center gap-2">
          <BaseTag type="info" :text="t('user.totalCount', { count: loginDevices.length })" />
          <BaseTag type="primary" :text="`${t('user.currentDevice')} ${currentDeviceLabel}`" />
        </div>
      </div>
    </template>

    <el-empty v-if="!loading && !loginDevices.length" :description="t('user.noLoginDevices')" />

    <!-- desktop: table -->
    <el-table
      v-else-if="!menuStore.isMobile"
      v-loading="loading"
      :data="loginDevices"
      :border="TABLE_CONFIG.border"
      table-layout="auto"
      class="custom-modern-table"
    >
      <el-table-column :label="t('user.loginDevices')" :min-width="deviceColumnWidth" class-name="auto-wrap-cell">
        <template #default="{ row }">
          <div class="device-cell">
            <span class="device-text">{{ row.device || '-' }}</span>
            <BaseTag v-if="isCurrentDevice(row)" type="success" :text="t('user.currentDevice')" />
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="ip" :label="t('user.extra.ipAddress')" :min-width="ipColumnWidth" class-name="auto-wrap-cell" />
      <el-table-column :label="t('user.loginTime')" min-width="170">
        <template #default="{ row }">{{ formatDateTime(row.loginTime) }}</template>
      </el-table-column>
      <el-table-column :label="t('user.latestRefresh')" min-width="170">
        <template #default="{ row }">{{ formatDateTime(row.updateTime) }}</template>
      </el-table-column>
      <el-table-column prop="sessionId" :label="t('user.sessionId')" :min-width="sessionIdColumnWidth" class-name="auto-wrap-cell" />
      <el-table-column v-for="key in extraColumns" :key="key" :label="getExtraColumnLabel(key)" :min-width="getExtraColumnWidth(key)" class-name="auto-wrap-cell">
        <template #default="{ row }">{{ formatCellValue(row[key]) }}</template>
      </el-table-column>
      <el-table-column :label="t('user.operation')" width="110">
        <template #default="{ row }">
          <span v-if="isCurrentDevice(row)" class="text-(--el-text-color-secondary)">-</span>
          <AdaptiveConfirm v-else :title="t('user.deleteDeviceConfirm')" @confirm="handleDeleteDevice(row)" :disabled="isDeletingLoginDevice(row.deviceId)">
            <template #reference>
              <el-button type="danger" link :loading="isDeletingLoginDevice(row.deviceId)">{{ t('common.delete') }}</el-button>
            </template>
          </AdaptiveConfirm>
        </template>
      </el-table-column>
    </el-table>

    <!-- mobile: cards -->
    <div v-else v-loading="loading" class="mobile-card-list">
      <div v-for="row in loginDevices" :key="row.deviceId" class="mobile-card">
        <div class="mobile-card-header">
          <span class="mobile-card-title">{{ row.device || '-' }}</span>
          <BaseTag v-if="isCurrentDevice(row)" type="success" :text="t('user.currentDevice')" />
        </div>
        <div class="mobile-card-meta">
          <span>{{ row.ip || '-' }}</span>
        </div>
        <div class="mobile-card-meta">
          <span>{{ t('user.loginAt', { time: formatDateTime(row.loginTime) }) }}</span>
          <span class="meta-sep">·</span>
          <span>{{ t('user.refreshAt', { time: formatDateTime(row.updateTime) }) }}</span>
        </div>
        <div class="mobile-card-meta mobile-card-sid">{{ row.sessionId || '-' }}</div>
        <div v-if="!isCurrentDevice(row)" class="mobile-card-footer">
          <AdaptiveConfirm :title="t('user.deleteDeviceConfirm')" @confirm="handleDeleteDevice(row)" :disabled="isDeletingLoginDevice(row.deviceId)">
            <template #reference>
              <el-button size="small" plain type="danger" :loading="isDeletingLoginDevice(row.deviceId)">{{ t('common.delete') }}</el-button>
            </template>
          </AdaptiveConfirm>
        </div>
      </div>
    </div>
  </BaseCard>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { TABLE_CONFIG } from '@/config/elementConfig'
import { useUserProfileStore } from '@/stores/user/profile'
import { useI18n } from 'vue-i18n'
const menuStore = useMenuStore()
const userProfileStore = useUserProfileStore()
const { t } = useI18n()
const {
  loading,
  loginDevices,
  currentDeviceLabel,
  extraColumns,
  deviceColumnWidth,
  ipColumnWidth,
  sessionIdColumnWidth,
} = storeToRefs(userProfileStore)

const {
  formatDateTime,
  formatCellValue,
  getExtraColumnLabel,
  getExtraColumnWidth,
  isCurrentDevice,
  isDeletingLoginDevice,
  getLoginDevices,
  deleteLoginDevice,
} = userProfileStore

const handleDeleteDevice = async (target: { deviceId: string }) => {
  const deleted = await deleteLoginDevice(target.deviceId)
  if (deleted) {
    ElMessage.success(t('user.deviceDeleted'))
  }
}

onMounted(() => {
  void getLoginDevices()
})
</script>

<style scoped lang="scss">
.device-cell {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.device-text {
  overflow-wrap: anywhere;
  word-break: break-word;
}

:deep(.auto-wrap-cell .cell) {
  white-space: normal;
  overflow-wrap: anywhere;
  word-break: break-word;
  line-height: 1.4;
}

:deep(.custom-modern-table .el-table__header-wrapper table),
:deep(.custom-modern-table .el-table__body-wrapper table) {
  min-width: 100%;
  width: max-content;
}

/* mobile */
.mobile-card-list { display: flex; flex-direction: column; gap: 0.55rem; }
.mobile-card { padding: 0.7rem 0.8rem; border: 1px solid var(--el-border-color-extra-light); border-radius: 0.6rem; background: var(--el-bg-color); }
.mobile-card-header { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.mobile-card-title { font-size: 0.88rem; font-weight: 700; color: var(--el-text-color-primary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.mobile-card-meta { display: flex; align-items: center; gap: 0.25rem; margin-top: 0.22rem; font-size: 0.72rem; color: var(--el-text-color-secondary); }
.meta-sep { color: var(--el-text-color-placeholder); }
.mobile-card-sid { font-family: monospace; font-size: 0.68rem; color: var(--el-text-color-placeholder); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.mobile-card-footer { margin-top: 0.55rem; padding-top: 0.5rem; border-top: 1px solid var(--el-border-color-extra-light); }
</style>
