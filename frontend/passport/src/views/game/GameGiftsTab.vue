<!-- 游戏空间 · 礼物箱（只读）：未领取 / 领取记录，按到期与来源筛选已加载的礼物。领取在游戏里完成。 -->
<template>
  <div class="gg">
    <div class="gg__bar">
      <div class="pp-seg" role="tablist" :aria-label="t('game.tabs.gifts')">
        <button
          v-for="v in views"
          :key="v.key"
          type="button"
          role="tab"
          :aria-selected="view === v.key"
          :class="{ 'is-on': view === v.key }"
          @click="view = v.key"
        >
          {{ v.label }}
        </button>
      </div>
      <div v-if="view === 'unreceived'" class="pp-scroll-x gg__chips">
        <button
          v-for="f in filters"
          :key="f.key"
          type="button"
          class="pp-chip"
          :class="{ 'is-on': filter === f.key }"
          :aria-pressed="filter === f.key"
          @click="filter = f.key"
        >
          {{ f.label }}
          <span v-if="f.count !== undefined" class="pp-chip__n">{{ fmtNum(f.count) }}</span>
        </button>
      </div>
    </div>

    <div class="gg__main">
      <section class="pp-card gg__list">
        <div v-if="shown.length" class="pp-rows">
          <div v-for="g in shown" :key="g.giftId" class="gg__row">
            <span class="gg__img" :class="{ 'gg__img--card': isCard(g) }">
              <GameImage :src="ctx.resourceImage(g.resourceType, g.resourceId)" />
            </span>
            <span class="gg__text">
              <span class="gg__name">
                {{ giftName(g) }}
                <span class="pp-num gg__qty">{{ t('game.gifts.qty', { n: fmtNum(g.quantity) }) }}</span>
              </span>
              <span class="pp-hint gg__msg">{{ giftSource(g) }}</span>
            </span>
            <span class="pp-hint pp-num gg__at">{{ fmtTime(g.createdAt) }}</span>
            <PPill :tone="expiry(g).tone" class="gg__pill">{{ expiry(g).text }}</PPill>
          </div>
        </div>
        <PEmpty
          v-else-if="!state.loading"
          :icon="GiftIcon"
          :title="emptyTitle"
        />
        <p v-if="state.error" class="pp-hint gg__err">{{ state.error }}</p>
        <div v-if="state.cursor || state.loading" class="gg__more">
          <PButton size="sm" :loading="state.loading" :disabled="!state.cursor" @click="load(true)">{{ t('game.loadMore') }}</PButton>
        </div>
      </section>
      <aside class="gg__aside">
        <div class="pp-card gg__info">
          <span class="gg__info-head">
            <InformationCircleIcon aria-hidden="true" />
            {{ t('game.gifts.receiveInGame') }}
          </span>
          <span class="gg__info-text">{{ t('game.gifts.receiveInGameDesc') }}</span>
        </div>
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { GiftIcon, InformationCircleIcon } from '@heroicons/vue/24/outline'
import GameImage from '@/components/game/GameImage.vue'
import PButton from '@passport/components/PButton.vue'
import PEmpty from '@passport/components/PEmpty.vue'
import PPill from '@passport/components/PPill.vue'
import { getGameView } from '@passport/api/passport'
import { errorText, fmtLeft, fmtNum, fmtTime } from '@passport/utils/format'
import type { ListPage } from '@/games/mgp'
import type { Gift } from '@/games/gakumas/types'
import { useGameContext } from './context'

type View = 'unreceived' | 'received'
type Filter = 'all' | 'today' | 'admin' | 'ingame'

const PAGE = 30
const DAY = 86_400_000

const { t } = useI18n()
const ctx = useGameContext()

const view = ref<View>('unreceived')
const filter = ref<Filter>('all')

const state = reactive({ items: [] as Gift[], cursor: '', loading: false, error: '' })

const views = computed(() => [
  { key: 'unreceived' as View, label: t('game.gifts.unreceived') },
  { key: 'received' as View, label: t('game.gifts.received') },
])

const summary = computed(() => ctx.overview.value?.gifts)

const filters = computed(() => {
  const s = summary.value
  return [
    { key: 'all' as Filter, label: t('game.gifts.all'), count: s?.unreceived },
    { key: 'today' as Filter, label: t('game.gifts.today'), count: s?.expiring },
    { key: 'admin' as Filter, label: t('game.gifts.admin'), count: s?.fromAdmin },
    { key: 'ingame' as Filter, label: t('game.gifts.ingame'), count: s ? Math.max(0, s.unreceived - s.fromAdmin) : undefined },
  ]
})

let seq = 0

