<!-- 智商检测 · 检测站点：标题（站点数 · 模型数 · 今天已检测次数）+ 新建站点；子导航（站点 / 提示词）；
     状态 chips（带数量）+ 搜索 + 卡片/列表切换（桌面可切换并记住，手机固定卡片）。
     卡片：贴纸头像 + 名称/协议/端点 + 状态；上次结果（成功数/模型数、平均耗时、首个失败原因、近 8 轮成功率走势）；
     检测模型 · 推理强度；间隔/上次/下次/次数/结果数/Key 信息格；公示页链接；立即检测 + 编辑 + ⋯（复制公示页链接 / 删除）。
     子弹窗：siteForm（创建编辑，含拉取模型列表勾选）。有站点正在检测时每 8 秒静默刷新。 -->
<template>
  <div class="iq-page">
    <PageHeader :title="t('iqcheck.nav.title')">
      <template #description>
        {{ t('iqcheck.nav.desc') }} ·
        {{ t('iqcheck.siteList.summary', { sites: stats.total, models: stats.modelCount }) }}
        <template v-if="stats.checksToday">
          · <span class="iq-page__ok">{{ t('iqcheck.siteList.checksToday', { count: stats.checksToday }) }}</span>
        </template>
      </template>
      <template #actions>
        <UButton v-permission="[PERM.IQCHECK_EDIT]" color="primary" icon="i-lucide-plus" @click="openForm()">
          {{ t('iqcheck.site.create') }}
        </UButton>
      </template>
    </PageHeader>

    <IqSubNav :site-count="stats.total" />

    <ListToolbar
      v-model:status="filter"
      v-model:keyword="keyword"
      :chips="chips"
      :chips-label="t('iqcheck.siteList.filterLabel')"
      :placeholder="t('iqcheck.siteList.searchPlaceholder')"
      @search="reload"
    >
      <div v-if="!menuStore.isMobile" class="iq-seg" role="group" :aria-label="t('iqcheck.siteList.view')">
        <button
          type="button"
          class="iq-seg__btn"
          :class="{ 'is-on': viewMode === 'card' }"
          :aria-pressed="viewMode === 'card'"
          :aria-label="t('iqcheck.site.viewCard')"
          :title="t('iqcheck.site.viewCard')"
          @click="setView('card')"
        >
          <UIcon name="i-lucide-layout-grid" />
        </button>
        <button
          type="button"
          class="iq-seg__btn"
          :class="{ 'is-on': viewMode === 'table' }"
          :aria-pressed="viewMode === 'table'"
          :aria-label="t('iqcheck.site.viewTable')"
          :title="t('iqcheck.site.viewTable')"
          @click="setView('table')"
        >
          <UIcon name="i-lucide-list" />
        </button>
      </div>
    </ListToolbar>

    <!-- 列表视图 -->
    <DataTable
      v-if="!useCards"
      :columns="columns"
      :rows="sites as unknown as Record<string, unknown>[]"
      row-key="id"
      :loading="loading"
      :empty-text="t('iqcheck.site.emptyTitle')"
    >
      <template #cell-name="{ row }">
        <div class="iq-cell-name">{{ asSite(row).name }}</div>
        <div class="iq-cell-ep m-mono" :title="asSite(row).endpoint">{{ asSite(row).endpoint }}</div>
      </template>
      <template #cell-protocol="{ row }">
        <span class="iq-proto">{{ protocolLabel(asSite(row).protocol) }}</span>
      </template>
      <template #cell-models="{ row }">
        <div class="iq-models is-tight">
          <span v-for="m in asSite(row).models" :key="m.model + m.effort" class="iq-mdl">
            <span class="m-mono">{{ m.model }}</span>
            <span v-if="m.effort" class="iq-eff" :class="effClass(m.effort)">{{ m.effort }}</span>
          </span>
          <span v-if="!asSite(row).models.length" class="iq-dim">—</span>
        </div>
      </template>
      <template #cell-interval="{ row }">
        <span class="m-num iq-strong">{{ intervalText(asSite(row).intervalMinutes) }}</span>
      </template>
      <template #cell-last="{ row }">
        <span class="m-num">{{ fmtRel(asSite(row).lastCheckAt) }}</span>
      </template>
      <template #cell-result="{ row }">
        <div class="iq-cell-result">
          <span v-if="asSite(row).lastRound" class="m-num iq-cell-score" :style="{ color: scoreColor(asSite(row)) }">
            {{ asSite(row).lastRound!.success }}/{{ roundTotal(asSite(row).lastRound!) }}
          </span>
          <StatusPill :variant="healthPill(asSite(row)).variant" :label="healthPill(asSite(row)).label" />
        </div>
      </template>
      <template #cell-public="{ row }">
        <a class="iq-link m-mono" :href="publicUrl(asSite(row).slug)" target="_blank" rel="noopener">/{{ asSite(row).slug }}</a>
      </template>
      <template #cell-operation="{ row }">
        <div class="iq-ops">
          <template v-if="canEdit">
            <AppIconButton
              icon="HOutline:PlayIcon"
              class="is-primary"
              :label="t('iqcheck.siteList.checkNowOf', { name: asSite(row).name })"
              :box="36"
              :size="18"
              :disabled="asSite(row).checking || !asSite(row).models.length"
              @click="checkNow(asSite(row))"
            />
            <AppIconButton
              icon="HOutline:PencilSquareIcon"
              :label="t('iqcheck.siteList.editOf', { name: asSite(row).name })"
              :box="36"
              :size="18"
              @click="openForm(asSite(row))"
            />
          </template>
          <ActionMenu :items="moreActions" @select="(key) => onRowAction(key, asSite(row))" />
        </div>
      </template>
    </DataTable>

    <!-- 卡片视图 -->
    <template v-else>
      <div v-if="loading && !sites.length" class="iq-grid">
        <div v-for="i in 4" :key="i" class="iq-skeleton" />
      </div>
      <EmptyState
        v-else-if="!sites.length"
        icon="HOutline:AdjustmentsHorizontalIcon"
        :title="stats.total ? t('iqcheck.siteList.noMatch') : t('iqcheck.site.emptyTitle')"
        :description="stats.total ? t('iqcheck.siteList.noMatchDesc') : t('iqcheck.site.emptyDesc')"
      >
        <template v-if="!stats.total && canEdit" #action>
          <UButton color="primary" variant="soft" icon="i-lucide-plus" @click="openForm()">{{ t('iqcheck.site.create') }}</UButton>
        </template>
      </EmptyState>
      <div v-else class="iq-grid">
        <article v-for="s in sites" :key="s.id" class="iq-card">
          <header class="iq-card__head">
            <span class="iq-card__av m-sticker" :class="`m-sticker--${toneOf(s.id)}`" :style="{ transform: `rotate(${rotOf(s.id)})` }">
              {{ initialOf(s.name) }}
            </span>
            <span class="iq-card__title">
              <span class="iq-card__namerow">
                <span class="iq-card__name" :title="s.name">{{ s.name }}</span>
                <span class="iq-proto">{{ protocolLabel(s.protocol) }}</span>
                <span v-if="!s.enabled" class="iq-tag">{{ t('iqcheck.siteList.disabled') }}</span>
              </span>
              <span class="iq-card__ep m-mono" :title="s.endpoint">{{ s.endpoint }}</span>
            </span>
            <StatusPill :variant="healthPill(s).variant" :label="healthPill(s).label" />
          </header>

          <div class="iq-card__score">
            <div class="iq-card__score-main">
              <span class="iq-card__label">{{ t('iqcheck.siteList.lastResult') }}</span>
              <span class="iq-card__score-val">
                <span class="m-num" :style="{ color: scoreColor(s) }">
                  {{ s.lastRound ? `${s.lastRound.success}/${roundTotal(s.lastRound)}` : '—' }}
                </span>
                <span class="iq-card__unit">{{ t('iqcheck.siteList.successUnit') }}</span>
              </span>
            </div>
            <div class="iq-card__score-note">
              <span class="m-num iq-strong" :class="deltaClass(s)">{{ deltaText(s) }}</span>
              <span class="iq-card__note" :title="noteText(s)">{{ noteText(s) }}</span>
            </div>
            <svg
              v-if="s.recentRounds.length > 1"
              class="iq-card__spark"
              viewBox="0 0 120 40"
              role="img"
              :aria-label="t('iqcheck.siteList.trend', { n: s.recentRounds.length })"
            >
              <polyline :points="sparkPoints(s)" :style="{ stroke: scoreColor(s) }" />
            </svg>
          </div>

          <div class="iq-card__block">
            <span class="iq-card__label">{{ t('iqcheck.siteList.modelsEffort') }}</span>
            <div class="iq-models">
              <span v-for="m in s.models" :key="m.model + m.effort" class="iq-mdl">
                <span class="m-mono">{{ m.model }}</span>
                <span class="iq-eff" :class="effClass(m.effort)">{{ m.effort || t('iqcheck.public.noEffort') }}</span>
              </span>
              <span v-if="!s.models.length" class="iq-dim">{{ t('iqcheck.site.form.noModels') }}</span>
            </div>
          </div>

          <dl class="iq-card__info">
            <div>
              <dt>{{ t('iqcheck.siteList.interval') }}</dt>
              <dd class="m-num">{{ intervalText(s.intervalMinutes) }}</dd>
            </div>
            <div>
              <dt>{{ t('iqcheck.siteList.lastRun') }}</dt>
              <dd class="m-num">{{ s.checking ? t('iqcheck.site.checking') : fmtRel(s.lastCheckAt) }}</dd>
            </div>
            <div>
              <dt>{{ t('iqcheck.siteList.nextRun') }}</dt>
              <dd class="m-num">{{ nextText(s) }}</dd>
            </div>
            <div>
              <dt>{{ t('iqcheck.site.colRuns') }}</dt>
              <dd class="m-num">{{ runsLabel(s) }}</dd>
            </div>
            <div>
              <dt>{{ t('iqcheck.siteList.results') }}</dt>
              <dd class="m-num">{{ t('iqcheck.siteList.resultsCount', { count: s.resultCount }) }}</dd>
            </div>
            <div>
              <dt>API Key</dt>
              <dd class="m-mono" :title="s.apiKey">{{ s.apiKey || '—' }}</dd>
            </div>
          </dl>

          <a class="iq-card__public" :href="publicUrl(s.slug)" target="_blank" rel="noopener">
            <UIcon name="i-lucide-globe" />
            <span class="iq-card__public-l">{{ t('iqcheck.site.colPublic') }}</span>
            <span class="iq-card__public-p m-mono">/public/iqcheck/{{ s.slug }}</span>
            <UIcon name="i-lucide-external-link" />
          </a>

          <footer class="iq-card__foot">
            <UButton
              v-if="canEdit"
              color="primary"
              size="sm"
              class="iq-card__run"
              :icon="s.checking ? 'i-lucide-loader-circle' : 'i-lucide-play'"
              :disabled="s.checking || !s.models.length"
              :ui="s.checking ? { leadingIcon: 'animate-spin' } : undefined"
              @click="checkNow(s)"
            >
              {{ s.checking ? t('iqcheck.site.checking') : t('iqcheck.site.checkNow') }}
            </UButton>
            <span class="iq-card__spacer" />
            <AppIconButton
              v-if="canEdit"
              icon="HOutline:PencilSquareIcon"
              :label="t('iqcheck.siteList.editOf', { name: s.name })"
              :box="menuStore.isMobile ? 44 : 36"
              :size="18"
              @click="openForm(s)"
            />
            <ActionMenu :items="moreActions" :label="t('iqcheck.siteList.moreOf', { name: s.name })" @select="(key) => onRowAction(key, s)" />
          </footer>
        </article>
      </div>
    </template>

    <Pagination
      v-if="pagination.total > pagination.pageSize"
      v-model:page="pagination.page"
      v-model:page-size="pagination.pageSize"
      :total="pagination.total"
      @change="loadSites()"
    />

    <SiteForm ref="formRef" @refresh="loadSites()" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import dayjs from 'dayjs'
