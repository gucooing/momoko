<!-- 右侧面板：Git（改动、暂存提交、分支、同步、储藏、历史）、文件（按目录懒加载）、任务（Agent 执行的命令和子 Agent）、会话（上下文、用量、会话信息）。 -->
<template>
  <section class="agc-card agc-right" :aria-label="t('agent.code.right.label')">
    <div class="ag-seg" role="tablist" :aria-label="t('agent.code.right.label')">
      <button v-for="tb in tabs" :key="tb.key" type="button" role="tab" :class="{ 'is-on': tab === tb.key }" :aria-selected="tab === tb.key" @click="tab = tb.key">
        {{ tb.label }}
      </button>
    </div>

    <!-- Git：会话目录（或它下面几个仓库）的改动、暂存提交、分支、同步、储藏、历史；看子 Agent 时换成它自己改过的文件 -->
    <div v-if="tab === 'git'" class="agc-right__body">
      <!-- 子 Agent：它自己的工具调用改过的文件 -->
      <template v-if="sub">
        <div class="agc-right__files ag-noscroll">
          <button v-for="d in subDiffs" :key="d.path" type="button" class="agc-fr" :class="{ 'is-on': d.path === subDiff?.path }" @click="subPicked = d.path">
            <span class="ag-mono agc-fr__k" :class="opCls(d)">{{ opLetter(d) }}</span>
            <span class="ag-mono ag-ellipsis agc-flex">{{ d.path }}</span>
          </button>
          <div v-if="!subDiffs.length" class="agc-right__note">{{ t('agent.code.sub.noChanges') }}</div>
        </div>
        <div v-if="subDiff" class="agc-right__patch ag-noscroll">
          <PatchView v-if="subDiff.op" :patch="patchBody(subDiff.patch)" :empty="t('agent.code.diff.noPatch')" />
          <DiffCard v-else :diff="subDiff" :path="subDiff.path" bare @show-diff="emit('openFile', $event)" />
        </div>
      </template>
      <GitPanel v-else />
    </div>

    <!-- 任务 -->
    <div v-else-if="tab === 'task'" class="agc-right__body agc-right__scroll ag-noscroll">
      <div v-if="subPlan.length" class="agc-taskgrp">
        <div class="ag-lbl agc-taskgrp__t">{{ t('agent.code.sub.plan') }} <span class="ag-num">{{ subPlan.length }}</span></div>
        <div v-for="(p, i) in subPlan" :key="i" class="agc-task">
          <div class="agc-task__head">
            <span v-if="p.status === 'in_progress'" class="agc-sd agc-sd--run agc-pulse" />
            <UIcon v-else :name="p.status === 'completed' ? 'i-lucide-check' : 'i-lucide-circle'" :class="p.status === 'completed' ? 'agc-ok' : 'ag-muted'" />
            <span class="agc-task__t">
              <span class="ag-ellipsis agc-task__n">{{ p.content }}</span>
              <span class="ag-muted agc-small">{{ planStatus(p.status) }}</span>
            </span>
          </div>
        </div>
      </div>
      <template v-for="g in taskGroups" :key="g.key">
        <div v-if="g.items.length" class="agc-taskgrp">
          <div class="ag-lbl agc-taskgrp__t">{{ g.title }} <span class="ag-num">{{ g.items.length }}</span></div>
          <div v-for="k in g.items" :key="k.key" class="agc-task">
            <div class="agc-task__row">
              <button type="button" class="agc-task__head" :aria-expanded="k.sub ? undefined : taskOpen.has(k.key)" @click="onTask(k)">
                <span v-if="k.running" class="agc-sd agc-sd--run agc-pulse" />
                <UIcon v-else :name="k.icon" :class="k.iconCls" />
                <span class="agc-task__t">
                  <span class="ag-ellipsis agc-task__n" :class="{ 'ag-mono': !k.sub }">{{ k.title }}</span>
                  <span class="ag-muted agc-small">{{ k.meta }}</span>
                </span>
                <UIcon :name="!k.sub && taskOpen.has(k.key) ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'" class="ag-muted" />
              </button>
              <button
                v-if="k.toolId && k.cancel"
                type="button"
                class="agc-sm agc-task__stop"
                :aria-label="t('agent.code.tool.stop')"
                :title="t('agent.code.tool.stop')"
                :disabled="taskStopping.has(k.toolId)"
                @click="stopTask(k.toolId)"
              >
                <UIcon name="i-lucide-square" />
              </button>
            </div>
            <!-- 还在跑的跟着最新的输出走（往上翻看时不打扰） -->
            <pre
              v-if="!k.sub && taskOpen.has(k.key) && k.output"
              v-follow="k.running"
              class="agc-term agc-term--flat ag-noscroll"
              >{{ k.output }}</pre
            >
          </div>
        </div>
      </template>
      <div v-if="!taskGroups[0]?.items.length && !taskGroups[1]?.items.length" class="agc-right__note">{{ t('agent.code.task.none') }}</div>
    </div>

    <!-- 会话 -->
    <div v-else-if="tab === 'info'" class="agc-right__body agc-right__scroll ag-noscroll">
      <template v-if="sub">
        <div class="agc-info__tags">
          <span class="ag-pill" :class="subPillCls">{{ subStateText }}</span>
          <AgAgentTag :kind="agentKindOf(session.runtimeKind)" :name="session.runtimeName" />
        </div>
        <div v-if="subCtxSize > 0" class="agc-infocard">
          <div class="agc-infocard__row">
            <span class="agc-flex agc-infocard__t">{{ t('agent.code.ctx.title') }}</span>
            <span class="ag-num agc-infocard__v">{{ fmtTokens(subCtxUsed) }} <span class="ag-muted">/ {{ fmtTokens(subCtxSize) }} · {{ subCtxPct }}</span></span>
          </div>
          <div class="ag-meter"><span :style="{ width: subCtxPct }" /></div>
        </div>
        <dl class="agc-kv">
          <template v-for="r in subRows" :key="r.k">
            <dt>{{ r.k }}</dt>
            <dd :class="{ 'ag-mono': r.mono }" :title="r.v">{{ r.v }}</dd>
          </template>
        </dl>
      </template>
      <template v-else>
      <div class="agc-info__tags">
        <span class="ag-pill" :class="pillCls">{{ t(`agent.status.${session.status}`) }}</span>
        <AgAgentTag :kind="agentKindOf(session.runtimeKind)" :name="session.runtimeName" />
        <span v-if="detail?.branch" class="ag-tag ag-mono"><UIcon name="i-lucide-git-branch" />{{ detail.branch }}</span>
      </div>

      <div v-if="ctxSize > 0" class="agc-infocard">
        <div class="agc-infocard__row">
          <span class="agc-flex agc-infocard__t">{{ t('agent.code.ctx.title') }}</span>
          <span class="ag-num agc-infocard__v">{{ fmtTokens(ctxUsed) }} <span class="ag-muted">/ {{ fmtTokens(ctxSize) }} · {{ ctxPct }}</span></span>
        </div>
        <div class="ag-meter"><span :style="{ width: ctxPct }" /></div>
        <div class="ag-num ag-muted agc-small">{{ t('agent.code.ctx.detail', { used: fmtTokens(ctxUsed), size: fmtTokens(ctxSize), left: fmtTokens(Math.max(0, ctxSize - ctxUsed)) }) }}</div>
        <div v-if="hasCommand('compact')" class="agc-infocard__acts">
          <button type="button" class="ag-btn ag-btn--sm" :disabled="busy" @click="emit('command', '/compact')">
            {{ t('agent.code.ctx.compact') }}
          </button>
        </div>
      </div>

      <div v-if="usage" class="agc-infocard">
        <div class="agc-infocard__row">
          <span class="agc-flex agc-infocard__t">{{ t('agent.code.usage.title') }}</span>
          <span class="ag-num agc-infocard__v">{{ fmtTokens(usage.total) }} <span class="ag-muted">· {{ t('agent.code.usage.turns', { n: session.turns }) }}</span></span>
        </div>
        <div class="agc-bar" aria-hidden="true">
          <span v-for="seg in usageRows.filter((r) => r.color)" :key="seg.key" :style="{ width: seg.pct, background: seg.color }" />
        </div>
        <div class="agc-usage">
          <div v-for="r in usageRows" :key="r.key" class="agc-usage__row">
            <span class="agc-usage__dot" :style="{ background: r.color || 'transparent' }" />
            <span class="agc-flex ag-ellipsis" :class="{ 'ag-muted': !r.color }">{{ r.label }}</span>
            <span class="ag-num ag-muted">{{ fmtTokens(r.value) }}</span>
            <span class="ag-num ag-muted agc-usage__pct">{{ r.color ? r.pct : '' }}</span>
          </div>
        </div>
        <div class="agc-usage__hit">
          <span class="agc-flex">{{ t('agent.code.usage.hitRate') }}</span>
          <span class="ag-muted">{{ t('agent.code.usage.hitHint') }}</span>
          <span class="ag-num agc-ok agc-usage__hitv">{{ hitRate(usage) }}</span>
        </div>
        <div v-if="lastUsage" class="agc-usage__last ag-muted">
          <span class="agc-flex">{{ t('agent.code.usage.last') }}</span>
          <span class="ag-num">{{ fmtTokens(lastUsage.total) }}<template v-if="hitRate(lastUsage)"> · {{ t('agent.code.usage.lastHit', { pct: hitRate(lastUsage) }) }}</template></span>
        </div>
      </div>

      <dl class="agc-kv">
        <template v-for="r in infoRows" :key="r.k">
          <dt>{{ r.k }}</dt>
          <dd :class="{ 'ag-mono': r.mono }" :title="r.v">{{ r.v }}</dd>
        </template>
      </dl>

      <div v-if="detail?.mcpNames?.length" class="agc-info__sec">
        <div class="ag-lbl agc-info__lbl">{{ t('agent.code.info.mcp') }}</div>
        <div v-for="m in detail.mcpNames" :key="m" class="agc-info__mcp">
          <UIcon name="i-lucide-plug" class="ag-muted" /><span class="ag-mono ag-ellipsis">{{ m }}</span>
        </div>
      </div>
      <div v-if="detail?.skillNames?.length" class="agc-info__sec">
        <div class="ag-lbl agc-info__lbl">{{ t('agent.code.info.skills') }}</div>
        <div class="agc-info__skills">
          <span v-for="s in detail.skillNames" :key="s" class="ag-tag"><UIcon name="i-lucide-book-open" />{{ s }}</span>
        </div>
      </div>
      </template>
    </div>

    <!-- 文件：第一次打开后一直留着（切走再回来展开的文件夹还在） -->
    <div v-if="filesSeen" v-show="tab === 'files'" class="agc-right__body">
      <FileTree
        ref="treeRef"
        :scope="{ sessionId: session.id }"
        :cwd="session.cwd"
        :empty="t('agent.code.files.empty')"
        :tip="(path) => t('agent.code.files.dragTip', { path })"
        :active="activeFile"
        can-ref
        @open="emit('openFile', $event)"
        @moved="(from, to) => emit('moved', from, to)"
        @deleted="emit('deleted', $event)"
        @reference="emit('reference', $event)"
      />
      <button type="button" class="ag-btn ag-btn--sm" @click="openInFiles"><UIcon name="i-lucide-folder" />{{ t('agent.code.files.openManager') }}</button>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { Directive } from 'vue'
