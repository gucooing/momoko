<!-- GM 运营 · 玩家：标题 + 人数（MGP 汇总 + Momoko 通行证绑定数）→ 筛选 chips（全部 / 在线 / 今日活跃 / 未绑定通行证 / 封禁，带数量）+ 搜索
     → PC 表格（可多选；玩家 / 制作人等级 / 钻石 / 通行证 / 最后登录 / 状态 / 发礼物·强制下线·查看·「⋯」其他玩家指令）
     手机卡片 → 分页 → 深色批量条（manifest 里目标为 players 的指令）。导出 CSV 按当前筛选逐页读取（每页 200）。 -->
<template>
  <div class="gpl game-skin">
    <PageHeader :back="false" :theme-toggle="false">
      <template #title>{{ t('gmData.players.title') }}</template>
      <template #description>
        <span class="m-num">{{ t('gmData.players.countAll', { n: num(summary.all ?? 0) }) }}</span>
        ·
        <span class="gpl__online m-num">{{ t('gmData.players.countOnline', { n: num(summary.online ?? 0) }) }}</span>
        ·
        <span class="m-num">{{ t('gmData.players.countLinked', { n: num(linkedCount) }) }}</span>
      </template>
      <template #actions>
        <UButton color="neutral" variant="outline" icon="i-lucide-download" :loading="exporting" @click="exportCsv">
          {{ exporting ? t('gmData.players.exporting', { n: exported }) : t('gmData.players.export') }}
        </UButton>
        <UButton v-if="ctx.hasCommand('gift.send')" color="primary" icon="i-lucide-gift" @click="openGift(selectedPlayers)">
          {{ t('gmData.players.sendGift') }}
        </UButton>
      </template>
    </PageHeader>

    <ListToolbar
      v-model:status="filter"
      v-model:keyword="keyword"
      :chips="chips"
      :chips-label="t('gmData.players.filterLabel')"
      :placeholder="t('gmData.players.search')"
      @search="reload"
    />

    <div v-if="error" class="gpl__error">{{ error }}</div>

    <DataTable
      v-if="!menuStore.isMobile"
      v-model="selected"
      :columns="columns"
      :rows="rows"
      row-key="pid"
      selectable
      :loading="loading"
      :empty-text="keyword || filter !== 'all' ? t('gmData.common.noMatch') : t('gmData.players.empty')"
      @row-click="(row) => view(p(row))"
    >
      <template #cell-player="{ row }">
        <span class="gpl__who">
          <PlayerAvatar :idol-card-id="p(row).avatarIdolCardId" :name="p(row).name" :size="44" :online="p(row).online" />
          <span class="gpl__who-text">
            <span class="gpl__name">{{ p(row).name || t('gm.player.noName') }}</span>
            <span class="gpl__dim m-num">ID {{ p(row).displayId }} · {{ p(row).account }}</span>
          </span>
        </span>
      </template>
      <template #cell-level="{ row }">
        <span class="gpl__level m-num">{{ p(row).level }}</span>
      </template>
      <template #cell-jewel="{ row }">
        <span class="gpl__jewel">
          <GameImage v-if="jewelImage" :src="jewelImage" class="gpl__jewel-img" width="22" height="22" />
          <span class="gpl__strong m-num">{{ num(p(row).jewelFree) }}</span>
          <span class="gpl__dim m-num">/ {{ num(p(row).jewelPaid) }}</span>
        </span>
      </template>
      <template #cell-passport="{ row }">
        <RouterLink v-if="links.get(p(row).pid)" class="gpl__pp" :to="passportLink(p(row).pid)">
          <component :is="menuStore.iconComponents['HOutline:LinkIcon']" class="gpl__pp-icon" />
          {{ linkName(p(row).pid) }}
        </RouterLink>
        <span v-else class="gpl__dim">{{ t('gmData.players.unlinked') }}</span>
      </template>
      <template #cell-last="{ row }">
        <span class="gpl__stack">
          <span class="gpl__strong m-num">{{ lastText(p(row)) }}</span>
          <span class="gpl__dim">{{ deviceOf(p(row)) }}</span>
        </span>
      </template>
      <template #cell-status="{ row }">
        <StatusPill :variant="statusOf(p(row)).variant" :label="statusOf(p(row)).label" />
      </template>
      <template #cell-op="{ row }">
        <div class="gpl__acts">
          <AppIconButton
            v-if="ctx.hasCommand('gift.send')"
            icon="HOutline:GiftIcon"
            :label="t('gmData.players.giftTo', { name: p(row).name })"
            :box="36"
            :size="18"
            @click="openGift([p(row)])"
          />
          <AppIconButton
            v-if="ctx.hasCommand('session.kick')"
            icon="HOutline:ArrowLeftStartOnRectangleIcon"
            :label="t('gmData.players.kick', { name: p(row).name })"
            :box="36"
            :size="18"
            :class="{ 'gpl__kick': p(row).online }"
            :disabled="!p(row).online"
            @click="run('session.kick', [p(row)])"
          />
          <AppIconButton icon="HOutline:EyeIcon" :label="t('gmData.players.view', { name: p(row).name })" :box="36" :size="18" @click="view(p(row))" />
          <ActionMenu
            v-if="rowMenu.length"
            :items="rowMenu"
            :label="t('gmData.players.more', { name: p(row).name })"
            @select="(key) => run(key, [p(row)])"
          />
        </div>
      </template>
    </DataTable>

    <template v-else>
      <div v-if="loading && !players.length" class="gpl__cards">
        <div v-for="i in 4" :key="i" class="gpl__skel" />
      </div>
      <EmptyState
        v-else-if="!players.length"
        icon="HOutline:UsersIcon"
        :title="keyword || filter !== 'all' ? t('gmData.common.noMatch') : t('gmData.players.empty')"
      />
      <div v-else class="gpl__cards">
        <article v-for="pl in players" :key="pl.pid" class="gpl__card">
          <button type="button" class="gpl__card-head" @click="view(pl)">
            <PlayerAvatar :idol-card-id="pl.avatarIdolCardId" :name="pl.name" :size="44" :online="pl.online" />
            <span class="gpl__who-text">
              <span class="gpl__name">{{ pl.name || t('gm.player.noName') }}</span>
              <span class="gpl__dim m-num">ID {{ pl.displayId }} · {{ pl.account }}</span>
            </span>
            <StatusPill :variant="statusOf(pl).variant" :label="statusOf(pl).label" />
          </button>
          <dl class="gpl__card-info">
            <div>
              <dt>{{ t('gmData.players.colLevel') }}</dt>
              <dd class="m-num">{{ pl.level }}</dd>
            </div>
            <div>
              <dt>{{ t('gmData.players.colJewel') }}</dt>
              <dd class="m-num">{{ num(pl.jewelFree) }} / {{ num(pl.jewelPaid) }}</dd>
            </div>
            <div>
              <dt>{{ t('gmData.players.colPassport') }}</dt>
              <dd>{{ links.get(pl.pid) ? linkName(pl.pid) : t('gmData.players.unlinked') }}</dd>
            </div>
            <div>
              <dt>{{ t('gmData.players.colLast') }}</dt>
              <dd class="m-num">{{ lastText(pl) }}</dd>
            </div>
          </dl>
          <footer class="gpl__card-foot">
            <span class="gpl__dim gpl__card-dev">{{ deviceOf(pl) }}</span>
            <AppIconButton v-if="ctx.hasCommand('gift.send')" icon="HOutline:GiftIcon" :label="t('gmData.players.giftTo', { name: pl.name })" :box="44" @click="openGift([pl])" />
            <AppIconButton
              v-if="ctx.hasCommand('session.kick') && pl.online"
              icon="HOutline:ArrowLeftStartOnRectangleIcon"
              :label="t('gmData.players.kick', { name: pl.name })"
              :box="44"
              class="gpl__kick"
              @click="run('session.kick', [pl])"
            />
            <ActionMenu v-if="rowMenu.length" :items="rowMenu" :label="t('gmData.players.more', { name: pl.name })" @select="(key) => run(key, [pl])" />
          </footer>
        </article>
      </div>
    </template>

    <div v-if="total" class="gpl__pager">
      <span class="gpl__dim m-num">{{ t('gmData.players.range', { from: rangeFrom, to: rangeTo, total: num(total) }) }}</span>
      <Pagination v-model:page="page" v-model:page-size="pageSize" :total="total" @change="load" />
    </div>

    <Transition name="gpl-pop">
      <div v-if="selected.length" class="gpl__batch" role="toolbar" :aria-label="t('gmData.players.batchLabel')">
        <span class="gpl__batch-count">{{ t('gmData.players.selected', { n: selected.length }) }}</span>
        <button v-for="cmd in batchCommands" :key="cmd.id" type="button" class="gpl__batch-btn" :class="{ 'is-danger': cmd.risk === 'high' || cmd.risk === 'danger' }" @click="runBatch(cmd)">
          <component :is="menuStore.iconComponents[COMMAND_ICONS[cmd.id] || '']" v-if="COMMAND_ICONS[cmd.id]" />
          {{ cmd.title }}
        </button>
        <button type="button" class="gpl__batch-btn is-icon" :aria-label="t('gmData.players.clearSelection')" @click="selected = []">
          <component :is="menuStore.iconComponents['HOutline:XMarkIcon']" />
        </button>
      </div>
    </Transition>

    <GiftDrawer v-model="giftOpen" :players="giftPlayers" @done="load" />
    <GmCommandDialog v-model="dialog.open" :command="dialog.command" :targets="dialog.targets" @done="load" />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import dayjs from 'dayjs'
