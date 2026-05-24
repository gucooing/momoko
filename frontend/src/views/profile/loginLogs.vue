<!-- 登录日志 -->
<template>
  <BaseCard>
    <el-empty v-if="!loginLogs.length" description="暂无登录日志" />

    <!-- desktop: table -->
    <div v-else-if="!menuStore.isMobile">
      <el-table
        :data="loginLogs"
        :border="TABLE_CONFIG.border"
        show-overflow-tooltip
        class="custom-modern-table"
      >
        <el-table-column prop="ip" label="IP 地址" min-width="150" />
        <el-table-column prop="userAgent" label="User Agent" min-width="280" />
        <el-table-column prop="operationTime" label="操作时间" min-width="170" />
        <el-table-column label="结果" width="80">
          <template #default="{ row }">
            <BaseTag :type="row.success ? 'success' : 'danger'" :text="row.success ? '成功' : '失败'" />
          </template>
        </el-table-column>
        <el-table-column label="详情" min-width="160" :show-overflow-tooltip="false">
          <template #default="{ row }">
            <span v-if="row.detail" class="detail-cell" @click.stop="openDetail(row)">{{ row.detail }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
      </el-table>

      <TablePagination
        v-model:current-page="loginLogsPagination.page"
        v-model:page-size="loginLogsPagination.pageSize"
        :total="loginLogsPagination.total"
        :is-mobile="false"
        @change="userProfileStore.getMyLoginLogs"
      />
    </div>

    <!-- mobile: cards -->
    <div v-else class="mobile-card-list">
      <div v-for="(row, idx) in loginLogs" :key="idx" class="mobile-card" @click="openDetail(row)">
        <div class="mobile-card-header">
          <span class="mobile-card-title">{{ row.ip || '-' }}</span>
          <BaseTag :type="row.success ? 'success' : 'danger'" :text="row.success ? '成功' : '失败'" />
        </div>
        <div class="mobile-card-meta">
          <span>{{ formatTime(row.operationTime) }}</span>
        </div>
        <div class="mobile-card-meta mobile-card-ua">{{ row.userAgent || '-' }}</div>
        <div v-if="row.detail" class="mobile-card-detail-hint">点击查看详情</div>
      </div>

      <TablePagination
        v-model:current-page="loginLogsPagination.page"
        v-model:page-size="loginLogsPagination.pageSize"
        :total="loginLogsPagination.total"
        :is-mobile="true"
        @change="userProfileStore.getMyLoginLogs"
      />
    </div>

    <BaseDialog v-model="detailVisible" title="登录详情" width="560">
      <el-scrollbar max-height="60vh">
        <pre v-if="isDetailJson" class="detail-json">{{ detailContent }}</pre>
        <div v-else class="detail-text">{{ detailContent }}</div>
      </el-scrollbar>
      <template #footer>
        <el-button type="primary" @click="detailVisible = false">关闭</el-button>
      </template>
    </BaseDialog>
  </BaseCard>
</template>

<script setup lang="ts">
import dayjs from 'dayjs'
import { useUserProfileStore } from '@/stores/user/profile'
import { TABLE_CONFIG } from '@/config/elementConfig'
import TablePagination from '@/components/pagination/TablePagination.vue'
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import type { LoginLogItem } from '@/stores/user/types'

const menuStore = useMenuStore()
const userProfileStore = useUserProfileStore()
const { loginLogs, loginLogsPagination } = storeToRefs(userProfileStore)

const detailVisible = ref(false)
const detailContent = ref('')
const isDetailJson = ref(false)

const formatTime = (value: unknown) => {
  if (!value) return '-'
  const d = dayjs(value as string | number | Date)
  return d.isValid() ? d.format('YYYY-MM-DD HH:mm:ss') : String(value)
}

const openDetail = (row: LoginLogItem) => {
  const detail = row.detail
  if (!detail) {
    detailContent.value = '-'
    isDetailJson.value = false
    detailVisible.value = true
    return
  }

  try {
    const parsed = JSON.parse(detail)
    detailContent.value = JSON.stringify(parsed, null, 2)
    isDetailJson.value = true
  } catch {
    detailContent.value = detail
    isDetailJson.value = false
  }
  detailVisible.value = true
}

onMounted(() => {
  userProfileStore.getMyLoginLogs()
})
</script>

<style scoped lang="scss">
.mobile-card-list {
  display: flex;
  flex-direction: column;
  gap: 0.55rem;
}

.mobile-card {
  padding: 0.7rem 0.8rem;
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: 0.6rem;
  background: var(--el-bg-color);

  &:active {
    background: var(--el-fill-color-light);
  }
}

.mobile-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.mobile-card-title {
  font-size: 0.9rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  font-family: monospace;
}

.mobile-card-meta {
  margin-top: 0.22rem;
  font-size: 0.72rem;
  color: var(--el-text-color-secondary);
}

.mobile-card-ua {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--el-text-color-placeholder);
}

.mobile-card-detail-hint {
  margin-top: 0.35rem;
  font-size: 0.68rem;
  color: var(--el-color-primary);
}

.detail-json {
  margin: 0;
  padding: 0.75rem 1rem;
  background: color-mix(in srgb, var(--el-fill-color) 60%, transparent);
  border-radius: 0.5rem;
  font-size: 0.8rem;
  font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
  line-height: 1.55;
  color: var(--el-text-color-primary);
  overflow-x: auto;
}

.detail-cell {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: pointer;
}

.detail-text {
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
