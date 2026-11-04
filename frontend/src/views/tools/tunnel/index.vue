<!-- 内网穿透：标题+计数说明（右上 客户端 / 新建）→ 子导航 → frps 服务端主色卡（状态、端口、传输、编辑/日志/重启）
     → 类型 chips + 启用分段 + 搜索 → 表格（PC）/ 信息卡（手机）→ 底部悬浮批量条（合并配置/启停/删除）。 -->
<template>
  <div class="tn-page">
    <PageHeader :title="t('tools.tunnel.title')">
      <template #description>
        {{ menuStore.isMobile ? t('tools.tunnel.summaryShort', { total: num(overview.total) }) : t('tools.tunnel.summary', { total: num(overview.total) }) }}
        <template v-if="num(overview.online)">
          · <span class="tn-tone-ok">{{ t('tools.tunnel.summaryOnline', { count: num(overview.online) }) }}</span>
        </template>
        <template v-if="num(overview.pending)">
          · <span class="tn-tone-warn">{{ t('tools.tunnel.summaryPending', { count: num(overview.pending) }) }}</span>
        </template>
      </template>
      <template #actions>
        <AppDropdown align="end" :width="340">
          <template #trigger>
            <UButton color="neutral" variant="outline" icon="i-lucide-server" :aria-label="t('tools.tunnel.clients.title')">
              <template v-if="!menuStore.isMobile">{{ t('tools.tunnel.clients.button') }}</template>
              <span class="tn-btn-count m-num">{{ clientList.length }}</span>
            </UButton>
          </template>
          <div class="tn-clients">
            <div class="tn-clients__head">
              <b>{{ t('tools.tunnel.clients.title') }}</b>
              <span>{{ t('tools.tunnel.clients.sub', { online: num(overview.clients) }) }}</span>
            </div>
            <div v-if="!clientList.length" class="tn-clients__empty">{{ t('tools.tunnel.clients.empty') }}</div>
            <div v-for="c in clientList" :key="`${c.client?.hostname}|${c.client?.address}`" class="tn-client">
              <span class="m-sticker tn-client__ico" :class="c.online ? 'm-sticker--ok' : 'm-sticker--a2'">
                <component :is="menuStore.iconComponents['HOutline:ComputerDesktopIcon']" />
              </span>
              <span class="tn-client__main">
                <span class="tn-client__name m-mono">{{ c.client?.hostname || '—' }}</span>
                <span class="tn-client__meta">
                  {{ [c.client?.os, c.client?.arch].filter(Boolean).join(' / ') }} · frpc {{ c.client?.version }} · {{ c.client?.address }}
                </span>
                <span class="tn-client__meta">
                  {{ t('tools.tunnel.clients.tunnels', { online: num(c.onlineTunnels), total: num(c.tunnels) }) }}
                  · {{ t('tools.tunnel.clients.login', { time: formatAgo(c.client?.loginTime, t) }) }}
                </span>
              </span>
              <StatusPill :variant="c.online ? 'success' : 'neutral'" :label="c.online ? t('tools.tunnel.status.online') : t('tools.tunnel.status.offline')" />
            </div>
          </div>
        </AppDropdown>
        <UButton v-if="!menuStore.isMobile" color="primary" icon="i-lucide-plus" @click="openCreateDialog()">
          {{ t('tools.tunnel.addTunnel') }}
        </UButton>
      </template>
    </PageHeader>


    <!-- frps 服务端 -->
    <section class="tn-hero" :aria-label="t('tools.tunnel.hero.title')">
      <span class="tn-hero__blob" aria-hidden="true" />
      <span class="tn-hero__ring" aria-hidden="true" />
      <div class="tn-hero__main">
        <div class="tn-hero__id">
          <span class="tn-hero__sticker"><component :is="menuStore.iconComponents['HOutline:GlobeAltIcon']" /></span>
          <div class="tn-hero__idtext">
            <h2 class="tn-hero__title">{{ t('tools.tunnel.hero.title') }}</h2>
            <span class="tn-hero__state">
              <i class="tn-hero__dot" :class="{ 'is-off': !frpsRunning }" />
              {{ heroStateText }}
            </span>
          </div>
        </div>
        <div class="tn-hero__addr m-mono">{{ heroAddr }}</div>
        <div v-if="frpsStatus?.error" class="tn-hero__err">{{ t('tools.tunnel.hero.startFailed', { err: frpsStatus.error }) }}</div>
        <div class="tn-hero__btns">
          <button type="button" class="tn-hero__btn is-solid" @click="frpsVisible = true">
            <component :is="menuStore.iconComponents['HOutline:PencilSquareIcon']" />{{ t('tools.tunnel.hero.edit') }}
          </button>
          <button type="button" class="tn-hero__btn" @click="logsVisible = true">
            <component :is="menuStore.iconComponents['HOutline:DocumentTextIcon']" />{{ t('tools.tunnel.hero.logs') }}
          </button>
          <button v-if="frpsConfig?.isEnable" type="button" class="tn-hero__btn" @click="confirmRestart">
            <component :is="menuStore.iconComponents['HOutline:ArrowPathIcon']" />{{ t('tools.tunnel.hero.restart') }}
          </button>
        </div>
      </div>
      <div class="tn-hero__side">
        <div class="tn-hero__tiles">
          <div v-for="tile in heroTiles" :key="tile.k" class="tn-hero__tile" :class="tile.cls">
            <span class="tn-hero__tk">{{ tile.k }}</span>
            <span class="tn-hero__tv m-num m-mono">{{ tile.v }}</span>
            <span class="tn-hero__ts">{{ tile.sub }}</span>
          </div>
        </div>
        <div class="tn-hero__chips">
          <span v-for="chip in heroChips" :key="chip" class="tn-hero__chip">{{ chip }}</span>
          <span class="tn-hero__spacer" />
          <span v-if="frpsConfig?.tlsForce" class="tn-hero__chip is-accent">{{ t('tools.tunnel.hero.tls') }}</span>
          <span class="tn-hero__chip">{{ t('tools.tunnel.hero.perTunnelAuth') }}</span>
        </div>
      </div>
    </section>

    <ListToolbar
      v-model:status="queryForm.type"
      v-model:keyword="queryForm.keywords"
      :chips="typeChips"
      :chips-label="t('tools.tunnel.chipsLabel')"
      :placeholder="t('tools.tunnel.searchPlaceholder')"
      @search="search"
    >
      <div class="tn-seg" role="radiogroup" :aria-label="t('tools.tunnel.enableLabel')">
        <button
          v-for="opt in enableOptions"
          :key="opt.value"
          type="button"
          role="radio"
          class="tn-seg__btn"
          :class="{ 'is-on': queryForm.isEnable === opt.value }"
          :aria-checked="queryForm.isEnable === opt.value"
          @click="setEnableFilter(opt.value)"
        >{{ opt.label }}</button>
      </div>
    </ListToolbar>

    <div class="tn-page__body">
      <DataTable
        v-if="!menuStore.isMobile"
        v-model="selectedIds"
        :columns="columns"
        :rows="tableRows"
        row-key="id"
        selectable
        :loading="loading"
        :empty-text="t('tools.tunnel.noData')"
      >
        <template #cell-name="{ row }">
          <div class="tn-name">
            <span class="tn-name__main">{{ info(row).name }}</span>
            <span v-if="info(row).remark" class="tn-name__sub">{{ info(row).remark }}</span>
          </div>
        </template>
        <template #cell-type="{ row }">
          <span class="tn-ty" :class="`is-${typeKey(info(row).type)}`">{{ typeKey(info(row).type) }}</span>
        </template>
        <template #cell-route="{ row }">
          <div class="tn-route m-mono">
            <span class="tn-route__local">{{ localText(info(row)) }}</span>
            <span class="tn-route__line">
              <component :is="menuStore.iconComponents['HOutline:ArrowTurnDownRightIcon']" class="tn-route__arrow" />
              <span class="tn-route__remote" :class="{ 'is-p2p': isP2p(info(row)) }" :title="remoteText(info(row))">{{ remoteText(info(row)) }}</span>
              <a
                v-if="entryUrl(info(row))"
                :href="entryUrl(info(row))"
                target="_blank"
                rel="noopener"
                class="tn-route__ext"
                :aria-label="t('tools.tunnel.openEntry', { url: entryUrl(info(row)) })"
              ><component :is="menuStore.iconComponents['HOutline:ArrowTopRightOnSquareIcon']" /></a>
            </span>
          </div>
        </template>
        <template #cell-client="{ row }">
          <div class="tn-client-cell">
            <StatusPill :variant="statusVariant(info(row))" :label="statusLabel(info(row))" />
            <span class="tn-client-cell__host m-mono">{{ info(row).client?.hostname || t('tools.tunnel.noClient') }}</span>
          </div>
        </template>
        <template #cell-traffic="{ row }">
          <div v-if="isOnline(info(row))" class="tn-traffic m-num">
            <span><i class="tn-up">↑</i> {{ trafficText(info(row), 'out') }}</span>
            <span><i class="tn-down">↓</i> {{ trafficText(info(row), 'in') }}</span>
          </div>
          <span v-else class="tn-dim">—</span>
        </template>
        <template #cell-connections="{ row }">
          <span class="m-num tn-strong">{{ isOnline(info(row)) ? num(info(row).activeConnections) : '—' }}</span>
        </template>
        <template #cell-operation="{ row }">
          <div class="tn-acts">
            <AppIconButton
              icon="HOutline:KeyIcon"
              :label="t('tools.tunnel.frpcConfig')"
              :box="36"
              :size="18"
              :active="frpcVisible && frpcRow?.id === info(row).id"
              @click="openFrpcDialog(info(row))"
            />
            <AppIconButton
              icon="HOutline:ChartBarIcon"
              :label="t('tools.tunnel.actionStats')"
              :box="36"
              :size="18"
              :active="statsVisible && statsRow?.id === info(row).id"
              @click="openStatsDialog(info(row))"
            />
            <AppIconButton
              :icon="info(row).isEnable ? 'HOutline:StopIcon' : 'HOutline:PlayIcon'"
              :label="info(row).isEnable ? t('tools.tunnel.actionDisable') : t('tools.tunnel.actionEnable')"
              :box="36"
              :size="18"
              :class="info(row).isEnable ? 'tn-act--stop' : 'tn-act--start'"
              @click="toggleEnable(info(row), !info(row).isEnable)"
            />
            <AppIconButton icon="HOutline:PencilSquareIcon" :label="t('tools.tunnel.edit')" :box="36" :size="18" @click="openEditDialog(info(row))" />
            <ActionMenu :items="rowActions(info(row))" @select="(key) => onRowAction(key, info(row))" />
          </div>
        </template>
      </DataTable>

      <template v-else>
        <div v-if="loading && !list.length" class="tn-cards">
          <div v-for="i in 3" :key="i" class="tn-skeleton" />
        </div>
        <EmptyState
          v-else-if="!list.length"
          icon="HOutline:GlobeAltIcon"
          :title="t('tools.tunnel.noData')"
          :description="t('tools.tunnel.emptyDesc')"
        >
          <template #action>
            <UButton color="primary" variant="soft" icon="i-lucide-plus" @click="openCreateDialog()">
              {{ t('tools.tunnel.addTunnel') }}
            </UButton>
          </template>
        </EmptyState>
        <div v-else class="tn-cards">
          <article
            v-for="(row, idx) in list"
            :key="row.id"
            class="tn-card"
            :class="{ 'is-selected': selectedIds.includes(row.id) }"
          >
            <header class="tn-card__head">
              <span class="m-sticker tn-card__sticker m-mono" :class="stickerClass(row)" :style="{ '--rot': ROTS[idx % ROTS.length] }">{{ typeKey(row.type) }}</span>
              <span class="tn-card__title">
                <span class="tn-card__name">{{ row.name }}</span>
                <span v-if="row.remark" class="tn-card__remark">{{ row.remark }}</span>
              </span>
              <StatusPill :variant="statusVariant(row)" :label="statusLabel(row)" />
              <label class="tn-card__check" :aria-label="t('tools.tunnel.selectTunnel', { name: row.name })">
                <input type="checkbox" :checked="selectedIds.includes(row.id)" @change="toggleSelect(row.id)" />
              </label>
            </header>
            <div class="tn-card__route m-mono">
              <span class="tn-route__local">{{ localText(row) }}</span>
              <span class="tn-card__remote">
                <component :is="menuStore.iconComponents['HOutline:ArrowTurnDownRightIcon']" class="tn-route__arrow" />
                <span :class="{ 'is-p2p': isP2p(row) }">{{ remoteText(row) }}</span>
              </span>
            </div>
            <dl class="tn-card__grid">
              <div>
                <dt>{{ t('tools.tunnel.colClient') }}</dt>
                <dd class="m-mono">{{ row.client?.hostname || '—' }}</dd>
              </div>
              <div>
                <dt>{{ t('tools.tunnel.stats.activeConnections') }}</dt>
                <dd class="m-num">{{ isOnline(row) ? num(row.activeConnections) : '—' }}</dd>
              </div>
              <div>
                <dt>{{ t('tools.tunnel.todayOut') }}</dt>
                <dd class="m-num"><i class="tn-up">↑</i> {{ trafficText(row, 'out') }}</dd>
              </div>
              <div>
                <dt>{{ t('tools.tunnel.todayIn') }}</dt>
                <dd class="m-num"><i class="tn-down">↓</i> {{ trafficText(row, 'in') }}</dd>
              </div>
            </dl>
            <footer class="tn-card__foot">
              <span class="tn-card__meta">
                <span class="tn-ty" :class="`is-${typeKey(row.type)}`">{{ typeKey(row.type) }}</span>
              </span>
              <AppIconButton icon="HOutline:KeyIcon" :label="t('tools.tunnel.frpcConfig')" :box="44" :size="20" @click="openFrpcDialog(row)" />
              <AppIconButton icon="HOutline:ChartBarIcon" :label="t('tools.tunnel.actionStats')" :box="44" :size="20" @click="openStatsDialog(row)" />
              <AppIconButton
                :icon="row.isEnable ? 'HOutline:StopIcon' : 'HOutline:PlayIcon'"
                :label="row.isEnable ? t('tools.tunnel.actionDisable') : t('tools.tunnel.actionEnable')"
                :box="44"
                :size="20"
                :class="row.isEnable ? 'tn-act--stop' : 'tn-act--start'"
                @click="toggleEnable(row, !row.isEnable)"
              />
              <AppIconButton icon="HOutline:PencilSquareIcon" :label="t('tools.tunnel.edit')" :box="44" :size="20" @click="openEditDialog(row)" />
              <ActionMenu :items="rowActions(row)" @select="(key) => onRowAction(key, row)" />
            </footer>
          </article>
        </div>
      </template>

      <Pagination
        v-model:page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        @change="getList()"
      />

      <p class="tn-note">
        <component :is="menuStore.iconComponents['HOutline:InformationCircleIcon']" />
        {{ t('tools.tunnel.footNote') }}
      </p>
    </div>

    <!-- 手机：右下悬浮新建 -->
    <div v-if="menuStore.isMobile && !selectedIds.length" class="tn-fab">
      <button type="button" class="tn-fab__btn" :aria-label="t('tools.tunnel.addTunnel')" @click="openCreateDialog()">
        <UIcon name="i-lucide-plus" />
      </button>
    </div>

    <Transition name="tn-pop">
      <div v-if="selectedIds.length" class="tn-batch" role="toolbar">
        <span class="tn-batch__count">{{ t('tools.tunnel.selectedTunnels', { count: selectedIds.length }) }}</span>
        <button type="button" class="tn-batch__btn" @click="openMergedFrpc">
          <component :is="menuStore.iconComponents['HOutline:KeyIcon']" />{{ t('tools.tunnel.mergeConfig') }}
        </button>
        <button type="button" class="tn-batch__btn" @click="batchSetEnable(true)">
          <component :is="menuStore.iconComponents['HOutline:PlayIcon']" />{{ t('tools.tunnel.actionEnable') }}
        </button>
        <button type="button" class="tn-batch__btn" @click="batchSetEnable(false)">
          <component :is="menuStore.iconComponents['HOutline:StopIcon']" />{{ t('tools.tunnel.actionDisable') }}
        </button>
        <button type="button" class="tn-batch__btn is-danger" @click="batchDelete">
          <component :is="menuStore.iconComponents['HOutline:TrashIcon']" />{{ t('tools.tunnel.delete') }}
        </button>
        <button type="button" class="tn-batch__btn is-icon" :aria-label="t('tools.tunnel.clearSelection')" @click="selectedIds = []">
          <component :is="menuStore.iconComponents['HOutline:XMarkIcon']" />
        </button>
      </div>
    </Transition>

    <TunnelCreate ref="createRef" @refresh="refresh" />
    <TunnelStatsDialog v-model="statsVisible" :row="statsRow" :remote="statsRow ? remoteText(statsRow) : ''" />
    <FrpcConfigDialog
      v-model="frpcVisible"
      :row="frpcRow"
      :rows="frpcRows"
      :frps="frpsConfig"
      @credential-reset="onCredentialReset"
    />
    <FrpsConfigDialog v-model="frpsVisible" @saved="onFrpsSaved" />
    <FrpsLogsDialog v-model="logsVisible" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import TunnelCreate from '@/views/tools/tunnel/create.vue'
