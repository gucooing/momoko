<!-- 会话栏：我的未归档会话按工作区分组，复刻出来的会话缩进在原会话下面；每组最多先显示 4 个。
     每个会话的菜单：重命名、复刻对话、在新工作树里复刻、移到外层、导出记录、归档、删除。 -->
<template>
  <section class="agc-card agc-side" :aria-label="t('agent.code.side.label')">
    <button type="button" class="ag-btn ag-btn--p agc-side__new" @click="emit('newSession', '')">
      <UIcon name="i-lucide-plus" />{{ t('agent.common.newSession') }}
    </button>
    <label class="ag-inp agc-side__search">
      <UIcon name="i-lucide-search" />
      <input v-model="keyword" type="search" :placeholder="t('agent.code.side.search')" :aria-label="t('agent.code.side.search')" />
    </label>

    <div ref="listEl" class="agc-side__list ag-noscroll">
      <div v-for="g in groups" :key="g.key" class="agc-grp">
        <div class="agc-wr">
          <button type="button" class="agc-wb" :aria-expanded="!collapsed.has(g.key)" @click="toggleSet(collapsed, g.key)">
            <UIcon :name="g.workspace ? 'i-lucide-folder' : 'i-lucide-message-square'" />
            <span class="ag-ellipsis agc-wb__t">{{ g.title }}</span>
            <span class="ag-mono ag-ellipsis agc-wb__br">{{ g.workspace?.branch }}</span>
            <UIcon :name="collapsed.has(g.key) ? 'i-lucide-chevron-right' : 'i-lucide-chevron-down'" class="agc-wb__chev" />
          </button>
          <button
            v-if="g.workspace"
            type="button"
            class="agc-sm"
            :class="{ 'is-on': info === g.key }"
            :aria-label="t('agent.code.side.wsInfo', { name: g.title })"
            :title="t('agent.code.side.wsInfoTip')"
            aria-haspopup="dialog"
            :aria-expanded="info === g.key"
            @click.stop="info = info === g.key ? '' : g.key"
          >
            <UIcon name="i-lucide-ellipsis" />
          </button>
          <button type="button" class="agc-sm" :aria-label="t('agent.code.side.newIn', { name: g.title })" :title="t('agent.common.newSession')" @click="emit('newSession', g.workspace?.id || '')">
            <UIcon name="i-lucide-plus" />
          </button>
          <div v-if="info === g.key && g.workspace" class="agc-wpop ag-menu" role="dialog" :aria-label="g.title">
            <div class="agc-wpop__row agc-wpop__name">
              <UIcon name="i-lucide-folder" /><span class="ag-ellipsis agc-flex">{{ g.title }}</span>
              <span v-if="g.workspace.branch" class="ag-tag ag-mono">{{ g.workspace.branch }}</span>
            </div>
            <div class="agc-wpop__row ag-muted"><UIcon name="i-lucide-message-square" />{{ t('agent.code.side.count', { n: g.all }) }}</div>
            <div class="agc-wpop__row ag-muted ag-mono"><UIcon name="i-lucide-folder" /><span class="ag-ellipsis">{{ g.workspace.path }}</span></div>
            <template v-if="canManage">
              <div class="ag-mi-sep" />
              <RouterLink class="ag-mi" to="/agent/manage?tab=workspaces"><UIcon name="i-lucide-settings" />{{ t('agent.code.side.editWs') }}</RouterLink>
            </template>
          </div>
        </div>

        <template v-if="!collapsed.has(g.key)">
          <div v-for="r in g.rows" :key="r.s.id" class="agc-sr" :class="{ 'is-on': r.s.id === currentId }">
            <button
              type="button"
              class="agc-ss"
              :style="{ paddingLeft: `${28 + r.depth * 14}px` }"
              :aria-current="r.s.id === currentId ? 'true' : undefined"
              :title="tipOf(r.s)"
              @click="emit('open', r.s.id)"
            >
              <UIcon v-if="r.fork" name="i-lucide-git-branch" class="agc-ss__fork" :title="t('agent.code.side.isFork')" />
              <span class="agc-sd" :class="dotCls(r.s.status)" />
              <span class="ag-ellipsis agc-ss__t">{{ sessionTitle(r.s.title) }}</span>
              <UIcon v-if="r.s.worktree" name="i-lucide-layers" class="agc-ss__wt" :title="t('agent.code.side.worktree', { name: r.s.worktree })" />
              <AgAgentTag dot :kind="agentKindOf(r.s.runtimeKind)" :name="r.s.runtimeName" />
            </button>
            <button
              type="button"
              class="agc-sm"
              :class="{ 'is-on': menu === r.s.id }"
              :aria-label="t('agent.code.side.actions', { title: sessionTitle(r.s.title) })"
              aria-haspopup="menu"
              :aria-expanded="menu === r.s.id"
              @click.stop="menu = menu === r.s.id ? '' : r.s.id"
            >
              <UIcon name="i-lucide-ellipsis" />
            </button>
            <div v-if="menu === r.s.id" class="agc-smenu ag-menu" role="menu" :aria-label="t('agent.code.side.actions', { title: sessionTitle(r.s.title) })">
              <template v-for="a in actionsOf(r.s)" :key="a.key">
                <div v-if="a.sep" class="ag-mi-sep" />
                <button type="button" class="ag-mi" :class="{ 'is-bad': a.key === 'delete' }" role="menuitem" @click="run(a.key, r.s)">
                  <UIcon :name="a.icon" />{{ a.label }}
                </button>
              </template>
            </div>
          </div>
          <button v-if="g.rest > 0" type="button" class="agc-sx" @click="more.add(g.key)">{{ t('agent.code.side.more', { n: g.rest }) }}</button>
        </template>
      </div>
      <div v-if="loaded && !groups.length" class="agc-side__empty">{{ keyword ? t('agent.code.side.noMatch') : t('agent.code.side.none') }}</div>
    </div>

    <RouterLink v-if="canManage" class="ag-btn ag-btn--sm agc-side__all" to="/agent/manage?tab=sessions">{{ t('agent.code.side.all') }}</RouterLink>

    <AgDialog v-model="renameOpen" :title="t('agent.code.side.renameTitle')" :width="420">
      <label class="ag-field">
        <span class="ag-fl">{{ t('agent.code.side.renameLabel') }}</span>
        <span class="ag-fld"><input v-model="renameValue" type="text" @keydown.enter="doRename" /></span>
      </label>
      <div class="ag-confirm__acts">
        <button type="button" class="ag-btn" @click="renameOpen = false">{{ t('agent.common.cancel') }}</button>
        <button type="button" class="ag-btn ag-btn--p" :disabled="!renameValue.trim()" @click="doRename">{{ t('agent.common.save') }}</button>
      </div>
    </AgDialog>

    <AgConfirm
      v-model="deleteOpen"
      :title="t('agent.code.side.deleteTitle')"
      :message="t('agent.code.side.deleteMsg', { title: sessionTitle(target?.title) })"
      :confirm-text="t('agent.common.delete')"
      danger
      :on-confirm="doDelete"
    >
      <label v-if="kidCount > 0" class="agc-check">
        <input v-model="withForks" type="checkbox" class="ag-cb" />
        <span>{{ t('agent.code.side.deleteKids', { n: kidCount }) }}</span>
      </label>
    </AgConfirm>
  </section>