import { listIQCheckSites, deleteIQCheckSite, triggerIQCheck } from '@/api/iqcheck'
import { PERM } from '@/config/permission'
import { Dialog } from '@/utils/dialog'
import {
  IQCheckProtocol,
  IQCheckSiteFilter,
  IQCheckSiteHealth,
  type IQCheckRoundSummary,
  type IQCheckSiteInfo,
  type IQCheckSiteStats,
} from '@/types/v1/iqcheck'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import type { ToolbarChip } from '@/components/ui/ListToolbar.vue'
import SiteForm from '@/views/iqcheck/site/siteForm.vue'
import IqSubNav from '@/views/iqcheck/components/IqSubNav.vue'

defineOptions({ name: 'IQCheckSiteView' })

const menuStore = useMenuStore()
const router = useRouter()
const { t } = useI18n()

const formRef = useTemplateRef<InstanceType<typeof SiteForm> | null>('formRef')
const canEdit = computed(() => menuStore.hasButtonPermission(PERM.IQCHECK_EDIT))

// 视图偏好：桌面端可在卡片/列表之间切换并记住；移动端固定卡片。
type ViewMode = 'table' | 'card'
const VIEW_STORAGE_KEY = 'iqcheck-site-view'
const readView = (): ViewMode => {
  try {
    return localStorage.getItem(VIEW_STORAGE_KEY) === 'table' ? 'table' : 'card'
  } catch {
    return 'card'
  }
}
const viewMode = ref<ViewMode>(readView())
const setView = (mode: ViewMode) => {
  viewMode.value = mode
  try {
    localStorage.setItem(VIEW_STORAGE_KEY, mode)
  } catch {
    // 浏览器禁用存储时忽略
  }
}
const useCards = computed(() => menuStore.isMobile || viewMode.value === 'card')

