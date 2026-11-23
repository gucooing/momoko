<!-- 右侧面板：改动（相对当前分支的未提交改动）、文件（按目录懒加载）、任务（Agent 执行的命令和子 Agent）、会话（上下文、用量、会话信息）。 -->
<template>
  <section class="agc-card agc-right" :aria-label="t('agent.code.right.label')">
    <div class="ag-seg" role="tablist" :aria-label="t('agent.code.right.label')">
      <button v-for="tb in tabs" :key="tb.key" type="button" role="tab" :class="{ 'is-on': tab === tb.key }" :aria-selected="tab === tb.key" @click="tab = tb.key">
        {{ tb.label }}
      </button>
    </div>

    <!-- 改动：点文件在下面看它的改动；会话目录下面有几个仓库时按仓库分组，点仓库那一行展开 / 收起它的文件列表 -->
    <div v-if="tab === 'diff'" class="agc-right__body">
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
      <template v-else-if="changes && !changes.isGit">
        <div class="agc-right__note">{{ t('agent.code.diff.noGit') }}</div>
      </template>
      <template v-else>
        <div class="agc-right__bar">
          <span class="agc-flex ag-muted agc-small">
            <span v-if="nested">{{ t('agent.code.diff.headRepos', { n: changes?.repos?.length ?? 0 }) }}</span>
            <i18n-t v-else scope="global" keypath="agent.code.diff.head" tag="span">
              <template #branch><span class="ag-mono">{{ changes?.branch || 'HEAD' }}</span></template>
            </i18n-t>
            · <span class="agc-plus ag-num">+{{ totalAdd }}</span> <span class="agc-minus ag-num">−{{ totalDel }}</span>
          </span>
          <button
            v-if="nested"
            type="button"
            class="ag-ib ag-ib--sm"
            :aria-label="allGroupsOpen ? t('agent.code.diff.collapseAll') : t('agent.code.diff.expandAll')"
            :title="allGroupsOpen ? t('agent.code.diff.collapseAll') : t('agent.code.diff.expandAll')"
            :disabled="!files.length"
            @click="setAllGroups(!allGroupsOpen)"
          >
            <UIcon :name="allGroupsOpen ? 'i-lucide-chevrons-down-up' : 'i-lucide-chevrons-up-down'" />
          </button>
          <button type="button" class="ag-ib ag-ib--sm" :aria-label="t('agent.code.diff.copy')" :title="t('agent.code.diff.copy')" :disabled="!files.length" @click="copyPatch">
            <UIcon name="i-lucide-copy" />
          </button>
          <button type="button" class="ag-ib ag-ib--sm" :aria-label="t('agent.code.diff.refresh')" :title="t('agent.code.diff.refresh')" @click="loadChanges(true)">
            <UIcon name="i-lucide-refresh-cw" :class="{ 'ag-spin': changesLoading }" />
          </button>
        </div>
        <div ref="diffList" class="agc-right__files ag-noscroll" :class="{ 'is-full': !current }">
          <template v-for="g in fileGroups" :key="g.repo">
            <!-- 会话目录下面放着几个仓库：按仓库分组，组里的路径相对仓库；点组头展开 / 收起这个仓库的文件 -->
            <button
              v-if="nested"
              type="button"
              class="agc-fgrp"
              :aria-expanded="openGroups.has(g.repo)"
              :title="openGroups.has(g.repo) ? t('agent.code.diff.collapseRepo', { repo: g.repo }) : t('agent.code.diff.expandRepo', { repo: g.repo })"
              @click="toggleGroup(g.repo)"
            >
              <UIcon :name="openGroups.has(g.repo) ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'" class="ag-muted agc-fr__chev" />
              <UIcon name="i-lucide-git-branch" class="ag-muted" />
              <span class="ag-mono ag-ellipsis agc-flex">{{ g.repo }}</span>
              <span v-if="g.branch" class="ag-tag ag-mono agc-fgrp__br">{{ g.branch }}</span>
              <span class="ag-num ag-muted agc-small">{{ g.files.length }}</span>
              <span class="agc-plus ag-num">+{{ g.added }}</span>
              <span class="agc-minus ag-num">−{{ g.deleted }}</span>
            </button>
            <template v-if="!nested || openGroups.has(g.repo)">
              <button
                v-for="f in g.files"
                :key="f.path"
                type="button"
                class="agc-fr"
                :class="{ 'is-on': f.path === current?.path, 'agc-fr--in': nested }"
                :data-path="f.path"
                :title="f.path"
                @click="picked = f.path"
              >
                <span class="ag-mono agc-fr__k" :class="f.kind === 'A' ? 'agc-ok' : f.kind === 'D' ? 'agc-bad' : 'agc-warn'">{{ f.kind }}</span>
                <span class="ag-mono ag-ellipsis agc-flex">{{ nested ? inRepo(f) : f.path }}</span>
                <span class="agc-plus ag-num">+{{ f.added }}</span>
                <span class="agc-minus ag-num">−{{ f.deleted }}</span>
              </button>
            </template>
          </template>
          <div v-if="changes && !files.length" class="agc-right__note">{{ t('agent.code.diff.none') }}</div>
        </div>
        <div v-if="current" class="agc-right__patch ag-noscroll">
          <PatchView :patch="current.patch" :empty="t('agent.code.diff.noPatch')" />
        </div>
        <div v-if="current" class="agc-right__acts">
          <button type="button" class="ag-btn ag-btn--sm agc-flex" :disabled="current.kind === 'D'" @click="emit('openFile', current.path)">
            <UIcon name="i-lucide-file" />{{ t('agent.code.diff.openTab') }}
          </button>
          <button type="button" class="ag-btn ag-btn--o ag-btn--sm" @click="revertOpen = true">
            <UIcon name="i-lucide-undo-2" />{{ t('agent.code.diff.revert') }}
          </button>
        </div>
      </template>
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
            <button type="button" class="agc-task__head" :aria-expanded="k.sub ? undefined : taskOpen.has(k.key)" @click="onTask(k)">
              <span v-if="k.running" class="agc-sd agc-sd--run agc-pulse" />
              <UIcon v-else :name="k.icon" :class="k.iconCls" />
              <span class="agc-task__t">
                <span class="ag-ellipsis agc-task__n" :class="{ 'ag-mono': !k.sub }">{{ k.title }}</span>
                <span class="ag-muted agc-small">{{ k.meta }}</span>
              </span>
              <UIcon :name="!k.sub && taskOpen.has(k.key) ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'" class="ag-muted" />
            </button>
            <pre v-if="!k.sub && taskOpen.has(k.key) && k.output" class="agc-term agc-term--flat ag-noscroll">{{ k.output }}</pre>
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

    <AgConfirm
      v-model="revertOpen"
      :title="t('agent.code.diff.revertTitle')"
      :message="t('agent.code.diff.revertMsg', { path: current?.path || '' })"
      :confirm-text="t('agent.code.diff.revert')"
      danger
      :on-confirm="doRevert"
    />
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { getAgentSessionChanges, getAgentSessionProcess, revertAgentSessionFile } from '@/api/agent'
import type {
  AgentChangedFile,
  AgentProcess,
  AgentSession,
  AgentSessionDetail,
  AgentUsage,
  GetAgentSessionChangesResponse,
} from '@/types/v1/agent'
import AgAgentTag from '@/views/agent/components/AgAgentTag.vue'
import AgConfirm from '@/views/agent/components/AgConfirm.vue'
import { agentKindOf, clockTime, fmtDuration, fmtTokens, sinceTime } from '@/views/agent/utils/format'
import { modeName, providerLabel, sessionTitle } from '@/views/agent/utils/labels'
import DiffCard from './DiffCard.vue'
import FileTree from './FileTree.vue'
import PatchView from './PatchView.vue'
import type { AgentDiff, AgentItem, AgentState } from './types'

