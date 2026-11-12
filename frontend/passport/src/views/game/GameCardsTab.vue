<!-- 游戏空间 · 卡牌：偶像卡 / 支援卡，按稀有度、偶像与关键字筛选（服务端筛选，页码分页），点卡在弹窗里看详情。
     关键字先查前端名称表（当前语言的译名与游戏原文名）换成卡 ID；查不到时原样交给服务端模糊匹配（ID 与游戏原文名）。
     偶像卡详情里的培育记录点开看记忆详情；支援卡与记忆可设为援助，设好后重新读取列表。 -->
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
      <input
        v-model="keyword"
        type="search"
        class="gc__search"
        :placeholder="kind === 'idol' ? t('game.cards.searchIdol') : t('game.cards.searchSupport')"
        :aria-label="t('game.cards.search')"
      />
      <span class="gc__spacer" />
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
          :title="filtered ? t('game.cards.emptyFiltered') : t('game.cards.emptyTitle')"
        />
        <p v-if="current.error" class="pp-hint gc__err">{{ current.error }}</p>
        <PPager
          v-if="current.loaded"
          :page="current.page"
          :total="current.total"
          :page-size="PAGE"
          :disabled="current.loading"
          @update:page="(p) => loadPage(kind, p)"
        />
      </section>

    </div>

    <PDialog v-model="detailOpen" :title="detailTitle" :width="selectedIdol ? 860 : 680">
      <CardDetail :idol="selectedIdol" :support="selectedSupport" class="gc__sheet" @memory="openMemory" @changed="reloadCurrent" />
    </PDialog>
    <MemoryDetailDialog v-model:memory-id="memoryId" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { RectangleStackIcon } from '@heroicons/vue/24/outline'
import GameImage from '@/components/game/GameImage.vue'
import PDialog from '@passport/components/PDialog.vue'
import PEmpty from '@passport/components/PEmpty.vue'
import PPager from '@passport/components/PPager.vue'
import { listMyIdolCards, listMySupportCards } from '@passport/api/gakumas'
import { i18n } from '@passport/locales'
import { fmtNum } from '@passport/utils/format'
import { offsetCursor, usePagedList } from '@passport/utils/paged'
import { searchGameNames } from '@/games'
import type { IdolCard, SupportCard } from '@/games/gakumas/types'
import type { IdolCardCounts } from '@/types/v1/gakumas/types'
import CardDetail from './CardDetail.vue'
import MemoryDetailDialog from './MemoryDetailDialog.vue'
import { charaArt, idolCardArt, rarityLabel, supportCardArt, useGameContext } from './context'

type Kind = 'idol' | 'support'

const PAGE = 20
const DEBOUNCE = 300

const { t } = useI18n()
const ctx = useGameContext()

const kind = ref<Kind>('idol')
const rarity = ref(0)
const chara = ref('')
const selectedKey = ref('')
const detailOpen = ref(false)
const keyword = ref('')
const memoryId = ref('')

const filtered = computed(() => !!rarity.value || !!chara.value || !!keyword.value.trim())

const kinds = computed(() => {
  const out: { key: Kind; label: string; count: number }[] = [
    { key: 'idol', label: t('game.cards.idol'), count: ctx.overview.value?.idolCardCount ?? idol.total },
    { key: 'support', label: t('game.cards.support'), count: ctx.overview.value?.supportCardCount ?? support.total },
  ]
  return out
})

const rarities = computed(() => [
  { value: 0, label: t('game.cards.all') },
  { value: 3, label: rarityLabel(3) },
  { value: 2, label: rarityLabel(2) },
  { value: 1, label: rarityLabel(1) },
])

const charas = computed(() => ctx.list('character') ?? [])

// 关键字 → 卡 ID（名称表里的译名与原文名；皮肤名经目录换成偶像卡，角色名换成角色的全部偶像卡）；
// 查不到时关键字原样交给服务端模糊匹配
interface KeywordFilter {
  q: string
  ids: string[]
}

