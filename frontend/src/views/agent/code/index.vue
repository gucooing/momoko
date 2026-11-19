<!-- Code：左边会话栏，中间对话（标签页：对话 / 子 Agent / 打开的文件），右边改动、文件、任务、会话信息。
     没有选中会话时是新会话页。地址里的 ?session= 是当前会话，?workspace= 是新会话预选的工作区。
     Momoko 自己的侧边栏收起还是展开由用户决定，这一页不动它。 -->
<template>
  <div class="ag agc" :class="{ 'agc--mobile': isMobile }">
    <div v-if="isMobile && leftOn" class="agc-scrim" @click="leftOn = false" />
    <SessionSidebar
      v-if="leftOn"
      v-model:keyword="keyword"
      :sessions="sideSessions"
      :workspaces="sideWorkspaces"
      :runtimes="options"
      :current-id="sessionId"
      :loaded="sideLoaded"
      @open="openSession"
      @new-session="newSession"
      @changed="loadSidebar"
      @removed="onRemoved"
    />

    <NewSession
      v-if="!sessionId"
      :options="options"
      :options-loaded="optionsLoaded"
      :initial-workspace="initialWorkspace"
      :config-version="configVersion"
      :left-on="leftOn"
      :right-on="rightOn && !isMobile"
      @toggle-left="leftOn = !leftOn"
      @toggle-right="rightOn = !rightOn"
      @created="onCreated"
      @workspaces-changed="loadSidebar"
    />

    <template v-else>
      <section class="agc-card agc-main" :aria-label="t('agent.code.chat')">
        <header class="agc-head">
          <button type="button" class="ag-ib ag-ib--sm" :aria-label="leftOn ? t('agent.code.hideSide') : t('agent.code.showSide')" :title="leftOn ? t('agent.code.hideSide') : t('agent.code.showSide')" :aria-pressed="leftOn" @click="leftOn = !leftOn">
            <UIcon name="i-lucide-panel-left" />
          </button>
          <div ref="tabsEl" class="agc-tabs ag-noscroll" role="tablist" :aria-label="t('agent.code.tabs')">
            <div class="agc-tt" :class="{ 'is-on': ctab === 'chat' }">
              <button type="button" class="agc-tt__btn" role="tab" :aria-selected="ctab === 'chat'" @click="ctab = 'chat'">
                <UIcon name="i-lucide-message-square" />
                <span class="ag-ellipsis agc-tt__title">{{ sessionTitle(session?.title) }}</span>
                <span class="agc-sd" :class="dotCls" :title="statusText" />
              </button>
            </div>
            <div v-if="subId" class="agc-tt" :class="{ 'is-on': ctab === 'sub' }">
              <button type="button" class="agc-tt__btn" role="tab" :aria-selected="ctab === 'sub'" @click="ctab = 'sub'">
                <UIcon name="i-lucide-bot" /><span class="ag-ellipsis agc-tt__title">{{ t('agent.code.sub.tab', { title: subInfo.title }) }}</span>
              </button>
              <button type="button" class="agc-tt__x" :aria-label="t('agent.code.sub.close')" @click="closeSub"><UIcon name="i-lucide-x" /></button>
            </div>
            <div v-for="p in openFiles" :key="p" class="agc-tt" :class="{ 'is-on': ctab === p }">
              <button type="button" class="agc-tt__btn" role="tab" :aria-selected="ctab === p" :title="p" @click="ctab = p">
                <UIcon name="i-lucide-file" /><span class="ag-mono agc-tt__file">{{ shortName(p) }}</span>
              </button>
              <button type="button" class="agc-tt__x" :aria-label="t('agent.code.closeTab', { path: p })" @click="closeFile(p)"><UIcon name="i-lucide-x" /></button>
            </div>
          </div>
          <div v-if="openFiles.length >= 2" ref="allTabsEl" class="agc-rel">
            <button type="button" class="agc-cl agc-cl--count" :aria-label="t('agent.code.allTabs', { n: tabCount })" :title="t('agent.code.allTabsTip')" aria-haspopup="menu" :aria-expanded="tabsOpen" @click="tabsOpen = !tabsOpen">
              <span class="ag-num">{{ tabCount }}</span><UIcon name="i-lucide-chevron-down" />
            </button>
            <div v-if="tabsOpen" class="ag-menu agc-tabsmenu" role="menu" :aria-label="t('agent.code.allTabsTip')">
              <button type="button" class="ag-mi" :class="{ 'is-on': ctab === 'chat' }" role="menuitemradio" :aria-checked="ctab === 'chat'" @click="pickTab('chat')">
                <UIcon name="i-lucide-message-square" class="ag-muted" /><span class="ag-ellipsis agc-flex">{{ sessionTitle(session?.title) }}</span>
              </button>
              <button v-if="subId" type="button" class="ag-mi" :class="{ 'is-on': ctab === 'sub' }" role="menuitemradio" :aria-checked="ctab === 'sub'" @click="pickTab('sub')">
                <UIcon name="i-lucide-bot" class="ag-muted" /><span class="ag-ellipsis agc-flex">{{ t('agent.code.sub.tab', { title: subInfo.title }) }}</span>
              </button>
              <div class="ag-mi-sep" />
              <div v-for="p in openFiles" :key="p" class="agc-tabsmenu__row">
                <button type="button" class="ag-mi agc-flex" :class="{ 'is-on': ctab === p }" role="menuitemradio" :aria-checked="ctab === p" @click="pickTab(p)">
                  <UIcon name="i-lucide-file" class="ag-muted" /><span class="ag-mono ag-ellipsis agc-flex">{{ p }}</span>
                </button>
                <button type="button" class="agc-sm" :aria-label="t('agent.code.closeTab', { path: p })" @click="closeFile(p)"><UIcon name="i-lucide-x" /></button>
              </div>
              <div class="ag-mi-sep" />
              <button type="button" class="ag-mi ag-muted" role="menuitem" @click="closeOthers">{{ t('agent.code.closeOthers') }}</button>
            </div>
          </div>
          <button type="button" class="ag-ib ag-ib--sm" :aria-label="rightOn ? t('agent.code.hideRight') : t('agent.code.showRight')" :title="rightOn ? t('agent.code.hideRight') : t('agent.code.showRight')" :aria-pressed="rightOn" @click="rightOn = !rightOn">
            <UIcon name="i-lucide-panel-right" />
          </button>
        </header>

        <div v-if="socket.openError.value" class="agc-gone">
          <div>{{ socket.openError.value }}</div>
          <button type="button" class="ag-btn ag-btn--p" @click="newSession('')">{{ t('agent.common.newSession') }}</button>
        </div>

        <template v-else-if="ctab === 'chat'">
          <ChatLog
            :items="mainItems"
            :session-id="session?.id || ''"
            :more="socket.more.value"
            :loading-more="socket.loadingMore.value"
            :agent-kind="agentKindOf(session?.runtimeKind)"
            :agent-name="session?.runtimeName || ''"
            :cwd="session?.cwd || ''"
            :busy="busy"
            :empty="socket.loaded.value ? '' : t('agent.common.loading')"
            @resolve="resolve"
            @show-diff="showDiff"
            @open-sub="openSub"
            @more="socket.loadMore"
          />
          <div class="agc-foot">
            <div v-if="plan.length" class="agc-plan">
              <ol v-if="planOpen" class="agc-plan__list" :aria-label="t('agent.code.plan.steps')">
                <li v-for="(p, i) in plan" :key="i" class="agc-plan__item" :class="{ 'ag-muted': p.status === 'pending' }">
                  <span class="agc-sd" :class="planDot(p.status)" /><span class="agc-flex">{{ p.content }}</span>
                  <span class="agc-small">{{ t(`agent.code.plan.${p.status}`) }}</span>
                </li>
              </ol>
              <button type="button" class="agc-plan__bar" :aria-expanded="planOpen" @click="planOpen = !planOpen">
                <UIcon name="i-lucide-list-checks" class="ag-muted" />
                <span class="agc-plan__count">{{ t('agent.code.plan.count', { done: planDone, total: plan.length }) }}</span>
                <span class="agc-plan__segs" aria-hidden="true"><span v-for="(p, i) in plan" :key="i" :class="planDot(p.status)" /></span>
                <span class="ag-ellipsis ag-muted agc-flex">{{ planNow }}</span>
                <UIcon :name="planOpen ? 'i-lucide-chevron-down' : 'i-lucide-chevron-up'" class="ag-muted" />
              </button>
            </div>
            <div v-if="session?.status === 'error' && session.error && !lastTurnError" class="agc-notice agc-notice--error">
              <UIcon name="i-lucide-circle-x" /><span>{{ session.error }}</span>
            </div>
            <div v-if="session?.status === 'archived'" class="agc-notice agc-notice--info">
              <UIcon name="i-lucide-archive" /><span class="agc-flex">{{ t('agent.code.archivedNote') }}</span>
              <button type="button" class="ag-btn ag-btn--sm" @click="unarchive">{{ t('agent.code.unarchive') }}</button>
            </div>
            <Composer
              v-else-if="session"
              ref="composer"
              :runtimes="options"
              :runtime-id="session.runtimeId"
              :provider-id="session.providerId"
              :model="session.model"
              :effort="session.effort"
              :mode="session.mode"
              :busy="busy"
              :commands="socket.state.value.commands ?? []"
              :skills="socket.detail.value?.skillNames ?? []"
              :session-id="session.id"
              :context-used="socket.contextUsed.value"
              :context-size="socket.contextSize.value"
              info-button
              :label="t('agent.code.inputLabel')"
              :placeholder="t('agent.code.placeholder')"
              :on-send="send"
              @select="selectOption"
              @stop="stop"
              @show-info="openInfo"
            />
          </div>
        </template>

        <div v-else-if="ctab === 'sub'" class="agc-subview">
          <div class="agc-subview__head">
            <UIcon name="i-lucide-bot" />
            <span class="ag-ellipsis agc-flex">{{ t('agent.code.sub.head', { meta: subInfo.meta }) }}</span>
            <AgAgentTag :kind="agentKindOf(session?.runtimeKind)" :name="session?.runtimeName || ''" />
            <button type="button" class="ag-btn ag-btn--sm" @click="ctab = 'chat'">{{ t('agent.code.sub.back') }}</button>
          </div>
          <ChatLog
            :items="subItems"
            :agent-kind="agentKindOf(session?.runtimeKind)"
            :agent-name="session?.runtimeName || ''"
            :cwd="session?.cwd || ''"
            :busy="subInfo.running"
            sub
            :empty="t('agent.code.sub.empty')"
            @resolve="resolve"
            @show-diff="showDiff"
            @open-sub="openSub"
          />
        </div>

        <FileTab
          v-else-if="session"
          :key="ctab"
          :session-id="session.id"
          :path="ctab"
          :cwd="session.cwd"
          :change="changes.find((c) => c.path === ctab) ?? null"
          @show-diff="showDiff"
        />
      </section>

      <div v-if="isMobile && rightOn" class="agc-scrim" @click="rightOn = false" />
      <RightPanel
        v-if="rightOn && session"
        v-model:tab="rightTab"
        v-model:focus="diffFocus"
        :session="session"
        :detail="socket.detail.value"
        :items="socket.items.value"
        :state="socket.state.value"
        :context-used="socket.contextUsed.value"
        :context-size="socket.contextSize.value"
        :busy="busy"
        :tick="turnTick"
        @open-file="openFile"
        @open-sub="openSub"
        @command="runCommand"
        @changes="changes = $event"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { onClickOutside, useDebounceFn, useWindowSize } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import {
  archiveAgentSessions,
  cancelAgentSession,
  getAgentOptions,
  getAgentSidebar,
  promptAgentSession,
  resolveAgentPermission,
  setAgentSessionOption,
} from '@/api/agent'
import { useFullBleed } from '@/composables/useAppLayout'
import type { AgentChangedFile, AgentRuntimeOptions, AgentSession, AgentSidebarWorkspace } from '@/types/v1/agent'
import AgAgentTag from '@/views/agent/components/AgAgentTag.vue'
import { useLastCodeSession } from '@/views/agent/composables/useLastCodeSession'
import { agentKindOf, fmtDuration } from '@/views/agent/utils/format'
import { sessionTitle } from '@/views/agent/utils/labels'
import ChatLog from './ChatLog.vue'
import Composer, { type DraftAttachment, type Selection } from './Composer.vue'
import FileTab from './FileTab.vue'
import NewSession from './NewSession.vue'
import RightPanel, { type RightTab } from './RightPanel.vue'
import SessionSidebar from './SessionSidebar.vue'
import { useAgentSocket } from './useAgentSocket'