import TunnelStatsDialog from '@/views/tools/tunnel/TunnelStatsDialog.vue'
import FrpcConfigDialog from '@/views/tools/tunnel/FrpcConfigDialog.vue'
import FrpsConfigDialog from '@/views/tools/tunnel/FrpsConfigDialog.vue'
import FrpsLogsDialog from '@/views/tools/tunnel/FrpsLogsDialog.vue'
import { formatBytes, formatAgo, formatDuration, toDate } from '@/views/tools/components/format'
import { listTunnels, deleteTunnel, updateTunnel, getFrpsConfig, restartFrps, resetTunnelCredential } from '@/api/tunnel'
import {
  TunnelType,
  TunnelStatus,
  type TunnelInfo,
  type FrpsConfig,
  type FrpsStatus,
  type TunnelOverview,
} from '@/types/v1/tunnel'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import type { ToolbarChip } from '@/components/ui/ListToolbar.vue'
import { Dialog } from '@/utils/dialog'

defineOptions({ name: 'TunnelView' })

type TypeFilter = '' | TunnelType
type EnableFilter = '' | 'true' | 'false'

const POLL_MS = 5000
const ROTS = ['-4deg', '3deg', '-2deg', '4deg']
const TYPE_ORDER: TunnelType[] = [
  TunnelType.TUNNEL_TYPE_TCP,
  TunnelType.TUNNEL_TYPE_UDP,
  TunnelType.TUNNEL_TYPE_HTTP,
  TunnelType.TUNNEL_TYPE_HTTPS,
  TunnelType.TUNNEL_TYPE_STCP,
  TunnelType.TUNNEL_TYPE_XTCP,
  TunnelType.TUNNEL_TYPE_TCPMUX,
]

