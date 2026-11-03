<!-- 任务与日志 · 操作记录（第二期）：标题 + 计数说明 + 导出 → LogsSubNav → 今日统计贴纸卡 → 时间/模块/类型/用户筛选 + 结果 chips(带数) + 搜索
     → PC 按天分组表格（模块 chip、动作 + 目标、目标节点/转发标签、结果 pill、IP）/ 手机按天分组卡片；点行打开右侧详情抽屉（转发链路、信息格、请求 JSON）。
     保留 listOperationLogs 契约（用户/类型/结果/路径筛选）与用户预加载；新增 keywords/module/时间范围、成功失败计数与统计接口。 -->
<template>
  <div class="au-page">
    <PageHeader :title="t('system.logsNav.label')">
      <template #description>
        <span class="m-num">{{ t('system.audit.summary', { total: fmtNum(stats.total) }) }}</span>
        <template v-if="stats.todayFailed">
          · <b class="au-tone-bad">{{ t('system.audit.summaryFailed', { n: fmtNum(stats.todayFailed) }) }}</b>
        </template>
        <template v-if="stats.earliest">
          · {{ t('system.audit.summarySince', { date: stats.earliest }) }}
        </template>
      </template>
      <template #actions>
        <UButton color="primary" icon="i-lucide-download" :loading="exporting" :disabled="!pagination.total" @click="exportCsv">
          {{ t('system.audit.export') }}
        </UButton>
      </template>
    </PageHeader>

    <LogsSubNav :counts="navCounts" />

    <div class="au-stats">
      <div v-for="s in statCards" :key="s.key" class="au-stat">
        <span class="m-sticker au-stat__icon" :class="`m-sticker--${s.tone}`" :style="{ transform: `rotate(${s.rot})` }">
          <component :is="menuStore.iconComponents[s.icon]" />
        </span>
        <span class="au-stat__text">
          <span class="au-stat__label">{{ s.label }}</span>
          <span class="au-stat__value m-num">{{ fmtNum(s.value) }}</span>
        </span>
      </div>
    </div>

    <div class="au-filters">
      <div class="au-filters__sel au-filters__sel--range">
        <AppSelect v-model="queryForm.range" :options="rangeOptions" fit @update:model-value="search" />
      </div>
      <div class="au-filters__sel">
        <AppSelect v-model="queryForm.module" :options="moduleOptions" fit @update:model-value="onModuleChange" />
      </div>
      <div class="au-filters__sel au-filters__sel--wide">
        <AppSelect
          v-model="queryForm.operationType"
          :options="typeSelectOptions"
          searchable
          fit
          :search-placeholder="t('system.operation.selectOperationType')"
          :no-match-text="t('system.common.noData')"
          @update:model-value="search"
        />
      </div>
      <div class="au-filters__sel au-filters__sel--wide">
        <AppSelect
          v-model="queryForm.userId"
          :options="userOptions"
          searchable
          fit
          :search-placeholder="t('system.operation.searchUserPlaceholder')"
          :no-match-text="t('system.common.noData')"
          @update:model-value="search"
        />
      </div>
    </div>

    <ListToolbar
      v-model:status="queryForm.result"
      v-model:keyword="queryForm.keywords"
      :chips="resultChips"
      :chips-label="t('system.operation.result')"
      :placeholder="t('system.audit.searchPlaceholder')"
      @search="search"
    >
      <UButton v-if="hasFilters" color="neutral" variant="ghost" icon="i-lucide-rotate-ccw" @click="reset">
        {{ t('system.common.reset') }}
      </UButton>
    </ListToolbar>

    <!-- PC：按天分组表格 -->
    <div v-if="!menuStore.isMobile" class="au-table">
      <table>
        <thead>
          <tr>
            <th class="au-col-time">{{ t('system.audit.time') }}</th>
            <th>{{ t('system.operation.user') }}</th>
            <th>{{ t('system.audit.module') }}</th>
            <th>{{ t('system.audit.action') }}</th>
            <th>{{ t('system.audit.targetNode') }}</th>
            <th>{{ t('system.operation.result') }}</th>
            <th>IP</th>
            <th class="au-col-go"><span class="au-sr">{{ t('system.operation.detail') }}</span></th>
          </tr>
        </thead>
        <tbody v-if="loading && !logs.length">
          <tr v-for="i in 6" :key="i"><td colspan="8"><div class="au-skeleton" /></td></tr>
        </tbody>
        <tbody v-else-if="!logs.length">
          <tr>
            <td colspan="8">
              <EmptyState icon="HOutline:ClipboardDocumentListIcon" :title="t('system.common.noData')" :description="t('system.operation.emptyDesc')" />
            </td>
          </tr>
        </tbody>
        <tbody v-for="g in groups" v-else :key="g.key">
          <tr class="au-group">
            <td colspan="8">{{ g.label }} <span class="m-num">{{ t('system.audit.groupCount', { n: g.rows.length }) }}</span></td>
          </tr>
          <tr
            v-for="r in g.rows"
            :key="r.key"
            class="au-row"
            :class="{ 'is-sel': current?.key === r.key }"
            tabindex="0"
            @click="openDetail(r)"
            @keydown.enter="openDetail(r)"
          >
            <td class="m-num au-time">{{ r.tm }}</td>
            <td>
              <span class="au-user">
                <span class="au-avatar" :class="r.userId ? 'm-sticker--a' : 'm-sticker--a2'">{{ r.userInitial }}</span>
                <span class="au-user__name">{{ r.userName }}</span>
              </span>
            </td>
            <td>
              <span class="au-mod" :class="`m-sticker--${r.module.tone}`">
                <component :is="menuStore.iconComponents[r.module.icon]" />
                {{ t(r.module.labelKey) }}
              </span>
            </td>
            <td class="au-act">
              <div class="au-act__title">{{ r.action }}</div>
              <div class="au-act__target m-mono">{{ r.target }}</div>
            </td>
            <td>
              <span class="au-node" :class="{ 'is-fwd': r.forwarded }">
                <component :is="menuStore.iconComponents[r.forwarded ? 'HOutline:ArrowsRightLeftIcon' : 'HOutline:ServerStackIcon']" />
                {{ r.nodeLabel }}
              </span>
            </td>
            <td>
              <StatusPill :variant="r.success ? 'success' : 'error'" :label="r.success ? t('system.common.success') : t('system.common.failed')" />
            </td>
            <td class="m-mono au-ip">{{ r.ip || '—' }}</td>
            <td class="au-col-go">
              <AppIconButton icon="HOutline:ChevronRightIcon" :label="t('system.audit.viewDetail', { name: r.action })" :box="36" @click.stop="openDetail(r)" />
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 手机：按天分组卡片 -->
    <template v-else>
      <div v-if="loading && !logs.length" class="au-cards">
        <div v-for="i in 4" :key="i" class="au-skeleton au-skeleton--card" />
      </div>
      <EmptyState
        v-else-if="!logs.length"
        icon="HOutline:ClipboardDocumentListIcon"
        :title="t('system.common.noData')"
        :description="t('system.operation.emptyDesc')"
      />
      <div v-else class="au-cards">
        <section v-for="g in groups" :key="g.key" class="au-cards__group">
          <h3 class="au-cards__day">{{ g.label }} <span class="m-num">{{ t('system.audit.groupCount', { n: g.rows.length }) }}</span></h3>
          <button v-for="r in g.rows" :key="r.key" type="button" class="au-card" @click="openDetail(r)">
            <span class="m-sticker au-card__icon" :class="`m-sticker--${r.module.tone}`">
              <component :is="menuStore.iconComponents[r.module.icon]" />
            </span>
            <span class="au-card__main">
              <span class="au-card__line">
                <b class="au-card__title">{{ r.action }}</b>
                <StatusPill :variant="r.success ? 'success' : 'error'" :label="r.success ? t('system.common.success') : t('system.common.failed')" />
              </span>
              <span class="au-card__target m-mono">{{ r.target }}</span>
              <span class="au-card__meta">
                <span class="m-num">{{ r.tm }}</span>
                <span>{{ r.userName }}</span>
                <span class="au-node au-node--sm" :class="{ 'is-fwd': r.forwarded }">{{ r.nodeLabel }}</span>
              </span>
            </span>
          </button>
        </section>
      </div>
    </template>

    <Pagination
      v-model:page="pagination.page"
      v-model:page-size="pagination.pageSize"
      :total="pagination.total"
      @change="getList"
    />

    <!-- 详情抽屉 -->
    <Teleport to="body">
      <Transition name="au-drawer">
        <div v-if="current" class="au-drawer" @keydown.esc="current = null">
          <button type="button" class="au-drawer__mask" :aria-label="t('system.common.close')" @click="current = null" />
          <aside class="au-drawer__panel" role="dialog" :aria-label="t('system.operation.detailTitle')">
            <header class="au-drawer__head">
              <span class="m-sticker au-drawer__icon" :class="`m-sticker--${current.module.tone}`">
                <component :is="menuStore.iconComponents[current.module.icon]" />
              </span>
              <div class="au-drawer__title">
                <span class="au-drawer__kicker">{{ t(current.module.labelKey) }} · {{ t('system.operation.detailTitle') }}</span>
                <h2>{{ current.action }}</h2>
                <span class="m-mono au-drawer__target">{{ current.target }}</span>
              </div>
              <AppIconButton icon="HOutline:XMarkIcon" :label="t('system.common.close')" @click="current = null" />
            </header>

            <div class="au-result" :class="current.success ? 'is-ok' : 'is-bad'">
              <StatusPill :variant="current.success ? 'success' : 'error'" :label="current.success ? t('system.common.success') : t('system.common.failed')" />
              <span class="au-result__msg">{{ current.success ? t('system.audit.done') : current.error || t('system.audit.failedNoReason') }}</span>
              <span class="m-num au-result__dur">{{ fmtMs(current.durationMs) }}</span>
            </div>

            <section v-if="current.forwarded" class="au-chain">
              <h3 class="au-label">{{ t('system.audit.chain') }}</h3>
              <div class="au-chain__row">
                <div class="au-chain__box">
                  <span>{{ current.apiKey ? t('system.audit.remoteMaster') : t('system.audit.master') }}</span>
                  <b>{{ current.operator || current.userName }}</b>
                  <span v-if="current.apiKey" class="m-mono">API Key · {{ current.apiKey }}</span>
                </div>
                <span class="m-sticker m-sticker--a2 au-chain__arrow">
                  <component :is="menuStore.iconComponents['HOutline:ArrowRightIcon']" />
                </span>
                <div class="au-chain__box is-node">
                  <span>{{ t('system.audit.node') }}</span>
                  <b>{{ current.apiKey ? t('system.audit.thisNode') : current.node }}</b>
                </div>
              </div>
            </section>

            <dl class="au-dl">
              <div><dt>{{ t('system.audit.time') }}</dt><dd class="m-num">{{ current.fullTime }}</dd></div>
              <div><dt>{{ t('system.operation.user') }}</dt><dd>{{ current.userName }}</dd></div>
              <div><dt>{{ t('system.audit.sourceIp') }}</dt><dd class="m-mono">{{ current.ip || '—' }}</dd></div>
              <div><dt>{{ t('system.audit.targetNode') }}</dt><dd>{{ current.nodeLabel }}</dd></div>
              <div><dt>{{ t('system.operation.operationType') }}</dt><dd>{{ current.action }}</dd></div>
              <div><dt>{{ t('system.operation.duration') }}</dt><dd class="m-num">{{ fmtMs(current.durationMs) }}</dd></div>
              <div class="is-full"><dt>{{ t('system.audit.endpoint') }}</dt><dd class="m-mono">{{ current.method }} {{ current.path || '—' }}</dd></div>
              <div class="is-full"><dt>{{ t('system.audit.client') }}</dt><dd class="m-mono">{{ current.userAgent || '—' }}</dd></div>
            </dl>

            <section class="au-json">
              <div class="au-json__head">
                <h3 class="au-label">{{ t('system.audit.request') }}</h3>
                <span v-if="current.method" class="au-tag m-mono">{{ current.method }}</span>
                <AppIconButton icon="HOutline:ClipboardDocumentIcon" :label="t('system.audit.copyRequest')" :box="36" @click="copyText(current.requestJson)" />
              </div>
              <pre class="m-mono">{{ current.requestJson }}</pre>
            </section>

            <footer class="au-drawer__foot">
              <UButton color="neutral" variant="soft" icon="i-lucide-copy" block @click="copyText(current.rawJson)">
                {{ t('system.audit.copyAll') }}
              </UButton>
              <UButton v-if="current.userId" color="neutral" variant="outline" icon="i-lucide-user" block @click="filterUser(current.userId)">
                {{ t('system.audit.userRecords') }}
              </UButton>
            </footer>
          </aside>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import { useClipboard } from '@vueuse/core'
