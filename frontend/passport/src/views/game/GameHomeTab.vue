<!-- 游戏空间 · 概览：名片、钱包、需要你处理、正在进行、最近获得。 -->
<template>
  <div class="gh">
    <div class="gh__top">
      <section class="gh__card">
        <GameImage v-if="cardArt" :src="cardArt" eager class="gh__card-img" />
        <span class="pp-tag gh__card-tag">{{ t('game.home.nameCard') }}</span>
        <div class="gh__plate">
          <span class="gh__bust">
            <GameImage v-if="bustArt" :src="bustArt" />
          </span>
          <span class="gh__plate-text">
            <span class="gh__name">{{ ov?.name || player?.name }}</span>
            <span class="pp-hint pp-num">{{ plateLine }}</span>
            <span class="gh__lv">
              <span class="pp-tag pp-tag--g pp-num">{{ t('game.hero.level', { n: ov?.level ?? 0 }) }}</span>
              <span class="pp-bar gh__bar" :title="expTitle"><span :style="{ width: `${expPct}%` }" /></span>
            </span>
          </span>
        </div>
      </section>
      <section class="pp-card gh__wallet">
        <div v-for="w in wallet" :key="w.key" class="gh__coin">
          <span class="gh__coin-img"><GameImage v-if="w.img" :src="w.img" /></span>
          <span class="gh__coin-text">
            <span class="pp-hint">{{ w.label }}</span>
            <span class="pp-num gh__coin-v">{{ w.value }}</span>
          </span>
        </div>
        <span class="pp-hint gh__wallet-hint">{{ t('game.home.walletHint', { server: ctx.data.value?.server?.name || '' }) }}</span>
      </section>
    </div>

    <section class="gh__sec">
      <h2 class="gh__h2">{{ t('game.home.todo') }}</h2>
      <div v-if="todos.length" class="gh__todos">
        <button v-for="x in todos" :key="x.key" type="button" class="gh__todo" @click="x.go()">
          <span class="gh__todo-icon">
            <GameImage v-if="x.img" :src="x.img" />
            <component :is="x.icon" v-else aria-hidden="true" />
          </span>
          <span class="gh__todo-text">
            <span class="gh__todo-t">{{ x.title }}</span>
            <span class="pp-hint">{{ x.sub }}</span>
          </span>
          <ChevronRightIcon class="gh__chev" aria-hidden="true" />
        </button>
      </div>
      <p v-else class="pp-hint gh__none">{{ t('game.home.todoNone') }}</p>
    </section>

    <section class="gh__sec">
      <div class="gh__sec-head">
        <h2 class="gh__h2">{{ t('game.home.ongoing') }}</h2>
        <span v-if="ctx.timezone.value" class="pp-hint">{{ t('game.home.timezone', { tz: ctx.timezone.value }) }}</span>
      </div>
      <div class="gh__ongoing pp-scroll-x">
        <div v-for="b in banners" :key="b.gashaId" class="gh__banner">
          <GameImage :src="bannerArt(b)" :alt="b.name" class="gh__banner-img" />
          <span class="gh__banner-name">{{ b.name || ctx.nameOf('gasha', b.gashaId) || t('game.unnamed') }}</span>
          <span class="pp-hint pp-num">{{ b.endTime ? t('game.home.endsAt', { time: fmtTime(b.endTime) }) : t('game.home.noEnd') }}</span>
        </div>
        <button v-if="pvpShown" type="button" class="gh__pvp" @click="ctx.setTab('records', { rec: 'pvp' })">
          <span class="gh__pvp-icon"><TrophyIcon aria-hidden="true" /></span>
          <span class="gh__pvp-t">{{ ctx.modeTitle('pvp') }}</span>
          <template v-if="pvp?.entered">
            <span class="pp-num gh__pvp-v">{{ pvpHeadline }}</span>
            <span class="pp-hint pp-num">{{ t('game.home.pvpSub', { rate: fmtNum(pvp.rate), left: pvp.remainingPlays }) }}</span>
          </template>
          <span v-else-if="pvp" class="pp-hint">{{ t('game.home.pvpNotEntered') }}</span>
        </button>
        <p v-if="ongoingLoaded && !banners.length && !pvpShown" class="pp-hint gh__none">{{ t('game.home.ongoingNone') }}</p>
      </div>
    </section>

    <section v-if="ctx.hasView('idolCards')" class="gh__sec">
      <div class="gh__sec-head">
        <h2 class="gh__h2">{{ t('game.home.recent') }}</h2>
        <button type="button" class="gh__link" @click="ctx.setTab('cards')">{{ t('game.home.allCards') }}</button>
      </div>
      <div v-if="recent.length" class="gh__recent">
        <button v-for="c in recent" :key="c.idolCardId" type="button" class="gh__rc" @click="ctx.setTab('cards')">
          <span class="gh__rc-img">
            <GameImage :src="idolCardArt(ctx.game.value, c.skinId || c.idolCardId, 'sq')" />
            <span class="gh__rar" :class="`gh__rar--${c.rarity}`">{{ rarityLabel(c.rarity) }}</span>
          </span>
          <span class="gh__rc-text">
            <span class="gh__rc-name">{{ ctx.nameOf('character', c.characterId) || t('game.unnamed') }}</span>
            <span class="pp-hint">{{ t('game.home.acquired', { time: fmtTime(c.createdTime) }) }}</span>
          </span>
        </button>
      </div>
      <p v-else-if="recentLoaded" class="pp-hint gh__none">{{ t('game.home.recentNone') }}</p>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, type Component } from 'vue'