const loading = ref(false)
const sites = ref<IQCheckSiteInfo[]>([])
const keyword = ref('')
const filter = ref<IQCheckSiteFilter>(IQCheckSiteFilter.IQ_CHECK_SITE_FILTER_ALL)
const pagination = ref({ page: 1, pageSize: 12, total: 0 })

const emptyStats = (): IQCheckSiteStats => ({
  total: 0,
  healthy: 0,
  failing: 0,
  disabled: 0,
  never: 0,
  checking: 0,
  modelCount: 0,
  checksToday: 0,
})
const stats = ref<IQCheckSiteStats>(emptyStats())

const chips = computed<ToolbarChip<IQCheckSiteFilter>[]>(() => {
  const s = stats.value
  const out: ToolbarChip<IQCheckSiteFilter>[] = [
    { value: IQCheckSiteFilter.IQ_CHECK_SITE_FILTER_ALL, label: t('iqcheck.siteList.chipAll'), count: Number(s.total) },
    { value: IQCheckSiteFilter.IQ_CHECK_SITE_FILTER_HEALTHY, label: t('iqcheck.siteList.chipHealthy'), count: Number(s.healthy) },
    { value: IQCheckSiteFilter.IQ_CHECK_SITE_FILTER_FAILING, label: t('iqcheck.siteList.chipFailing'), count: Number(s.failing) },
  ]
  if (Number(s.never)) out.push({ value: IQCheckSiteFilter.IQ_CHECK_SITE_FILTER_NEVER, label: t('iqcheck.siteList.chipNever'), count: Number(s.never) })
  out.push({ value: IQCheckSiteFilter.IQ_CHECK_SITE_FILTER_DISABLED, label: t('iqcheck.siteList.chipDisabled'), count: Number(s.disabled) })
  return out
})

