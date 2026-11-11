<!-- 智商检测 · 检测站点：标题（标语 · 站点数 · 模型数 · 今天已检测次数）+ 新建站点（手机为右下悬浮按钮）；
     子导航（站点 / 提示词，手机为分段切换）；状态 chips（全部 / 正常 / 部分失败 / 检测失败 / 未检测 / 已停用，带数量）+ 搜索 + 卡片/列表切换（记住偏好）。
     卡片：贴纸头像 + 名称/协议/端点 + 状态；最近一轮（成功数/结果数、较上轮变化或上次成功时间、本轮规模/失败原因、近 8 轮成功率走势）；
     检测模型 · 推理强度；间隔/上次/下次/次数/结果数/Key 信息格；公示页链接；立即检测 + 拉取模型 + 编辑 + ⋯（打开/复制公示页链接 / 删除）。
     桌面列表视图为表格；手机列表视图为紧凑行。子弹窗：siteForm（创建编辑，含拉取模型列表勾选）。有站点正在检测时每 8 秒静默刷新。 -->
<template>
  <div class="iq-page">
    <PageHeader :title="t('iqcheck.nav.title')">
      <template #description>
        <template v-if="!menuStore.isMobile">{{ t('iqcheck.ui.tagline') }} · </template>
        {{ t('iqcheck.ui.sitesCount', { n: stats.total }) }}
        <template v-if="!menuStore.isMobile"> · {{ t('iqcheck.ui.modelsCount', { n: stats.modelCount }) }}</template>
        <template v-if="stats.checksToday">
          · <span class="iq-page__ok">{{ t('iqcheck.siteList.checksToday', { count: stats.checksToday }) }}</span>
        </template>
      </template>
      <template v-if="!menuStore.isMobile" #actions>
        <UButton v-permission="[PERM.IQCHECK_EDIT]" color="primary" icon="i-lucide-plus" @click="openForm()">
          {{ t('iqcheck.site.create') }}
        </UButton>
      </template>
    </PageHeader>

    <IqSubNav :site-count="Number(stats.total)" />

    <ListToolbar
      v-model:status="filter"
      v-model:keyword="keyword"
      :chips="chips"
      :chips-label="t('iqcheck.siteList.filterLabel')"
      :placeholder="t('iqcheck.siteList.searchPlaceholder')"
      @search="reload"
    >
      <div class="iq-seg" role="group" :aria-label="t('iqcheck.siteList.view')">
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

    <!-- 桌面列表视图：表格 -->
    <DataTable
      v-if="viewMode === 'table' && !menuStore.isMobile"
      :columns="columns"
      :rows="sites as unknown as Record<string, unknown>[]"
      row-key="id"
      :loading="loading"
      :empty-text="stats.total ? t('iqcheck.siteList.noMatch') : t('iqcheck.site.emptyTitle')"
    >
      <template #cell-name="{ row }">
        <div class="iq-cell-name">
          {{ asSite(row).name }}
          <span v-if="!asSite(row).enabled" class="iq-tag">{{ t('iqcheck.siteList.disabled') }}</span>
        </div>
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
            {{ scoreText(asSite(row)) }}
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
              :icon="asSite(row).checking ? 'HOutline:ArrowPathIcon' : 'HOutline:PlayIcon'"
              class="is-primary"
              :class="{ 'is-spin': asSite(row).checking }"
              :label="t('iqcheck.siteList.checkNowOf', { name: asSite(row).name })"
              :box="36"
              :size="18"
              :disabled="asSite(row).checking || !asSite(row).models.length"
              @click="checkNow(asSite(row))"
            />
            <AppIconButton
              icon="HOutline:ArrowDownTrayIcon"
              :label="t('iqcheck.ui.site.fetchModelsOf', { name: asSite(row).name })"
              :box="36"
              :size="18"
              @click="openForm(asSite(row), true)"
            />
            <AppIconButton
              icon="HOutline:PencilSquareIcon"
              :label="t('iqcheck.siteList.editOf', { name: asSite(row).name })"
              :box="36"
              :size="18"
              @click="openForm(asSite(row))"
            />
          </template>
          <ActionMenu
            :items="moreActions"
            :label="t('iqcheck.siteList.moreOf', { name: asSite(row).name })"
            @select="(key) => onRowAction(key, asSite(row))"
          />
        </div>
      </template>
    </DataTable>

    <template v-else>
      <div v-if="loading && !sites.length" :class="viewMode === 'table' ? 'iq-rows-skel' : 'iq-grid'">
        <div v-for="i in 4" :key="i" class="iq-skeleton" :class="{ 'is-row': viewMode === 'table' }" />
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

      <!-- 手机列表视图：紧凑行 -->
      <div v-else-if="viewMode === 'table'" class="iq-rows" :class="{ 'is-loading': loading }">
        <div v-for="s in sites" :key="s.id" class="iq-row">
          <span class="iq-row__score m-num" :style="{ color: scoreColor(s) }">{{ s.lastRound ? scoreText(s) : '—' }}</span>
          <button type="button" class="iq-row__main" :disabled="!canEdit" @click="openForm(s)">
            <span class="iq-row__name">
              <b>{{ s.name }}</b>
              <span class="iq-proto is-sm">{{ protocolLabel(s.protocol) }}</span>
            </span>
            <span class="iq-row__meta m-num">
              {{ t('iqcheck.ui.site.listMeta', { time: s.checking ? t('iqcheck.site.checking') : fmtRel(s.lastCheckAt), models: s.models.length }) }}
            </span>
          </button>
          <StatusPill :variant="healthPill(s).variant" :label="healthPill(s).label" />
          <AppIconButton
            v-if="canEdit"
            :icon="s.checking ? 'HOutline:ArrowPathIcon' : 'HOutline:PlayIcon'"
            class="is-primary"
            :class="{ 'is-spin': s.checking }"
            :label="t('iqcheck.siteList.checkNowOf', { name: s.name })"
            :box="44"
            :size="20"
            :disabled="s.checking || !s.models.length"
            @click="checkNow(s)"
          />
        </div>
      </div>

      <!-- 卡片视图 -->
      <div v-else class="iq-grid" :class="{ 'is-loading': loading }">
        <article v-for="s in sites" :key="s.id" class="iq-card">
          <header class="iq-card__head">
            <span class="iq-card__av m-sticker" :class="`m-sticker--${toneOf(s.id)}`" :style="{ transform: `rotate(${rotOf(s.id)})` }">
              {{ initialOf(s.name) }}
            </span>
            <span class="iq-card__title">
              <span class="iq-card__namerow">
                <span class="iq-card__name" :title="s.name">{{ s.name }}</span>
                <span v-if="!menuStore.isMobile" class="iq-proto">{{ protocolLabel(s.protocol) }}</span>
                <span v-if="!s.enabled" class="iq-tag">{{ t('iqcheck.siteList.disabled') }}</span>
              </span>
              <span class="iq-card__ep m-mono" :title="s.endpoint">{{ menuStore.isMobile ? hostOf(s.endpoint) : s.endpoint }}</span>
            </span>
            <StatusPill :variant="healthPill(s).variant" :label="healthPill(s).label" />
          </header>

          <div class="iq-card__score">
            <div class="iq-card__score-main">
              <span v-if="!menuStore.isMobile" class="iq-card__label">{{ t('iqcheck.siteList.lastResult') }}</span>
              <span class="iq-card__score-val">
                <span class="m-num" :style="{ color: scoreColor(s) }">{{ s.lastRound ? scoreText(s) : '—' }}</span>
                <span v-if="s.lastRound" class="iq-card__unit">{{ t('iqcheck.siteList.successUnit') }}</span>
              </span>
            </div>
            <div class="iq-card__score-note">
              <template v-if="menuStore.isMobile && isFailed(s)">
                <span class="m-num iq-strong is-bad iq-ellipsis" :title="noteText(s)">{{ noteText(s) }}</span>
              </template>
              <template v-else>
                <span class="m-num iq-strong iq-ellipsis" :class="deltaClass(s)">{{ deltaText(s) }}</span>
                <span v-if="!menuStore.isMobile" class="iq-card__note" :title="noteText(s)">{{ noteText(s) }}</span>
              </template>
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
            <span v-if="!menuStore.isMobile" class="iq-card__label">{{ t('iqcheck.siteList.modelsEffort') }}</span>
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
            <template v-if="!menuStore.isMobile">
              <div>
                <dt>{{ t('iqcheck.siteList.results') }}</dt>
                <dd class="m-num">{{ t('iqcheck.siteList.resultsCount', { count: s.resultCount }) }}</dd>
              </div>
              <div>
                <dt>API Key</dt>
                <dd class="m-mono" :title="s.apiKey">{{ s.apiKey || '—' }}</dd>
              </div>
            </template>
            <template v-else>
              <div class="is-wide">
                <dt>{{ t('iqcheck.ui.site.endpoint') }}</dt>
                <dd class="m-mono" :title="s.endpoint">{{ s.endpoint }}</dd>
              </div>
              <div class="is-wide">
                <dt>{{ t('iqcheck.site.colPublic') }}</dt>
                <dd class="m-mono">
                  <a class="iq-link" :href="publicUrl(s.slug)" target="_blank" rel="noopener">/public/iqcheck/{{ s.slug }}</a>
                </dd>
              </div>
            </template>
          </dl>

          <a v-if="!menuStore.isMobile" class="iq-card__public" :href="publicUrl(s.slug)" target="_blank" rel="noopener">
            <UIcon name="i-lucide-globe" />
            <span class="iq-card__public-l">{{ t('iqcheck.site.colPublic') }}</span>
            <span class="iq-card__public-p m-mono">/public/iqcheck/{{ s.slug }}</span>
            <UIcon name="i-lucide-external-link" />
          </a>

          <!-- 手机：左侧协议标签，右侧一排小图标 -->
          <footer v-if="menuStore.isMobile" class="iq-card__foot is-mobile">
            <span class="iq-card__tags">
              <span class="iq-proto">{{ protocolLabel(s.protocol) }}</span>
            </span>
            <AppIconButton
              v-if="canEdit"
              :icon="s.checking ? 'HOutline:ArrowPathIcon' : 'HOutline:PlayIcon'"
              class="is-primary"
              :class="{ 'is-spin': s.checking }"
              :label="t('iqcheck.siteList.checkNowOf', { name: s.name })"
              :box="44"
              :size="20"
              :disabled="s.checking || !s.models.length"
              @click="checkNow(s)"
            />
            <AppIconButton
              v-if="canEdit"
              icon="HOutline:ArrowDownTrayIcon"
              :label="t('iqcheck.ui.site.fetchModelsOf', { name: s.name })"
              :box="44"
              :size="20"
              @click="openForm(s, true)"
            />
            <a
              class="iq-ib"
              :href="publicUrl(s.slug)"
              target="_blank"
              rel="noopener"
              :aria-label="t('iqcheck.ui.site.openPublicOf', { name: s.name })"
              :title="t('iqcheck.ui.site.openPublicOf', { name: s.name })"
            >
              <component :is="menuStore.iconComponents['HOutline:GlobeAltIcon']" />
            </a>
            <AppIconButton
              v-if="canEdit"
              icon="HOutline:PencilSquareIcon"
              :label="t('iqcheck.siteList.editOf', { name: s.name })"
              :box="44"
              :size="20"
              @click="openForm(s)"
            />
            <ActionMenu :items="moreActions" :label="t('iqcheck.siteList.moreOf', { name: s.name })" @select="(key) => onRowAction(key, s)" />
          </footer>

          <!-- 桌面：立即检测 + 拉取模型，右侧编辑 / 更多 -->
          <footer v-else class="iq-card__foot">
            <template v-if="canEdit">
              <UButton
                color="primary"
                size="sm"
                class="iq-card__btn"
                :icon="s.checking ? 'i-lucide-loader-circle' : 'i-lucide-play'"
                :disabled="s.checking || !s.models.length"
                :ui="s.checking ? { leadingIcon: 'animate-spin' } : undefined"
                @click="checkNow(s)"
              >
                {{ s.checking ? t('iqcheck.site.checking') : t('iqcheck.site.checkNow') }}
              </UButton>
              <UButton color="neutral" variant="soft" size="sm" class="iq-card__btn" icon="i-lucide-download" @click="openForm(s, true)">
                {{ t('iqcheck.ui.site.fetchModels') }}
              </UButton>
            </template>
            <span class="iq-card__spacer" />
            <AppIconButton
              v-if="canEdit"
              icon="HOutline:PencilSquareIcon"
              :label="t('iqcheck.siteList.editOf', { name: s.name })"
              :box="36"
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

    <!-- 手机：悬浮新建 -->
    <div v-if="menuStore.isMobile && canEdit" class="iq-fab">
      <UButton color="primary" size="xl" icon="i-lucide-plus" class="iq-fab__btn" @click="openForm()">
        {{ t('iqcheck.site.create') }}
      </UButton>
    </div>

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

