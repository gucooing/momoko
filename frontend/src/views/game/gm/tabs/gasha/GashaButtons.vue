<!-- 「4 抽卡按钮」：卡池引用的 GashaButton（客户端主数据）、按钮单独用的抽选表与次数上限。
     按钮按名称选（名称来自前端名称表；同名按钮用「哪些卡池在用」区分），抽选表按用途称呼，界面上不显示 ID。
     消耗、抽数和限次类型都在客户端主数据里，MGP 不提供。阶梯 / 自选 / 连抽卡池的特殊设置只读显示。 -->
<template>
  <section v-if="g && cfg" class="gmc-sec">
    <div class="gmc-sh"><span class="gmc-sn">4</span><span class="gmc-sh__title">{{ t('gmConfig.gasha.sec.buttons') }}</span></div>

    <template v-if="!stepOnly">
      <div class="gbt__head" aria-hidden="true">
        <span>{{ t('gmConfig.gasha.buttonId') }}</span>
        <span>{{ t('gmConfig.gasha.normalSlot') }}</span>
        <span>{{ t('gmConfig.gasha.fixSlot') }}</span>
        <span>{{ t('gmConfig.gasha.limitCount') }}</span>
        <span />
      </div>
      <div v-for="(b, i) in g.buttons" :key="i" class="gbt__row">
        <div class="gbt__cell">
          <span class="gbt__label">{{ t('gmConfig.gasha.buttonId') }}</span>
          <AppSelect v-model="b.id" :options="buttonOptions" :error="!b.id" :placeholder="t('gmConfig.gasha.buttonIdHint')" searchable />
        </div>
        <div class="gbt__cell">
          <span class="gbt__label">{{ t('gmConfig.gasha.normalSlot') }}</span>
          <AppSelect v-model="b.lotteryId" :options="lotteryOptions" searchable />
        </div>
        <div class="gbt__cell">
          <span class="gbt__label">{{ t('gmConfig.gasha.fixSlot') }}</span>
          <AppSelect v-model="b.fixLotteryId" :options="lotteryOptions" searchable />
        </div>
        <label class="gbt__cell">
          <span class="gbt__label">{{ t('gmConfig.gasha.limitCount') }}</span>
          <input
            class="gmc-input gmc-input--sm m-num"
            type="number"
            min="0"
            step="1"
            :value="b.limitCount || ''"
            :placeholder="t('gmConfig.gasha.limitMaster')"
            @change="setLimit(i, ($event.target as HTMLInputElement).valueAsNumber)"
          />
        </label>
        <AppIconButton icon="HOutline:XMarkIcon" :label="t('gmConfig.gasha.removeButton', { name: b.id ? labels.button(b.id) : '—' })" :box="40" class="gbt__del" @click="g.buttons.splice(i, 1)" />
      </div>
      <UButton class="gmc-btn gbt__add" size="sm" color="neutral" variant="soft" icon="i-lucide-plus" @click="addButton">{{ t('gmConfig.gasha.addButton') }}</UButton>
      <p class="gmc-hint">{{ t('gmConfig.gasha.buttonsHint') }}</p>
    </template>

    <div v-for="s in steps" :key="s.number" class="gbt__step">
      <span class="gbt__step-n m-num">STEP {{ s.number }}</span>
      <span>{{ labels.button(s.gashaButtonId) }}</span>
      <span v-if="s.pickupCount" class="gmc-tag gmc-tag--g">{{ t('gmConfig.gasha.stepPick', { n: s.pickupCount }) }}</span>
    </div>
    <p v-if="readonlyShape" class="gmc-note gmc-note--info">
      <UIcon name="i-lucide-info" class="gmc-note__icon" />
      <span class="gmc-note__text">{{ readonlyShape }}</span>
    </p>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { gameNameIds } from '@/games/names'
import { getCurrentLocale } from '@/locales'
import { useGashaEditor } from './editor'
import { useGashaLabels } from './labels'

const { t } = useI18n()
const ed = useGashaEditor()
const labels = useGashaLabels()
const g = computed(() => ed.gasha.value)
const cfg = computed(() => ed.cfg.value)

const stepOnly = computed(() => g.value?.type === 2 || g.value?.type === 5)

const lotteryOptions = computed(() => {
  const c = cfg.value
  return [
    { label: t('gmConfig.gasha.followGasha'), value: '' },
    ...(c ? c.lotteries.map((l) => ({ label: labels.lottery(c, l.id), value: l.id })) : []),
  ]
})