const menuStore = useMenuStore()
const { t } = useI18n()
const createRef = useTemplateRef<InstanceType<typeof TunnelCreate> | null>('createRef')

const selectedIds = ref<string[]>([])
const list = ref<TunnelInfo[]>([])
const loading = ref(false)
const statsVisible = ref(false)
const statsRow = ref<TunnelInfo | null>(null)
const frpcVisible = ref(false)
const frpcRow = ref<TunnelInfo | null>(null)
const frpcRows = ref<TunnelInfo[]>([])
const frpsVisible = ref(false)
const logsVisible = ref(false)
const frpsConfig = ref<FrpsConfig | null>(null)
const frpsStatus = ref<FrpsStatus | null>(null)
const now = ref(Date.now())

const emptyOverview = (): TunnelOverview => ({
  total: 0, online: 0, pending: 0, offline: 0, clients: 0, typeCounts: [],
  todayBytesIn: 0, todayBytesOut: 0, activeConnections: 0, clientList: [],
})
const overview = ref<TunnelOverview>(emptyOverview())

const queryForm = reactive({ keywords: '', type: '' as TypeFilter, isEnable: '' as EnableFilter })
const pagination = ref({ page: 1, pageSize: 20, total: 0 })

const num = (v: unknown) => Number(v || 0)
const info = (row: Record<string, unknown>) => row as unknown as TunnelInfo
const tableRows = computed(() => list.value as unknown as Record<string, unknown>[])
const clientList = computed(() => overview.value.clientList || [])