import { listOperationLogs, getOperationLogStats } from '@/api/system'
import { listTasks } from '@/api/task'
import { userPage } from '@/api/user'
import { PERM } from '@/config/permission'
import { OperationType, type OperationLogInfo } from '@/types/v1/system'
import type { UserInfo } from '@/types/v1/user'
import type { ToolbarChip } from '@/components/ui/ListToolbar.vue'
import LogsSubNav from '@/views/system/components/LogsSubNav.vue'
import { operationTypeLabelKeys, OPERATION_MODULES, moduleOf, type OperationModule } from './operationTypes'

defineOptions({ name: 'OperationLogView' })

const menuStore = useMenuStore()
const { t, locale } = useI18n()
const { copy } = useClipboard()

type RangeKey = 'today' | '7d' | '30d' | 'all'
type ResultKey = '' | 'ok' | 'fail'

const loading = ref(false)
const exporting = ref(false)
const logs = ref<OperationLogInfo[]>([])
const users = ref<UserInfo[]>([])
const counts = ref({ success: 0, failed: 0 })
const stats = ref({ todayTotal: 0, todayFailed: 0, todayActiveUsers: 0, todayForwarded: 0, total: 0, earliest: '' })
const taskTotal = ref<number | null>(null)

const emptyQuery = () => ({
  userId: '' as string,
  operationType: '' as string,
  module: '' as string,
  result: '' as ResultKey,
  keywords: '',
  range: '7d' as RangeKey,
})
const queryForm = ref(emptyQuery())
const pagination = ref({ page: 1, pageSize: 20, total: 0 })