defineOptions({ name: 'AgentCodeView' })

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
useFullBleed()

const PAGE = '/agent/code'
const { width } = useWindowSize()
const isMobile = computed(() => width.value < 768)

// 长连接：页面在（包括从缓存里切回来）就连着
onMounted(() => socket.start())
onActivated(() => socket.start())

// —— 两侧栏的开关（记住上次的） ——
const persisted = (key: string, def: boolean) => {
  const r = ref(localStorage.getItem(key) == null ? def : localStorage.getItem(key) === '1')
  watch(r, (v) => {
    if (!isMobile.value) localStorage.setItem(key, v ? '1' : '0')
  })
  return r
}
const leftOn = persisted('agent.code.left', true)
const rightOn = persisted('agent.code.right', true)
watch(
  isMobile,
  (m) => {
    if (m) {
      leftOn.value = false
      rightOn.value = false
    }
  },
  { immediate: true },
)

// —— 选项 ——
const options = ref<AgentRuntimeOptions[]>([])
const optionsLoaded = ref(false)
const loadOptions = async () => {
  try {
    const { data } = await getAgentOptions()
    options.value = data.runtimes ?? []
  } catch {
    /* shown */
  } finally {
    optionsLoaded.value = true
  }
}

// —— 会话栏 ——
const keyword = ref('')
const sideSessions = ref<AgentSession[]>([])
const sideWorkspaces = ref<AgentSidebarWorkspace[]>([])
const sideLoaded = ref(false)
let sideSeq = 0
const loadSidebar = async () => {
  const id = ++sideSeq
  try {
    const { data } = await getAgentSidebar(keyword.value.trim())
    if (id !== sideSeq) return
    sideSessions.value = data.sessions ?? []
    sideWorkspaces.value = data.workspaces ?? []
  } catch {
    /* shown */
  } finally {
    if (id === sideSeq) sideLoaded.value = true
  }
}
watch(keyword, useDebounceFn(() => void loadSidebar(), 250))