// ---- frps 卡 ----
const frpsRunning = computed(() => !!frpsStatus.value?.running)
const heroStateText = computed(() => {
  const st = frpsStatus.value
  if (!frpsConfig.value?.isEnable) return t('tools.tunnel.hero.offHint')
  if (!st?.running) return t('tools.tunnel.hero.stopped')
  const parts = [t('tools.tunnel.hero.running')]
  const start = toDate(st.startTime)
  if (start) parts.push(t('tools.tunnel.hero.uptime', { d: formatDuration(now.value - start.getTime(), t) }))
  if (st.version) parts.push(`frp v${st.version}`)
  return parts.join(' · ')
})
const serverHost = computed(() => {
  const c = frpsConfig.value
  const addr = c?.serverAddr || (c?.bindAddr && c.bindAddr !== '0.0.0.0' ? c.bindAddr : '')
  return addr
})
const heroAddr = computed(() => {
  const c = frpsConfig.value
  if (!c) return '—'
  return serverHost.value ? `${serverHost.value}:${c.bindPort || 7000}` : t('tools.tunnel.hero.addrUnset')
})
const portText = (p?: number) => (p ? String(p) : t('tools.tunnel.hero.notListening'))
const heroTiles = computed(() => {
  const c = frpsConfig.value
  const tiles = [
    { k: t('tools.tunnel.hero.bind'), v: String(c?.bindPort || 7000), sub: t('tools.tunnel.hero.bindSub'), cls: '' },
    { k: t('tools.tunnel.hero.http'), v: portText(c?.vhostHttpPort), sub: t('tools.tunnel.hero.httpSub'), cls: 'is-pc' },
    { k: t('tools.tunnel.hero.https'), v: portText(c?.vhostHttpsPort), sub: t('tools.tunnel.hero.httpsSub'), cls: 'is-pc' },
    {
      k: 'HTTP / HTTPS',
      v: `${c?.vhostHttpPort || '—'} / ${c?.vhostHttpsPort || '—'}`,
      sub: '',
      cls: 'is-mobile',
    },
    {
      k: t('tools.tunnel.hero.subdomain'),
      v: c?.subdomainHost ? `*.${c.subdomainHost}` : t('tools.tunnel.hero.notSet'),
      sub: t('tools.tunnel.hero.subdomainSub'),
      cls: '',
    },
    {
      k: t('tools.tunnel.hero.clientsOnline'),
      v: `${num(overview.value.clients)} / ${clientList.value.length}`,
      sub: '',
      cls: 'is-mobile',
    },
  ]
  return tiles
})
const heroChips = computed(() => {
  const c = frpsConfig.value
  const out = [t('tools.tunnel.hero.tcp', { port: c?.bindPort || 7000 })]
  if (c?.kcpBindPort) out.push(t('tools.tunnel.hero.kcp', { port: c.kcpBindPort }))
  if (c?.quicBindPort) out.push(t('tools.tunnel.hero.quic', { port: c.quicBindPort }))
  return out
})

// ---- 筛选 ----
const typeKey = (type?: TunnelType | string) => String(type || '').replace('TUNNEL_TYPE_', '').toLowerCase() || 'tcp'
const typeChips = computed<ToolbarChip<TypeFilter>[]>(() => {
  const counts = new Map((overview.value.typeCounts || []).map((c) => [c.type, num(c.count)]))
  const chips: ToolbarChip<TypeFilter>[] = [{ label: t('tools.tunnel.chipAll'), value: '', count: num(overview.value.total) }]
  for (const ty of TYPE_ORDER) {
    const n = counts.get(ty) || 0
    if (n > 0 || queryForm.type === ty) chips.push({ label: typeKey(ty), value: ty, count: n })
  }
  return chips
})
const enableOptions = computed(() => [
  { label: t('tools.tunnel.segAll'), value: '' as EnableFilter },
  { label: t('tools.tunnel.segOn'), value: 'true' as EnableFilter },
  { label: t('tools.tunnel.segOff'), value: 'false' as EnableFilter },
])
const setEnableFilter = (v: EnableFilter) => {
  queryForm.isEnable = v
  search()
}

