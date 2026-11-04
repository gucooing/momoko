<!-- SSH Web 终端（按原型）：页头（返回 SSH 主机 + 面包屑「SSH 主机 / 主机名」+ 标题「Web 终端」+ 快捷命令）+ 终端卡片。
     支持同一页同时开多个会话：顶栏左侧是会话标签（状态点 + user@名称 + 关闭），「+」从主机列表再开一个；
     每个会话是一个 SshSession（独立 ws + xterm，v-show 切换，切走不断线）。路由 ?id= 为首个会话。
     快捷命令按主机保存（主机编辑弹窗里维护），点一下整行发送到当前会话；主机所有者可直接在这里编辑。
     手机端：标签区换成 返回 + user@主机 + 「● 已连接 · 延迟 · 列×行」，快捷命令 / 会话切换为图标，工具收进「⋯」，最右「断开」。 -->
<template>
  <div class="ssh-term">
    <header v-if="!menuStore.isMobile" class="ssh-term__head">
      <AppIconButton icon="HOutline:ChevronLeftIcon" :label="t('ssh.term.back')" :box="44" @click="goBack" />
      <div class="ssh-term__titles">
        <div class="ssh-term__crumb">
          <RouterLink to="/openssh/management">{{ t('ssh.list.title') }}</RouterLink>
          <span>/</span>
          <span>{{ activeSession?.info?.name || activeSession?.label || '—' }}</span>
        </div>
        <h1 class="ssh-term__title">{{ t('ssh.term.webTitle') }}</h1>
      </div>
      <AppDropdown v-if="activeSession" align="end" :width="320">
        <template #trigger>
          <UButton color="neutral" variant="soft" icon="i-lucide-zap">{{ t('ssh.term.quickCommands') }}</UButton>
        </template>
        <template #default="{ close }">
          <div class="ssh-term__menu">
            <div class="ssh-term__menu-label">{{ t('ssh.term.quickCommandsOf', { name: activeSession.info?.name || activeSession.label }) }}</div>
            <button
              v-for="cmd in activeCommands"
              :key="cmd"
              type="button"
              class="ssh-term__menu-item ssh-term__cmd"
              :title="cmd"
              @click="runCommand(cmd); close()"
            >
              <UIcon name="i-lucide-corner-down-left" />
              <span class="m-mono">{{ cmd }}</span>
            </button>
            <div v-if="!activeCommands.length" class="ssh-term__menu-empty">{{ t('ssh.term.quickCommandsEmpty') }}</div>
            <button
              v-if="activeIsOwner"
              type="button"
              class="ssh-term__menu-item ssh-term__menu-edit"
              @click="editQuickCommands(); close()"
            >
              <UIcon name="i-lucide-pencil" />
              <span>{{ t('ssh.term.editQuickCommands') }}</span>
            </button>
          </div>
        </template>
      </AppDropdown>
    </header>

    <EmptyState
      v-if="!sessions.length"
      icon="HOutline:CommandLineIcon"
      :title="t('ssh.term.noSession')"
      :description="t('ssh.term.noSessionDesc')"
    >
      <template #action>
        <UButton color="primary" icon="i-lucide-server" to="/openssh/management">{{ t('ssh.list.title') }}</UButton>
      </template>
    </EmptyState>

    <SshSession
      v-for="s in sessions"
      v-show="s.key === activeKey"
      :key="s.key"
      :ref="(el) => setSessionRef(s.key, el)"
      :host-id="s.hostId"
      @info="(info) => onInfo(s.key, info)"
      @status="(st) => (s.status = st)"
    >
      <template #lead="{ latency, cols, rows }">
        <!-- 手机：返回 + 主机 + 状态（延迟 · 列×行）+ 快捷命令 / 会话切换 -->
        <template v-if="menuStore.isMobile">
          <AppIconButton icon="HOutline:ChevronLeftIcon" :label="t('ssh.term.back')" :box="44" @click="goBack" />
          <span class="ssh-term__mhost">
            <span class="ssh-term__mname m-mono">{{ s.label }}</span>
            <span class="ssh-term__mstatus m-num" :class="`is-${toneOf(s.status)}`">● {{ mobileStatus(s.status, latency, cols, rows) }}</span>
          </span>
          <!-- 一个菜单：快捷命令 + 会话切换 / 新建（手机顶栏宽度有限） -->
          <AppDropdown v-if="mobileMenuVisible(s)" align="end" :width="280">
            <template #trigger>
              <button type="button" class="ssh-term__switch" :aria-label="t('ssh.term.mobileMenu')">
                <UIcon :name="sessions.length > 1 || addableHosts.length ? 'i-lucide-chevrons-up-down' : 'i-lucide-zap'" />
              </button>
            </template>
            <template #default="{ close }">
              <div class="ssh-term__menu">
                <template v-if="s.info?.quickCommands?.length || isOwnerOf(s)">
                  <div class="ssh-term__menu-label">{{ t('ssh.term.quickCommands') }}</div>
                  <button
                    v-for="cmd in s.info?.quickCommands || []"
                    :key="cmd"
                    type="button"
                    class="ssh-term__menu-item ssh-term__cmd"
                    @click="runCommand(cmd); close()"
                  >
                    <UIcon name="i-lucide-corner-down-left" />
                    <span class="m-mono">{{ cmd }}</span>
                  </button>
                  <div v-if="!s.info?.quickCommands?.length" class="ssh-term__menu-empty">{{ t('ssh.term.quickCommandsEmpty') }}</div>
                  <button
                    v-if="isOwnerOf(s)"
                    type="button"
                    class="ssh-term__menu-item ssh-term__menu-edit"
                    @click="editQuickCommands(); close()"
                  >
                    <UIcon name="i-lucide-pencil" />
                    <span>{{ t('ssh.term.editQuickCommands') }}</span>
                  </button>
                </template>
                <template v-if="sessions.length > 1">
                  <div class="ssh-term__menu-label">{{ t('ssh.term.sessions') }}</div>
                  <button
                    v-for="o in sessions"
                    :key="o.key"
                    type="button"
                    class="ssh-term__menu-item"
                    :class="{ 'is-on': o.key === activeKey }"
                    @click="activate(o.key); close()"
                  >
                    <span class="ssh-term__dot" :class="`is-${toneOf(o.status)}`" />
                    <span class="m-mono">{{ o.label }}</span>
                  </button>
                </template>
                <template v-if="addableHosts.length">
                  <div class="ssh-term__menu-label">{{ t('ssh.term.newSession') }}</div>
                  <button
                    v-for="h in addableHosts"
                    :key="h.id"
                    type="button"
                    class="ssh-term__menu-item"
                    @click="openSession(h.id); close()"
                  >
                    <UIcon name="i-lucide-plus" />
                    <span>{{ h.name }}</span>
                  </button>
                </template>
              </div>
            </template>
          </AppDropdown>
        </template>

        <!-- PC：会话标签 -->
        <div v-else class="ssh-term__tabs" role="tablist" :aria-label="t('ssh.term.sessions')">
          <div
            v-for="o in sessions"
            :key="o.key"
            class="ssh-term__tab"
            :class="{ 'is-on': o.key === activeKey }"
          >
            <button
              type="button"
              role="tab"
              class="ssh-term__tab-btn"
              :aria-selected="o.key === activeKey"
              @click="activate(o.key)"
            >
              <span class="ssh-term__dot" :class="`is-${toneOf(o.status)}`" />
              <span class="m-mono">{{ o.label }}</span>
            </button>
            <button
              type="button"
              class="ssh-term__tab-x"
              :aria-label="t('ssh.term.closeSession', { name: o.label })"
              :title="t('ssh.term.closeSession', { name: o.label })"
              @click="closeSession(o.key)"
            >
              <UIcon name="i-lucide-x" />
            </button>
          </div>
          <AppDropdown align="start" :width="260">
            <template #trigger>
              <AppIconButton icon="HOutline:PlusIcon" :label="t('ssh.term.newSession')" :box="36" :size="18" />
            </template>
            <template #default="{ close }">
              <div class="ssh-term__menu">
                <div class="ssh-term__menu-label">{{ t('ssh.term.newSession') }}</div>
                <button
                  v-for="h in hostOptions"
                  :key="h.id"
                  type="button"
                  class="ssh-term__menu-item"
                  @click="openSession(h.id); close()"
                >
                  <span class="ssh-term__menu-name">{{ h.name }}</span>
                  <span class="ssh-term__menu-sub m-mono">{{ h.username }}@{{ h.host }}</span>
                </button>
                <div v-if="!hostOptions.length" class="ssh-term__menu-empty">{{ t('ssh.common.noData') }}</div>
              </div>
            </template>
          </AppDropdown>
        </div>
      </template>
    </SshSession>

    <SshConnectionCreate ref="createRef" @refresh="onHostEdited" />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, useTemplateRef } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { getSshHosts } from '@/api/openssh'
