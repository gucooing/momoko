<!-- 游戏空间 · 记录：抽卡、竞赛、培育、研修（只显示服务端实际提供的，标题来自 manifest）。 -->
<template>
  <div class="gr">
    <nav class="pp-utabs" role="tablist" :aria-label="t('game.tabs.records')">
      <button
        v-for="x in subs"
        :key="x.key"
        type="button"
        role="tab"
        :aria-selected="rec === x.key"
        :class="{ 'is-on': rec === x.key }"
        @click="setRec(x.key)"
      >
        <component :is="x.icon" class="gr__ico" aria-hidden="true" />
        {{ x.label }}
      </button>
    </nav>

    <!-- 抽卡 -->
    <template v-if="rec === 'gasha'">
      <div v-if="gasha.summary" class="gr__stats">
        <div class="pp-card gr__stat">
          <span class="pp-hint gr__stat-k">{{ t('game.records.totalRequests') }}</span>
          <span class="pp-num gr__stat-v">{{ fmtNum(gasha.summary.requests) }}</span>
          <span class="pp-hint">{{ t('game.records.totalDraws', { n: fmtNum(gasha.summary.draws) }) }}</span>
        </div>
        <div class="pp-card gr__stat">
          <span class="pp-hint gr__stat-k">{{ t('game.records.ssr') }}</span>
          <span class="pp-num gr__stat-v gr__stat-v--g">{{ fmtNum(gasha.summary.ssr) }}</span>
          <span class="pp-hint">{{ t('game.records.pickup', { n: fmtNum(gasha.summary.pickup) }) }}</span>
        </div>
        <div class="pp-card gr__stat">
          <span class="pp-hint gr__stat-k">{{ t('game.records.jewel') }}</span>
          <span class="pp-num gr__stat-v">{{ fmtNum(gasha.summary.jewelFree + gasha.summary.jewelPaid) }}</span>
          <span class="pp-hint">
            {{ t('game.records.jewelSub', { free: fmtNum(gasha.summary.jewelFree), paid: fmtNum(gasha.summary.jewelPaid) }) }}
          </span>
        </div>
      </div>
      <section class="pp-card gr__list">
        <div v-if="gasha.items.length" class="pp-rows">
          <div v-for="d in gasha.items" :key="d.id" class="gr__draw">
            <span class="pp-hint pp-num gr__at">{{ fmtTime(d.createdAt) }}</span>
            <span class="gr__banner">
              <GameImage :src="gashaImage(d.gashaId)" :alt="d.gashaName" class="gr__banner-img" />
              <span class="pp-hint gr__banner-name">{{ d.gashaName || t('game.unnamed') }} · {{ t('game.records.draws', { n: d.drawCount }) }}</span>
            </span>
            <span class="gr__cards">
              <span
                v-for="(c, i) in d.cards"
                :key="i"
                class="gr__tile"
                :class="[`gr__tile--${c.rarity}`, { 'gr__tile--up': c.isPickup }]"
                :title="ctx.resourceName(c.resourceType, c.resourceId)"
              >
                <GameImage :src="ctx.resourceImage(c.resourceType, c.resourceId)" />
                <span class="gr__tile-r">{{ rarityLabel(c.rarity) }}</span>
              </span>
            </span>
            <span class="gr__cost">
              <span class="gr__cost-img"><GameImage :src="ctx.resourceImage(d.costResourceType, d.costResourceId)" /></span>
              <span class="pp-num">{{ fmtNum(d.costQuantity) }}</span>
            </span>
          </div>
        </div>
        <PEmpty v-else-if="!gasha.loading" compact :icon="SparklesIcon" :title="gasha.error || t('game.records.gashaEmpty')" />
        <PPager
          v-if="gasha.loaded"
          :page="gasha.page"
          :total="gasha.total"
          :page-size="PAGE"
          :disabled="gasha.loading"
          @update:page="(p) => void gashaPages.load(p)"
        />
      </section>
    </template>

    <!-- 竞赛 -->
    <template v-else-if="rec === 'pvp'">
      <div class="gr__pvp">
        <div class="pp-card gr__pvp-head">
          <template v-if="pvp?.entered">
            <span class="pp-hint gr__stat-k">{{ ctx.modeTitle('pvp') }} · {{ t('game.records.season') }}</span>
            <span class="pp-num gr__pvp-big">
              {{ t('game.records.grade', { n: pvp.grade }) }}<template v-if="pvp.rank"> · {{ t('game.records.rank', { n: pvp.rank }) }}</template>
            </span>
            <span class="pp-num gr__pvp-line">
              {{ t('game.records.rate', { n: fmtNum(pvp.rate) }) }} ·
              {{ t('game.records.winLose', { win: pvp.winCount, lose: Math.max(0, pvp.playCount - pvp.winCount) }) }}
            </span>
            <span class="pp-hint">{{ t('game.records.playsLeft', { left: pvp.remainingPlays, max: pvp.maxDailyPlays }) }}</span>
          </template>
          <template v-else-if="pvp">
            <span class="pp-hint gr__stat-k">{{ ctx.modeTitle('pvp') }}</span>
            <span class="gr__pvp-line">{{ t('game.records.notEntered') }}</span>
          </template>
          <span v-else class="pp-hint">{{ pvpError || t('common.loading') }}</span>
        </div>
        <section class="pp-card gr__list gr__battles">
          <div v-if="pvp?.battles?.length" class="pp-rows">
            <div v-for="(b, i) in pvp.battles" :key="i" class="gr__battle">
              <span class="pp-hint pp-num">{{ fmtTime(b.playTime) }}</span>
              <PPill :tone="b.isWin ? 'ok' : 'bad'" :dot="false">{{ b.isWin ? t('game.records.win') : t('game.records.lose') }}</PPill>
              <span class="gr__rival">
                {{ b.rivalName || t('game.records.npc') }}
                <span class="pp-hint">· {{ t('game.records.grade', { n: b.grade }) }}</span>
              </span>
              <span class="pp-num gr__delta" :class="b.earnedRate >= 0 ? 'is-up' : 'is-down'">{{ signed(b.earnedRate) }}</span>
            </div>
          </div>
          <PEmpty v-else-if="pvp" compact :icon="TrophyIcon" :title="t('game.records.battlesEmpty')" />
        </section>
      </div>
    </template>

    <!-- 培育：按稀有度与评价筛选（服务端筛选分页，条数来自列表的 summary） -->
    <template v-else-if="rec === 'produce'">
      <div class="gr__filters">
        <div class="pp-seg" role="group" :aria-label="t('game.records.rarity')">
          <button
            v-for="r in rarityOptions"
            :key="r.value"
            type="button"
            :class="{ 'is-on': produceRarity === r.value }"
            :aria-pressed="produceRarity === r.value"
            @click="produceRarity = r.value"
          >
            {{ r.label }} <span v-if="r.count != null" class="pp-num">{{ fmtNum(r.count) }}</span>
          </button>
        </div>
        <div class="pp-seg" role="group" :aria-label="t('game.records.gradeFilter')">
          <button
            v-for="g in gradeOptions"
            :key="g.value"
            type="button"
            :class="{ 'is-on': produceGrade === g.value }"
            :aria-pressed="produceGrade === g.value"
            @click="produceGrade = g.value"
          >
            {{ g.label }} <span v-if="g.count != null" class="pp-num">{{ fmtNum(g.count) }}</span>
          </button>
        </div>
      </div>
      <section class="pp-card gr__list">
        <div v-if="produce.items.length" class="pp-rows">
          <button v-for="m in produce.items" :key="m.memoryId" type="button" class="gr__memory" @click="memoryId = m.memoryId">
            <span class="pp-hint pp-num">{{ fmtTime(m.createdTime) }}</span>
            <span class="gr__chara"><GameImage :src="charaArt(ctx.game.value, m.characterId)" /></span>
            <span class="gr__mem-text">
              <span class="gr__mem-name">{{ memoryTitle(m) }}</span>
              <span class="pp-hint pp-num">{{ t('game.records.score', { n: fmtNum(m.score) }) }} · Vo {{ fmtNum(m.vocal) }} · Da {{ fmtNum(m.dance) }} · Vi {{ fmtNum(m.visual) }}</span>
            </span>
            <span class="pp-tag pp-tag--g pp-num gr__grade">{{ gradeLabel(m.grade) }}</span>
          </button>
        </div>
        <PEmpty
          v-else-if="!produce.loading"
          compact
          :icon="StarIcon"
          :title="produce.error || (produceRarity || produceGrade ? t('game.records.produceNoMatch') : t('game.records.produceEmpty'))"
        />
        <PPager
          v-if="produce.loaded"
          :page="produce.page"
          :total="produce.total"
          :page-size="PAGE"
          :disabled="produce.loading"
          @update:page="(p) => void producePages.load(p)"
        />
      </section>
      <MemoryDetailDialog v-model:memory-id="memoryId" />
    </template>

    <!-- 研修 -->
    <template v-else-if="rec === 'seminar'">
      <div v-if="seminar.length" class="gr__seminars">
        <div v-for="s in seminar" :key="s.groupId" class="pp-card gr__seminar">
          <span class="gr__sem-name">{{ ctx.nameOf('seminar', s.groupId) || t('game.unnamed') }}</span>
          <span class="pp-num gr__sem-v">{{ s.cleared }} / {{ s.total }}</span>
          <span class="pp-bar"><span :style="{ width: `${s.total ? Math.round((s.cleared / s.total) * 100) : 0}%` }" /></span>
        </div>
      </div>
      <PEmpty v-else-if="seminarLoaded" :icon="BookOpenIcon" :title="seminarError || t('game.records.seminarEmpty')" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, type Component } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { BookOpenIcon, SparklesIcon, StarIcon, TrophyIcon } from '@heroicons/vue/24/outline'
