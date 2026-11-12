<!-- 游戏服务器详情：/game/servers/:id?tab=overview|console|settings|update|backup|files|logs|connection。
     页头是服务器卡（状态、接入方式、位置、运行时长、目录、操作、进程），下面是页内标签；
     托管服务器有控制台 / 专用设置 / 更新 / 备份 / 文件 / 日志，仅 GM 的服务器只有概览与接入设置。 -->
<template>
  <div class="gsd" :data-game="info?.brief?.game || undefined">
    <RouterLink to="/game/servers" class="gsd__back"><UIcon name="i-lucide-chevron-left" />{{ t('gameServers.detail.back') }}</RouterLink>

    <EmptyState v-if="loadError && !info" icon="HOutline:ExclamationTriangleIcon" :title="t('gameServers.detail.loadFailed')" :description="loadError" />

    <template v-else-if="info">
      <section class="gsd__hero">
        <div class="gsd__hero-main">
          <div class="gsd__id">
            <span class="gsd__logo" aria-hidden="true">{{ initial }}</span>
            <div class="gsd__id-text">
              <div class="gsd__title-row">
                <h1>{{ info.brief?.name }}</h1>
                <StatusPill :variant="stateVariant(state)" :label="t(`gameServers.state.${state}`)" />
                <span class="gsd__tag" :class="hosted ? 'is-hosted' : 'is-remote'">{{ t(`gameServers.mode.${info.brief?.mode || 'remote'}`) }}</span>
                <span class="gsd__gtag">{{ gameName }}</span>
              </div>
              <div class="gsd__meta">
                <span>{{ where }}</span>
                <span v-if="uptime" class="m-num">{{ uptime }}</span>
                <span v-if="info.installDir" class="m-mono gsd__dir" :title="info.installDir">{{ info.installDir }}</span>
              </div>
            </div>
          </div>
          <div class="gsd__actions">
            <template v-if="hosted">
              <UButton
                v-if="running"
                color="error"
                variant="soft"
                icon="i-lucide-square"
                :loading="acting === 'stop'"
                :disabled="busyTask"
                @click="control('stop')"
              >
                {{ t('gameServers.actions.stop') }}
              </UButton>
              <UButton v-else color="primary" icon="i-lucide-play" :loading="acting === 'start'" :disabled="busyTask" @click="control('start')">
                {{ t('gameServers.actions.start') }}
              </UButton>
              <UButton
                color="neutral"
                variant="outline"
                icon="i-lucide-rotate-ccw"
                :loading="acting === 'restart'"
                :disabled="!running || busyTask"
                @click="control('restart')"
              >
                {{ t('gameServers.actions.restart') }}
              </UButton>
            </template>
            <UButton v-if="canGm" color="neutral" variant="soft" icon="i-lucide-wand-sparkles" class="game-skin" :to="`/game/gm/${encodeURIComponent(id)}`">
              {{ t('gameServers.detail.enterGm') }}
            </UButton>
            <ActionMenu :items="menuItems" :label="t('gameServers.actions.more')" @select="onMenu" />
          </div>
          <div v-if="hosted && info.processes.length" class="gsd__procs">
            <div v-for="p in info.processes" :key="p.role" class="gsd__proc">
              <span class="gsd__proc-ico m-sticker" :class="p.role === 'main' ? 'm-sticker--p' : 'm-sticker--a2'">
                <UIcon :name="p.role === 'main' ? 'i-lucide-cpu' : 'i-lucide-cloud'" />
              </span>
              <span class="gsd__proc-text">
                <b>{{ t(`gameServers.role.${p.role}`, p.role) }}</b>
                <small class="m-mono" :title="p.command">{{ procLine(p) }}</small>
              </span>
              <StatusPill :variant="stateVariant(p.state)" :label="t(`gameServers.state.${p.state}`, p.state)" />
            </div>
          </div>
          <p v-if="missingProcess" class="gsd__warn">
            {{ t('gameServers.detail.processMissing') }}
            <UButton size="sm" color="neutral" variant="soft" :loading="recreating" @click="recreate">{{ t('gameServers.detail.recreate') }}</UButton>
          </p>
        </div>
        <div class="gsd__cover">
          <GameImage v-if="coverUrl" :src="coverUrl" :alt="info.brief?.name || ''" eager class="gsd__cover-img" />
          <div class="gsd__cover-tags">
            <span v-if="(info.brief?.online ?? -1) >= 0" class="gsd__cover-tag">{{ t('gameServers.detail.online') }} <b class="m-num">{{ info.brief?.online }}</b></span>
            <span v-if="resource !== '—'" class="gsd__cover-tag">{{ t('gameServers.detail.resource') }} <b class="m-num">{{ resource }}</b></span>
          </div>
        </div>
      </section>

      <div v-for="task in info.runningTasks" :key="task.id" class="gsd__task">
        <UIcon name="i-lucide-loader-circle" class="gsd__spin" />
        <span>{{ t('gameServers.detail.runningTask', { task: t(`gameServers.task.kind.${task.kind}`, task.kind) }) }}</span>
        <span class="gsd__task-step">{{ currentStep(task) }}</span>
      </div>

      <nav class="gsd__tabs" :aria-label="info.brief?.name">
        <RouterLink
          v-for="tab in tabs"
          :key="tab.key"
          :to="{ path: route.path, query: { tab: tab.key } }"
          replace
          class="gsd__tab"
          :class="{ 'is-on': currentTab === tab.key }"
          :aria-current="currentTab === tab.key ? 'page' : undefined"
        >
          <UIcon :name="tab.icon" />{{ t(`gameServers.detail.tabs.${tab.key}`) }}
        </RouterLink>
      </nav>

      <KeepAlive>
        <component :is="tabView" :key="`${id}:${currentTab}`" :server="info" :adapter="adapter" @changed="load(true)" />
      </KeepAlive>
    </template>

    <ServerEditDialog v-model="editOpen" :server="info" @saved="load(true)" />
    <ServerDeleteDialog v-model="deleteOpen" :server="info" @deleted="router.replace('/game/servers')" />
    <RotateKeyDialog v-model="rotateOpen" :server="info" @done="load(true)" />
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import '@/games'
import { gameAsset } from '@/games'
import { controlGameServer, getGameServer, listGameAdapters, refreshGameServer, updateGameServerHosting } from '@/api/gameServer'
import type { GameServerAction } from '@/api/gameServer'
import { getRequestErrorMessage } from '@/utils/request'
import { useAccessibleRoutes } from '@/composables/useAccessibleRoutes'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import type { GameAdapter, GameServerInfo, GameServerProcessInfo, GameServerTask } from '@/types/v1/game_server'
import ServerEditDialog from '../components/ServerEditDialog.vue'
import ServerDeleteDialog from '../components/ServerDeleteDialog.vue'
import RotateKeyDialog from '../components/RotateKeyDialog.vue'
import { fmtDuration, fmtTime, resourceText, stateVariant, statusOf, toMs } from '../shared'

