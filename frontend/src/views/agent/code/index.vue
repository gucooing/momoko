<!-- Code：左边会话栏，中间对话（标签页：对话 / 子 Agent（可以开好几个）/ 打开的文件），右边改动、文件、任务、会话信息。
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
            <div v-for="id in openSubs" :key="subTab(id)" class="agc-tt" :class="{ 'is-on': ctab === subTab(id) }">
              <button
                type="button"
                class="agc-tt__btn"
                role="tab"
                :aria-selected="ctab === subTab(id)"
                :title="t('agent.code.sub.tab', { title: subTitle(id) })"
                @click="ctab = subTab(id)"
              >
                <UIcon name="i-lucide-bot" /><span class="ag-ellipsis agc-tt__title">{{ subTitle(id) }}</span>
                <span v-if="subDot(id)" class="agc-sd" :class="subDot(id)" />
              </button>
              <button type="button" class="agc-tt__x" :aria-label="t('agent.code.sub.close')" @click="closeSub(id)"><UIcon name="i-lucide-x" /></button>
            </div>
            <div v-if="commitTab" class="agc-tt" :class="{ 'is-on': ctab === COMMIT_TAB }">
              <button type="button" class="agc-tt__btn" role="tab" :aria-selected="ctab === COMMIT_TAB" :title="commitTab.subject" @click="ctab = COMMIT_TAB">
                <UIcon :name="commitTab.stash ? 'i-lucide-archive' : 'i-lucide-git-commit-horizontal'" />
                <span class="ag-ellipsis agc-tt__title"><span class="ag-mono">{{ commitTab.short }}</span> {{ commitTab.subject }}</span>
              </button>
              <button type="button" class="agc-tt__x" :aria-label="t('agent.git.view.close')" @click="closeCommit"><UIcon name="i-lucide-x" /></button>
            </div>
            <div v-for="p in openFiles" :key="p" class="agc-tt" :class="{ 'is-on': ctab === p }">
              <button type="button" class="agc-tt__btn" role="tab" :aria-selected="ctab === p" :title="p" @click="ctab = p">
                <UIcon name="i-lucide-file" /><span class="ag-mono agc-tt__file">{{ shortName(p) }}</span>
              </button>
              <button
                type="button"
                class="agc-tt__x"
                :class="{ 'is-dirty': tabDirty(p) }"
                :aria-label="t('agent.code.closeTab', { path: p })"
                :title="tabDirty(p) ? t('agent.code.file.unsaved') : undefined"
                @click="closeFile(p)"
              >
                <span class="agc-tt__dot" /><UIcon name="i-lucide-x" />
              </button>
            </div>
          </div>
          <div v-if="openFiles.length + openSubs.length >= 2" ref="allTabsEl" class="agc-rel">
            <button type="button" class="agc-cl agc-cl--count" :aria-label="t('agent.code.allTabs', { n: tabCount })" :title="t('agent.code.allTabsTip')" aria-haspopup="menu" :aria-expanded="tabsOpen" @click="tabsOpen = !tabsOpen">
              <span class="ag-num">{{ tabCount }}</span><UIcon name="i-lucide-chevron-down" />
            </button>
            <div v-if="tabsOpen" class="ag-menu agc-tabsmenu" role="menu" :aria-label="t('agent.code.allTabsTip')">
              <button type="button" class="ag-mi" :class="{ 'is-on': ctab === 'chat' }" role="menuitemradio" :aria-checked="ctab === 'chat'" @click="pickTab('chat')">
                <UIcon name="i-lucide-message-square" class="ag-muted" /><span class="ag-ellipsis agc-flex">{{ sessionTitle(session?.title) }}</span>
              </button>
              <div v-for="id in openSubs" :key="subTab(id)" class="agc-tabsmenu__row">
                <button type="button" class="ag-mi agc-flex" :class="{ 'is-on': ctab === subTab(id) }" role="menuitemradio" :aria-checked="ctab === subTab(id)" @click="pickTab(subTab(id))">
                  <UIcon name="i-lucide-bot" class="ag-muted" /><span class="ag-ellipsis agc-flex">{{ t('agent.code.sub.tab', { title: subTitle(id) }) }}</span>
                </button>
                <button type="button" class="agc-sm" :aria-label="t('agent.code.sub.close')" @click="closeSub(id)"><UIcon name="i-lucide-x" /></button>
              </div>
              <button v-if="commitTab" type="button" class="ag-mi" :class="{ 'is-on': ctab === COMMIT_TAB }" role="menuitemradio" :aria-checked="ctab === COMMIT_TAB" @click="pickTab(COMMIT_TAB)">
                <UIcon name="i-lucide-git-commit-horizontal" class="ag-muted" /><span class="ag-ellipsis agc-flex">{{ commitTab.short }} {{ commitTab.subject }}</span>
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
            :subagents="socket.state.value.subagents"
            :session-id="session?.id || ''"
            :more="socket.more.value"
            :loading-more="socket.loadingMore.value"
            :agent-kind="agentKindOf(session?.runtimeKind)"
            :agent-name="session?.runtimeName || ''"
            :cwd="session?.cwd || ''"
            :busy="busy"
            :background="bgTools"
            :empty="socket.loaded.value ? '' : t('agent.common.loading')"
            :queued="queue"
            @resolve="resolve"
            @answer="answerQuestion"
            @stop="stopItem"
            @show-diff="showDiff"
            @open-sub="openSub"
            @more="socket.loadMore"
            @unqueue="unqueue"
            @send-now="sendNow"
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
              :unrestricted="session.unrestricted"
              :busy="busy"
              :commands="socket.state.value.commands ?? []"
              :skills="socket.detail.value?.skillNames ?? []"
              :session-id="session.id"
              :context-used="socket.contextUsed.value"
              :context-size="socket.contextSize.value"
              :label="t('agent.code.inputLabel')"
              :placeholder="busy ? t('agent.code.placeholderBusy') : t('agent.code.placeholder')"
              :on-send="send"
              @select="selectOption"
              @stop="stop"
            />
          </div>
        </template>

        <div v-else-if="curSub" class="agc-subview">
          <div class="agc-subview__head">
            <UIcon name="i-lucide-bot" />
            <span class="ag-ellipsis agc-flex">{{ t('agent.code.sub.head', { meta: subInfo.meta }) }}</span>
            <AgAgentTag :kind="agentKindOf(session?.runtimeKind)" :name="session?.runtimeName || ''" />
            <button type="button" class="ag-btn ag-btn--sm" @click="ctab = 'chat'">{{ t('agent.code.sub.back') }}</button>
          </div>
          <ChatLog
            :key="curSub"
            :items="subItems"
            :subagents="socket.state.value.subagents"
            :session-id="session?.id || ''"
            :agent-kind="agentKindOf(session?.runtimeKind)"
            :agent-name="session?.runtimeName || ''"
            :cwd="session?.cwd || ''"
            :busy="subInfo.running"
            sub
            :empty="t('agent.code.sub.empty')"
            @resolve="resolve"
            @answer="answerQuestion"
            @stop="stopItem"
            @show-diff="showDiff"
            @open-sub="openSub"
          />
        </div>

        <GitCommitView
          v-else-if="ctab === COMMIT_TAB && commitTab && session"
          :key="`${session.id}|${commitTab.repo}|${commitTab.hash}`"
          :session-id="session.id"
          :target="commitTab"
          @open-file="openFile"
        />

        <FileTab
          v-else-if="session"
          :key="`${session.id}|${tabKeys.key(ctab)}`"
          :session-id="session.id"
          :path="ctab"
          :cwd="session.cwd"
          :change="changes.find((c) => c.path === ctab) ?? null"
          :tick="turnTick + gitTick"
          @show-diff="showDiff"
          @attach="attachSnippet"
          @open="openFile"
          @navigate="onTabNavigate"
        />
      </section>

      <div v-if="isMobile && rightOn" class="agc-scrim" @click="rightOn = false" />
      <div
        v-if="rightOn && session && !isMobile"
        class="agc-grip"
        role="separator"
        aria-orientation="vertical"
        tabindex="0"
        :aria-valuenow="rightWidth"
        :aria-label="t('agent.code.resizeRight')"
        :title="t('agent.code.resizeRightTip')"
        @pointerdown="startRightResize"
        @dblclick="resetRightWidth"
        @keydown="onRightGripKey"
      />
      <RightPanel
        v-if="rightOn && session"
        :style="rightStyle(isMobile)"
        v-model:tab="rightTab"
        v-model:focus="diffFocus"
        :session="session"
        :detail="socket.detail.value"
        :items="socket.items.value"
        :state="socket.state.value"
        :background="bgTools"
        :context-used="socket.contextUsed.value"
        :context-size="socket.contextSize.value"
        :busy="busy"
        :tick="turnTick"
        :sub="curSub"
        :active-file="openFiles.includes(ctab) ? ctab : ''"
        :opened-commit="ctab === COMMIT_TAB ? commitTab?.hash : ''"
        @open-file="openFile"
        @moved="onFileMoved"
        @deleted="onFilesDeleted"
        @reference="referenceFile"
        @open-sub="openSub"
        @stop="stopItem"
        @command="runCommand"
        @changes="changes = $event"
        @git-changed="gitTick++"
        @open-commit="openCommit"
      />
    </template>

    <AgConfirm
      v-model="closeAskOpen"
      :title="t('agent.code.file.closeDirtyTitle')"
      :message="t('agent.code.file.closeDirtyMsg', { names: closeAsk.filter(tabDirty).map(shortName).join('、') })"
      :confirm-text="t('agent.code.file.closeDirty')"
      danger
      :on-confirm="() => closeFiles(closeAsk)"
    />
  </div>