// 会话变了：会话栏里换成新的；归档了就拿掉；自己的、会话栏里还没有的（别处新建、取消归档）插到最前面。
// 别人的会话（有管理权限的人打开看的）不进会话栏。
const userStore = useUserStore()
const onSessionPush = (s: AgentSession) => {
  const list = sideSessions.value
  const i = list.findIndex((x) => x.id === s.id)
  if (s.status === 'archived') {
    if (i >= 0) list.splice(i, 1)
    return
  }
  if (i >= 0) list[i] = s
  else if (s.userId === userStore.userInfo?.userId) list.unshift(s)
}
const onGone = (id: string) => {
  sideSessions.value = sideSessions.value.filter((s) => s.id !== id)
  if (id !== sessionId.value) return
  // 人在别的页：不拉回来，只是忘掉它，下次回 Code 就是新会话页
  if (route.path === PAGE) newSession('')
  else {
    remember('')
    socket.open('')
  }
}

// Agent 配置在管理页（或别的页面、别人）改了：这一页开着就马上重读选项和会话栏，在后台就等切回来再读。
let visible = true
let configStale = false
const configVersion = ref(0)
const reloadConfig = useDebounceFn(() => {
  configStale = false
  configVersion.value += 1
  void loadOptions()
  void loadSidebar()
}, 300)
const onConfig = () => {
  if (visible) void reloadConfig()
  else configStale = true
}
onActivated(() => {
  visible = true
  if (configStale) void reloadConfig()
})
onDeactivated(() => {
  visible = false
})

