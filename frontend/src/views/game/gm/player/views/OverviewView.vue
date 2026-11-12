<!-- 玩家概览：进度（竞赛、培育、偶像卡、支援卡、研修、记忆，点了去对应分页）、最近动态（抽卡、培育、竞赛、登录、GM 记录合并按时间排序）、
     账号信息与礼物箱。数据来自 MGP 玩家摘要与各玩家视图（每类只取最近 5 条），以及 Momoko 的 GM 记录。 -->
<template>
  <div class="ov">
    <div class="ov__main">
      <AppPanel :title="t('gmData.overview.progress')">
        <div class="ov__prog">
          <button
            v-for="cell in progress"
            :key="cell.key"
            type="button"
            class="ov__pg"
            :disabled="!cell.tab"
            @click="cell.tab && page.goto(cell.tab)"
          >
            <span class="ov__pg-t">{{ cell.title }}</span>
            <span class="ov__pg-v m-num">{{ cell.value }}</span>
            <span class="ov__pg-s">{{ cell.sub }}</span>
          </button>
        </div>
      </AppPanel>

      <AppPanel :title="t('gmData.overview.feed')">
        <div v-if="feedLoading && !feed.length" class="ov__hint">{{ t('gmData.common.loading') }}</div>
        <EmptyState v-else-if="!feed.length" icon="HOutline:ClockIcon" :title="t('gmData.overview.noFeed')" />
        <ul v-else class="ov__feed">
          <li v-for="f in feed" :key="f.key" class="ov__feed-row">
            <GameImage v-if="f.image" :src="f.image" class="ov__feed-img" :class="{ 'is-round': f.round }" width="32" height="32" />
            <span v-else class="ov__feed-icon" aria-hidden="true">
              <component :is="menuStore.iconComponents[f.icon]" />
            </span>
            <span class="ov__feed-text">{{ f.text }}</span>
            <span class="ov__feed-tag">{{ f.tag }}</span>
            <span class="ov__feed-at m-num">{{ when(f.time) }}</span>
          </li>
        </ul>
      </AppPanel>
    </div>

    <div class="ov__side">
      <AppPanel :title="t('gmData.overview.account')">
        <dl class="ov__facts">
          <div v-for="f in facts" :key="f.key" class="ov__fact">
            <dt>{{ f.label }}</dt>
            <dd>{{ f.value }}</dd>
          </div>
        </dl>
      </AppPanel>

      <button v-if="o && page.hasTab('giftBox')" type="button" class="ov__gift" @click="page.goto('giftBox')">
        <span class="ov__gift-icon" aria-hidden="true">
          <component :is="menuStore.iconComponents['HOutline:GiftIcon']" />
        </span>
        <span class="ov__gift-text">
          <span class="ov__gift-title">{{ t('gmData.overview.giftTitle', { n: o.gifts.unreceived }) }}</span>
          <span class="ov__hint">{{ t('gmData.overview.giftSub', { expiring: o.gifts.expiring, admin: o.gifts.fromAdmin }) }}</span>
        </span>
        <component :is="menuStore.iconComponents['HOutline:ChevronRightIcon']" class="ov__gift-chev" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getGmPlayerView, listGmExecutions } from '@/api/gameGm'
import type { ListPage, ManifestView } from '@/games/mgp'
import type { GashaHistoryPage, Memory, PvpView, Session } from '@/games/gakumas/types'
import type { GmExecution } from '@/types/v1/game_gm'
import { useGmContext } from '../../context'
import { usePlayerPage } from '../context'
import { catalogImage, gradeLabel, rarityLabel } from '../shared/game'
import { gmExecutionsQuery } from '../shared/executions'
import { toMs, useGmFormat } from '../shared/format'
import { useCatalogLookup } from '../shared/lookup'

defineProps<{ view?: ManifestView }>()

const { t } = useI18n()
const ctx = useGmContext()
const page = usePlayerPage()
const lookup = useCatalogLookup()
const menuStore = useMenuStore()
const { num, when, dateTime, date } = useGmFormat()

const o = computed(() => page.player.value?.overview)

