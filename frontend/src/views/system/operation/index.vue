<!-- 任务与日志 · 操作记录：标题 + 「近 7 天共 N 条 · N 条失败 · 记录保留 N 天」+ 保留策略 / 导出 → LogsSubNav
     → 今日统计贴纸卡（PC）→ 时间/模块/用户/类型筛选 + 结果 chips(带数) + 搜索
     → PC 按天分组表格（模块 chip、动作 + 目标、目标节点/转发链路、结果 pill、IP），点行打开右侧详情抽屉；
       手机按天分组时间线卡片（上拉加载更多），点卡片打开底部详情面板。
     详情：结果条、转发链路、信息格（归属地/请求 ID/客户端…）、请求 JSON、响应（状态码）、复制 / 该用户的全部记录。 -->
<template>
  <div class="au-page">
    <PageHeader :title="t('system.logsNav.label')">
      <template #description>
        <span class="m-num">{{ summaryText }}</span>
        <template v-if="counts.failed">
          · <b class="au-tone-bad">{{ t('systemV2.audit.summaryFailed', { n: fmtNum(counts.failed) }) }}</b>
        </template>
        · {{ retentionText }}
      </template>
      <template #actions>
        <template v-if="!menuStore.isMobile">
          <UButton v-if="canEdit" color="neutral" variant="outline" icon="i-lucide-settings" @click="openRetention">
            {{ t('systemV2.audit.retention') }}
          </UButton>
          <UButton color="primary" icon="i-lucide-download" :loading="exporting" :disabled="!pagination.total" @click="exportCsv">
            {{ t('systemV2.audit.export') }}
          </UButton>
        </template>
        <template v-else>
          <AppIconButton v-if="canEdit" icon="HOutline:Cog6ToothIcon" :label="t('systemV2.audit.retention')" :box="44" @click="openRetention" />
          <AppIconButton
            icon="HOutline:ArrowDownTrayIcon"
            :label="t('systemV2.audit.exportCsv')"
            :box="44"
            :disabled="!pagination.total || exporting"
            @click="exportCsv"
          />
        </template>
      </template>
    </PageHeader>

    <LogsSubNav :counts="navCounts" />

    <div v-if="!menuStore.isMobile" class="au-stats">
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

    <ListToolbar
      v-model:status="queryForm.result"
      v-model:keyword="queryForm.keywords"
      :chips="resultChips"
      :chips-label="t('system.operation.result')"
      :placeholder="t('systemV2.audit.searchPlaceholder')"
      class="au-toolbar"
      @search="search"
    >
      <div class="au-filters" role="group" :aria-label="t('systemV2.audit.filters')">
        <div class="au-filter au-filter--range">
          <component :is="menuStore.iconComponents['HOutline:CalendarIcon']" class="au-filter__icon" />
          <AppSelect v-model="queryForm.range" :options="rangeOptions" fit @update:model-value="onRangeChange" />
        </div>
        <template v-if="queryForm.range === 'custom'">
          <input
            v-model="queryForm.from"
            type="date"
            class="au-date m-num"
            :max="queryForm.to || undefined"
            :aria-label="t('systemV2.audit.rangeStart')"
            @change="search"
          />
          <span class="au-date__to">{{ t('systemV2.audit.rangeTo') }}</span>
          <input
            v-model="queryForm.to"
            type="date"
            class="au-date m-num"
            :min="queryForm.from || undefined"
            :aria-label="t('systemV2.audit.rangeEnd')"
            @change="search"
          />
        </template>
        <div class="au-filter">
          <AppSelect v-model="queryForm.module" :options="moduleOptions" fit @update:model-value="onModuleChange" />
        </div>
        <div class="au-filter">
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
        <div class="au-filter">
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
        <UButton v-if="hasFilters" color="neutral" variant="ghost" icon="i-lucide-rotate-ccw" class="au-reset" @click="reset">
          {{ t('system.common.reset') }}
        </UButton>
        <span class="au-filters__sep" aria-hidden="true" />
      </div>
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
                <span class="au-avatar" :class="`m-sticker--${r.userTone}`">{{ r.userInitial }}</span>
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
              <div v-if="r.forwarded" class="au-node__line">{{ r.fwdLine }}</div>
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

    <!-- 手机：按天分组时间线 -->
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
          <h3 class="au-cards__day">{{ g.label }} · <span class="m-num">{{ t('system.audit.groupCount', { n: g.rows.length }) }}</span></h3>
          <ol class="au-tl">
            <li v-for="r in g.rows" :key="r.key" class="au-tl__item">
              <span class="au-tl__dot" :class="r.success ? 'is-ok' : 'is-bad'" aria-hidden="true" />
              <button
                type="button"
                class="au-ent"
                :class="{ 'is-on': current?.key === r.key }"
                :aria-label="`${r.action} ${r.target}`"
                @click="openDetail(r)"
              >
                <span class="au-ent__line">
                  <b class="au-ent__title">{{ r.action }}</b>
                  <StatusPill :variant="r.success ? 'success' : 'error'" :label="r.success ? t('system.common.success') : t('system.common.failed')" />
                </span>
                <span class="au-ent__target m-mono">{{ r.target }}</span>
                <span class="au-ent__meta">
                  <span class="au-avatar au-avatar--sm" :class="`m-sticker--${r.userTone}`">{{ r.userInitial }}</span>
                  <b>{{ r.userName }}</b>
                  <span class="au-mod au-mod--sm" :class="`m-sticker--${r.module.tone}`">{{ t(r.module.labelKey) }}</span>
                  <span class="au-ent__tm m-num">{{ r.hm }}</span>
                </span>
                <span v-if="r.forwarded" class="au-ent__fwd">
                  <component :is="menuStore.iconComponents['HOutline:ArrowsRightLeftIcon']" />
                  {{ r.fwdLine }}
                </span>
              </button>
            </li>
          </ol>
        </section>
        <div ref="moreRef" class="au-more">
          {{ loadingMore ? t('systemV2.audit.loadingMore') : hasMore ? t('systemV2.audit.loadMore') : t('systemV2.audit.noMore') }}
        </div>
      </div>
    </template>

    <Pagination
      v-if="!menuStore.isMobile"
      v-model:page="pagination.page"
      v-model:page-size="pagination.pageSize"
      :total="pagination.total"
      @change="getList"
    />

    <!-- 详情：PC 右侧抽屉 / 手机底部面板 -->
    <Teleport to="body">
      <Transition :name="menuStore.isMobile ? 'au-sheet' : 'au-drawer'">
        <div v-if="current" class="au-drawer" :class="{ 'is-sheet': menuStore.isMobile }">
          <button type="button" class="au-drawer__mask" :aria-label="t('system.common.close')" @click="current = null" />
          <aside class="au-drawer__panel" role="dialog" aria-modal="true" :aria-label="t('system.operation.detailTitle')">
            <span v-if="menuStore.isMobile" class="au-sheet__handle" aria-hidden="true" />
            <header class="au-drawer__head">
              <span class="m-sticker au-drawer__icon" :class="`m-sticker--${current.module.tone}`">
                <component :is="menuStore.iconComponents[current.module.icon]" />
              </span>
              <div class="au-drawer__title">
                <span class="au-drawer__kicker">
                  {{ t(current.module.labelKey) }} ·
                  {{ menuStore.isMobile ? current.fullTime : t('system.operation.detailTitle') }}
                </span>
                <h2>{{ current.action }}</h2>
                <span class="m-mono au-drawer__target">{{ current.target }}</span>
              </div>
              <StatusPill
                v-if="menuStore.isMobile"
                :variant="current.success ? 'success' : 'error'"
                :label="current.success ? t('system.common.success') : t('system.common.failed')"
              />
              <AppIconButton v-else icon="HOutline:XMarkIcon" :label="t('system.common.close')" @click="current = null" />
            </header>

            <div v-if="!menuStore.isMobile" class="au-result" :class="current.success ? 'is-ok' : 'is-bad'">
              <StatusPill :variant="current.success ? 'success' : 'error'" :label="current.success ? t('system.common.success') : t('system.common.failed')" />
              <span class="au-result__msg">{{ current.success ? t('system.audit.done') : current.error || t('system.audit.failedNoReason') }}</span>
              <span class="m-num au-result__dur">{{ fmtMs(current.durationMs) }}</span>
            </div>
            <p v-else-if="!current.success" class="au-result au-result--line is-bad">
              {{ current.error || t('system.audit.failedNoReason') }}
            </p>

            <section v-if="current.forwarded" class="au-chain">
              <h3 v-if="!menuStore.isMobile" class="au-label">{{ t('system.audit.chain') }}</h3>
              <div class="au-chain__row">
                <div class="au-chain__box">
                  <span>{{ current.apiKey ? t('system.audit.remoteMaster') : t('system.audit.master') }}</span>
                  <b>{{ current.operator || current.userName }}</b>
                  <span v-if="current.apiKey && !menuStore.isMobile" class="m-mono">API Key · {{ current.apiKey }}</span>
                </div>
                <span class="m-sticker m-sticker--a2 au-chain__arrow">
                  <component :is="menuStore.iconComponents['HOutline:ArrowRightIcon']" />
                </span>
                <div class="au-chain__box is-node">
                  <span>{{ t('system.audit.node') }}</span>
                  <b>{{ current.apiKey ? t('system.audit.thisNode') : current.node }}</b>
                </div>
              </div>
              <p v-if="!menuStore.isMobile" class="au-chain__hint">{{ current.chainHint }}</p>
            </section>

            <dl class="au-dl">
              <template v-if="!menuStore.isMobile">
                <div><dt>{{ t('system.audit.time') }}</dt><dd class="m-num">{{ current.fullTime }}</dd></div>
                <div><dt>{{ t('system.operation.user') }}</dt><dd>{{ current.userName }}</dd></div>
                <div><dt>{{ t('system.audit.sourceIp') }}</dt><dd class="m-mono">{{ current.ip || '—' }}</dd></div>
                <div><dt>{{ t('systemV2.audit.location') }}</dt><dd>{{ current.scopeLabel }}</dd></div>
                <div><dt>{{ t('system.audit.targetNode') }}</dt><dd>{{ current.nodeLabel }}</dd></div>
                <div><dt>{{ t('systemV2.audit.requestId') }}</dt><dd class="m-mono">{{ current.rid }}</dd></div>
                <div class="is-full"><dt>{{ t('system.audit.endpoint') }}</dt><dd class="m-mono">{{ current.method }} {{ current.path || '—' }}</dd></div>
                <div class="is-full"><dt>{{ t('system.audit.client') }}</dt><dd class="m-mono is-clip">{{ current.userAgent || '—' }}</dd></div>
              </template>
              <template v-else>
                <div><dt>{{ t('system.operation.user') }}</dt><dd>{{ current.userName }}</dd></div>
                <div><dt>{{ t('system.audit.sourceIp') }}</dt><dd class="m-mono">{{ current.ip || '—' }}</dd></div>
                <div><dt>{{ t('system.audit.targetNode') }}</dt><dd>{{ current.nodeLabel }}</dd></div>
                <div><dt>{{ t('system.operation.duration') }}</dt><dd class="m-num">{{ fmtMs(current.durationMs) }}</dd></div>
                <div><dt>{{ t('systemV2.audit.location') }}</dt><dd>{{ current.scopeLabel }}</dd></div>
                <div><dt>{{ t('systemV2.audit.requestId') }}</dt><dd class="m-mono">{{ current.rid }}</dd></div>
              </template>
            </dl>

            <section class="au-json">
              <div class="au-json__head">
                <h3 class="au-label">{{ t('systemV2.audit.requestDetail') }}</h3>
                <span v-if="current.method" class="au-tag m-mono">
                  {{ current.method }}<template v-if="menuStore.isMobile"> · HTTP {{ current.statusCode }}</template>
                </span>
                <AppIconButton
                  v-if="!menuStore.isMobile"
                  icon="HOutline:ClipboardDocumentIcon"
                  :label="t('system.audit.copyRequest')"
                  :box="36"
                  @click="copyText(current.requestJson)"
                />
              </div>
              <pre class="m-mono">{{ current.requestJson }}</pre>
            </section>

            <section v-if="!menuStore.isMobile" class="au-json">
              <div class="au-json__head">
                <h3 class="au-label">{{ t('systemV2.audit.response') }}</h3>
                <span class="au-tag m-mono" :class="current.success ? 'is-ok' : 'is-bad'">HTTP {{ current.statusCode }}</span>
              </div>
              <pre class="m-mono">{{ current.responseJson }}</pre>
            </section>

            <footer class="au-drawer__foot">
              <template v-if="!menuStore.isMobile">
                <UButton color="neutral" variant="soft" icon="i-lucide-copy" block @click="copyText(current.rawJson)">
                  {{ t('systemV2.audit.copyAll') }}
                </UButton>
                <UButton v-if="current.userId" color="neutral" variant="outline" icon="i-lucide-user" block @click="filterUser(current.userId)">
                  {{ t('systemV2.audit.userRecords') }}
                </UButton>
              </template>
              <template v-else>
                <UButton color="neutral" variant="soft" icon="i-lucide-copy" block @click="copyText(current.rawJson)">
                  {{ t('systemV2.audit.copyJson') }}
                </UButton>
                <UButton color="neutral" variant="outline" block @click="current = null">
                  {{ t('systemV2.audit.close') }}
                </UButton>
                <UButton
                  v-if="current.userId"
                  color="neutral"
                  variant="ghost"
                  icon="i-lucide-user"
                  block
                  class="au-drawer__wide"
                  @click="filterUser(current.userId)"
                >
                  {{ t('systemV2.audit.userRecords') }}
                </UButton>
              </template>
            </footer>
          </aside>
        </div>
      </Transition>
    </Teleport>

    <!-- 保留策略 -->
    <FormDialog
      v-model="retentionOpen"
      :title="t('systemV2.audit.retentionTitle')"
      :width="460"
      :loading="retentionSaving"
      @confirm="saveRetention"
    >
      <div class="au-ret">
        <p class="au-ret__desc">{{ t('systemV2.audit.retentionDesc') }}</p>
        <div class="au-ret__opts" role="radiogroup" :aria-label="t('systemV2.audit.retention')">
          <button
            v-for="o in retentionChoices"
            :key="o.value"
            type="button"
            role="radio"
            class="au-ret__opt"
            :class="{ 'is-on': retentionPick === o.value }"
            :aria-checked="retentionPick === o.value"
            @click="retentionPick = o.value"
          >
            {{ o.label }}
          </button>
        </div>
        <label v-if="retentionPick === 'custom'" class="app-field au-ret__custom">
          <span class="app-label">{{ t('systemV2.audit.retentionCustom') }}</span>
          <input v-model.number="retentionCustom" type="number" min="7" max="3650" class="app-input m-num" :class="{ 'is-error': retentionError }" />
          <span class="au-ret__hint" :class="{ 'is-bad': retentionError }">
            {{ retentionError ? t('systemV2.audit.retentionInvalid') : t('systemV2.audit.retentionCustomHint') }}
          </span>
        </label>
      </div>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import { useClipboard, useIntersectionObserver } from '@vueuse/core'
