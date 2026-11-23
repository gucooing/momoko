<!-- Agent 设置 · 会话：状态筛选（带数量）、按标题搜索、每行归档 / 取消归档和删除、勾选后批量归档 / 导出 / 删除，
     「已归档」筛选下可以删除全部归档会话。有 Agent 进程在跑的会话（一轮结束后进程还留着，空闲的也可能有）
     行上有「进程」标记，点开看进程和它派出的子 Agent（只读）。
     这一页不能停止会话、不能结束进程；还在运行的会话不能归档。筛选、搜索、分页都在服务端做。
     只有自己的会话；在 Agent 管理页里（useAgentAll）是所有人的，行上带主人，标题不能点开（别人的会话打不开），没有新会话和删除全部归档。 -->
<template>
  <div class="ag ag-page">
    <AgManageHeader>
      <template #default>
        <span v-if="head" class="ag-ses-head">
          <span>{{ t('agent.sessions.total', { n: head.total }) }}</span>
          <span v-if="head.running" class="ag-ses-head__run">
            {{ t('agent.sessions.runningCount', { n: head.running }) }}
          </span>
          <span v-if="head.waiting" class="ag-ses-head__wait">
            {{ t('agent.sessions.waitingCount', { n: head.waiting }) }}
          </span>
        </span>
      </template>
      <template v-if="!all" #actions>
        <button type="button" class="ag-btn ag-btn--p" @click="newSession">
          <UIcon name="i-lucide-plus" />{{ t('agent.common.newSession') }}
        </button>
      </template>
    </AgManageHeader>

    <div class="ag-body">
      <div class="ag-bar">
        <div class="ag-ses-chips" role="tablist" :aria-label="t('agent.sessions.filterLabel')">
          <button
            v-for="f in filters"
            :key="f.key"
            type="button"
            class="ag-chip"
            role="tab"
            :class="{ 'is-on': status === f.key }"
            :aria-selected="status === f.key"
            @click="pickStatus(f.key)"
          >
            {{ f.label }}
            <span v-if="f.count != null" class="ag-n">{{ f.count }}</span>
          </button>
        </div>
        <label class="ag-inp ag-ses-search">
          <UIcon name="i-lucide-search" />
          <input
            v-model="keyword"
            type="search"
            :placeholder="t('agent.sessions.searchPlaceholder')"
            :aria-label="t('agent.sessions.searchLabel')"
          />
        </label>
        <button v-if="showPurge" type="button" class="ag-btn ag-btn--stop ag-btn--sm" @click="askPurge">
          <UIcon name="i-lucide-trash-2" />{{ t('agent.sessions.purge') }}
        </button>
      </div>

      <div class="ag-grp">
        <div v-for="s in sessions" :key="s.id" class="ag-rwx">
          <div class="ag-rw ag-ses-row">
            <label class="ag-ses-check">
              <input
                class="ag-cb"
                type="checkbox"
                :checked="selected.has(s.id)"
                :aria-label="t('agent.sessions.select', { title: titleOf(s) })"
                @change="toggleSelect(s.id)"
              />
            </label>
            <div class="ag-rm">
              <div class="ag-rt">
                <span v-if="all" class="ag-ses-title">{{ titleOf(s) }}</span>
                <RouterLink v-else class="ag-ses-title" :to="{ path: '/agent/code', query: { session: s.id } }">
                  {{ titleOf(s) }}
                </RouterLink>
                <AgAgentTag v-if="s.runtimeName" :kind="agentKindOf(s.runtimeKind)" :name="s.runtimeName" />
              </div>
              <div class="ag-rs">{{ subOf(s) }}</div>
            </div>
            <div class="ag-rc">
              <span class="ag-num ag-ses-time">{{ relTime(s.activeAt) }}</span>
              <span class="ag-pill ag-ses-pill" :class="PILL[s.status]">{{ statusText(s.status) }}</span>
              <button
                v-if="isLive(s)"
                type="button"
                class="ag-tag ag-ses-procbtn"
                :class="{ 'is-open': opened.has(s.id) }"
                :aria-label="t('agent.sessions.processOf', { title: titleOf(s) })"
                :title="t('agent.sessions.process')"
                :aria-expanded="opened.has(s.id)"
                @click="toggleProcess(s.id)"
              >
                <UIcon name="i-lucide-cpu" />{{ t('agent.sessions.processTag') }}
                <UIcon name="i-lucide-chevron-down" class="ag-ses-chev" :class="{ 'is-open': opened.has(s.id) }" />
              </button>
              <button
                type="button"
                class="ag-ib ag-ib--sm"
                :aria-label="
                  t(s.status === 'archived' ? 'agent.sessions.unarchiveOf' : 'agent.sessions.archiveOf', {
                    title: titleOf(s),
                  })
                "
                :title="t(s.status === 'archived' ? 'agent.sessions.unarchive' : 'agent.sessions.archive')"
                :disabled="busy.has(s.id)"
                @click="toggleArchive(s)"
              >
                <UIcon :name="s.status === 'archived' ? 'i-lucide-undo-2' : 'i-lucide-archive'" />
              </button>
              <button
                type="button"
                class="ag-ib ag-ib--sm"
                :aria-label="t('agent.sessions.deleteOf', { title: titleOf(s) })"
                :title="t('agent.common.delete')"
                :disabled="busy.has(s.id)"
                @click="askDelete(s)"
              >
                <UIcon name="i-lucide-trash-2" />
              </button>
            </div>
          </div>
          <div v-if="opened.has(s.id)" class="ag-ses-proc">
            <template v-if="procs.get(s.id)">
              <span class="ag-mono ag-ses-proc__cmd">{{ [procs.get(s.id)!.command, ...(procs.get(s.id)!.args ?? [])].join(' ') }}</span>
              <span v-if="Number(procs.get(s.id)!.pid) > 0" class="ag-num">PID {{ procs.get(s.id)!.pid }}</span>
              <span v-if="Number(procs.get(s.id)!.memoryBytes) > 0" class="ag-num">
                {{ t('agent.sessions.memory', { size: fmtBytes(procs.get(s.id)!.memoryBytes) }) }}
              </span>
              <span v-if="procs.get(s.id)!.startedAt">
                {{ t('agent.sessions.uptime', { time: sinceTime(procs.get(s.id)!.startedAt) }) }}
              </span>
              <div v-if="procs.get(s.id)!.subagents?.length" class="ag-ses-subs">
                <span class="ag-ses-subs__label">{{ t('agent.sessions.subagents') }}</span>
                <span v-for="sa in procs.get(s.id)!.subagents" :key="sa.id" class="ag-ses-sub">
                  <span class="ag-ses-sub__title">{{ sa.title || sa.id }}</span>
                  <span class="ag-pill" :class="SUB_PILL[sa.state]">{{ subState(sa.state) }}</span>
                </span>
              </div>
            </template>
            <span v-else class="ag-muted">{{ t(procs.has(s.id) ? 'agent.sessions.noProcess' : 'agent.common.loading') }}</span>
          </div>
        </div>
        <div v-if="!loaded" class="ag-empty">{{ t('agent.common.loading') }}</div>
        <div v-else-if="!sessions.length" class="ag-empty">{{ t('agent.sessions.empty') }}</div>
      </div>

      <Pagination
        v-if="total > PAGE_SIZE"
        v-model:page="page"
        :page-size="PAGE_SIZE"
        :page-sizes="[PAGE_SIZE]"
        :total="total"
        @change="load()"
      />

      <div
        v-if="selectedRows.length"
        class="ag-bulk ag-ses-bulk ag-noscroll"
        role="toolbar"
        :aria-label="t('agent.sessions.bulkLabel')"
      >
        <span class="ag-ses-bulk__n">{{ t('agent.sessions.selected', { n: selectedRows.length }) }}</span>
        <button type="button" :disabled="bulkBusy" @click="bulkArchive">
          <UIcon :name="allArchived ? 'i-lucide-undo-2' : 'i-lucide-archive'" />
          {{ t(allArchived ? 'agent.sessions.unarchive' : 'agent.sessions.archive') }}
        </button>
        <button type="button" :disabled="bulkBusy" @click="bulkExport">
          <UIcon name="i-lucide-download" />{{ t('agent.sessions.export') }}
        </button>
        <button type="button" class="ag-ses-bulk__del" :disabled="bulkBusy" @click="askBulkDelete">
          <UIcon name="i-lucide-trash-2" />{{ t('agent.common.delete') }}
        </button>
        <button
          type="button"
          class="ag-ses-bulk__x"
          :aria-label="t('agent.sessions.clearSelection')"
          :title="t('agent.sessions.clearSelection')"
          @click="selected.clear()"
        >
          <UIcon name="i-lucide-x" />
        </button>
      </div>
    </div>

    <AgConfirm
      v-model="confirm.open"
      danger
      :title="confirm.title"
      :message="confirm.message"
      :confirm-text="confirm.confirmText"
      :on-confirm="runConfirm"
    >
      <label v-if="confirm.forks > 0" class="ag-ses-forks">
        <input v-model="confirm.withForks" class="ag-cb" type="checkbox" />
        <span>{{ t('agent.sessions.deleteWithForks', { n: confirm.forks }) }}</span>
      </label>
    </AgConfirm>
  </div>