const OverviewTab = defineAsyncComponent(() => import('./components/OverviewTab.vue'))
const ConsoleTab = defineAsyncComponent(() => import('./components/ConsoleTab.vue'))
const SettingsTab = defineAsyncComponent(() => import('./components/SettingsTab.vue'))
const UpdateTab = defineAsyncComponent(() => import('./components/UpdateTab.vue'))
const BackupTab = defineAsyncComponent(() => import('./components/BackupTab.vue'))
const FilesTab = defineAsyncComponent(() => import('./components/FilesTab.vue'))
const LogsTab = defineAsyncComponent(() => import('./components/LogsTab.vue'))
const ConnectionTab = defineAsyncComponent(() => import('./components/ConnectionTab.vue'))

defineOptions({ name: 'GameServerDetailView' })

const { t, te } = useI18n()
const route = useRoute()
const router = useRouter()
const fb = useFeedback()
const accessible = useAccessibleRoutes()

const id = computed(() => String(route.params.id || ''))
const info = ref<GameServerInfo | null>(null)
const adapter = ref<GameAdapter | null>(null)
const loadError = ref('')
const acting = ref<'' | GameServerAction>('')
const recreating = ref(false)
const now = ref(Date.now())

const load = async (silent = false) => {
  if (!id.value) return
  try {
    const { data } = await getGameServer(id.value)
    info.value = data
    loadError.value = ''
    if (!adapter.value || adapter.value.id !== data.brief?.game) {
      const res = await listGameAdapters()
      adapter.value = (res.data.items ?? []).find((a) => a.id === data.brief?.game) || null
    }
  } catch (e) {
    if (!silent || !info.value) loadError.value = getRequestErrorMessage(e, t('gameServers.detail.loadFailed'))
  }
}