import { listOperationLogs, getOperationLogStats, updateOperationLogRetention } from '@/api/system'
import { listTasks } from '@/api/task'
import { userPage } from '@/api/user'
import { showRequestError } from '@/utils/request'
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

type RangeKey = 'today' | '7d' | '30d' | 'all' | 'custom'
type ResultKey = '' | 'ok' | 'fail'

const loading = ref(false)
const loadingMore = ref(false)
const exporting = ref(false)
const logs = ref<OperationLogInfo[]>([])
const users = ref<UserInfo[]>([])
const counts = ref({ success: 0, failed: 0 })
const stats = ref({ todayTotal: 0, todayFailed: 0, todayActiveUsers: 0, todayForwarded: 0, total: 0, retentionDays: 0 })
const taskTotal = ref<number | null>(null)
const canEdit = computed(() => menuStore.hasButtonPermission(PERM.SYSTEM_CONFIG_EDIT))

const emptyQuery = () => ({
  userId: '' as string,
  operationType: '' as string,
  module: '' as string,
  result: '' as ResultKey,
  keywords: '',
  range: '7d' as RangeKey,
  from: '',
  to: '',
})
const queryForm = ref(emptyQuery())
const pagination = ref({ page: 1, pageSize: 20, total: 0 })
const hasMore = computed(() => logs.value.length < pagination.value.total)

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
  { label: t('systemV2.audit.allUsers'), value: '' },
  ...users.value.map((u) => ({ label: `${u.username}${u.name ? ` (${u.name})` : ''}`, value: u.userId })),
])