</template>

<script setup lang="ts">
import { useDebounceFn } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import {
  archiveAgentSessions,
  deleteAgentSessions,
  exportAgentSessions,
  getAgentSessionProcess,
  listAgentSessions,
  purgeAgentSessions,
} from '@/api/agent'
import type { AgentProcess, AgentSession, ListAgentSessionsResponse } from '@/types/v1/agent'
import { agentSend, useAgentConn, type AgentServerMessage } from '@/views/agent/composables/agentConn'
import AgAgentTag from '@/views/agent/components/AgAgentTag.vue'
import AgConfirm from '@/views/agent/components/AgConfirm.vue'
import AgManageHeader from '@/views/agent/components/AgManageHeader.vue'
import { useAgentAll } from '@/views/agent/composables/useAgentAll'
import { useAgentOverview } from '@/views/agent/composables/useAgentOverview'
import { useLastCodeSession } from '@/views/agent/composables/useLastCodeSession'
import { agentKindOf, fmtBytes, relTime, sinceTime } from '@/views/agent/utils/format'
import { modelName, useAgentModels } from '@/views/agent/composables/useAgentModels'
import { downloadText, exportMarkdown, providerLabel, safeFileName, sessionTitle } from '@/views/agent/utils/labels'

defineOptions({ name: 'AgentManageSessionsView' })

