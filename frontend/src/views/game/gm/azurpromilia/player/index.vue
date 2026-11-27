<!-- 蓝色星原玩家详情：/game/gm/:serverId/azurpromilia/players/:pid?tab=…（独立路由，自己提供 GM 上下文）。
     玩家卡（昵称、ID、在线与封禁、常用操作）→ 分页：概览、道具、货币、角色、奇波、邮箱、登录记录、GM 记录。
     道具与货币点一行直接改数量：按差值执行「调整道具与货币」（增加为发放、减少为扣除，写 GM 记录）；
     只有货币与普通道具能扣除（服务端规则），其余种类只能改大。
     角色与奇波点一行直接改：角色的等级、突破、星级、好感（hero.set），奇波的等级、星魂、锁定、名字（pet.set），只提交改了的字段。
     道具、角色、奇波按名称或 ID 搜索（名称经名称表换成 ID 交给服务端筛选），货币在本页筛选。 -->
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
        <span class="azd__avatar">
          <GameImage v-if="player.sex" :src="azurpromilia.headImage(player.head, player.sex)" :alt="player.name" eager />
          <template v-else>{{ (player.name || player.account || '?').slice(0, 1) }}</template>
        </span>
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
        <ListToolbar v-model:keyword="currencyQ" class="azd__toolbar" :placeholder="t('azurGm.player.search.currencies')" />
        <DataTable :columns="itemColumns" :rows="currencyRows" row-key="key" :loading="currenciesLoading" :empty-text="emptyText" @row-click="editRow">
          <template #cell-name="{ row }">
            <span class="azd__named">
              <GameImage v-if="row.img" :src="String(row.img)" :alt="String(row.name)" class="azd__icon" />
              {{ row.name }}
            </span>
          </template>
        </DataTable>
      </AppPanel>
      <AppPanel v-else>
        <ListToolbar
          v-if="searchTab"
          v-model:keyword="query[searchTab]"
          class="azd__toolbar"
          :placeholder="t(`azurGm.player.search.${searchTab}`)"
          @search="list.reset()"
        />
        <DataTable
          :columns="listColumns"
          :rows="listRows"
          row-key="key"
          :loading="list.loading.value"
          :empty-text="emptyText"
          v-on="rowEditor ? { rowClick: rowEditor } : {}"
        >
          <template #cell-name="{ row }">
            <span class="azd__named">
              <GameImage v-if="row.img" :src="String(row.img)" :alt="String(row.name)" class="azd__icon" />
              {{ row.name }}
            </span>
          </template>
          <template #cell-rewards="{ row }">
            <AzurRewardList :rewards="row.rewards as Reward[]" :name="itemName" compact />
          </template>
        </DataTable>
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

    <FormDialog v-model="editOpen" :title="t('azurGm.player.edit.title')" :width="420" :loading="editing" :confirm-text="t('azurGm.common.run')" @confirm="saveEdit">
      <div v-if="editTarget" class="azd__edit">
        <p class="azd__edit-name">
          {{ itemName(editTarget.type, editTarget.id) }}
          <span class="m-num">#{{ editTarget.id }} · {{ t('azurGm.player.edit.current', { num: editTarget.num }) }}</span>
        </p>
        <label class="azd__edit-field">
          <span>{{ t('azurGm.player.edit.num') }}</span>
          <UInput v-model="editNum" type="number" class="m-num" :min="editMin" step="1" autofocus @blur="clampEdit" @keydown.enter="saveEdit" />
        </label>
        <small v-if="!editCanTake">{{ t('azurGm.player.edit.onlyIncrease') }}</small>
      </div>
    </FormDialog>

    <FormDialog v-model="heroOpen" :title="t('azurGm.player.edit.heroTitle')" :width="460" :loading="heroSaving" :confirm-text="t('azurGm.common.run')" @confirm="saveHero">
      <div v-if="heroTarget" class="azd__edit">
        <p class="azd__edit-name">
          {{ kindName('hero', heroTarget.id) }}
          <span class="m-num">#{{ heroTarget.id }}</span>
        </p>
        <div class="azd__edit-grid">
          <label v-for="f in HERO_FIELDS" :key="f" class="azd__edit-field">
            <span>{{ t(`azurGm.player.cols.${f === 'favorLevel' ? 'favor' : f}`) }}</span>
            <UInput
              v-model="heroForm[f]"
              type="number"
              class="m-num"
              :min="heroRange(f)?.[0]"
              :max="heroRange(f)?.[1]"
              step="1"
              @blur="clampHeroField(f)"
              @keydown.enter="saveHero"
            />
          </label>
        </div>
        <p v-if="heroRankAfter !== undefined" class="azd__edit-rank">
          {{ t('azurGm.player.cols.rank') }}
          <span class="m-num">{{ heroTarget.rank === heroRankAfter ? heroRankAfter : `${heroTarget.rank} → ${heroRankAfter}` }}</span>
        </p>
        <small>{{ t('azurGm.player.edit.heroHint') }}</small>
      </div>
    </FormDialog>

    <FormDialog v-model="petOpen" :title="t('azurGm.player.edit.petTitle')" :width="460" :loading="petSaving" :confirm-text="t('azurGm.common.run')" @confirm="savePet">
      <div v-if="petTarget" class="azd__edit">
        <p class="azd__edit-name">
          {{ kindName('pet', petTarget.id) }}
          <span class="m-num">#{{ petTarget.id }} · guid {{ petTarget.guid }}</span>
        </p>
        <div class="azd__edit-grid">
          <label class="azd__edit-field">
            <span>{{ t('azurGm.player.cols.level') }}</span>
            <UInput
              v-model="petForm.level"
              type="number"
              class="m-num"
              :min="petLevelRange[0]"
              :max="petLevelRange[1]"
              step="1"
              @blur="clampPetForm"
              @keydown.enter="savePet"
            />
          </label>
          <label class="azd__edit-field">
            <span>{{ t('azurGm.player.cols.petName') }}</span>
            <UInput v-model="petForm.name" maxlength="32" @keydown.enter="savePet" />
          </label>
          <label class="azd__edit-field">
            <span>{{ t('azurGm.player.edit.starStage') }}</span>
            <UInput
              v-model="petForm.starStage"
              type="number"
              class="m-num"
              :disabled="!petSoul"
              step="1"
              @blur="clampPetForm"
              @keydown.enter="savePet"
            />
          </label>
          <label class="azd__edit-field">
            <span>{{ t('azurGm.player.edit.starNode') }}</span>
            <UInput
              v-model="petForm.starNode"
              type="number"
              class="m-num"
              :disabled="!petSoul"
              step="1"
              @blur="clampPetForm"
              @keydown.enter="savePet"
            />
          </label>
        </div>
        <label class="azd__edit-check">
          <AppSwitch v-model="petForm.locked" :aria-label="t('azurGm.player.cols.locked')" />
          {{ t('azurGm.player.cols.locked') }}
        </label>
        <small>{{ t(petSoul ? 'azurGm.player.edit.petHint' : 'azurGm.player.edit.petHintNoSoul') }}</small>
      </div>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { listGmExecutions } from '@/api/gameGm'
