<!-- 游戏空间：只看自己在这台服务器上绑定的游戏账号；用游戏皮肤（data-game + game-skin）。
     ?tab=home|cards|gifts|records|self 切换页内标签，不换路由；每个标签第一次显示时才加载数据。 -->
<template>
  <div class="gs game-skin" :data-game="game || undefined">
    <!-- 手机：自带头部 -->
    <header v-if="ui.isMobile" class="gs__mhead">
      <RouterLink to="/games" class="gs__ib" :aria-label="t('game.hero.back')">
        <ChevronLeftIcon aria-hidden="true" />
      </RouterLink>
      <GameMark v-if="server" :game="server.game" :name="server.gameName" :src="server.appIcon" :size="34" />
      <span class="gs__mtitle">
        <span class="gs__mname">{{ server?.name || '…' }}</span>
        <span class="pp-hint gs__mgame">{{ server?.gameName }}</span>
      </span>
      <PButton v-if="server" size="sm" variant="soft" :icon="ChevronUpDownIcon" @click="switchOpen = true">
        {{ t('game.hero.switchShort') }}
      </PButton>
      <button type="button" class="gs__ib" :aria-label="t('game.hero.toggleDark')" @click="ui.toggleDark()">
        <component :is="ui.dark ? SunIcon : MoonIcon" aria-hidden="true" />
      </button>
    </header>

    <div class="gs__wrap">
      <!-- 电脑：横幅 -->
      <section v-if="!ui.isMobile && server" class="gs__hero">
        <ServerBackground v-if="server.appBackground" :src="server.appBackground" eager class="gs__hero-img" />
        <img v-else-if="meishiImage" :src="meishiImage" alt="" class="gs__hero-img gs__hero-img--meishi" />
        <GameImage v-else-if="heroArt" :src="heroArt" eager class="gs__hero-img" />
        <div class="gs__hero-row">
          <div class="gs__hero-card">
            <GameMark :game="server.game" :name="server.gameName" :src="server.appIcon" :size="44" />
            <span class="gs__hero-text">
              <span class="pp-hint">
                <RouterLink to="/games" class="gs__crumb">{{ t('game.hero.myGames') }}</RouterLink>
                › {{ server.gameName || server.game }}
              </span>
              <span class="gs__hero-name">{{ server.name }}</span>
            </span>
            <button type="button" class="gs__switch" :aria-label="t('game.hero.switchTitle')" @click="switchOpen = true">
              {{ t('game.hero.switch') }}
              <ChevronUpDownIcon aria-hidden="true" />
            </button>
          </div>
          <span class="gs__spacer" />
          <span v-if="link" class="gs__me">
            <span class="gs__me-av">
              <GameImage v-if="avatarArt" :src="avatarArt" />
              <span v-else>{{ (link.nickname || '?').slice(0, 1) }}</span>
            </span>
            {{ link.nickname }} · {{ t('game.hero.level', { n: level }) }}
          </span>
        </div>
      </section>

      <!-- 页内标签（手机上横向滚动、用短名） -->
      <nav v-if="ready" class="gs__tabs pp-scroll-x" :aria-label="t('nav.games')">
        <button
          v-for="x in tabs"
          :key="x.key"
          type="button"
          class="gs__tab"
          :class="{ 'is-on': tab === x.key }"
          :aria-current="tab === x.key ? 'page' : undefined"
          @click="setTab(x.key)"
        >
          <component :is="x.icon" aria-hidden="true" />
          {{ ui.isMobile ? x.short : x.label }}
          <span v-if="x.count" class="pp-num gs__tab-n">{{ fmtNum(x.count) }}</span>
        </button>
      </nav>

      <div v-if="loading && !data" class="gs__state pp-hint">{{ t('game.loading') }}</div>
      <PEmpty
        v-else-if="loadError"
        :icon="LinkSlashIcon"
        :title="notLinked ? t('game.notLinked.title') : errorText(loadError)"
        :description="notLinked ? t('game.notLinked.desc') : ''"
      >
        <PButton v-if="notLinked" variant="primary" to="/bind">{{ t('game.notLinked.bind') }}</PButton>
        <PButton v-else :icon="ArrowPathIcon" @click="load">{{ t('game.retry') }}</PButton>
      </PEmpty>
      <PEmpty
        v-else-if="data && !player && !azPlayer"
        :icon="SignalSlashIcon"
        :title="t('game.playerError.title')"
        :description="t('game.playerError.desc', { code: playerError || data.playerError || '—' })"
      >
        <PButton :icon="ArrowPathIcon" :loading="loading" @click="load">{{ t('game.retry') }}</PButton>
      </PEmpty>
      <div v-else-if="ready" :key="serverId" class="gs__body">
        <KeepAlive>
          <component :is="tabComponent" :key="tab" />
        </KeepAlive>
      </div>
      <AzurSpace v-else-if="data && azPlayer" :key="serverId" :server-id="serverId" :server-name="server?.name || ''" :detail="azPlayer" />
    </div>

    <PDialog v-model="switchOpen" :title="t('game.hero.switchTitle')" :width="440">
      <div v-if="others.length" class="gs__others">
        <RouterLink
          v-for="o in others"
          :key="o.id"
          :to="`/games/${encodeURIComponent(o.server?.id || '')}`"
          class="gs__other"
          @click="switchOpen = false"
        >
          <GameMark :game="o.game" :name="o.server?.gameName" :src="o.server?.appIcon" :size="32" />
          <span class="gs__other-text">
            <span class="gs__other-name">{{ o.server?.name || t('game.unnamed') }}</span>
            <span class="pp-hint">{{ o.nickname || t('game.noName') }} · {{ t('game.hero.level', { n: o.level }) }}</span>
          </span>
          <ChevronRightIcon class="gs__chev" aria-hidden="true" />
        </RouterLink>
      </div>
      <p v-else class="pp-hint">{{ t('game.hero.noOthers') }}</p>
      <template #footer>
        <PButton variant="soft" :icon="PlusIcon" to="/bind" @click="switchOpen = false">{{ t('game.hero.bindMore') }}</PButton>
      </template>
    </PDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, type Component } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  ArrowPathIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ChevronUpDownIcon,
  ClockIcon,
  GiftIcon,
  LinkSlashIcon,
  MoonIcon,
  PlusIcon,
  PresentationChartLineIcon,
  RectangleStackIcon,
  SignalSlashIcon,
  SparklesIcon,
  SunIcon,
} from '@heroicons/vue/24/outline'
import GameImage from '@/components/game/GameImage.vue'
import { gameAsset, gameCover, gameLinkAvatar } from '@/games'
import ServerBackground from '@/components/game/ServerBackground.vue'
import GameMark from '@passport/components/GameMark.vue'
import PButton from '@passport/components/PButton.vue'
import PDialog from '@passport/components/PDialog.vue'
import PEmpty from '@passport/components/PEmpty.vue'
import { getGame } from '@passport/api/passport'
import * as gk from '@passport/api/gakumas'
import * as azur from '@passport/api/azurpromilia'
import { PassportError } from '@passport/api/request'
import { useUiStore } from '@passport/stores/ui'
import { errorText, fmtNum } from '@passport/utils/format'
import type { PassportGameResponse } from '@/types/v1/passport'
import type { PlayerDetail } from '@/games/gakumas/types'
import type { GakumasMode } from '@/games/gakumas/modes'
import { imageDataUrl } from '@/games/gakumas/display'
import type { PlayerGame } from '@/types/v1/gakumas/passport'
import type { PlayerDetail as MyPlayer } from '@/types/v1/gakumas/types'
import type { PlayerDetail as AzurPlayer } from '@/types/v1/azurpromilia/types'
import AzurSpace from './azurpromilia/AzurSpace.vue'
import { createCatalogs, idolCardArt, modeTitleOf, provideGameContext, type GameTab } from './context'
import GameHomeTab from './GameHomeTab.vue'
import GameCardsTab from './GameCardsTab.vue'
import GameGiftsTab from './GameGiftsTab.vue'
import GameRecordsTab from './GameRecordsTab.vue'
import GameSelfTab from './GameSelfTab.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const ui = useUiStore()