// ---------- 进度 ----------
const pvp = shallowRef<PvpView>()
const rarities = (counts: Record<string, number> | undefined) =>
  ['3', '2', '1']
    .filter((r) => counts?.[r])
    .map((r) => `${rarityLabel(Number(r))} ${counts?.[r]}`)
    .join(' · ')

const progress = computed(() => {
  const ov = o.value
  if (!ov) return []
  const out: { key: string; title: string; value: string; sub: string; tab?: string }[] = []
  const tab = (id: string) => (page.hasTab(id) ? id : undefined)
  if (ctx.hasView('pvp')) {
    const p = pvp.value
    out.push({
      key: 'pvp',
      title: t('gmData.overview.pvp'),
      value: !p ? '…' : p.rank ? t('gmData.overview.rank', { n: p.rank }) : t('gmData.overview.notEntered'),
      sub: p?.entered ? t('gmData.overview.pvpSub', { rate: num(p.rate), grade: p.grade }) : '',
      tab: tab('pvp'),
    })
  }
  const best = ov.bestProduce
  out.push({
    key: 'produce',
    title: t('gmData.overview.produce'),
    value: t('gmData.overview.times', { n: num(ov.produceCount) }),
    sub: best ? t('gmData.overview.best', { grade: gradeLabel(best.grade), name: lookup.name('character', best.characterId) }) : '',
    tab: tab('produceHistory'),
  })
  out.push({
    key: 'idol',
    title: t('gmData.overview.idolCards'),
    value: t('gmData.overview.cards', { n: num(ov.idolCardCount) }),
    sub: rarities(ov.idolCardRarities),
    tab: tab('idolCards'),
  })
  out.push({
    key: 'support',
    title: t('gmData.overview.supportCards'),
    value: t('gmData.overview.cards', { n: num(ov.supportCardCount) }),
    sub: rarities(ov.supportCardRarities),
    tab: tab('supportCards'),
  })
  if (ov.seminar?.length) {
    const total = ov.seminar.reduce((s, g) => s + g.Total, 0)
    const cleared = ov.seminar.reduce((s, g) => s + g.Cleared, 0)
    const full = ov.seminar.filter((g) => g.Total > 0 && g.Cleared >= g.Total).length
    out.push({
      key: 'seminar',
      title: t('gmData.overview.seminar'),
      value: t('gmData.overview.exams', { cleared, total }),
      sub: t('gmData.overview.seminarSub', { n: full, total: ov.seminar.length }),
      tab: tab('seminar'),
    })
  }
  out.push({
    key: 'memories',
    title: t('gmData.overview.memories'),
    value: num(ov.memoryCount),
    sub: t('gmData.overview.items', { n: num(ov.itemCount) }),
    tab: tab('memories'),
  })
  return out
})

// ---------- 账号 ----------
const facts = computed(() => {
  const p = page.player.value
  const ov = o.value
  if (!p) return []
  const run = ov?.produce
  const ban = p.bannedUntil
    ? `${p.bannedUntil >= Number.MAX_SAFE_INTEGER ? t('gmData.players.bannedForever') : t('gmData.players.bannedUntil', { date: date(p.bannedUntil) })}${p.banReason ? ` · ${p.banReason}` : ''}`
    : t('gmData.overview.none')
  return [
    { key: 'created', label: t('gmData.overview.created'), value: dateTime(p.createdAt) },
    { key: 'logins', label: t('gmData.overview.logins'), value: t('gmData.overview.loginCount', { total: num(p.logins?.total), today: num(p.logins?.today) }) },
    { key: 'last', label: t('gmData.overview.lastLogin'), value: p.lastLoginAt ? dateTime(p.lastLoginAt) : t('gmData.players.neverLogin') },
    { key: 'device', label: t('gmData.overview.device'), value: [p.platform, p.device, p.remoteAddr].filter(Boolean).join(' · ') || '—' },
    { key: 'tutorial', label: t('gmData.overview.tutorial'), value: ov ? (ov.tutorialCleared ? t('gmData.overview.done') : t('gmData.overview.notDone')) : '—' },
    {
      key: 'produce',
      label: t('gmData.overview.currentProduce'),
      value: run
        ? t('gmData.overview.produceRun', {
            produce: lookup.name('produce', run.produceId),
            name: lookup.name('character', run.characterId),
            step: run.stepNumber,
          })
        : t('gmData.overview.none'),
    },
    { key: 'fans', label: t('gmData.overview.fans'), value: num(ov?.totalFanCount) },
    { key: 'comment', label: t('gmData.overview.comment'), value: ov?.comment || '—' },
    { key: 'ban', label: t('gmData.overview.ban'), value: ban },
  ]
})