import { listGmPlayers } from '@/api/gameGm'
import { getRequestErrorMessage } from '@/utils/request'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ToolbarChip } from '@/components/ui/ListToolbar.vue'
import type { ManifestCommand } from '@/games/mgp'
import type { PlayerListItem } from '@/games/gakumas/types'
import type { GmPlayerLink } from '@/types/v1/game_gm'
import { useGmContext } from '../../context'
import GmCommandDialog from '../../components/GmCommandDialog.vue'
import type { GmTargetRef } from '../../components/types'
import GiftDrawer, { type GiftTarget } from '../../player/GiftDrawer.vue'
import PlayerAvatar from '../../player/shared/PlayerAvatar.vue'
import { COMMAND_ICONS } from '../../player/shared/executions'
import { catalogImage, isPermanentBan } from '../../player/shared/game'
import { useGmFormat } from '../../player/shared/format'

defineOptions({ name: 'GmPlayersTab' })

type Filter = 'all' | 'online' | 'activeToday' | 'unlinked' | 'banned'

const { t } = useI18n()
const router = useRouter()
const ctx = useGmContext()
const menuStore = useMenuStore()
const { num, when, date, duration } = useGmFormat()

const filter = ref<Filter>('all')
const keyword = ref('')
const page = ref(1)
const pageSize = ref(20)
const players = shallowRef<PlayerListItem[]>([])
const total = ref(0)
const summary = ref<Record<string, number>>({})
const linkedCount = ref(ctx.linkedAccounts.value)
const links = shallowRef(new Map<string, GmPlayerLink>())
const loading = ref(false)
const error = ref('')
const selected = ref<string[]>([])
// 选中过的玩家（翻页后仍要知道名字与头像）
const known = new Map<string, PlayerListItem>()
let seq = 0