import type { SSHHostInfo } from '@/types/v1/openssh'
import type { ConsoleSocketStatus } from '@/stores/instance/types'
import SshSession from '@/views/openssh/terminal/SshSession.vue'
import SshConnectionCreate from '@/views/openssh/management/create.vue'

defineOptions({ name: 'SshTerminalView' })

interface SessionItem {
  key: string
  hostId: string
  label: string
  status: ConsoleSocketStatus
  /** 主机详情（连接时拉取；含快捷命令、私钥算法、访问身份） */
  info?: SSHHostInfo
}

const OWNER_ROLE = 'SSH_HOST_ACCESS_ROLE_OWNER'

const route = useRoute()
const router = useRouter()
const tabsStore = useTabsStore()
const menuStore = useMenuStore()
const { t } = useI18n()

const initialId = (route.query.id as string) || ''
let seq = 0
const sessions = ref<SessionItem[]>(
  initialId ? [{ key: `s${seq++}`, hostId: initialId, label: '…', status: 'connecting' }] : [],
)
const activeKey = ref(sessions.value[0]?.key || '')
const activeSession = computed(() => sessions.value.find((s) => s.key === activeKey.value))

type SessionRef = InstanceType<typeof SshSession>
const sessionRefs = new Map<string, SessionRef>()
const setSessionRef = (key: string, el: unknown) => {
  if (el) sessionRefs.set(key, el as SessionRef)
  else sessionRefs.delete(key)
}