import GameImage from '@/components/game/GameImage.vue'
import PEmpty from '@passport/components/PEmpty.vue'
import PPager from '@passport/components/PPager.vue'
import PPill from '@passport/components/PPill.vue'
import { getMyPvp, getMySeminar, listMyGashaDraws, listMyMemories } from '@passport/api/gakumas'
import { errorText, fmtNum, fmtTime } from '@passport/utils/format'
import { offsetCursor, usePagedList } from '@passport/utils/paged'
import type { DrawTotals, GashaDraw, Memory, PlayerPvp, SeminarProgress } from '@/games/gakumas/types'
import { GRADE_GROUPS, RARITY_VALUES, countTotal, gradeGroupCount, gradeGroupGrades } from '@/games/gakumas/display'
import type { MemoryCounts } from '@/types/v1/gakumas/types'
import MemoryDetailDialog from './MemoryDetailDialog.vue'
import { charaArt, gashaArt, gradeLabel, rarityLabel, useGameContext } from './context'

type Rec = 'gasha' | 'pvp' | 'produce' | 'seminar'

const { t } = useI18n()
const ctx = useGameContext()
const route = useRoute()
const router = useRouter()

const subs = computed(() => {
  const out: { key: Rec; label: string; icon: Component }[] = []
  out.push({ key: 'gasha', label: t('game.records.gasha'), icon: SparklesIcon })
  if (ctx.hasMode('pvp')) out.push({ key: 'pvp', label: ctx.modeTitle('pvp'), icon: TrophyIcon })
  if (ctx.hasMode('produce')) out.push({ key: 'produce', label: ctx.modeTitle('produce'), icon: StarIcon })
  if (ctx.hasMode('seminar')) out.push({ key: 'seminar', label: ctx.modeTitle('seminar'), icon: BookOpenIcon })
  return out
})