const hosted = computed(() => info.value?.brief?.mode === 'hosted')
const state = computed(() => info.value?.brief?.state || 'unknown')
const running = computed(() => ['running', 'maintenance', 'starting', 'restarting'].includes(state.value))
const busyTask = computed(() => (info.value?.runningTasks || []).some((x) => !x.kind.startsWith('resource')))
const gameName = computed(() => {
  const key = `game.adapters.${info.value?.brief?.game}`
  return info.value?.brief?.gameName || (te(key) ? t(key) : t('game.unnamed'))
})
const initial = computed(() => (gameName.value || '?').charAt(0).toUpperCase())
const coverUrl = computed(() => (info.value?.brief?.cover ? gameAsset(info.value.brief.game, info.value.brief.cover) : ''))
const resource = computed(() => resourceText(statusOf(info.value)))
const canGm = computed(() => accessible.value.has('/game/gm') && (info.value?.capabilities || []).includes('gm'))
const mainProc = computed(() => info.value?.processes.find((p) => p.role === 'main'))
const missingProcess = computed(() => hosted.value && (info.value?.processes || []).some((p) => p.state === 'missing'))

const where = computed(() => {
  const s = info.value
  if (!s) return ''
  if (hosted.value) return s.hostName ? t('gameServers.common.localHost', { host: s.hostName }) : t('gameServers.common.local')
  return s.mgpEndpoint || t('gameServers.list.mgpNone')
})
const uptime = computed(() => {
  const p = mainProc.value
  if (hosted.value && p?.state === 'running' && p.startTime) return t('gameServers.detail.uptime', { d: fmtDuration(t, now.value - toMs(p.startTime)) })
  if (hosted.value && p?.state === 'stopped' && p.stopTime) return t('gameServers.detail.stoppedSince', { t: fmtTime(p.stopTime) })
  const st = statusOf(info.value)
  if (!hosted.value && st?.uptimeSeconds && info.value?.brief?.mgpOk) return t('gameServers.detail.uptime', { d: fmtDuration(t, st.uptimeSeconds * 1000) })
  return ''
})
const procLine = (p: GameServerProcessInfo) => {
  const parts = [p.command]
  if (p.pid) parts.push(`PID ${p.pid}`)
  return parts.filter(Boolean).join(' · ')
}
const currentStep = (task: GameServerTask) => {
  const s = task.steps.find((x) => x.state === 'running')
  return s ? t(`gameServers.task.step.${s.key}`, s.key) : ''
}