// ---- 行展示 ----
const isWeb = (row: TunnelInfo) => row.type === TunnelType.TUNNEL_TYPE_HTTP || row.type === TunnelType.TUNNEL_TYPE_HTTPS
const isP2p = (row: TunnelInfo) => row.type === TunnelType.TUNNEL_TYPE_STCP || row.type === TunnelType.TUNNEL_TYPE_XTCP
const isOnline = (row: TunnelInfo) => row.status === TunnelStatus.TUNNEL_STATUS_ONLINE
const localText = (row: TunnelInfo) => `${row.localIp || '127.0.0.1'}:${row.localPort || 0}`
const firstDomain = (row: TunnelInfo) => {
  const d = (row.customDomains || '').split(',').map((s) => s.trim()).filter(Boolean)[0]
  if (d) return d
  if (row.subdomain) return frpsConfig.value?.subdomainHost ? `${row.subdomain}.${frpsConfig.value.subdomainHost}` : row.subdomain
  return ''
}
const remoteText = (row: TunnelInfo) => {
  if (isWeb(row)) return firstDomain(row) || '—'
  if (isP2p(row)) return t('tools.tunnel.p2pEntry')
  if (!row.remotePort) return '—'
  return serverHost.value ? `${serverHost.value}:${row.remotePort}` : `:${row.remotePort}`
}
const entryUrl = (row: TunnelInfo) => {
  if (!isWeb(row)) return ''
  const domain = firstDomain(row)
  if (!domain) return ''
  const https = row.type === TunnelType.TUNNEL_TYPE_HTTPS
  const port = https ? frpsConfig.value?.vhostHttpsPort : frpsConfig.value?.vhostHttpPort
  const def = https ? 443 : 80
  return `${https ? 'https' : 'http'}://${domain}${port && port !== def ? `:${port}` : ''}`
}
const trafficText = (row: TunnelInfo, dir: 'in' | 'out') => {
  if (!isOnline(row)) return '—'
  return formatBytes(dir === 'in' ? row.bytesIn : row.bytesOut)
}
const statusVariant = (row: TunnelInfo) => {
  if (isOnline(row)) return 'success'
  if (row.status === TunnelStatus.TUNNEL_STATUS_PENDING && !toDate(row.lastOfflineTime)) return 'warning'
  return 'neutral'
}
const statusLabel = (row: TunnelInfo) => {
  if (isOnline(row)) return t('tools.tunnel.status.online')
  if (!row.isEnable) return t('tools.tunnel.statusDisabled')
  if (toDate(row.lastOfflineTime)) return t('tools.tunnel.offlineAgo', { ago: formatAgo(row.lastOfflineTime, t) })
  return t('tools.tunnel.status.pending')
}
const stickerClass = (row: TunnelInfo) => {
  const k = typeKey(row.type)
  if (k === 'udp') return 'm-sticker--a'
  if (k === 'http') return 'm-sticker--p'
  if (k === 'https') return 'm-sticker--ok'
  return 'm-sticker--a2'
}

const columns = computed<DataTableColumn[]>(() => [
  { key: 'name', title: t('tools.tunnel.name'), minWidth: 150 },
  { key: 'type', title: t('tools.tunnel.proxyType'), width: 84 },
  { key: 'route', title: t('tools.tunnel.colRoute'), minWidth: 220 },
  { key: 'client', title: t('tools.tunnel.colClient'), minWidth: 170 },
  { key: 'traffic', title: t('tools.tunnel.colTraffic'), width: 110, align: 'right' },
  { key: 'connections', title: t('tools.tunnel.colConn'), width: 70, align: 'right' },
  { key: 'operation', title: t('tools.tunnel.operation'), width: 212, align: 'right' },
])

const rowActions = (row: TunnelInfo): ActionMenuItem[] => [
  { key: 'copy', label: t('tools.tunnel.copyEntry'), icon: 'HOutline:ClipboardDocumentIcon', hidden: isP2p(row) || remoteText(row) === '—' },
  { key: 'resetCredential', label: t('tools.tunnel.resetCredential'), icon: 'HOutline:ArrowPathRoundedSquareIcon' },
  { key: 'delete', label: t('tools.tunnel.delete'), icon: 'HOutline:TrashIcon', danger: true },
]

const toggleSelect = (id: string) => {
  selectedIds.value = selectedIds.value.includes(id)
    ? selectedIds.value.filter((d) => d !== id)
    : [...selectedIds.value, id]
}

const findRow = (id: string) => list.value.find((x) => x.id === id)
const onRowAction = (key: string, row: TunnelInfo) => {
  const record = findRow(row.id)
  if (!record) return
  if (key === 'edit') openEditDialog(record)
  else if (key === 'copy') copyEntry(record)
  else if (key === 'resetCredential') confirmResetCredential(record)
  else if (key === 'delete') confirmDelete(record)
}

// ---- 数据 ----
let pollTimer: ReturnType<typeof setInterval> | null = null

const getList = async (silent = false) => {
  if (!silent) loading.value = true
  try {
    const { data } = await listTunnels({
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      keywords: queryForm.keywords || undefined,
      type: queryForm.type || undefined,
      isEnable: queryForm.isEnable === '' ? undefined : queryForm.isEnable === 'true',
    })
    list.value = data?.infos || []
    pagination.value.total = Number(data?.total) || 0
    overview.value = { ...emptyOverview(), ...(data?.overview || {}) }
    const ids = new Set(list.value.map((r) => r.id))
    if (selectedIds.value.some((id) => !ids.has(id))) selectedIds.value = selectedIds.value.filter((id) => ids.has(id))
    if (statsRow.value) statsRow.value = findRow(statsRow.value.id) || statsRow.value
  } catch {
    if (!silent) {
      list.value = []
      pagination.value.total = 0
    }
  } finally {
    loading.value = false
  }
}
const search = () => {
  pagination.value.page = 1
  getList()
}

const loadFrpsConfig = async () => {
  try {
    const { data } = await getFrpsConfig()
    frpsConfig.value = data?.config || null
    frpsStatus.value = data?.status || null
  } catch {
    /* non-fatal */
  }
}
const onFrpsSaved = (config: FrpsConfig) => {
  frpsConfig.value = config
  loadFrpsConfig()
  getList()
}

const confirmRestart = () => {
  Dialog.confirm({
    title: t('tools.tunnel.hero.restartConfirmTitle'),
    content: t('tools.tunnel.hero.restartConfirm'),
    confirmText: t('tools.tunnel.hero.restartOk'),
    cancelText: t('tools.tunnel.cancel'),
    onConfirm: async () => {
      const { data } = await restartFrps()
      frpsStatus.value = data?.status || frpsStatus.value
      feedback.success(t('tools.tunnel.hero.restartDone'))
      getList(true)
    },
  })
}

