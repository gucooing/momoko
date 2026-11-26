<!-- 蓝色星原 游戏空间的列表分页：道具、货币、角色、奇波、邮箱、登录记录（取自游戏服务端已保存的数据，逐页前后翻）。 -->
<template>
  <div class="azl">
    <p v-if="!hasCharacter" class="pp-hint">{{ t('game.azur.noData') }}</p>
    <template v-else>
      <p v-if="error" class="pp-hint azl__error">{{ t('game.loadFailed', { msg: error }) }}</p>
      <div class="pp-card azl__table">
        <table>
          <thead>
            <tr>
              <th v-for="c in columns" :key="c.key" :class="{ 'is-num': c.num }">{{ c.label }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in rows" :key="r.key">
              <td v-for="c in columns" :key="c.key" :class="{ 'is-num': c.num, 'pp-num': c.num }">{{ r[c.key] }}</td>
            </tr>
            <tr v-if="!rows.length && !loading">
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
import { computed, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import PButton from '@passport/components/PButton.vue'
import * as azur from '@passport/api/azurpromilia'
import { errorText, fmtNum, fmtTime } from '@passport/utils/format'
import { itemName, kindName, rewardsText } from './names'

type Kind = 'items' | 'currencies' | 'heroes' | 'pets' | 'mails' | 'logins'
type Row = Record<string, string | number> & { key: string }

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
const total = ref(0)
const page = ref(1)
const next = ref('')
const loading = ref(false)
const error = ref('')
const cursors = ref<string[]>([''])

const fetchPage = async (cursor: string): Promise<{ items: Row[]; nextCursor: string; total: number }> => {
  const id = props.serverId
  switch (props.kind) {
    case 'items': {
      const r = await azur.listMyItems(id, cursor, LIMIT)
      return { ...r, items: r.items.map((x) => ({ key: `${x.id}`, name: itemName(x.type, x.id), num: fmtNum(x.num), deadTime: time(x.deadTime) })) }
    }
    case 'currencies': {
      const r = await azur.listMyCurrencies(id)
      return { ...r, nextCursor: '', items: r.items.map((x) => ({ key: `${x.id}`, name: itemName(x.type, x.id), num: fmtNum(x.num), deadTime: '—' })) }
    }
    case 'heroes': {
      const r = await azur.listMyHeroes(id, cursor, LIMIT)
      return {
        ...r,
        items: r.items.map((h) => ({ key: `${h.seq}`, name: kindName('hero', h.id), level: h.level, rank: h.rank, grade: h.grade, favor: h.favorLevel, obtained: time(h.obtainedAt) })),
      }
    }
    case 'pets': {
      const r = await azur.listMyPets(id, cursor, LIMIT)
      return {
        ...r,
        items: r.items.map((p) => ({ key: `${p.guid}`, name: kindName('pet', p.id), petName: p.name || '—', level: p.level, rank: p.rank, star: p.starStage })),
      }
    }
    case 'mails': {
      const r = await azur.listMyMails(id, cursor, LIMIT)
      return {
        ...r,
        items: r.items.map((m) => ({
          key: `${m.guid}`,
          title: m.title,
          sender: m.sender || '—',
          rewards: rewardsText(m.rewards),
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
</script>

<style scoped>
.azl {
  display: flex;
  flex-direction: column;
  gap: 10px;
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