// ---------- 最近动态 ----------
const draws = shallowRef<GashaHistoryPage>()
const produces = shallowRef<ListPage<Memory>>()
const sessions = shallowRef<ListPage<Session>>()
const gmLogs = shallowRef<GmExecution[]>([])
const feedLoading = ref(false)
const FEED_EACH = 5

const loadFeed = async () => {
  const sid = ctx.serverId.value
  const pid = page.pid.value
  if (!sid || !pid || !ctx.manifest.value) return
  feedLoading.value = true
  const q = `limit=${FEED_EACH}`
  await Promise.allSettled([
    ctx.hasView('pvp') ? getGmPlayerView<PvpView>(sid, pid, 'pvp').then((v) => (pvp.value = v)) : undefined,
    ctx.hasView('gashaHistory') ? getGmPlayerView<GashaHistoryPage>(sid, pid, 'gashaHistory', q).then((v) => (draws.value = v)) : undefined,
    ctx.hasView('produceHistory') ? getGmPlayerView<ListPage<Memory>>(sid, pid, 'produceHistory', q).then((v) => (produces.value = v)) : undefined,
    ctx.hasView('sessions') ? getGmPlayerView<ListPage<Session>>(sid, pid, 'sessions', q).then((v) => (sessions.value = v)) : undefined,
    listGmExecutions(gmExecutionsQuery(sid, { pid, pageSize: FEED_EACH })).then(({ data }) => (gmLogs.value = data.items || [])),
  ])
  feedLoading.value = false
}

watch(
  () => [ctx.manifest.value, page.version.value] as const,
  () => void loadFeed(),
  { immediate: true },
)

interface FeedItem {
  key: string
  time: number
  text: string
  tag: string
  image?: string
  round?: boolean
  icon: string
}

const resourceImage = (resourceType: number, resourceId: string) => {
  const rr = lookup.rewardRef(resourceType, resourceId)
  return rr ? catalogImage(ctx.game.value, rr.kind, { id: rr.id }) : ''
}

const feed = computed<FeedItem[]>(() => {
  const out: FeedItem[] = []
  for (const d of draws.value?.items || []) {
    const top = [...(d.cards || [])].sort((a, b) => b.rarity - a.rarity)[0]
    out.push({
      key: `g${d.id}`,
      time: d.createdAt,
      text: d.ssrCount
        ? t('gmData.overview.feedGashaSsr', { gasha: lookup.name('gasha', d.gashaId), n: d.drawCount, ssr: d.ssrCount })
        : t('gmData.overview.feedGasha', { gasha: lookup.name('gasha', d.gashaId), n: d.drawCount }),
      tag: t('gmData.overview.tagGasha'),
      image: top ? resourceImage(top.resourceType, top.resourceId) : '',
      icon: 'HOutline:SparklesIcon',
    })
  }
  for (const m of produces.value?.items || []) {
    out.push({
      key: `p${m.memoryId}`,
      time: m.createdTime,
      text: t('gmData.overview.feedProduce', { name: lookup.name('character', m.characterId), grade: gradeLabel(m.grade), score: num(m.score) }),
      tag: t('gmData.overview.tagProduce'),
      image: catalogImage(ctx.game.value, 'idolCardSkin', { id: m.idolCardId }),
      icon: 'HOutline:SparklesIcon',
    })
  }
  for (const [i, b] of (pvp.value?.battles || []).slice(0, FEED_EACH).entries()) {
    out.push({
      key: `b${i}${b.playTime}`,
      time: b.playTime,
      text: t(b.isWin ? 'gmData.overview.feedWin' : 'gmData.overview.feedLose', {
        rival: b.rivalName || (b.rivalDisplayId ? t('gm.player.noName') : t('gmData.pvp.npc')),
        delta: b.earnedRate > 0 ? `+${b.earnedRate}` : String(b.earnedRate),
      }),
      tag: t('gmData.overview.tagPvp'),
      icon: 'HOutline:TrophyIcon',
    })
  }
  for (const s of sessions.value?.items || []) {
    out.push({
      key: `s${s.id}`,
      time: s.loginAt,
      text: t('gmData.overview.feedLogin', { where: [s.platform, s.device].filter(Boolean).join(' · ') || '—' }),
      tag: t('gmData.overview.tagLogin'),
      icon: 'HOutline:ArrowRightOnRectangleIcon',
    })
  }
  for (const e of gmLogs.value) {
    out.push({
      key: `x${e.id}`,
      time: toMs(e.createTime),
      text: t('gmData.overview.feedGm', { command: e.commandTitle || e.command, actor: e.actorName || '—' }),
      tag: e.actorKind === 'player' ? t('gmData.overview.tagSelf') : t('gmData.overview.tagGm'),
      icon: 'HOutline:WrenchScrewdriverIcon',
    })
  }
  return out.filter((f) => f.time).sort((a, b) => b.time - a.time).slice(0, 8)
})
</script>