const rec = computed<Rec | ''>(() => {
  const q = String(route.query.rec || '')
  return subs.value.find((x) => x.key === q)?.key ?? subs.value[0]?.key ?? ''
})

const setRec = (key: Rec) => void router.replace({ query: { ...route.query, tab: 'records', rec: key } })

// 记录列表按页读取（页码分页）
const PAGE = 20

// ---------- 抽卡 ----------
const gashaPages = usePagedList<GashaDraw, DrawTotals>(
  (page, size) =>
    listMyGashaDraws({ serverId: ctx.serverId.value, gashaIds: [], cardIds: [], cursor: offsetCursor(page, size), limit: size }),
  PAGE,
)
const gasha = gashaPages.list

// 卡池横幅按卡池 ID 取素材（通行证上没有卡池目录；名称在抽卡记录的 gashaName 里）
const gashaImage = (gashaId: string) => gashaArt(ctx.game.value, gashaId)

// ---------- 竞赛 ----------
const pvp = ref<PlayerPvp | null>(null)
const pvpError = ref('')
const pvpLoaded = ref(false)

async function loadPvp() {
  pvpLoaded.value = true
  try {
    pvp.value = await getMyPvp(ctx.serverId.value)
  } catch (e) {
    pvpError.value = errorText(e)
  }
}

