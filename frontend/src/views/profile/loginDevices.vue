<!-- 登录设备：紧凑行列表；点击行看详情；下线 AdaptiveConfirm（阻止冒泡）。 -->
<template>
  <AppPanel :title="t('user.loginDevices')" title-icon="HOutline:DevicePhoneMobileIcon" :padded="false">
    <template #actions>
      <StatusPill
        variant="info"
        :dot="false"
        :label="t('user.totalCount', { count: loginDevices.length })"
      />
    </template>

    <div v-if="loading" class="dev-list">
      <div v-for="i in 5" :key="i" class="dev-skel" />
    </div>
    <EmptyState
      v-else-if="!loginDevices.length"
      icon="HOutline:DevicePhoneMobileIcon"
      :title="t('user.noLoginDevices')"
      class="dev-empty"
    />
    <ul v-else class="dev-list" role="list">
      <li
        v-for="row in loginDevices"
        :key="row.sessionId || row.deviceId"
        class="dev-row"
        :class="{ 'is-current': isCurrentDevice(row) }"
        role="button"
        tabindex="0"
        @click="openDetail(row)"
        @keydown.enter.prevent="openDetail(row)"
      >
        <div class="dev-row__main">
          <div class="dev-row__title">
            <span class="dev-row__name" :title="row.device || undefined">{{ row.device || '—' }}</span>
            <StatusPill
              v-if="isCurrentDevice(row)"
              variant="success"
              :dot="false"
              :label="t('user.currentDevice')"
            />
          </div>
          <div class="dev-row__meta">
            <span class="dev-row__mono">{{ row.ip || '—' }}</span>
            <span class="dev-row__sep">·</span>
            <span>{{ formatDateTime(row.loginTime) }}</span>
            <span class="dev-row__sep">·</span>
            <span class="dev-row__muted">{{ formatDateTime(row.updateTime) }}</span>
          </div>
        </div>
        <div class="dev-row__action" @click.stop>
          <span v-if="isCurrentDevice(row)" class="dev-row__hint">{{ t('user.thisSession') }}</span>
          <AdaptiveConfirm
            v-else
            :title="t('user.deleteDeviceConfirm')"
            :disabled="isDeletingLoginDevice(row.sessionId)"
            @confirm="handleDelete(row.sessionId)"
          >
            <template #reference>
              <UButton
                color="error"
                variant="ghost"
                size="xs"
                :loading="isDeletingLoginDevice(row.sessionId)"
              >
                {{ t('user.logoutDevice') }}
              </UButton>
            </template>
          </AdaptiveConfirm>
        </div>
        <component
          :is="menuStore.iconComponents['HOutline:ChevronRightIcon']"
          class="dev-row__chev"
          aria-hidden="true"
        />
      </li>
    </ul>

    <!-- 设备详情 -->
    <FormDialog
      v-model="detailOpen"
      :title="t('user.deviceDetail')"
      :width="480"
      @confirm="detailOpen = false"
    >
      <template #footer="{ close }">
        <UButton
          v-if="detailRow && !isCurrentDevice(detailRow)"
          color="error"
          variant="soft"
          :loading="detailRow ? isDeletingLoginDevice(detailRow.sessionId) : false"
          @click="confirmLogoutFromDetail"
        >
          {{ t('user.logoutDevice') }}
        </UButton>
        <UButton color="primary" @click="close">{{ t('common.close') }}</UButton>
      </template>

      <DescriptionList v-if="detailItems.length" :items="detailItems" :columns="1" />
      <p v-else class="dev-detail-empty">—</p>
    </FormDialog>
  </AppPanel>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useUserProfileStore } from '@/stores/user/profile'
import type { LoginDeviceRow } from '@/stores/user/types'
import { useFeedback } from '@/utils/feedback'
import { useI18n } from 'vue-i18n'

const userProfileStore = useUserProfileStore()
const menuStore = useMenuStore()
const { t } = useI18n()
const fb = useFeedback()