</template>

<script setup lang="ts">
import { onClickOutside, useDebounceFn, useEventListener, useWindowSize } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import {
  archiveAgentSessions,
  cancelAgentSession,
  getAgentSidebar,
  promptAgentSession,
  resolveAgentPermission,
  stopAgentSessionItem,
  deleteAgentQueuedPrompt,
  sendAgentQueuedPrompt,
  setAgentSessionOption,
} from '@/api/agent'
import { useFullBleed } from '@/composables/useAppLayout'
import { useDockSessionDir } from '@/layouts/app/termDock/state'
import type { AgentSession, AgentSidebarWorkspace } from '@/types/v1/agent'
import AgAgentTag from '@/views/agent/components/AgAgentTag.vue'
import AgConfirm from '@/views/agent/components/AgConfirm.vue'
import { useAgentModels } from '@/views/agent/composables/useAgentModels'
import { useLastCodeSession } from '@/views/agent/composables/useLastCodeSession'
import { agentKindOf, fmtDuration } from '@/views/agent/utils/format'
import { sessionTitle, subagentStateName, subagentWorking } from '@/views/agent/utils/labels'
import ChatLog from './ChatLog.vue'
import Composer, { type DraftAttachment, type DraftSnippet, type Selection, type SendMode } from './Composer.vue'
import FileTab from './FileTab.vue'
import { onRightGripKey, resetRightWidth, rightStyle, rightWidth, startRightResize } from './rightWidth'
import { dirtyFiles, dropFileState, isDirty, isUnder, movedPath, moveFileState, useTabKeys } from './fileState'
import NewSession from './NewSession.vue'
import RightPanel, { type RightTab } from './RightPanel.vue'
import GitCommitView from './git/GitCommitView.vue'
import type { GitChange, GitCommitTarget } from './git/useGit'
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

