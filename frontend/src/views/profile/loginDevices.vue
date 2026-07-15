<!-- 登录设备：DataTable / 移动 EntityCard；下线非当前设备 -->
<template>
  <AppPanel :title="t('user.loginDevices')" title-icon="HOutline:DevicePhoneMobileIcon">
    <template #actions>
      <div class="dev-head-meta">
        <StatusPill variant="info" :dot="false" :label="t('user.totalCount', { count: loginDevices.length })" />
        <StatusPill
          v-if="currentDeviceLabel && currentDeviceLabel !== '-'"
          variant="primary"
          :dot="false"
          :label="`${t('user.currentDevice')} · ${currentDeviceLabel}`"
        />
      </div>
    </template>

    <!-- 桌面表 -->
    <DataTable
      v-if="!menuStore.isMobile"
      :columns="columns"
      :rows="(loginDevices as unknown as Record<string, unknown>[])"
      row-key="deviceId"
      :loading="loading"
      :empty-text="t('user.noLoginDevices')"
    >
      <template #cell-device="{ row }">
        <div class="dev-cell">
          <span class="dev-cell__name">{{ row.device || '-' }}</span>
          <StatusPill
            v-if="isCurrent(row)"
            variant="success"
            :dot="false"
            :label="t('user.currentDevice')"
          />
        </div>
      </template>
      <template #cell-loginTime="{ row }">{{ formatDateTime(row.loginTime) }}</template>
      <template #cell-updateTime="{ row }">{{ formatDateTime(row.updateTime) }}</template>
      <template #cell-operation="{ row }">
        <span v-if="isCurrent(row)" class="text-dim">—</span>
        <UButton
          v-else
          color="error"
          variant="ghost"
          size="sm"
          :loading="isDeletingLoginDevice(String(row.deviceId))"
          @click="askDelete(String(row.deviceId))"
        >
          {{ t('common.delete') }}
        </UButton>
      </template>
    </DataTable>

    <!-- 移动卡 -->
    <template v-else>
      <div v-if="loading" class="dev-cards">
        <div v-for="i in 4" :key="i" class="dev-skeleton" />
      </div>
      <EmptyState
        v-else-if="!loginDevices.length"
        icon="HOutline:DevicePhoneMobileIcon"
        :title="t('user.noLoginDevices')"
      />
      <div v-else class="dev-cards">
        <EntityCard v-for="row in loginDevices" :key="row.deviceId">
          <template #title>
            <span class="dev-cell__name">{{ row.device || '-' }}</span>
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
            <span>{{ row.ip || '-' }}</span>
            <span class="meta-sep">·</span>
            <span>{{ t('user.loginAt', { time: formatDateTime(row.loginTime) }) }}</span>
          </template>
          <template #footer>
            <div class="dev-card-foot">
              <span class="dev-sid">{{ row.sessionId || '-' }}</span>
              <UButton
                v-if="!isCurrentDevice(row)"
                color="error"
                variant="soft"
                size="sm"
                :loading="isDeletingLoginDevice(row.deviceId)"
                @click="askDelete(row.deviceId)"
              >
                {{ t('common.delete') }}
              </UButton>
            </div>
          </template>
        </EntityCard>
      </div>
    </template>

    <FormDialog
      v-model="deleteOpen"
      :title="t('user.deleteDeviceConfirm')"
      :width="400"
      :confirm-text="t('common.delete')"
      :loading="deleting"
      @confirm="doDelete"
    >
      <p class="confirm-text">{{ t('user.deleteDeviceConfirm') }}</p>
    </FormDialog>
  </AppPanel>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useUserProfileStore } from '@/stores/user/profile'
import { useFeedback } from '@/utils/feedback'
import type { LoginDeviceRow } from '@/stores/user/types'
import { useI18n } from 'vue-i18n'

const menuStore = useMenuStore()
const userProfileStore = useUserProfileStore()
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

const columns = computed(() => [
  { key: 'device', title: t('user.loginDevices'), minWidth: 200 },
  { key: 'ip', title: t('user.extra.ipAddress'), minWidth: 130 },
  { key: 'loginTime', title: t('user.loginTime'), width: 160 },
  { key: 'updateTime', title: t('user.latestRefresh'), width: 160 },
  { key: 'sessionId', title: t('user.sessionId'), minWidth: 180 },
  { key: 'operation', title: t('user.operation'), width: 90, align: 'center' as const },
])

const isCurrent = (row: Record<string, unknown> | LoginDeviceRow) =>
  isCurrentDevice(row as LoginDeviceRow)

const deleteOpen = ref(false)
const pendingId = ref('')
const deleting = ref(false)

const askDelete = (deviceId: string) => {
  pendingId.value = deviceId
  deleteOpen.value = true
}

const doDelete = async () => {
  if (!pendingId.value) return
  deleting.value = true
  try {
    const ok = await deleteLoginDevice(pendingId.value)
    if (ok) fb.success(t('user.deviceDeleted'))
    deleteOpen.value = false
  } finally {
    deleting.value = false
    pendingId.value = ''
  }
}

onMounted(() => {
  void getLoginDevices()
})
</script>

<style scoped lang="scss">
.dev-head-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.4rem;
  justify-content: flex-end;
}
.dev-cell {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  flex-wrap: wrap;
  min-width: 0;
}
.dev-cell__name {
  overflow-wrap: anywhere;
  word-break: break-word;
}
.text-dim {
  color: var(--el-text-color-placeholder);
}
.dev-cards {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
.dev-skeleton {
  height: 88px;
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
.dev-card-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  width: 100%;
}
.dev-sid {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 0.6875rem;
  color: var(--el-text-color-placeholder);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}
.meta-sep {
  color: var(--el-text-color-placeholder);
}
.confirm-text {
  margin: 0;
  font-size: 0.875rem;
  line-height: 1.5;
  color: var(--el-text-color-regular);
}
</style>