const socket = useAgentSocket({ onSession: onSessionPush, onGone, onConfig })

// —— 当前会话 ——
// 从别的页点回 Code、地址里没带会话时回到上次看的会话，它没了才是新会话页
const { id: lastSession, remember } = useLastCodeSession()
const sessionId = computed(() => {
  if (route.path !== PAGE) return socket.sessionId.value
  return String(route.query.session || (route.query.workspace ? '' : lastSession.value))
})
// 地址里补上回到的会话；打不开（被删了）就去新会话页
let restored = ''
watch(
  () => route.fullPath,
  () => {
    if (route.path !== PAGE || route.query.session || route.query.workspace || !lastSession.value) return
    restored = lastSession.value
    void router.replace({ path: PAGE, query: { session: restored } })
  },
  { immediate: true },
)
watch(socket.openError, (err) => {
  if (!err || !restored || sessionId.value !== restored) return
  restored = ''
  remember('')
  void router.replace({ path: PAGE })
})
watch(socket.session, (s) => {
  if (s && s.id === restored) restored = ''
})
const initialWorkspace = computed(() => String(route.query.workspace || ''))
const session = computed<AgentSession | null>(() => socket.session.value ?? sideSessions.value.find((s) => s.id === sessionId.value) ?? null)
const busy = computed(() => session.value?.status === 'running' || session.value?.status === 'waiting')
const DOT: Record<string, string> = { running: 'agc-sd--run agc-pulse', waiting: 'agc-sd--wait', error: 'agc-sd--bad' }
const dotCls = computed(() => DOT[session.value?.status ?? ''] ?? '')
const statusText = computed(() => (session.value ? t(`agent.status.${session.value.status}`) : ''))