type StatusKey = 'all' | 'running' | 'waiting' | 'idle' | 'error' | 'archived'

const PAGE_SIZE = 20
const PILL: Record<string, string> = {
  running: 'ag-pill--ok',
  waiting: 'ag-pill--warn',
  error: 'ag-pill--bad',
}
const STATUSES = ['running', 'waiting', 'idle', 'error', 'archived']
// 子 Agent 的状态，和对话里一样：running 运行中，requires_action 等待批准，其它（idle）做完了
const SUB_PILL: Record<string, string> = {
  running: 'ag-pill--ok',
  requires_action: 'ag-pill--warn',
}

const { t } = useI18n()
const router = useRouter()
/** 在 Agent 管理页里：所有人的会话 */
const all = useAgentAll()
const { overview, refresh: refreshOverview } = useAgentOverview()
// 模型显示名从模型供应商同步、配置的那份清单里取
const { refresh: refreshModels } = useAgentModels()
const { remember: rememberCodeSession } = useLastCodeSession()
const newSession = () => {
  rememberCodeSession('')
  void router.push('/agent/code')
}

// —— 列表 ——
const status = ref<StatusKey>('all')
const keyword = ref('')
const page = ref(1)
const data = ref<ListAgentSessionsResponse | null>(null)
const loaded = ref(false)
const selected = reactive(new Set<string>())
const opened = reactive(new Set<string>())
const busy = reactive(new Set<string>())

