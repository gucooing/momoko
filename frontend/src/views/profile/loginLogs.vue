<!-- 最近登录：卡片显示最近 5 次 + 成功/失败统计；「全部记录」弹窗内按结果筛选、分页、看详情。 -->
<template>
  <AppPanel class="logs">
    <template #header>
      <h2 class="pf-side-title">{{ t('profile.logs.title') }}</h2>
    </template>
    <template #actions>
      <button type="button" class="pf-link" @click="openAll">{{ t('profile.logs.all') }} →</button>
    </template>

    <div class="logs-stats m-num">
      <span class="logs-stat"><i class="is-ok" />{{ t('profile.logs.successN', { n: stats.success }) }}</span>
      <span class="logs-stat"><i class="is-bad" />{{ t('profile.logs.failedN', { n: stats.failed }) }}</span>
    </div>

    <EmptyState v-if="!recent.length" icon="HOutline:ClockIcon" :title="t('user.loginLogsEmpty')" />
    <div v-else class="logs-list">
      <button
        v-for="(row, idx) in recent"
        :key="idx"
        type="button"
        class="logs-row"
        @click="openDetail(row)"
      >
        <span class="logs-row__main">
          <span class="logs-row__time m-num">{{ relTime(row.time) }}</span>
          <span class="logs-row__meta">
            <span class="m-mono">{{ row.ip || '—' }}</span>
            <template v-if="scopeLabel(row.ipScope)"> · {{ scopeLabel(row.ipScope) }}</template>
            <template v-if="compact"> · {{ methodLabel(row.method) }}</template>
          </span>
        </span>
        <span v-if="!compact" class="logs-tag">{{ methodLabel(row.method) }}</span>
        <StatusPill
          :variant="row.success ? 'success' : 'error'"
          :label="row.success ? t('common.success') : t('common.failed')"
        />
      </button>
    </div>

    <!-- 全部记录 -->
    <FormDialog v-model="allOpen" :title="t('user.tabs.logs')" :width="760" :show-footer="false">
      <div class="logs-chips" role="tablist">
        <button
          v-for="chip in chips"
          :key="chip.key"
          type="button"
          role="tab"
          class="logs-chip"
          :class="{ 'is-on': loginLogsFilter === chip.key }"
          :aria-selected="loginLogsFilter === chip.key"
          @click="setFilter(chip.key)"
        >
          {{ chip.label }}<span class="logs-chip__n m-num">{{ chip.count }}</span>
        </button>
      </div>

      <div v-if="loginLogsLoading && !loginLogs.length" class="logs-skel" />
      <EmptyState v-else-if="!loginLogs.length" icon="HOutline:ClockIcon" :title="t('user.loginLogsEmpty')" />
      <div v-else class="logs-full">
        <button
          v-for="(row, idx) in loginLogs"
          :key="idx"
          type="button"
          class="logs-full__row"
          @click="openDetail(row)"
        >
          <span class="m-sticker logs-full__sk" :class="row.success ? 'm-sticker--ok' : 'm-sticker--bad'" aria-hidden="true">
            <UIcon :name="deviceIcon(row.deviceType)" />
          </span>
          <span class="logs-full__main">
            <span class="logs-full__title">
              <span class="m-num">{{ fullTime(row.time) }}</span>
              <StatusPill
                :variant="row.success ? 'success' : 'error'"
                :label="row.success ? t('common.success') : t('common.failed')"
              />
            </span>
            <span class="logs-full__meta">
              <span class="m-mono">{{ row.ip || '—' }}</span>
              <template v-if="scopeLabel(row.ipScope)"> · {{ scopeLabel(row.ipScope) }}</template>
              · {{ methodLabel(row.method) }} · {{ deviceName(row.device) }}
            </span>
            <span v-if="!row.success && row.error" class="logs-full__err">{{ row.error }}</span>
          </span>
          <UIcon name="i-lucide-chevron-right" class="logs-full__chev" aria-hidden="true" />
        </button>
      </div>

      <div v-if="loginLogsPagination.total > loginLogsPagination.pageSize" class="logs-pager">
        <Pagination
          :page="loginLogsPagination.page"
          :page-size="loginLogsPagination.pageSize"
          :total="loginLogsPagination.total"
          @update:page="onPage"
          @update:page-size="onPageSize"
        />
      </div>
    </FormDialog>

    <!-- 单条详情 -->
    <FormDialog v-model="detailVisible" :title="t('user.loginDetail')" :width="560">
      <template #footer="{ close }">
        <UButton color="primary" @click="close">{{ t('common.close') }}</UButton>
      </template>
      <template v-if="detailRow">
        <dl class="logs-detail">
          <div><dt>{{ t('user.operationTime') }}</dt><dd class="m-num">{{ fullTime(detailRow.time) }}</dd></div>
          <div><dt>{{ t('user.result') }}</dt><dd>{{ detailRow.success ? t('common.success') : t('common.failed') }}</dd></div>
          <div><dt>{{ t('user.extra.ipAddress') }}</dt><dd class="m-mono">{{ detailRow.ip || '—' }}<template v-if="scopeLabel(detailRow.ipScope)"> · {{ scopeLabel(detailRow.ipScope) }}</template></dd></div>
          <div><dt>{{ t('profile.devices.loginMethod') }}</dt><dd>{{ methodLabel(detailRow.method) }}</dd></div>
          <div class="is-wide"><dt>{{ t('profile.logs.device') }}</dt><dd>{{ deviceName(detailRow.device) }} · {{ deviceTypeLabel(detailRow.deviceType) }}</dd></div>
          <div v-if="detailRow.error" class="is-wide"><dt>{{ t('profile.logs.reason') }}</dt><dd class="is-bad">{{ detailRow.error }}</dd></div>
          <div class="is-wide"><dt>User Agent</dt><dd class="m-mono">{{ detailRow.userAgent || '—' }}</dd></div>
        </dl>
        <details v-if="detailJson" class="logs-raw">
          <summary>{{ t('user.detail') }}</summary>
          <pre class="detail-json">{{ detailJson }}</pre>
        </details>
      </template>
    </FormDialog>
  </AppPanel>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { myLoginLogs } from '@/api/login'