import { useI18n } from 'vue-i18n'
import { ChevronRightIcon, ClipboardDocumentCheckIcon, GiftIcon, SparklesIcon, TrophyIcon } from '@heroicons/vue/24/outline'
import GameImage from '@/components/game/GameImage.vue'
import { getGameView, listGameRequests, listOngoing, listSelfService } from '@passport/api/passport'
import { fmtNum, fmtTime, toDate } from '@passport/utils/format'
import type { ListPage } from '@/games/mgp'
import type { IdolCard, PvpView } from '@/games/gakumas/types'
import type { PassportSelfService } from '@/types/v1/passport'
import { charaArt, currencyArt, gashaArt, idolCardArt, rarityLabel, spacedId, useGameContext } from './context'

interface OngoingBanner {
  gashaId: string
  name: string
  bannerAssetId: string
  startTime: number
  endTime: number
}

const { t } = useI18n()
const ctx = useGameContext()

const ov = computed(() => ctx.overview.value)
const player = computed(() => ctx.player.value)
const game = computed(() => ctx.game.value)

const cardArt = computed(() => idolCardArt(game.value, player.value?.avatarIdolCardId || '', 'land'))
const bustArt = computed(() => charaArt(game.value, ov.value?.homeCharacterId || '', 'bust'))

const plateLine = computed(() => {
  const parts: string[] = []
  const id = player.value?.displayId
  if (id) parts.push(t('game.home.id', { id: spacedId(id) }))
  const home = ov.value?.homeCharacterId
  if (home) parts.push(t('game.home.partner', { name: ctx.nameOf('character', home) || home }))
  if (ov.value?.loginCount) parts.push(t('game.home.loginDays', { n: ov.value.loginCount }))
  return parts.join(' · ')
})

const expPct = computed(() => {
  const o = ov.value
  if (!o) return 0
  if (!o.nextLevelExp) return 100
  return Math.min(100, Math.round((o.levelExp / o.nextLevelExp) * 100))
})
const expTitle = computed(() => {
  const o = ov.value
  if (!o) return ''
  return o.nextLevelExp ? `${fmtNum(o.levelExp)} / ${fmtNum(o.nextLevelExp)}` : t('game.home.maxLevel')
})

const wallet = computed(() => {
  const o = ov.value
  if (!o) return []
  return [
    { key: 'jewelFree', label: t('game.home.wallet.jewelFree'), value: fmtNum(o.jewelFree), img: currencyArt(game.value, 'jewelFree') },
    {
      key: 'jewelPaid',
      label: t('game.home.wallet.jewelPaid'),
      value: fmtNum(o.jewelPaid + (o.jewelWebStorePaid || 0)),
      img: currencyArt(game.value, 'jewelPaid'),
    },
    { key: 'money', label: t('game.home.wallet.money'), value: fmtNum(o.money), img: currencyArt(game.value, 'money') },
    {
      key: 'ap',
      label: t('game.home.wallet.actionPoint'),
      value: `${fmtNum(o.actionPoint)} / ${fmtNum(o.actionPointMax)}`,
      img: currencyArt(game.value, 'actionPoint'),
    },
  ]
})

// ---------- 需要你处理 ----------
const selfService = ref<PassportSelfService[]>([])
const pendingCount = ref(0)

interface Todo {
  key: string
  title: string
  sub: string
  img?: string
  icon?: Component
  go: () => void
}

const usable = (s: PassportSelfService) => {
  const next = toDate(s.nextAvailable)
  return s.remaining !== 0 && !s.hasPending && !(next && next.getTime() > Date.now())
}