const load = async () => {
  const id = ++seq
  loading.value = true
  error.value = ''
  try {
    const { data } = await listGmPlayers({
      serverId: ctx.serverId.value,
      q: keyword.value.trim(),
      filter: filter.value,
      cursor: page.value > 1 ? String((page.value - 1) * pageSize.value) : '',
      limit: pageSize.value,
    })
    if (id !== seq) return
    const items = (data.items as PlayerListItem[] | undefined) || []
    players.value = items
    for (const it of items) known.set(it.pid, it)
    total.value = data.total || 0
    summary.value = data.summary || {}
    linkedCount.value = data.linkedCount || 0
    links.value = new Map((data.links || []).map((l) => [l.pid, l]))
  } catch (e) {
    if (id === seq) error.value = getRequestErrorMessage(e)
  } finally {
    if (id === seq) loading.value = false
  }
}
const reload = () => {
  page.value = 1
  return load()
}
void load()

const chips = computed<ToolbarChip<Filter>[]>(() => {
  const s = summary.value
  return [
    { label: t('gmData.players.filter.all'), value: 'all', count: s.all ?? null },
    { label: t('gmData.players.filter.online'), value: 'online', count: s.online ?? null },
    { label: t('gmData.players.filter.activeToday'), value: 'activeToday', count: s.activeToday ?? null },
    { label: t('gmData.players.filter.unlinked'), value: 'unlinked', count: s.unlinked ?? null },
    { label: t('gmData.players.filter.banned'), value: 'banned', count: s.banned ?? null },
  ]
})

const columns = computed<DataTableColumn[]>(() => [
  { key: 'player', title: t('gmData.players.colPlayer'), minWidth: 240 },
  { key: 'level', title: t('gmData.players.colLevel'), width: 100 },
  { key: 'jewel', title: t('gmData.players.colJewel'), width: 170 },
  { key: 'passport', title: t('gmData.players.colPassport'), minWidth: 130 },
  { key: 'last', title: t('gmData.players.colLast'), width: 170 },
  { key: 'status', title: t('gmData.players.colStatus'), width: 150 },
  { key: 'op', title: t('gmData.players.colOp'), width: 170, align: 'right' },
])
const rows = computed(() => players.value as unknown as Record<string, unknown>[])
const p = (row: Record<string, unknown>) => row as unknown as PlayerListItem