const openCreateDialog = () => createRef.value?.showDialog()
const openEditDialog = (row: TunnelInfo) => createRef.value?.showDialog(row)
const openStatsDialog = (row: TunnelInfo) => {
  statsRow.value = row
  statsVisible.value = true
}
const openFrpcDialog = (row: TunnelInfo) => {
  frpcRow.value = row
  frpcRows.value = []
  frpcVisible.value = true
}
const openMergedFrpc = () => {
  const rows = selectedIds.value.map((id) => findRow(id)).filter((r): r is TunnelInfo => !!r)
  if (!rows.length) return
  frpcRow.value = rows[0]!
  frpcRows.value = rows
  frpcVisible.value = true
}
const onCredentialReset = (next: TunnelInfo) => {
  const record = findRow(next.id)
  if (record) Object.assign(record, next)
  if (frpcRow.value?.id === next.id) frpcRow.value = { ...frpcRow.value, ...next }
  frpcRows.value = frpcRows.value.map((r) => (r.id === next.id ? { ...r, ...next } : r))
}

const toggleEnable = async (row: TunnelInfo, val: boolean) => {
  const record = findRow(row.id)
  if (!record) return
  try {
    const { data } = await updateTunnel({ id: record.id, isEnable: val })
    if (data?.info) {
      Object.assign(record, data.info)
      feedback.success(data.info.isEnable ? t('tools.tunnel.enabled') : t('tools.tunnel.disabled'))
    }
    getList(true)
  } catch {
    /* interceptor */
  }
}

const batchSetEnable = async (val: boolean) => {
  const ids = [...selectedIds.value]
  const results = await Promise.allSettled(ids.map((id) => updateTunnel({ id, isEnable: val })))
  const ok = results.filter((r) => r.status === 'fulfilled').length
  feedback.success(val ? t('tools.tunnel.batchEnableSuccess', { count: ok }) : t('tools.tunnel.batchDisableSuccess', { count: ok }))
  getList(true)
}

const copyEntry = async (row: TunnelInfo) => {
  const addr = entryUrl(row) || remoteText(row)
  try {
    await navigator.clipboard.writeText(addr)
    feedback.success(t('tools.tunnel.entryCopied', { addr }))
  } catch {
    feedback.error(t('tools.tunnel.frpc.copyFailed'))
  }
}

const confirmResetCredential = (row: TunnelInfo) => {
  Dialog.confirm({
    title: t('tools.tunnel.resetCredential'),
    content: t('tools.tunnel.resetCredentialConfirm'),
    confirmText: t('tools.tunnel.resetCredentialOk'),
    cancelText: t('tools.tunnel.cancel'),
    onConfirm: async () => {
      const { data } = await resetTunnelCredential({ id: row.id })
      if (data?.info) onCredentialReset(data.info)
      feedback.success(t('tools.tunnel.resetCredentialDone'))
    },
  })
}

const confirmDelete = (row: TunnelInfo) => {
  Dialog.confirm({
    title: t('tools.tunnel.confirmDeleteTitle'),
    content: t('tools.tunnel.confirmDeleteRule'),
    confirmText: t('tools.tunnel.confirmDeleteText'),
    cancelText: t('tools.tunnel.cancel'),
    onConfirm: async () => {
      await deleteTunnel({ id: row.id })
      feedback.success(t('tools.tunnel.deleteSuccess'))
      selectedIds.value = selectedIds.value.filter((d) => d !== row.id)
      getList()
    },
  })
}

const batchDelete = () => {
  if (!selectedIds.value.length) return
  Dialog.confirm({
    title: t('tools.tunnel.confirmDeleteTitle'),
    content: t('tools.tunnel.confirmDeleteSelected'),
    confirmText: t('tools.tunnel.confirmDeleteText'),
    cancelText: t('tools.tunnel.cancel'),
    onConfirm: async () => {
      await Promise.all(selectedIds.value.map((id) => deleteTunnel({ id })))
      feedback.success(t('tools.tunnel.batchDeleteSuccess'))
      selectedIds.value = []
      getList()
    },
  })
}

const refresh = (type: 'create' | 'update') => {
  if (type === 'create') pagination.value.page = 1
  getList()
}

const startPoll = () => {
  pollTimer = setInterval(() => {
    now.value = Date.now()
    if (!document.hidden && !loading.value) void getList(true)
  }, POLL_MS)
}
const stopPoll = () => {
  if (pollTimer) clearInterval(pollTimer)
  pollTimer = null
}
onMounted(() => {
  getList()
  loadFrpsConfig()
  startPoll()
})
// 页面被缓存（切到同组其它页）时暂停轮询；回来时静默刷新一次再继续
onActivated(() => {
  if (pollTimer) return
  now.value = Date.now()
  void getList(true)
  startPoll()
})
onDeactivated(stopPoll)
onBeforeUnmount(stopPoll)
</script>