const todos = computed<Todo[]>(() => {
  const out: Todo[] = []
  const gifts = ov.value?.gifts
  if (gifts && gifts.unreceived > 0 && ctx.hasView('giftBox')) {
    out.push({
      key: 'gifts',
      title: t('game.home.giftTitle', { n: gifts.unreceived }),
      sub: gifts.expiring > 0 ? t('game.home.giftExpiring', { n: gifts.expiring }) : t('game.home.giftNoExpiring'),
      icon: GiftIcon,
      go: () => ctx.setTab('gifts'),
    })
  }
  const first = selfService.value.find(usable)
  if (first) {
    out.push({
      key: 'self',
      title: t('game.home.selfTitle', { title: first.title }),
      sub: first.remaining > 0 ? t('game.home.selfLeft', { n: first.remaining }) : t('game.home.selfUnlimited'),
      icon: SparklesIcon,
      go: () => ctx.setTab('self'),
    })
  }
  if (pendingCount.value > 0) {
    out.push({
      key: 'pending',
      title: t('game.home.pendingTitle', { n: pendingCount.value }),
      sub: t('game.home.pendingSub'),
      icon: ClipboardDocumentCheckIcon,
      go: () => ctx.setTab('self'),
    })
  }
  return out
})

// ---------- 正在进行 ----------
const banners = ref<OngoingBanner[]>([])
const ongoingLoaded = ref(false)
const pvp = ref<PvpView | null>(null)
const pvpShown = computed(() => ctx.hasMode('pvp') && ctx.hasView('pvp') && !!pvp.value)

const bannerArt = (b: OngoingBanner) => gashaArt(game.value, b.bannerAssetId || b.gashaId)

const pvpHeadline = computed(() => {
  const p = pvp.value
  if (!p) return ''
  return p.rank ? t('game.home.pvpRank', { grade: p.grade, rank: p.rank }) : t('game.home.pvpGrade', { grade: p.grade })
})

// ---------- 最近获得 ----------
const recent = ref<IdolCard[]>([])
const recentLoaded = ref(false)

onMounted(() => {
  const id = ctx.serverId.value
  void listSelfService(id)
    .then((r) => (selfService.value = r.items ?? []))
    .catch(() => undefined)
  void listGameRequests(id, 1, 20)
    .then((r) => (pendingCount.value = (r.items ?? []).filter((x) => x.status === 'pending').length))
    .catch(() => undefined)
  void listOngoing<OngoingBanner[]>(id)
    .then((list) => (banners.value = list ?? []))
    .catch(() => undefined)
    .finally(() => (ongoingLoaded.value = true))
  if (ctx.hasMode('pvp') && ctx.hasView('pvp')) {
    void getGameView<PvpView>(id, 'pvp')
      .then((v) => (pvp.value = v))
      .catch(() => undefined)
  }
  if (ctx.hasView('idolCards')) {
    void getGameView<ListPage<IdolCard>>(id, 'idolCards', { limit: 4 })
      .then((p) => (recent.value = p?.items ?? []))
      .catch(() => undefined)
      .finally(() => (recentLoaded.value = true))
  }
})
</script>