const openSession = (id: string) => {
  if (isMobile.value) leftOn.value = false
  if (id !== sessionId.value) router.push({ path: PAGE, query: { session: id } })
}
const newSession = (workspaceId: string) => {
  if (isMobile.value) leftOn.value = false
  remember('')
  router.push({ path: PAGE, query: workspaceId ? { workspace: workspaceId } : {} })
}
const onCreated = (s: AgentSession) => {
  sideSessions.value = [s, ...sideSessions.value.filter((x) => x.id !== s.id)]
  router.replace({ path: PAGE, query: { session: s.id } })
}
const onRemoved = (ids: string[]) => {
  if (ids.includes(sessionId.value)) newSession('')
}

// 每个会话自己的标签页（打开的文件、子 Agent），切回来还在
interface Tabs {
  files: string[]
  sub: string
  ctab: string
}
const tabsBySession = new Map<string, Tabs>()
const openFiles = ref<string[]>([])
const subId = ref('')
const ctab = ref('chat')
const planOpen = ref(false)
/** 右侧「改动」最近一次读到的未提交改动：文件标签页拿它显示 +/− */
const changes = ref<AgentChangedFile[]>([])
const saveTabs = (id: string) => {
  if (id) tabsBySession.set(id, { files: [...openFiles.value], sub: subId.value, ctab: ctab.value })
}

watch(
  sessionId,
  (id, old) => {
    if (route.path !== PAGE && old !== undefined) return
    if (route.path === PAGE) remember(id)
    if (old) saveTabs(old)
    const saved = tabsBySession.get(id)
    openFiles.value = saved?.files ?? []
    subId.value = saved?.sub ?? ''
    ctab.value = saved?.ctab ?? 'chat'
    planOpen.value = false
    changes.value = []
    socket.open(id)
  },
  { immediate: true },
)

// —— 标签页 ——
const tabsEl = ref<HTMLElement | null>(null)
const allTabsEl = ref<HTMLElement | null>(null)
const tabsOpen = ref(false)
onClickOutside(allTabsEl, () => (tabsOpen.value = false))
const tabCount = computed(() => openFiles.value.length + 1 + (subId.value ? 1 : 0))
const shortName = (p: string) => p.split('/').slice(-2).join('/')
const pickTab = (k: string) => {
  ctab.value = k
  tabsOpen.value = false
}
const openFile = (path: string) => {
  if (!openFiles.value.includes(path)) openFiles.value.push(path)
  ctab.value = path
  if (isMobile.value) rightOn.value = false
}
const closeFile = (path: string) => {
  openFiles.value = openFiles.value.filter((p) => p !== path)
  if (ctab.value === path) ctab.value = 'chat'
}
const closeOthers = () => {
  openFiles.value = openFiles.value.filter((p) => p === ctab.value)
  tabsOpen.value = false
}
watch(ctab, () =>
  nextTick(() => tabsEl.value?.querySelector('[aria-selected="true"]')?.scrollIntoView({ inline: 'nearest', block: 'nearest' })),
)