const hasFilters = computed(() => {
  const q = queryForm.value
  return !!(q.userId || q.operationType || q.module || q.result || q.keywords || q.range !== '7d')
})
const navCounts = computed(() => ({ operation: stats.value.total, task: taskTotal.value }))
const fmtNum = (n: number) => new Intl.NumberFormat(locale.value).format(n)

// —— 用户：筛选下拉 + userId→名称 ——
const userMap = computed<Record<string, UserInfo>>(() => {
  const m: Record<string, UserInfo> = {}
  for (const u of users.value) m[u.userId] = u
  return m
})
const userDisplay = (id?: string) => {
  if (!id) return ''
  const u = userMap.value[id]
  return u ? u.name || u.username : id
}
const userOptions = computed(() => [
  { label: t('system.operation.allUsers'), value: '' },
  ...users.value.map((u) => ({ label: `${u.username}${u.name ? ` (${u.name})` : ''}`, value: u.userId })),
])

const rangeOptions = computed<{ label: string; value: RangeKey }[]>(() => [
  { label: t('system.audit.rangeToday'), value: 'today' },
  { label: t('system.audit.range7d'), value: '7d' },
  { label: t('system.audit.range30d'), value: '30d' },
  { label: t('system.audit.rangeAll'), value: 'all' },
])
const moduleOptions = computed(() => [
  { label: t('system.audit.allModules'), value: '' },
  ...OPERATION_MODULES.map((m) => ({ label: t(m.labelKey), value: m.key })),
])
const getOperationTypeLabel = (operationType: string) => {
  const labelKey = operationTypeLabelKeys[operationType]
  return labelKey ? t(labelKey) : operationType
}
const typeSelectOptions = computed(() => [
  { label: t('system.operation.allTypes'), value: '' },
  ...Object.entries(operationTypeLabelKeys)
    .filter(([value]) => value !== OperationType.UNRECOGNIZED)
    .filter(([value]) => !queryForm.value.module || moduleOf(value).key === queryForm.value.module)
    .map(([value, labelKey]) => ({ label: t(labelKey), value })),
])
const resultChips = computed<ToolbarChip<ResultKey>[]>(() => [
  { label: t('system.common.all'), value: '', count: counts.value.success + counts.value.failed },
  { label: t('system.common.success'), value: 'ok', count: counts.value.success },
  { label: t('system.common.failed'), value: 'fail', count: counts.value.failed },
])

