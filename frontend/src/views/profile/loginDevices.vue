<!-- 登录设备：设备类型贴纸 + 名称 + 最近活跃 IP/网络/时间；当前设备标记；单个下线 / 下线其它设备；点行看详情。 -->
<template>
  <AppPanel class="dev">
    <template #header>
      <div class="dev-head">
        <h2 class="pf-card-title">{{ t('user.loginDevices') }}</h2>
        <span class="dev-head__sub">
          {{ t('profile.devices.summary', { n: loginDevices.length }) }}
        </span>
      </div>
    </template>
    <template v-if="otherCount > 0" #actions>
      <AdaptiveConfirm
        :title="t('profile.devices.kickOthersConfirm', { n: otherCount })"
        :disabled="loggingOutOthers"
        @confirm="handleKickOthers"
      >
        <template #reference>
          <UButton
            color="error"
            variant="soft"
            size="sm"
            class="dev-kick"
            icon="i-lucide-log-out"
            :loading="loggingOutOthers"
          >
            {{ compact ? t('profile.devices.kickOthersShort') : t('profile.devices.kickOthers') }}
          </UButton>
        </template>
      </AdaptiveConfirm>
    </template>

    <div v-if="loading && !loginDevices.length" class="dev-skel-list">
      <div v-for="i in 3" :key="i" class="dev-skel" />
    </div>
    <EmptyState
      v-else-if="!loginDevices.length"
      icon="HOutline:DevicePhoneMobileIcon"
      :title="t('user.noLoginDevices')"
    />
    <div v-else class="dev-list">
      <ProfileRow
        v-for="(row, idx) in shownDevices"
        :key="row.sessionId || row.deviceId"
        :icon="deviceIcon(row.deviceType)"
        :tone="tones[idx % tones.length]"
        :tilt="tilts[idx % tilts.length]"
        clickable
        num-sub
        @click="openDetail(row)"
      >
        <template #title>
          <span class="dev-name">{{ deviceName(row.device) }}</span>
          <StatusPill
            v-if="isCurrentDevice(row)"
            variant="success"
            :label="t('profile.devices.current')"
          />
        </template>
        <template #sub>{{ metaLine(row) }}</template>

        <span v-if="isCurrentDevice(row)" class="dev-using">{{ t('profile.devices.inUse') }}</span>
        <AdaptiveConfirm
          v-else
          :title="t('user.deleteDeviceConfirm')"
          :disabled="isDeletingLoginDevice(row.sessionId)"
          @confirm="handleDelete(row.sessionId)"
        >
          <template #reference>
            <UButton
              color="error"
              variant="soft"
              size="sm"
              class="dev-kick"
              :loading="isDeletingLoginDevice(row.sessionId)"
              :aria-label="t('profile.devices.kickAria', { name: deviceName(row.device) })"
            >
              {{ t('profile.devices.kick') }}
            </UButton>
          </template>
        </AdaptiveConfirm>
      </ProfileRow>
      <button
        v-if="loginDevices.length > COLLAPSE_AT"
        type="button"
        class="dev-more"
        @click="expanded = !expanded"
      >
        {{ expanded ? t('profile.devices.collapse') : t('profile.devices.expand', { n: loginDevices.length - COLLAPSE_AT }) }}
        <UIcon :name="expanded ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'" aria-hidden="true" />
      </button>
    </div>

    <!-- 设备详情 -->
    <FormDialog v-model="detailOpen" :title="t('user.deviceDetail')" :width="500">
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

      <div v-if="detailRow" class="dev-detail">
        <div class="dev-detail__head">
          <span class="m-sticker m-sticker--p dev-detail__sk" aria-hidden="true">
            <UIcon :name="deviceIcon(detailRow.deviceType)" />
          </span>
          <div class="dev-detail__name">
            <strong>{{ deviceName(detailRow.device) }}</strong>
            <span>{{ deviceTypeLabel(detailRow.deviceType) }}</span>
          </div>
          <StatusPill
            v-if="isCurrentDevice(detailRow)"
            variant="success"
            :label="t('profile.devices.current')"
          />
        </div>
        <dl class="dev-detail__grid">
          <div v-for="item in detailItems" :key="item.label" :class="{ 'is-wide': item.wide }">
            <dt>{{ item.label }}</dt>
            <dd :class="{ 'm-mono': item.mono }">{{ item.value || '—' }}</dd>
          </div>
        </dl>
      </div>
    </FormDialog>
  </AppPanel>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useUserProfileStore } from '@/stores/user/profile'
import type { LoginDeviceRow } from '@/stores/user/types'
import { useFeedback } from '@/utils/feedback'
import { useI18n } from 'vue-i18n'
import ProfileRow from './components/ProfileRow.vue'
import { useProfileFormat } from './components/useProfileFormat'