const sessions = computed(() => data.value?.sessions ?? [])
const total = computed(() => Number(data.value?.total ?? 0))
const selectedRows = computed(() => sessions.value.filter((s) => selected.has(s.id)))
const allArchived = computed(
  () => selectedRows.value.length > 0 && selectedRows.value.every((s) => s.status === 'archived'),
)

// 有 Agent 进程在跑：一轮结束后进程还留在池子里，空闲超时才结束，所以看服务端给的，不看状态
const isLive = (s: AgentSession) => s.live
const subState = (state: string) =>
  state === 'running'
    ? t('agent.code.sub.running')
    : state === 'requires_action'
      ? t('agent.code.sub.waiting')
      : t('agent.code.sub.done')
const titleOf = (s: AgentSession) => s.title || t('agent.sessions.untitled')
const statusText = (value: string) => (STATUSES.includes(value) ? t(`agent.status.${value}`) : value)
// 主人只在 Agent 管理页里显示（自己的列表里都是自己）
const userOf = (s: AgentSession) => {
  if (!all) return ''
  const u = data.value?.users?.find((x) => x.id === s.userId)
  return u ? u.name || u.username : t('agent.admin.goneUser')
}
const forksOf = (s: AgentSession) => Number(data.value?.forkCounts?.[s.id] ?? 0)
const subOf = (s: AgentSession) => {
  const tail = s.status === 'error' && s.error ? s.error : ''
  const provider = providerLabel({ kind: s.providerKind, vendor: s.providerVendor, name: s.providerName })
  return [s.workspaceName || t('agent.common.noWorkspace'), provider, modelName(s.runtimeId, s.providerId, s.model), userOf(s), tail]
    .filter(Boolean)
    .join(' · ')
}

let seq = 0

const load = async (): Promise<void> => {
  const id = ++seq
  const kw = keyword.value.trim()
  try {
    const { data: res } = await listAgentSessions({
      status: status.value,
      keyword: kw,
      page: page.value,
      pageSize: PAGE_SIZE,
      all,
    })
    if (id !== seq) return
    // 这一页被删空了：退回最后一页
    const lastPage = Math.max(1, Math.ceil(Number(res.total || 0) / PAGE_SIZE))
    if (page.value > lastPage) {
      page.value = lastPage
      return load()
    }
    data.value = res
    const ids = new Set((res.sessions ?? []).map((s) => s.id))
    for (const sid of [...selected]) if (!ids.has(sid)) selected.delete(sid)
    // 进程已经结束的收起来
    const live = new Set((res.sessions ?? []).filter((s) => s.live).map((s) => s.id))
    for (const sid of [...opened]) if (!live.has(sid)) opened.delete(sid)
  } catch {
    // 请求层已经提示过
  } finally {
    if (id === seq) loaded.value = true
  }
}

/** 改动之后：列表和标签上的数字一起刷新 */
const reload = async () => {
  await Promise.all([load(), refreshOverview()])
}

const filters = computed(() => {
  const d = data.value
  const n = (value: number | undefined) => (d ? Number(value ?? 0) : null)
  const list: { key: StatusKey; label: string; count: number | null }[] = [
    { key: 'all', label: t('agent.sessions.all'), count: n(d?.countAll) },
    { key: 'running', label: t('agent.status.running'), count: n(d?.countRunning) },
    { key: 'waiting', label: t('agent.status.waiting'), count: n(d?.countWaiting) },
    { key: 'idle', label: t('agent.status.idle'), count: n(d?.countIdle) },
    { key: 'error', label: t('agent.status.error'), count: n(d?.countError) },
    { key: 'archived', label: t('agent.status.archived'), count: n(d?.countArchived) },
  ]
  return list
})

