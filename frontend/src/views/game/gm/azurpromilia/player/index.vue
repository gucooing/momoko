<!-- 蓝色星原玩家详情：/game/gm/:serverId/azurpromilia/players/:pid?tab=…（独立路由，自己提供 GM 上下文）。
     玩家卡（昵称、ID、在线与封禁、常用操作）→ 分页：概览、道具、货币、角色、奇波、邮箱、登录记录、GM 记录。 -->
<template>
  <div class="azd" data-game="azurpromilia">
    <div class="azd__top">
      <RouterLink class="azd__back" :to="playersLink">
        <component :is="menuStore.iconComponents['HOutline:ChevronLeftIcon']" class="azd__back-icon" />
        {{ t('azurGm.player.back') }}
      </RouterLink>
      <span v-if="ctx.server.value" class="azd__server">{{ ctx.server.value.name }}</span>
    </div>

    <EmptyState v-if="error" icon="HOutline:ExclamationTriangleIcon" :title="t('azurGm.common.loadFailed', { msg: error })" />
    <template v-else-if="player">
      <section class="azd__card">
        <span class="azd__avatar">{{ (player.name || player.account || '?').slice(0, 1) }}</span>
        <span class="azd__who">
          <span class="azd__name">{{ player.name || t('azurGm.common.noCharacter') }}</span>
          <span class="azd__sub m-num">
            {{ player.displayId ? `${t('azurGm.player.fields.displayId')} ${player.displayId} · ` : '' }}{{ t('azurGm.player.fields.pid') }} {{ player.pid }}
          </span>
        </span>
        <StatusPill v-if="player.banned" variant="error" :label="banText" />
        <StatusPill v-else :variant="player.online ? 'success' : 'neutral'" :label="player.online ? t('azurGm.players.online') : t('azurGm.players.offline')" dot />
        <span class="azd__spacer" />
        <div class="azd__actions">
          <UButton size="sm" color="neutral" variant="soft" @click="act('player.mail')">{{ t('azurGm.player.actions.mail') }}</UButton>
          <UButton size="sm" color="neutral" variant="soft" @click="act('item.adjust')">{{ t('azurGm.player.actions.items') }}</UButton>
          <UButton v-if="player.online" size="sm" color="neutral" variant="soft" @click="act('player.kick')">{{ t('azurGm.player.actions.kick') }}</UButton>
          <UButton v-if="player.banned" size="sm" color="neutral" variant="soft" @click="act('player.unban')">{{ t('azurGm.player.actions.unban') }}</UButton>
          <UButton v-else size="sm" color="neutral" variant="soft" @click="act('player.ban')">{{ t('azurGm.player.actions.ban') }}</UButton>
          <UButton size="sm" color="neutral" variant="soft" @click="act('player.rename')">{{ t('azurGm.player.actions.rename') }}</UButton>
          <UButton size="sm" color="neutral" variant="soft" @click="act('player.password')">{{ t('azurGm.player.actions.password') }}</UButton>
        </div>
      </section>

      <nav class="azd__tabs">
        <RouterLink
          v-for="k in TABS"
          :key="k"
          :to="{ query: { tab: k } }"
          replace
          class="azd__tab"
          :class="{ 'is-on': tab === k }"
        >
          {{ t(`azurGm.player.tabs.${k}`) }}
        </RouterLink>
      </nav>

      <AppPanel v-if="tab === 'overview'">
        <DescriptionList :items="overviewItems" :columns="3" />
        <p v-if="!overview" class="azd__hint">{{ t('azurGm.player.noData') }}</p>
        <p v-else-if="overview.sign" class="azd__sign">{{ overview.sign }}</p>
      </AppPanel>
      <AppPanel v-else-if="tab === 'currencies'">
        <DataTable :columns="itemColumns" :rows="currencyRows" row-key="key" :loading="currenciesLoading" :empty-text="emptyText" />
      </AppPanel>
      <AppPanel v-else>
        <DataTable :columns="listColumns" :rows="listRows" row-key="key" :loading="list.loading.value" :empty-text="emptyText" />
        <p v-if="list.error.value" class="azd__error">{{ list.error.value }}</p>
        <CursorPager
          :page="list.page.value"
          :total="list.total.value"
          :has-prev="list.hasPrev.value"
          :has-next="list.hasNext.value"
          :loading="list.loading.value"
          @prev="list.prev()"
          @next="list.next()"
        />
      </AppPanel>
    </template>

    <FormDialog v-model="dialog" :title="op ? t(`gm.commands.azurpromilia.${op}`) : ''" :width="640" :show-footer="false">
      <OperationForm v-if="op && player" :server-id="serverId" :pids="[player.pid]" :fixed="op" @done="reload" />
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { listGmExecutions } from '@/api/gameGm'
import * as api from '@/api/azurpromilia'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { GetPlayerResponse } from '@/types/v1/azurpromilia/gm'
import type { Item } from '@/types/v1/azurpromilia/types'
import { provideGmContext } from '../../context'
import CursorPager from '../CursorPager.vue'
import OperationForm from '../OperationForm.vue'
import { errorMessage, fmtTime, itemName, kindName, rewardsText, useCursorPages } from '../shared'