const jewelImage = computed(() => catalogImage(ctx.game.value, 'currency', { id: 'jewelFree' }))

const linkName = (pid: string) => {
  const l = links.value.get(pid)
  return l ? l.username || l.name : ''
}
const passportLink = (pid: string) => ({ path: '/game/passport', query: { user: links.value.get(pid)?.passportUserId } })

const deviceOf = (pl: PlayerListItem) => [pl.platform, pl.device].filter(Boolean).join(' · ') || '—'
const lastText = (pl: PlayerListItem) => {
  if (pl.online && pl.sessionLoginAt) return t('gmData.players.onlineFor', { d: duration(Date.now() - pl.sessionLoginAt) })
  return pl.lastLoginAt ? when(pl.lastLoginAt) : t('gmData.players.neverLogin')
}
const statusOf = (pl: PlayerListItem): { variant: 'success' | 'error' | 'neutral'; label: string } => {
  if (pl.bannedUntil) {
    return {
      variant: 'error',
      label: isPermanentBan(pl.bannedUntil) ? t('gmData.players.bannedForever') : t('gmData.players.bannedUntil', { date: date(pl.bannedUntil) }),
    }
  }
  return pl.online ? { variant: 'success', label: t('gmData.players.online') } : { variant: 'neutral', label: t('gmData.players.offline') }
}

const rangeFrom = computed(() => (total.value ? (page.value - 1) * pageSize.value + 1 : 0))
const rangeTo = computed(() => (page.value - 1) * pageSize.value + players.value.length)

const view = (pl: PlayerListItem) => {
  void router.push(`/game/gm/${encodeURIComponent(ctx.serverId.value)}/players/${encodeURIComponent(pl.pid)}`)
}

// ---------- 指令 ----------
const ICON_SHOWN = ['gift.send', 'session.kick']
const rowMenu = computed<ActionMenuItem[]>(() =>
  (ctx.manifest.value?.commands || [])
    .filter((c) => (c.target === 'player' || c.target === 'players') && !ICON_SHOWN.includes(c.id))
    .map((c) => ({ key: c.id, label: c.title, danger: c.risk === 'danger' })),
)
const batchCommands = computed(() => (ctx.manifest.value?.commands || []).filter((c) => c.target === 'players'))

const dialog = reactive<{ open: boolean; command?: ManifestCommand; targets: GmTargetRef[] }>({ open: false, command: undefined, targets: [] })
const run = (commandId: string, list: PlayerListItem[]) => {
  const cmd = ctx.command(commandId)
  if (!cmd) return
  if (cmd.id === 'gift.send') return openGift(list)
  dialog.command = cmd
  dialog.targets = list.map((x) => ({ pid: x.pid, name: x.name }))
  dialog.open = true
}

const selectedPlayers = computed(() => selected.value.map((pid) => known.get(pid) || ({ pid, name: '' } as PlayerListItem)))
const runBatch = (cmd: ManifestCommand) => run(cmd.id, selectedPlayers.value)

const giftOpen = ref(false)
const giftPlayers = ref<GiftTarget[]>([])
const openGift = (list: PlayerListItem[]) => {
  giftPlayers.value = list.map((x) => ({ pid: x.pid, name: x.name, avatarIdolCardId: x.avatarIdolCardId }))
  giftOpen.value = true
}