const statCards = computed(() => [
  { key: 'total', label: t('system.audit.statToday'), value: stats.value.todayTotal, icon: 'HOutline:BoltIcon', tone: 'solid-p', rot: '-4deg' },
  { key: 'failed', label: t('system.audit.statFailed'), value: stats.value.todayFailed, icon: 'HOutline:ExclamationTriangleIcon', tone: 'bad', rot: '3deg' },
  { key: 'users', label: t('system.audit.statUsers'), value: stats.value.todayActiveUsers, icon: 'HOutline:UsersIcon', tone: 'solid-a', rot: '-2deg' },
  { key: 'fwd', label: t('system.audit.statForwarded'), value: stats.value.todayForwarded, icon: 'HOutline:ArrowsRightLeftIcon', tone: 'solid-a2', rot: '4deg' },
])

// —— 行视图 ——
interface DetailJson {
  method?: string
  path?: string
  operation?: string
  request?: unknown
  success?: boolean
  error?: string
  duration_ms?: number
  node?: string
  api_key?: string
  operator?: string
}
interface LogRow {
  key: string
  day: string
  tm: string
  fullTime: string
  userId: string
  userName: string
  userInitial: string
  module: OperationModule
  action: string
  target: string
  node: string
  nodeLabel: string
  forwarded: boolean
  apiKey: string
  operator: string
  success: boolean
  error: string
  ip: string
  userAgent: string
  method: string
  path: string
  durationMs: number
  requestJson: string
  rawJson: string
}