defineOptions({ name: 'AzurGmPlayerView' })

const TABS = ['overview', 'items', 'currencies', 'heroes', 'pets', 'mails', 'logins', 'logs'] as const
type Tab = (typeof TABS)[number]
type Op = 'player.mail' | 'item.adjust' | 'player.kick' | 'player.ban' | 'player.unban' | 'player.rename' | 'player.password'

const { t } = useI18n()
const route = useRoute()
const menuStore = useMenuStore()

const serverId = computed(() => String(route.params.serverId || ''))
const pid = computed(() => String(route.params.pid || ''))
const serverRef = ref('')
watch(serverId, (id) => (serverRef.value = id), { immediate: true })
const ctx = provideGmContext(serverRef)
watch(serverRef, () => void ctx.refresh(), { immediate: true })

const tab = computed<Tab>(() => (TABS as readonly string[]).includes(String(route.query.tab)) ? (route.query.tab as Tab) : 'overview')
const playersLink = computed(() => ({ path: `/game/gm/${encodeURIComponent(serverId.value)}`, query: { tab: 'players' } }))

const detail = shallowRef<GetPlayerResponse>()
const error = ref('')
const player = computed(() => detail.value?.player)
const overview = computed(() => detail.value?.overview)
const emptyText = computed(() => (player.value?.displayId ? t('azurGm.common.empty') : t('azurGm.player.noData')))

const banText = computed(() => {
  const p = player.value
  if (!p?.banned) return ''
  return p.bannedUntil ? t('azurGm.player.banUntil', { time: fmtTime(p.bannedUntil) }) : t('azurGm.player.banForever')
})

const load = async () => {
  if (!serverId.value || !pid.value) return
  try {
    detail.value = await api.getPlayer(serverId.value, pid.value)
    error.value = ''
  } catch (e) {
    error.value = errorMessage(e)
  }
}

const overviewItems = computed(() => {
  const p = player.value
  const o = overview.value
  if (!p) return []
  const items: { label: string; value?: string | number | null }[] = [
    { label: t('azurGm.player.fields.account'), value: p.account || '—' },
    { label: t('azurGm.player.fields.passport'), value: detail.value?.link?.username || t('azurGm.player.noPassport') },
    { label: t('azurGm.player.fields.created'), value: fmtTime(p.createdAt) },
    { label: t('azurGm.player.fields.lastLogin'), value: fmtTime(p.lastLoginAt) },
    { label: t('azurGm.player.fields.gmExecutions'), value: detail.value?.gmExecutions ?? 0 },
  ]
  if (p.banned) items.push({ label: t('azurGm.player.fields.ban'), value: `${banText.value}${p.banReason ? ` · ${p.banReason}` : ''}` })
  if (o) {
    items.push(
      { label: t('azurGm.player.fields.level'), value: p.level },
      { label: t('azurGm.player.fields.exp'), value: o.exp },
      { label: t('azurGm.player.fields.registered'), value: fmtTime(o.registerAt) },
      { label: t('azurGm.player.fields.lastLogout'), value: fmtTime(o.logoutAt) },
      { label: t('azurGm.player.fields.birthday'), value: o.birthdayMonth ? t('azurGm.player.birthday', { m: o.birthdayMonth, d: o.birthdayDay }) : '—' },
      { label: t('azurGm.player.fields.heroes'), value: o.heroes },
      { label: t('azurGm.player.fields.pets'), value: o.pets },
      { label: t('azurGm.player.fields.achievements'), value: o.achievements },
      { label: t('azurGm.player.fields.logins'), value: o.logins },
    )
  }
  return items
})