import { useI18n } from 'vue-i18n'
import { getAgentSessionProcess } from '@/api/agent'
import type { AgentProcess, AgentSession, AgentSessionDetail, AgentUsage } from '@/types/v1/agent'
import AgAgentTag from '@/views/agent/components/AgAgentTag.vue'
import { agentKindOf, clockTime, fmtDuration, fmtTokens, sinceTime } from '@/views/agent/utils/format'
import { effortName, modelName } from '@/views/agent/composables/useAgentModels'
import { modeName, providerLabel, sessionTitle } from '@/views/agent/utils/labels'
import DiffCard from './DiffCard.vue'
import FileTree from './FileTree.vue'
import GitPanel from './git/GitPanel.vue'
import { createGitStore, GIT_KEY, type GitChange, type GitCommitTarget } from './git/useGit'
import PatchView from './PatchView.vue'
import type { AgentDiff, AgentItem, AgentState } from './types'

export type RightTab = 'git' | 'files' | 'task' | 'info'

const props = defineProps<{
  session: AgentSession
  /** 打开会话时才给的信息（分支、MCP、技能、复刻来源、Agent 版本） */
  detail: AgentSessionDetail | null
  items: AgentItem[]
  state: AgentState
  /** 转到后台还在跑的工具调用 id（工具调用本身报了完成，命令其实还在跑）：算进行中，能看输出、能停 */
  background: string[]
  contextUsed: number
  contextSize: number
  busy: boolean
  /** 每一轮结束加一：改动和文件跟着刷新 */
  tick: number
  /** 正在看的子 Agent 的会话 id：改动、任务、会话三个标签换成它自己的 */
  sub?: string
  /** 中间正在看的文件标签：文件树里高亮出来 */
  activeFile?: string
  /** 中间正开着的提交（或储藏）的哈希：Git 的历史、储藏里标出来 */
  openedCommit?: string
}>()
const emit = defineEmits<{
  openFile: [path: string]
  openSub: [id: string]
  /** 只停一个命令（ACP v2）：sub 是它所在的子 Agent */
  stop: [sub: string, toolId: string]
  command: [text: string]
  changes: [files: GitChange[]]
  /** Git 里做了会改文件的操作：打开的文件跟着重读 */
  gitChanged: []
  /** 在中间打开一个提交（或储藏） */
  openCommit: [target: GitCommitTarget]
  /** 文件树里改名、挪位置、删除、「添加到对话」 */
  moved: [from: string, to: string]
  deleted: [paths: string[]]
  reference: [path: string]
}>()
const tab = defineModel<RightTab>('tab', { default: 'git' })
/** 要在 Git 里选中的文件（对话里点「完整改动」），相对会话目录 */
const focus = defineModel<string>('focus', { default: '' })
const { t } = useI18n()
const router = useRouter()

