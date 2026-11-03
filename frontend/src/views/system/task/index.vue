<!-- 任务与日志 · 后台任务（第二期）：标题 + 计数说明 + 自动刷新/清理已完成 → LogsSubNav → 状态 chips(带数) + 类型筛选 + 搜索
     → PC 可展开行表格（贴纸图标、进度条、行内 详情/取消/重试/删除 小图标）/ 手机主色 hero + 卡片；多选后底部深色批量条。
     保留 listTasks / cancelTask / retryTask / deleteTask 契约与 2.5s 静默轮询；新增 clearTasks、状态/类型统计（后端 counts/types）。 -->
<template>
  <div class="tk-page">
    <PageHeader :title="t('system.logsNav.label')">
      <template #description>
        <span class="m-num">{{ t('system.tasks.summary', { total: counts.all }) }}</span>
        <template v-if="counts.running">
          · <b class="tk-tone-p">{{ t('system.tasks.summaryRunning', { n: counts.running }) }}</b>
        </template>
        <template v-if="counts.failed">
          · <b class="tk-tone-bad">{{ t('system.tasks.summaryFailed', { n: counts.failed }) }}</b>
        </template>
      </template>
      <template #actions>
        <UButton
          :color="autoRefresh ? 'primary' : 'neutral'"
          :variant="autoRefresh ? 'soft' : 'outline'"
          icon="i-lucide-refresh-cw"
          :aria-pressed="autoRefresh"
          @click="toggleAuto"
        >
          {{ autoRefresh ? t('system.tasks.autoOn') : t('system.tasks.autoOff') }}
        </UButton>
        <UButton
          color="neutral"
          variant="outline"
          icon="i-lucide-trash-2"
          :disabled="!finishedCount"
          @click="confirmClear"
        >
          {{ t('system.tasks.clearFinished') }}
        </UButton>
      </template>
    </PageHeader>

    <LogsSubNav :counts="navCounts" />

    <!-- 手机：主色 hero -->
    <section v-if="menuStore.isMobile" class="tk-hero">
      <span class="tk-hero__sun" aria-hidden="true" />
      <span class="tk-hero__ring" aria-hidden="true" />
      <div class="tk-hero__body">
        <span class="tk-hero__label">{{ t('system.tasks.heroLabel') }}</span>
        <span class="tk-hero__line">
          <b class="tk-hero__num m-num">{{ counts.running }}</b>
          <span>{{ t('system.tasks.heroUnit', { pending: counts.pending }) }}</span>
        </span>
        <span class="tk-hero__hint">{{ t('system.tasks.heroHint') }}</span>
      </div>
    </section>

    <ListToolbar
      v-model:status="statusFilter"
      v-model:keyword="keywords"
      :chips="statusChips"
      :chips-label="t('taskManager.status')"
      :placeholder="t('taskManager.searchPlaceholder')"
      @search="reload"
    >
      <div class="tk-type">
        <AppSelect v-model="typeFilter" :options="typeOptions" fit :aria-label="t('taskManager.type')" />
      </div>
    </ListToolbar>

    <!-- PC：可展开行表格 -->
    <div v-if="!menuStore.isMobile" class="tk-table">
      <div class="tk-row tk-row--head" role="row">
        <span class="tk-cell tk-cell--check">
          <input
            type="checkbox"
            class="tk-cb"
            :checked="allSelected"
            :indeterminate.prop="someSelected"
            :aria-label="t('system.tasks.selectAll')"
            @change="toggleAll"
          />
        </span>
        <span>{{ t('taskManager.title') }}</span>
        <span>{{ t('taskManager.status') }}</span>
        <span>{{ t('taskManager.creator') }}</span>
        <span>{{ t('system.tasks.startTime') }}</span>
        <span>{{ t('system.tasks.duration') }}</span>
        <span class="tk-cell--end">{{ t('system.common.operation') }}</span>
      </div>

      <div v-if="loading && !items.length" class="tk-skel-list">
        <div v-for="i in 5" :key="i" class="tk-skeleton" />
      </div>
      <EmptyState
        v-else-if="!items.length"
        icon="HOutline:QueueListIcon"
        :title="t('taskManager.empty')"
        :description="t('taskManager.emptyDesc')"
      />

      <div
        v-for="row in items"
        v-else
        :key="row.id"
        class="tk-item"
        :class="{ 'is-open': openId === row.id, 'is-sel': selected.has(row.id) }"
      >
        <div class="tk-row" role="row">
          <span class="tk-cell tk-cell--check">
            <input
              type="checkbox"
              class="tk-cb"
              :checked="selected.has(row.id)"
              :aria-label="t('system.tasks.selectOne', { name: taskTitle(row) })"
              @change="toggleSel(row.id)"
            />
          </span>
          <span class="tk-name">
            <span class="m-sticker tk-sticker" :class="`m-sticker--${kindOf(row).tone}`" :style="{ transform: `rotate(${kindOf(row).rot})` }">
              <component :is="menuStore.iconComponents[kindOf(row).icon]" />
            </span>
            <span class="tk-name__text">
              <span class="tk-name__title">{{ taskTitle(row) }}</span>
              <span class="tk-name__sub m-mono">{{ shortId(row.id) }} · {{ typeLabel(row.type) }}</span>
            </span>
          </span>
          <span class="tk-state">
            <span class="tk-state__line">
              <StatusPill :variant="statusVariant(row.status)" :label="statusLabel(row.status)" />
              <b v-if="showPct(row)" class="tk-state__pct m-num">{{ percent(row) }}%</b>
            </span>
            <span
              v-if="showPct(row)"
              class="tk-bar"
              role="progressbar"
              :aria-valuenow="percent(row)"
              aria-valuemin="0"
              aria-valuemax="100"
              :aria-label="taskTitle(row)"
            >
              <span :class="`is-${progressTone(row.status)}`" :style="{ width: `${percent(row)}%` }" />
            </span>
            <span v-if="subText(row)" class="tk-state__sub" :class="{ 'is-bad': !!row.error }">{{ subText(row) }}</span>
          </span>
          <span class="tk-by">
            <span class="tk-avatar" :class="row.userName ? 'm-sticker--a' : 'm-sticker--ok'">{{ initial(row) }}</span>
            <span class="tk-by__name">{{ row.userName || t('taskManager.system') }}</span>
          </span>
          <span class="tk-time m-num">{{ formatStart(row.createTime) }}</span>
          <span class="tk-dur m-num">{{ durationText(row) }}</span>
          <span class="tk-acts">
            <UButton
              size="sm"
              :color="openId === row.id ? 'primary' : 'neutral'"
              :variant="openId === row.id ? 'soft' : 'ghost'"
              :trailing-icon="openId === row.id ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
              :aria-expanded="openId === row.id"
              @click="toggleOpen(row.id)"
            >
              {{ openId === row.id ? t('system.tasks.collapse') : t('system.tasks.details') }}
            </UButton>
            <AppIconButton
              v-if="isActive(row.status)"
              icon="HOutline:XMarkIcon"
              class="is-stop"
              :label="t('taskManager.cancel')"
              :box="36"
              @click="cancel(row.id)"
            />
            <AppIconButton
              v-if="row.status === TaskStatus.TASK_STATUS_FAILED"
              icon="HOutline:ArrowPathIcon"
              class="is-retry"
              :label="t('taskManager.retry')"
              :box="36"
              @click="retry(row.id)"
            />
            <AppIconButton
              v-if="isTerminal(row.status)"
              icon="HOutline:TrashIcon"
              :label="t('system.common.delete')"
              :box="36"
              @click="confirmDelete(row.id)"
            />
          </span>
        </div>

        <div v-if="openId === row.id" class="tk-detail">
          <section class="tk-detail__card">
            <div class="tk-detail__head">
              <h3>{{ t('system.tasks.results') }}</h3>
              <span class="m-num">{{ resultsDone(row) }} / {{ row.results.length }}</span>
            </div>
            <ol v-if="row.results.length" class="tk-steps">
              <li v-for="(r, idx) in row.results.slice(0, 12)" :key="idx" class="tk-step">
                <span class="tk-step__dot" :class="r.success ? 'is-ok' : 'is-bad'">
                  <component :is="menuStore.iconComponents[r.success ? 'HOutline:CheckIcon' : 'HOutline:XMarkIcon']" />
                </span>
                <span class="tk-step__text">
                  <span class="tk-step__path m-mono">{{ r.path || '—' }}</span>
                  <span v-if="r.message" class="tk-step__msg" :class="{ 'is-bad': !r.success }">{{ r.message }}</span>
                </span>
              </li>
              <li v-if="row.results.length > 12" class="tk-step__more">
                {{ t('system.tasks.moreResults', { n: row.results.length - 12 }) }}
              </li>
            </ol>
            <p v-else class="tk-detail__empty">{{ t('system.tasks.noResults') }}</p>
            <dl class="tk-dl">
              <div><dt>{{ t('taskManager.kind') }}</dt><dd>{{ kindLabel(row.kind) }}</dd></div>
              <div><dt>{{ t('taskManager.type') }}</dt><dd class="m-mono">{{ row.type }}</dd></div>
              <div><dt>{{ t('taskManager.progress') }}</dt><dd class="m-num">{{ progressText(row) }}</dd></div>
              <div><dt>{{ t('system.tasks.endTime') }}</dt><dd class="m-num">{{ formatFull(row.endTime) }}</dd></div>
            </dl>
          </section>
          <section class="tk-log" :aria-label="t('system.tasks.output')">
            <div class="tk-log__head">
              <component :is="menuStore.iconComponents['HOutline:CommandLineIcon']" />
              <span class="tk-log__title">{{ t('system.tasks.output') }}</span>
              <span class="tk-log__count m-mono">{{ t('system.tasks.lines', { n: logLines(row).length }) }}</span>
              <AppIconButton
                icon="HOutline:ClipboardDocumentIcon"
                :label="t('system.common.copy')"
                :box="32"
                @click="copyLog(row)"
              />
            </div>
            <div class="tk-log__body m-mono">
              <div v-for="(l, idx) in logLines(row)" :key="idx" class="tk-log__line">
                <span class="tk-log__tm">{{ l.tm }}</span>
                <span :class="`is-${l.tone}`">{{ l.text }}</span>
              </div>
            </div>
          </section>
        </div>
      </div>
    </div>

    <!-- 手机：卡片 -->
    <template v-else>
      <div v-if="loading && !items.length" class="tk-cards">
        <div v-for="i in 4" :key="i" class="tk-skeleton tk-skeleton--card" />
      </div>
      <EmptyState
        v-else-if="!items.length"
        icon="HOutline:QueueListIcon"
        :title="t('taskManager.empty')"
        :description="t('taskManager.emptyDesc')"
      />
      <div v-else class="tk-cards">
        <article v-for="row in items" :key="row.id" class="tk-card" :class="{ 'is-sel': selected.has(row.id) }">
          <header class="tk-card__head">
            <span class="m-sticker tk-sticker" :class="`m-sticker--${kindOf(row).tone}`" :style="{ transform: `rotate(${kindOf(row).rot})` }">
              <component :is="menuStore.iconComponents[kindOf(row).icon]" />
            </span>
            <span class="tk-name__text">
              <span class="tk-name__title">{{ taskTitle(row) }}</span>
              <span class="tk-name__sub m-mono">{{ typeLabel(row.type) }}</span>
            </span>
            <StatusPill :variant="statusVariant(row.status)" :label="statusLabel(row.status)" />
          </header>
          <div v-if="showPct(row)" class="tk-card__prog">
            <span class="tk-bar">
              <span :class="`is-${progressTone(row.status)}`" :style="{ width: `${percent(row)}%` }" />
            </span>
            <b class="m-num">{{ percent(row) }}%</b>
          </div>
          <span v-if="subText(row)" class="tk-state__sub" :class="{ 'is-bad': !!row.error }">{{ subText(row) }}</span>
          <dl class="tk-dl tk-dl--card">
            <div><dt>{{ t('taskManager.creator') }}</dt><dd>{{ row.userName || t('taskManager.system') }}</dd></div>
            <div><dt>{{ t('taskManager.kind') }}</dt><dd>{{ kindLabel(row.kind) }}</dd></div>
            <div><dt>{{ t('system.tasks.startTime') }}</dt><dd class="m-num">{{ formatStart(row.createTime) }}</dd></div>
            <div><dt>{{ t('system.tasks.duration') }}</dt><dd class="m-num">{{ durationText(row) }}</dd></div>
          </dl>
          <div v-if="openId === row.id" class="tk-log tk-log--card m-mono">
            <div v-for="(l, idx) in logLines(row)" :key="idx" class="tk-log__line">
              <span class="tk-log__tm">{{ l.tm }}</span>
              <span :class="`is-${l.tone}`">{{ l.text }}</span>
            </div>
          </div>
          <footer class="tk-card__foot">
            <label class="tk-card__check">
              <input type="checkbox" class="tk-cb" :checked="selected.has(row.id)" @change="toggleSel(row.id)" />
              <span class="m-mono">{{ shortId(row.id) }}</span>
            </label>
            <AppIconButton
              :icon="openId === row.id ? 'HOutline:ChevronUpIcon' : 'HOutline:ChevronDownIcon'"
              :label="openId === row.id ? t('system.tasks.collapse') : t('system.tasks.details')"
              :active="openId === row.id"
              @click="toggleOpen(row.id)"
            />
            <AppIconButton
              v-if="isActive(row.status)"
              icon="HOutline:XMarkIcon"
              class="is-stop"
              :label="t('taskManager.cancel')"
              @click="cancel(row.id)"
            />
            <AppIconButton
              v-if="row.status === TaskStatus.TASK_STATUS_FAILED"
              icon="HOutline:ArrowPathIcon"
              class="is-retry"
              :label="t('taskManager.retry')"
              @click="retry(row.id)"
            />
            <AppIconButton
              v-if="isTerminal(row.status)"
              icon="HOutline:TrashIcon"
              :label="t('system.common.delete')"
              @click="confirmDelete(row.id)"
            />
          </footer>
        </article>
      </div>
    </template>

    <Pagination
      v-model:page="page"
      v-model:page-size="pageSize"
      :total="total"
      @change="() => getList()"
    />

    <!-- 批量操作条 -->
    <Transition name="tk-bulk">
      <div v-if="selected.size" class="tk-bulk" role="toolbar" :aria-label="t('system.tasks.bulk')">
        <span class="tk-bulk__count">{{ t('system.tasks.selected', { n: selected.size }) }}</span>
        <button type="button" class="tk-bulk__btn" :disabled="!selRetryable.length" @click="bulkRetry">
          <component :is="menuStore.iconComponents['HOutline:ArrowPathIcon']" />
          <span>{{ t('taskManager.retry') }}</span>
        </button>
        <button type="button" class="tk-bulk__btn" :disabled="!selCancelable.length" @click="bulkCancel">
          <component :is="menuStore.iconComponents['HOutline:XMarkIcon']" />
          <span>{{ t('taskManager.cancel') }}</span>
        </button>
        <button type="button" class="tk-bulk__btn is-warn" :disabled="!selDeletable.length" @click="bulkDelete">
          <component :is="menuStore.iconComponents['HOutline:TrashIcon']" />
          <span>{{ t('system.tasks.deleteRecords') }}</span>
        </button>
        <button type="button" class="tk-bulk__btn tk-bulk__x" :aria-label="t('system.tasks.clearSelection')" @click="selected.clear()">
          <component :is="menuStore.iconComponents['HOutline:XMarkIcon']" />
        </button>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import { useClipboard } from '@vueuse/core'
