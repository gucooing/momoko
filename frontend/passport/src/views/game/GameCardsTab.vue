<!-- 游戏空间 · 卡牌：偶像卡 / 支援卡，按稀有度、偶像筛选（筛选已加载的页），点卡在弹窗里看详情。 -->
<template>
  <div class="gc">
    <div class="gc__bar">
      <div class="pp-seg" role="tablist" :aria-label="t('game.tabs.cards')">
        <button
          v-for="k in kinds"
          :key="k.key"
          type="button"
          role="tab"
          :aria-selected="kind === k.key"
          :class="{ 'is-on': kind === k.key }"
          @click="kind = k.key"
        >
          {{ k.label }} <span class="pp-num">{{ fmtNum(k.count) }}</span>
        </button>
      </div>
      <div class="pp-seg" role="group">
        <button
          v-for="r in rarities"
          :key="r.value"
          type="button"
          :class="{ 'is-on': rarity === r.value }"
          :aria-pressed="rarity === r.value"
          @click="rarity = r.value"
        >
          {{ r.label }}
        </button>
      </div>
      <span class="gc__spacer" />
      <span class="pp-hint">{{ t('game.cards.loaded', { n: current.items.length, total: current.total }) }}</span>
    </div>

    <div v-if="kind === 'idol' && charas.length" class="gc__charas pp-scroll-x" role="group" :aria-label="t('game.cards.byChara')">
      <button
        v-for="c in charas"
        :key="c.id"
        type="button"
        class="gc__chara"
        :class="{ 'is-on': chara === c.id }"
        :aria-pressed="chara === c.id"
        :aria-label="c.name"
        :title="c.name"
        @click="chara = chara === c.id ? '' : c.id"
      >
        <GameImage :src="charaArt(ctx.game.value, c.id)" :alt="c.name" />
      </button>
    </div>

    <div class="gc__main">
      <section class="gc__grid-wrap">
        <div v-if="shown.length" class="gc__grid">
          <button
            v-for="c in shown"
            :key="c.key"
            type="button"
            class="gc__card"
            :class="{ 'is-on': selectedKey === c.key }"
            :aria-pressed="selectedKey === c.key"
            @click="select(c.key)"
          >
            <span class="gc__img">
              <GameImage :src="c.img" :alt="c.name" />
              <span class="gc__rar" :class="`gc__rar--${c.rarity}`">{{ rarityLabel(c.rarity) }}</span>
            </span>
            <span class="gc__name">{{ c.name }}</span>
            <span class="gc__tags">
              <span v-for="x in c.tags" :key="x" class="pp-tag">{{ x }}</span>
            </span>
          </button>
        </div>
        <PEmpty
          v-else-if="!current.loading"
          :icon="RectangleStackIcon"
          :title="current.items.length ? t('game.cards.emptyFiltered') : t('game.cards.emptyTitle')"
        />
        <p v-if="current.error" class="pp-hint gc__err">{{ current.error }}</p>
        <div v-if="current.cursor || current.loading" class="gc__more">
          <PButton :loading="current.loading" :disabled="!current.cursor" @click="loadMore">{{ t('game.loadMore') }}</PButton>
        </div>
      </section>

    </div>

    <PDialog v-model="detailOpen" :title="detailTitle" :width="560">
      <CardDetail :idol="selectedIdol" :support="selectedSupport" class="gc__sheet" />
    </PDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { RectangleStackIcon } from '@heroicons/vue/24/outline'
import GameImage from '@/components/game/GameImage.vue'
import PButton from '@passport/components/PButton.vue'
import PDialog from '@passport/components/PDialog.vue'
import PEmpty from '@passport/components/PEmpty.vue'
import { getGameView } from '@passport/api/passport'
import { errorText, fmtNum } from '@passport/utils/format'
import type { ListPage } from '@/games/mgp'
import type { IdolCard, SupportCard } from '@/games/gakumas/types'
import CardDetail from './CardDetail.vue'
import { charaArt, idolCardArt, rarityLabel, supportCardArt, useGameContext } from './context'

type Kind = 'idol' | 'support'

interface PageState<T> {
  items: T[]
  cursor: string
  total: number
  loading: boolean
  loaded: boolean
  error: string
}

const PAGE = 48

const { t } = useI18n()
const ctx = useGameContext()

const kind = ref<Kind>(ctx.hasView('idolCards') ? 'idol' : 'support')
const rarity = ref(0)
const chara = ref('')
const selectedKey = ref('')
const detailOpen = ref(false)

const idol = reactive<PageState<IdolCard>>({ items: [], cursor: '', total: 0, loading: false, loaded: false, error: '' })
const support = reactive<PageState<SupportCard>>({ items: [], cursor: '', total: 0, loading: false, loaded: false, error: '' })
const current = computed(() => (kind.value === 'idol' ? idol : support))

const kinds = computed(() => {
  const out: { key: Kind; label: string; count: number }[] = []
  if (ctx.hasView('idolCards')) out.push({ key: 'idol', label: t('game.cards.idol'), count: ctx.overview.value?.idolCardCount ?? idol.total })
  if (ctx.hasView('supportCards'))
    out.push({ key: 'support', label: t('game.cards.support'), count: ctx.overview.value?.supportCardCount ?? support.total })
  return out
})

const rarities = computed(() => [
  { value: 0, label: t('game.cards.all') },
  { value: 3, label: rarityLabel(3) },
  { value: 2, label: rarityLabel(2) },
  { value: 1, label: rarityLabel(1) },
])