// 视图偏好：卡片 / 列表（桌面为表格、手机为紧凑行），记住选择。
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

const loading = ref(false)
const sites = ref<IQCheckSiteInfo[]>([])
const keyword = ref('')
const filter = ref<IQCheckSiteFilter>(IQCheckSiteFilter.IQ_CHECK_SITE_FILTER_ALL)
const pagination = ref({ page: 1, pageSize: 12, total: 0 })

const emptyStats = (): IQCheckSiteStats => ({
  total: 0,
  healthy: 0,
  partial: 0,
  failed: 0,
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
    { value: IQCheckSiteFilter.IQ_CHECK_SITE_FILTER_PARTIAL, label: t('iqcheck.ui.site.chipPartial'), count: Number(s.partial) },
    { value: IQCheckSiteFilter.IQ_CHECK_SITE_FILTER_FAILED, label: t('iqcheck.ui.site.chipFailed'), count: Number(s.failed) },
  ]
  if (Number(s.never)) out.push({ value: IQCheckSiteFilter.IQ_CHECK_SITE_FILTER_NEVER, label: t('iqcheck.siteList.chipNever'), count: Number(s.never) })
  if (Number(s.disabled)) {
    out.push({ value: IQCheckSiteFilter.IQ_CHECK_SITE_FILTER_DISABLED, label: t('iqcheck.siteList.chipDisabled'), count: Number(s.disabled) })
  }
  return out
})

