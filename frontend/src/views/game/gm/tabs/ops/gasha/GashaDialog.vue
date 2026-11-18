<!-- 编辑一个卡池（运营管理里点日历上的卡池直接打开）：基本信息、UP 角色与卡牌、概率、抽卡按钮、交换点，下面是模拟抽卡与发布前检查。 -->
<template>
  <ItemDialog
    :model-value="modelValue"
    :title="labels.gasha(gasha || publishedGasha)"
    :width="1180"
    :state="gasha ? 'ok' : publishedGasha ? 'removed' : 'missing'"
    :status="gasha ? periodStatus(gasha.startTime, gasha.endTime, clock.now.value) : undefined"
    :draft="draftOf(publishedGasha, gasha)"
    :copy-label="t('gmConfig.ops.copy')"
    @update:model-value="emit('update:modelValue', $event)"
    @copy="copyOpen = true"
    @delete="remove"
    @restore="restore"
  >
    <GashaBasics />
    <GashaUps />
    <GashaRates />
    <GashaButtons />
    <GashaPoint />
    <div class="ged__side">
      <GashaSimulator class="ged__side-item" />
      <GashaChecks class="ged__side-item" />
    </div>
  </ItemDialog>
  <GashaCreateDialog v-model="copyOpen" :template="gashaId" @created="emit('open', { kind: 'gasha', id: $event })" />
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useGmContext } from '../../../context'
import type { ItemRef } from '../calendar'
import { useOpsDomains } from '../domains'
import { cloneJson, draftOf, periodStatus } from '../shared/entry'
import ItemDialog from '../shared/ItemDialog.vue'
import { useServerClock } from '../shared/serverClock'
import GashaBasics from './GashaBasics.vue'
import GashaButtons from './GashaButtons.vue'
import GashaChecks from './GashaChecks.vue'
import GashaCreateDialog from './GashaCreateDialog.vue'
import GashaPoint from './GashaPoint.vue'
import GashaRates from './GashaRates.vue'
import GashaSimulator from './GashaSimulator.vue'
import GashaUps from './GashaUps.vue'
import { provideGashaEditor } from './editor'
import { useGashaLabels } from './labels'

const props = defineProps<{ modelValue: boolean; gashaId: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; open: [ref: ItemRef] }>()

const { t } = useI18n()
const ctx = useGmContext()
const clock = useServerClock(ctx)
const domains = useOpsDomains()
const labels = useGashaLabels()
const state = domains.gasha.state

const cfg = computed(() => state.working.value)
const gasha = computed(() => cfg.value?.gashas.find((g) => g.id === props.gashaId))
const publishedGasha = computed(() => state.published.value?.gashas.find((g) => g.id === props.gashaId))

provideGashaEditor({
  state,
  cfg,
  gasha,
  publishedGasha,
  zone: clock.zone,
  now: clock.now,
  game: ctx.game,
  lookup: domains.lookup,
  rarityOf: domains.rarityOf,
  cardStatus: domains.cardStatus,
})

const copyOpen = ref(false)
const remove = () => {
  const c = cfg.value
  const i = c && gasha.value ? c.gashas.indexOf(gasha.value) : -1
  if (c && i >= 0) c.gashas.splice(i, 1)
}
const restore = () => {
  if (cfg.value && publishedGasha.value) cfg.value.gashas.push(cloneJson(publishedGasha.value))
}
</script>

<style scoped lang="scss">
.ged__side {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  align-items: stretch;
}
.ged__side-item {
  flex: 1 1 320px;
  min-width: 0;
}
</style>