// —— 选项：和别的页面共用一份模型清单（模型供应商页同步、配置的） ——
const { runtimes: options, loaded: optionsLoaded, refresh: loadOptions } = useAgentModels()

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
// 顶栏终端的「在当前会话的目录打开」
useDockSessionDir(() => session.value)
/**
 * 主会话这一轮在跑（输入框的停止和排队、对话末尾的「正在处理」看它）。会话状态在子 Agent 干活时也算运行中，
 * 所以 Agent 报了没有一轮在跑、只是子 Agent 在干活时不算；进程刚起来还没报的时候照会话状态算。
 */
const busy = computed(() => {
  const status = session.value?.status
  if (status !== 'running' && status !== 'waiting') return false
  const st = socket.state.value
  return !!st.turn || !(st.subagents ?? []).some(subagentWorking)
})
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
  subs: string[]
  commit: GitCommitTarget | null
  ctab: string
}
const tabsBySession = new Map<string, Tabs>()
const openFiles = ref<string[]>([])
/** 开着的子 Agent 标签（它们的会话 id），和文件一样可以开好几个 */
const openSubs = ref<string[]>([])
/** 子 Agent 标签的 key：冒号开头，和文件路径分开 */
const SUB_TAB = ':sub:'
const subTab = (id: string) => SUB_TAB + id
/** 中间开着的提交（或储藏）：在右侧 Git 的历史、储藏里点的，一次只开一个 */
const COMMIT_TAB = ':commit'
const commitTab = ref<GitCommitTarget | null>(null)
const ctab = ref('chat')
/** 当前标签是子 Agent 时是它的会话 id */
const curSub = computed(() => (ctab.value.startsWith(SUB_TAB) ? ctab.value.slice(SUB_TAB.length) : ''))
const planOpen = ref(false)
/** 右侧「改动」最近一次读到的未提交改动：文件标签页拿它显示 +/− */
const changes = ref<GitChange[]>([])
/** Git 里做了会改文件的操作（撤销、切换分支、拉取……）就加一：打开的文件跟着重读 */
const gitTick = ref(0)
const saveTabs = (id: string) => {
  if (id) tabsBySession.set(id, { files: [...openFiles.value], subs: [...openSubs.value], commit: commitTab.value, ctab: ctab.value })
}