</template>

<script setup lang="ts">
import { onClickOutside } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import {
  archiveAgentSessions,
  deleteAgentSessions,
  detachAgentSession,
  exportAgentSessions,
  forkAgentSession,
  renameAgentSession,
} from '@/api/agent'
import { useMenuStore } from '@/stores/menu'
import type { AgentRuntimeOptions, AgentSession, AgentSidebarWorkspace } from '@/types/v1/agent'
import AgAgentTag from '@/views/agent/components/AgAgentTag.vue'
import AgConfirm from '@/views/agent/components/AgConfirm.vue'
import AgDialog from '@/views/agent/components/AgDialog.vue'
import { agentKindOf, relTime } from '@/views/agent/utils/format'
import { downloadText, exportMarkdown, safeFileName, sessionTitle } from '@/views/agent/utils/labels'

const props = defineProps<{
  sessions: AgentSession[]
  workspaces: AgentSidebarWorkspace[]
  runtimes: AgentRuntimeOptions[]
  currentId: string
  loaded: boolean
}>()
const emit = defineEmits<{
  open: [id: string]
  newSession: [workspaceId: string]
  /** 会话有增减或改动，重新拉会话栏 */
  changed: []
  /** 当前会话被删除了 */
  removed: [ids: string[]]
}>()
const keyword = defineModel<string>('keyword', { default: '' })
const { t } = useI18n()
const menuStore = useMenuStore()
// 能进 Agent 管理页（自己的工作区、全部会话在那里管）
const canManage = computed(() => menuStore.hasMenuPath('/agent/manage'))