const TAB_DEFS = [
  { key: 'overview', icon: 'i-lucide-activity', hosted: false },
  { key: 'console', icon: 'i-lucide-terminal', hosted: true, cap: 'console' },
  { key: 'settings', icon: 'i-lucide-sliders-horizontal', hosted: true, cap: 'settings' },
  { key: 'update', icon: 'i-lucide-cloud-download', hosted: true, cap: 'update' },
  { key: 'backup', icon: 'i-lucide-archive', hosted: true, cap: 'backup' },
  { key: 'files', icon: 'i-lucide-folder', hosted: true, cap: 'files' },
  { key: 'logs', icon: 'i-lucide-scroll-text', hosted: true, cap: 'logs' },
  { key: 'connection', icon: 'i-lucide-plug', hosted: false },
]
const tabs = computed(() => {
  const caps = new Set(info.value?.capabilities || [])
  return TAB_DEFS.filter((d) => !d.hosted || (hosted.value && (!d.cap || caps.has(d.cap))))
})
const currentTab = computed(() => {
  const q = String(route.query.tab || '')
  return tabs.value.some((x) => x.key === q) ? q : 'overview'
})
const VIEWS = {
  overview: OverviewTab,
  console: ConsoleTab,
  settings: SettingsTab,
  update: UpdateTab,
  backup: BackupTab,
  files: FilesTab,
  logs: LogsTab,
  connection: ConnectionTab,
} as const
const tabView = computed(() => VIEWS[currentTab.value as keyof typeof VIEWS] ?? OverviewTab)

const control = async (action: GameServerAction) => {
  acting.value = action
  try {
    const { data } = await controlGameServer(id.value, action)
    info.value = data
    fb.success(t(action === 'start' ? 'gameServers.actions.started' : action === 'stop' ? 'gameServers.actions.stopping' : 'gameServers.actions.restarting'))
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    acting.value = ''
  }
}

const recreate = async () => {
  const h = info.value?.hosting
  if (!h) return
  recreating.value = true
  try {
    const { data } = await updateGameServerHosting({
      id: id.value,
      autoStart: h.autoStart,
      autoRestart: h.autoRestart,
      resourceServer: h.resourceServer,
      resourceListen: h.resourceListen,
      resourceOrigin: h.resourceOrigin,
      resourceRoot: h.resourceRoot,
    })
    info.value = data
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    recreating.value = false
  }
}

const editOpen = ref(false)
const deleteOpen = ref(false)
const rotateOpen = ref(false)
const menuItems = computed<ActionMenuItem[]>(() => [
  { key: 'edit', label: t('gameServers.actions.edit'), icon: 'HOutline:PencilSquareIcon' },
  { key: 'forceStop', label: t('gameServers.actions.forceStop'), icon: 'HOutline:StopIcon', hidden: !hosted.value || !running.value },
  { key: 'rotate', label: t('gameServers.actions.rotateKey'), icon: 'HOutline:KeyIcon' },
  { key: 'refresh', label: t('gameServers.actions.refresh'), icon: 'HOutline:ArrowPathRoundedSquareIcon', hidden: !info.value?.brief?.mgpConfigured },
  { key: 'switch', label: t('gameServers.actions.switchMode'), icon: 'HOutline:ArrowsPointingInIcon' },
  { key: 'delete', label: t('gameServers.actions.delete'), icon: 'HOutline:TrashIcon', danger: true },
])
const onMenu = async (key: string) => {
  switch (key) {
    case 'edit':
      editOpen.value = true
      break
    case 'delete':
      deleteOpen.value = true
      break
    case 'rotate':
      rotateOpen.value = true
      break
    case 'switch':
      router.replace({ path: route.path, query: { tab: 'connection' } })
      break
    case 'forceStop':
      try {
        info.value = (await controlGameServer(id.value, 'stop', '', true)).data
      } catch (e) {
        fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
      }
      break
    case 'refresh':
      try {
        info.value = (await refreshGameServer(id.value)).data
        fb.success(t('gameServers.actions.refreshed'))
      } catch (e) {
        fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
      }
      break
  }
}

// 状态（进程、MGP、进行中的任务）定时刷新
let timer: ReturnType<typeof setInterval> | undefined
onMounted(() => {
  load()
  timer = setInterval(() => {
    now.value = Date.now()
    if (document.visibilityState === 'visible') load(true)
  }, 5000)
})
onBeforeUnmount(() => clearInterval(timer))
watch(id, () => {
  info.value = null
  load()
})
</script>