const RANGE_KEYS: Record<Exclude<RangeKey, 'custom'>, string> = {
  today: 'system.audit.rangeToday',
  '7d': 'system.audit.range7d',
  '30d': 'system.audit.range30d',
  all: 'system.audit.rangeAll',
}
const rangeOptions = computed<{ label: string; value: RangeKey }[]>(() => [
  ...(Object.keys(RANGE_KEYS) as Exclude<RangeKey, 'custom'>[]).map((k) => ({ label: t(RANGE_KEYS[k]), value: k })),
  { label: t('systemV2.audit.rangeCustom'), value: 'custom' as const },
])
const moduleOptions = computed(() => [
  { label: t('systemV2.audit.allModules'), value: '' },
  ...OPERATION_MODULES.map((m) => ({ label: t(m.labelKey), value: m.key })),
])
const getOperationTypeLabel = (operationType: string) => {
  const labelKey = operationTypeLabelKeys[operationType]
  return labelKey ? t(labelKey) : operationType
}
const typeSelectOptions = computed(() => [
  { label: t('systemV2.audit.allTypes'), value: '' },
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

// —— 页头计数说明 ——
const summaryText = computed(() => {
  const total = fmtNum(pagination.value.total)
  const q = queryForm.value
  if (q.range === 'all') return t('systemV2.audit.summaryAll', { total })
  const range =
    q.range === 'custom'
      ? `${q.from || '…'} ${t('systemV2.audit.rangeTo')} ${q.to || '…'} `
      : t(RANGE_KEYS[q.range])
  return t('systemV2.audit.summaryRange', { range, total })
})
const retentionText = computed(() =>
  stats.value.retentionDays > 0
    ? t('systemV2.audit.retentionDays', { days: stats.value.retentionDays })
    : t('systemV2.audit.retentionForever'),
)

// —— 行视图 ——
interface DetailJson {
  method?: string
  path?: string
  operation?: string
  request?: unknown
  success?: boolean
  error?: string
  code?: number
  duration_ms?: number
  node?: string
  api_key?: string
  operator?: string
}
interface LogRow {
  key: string
  day: string
  tm: string
  hm: string
  fullTime: string
  userId: string
  userName: string
  userInitial: string
  userTone: string
  module: OperationModule
  action: string
  target: string
  node: string
  nodeLabel: string
  forwarded: boolean
  fwdLine: string
  chainHint: string
  apiKey: string
  operator: string
  success: boolean
  error: string
  ip: string
  scopeLabel: string
  rid: string
  userAgent: string
  method: string
  path: string
  statusCode: number
  durationMs: number
  requestJson: string
  responseJson: string
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
const TARGET_KEYS = ['name', 'title', 'username', 'reference', 'image', 'path', 'paths', 'host', 'email', 'id', 'ids']
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
const SCOPES = ['loopback', 'lan', 'public']

const toRow = (log: OperationLogInfo, idx: number): LogRow => {
  const d = parseDetail(log.detail)
  const time = log.operationTime ? dayjs(log.operationTime) : null
  const userId = log.userId || ''
  const operator = d.operator || ''
  const userName = userDisplay(userId) || operator || (d.api_key ? `API Key · ${d.api_key}` : t('system.audit.anonymous'))
  const node = d.node || ''
  const forwarded = !!(node || d.api_key)
  const path = log.path || d.path || ''
  const nodeLabel = d.api_key ? t('system.audit.fromMaster') : node || t('system.audit.local')
  const statusCode = Number(log.statusCode) || (log.success ? 200 : 500)
  const chainUser = operator || userName
  const response: Record<string, unknown> = { code: statusCode, success: log.success }
  if (d.error) response.error = d.error
  response.duration_ms = Number(log.durationMs) || 0
  return {
    key: log.id ? String(log.id) : `${time?.valueOf() ?? 0}-${idx}`,
    day: time ? time.format('YYYY-MM-DD') : '',
    tm: time ? time.format('HH:mm:ss') : '—',
    hm: time ? time.format('HH:mm') : '—',
    fullTime: time ? time.format('YYYY-MM-DD HH:mm:ss') : '—',
    userId,
    userName,
    userInitial: userId || operator ? ([...userName][0]?.toUpperCase() ?? '?') : '?',
    userTone: userId ? 'a' : d.api_key ? 'ok' : 'ink',
    module: moduleOf(String(log.operationType)),
    action: getOperationTypeLabel(String(log.operationType)),
    target: pickTarget(d.request, path || '—'),
    node,
    nodeLabel,
    forwarded,
    fwdLine: t('systemV2.audit.fwdLine', { user: chainUser, node: d.api_key ? t('system.audit.thisNode') : node }),
    chainHint: d.api_key
      ? t('systemV2.audit.chainHintRemote', { user: chainUser, key: d.api_key })
      : t('systemV2.audit.chainHint', { user: chainUser, node }),
    apiKey: d.api_key || '',
    operator,
    success: log.success,
    error: d.error || '',
    ip: log.ip,
    scopeLabel: SCOPES.includes(log.ipScope) ? t(`systemV2.audit.scopes.${log.ipScope}`) : '—',
    rid: log.id ? `#${log.id}` : '—',
    userAgent: log.userAgent,
    method: d.method || '',
    path,
    statusCode,
    durationMs: Number(log.durationMs) || 0,
    requestJson: JSON.stringify(d.request ?? {}, null, 2),
    responseJson: JSON.stringify(response, null, 2),
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

// —— 详情 ——
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
const rangeBounds = (): { startTime?: Date; endTime?: Date } => {
  const q = queryForm.value
  const today = dayjs().startOf('day')
  switch (q.range) {
    case 'today':
      return { startTime: today.toDate() }
    case '7d':
      return { startTime: today.subtract(6, 'day').toDate() }
    case '30d':
      return { startTime: today.subtract(29, 'day').toDate() }
    case 'custom':
      return {
        startTime: q.from ? dayjs(q.from).startOf('day').toDate() : undefined,
        endTime: q.to ? dayjs(q.to).endOf('day').toDate() : undefined,
      }
    default:
      return {}
  }
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
    ...rangeBounds(),
  } as Parameters<typeof listOperationLogs>[0]
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

// 手机：上拉加载下一页并追加
const loadMore = async () => {
  if (loading.value || loadingMore.value || !hasMore.value) return
  loadingMore.value = true
  try {
    const next = pagination.value.page + 1
    const { data } = await listOperationLogs(buildQuery(next, pagination.value.pageSize))
    logs.value = [...logs.value, ...(data?.logs || [])]
    pagination.value.page = next
    pagination.value.total = Number(data?.total || pagination.value.total)
  } catch {
    /* 下次滚动再试 */
  } finally {
    loadingMore.value = false
  }
}
const moreRef = ref<HTMLElement | null>(null)
useIntersectionObserver(moreRef, ([entry]) => {
  if (entry?.isIntersecting && menuStore.isMobile) void loadMore()
})

const loadStats = async () => {
  try {
    const { data } = await getOperationLogStats()
    stats.value = {
      todayTotal: Number(data?.todayTotal) || 0,
      todayFailed: Number(data?.todayFailed) || 0,
      todayActiveUsers: Number(data?.todayActiveUsers) || 0,
      todayForwarded: Number(data?.todayForwarded) || 0,
      total: Number(data?.total) || 0,
      retentionDays: Number(data?.retentionDays) || 0,
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
const onRangeChange = () => {
  const q = queryForm.value
  if (q.range === 'custom' && !q.from && !q.to) {
    q.from = dayjs().subtract(6, 'day').format('YYYY-MM-DD')
    q.to = dayjs().format('YYYY-MM-DD')
  }
  search()
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

// —— 保留策略 ——
const RETENTION_PRESETS = [30, 90, 180, 365]
const retentionOpen = ref(false)
const retentionSaving = ref(false)
const retentionPick = ref<string>('180')
const retentionCustom = ref<number>(180)
const retentionChoices = computed(() => [
  ...RETENTION_PRESETS.map((n) => ({ value: String(n), label: t('systemV2.audit.retentionDaysOpt', { n }) })),
  { value: '0', label: t('systemV2.audit.retentionForeverOpt') },
  { value: 'custom', label: t('systemV2.audit.retentionCustom') },
])
const retentionError = computed(
  () => retentionPick.value === 'custom' && !(Number.isInteger(retentionCustom.value) && retentionCustom.value >= 7 && retentionCustom.value <= 3650),
)
const openRetention = () => {
  const days = stats.value.retentionDays
  if (days === 0 || RETENTION_PRESETS.includes(days)) {
    retentionPick.value = String(days)
  } else {
    retentionPick.value = 'custom'
  }
  retentionCustom.value = days || 180
  retentionOpen.value = true
}
const saveRetention = async () => {
  if (retentionError.value) return
  const days = retentionPick.value === 'custom' ? retentionCustom.value : Number(retentionPick.value)
  retentionSaving.value = true
  try {
    const { data } = await updateOperationLogRetention({ retentionDays: days })
    const deleted = Number(data?.deleted) || 0
    feedback.success(deleted ? t('systemV2.audit.retentionSavedPurged', { n: fmtNum(deleted) }) : t('systemV2.audit.retentionSaved'))
    retentionOpen.value = false
    loadStats()
    if (deleted) search()
  } catch (error) {
    showRequestError(error)
  } finally {
    retentionSaving.value = false
  }
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
      'HTTP',
      'IP',
      t('systemV2.audit.location'),
      t('system.operation.duration'),
      t('system.audit.endpoint'),
      t('system.audit.client'),
      t('systemV2.audit.requestId'),
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
        String(r.statusCode),
        r.ip,
        r.scopeLabel,
        String(r.durationMs),
        `${r.method} ${r.path}`.trim(),
        r.userAgent,
        r.rid,
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
  padding: calc(var(--cmp-card-unit) * 16) calc(var(--cmp-card-unit) * 18);
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

/* 筛选：时间/模块/用户/类型 放在结果 chips 之前 */
.au-toolbar :deep(.ltb__chips) {
  order: 1;
}
.au-toolbar :deep(.ltb__spacer) {
  order: 2;
}
.au-toolbar :deep(.ltb__search) {
  order: 3;
  flex-basis: 240px;
}
.au-filters {
  order: 0;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.au-filters__sep {
  width: 1px;
  height: 26px;
  margin: 0 4px;
  background: var(--el-border-color-light);
}
.au-filter {
  position: relative;
  flex: none;
  min-width: 0;
}
.au-filter :deep(.app-sel__trigger) {
  min-height: 40px;
  padding-inline: 16px;
  border-radius: var(--app-radius-pill);
  border-width: 1.5px;
  font-weight: 600;
}
.au-filter--range :deep(.app-sel__trigger) {
  padding-left: 38px;
  color: var(--el-text-color-primary);
}
.au-filter__icon {
  position: absolute;
  left: 14px;
  top: 50%;
  z-index: 1;
  width: 16px;
  height: 16px;
  transform: translateY(-50%);
  color: var(--el-text-color-primary);
  pointer-events: none;
}
.au-date {
  min-height: 40px;
  padding: 0 12px;
  border-radius: var(--app-radius-pill);
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 600;
}
.au-date:focus {
  outline: none;
  border-color: var(--m-p);
}
.au-date__to {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.au-reset {
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
  min-width: 1040px;
  border-collapse: separate;
  border-spacing: 0;
}
.au-table th {
  padding: 14px 12px;
  text-align: left;
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--el-text-color-secondary);
  border-bottom: 1px solid var(--el-border-color-lighter);
  white-space: nowrap;
}
.au-table td {
  padding: 11px 12px;
  vertical-align: middle;
  font-size: 0.875rem;
  color: var(--el-text-color-primary);
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.au-col-time {
  width: 96px;
}
.au-col-go {
  width: 52px;
  position: relative;
}
.au-group td {
  padding: 16px 12px 8px;
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
}
.au-row:hover td {
  background: var(--el-fill-color-light);
}
.au-row.is-sel td {
  background: color-mix(in srgb, var(--m-pSoft) 70%, var(--el-bg-color));
}
.au-row.is-sel td:first-child {
  box-shadow: inset 4px 0 0 var(--m-p);
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
.au-avatar--sm {
  width: 20px;
  height: 20px;
  font-size: 0.625rem;
}
.au-user__name {
  font-weight: 700;
  white-space: nowrap;
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
}
.au-mod {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 26px;
  padding: 0 10px;
  border-radius: 9px;
  font-size: 0.75rem;
  font-weight: 700;
  white-space: nowrap;
}
.au-mod :deep(svg) {
  width: 14px;
  height: 14px;
}
.au-mod--sm {
  height: 22px;
  padding: 0 8px;
  border-radius: 7px;
  font-size: 0.6875rem;
}
.m-sticker--ink {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
}
.au-act {
  max-width: 280px;
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
  height: 24px;
  padding: 0 9px;
  border-radius: var(--app-radius-xs);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
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
.au-node__line {
  margin-top: 4px;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
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
  height: 110px;
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

/* 手机时间线 */
.au-cards {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.au-cards__group {
  display: flex;
  flex-direction: column;
}
.au-cards__day {
  margin: 6px 0 10px;
  font-size: 0.8125rem;
  font-weight: 800;
  letter-spacing: 0.04em;
  color: var(--el-text-color-secondary);
}
.au-tl {
  list-style: none;
  margin: 0 0 8px;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.au-tl__item {
  position: relative;
  display: flex;
  gap: 12px;
}
.au-tl__item::before {
  content: '';
  position: absolute;
  left: 7px;
  top: 0;
  bottom: -12px;
  width: 2px;
  background: var(--el-border-color-light);
}
.au-tl__item:last-child::before {
  bottom: 50%;
}
.au-tl__dot {
  position: relative;
  z-index: 1;
  flex: none;
  width: 16px;
  height: 16px;
  margin-top: 20px;
  border-radius: 50%;
  border: 3px solid var(--el-bg-color-page, var(--m-bg));
}
.au-tl__dot.is-ok {
  background: var(--m-ok);
}
.au-tl__dot.is-bad {
  background: var(--m-bad);
}
.au-ent {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px 14px 12px;
  border-radius: 20px;
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  font: inherit;
  color: var(--el-text-color-primary);
  text-align: left;
  cursor: pointer;
}
.au-ent.is-on {
  border-color: var(--m-p);
  box-shadow: 0 0 0 3px var(--m-pSoft);
}
.au-ent__line {
  display: flex;
  align-items: center;
  gap: 8px;
}
.au-ent__title {
  flex: 1;
  min-width: 0;
  font-size: 0.9375rem;
  font-weight: 800;
}
.au-ent__target {
  width: 100%;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.au-ent__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.au-ent__meta b {
  color: var(--el-text-color-primary);
  font-weight: 700;
}
.au-ent__tm {
  margin-left: auto;
  font-weight: 700;
}
.au-ent__fwd {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  border-radius: 10px;
  background: var(--m-a2Soft);
  color: var(--m-ink);
  font-size: 0.75rem;
  font-weight: 700;
}
.au-ent__fwd :deep(svg) {
  width: 14px;
  height: 14px;
  flex: none;
}
.au-more {
  padding: 4px 0 8px;
  text-align: center;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}

/* 详情：抽屉 / 底部面板 */
.au-drawer {
  position: fixed;
  inset: 0;
  z-index: 2100;
}
.au-drawer__mask {
  position: absolute;
  inset: 0;
  border: 0;
  padding: 0;
  background: color-mix(in srgb, var(--m-ink) 16%, transparent);
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
  box-shadow: var(--app-shadow-lg);
}
.au-drawer__panel::-webkit-scrollbar {
  display: none;
}
.au-drawer.is-sheet .au-drawer__mask {
  background: color-mix(in srgb, var(--m-ink) 50%, transparent);
}
.au-drawer.is-sheet .au-drawer__panel {
  top: auto;
  left: 0;
  width: 100%;
  max-height: 86%;
  gap: 14px;
  padding: 10px 16px calc(28px + env(safe-area-inset-bottom, 0px));
  border-left: none;
  border-radius: 30px 30px 0 0;
}
.au-sheet__handle {
  align-self: center;
  flex: none;
  width: 44px;
  height: 5px;
  border-radius: 999px;
  background: var(--el-border-color-light);
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
.is-sheet .au-drawer__title h2 {
  font-size: 1.1875rem;
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
  margin: 0;
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
.au-result--line {
  font-size: 0.8125rem;
  font-weight: 600;
  word-break: break-word;
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
.au-chain__hint {
  margin: 0;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.is-sheet .au-chain {
  padding: 12px;
  border-radius: var(--app-radius);
  background: var(--m-a2Soft);
}
.is-sheet .au-chain__box {
  padding: 0;
  border: none;
  background: transparent;
}
.is-sheet .au-chain__box.is-node {
  text-align: right;
}
.is-sheet .au-chain__arrow {
  background: transparent;
}
.au-dl {
  margin: 0;
  padding: 14px 16px;
  border-radius: 18px;
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
.au-dl dd.m-mono {
  font-size: 0.75rem;
}
.au-dl dd.is-clip {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 400;
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
  padding: 0 9px;
  border-radius: var(--app-radius-xs);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  font-weight: 700;
}
.au-tag.is-ok {
  background: var(--m-okSoft);
  color: var(--m-ok);
}
.au-tag.is-bad {
  background: var(--m-badSoft);
  color: var(--m-bad);
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
.is-sheet .au-json pre {
  max-height: 210px;
}
.au-json pre::-webkit-scrollbar {
  display: none;
}
.au-drawer__foot {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: auto;
  padding-top: 4px;
}
.au-drawer__foot > * {
  flex: 1 1 0;
  min-height: 44px;
}
.is-sheet .au-drawer__foot > * {
  min-height: 50px;
}
.au-drawer__wide {
  flex-basis: 100%;
}
.au-drawer-enter-active,
.au-drawer-leave-active,
.au-sheet-enter-active,
.au-sheet-leave-active {
  transition: opacity 0.2s;
}
.au-drawer-enter-active .au-drawer__panel,
.au-drawer-leave-active .au-drawer__panel,
.au-sheet-enter-active .au-drawer__panel,
.au-sheet-leave-active .au-drawer__panel {
  transition: transform 0.22s cubic-bezier(0.4, 0, 0.2, 1);
}
.au-drawer-enter-from,
.au-drawer-leave-to,
.au-sheet-enter-from,
.au-sheet-leave-to {
  opacity: 0;
}
.au-drawer-enter-from .au-drawer__panel,
.au-drawer-leave-to .au-drawer__panel {
  transform: translateX(24px);
}
.au-sheet-enter-from .au-drawer__panel,
.au-sheet-leave-to .au-drawer__panel {
  transform: translateY(32px);
}

/* 保留策略 */
.au-ret {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.au-ret__desc {
  margin: 0;
  font-size: 0.8125rem;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}
.au-ret__opts {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.au-ret__opt {
  min-height: 40px;
  padding: 0 16px;
  border-radius: var(--app-radius-pill);
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.875rem;
  font-weight: 700;
  cursor: pointer;
}
.au-ret__opt:hover {
  border-color: var(--m-p);
  color: var(--m-pInk);
}
.au-ret__opt.is-on {
  background: var(--m-ink);
  border-color: var(--m-ink);
  color: var(--m-bg);
}
.au-ret__custom .app-input {
  max-width: 200px;
}
.au-ret__hint {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.au-ret__hint.is-bad {
  color: var(--m-bad);
}

@media (width <= 1100px) {
  .au-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
@media (width < 768px) {
  .au-filters {
    width: 100%;
    flex-wrap: nowrap;
    overflow-x: auto;
    scrollbar-width: none;
  }
  .au-filters::-webkit-scrollbar {
    display: none;
  }
  .au-filter :deep(.app-sel__trigger) {
    min-height: 44px;
  }
  .au-filters__sep {
    display: none;
  }
  .au-toolbar :deep(.ltb__chips) {
    order: 1;
  }
  .au-toolbar :deep(.ltb__search) {
    order: 2;
    flex-basis: 100%;
  }
  .au-ret__opt {
    min-height: 44px;
  }
}
</style>