const columns = computed<DataTableColumn[]>(() => [
  { key: 'name', title: t('iqcheck.siteList.colSite'), minWidth: 220 },
  { key: 'protocol', title: t('iqcheck.site.colProtocol'), width: 110 },
  { key: 'models', title: t('iqcheck.siteList.colModels'), minWidth: 240 },
  { key: 'interval', title: t('iqcheck.site.colInterval'), width: 120 },
  { key: 'last', title: t('iqcheck.siteList.lastRun'), width: 130 },
  { key: 'result', title: t('iqcheck.siteList.lastResult'), width: 170 },
  { key: 'public', title: t('iqcheck.site.colPublic'), width: 140 },
  { key: 'operation', title: t('system.common.operation'), width: 140, align: 'right' },
])

const asSite = (row: Record<string, unknown>) => row as unknown as IQCheckSiteInfo

const moreActions = computed<ActionMenuItem[]>(() => [
  { key: 'open', label: t('iqcheck.site.openPublic'), icon: 'i-lucide-external-link' },
  { key: 'copy', label: t('iqcheck.site.copyPublic'), icon: 'i-lucide-link' },
  { key: 'delete', label: t('system.common.delete'), icon: 'HOutline:TrashIcon', danger: true, hidden: !canEdit.value },
])

// —— 展示 ——
const TONES = ['p', 'a2', 'a', 'ok']
const ROTS = ['-4deg', '3deg', '-2deg', '4deg']
const hashOf = (id: string) => [...id].reduce((n, c) => (n * 31 + c.charCodeAt(0)) >>> 0, 7)
const toneOf = (id: string) => TONES[hashOf(id) % TONES.length]
const rotOf = (id: string) => ROTS[hashOf(id) % ROTS.length]
const initialOf = (name: string) => (name || '?').trim().charAt(0).toUpperCase()