const pickStatus = (key: StatusKey) => {
  if (status.value === key) return
  status.value = key
  page.value = 1
  void load()
}

watch(
  keyword,
  useDebounceFn(() => {
    page.value = 1
    void load()
  }, 300),
)

const toggleSelect = (id: string) => {
  if (selected.has(id)) selected.delete(id)
  else selected.add(id)
}
// 进程信息（命令、进程号、内存、已运行）展开时才读：读内存要遍历进程树。
const procs = reactive(new Map<string, AgentProcess | null>())
const toggleProcess = async (id: string) => {
  if (opened.has(id)) {
    opened.delete(id)
    return
  }
  opened.add(id)
  procs.delete(id)
  try {
    const { data: res } = await getAgentSessionProcess(id)
    procs.set(id, res.process ?? null)
  } catch {
    opened.delete(id)
  }
}

// 页头：数量跟标签上的一致（不受搜索影响）；概览还没拿到时先用列表里的
const head = computed(() => {
  const o = overview.value
  if (o) {
    return {
      total: Number(o.sessions || 0),
      running: Number(o.sessionsRunning || 0),
      waiting: Number(o.sessionsWaiting || 0),
    }
  }
  const d = data.value
  if (!d) return null
  return {
    total: Number(d.countAll || 0),
    running: Number(d.countRunning || 0),
    waiting: Number(d.countWaiting || 0),
  }
})

// 会话状态的变化由长连接推来（自己的会话总会推来；Agent 管理页另外关注所有人的）：
// 这一页上的行直接换成新的；状态变了、或者有会话没了，过一会儿把列表和数量重新读一次。
const reloadSoon = useDebounceFn(() => void reload(), 800)
const onPush = (msg: AgentServerMessage) => {
  // 模型供应商里重新检查、改了配置：模型的显示名跟着换
  if (msg.t === 'config') {
    void refreshModels()
    return
  }
  if (msg.t === 'gone') {
    void reloadSoon()
    return
  }
  if (msg.t !== 'session' || !data.value) return
  const s = msg.session as AgentSession
  const list = data.value.sessions ?? []
  const i = list.findIndex((x) => x.id === s.id)
  if (i >= 0 && list[i]!.status === s.status) {
    list[i] = s
    if (!s.live) opened.delete(s.id)
    return
  }
  void reloadSoon()
}
const conn = useAgentConn(onPush, () => {
  if (all) agentSend({ op: 'watch', on: true })
})

onMounted(() => {
  conn.start()
  void load()
})
// 标签页是 keep-alive 的：切走就不再收推送，切回来悄悄刷新一次
onActivated(() => {
  conn.start()
  if (loaded.value) {
    void reload()
    void refreshModels()
  }
})
onDeactivated(() => {
  if (all) agentSend({ op: 'watch', on: false })
  conn.stop()
})

// —— 确认弹窗（删除一个、批量删除、删除全部归档共用）——
const confirm = reactive({
  open: false,
  title: '',
  message: '',
  confirmText: '',
  /** 要删的会话复刻出了几个会话；大于 0 时出现「连同它们一起删」的勾选 */
  forks: 0,
  withForks: true,
  run: (): Promise<void> => Promise.resolve(),
})
const ask = (opts: {
  title: string
  message: string
  confirmText: string
  forks?: number
  run: () => Promise<void>
}) => {
  confirm.title = opts.title
  confirm.message = opts.message
  confirm.confirmText = opts.confirmText
  confirm.forks = opts.forks ?? 0
  confirm.withForks = true
  confirm.run = opts.run
  confirm.open = true
}
const runConfirm = () => confirm.run()

