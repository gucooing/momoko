<!-- 登录设备 -->
<template>
  <BaseCard>
    <template #header>
      <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div class="space-y-1">
          <h1 class="text-2xl font-semibold">登录设备</h1>
        </div>
        <div class="flex items-center gap-2">
          <BaseTag type="info" :text="`总计 ${loginDevices.length}`" />
          <BaseTag type="primary" :text="`当前设备 ${currentDeviceLabel}`" />
        </div>
      </div>
    </template>

    <el-empty v-if="!loading && !loginDevices.length" description="暂无登录设备" />

    <!-- desktop: table -->
    <el-table
      v-else-if="!menuStore.isMobile"
      v-loading="loading"
      :data="loginDevices"
      :border="TABLE_CONFIG.border"
      table-layout="auto"
      class="custom-modern-table"
    >
      <el-table-column label="登录设备" :min-width="deviceColumnWidth" class-name="auto-wrap-cell">
        <template #default="{ row }">
          <div class="device-cell">
            <span class="device-text">{{ row.device || '-' }}</span>
            <BaseTag v-if="isCurrentDevice(row)" type="success" text="当前设备" />
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="ip" label="IP 地址" :min-width="ipColumnWidth" class-name="auto-wrap-cell" />
      <el-table-column label="登录时间" min-width="170">
        <template #default="{ row }">{{ formatDateTime(row.loginTime) }}</template>
      </el-table-column>
      <el-table-column label="最近刷新" min-width="170">
        <template #default="{ row }">{{ formatDateTime(row.updateTime) }}</template>
      </el-table-column>
      <el-table-column prop="sessionId" label="会话 ID" :min-width="sessionIdColumnWidth" class-name="auto-wrap-cell" />
      <el-table-column v-for="key in extraColumns" :key="key" :label="getExtraColumnLabel(key)" :min-width="getExtraColumnWidth(key)" class-name="auto-wrap-cell">
        <template #default="{ row }">{{ formatCellValue(row[key]) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="110">
        <template #default="{ row }">
          <span v-if="isCurrentDevice(row)" class="text-(--el-text-color-secondary)">-</span>
          <AdaptiveConfirm v-else title="确定删除该登录设备吗？" @confirm="handleDeleteDevice(row)" :disabled="isDeletingLoginDevice(row.deviceId)">
            <template #reference>
              <el-button type="danger" link :loading="isDeletingLoginDevice(row.deviceId)">删除</el-button>
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
          <BaseTag v-if="isCurrentDevice(row)" type="success" text="当前" />
        </div>
        <div class="mobile-card-meta">
          <span>{{ row.ip || '-' }}</span>
        </div>
        <div class="mobile-card-meta">
          <span>登录: {{ formatDateTime(row.loginTime) }}</span>
          <span class="meta-sep">·</span>
          <span>刷新: {{ formatDateTime(row.updateTime) }}</span>
        </div>
        <div class="mobile-card-meta mobile-card-sid">{{ row.sessionId || '-' }}</div>
        <div v-if="!isCurrentDevice(row)" class="mobile-card-footer">
          <AdaptiveConfirm title="确定删除该登录设备吗？" @confirm="handleDeleteDevice(row)" :disabled="isDeletingLoginDevice(row.deviceId)">
            <template #reference>
              <el-button size="small" plain type="danger" :loading="isDeletingLoginDevice(row.deviceId)">删除</el-button>
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
const menuStore = useMenuStore()
const userProfileStore = useUserProfileStore()
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
    ElMessage.success('设备已删除')
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