const hostOptions = ref<SSHHostInfo[]>([])
const addableHosts = computed(() => hostOptions.value.filter((h) => !sessions.value.some((s) => s.hostId === h.id)))

const loadHosts = async () => {
  try {
    const { data } = await getSshHosts({ page: 1, pageSize: 50 })
    hostOptions.value = data?.infos || []
  } catch {
    hostOptions.value = []
  }
}

const createRef = useTemplateRef<InstanceType<typeof SshConnectionCreate> | null>('createRef')

const isOwnerOf = (s?: SessionItem) => s?.info?.accessRole === OWNER_ROLE
const activeIsOwner = computed(() => isOwnerOf(activeSession.value))
const activeCommands = computed(() => activeSession.value?.info?.quickCommands || [])

const runCommand = (cmd: string) => sessionRefs.get(activeKey.value)?.sendCommand(cmd)

// 快捷命令在主机编辑弹窗里维护；保存后刷新同一主机所有会话的信息（不断开连接）
const editQuickCommands = () => {
  const info = activeSession.value?.info
  if (info) createRef.value?.showDialog(info)
}
const onHostEdited = () => {
  const hostId = activeSession.value?.hostId
  if (!hostId) return
  for (const s of sessions.value) {
    if (s.hostId === hostId) void sessionRefs.get(s.key)?.reloadInfo()
  }
  void loadHosts()
}

const mobileMenuVisible = (s: SessionItem) =>
  !!s.info?.quickCommands?.length || isOwnerOf(s) || sessions.value.length > 1 || addableHosts.value.length > 0

const mobileStatus = (st: ConsoleSocketStatus, latency: number, cols: number, rows: number) => {
  if (st !== 'connected') return statusText(st)
  const parts = [statusText(st)]
  if (latency) parts.push(t('ssh.term.latency', { ms: latency }))
  if (cols && rows) parts.push(`${cols}×${rows}`)
  return parts.join(' · ')
}

const onInfo = (key: string, info: SSHHostInfo) => {
  const s = sessions.value.find((x) => x.key === key)
  if (s) {
    s.label = `${info.username}@${info.name || info.host}`
    s.info = info
  }
  if (s?.hostId === initialId) {
    const tab = tabsStore.tabs.find((item) => item.fullPath === route.fullPath)
    if (tab) tab.title = info.name || `${info.username}@${info.host}`
  }
}

const activate = async (key: string) => {
  activeKey.value = key
  await nextTick()
  const r = sessionRefs.get(key)
  r?.fit()
  r?.focus()
}

const openSession = (hostId: string) => {
  const key = `s${seq++}`
  const host = hostOptions.value.find((h) => h.id === hostId)
  sessions.value.push({
    key,
    hostId,
    label: host ? `${host.username}@${host.name || host.host}` : '…',
    status: 'connecting',
  })
  activeKey.value = key
}

const closeSession = (key: string) => {
  const idx = sessions.value.findIndex((s) => s.key === key)
  if (idx < 0) return
  sessionRefs.get(key)?.disconnect()
  sessions.value.splice(idx, 1)
  if (!sessions.value.length) {
    goBack()
    return
  }
  if (activeKey.value === key) void activate(sessions.value[Math.max(0, idx - 1)]!.key)
}

const goBack = () => router.push('/openssh/management')