// —— 每行的操作 ——
const toggleArchive = async (s: AgentSession) => {
  if (isLive(s)) {
    feedback.warning(t('agent.sessions.stillRunning'))
    return
  }
  if (busy.has(s.id)) return
  const archive = s.status !== 'archived'
  busy.add(s.id)
  try {
    const { data: res } = await archiveAgentSessions([s.id], archive)
    // 列表上还是空闲、实际刚开始跑的会话，服务端不会归档
    if (archive && Number(res.skipped || 0) > 0) feedback.warning(t('agent.sessions.stillRunning'))
    else feedback.success(t(archive ? 'agent.sessions.archived' : 'agent.sessions.unarchived', { title: titleOf(s) }))
    await reload()
  } catch {
    // 请求层已经提示过
  } finally {
    busy.delete(s.id)
  }
}

const askDelete = (s: AgentSession) => {
  const forks = forksOf(s)
  const title = titleOf(s)
  ask({
    title: t('agent.sessions.deleteTitle'),
    message: t('agent.sessions.deleteMessage', { title }),
    confirmText: t('agent.common.delete'),
    forks,
    run: async () => {
      await deleteAgentSessions([s.id], forks > 0 && confirm.withForks)
      feedback.success(t('agent.sessions.deleted', { title }))
      await reload()
    },
  })
}

// —— 批量 ——
const bulkBusy = ref(false)

const bulkArchive = async () => {
  const ids = selectedRows.value.map((s) => s.id)
  if (!ids.length || bulkBusy.value) return
  const archive = !allArchived.value
  bulkBusy.value = true
  try {
    const { data: res } = await archiveAgentSessions(ids, archive)
    const n = Number(res.changed || 0)
    const m = Number(res.skipped || 0)
    if (!archive) feedback.success(t('agent.sessions.bulkUnarchived', { n }))
    else if (m > 0) feedback.warning(t('agent.sessions.bulkArchivedSkipped', { n, m }))
    else feedback.success(t('agent.sessions.bulkArchived', { n }))
    selected.clear()
    await reload()
  } catch {
    // 请求层已经提示过
  } finally {
    bulkBusy.value = false
  }
}

const bulkExport = async () => {
  const ids = selectedRows.value.map((s) => s.id)
  if (!ids.length || bulkBusy.value) return
  bulkBusy.value = true
  try {
    const { data: res } = await exportAgentSessions(ids)
    const list = res.sessions ?? []
    const name = list.length === 1 ? sessionTitle(list[0]?.session?.title) : t('agent.sessions.exportName', { n: list.length })
    downloadText(exportMarkdown(list), `${safeFileName(name)}.md`)
    feedback.success(t('agent.sessions.exported', { n: ids.length }))
  } catch {
    // 请求层已经提示过
  } finally {
    bulkBusy.value = false
  }
}

const askBulkDelete = () => {
  const rows = selectedRows.value
  if (!rows.length) return
  const ids = rows.map((s) => s.id)
  const hasForks = rows.some((s) => forksOf(s) > 0)
  ask({
    title: t('agent.sessions.bulkDeleteTitle', { n: ids.length }),
    message: t(hasForks ? 'agent.sessions.bulkDeleteMessageForks' : 'agent.sessions.bulkDeleteMessage'),
    confirmText: t('agent.common.delete'),
    run: async () => {
      await deleteAgentSessions(ids, true)
      feedback.success(t('agent.sessions.bulkDeleted', { n: ids.length }))
      selected.clear()
      await reload()
    },
  })
}

// —— 删除全部归档会话 ——
const showPurge = computed(() => !all && status.value === 'archived' && Number(data.value?.countArchived ?? 0) > 0)