const parseDetail = (raw: string): DetailJson => {
  if (!raw) return {}
  try {
    const v = JSON.parse(raw)
    return v && typeof v === 'object' ? (v as DetailJson) : {}
  } catch {
    return {}
  }
}
const TARGET_KEYS = ['name', 'title', 'username', 'image', 'path', 'paths', 'host', 'email', 'id', 'ids']
const pickTarget = (req: unknown, fallback: string) => {
  if (req && typeof req === 'object') {
    const obj = req as Record<string, unknown>
    for (const k of TARGET_KEYS) {
      const v = obj[k]
      if (typeof v === 'string' && v) return v
      if (typeof v === 'number') return String(v)
      if (Array.isArray(v) && v.length) return v.slice(0, 3).join(', ') + (v.length > 3 ? ` +${v.length - 3}` : '')
    }
  }
  return fallback
}

const toRow = (log: OperationLogInfo, idx: number): LogRow => {
  const d = parseDetail(log.detail)
  const time = log.operationTime ? dayjs(log.operationTime) : null
  const userId = log.userId || ''
  const operator = d.operator || ''
  const userName = userDisplay(userId) || operator || (d.api_key ? `API Key · ${d.api_key}` : t('system.audit.anonymous'))
  const node = d.node || ''
  const forwarded = !!(node || d.api_key)
  const path = log.path || d.path || ''
  return {
    key: `${time?.valueOf() ?? 0}-${idx}`,
    day: time ? time.format('YYYY-MM-DD') : '',
    tm: time ? time.format('HH:mm:ss') : '—',
    fullTime: time ? time.format('YYYY-MM-DD HH:mm:ss') : '—',
    userId,
    userName,
    userInitial: [...userName][0]?.toUpperCase() ?? '?',
    module: moduleOf(String(log.operationType)),
    action: getOperationTypeLabel(String(log.operationType)),
    target: pickTarget(d.request, path || '—'),
    node,
    nodeLabel: d.api_key ? t('system.audit.fromMaster') : node || t('system.audit.local'),
    forwarded,
    apiKey: d.api_key || '',
    operator,
    success: log.success,
    error: d.error || '',
    ip: log.ip,
    userAgent: log.userAgent,
    method: d.method || '',
    path,
    durationMs: Number(log.durationMs) || 0,
    requestJson: JSON.stringify(d.request ?? {}, null, 2),
    rawJson: (() => {
      try {
        return JSON.stringify(JSON.parse(log.detail), null, 2)
      } catch {
        return log.detail || ''
      }
    })(),
  }
}

const rows = computed(() => logs.value.map(toRow))
const dayLabel = (day: string) => {
  const d = dayjs(day)
  const today = dayjs().startOf('day')
  const date = d.toDate().toLocaleDateString(locale.value, { month: 'long', day: 'numeric', weekday: 'short' })
  if (d.isSame(today, 'day')) return `${t('system.tasks.today')} · ${date}`
  if (d.isSame(today.subtract(1, 'day'), 'day')) return `${t('system.tasks.yesterday')} · ${date}`
  return date
}
const groups = computed(() => {
  const out: { key: string; label: string; rows: LogRow[] }[] = []
  for (const r of rows.value) {
    let g = out[out.length - 1]
    if (!g || g.key !== r.day) {
      g = { key: r.day, label: r.day ? dayLabel(r.day) : '—', rows: [] }
      out.push(g)
    }
    g.rows.push(r)
  }
  return out
})

const fmtMs = (ms: number) => (ms >= 1000 ? `${(ms / 1000).toFixed(ms >= 10000 ? 0 : 1)} s` : `${ms} ms`)

// —— 详情抽屉 ——
const current = ref<LogRow | null>(null)
const openDetail = (r: LogRow) => {
  current.value = r
}
const copyText = async (value: string) => {
  try {
    await copy(value)
    feedback.success(t('system.operation.copied'))
  } catch {
    feedback.error(t('system.operation.copyFailed'))
  }
}
const filterUser = (userId: string) => {
  current.value = null
  queryForm.value = { ...emptyQuery(), userId, range: 'all' }
  search()
}

// —— 查询 ——
const rangeStart = (range: RangeKey) => {
  if (range === 'today') return dayjs().startOf('day').toDate()
  if (range === '7d') return dayjs().startOf('day').subtract(6, 'day').toDate()
  if (range === '30d') return dayjs().startOf('day').subtract(29, 'day').toDate()
  return undefined
}
const buildQuery = (page: number, pageSize: number) => {
  const q = queryForm.value
  return {
    userId: q.userId || undefined,
    operationType: (q.operationType || undefined) as OperationType | undefined,
    module: q.module || undefined,
    success: q.result === '' ? undefined : q.result === 'ok',
    keywords: q.keywords.trim() || undefined,
    page,
    pageSize,
    startTime: rangeStart(q.range),
    endTime: undefined,
  }
}