// —— Git ——
// 状态放在这里（不放在 Git 标签里）：切到别的标签再回来，提交说明、选中的仓库和视图都还在；标签上的数字也一直有。
const git = createGitStore({
  sessionId: () => props.session.id,
  onChanged: () => {
    // 文件树、打开的文件跟着变（撤销、切换分支、拉取……）
    treeStale = true
    if (tab.value === 'files') {
      treeStale = false
      void reloadTree()
    }
    emit('gitChanged')
  },
  openFile: (path) => emit('openFile', path),
  openCommit: (c) => emit('openCommit', c),
  reference: (path) => emit('reference', path),
})
provide(GIT_KEY, git)
watch(
  () => props.openedCommit,
  (hash) => (git.opened.value = hash || ''),
  { immediate: true },
)
watch(git.changes, (list) => emit('changes', list))

// 对话里点「完整改动」：切到 Git 标签选中那个文件。手上的状态里还没有它（还没读、读的是旧的）就重读一次，读到了再选
let pendingFocus = ''
const revealDiff = (path: string) => {
  pendingFocus = ''
  if (git.reveal(path)) return
  pendingFocus = path
  if (!git.loading.value) void git.load()
}
watch(focus, (path) => {
  if (!path) return
  revealDiff(path)
  focus.value = ''
})
watch(git.repos, () => {
  // 只再试这一次：重读之后还没有，就是它已经没有改动了
  const path = pendingFocus
  pendingFocus = ''
  if (path) git.reveal(path)
})