const protocolLabels: Record<string, string> = {
  [IQCheckProtocol.IQ_CHECK_PROTOCOL_CHAT]: 'chat',
  [IQCheckProtocol.IQ_CHECK_PROTOCOL_RESPONSES]: 'responses',
  [IQCheckProtocol.IQ_CHECK_PROTOCOL_MESSAGES]: 'messages',
}
const protocolLabel = (p: IQCheckProtocol) => protocolLabels[p] || '—'

const effClass = (effort: string) => {
  const e = (effort || '').toLowerCase()
  if (['high', 'xhigh', 'max'].includes(e)) return 'is-high'
  if (e === 'medium') return 'is-medium'
  return 'is-low'
}

const roundTotal = (r: IQCheckRoundSummary) => Number(r.success) + Number(r.failed)
const ratioOf = (r: IQCheckRoundSummary) => (roundTotal(r) ? Number(r.success) / roundTotal(r) : 0)

const healthPill = (s: IQCheckSiteInfo): { variant: 'success' | 'warning' | 'error' | 'neutral'; label: string } => {
  if (s.checking) return { variant: 'warning', label: t('iqcheck.site.checking') }
  switch (s.health) {
    case IQCheckSiteHealth.IQ_CHECK_SITE_HEALTH_HEALTHY:
      return { variant: 'success', label: t('iqcheck.siteList.healthy') }
    case IQCheckSiteHealth.IQ_CHECK_SITE_HEALTH_PARTIAL:
      return { variant: 'warning', label: t('iqcheck.siteList.partial') }
    case IQCheckSiteHealth.IQ_CHECK_SITE_HEALTH_FAILED:
      return { variant: 'error', label: t('iqcheck.siteList.failed') }
    default:
      return { variant: 'neutral', label: t('iqcheck.siteList.never') }
  }
}

const scoreColor = (s: IQCheckSiteInfo) => {
  switch (s.health) {
    case IQCheckSiteHealth.IQ_CHECK_SITE_HEALTH_HEALTHY:
      return 'var(--m-ok)'
    case IQCheckSiteHealth.IQ_CHECK_SITE_HEALTH_PARTIAL:
      return 'var(--m-warn)'
    case IQCheckSiteHealth.IQ_CHECK_SITE_HEALTH_FAILED:
      return 'var(--m-bad)'
    default:
      return 'var(--el-text-color-secondary)'
  }
}

const seconds = (ms: number) => (Number(ms) / 1000).toFixed(1)