export type RightTab = 'diff' | 'files' | 'task' | 'info'

const props = defineProps<{
  session: AgentSession
  /** 打开会话时才给的信息（分支、MCP、技能、复刻来源、Agent 版本） */
  detail: AgentSessionDetail | null
  items: AgentItem[]
  state: AgentState
  contextUsed: number
  contextSize: number
  busy: boolean
  /** 每一轮结束加一：改动和文件跟着刷新 */
  tick: number
  /** 正在看的子 Agent 的会话 id：改动、任务、会话三个标签换成它自己的 */
  sub?: string
  /** 中间正在看的文件标签：文件树里高亮出来 */
  activeFile?: string
}>()
const emit = defineEmits<{
  openFile: [path: string]
  openSub: [id: string]
  command: [text: string]
  changes: [files: AgentChangedFile[]]
  /** 文件树里改名、挪位置、删除、「添加到对话」 */
  moved: [from: string, to: string]
  deleted: [paths: string[]]
  reference: [path: string]
}>()
const tab = defineModel<RightTab>('tab', { default: 'diff' })
/** 要在改动里选中的文件（对话里点「完整改动」） */
const focus = defineModel<string>('focus', { default: '' })
const { t } = useI18n()
const router = useRouter()

// —— 改动 ——
const changes = ref<GetAgentSessionChangesResponse | null>(null)
const changesLoading = ref(false)
const picked = ref('')
const revertOpen = ref(false)
const diffList = ref<HTMLElement | null>(null)
/** 展开着文件列表的仓库（按仓库分组时）；默认都收着 */
const openGroups = reactive(new Set<string>())
const toggleGroup = (repo: string) => (openGroups.has(repo) ? openGroups.delete(repo) : openGroups.add(repo))
const files = computed(() => changes.value?.files ?? [])
/** 会话目录本身不是仓库，改动来自下面的几个仓库 */
const nested = computed(() => !!changes.value?.repos?.length)
/** 按仓库分组（会话目录本身是仓库时只有一组）；没有改动的仓库不列 */
const fileGroups = computed(() => {
  const branches = new Map((changes.value?.repos ?? []).map((r) => [r.path, r.branch]))
  const groups = new Map<string, { repo: string; branch: string; files: AgentChangedFile[]; added: number; deleted: number }>()
  for (const f of files.value) {
    const repo = f.repo || ''
    let g = groups.get(repo)
    if (!g) {
      g = { repo, branch: branches.get(repo) || '', files: [], added: 0, deleted: 0 }
      groups.set(repo, g)
    }
    g.files.push(f)
    g.added += Number(f.added || 0)
    g.deleted += Number(f.deleted || 0)
  }
  return [...groups.values()]
})
/** 文件在它的仓库里的路径 */
const inRepo = (f: AgentChangedFile) => (f.repo && f.path.startsWith(`${f.repo}/`) ? f.path.slice(f.repo.length + 1) : f.path)
/** 选中的文件；按仓库分组时不默认选第一个（它可能在收着的组里） */
const current = computed(() => files.value.find((f) => f.path === picked.value) ?? (nested.value ? undefined : files.value[0]))
const allGroupsOpen = computed(() => fileGroups.value.length > 0 && fileGroups.value.every((g) => openGroups.has(g.repo)))
const setAllGroups = (open: boolean) => {
  openGroups.clear()
  if (open) for (const g of fileGroups.value) openGroups.add(g.repo)
}
const totalAdd = computed(() => files.value.reduce((n, f) => n + Number(f.added || 0), 0))
const totalDel = computed(() => files.value.reduce((n, f) => n + Number(f.deleted || 0), 0))