// —— 文件 ——
const filesSeen = ref(false)
const treeRef = ref<InstanceType<typeof FileTree> | null>(null)
const reloadTree = async () => {
  await treeRef.value?.reload()
}
const openInFiles = () => router.push({ path: '/file', query: { workdir: props.session.cwd } })

// —— 任务 ——
interface Task {
  key: string
  title: string
  meta: string
  running: boolean
  icon: string
  iconCls: string
  output: string
  sub: string
  /** 命令的工具调用 id，cancel 是现在能不能单独停 */
  toolId?: string
  cancel?: boolean
}
const taskOpen = reactive(new Set<string>())
/** 转到后台还在跑的命令：后台列表只有主会话的，看子 Agent 时不用 */
const bgTools = computed(() => new Set(props.sub ? [] : props.background))
const tasks = computed<Task[]>(() => {
  const out: Task[] = []
  for (const item of props.items) {
    if ((item.sub || '') !== (props.sub || '')) continue
    if (item.type === 'tool' && item.tool?.kind === 'execute') {
      const tool = item.tool
      const inBg = bgTools.value.has(tool.id)
      const running = inBg || tool.status === 'pending' || tool.status === 'in_progress'
      const failed = tool.status === 'failed' || tool.status === 'cancelled'
      const meta = inBg
        ? t('agent.code.task.background')
        : running
          ? t('agent.code.task.running')
          : [t(`agent.code.task.${failed ? (tool.status === 'cancelled' ? 'cancelled' : 'failed') : 'done'}`), item.ms ? fmtDuration(item.ms) : ''].filter(Boolean).join(' · ')
      out.push({
        key: `t${item.seq}`,
        title: tool.command || tool.title,
        meta,
        running,
        icon: failed ? 'i-lucide-x' : 'i-lucide-check',
        iconCls: failed ? 'ag-muted' : 'agc-ok',
        output: tool.output || '',
        sub: '',
        toolId: tool.id,
        cancel: running && !!tool.cancel,
      })
    } else if (item.type === 'subagent' && item.subagent) {
      const sa = item.subagent
      const running = sa.state === 'running' || sa.state === 'requires_action'
      const took = sa.endAt && item.at ? fmtDuration(sa.endAt - item.at) : ''
      out.push({
        key: `s${item.seq}`,
        title: sa.title || sa.description || sa.id,
        meta: [t('agent.code.sub.name'), running ? t('agent.code.sub.running') : t('agent.code.sub.done'), took ? t('agent.code.turn.took', { time: took }) : '']
          .filter(Boolean)
          .join(' · '),
        running,
        icon: 'i-lucide-bot',
        iconCls: 'agc-ok',
        output: '',
        sub: sa.id,
      })
    }
  }
  return out.reverse()
})
const taskGroups = computed(() => [
  { key: 'run', title: t('agent.code.task.active'), items: tasks.value.filter((k) => k.running) },
  { key: 'done', title: t('agent.code.task.finished'), items: tasks.value.filter((k) => !k.running) },
])
const onTask = (k: Task) => {
  if (k.sub) {
    emit('openSub', k.sub)
    return
  }
  if (taskOpen.has(k.key)) taskOpen.delete(k.key)
  else taskOpen.add(k.key)
}
/** 单独停一个命令（ACP v2）：和对话里工具卡片上的「停止」一样 */
const taskStopping = reactive(new Set<string>())
const stopTask = (toolId: string) => {
  taskStopping.add(toolId)
  emit('stop', props.sub || '', toolId)
  setTimeout(() => taskStopping.delete(toolId), 3000)
}
/** 输出更新时，原来就停在最底下的话接着滚到底；往上翻着看就不动。value 为 false 时不跟 */
const vFollow: Directive<HTMLElement, boolean> = {
  mounted(el, { value }) {
    if (value) el.scrollTop = el.scrollHeight
  },
  beforeUpdate(el) {
    el.dataset.atEnd = el.scrollHeight - el.scrollTop - el.clientHeight < 24 ? '1' : ''
  },
  updated(el, { value }) {
    if (value && el.dataset.atEnd) el.scrollTop = el.scrollHeight
  },
}

