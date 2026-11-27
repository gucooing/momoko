<!-- 蓝色星原 游戏空间的列表分页：道具、货币、角色、奇波、邮箱、登录记录（取自游戏服务端已保存的数据，逐页前后翻）。
     道具、角色、奇波按名称或 ID 搜索（名称经名称表换成 ID 交给服务端筛选），货币在本页筛选。
     名称取自按需加载的名称表：行里放取名称的函数，渲染时再取，名称表加载完后自动换成名称。 -->
<template>
  <div class="azl">
    <p v-if="!hasCharacter" class="pp-hint">{{ t('game.azur.noData') }}</p>
    <template v-else>
      <input
        v-if="searchable"
        v-model="keyword"
        type="search"
        class="azl__search"
        :placeholder="t(`game.azur.search.${kind}`)"
        :aria-label="t(`game.azur.search.${kind}`)"
      />
      <p v-if="error" class="pp-hint azl__error">{{ t('game.loadFailed', { msg: error }) }}</p>
      <div class="pp-card azl__table">
        <table>
          <thead>
            <tr>
              <th v-for="c in columns" :key="c.key" :class="{ 'is-num': c.num }">{{ c.label }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in shown" :key="r.key">
              <td v-for="c in columns" :key="c.key" :class="{ 'is-num': c.num, 'pp-num': c.num }">
                <span v-if="c.key === 'name' && r.img" class="azl__named">
                  <GameImage :src="String(r.img)" :alt="String(r.name)" class="azl__icon" />
                  {{ r.name }}
                </span>
                <AzurRewardList v-else-if="c.key === 'rewards'" :rewards="rewardsOf(r.key)" :name="itemName" compact />
                <template v-else>{{ r[c.key] }}</template>
              </td>
            </tr>
            <tr v-if="!shown.length && !loading">
              <td :colspan="columns.length" class="pp-hint azl__empty">{{ t('game.azur.empty') }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-if="total > 0 || page > 1" class="azl__pager">
        <span class="pp-hint pp-num">{{ t('game.azur.page', { n: page, total }) }}</span>
        <PButton size="sm" variant="soft" :disabled="page <= 1 || loading" @click="load(page - 1)">{{ t('game.azur.prev') }}</PButton>
        <PButton size="sm" variant="soft" :disabled="!next || loading" @click="load(page + 1)">{{ t('game.azur.next') }}</PButton>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import PButton from '@passport/components/PButton.vue'
import GameImage from '@/components/game/GameImage.vue'
import AzurRewardList from '@/components/game/AzurRewardList.vue'
import type { Reward } from '@/types/v1/azurpromilia/types'
import { azurpromilia } from '@/games'
import * as azur from '@passport/api/azurpromilia'
import { errorText, fmtNum, fmtTime } from '@passport/utils/format'
import { typeOfKind } from '@/games/azurpromilia/names'
import { fuzzyMatch } from '@/games/search'
import { itemName, kindName, searchItems } from './names'

type Kind = 'items' | 'currencies' | 'heroes' | 'pets' | 'mails' | 'logins'
type Cell = string | number | (() => string)
type Row = Record<string, Cell> & { key: string }

const props = defineProps<{ serverId: string; kind: Kind; hasCharacter: boolean }>()
const { t } = useI18n()
const LIMIT = 20

const time = (ms: number) => (ms ? fmtTime(ms) : '—')

const columns = computed(() => {
  const c = (key: string, label: string, num = false) => ({ key, label: t(label), num })
  switch (props.kind) {
    case 'items':
    case 'currencies':
      return [c('name', 'game.azur.cols.item'), c('num', 'game.azur.cols.num', true), c('deadTime', 'game.azur.cols.deadTime')]
    case 'heroes':
      return [c('name', 'game.azur.cols.hero'), c('level', 'game.azur.cols.level', true), c('rank', 'game.azur.cols.rank', true), c('grade', 'game.azur.cols.grade', true), c('favor', 'game.azur.cols.favor', true), c('obtained', 'game.azur.cols.obtained')]
    case 'pets':
      return [c('name', 'game.azur.cols.pet'), c('petName', 'game.azur.cols.petName'), c('level', 'game.azur.cols.level', true), c('rank', 'game.azur.cols.rank', true), c('star', 'game.azur.cols.star', true)]
    case 'mails':
      return [c('title', 'game.azur.cols.title'), c('sender', 'game.azur.cols.sender'), c('rewards', 'game.azur.cols.rewards'), c('created', 'game.azur.cols.created'), c('expire', 'game.azur.cols.expire'), c('state', 'game.azur.cols.state')]
    case 'logins':
      return [c('loginAt', 'game.azur.cols.loginAt'), c('logoutAt', 'game.azur.cols.logoutAt'), c('device', 'game.azur.cols.device'), c('platform', 'game.azur.cols.platform'), c('ip', 'game.azur.cols.ip')]
  }
  return []
})

const rows = shallowRef<Row[]>([])
// 邮件的附件（按行 key），在表格里显示成图标
const mailRewards = shallowRef(new Map<string, Reward[]>())
const rewardsOf = (key: unknown) => mailRewards.value.get(String(key))
// 显示的行：取名称的函数在这里取值；货币按关键字在本页筛选
const shown = computed(() => {
  const list = rows.value.map((r) => Object.fromEntries(Object.entries(r).map(([k, v]) => [k, typeof v === 'function' ? v() : v])) as Record<string, string | number>)
  const q = keyword.value.trim()
  if (props.kind !== 'currencies' || !q) return list
  return list.filter((r) => String(r.id) === q || fuzzyMatch(q, [String(r.name)]))
})
const total = ref(0)
const page = ref(1)
const next = ref('')
const loading = ref(false)
const error = ref('')
const cursors = ref<string[]>([''])

// 搜索：背包里都是普通道具（common_item），角色与奇波分别是 hero、pet 表；服务端一次最多收 100 个 ID
const SEARCH_KINDS: Partial<Record<Kind, string>> = { items: 'common_item', heroes: 'hero', pets: 'pet' }
const SEARCH_IDS_MAX = 100
const searchable = computed(() => props.kind === 'currencies' || !!SEARCH_KINDS[props.kind])
const keyword = ref('')
// 分页的 ID 筛选：没有关键字时为 []（不筛选），关键字换不出 ID 时为 null（结果为空）
const searchIds = async (): Promise<number[] | null> => {
  const kind = SEARCH_KINDS[props.kind]
  const q = keyword.value.trim()
  if (!kind || !q) return []
  const ids = new Set<number>()
  if (/^\d+$/.test(q)) ids.add(Number(q))
  for (const hit of await searchItems(typeOfKind(kind), q, SEARCH_IDS_MAX)) ids.add(hit.id)
  return ids.size ? [...ids].slice(0, SEARCH_IDS_MAX) : null
}
const EMPTY_PAGE = { items: [], nextCursor: '', total: 0 }

const fetchPage = async (cursor: string): Promise<{ items: Row[]; nextCursor: string; total: number }> => {
  const id = props.serverId
  switch (props.kind) {
    case 'items': {
      const ids = await searchIds()
      if (!ids) return EMPTY_PAGE
      const r = await azur.listMyItems(id, cursor, LIMIT, ids)
      return {
        ...r,
        items: r.items.map((x) => ({
          key: `${x.id}`,
          img: () => azurpromilia.rewardImage(x.type, x.id),
          name: () => itemName(x.type, x.id),
          num: fmtNum(x.num),
          deadTime: time(x.deadTime),
        })),
      }
    }
    case 'currencies': {
      const r = await azur.listMyCurrencies(id)
      return {
        ...r,
        nextCursor: '',
        items: r.items.map((x) => ({
          key: `${x.id}`,
          id: x.id,
          img: () => azurpromilia.rewardImage(x.type, x.id),
          name: () => itemName(x.type, x.id),
          num: fmtNum(x.num),
          deadTime: '—',
        })),
      }
    }
    case 'heroes': {
      const ids = await searchIds()
      if (!ids) return EMPTY_PAGE
      const r = await azur.listMyHeroes(id, cursor, LIMIT, ids)
      return {
        ...r,
        items: r.items.map((h) => ({
          key: `${h.seq}`,
          img: () => azurpromilia.heroImage(h.id),
          name: () => kindName('hero', h.id),
          level: h.level,
          rank: h.rank,
          grade: h.grade,
          favor: h.favorLevel,
          obtained: time(h.obtainedAt),
        })),
      }
    }
    case 'pets': {
      const ids = await searchIds()
      if (!ids) return EMPTY_PAGE
      const r = await azur.listMyPets(id, cursor, LIMIT, ids)
      return {
        ...r,
        items: r.items.map((p) => ({
          key: `${p.guid}`,
          img: () => azurpromilia.petImage(p.id),
          name: () => kindName('pet', p.id),
          petName: p.name || '—',
          level: p.level,
          rank: p.rank,
          star: `${p.starStage}-${p.starNode}`,
        })),
      }
    }
    case 'mails': {
      const r = await azur.listMyMails(id, cursor, LIMIT)
      mailRewards.value = new Map(r.items.map((m) => [`${m.guid}`, m.rewards]))
      return {
        ...r,
        items: r.items.map((m) => ({
          key: `${m.guid}`,
          title: m.title,
          sender: m.sender || '—',
          rewards: '',
          created: time(m.createdAt),
          expire: time(m.expireAt),
          state: [
            m.claimed ? t('game.azur.mailState.claimed') : m.read ? t('game.azur.mailState.read') : t('game.azur.mailState.unread'),
            m.globalId ? t('game.azur.mailState.global') : '',
          ]
            .filter(Boolean)
            .join(' · '),
        })),
      }
    }
    case 'logins': {
      const r = await azur.listMyLogins(id, cursor, LIMIT)
      return {
        ...r,
        items: r.items.map((l, i) => ({
          key: `${l.loginAt}:${i}`,
          loginAt: time(l.loginAt),
          logoutAt: l.logoutAt ? time(l.logoutAt) : t('game.azur.noLogout'),
          device: l.deviceModel || '—',
          platform: [l.platform, l.osVersion, l.appVersion].filter(Boolean).join(' · ') || '—',
          ip: l.remoteAddr,
        })),
      }
    }
  }
  return { items: [], nextCursor: '', total: 0 }
}

const load = async (n: number) => {
  loading.value = true
  error.value = ''
  try {
    const res = await fetchPage(cursors.value[n - 1] || '')
    rows.value = res.items
    total.value = res.total
    next.value = res.nextCursor
    page.value = n
    if (res.nextCursor) cursors.value[n] = res.nextCursor
  } catch (e) {
    error.value = errorText(e)
  } finally {
    loading.value = false
  }
}

if (props.hasCharacter) void load(1)

// 关键字变化后从第一页重新读（货币只在本页筛选，不重新读）
let timer: ReturnType<typeof setTimeout> | undefined
watch(keyword, () => {
  if (props.kind === 'currencies') return
  clearTimeout(timer)
  timer = setTimeout(() => {
    cursors.value = ['']
    void load(1)
  }, 300)
})
</script>

<style scoped>
.azl {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.azl__search {
  align-self: flex-start;
  width: min(100%, 280px);
  height: 32px;
  padding: 0 14px;
  border: 1px solid var(--m-line);
  border-radius: 999px;
  background: var(--m-s);
  color: var(--m-ink);
  font: inherit;
  font-size: 13px;
}
.azl__search:focus {
  outline: none;
  border-color: var(--m-g);
}
.azl__named {
  display: inline-flex;
  align-items: center;
  gap: 10px;
}
.azl__icon {
  flex: none;
  width: 32px;
  height: 32px;
  object-fit: contain;
}
.azl__error {
  color: var(--m-bad);
}
.azl__table {
  overflow-x: auto;
}
.azl__table table {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
}
.azl__table th,
.azl__table td {
  padding: 10px 14px;
  text-align: left;
  white-space: nowrap;
  border-top: 1px solid var(--m-line);
}
.azl__table th {
  border-top: none;
  font-size: 12px;
  font-weight: 500;
  color: var(--m-ink2);
}
.azl__table .is-num {
  text-align: right;
}
.azl__empty {
  text-align: center;
}
.azl__pager {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}
</style>