import { showRequestError } from '@/utils/request'
import { Dialog } from '@/utils/dialog'
import { listTasks, cancelTask, retryTask, deleteTask, clearTasks } from '@/api/task'
import { getOperationLogStats } from '@/api/system'
import { PERM } from '@/config/permission'
import { TaskStatus, TaskKind, type TaskInfo, type TaskStatusCounts, type TaskTypeCount } from '@/types/v1/task'
import type { ToolbarChip } from '@/components/ui/ListToolbar.vue'
import LogsSubNav from '@/views/system/components/LogsSubNav.vue'

defineOptions({ name: 'TaskManagerView' })

const menuStore = useMenuStore()
const { t } = useI18n()
const { copy } = useClipboard()

const REFRESH_INTERVAL_MS = 2500

const loading = ref(false)
const items = ref<TaskInfo[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keywords = ref('')
const statusFilter = ref('')
const typeFilter = ref('')
const autoRefresh = ref(true)
const openId = ref<string | null>(null)
const selected = reactive(new Set<string>())
const counts = ref<TaskStatusCounts>({ all: 0, pending: 0, running: 0, success: 0, failed: 0, canceled: 0 })
const types = ref<TaskTypeCount[]>([])
const opTotal = ref<number | null>(null)
let timer: ReturnType<typeof setInterval> | undefined

const finishedCount = computed(() => counts.value.success + counts.value.failed + counts.value.canceled)
const navCounts = computed(() => ({ task: counts.value.all, operation: opTotal.value }))

// 状态值沿用后端小写过滤契约
const statusChips = computed<ToolbarChip<string>[]>(() => [
  { label: t('system.common.all'), value: '', count: counts.value.all },
  { label: t('taskManager.statusRunning'), value: 'running', count: counts.value.running },
  { label: t('taskManager.statusPending'), value: 'pending', count: counts.value.pending },
  { label: t('taskManager.statusSuccess'), value: 'success', count: counts.value.success },
  { label: t('taskManager.statusFailed'), value: 'failed', count: counts.value.failed },
  { label: t('taskManager.statusCanceled'), value: 'canceled', count: counts.value.canceled },
])

// —— 任务类型：展示名 + 贴纸图标 ——
const TYPE_META: Record<string, { key: string; icon: string; tone: string }> = {
  'docker.image_pull': { key: 'imagePull', icon: 'HOutline:ArrowDownTrayIcon', tone: 'a2' },
  'docker.container_recreate': { key: 'containerRecreate', icon: 'HOutline:CubeIcon', tone: 'ok' },
  'docker.network_recreate': { key: 'networkRecreate', icon: 'HOutline:ShareIcon', tone: 'a2' },
  'file.upload_finalize': { key: 'uploadFinalize', icon: 'HOutline:ArrowUpTrayIcon', tone: 'p' },
  'file.upload_gc': { key: 'uploadGc', icon: 'HOutline:SparklesIcon', tone: 'warn' },
  'file.copy': { key: 'fileCopy', icon: 'HOutline:DocumentDuplicateIcon', tone: 'a' },
  'file.move': { key: 'fileMove', icon: 'HOutline:FolderArrowDownIcon', tone: 'a' },
}
const ROTS = ['-4deg', '3deg', '-2deg', '4deg', '-3deg']
const hashRot = (s: string) => ROTS[[...s].reduce((a, c) => a + c.charCodeAt(0), 0) % ROTS.length] ?? '0deg'

const kindOf = (row: TaskInfo) => {
  const meta = TYPE_META[row.type]
  if (meta) return { ...meta, rot: hashRot(row.id) }
  const tone = row.type.startsWith('docker.') ? 'a2' : row.type.startsWith('file.') ? 'a' : 'p'
  return { key: '', icon: 'HOutline:CogIcon', tone, rot: hashRot(row.id) }
}
const typeLabel = (type: string) => {
  const meta = TYPE_META[type]
  return meta ? t(`system.tasks.types.${meta.key}`) : type
}
const typeOptions = computed(() => [
  { label: t('system.tasks.allTypes'), value: '' },
  ...types.value.map((ty) => ({ label: `${typeLabel(ty.type)} · ${ty.count}`, value: ty.type })),
])

const taskTitle = (row: TaskInfo) => row.title || typeLabel(row.type)
const shortId = (id: string) => (id.length > 12 ? id.slice(0, 8) : id)
const initial = (row: TaskInfo) => (row.userName ? [...row.userName][0]! : t('taskManager.system').slice(0, 1)).toUpperCase()

const kindLabel = (kind: TaskKind) => {
  switch (kind) {
    case TaskKind.TASK_KIND_ONESHOT:
      return t('taskManager.kindOneshot')
    case TaskKind.TASK_KIND_SCHEDULED:
      return t('taskManager.kindScheduled')
    case TaskKind.TASK_KIND_DAEMON:
      return t('taskManager.kindDaemon')
    default:
      return String(kind)
  }
}

const statusLabel = (status: TaskStatus) => {
  switch (status) {
    case TaskStatus.TASK_STATUS_PENDING:
      return t('taskManager.statusPending')
    case TaskStatus.TASK_STATUS_RUNNING:
      return t('taskManager.statusRunning')
    case TaskStatus.TASK_STATUS_SUCCESS:
      return t('taskManager.statusSuccess')
    case TaskStatus.TASK_STATUS_FAILED:
      return t('taskManager.statusFailed')
    case TaskStatus.TASK_STATUS_CANCELED:
      return t('taskManager.statusCanceled')
    default:
      return String(status)
  }
}

const statusVariant = (status: TaskStatus) => {
  switch (status) {
    case TaskStatus.TASK_STATUS_RUNNING:
      return 'primary' as const
    case TaskStatus.TASK_STATUS_SUCCESS:
      return 'success' as const
    case TaskStatus.TASK_STATUS_FAILED:
      return 'error' as const
    case TaskStatus.TASK_STATUS_PENDING:
      return 'warning' as const
    default:
      return 'neutral' as const
  }
}

const progressTone = (status: TaskStatus) => {
  if (status === TaskStatus.TASK_STATUS_SUCCESS) return 'ok'
  if (status === TaskStatus.TASK_STATUS_FAILED) return 'bad'
  return 'p'
}

const percent = (row: TaskInfo) => {
  const totalNum = Number(row.total) || 0
  if (totalNum <= 0) return 0
  return Math.min(100, Math.round((Number(row.finished) / totalNum) * 100))
}
const isActive = (status: TaskStatus) =>
  status === TaskStatus.TASK_STATUS_RUNNING || status === TaskStatus.TASK_STATUS_PENDING
const isTerminal = (status: TaskStatus) =>
  status === TaskStatus.TASK_STATUS_SUCCESS ||
  status === TaskStatus.TASK_STATUS_FAILED ||
  status === TaskStatus.TASK_STATUS_CANCELED
const showPct = (row: TaskInfo) => Number(row.total) > 0 && row.status === TaskStatus.TASK_STATUS_RUNNING
const subText = (row: TaskInfo) => row.error || row.message || ''
const progressText = (row: TaskInfo) =>
  Number(row.total) > 0 ? `${Number(row.finished) || 0} / ${Number(row.total)}` : String(Number(row.finished) || 0)
const resultsDone = (row: TaskInfo) => row.results.filter((r) => r.success).length

// —— 时间 ——
const now = ref(Date.now())
const formatStart = (v?: Date) => {
  if (!v) return '—'
  const d = dayjs(v)
  const today = dayjs().startOf('day')
  if (d.isAfter(today)) return `${t('system.tasks.today')} ${d.format('HH:mm:ss')}`
  if (d.isAfter(today.subtract(1, 'day'))) return `${t('system.tasks.yesterday')} ${d.format('HH:mm:ss')}`
  return d.format('MM-DD HH:mm')
}
const formatFull = (v?: Date) => (v ? dayjs(v).format('YYYY-MM-DD HH:mm:ss') : '—')
const fmtDuration = (ms: number) => {
  const s = Math.max(0, Math.round(ms / 1000))
  if (s < 60) return t('system.tasks.durSec', { s })
  const m = Math.floor(s / 60)
  if (m < 60) return t('system.tasks.durMin', { m, s: s % 60 })
  return t('system.tasks.durHour', { h: Math.floor(m / 60), m: m % 60 })
}
const durationText = (row: TaskInfo) => {
  if (!row.createTime) return '—'
  if (row.status === TaskStatus.TASK_STATUS_PENDING) return '—'
  const end = row.endTime ? dayjs(row.endTime).valueOf() : isActive(row.status) ? now.value : null
  if (end == null) return '—'
  return fmtDuration(end - dayjs(row.createTime).valueOf())
}

// —— 任务输出（由消息、结果明细与错误拼成时间线）——
interface LogLine {
  tm: string
  text: string
  tone: 'ink' | 'dim' | 'ok' | 'bad' | 'warn'
}
const logLines = (row: TaskInfo): LogLine[] => {
  const start = row.createTime ? dayjs(row.createTime).format('HH:mm:ss') : '--:--:--'
  const end = row.endTime ? dayjs(row.endTime).format('HH:mm:ss') : ''
  const lines: LogLine[] = [{ tm: start, text: t('system.tasks.logStart', { name: taskTitle(row) }), tone: 'ink' }]
  if (Number(row.total) > 0) lines.push({ tm: start, text: t('system.tasks.logProgress', { p: progressText(row) }), tone: 'dim' })
  for (const r of row.results) {
    lines.push({ tm: end || start, text: `${r.success ? '✓' : '✗'} ${r.path}${r.message ? ` · ${r.message}` : ''}`, tone: r.success ? 'ok' : 'bad' })
  }
  if (row.message) lines.push({ tm: end || start, text: row.message, tone: 'dim' })
  if (row.error) lines.push({ tm: end || start, text: row.error, tone: 'bad' })
  if (end) {
    const tone = row.status === TaskStatus.TASK_STATUS_SUCCESS ? 'ok' : row.status === TaskStatus.TASK_STATUS_FAILED ? 'bad' : 'warn'
    lines.push({ tm: end, text: t('system.tasks.logEnd', { status: statusLabel(row.status) }), tone })
  }
  return lines
}
const copyLog = async (row: TaskInfo) => {
  await copy(logLines(row).map((l) => `${l.tm}  ${l.text}`).join('\n'))
  feedback.success(t('system.operation.copied'))
}

// —— 选择 ——
const allSelected = computed(() => items.value.length > 0 && items.value.every((r) => selected.has(r.id)))
const someSelected = computed(() => !allSelected.value && items.value.some((r) => selected.has(r.id)))
const toggleAll = () => {
  if (allSelected.value) items.value.forEach((r) => selected.delete(r.id))
  else items.value.forEach((r) => selected.add(r.id))
}
const toggleSel = (id: string) => {
  if (selected.has(id)) selected.delete(id)
  else selected.add(id)
}
const selRows = computed(() => items.value.filter((r) => selected.has(r.id)))
const selRetryable = computed(() => selRows.value.filter((r) => r.status === TaskStatus.TASK_STATUS_FAILED))
const selCancelable = computed(() => selRows.value.filter((r) => isActive(r.status)))
const selDeletable = computed(() => selRows.value.filter((r) => isTerminal(r.status)))

const toggleOpen = (id: string) => {
  openId.value = openId.value === id ? null : id
}

const getList = async (silent = false) => {
  if (!silent) loading.value = true
  try {
    const { data } = await listTasks({
      page: page.value,
      pageSize: pageSize.value,
      keywords: keywords.value || undefined,
      status: statusFilter.value || undefined,
      type: typeFilter.value || undefined,
    })
    items.value = data.items ?? []
    total.value = Number(data.total) || 0
    // int64 经 JSON 为字符串，统一转数字
    const c = data.counts
    counts.value = {
      all: Number(c?.all) || 0,
      pending: Number(c?.pending) || 0,
      running: Number(c?.running) || 0,
      success: Number(c?.success) || 0,
      failed: Number(c?.failed) || 0,
      canceled: Number(c?.canceled) || 0,
    }
    types.value = (data.types ?? []).map((ty) => ({ type: ty.type, count: Number(ty.count) || 0 }))
    now.value = Date.now()
    // 已不在当前页的选中项剔除
    const ids = new Set(items.value.map((r) => r.id))
    for (const id of [...selected]) if (!ids.has(id)) selected.delete(id)
  } catch (error) {
    if (!silent) showRequestError(error, t('taskManager.loadFailed'))
  } finally {
    if (!silent) loading.value = false
  }
}

const reload = () => {
  page.value = 1
  getList()
}
watch(typeFilter, reload)

const startTimer = () => {
  stopTimer()
  timer = setInterval(() => getList(true), REFRESH_INTERVAL_MS)
}
const stopTimer = () => {
  if (timer) clearInterval(timer)
  timer = undefined
}
const toggleAuto = () => {
  autoRefresh.value = !autoRefresh.value
  if (autoRefresh.value) {
    getList(true)
    startTimer()
  } else stopTimer()
}

const cancel = async (id: string) => {
  try {
    await cancelTask(id)
    feedback.success(t('taskManager.cancelSuccess'))
    getList(true)
  } catch (error) {
    showRequestError(error, t('taskManager.actionFailed'))
  }
}

const retry = async (id: string) => {
  try {
    await retryTask(id)
    feedback.success(t('taskManager.retrySuccess'))
    getList(true)
  } catch (error) {
    showRequestError(error, t('taskManager.actionFailed'))
  }
}

const confirmDelete = (id: string) => {
  Dialog.info({
    showCancelButton: true,
    content: t('taskManager.confirmDelete'),
    confirmText: t('system.common.delete'),
    cancelText: t('system.common.cancel'),
    onConfirm: async () => {
      try {
        await deleteTask(id)
        feedback.success(t('taskManager.deleteSuccess'))
        selected.delete(id)
        getList(true)
      } catch (error) {
        showRequestError(error, t('taskManager.actionFailed'))
      }
    },
  })
}

const confirmClear = () => {
  Dialog.info({
    showCancelButton: true,
    content: t('system.tasks.clearConfirm', { n: finishedCount.value }),
    confirmText: t('system.tasks.clearFinished'),
    cancelText: t('system.common.cancel'),
    onConfirm: async () => {
      try {
        const { data } = await clearTasks({})
        feedback.success(t('system.tasks.clearSuccess', { n: Number(data?.deleted) || 0 }))
        selected.clear()
        reload()
      } catch (error) {
        showRequestError(error, t('taskManager.actionFailed'))
      }
    },
  })
}

// 批量：逐个调用单条接口，汇总结果
const runBulk = async (rows: TaskInfo[], fn: (id: string) => Promise<unknown>) => {
  let ok = 0
  for (const r of rows) {
    try {
      await fn(r.id)
      ok++
    } catch {
      /* 单条失败已由请求层提示 */
    }
  }
  feedback.success(t('system.tasks.bulkDone', { ok, all: rows.length }))
  selected.clear()
  getList(true)
}
const bulkRetry = () => runBulk(selRetryable.value, retryTask)
const bulkCancel = () => runBulk(selCancelable.value, cancelTask)
const bulkDelete = () => {
  const rows = selDeletable.value
  Dialog.info({
    showCancelButton: true,
    content: t('system.tasks.bulkDeleteConfirm', { n: rows.length }),
    confirmText: t('system.common.delete'),
    cancelText: t('system.common.cancel'),
    onConfirm: () => runBulk(rows, deleteTask),
  })
}

const loadOpTotal = async () => {
  if (!menuStore.hasButtonPermission(PERM.SYSTEM_CONFIG_VIEW)) return
  try {
    const { data } = await getOperationLogStats()
    opTotal.value = Number(data?.total) || 0
  } catch {
    opTotal.value = null
  }
}

onMounted(() => {
  getList()
  loadOpTotal()
  startTimer()
})
// 页面被缓存（切到同组其它页）时暂停轮询；回来时静默刷新一次再继续
onActivated(() => {
  if (timer) return
  void getList(true)
  startTimer()
})
onDeactivated(stopTimer)
onBeforeUnmount(stopTimer)
</script>

<style scoped lang="scss">
.tk-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.tk-tone-p {
  color: var(--m-pInk);
  font-weight: 700;
}
.tk-tone-bad {
  color: var(--m-bad);
  font-weight: 700;
}
.tk-type {
  flex: 0 0 auto;
  min-width: 150px;
}
.tk-type :deep(.app-select__trigger),
.tk-type :deep(button) {
  min-height: 40px;
  border-radius: var(--app-radius-pill);
}

/* —— hero（手机）—— */
.tk-hero {
  position: relative;
  overflow: hidden;
  padding: 18px;
  border-radius: var(--app-radius-lg);
  background: var(--m-p);
  color: var(--m-pOn);
}
.tk-hero__sun {
  position: absolute;
  right: -44px;
  top: -54px;
  width: 140px;
  height: 140px;
  border-radius: 50%;
  background: var(--m-a);
}
.tk-hero__ring {
  position: absolute;
  right: 78px;
  bottom: -22px;
  width: 50px;
  height: 50px;
  border-radius: 50%;
  border: 9px solid var(--m-a2);
}
.tk-hero__body {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.tk-hero__label {
  font-size: 0.75rem;
  font-weight: 700;
  opacity: 0.9;
}
.tk-hero__line {
  display: flex;
  align-items: baseline;
  gap: 8px;
  font-size: 0.9375rem;
  font-weight: 700;
}
.tk-hero__num {
  font-family: var(--app-font-display);
  font-size: 2.5rem;
  line-height: 1;
  font-weight: 800;
}
.tk-hero__hint {
  font-size: 0.8125rem;
  opacity: 0.92;
}

/* —— 表格 —— */
.tk-table {
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
  padding: 4px 8px;
  overflow-x: auto;
  scrollbar-width: none;
}
.tk-table::-webkit-scrollbar {
  display: none;
}
.tk-row {
  display: grid;
  grid-template-columns: 36px minmax(240px, 2.2fr) minmax(190px, 1.6fr) minmax(130px, 1fr) 128px 96px minmax(176px, auto);
  align-items: center;
  gap: 12px;
  min-width: 1040px;
  padding: 12px;
}
.tk-row--head {
  padding-top: 10px;
  padding-bottom: 10px;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.tk-item {
  border-radius: var(--app-radius);
  transition: background 0.15s;
}
.tk-item + .tk-item {
  border-top: 1px solid var(--el-border-color-lighter);
}
.tk-item.is-sel {
  background: color-mix(in srgb, var(--m-pSoft) 55%, transparent);
}
.tk-item.is-open {
  background: var(--el-fill-color-light);
  border-top-color: transparent;
}
.tk-item.is-open + .tk-item {
  border-top-color: transparent;
}
.tk-cell--check {
  display: flex;
  align-items: center;
  justify-content: center;
}
.tk-cell--end {
  text-align: right;
}
.tk-cb {
  width: 18px;
  height: 18px;
  accent-color: var(--m-p);
  cursor: pointer;
}
.tk-name {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.tk-sticker {
  flex: none;
  width: 40px;
  height: 40px;
  border-radius: 13px;
}
.tk-sticker :deep(svg) {
  width: 20px;
  height: 20px;
}
.tk-name__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.tk-name__title {
  font-size: 0.9375rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.tk-name__sub {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.tk-state {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}
.tk-state__line {
  display: flex;
  align-items: center;
  gap: 10px;
}
.tk-state__pct {
  font-size: 0.875rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.tk-state__sub {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.tk-state__sub.is-bad {
  color: var(--m-bad);
}
.tk-bar {
  display: block;
  height: 8px;
  border-radius: var(--app-radius-pill);
  background: var(--el-fill-color-light);
  overflow: hidden;
}
.tk-item.is-open .tk-bar {
  background: var(--el-bg-color);
}
.tk-bar > span {
  display: block;
  height: 100%;
  border-radius: inherit;
  transition: width 0.3s ease;
}
.tk-bar > .is-p {
  background: var(--m-p);
}
.tk-bar > .is-ok {
  background: var(--m-ok);
}
.tk-bar > .is-bad {
  background: var(--m-bad);
}
.tk-by {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.tk-avatar {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  font-size: 0.75rem;
  font-weight: 800;
}
.tk-by__name {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.tk-time {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.tk-dur {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.tk-acts {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 2px;
}
.tk-acts :deep(.is-stop),
.tk-card__foot :deep(.is-stop) {
  color: var(--m-bad);
}
.tk-acts :deep(.is-stop:hover),
.tk-card__foot :deep(.is-stop:hover) {
  background: var(--m-badSoft);
}
.tk-acts :deep(.is-retry),
.tk-card__foot :deep(.is-retry) {
  color: var(--m-pInk);
}

/* —— 展开详情 —— */
.tk-detail {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  min-width: 1040px;
  padding: 2px 12px 16px 60px;
}
.tk-detail__card {
  flex: 1 1 300px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 18px;
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
}
.tk-detail__head {
  display: flex;
  align-items: center;
  gap: 8px;
}
.tk-detail__head h3 {
  flex: 1;
  margin: 0;
  font-size: 0.9375rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.tk-detail__head span {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.tk-detail__empty {
  margin: 0;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.tk-steps {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 260px;
  overflow-y: auto;
  scrollbar-width: none;
}
.tk-steps::-webkit-scrollbar {
  display: none;
}
.tk-step {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}
.tk-step__dot {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 50%;
}
.tk-step__dot :deep(svg) {
  width: 14px;
  height: 14px;
  stroke-width: 2.6;
}
.tk-step__dot.is-ok {
  background: var(--m-okSoft);
  color: var(--m-ok);
}
.tk-step__dot.is-bad {
  background: var(--m-badSoft);
  color: var(--m-bad);
}
.tk-step__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding-top: 3px;
}
.tk-step__path {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  word-break: break-all;
}
.tk-step__msg {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.tk-step__msg.is-bad {
  color: var(--m-bad);
}
.tk-step__more {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  padding-left: 36px;
}
.tk-dl {
  margin: 0;
  padding: 12px 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 14px;
}
.tk-dl dt {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.tk-dl dd {
  margin: 2px 0 0;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.tk-log {
  flex: 2 1 440px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-radius: var(--app-radius-lg);
  background: var(--m-s2, var(--el-fill-color-light));
  border: 1px solid var(--el-border-color-light);
}
.tk-log__head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px 6px 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  color: var(--el-text-color-primary);
}
.tk-log__head > :deep(svg) {
  width: 16px;
  height: 16px;
}
.tk-log__title {
  flex: 1;
  font-size: 0.8125rem;
  font-weight: 700;
}
.tk-log__count {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.tk-log__body {
  flex: 1;
  max-height: 260px;
  overflow-y: auto;
  padding: 12px 16px 14px;
  font-size: 0.78rem;
  line-height: 1.75;
  scrollbar-width: none;
}
.tk-log__body::-webkit-scrollbar {
  display: none;
}
.tk-log__line {
  white-space: pre-wrap;
  word-break: break-all;
}
.tk-log__tm {
  color: var(--el-text-color-secondary);
  margin-right: 10px;
}
.tk-log .is-ink {
  color: var(--el-text-color-primary);
}
.tk-log .is-dim {
  color: var(--el-text-color-secondary);
}
.tk-log .is-ok {
  color: var(--m-ok);
}
.tk-log .is-bad {
  color: var(--m-bad);
}
.tk-log .is-warn {
  color: var(--m-warn);
}

/* —— 骨架 —— */
.tk-skel-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 8px 0;
}
.tk-skeleton {
  height: 64px;
  border-radius: var(--app-radius);
  background: linear-gradient(100deg, var(--el-fill-color-light) 30%, var(--el-fill-color) 50%, var(--el-fill-color-light) 70%);
  background-size: 200% 100%;
  animation: tk-shimmer 1.4s ease-in-out infinite;
}
.tk-skeleton--card {
  height: 170px;
  border-radius: var(--app-radius-lg);
}
@keyframes tk-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}

/* —— 手机卡片 —— */
.tk-cards {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.tk-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 16px 12px 6px 16px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.tk-card.is-sel {
  border-color: var(--m-p);
}
.tk-card__head {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-right: 4px;
}
.tk-card__prog {
  display: flex;
  align-items: center;
  gap: 10px;
  padding-right: 4px;
}
.tk-card__prog .tk-bar {
  flex: 1;
}
.tk-card__prog b {
  font-size: 0.875rem;
  font-weight: 800;
}
.tk-dl--card {
  margin-right: 4px;
}
.tk-log--card {
  margin-right: 4px;
  padding: 10px 12px;
  font-size: 0.72rem;
  line-height: 1.7;
  max-height: 220px;
  overflow-y: auto;
  scrollbar-width: none;
}
.tk-card__foot {
  display: flex;
  align-items: center;
  gap: 2px;
  padding-top: 6px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.tk-card__check {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 44px;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}

/* —— 批量条 —— */
.tk-bulk {
  position: sticky;
  bottom: 24px;
  z-index: 5;
  align-self: center;
  display: flex;
  align-items: center;
  gap: 4px;
  max-width: 100%;
  padding: 6px 6px 6px 20px;
  border-radius: var(--app-radius-pill);
  background: var(--m-ink);
  color: var(--m-bg);
  box-shadow: var(--app-shadow-brand);
}
.tk-bulk__count {
  margin-right: 10px;
  font-size: 0.875rem;
  font-weight: 700;
  white-space: nowrap;
}
.tk-bulk__btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 0 14px;
  border: none;
  border-radius: var(--app-radius-pill);
  background: transparent;
  color: inherit;
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
  white-space: nowrap;
}
.tk-bulk__btn:hover:not(:disabled) {
  background: color-mix(in srgb, var(--m-bg) 14%, transparent);
}
.tk-bulk__btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
.tk-bulk__btn.is-warn {
  color: var(--m-a);
}
.tk-bulk__btn :deep(svg) {
  width: 16px;
  height: 16px;
}
.tk-bulk__x {
  width: 40px;
  padding: 0;
  justify-content: center;
}
.tk-bulk-enter-active,
.tk-bulk-leave-active {
  transition: opacity 0.18s, transform 0.18s;
}
.tk-bulk-enter-from,
.tk-bulk-leave-to {
  opacity: 0;
  transform: translateY(8px);
}

@media (width <= 768px) {
  .tk-type {
    flex: 1 1 100%;
  }
  .tk-bulk {
    bottom: calc(76px + env(safe-area-inset-bottom, 0px));
    padding-left: 14px;
  }
  .tk-bulk__btn {
    min-height: 44px;
    padding: 0 10px;
  }
  .tk-bulk__btn span {
    display: none;
  }
  .tk-bulk__count {
    margin-right: 4px;
  }
}
</style>
