<!-- 任务与日志 · 后台任务：标题 + 计数说明 + 自动刷新/清理已完成 → LogsSubNav → 状态 chips(带数) + 类型筛选 + 搜索
     → PC 可展开行表格（贴纸图标、id · 节点、进度条、发起人、开始/耗时、行内 详情/取消/重试/删除）
       展开：执行步骤时间线 + 触发方式/节点/预计剩余等信息格 + 深色实时日志（复制/下载）
     → 手机主色 hero + 信息卡片（进度说明、两列信息格、#类型 标签 + 小图标操作）；多选后底部深色批量条。
     数据：listTasks（状态/类型统计、保留天数）+ 展开时 getTask（步骤与日志），2.5s 静默轮询；
     cancelTasks / retryTasks / deleteTasks 批量接口，clearTasks 清理已结束。 -->
<template>
  <div class="tk-page">
    <PageHeader :title="t('system.logsNav.label')">
      <template #description>
        <span class="m-num">{{ t('systemV2.tasks.summary', { total: fmtNum(counts.all) }) }}</span>
        <template v-if="counts.running">
          · <b class="tk-tone-p">{{ t('systemV2.tasks.summaryRunning', { n: fmtNum(counts.running) }) }}</b>
        </template>
        <template v-if="counts.failed">
          · <b class="tk-tone-bad">{{ t('systemV2.tasks.summaryFailed', { n: fmtNum(counts.failed) }) }}</b>
        </template>
      </template>
      <template #actions>
        <template v-if="!menuStore.isMobile">
          <UButton
            color="neutral"
            :variant="autoRefresh ? 'soft' : 'outline'"
            icon="i-lucide-refresh-cw"
            :aria-pressed="autoRefresh"
            @click="toggleAuto"
          >
            {{ autoRefresh ? t('systemV2.tasks.autoOn', { s: REFRESH_SECONDS }) : t('systemV2.tasks.autoOff') }}
          </UButton>
          <UButton color="neutral" variant="outline" icon="i-lucide-trash-2" :disabled="!finishedCount" @click="confirmClear">
            {{ t('systemV2.tasks.clearFinished') }}
          </UButton>
        </template>
        <template v-else>
          <AppIconButton
            icon="HOutline:ArrowPathIcon"
            :label="autoRefresh ? t('systemV2.tasks.autoOn', { s: REFRESH_SECONDS }) : t('systemV2.tasks.autoOff')"
            :active="autoRefresh"
            :box="44"
            :aria-pressed="autoRefresh"
            @click="toggleAuto"
          />
          <AppIconButton
            icon="HOutline:TrashIcon"
            :label="t('systemV2.tasks.clearFinished')"
            :box="44"
            :disabled="!finishedCount"
            @click="confirmClear"
          />
        </template>
      </template>
    </PageHeader>

    <LogsSubNav :counts="navCounts" />

    <!-- 手机：主色 hero -->
    <section v-if="menuStore.isMobile" class="tk-hero">
      <span class="tk-hero__sun" aria-hidden="true" />
      <span class="tk-hero__ring" aria-hidden="true" />
      <div class="tk-hero__body">
        <span class="tk-hero__label">{{ t('systemV2.tasks.heroLabel') }}</span>
        <span class="tk-hero__line">
          <b class="tk-hero__num m-num">{{ counts.running }}</b>
          <span>{{ t('systemV2.tasks.heroUnit', { pending: counts.pending }) }}</span>
        </span>
        <span class="tk-hero__hint">{{ heroHint }}</span>
      </div>
    </section>

    <ListToolbar
      v-model:status="statusFilter"
      v-model:keyword="keywords"
      :chips="statusChips"
      :chips-label="t('systemV2.tasks.colStatus')"
      :placeholder="t('systemV2.tasks.searchPlaceholder')"
      class="tk-toolbar"
      @search="reload"
    >
      <div class="tk-type">
        <AppSelect v-model="typeFilter" :options="typeOptions" fit :aria-label="t('systemV2.tasks.type')" />
      </div>
    </ListToolbar>

    <!-- PC：可展开行表格 -->
    <div v-if="!menuStore.isMobile" class="tk-table">
      <div class="tk-row tk-row--head" role="row">
        <span class="tk-cell--check">
          <input
            type="checkbox"
            class="tk-cb"
            :checked="allSelected"
            :indeterminate.prop="someSelected"
            :aria-label="t('systemV2.tasks.selectAll')"
            @change="toggleAll"
          />
        </span>
        <span>{{ t('systemV2.tasks.colTask') }}</span>
        <span>{{ t('systemV2.tasks.colStatus') }}</span>
        <span>{{ t('systemV2.tasks.colBy') }}</span>
        <span>{{ t('systemV2.tasks.colStart') }}</span>
        <span>{{ t('systemV2.tasks.colDuration') }}</span>
        <span class="tk-cell--end">{{ t('systemV2.tasks.colActions') }}</span>
      </div>

      <div v-if="loading && !rows.length" class="tk-skel-list">
        <div v-for="i in 5" :key="i" class="tk-skeleton" />
      </div>
      <EmptyState
        v-else-if="!rows.length"
        icon="HOutline:QueueListIcon"
        :title="t('taskManager.empty')"
        :description="t('taskManager.emptyDesc')"
      />

      <div
        v-for="r in rows"
        v-else
        :key="r.id"
        class="tk-item"
        :class="{ 'is-open': openId === r.id, 'is-sel': selected.has(r.id) && openId !== r.id }"
      >
        <div class="tk-row" role="row">
          <span class="tk-cell--check">
            <input
              type="checkbox"
              class="tk-cb"
              :checked="selected.has(r.id)"
              :aria-label="t('systemV2.tasks.selectOf', { name: r.title })"
              @change="toggleSel(r.id)"
            />
          </span>
          <span class="tk-name">
            <span class="m-sticker tk-sticker" :class="`m-sticker--${r.meta.tone}`" :style="{ transform: `rotate(${r.rot})` }">
              <component :is="menuStore.iconComponents[r.meta.icon]" />
            </span>
            <span class="tk-name__text">
              <span class="tk-name__title">{{ r.title }}</span>
              <span class="tk-name__sub m-mono">{{ r.shortId }} · {{ nodeLabel }}</span>
            </span>
          </span>
          <span class="tk-state">
            <span class="tk-state__line">
              <StatusPill :variant="r.variant" :label="r.statusText" :class="{ 'tk-pill--run': r.running }" />
              <b v-if="r.showPct" class="tk-state__pct m-num">{{ r.pct }}%</b>
            </span>
            <span
              v-if="r.showPct"
              class="tk-bar"
              role="progressbar"
              :aria-valuenow="r.pct"
              aria-valuemin="0"
              aria-valuemax="100"
              :aria-label="t('systemV2.tasks.progressOf', { name: r.title })"
            >
              <span :style="{ width: `${r.pct}%` }" />
            </span>
            <span v-if="r.sub" class="tk-state__sub" :class="`is-${r.subTone}`">{{ r.sub }}</span>
          </span>
          <span class="tk-by">
            <span class="tk-avatar" :class="`m-sticker--${r.by.tone}`">{{ r.by.initial }}</span>
            <span class="tk-by__name">{{ r.by.name }}</span>
          </span>
          <span class="tk-time m-num">{{ r.start }}</span>
          <span class="tk-dur m-num">{{ r.dur }}</span>
          <span class="tk-acts">
            <UButton
              size="sm"
              color="neutral"
              :variant="openId === r.id ? 'solid' : 'soft'"
              :trailing-icon="openId === r.id ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
              :aria-expanded="openId === r.id"
              class="tk-detail-btn"
              @click="toggleOpen(r.id)"
            >
              {{ openId === r.id ? t('systemV2.tasks.collapse') : t('systemV2.tasks.details') }}
            </UButton>
            <AppIconButton
              v-if="r.canCancel"
              icon="HOutline:XMarkIcon"
              class="is-stop"
              :label="t('systemV2.tasks.cancelOf', { name: r.title })"
              :box="36"
              :size="16"
              @click="cancelOne(r.id)"
            />
            <AppIconButton
              v-if="r.canRetry"
              icon="HOutline:ArrowPathIcon"
              class="is-retry"
              :label="t('systemV2.tasks.retryOf', { name: r.title })"
              :box="36"
              :size="16"
              @click="retryOne(r.id)"
            />
            <AppIconButton
              v-if="r.canDelete"
              icon="HOutline:TrashIcon"
              :label="t('systemV2.tasks.deleteOf', { name: r.title })"
              :box="36"
              :size="16"
              @click="confirmDelete(r.id)"
            />
          </span>
        </div>

        <div v-if="openId === r.id && openView" class="tk-detail">
          <section class="tk-detail__card" :aria-label="openView.stepsTitle">
            <div class="tk-detail__head">
              <h3>{{ openView.stepsTitle }}</h3>
              <span class="m-num">{{ openView.stepsDone }} / {{ openView.steps.length }}</span>
            </div>
            <ol class="tk-steps">
              <li v-for="(s, idx) in openView.steps" :key="idx" class="tk-step" :class="{ 'has-line': idx < openView.steps.length - 1 }">
                <span class="tk-step__dot" :class="`is-${s.state}`">
                  <component :is="menuStore.iconComponents[STEP_ICON[s.state]]" />
                </span>
                <span class="tk-step__text">
                  <span class="tk-step__title" :class="{ 'm-mono': s.mono }">{{ s.title }}</span>
                  <span v-if="s.sub" class="tk-step__sub" :class="`is-${s.subTone}`">{{ s.sub }}</span>
                </span>
                <span v-if="s.dur" class="tk-step__dur m-num">{{ s.dur }}</span>
              </li>
              <li v-if="openView.more" class="tk-step__more">{{ t('systemV2.tasks.moreResults', { n: openView.more }) }}</li>
            </ol>
            <dl class="tk-dl tk-dl--3">
              <div v-for="f in openView.fields" :key="f.label" :class="{ 'is-wide': f.wide }">
                <dt>{{ f.label }}</dt>
                <dd :class="{ 'm-mono': f.mono, 'm-num': f.num }" :title="f.value">{{ f.value }}</dd>
              </div>
            </dl>
          </section>
          <section class="tk-log" :aria-label="openView.logTitle">
            <div class="tk-log__head">
              <component :is="menuStore.iconComponents['HOutline:CommandLineIcon']" />
              <span class="tk-log__title">{{ openView.logTitle }}</span>
              <span class="tk-log__count m-mono">{{ t('systemV2.tasks.lines', { n: openView.logs.length }) }}</span>
              <button type="button" class="tk-log__btn" :aria-label="t('systemV2.tasks.copyLog')" :title="t('systemV2.tasks.copyLog')" @click="copyLog">
                <component :is="menuStore.iconComponents['HOutline:ClipboardDocumentIcon']" />
              </button>
              <button type="button" class="tk-log__btn" :aria-label="t('systemV2.tasks.downloadLog')" :title="t('systemV2.tasks.downloadLog')" @click="downloadLog">
                <component :is="menuStore.iconComponents['HOutline:ArrowDownTrayIcon']" />
              </button>
            </div>
            <div ref="logBodyRef" class="tk-log__body m-mono">
              <div v-for="(l, idx) in openView.logs" :key="idx" class="tk-log__line">
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
      <div v-if="loading && !rows.length" class="tk-cards">
        <div v-for="i in 4" :key="i" class="tk-skeleton tk-skeleton--card" />
      </div>
      <EmptyState
        v-else-if="!rows.length"
        icon="HOutline:QueueListIcon"
        :title="t('taskManager.empty')"
        :description="t('taskManager.emptyDesc')"
      />
      <div v-else class="tk-cards">
        <article v-for="r in rows" :key="r.id" class="tk-card" :class="{ 'is-sel': selected.has(r.id) }">
          <header class="tk-card__head">
            <span class="m-sticker tk-sticker tk-sticker--lg" :class="`m-sticker--${r.meta.tone}`" :style="{ transform: `rotate(${r.rot})` }">
              <component :is="menuStore.iconComponents[r.meta.icon]" />
            </span>
            <span class="tk-name__text">
              <span class="tk-card__title">{{ r.title }}</span>
              <span class="tk-card__id m-mono">{{ r.shortId }}</span>
            </span>
            <StatusPill :variant="r.variant" :label="r.statusText" :class="{ 'tk-pill--run': r.running }" />
          </header>

          <div v-if="r.showPct" class="tk-card__prog">
            <div class="tk-card__bar">
              <span class="tk-bar" role="progressbar" :aria-valuenow="r.pct" aria-valuemin="0" aria-valuemax="100" :aria-label="t('systemV2.tasks.progressOf', { name: r.title })">
                <span :style="{ width: `${r.pct}%` }" />
              </span>
              <b class="m-num">{{ r.pct }}%</b>
            </div>
            <span v-if="r.progText" class="tk-card__progtext">{{ r.progText }}</span>
          </div>
          <span v-else-if="r.running && r.progText" class="tk-card__progtext">{{ r.progText }}</span>
          <span v-if="r.sub" class="tk-card__sub" :class="`is-${r.subTone}`">{{ r.sub }}</span>

          <dl class="tk-dl tk-dl--card">
            <div>
              <dt>{{ t('systemV2.tasks.colBy') }}</dt>
              <dd class="tk-dl__by">
                <span class="tk-avatar tk-avatar--sm" :class="`m-sticker--${r.by.tone}`">{{ r.by.initial }}</span>
                <span>{{ r.by.name }}</span>
              </dd>
            </div>
            <div><dt>{{ t('systemV2.tasks.node') }}</dt><dd>{{ nodeLabel }}</dd></div>
            <div><dt>{{ t('systemV2.tasks.colStart') }}</dt><dd class="m-num">{{ r.start }}</dd></div>
            <div><dt>{{ t('systemV2.tasks.colDuration') }}</dt><dd class="m-num">{{ r.dur }}</dd></div>
          </dl>

          <div v-if="openId === r.id && openView" class="tk-card__detail">
            <div class="tk-detail__head">
              <h3>{{ openView.stepsTitle }}</h3>
              <span class="m-num">{{ openView.stepsDone }} / {{ openView.steps.length }}</span>
            </div>
            <ol class="tk-steps">
              <li v-for="(s, idx) in openView.steps" :key="idx" class="tk-step" :class="{ 'has-line': idx < openView.steps.length - 1 }">
                <span class="tk-step__dot" :class="`is-${s.state}`">
                  <component :is="menuStore.iconComponents[STEP_ICON[s.state]]" />
                </span>
                <span class="tk-step__text">
                  <span class="tk-step__title" :class="{ 'm-mono': s.mono }">{{ s.title }}</span>
                  <span v-if="s.sub" class="tk-step__sub" :class="`is-${s.subTone}`">{{ s.sub }}</span>
                </span>
                <span v-if="s.dur" class="tk-step__dur m-num">{{ s.dur }}</span>
              </li>
              <li v-if="openView.more" class="tk-step__more">{{ t('systemV2.tasks.moreResults', { n: openView.more }) }}</li>
            </ol>
            <dl class="tk-dl tk-dl--card">
              <div v-for="f in openView.fields" :key="f.label" :class="{ 'is-wide': f.wide }">
                <dt>{{ f.label }}</dt>
                <dd :class="{ 'm-mono': f.mono, 'm-num': f.num }">{{ f.value }}</dd>
              </div>
            </dl>
            <section class="tk-log tk-log--card" :aria-label="openView.logTitle">
              <div class="tk-log__head">
                <component :is="menuStore.iconComponents['HOutline:CommandLineIcon']" />
                <span class="tk-log__title">{{ openView.logTitle }}</span>
                <span class="tk-log__count m-mono">{{ t('systemV2.tasks.lines', { n: openView.logs.length }) }}</span>
                <button type="button" class="tk-log__btn" :aria-label="t('systemV2.tasks.copyLog')" @click="copyLog">
                  <component :is="menuStore.iconComponents['HOutline:ClipboardDocumentIcon']" />
                </button>
              </div>
              <div class="tk-log__body m-mono">
                <div v-for="(l, idx) in openView.logs" :key="idx" class="tk-log__line">
                  <span class="tk-log__tm">{{ l.tm }}</span>
                  <span :class="`is-${l.tone}`">{{ l.text }}</span>
                </div>
              </div>
            </section>
          </div>

          <footer class="tk-card__foot">
            <label class="tk-card__check">
              <input
                type="checkbox"
                class="tk-cb"
                :checked="selected.has(r.id)"
                :aria-label="t('systemV2.tasks.selectOf', { name: r.title })"
                @change="toggleSel(r.id)"
              />
              <span class="tk-tag">#{{ r.tag }}</span>
            </label>
            <AppIconButton
              icon="HOutline:DocumentTextIcon"
              :label="t('systemV2.tasks.detailsOf', { name: r.title })"
              :active="openId === r.id"
              :box="44"
              :aria-expanded="openId === r.id"
              @click="toggleOpen(r.id)"
            />
            <AppIconButton
              v-if="r.canCancel"
              icon="HOutline:XMarkIcon"
              class="is-stop"
              :label="t('systemV2.tasks.cancelOf', { name: r.title })"
              :box="44"
              @click="cancelOne(r.id)"
            />
            <AppIconButton
              v-if="r.canRetry"
              icon="HOutline:ArrowPathIcon"
              class="is-retry"
              :label="t('systemV2.tasks.retryOf', { name: r.title })"
              :box="44"
              @click="retryOne(r.id)"
            />
            <AppIconButton
              v-if="r.canDelete"
              icon="HOutline:TrashIcon"
              :label="t('systemV2.tasks.deleteOf', { name: r.title })"
              :box="44"
              @click="confirmDelete(r.id)"
            />
          </footer>
        </article>
      </div>
    </template>

    <div class="tk-foot">
      <span class="tk-foot__text">
        <template v-if="total">
          {{
            menuStore.isMobile
              ? t('systemV2.tasks.footerMobile', { shown: rows.length, total: fmtNum(total) })
              : t('systemV2.tasks.footer', { from: rangeFrom, to: rangeTo, total: fmtNum(total) })
          }}
          <template v-if="retentionDays"> · </template>
        </template>
        <template v-if="retentionDays">{{ t('systemV2.tasks.retention', { days: retentionDays }) }}</template>
      </span>
      <Pagination v-model:page="page" v-model:page-size="pageSize" :total="total" class="tk-foot__pager" @change="() => getList()" />
    </div>

    <!-- 批量操作条 -->
    <Transition name="tk-bulk">
      <div v-if="selected.size" class="tk-bulk" role="toolbar" :aria-label="t('systemV2.tasks.bulk')">
        <span class="tk-bulk__count">{{ t('systemV2.tasks.selected', { n: selected.size }) }}</span>
        <button type="button" class="tk-bulk__btn" :disabled="!selRetryable.length" @click="bulkRetry">
          <component :is="menuStore.iconComponents['HOutline:ArrowPathIcon']" />
          <span>{{ t('systemV2.tasks.retry') }}</span>
        </button>
        <button type="button" class="tk-bulk__btn" :disabled="!selCancelable.length" @click="bulkCancel">
          <component :is="menuStore.iconComponents['HOutline:XMarkIcon']" />
          <span>{{ t('systemV2.tasks.cancel') }}</span>
        </button>
        <button type="button" class="tk-bulk__btn is-warn" :disabled="!selDeletable.length" @click="bulkDelete">
          <component :is="menuStore.iconComponents['HOutline:TrashIcon']" />
          <span>{{ t('systemV2.tasks.deleteRecords') }}</span>
        </button>
        <button type="button" class="tk-bulk__btn tk-bulk__x" :aria-label="t('systemV2.tasks.clearSelection')" @click="selected.clear()">
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
import { listTasks, getTask, cancelTasks, retryTasks, deleteTasks, clearTasks } from '@/api/task'
import { getOperationLogStats } from '@/api/system'
import { PERM } from '@/config/permission'
import {
  TaskStatus,
  TaskKind,
  TaskTrigger,
  TaskStepState,
  TaskLogLevel,
  type TaskInfo,
  type TaskStatusCounts,
  type TaskTypeCount,
  type TaskBatchResponse,
} from '@/types/v1/task'
import type { ToolbarChip } from '@/components/ui/ListToolbar.vue'
import LogsSubNav from '@/views/system/components/LogsSubNav.vue'