// —— 对话 ——
const mainItems = computed(() => socket.items.value.filter((i) => !i.sub))
const subItems = computed(() => socket.items.value.filter((i) => i.sub === subId.value))
const subInfo = computed(() => {
  const item = socket.items.value.find((i) => i.type === 'subagent' && i.subagent?.id === subId.value)
  const sa = item?.subagent
  const running = sa?.state === 'running' || sa?.state === 'requires_action'
  const took = sa?.endAt && item?.at ? fmtDuration(sa.endAt - item.at) : ''
  return {
    title: sa?.title || sa?.description || t('agent.code.sub.name'),
    running,
    meta: [running ? t('agent.code.sub.running') : t('agent.code.sub.done'), took ? t('agent.code.turn.took', { time: took }) : ''].filter(Boolean).join(' · '),
  }
})
const openSub = (id: string) => {
  subId.value = id
  ctab.value = 'sub'
}
const closeSub = () => {
  subId.value = ''
  if (ctab.value === 'sub') ctab.value = 'chat'
}
const lastTurnError = computed(() => {
  for (let i = mainItems.value.length - 1; i >= 0; i -= 1) {
    const it = mainItems.value[i]
    if (it?.type === 'turn') return !!it.turn?.error
    if (it?.type === 'user') return false
  }
  return false
})

/** 最近一轮结束的那条事件：变了说明又结束了一轮，右侧的改动和文件跟着刷新 */
const turnTick = computed(() => {
  for (let i = mainItems.value.length - 1; i >= 0; i -= 1) {
    if (mainItems.value[i]?.type === 'turn') return mainItems.value[i]!.seq
  }
  return 0
})

// —— 计划 ——
const plan = computed(() => socket.state.value.plan ?? [])
const planDone = computed(() => plan.value.filter((p) => p.status === 'completed').length)
const planNow = computed(() => {
  const now = plan.value.find((p) => p.status === 'in_progress')
  if (now) return t('agent.code.plan.now', { step: now.content })
  return planDone.value === plan.value.length ? t('agent.code.plan.allDone') : t('agent.code.plan.next', { step: plan.value.find((p) => p.status === 'pending')?.content || '' })
})
const planDot = (status: string) => (status === 'completed' ? 'agc-sd--ok' : status === 'in_progress' ? 'agc-sd--wait' : '')

// —— 动作 ——
const composer = ref<InstanceType<typeof Composer> | null>(null)
const send = async (text: string, attachments: DraftAttachment[]): Promise<boolean> => {
  const s = session.value
  if (!s) return false
  try {
    await promptAgentSession(s.id, { text, attachments })
    return true
  } catch {
    return false
  }
}
const stop = async () => {
  if (!session.value) return
  try {
    await cancelAgentSession(session.value.id)
  } catch {
    /* shown */
  }
}
const resolve = async (requestId: string, optionId: string) => {
  if (!session.value) return
  try {
    await resolveAgentPermission(session.value.id, requestId, optionId)
  } catch {
    /* shown */
  }
}
const selectOption = async (patch: Partial<Selection>) => {
  const s = session.value
  if (!s) return
  let req: { kind: 'mode' | 'model' | 'effort'; value: string; providerId?: string } | null = null
  if (patch.mode != null) req = { kind: 'mode', value: patch.mode }
  else if (patch.model != null) req = { kind: 'model', value: patch.model, providerId: patch.providerId ?? s.providerId }
  else if (patch.effort != null) req = { kind: 'effort', value: patch.effort }
  if (!req) return
  try {
    const { data } = await setAgentSessionOption(s.id, req)
    if (data.session) {
      socket.session.value = data.session
      onSessionPush(data.session)
    }
  } catch {
    /* shown */
  }
}
// 右侧上下文卡片上的「压缩上下文」：发 Agent 自己的 /compact，结果照常出现在对话里
const runCommand = async (text: string) => {
  if (!session.value) return
  try {
    await promptAgentSession(session.value.id, { text })
  } catch {
    /* shown */
  }
}
const unarchive = async () => {
  if (!session.value) return
  try {
    await archiveAgentSessions([session.value.id], false)
    feedback.success(t('agent.code.unarchived'))
    void loadSidebar()
  } catch {
    /* shown */
  }
}

// —— 右侧 ——
const rightTab = ref<RightTab>('diff')
const diffFocus = ref('')
const showDiff = (path: string) => {
  rightOn.value = true
  rightTab.value = 'diff'
  diffFocus.value = path
}
const openInfo = () => {
  rightOn.value = true
  rightTab.value = 'info'
}

onMounted(() => {
  void loadOptions()
  void loadSidebar()
})
</script>

<style lang="scss">
@use '@/views/agent/styles/agent.scss';
@use './code.scss';
</style>
