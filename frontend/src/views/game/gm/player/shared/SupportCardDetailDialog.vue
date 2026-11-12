<!-- 支援卡详情弹窗（支援卡网格里点开）：原画（16:9 高清）、卡名、稀有度、等级、上限解放、库存、是否援助中与获得时间。
     「设为援助支援卡」先关闭弹窗，再用操作对话框执行 supportCard.setRental（试运行、确认与 GM 记录同其他操作）。 -->
<template>
  <FormDialog :model-value="!!card" :title="t('gmData.cardDetail.supportTitle')" :width="720" @update:model-value="onToggle">
    <div v-if="card" class="cdd">
      <GameImage :src="art" :alt="cardName" class="cdd__art" eager />
      <div class="cdd__head">
        <RarityBadge :rarity="card.rarity" />
        <span class="cdd__name">{{ cardName }}</span>
        <span v-if="card.lent" class="pv__tag is-g">{{ t('gmData.cardDetail.lent') }}</span>
      </div>
      <div class="cdd__facts">
        <div class="cdd__fact">
          <span class="pv__dim">{{ t('gmData.cardDetail.level') }}</span>
          <span class="cdd__v m-num">{{ card.level }}</span>
        </div>
        <div class="cdd__fact">
          <span class="pv__dim">{{ t('gmData.cardDetail.limit') }}</span>
          <span class="cdd__v m-num">{{ card.levelLimitRank }}</span>
        </div>
        <div class="cdd__fact">
          <span class="pv__dim">{{ t('gmData.cardDetail.stock') }}</span>
          <span class="cdd__v m-num">{{ num(card.stockQuantity) }}</span>
        </div>
        <div class="cdd__fact">
          <span class="pv__dim">{{ t('gmData.cardDetail.acquired') }}</span>
          <span class="cdd__v cdd__v--sm m-num">{{ dateTime(card.createdTime) }}</span>
        </div>
      </div>
    </div>
    <template #footer="{ close }">
      <UButton color="neutral" variant="soft" @click="close">{{ t('gmData.common.close') }}</UButton>
      <UButton v-if="canSetRental" color="primary" :disabled="!card || card.lent" @click="setRental">
        {{ card?.lent ? t('gmData.cardDetail.lent') : t('gmData.cardDetail.setRental') }}
      </UButton>
    </template>
  </FormDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import FormDialog from '@/components/ui/FormDialog.vue'
import type { SupportCard } from '@/games/gakumas/types'
import { useGmContext } from '../../context'
import { usePlayerPage } from '../context'
import RarityBadge from './RarityBadge.vue'
import { supportCardArt } from './game'
import { useGmFormat } from './format'
import { useCatalogLookup } from './lookup'

const props = defineProps<{ card: SupportCard | null }>()
const emit = defineEmits<{ 'update:card': [value: SupportCard | null] }>()

const { t } = useI18n()
const ctx = useGmContext()
const page = usePlayerPage()
const lookup = useCatalogLookup()
const { num, dateTime } = useGmFormat()

const cardName = computed(() => (props.card ? lookup.name('supportCard', props.card.supportCardId) : ''))
const art = computed(() => supportCardArt(ctx.game.value, props.card?.supportCardId || ''))

const canSetRental = computed(() => !!ctx.operation('supportCard.setRental'))

const onToggle = (open: boolean) => {
  if (!open) emit('update:card', null)
}

const setRental = () => {
  const id = props.card?.supportCardId || ''
  emit('update:card', null)
  if (id) page.run('supportCard.setRental', { supportCardId: id })
}
</script>

<style scoped lang="scss">
@use './views';

.cdd {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.cdd__art {
  width: 100%;
  aspect-ratio: 16 / 9;
  object-fit: cover;
  border-radius: 14px;
}
.cdd__head {
  display: flex;
  align-items: center;
  gap: 10px;
}
.cdd__name {
  flex: 1;
  min-width: 0;
  font-size: 18px;
  font-weight: 800;
}
.cdd__facts {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
}
.cdd__fact {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 8px 10px;
  border-radius: 12px;
  background: var(--el-fill-color-light);
  font-size: 12px;
}
.cdd__v {
  font-size: 18px;
  font-weight: 800;
}
.cdd__v--sm {
  font-size: 13px;
}
@media (max-width: 560px) {
  .cdd__facts {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