watch(
  sessionId,
  (id, old) => {
    if (route.path !== PAGE && old !== undefined) return
    if (route.path === PAGE) remember(id)
    if (old) saveTabs(old)
    const saved = tabsBySession.get(id)
    openFiles.value = saved?.files ?? []
    openSubs.value = saved?.subs ?? []
    commitTab.value = saved?.commit ?? null
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
const tabCount = computed(() => openFiles.value.length + 1 + openSubs.value.length + (commitTab.value ? 1 : 0))
const shortName = (p: string) => p.split('/').slice(-2).join('/')
const pickTab = (k: string) => {
  ctab.value = k
  tabsOpen.value = false
}
const tabKeys = useTabKeys()
const openFile = (path: string) => {
  if (!openFiles.value.includes(path)) {
    openFiles.value.push(path)
    tabKeys.fresh(path)
  }
  ctab.value = path
  if (isMobile.value) rightOn.value = false
}
// —— 打开的文件：没保存的改动、树里改名 / 挪动 / 删除 ——
const tabDirty = (p: string) => isDirty(sessionId.value, p)
/** 关掉这些标签（没保存的改动丢掉） */
const closeFiles = (paths: string[]) => {
  dropFileState(sessionId.value, paths)
  for (const p of paths) tabKeys.drop(p)
  openFiles.value = openFiles.value.filter((p) => !paths.includes(p))
  if (paths.includes(ctab.value)) ctab.value = 'chat'
}
/** 要关的标签里有没保存的：先问一下 */
const closeAsk = ref<string[]>([])
const closeAskOpen = ref(false)
const askClose = (paths: string[]) => {
  if (!paths.some(tabDirty)) {
    closeFiles(paths)
    return
  }
  closeAsk.value = paths
  closeAskOpen.value = true
}
const closeFile = (path: string) => askClose([path])
const closeOthers = () => {
  tabsOpen.value = false
  askClose(openFiles.value.filter((p) => p !== ctab.value))
}
const onFileMoved = (from: string, to: string) => {
  moveFileState(sessionId.value, from, to)
  const wasOpen = openFiles.value.includes(ctab.value)
  openFiles.value = openFiles.value.map((p) => {
    const next = movedPath(p, from, to)
    if (next !== p) {
      tabKeys.drop(p)
      tabKeys.fresh(next)
    }
    return next
  })
  if (wasOpen) ctab.value = movedPath(ctab.value, from, to)
}
// 文件标签的 HTML 预览里跳到了别的页面：标签就地换成那个文件（已经开着那个文件的标签合并掉）
const onTabNavigate = (from: string, to: string) => {
  openFiles.value = openFiles.value.filter((p) => p !== to).map((p) => (p === from ? to : p))
  tabKeys.carry(from, to)
  if (ctab.value === from) ctab.value = to
}
const onFilesDeleted = (paths: string[]) => {
  dropFileState(sessionId.value, paths)
  const gone = openFiles.value.filter((p) => paths.some((d) => isUnder(p, d)))
  if (gone.length) closeFiles(gone)
}
// 文件树右键「添加到对话」：回到对话，加一个引用
const referenceFile = async (path: string) => {
  ctab.value = 'chat'
  if (isMobile.value) rightOn.value = false
  await nextTick()
  composer.value?.addRef(path)
  composer.value?.focus()
}
// 有没保存的文件时关掉或刷新页面，浏览器先问一下
useEventListener(window, 'beforeunload', (e: BeforeUnloadEvent) => {
  if (!dirtyFiles.size) return
  e.preventDefault()
  e.returnValue = ''
})
watch(ctab, () =>
  nextTick(() => tabsEl.value?.querySelector('[aria-selected="true"]')?.scrollIntoView({ inline: 'nearest', block: 'nearest' })),
)

// —— 对话 ——
const mainItems = computed(() => socket.items.value.filter((i) => !i.sub))
const subItems = computed(() => socket.items.value.filter((i) => i.sub === curSub.value))
/** 一个子 Agent：状态只看 Agent 经 ACP 报的那份；不在里面的（上一个进程派出的）只从对话里的卡片取标题 */
const subOf = (id: string) => {
  const live = socket.state.value.subagents?.find((sa) => sa.id === id)
  const item = live ? undefined : socket.items.value.find((i) => i.type === 'subagent' && i.subagent?.id === id)
  return { live, sa: live ?? item?.subagent, at: item?.at ?? 0 }
}
const subTitle = (id: string) => {
  const { sa } = subOf(id)
  return sa?.title || sa?.description || t('agent.code.sub.name')
}
const SUB_DOT: Record<string, string> = { running: 'agc-sd--run agc-pulse', requires_action: 'agc-sd--wait' }
/** 标签上的小圆点：在跑、等人回应时才有 */
const subDot = (id: string) => SUB_DOT[subOf(id).live?.state ?? ''] ?? ''
const subInfo = computed(() => {
  const { live, sa, at } = subOf(curSub.value)
  const running = subagentWorking(live)
  const startAt = sa?.startAt || at
  const took = !running && sa?.endAt && startAt ? fmtDuration(sa.endAt - startAt) : ''
  return {
    running,
    meta: [subagentStateName(live?.state), took ? t('agent.code.turn.took', { time: took }) : ''].filter(Boolean).join(' · '),
  }
})
/** 打开一个子 Agent 的标签：已经开着就切过去 */
const openSub = (id: string) => {
  if (!openSubs.value.includes(id)) openSubs.value.push(id)
  ctab.value = subTab(id)
}
const closeSub = (id: string) => {
  openSubs.value = openSubs.value.filter((s) => s !== id)
  if (ctab.value === subTab(id)) ctab.value = 'chat'
}
const openCommit = (c: GitCommitTarget) => {
  commitTab.value = c
  ctab.value = COMMIT_TAB
  if (isMobile.value) rightOn.value = false
}
const closeCommit = () => {
  commitTab.value = null
  if (ctab.value === COMMIT_TAB) ctab.value = 'chat'
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

// —— 后台 ——
/** 转到后台还在跑的工具调用 id（ACP v2，Agent 报的后台列表）：工具调用本身报了完成，命令其实还在跑（起的服务之类）。
 *  进程没了后台的也就没了，所以只在有进程时算 */
const bgTools = computed(() => {
  if (!session.value?.live) return []
  return (socket.state.value.background ?? []).flatMap((b) => (b.type !== 'subagent' && b.id ? [b.id] : []))
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
const send = async (text: string, attachments: DraftAttachment[], mode?: SendMode): Promise<boolean> => {
  const s = session.value
  if (!s) return false
  try {
    await promptAgentSession(s.id, { text, attachments, mode })
    return true
  } catch {
    return false
  }
}
// 排着队的消息：这一轮结束后按顺序发出，没发出去之前可以撤回
const queue = computed(() => socket.state.value.queue ?? [])
const unqueue = async (queueId: string) => {
  if (!session.value) return
  try {
    await deleteAgentQueuedPrompt(session.value.id, queueId)
  } catch {
    /* shown */
  }
}
// 排着的一条马上发：插进正在跑的一轮（ACP v2），插不进去的停下这一轮再发
const sendNow = async (queueId: string) => {
  if (!session.value) return
  try {
    await sendAgentQueuedPrompt(session.value.id, queueId)
  } catch {
    /* shown */
  }
}
// 打开的文件里选中一段「添加到对话」：回到对话标签，加进输入框的附件
const attachSnippet = async (sn: DraftSnippet) => {
  ctab.value = 'chat'
  await nextTick()
  composer.value?.addSnippet(sn)
}
const stop = async () => {
  if (!session.value) return
  // 停下来时排着的也不发了：文字放回输入框
  const queued = queue.value.map((q) => q.text).filter(Boolean)
  if (queued.length) composer.value?.restore(queued.join('\n\n'))
  try {
    await cancelAgentSession(session.value.id)
  } catch {
    /* shown */
  }
}
/** 回答 Agent 的提问（走会话的长连接）：答上了卡片自己会变，答不了在这里提示原因 */
const answerQuestion = async (questionId: string, action: 'accept' | 'decline', answers: Record<string, string[]>) => {
  const r = await socket.answer(questionId, action, answers)
  if (!r.ok) feedback.error(r.reason === 'error' ? r.error || '' : t(`agent.code.ask.${r.reason}`))
}
const resolve = async (requestId: string, optionId: string) => {
  if (!session.value) return
  try {
    await resolveAgentPermission(session.value.id, requestId, optionId)
  } catch {
    /* shown */
  }
}
// 只停一项（ACP v2）：一个工具调用，或者一个子 Agent
const stopItem = async (sub: string, toolId: string) => {
  if (!session.value) return
  try {
    await stopAgentSessionItem(session.value.id, sub, toolId)
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
const rightTab = ref<RightTab>('git')
const diffFocus = ref('')
const showDiff = (path: string) => {
  rightOn.value = true
  rightTab.value = 'git'
  diffFocus.value = path
}

// 选项（模型清单）由 useAgentModels 在第一次用到时读，配置改了经推送重读，这里不再另读一次
onMounted(() => {
  void loadSidebar()
})
</script>

<style lang="scss">
@use '@/views/agent/styles/agent.scss';
@use './code.scss';
@use './git/git.scss';
</style>