const columns = computed<DataTableColumn[]>(() => [
  { key: 'name', title: t('iqcheck.siteList.colSite'), minWidth: 180 },
  { key: 'protocol', title: t('iqcheck.site.colProtocol'), width: 110 },
  { key: 'models', title: t('iqcheck.siteList.colModels'), minWidth: 200 },
  { key: 'interval', title: t('iqcheck.site.colInterval'), width: 120 },
  { key: 'last', title: t('iqcheck.siteList.lastRun'), width: 130 },
  { key: 'result', title: t('iqcheck.siteList.lastResult'), width: 170 },
  { key: 'public', title: t('iqcheck.site.colPublic'), width: 140 },
  { key: 'operation', title: t('system.common.operation'), width: 180, align: 'right' },
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

const hostOf = (endpoint: string) => {
  try {
    return new URL(endpoint).host || endpoint
  } catch {
    return endpoint
  }
}

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
const scoreText = (s: IQCheckSiteInfo) => (s.lastRound ? `${s.lastRound.success}/${roundTotal(s.lastRound)}` : '—')
const isFailed = (s: IQCheckSiteInfo) => s.health === IQCheckSiteHealth.IQ_CHECK_SITE_HEALTH_FAILED

const healthPill = (s: IQCheckSiteInfo): { variant: 'success' | 'warning' | 'error' | 'neutral'; label: string } => {
  if (s.checking) return { variant: 'neutral', label: t('iqcheck.site.checking') }
  switch (s.health) {
    case IQCheckSiteHealth.IQ_CHECK_SITE_HEALTH_HEALTHY:
      return { variant: 'success', label: t('iqcheck.siteList.chipHealthy') }
    case IQCheckSiteHealth.IQ_CHECK_SITE_HEALTH_PARTIAL:
      return { variant: 'warning', label: t('iqcheck.ui.site.chipPartial') }
    case IQCheckSiteHealth.IQ_CHECK_SITE_HEALTH_FAILED:
      return { variant: 'error', label: t('iqcheck.ui.site.chipFailed') }
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

const lastSuccessText = (s: IQCheckSiteInfo) =>
  s.lastSuccessAt
    ? t('iqcheck.ui.site.lastSuccess', { time: fmtRel(s.lastSuccessAt) })
    : t('iqcheck.ui.site.neverSucceeded')

// 「与上一轮相比」：成功率变化；整轮失败时改为显示上次成功的时间
const deltaText = (s: IQCheckSiteInfo) => {
  const rounds = s.recentRounds
  const last = s.lastRound
  if (!last) return t('iqcheck.siteList.noResultYet')
  if (isFailed(s)) return lastSuccessText(s)
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
  if (!s.lastRound || rounds.length < 2 || isFailed(s)) return ''
  const diff = ratioOf(s.lastRound) - ratioOf(rounds[rounds.length - 2]!)
  if (diff <= -0.05) return 'is-warn'
  if (diff < 0) return ''
  return diff > 0 ? 'is-ok' : ''
}
// 备注：有失败时显示第一条失败原因；否则显示本轮规模（模型数 · 随机抽到的题目）与平均耗时
const noteText = (s: IQCheckSiteInfo) => {
  const last = s.lastRound
  if (!last) return s.models.length ? t('iqcheck.siteList.waitFirst') : t('iqcheck.site.form.noModels')
  if (Number(last.failed) && last.error) return last.error
  const parts = [
    last.promptTitle
      ? t('iqcheck.ui.site.roundSize', { models: last.modelCount, title: last.promptTitle })
      : t('iqcheck.ui.site.roundModels', { models: last.modelCount }),
  ]
  if (Number(last.avgDurationMs)) parts.push(t('iqcheck.siteList.avgDuration', { s: seconds(last.avgDurationMs) }))
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

// fetchModels：打开编辑弹窗并立即拉取上游模型列表（卡片上的「拉取模型」）
const openForm = (row?: IQCheckSiteInfo, fetchModels = false) => formRef.value?.showDialog(row, { fetchModels })

const onRowAction = (key: string, site: IQCheckSiteInfo) => {
  if (key === 'edit') openForm(site)
  else if (key === 'check') checkNow(site)
  else if (key === 'copy') copyPublic(site)
  else if (key === 'open') window.open(publicUrl(site.slug), '_blank', 'noopener')
  else if (key === 'delete') confirmDelete(site)
}

const checkNow = async (site: IQCheckSiteInfo) => {
  const { data } = await triggerIQCheck({ id: site.id })
  feedback.success(
    t('iqcheck.ui.site.checkStarted', { models: data?.modelCount ?? site.models.length }),
  )
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
.iq-ellipsis {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.iq-dim {
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
}
.is-ok {
  color: var(--m-ok);
}
.is-warn {
  color: var(--m-warn);
}
.is-bad {
  color: var(--m-bad);
}

/* 视图切换 */
.iq-seg {
  display: inline-flex;
  flex: none;
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
  box-shadow: var(--app-shadow-sm);
}
.iq-seg__btn :deep(.iconify) {
  width: 16px;
  height: 16px;
}

/* 协议 / 模型 / 推理强度 */
.iq-proto {
  display: inline-flex;
  align-items: center;
  flex: none;
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
.iq-proto.is-sm {
  height: 20px;
  padding: 0 7px;
  font-size: 0.6875rem;
}
.iq-tag {
  display: inline-flex;
  align-items: center;
  flex: none;
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
  color: var(--m-aOn);
}
html.dark .iq-eff.is-medium {
  color: var(--m-a);
}
.iq-eff.is-low {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
}

/* 表格单元格 */
.iq-cell-name {
  display: flex;
  align-items: center;
  gap: 6px;
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
.iq-page :deep(.is-primary) {
  color: var(--m-pInk);
}
.iq-page :deep(.is-spin svg) {
  animation: iq-spin 1s linear infinite;
}
@keyframes iq-spin {
  to {
    transform: rotate(360deg);
  }
}
.iq-page :deep(.app-icon-btn:disabled) {
  opacity: 0.45;
  cursor: not-allowed;
}

/* 手机紧凑行 */
.iq-rows {
  display: flex;
  flex-direction: column;
  padding: calc(var(--cmp-card-unit) * 4) calc(var(--cmp-card-unit) * 6) calc(var(--cmp-card-unit) * 4) calc(var(--cmp-card-unit) * 14);
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-xl);
  background: var(--el-bg-color);
  transition: opacity 0.15s;
}
.iq-rows.is-loading,
.iq-grid.is-loading {
  opacity: 0.6;
}
.iq-row {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 68px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.iq-row:last-child {
  border-bottom: 0;
}
.iq-row__score {
  flex: none;
  width: 48px;
  font-family: var(--app-font-display);
  font-size: 1.25rem;
  font-weight: 800;
}
.iq-row__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  min-height: 44px;
  justify-content: center;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.iq-row__main:disabled {
  cursor: default;
}
.iq-row__name {
  display: flex;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  min-width: 0;
}
.iq-row__name b {
  min-width: 0;
  font-size: 0.9375rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.iq-row__meta {
  max-width: 100%;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.iq-rows-skel {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

/* 卡片 */
.iq-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 440px), 1fr));
  gap: 16px;
  transition: opacity 0.15s;
}
.iq-card {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
  padding: calc(var(--cmp-card-unit) * 20);
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
.iq-card__info > .is-wide {
  grid-column: 1 / -1;
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
.iq-card__btn {
  min-height: 36px;
  padding: 0 14px;
  border-radius: var(--app-radius-pill);
  font-weight: 700;
}
.iq-card__spacer {
  flex: 1;
}
.iq-card__tags {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.iq-ib {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 44px;
  height: 44px;
  border-radius: 14px;
  color: var(--el-text-color-secondary);
  text-decoration: none;
  transition:
    background 0.15s,
    color 0.15s;
}
.iq-ib:hover {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.iq-ib svg {
  width: 20px;
  height: 20px;
}
.iq-skeleton {
  height: 420px;
  border-radius: var(--app-radius-xl);
  background: var(--el-fill-color-light);
}
.iq-skeleton.is-row {
  height: 68px;
  border-radius: var(--app-radius);
}

/* 手机悬浮新建 */
.iq-fab {
  position: sticky;
  bottom: 0;
  display: flex;
  justify-content: flex-end;
  pointer-events: none;
  z-index: 5;
}
.iq-fab__btn {
  pointer-events: auto;
  min-height: 56px;
  padding: 0 22px;
  border-radius: var(--app-radius-pill);
  font-weight: 700;
  box-shadow: var(--app-shadow-brand);
}

@media (width <= 768px) {
  .iq-page {
    gap: 14px;
  }
  .iq-grid {
    gap: 12px;
  }
  .iq-card {
    padding: 16px 12px 6px 16px;
    gap: 12px;
  }
  .iq-card__head {
    align-items: center;
    padding-right: 4px;
  }
  .iq-card__av {
    width: 46px;
    height: 46px;
    border-radius: 15px;
  }
  .iq-card__ep {
    font-size: 0.6875rem;
  }
  .iq-card__score {
    gap: 12px;
    margin-right: 4px;
    padding: 10px 12px;
    border-radius: 14px;
  }
  .iq-card__score-main {
    min-width: 0;
  }
  .iq-card__score-val .m-num {
    font-size: 1.875rem;
  }
  .iq-card__score-note {
    font-size: 0.75rem;
  }
  .iq-card__spark {
    width: 96px;
    height: 32px;
  }
  .iq-card__block {
    padding-right: 4px;
  }
  .iq-card__info {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    margin-right: 4px;
    padding: 10px 12px;
    border-radius: 14px;
    background: var(--el-fill-color-light);
  }
  .iq-card__foot.is-mobile {
    gap: 0;
    padding-top: 6px;
  }
  .iq-card__foot :deep(.action-menu__trigger) {
    width: 44px;
    height: 44px;
    border-radius: 14px;
  }
  .iq-seg__btn {
    min-width: 44px;
    min-height: 40px;
  }
}
</style>