const serverId = computed(() => String(route.params.serverId || ''))
const data = ref<PassportGameResponse | null>(null)
// 游戏模块的数据：玩法与我的玩家条目
const playerGame = ref<PlayerGame | null>(null)
const me = ref<MyPlayer | null>(null)
// 蓝色星原：我的玩家条目与概览
const azPlayer = ref<AzurPlayer | null>(null)
const playerError = ref('')
const loading = ref(false)
const loadError = ref<unknown>(null)
const switchOpen = ref(false)

const server = computed(() => data.value?.server)
const link = computed(() => data.value?.link)
const others = computed(() => data.value?.others ?? [])
const game = computed(() => server.value?.game || '')
const modes = computed(() => playerGame.value?.modes)
const player = computed<PlayerDetail | null>(() =>
  me.value?.player ? { ...me.value.player, overview: me.value.overview, logins: me.value.logins } : null,
)
const overview = computed(() => player.value?.overview ?? null)
const timezone = computed(() => data.value?.timezone || '')
const level = computed(
  () => overview.value?.level ?? player.value?.level ?? azPlayer.value?.player?.level ?? link.value?.level ?? 0,
)
const notLinked = computed(() => loadError.value instanceof PassportError && loadError.value.code === 'PASSPORT_LINK_NOT_FOUND')
const ready = computed(() => !!data.value && !!player.value && !loadError.value)