// 「与上一轮相比」：成功率变化；只有一轮时显示平均耗时
const deltaText = (s: IQCheckSiteInfo) => {
  const rounds = s.recentRounds
  const last = s.lastRound
  if (!last) return t('iqcheck.siteList.noResultYet')
  if (rounds.length >= 2) {
    const prev = rounds[rounds.length - 2]!
    const diff = Math.round((ratioOf(last) - ratioOf(prev)) * 100)
    if (diff > 0) return t('iqcheck.siteList.deltaUp', { n: diff })
    if (diff < 0) return t('iqcheck.siteList.deltaDown', { n: -diff })
    return t('iqcheck.siteList.deltaSame')
  }
  return t('iqcheck.siteList.firstRound')
}
const deltaClass = (s: IQCheckSiteInfo) => {
  const rounds = s.recentRounds
  if (!s.lastRound || rounds.length < 2) return ''
  const diff = ratioOf(s.lastRound) - ratioOf(rounds[rounds.length - 2]!)
  return diff < 0 ? 'is-bad' : diff > 0 ? 'is-ok' : ''
}
const noteText = (s: IQCheckSiteInfo) => {
  const last = s.lastRound
  if (!last) return s.models.length ? t('iqcheck.siteList.waitFirst') : t('iqcheck.site.form.noModels')
  if (Number(last.failed) && last.error) return last.error
  const parts = [t('iqcheck.siteList.avgDuration', { s: seconds(last.avgDurationMs) })]
  if (last.promptTitle) parts.push(last.promptTitle)
  return parts.join(' · ')
}

const sparkPoints = (s: IQCheckSiteInfo) => {
  const h = s.recentRounds.map(ratioOf)
  const step = 116 / Math.max(h.length - 1, 1)
  return h.map((v, k) => `${(k * step + 2).toFixed(1)},${(36 - v * 32).toFixed(1)}`).join(' ')
}

const intervalText = (minutes: number) => {
  const m = Number(minutes) || 0
  if (m > 0 && m % 1440 === 0) return t('iqcheck.siteList.everyDays', { n: m / 1440 })
  if (m > 0 && m % 60 === 0) return t('iqcheck.siteList.everyHours', { n: m / 60 })
  return t('iqcheck.siteList.everyMinutes', { n: m })
}

const runsLabel = (site: IQCheckSiteInfo) =>
  site.maxRuns > 0
    ? t('iqcheck.site.runsLimited', { count: site.runCount, max: site.maxRuns })
    : t('iqcheck.site.runsUnlimited', { count: site.runCount })

const fmtRel = (value: unknown): string => {
  if (!value) return t('iqcheck.public.never')
  const d = dayjs(value as string | Date)
  if (!d.isValid()) return '—'
  const now = dayjs()
  const time = d.format('HH:mm')
  if (d.isSame(now, 'day')) return t('iqcheck.siteList.todayAt', { time })
  if (d.isSame(now.subtract(1, 'day'), 'day')) return t('iqcheck.siteList.yesterdayAt', { time })
  if (d.isSame(now.add(1, 'day'), 'day')) return t('iqcheck.siteList.tomorrowAt', { time })
  return d.isSame(now, 'year') ? d.format('MM-DD HH:mm') : d.format('YYYY-MM-DD')
}
const nextText = (s: IQCheckSiteInfo) => {
  if (!s.enabled) return t('iqcheck.siteList.disabled')
  if (s.maxRuns > 0 && s.runCount >= s.maxRuns) return t('iqcheck.siteList.limitReached')
  if (!s.nextCheckAt) return '—'
  return dayjs(s.nextCheckAt as unknown as string).diff(dayjs(), 'minute') < 1
    ? t('iqcheck.siteList.soon')
    : fmtRel(s.nextCheckAt)
}

// 公示页地址：前端固定路由前缀 /public/iqcheck + 站点随机路径。
const publicUrl = (slug: string) => {
  const href = router.resolve({ name: 'PublicIQCheck', params: { slug } }).href
  return new URL(href, window.location.origin).href
}

// —— 数据 ——
let pollTimer: number | undefined
const schedulePoll = () => {
  window.clearTimeout(pollTimer)
  if (sites.value.some((s) => s.checking)) {
    pollTimer = window.setTimeout(() => loadSites(true), 8000)
  }
}