<style scoped>
.gh {
  display: flex;
  flex-direction: column;
  gap: 24px;
}
.gh__top {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
  align-items: stretch;
}
.gh__card {
  position: relative;
  flex: 3 1 520px;
  min-width: 0;
  min-height: 250px;
  border-radius: 28px;
  overflow: hidden;
  background: var(--m-gSoft);
}
.gh__card-img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-position: 60% 30%;
}
.gh__card-tag {
  position: absolute;
  right: 14px;
  top: 14px;
  background: var(--m-s);
  color: var(--m-ink);
}
.gh__plate {
  position: absolute;
  left: 16px;
  bottom: 16px;
  width: 400px;
  max-width: calc(100% - 32px);
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px;
  border-radius: 22px;
  background: var(--m-s);
  box-sizing: border-box;
}
.gh__bust {
  flex: none;
  width: 72px;
  height: 72px;
  border-radius: 50%;
  overflow: hidden;
  background: var(--m-gSoft);
  box-shadow: 0 0 0 3px var(--m-g);
}
.gh__bust :deep(img) {
  width: 112px;
  height: 112px;
  margin: -4px 0 0 -20px;
  object-fit: cover;
}
.gh__plate-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.gh__name {
  font-size: 21px;
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gh__lv {
  display: flex;
  align-items: center;
  gap: 8px;
}
.gh__bar {
  flex: 1;
}
.gh__wallet {
  flex: 2 1 320px;
  min-width: 0;
  padding: 18px;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  align-content: start;
}
.gh__coin {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px;
  border-radius: 18px;
  background: var(--m-s2);
  min-width: 0;
}
.gh__coin-img {
  flex: none;
  width: 40px;
  height: 40px;
}
.gh__coin-img :deep(img) {
  width: 40px;
  height: 40px;
  object-fit: contain;
  background: transparent;
}
.gh__coin-text {
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.gh__coin-v {
  font-size: 18px;
  font-weight: 800;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.gh__wallet-hint {
  grid-column: 1 / -1;
  padding: 2px 4px;
}
.gh__sec {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.gh__sec-head {
  display: flex;
  align-items: baseline;
  gap: 10px;
}
.gh__sec-head .gh__h2 {
  flex: 1;
}
.gh__h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 800;
}
.gh__link {
  min-height: 40px;
  padding: 0 4px;
  border: 0;
  background: transparent;
  color: var(--m-gInk);
  font: inherit;
  font-size: 14px;
  font-weight: 700;
  cursor: pointer;
}
.gh__none {
  margin: 0;
  padding: 8px 2px;
}
.gh__todos {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
}
.gh__todo {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 72px;
  padding: 14px 16px;
  border-radius: 20px;
  border: 1.5px solid var(--m-line);
  background: var(--m-s);
  color: var(--m-ink);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.gh__todo:hover {
  border-color: var(--m-g);
}
.gh__todo-icon {
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
.gh__todo-icon svg {
  width: 22px;
  height: 22px;
}
.gh__todo-icon :deep(img) {
  width: 44px;
  height: 44px;
  object-fit: contain;
  background: transparent;
}
.gh__todo-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.gh__todo-t {
  font-size: 15px;
  font-weight: 800;
}
.gh__chev {
  width: 18px;
  height: 18px;
  color: var(--m-ink2);
  flex: none;
}
.gh__ongoing {
  gap: 14px;
  align-items: stretch;
}
.gh__banner {
  flex: 0 0 240px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.gh__banner-img {
  width: 240px;
  aspect-ratio: 2;
  border-radius: 16px;
  object-fit: contain;
}
.gh__banner-name {
  font-size: 13px;
  font-weight: 700;
}
.gh__pvp {
  flex: 0 0 240px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 16px;
  border-radius: 24px;
  border: 1px solid var(--m-gLine);
  background: var(--m-gSoft);
  color: var(--m-ink);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.gh__pvp-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 13px;
  background: var(--m-s);
  color: var(--m-gInk);
}
.gh__pvp-icon svg {
  width: 22px;
  height: 22px;
}
.gh__pvp-t {
  font-size: 15px;
  font-weight: 800;
}
.gh__pvp-v {
  font-size: 20px;
  font-weight: 800;
  color: var(--m-gInk);
}
.gh__recent {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
}
.gh__rc {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px;
  border-radius: 20px;
  background: var(--m-s);
  border: 1px solid var(--m-line);
  color: var(--m-ink);
  font: inherit;
  text-align: left;
  cursor: pointer;
  min-width: 0;
}
.gh__rc-img {
  position: relative;
  flex: none;
  width: 64px;
  height: 64px;
  border-radius: 14px;
  overflow: hidden;
}
.gh__rc-img :deep(img) {
  width: 64px;
  height: 64px;
}
.gh__rc-text {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.gh__rc-name {
  font-size: 14px;
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gh__rar {
  position: absolute;
  left: 4px;
  top: 4px;
  padding: 1px 6px;
  border-radius: 999px;
  background: var(--m-s);
  color: var(--m-ink);
  font-size: 10px;
  font-weight: 800;
}
.gh__rar--3 {
  background: var(--m-a);
  color: var(--m-aOn);
}
.gh__rar--2 {
  background: var(--m-a2Soft);
}
@media (max-width: 768px) {
  .gh__card {
    flex-basis: 100%;
  }
  .gh__plate {
    left: 12px;
    right: 12px;
    bottom: 12px;
    width: auto;
    max-width: none;
  }
  .gh__wallet {
    flex-basis: 100%;
    padding: 0;
    background: transparent;
    border: 0;
  }
  .gh__coin {
    background: var(--m-s);
    border: 1px solid var(--m-line);
  }
  .gh__todos {
    grid-template-columns: minmax(0, 1fr);
  }
  .gh__todo {
    min-height: 64px;
  }
  .gh__banner,
  .gh__pvp {
    flex-basis: 220px;
  }
  .gh__banner-img {
    width: 220px;
  }
  .gh__recent {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .gh__rc {
    flex-direction: column;
    align-items: stretch;
  }
  .gh__rc-img,
  .gh__rc-img :deep(img) {
    width: 100%;
    height: auto;
    aspect-ratio: 1;
  }
}
</style>