defineOptions({ name: 'TaskManagerView' })

const menuStore = useMenuStore()
const nodeStore = useNodeStore()
const { t, locale } = useI18n()
const { copy } = useClipboard()

const REFRESH_INTERVAL_MS = 2500
const REFRESH_SECONDS = 2.5
const MAX_RESULT_STEPS = 12

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
const detail = ref<TaskInfo | null>(null)
const selected = reactive(new Set<string>())
const counts = ref<TaskStatusCounts>({ all: 0, pending: 0, running: 0, success: 0, failed: 0, canceled: 0 })
const types = ref<TaskTypeCount[]>([])
const retentionDays = ref(0)
const opTotal = ref<number | null>(null)
const now = ref(Date.now())
const logBodyRef = ref<HTMLElement[] | HTMLElement | null>(null)
let timer: ReturnType<typeof setInterval> | undefined

const fmtNum = (n: number) => new Intl.NumberFormat(locale.value).format(n)
const finishedCount = computed(() => counts.value.success + counts.value.failed + counts.value.canceled)
const navCounts = computed(() => ({ task: counts.value.all, operation: opTotal.value }))
const nodeLabel = computed(() =>
  nodeStore.isRemote ? nodeStore.contextInfo?.name || nodeStore.nodeId : t('systemV2.tasks.local'),
)
const rangeFrom = computed(() => (total.value ? (page.value - 1) * pageSize.value + 1 : 0))
const rangeTo = computed(() => Math.min(total.value, (page.value - 1) * pageSize.value + items.value.length))