const charas = computed(() => ctx.list('character') ?? [])

async function load(k: Kind, more = false) {
  const st = k === 'idol' ? idol : support
  if (st.loading) return
  st.loading = true
  st.error = ''
  try {
    const page = await getGameView<ListPage<IdolCard | SupportCard>>(ctx.serverId.value, k === 'idol' ? 'idolCards' : 'supportCards', {
      limit: PAGE,
      cursor: more ? st.cursor : undefined,
    })
    const items = page?.items ?? []
    if (k === 'idol') idol.items = more ? [...idol.items, ...(items as IdolCard[])] : (items as IdolCard[])
    else support.items = more ? [...support.items, ...(items as SupportCard[])] : (items as SupportCard[])
    st.cursor = page?.nextCursor || ''
    st.total = page?.total ?? st.items.length
    st.loaded = true
  } catch (e) {
    st.error = errorText(e)
  } finally {
    st.loading = false
  }
}

const loadMore = () => void load(kind.value, true)

watch(
  kind,
  (k) => {
    selectedKey.value = ''
    if (!(k === 'idol' ? idol : support).loaded) void load(k)
  },
  { immediate: true },
)

interface CardView {
  key: string
  img: string
  name: string
  rarity: number
  tags: string[]
}

const idolKey = (c: IdolCard) => `i:${c.idolCardId}`
const supportKey = (c: SupportCard) => `s:${c.supportCardId}`

const shown = computed<CardView[]>(() => {
  const game = ctx.game.value
  if (kind.value === 'idol') {
    return idol.items
      .filter((c) => (!rarity.value || c.rarity === rarity.value) && (!chara.value || c.characterId === chara.value))
      .map((c) => ({
        key: idolKey(c),
        img: idolCardArt(game, c.skinId || c.idolCardId, 'sq'),
        name: ctx.nameOf('character', c.characterId) || t('game.unnamed'),
        rarity: c.rarity,
        tags: [`${t('game.cards.potential')} ${c.potentialRank}`, `${t('game.cards.limit')} ${c.levelLimitRank}`],
      }))
  }
  return support.items
    .filter((c) => !rarity.value || c.rarity === rarity.value)
    .map((c) => ({
      key: supportKey(c),
      img: supportCardArt(game, c.supportCardId, 'sq'),
      name: ctx.nameOf('supportCard', c.supportCardId) || t('game.unnamed'),
      rarity: c.rarity,
      tags: [t('game.cards.levelShort', { n: c.level }), `${t('game.cards.limit')} ${c.levelLimitRank}`],
    }))
})

const selectedIdol = computed(() => idol.items.find((c) => idolKey(c) === selectedKey.value) ?? null)
const selectedSupport = computed(() => support.items.find((c) => supportKey(c) === selectedKey.value) ?? null)
const detailTitle = computed(() => (selectedIdol.value ? t('game.cards.idol') : t('game.cards.support')))

const select = (key: string) => {
  selectedKey.value = key
  detailOpen.value = true
}
</script>

<style scoped>
.gc {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.gc__bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}
.gc__spacer {
  flex: 1;
}
.gc__charas {
  gap: 6px;
  padding: 4px 0;
}
.gc__chara {
  flex: none;
  width: 48px;
  height: 48px;
  padding: 0;
  border-radius: 50%;
  border: 3px solid transparent;
  background: var(--m-s2);
  overflow: hidden;
  cursor: pointer;
}
.gc__chara :deep(img) {
  width: 100%;
  height: 100%;
}
.gc__chara.is-on {
  border-color: var(--m-g);
}
.gc__main {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
  align-items: flex-start;
}
.gc__grid-wrap {
  flex: 1 1 auto;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.gc__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 12px;
}
.gc__card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 8px;
  border-radius: 20px;
  border: 2px solid transparent;
  background: var(--m-s);
  color: var(--m-ink);
  font: inherit;
  text-align: left;
  cursor: pointer;
  min-width: 0;
}
.gc__card:hover {
  border-color: var(--m-gLine);
}
.gc__card.is-on {
  border-color: var(--m-g);
  box-shadow: 0 0 0 4px var(--m-gSoft);
}
.gc__img {
  position: relative;
  display: block;
  border-radius: 14px;
  overflow: hidden;
}
.gc__img :deep(img) {
  width: 100%;
  aspect-ratio: 1;
}
.gc__rar {
  position: absolute;
  left: 6px;
  top: 6px;
  padding: 1px 7px;
  border-radius: 999px;
  background: var(--m-s);
  font-size: 11px;
  font-weight: 800;
}
.gc__rar--3 {
  background: var(--m-a);
  color: var(--m-aOn);
}
.gc__rar--2 {
  background: var(--m-a2Soft);
}
.gc__name {
  padding: 0 4px;
  font-size: 14px;
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gc__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  padding: 0 4px 4px;
}
.gc__more {
  display: flex;
  justify-content: center;
}
.gc__err {
  color: var(--m-bad);
}
.gc__sheet {
  margin: -8px -20px -18px;
}
@media (max-width: 768px) {
  .gc__grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 8px;
  }
  .gc__card {
    padding: 4px;
    border-radius: 16px;
  }
  .gc__tags {
    display: none;
  }
  .gc__name {
    font-size: 12px;
  }
  .gc__bar .pp-seg {
    width: 100%;
  }
}
</style>