const signed = (n: number) => (n > 0 ? `+${fmtNum(n)}` : n < 0 ? `−${fmtNum(-n)}` : '0')

// ---------- 培育 ----------
const memoryId = ref('')
const produceRarity = ref(0)
const produceGrade = ref('')
const producePages = usePagedList<Memory, MemoryCounts>(
  (page, size) =>
    listMyMemories({
      serverId: ctx.serverId.value,
      produced: true,
      q: '',
      rarity: produceRarity.value,
      grades: gradeGroupGrades(produceGrade.value),
      idolCardIds: [],
      cursor: offsetCursor(page, size),
      limit: size,
    }),
  PAGE,
)
const produce = producePages.list

watch([produceRarity, produceGrade], () => void producePages.reload())

// 筛选项：条数为 0 且未选中的不显示
const rarityOptions = computed(() => {
  const counts = produce.summary?.rarities
  return [
    { value: 0, label: t('game.cards.all'), count: counts ? countTotal(counts) : null },
    ...RARITY_VALUES.filter((r) => counts?.[r] || produceRarity.value === r).map((r) => ({
      value: r,
      label: rarityLabel(r),
      count: counts?.[r] ?? 0,
    })),
  ]
})
const gradeOptions = computed(() => {
  const counts = produce.summary?.grades
  return [
    { value: '', label: t('game.records.allGrades'), count: counts ? countTotal(counts) : null },
    ...GRADE_GROUPS.filter((g) => gradeGroupCount(counts, g) || produceGrade.value === g.key).map((g) => ({
      value: g.key,
      label: g.label,
      count: gradeGroupCount(counts, g),
    })),
  ]
})

const memoryTitle = (m: Memory) => {
  const chara = ctx.nameOf('character', m.characterId) || t('game.unnamed')
  const prod = m.produceId ? ctx.nameOf('produce', m.produceId) : ''
  return prod ? `${chara} · ${prod}` : chara
}

// ---------- 研修 ----------
const seminar = ref<SeminarProgress[]>([])
const seminarLoaded = ref(false)
const seminarError = ref('')

async function loadSeminar() {
  try {
    seminar.value = (await getMySeminar(ctx.serverId.value)).groups
  } catch (e) {
    seminarError.value = errorText(e)
  } finally {
    seminarLoaded.value = true
  }
}

watch(
  rec,
  (r) => {
    if (r === 'gasha' && !gasha.loaded) void gashaPages.reload()
    if (r === 'pvp' && !pvpLoaded.value) void loadPvp()
    if (r === 'produce' && !produce.loaded) void producePages.reload()
    if (r === 'seminar' && !seminarLoaded.value) void loadSeminar()
  },
  { immediate: true },
)
</script>