import * as api from '@/api/azurpromilia'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { GetPlayerResponse, OperationResponse, SetPetRequest } from '@/types/v1/azurpromilia/gm'
import type { Item, Pet, Reward } from '@/types/v1/azurpromilia/types'
import { typeOfKind } from '@/games/azurpromilia/names'
import {
  clampPetLevel,
  clampStar,
  favorRange,
  heroGrowth,
  heroLevelRange,
  petLevelRange,
  petStarsoul,
  rankForLevel,
} from '@/games/azurpromilia/growth'
import { azurpromilia } from '@/games'
import GameImage from '@/components/game/GameImage.vue'
import AzurRewardList from '@/components/game/AzurRewardList.vue'
import { fuzzyMatch } from '@/games/search'
import { provideGmContext } from '../../context'
import CursorPager from '../CursorPager.vue'
import OperationForm from '../OperationForm.vue'
import { canTake, errorMessage, fmtTime, itemName, kindName, searchItems, useCursorPages } from '../shared'

defineOptions({ name: 'AzurGmPlayerView' })

const TABS = ['overview', 'items', 'currencies', 'heroes', 'pets', 'mails', 'logins', 'logs'] as const
type Tab = (typeof TABS)[number]
type Op = 'player.mail' | 'item.adjust' | 'player.kick' | 'player.ban' | 'player.unban' | 'player.rename' | 'player.password'

const { t, te } = useI18n()
const route = useRoute()
const menuStore = useMenuStore()
const fb = useFeedback()

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
// 货币一次读全，搜索在本页按名称与 ID 筛选
const currencyQ = ref('')
const currencyRows = computed(() => {
  const q = currencyQ.value.trim()
  return currencies.value
    .map((c) => ({
      key: `${c.type}:${c.id}`,
      type: c.type,
      img: azurpromilia.rewardImage(c.type, c.id),
      name: itemName(c.type, c.id),
      id: c.id,
      num: c.num,
      deadTime: '—',
    }))
    .filter((r) => !q || String(r.id) === q || fuzzyMatch(q, [r.name]))
})
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
// 名称取自按需加载的名称表：行里放取名称的函数，listRows 渲染时再取，名称表加载完后自动换成名称。
type Row = Record<string, unknown> & { key: string }