const FIRST = 4
const collapsed = reactive(new Set<string>())
const more = reactive(new Set<string>())
const info = ref('')
const menu = ref('')
const listEl = ref<HTMLElement | null>(null)
onClickOutside(listEl, () => {
  info.value = ''
  menu.value = ''
})
const toggleSet = (set: Set<string>, key: string) => (set.has(key) ? set.delete(key) : set.add(key))

// —— 分组和缩进 ——
interface Row {
  s: AgentSession
  depth: number
  fork: boolean
}
const nest = (list: AgentSession[]): Row[] => {
  const ids = new Set(list.map((s) => s.id))
  const kids = new Map<string, AgentSession[]>()
  for (const s of list) {
    if (s.parentId && !s.detached && ids.has(s.parentId)) {
      const arr = kids.get(s.parentId) ?? []
      arr.push(s)
      kids.set(s.parentId, arr)
    }
  }
  const out: Row[] = []
  const add = (s: AgentSession, depth: number) => {
    out.push({ s, depth, fork: !!s.parentId })
    for (const k of kids.get(s.id) ?? []) add(k, depth + 1)
  }
  for (const s of list) {
    if (!s.parentId || s.detached || !ids.has(s.parentId)) add(s, 0)
  }
  return out
}

const groups = computed(() => {
  const byWs = new Map<string, AgentSession[]>()
  for (const s of props.sessions) {
    const key = props.workspaces.some((w) => w.id === s.workspaceId) ? s.workspaceId : ''
    const arr = byWs.get(key) ?? []
    arr.push(s)
    byWs.set(key, arr)
  }
  const make = (key: string, title: string, workspace?: AgentSidebarWorkspace) => {
    const all = nest(byWs.get(key) ?? [])
    const full = more.has(key || '-') || all.length <= FIRST
    return {
      key: key || '-',
      title,
      workspace,
      all: all.length,
      rows: full ? all : all.slice(0, FIRST),
      rest: full ? 0 : all.length - FIRST,
      active: Math.max(0, ...(byWs.get(key) ?? []).map((s) => new Date(s.activeAt ?? 0).getTime())),
    }
  }
  const list = props.workspaces
    .map((w) => make(w.id, w.name, w))
    .filter((g) => g.all > 0 || !keyword.value.trim())
    .sort((a, b) => b.active - a.active)
  const loose = make('', t('agent.common.noWorkspace'))
  if (loose.all > 0) list.push(loose)
  return list
})

const DOT: Record<string, string> = { running: 'agc-sd--run agc-pulse', waiting: 'agc-sd--wait', error: 'agc-sd--bad' }
const dotCls = (status: string) => DOT[status] ?? ''
const tipOf = (s: AgentSession) =>
  [s.runtimeName, s.status === 'running' || s.status === 'waiting' ? t(`agent.status.${s.status}`) : relTime(s.activeAt)].filter(Boolean).join(' · ')