const loadUsers = async () => {
  try {
    const { data } = await userPage({ page: 1, pageSize: 1000 })
    users.value = data?.users || []
  } catch {
    users.value = []
  }
}

const getList = async () => {
  loading.value = true
  try {
    const { data } = await listOperationLogs(buildQuery(pagination.value.page, pagination.value.pageSize))
    logs.value = data?.logs || []
    pagination.value.total = Number(data?.total || 0)
    pagination.value.page = Number(data?.page || pagination.value.page)
    pagination.value.pageSize = Number(data?.pageSize || pagination.value.pageSize)
    counts.value = { success: Number(data?.successCount) || 0, failed: Number(data?.failedCount) || 0 }
  } catch {
    logs.value = []
  } finally {
    loading.value = false
  }
}

const loadStats = async () => {
  try {
    const { data } = await getOperationLogStats()
    stats.value = {
      todayTotal: Number(data?.todayTotal) || 0,
      todayFailed: Number(data?.todayFailed) || 0,
      todayActiveUsers: Number(data?.todayActiveUsers) || 0,
      todayForwarded: Number(data?.todayForwarded) || 0,
      total: Number(data?.total) || 0,
      earliest: data?.earliestTime ? dayjs(data.earliestTime).format('YYYY-MM-DD') : '',
    }
  } catch {
    /* 统计失败不影响列表 */
  }
}

const loadTaskTotal = async () => {
  if (!menuStore.hasButtonPermission('task:manage')) return
  try {
    const { data } = await listTasks({ page: 1, pageSize: 1 })
    taskTotal.value = Number(data?.counts?.all ?? data?.total) || 0
  } catch {
    taskTotal.value = null
  }
}

const search = () => {
  pagination.value.page = 1
  getList()
}
const onModuleChange = () => {
  if (queryForm.value.operationType && moduleOf(queryForm.value.operationType).key !== queryForm.value.module) {
    queryForm.value.operationType = ''
  }
  search()
}
const reset = () => {
  queryForm.value = emptyQuery()
  search()
}

// 导出：按当前筛选逐页拉取（上限 5000 条），生成 CSV
const EXPORT_LIMIT = 5000
const csvCell = (v: string) => `"${v.replace(/"/g, '""')}"`
const exportCsv = async () => {
  exporting.value = true
  try {
    const all: OperationLogInfo[] = []
    for (let page = 1; all.length < EXPORT_LIMIT; page++) {
      const { data } = await listOperationLogs(buildQuery(page, 100))
      const batch = data?.logs || []
      all.push(...batch)
      if (batch.length < 100) break
    }
    const head = [
      t('system.audit.time'),
      t('system.operation.user'),
      t('system.audit.module'),
      t('system.operation.operationType'),
      t('system.audit.target'),
      t('system.audit.targetNode'),
      t('system.operation.result'),
      'IP',
      t('system.operation.duration'),
      t('system.audit.endpoint'),
      t('system.audit.client'),
    ]
    const lines = all.map(toRow).map((r) =>
      [
        r.fullTime,
        r.userName,
        t(r.module.labelKey),
        r.action,
        r.target,
        r.nodeLabel,
        r.success ? t('system.common.success') : t('system.common.failed'),
        r.ip,
        String(r.durationMs),
        `${r.method} ${r.path}`.trim(),
        r.userAgent,
      ]
        .map(csvCell)
        .join(','),
    )
    const blob = new Blob([`﻿${[head.map(csvCell).join(','), ...lines].join('\n')}`], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `operation-logs-${dayjs().format('YYYYMMDD-HHmm')}.csv`
    a.click()
    URL.revokeObjectURL(url)
  } finally {
    exporting.value = false
  }
}

const onKey = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && current.value) current.value = null
}

onMounted(() => {
  loadUsers()
  getList()
  loadStats()
  loadTaskTotal()
  window.addEventListener('keydown', onKey)
})
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))
</script>

<style scoped lang="scss">
.au-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.au-tone-bad {
  color: var(--m-bad);
  font-weight: 700;
}
.au-sr {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0 0 0 0);
}