<style scoped lang="scss">
.gsd {
  display: flex;
  flex-direction: column;
  gap: 18px;
  min-width: 0;
}
.gsd__back {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  align-self: flex-start;
  min-height: 40px;
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  text-decoration: none;
}
.gsd__hero {
  display: flex;
  flex-wrap: wrap;
  overflow: hidden;
  border-radius: var(--app-radius-lg);
  border: var(--cmp-card-border-width, 1px) solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.gsd__hero-main {
  flex: 1 1 520px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 22px 24px;
}
.gsd__id {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 16px;
}
.gsd__logo {
  flex: none;
  width: 68px;
  height: 68px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 20px;
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--m-gInk, var(--m-pInk));
  font-family: var(--app-font-display);
  font-size: 1.75rem;
  font-weight: 800;
  transform: rotate(-5deg);
  box-shadow: 4px 4px 0 var(--m-g, var(--m-p));
}
.gsd__id-text {
  flex: 1 1 260px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.gsd__title-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 10px;
}
.gsd__title-row h1 {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 1.75rem;
  font-weight: 800;
  overflow-wrap: anywhere;
}
.gsd__tag,
.gsd__gtag {
  display: inline-flex;
  align-items: center;
  height: 26px;
  padding: 0 10px;
  border-radius: 8px;
  font-size: 0.75rem;
  font-weight: 700;
}
.gsd__tag.is-hosted {
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.gsd__tag.is-remote {
  background: var(--m-a2Soft);
  color: var(--m-ink);
}
.gsd__gtag {
  border-radius: 999px;
  background: var(--m-gSoft, var(--m-s2));
  color: var(--m-gInk, var(--m-ink2));
}
.gsd__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 16px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  min-width: 0;
}
.gsd__dir {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}
.gsd__actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.gsd__procs {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 10px;
}
.gsd__proc {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  padding: 12px 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.gsd__proc-ico {
  flex: none;
  width: 38px;
  height: 38px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
}
.gsd__proc-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 0.875rem;
}
.gsd__proc-text small {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gsd__warn {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  margin: 0;
  padding: 10px 14px;
  border-radius: var(--app-radius);
  background: var(--m-warnSoft);
  color: var(--m-warn);
  font-size: 0.8125rem;
  font-weight: 700;
}
.gsd__cover {
  position: relative;
  flex: 1 1 320px;
  min-height: 200px;
  background: var(--m-gSoft, var(--m-s2));
}
.gsd__cover-img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: 60% 35%;
}
.gsd__cover-tags {
  position: absolute;
  left: 16px;
  bottom: 16px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.gsd__cover-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 28px;
  padding: 0 10px;
  border-radius: 999px;
  background: var(--el-bg-color);
  font-size: 0.75rem;
  font-weight: 600;
}
.gsd__task {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  border-radius: var(--app-radius);
  background: var(--m-pSoft);
  color: var(--m-pInk);
  font-size: 0.875rem;
  font-weight: 700;
}
.gsd__task-step {
  font-weight: 500;
}
.gsd__spin {
  animation: gsd-spin 1s linear infinite;
}
@keyframes gsd-spin {
  to {
    transform: rotate(360deg);
  }
}
.gsd__tabs {
  display: flex;
  gap: 24px;
  border-bottom: 1px solid var(--el-border-color-light);
  overflow-x: auto;
  scrollbar-width: none;
}
.gsd__tabs::-webkit-scrollbar {
  display: none;
}
.gsd__tab {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 44px;
  padding: 0 2px;
  border-bottom: 2.5px solid transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.9375rem;
  font-weight: 700;
  text-decoration: none;
  white-space: nowrap;
}
.gsd__tab:hover {
  color: var(--el-text-color-primary);
}
.gsd__tab.is-on {
  color: var(--el-text-color-primary);
  border-bottom-color: var(--m-p);
}
@media (width <= 768px) {
  .gsd__hero-main {
    padding: 18px 16px;
  }
  .gsd__cover {
    min-height: 140px;
  }
  .gsd__title-row h1 {
    font-size: 1.375rem;
  }
  .gsd__actions > * {
    min-height: 44px;
  }
}
</style>