// —— 会话 ——
const pillCls = computed(() => ({ running: 'ag-pill--ok', waiting: 'ag-pill--warn', error: 'ag-pill--bad' })[props.session.status] ?? '')
const ctxUsed = computed(() => Number(props.contextUsed || 0))
const ctxSize = computed(() => Number(props.contextSize || 0))
const ctxPct = computed(() => (ctxSize.value > 0 ? `${Math.min(100, Math.round((ctxUsed.value / ctxSize.value) * 100))}%` : ''))
// 压缩上下文是 Agent 自己的 /compact 命令：Agent 提供了才有这个按钮
const hasCommand = (name: string) => (props.state.commands ?? []).some((c) => c.name === name)

const num = (u: AgentUsage | undefined, k: keyof AgentUsage) => Number(u?.[k] || 0)
const usage = computed(() => (props.session.usage && num(props.session.usage, 'total') > 0 ? props.session.usage : null))
const lastUsage = computed(() => (props.session.lastUsage && num(props.session.lastUsage, 'total') > 0 ? props.session.lastUsage : null))
/** 输入里有多少走了缓存：缓存命中 ÷（缓存命中 + 没命中的输入 + 缓存写入） */
const hitRate = (u: AgentUsage) => {
  const read = num(u, 'cacheRead')
  const all = read + num(u, 'input') + num(u, 'cacheWrite')
  return all > 0 && read > 0 ? `${((read / all) * 100).toFixed(1)}%` : ''
}
const usageRows = computed(() => {
  const u = usage.value
  if (!u) return []
  const read = num(u, 'cacheRead')
  const miss = num(u, 'input') + num(u, 'cacheWrite')
  const output = num(u, 'output')
  const sum = read + miss + output || 1
  const pct = (n: number) => `${((n / sum) * 100).toFixed(1)}%`
  return [
    { key: 'hit', label: t('agent.code.usage.inputHit'), value: read, pct: pct(read), color: 'var(--m-ok)' },
    { key: 'miss', label: t('agent.code.usage.inputMiss'), value: miss, pct: pct(miss), color: 'var(--m-p)' },
    { key: 'out', label: t('agent.code.usage.outputRow'), value: output, pct: pct(output), color: 'var(--m-a2)' },
    { key: 'think', label: t('agent.code.usage.thoughtRow'), value: num(u, 'thought'), pct: '', color: '' },
    { key: 'write', label: t('agent.code.usage.cacheWrite'), value: num(u, 'cacheWrite'), pct: '', color: '' },
  ].filter((r) => r.color || r.value > 0)
})