let changesSeq = 0
const loadChanges = async (manual = false) => {
  const id = ++changesSeq
  changesLoading.value = true
  try {
    const { data } = await getAgentSessionChanges(props.session.id)
    if (id !== changesSeq) return
    changes.value = data
    emit('changes', data.files ?? [])
    if (pendingFocus) revealDiff(pendingFocus)
    if (manual) feedback.success(t('agent.code.diff.refreshed'))
  } catch {
    /* shown */
  } finally {
    if (id === changesSeq) changesLoading.value = false
  }
}

const copyPatch = async () => {
  const text = files.value.map((f) => `--- a/${f.path}\n+++ b/${f.path}\n${f.patch}`).join('\n')
  try {
    await navigator.clipboard.writeText(text)
    feedback.success(t('agent.code.diff.copied'))
  } catch {
    feedback.error(t('agent.code.diff.copyFailed'))
  }
}

const doRevert = async () => {
  const path = current.value?.path
  if (!path) return
  await revertAgentSessionFile(props.session.id, path)
  feedback.success(t('agent.code.diff.reverted', { path }))
  picked.value = ''
  await loadChanges()
  void reloadTree()
}

// 对话里点「完整改动」：选中那个文件（展开它所在的仓库）并滚过去；改动还没读到就等读到了再选
let pendingFocus = ''
const revealDiff = (path: string) => {
  pendingFocus = ''
  const f = files.value.find((x) => x.path === path)
  if (!f) {
    if (!changes.value || changesLoading.value) pendingFocus = path
    return
  }
  picked.value = path
  openGroups.add(f.repo || '')
  void nextTick(() => diffList.value?.querySelector(`[data-path="${CSS.escape(path)}"]`)?.scrollIntoView({ block: 'nearest' }))
}
watch(focus, (path) => {
  if (!path) return
  revealDiff(path)
  focus.value = ''
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
}
const taskOpen = reactive(new Set<string>())
const tasks = computed<Task[]>(() => {
  const out: Task[] = []
  for (const item of props.items) {
    if ((item.sub || '') !== (props.sub || '')) continue
    if (item.type === 'tool' && item.tool?.kind === 'execute') {
      const tool = item.tool
      const running = tool.status === 'pending' || tool.status === 'in_progress'
      const failed = tool.status === 'failed' || tool.status === 'cancelled'
      const meta = running
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
  rows.push({ k: t('agent.code.info.model'), v: [provider, s.model].filter(Boolean).join(' · ') || '—' })
  rows.push({ k: t('agent.code.info.effort'), v: s.effort || t('agent.common.notAdjustable') })
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
    key: 'diff' as const,
    label: props.sub
      ? t('agent.code.right.diffN', { n: subDiffs.value.length })
      : changes.value
        ? t('agent.code.right.diffN', { n: files.value.length })
        : t('agent.code.right.diff'),
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
// 改动和文件都要跑 git，只在对应的标签开着时读：别的标签开着时一轮结束只记下「过期了」，切过去再读。
let changesStale = true
let treeStale = true
const refresh = () => {
  if (tab.value === 'diff' && changesStale) {
    changesStale = false
    void loadChanges()
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
    changes.value = null
    picked.value = ''
    openGroups.clear()
    pendingFocus = ''
    taskOpen.clear()
    proc.value = null
    // 换了会话文件树自己会重读
    changesStale = true
    treeStale = false
    refresh()
  },
  { immediate: true },
)
watch(tab, refresh)
watch(
  () => props.tick,
  () => {
    changesStale = treeStale = true
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
