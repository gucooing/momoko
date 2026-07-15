<!-- 登录日志：DataTable / 移动卡 + Pagination + 详情 FormDialog -->
<template>
  <AppPanel :title="t('user.tabs.logs')" title-icon="HOutline:ClockIcon">
    <div class="logs-bar">
      <span class="logs-bar__hint">
        {{ t('system.common.total', { total: loginLogsPagination.total }) }}
      </span>
    </div>

    <!-- 桌面 -->
    <DataTable
      v-if="!menuStore.isMobile"
      :columns="columns"
      :rows="(loginLogs as unknown as Record<string, unknown>[])"
      :empty-text="t('user.loginLogsEmpty')"
    >
      <template #cell-success="{ row }">
        <StatusPill
          :variant="row.success ? 'success' : 'error'"
          :label="row.success ? t('common.success') : t('common.failed')"
        />
      </template>
      <template #cell-detail="{ row }">
        <button
          v-if="row.detail"
          type="button"
          class="logs-detail-link"
          @click="openDetail(row as unknown as LoginLogItem)"
        >
          {{ row.detail }}
        </button>
        <span v-else class="text-dim">—</span>
      </template>
    </DataTable>

    <!-- 移动 -->
    <template v-else>
      <EmptyState
        v-if="!loginLogs.length"
        icon="HOutline:ClockIcon"
        :title="t('user.loginLogsEmpty')"
      />
      <div v-else class="logs-cards">
        <EntityCard
          v-for="(row, idx) in loginLogs"
          :key="idx"
          clickable
          @click="openDetail(row)"
        >
          <template #title>
            <span class="logs-ip">{{ row.ip || '-' }}</span>
          </template>
          <template #status>
            <StatusPill
              :variant="row.success ? 'success' : 'error'"
              :label="row.success ? t('common.success') : t('common.failed')"
            />
          </template>
          <template #meta>
            <span>{{ row.operationTime || '-' }}</span>
            <span class="logs-ua" :title="row.userAgent">{{ row.userAgent || '-' }}</span>
            <span v-if="row.detail" class="logs-hint">{{ t('user.detailHint') }}</span>
          </template>
        </EntityCard>
      </div>
    </template>

    <Pagination
      v-if="loginLogsPagination.total > 0"
      :page="loginLogsPagination.page"
      :page-size="loginLogsPagination.pageSize"
      :total="loginLogsPagination.total"
      @update:page="onPage"
      @update:page-size="onPageSize"
    />

    <FormDialog
      v-model="detailVisible"
      :title="t('user.loginDetail')"
      :width="560"
      @confirm="detailVisible = false"
    >
      <template #footer="{ close }">
        <UButton color="primary" @click="close">{{ t('common.close') }}</UButton>
      </template>
      <pre v-if="isDetailJson" class="detail-json">{{ detailContent }}</pre>
      <div v-else class="detail-text">{{ detailContent }}</div>
    </FormDialog>
  </AppPanel>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useUserProfileStore } from '@/stores/user/profile'
import type { LoginLogItem } from '@/stores/user/types'
import { useI18n } from 'vue-i18n'

const menuStore = useMenuStore()
const userProfileStore = useUserProfileStore()
const { loginLogs, loginLogsPagination } = storeToRefs(userProfileStore)
const { t } = useI18n()

const columns = computed(() => [
  { key: 'ip', title: t('user.extra.ipAddress'), minWidth: 130 },
  { key: 'userAgent', title: 'User Agent', minWidth: 240 },
  { key: 'operationTime', title: t('user.operationTime'), width: 160 },
  { key: 'success', title: t('user.result'), width: 90 },
  { key: 'detail', title: t('user.detail'), minWidth: 160 },
])

const detailVisible = ref(false)
const detailContent = ref('')
const isDetailJson = ref(false)

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

const onPage = (page: number) => {
  loginLogsPagination.value.page = page
  void userProfileStore.getMyLoginLogs()
}
const onPageSize = (size: number) => {
  loginLogsPagination.value.pageSize = size
  loginLogsPagination.value.page = 1
  void userProfileStore.getMyLoginLogs()
}

onMounted(() => {
  void userProfileStore.getMyLoginLogs()
})
</script>

<style scoped lang="scss">
.logs-bar {
  display: flex;
  align-items: center;
  margin-bottom: 0.5rem;
}
.logs-bar__hint {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}
.logs-detail-link {
  display: block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  border: 0;
  background: none;
  padding: 0;
  font: inherit;
  color: var(--el-color-primary);
  cursor: pointer;
  text-align: left;
}
.text-dim {
  color: var(--el-text-color-placeholder);
}
.logs-cards {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
.logs-ip {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-weight: 600;
}
.logs-ua {
  display: block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}
.logs-hint {
  font-size: 0.75rem;
  color: var(--el-color-primary);
}
.detail-json {
  margin: 0;
  padding: 0.75rem 0.85rem;
  background: var(--el-fill-color-light);
  border-radius: var(--app-radius);
  font-size: 0.8rem;
  font-family: ui-monospace, 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
  line-height: 1.55;
  color: var(--el-text-color-primary);
  overflow-x: auto;
  max-height: 55vh;
}
.detail-text {
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 0.875rem;
  line-height: 1.5;
  color: var(--el-text-color-regular);
}
</style>