// 道具、角色、奇波的搜索：关键字经名称表换成 ID（数字直接当 ID）交给服务端筛选，服务端一次最多收 100 个 ID。
// 背包里都是普通道具（common_item），角色与奇波分别是 hero、pet 表。
const SEARCH_KINDS = { items: 'common_item', heroes: 'hero', pets: 'pet' } as const
type SearchTab = keyof typeof SEARCH_KINDS
const SEARCH_IDS_MAX = 100
const query = reactive<Record<SearchTab, string>>({ items: '', heroes: '', pets: '' })
const searchTab = computed(() => (tab.value in SEARCH_KINDS ? (tab.value as SearchTab) : undefined))
// 分页的 ID 筛选：没有关键字时为 []（不筛选），关键字换不出 ID 时为 null（结果为空）
const searchIds = async (which: SearchTab): Promise<number[] | null> => {
  const q = query[which].trim()
  if (!q) return []
  const ids = new Set<number>()
  if (/^\d+$/.test(q)) ids.add(Number(q))
  for (const hit of await searchItems(typeOfKind(SEARCH_KINDS[which]), q, SEARCH_IDS_MAX)) ids.add(hit.id)
  return ids.size ? [...ids].slice(0, SEARCH_IDS_MAX) : null
}
const EMPTY_PAGE = { items: [], nextCursor: '', total: 0 }

