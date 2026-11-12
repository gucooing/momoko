<!-- 模拟抽卡：按草稿里的抽选表在浏览器里抽 N 张（与服务端相同的展开规则），对比实际出现率与设定值，
     以及抽到 UP 平均要多少抽。不经过服务端、不影响任何玩家。 -->
<template>
  <AppPanel :title="t('gmConfig.sim.title')" title-icon="HOutline:CubeIcon" class="gs">
    <template #actions>
      <UButton class="gmc-btn" size="sm" color="neutral" variant="soft" icon="i-lucide-dices" :disabled="!prepared" @click="run">{{ t('gmConfig.sim.run') }}</UButton>
    </template>
    <div class="gs__bar">
      <div v-if="lotteries.length > 1" class="gmc-seg" role="radiogroup" :aria-label="t('gmConfig.sim.lottery')">
        <button
          v-for="l in lotteries"
          :key="l.id"
          type="button"
          role="radio"
          class="gmc-seg__btn"
          :class="{ 'is-on': l.id === lotteryId }"
          :aria-checked="l.id === lotteryId"
          @click="lotteryId = l.id"
        >
          {{ l.label }}
        </button>
      </div>
      <div class="gmc-seg" role="radiogroup" :aria-label="t('gmConfig.sim.samples')">
        <button
          v-for="n in SIZES"
          :key="n"
          type="button"
          role="radio"
          class="gmc-seg__btn m-num"
          :class="{ 'is-on': n === size }"
          :aria-checked="n === size"
          @click="size = n"
        >
          {{ n.toLocaleString() }}
        </button>
      </div>
    </div>

    <p v-if="problems.length" class="gmc-note">
      <UIcon name="i-lucide-triangle-alert" class="gmc-note__icon" />
      <span class="gmc-note__text">{{ t('gmConfig.sim.problems', { n: problems.length }) }}</span>
    </p>

    <template v-if="result">
      <p class="gmc-hint">{{ t('gmConfig.sim.done', { n: result.n.toLocaleString(), lottery: resultLottery }) }}<template v-if="outdated"> · {{ t('gmConfig.sim.outdated') }}</template></p>
      <div class="gs__grid">
        <div v-for="grp in result.groups" :key="`${grp.resourceType}:${grp.rarity}`" class="gs__tile">
          <span class="gs__label">{{ rarityLabel(grp.rarity) }} {{ kindLabel(grp.resourceType) }}</span>
          <span class="gs__value m-num">{{ pct(grp.observed) }}</span>
          <span class="gs__sub m-num">{{ t('gmConfig.sim.expected', { v: pct(grp.expected) }) }}</span>
        </div>
        <div v-for="p in result.pickups" :key="p.id" class="gs__tile is-up">
          <span class="gs__label">UP {{ nameOf(p.resourceType, p.id) }}</span>
          <span class="gs__value m-num">{{ pct(p.observed) }}</span>
          <span class="gs__sub m-num">{{ t('gmConfig.sim.expected', { v: pct(p.expected) }) }}</span>
        </div>
        <div v-if="result.expectedFirstUp" class="gs__tile is-up">
          <span class="gs__label">{{ t('gmConfig.sim.firstUp') }}</span>
          <span class="gs__value m-num">
            {{ result.firstUp ? t('gmConfig.sim.draws', { n: Math.round(result.firstUp.mean) }) : t('gmConfig.sim.noHit') }}
          </span>
          <span class="gs__sub m-num">
            <template v-if="result.firstUp">{{ t('gmConfig.sim.median', { n: result.firstUp.median }) }} · </template>
            {{ t('gmConfig.sim.theory', { n: Math.round(result.expectedFirstUp) }) }}
          </span>
        </div>
        <div v-if="exchangeAt" class="gs__tile">
          <span class="gs__label">{{ t('gmConfig.sim.exchange') }}</span>
          <span class="gs__value m-num">{{ t('gmConfig.sim.draws', { n: exchangeAt }) }}</span>
          <span class="gs__sub">{{ t('gmConfig.sim.exchangeSub', { n: exchangeAt }) }}</span>
        </div>
      </div>
    </template>
    <p v-else class="gmc-hint">{{ t('gmConfig.sim.hint') }}</p>
  </AppPanel>