<style scoped>
.ov {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  align-items: flex-start;
}
.ov__main {
  flex: 3 1 520px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.ov__side {
  flex: 1 1 300px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.ov__hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}
.ov__prog {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}
.ov__pg {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  min-height: 92px;
  padding: 12px 14px;
  border: 0;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
  min-width: 0;
}
.ov__pg:hover:not(:disabled) {
  background: var(--m-gSoft);
}
.ov__pg:disabled {
  cursor: default;
}
.ov__pg-t {
  font-size: 12px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.ov__pg-v {
  font-size: 20px;
  font-weight: 800;
}
.ov__pg-s {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}
.ov__feed {
  margin: 0;
  padding: 0;
  list-style: none;
}
.ov__feed-row {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 52px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.ov__feed-row:first-child {
  border-top: 0;
}
.ov__feed-img {
  flex: none;
  width: 32px;
  height: 32px;
  border-radius: 8px;
}
.ov__feed-img.is-round {
  border-radius: 50%;
}
.ov__feed-icon {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 10px;
  background: var(--m-gSoft);
  color: var(--m-gInk);
}
.ov__feed-icon svg {
  width: 16px;
  height: 16px;
}
.ov__feed-text {
  flex: 1;
  min-width: 0;
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ov__feed-tag {
  flex: none;
  padding: 2px 10px;
  border-radius: 999px;
  background: var(--m-s2);
  font-size: 12px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.ov__feed-at {
  flex: none;
  width: 92px;
  text-align: right;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.ov__facts {
  margin: 0;
  display: flex;
  flex-direction: column;
}
.ov__fact {
  display: flex;
  gap: 10px;
  padding: 8px 0;
  border-top: 1px solid var(--el-border-color-lighter);
  font-size: 13px;
}
.ov__fact:first-child {
  border-top: 0;
}
.ov__fact dt {
  flex: none;
  width: 92px;
  color: var(--el-text-color-secondary);
}
.ov__fact dd {
  margin: 0;
  min-width: 0;
  font-weight: 600;
  word-break: break-all;
}
.ov__gift {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 18px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  box-shadow: var(--app-shadow-card);
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.ov__gift:hover {
  border-color: var(--m-gLine);
}
.ov__gift-icon {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: 14px;
  background: var(--m-gSoft);
  color: var(--m-gInk);
}
.ov__gift-icon svg {
  width: 22px;
  height: 22px;
}
.ov__gift-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.ov__gift-title {
  font-size: 15px;
  font-weight: 800;
}
.ov__gift-chev {
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
@media (max-width: 760px) {
  .ov__prog {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .ov__feed-tag {
    display: none;
  }
  .ov__feed-at {
    width: auto;
  }
}
</style>