/* 统计卡 */
.au-stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
}
.au-stat {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px 18px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  min-width: 0;
}
.au-stat__icon {
  flex: none;
  width: 44px;
  height: 44px;
  border-radius: 15px;
}
.au-stat__icon :deep(svg) {
  width: 22px;
  height: 22px;
}
.au-stat__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.au-stat__label {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.au-stat__value {
  font-family: var(--app-font-display);
  font-size: 1.625rem;
  line-height: 1.05;
  font-weight: 800;
  color: var(--el-text-color-primary);
}

/* 筛选 */
.au-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
.au-filters__sel {
  flex: 0 0 150px;
  min-width: 0;
}
.au-filters__sel--range {
  flex-basis: 140px;
}
.au-filters__sel--wide {
  flex-basis: 200px;
}
.au-filters__sel :deep(button) {
  min-height: 40px;
  border-radius: var(--app-radius-pill);
}

/* 表格 */
.au-table {
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
  padding: 4px 8px;
  overflow-x: auto;
  scrollbar-width: none;
}
.au-table::-webkit-scrollbar {
  display: none;
}
.au-table table {
  width: 100%;
  min-width: 980px;
  border-collapse: separate;
  border-spacing: 0;
}
.au-table th {
  padding: 12px 10px;
  text-align: left;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  border-bottom: 1px solid var(--el-border-color-lighter);
  white-space: nowrap;
}
.au-table td {
  padding: 12px 10px;
  vertical-align: middle;
  font-size: 0.875rem;
  color: var(--el-text-color-primary);
}
.au-col-time {
  width: 96px;
}
.au-col-go {
  width: 52px;
  position: relative;
}
.au-group td {
  padding: 16px 10px 6px;
  font-size: 0.8125rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.au-group td span {
  margin-left: 6px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.au-row {
  cursor: pointer;
  transition: background 0.15s;
}
.au-row td {
  border-top: 1px solid var(--el-border-color-lighter);
}
.au-row td:first-child {
  border-radius: var(--app-radius-sm) 0 0 var(--app-radius-sm);
}
.au-row td:last-child {
  border-radius: 0 var(--app-radius-sm) var(--app-radius-sm) 0;
}
.au-row:hover {
  background: var(--el-fill-color-light);
}
.au-row.is-sel {
  background: var(--m-pSoft);
}
.au-row:focus-visible {
  outline: 3px solid var(--m-a2);
  outline-offset: -3px;
}
.au-time {
  font-weight: 700;
}
.au-user {
  display: flex;
  align-items: center;
  gap: 8px;
}
.au-avatar {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 50%;
  font-size: 0.75rem;
  font-weight: 800;
}
.au-user__name {
  font-weight: 700;
  white-space: nowrap;
  max-width: 140px;
  overflow: hidden;
  text-overflow: ellipsis;
}
.au-mod {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 28px;
  padding: 0 10px;
  border-radius: var(--app-radius-pill);
  font-size: 0.75rem;
  font-weight: 700;
  white-space: nowrap;
}
.au-mod :deep(svg) {
  width: 14px;
  height: 14px;
}
.m-sticker--ink {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
}
.au-act {
  max-width: 300px;
}
.au-act__title {
  font-weight: 800;
}
.au-act__target {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.au-node {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 26px;
  padding: 0 9px;
  border-radius: var(--app-radius-xs);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-regular);
  font-size: 0.75rem;
  font-weight: 700;
  white-space: nowrap;
}
.au-node :deep(svg) {
  width: 13px;
  height: 13px;
}
.au-node.is-fwd {
  background: var(--m-a2Soft);
  color: var(--m-ink);
}
.au-node--sm {
  height: 22px;
  padding: 0 7px;
}
.au-ip {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}
.au-skeleton {
  height: 44px;
  border-radius: var(--app-radius-sm);
  background: linear-gradient(100deg, var(--el-fill-color-light) 30%, var(--el-fill-color) 50%, var(--el-fill-color-light) 70%);
  background-size: 200% 100%;
  animation: au-shimmer 1.4s ease-in-out infinite;
}
.au-skeleton--card {
  height: 96px;
  border-radius: var(--app-radius-lg);
}
@keyframes au-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}

/* 手机卡片 */
.au-cards {
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.au-cards__group {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.au-cards__day {
  margin: 0;
  font-size: 0.8125rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.au-cards__day span {
  margin-left: 4px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.au-card {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  width: 100%;
  padding: 14px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  font: inherit;
  text-align: left;
  color: inherit;
  cursor: pointer;
}
.au-card__icon {
  flex: none;
  width: 40px;
  height: 40px;
  border-radius: 13px;
}
.au-card__icon :deep(svg) {
  width: 20px;
  height: 20px;
}
.au-card__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.au-card__line {
  display: flex;
  align-items: center;
  gap: 8px;
}
.au-card__title {
  flex: 1;
  min-width: 0;
  font-size: 0.9375rem;
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.au-card__target {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.au-card__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px 10px;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}

/* 抽屉 */
.au-drawer {
  position: fixed;
  inset: 0;
  z-index: 2000;
}
.au-drawer__mask {
  position: absolute;
  inset: 0;
  border: 0;
  padding: 0;
  background: color-mix(in srgb, var(--m-ink) 18%, transparent);
  cursor: pointer;
}
.au-drawer__panel {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  width: 470px;
  max-width: 100%;
  overflow-y: auto;
  scrollbar-width: none;
  display: flex;
  flex-direction: column;
  gap: 18px;
  padding: 24px;
  background: var(--el-bg-color);
  border-left: 1px solid var(--el-border-color-light);
  box-shadow: var(--app-shadow-brand);
}
.au-drawer__panel::-webkit-scrollbar {
  display: none;
}
.au-drawer__head {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}
.au-drawer__icon {
  flex: none;
  width: 48px;
  height: 48px;
  border-radius: 16px;
  transform: rotate(-4deg);
}
.au-drawer__icon :deep(svg) {
  width: 24px;
  height: 24px;
}
.au-drawer__title {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.au-drawer__kicker {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.au-drawer__title h2 {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 1.375rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.au-drawer__target {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  word-break: break-all;
}
.au-result {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  border-radius: var(--app-radius);
}
.au-result.is-ok {
  background: var(--m-okSoft);
  color: var(--m-ok);
}
.au-result.is-bad {
  background: var(--m-badSoft);
  color: var(--m-bad);
}
.au-result :deep(.status-pill) {
  background: var(--el-bg-color) !important;
}
.au-result__msg {
  flex: 1;
  min-width: 0;
  font-size: 0.8125rem;
  font-weight: 600;
  word-break: break-word;
}
.au-result__dur {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.au-label {
  margin: 0;
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--el-text-color-secondary);
}
.au-chain {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.au-chain__row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.au-chain__box {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 12px;
  border-radius: var(--app-radius);
  border: 1.5px solid var(--el-border-color-light);
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.au-chain__box b {
  font-size: 0.875rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.au-chain__box.is-node {
  border-color: transparent;
  background: var(--m-a2Soft);
}
.au-chain__arrow {
  flex: none;
  width: 32px;
  height: 32px;
  border-radius: 50%;
}
.au-chain__arrow :deep(svg) {
  width: 16px;
  height: 16px;
}
.au-dl {
  margin: 0;
  padding: 14px 16px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px 16px;
}
.au-dl > div {
  min-width: 0;
}
.au-dl .is-full {
  grid-column: 1 / -1;
}
.au-dl dt {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.au-dl dd {
  margin: 2px 0 0;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  word-break: break-all;
}
.au-json {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.au-json__head {
  display: flex;
  align-items: center;
  gap: 8px;
}
.au-json__head .au-label {
  flex: 1;
}
.au-tag {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 8px;
  border-radius: var(--app-radius-xs);
  background: var(--el-fill-color-light);
  font-size: 0.75rem;
  font-weight: 700;
}
.au-json pre {
  margin: 0;
  padding: 14px 16px;
  max-height: 320px;
  overflow: auto;
  scrollbar-width: none;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-light);
  font-size: 0.75rem;
  line-height: 1.65;
  color: var(--el-text-color-primary);
  white-space: pre;
}
.au-json pre::-webkit-scrollbar {
  display: none;
}
.au-drawer__foot {
  display: flex;
  gap: 10px;
  margin-top: auto;
  padding-top: 4px;
}
.au-drawer__foot > * {
  flex: 1;
  min-height: 44px;
}
.au-drawer-enter-active,
.au-drawer-leave-active {
  transition: opacity 0.2s;
}
.au-drawer-enter-active .au-drawer__panel,
.au-drawer-leave-active .au-drawer__panel {
  transition: transform 0.22s cubic-bezier(0.4, 0, 0.2, 1);
}
.au-drawer-enter-from,
.au-drawer-leave-to {
  opacity: 0;
}
.au-drawer-enter-from .au-drawer__panel,
.au-drawer-leave-to .au-drawer__panel {
  transform: translateX(24px);
}

@media (width <= 1100px) {
  .au-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
@media (width <= 768px) {
  .au-stats {
    gap: 10px;
  }
  .au-stat {
    padding: 12px;
    gap: 10px;
  }
  .au-stat__icon {
    width: 38px;
    height: 38px;
    border-radius: 13px;
  }
  .au-stat__value {
    font-size: 1.375rem;
  }
  .au-filters {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
  }
  .au-filters__sel--wide {
    grid-column: 1 / -1;
  }
  .au-drawer__panel {
    width: 100%;
    padding: 20px 16px calc(20px + env(safe-area-inset-bottom, 0px));
  }
  .au-drawer__foot {
    flex-direction: column;
  }
}
</style>
