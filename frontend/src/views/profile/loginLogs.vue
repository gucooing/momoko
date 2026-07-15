<!-- 登录日志：桌面 DataTable 紧凑行；移动同密度列表行（非大卡）。详情 FormDialog。 -->
<template>
  <AppPanel :title="t('user.tabs.logs')" title-icon="HOutline:ClockIcon" :padded="false">
    <template #actions>
      <span class="logs-total">{{ t('system.common.total', { total: loginLogsPagination.total }) }}</span>
    </template>

    <!-- 桌面：表 -->
    <div v-if="!menuStore.isMobile" class="logs-table-wrap">
      <DataTable
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
            {{ summarizeDetail(String(row.detail)) }}
          </button>
          <span v-else class="text-dim">—</span>
        </template>
      </DataTable>
    </div>

    <!-- 移动：紧凑行（同设备列表密度） -->
    <template v-else>
      <EmptyState
        v-if="!loginLogs.length"
        icon="HOutline:ClockIcon"
        :title="t('user.loginLogsEmpty')"
        class="logs-empty"
      />
      <ul v-else class="logs-list" role="list">
        <li
          v-for="(row, idx) in loginLogs"
          :key="idx"
          class="logs-row"
          role="button"
          tabindex="0"
          @click="openDetail(row)"
          @keydown.enter="openDetail(row)"
        >
          <div class="logs-row__main">
            <div class="logs-row__title">
              <span class="logs-row__ip">{{ row.ip || '—' }}</span>
              <StatusPill
                :variant="row.success ? 'success' : 'error'"
                :label="row.success ? t('common.success') : t('common.failed')"
              />
            </div>
            <div class="logs-row__meta">
              <span>{{ row.operationTime || '—' }}</span>
              <span class="logs-row__sep">·</span>
              <span class="logs-row__ua" :title="row.userAgent">{{ row.userAgent || '—' }}</span>
            </div>
          </div>
          <component
            :is="menuStore.iconComponents['HOutline:ChevronRightIcon']"
            class="logs-row__chev"
          />
        </li>
      </ul>
    </template>

    <div v-if="loginLogsPagination.total > 0" class="logs-pager">
      <Pagination
        :page="loginLogsPagination.page"
        :page-size="loginLogsPagination.pageSize"
        :total="loginLogsPagination.total"
        @update:page="onPage"
        @update:page-size="onPageSize"
      />
    </div>

    <FormDialog v-model="detailVisible" :title="t('user.loginDetail')" :width="560">
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
  { key: 'ip', title: t('user.extra.ipAddress'), minWidth: 110 },
  { key: 'userAgent', title: 'User Agent', minWidth: 200 },
  { key: 'operationTime', title: t('user.operationTime'), width: 150 },
  { key: 'success', title: t('user.result'), width: 80 },
  { key: 'detail', title: t('user.detail'), minWidth: 140 },
])

const detailVisible = ref(false)
const detailContent = ref('')
const isDetailJson = ref(false)

const summarizeDetail = (detail: string) => {
  try {
    const parsed = JSON.parse(detail) as { path?: string; operation?: string }
    return parsed.path || parsed.operation || detail.slice(0, 48)
  } catch {
    return detail.length > 48 ? `${detail.slice(0, 46)}…` : detail
  }
}

const openDetail = (row: LoginLogItem) => {
  const detail = row.detail
  if (!detail) {
    detailContent.value = '—'
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
.logs-total {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}
.logs-table-wrap {
  min-width: 0;
  /* flush 面板内表贴边，与设备行一致 */
  :deep(.data-table__scroll) {
    border: 0;
    border-radius: 0;
  }
}
.logs-empty {
  padding: 1.25rem 1rem;
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

.logs-list {
  list-style: none;
  margin: 0;
  padding: 0;
}
.logs-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  min-height: 44px;
  padding: 8px 14px;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  cursor: pointer;
}
.logs-row:last-child {
  border-bottom: 0;
}
.logs-row:hover,
.logs-row:active {
  background: var(--el-fill-color-lighter);
}
.logs-row__main {
  min-width: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.logs-row__title {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}
.logs-row__ip {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.logs-row__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.2rem 0.35rem;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
  min-width: 0;
}
.logs-row__sep {
  color: var(--el-text-color-placeholder);
}
.logs-row__ua {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
  color: var(--el-text-color-placeholder);
}
.logs-row__chev {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  color: var(--el-text-color-placeholder);
}
.logs-pager {
  padding: 8px 12px 12px;
  border-top: 1px solid var(--el-border-color-extra-light);
}

.detail-json {
  margin: 0;
  padding: 0.75rem 0.85rem;
  background: var(--el-fill-color-light);
  border-radius: var(--app-radius);
  font-size: 0.8rem;
  font-family: ui-monospace, 'SF Mono', 'Fira Code', monospace;
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