const { loading, loginDevices, extraColumns } = storeToRefs(userProfileStore)
const {
  formatDateTime,
  formatCellValue,
  getExtraColumnLabel,
  isCurrentDevice,
  isDeletingLoginDevice,
  getLoginDevices,
  deleteLoginDevice,
} = userProfileStore

const detailOpen = ref(false)
const detailRow = ref<LoginDeviceRow | null>(null)

const detailItems = computed(() => {
  const row = detailRow.value
  if (!row) return [] as { label: string; value?: string | number | null }[]

  const items: { label: string; value?: string | number | null }[] = [
    { label: t('user.loginDevices'), value: row.device || '—' },
    {
      label: t('user.currentDevice'),
      value: isCurrentDevice(row) ? t('common.yes') : t('common.no'),
    },
    { label: t('user.extra.ipAddress'), value: row.ip || '—' },
    { label: t('user.loginTime'), value: formatDateTime(row.loginTime) },
    { label: t('user.latestRefresh'), value: formatDateTime(row.updateTime) },
    { label: t('user.sessionId'), value: row.sessionId || '—' },
    { label: t('user.deviceId'), value: row.deviceId || '—' },
  ]

  // 额外字段（browser/os 等），策展展示有值的
  for (const key of extraColumns.value) {
    const raw = (row as Record<string, unknown>)[key]
    if (raw === null || raw === undefined || raw === '') continue
    items.push({
      label: getExtraColumnLabel(key),
      value: formatCellValue(raw),
    })
  }

  return items
})

const openDetail = (row: LoginDeviceRow) => {
  detailRow.value = row
  detailOpen.value = true
}

const handleDelete = async (sessionId: string) => {
  if (!sessionId) return
  const ok = await deleteLoginDevice(sessionId)
  if (ok) {
    fb.success(t('user.deviceDeleted'))
    if (detailRow.value?.sessionId === sessionId) {
      detailOpen.value = false
      detailRow.value = null
    }
  }
}

const confirmLogoutFromDetail = async () => {
  if (!detailRow.value?.sessionId) return
  // 详情内下线：按 sessionId 调用 logout（后端已取消 DELETE devices）。
  await handleDelete(detailRow.value.sessionId)
}

onMounted(() => {
  void getLoginDevices()
})
</script>

<style scoped lang="scss">
.dev-empty {
  padding: 1.25rem 1rem;
}
.dev-list {
  list-style: none;
  margin: 0;
  padding: 0;
  min-width: 0;
}
.dev-skel {
  height: 44px;
  margin: 0 12px;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  background: linear-gradient(
    90deg,
    var(--el-fill-color-light) 25%,
    var(--el-fill-color) 50%,
    var(--el-fill-color-light) 75%
  );
  background-size: 200% 100%;
  animation: shimmer 1.2s infinite;
}
@keyframes shimmer {
  0% {
    background-position: 200% 0;
  }
  100% {
    background-position: -200% 0;
  }
}

.dev-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  min-height: 44px;
  padding: 8px 12px 8px 16px;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  min-width: 0;
  cursor: pointer;
}
.dev-row:last-child {
  border-bottom: 0;
}
.dev-row:hover {
  background: var(--el-fill-color-lighter);
}
.dev-row.is-current {
  background: color-mix(in srgb, var(--el-color-primary) 5%, transparent);
}
.dev-row__main {
  min-width: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.dev-row__title {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  min-width: 0;
}
.dev-row__name {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.dev-row__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.2rem 0.35rem;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
  min-width: 0;
}
.dev-row__mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}
.dev-row__muted {
  color: var(--el-text-color-placeholder);
}
.dev-row__sep {
  color: var(--el-text-color-placeholder);
}
.dev-row__action {
  flex-shrink: 0;
  display: flex;
  align-items: center;
}
.dev-row__hint {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}
.dev-row__chev {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  color: var(--el-text-color-placeholder);
}
.dev-detail-empty {
  margin: 0;
  color: var(--el-text-color-placeholder);
  font-size: 0.875rem;
}

@media (width < 640px) {
  .dev-row {
    padding: 10px 10px 10px 14px;
  }
}
</style>