// ---------- 货币 ----------
const currencies = shallowRef<Item[]>([])
const currenciesLoading = ref(false)
const itemColumns = computed<DataTableColumn[]>(() => [
  { key: 'name', title: t('azurGm.player.cols.item'), minWidth: 200 },
  { key: 'id', title: t('azurGm.common.id'), width: 110 },
  { key: 'num', title: t('azurGm.player.cols.num'), width: 120, align: 'right' },
  { key: 'deadTime', title: t('azurGm.player.cols.deadTime'), width: 150 },
])
const currencyRows = computed(() => currencies.value.map((c) => ({ key: `${c.type}:${c.id}`, name: itemName(c.type, c.id), id: c.id, num: c.num, deadTime: '—' })))
const loadCurrencies = async () => {
  currenciesLoading.value = true
  try {
    currencies.value = (await api.listPlayerCurrencies(serverId.value, pid.value)).items
  } catch {
    currencies.value = []
  } finally {
    currenciesLoading.value = false
  }
}

// ---------- 分页列表 ----------
type Row = Record<string, unknown> & { key: string }
const list = useCursorPages<Row>(async (cursor) => {
  const req = { serverId: serverId.value, pid: pid.value, cursor, limit: 20 }
  switch (tab.value) {
    case 'items': {
      const r = await api.listPlayerItems({ ...req, ids: [] })
      return { ...r, items: r.items.map((x) => ({ key: `${x.id}`, name: itemName(x.type, x.id), id: x.id, num: x.num, deadTime: fmtTime(x.deadTime) })) }
    }
    case 'heroes': {
      const r = await api.listPlayerHeroes(req)
      return {
        ...r,
        items: r.items.map((h) => ({
          key: `${h.seq}`,
          name: kindName('hero', h.id),
          level: h.level,
          rank: h.rank,
          grade: h.grade,
          favor: h.favorLevel,
          obtained: fmtTime(h.obtainedAt),
        })),
      }
    }
    case 'pets': {
      const r = await api.listPlayerPets(req)
      return {
        ...r,
        items: r.items.map((p) => ({
          key: `${p.guid}`,
          name: kindName('pet', p.id),
          petName: p.name || '—',
          level: p.level,
          rank: p.rank,
          star: p.starStage,
          locked: p.locked ? '✓' : '',
        })),
      }
    }
    case 'mails': {
      const r = await api.listPlayerMails(req)
      return {
        ...r,
        items: r.items.map((m) => ({
          key: `${m.guid}`,
          title: m.title,
          sender: m.sender || '—',
          rewards: rewardsText(m.rewards),
          created: fmtTime(m.createdAt),
          expire: fmtTime(m.expireAt),
          state: [
            m.claimed ? t('azurGm.player.mailState.claimed') : m.read ? t('azurGm.player.mailState.read') : t('azurGm.player.mailState.unread'),
            m.globalId ? t('azurGm.player.mailState.global') : '',
          ]
            .filter(Boolean)
            .join(' · '),
        })),
      }
    }
    case 'logins': {
      const r = await api.listPlayerLogins(req)
      return {
        ...r,
        items: r.items.map((l, i) => ({
          key: `${l.loginAt}:${i}`,
          loginAt: fmtTime(l.loginAt),
          logoutAt: l.logoutAt ? fmtTime(l.logoutAt) : t('azurGm.player.stillOnline'),
          device: l.deviceModel || '—',
          platform: [l.platform, l.osVersion, l.appVersion].filter(Boolean).join(' · ') || '—',
          ip: l.remoteAddr,
        })),
      }
    }
    case 'logs': {
      const page = Number(cursor || 1)
      const { data } = await listGmExecutions({
        serverId: serverId.value,
        actorKind: '',
        status: '',
        risk: '',
        command: '',
        pid: pid.value,
        q: '',
        since: undefined,
        until: undefined,
        includeDryRun: false,
        page,
        pageSize: 20,
        actorId: '',
      })
      return {
        total: data.total,
        nextCursor: page * 20 < data.total ? String(page + 1) : '',
        items: data.items.map((e) => ({
          key: e.id,
          title: e.commandTitle,
          actor: e.actorName,
          status: e.status,
          result: `${e.successCount}/${e.successCount + e.failedCount}`,
        })),
      }
    }
  }
  return { items: [], nextCursor: '', total: 0 }
})