// 状态值沿用后端小写过滤契约
const statusChips = computed<ToolbarChip<string>[]>(() => [
  { label: t('systemV2.tasks.statusAll'), value: '', count: counts.value.all },
  { label: t('systemV2.tasks.statusRunning'), value: 'running', count: counts.value.running },
  { label: t('systemV2.tasks.statusPending'), value: 'pending', count: counts.value.pending },
  { label: t('systemV2.tasks.statusSuccess'), value: 'success', count: counts.value.success },
  { label: t('systemV2.tasks.statusFailed'), value: 'failed', count: counts.value.failed },
  { label: t('systemV2.tasks.statusCanceled'), value: 'canceled', count: counts.value.canceled },
])

// —— 任务类型：展示名、贴纸图标、标签 ——
interface TypeMeta {
  key: string
  icon: string
  tone: string
  tag: string
}
const TYPE_META: Record<string, TypeMeta> = {
  'docker.image_pull': { key: 'imagePull', icon: 'HOutline:ArrowDownTrayIcon', tone: 'a2', tag: 'image' },
  'docker.container_recreate': { key: 'containerRecreate', icon: 'HOutline:ArrowPathRoundedSquareIcon', tone: 'ok', tag: 'docker' },
  'docker.container_create': { key: 'containerCreate', icon: 'HOutline:CubeIcon', tone: 'p', tag: 'docker' },
  'docker.network_recreate': { key: 'networkRecreate', icon: 'HOutline:ShareIcon', tone: 'a2', tag: 'network' },
  'file.upload_finalize': { key: 'uploadFinalize', icon: 'HOutline:ArrowUpTrayIcon', tone: 'p', tag: 'upload' },
  'file.upload_gc': { key: 'uploadGc', icon: 'HOutline:SparklesIcon', tone: 'warn', tag: 'cleanup' },
  'file.copy': { key: 'fileCopy', icon: 'HOutline:DocumentDuplicateIcon', tone: 'a', tag: 'file' },
  'file.move': { key: 'fileMove', icon: 'HOutline:FolderArrowDownIcon', tone: 'a', tag: 'file' },
  'share.cleanup': { key: 'shareCleanup', icon: 'HOutline:LinkIcon', tone: 'warn', tag: 'share' },
  'iqcheck.tick': { key: 'iqcheckTick', icon: 'HOutline:LightBulbIcon', tone: 'a2', tag: 'iqcheck' },
  'task.cleanup': { key: 'taskCleanup', icon: 'HOutline:TrashIcon', tone: 'bad', tag: 'cleanup' },
  'audit.cleanup': { key: 'auditCleanup', icon: 'HOutline:ClipboardDocumentListIcon', tone: 'warn', tag: 'cleanup' },
  'monitor.cleanup': { key: 'monitorCleanup', icon: 'HOutline:PresentationChartLineIcon', tone: 'a2', tag: 'cleanup' },
}
const ROTS = ['-4deg', '3deg', '-2deg', '4deg', '-3deg']
const hashRot = (s: string) => ROTS[[...s].reduce((a, c) => a + c.charCodeAt(0), 0) % ROTS.length] ?? '0deg'
const metaOf = (type: string): TypeMeta =>
  TYPE_META[type] ?? {
    key: '',
    icon: 'HOutline:Cog6ToothIcon',
    tone: type.startsWith('docker.') ? 'a2' : type.startsWith('file.') ? 'a' : 'p',
    tag: type.startsWith('docker.') ? 'docker' : type.startsWith('file.') ? 'file' : 'other',
  }