async function load(more = false) {
  if (state.loading && more) return
  const id = ++seq
  state.loading = true
  state.error = ''
  const forView = view.value
  try {
    const page = await getGameView<ListPage<Gift>>(ctx.serverId.value, 'giftBox', {
      state: forView,
      limit: PAGE,
      cursor: more ? state.cursor : undefined,
    })
    if (id !== seq) return
    state.items = more ? [...state.items, ...(page?.items ?? [])] : (page?.items ?? [])
    state.cursor = page?.nextCursor || ''
  } catch (e) {
    if (id === seq) state.error = errorText(e)
  } finally {
    if (id === seq) state.loading = false
  }
}

watch(
  view,
  () => {
    filter.value = 'all'
    state.items = []
    state.cursor = ''
    void load()
  },
  { immediate: true },
)

const expiresWithinDay = (g: Gift) => g.limitTime > 0 && g.limitTime - Date.now() <= DAY

const shown = computed(() => {
  if (view.value !== 'unreceived') return state.items
  switch (filter.value) {
    case 'today':
      return state.items.filter(expiresWithinDay)
    case 'admin':
      return state.items.filter((g) => g.sourceKind !== 0)
    case 'ingame':
      return state.items.filter((g) => g.sourceKind === 0)
    default:
      return state.items
  }
})

const emptyTitle = computed(() => {
  if (state.items.length) return t('game.gifts.emptyFiltered')
  return view.value === 'received' ? t('game.gifts.emptyReceived') : t('game.gifts.emptyTitle')
})

const isCard = (g: Gift) => {
  const kind = ctx.resourceRef(g.resourceType, g.resourceId).kind
  return kind === 'idolCardSkin' || kind === 'supportCard'
}

const giftName = (g: Gift) => {
  return ctx.resourceName(g.resourceType, g.resourceId) || t('game.unnamed')
}

const giftSource = (g: Gift) => {
  const src = t(`game.gifts.source.${g.sourceKind}`)
  return g.message ? `${g.message} · ${src}` : src
}

const expiry = (g: Gift): { tone: 'ok' | 'warn' | 'bad' | 'off'; text: string } => {
  if (view.value === 'received' || g.receivedAt) {
    return { tone: 'ok', text: t('game.gifts.receivedAt', { time: fmtTime(g.receivedAt) }) }
  }
  if (g.state === 'expired') return { tone: 'off', text: t('game.gifts.expired') }
  if (!g.limitTime) return { tone: 'off', text: t('game.gifts.noExpiry') }
  const left = g.limitTime - Date.now()
  if (left <= DAY) return { tone: 'bad', text: t('game.gifts.expiresToday') }
  if (left <= 7 * DAY) return { tone: 'warn', text: t('game.gifts.expiresIn', { left: fmtLeft(g.limitTime) }) }
  return { tone: 'off', text: t('game.gifts.expiresIn', { left: fmtLeft(g.limitTime) }) }
}
</script>

<style scoped>
.gg {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.gg__bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.gg__chips {
  max-width: 100%;
}
.gg__main {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
  align-items: flex-start;
}
.gg__list {
  flex: 3 1 600px;
  min-width: 0;
  padding: 6px 18px 10px;
}
.gg__row {
  display: grid;
  grid-template-columns: 52px minmax(0, 1fr) 130px 120px;
  gap: 14px;
  align-items: center;
  min-height: 72px;
  padding: 6px 0;
}
.gg__img {
  width: 48px;
  height: 48px;
  display: block;
}
.gg__img :deep(img) {
  width: 48px;
  height: 48px;
  object-fit: contain;
  background: transparent;
}
.gg__img--card :deep(img) {
  object-fit: cover;
  border-radius: 10px;
}
.gg__text {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.gg__name {
  font-size: 15px;
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gg__qty {
  margin-left: 4px;
  color: var(--m-gInk);
}
.gg__msg {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gg__pill {
  justify-self: end;
}
.gg__more {
  display: flex;
  justify-content: center;
  padding: 10px 0 4px;
}
.gg__err {
  color: var(--m-bad);
}
.gg__aside {
  flex: 1 1 300px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.gg__info {
  padding: 18px 20px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.gg__info-head {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 15px;
  font-weight: 800;
}
.gg__info-head svg {
  width: 22px;
  height: 22px;
  color: var(--m-gInk);
}
.gg__info-text {
  font-size: 13px;
  line-height: 1.7;
  color: var(--m-ink2);
}
@media (max-width: 768px) {
  .gg__list {
    padding: 4px 14px 8px;
  }
  .gg__row {
    grid-template-columns: 48px minmax(0, 1fr);
    grid-template-areas:
      'img text'
      'img meta';
    row-gap: 4px;
    padding: 10px 0;
  }
  .gg__img {
    grid-area: img;
    align-self: start;
  }
  .gg__text {
    grid-area: text;
  }
  .gg__at {
    display: none;
  }
  .gg__pill {
    grid-area: meta;
    justify-self: start;
  }
}
</style>