import { useUserProfileStore } from '@/stores/user/profile'
import type { LoginLogItem } from '@/stores/user/types'
import { useI18n } from 'vue-i18n'
import { useProfileFormat } from './components/useProfileFormat'

const menuStore = useMenuStore()
const userProfileStore = useUserProfileStore()
const { loginLogs, loginLogsPagination, loginLogsFilter, loginLogsStats, loginLogsLoading } =
  storeToRefs(userProfileStore)
const { t } = useI18n()
const { relTime, fullTime, scopeLabel, methodLabel, deviceIcon, deviceName, deviceTypeLabel } =
  useProfileFormat()

const compact = computed(() => menuStore.isMobile)
const recent = ref<LoginLogItem[]>([])
const stats = ref({ success: 0, failed: 0 })

const loadRecent = async () => {
  try {
    const { data } = await myLoginLogs({ page: 1, pageSize: 5 })
    recent.value = (data?.logs || []).map((log) => ({
      time: log.time ? String(log.time) : '',
      ip: log.ip,
      ipScope: log.ipScope,
      userAgent: log.userAgent,
      device: log.device,
      deviceType: log.deviceType,
      method: log.method,
      success: log.success,
      error: log.error,
      detail: log.detail,
    }))
    stats.value = { success: Number(data?.successCount || 0), failed: Number(data?.failedCount || 0) }
  } catch {
    recent.value = []
  }
}

const chips = computed(() => [
  {
    key: 'all' as const,
    label: t('profile.logs.allChip'),
    count: loginLogsStats.value.success + loginLogsStats.value.failed,
  },
  { key: 'success' as const, label: t('common.success'), count: loginLogsStats.value.success },
  { key: 'failed' as const, label: t('common.failed'), count: loginLogsStats.value.failed },
])