const typeLabel = (type: string) => {
  const meta = TYPE_META[type]
  return meta ? t(`systemV2.tasks.types.${meta.key}`) : type
}
const typeOptions = computed(() => [
  { label: t('systemV2.tasks.allTypes'), value: '' },
  ...types.value.map((ty) => ({ label: `${typeLabel(ty.type)} · ${ty.count}`, value: ty.type })),
])

// —— 状态 ——
const STATUS_TEXT: Partial<Record<TaskStatus, string>> = {
  [TaskStatus.TASK_STATUS_PENDING]: 'statusPending',
  [TaskStatus.TASK_STATUS_RUNNING]: 'statusRunning',
  [TaskStatus.TASK_STATUS_SUCCESS]: 'statusSuccess',
  [TaskStatus.TASK_STATUS_FAILED]: 'statusFailed',
  [TaskStatus.TASK_STATUS_CANCELED]: 'statusCanceled',
}
const statusLabel = (s: TaskStatus) => (STATUS_TEXT[s] ? t(`systemV2.tasks.${STATUS_TEXT[s]}`) : String(s))
/** 定时任务两次触发之间是「等待下次执行」，不是排队。 */
const isWaitingSchedule = (row: TaskInfo) => row.status === TaskStatus.TASK_STATUS_PENDING && row.kind === TaskKind.TASK_KIND_SCHEDULED
const rowStatusLabel = (row: TaskInfo) => (isWaitingSchedule(row) ? t('systemV2.tasks.statusWaiting') : statusLabel(row.status))
const statusVariant = (s: TaskStatus) => {
  switch (s) {
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
const isActive = (s: TaskStatus) => s === TaskStatus.TASK_STATUS_RUNNING || s === TaskStatus.TASK_STATUS_PENDING
const isTerminal = (s: TaskStatus) =>
  s === TaskStatus.TASK_STATUS_SUCCESS || s === TaskStatus.TASK_STATUS_FAILED || s === TaskStatus.TASK_STATUS_CANCELED

// —— 时间 ——
const ms = (v?: Date) => (v ? dayjs(v).valueOf() : null)
const fmtDuration = (milli: number) => {
  if (milli < 1000) return t('systemV2.tasks.durMs', { n: Math.max(0, Math.round(milli)) })
  const s = Math.round(milli / 1000)
  if (s < 60) return t('systemV2.tasks.durSec', { s })
  const m = Math.floor(s / 60)
  if (m < 60) return t('systemV2.tasks.durMin', { m, s: String(s % 60).padStart(2, '0') })
  return t('systemV2.tasks.durHour', { h: Math.floor(m / 60), m: m % 60 })
}
const fmtInterval = (milli: number) => {
  const s = Math.round(milli / 1000)
  if (s < 60) return t('systemV2.tasks.everySec', { n: s })
  if (s < 3600) return t('systemV2.tasks.everyMin', { n: Math.round(s / 60) })
  if (s < 86400) return t('systemV2.tasks.everyHour', { n: Math.round(s / 3600) })
  return t('systemV2.tasks.everyDay', { n: Math.round(s / 86400) })
}
const fmtStart = (v?: Date) => {
  if (!v) return '—'
  const d = dayjs(v)
  const today = dayjs().startOf('day')
  if (d.isAfter(today)) return `${t('systemV2.tasks.today')} ${d.format('HH:mm:ss')}`
  if (d.isAfter(today.subtract(1, 'day'))) return `${t('systemV2.tasks.yesterday')} ${d.format('HH:mm:ss')}`
  return d.format('MM-DD HH:mm')
}
const fmtFull = (v?: Date) => (v ? dayjs(v).format('YYYY-MM-DD HH:mm:ss') : '—')
const startMs = (row: TaskInfo) => ms(row.startTime)
const durationMs = (row: TaskInfo) => {
  const start = startMs(row)
  if (start == null || row.status === TaskStatus.TASK_STATUS_PENDING) return null
  const end = ms(row.endTime) ?? (isActive(row.status) ? now.value : null)
  return end == null ? null : end - start
}

// —— 进度 ——
const percent = (row: TaskInfo) => {
  const totalNum = Number(row.total) || 0
  if (totalNum <= 0) return 0
  return Math.min(100, Math.round(((Number(row.finished) || 0) / totalNum) * 100))
}
/** 预计剩余（毫秒）：按已执行时长与完成比例线性估算。 */
const etaMs = (row: TaskInfo) => {
  if (row.status !== TaskStatus.TASK_STATUS_RUNNING) return null
  const pct = percent(row)
  const elapsed = durationMs(row)
  if (!pct || pct >= 100 || elapsed == null || elapsed < 1000) return null
  return (elapsed * (100 - pct)) / pct
}
const etaText = (row: TaskInfo) => {
  const eta = etaMs(row)
  return eta == null ? '' : t('systemV2.tasks.about', { t: fmtDuration(eta) })
}

// —— 来源：触发方式与发起人 ——
const triggerText = (row: TaskInfo) => {
  switch (row.trigger) {
    case TaskTrigger.TASK_TRIGGER_MANUAL: {
      const key = sourceKey(row.source)
      return key
        ? t('systemV2.tasks.triggers.manualFrom', { source: t(`systemV2.tasks.sources.${key}`) })
        : t('systemV2.tasks.triggers.manual')
    }
    case TaskTrigger.TASK_TRIGGER_SCHEDULE:
      return t('systemV2.tasks.triggers.schedule', { interval: fmtInterval(Number(row.intervalMs) || 3600000) })
    case TaskTrigger.TASK_TRIGGER_MASTER:
      return t('systemV2.tasks.triggers.master')
    default:
      return t('systemV2.tasks.triggers.system')
  }
}
const SOURCE_KEYS: Record<string, string> = {
  'docker.container': 'dockerContainer',
  'docker.image': 'dockerImage',
  'docker.network': 'dockerNetwork',
  file: 'file',
  'instance.file': 'instanceFile',
  task: 'task',
}
const sourceKey = (source: string) => SOURCE_KEYS[source] || ''

const byOf = (row: TaskInfo) => {
  let name: string
  let tone: string
  if (row.trigger === TaskTrigger.TASK_TRIGGER_MASTER) {
    name = t('systemV2.tasks.byMaster', { name: row.operator || row.userName || '—' })
    tone = 'p'
  } else if (row.userName) {
    name = row.userName
    tone = 'a'
  } else if (row.trigger === TaskTrigger.TASK_TRIGGER_SCHEDULE) {
    name = t('systemV2.tasks.bySchedule')
    tone = 'ok'
  } else {
    name = t('systemV2.tasks.bySystem')
    tone = 'a2'
  }
  const base = row.trigger === TaskTrigger.TASK_TRIGGER_MASTER ? row.operator || name : name
  return { name, tone, initial: ([...base][0] ?? '?').toUpperCase() }
}

const subOf = (row: TaskInfo): { text: string; tone: string } => {
  switch (row.status) {
    case TaskStatus.TASK_STATUS_FAILED:
      return { text: row.error || '', tone: 'bad' }
    case TaskStatus.TASK_STATUS_CANCELED:
      return {
        text: row.canceledByName
          ? t('systemV2.tasks.canceledBy', { name: row.canceledByName })
          : row.error || t('systemV2.tasks.canceledBySystem'),
        tone: 'dim',
      }
    case TaskStatus.TASK_STATUS_PENDING:
      if (row.nextRunTime) return { text: t('systemV2.tasks.nextRun', { time: fmtStart(row.nextRunTime) }), tone: 'dim' }
      return { text: row.message || '', tone: 'warn' }
    default:
      return { text: '', tone: 'dim' }
  }
}

// uuid 只显示前 8 位；单例任务的 id 就是类型键，原样显示
const shortId = (id: string) => (/^[0-9a-f]{8}-[0-9a-f-]{27}$/i.test(id) ? id.slice(0, 8) : id)

// —— 行视图 ——
const rows = computed(() =>
  items.value.map((row) => {
    const meta = metaOf(row.type)
    const running = row.status === TaskStatus.TASK_STATUS_RUNNING
    const showPct = running && Number(row.total) > 0
    const sub = subOf(row)
    const dur = durationMs(row)
    const eta = etaText(row)
    return {
      id: row.id,
      raw: row,
      title: row.title || typeLabel(row.type),
      shortId: shortId(row.id),
      meta,
      rot: hashRot(row.id),
      tag: t(`systemV2.tasks.tags.${meta.tag}`),
      running,
      variant: isWaitingSchedule(row) ? ('neutral' as const) : statusVariant(row.status),
      statusText: rowStatusLabel(row),
      showPct,
      pct: percent(row),
      progText: [row.message, eta].filter(Boolean).join(' · '),
      sub: sub.text,
      subTone: sub.tone,
      by: byOf(row),
      start: fmtStart(row.startTime),
      dur: dur == null ? '—' : fmtDuration(dur),
      canCancel: isActive(row.status),
      canRetry: row.retryable && (row.status === TaskStatus.TASK_STATUS_FAILED || row.status === TaskStatus.TASK_STATUS_CANCELED),
      canDelete: isTerminal(row.status),
    }
  }),
)
type RowView = (typeof rows.value)[number]

// —— 展开详情：步骤 + 信息格 + 日志 ——
type StepTone = 'ok' | 'run' | 'wait' | 'fail' | 'skip'
const STEP_ICON: Record<StepTone, string> = {
  ok: 'HOutline:CheckIcon',
  run: 'HOutline:ArrowPathIcon',
  wait: 'HOutline:MinusIcon',
  fail: 'HOutline:XMarkIcon',
  skip: 'HOutline:MinusIcon',
}
const STEP_TONE: Partial<Record<TaskStepState, StepTone>> = {
  [TaskStepState.TASK_STEP_STATE_WAITING]: 'wait',
  [TaskStepState.TASK_STEP_STATE_RUNNING]: 'run',
  [TaskStepState.TASK_STEP_STATE_SUCCESS]: 'ok',
  [TaskStepState.TASK_STEP_STATE_FAILED]: 'fail',
  [TaskStepState.TASK_STEP_STATE_SKIPPED]: 'skip',
}
interface StepView {
  title: string
  sub: string
  subTone: string
  dur: string
  state: StepTone
  mono?: boolean
}
interface LogLine {
  tm: string
  text: string
  tone: 'ink' | 'dim' | 'ok' | 'bad' | 'warn'
}
const LOG_TONE: Partial<Record<TaskLogLevel, LogLine['tone']>> = {
  [TaskLogLevel.TASK_LOG_LEVEL_SUCCESS]: 'ok',
  [TaskLogLevel.TASK_LOG_LEVEL_WARN]: 'warn',
  [TaskLogLevel.TASK_LOG_LEVEL_ERROR]: 'bad',
}

const spanText = (start?: Date, end?: Date, active = false) => {
  const s = ms(start)
  if (s == null) return ''
  const e = ms(end) ?? (active ? now.value : null)
  return e == null ? '' : fmtDuration(e - s)
}

const buildSteps = (row: TaskInfo, title: string): { title: string; steps: StepView[]; done: number; more: number } => {
  const steps = row.steps ?? []
  if (steps.length) {
    const list = steps.map<StepView>((s) => {
      const state = STEP_TONE[s.state] ?? 'wait'
      return {
        title: s.title,
        sub: state === 'skip' && !s.detail ? t('systemV2.tasks.skipped') : s.detail,
        subTone: state === 'fail' ? 'bad' : state === 'run' ? 'p' : 'dim',
        dur: state === 'run' ? spanText(s.startTime, undefined, true) : spanText(s.startTime, s.endTime),
        state,
      }
    })
    return { title: t('systemV2.tasks.steps'), steps: list, done: list.filter((s) => s.state === 'ok').length, more: 0 }
  }
  const results = row.results ?? []
  if (results.length) {
    const list = results.slice(0, MAX_RESULT_STEPS).map<StepView>((r) => ({
      title: r.path || '—',
      sub: r.message,
      subTone: r.success ? 'dim' : 'bad',
      dur: '',
      state: r.success ? 'ok' : 'fail',
      mono: true,
    }))
    const pending = Math.max(0, (Number(row.total) || 0) - results.length)
    if (pending && isActive(row.status)) {
      list.push({ title: t('systemV2.tasks.statusRunning'), sub: `${results.length} / ${Number(row.total)}`, subTone: 'p', dur: '', state: 'run' })
    }
    return {
      title: t('systemV2.tasks.results'),
      steps: list,
      done: results.filter((r) => r.success).length,
      more: Math.max(0, results.length - MAX_RESULT_STEPS),
    }
  }
  // 没有步骤与结果：按生命周期给出 提交 → 执行 → 收尾
  const st = row.status
  const runState: StepTone =
    st === TaskStatus.TASK_STATUS_SUCCESS ? 'ok' : st === TaskStatus.TASK_STATUS_RUNNING ? 'run' : st === TaskStatus.TASK_STATUS_PENDING ? 'wait' : 'fail'
  const list: StepView[] = [
    { title: t('systemV2.tasks.stepSubmit'), sub: fmtFull(row.createTime), subTone: 'dim', dur: '', state: 'ok' },
    {
      title: title,
      sub: st === TaskStatus.TASK_STATUS_FAILED ? row.error : row.message,
      subTone: runState === 'fail' ? 'bad' : runState === 'run' ? 'p' : 'dim',
      dur: spanText(row.startTime, row.endTime, isActive(st)),
      state: runState,
    },
    {
      title: t('systemV2.tasks.stepFinish'),
      sub: st === TaskStatus.TASK_STATUS_SUCCESS ? fmtFull(row.endTime) : isTerminal(st) ? t('systemV2.tasks.skipped') : '',
      subTone: 'dim',
      dur: '',
      state: st === TaskStatus.TASK_STATUS_SUCCESS ? 'ok' : isTerminal(st) ? 'skip' : 'wait',
    },
  ]
  return { title: t('systemV2.tasks.steps'), steps: list, done: list.filter((s) => s.state === 'ok').length, more: 0 }
}

const buildLogs = (row: TaskInfo, title: string): LogLine[] => {
  const hm = (v?: Date) => (v ? dayjs(v).format('HH:mm:ss') : '--:--:--')
  const lines: LogLine[] = [{ tm: hm(row.startTime || row.createTime), text: t('systemV2.tasks.logStart', { name: title }), tone: 'ink' }]
  for (const l of row.logs ?? []) {
    lines.push({ tm: hm(l.time), text: l.message, tone: LOG_TONE[l.level] ?? 'dim' })
  }
  if (row.endTime) {
    const tone = row.status === TaskStatus.TASK_STATUS_SUCCESS ? 'ok' : row.status === TaskStatus.TASK_STATUS_FAILED ? 'bad' : 'warn'
    lines.push({ tm: hm(row.endTime), text: t('systemV2.tasks.logEnd', { status: statusLabel(row.status) }), tone })
  }
  return lines
}

const openView = computed(() => {
  if (!openId.value) return null
  const listed = items.value.find((r) => r.id === openId.value)
  if (!listed) return null
  // 详情接口带步骤与日志；列表轮询比详情新时以列表的状态/进度为准
  const row: TaskInfo = detail.value && detail.value.id === listed.id ? { ...detail.value, ...pickLive(listed) } : listed
  const title = listed.title || typeLabel(listed.type)
  const steps = buildSteps(row, title)
  const eta = etaText(row)
  const fields: { label: string; value: string; mono?: boolean; num?: boolean; wide?: boolean }[] = [
    { label: t('systemV2.tasks.trigger'), value: triggerText(row) },
    { label: t('systemV2.tasks.node'), value: nodeLabel.value },
    { label: t('systemV2.tasks.eta'), value: eta || '—', num: true },
    { label: t('systemV2.tasks.type'), value: typeLabel(row.type) },
    {
      label: t('systemV2.tasks.progress'),
      value: Number(row.total) > 0 ? `${Number(row.finished) || 0} / ${Number(row.total)}` : '—',
      num: true,
    },
    { label: row.endTime ? t('systemV2.tasks.endTime') : t('systemV2.tasks.submitTime'), value: fmtFull(row.endTime || row.createTime), num: true },
  ]
  if (row.kind === TaskKind.TASK_KIND_SCHEDULED) {
    fields[2] = { label: t('systemV2.tasks.nextRunLabel'), value: row.nextRunTime ? fmtFull(row.nextRunTime) : '—', num: true }
    fields[5] = { label: t('systemV2.tasks.lastRun'), value: fmtFull(row.startTime), num: true }
  }
  if (row.target) fields.push({ label: t('systemV2.tasks.target'), value: row.target, mono: true, wide: true })
  return {
    stepsTitle: steps.title,
    steps: steps.steps,
    stepsDone: steps.done,
    more: steps.more,
    fields,
    logTitle: row.status === TaskStatus.TASK_STATUS_RUNNING ? t('systemV2.tasks.liveLog') : t('systemV2.tasks.taskLog'),
    logs: buildLogs(row, title),
    title,
  }
})
// 列表里与运行态有关的字段（比详情更新时覆盖）
const pickLive = (r: TaskInfo): Partial<TaskInfo> => {
  const live: Partial<TaskInfo> = {
    status: r.status,
    total: r.total,
    finished: r.finished,
    message: r.message,
    error: r.error,
    endTime: r.endTime,
    canceledByName: r.canceledByName,
  }
  if (r.steps?.length) live.steps = r.steps
  return live
}

const logText = () => (openView.value?.logs ?? []).map((l) => `${l.tm}  ${l.text}`).join('\n')
const copyLog = async () => {
  try {
    await copy(logText())
    feedback.success(t('system.operation.copied'))
  } catch {
    feedback.error(t('system.operation.copyFailed'))
  }
}
const downloadLog = () => {
  if (!openId.value) return
  const blob = new Blob([logText()], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `task-${shortId(openId.value)}-${dayjs().format('YYYYMMDD-HHmmss')}.log`
  a.click()
  URL.revokeObjectURL(url)
}

const loadDetail = async (id: string) => {
  try {
    const { data } = await getTask(id)
    if (openId.value === id && data?.task) detail.value = data.task
  } catch {
    /* 详情失败时退回列表数据 */
  }
}
const toggleOpen = (id: string) => {
  if (openId.value === id) {
    openId.value = null
    detail.value = null
    return
  }
  openId.value = id
  detail.value = null
  void loadDetail(id)
}
// 运行中的任务日志自动滚到底
watch(
  () => openView.value?.logs.length,
  () => {
    if (!openView.value || items.value.find((r) => r.id === openId.value)?.status !== TaskStatus.TASK_STATUS_RUNNING) return
    nextTick(() => {
      const el = Array.isArray(logBodyRef.value) ? logBodyRef.value[0] : logBodyRef.value
      if (el) el.scrollTop = el.scrollHeight
    })
  },
)

const heroHint = computed(() => {
  const etas = items.value.map(etaMs).filter((v): v is number => v != null)
  return etas.length
    ? t('systemV2.tasks.heroEta', { eta: fmtDuration(Math.max(...etas)) })
    : t('systemV2.tasks.heroHint')
})

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
const selRows = computed<RowView[]>(() => rows.value.filter((r) => selected.has(r.id)))
const selRetryable = computed(() => selRows.value.filter((r) => r.canRetry).map((r) => r.id))
const selCancelable = computed(() => selRows.value.filter((r) => r.canCancel).map((r) => r.id))
const selDeletable = computed(() => selRows.value.filter((r) => r.canDelete).map((r) => r.id))

// —— 加载 ——
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
    retentionDays.value = Number(data.retentionDays) || 0
    now.value = Date.now()
    // 已不在当前页的选中项与展开项剔除
    const ids = new Set(items.value.map((r) => r.id))
    for (const id of [...selected]) if (!ids.has(id)) selected.delete(id)
    if (openId.value && !ids.has(openId.value)) {
      openId.value = null
      detail.value = null
    }
  } catch (error) {
    if (!silent) showRequestError(error, t('taskManager.loadFailed'))
  } finally {
    if (!silent) loading.value = false
  }
}
const refreshOpen = () => {
  const id = openId.value
  if (!id) return
  const row = items.value.find((r) => r.id === id)
  // 运行中的任务持续拉日志；刚结束的再拉一次拿到终态日志
  if (row && (isActive(row.status) || (detail.value && detail.value.status !== row.status))) void loadDetail(id)
}

const reload = () => {
  page.value = 1
  getList()
}
watch(typeFilter, reload)

const tick = async () => {
  await getList(true)
  refreshOpen()
}
const startTimer = () => {
  stopTimer()
  timer = setInterval(tick, REFRESH_INTERVAL_MS)
}
const stopTimer = () => {
  if (timer) clearInterval(timer)
  timer = undefined
}
const toggleAuto = () => {
  autoRefresh.value = !autoRefresh.value
  if (autoRefresh.value) {
    void tick()
    startTimer()
  } else stopTimer()
}

// —— 操作 ——
const reportBatch = (res: TaskBatchResponse | undefined, okKey?: string) => {
  const ok = Number(res?.succeeded) || 0
  const failures = res?.failures ?? []
  if (failures.length) {
    feedback.warning(
      t('systemV2.tasks.bulkPartial', { ok, failed: failures.length, reason: failures[0]?.message || '' }),
    )
  } else {
    feedback.success(okKey ? t(okKey) : t('systemV2.tasks.bulkDone', { ok }))
  }
}
const runBatch = async (fn: (ids: string[]) => Promise<{ data: TaskBatchResponse }>, ids: string[], okKey?: string) => {
  if (!ids.length) return
  try {
    const { data } = await fn(ids)
    reportBatch(data, okKey)
    for (const id of ids) selected.delete(id)
    await getList(true)
    refreshOpen()
  } catch (error) {
    showRequestError(error, t('taskManager.actionFailed'))
  }
}
const cancelOne = (id: string) => runBatch(cancelTasks, [id], 'taskManager.cancelSuccess')
const retryOne = (id: string) => runBatch(retryTasks, [id], 'taskManager.retrySuccess')
const confirmDelete = (id: string) => {
  Dialog.info({
    showCancelButton: true,
    content: t('systemV2.tasks.deleteConfirm'),
    confirmText: t('system.common.delete'),
    cancelText: t('system.common.cancel'),
    onConfirm: () => runBatch(deleteTasks, [id], 'taskManager.deleteSuccess'),
  })
}
const confirmClear = () => {
  Dialog.info({
    showCancelButton: true,
    content: t('systemV2.tasks.clearConfirm', { n: finishedCount.value }),
    confirmText: t('systemV2.tasks.clearFinished'),
    cancelText: t('system.common.cancel'),
    onConfirm: async () => {
      try {
        const { data } = await clearTasks({})
        feedback.success(t('systemV2.tasks.clearSuccess', { n: Number(data?.deleted) || 0 }))
        selected.clear()
        reload()
      } catch (error) {
        showRequestError(error, t('taskManager.actionFailed'))
      }
    },
  })
}
const bulkRetry = () => runBatch(retryTasks, selRetryable.value)
const bulkCancel = () => runBatch(cancelTasks, selCancelable.value)
const bulkDelete = () => {
  const ids = selDeletable.value
  Dialog.info({
    showCancelButton: true,
    content: t('systemV2.tasks.bulkDeleteConfirm', { n: ids.length }),
    confirmText: t('system.common.delete'),
    cancelText: t('system.common.cancel'),
    onConfirm: () => runBatch(deleteTasks, ids),
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
  if (autoRefresh.value) startTimer()
})
// 页面被缓存（切到同组其它标签）时暂停轮询；回来时静默刷新一次再继续
onActivated(() => {
  if (timer || !autoRefresh.value) return
  void tick()
  startTimer()
})
onDeactivated(stopTimer)
onBeforeUnmount(stopTimer)
</script>

<style scoped lang="scss">
.tk-page {
  --tk-term-bg: #0f0b14;
  --tk-term-fg: #e8e3ee;
  --tk-term-dim: #9a90a6;
  --tk-term-line: #2a2333;
  --tk-term-ok: #7ee2b8;
  --tk-term-warn: #ffd98a;
  --tk-term-bad: #ffb3c1;
  --tk-term-muted: #b5abc2;
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
.tk-toolbar :deep(.ltb__search) {
  order: 2;
}
.tk-type {
  order: 1;
  flex: 0 0 auto;
  min-width: 150px;
}
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
  grid-template-columns: 32px minmax(0, 2.4fr) minmax(0, 1.5fr) minmax(0, 1.15fr) minmax(0, 1fr) minmax(0, 0.7fr) 184px;
  align-items: center;
  gap: 14px;
  min-width: 1040px;
  padding: 12px;
}
.tk-row--head {
  padding-top: 14px;
  padding-bottom: 14px;
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--el-text-color-secondary);
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.tk-item {
  border-bottom: 1px solid var(--el-border-color-lighter);
  transition: background 0.15s;
}
.tk-item:last-child {
  border-bottom: none;
}
.tk-item:hover {
  background: var(--el-fill-color-light);
}
.tk-item.is-sel {
  background: color-mix(in srgb, var(--m-pSoft) 70%, var(--el-bg-color));
}
.tk-item.is-open {
  margin: 6px 0;
  border-radius: var(--app-radius-lg);
  border-bottom-color: transparent;
  background: var(--el-fill-color-light);
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
  width: 20px;
  height: 20px;
  margin: 0;
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
.tk-sticker--lg {
  width: 44px;
  height: 44px;
  border-radius: 15px;
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
.tk-pill--run :deep(.status-pill__dot) {
  animation: tk-blink 1.2s ease-in-out infinite;
}
@keyframes tk-blink {
  50% {
    opacity: 0.25;
  }
}
.tk-state__sub,
.tk-card__sub {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.is-bad.tk-state__sub,
.is-bad.tk-card__sub {
  color: var(--m-bad);
}
.is-warn.tk-state__sub,
.is-warn.tk-card__sub {
  color: var(--m-warn);
}
.tk-bar {
  display: block;
  width: 100%;
  max-width: 150px;
  height: 6px;
  border-radius: var(--app-radius-pill);
  background: var(--el-border-color-light);
  overflow: hidden;
}
.tk-bar > span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--m-p);
  transition: width 0.3s ease;
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
.tk-avatar--sm {
  width: 20px;
  height: 20px;
  font-size: 0.625rem;
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
.tk-detail-btn {
  min-height: 36px;
  padding-inline: 12px;
  border-radius: var(--app-radius-pill);
  font-weight: 700;
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
  padding: 2px 12px 16px 58px;
}
.tk-detail__card {
  flex: 1 1 300px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: calc(var(--cmp-card-unit) * 18);
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
.tk-steps {
  list-style: none;
  margin: 0;
  padding: 0;
  max-height: 320px;
  overflow-y: auto;
  scrollbar-width: none;
}
.tk-steps::-webkit-scrollbar {
  display: none;
}
.tk-step {
  position: relative;
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding-bottom: 16px;
}
.tk-step:last-child {
  padding-bottom: 0;
}
.tk-step.has-line::after {
  content: '';
  position: absolute;
  left: 13px;
  top: 30px;
  bottom: 0;
  width: 2px;
  background: var(--el-border-color-light);
}
.tk-step__dot {
  position: relative;
  z-index: 1;
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 50%;
}
.tk-step__dot :deep(svg) {
  width: 16px;
  height: 16px;
  stroke-width: 2.6;
}
.tk-step__dot.is-ok {
  background: var(--m-okSoft);
  color: var(--m-ok);
}
.tk-step__dot.is-run {
  background: var(--m-p);
  color: var(--m-pOn);
}
.tk-step__dot.is-run :deep(svg) {
  animation: tk-spin 1.6s linear infinite;
}
.tk-step__dot.is-wait,
.tk-step__dot.is-skip {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  border: 1.5px dashed var(--el-border-color-light);
}
.tk-step__dot.is-fail {
  background: var(--m-badSoft);
  color: var(--m-bad);
}
@keyframes tk-spin {
  to {
    transform: rotate(360deg);
  }
}
.tk-step__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding-top: 4px;
}
.tk-step__title {
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  word-break: break-all;
}
.tk-step__title.m-mono {
  font-size: 0.8125rem;
}
.tk-step__sub {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  word-break: break-word;
}
.tk-step__sub.is-bad {
  color: var(--m-bad);
}
.tk-step__sub.is-p {
  color: var(--m-pInk);
}
.tk-step__dur {
  flex: none;
  padding-top: 6px;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.tk-step__more {
  padding-left: 40px;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.tk-dl {
  margin: 0;
  padding: 12px 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 12px;
}
.tk-dl--3 {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}
.tk-dl > div {
  min-width: 0;
}
.tk-dl .is-wide {
  grid-column: 1 / -1;
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
.tk-dl dd.m-mono {
  font-size: 0.75rem;
}
/* 时间在窄格里换到第二行显示，不截断 */
.tk-dl dd.m-num {
  white-space: normal;
}
.tk-dl__by {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 4px !important;
}
.tk-dl__by > span:last-child {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* —— 日志（固定终端配色）—— */
.tk-log {
  flex: 2 1 440px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-radius: var(--app-radius-lg);
  background: var(--tk-term-bg);
  color: var(--tk-term-fg);
}
.tk-log__head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 8px 8px 16px;
  border-bottom: 1px solid var(--tk-term-line);
}
.tk-log__head > :deep(svg) {
  width: 16px;
  height: 16px;
  flex: none;
}
.tk-log__title {
  flex: 1;
  font-size: 0.8125rem;
  font-weight: 700;
}
.tk-log__count {
  font-size: 0.75rem;
  color: var(--tk-term-dim);
}
.tk-log__btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: none;
  border-radius: 10px;
  background: transparent;
  color: var(--tk-term-fg);
  cursor: pointer;
}
.tk-log__btn:hover {
  background: var(--tk-term-line);
}
.tk-log__btn:focus-visible {
  outline: 3px solid var(--m-a2);
  outline-offset: 2px;
}
.tk-log__btn :deep(svg) {
  width: 16px;
  height: 16px;
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
  margin-right: 12px;
  color: var(--tk-term-dim);
}
.tk-log .is-ink {
  color: var(--tk-term-fg);
}
.tk-log .is-dim {
  color: var(--tk-term-muted);
}
.tk-log .is-ok {
  color: var(--tk-term-ok);
}
.tk-log .is-bad {
  color: var(--tk-term-bad);
}
.tk-log .is-warn {
  color: var(--tk-term-warn);
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
  height: 190px;
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
  gap: 12px;
  padding: calc(var(--cmp-card-unit) * 16) calc(var(--cmp-card-unit) * 12) calc(var(--cmp-card-unit) * 6) calc(var(--cmp-card-unit) * 16);
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.tk-card > :not(.tk-card__foot) {
  margin-right: 4px;
}
.tk-card.is-sel {
  border-color: var(--m-p);
}
.tk-card__head {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}
.tk-card__title {
  font-size: 1rem;
  font-weight: 800;
  line-height: 1.3;
  color: var(--el-text-color-primary);
  word-break: break-all;
}
.tk-card__id {
  font-size: 0.6875rem;
  color: var(--el-text-color-secondary);
}
.tk-card__prog {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.tk-card__bar {
  display: flex;
  align-items: center;
  gap: 10px;
}
.tk-card__bar .tk-bar {
  flex: 1;
  max-width: none;
  height: 8px;
  background: var(--el-fill-color-light);
}
.tk-card__bar b {
  font-size: 0.9375rem;
  font-weight: 800;
}
.tk-card__progtext {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.tk-card__sub {
  font-size: 0.8125rem;
  white-space: normal;
}
.tk-card__detail {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.tk-card__detail .tk-detail__head h3 {
  font-size: 0.875rem;
}
.tk-log--card {
  flex: none;
  border-radius: 18px;
}
.tk-log--card .tk-log__head {
  padding: 4px 4px 4px 14px;
}
.tk-log--card .tk-log__body {
  max-height: 220px;
  padding: 10px 14px 12px;
  font-size: 0.72rem;
  line-height: 1.7;
}
.tk-log--card .tk-log__btn {
  width: 44px;
  height: 44px;
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
  gap: 10px;
  min-height: 44px;
}
.tk-tag {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 9px;
  border-radius: var(--app-radius-xs);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  font-weight: 600;
  white-space: nowrap;
}

/* —— 页脚 —— */
.tk-foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}
.tk-foot__text {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.tk-foot__pager {
  flex: 1 1 auto;
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
  box-shadow: var(--app-shadow-lg);
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
  font-size: 0.875rem;
  font-weight: 600;
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
    order: 4;
  }
  .tk-type :deep(button) {
    min-height: 44px;
  }
  .tk-foot {
    flex-direction: column;
    align-items: stretch;
  }
  .tk-foot__text {
    text-align: center;
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
