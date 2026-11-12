<!-- 登录设备：当前设备在最前；其他设备可以下线，或一次退出其他所有设备。 -->
<template>
  <section class="pp-card pp-section">
    <div class="pp-section__head">
      <h1 class="pp-section__title">{{ t('account.devices.title') }}</h1>
      <PButton v-if="others > 0" size="sm" variant="danger" :loading="busyAll" @click="logoutAll">{{ t('account.devices.logoutOthers') }}</PButton>
    </div>
    <p class="pp-section__desc">{{ t('account.devices.desc') }}</p>
    <p v-if="error" class="pp-error">{{ error }}</p>
    <PEmpty v-else-if="loaded && !items.length" compact :icon="ComputerDesktopIcon" :title="t('account.devices.empty')" />
    <div v-for="d in items" :key="d.sessionId" class="pp-frow pp-frow--icon">
      <span class="ico pp-sticker"><component :is="iconOf(d.deviceType)" aria-hidden="true" /></span>
      <span class="body">
        <span class="name">
          {{ deviceTitle(d) }}
          <PPill v-if="d.current" tone="ok">{{ t('account.devices.current') }}</PPill>
        </span>
        <span class="pp-hint">{{ deviceLine(d) }}</span>
      </span>
      <div class="pp-frow__actions">
        <PButton v-if="!d.current" size="sm" :loading="busyId === d.sessionId" @click="revoke(d.sessionId)">{{ t('account.devices.signOut') }}</PButton>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onActivated, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { CommandLineIcon, ComputerDesktopIcon, DevicePhoneMobileIcon, DeviceTabletIcon } from '@heroicons/vue/24/outline'
import PButton from '@passport/components/PButton.vue'
import PEmpty from '@passport/components/PEmpty.vue'
import PPill from '@passport/components/PPill.vue'
import * as api from '@passport/api/passport'
import { useSessionStore } from '@passport/stores/session'
import { useUiStore } from '@passport/stores/ui'
import { errorText, fmtTime } from '@passport/utils/format'
import type { PassportDevice } from '@/types/v1/passport'

const { t, te } = useI18n()
const session = useSessionStore()
const ui = useUiStore()

const items = ref<PassportDevice[]>([])
const loaded = ref(false)
const error = ref('')
const busyId = ref('')
const busyAll = ref(false)

const others = computed(() => items.value.filter((d) => !d.current).length)

const iconOf = (type: string) =>
  type === 'mobile' ? DevicePhoneMobileIcon : type === 'tablet' ? DeviceTabletIcon : type === 'cli' ? CommandLineIcon : ComputerDesktopIcon

const deviceTitle = (d: PassportDevice) => {
  const base = d.device || (te(`common.device.${d.deviceType}`) ? t(`common.device.${d.deviceType}`) : t('common.unknown'))
  return d.appName ? `${base} · ${d.appName}` : base
}

const deviceLine = (d: PassportDevice) => {
  const method = te(`common.method.${d.loginMethod}`) ? t(`common.method.${d.loginMethod}`) : ''
  return [d.lastIp || d.ip, fmtTime(d.lastActiveTime), method].filter(Boolean).join(' · ')
}

const load = async () => {
  try {
    items.value = (await api.listDevices()).items || []
    error.value = ''
  } catch (e) {
    error.value = errorText(e)
  } finally {
    loaded.value = true
  }
}
onMounted(load)
onActivated(() => {
  if (loaded.value) void load()
})

const revoke = async (id: string) => {
  busyId.value = id
  try {
    await api.revokeDevice(id)
    items.value = items.value.filter((d) => d.sessionId !== id)
    void session.loadMe().catch(() => undefined)
  } catch (e) {
    ui.toast(errorText(e), 'bad')
  } finally {
    busyId.value = ''
  }
}

const logoutAll = async () => {
  busyAll.value = true
  try {
    await api.logoutOthers()
    items.value = items.value.filter((d) => d.current)
    ui.toast(t('account.devices.othersDone'), 'ok')
    void session.loadMe().catch(() => undefined)
  } catch (e) {
    ui.toast(errorText(e), 'bad')
  } finally {
    busyAll.value = false
  }
}
</script>

<style scoped>
.ico {
  width: 44px;
  height: 44px;
  border-radius: 14px;
  background: var(--m-a2Soft);
  color: var(--m-ink);
}
.ico svg {
  width: 22px;
  height: 22px;
}
.body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.name {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 800;
}
</style>