const userProfileStore = useUserProfileStore()
const menuStore = useMenuStore()
const { t } = useI18n()
const fb = useFeedback()
const { relTime, fullTime, scopeLabel, methodLabel, deviceIcon, deviceTypeLabel, deviceName } =
  useProfileFormat()

const { loading, loginDevices, extraColumns, loggingOutOthers } = storeToRefs(userProfileStore)
const {
  formatCellValue,
  getExtraColumnLabel,
  isCurrentDevice,
  isDeletingLoginDevice,
  getLoginDevices,
  deleteLoginDevice,
  logoutOtherDevices,
} = userProfileStore

const compact = computed(() => menuStore.isMobile)
const tones = ['p', 'a2', 'a', 'ok'] as const
const tilts = [-3, 3, -2, 4]
const COLLAPSE_AT = 5
const expanded = ref(false)
const shownDevices = computed(() =>
  expanded.value ? loginDevices.value : loginDevices.value.slice(0, COLLAPSE_AT),
)
const otherCount = computed(() => loginDevices.value.filter((d) => !isCurrentDevice(d)).length)

const ipOf = (row: LoginDeviceRow) => row.lastIp || row.ip || ''

const metaLine = (row: LoginDeviceRow) => {
  const ip = ipOf(row)
  const parts: string[] = []
  if (ip) parts.push(ip)
  const scope = scopeLabel(row.ipScope)
  if (scope) parts.push(scope)
  parts.push(isCurrentDevice(row) ? t('profile.devices.activeNow') : relTime(row.lastActiveTime || row.updateTime))
  return parts.join(' · ')
}

const detailOpen = ref(false)
const detailRow = ref<LoginDeviceRow | null>(null)

const detailItems = computed(() => {
  const row = detailRow.value
  if (!row) return [] as { label: string; value?: string; mono?: boolean; wide?: boolean }[]
  const items: { label: string; value?: string; mono?: boolean; wide?: boolean }[] = [
    { label: t('profile.devices.lastActive'), value: fullTime(row.lastActiveTime) },
    { label: t('profile.devices.lastIp'), value: [ipOf(row), scopeLabel(row.ipScope)].filter(Boolean).join(' · '), mono: true },
    { label: t('user.loginTime'), value: fullTime(row.loginTime) },
    { label: t('profile.devices.loginIp'), value: row.ip, mono: true },
    { label: t('profile.devices.loginMethod'), value: methodLabel(row.loginMethod) },
    { label: t('profile.devices.expire'), value: fullTime(row.expireTime) },
    { label: t('user.latestRefresh'), value: fullTime(row.updateTime) },
    { label: t('user.sessionId'), value: row.sessionId, mono: true, wide: true },
    { label: t('user.deviceId'), value: row.deviceId, mono: true, wide: true },
    { label: 'User Agent', value: row.userAgent, mono: true, wide: true },
  ]
  for (const key of extraColumns.value) {
    const raw = (row as Record<string, unknown>)[key]
    if (raw === null || raw === undefined || raw === '') continue
    items.push({ label: getExtraColumnLabel(key), value: formatCellValue(raw), wide: true })
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
  await handleDelete(detailRow.value.sessionId)
}

const handleKickOthers = async () => {
  const ok = await logoutOtherDevices()
  if (ok) fb.success(t('profile.devices.kickOthersDone'))
}

onMounted(() => {
  void getLoginDevices()
})
</script>

<style scoped lang="scss">
.dev :deep(.app-panel__body) {
  padding-top: 8px;
  padding-bottom: 8px;
}
.pf-card-title {
  font-family: var(--app-font-display);
  font-size: 1.25rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.dev-head {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.dev-head__sub {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.dev-name {
  overflow-wrap: anywhere;
}
.dev-using {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}
.dev-more {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: 100%;
  min-height: 44px;
  border: 0;
  border-top: 1px solid var(--el-border-color-light);
  background: transparent;
  color: var(--m-pInk);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
}
.dev-kick {
  min-height: 36px;
}
.dev-skel-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 8px 0;
}
.dev-skel {
  height: 56px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.dev-detail {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.dev-detail__head {
  display: flex;
  align-items: center;
  gap: 12px;
}
.dev-detail__sk {
  width: 46px;
  height: 46px;
  border-radius: 15px;
  font-size: 22px;
  transform: rotate(-3deg);
}
.dev-detail__name {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  strong {
    font-size: 1rem;
    font-weight: 800;
    color: var(--el-text-color-primary);
    overflow-wrap: anywhere;
  }
  span {
    font-size: 0.8125rem;
    color: var(--el-text-color-secondary);
  }
}
.dev-detail__grid {
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
}
@media (width < 640px) {
  .dev-detail__grid {
    grid-template-columns: minmax(0, 1fr);
  }
  .dev-kick {
    min-height: 44px;
  }
}
</style>