<style scoped>
.gr {
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.gr__ico {
  width: 18px;
  height: 18px;
}
.gr__stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
}
.gr__stat {
  padding: 16px 18px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.gr__stat-k {
  font-weight: 700;
}
.gr__stat-v {
  font-size: 28px;
  font-weight: 800;
}
.gr__stat-v--g {
  color: var(--m-gInk);
}
.gr__list {
  padding: 4px 18px 8px;
  min-width: 0;
}
.gr__draw {
  display: grid;
  grid-template-columns: 110px 160px minmax(0, 1fr) 90px;
  gap: 14px;
  align-items: center;
  padding: 12px 0;
}
.gr__banner {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}
.gr__banner-img {
  width: 150px;
  aspect-ratio: 2;
  border-radius: 10px;
  object-fit: contain;
}
.gr__banner-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gr__cards {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  min-width: 0;
}
.gr__tile {
  position: relative;
  width: 46px;
  height: 46px;
  border-radius: 10px;
  overflow: hidden;
  background: var(--m-s2);
}
.gr__tile :deep(img) {
  width: 46px;
  height: 46px;
}
.gr__tile--3 {
  box-shadow: 0 0 0 2px var(--m-a);
}
.gr__tile--up::after {
  content: '';
  position: absolute;
  right: 3px;
  top: 3px;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--m-g);
  border: 1.5px solid var(--m-s);
}
.gr__tile-r {
  position: absolute;
  left: 2px;
  bottom: 2px;
  padding: 0 4px;
  border-radius: 999px;
  background: var(--m-s);
  font-size: 9px;
  font-weight: 800;
}
.gr__tile--3 .gr__tile-r {
  background: var(--m-a);
  color: var(--m-aOn);
}
.gr__cost {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
  font-weight: 800;
}
.gr__cost-img,
.gr__cost-img :deep(img) {
  width: 22px;
  height: 22px;
  object-fit: contain;
  background: transparent;
}
.gr__filters {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-bottom: 12px;
}
.gr__pvp {
  display: flex;
  flex-wrap: wrap;
  gap: 18px;
  align-items: flex-start;
}
.gr__pvp-head {
  flex: 1 1 300px;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  background: var(--m-gSoft);
  border-color: var(--m-gLine);
}
.gr__pvp-big {
  font-size: 30px;
  font-weight: 800;
  color: var(--m-gInk);
}
.gr__pvp-line {
  font-size: 15px;
  font-weight: 700;
}
.gr__battles {
  flex: 2 1 520px;
}
.gr__battle {
  display: grid;
  grid-template-columns: 110px 56px minmax(0, 1fr) 80px;
  gap: 12px;
  align-items: center;
  min-height: 60px;
}
.gr__rival {
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gr__delta {
  text-align: right;
  font-weight: 800;
}
.gr__delta.is-up {
  color: var(--m-ok);
}
.gr__delta.is-down {
  color: var(--m-bad);
}
.gr__memory {
  display: grid;
  grid-template-columns: 110px 48px minmax(0, 1fr) auto;
  gap: 12px;
  align-items: center;
  width: 100%;
  min-height: 64px;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.gr__memory:hover .gr__mem-name {
  color: var(--m-gInk);
}
.gr__chara,
.gr__chara :deep(img) {
  width: 40px;
  height: 40px;
  border-radius: 50%;
}
.gr__mem-text {
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.gr__mem-name {
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gr__grade {
  height: 26px;
  font-size: 14px;
}
.gr__seminars {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 14px;
}
.gr__seminar {
  padding: 18px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.gr__sem-name {
  font-size: 16px;
  font-weight: 800;
}
.gr__sem-v {
  font-size: 26px;
  font-weight: 800;
  color: var(--m-gInk);
}
@media (max-width: 768px) {
  .gr__stats {
    grid-template-columns: minmax(0, 1fr);
  }
  .gr__list {
    padding: 4px 14px 8px;
  }
  .gr__draw {
    grid-template-columns: minmax(0, 1fr) auto;
    grid-template-areas:
      'banner cost'
      'cards cards'
      'at at';
    gap: 8px;
  }
  .gr__banner {
    grid-area: banner;
  }
  .gr__banner-img {
    width: 132px;
  }
  .gr__cards {
    grid-area: cards;
  }
  .gr__cost {
    grid-area: cost;
  }
  .gr__at {
    grid-area: at;
  }
  .gr__tile,
  .gr__tile :deep(img) {
    width: 40px;
    height: 40px;
  }
  .gr__battle {
    grid-template-columns: 52px minmax(0, 1fr) auto;
    padding: 8px 0;
  }
  .gr__battle > :first-child {
    grid-column: 1 / -1;
    margin-bottom: -6px;
  }
  .gr__memory {
    grid-template-columns: 44px minmax(0, 1fr) auto;
    padding: 8px 0;
  }
  .gr__memory > :first-child {
    grid-column: 1 / -1;
    margin-bottom: -6px;
  }
}
</style>