const toneOf = (st: ConsoleSocketStatus) =>
  st === 'connected' ? 'ok' : st === 'connecting' ? 'warn' : st === 'error' ? 'bad' : 'off'
const statusText = (st: ConsoleSocketStatus) => {
  switch (st) {
    case 'connected':
      return t('ssh.common.connected')
    case 'connecting':
      return t('ssh.common.connecting')
    case 'error':
      return t('ssh.common.connectionFailed')
    default:
      return t('ssh.term.closed')
  }
}

onMounted(() => {
  if (!initialId) feedback.warning(t('ssh.common.missingParams'))
  void loadHosts()
})
</script>

<style scoped lang="scss">
.ssh-term {
  display: flex;
  flex-direction: column;
  gap: 18px;
  height: 100%;
  min-height: 0;
  padding: 24px 32px 28px;
}
.ssh-term__head {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}
.ssh-term__titles {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.ssh-term__crumb {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.ssh-term__crumb a {
  color: var(--el-text-color-secondary);
  text-decoration: none;
}
.ssh-term__crumb a:hover {
  color: var(--m-pInk);
}
.ssh-term__title {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 1.75rem;
  font-weight: 800;
  letter-spacing: -0.01em;
  color: var(--el-text-color-primary);
}

/* 会话标签 */
.ssh-term__tabs {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  overflow-x: auto;
  scrollbar-width: none;
}
.ssh-term__tabs::-webkit-scrollbar {
  display: none;
}
.ssh-term__tab {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 2px;
  height: 40px;
  padding: 0 4px 0 14px;
  border-radius: 14px;
  color: var(--el-text-color-secondary);
}
.ssh-term__tab:hover {
  background: var(--el-fill-color-light);
}
.ssh-term__tab.is-on {
  background: var(--m-ink);
  color: var(--m-bg);
}
.ssh-term__tab-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 40px;
  padding: 0 4px 0 0;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
}
.ssh-term__tab-x {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border: 0;
  border-radius: 10px;
  background: transparent;
  color: inherit;
  cursor: pointer;
}
.ssh-term__tab-x:hover {
  background: color-mix(in srgb, currentColor 16%, transparent);
}
.ssh-term__tab-x :deep(.iconify) {
  width: 14px;
  height: 14px;
}
.ssh-term__dot {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--m-ink2);
}
.ssh-term__dot.is-ok {
  background: var(--m-ok);
}
.ssh-term__dot.is-warn {
  background: var(--m-warn);
}
.ssh-term__dot.is-bad {
  background: var(--m-bad);
}

/* 新建会话菜单 */
.ssh-term__menu {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 6px;
  max-height: 360px;
  overflow-y: auto;
}
.ssh-term__menu-label {
  padding: 6px 10px 4px;
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--el-text-color-secondary);
}
.ssh-term__menu-item {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 2px 8px;
  min-height: 44px;
  padding: 6px 10px;
  border: 0;
  border-radius: 10px;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.875rem;
  font-weight: 600;
  text-align: left;
  cursor: pointer;
}
.ssh-term__menu-item:hover,
.ssh-term__menu-item.is-on {
  background: var(--el-fill-color-light);
}
.ssh-term__menu-name {
  flex-basis: 100%;
}
.ssh-term__menu-sub {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--el-text-color-secondary);
}
.ssh-term__menu-empty {
  padding: 10px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.ssh-term__cmd {
  flex-wrap: nowrap;
  gap: 10px;
}
.ssh-term__cmd .m-mono {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.8125rem;
}
.ssh-term__cmd :deep(.iconify),
.ssh-term__menu-edit :deep(.iconify) {
  flex: none;
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.ssh-term__menu-edit {
  flex-wrap: nowrap;
  gap: 10px;
  margin-top: 4px;
  border-top: 1px solid var(--el-border-color-lighter);
  border-radius: 0 0 10px 10px;
  color: var(--m-pInk);
}

/* 手机：返回 + 主机 + 状态 */
.ssh-term__mhost {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.ssh-term__mname {
  font-size: 0.9375rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.ssh-term__mstatus {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.ssh-term__mstatus.is-ok {
  color: var(--m-ok);
}
.ssh-term__mstatus.is-warn {
  color: var(--m-warn);
}
.ssh-term__mstatus.is-bad {
  color: var(--m-bad);
}
.ssh-term__switch {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border: 0;
  border-radius: 14px;
  background: transparent;
  color: var(--el-text-color-secondary);
  cursor: pointer;
}

@media (width <= 768px) {
  .ssh-term {
    gap: 0;
    padding: 0;
  }
}
</style>