const askPurge = async () => {
  let n = Number(data.value?.countArchived ?? 0)
  if (keyword.value.trim()) {
    // 列表里的数量受搜索影响，而这里删的是全部归档会话：按不带搜索的数量问
    try {
      const { data: res } = await listAgentSessions({ status: 'archived', keyword: '', page: 1, pageSize: 1 })
      n = Number(res.countArchived ?? n)
    } catch {
      return
    }
  }
  ask({
    title: t('agent.sessions.purgeTitle', { n }),
    message: t('agent.sessions.purgeMessage'),
    confirmText: t('agent.sessions.purgeConfirm'),
    run: async () => {
      const { data: res } = await purgeAgentSessions()
      feedback.success(t('agent.sessions.purged', { n: Number(res.deleted ?? n) }))
      selected.clear()
      await reload()
    },
  })
}
</script>

<style lang="scss">
.ag-ses-head > span + span::before {
  content: '·';
  margin: 0 0.45em;
  font-weight: 400;
  color: var(--m-ink2);
}
.ag-ses-head__run {
  color: var(--m-ok);
  font-weight: 600;
}
.ag-ses-head__wait {
  color: var(--m-warn);
  font-weight: 600;
}
.ag-ses-chips {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.ag-ses-search {
  flex: 0 1 200px;
}
.ag-ses-check {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}
.ag-ses-title {
  color: var(--m-ink);
  text-decoration: none;
}
.ag-ses-title:hover {
  color: var(--m-pInk);
}
.ag-ses-time {
  margin-right: 6px;
  font-size: 12px;
  color: var(--m-ink2);
  white-space: nowrap;
}
.ag-ses-pill {
  margin-right: 4px;
}
.ag-ses-procbtn {
  margin-right: 4px;
  border: 0;
  font-family: inherit;
  cursor: pointer;
}
.ag-ses-procbtn:hover,
.ag-ses-procbtn.is-open {
  color: var(--m-ink);
}
.ag-ses-chev {
  transition: transform 0.15s;
}
.ag-ses-chev.is-open {
  transform: rotate(180deg);
}
.ag-ses-proc {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px 14px;
  margin: 0 0 8px 25px;
  padding: 6px 12px;
  border-radius: 10px;
  background: var(--m-s2);
  font-size: 12px;
  color: var(--m-ink2);
}
.ag-ses-proc__cmd {
  font-weight: 600;
  color: var(--m-ink);
}
.ag-ses-subs {
  flex-basis: 100%;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px 12px;
}
.ag-ses-subs__label {
  font-weight: 600;
  color: var(--m-ink);
}
.ag-ses-sub {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.ag-ses-sub__title {
  max-width: 240px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ag-ses-bulk {
  max-width: 100%;
  overflow-x: auto;
}
.ag-ses-bulk button {
  flex: none;
  white-space: nowrap;
}
.ag-ses-bulk button:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}
.ag-ses-bulk__n {
  margin-right: 8px;
  white-space: nowrap;
}
.ag-ses-bulk .ag-ses-bulk__del {
  color: var(--m-a);
}
.ag-ses-bulk .ag-ses-bulk__x {
  width: 28px;
  padding: 0;
  justify-content: center;
}
.ag-ses-forks {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  font-size: 12.5px;
  line-height: 1.55;
  cursor: pointer;
}
.ag-ses-forks .ag-cb {
  margin-top: 3px;
}
@media (width < 992px) {
  .ag-ses-search {
    flex: 1 1 100%;
    min-height: 40px;
  }
  // 勾选框和标题留在同一行，时间、状态和操作换到下一行
  .ag-ses-row > .ag-rm {
    flex: 1 1 0;
  }
  .ag-ses-check {
    width: 32px;
    min-height: 40px;
    margin-left: -8px;
  }
  .ag-ses-time {
    margin-right: auto;
  }
  .ag-ses-proc {
    margin-left: 0;
  }
  .ag-ses-bulk button {
    min-height: 40px;
  }
  .ag-ses-bulk .ag-ses-bulk__x {
    width: 40px;
  }
}
</style>