const avatarCard = computed(() => player.value?.avatarIdolCardId || link.value?.avatarAsset || '')
// 蓝色星原：头图用服务器封面（没设时用游戏主视觉），头像用名片头像（玩家数据里的头像部件与性别）
const azurAvatar = computed(() => {
  const p = azPlayer.value?.player
  return p?.sex ? `${p.head}:${p.sex}` : link.value?.avatarAsset || ''
})
const heroArt = computed(() => {
  if (game.value !== 'azurpromilia') return idolCardArt(game.value, avatarCard.value, 'land')
  const s = server.value
  return s?.cover ? gameAsset(game.value, s.cover) : gameCover(game.value, s?.id || '')
})

// 头图优先用服务器的应用背景（接入设置里上传，与管理端、启动器同步）；没有时和概览名片一样用资料名片
// （游戏里保存名片时上传的图片）；默认名片没有图片，这时用头像偶像卡的卡面
const meishiImage = ref('')
const meishiPath = computed(() => overview.value?.meishiImagePath || '')
watch(
  () => [serverId.value, meishiPath.value] as const,
  async ([sid, path]) => {
    meishiImage.value = ''
    if (!sid || !path) return
    try {
      const img = await gk.getMyMeishiImage(sid)
      if (sid === serverId.value && path === meishiPath.value) meishiImage.value = imageDataUrl(img)
    } catch {
      // 名片图片读不到时保持卡面
    }
  },
  { immediate: true },
)
const avatarArt = computed(() =>
  game.value === 'azurpromilia' ? gameLinkAvatar(game.value, azurAvatar.value) : idolCardArt(game.value, avatarCard.value, 'sq'),
)

const hasMode = (id: GakumasMode) => !!modes.value?.[id]

interface TabItem {
  key: GameTab
  label: string
  short: string
  icon: Component
  count?: number
}

const tabs = computed<TabItem[]>(() => {
  const out: TabItem[] = [{ key: 'home', label: t('game.tabs.home'), short: t('game.tabs.home'), icon: PresentationChartLineIcon }]
  const n = (overview.value?.idolCardCount ?? 0) + (overview.value?.supportCardCount ?? 0)
  out.push({ key: 'cards', label: t('game.tabs.cards'), short: t('game.tabs.cards'), icon: RectangleStackIcon, count: n })
  out.push({
    key: 'gifts',
    label: t('game.tabs.gifts'),
    short: t('game.tabs.gifts'),
    icon: GiftIcon,
    count: overview.value?.gifts?.unreceived ?? 0,
  })
  out.push({ key: 'records', label: t('game.tabs.records'), short: t('game.tabs.records'), icon: ClockIcon })
  out.push({ key: 'self', label: t('game.tabs.self'), short: t('game.tabs.selfShort'), icon: SparklesIcon })
  return out
})

const tab = computed<GameTab>(() => {
  const q = String(route.query.tab || 'home')
  return (tabs.value.find((x) => x.key === q)?.key ?? 'home') as GameTab
})

const components: Record<GameTab, Component> = {
  home: GameHomeTab,
  cards: GameCardsTab,
  gifts: GameGiftsTab,
  records: GameRecordsTab,
  self: GameSelfTab,
}
const tabComponent = computed(() => components[tab.value])

const setTab = (next: GameTab, extra: Record<string, string> = {}) => {
  const query: Record<string, string> = { tab: next, ...extra }
  void router.replace({ query })
  if (ui.isMobile) window.scrollTo({ top: 0 })
}

async function load() {
  if (!serverId.value) return
  loading.value = true
  loadError.value = null
  try {
    const id = serverId.value
    const space = await getGame(id)
    let game: PlayerGame | null = null
    let mine: MyPlayer | null = null
    let azMine: AzurPlayer | null = null
    let mineError = ''
    if (space.server?.game === 'gakumas') {
      const [g, p] = await Promise.allSettled([gk.getGame(id), gk.getMyPlayer(id)])
      if (g.status === 'fulfilled') game = g.value
      if (p.status === 'fulfilled') mine = p.value
      else mineError = p.reason instanceof PassportError ? p.reason.code : String(p.reason)
    } else if (space.server?.game === 'azurpromilia') {
      try {
        azMine = await azur.getMyPlayer(id)
      } catch (e) {
        mineError = e instanceof PassportError ? e.code : String(e)
      }
    }
    if (serverId.value !== id) return
    data.value = space
    playerGame.value = game
    me.value = mine
    azPlayer.value = azMine?.player ? azMine : null
    playerError.value = mineError
  } catch (e) {
    loadError.value = e
    data.value = null
    playerGame.value = null
    me.value = null
    azPlayer.value = null
  } finally {
    loading.value = false
  }
}