async function idolFilter(text: string): Promise<KeywordFilter> {
  const game = ctx.game.value
  const lc = String(i18n.global.locale.value)
  const [cards, skins, charas] = await Promise.all([
    searchGameNames(game, lc, 'idolCard', text),
    searchGameNames(game, lc, 'idolCardSkin', text),
    searchGameNames(game, lc, 'character', text),
  ])
  const ids = new Set(cards)
  const pages = await Promise.all([
    skins.size ? ctx.catalog('idolCardSkin', { ids: [...skins], limit: skins.size }) : undefined,
    ...[...charas].map((c) => ctx.catalog('idolCardSkin', { characterId: c, limit: 500 })),
  ])
  for (const p of pages) for (const e of p?.items ?? []) if (e.parentId) ids.add(e.parentId)
  return ids.size ? { q: '', ids: [...ids] } : { q: text, ids: [] }
}

async function supportFilter(text: string): Promise<KeywordFilter> {
  const ids = await searchGameNames(ctx.game.value, String(i18n.global.locale.value), 'supportCard', text)
  return ids.size ? { q: '', ids: [...ids] } : { q: text, ids: [] }
}

// 稀有度、偶像与关键字由服务端筛选，按页读取
const idolPages = usePagedList<IdolCard, IdolCardCounts>(async (page, size) => {
  const text = keyword.value.trim()
  const f = text ? await idolFilter(text) : { q: '', ids: [] }
  return listMyIdolCards({
    serverId: ctx.serverId.value,
    q: f.q,
    characterId: chara.value,
    rarity: rarity.value,
    idolCardIds: f.ids,
    cursor: offsetCursor(page, size),
    limit: size,
  })
}, PAGE)
const supportPages = usePagedList<SupportCard>(async (page, size) => {
  const text = keyword.value.trim()
  const f = text ? await supportFilter(text) : { q: '', ids: [] }
  return listMySupportCards({
    serverId: ctx.serverId.value,
    q: f.q,
    rarity: rarity.value,
    supportCardIds: f.ids,
    cursor: offsetCursor(page, size),
    limit: size,
  })
}, PAGE)
const idol = idolPages.list
const support = supportPages.list
const current = computed(() => (kind.value === 'idol' ? idol : support))

const loadPage = (k: Kind, page: number) => void (k === 'idol' ? idolPages : supportPages).load(page)

watch(
  kind,
  (k) => {
    selectedKey.value = ''
    if (!(k === 'idol' ? idol : support).loaded) loadPage(k, 1)
  },
  { immediate: true },
)
// 筛选变化：当前这一类回到第一页，另一类下次切过去时再读
const reloadAll = () => {
  idol.loaded = false
  support.loaded = false
  loadPage(kind.value, 1)
}
watch([rarity, chara], reloadAll)
let timer: ReturnType<typeof setTimeout> | undefined
watch(keyword, () => {
  clearTimeout(timer)
  timer = setTimeout(reloadAll, DEBOUNCE)
})

// 设为援助之后重新读当前页（详情按 key 取最新的条目）
const reloadCurrent = () => loadPage(kind.value, current.value.page)

// 偶像卡详情里点开一条培育记录：关掉卡的详情，打开记忆详情
const openMemory = (id: string) => {
  detailOpen.value = false
  memoryId.value = id
}

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
    return idol.items.map((c) => ({
        key: idolKey(c),
        img: idolCardArt(game, c.skinId || c.idolCardId, 'sq'),
        name: ctx.nameOf('character', c.characterId) || t('game.unnamed'),
        rarity: c.rarity,
        tags: [`${t('game.cards.potential')} ${c.potentialRank}`, `${t('game.cards.limit')} ${c.levelLimitRank}`],
      }))
  }
  return support.items.map((c) => ({
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
.gc__search {
  flex: 0 1 240px;
  min-width: 0;
  height: 36px;
  padding: 0 14px;
  border: 1px solid var(--m-line);
  border-radius: 999px;
  background: var(--m-s);
  color: var(--m-ink);
  font: inherit;
  font-size: 14px;
}
.gc__search:focus {
  outline: none;
  border-color: var(--m-g);
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
  .gc__bar .pp-seg,
  .gc__search {
    width: 100%;
    flex-basis: 100%;
  }
}
</style>
