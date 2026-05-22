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
    <el-table
      v-else
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

      <el-table-column
        prop="ip"
        label="IP 地址"
        :min-width="ipColumnWidth"
        class-name="auto-wrap-cell"
      />

      <el-table-column label="登录时间" min-width="170">
        <template #default="{ row }">
          {{ formatDateTime(row.loginTime) }}
        </template>
      </el-table-column>

      <el-table-column label="最近刷新" min-width="170">
        <template #default="{ row }">
          {{ formatDateTime(row.updateTime) }}
        </template>
      </el-table-column>

      <el-table-column
        prop="sessionId"
        label="会话 ID"
        :min-width="sessionIdColumnWidth"
        class-name="auto-wrap-cell"
      />

      <el-table-column
        v-for="key in extraColumns"
        :key="key"
        :label="getExtraColumnLabel(key)"
        :min-width="getExtraColumnWidth(key)"
        class-name="auto-wrap-cell"
      >
        <template #default="{ row }">
          {{ formatCellValue(row[key]) }}
        </template>
      </el-table-column>

      <el-table-column label="操作" width="110">
        <template #default="{ row }">
          <span v-if="isCurrentDevice(row)" class="text-(--el-text-color-secondary)">-</span>
          <AdaptiveConfirm
            v-else
            title="确定删除该登录设备吗？"
            @confirm="handleDeleteDevice(row)"
            :disabled="isDeletingLoginDevice(row.deviceId)"
          >
            <template #reference>
              <el-button type="danger" link :loading="isDeletingLoginDevice(row.deviceId)">
                删除
              </el-button>
            </template>
          </AdaptiveConfirm>
        </template>
      </el-table-column>
    </el-table>
  </BaseCard>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { TABLE_CONFIG } from '@/config/elementConfig'
import { useUserProfileStore } from '@/stores/user/profile'
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
</style>