const infoRows = computed(() => {
  const s = props.session
  const provider = providerLabel({ kind: s.providerKind, vendor: s.providerVendor, name: s.providerName })
  const rows: { k: string; v: string; mono?: boolean }[] = [
    { k: t('agent.code.info.title'), v: sessionTitle(s.title) },
  ]
  const d = props.detail
  if (s.parentId) rows.push({ k: t('agent.code.info.forkOf'), v: sessionTitle(d?.parentTitle) })
  rows.push({
    k: t('agent.code.info.model'),
    v: [provider, modelName(s.runtimeId, s.providerId, s.model)].filter(Boolean).join(' · ') || '—',
  })
  rows.push({
    k: t('agent.code.info.effort'),
    v: effortName(s.runtimeId, s.providerId, s.model, s.effort) || t('agent.common.notAdjustable'),
  })
  if (s.mode) rows.push({ k: t('agent.code.info.mode'), v: modeName(s.mode, s.modeName) })
  rows.push({ k: t('agent.code.info.cwd'), v: s.cwd, mono: true })
  if (d?.branch) rows.push({ k: t('agent.code.info.branch'), v: d.branch, mono: true })
  if (s.worktree) rows.push({ k: t('agent.code.info.worktree'), v: s.worktree, mono: true })
  for (const dir of s.extraDirs ?? []) rows.push({ k: t('agent.code.info.extraDir'), v: dir, mono: true })
  if (d?.agentName) rows.push({ k: t('agent.code.info.adapter'), v: ['ACP', [d.agentName.split('/').pop(), d.agentVersion].filter(Boolean).join(' ')].join(' · ') })
  const p = proc.value
  if (p?.pid) rows.push({ k: t('agent.code.info.process'), v: t('agent.code.info.processValue', { pid: p.pid, time: sinceTime(p.startedAt) }) })
  if (s.acpSessionId) rows.push({ k: t('agent.code.info.sessionId'), v: s.acpSessionId, mono: true })
  return rows
})

const tabs = computed(() => [
  {
    key: 'git' as const,
    label: props.sub
      ? t('agent.code.right.diffN', { n: subDiffs.value.length })
      : git.loaded.value && git.repos.value.length
        ? t('agent.code.right.gitN', { n: git.total.value })
        : t('agent.code.right.git'),
  },
  { key: 'files' as const, label: t('agent.code.right.files') },
  { key: 'task' as const, label: t('agent.code.right.taskN', { n: taskGroups.value[0]?.items.length ?? 0 }) },
  { key: 'info' as const, label: t('agent.code.right.info') },
])