const loadSites = async (silent = false) => {
  if (!silent) loading.value = true
  try {
    const { data } = await listIQCheckSites({
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      keywords: keyword.value.trim() || undefined,
      filter: filter.value,
    })
    sites.value = data?.sites || []
    pagination.value.total = Number(data?.total || 0)
    stats.value = { ...emptyStats(), ...(data?.stats || {}) }
    schedulePoll()
  } finally {
    if (!silent) loading.value = false
  }
}

const reload = () => {
  pagination.value.page = 1
  loadSites()
}

const openForm = (row?: IQCheckSiteInfo) => formRef.value?.showDialog(row)

const onRowAction = (key: string, site: IQCheckSiteInfo) => {
  if (key === 'edit') openForm(site)
  else if (key === 'check') checkNow(site)
  else if (key === 'copy') copyPublic(site)
  else if (key === 'open') window.open(publicUrl(site.slug), '_blank', 'noopener')
  else if (key === 'delete') confirmDelete(site)
}

const checkNow = async (site: IQCheckSiteInfo) => {
  const { data } = await triggerIQCheck({ id: site.id })
  feedback.success(t('iqcheck.site.checkStarted', { count: data?.modelCount ?? site.models.length }))
  loadSites(true)
}

const copyPublic = async (site: IQCheckSiteInfo) => {
  await navigator.clipboard.writeText(publicUrl(site.slug))
  feedback.success(t('iqcheck.site.copied'))
}

const confirmDelete = (site: IQCheckSiteInfo) => {
  Dialog.info({
    showCancelButton: true,
    content: t('iqcheck.site.deleteConfirm', { name: site.name }),
    confirmText: t('common.delete'),
    cancelText: t('common.cancel'),
    onConfirm: async () => {
      await deleteIQCheckSite({ id: site.id })
      feedback.success(t('common.deleteSuccess'))
      loadSites()
    },
  })
}

onMounted(() => loadSites())
onBeforeUnmount(() => window.clearTimeout(pollTimer))
</script>

<style scoped lang="scss">
.iq-page {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.iq-page__ok {
  color: var(--m-ok);
  font-weight: 700;
}
.iq-strong {
  font-weight: 700;
}
.iq-dim {
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
}

/* 视图切换 */
.iq-seg {
  display: inline-flex;
  gap: 2px;
  padding: 4px;
  border-radius: var(--app-radius-pill);
  background: var(--el-fill-color-light);
}
.iq-seg__btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 40px;
  min-height: 34px;
  padding: 0 10px;
  border: 0;
  border-radius: var(--app-radius-pill);
  background: transparent;
  color: var(--el-text-color-secondary);
  cursor: pointer;
}
.iq-seg__btn.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: 0 1px 3px rgba(20, 10, 30, 0.12);
}
.iq-seg__btn :deep(.iconify) {
  width: 16px;
  height: 16px;
}

