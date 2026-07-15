<!-- 登录设备（06a）：EntityCard 流（桌面网格 + 移动单列）；下线用 AdaptiveConfirm -->
<template>
  <AppPanel :title="t('user.loginDevices')" title-icon="HOutline:DevicePhoneMobileIcon">
    <template #actions>
      <!-- 头部只放总数；当前设备长串放到 body 提示，避免窄屏撑破面板头 -->
      <StatusPill
        variant="info"
        :dot="false"
        :label="t('user.totalCount', { count: loginDevices.length })"
      />
    </template>

    <p
      v-if="currentDeviceLabel && currentDeviceLabel !== '-'"
      class="dev-current"
      :title="currentDeviceLabel"
    >
      {{ t('user.currentDevice') }} · {{ shortDevice(currentDeviceLabel) }}
    </p>

    <div v-if="loading" class="dev-grid">
      <div v-for="i in 4" :key="i" class="dev-skeleton" />
    </div>
    <EmptyState
      v-else-if="!loginDevices.length"
      icon="HOutline:DevicePhoneMobileIcon"
      :title="t('user.noLoginDevices')"
    />
    <div v-else class="dev-grid">
      <EntityCard v-for="row in loginDevices" :key="row.deviceId">
        <template #title>
          <span class="dev-name" :title="row.device || undefined">{{ row.device || '—' }}</span>
        </template>
        <template #status>
          <StatusPill
            v-if="isCurrentDevice(row)"
            variant="success"
            :dot="false"
            :label="t('user.currentDevice')"
          />
        </template>
        <template #meta>
          <span class="dev-kv">
            <em>{{ t('user.extra.ipAddress') }}</em>
            {{ row.ip || '—' }}
          </span>
          <span class="dev-kv">
            <em>{{ t('user.loginTime') }}</em>
            {{ formatDateTime(row.loginTime) }}
          </span>
          <span class="dev-kv">
            <em>{{ t('user.latestRefresh') }}</em>
            {{ formatDateTime(row.updateTime) }}
          </span>
          <span class="dev-kv dev-kv--mono" :title="row.sessionId">
            <em>{{ t('user.sessionId') }}</em>
            {{ shortId(row.sessionId) }}
          </span>
        </template>
        <template #footer>
          <div class="dev-foot">
            <span v-if="isCurrentDevice(row)" class="dev-foot__hint">{{ t('user.thisSession') }}</span>
            <AdaptiveConfirm
              v-else
              :title="t('user.deleteDeviceConfirm')"
              :disabled="isDeletingLoginDevice(row.deviceId)"
              @confirm="handleDelete(row.deviceId)"
            >
              <template #reference>
                <UButton
                  color="error"
                  variant="soft"
                  size="sm"
                  :loading="isDeletingLoginDevice(row.deviceId)"
                >
                  {{ t('user.logoutDevice') }}
                </UButton>
              </template>
            </AdaptiveConfirm>
          </div>
        </template>
      </EntityCard>
    </div>
  </AppPanel>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useUserProfileStore } from '@/stores/user/profile'
import { useFeedback } from '@/utils/feedback'
import { useI18n } from 'vue-i18n'

const userProfileStore = useUserProfileStore()
const menuStore = useMenuStore()
const { t } = useI18n()
const fb = useFeedback()

const { loading, loginDevices, currentDeviceLabel } = storeToRefs(userProfileStore)
const {
  formatDateTime,
  isCurrentDevice,
  isDeletingLoginDevice,
  getLoginDevices,
  deleteLoginDevice,
} = userProfileStore

const shortDevice = (label: string) => {
  // 窄屏提示条：更短截断
  const max = menuStore.isMobile ? 28 : 42
  if (label.length <= max) return label
  return `${label.slice(0, max - 1)}…`
}
const shortId = (id?: string) => {
  if (!id) return '—'
  if (id.length <= 14) return id
  return `${id.slice(0, 8)}…${id.slice(-4)}`
}

const handleDelete = async (deviceId: string) => {
  const ok = await deleteLoginDevice(deviceId)
  if (ok) fb.success(t('user.deviceDeleted'))
}

onMounted(() => {
  void getLoginDevices()
})
</script>

<style scoped lang="scss">
.dev-current {
  margin: 0 0 0.65rem;
  font-size: 0.75rem;
  line-height: 1.4;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.dev-grid {
  display: grid;
  /* minmax(0,1fr)：避免 1fr 默认 min=auto 被长 UA 字符串撑破容器 */
  grid-template-columns: minmax(0, 1fr);
  gap: 12px;
  min-width: 0;
}
.dev-grid > * {
  min-width: 0;
  max-width: 100%;
}
@media (width >= 768px) {
  .dev-grid {
    grid-template-columns: repeat(auto-fill, minmax(min(100%, 280px), 1fr));
  }
}
.dev-skeleton {
  height: 148px;
  border-radius: var(--app-radius);
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
.dev-name {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}
.dev-kv {
  display: flex;
  gap: 0.4rem;
  min-width: 0;
  font-size: 0.75rem;
  color: var(--el-text-color-regular);
  line-height: 1.4;
  overflow: hidden;
  em {
    flex-shrink: 0;
    font-style: normal;
    color: var(--el-text-color-placeholder);
    min-width: 4.5rem;
  }
}
.dev-kv--mono {
  font-variant-numeric: tabular-nums;
  overflow: hidden;
}
.dev-foot {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  width: 100%;
  min-height: 32px;
}
.dev-foot__hint {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}
</style>