// 可选的按钮：客户端主数据里的全部按钮（名称表的原文表就是主数据的完整列表）加上配置里已经用到的。
// GashaButton 没有 MGP 目录；名称相同的按钮很多（例如每个卡池各有一个「10 回ガシャ」），用在用的卡池区分。
const masterButtons = ref<string[]>([])
void gameNameIds(ed.game.value, 'gashaButton').then((ids) => {
  masterButtons.value = ids
})
const buttonOptions = computed(() => {
  const c = cfg.value
  const users = new Map<string, string[]>()
  const use = (id: string, gasha: string) => {
    if (!id) return
    const list = users.get(id) || []
    if (!list.includes(gasha)) list.push(gasha)
    users.set(id, list)
  }
  for (const x of c?.gashas || []) {
    for (const b of x.buttons || []) use(b.id, labels.gasha(x))
    for (const s of [...(x.stepUp?.steps || []), ...(x.stepUpSelectPickup?.steps || [])]) use(s.gashaButtonId, labels.gasha(x))
  }
  const ids = [...new Set([...users.keys(), ...masterButtons.value])]
  return ids
    .map((id) => {
      const list = users.get(id)
      return {
        label: labels.button(id),
        value: id,
        description: list?.length ? t('gmConfig.gasha.usedBy', { gashas: list.join('、') }) : t('gmConfig.gasha.unused'),
        used: !!list?.length,
      }
    })
    .sort((a, b) => Number(b.used) - Number(a.used) || a.label.localeCompare(b.label, getCurrentLocale()))
    .map(({ label, value, description }) => ({ label, value, description }))
})

const steps = computed(() => [...(g.value?.stepUp?.steps || []), ...(g.value?.stepUpSelectPickup?.steps || [])])

const readonlyShape = computed(() => {
  const x = g.value
  if (!x) return ''
  if (x.stepUp) return t('gmConfig.gasha.shapeStepUp', { n: x.stepUp.steps?.length || 0, loop: x.stepUp.isLimited ? t('gmConfig.gasha.stepLimited') : t('gmConfig.gasha.stepLoop') })
  if (x.stepUpSelectPickup) return t('gmConfig.gasha.shapeStepUpSelect', { n: x.stepUpSelectPickup.steps?.length || 0 })
  if (x.selectPickup) {
    const s = x.selectPickup
    return t('gmConfig.gasha.shapeSelect', { n: s.pickupCount, total: (s.idolCardSkinIds?.length || 0) + (s.supportCardIds?.length || 0) })
  }
  if (x.continuous) {
    const c = x.continuous
    return t('gmConfig.gasha.shapeContinuous', { first: c.firstDrawCount, max: c.maxContinueCount })
  }
  return ''
})

const setLimit = (i: number, value: number) => {
  const b = g.value?.buttons?.[i]
  if (!b) return
  if (Number.isFinite(value) && value > 0) b.limitCount = Math.trunc(value)
  else delete b.limitCount
}

const addButton = () => {
  const x = g.value
  if (!x) return
  if (!x.buttons) x.buttons = []
  x.buttons.push({ id: '', lotteryId: '', fixLotteryId: '' })
}
</script>

<style lang="scss" src="./shared/form.scss"></style>
<style scoped lang="scss">
.gbt__head,
.gbt__row {
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) minmax(0, 1fr) minmax(0, 1fr) 120px 40px;
  gap: 10px;
  align-items: center;
}
.gbt__head {
  font-size: 12px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.gbt__row {
  padding: 8px 0;
  border-top: 1px solid var(--el-border-color-lighter);
}
.gbt__cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}
.gbt__label {
  display: none;
  font-size: 12px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.gbt__add {
  align-self: flex-start;
  min-height: 40px;
}
.gbt__step {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: 12px;
  background: var(--el-fill-color-light);
  font-size: 13px;
}
.gbt__step-n {
  font-weight: 800;
}
@media (max-width: 720px) {
  .gbt__head {
    display: none;
  }
  .gbt__row {
    grid-template-columns: minmax(0, 1fr) 44px;
    padding: 12px;
    border: 1px solid var(--el-border-color-light);
    border-radius: 16px;
    margin-bottom: 8px;
  }
  .gbt__cell {
    grid-column: 1;
  }
  .gbt__del {
    grid-column: 2;
    grid-row: 1;
  }
  .gbt__label {
    display: block;
  }
}
</style>