watch(serverId, () => void load(), { immediate: true })

const catalogs = createCatalogs(serverId, game)

provideGameContext({
  serverId,
  game,
  data,
  modes,
  player,
  overview,
  meishiImage,
  timezone,
  hasMode,
  modeTitle: modeTitleOf,
  ...catalogs,
  setTab,
  reload: load,
})
</script>

<style scoped>
.gs {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}
.gs__wrap {
  flex: 1;
  width: 100%;
  max-width: 1240px;
  margin: 0 auto;
  padding: 16px 20px 42px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  box-sizing: border-box;
}
.gs__hero {
  position: relative;
  min-height: 200px;
  border-radius: 20px;
  overflow: hidden;
  background: var(--m-gSoft);
}
.gs__hero-img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: 50% 28%;
}
.gs__hero-img--meishi {
  object-position: 50% 50%;
}
.gs__hero-row {
  position: relative;
  min-height: 150px;
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: 14px;
  padding: 16px 16px;
  box-sizing: border-box;
}
.gs__hero-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 14px 12px 12px;
  border-radius: 16px;
  background: var(--m-s);
}
.gs__hero-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.gs__crumb {
  color: var(--m-ink2);
  text-decoration: none;
}
.gs__crumb:hover {
  color: var(--m-ink);
}
.gs__hero-name {
  font-size: 18px;
  font-weight: 600;
}
.gs__switch {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 32px;
  margin-left: 8px;
  padding: 0 12px;
  border-radius: 999px;
  border: 1.5px solid var(--m-gLine);
  background: var(--m-gSoft);
  color: var(--m-gInk);
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}
.gs__switch svg {
  width: 16px;
  height: 16px;
}
.gs__spacer {
  flex: 1;
}
.gs__me {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  padding: 0 14px 0 4px;
  border-radius: 999px;
  background: var(--m-s);
  font-size: 13px;
  font-weight: 600;
}
.gs__me-av {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  overflow: hidden;
  background: var(--m-gSoft);
  color: var(--m-gInk);
  font-weight: 600;
}
.gs__me-av :deep(img) {
  width: 100%;
  height: 100%;
}
.gs__tabs {
  align-self: flex-start;
  max-width: 100%;
  margin-top: -8px;
  padding: 4px;
  border-radius: 999px;
  background: var(--m-s);
  border: 1px solid var(--m-line);
  box-sizing: border-box;
}
.gs__tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex: none;
  min-height: 30px;
  padding: 0 12px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--m-ink2);
  font: inherit;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
}
.gs__tab svg {
  width: 16px;
  height: 16px;
}
.gs__tab:hover {
  background: var(--m-s2);
  color: var(--m-ink);
}
.gs__tab.is-on {
  background: var(--m-g);
  color: var(--m-gOn);
}
.gs__tab-n {
  font-size: 12px;
  opacity: 0.8;
}
.gs__state {
  padding: 36px 0;
  text-align: center;
}
.gs__body {
  min-width: 0;
}
.gs__mhead {
  position: sticky;
  top: 0;
  z-index: 50;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 10px;
  background: var(--m-bg);
  border-bottom: 1px solid var(--m-line);
}
.gs__mtitle {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.gs__mname {
  font-size: 14px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gs__mgame {
  font-size: 11px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gs__ib {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 34px;
  height: 34px;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--m-ink2);
  cursor: pointer;
}
.gs__ib svg {
  width: 22px;
  height: 22px;
}
.gs__others {
  display: flex;
  flex-direction: column;
}
.gs__other {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 60px;
  padding: 6px 8px;
  border-radius: 14px;
  color: var(--m-ink);
  text-decoration: none;
}
.gs__other:hover {
  background: var(--m-s2);
}
.gs__other-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.gs__other-name {
  font-weight: 600;
}
.gs__chev {
  width: 18px;
  height: 18px;
  color: var(--m-ink2);
}
@media (max-width: 768px) {
  .gs__wrap {
    padding: 12px 12px calc(24px + env(safe-area-inset-bottom));
    gap: 14px;
  }
  .gs__tabs {
    align-self: stretch;
    margin-top: 0;
  }
}
</style>