<style scoped lang="scss">
.tn-page {
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.tn-page__body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.tn-tone-ok { color: var(--m-ok); font-weight: 700; }
.tn-tone-warn { color: var(--m-warn); font-weight: 700; }
.tn-btn-count { margin-left: 2px; color: var(--el-text-color-secondary); font-weight: 700; }

/* 客户端下拉 */
.tn-clients { display: flex; flex-direction: column; padding: 8px; max-height: 420px; overflow-y: auto; }
.tn-clients__head {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 6px 8px 10px;
  b { font-size: 0.9375rem; font-weight: 800; color: var(--el-text-color-primary); }
  span { font-size: 0.75rem; color: var(--el-text-color-secondary); }
}
.tn-clients__empty { padding: 16px 8px; font-size: 0.8125rem; color: var(--el-text-color-secondary); text-align: center; }
.tn-client {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 8px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.tn-client__ico {
  width: 36px;
  height: 36px;
  flex: none;
  border-radius: 12px;
  :deep(svg) { width: 18px; height: 18px; }
}
.tn-client__main { display: flex; flex-direction: column; gap: 2px; flex: 1; min-width: 0; }
.tn-client__name { font-size: 0.8125rem; font-weight: 800; color: var(--el-text-color-primary); }
.tn-client__meta {
  font-size: 0.72rem;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* frps 主色卡 */
.tn-hero {
  position: relative;
  overflow: hidden;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 22px;
  padding: 24px 26px;
  border-radius: var(--app-radius-xl);
  background: var(--m-p);
  color: var(--m-pOn);
}
.tn-hero__blob {
  position: absolute;
  right: 240px;
  bottom: -110px;
  width: 170px;
  height: 170px;
  border-radius: 50%;
  background: var(--m-a);
}
.tn-hero__ring {
  position: absolute;
  right: -26px;
  top: -30px;
  width: 96px;
  height: 96px;
  border-radius: 50%;
  border: 14px solid var(--m-a2);
}
.tn-hero__main {
  position: relative;
  flex: 1 1 280px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.tn-hero__id { display: flex; align-items: center; gap: 12px; min-width: 0; }
.tn-hero__sticker {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 52px;
  height: 52px;
  flex: none;
  border-radius: 17px;
  background: var(--m-pOn);
  color: var(--m-p);
  transform: rotate(-6deg);
  :deep(svg) { width: 26px; height: 26px; }
}
.tn-hero__idtext { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.tn-hero__title {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 1.375rem;
  font-weight: 800;
}
.tn-hero__state { display: inline-flex; align-items: center; gap: 6px; font-size: 0.8125rem; font-weight: 700; }
.tn-hero__dot { width: 8px; height: 8px; flex: none; border-radius: 50%; background: var(--m-pOn); }
.tn-hero__dot.is-off { background: transparent; border: 2px solid var(--m-pOn); }
.tn-hero__addr { font-size: 0.875rem; font-weight: 700; word-break: break-all; }
.tn-hero__err {
  align-self: flex-start;
  padding: 6px 10px;
  border-radius: 10px;
  background: var(--m-s);
  color: var(--m-bad);
  font-size: 0.75rem;
  font-weight: 700;
}
.tn-hero__btns { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 4px; }
.tn-hero__btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 44px;
  padding: 0 18px;
  border: none;
  border-radius: 999px;
  background: color-mix(in srgb, var(--m-pOn) 18%, transparent);
  color: var(--m-pOn);
  font-size: 0.875rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
  transition: filter 0.15s, transform 0.15s;
  :deep(svg) { width: 16px; height: 16px; }
}
.tn-hero__btn:hover { filter: brightness(1.08); }
.tn-hero__btn:active { transform: scale(0.97); }
.tn-hero__btn.is-solid { background: var(--m-pOn); color: var(--m-p); }
.tn-hero__side {
  position: relative;
  flex: 2 1 520px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.tn-hero__tiles { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; }
.tn-hero__tile {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  padding: 12px 14px;
  border-radius: var(--app-radius-lg);
  background: var(--m-s);
  color: var(--m-ink);
}
.tn-hero__tile.is-mobile { display: none; }
.tn-hero__tk { font-size: 0.75rem; font-weight: 700; color: var(--m-ink2); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.tn-hero__tv {
  font-size: 1.25rem;
  font-weight: 800;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.tn-hero__ts { font-size: 0.75rem; color: var(--m-ink2); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.tn-hero__chips { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; }
.tn-hero__spacer { flex: 1; }
.tn-hero__chip {
  display: inline-flex;
  align-items: center;
  height: 26px;
  padding: 0 10px;
  border-radius: 999px;
  background: var(--m-s);
  color: var(--m-ink);
  font-size: 0.75rem;
  font-weight: 800;
  white-space: nowrap;
}
.tn-hero__chip.is-accent { background: var(--m-a); color: var(--m-ink); }

/* 启用分段 */
.tn-seg {
  display: inline-flex;
  padding: 3px;
  gap: 2px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}
.tn-seg__btn {
  min-height: 34px;
  padding: 0 14px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
}
.tn-seg__btn.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}

/* 表格单元 */
.tn-name { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.tn-name__main { font-weight: 800; color: var(--el-text-color-primary); }
.tn-name__sub {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.tn-ty {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 24px;
  min-width: 48px;
  padding: 0 8px;
  border-radius: 8px;
  font-family: var(--app-font-mono);
  font-size: 0.72rem;
  font-weight: 800;
  background: var(--m-a2Soft);
  color: var(--m-ink);
}
.tn-ty.is-udp { background: var(--m-aSoft); color: var(--m-ink); }
.tn-ty.is-http { background: var(--m-pSoft); color: var(--m-pInk); }
.tn-ty.is-https { background: var(--m-okSoft); color: var(--m-ok); }
.tn-ty.is-stcp,
.tn-ty.is-xtcp { background: var(--el-bg-color); color: var(--m-ink); box-shadow: inset 0 0 0 1.5px var(--m-ink2); }
.tn-route {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
  font-size: 0.8125rem;
}
.tn-route__local { color: var(--el-text-color-secondary); font-size: 0.75rem; }
.tn-route__line { display: flex; align-items: center; gap: 6px; min-width: 0; }
.tn-route__arrow { width: 14px; height: 14px; flex: none; color: var(--m-pInk); }
.tn-route__remote {
  min-width: 0;
  font-weight: 700;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.tn-route__remote.is-p2p,
.tn-card__remote .is-p2p { font-family: inherit; font-weight: 600; color: var(--el-text-color-secondary); }
.tn-route__ext {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  flex: none;
  border-radius: 8px;
  color: var(--el-text-color-secondary);
  :deep(svg) { width: 15px; height: 15px; }
}
.tn-route__ext:hover { background: var(--el-fill-color-light); color: var(--m-pInk); }
.tn-client-cell { display: flex; align-items: center; gap: 8px; min-width: 0; }
.tn-client-cell__host {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.tn-traffic {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  font-size: 0.8125rem;
  font-weight: 700;
  line-height: 1.4;
  color: var(--el-text-color-primary);
  white-space: nowrap;
}
.tn-up, .tn-down { font-style: normal; font-weight: 800; }
.tn-up { color: var(--m-a2); }
.tn-down { color: var(--m-pInk); }
.tn-dim { color: var(--el-text-color-secondary); font-weight: 700; }
.tn-strong { font-size: 1rem; font-weight: 800; color: var(--el-text-color-primary); }
.tn-acts { display: inline-flex; align-items: center; justify-content: flex-end; gap: 2px; }
.tn-acts :deep(.tn-act--stop), .tn-card__foot :deep(.tn-act--stop) { color: var(--m-bad); }
.tn-acts :deep(.tn-act--start), .tn-card__foot :deep(.tn-act--start) { color: var(--m-ok); }
.tn-acts :deep(.tn-act--stop:hover), .tn-card__foot :deep(.tn-act--stop:hover) { color: var(--m-bad); background: var(--m-badSoft); }
.tn-acts :deep(.tn-act--start:hover), .tn-card__foot :deep(.tn-act--start:hover) { color: var(--m-ok); background: var(--m-okSoft); }

.tn-note {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  :deep(svg) { width: 16px; height: 16px; flex: none; }
}

/* 手机卡片 */
.tn-cards { display: flex; flex-direction: column; gap: 12px; }
.tn-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px 12px 6px 16px;
  border-radius: var(--app-radius-xl);
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
}
.tn-card.is-selected { border-color: var(--m-p); box-shadow: 0 0 0 2px var(--m-pSoft); }
.tn-card__head { display: flex; align-items: center; gap: 10px; min-width: 0; }
.tn-card__sticker {
  width: 46px;
  height: 46px;
  flex: none;
  border-radius: 15px;
  font-size: 0.75rem;
  font-weight: 800;
  transform: rotate(var(--rot, 0deg));
}
.tn-card__title { display: flex; flex-direction: column; gap: 2px; flex: 1; min-width: 0; }
.tn-card__name {
  font-size: 1rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.tn-card__remark {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.tn-card__check {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  margin: 0 -8px 0 -4px;
  flex: none;
  input { width: 18px; height: 18px; accent-color: var(--m-p); }
}
.tn-card__route {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding-right: 4px;
  font-size: 0.75rem;
  min-width: 0;
}
.tn-card__remote {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  font-weight: 700;
  color: var(--el-text-color-primary);
  span { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
}
.tn-card__grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px 12px;
  margin: 0 4px 0 0;
  padding: 10px 12px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  > div { min-width: 0; }
  dt { font-size: 0.6875rem; font-weight: 700; color: var(--el-text-color-secondary); }
  dd {
    margin: 2px 0 0;
    font-size: 0.8125rem;
    font-weight: 700;
    color: var(--el-text-color-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}
.tn-card__foot {
  display: flex;
  align-items: center;
  gap: 2px;
  padding-top: 6px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.tn-card__meta { flex: 1; min-width: 0; display: flex; gap: 6px; }
.tn-skeleton {
  height: 240px;
  border-radius: var(--app-radius-xl);
  background: var(--el-fill-color-light);
  animation: tn-pulse 1.4s ease-in-out infinite;
}
@keyframes tn-pulse { 50% { opacity: 0.55; } }

/* 手机悬浮新建 */
.tn-fab {
  position: sticky;
  bottom: 0;
  z-index: 15;
  display: flex;
  justify-content: flex-end;
  pointer-events: none;
}
.tn-fab__btn {
  pointer-events: auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 60px;
  height: 60px;
  border: none;
  border-radius: 20px;
  background: var(--m-p);
  color: var(--m-pOn);
  box-shadow: var(--app-shadow-brand);
  cursor: pointer;
  :deep(.iconify) { width: 26px; height: 26px; }
}
.tn-fab__btn:active { transform: scale(0.97); }
.tn-card__foot :deep(.action-menu__trigger) {
  width: 44px;
  height: 44px;
  border-radius: 14px;
}

/* 批量条 */
.tn-batch {
  position: sticky;
  bottom: 18px;
  z-index: 20;
  align-self: center;
  display: flex;
  align-items: center;
  gap: 2px;
  max-width: 100%;
  padding: 6px 6px 6px 18px;
  border-radius: 999px;
  background: var(--m-ink);
  color: var(--m-bg);
  box-shadow: var(--app-shadow-lg);
  overflow-x: auto;
  scrollbar-width: none;
}
.tn-batch::-webkit-scrollbar { display: none; }
.tn-batch__count { margin-right: 10px; font-size: 0.875rem; font-weight: 800; white-space: nowrap; }
.tn-batch__btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 0 14px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: inherit;
  font-size: 0.875rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
}
.tn-batch__btn:hover { background: color-mix(in srgb, var(--m-bg) 14%, transparent); }
.tn-batch__btn.is-danger { color: var(--m-a); }
.tn-batch__btn.is-icon { width: 40px; padding: 0; justify-content: center; }
.tn-batch__btn :deep(svg) { width: 16px; height: 16px; }
.tn-pop-enter-active, .tn-pop-leave-active { transition: opacity 0.18s, transform 0.18s; }
.tn-pop-enter-from, .tn-pop-leave-to { opacity: 0; transform: translateY(12px); }

@media (width <= 1180px) {
  .tn-hero__side { flex-basis: 100%; }
}
@media (width <= 768px) {
  .tn-page { gap: 14px; }
  .tn-hero { flex-direction: column; align-items: stretch; gap: 12px; padding: 18px; border-radius: var(--app-radius-xl); }
  .tn-hero__blob { right: -44px; top: 40px; bottom: auto; width: 150px; height: 150px; }
  .tn-hero__ring { right: 70px; top: -22px; width: 54px; height: 54px; border-width: 9px; }
  .tn-hero__sticker { width: 46px; height: 46px; border-radius: 15px; }
  .tn-hero__title { font-size: 1.1875rem; }
  .tn-hero__state { font-size: 0.75rem; }
  .tn-hero__addr { font-size: 0.8125rem; }
  .tn-hero__main { display: contents; }
  .tn-hero__id, .tn-hero__addr, .tn-hero__err { position: relative; }
  .tn-hero__btns { position: relative; order: 3; margin-top: 0; }
  .tn-hero__btn { flex: 1; justify-content: center; padding: 0 12px; }
  .tn-hero__side { flex-basis: auto; }
  .tn-hero__tiles { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px; }
  .tn-hero__tile { padding: 10px 12px; border-radius: var(--app-radius); }
  .tn-hero__tile.is-pc { display: none; }
  .tn-hero__tile.is-mobile { display: flex; }
  .tn-hero__tv { font-size: 1rem; }
  .tn-hero__ts { display: none; }
  .tn-hero__chip { height: 24px; padding: 0 9px; font-size: 0.6875rem; }
  .tn-seg { width: 100%; }
  .tn-seg__btn { flex: 1; min-height: 44px; }
  .tn-batch { bottom: calc(78px + env(safe-area-inset-bottom)); padding-left: 14px; }
  .tn-batch__btn { padding: 0 10px; }
  .tn-batch__btn.is-icon { width: 40px; padding: 0; }
}
</style>