// ---------- 导出 ----------
const exporting = ref(false)
const exported = ref(0)
const csvCell = (v: unknown) => {
  const s = v === null || v === undefined ? '' : String(v)
  return /[",\n\r]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s
}
const exportCsv = async () => {
  if (exporting.value) return
  exporting.value = true
  exported.value = 0
  const all: PlayerListItem[] = []
  const allLinks = new Map<string, GmPlayerLink>()
  try {
    let cursor = ''
    const seen = new Set<string>()
    do {
      const { data } = await listGmPlayers({ serverId: ctx.serverId.value, q: keyword.value.trim(), filter: filter.value, cursor, limit: 200 })
      all.push(...((data.items as PlayerListItem[] | undefined) || []))
      for (const l of data.links || []) allLinks.set(l.pid, l)
      exported.value = all.length
      seen.add(cursor)
      cursor = data.nextCursor || ''
    } while (cursor && !seen.has(cursor))

    const head = [
      'User ID',
      t('gmData.players.csv.account'),
      t('gmData.players.csv.name'),
      t('gmData.players.csv.level'),
      t('gmData.players.csv.jewelFree'),
      t('gmData.players.csv.jewelPaid'),
      t('gmData.players.csv.passport'),
      t('gmData.players.csv.created'),
      t('gmData.players.csv.lastLogin'),
      t('gmData.players.csv.online'),
      t('gmData.players.csv.platform'),
      t('gmData.players.csv.device'),
      t('gmData.players.csv.addr'),
      t('gmData.players.csv.bannedUntil'),
      t('gmData.players.csv.banReason'),
    ]
    const fmt = (ms: number) => (ms ? dayjs(ms).format('YYYY-MM-DD HH:mm:ss') : '')
    const lines = all.map((x) =>
      [
        x.displayId,
        x.account,
        x.name,
        x.level,
        x.jewelFree,
        x.jewelPaid,
        allLinks.get(x.pid)?.username || '',
        fmt(x.createdAt),
        fmt(x.lastLoginAt),
        x.online ? 1 : 0,
        x.platform,
        x.device,
        x.remoteAddr,
        x.bannedUntil ? (isPermanentBan(x.bannedUntil) ? 'permanent' : fmt(x.bannedUntil)) : '',
        x.banReason,
      ]
        .map(csvCell)
        .join(','),
    )
    const blob = new Blob([`﻿${[head.map(csvCell).join(','), ...lines].join('\r\n')}`], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `players-${ctx.server.value?.name || 'server'}-${filter.value}-${dayjs().format('YYYYMMDD-HHmm')}.csv`
    a.click()
    URL.revokeObjectURL(url)
    feedback.success(t('gmData.players.exported', { n: all.length }))
  } catch (e) {
    feedback.error(getRequestErrorMessage(e))
  } finally {
    exporting.value = false
  }
}
</script>

<style scoped lang="scss">
.gpl {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}
.gpl__online {
  color: var(--m-ok);
  font-weight: 700;
}
.gpl__error {
  font-size: 13px;
  color: var(--m-bad);
}
.gpl__who {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.gpl__who-text {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
  flex: 1;
  text-align: left;
}
.gpl__name {
  font-weight: 800;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gpl__dim {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.gpl__strong {
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.gpl__stack {
  display: inline-flex;
  flex-direction: column;
}
.gpl__level {
  font-size: 16px;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.gpl__jewel {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.gpl__jewel-img {
  width: 22px;
  height: 22px;
  object-fit: contain;
  background: transparent;
}
.gpl__pp {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 700;
  color: var(--m-pInk);
  text-decoration: none;
}
.gpl__pp-icon {
  width: 14px;
  height: 14px;
}
.gpl__acts {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 2px;
}
.gpl__kick {
  color: var(--m-bad);
}
.gpl__pager {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}
.gpl__cards {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.gpl__skel {
  height: 168px;
  border-radius: var(--app-radius-lg);
  background: var(--el-fill-color-light);
}
.gpl__card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  min-width: 0;
}
.gpl__card-head {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 44px;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  cursor: pointer;
  min-width: 0;
}
.gpl__card-info {
  margin: 0;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  padding: 10px 12px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.gpl__card-info div {
  min-width: 0;
}
.gpl__card-info dt {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.gpl__card-info dd {
  margin: 0;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gpl__card-foot {
  display: flex;
  align-items: center;
  gap: 2px;
}
.gpl__card-dev {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gpl__batch {
  position: sticky;
  bottom: 24px;
  z-index: 5;
  align-self: center;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
  max-width: 100%;
  padding: 6px 6px 6px 20px;
  border-radius: 999px;
  background: var(--m-ink);
  color: var(--m-bg);
  box-shadow: 0 16px 40px -12px color-mix(in srgb, var(--m-ink) 55%, transparent);
}
.gpl__batch-count {
  margin-right: 10px;
  font-size: 14px;
  font-weight: 700;
}
.gpl__batch-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 0 14px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--m-bg);
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}
.gpl__batch-btn:hover {
  background: color-mix(in srgb, var(--m-bg) 14%, transparent);
}
.gpl__batch-btn.is-danger {
  color: var(--m-a);
}
.gpl__batch-btn.is-icon {
  width: 40px;
  padding: 0;
  justify-content: center;
}
.gpl__batch-btn svg {
  width: 16px;
  height: 16px;
}
.gpl-pop-enter-active,
.gpl-pop-leave-active {
  transition:
    opacity 0.18s ease,
    transform 0.18s ease;
}
.gpl-pop-enter-from,
.gpl-pop-leave-to {
  opacity: 0;
  transform: translateY(12px);
}
</style>
