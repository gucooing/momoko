<!-- 自助撤回礼物：列出我礼物箱里管理端发出（运营发放、邮件、兑换码）且未领取的礼物，页码分页，勾选要撤回的；
     一次只能撤回同一种来源的礼物，勾选第一件后其他来源的不能再勾。 -->
<template>
  <div class="sgp">
    <div v-if="list.items.length" class="sgp__rows">
      <label v-for="g in list.items" :key="g.giftId" class="sgp__row" :class="{ 'is-off': !pickable(g) }">
        <input type="checkbox" :checked="modelValue.ids.includes(g.giftId)" :disabled="!pickable(g)" @change="toggle(g)" />
        <span class="sgp__img"><GameImage :src="ctx.resourceImage(g.resourceType, g.resourceId)" /></span>
        <span class="sgp__text">
          <span class="sgp__name">
            {{ ctx.resourceName(g.resourceType, g.resourceId) || t('game.unnamed') }}
            <span class="pp-num pp-hint">{{ t('game.gifts.qty', { n: fmtNum(g.quantity) }) }}</span>
          </span>
          <span class="pp-hint sgp__src">{{ g.message ? `${g.message} · ` : '' }}{{ t(`game.gifts.source.${g.sourceKind}`) }}</span>
        </span>
      </label>
    </div>
    <p v-else-if="!list.loading && list.loaded" class="pp-hint sgp__empty">{{ t('game.self.noRevocable') }}</p>
    <p v-if="list.error" class="pp-hint sgp__err">{{ list.error }}</p>
    <PPager v-if="list.loaded" :page="list.page" :total="list.total" :page-size="PAGE" :disabled="list.loading" @update:page="(p) => void paged.load(p)" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import GameImage from '@/components/game/GameImage.vue'
import type { Gift } from '@/games/gakumas/types'
import PPager from '@passport/components/PPager.vue'
import { listMyGifts } from '@passport/api/gakumas'
import { fmtNum } from '@passport/utils/format'
import { offsetCursor, usePagedList } from '@passport/utils/paged'
import { useGameContext } from './context'

/** 勾选的礼物：来源（1 运营发放、2 邮件、3 兑换码；没勾选为 0）与礼物 ID。 */
export interface GiftPick {
  kind: number
  ids: string[]
}

const props = defineProps<{ modelValue: GiftPick }>()
const emit = defineEmits<{ 'update:modelValue': [value: GiftPick] }>()

const PAGE = 20

const { t } = useI18n()
const ctx = useGameContext()

const paged = usePagedList<Gift>(
  (page, size) =>
    listMyGifts({
      serverId: ctx.serverId.value,
      state: 'unreceived',
      source: 'admin',
      expiring: false,
      cursor: offsetCursor(page, size),
      limit: size,
    }),
  PAGE,
)
const list = paged.list
void paged.reload()

const pickable = (g: Gift) => !props.modelValue.kind || g.sourceKind === props.modelValue.kind

const toggle = (g: Gift) => {
  const ids = props.modelValue.ids.includes(g.giftId) ? props.modelValue.ids.filter((id) => id !== g.giftId) : [...props.modelValue.ids, g.giftId]
  emit('update:modelValue', { kind: ids.length ? g.sourceKind : 0, ids })
}
</script>

<style scoped>
.sgp {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.sgp__rows {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.sgp__row {
  display: grid;
  grid-template-columns: 20px 40px minmax(0, 1fr);
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: 12px;
  background: var(--m-s2);
  cursor: pointer;
}
.sgp__row.is-off {
  opacity: 0.5;
  cursor: default;
}
.sgp__row input {
  width: 18px;
  height: 18px;
  accent-color: var(--m-p);
}
.sgp__img {
  width: 40px;
  height: 40px;
}
.sgp__img :deep(img) {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
.sgp__text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.sgp__name {
  font-size: 13px;
  font-weight: 700;
}
.sgp__src {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.sgp__empty,
.sgp__err {
  margin: 0;
}
.sgp__err {
  color: var(--m-bad);
}
</style>