/* 协议 / 模型 / 推理强度 */
.iq-proto {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 9px;
  border-radius: 8px;
  background: var(--m-a2Soft);
  color: var(--m-ink);
  font-family: var(--app-font-mono);
  font-size: 0.75rem;
  font-weight: 700;
  white-space: nowrap;
}
.iq-tag {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  border-radius: 7px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.6875rem;
  font-weight: 700;
  white-space: nowrap;
}
.iq-models {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.iq-models.is-tight {
  gap: 4px;
  max-width: 300px;
}
.iq-mdl {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  height: 30px;
  padding: 0 4px 0 10px;
  border-radius: 10px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  font-size: 0.8125rem;
  font-weight: 700;
  white-space: nowrap;
}
.iq-mdl .m-mono {
  overflow: hidden;
  text-overflow: ellipsis;
}
.iq-models.is-tight .iq-mdl {
  height: 26px;
  font-size: 0.75rem;
}
.iq-eff {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 7px;
  border-radius: 7px;
  font-size: 0.6875rem;
  font-weight: 800;
  letter-spacing: 0.02em;
  flex: none;
}
.iq-eff.is-high {
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.iq-eff.is-medium {
  background: var(--m-aSoft);
  color: var(--m-ink);
}
.iq-eff.is-low {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
}

/* 表格单元格 */
.iq-cell-name {
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.iq-cell-ep {
  max-width: 260px;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.iq-cell-result {
  display: flex;
  align-items: center;
  gap: 8px;
}
.iq-cell-score {
  font-family: var(--app-font-display);
  font-size: 1.125rem;
  font-weight: 800;
}
.iq-link {
  color: var(--m-pInk);
  font-size: 0.75rem;
  font-weight: 600;
  text-decoration: none;
}
.iq-link:hover {
  text-decoration: underline;
}
.iq-ops {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 2px;
}
.iq-ops :deep(.is-primary),
.iq-card__foot :deep(.is-primary) {
  color: var(--m-pInk);
}

/* 卡片 */
.iq-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 440px), 1fr));
  gap: 16px;
}
.iq-card {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
  padding: 20px;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-xl);
  background: var(--el-bg-color);
}
.iq-card__head {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  min-width: 0;
}
.iq-card__av {
  width: 48px;
  height: 48px;
  border-radius: 16px;
  font-size: 1.25rem;
  flex: none;
}
.iq-card__title {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.iq-card__namerow {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.iq-card__name {
  min-width: 0;
  font-family: var(--app-font-display);
  font-size: 1.0625rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.iq-card__ep {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.iq-card__label {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.iq-card__score {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 14px;
  border-radius: 16px;
  background: var(--el-fill-color-light);
}
.iq-card__score-main {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 96px;
}
.iq-card__score-val {
  display: flex;
  align-items: baseline;
  gap: 4px;
}
.iq-card__score-val .m-num {
  font-family: var(--app-font-display);
  font-size: 2rem;
  font-weight: 800;
  line-height: 1;
}
.iq-card__unit {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.iq-card__score-note {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 0.8125rem;
}
.iq-card__score-note .is-ok {
  color: var(--m-ok);
}
.iq-card__score-note .is-bad {
  color: var(--m-bad);
}
.iq-card__note {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.iq-card__spark {
  flex: none;
  width: 120px;
  height: 40px;
}
.iq-card__spark polyline {
  fill: none;
  stroke-width: 2.5;
  stroke-linecap: round;
  stroke-linejoin: round;
}
.iq-card__block {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.iq-card__info {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px 12px;
  margin: 0;
}
.iq-card__info > div {
  min-width: 0;
}
.iq-card__info dt {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.iq-card__info dd {
  margin: 2px 0 0;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.iq-card__info dd.m-mono {
  font-size: 0.75rem;
  font-weight: 500;
}
.iq-card__public {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  padding: 0 12px;
  border-radius: 12px;
  border: 1.5px dashed var(--el-border-color-light);
  color: var(--m-pInk);
  font-size: 0.8125rem;
  text-decoration: none;
}
.iq-card__public:hover {
  border-color: var(--m-p);
}
.iq-card__public :deep(.iconify) {
  width: 16px;
  height: 16px;
  flex: none;
}
.iq-card__public-l {
  font-weight: 700;
  color: var(--el-text-color-primary);
  white-space: nowrap;
}
.iq-card__public-p {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.iq-card__foot {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: auto;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.iq-card__run {
  border-radius: var(--app-radius-pill);
  min-height: 36px;
  padding: 0 14px;
}
.iq-card__spacer {
  flex: 1;
}
.iq-skeleton {
  height: 420px;
  border-radius: var(--app-radius-xl);
  background: var(--el-fill-color-light);
}

@media (width <= 768px) {
  .iq-page {
    gap: 14px;
  }
  .iq-grid {
    gap: 12px;
  }
  .iq-card {
    padding: 16px 14px 8px 16px;
    gap: 12px;
  }
  .iq-card__score {
    gap: 12px;
  }
  .iq-card__spark {
    width: 80px;
  }
  .iq-card__info {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    padding: 10px 12px;
    border-radius: 14px;
    background: var(--el-fill-color-light);
  }
  .iq-card__run {
    min-height: 44px;
  }
  .iq-card__foot :deep(.action-menu__trigger) {
    width: 44px;
    height: 44px;
    border-radius: 14px;
  }
}
</style>