const listColumns = computed<DataTableColumn[]>(() => {
  const c = (key: string, title: string, extra: Partial<DataTableColumn> = {}) => ({ key, title, ...extra })
  switch (tab.value) {
    case 'items':
      return itemColumns.value
    case 'heroes':
      return [
        c('name', t('azurGm.player.cols.hero'), { minWidth: 160 }),
        c('level', t('azurGm.player.cols.level'), { width: 80, align: 'right' }),
        c('rank', t('azurGm.player.cols.rank'), { width: 80, align: 'right' }),
        c('grade', t('azurGm.player.cols.grade'), { width: 80, align: 'right' }),
        c('favor', t('azurGm.player.cols.favor'), { width: 80, align: 'right' }),
        c('obtained', t('azurGm.player.cols.obtained'), { width: 150 }),
      ]
    case 'pets':
      return [
        c('name', t('azurGm.player.cols.pet'), { minWidth: 140 }),
        c('petName', t('azurGm.player.cols.petName'), { minWidth: 120 }),
        c('level', t('azurGm.player.cols.level'), { width: 80, align: 'right' }),
        c('rank', t('azurGm.player.cols.rank'), { width: 80, align: 'right' }),
        c('star', t('azurGm.player.cols.star'), { width: 80, align: 'right' }),
        c('locked', t('azurGm.player.cols.locked'), { width: 70 }),
      ]
    case 'mails':
      return [
        c('title', t('azurGm.player.cols.title'), { minWidth: 160, ellipsis: true }),
        c('sender', t('azurGm.player.cols.sender'), { width: 120, ellipsis: true }),
        c('rewards', t('azurGm.player.cols.rewards'), { minWidth: 180, ellipsis: true }),
        c('created', t('azurGm.player.cols.created'), { width: 150 }),
        c('expire', t('azurGm.player.cols.expire'), { width: 150 }),
        c('state', t('azurGm.player.cols.state'), { width: 120 }),
      ]
    case 'logins':
      return [
        c('loginAt', t('azurGm.player.cols.loginAt'), { width: 150 }),
        c('logoutAt', t('azurGm.player.cols.logoutAt'), { width: 150 }),
        c('device', t('azurGm.player.cols.device'), { minWidth: 140, ellipsis: true }),
        c('platform', t('azurGm.player.cols.platform'), { minWidth: 180, ellipsis: true }),
        c('ip', t('azurGm.player.cols.ip'), { width: 150 }),
      ]
    case 'logs':
      return [
        c('title', t('azurGm.ops.choose'), { minWidth: 160 }),
        c('actor', t('azurGm.mail.cols.operator'), { width: 140 }),
        c('status', t('azurGm.mail.cols.state'), { width: 100 }),
        c('result', t('azurGm.mail.cols.result'), { width: 100 }),
      ]
  }
  return []
})
const listRows = computed(() => list.items.value)

const loadTab = () => {
  if (!player.value) return
  if (tab.value === 'currencies') void loadCurrencies()
  else if (tab.value !== 'overview') void list.reset()
}

watch([serverId, pid], () => void load().then(loadTab), { immediate: true })
watch(tab, loadTab)

const dialog = ref(false)
const op = ref<Op>()
const act = (o: Op) => {
  op.value = o
  dialog.value = true
}
const reload = () => {
  void load().then(loadTab)
}
</script>

<style scoped lang="scss">
.azd {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
}
.azd__top {
  display: flex;
  align-items: center;
  gap: 12px;
}
.azd__back {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--el-text-color-secondary);
  text-decoration: none;
}
.azd__back-icon {
  width: 16px;
  height: 16px;
}
.azd__server {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.azd__card {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  padding: 16px;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
}
.azd__avatar {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--m-pSoft);
  color: var(--m-pInk);
  font-family: var(--app-font-display);
  font-size: 18px;
  font-weight: 600;
}
.azd__who {
  display: flex;
  flex-direction: column;
}
.azd__name {
  font-size: 17px;
  font-weight: 600;
}
.azd__sub {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.azd__spacer {
  flex: 1;
}
.azd__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.azd__tabs {
  display: flex;
  gap: 4px;
  overflow-x: auto;
  scrollbar-width: none;
  border-bottom: 1px solid var(--el-border-color-light);
}
.azd__tab {
  padding: 8px 12px;
  border-bottom: 2px solid transparent;
  color: var(--el-text-color-secondary);
  text-decoration: none;
  white-space: nowrap;
  &.is-on {
    border-bottom-color: var(--m-p);
    color: var(--el-text-color-primary);
    font-weight: 600;
  }
}
.azd__hint,
.azd__error {
  margin: 10px 0 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.azd__error {
  color: var(--m-bad);
}
.azd__sign {
  margin: 12px 0 0;
  white-space: pre-wrap;
  color: var(--el-text-color-secondary);
}
</style>
