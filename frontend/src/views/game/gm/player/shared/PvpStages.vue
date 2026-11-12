<!-- 竞赛阵容：每个阶段一张卡（阶段编成力 + 段位允许的槽位），每个槽位一张主记忆与至多一张副记忆。
     数据来自 MGP 玩家视图 pvp / pvp-unit，只读。 -->
<template>
  <div class="pvs">
    <div v-for="stage in stages" :key="stage.stage" class="pvs__stage">
      <div class="pvs__head">
        <span class="pvs__title">{{ t('gmData.pvp.stage', { n: stage.stage }) }}</span>
        <span class="pvs__power m-num">{{ num(stage.power) }}</span>
      </div>
      <div v-if="stage.slots?.length" class="pvs__slots">
        <div v-for="(slot, i) in stage.slots" :key="i" class="pvs__slot">
          <div v-for="part in parts(slot)" :key="part.key" class="pvs__mem" :class="`is-${part.key}`">
            <template v-if="part.memory">
              <GameImage :src="memoryImage(part.memory)" :alt="charaName(part.memory)" class="pvs__img" />
              <span class="pvs__cap">
                <GradeBadge :grade="part.memory.grade" small />
                <span class="pvs__name">{{ charaName(part.memory) }}</span>
              </span>
            </template>
            <template v-else>
              <span class="pvs__empty">{{ t('gmData.pvp.emptySlot') }}</span>
              <span class="pvs__cap pvs__cap--dim">{{ part.key === 'main' ? t('gmData.pvp.main') : t('gmData.pvp.sub') }}</span>
            </template>
          </div>
        </div>
      </div>
      <div v-else class="pvs__none">{{ t('gmData.pvp.noSlots') }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { Memory, PvpSlot, PvpStage } from '@/games/gakumas/types'
import { useGmContext } from '../../context'
import { catalogImage } from './game'
import GradeBadge from './GradeBadge.vue'
import { useGmFormat } from './format'
import { useCatalogLookup } from './lookup'

defineProps<{ stages: PvpStage[] }>()

const { t } = useI18n()
const ctx = useGmContext()
const lookup = useCatalogLookup()
const { num } = useGmFormat()

const parts = (slot: PvpSlot) => [
  { key: 'main', memory: slot.main },
  { key: 'sub', memory: slot.sub },
]
const memoryImage = (m: Memory) => catalogImage(ctx.game.value, 'idolCardSkin', { id: m.idolCardId })
const charaName = (m: Memory) => (m.characterId ? lookup.name('character', m.characterId) : m.idolCardId)
</script>

<style scoped>
.pvs {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 12px;
}
.pvs__stage {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px;
  border-radius: var(--app-radius-lg);
  background: var(--m-s2);
  min-width: 0;
}
.pvs__head {
  display: flex;
  align-items: center;
  gap: 8px;
}
.pvs__title {
  flex: 1;
  font-size: 14px;
  font-weight: 800;
}
.pvs__power {
  font-size: 13px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.pvs__slots {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.pvs__slot {
  display: grid;
  grid-template-columns: 1.4fr 1fr;
  gap: 8px;
  align-items: end;
}
.pvs__mem {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}
.pvs__img,
.pvs__empty {
  width: 100%;
  aspect-ratio: 1;
  border-radius: 14px;
}
.pvs__empty {
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1.5px dashed var(--el-border-color);
  font-size: 11px;
  color: var(--el-text-color-secondary);
}
.pvs__cap {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
}
.pvs__cap--dim {
  font-size: 11px;
  color: var(--el-text-color-secondary);
}
.pvs__name {
  font-size: 11px;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.pvs__none {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
</style>