// —— 菜单 ——
type Act = 'rename' | 'fork' | 'forkwt' | 'detach' | 'export' | 'archive' | 'delete'
const actionsOf = (s: AgentSession) => {
  const rt = props.runtimes.find((r) => r.runtimeId === s.runtimeId)
  const canFork = !!rt?.caps?.fork && !!s.acpSessionId
  const ws = props.workspaces.find((w) => w.id === s.workspaceId)
  const list: { key: Act; icon: string; label: string; sep?: boolean }[] = [{ key: 'rename', icon: 'i-lucide-pencil', label: t('agent.code.side.rename') }]
  if (canFork) list.push({ key: 'fork', icon: 'i-lucide-git-branch', label: t('agent.code.side.fork'), sep: true })
  // 已经在独立工作树里的会话，「复刻对话」本身就会给它一个新的工作树
  if (canFork && ws?.isGit && !s.worktree) list.push({ key: 'forkwt', icon: 'i-lucide-layers', label: t('agent.code.side.forkWt'), sep: !canFork })
  if (s.parentId && !s.detached) list.push({ key: 'detach', icon: 'i-lucide-arrow-left-to-line', label: t('agent.code.side.detach'), sep: !canFork })
  list.push({ key: 'export', icon: 'i-lucide-download', label: t('agent.code.side.export'), sep: true })
  list.push({ key: 'archive', icon: 'i-lucide-archive', label: t('agent.code.side.archive') })
  list.push({ key: 'delete', icon: 'i-lucide-trash-2', label: t('agent.common.delete') })
  return list
}

const target = ref<AgentSession | null>(null)
const renameOpen = ref(false)
const renameValue = ref('')
const deleteOpen = ref(false)
const withForks = ref(true)

/** 这个会话复刻出来的会话（一层层往下，不含已移到外层的） */
const kidsOf = (id: string): string[] => {
  const out: string[] = []
  const walk = (pid: string) => {
    for (const s of props.sessions) {
      if (s.parentId === pid && !s.detached) {
        out.push(s.id)
        walk(s.id)
      }
    }
  }
  walk(id)
  return out
}
const kidCount = computed(() => (target.value ? kidsOf(target.value.id).length : 0))

const run = async (act: Act, s: AgentSession) => {
  menu.value = ''
  target.value = s
  try {
    switch (act) {
      case 'rename':
        renameValue.value = sessionTitle(s.title)
        renameOpen.value = true
        return
      case 'delete':
        withForks.value = true
        deleteOpen.value = true
        return
      case 'fork':
      case 'forkwt': {
        const { data } = await forkAgentSession(s.id, act === 'forkwt')
        const wt = data.session?.worktree
        feedback.success(wt ? t('agent.code.side.forkedWt', { name: wt }) : t('agent.code.side.forked'))
        emit('changed')
        return
      }
      case 'detach':
        await detachAgentSession(s.id)
        feedback.success(t('agent.code.side.detached'))
        emit('changed')
        return
      case 'export': {
        const { data } = await exportAgentSessions([s.id])
        downloadText(exportMarkdown(data.sessions ?? []), `${safeFileName(sessionTitle(s.title))}.md`)
        feedback.success(t('agent.code.side.exported', { name: `${safeFileName(sessionTitle(s.title))}.md` }))
        return
      }
      case 'archive': {
        const { data } = await archiveAgentSessions([s.id], true)
        if (Number(data.skipped || 0) > 0) feedback.warning(t('agent.code.side.stillRunning'))
        else {
          feedback.success(t('agent.code.side.archived'))
          emit('changed')
          emit('removed', [s.id])
        }
        return
      }
    }
  } catch {
    // 请求层已经提示过
  }
}

const doRename = async () => {
  const s = target.value
  const title = renameValue.value.trim()
  if (!s || !title) return
  try {
    await renameAgentSession(s.id, title)
    renameOpen.value = false
    feedback.success(t('agent.code.side.renamed'))
    emit('changed')
  } catch {
    /* shown */
  }
}

const doDelete = async () => {
  const s = target.value
  if (!s) return
  const kids = withForks.value ? kidsOf(s.id) : []
  await deleteAgentSessions([s.id], withForks.value)
  feedback.success(kids.length ? t('agent.code.side.deletedWith', { n: kids.length }) : t('agent.code.side.deleted'))
  emit('changed')
  emit('removed', [s.id, ...kids])
}
</script>