// —— 子 Agent ——
const subItem = computed(() =>
  props.sub ? props.items.find((i) => i.type === 'subagent' && i.subagent?.id === props.sub) : undefined,
)
/** 它自己的工具调用改过的文件，同一个文件留最后一次 */
const subPicked = ref('')
const subDiff = computed(() => subDiffs.value.find((d) => d.path === subPicked.value) ?? subDiffs.value[0])
const subDiffs = computed<AgentDiff[]>(() => {
  if (!props.sub) return []
  const byPath = new Map<string, AgentDiff>()
  for (const item of props.items) {
    if (item.sub !== props.sub) continue
    for (const d of item.tool?.diffs ?? []) byPath.set(d.path, d)
  }
  return [...byPath.values()]
})
const OP_LETTER: Record<string, string> = { add: 'A', delete: 'D', modify: 'M', move: 'R', copy: 'C' }
const opLetter = (d: AgentDiff) => (d.op ? OP_LETTER[d.op] || 'M' : d.oldText == null ? 'A' : 'M')
const opCls = (d: AgentDiff) => {
  const l = opLetter(d)
  return l === 'A' ? 'agc-ok' : l === 'D' ? 'agc-bad' : 'agc-warn'
}
/** git patch 从第一个 @@ 段开始的部分 */
const patchBody = (patch?: string) => {
  const p = patch || ''
  if (p.startsWith('@@')) return p
  const at = p.indexOf('\n@@')
  return at >= 0 ? p.slice(at + 1) : ''
}
const PLAN_STATUS = ['pending', 'in_progress', 'completed']
const planStatus = (status: string) => (PLAN_STATUS.includes(status) ? t(`agent.code.plan.${status}`) : status)
const subPlan = computed(() => (props.sub ? (props.state.subs?.[props.sub]?.plan ?? []) : []))
const subCtxUsed = computed(() => Number(props.sub ? props.state.subs?.[props.sub]?.contextUsed || 0 : 0))
const subCtxSize = computed(() => Number(props.sub ? props.state.subs?.[props.sub]?.contextSize || 0 : 0))
const subCtxPct = computed(() =>
  subCtxSize.value > 0 ? `${Math.min(100, Math.round((subCtxUsed.value / subCtxSize.value) * 100))}%` : '',
)
const subStateText = computed(() => {
  const state = subItem.value?.subagent?.state
  if (state === 'running') return t('agent.code.sub.running')
  if (state === 'requires_action') return t('agent.code.sub.waiting')
  return t('agent.code.sub.done')
})
const subPillCls = computed(
  () => ({ running: 'ag-pill--ok', requires_action: 'ag-pill--warn' })[subItem.value?.subagent?.state || ''] ?? '',
)
const subRows = computed(() => {
  const item = subItem.value
  const sa = item?.subagent
  if (!item || !sa) return []
  const rows: { k: string; v: string; mono?: boolean }[] = [
    { k: t('agent.code.info.title'), v: sa.title || sa.description || sa.id },
  ]
  if (sa.title && sa.description) rows.push({ k: t('agent.code.sub.description'), v: sa.description })
  rows.push({ k: t('agent.code.info.cwd'), v: sa.cwd || props.session.cwd, mono: true })
  rows.push({ k: t('agent.code.sub.started'), v: clockTime(item.at) })
  if (sa.endAt) rows.push({ k: t('agent.code.sub.took'), v: fmtDuration(sa.endAt - item.at) })
  const tools = props.items.filter((i) => i.sub === props.sub && i.type === 'tool').length
  rows.push({ k: t('agent.code.sub.tools'), v: String(tools) })
  rows.push({ k: t('agent.code.info.sessionId'), v: sa.id, mono: true })
  return rows
})

// —— 进程 ——
// 进程号和已运行时间只在「会话」标签里显示：切到这个标签、或者会话状态变了时读一次（读内存要遍历进程树）。
const proc = ref<AgentProcess | null>(null)
const loadProcess = async () => {
  try {
    const { data } = await getAgentSessionProcess(props.session.id)
    proc.value = data.process ?? null
  } catch {
    proc.value = null
  }
}

// —— 刷新时机 ——
// Git 和文件都要跑 git，只在对应的标签开着时读：别的标签开着时一轮结束只记下「过期了」，切过去再读。
let gitStale = true
let treeStale = true
const refresh = () => {
  if (tab.value === 'git' && gitStale) {
    gitStale = false
    void git.load()
  }
  if (tab.value === 'files') {
    // 第一次打开时文件树自己会读；之后一轮结束过才重读
    if (!filesSeen.value) filesSeen.value = true
    else if (treeStale) void reloadTree()
    treeStale = false
  }
  if (tab.value === 'info') void loadProcess()
}
watch(
  () => props.session.id,
  () => {
    git.reset()
    pendingFocus = ''
    taskOpen.clear()
    proc.value = null
    // 换了会话文件树自己会重读
    gitStale = true
    treeStale = false
    refresh()
  },
  { immediate: true },
)
watch(tab, refresh)
watch(
  () => props.tick,
  () => {
    gitStale = treeStale = true
    refresh()
  },
)
watch(
  () => props.session.status,
  () => {
    if (tab.value === 'info') void loadProcess()
  },
)
</script>