const list = useCursorPages<Row>(async (cursor) => {
  const req = { serverId: serverId.value, pid: pid.value, cursor, limit: 20 }
  switch (tab.value) {
    case 'items': {
      const ids = await searchIds('items')
      if (!ids) return EMPTY_PAGE
      const r = await api.listPlayerItems({ ...req, ids })
      return {
        ...r,
        items: r.items.map((x) => ({
          key: `${x.id}`,
          type: x.type,
          img: () => azurpromilia.rewardImage(x.type, x.id),
          name: () => itemName(x.type, x.id),
          id: x.id,
          num: x.num,
          deadTime: fmtTime(x.deadTime),
        })),
      }
    }
    case 'heroes': {
      const ids = await searchIds('heroes')
      if (!ids) return EMPTY_PAGE
      const r = await api.listPlayerHeroes({ ...req, ids })
      return {
        ...r,
        items: r.items.map((h) => ({
          key: `${h.seq}`,
          seq: h.seq,
          id: h.id,
          favorLevel: h.favorLevel,
          img: () => azurpromilia.heroImage(h.id),
          name: () => kindName('hero', h.id),
          level: h.level,
          rank: h.rank,
          grade: h.grade,
          favor: h.favorLevel,
          obtained: fmtTime(h.obtainedAt),
        })),
      }
    }
    case 'pets': {
      const ids = await searchIds('pets')
      if (!ids) return EMPTY_PAGE
      const r = await api.listPlayerPets({ ...req, ids })
      return {
        ...r,
        items: r.items.map((p) => ({
          key: `${p.guid}`,
          raw: p,
          img: () => azurpromilia.petImage(p.id),
          name: () => kindName('pet', p.id),
          petName: p.name || '—',
          level: p.level,
          rank: p.rank,
          star: `${p.starStage}-${p.starNode}`,
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
          rewards: m.rewards,
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
const listRows = computed(() =>
  list.items.value.map((r) => Object.fromEntries(Object.entries(r).map(([k, v]) => [k, typeof v === 'function' ? v() : v])) as Row),
)

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

// ---------- 道具与货币：点一行改数量 ----------
// 按差值执行「调整道具与货币」：改大为发放、改小为扣除（只有 canTake 的种类能扣），写 GM 记录。
const INT32_MAX = 2147483647
const editOpen = ref(false)
const editing = ref(false)
const editTarget = ref<{ type: number; id: number; num: number }>()
const editNum = ref('')
const editCanTake = computed(() => !!editTarget.value && canTake(editTarget.value.type))
const editMin = computed(() => (editCanTake.value ? 0 : editTarget.value?.num ?? 0))

const editRow = (row: Record<string, unknown>) => {
  editTarget.value = { type: Number(row.type), id: Number(row.id), num: Number(row.num) }
  editNum.value = String(row.num)
  editOpen.value = true
}

// 数量超出范围时改成最小值（不能扣减的种类为当前数量，其余为 0）或最大值（一次最多增加 INT32_MAX），写回输入框
const clampEdit = () => {
  const target = editTarget.value
  const raw = numOf(editNum.value)
  if (!target || raw === undefined) return undefined
  const next = Math.min(target.num + INT32_MAX, Math.max(editMin.value, raw))
  editNum.value = String(next)
  return next
}

const saveEdit = async () => {
  const target = editTarget.value
  const who = player.value?.pid
  if (!target || !who || editing.value) return
  const next = clampEdit()
  if (next === undefined) {
    fb.warning(t('azurGm.player.edit.invalid'))
    return
  }
  const delta = next - target.num
  if (delta === 0) {
    editOpen.value = false
    return
  }
  editing.value = true
  try {
    const change = [{ type: target.type, id: target.id, num: Math.abs(delta) }]
    const res = await api.adjustItems({
      serverId: serverId.value,
      pids: [who],
      dryRun: false,
      grant: delta > 0 ? change : [],
      take: delta < 0 ? change : [],
    })
    const result = res.results[0]?.target
    if (result && result.status !== 'success') {
      fb.error(result.message || result.code)
      return
    }
    fb.success(t('azurGm.player.edit.done'))
    editOpen.value = false
    if (tab.value === 'currencies') void loadCurrencies()
    else void list.load()
  } catch (e) {
    fb.error(errorMessage(e))
  } finally {
    editing.value = false
  }
}

// 单个玩家的操作（hero.set / pet.set）：失败时提示原因，成功时关掉弹窗并刷新当前页。
const finishSet = async (run: () => Promise<OperationResponse>, close: () => void) => {
  try {
    const res = await run()
    const result = res.results[0]?.target
    if (result && result.status !== 'success') {
      // 游戏规则拒绝时 message 为游戏的错误码名
      const key = `azurGm.player.edit.errors.${result.message}`
      fb.error(result.message && te(key) ? t(key) : result.message || result.code)
      return
    }
    fb.success(t('azurGm.player.edit.done'))
    close()
    void list.load()
  } catch (e) {
    fb.error(errorMessage(e))
  }
}

// 数字输入框（UInput type=number）运行时回传的是数字（清空时是空串），这里两种都接受；小数取整，不是数字时为 undefined
const numOf = (v: unknown) => {
  const s = String(v ?? '').trim()
  const n = Number(s)
  return s !== '' && Number.isFinite(n) ? Math.trunc(n) : undefined
}

// ---------- 角色：点一行改等级、星级、好感；突破跟着等级走 ----------
// 超出范围的值改成最大或最小值（范围来自 growth.json，与游戏服务端的配置一致）；突破取设定等级允许的最高一档。
const HERO_FIELDS = ['level', 'grade', 'favorLevel'] as const
type HeroField = (typeof HERO_FIELDS)[number]
const heroOpen = ref(false)
const heroSaving = ref(false)
const heroTarget = ref<{ seq: number; id: number; rank: number } & Record<HeroField, number>>()
const heroForm = reactive<Record<HeroField, string>>({ level: '', grade: '', favorLevel: '' })
const heroConf = computed(() => (heroTarget.value ? heroGrowth(heroTarget.value.id) : undefined))

const heroRange = (f: HeroField): [number, number] | undefined => {
  if (f === 'level') return heroLevelRange
  if (f === 'favorLevel') return favorRange
  return heroConf.value?.grade
}

const clampHeroValue = (f: HeroField, v: number) => {
  const r = heroRange(f)
  return r ? Math.min(r[1], Math.max(r[0], v)) : v
}

const clampHeroField = (f: HeroField) => {
  const v = numOf(heroForm[f])
  if (v !== undefined) heroForm[f] = String(clampHeroValue(f, v))
}

const heroRankAfter = computed(() => {
  const conf = heroConf.value
  const level = numOf(heroForm.level)
  if (!conf || level === undefined) return undefined
  return rankForLevel(conf, clampHeroValue('level', level))
})

const editHero = (row: Record<string, unknown>) => {
  const h = {
    seq: Number(row.seq),
    id: Number(row.id),
    level: Number(row.level),
    rank: Number(row.rank),
    grade: Number(row.grade),
    favorLevel: Number(row.favorLevel),
  }
  heroTarget.value = h
  for (const f of HERO_FIELDS) heroForm[f] = String(h[f])
  heroOpen.value = true
}

const saveHero = async () => {
  const h = heroTarget.value
  const who = player.value?.pid
  if (!h || !who || heroSaving.value) return
  const edit: Partial<Record<HeroField | 'rank', number>> = {}
  for (const f of HERO_FIELDS) {
    const raw = numOf(heroForm[f])
    if (raw === undefined) {
      fb.warning(t('azurGm.player.edit.invalid'))
      return
    }
    const v = clampHeroValue(f, raw)
    heroForm[f] = String(v)
    if (v !== h[f]) edit[f] = v
  }
  const rank = heroRankAfter.value
  if (rank !== undefined && rank !== h.rank) edit.rank = rank
  if (!Object.keys(edit).length) {
    heroOpen.value = false
    return
  }
  heroSaving.value = true
  await finishSet(
    () => api.setPlayerHero({ serverId: serverId.value, pid: who, seq: h.seq, dryRun: false, ...edit }),
    () => (heroOpen.value = false),
  )
  heroSaving.value = false
}

// ---------- 奇波：点一行改等级、星魂（阶段与节点一起）、锁定、名字 ----------
// 等级超出范围时改成最大或最小值；星魂改成模板里能到达、且这个等级够得着的状态（太大取最远的，太小取第 1 阶第 1 节点）。
// 没有星魂的奇波不能改星魂。
const petOpen = ref(false)
const petSaving = ref(false)
const petTarget = ref<Pet>()
const petForm = reactive({ level: '', starStage: '', starNode: '', locked: false, name: '' })
const petSoul = computed(() => (petTarget.value ? petStarsoul(petTarget.value.id) : undefined))

// 按当前输入夹取等级与星魂并写回输入框；输入不是数字时为 undefined
const clampPetForm = () => {
  const p = petTarget.value
  const rawLevel = numOf(petForm.level)
  if (!p || rawLevel === undefined) return undefined
  const level = clampPetLevel(rawLevel)
  petForm.level = String(level)
  let star = { stage: p.starStage, node: p.starNode }
  const soul = petSoul.value
  if (soul) {
    const stage = numOf(petForm.starStage)
    const node = numOf(petForm.starNode)
    if (stage === undefined || node === undefined) return undefined
    star = clampStar(soul, stage, node, level)
  }
  petForm.starStage = String(star.stage)
  petForm.starNode = String(star.node)
  return { level, ...star }
}

const editPet = (row: Record<string, unknown>) => {
  const p = row.raw as Pet
  petTarget.value = p
  Object.assign(petForm, { level: String(p.level), starStage: String(p.starStage), starNode: String(p.starNode), locked: p.locked, name: p.name })
  petOpen.value = true
}

const savePet = async () => {
  const p = petTarget.value
  const who = player.value?.pid
  if (!p || !who || petSaving.value) return
  const values = clampPetForm()
  if (!values) {
    fb.warning(t('azurGm.player.edit.invalid'))
    return
  }
  const { level, stage, node } = values
  const name = petForm.name.trim()
  const edit: Partial<Pick<SetPetRequest, 'level' | 'starStage' | 'starNode' | 'locked' | 'name'>> = {}
  if (level !== p.level) edit.level = level
  if (stage !== p.starStage || node !== p.starNode) Object.assign(edit, { starStage: stage, starNode: node })
  if (petForm.locked !== p.locked) edit.locked = petForm.locked
  if (name !== p.name) edit.name = name
  if (!Object.keys(edit).length) {
    petOpen.value = false
    return
  }
  petSaving.value = true
  await finishSet(
    () => api.setPlayerPet({ serverId: serverId.value, pid: who, guid: p.guid, dryRun: false, ...edit }),
    () => (petOpen.value = false),
  )
  petSaving.value = false
}

// 列表点一行打开的修改（道具、角色、奇波；其余分页不可点）
const rowEditor = computed(() => (searchTab.value ? { items: editRow, heroes: editHero, pets: editPet }[searchTab.value] : undefined))
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
  overflow: hidden;
  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
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
.azd__named {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.azd__icon {
  flex: none;
  width: 32px;
  height: 32px;
  object-fit: contain;
}
.azd__toolbar {
  margin-bottom: 12px;
}
.azd__edit {
  display: flex;
  flex-direction: column;
  gap: 12px;
  > small {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
}
.azd__edit-name {
  margin: 0;
  font-weight: 600;
  > span {
    margin-left: 6px;
    font-weight: 400;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
}
.azd__edit-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
.azd__edit-rank {
  margin: 0;
  font-size: 13px;
  > span {
    margin-left: 6px;
    font-weight: 600;
  }
}
.azd__edit-check {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}
.azd__edit-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  > span {
    font-size: 13px;
    color: var(--el-text-color-secondary);
  }
}
</style>