const allOpen = ref(false)
const openAll = () => {
  allOpen.value = true
  loginLogsPagination.value.page = 1
  void userProfileStore.getMyLoginLogs()
}
const setFilter = (key: 'all' | 'success' | 'failed') => {
  loginLogsFilter.value = key
  loginLogsPagination.value.page = 1
  void userProfileStore.getMyLoginLogs()
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

const detailVisible = ref(false)
const detailRow = ref<LoginLogItem | null>(null)
const detailJson = computed(() => {
  const raw = detailRow.value?.detail
  if (!raw) return ''
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
})
const openDetail = (row: LoginLogItem) => {
  detailRow.value = row
  detailVisible.value = true
}

onMounted(() => {
  void loadRecent()
})
</script>

<style scoped lang="scss">
.logs :deep(.app-panel__body) {
  padding-top: 10px;
  padding-bottom: 10px;
}
.pf-side-title {
  font-family: var(--app-font-display);
  font-size: 1.125rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.pf-link {
  min-height: 36px;
  padding: 0 2px;
  border: 0;
  background: none;
  color: var(--m-pInk);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
}
.pf-link:hover {
  color: var(--el-text-color-primary);
}
.logs-stats {
  display: flex;
  gap: 16px;
  margin-bottom: 4px;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.logs-stat {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  i {
    width: 8px;
    height: 8px;
    border-radius: 50%;
  }
  .is-ok {
    background: var(--m-ok);
  }
  .is-bad {
    background: var(--m-bad);
  }
}
.logs-list {
  display: flex;
  flex-direction: column;
}
.logs-row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: 58px;
  padding: 8px 0;
  border: 0;
  border-top: 1px solid var(--el-border-color-light);
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.logs-row:hover .logs-row__time {
  color: var(--m-pInk);
}
.logs-row__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.logs-row__time {
  font-size: 0.875rem;
  font-weight: 700;
}
.logs-row__meta {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.logs-tag {
  flex: none;
  padding: 3px 10px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  font-weight: 700;
}
.logs-chips {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  overflow-x: auto;
  scrollbar-width: none;
  &::-webkit-scrollbar {
    display: none;
  }
}
.logs-chip {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  padding: 0 14px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 999px;
  background: var(--el-bg-color);
  color: var(--el-text-color-regular);
  font: inherit;
  font-size: 0.875rem;
  font-weight: 700;
  cursor: pointer;
}
.logs-chip.is-on {
  border-color: var(--m-ink);
  background: var(--m-ink);
  color: var(--m-bg);
}
.logs-chip__n {
  font-size: 0.75rem;
  opacity: 0.8;
}
.logs-skel {
  height: 180px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.logs-full {
  display: flex;
  flex-direction: column;
}
.logs-full__row {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  min-height: 64px;
  padding: 10px 4px;
  border: 0;
  border-top: 1px solid var(--el-border-color-light);
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.logs-full__row:first-child {
  border-top: 0;
}
.logs-full__row:hover {
  background: color-mix(in srgb, var(--el-fill-color-light) 60%, transparent);
}
.logs-full__sk {
  width: 38px;
  height: 38px;
  border-radius: 13px;
  font-size: 18px;
}
.logs-full__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.logs-full__title {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 0.875rem;
  font-weight: 700;
}
.logs-full__meta {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow-wrap: anywhere;
}
.logs-full__err {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--m-bad);
}
.logs-full__chev {
  flex: none;
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.logs-pager {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-light);
}
.logs-detail {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px 16px;
  margin: 0;
  padding: 14px 16px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  > div {
    min-width: 0;
  }
  .is-wide {
    grid-column: 1 / -1;
  }
  dt {
    font-size: 0.75rem;
    color: var(--el-text-color-secondary);
  }
  dd {
    margin: 2px 0 0;
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--el-text-color-primary);
    overflow-wrap: anywhere;
  }
  .is-bad {
    color: var(--m-bad);
  }
}
.logs-raw {
  margin-top: 12px;
  summary {
    min-height: 36px;
    display: flex;
    align-items: center;
    font-size: 0.8125rem;
    font-weight: 700;
    color: var(--m-pInk);
    cursor: pointer;
  }
}
.detail-json {
  margin: 0;
  padding: 0.75rem 0.85rem;
  background: var(--el-fill-color-light);
  border-radius: var(--app-radius);
  font-size: 0.78rem;
  font-family: var(--app-font-mono);
  line-height: 1.55;
  color: var(--el-text-color-primary);
  overflow-x: auto;
  max-height: 45vh;
}
@media (width < 640px) {
  .logs-detail {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