</template>

<script setup lang="ts">
import { computed, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useGashaEditor } from './editor'
import { useGashaLabels } from './labels'
import {
  catalogKindOf,
  lotteryById,
  prepareLottery,
  rarityLabel,
  RESOURCE_IDOL,
  RESOURCE_SUPPORT,
  simulate,
  type SimResult,
} from './gasha'

const SIZES = [1000, 10000, 100000] as const

const { t } = useI18n()
const ed = useGashaEditor()
const labels = useGashaLabels()

const size = ref<number>(10000)
const lotteryId = ref('')
const result = shallowRef<SimResult>()
const resultLottery = ref('')
const resultText = ref('')

const lotteries = computed(() => {
  const g = ed.gasha.value
  if (!g) return []
  const out: { id: string; label: string }[] = []
  if (g.lotteryId) out.push({ id: g.lotteryId, label: t('gmConfig.gasha.normalSlot') })
  if (g.fixLotteryId && g.fixLotteryId !== g.lotteryId) out.push({ id: g.fixLotteryId, label: t('gmConfig.gasha.fixSlot') })
  return out
})

watch(
  () => ed.gasha.value?.id,
  () => {
    result.value = undefined
    lotteryId.value = lotteries.value[0]?.id || ''
  },
  { immediate: true },
)
watch(lotteries, (list) => {
  if (!list.some((l) => l.id === lotteryId.value)) lotteryId.value = list[0]?.id || ''
})

const prepared = computed(() => {
  const c = ed.cfg.value
  const l = c && lotteryId.value ? lotteryById(c, lotteryId.value) : undefined
  return c && l ? prepareLottery(c, l, ed.rarityOf) : undefined
})
const problems = computed(() => prepared.value?.problems || [])

const lotteryText = computed(() => {
  const c = ed.cfg.value
  const l = c && lotteryId.value ? lotteryById(c, lotteryId.value) : undefined
  return l ? JSON.stringify(l) : ''
})
const outdated = computed(() => !!result.value && resultText.value !== lotteryText.value)

const run = () => {
  if (!prepared.value) return
  result.value = simulate(prepared.value, size.value)
  resultLottery.value = lotteries.value.find((l) => l.id === lotteryId.value)?.label || ''
  resultText.value = lotteryText.value
}

// 交换点：能换到 UP 卡的最少点数（每抽 1 点）
const exchangeAt = computed(() => {
  const g = ed.gasha.value
  const c = ed.cfg.value
  const p = g?.gashaPointId ? c?.points.find((x) => x.id === g.gashaPointId) : undefined
  if (!g || !p) return 0
  const ups = new Set([
    ...(g.pickupIdolCardSkinIds || []).map((id) => `${RESOURCE_IDOL}:${id}`),
    ...(g.pickupSupportCardIds || []).map((id) => `${RESOURCE_SUPPORT}:${id}`),
  ])
  const need = (p.exchanges || [])
    .filter((x) => x.requiredQuantity > 0 && (x.rewards || []).some((r) => ups.has(`${r.resourceType}:${r.resourceId}`)))
    .map((x) => x.requiredQuantity)
  return need.length ? Math.min(...need) : 0
})

const pct = (v: number) => `${(v * 100).toFixed(3)}%`
const kindLabel = (resourceType: number) => t(`gmConfig.gasha.picker.${catalogKindOf(resourceType)}`)
const nameOf = (resourceType: number, id: string) => labels.card(resourceType, id)
</script>

<style lang="scss" src="./shared/form.scss"></style>
<style scoped lang="scss">
.gs__bar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}
.gs__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 10px;
  margin-top: 10px;
}
.gs__tile {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 10px 12px;
  border-radius: 14px;
  background: var(--m-s2);
  min-width: 0;
}
.gs__tile.is-up {
  background: var(--m-gSoft, var(--m-pSoft));
}
.gs__label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gs__value {
  font-family: var(--app-font-display);
  font-size: 18px;
  font-weight: 800;
}
.gs__sub {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
