<!-- 偶像卡详情弹窗（偶像卡网格里点开）：左侧原画（9:16 高清），右侧卡名与角色、稀有度、才能开花、上限解放、Prima Stella、获得时间、
     持有的皮肤（点一套皮肤，原画与卡名换成这套皮肤预览；只在页面上预览，不改玩家的皮肤），
     以及这张卡保留的培育记录（从新到旧，服务端分页，点开看记忆详情）。 -->
<template>
  <FormDialog :model-value="!!card" :title="t('gmData.cardDetail.idolTitle')" :width="860" :show-footer="false" @update:model-value="onToggle">
    <div v-if="card" class="cdd">
      <div class="cdd__top">
      <GameImage :src="art" :alt="cardName" class="cdd__art" eager />
      <div class="cdd__side">
      <div class="cdd__head">
        <RarityBadge :rarity="card.rarity" />
        <span class="cdd__name">{{ cardName }}</span>
      </div>
      <span v-if="charaName" class="pv__dim">{{ charaName }}</span>
      <div class="cdd__facts">
        <div class="cdd__fact">
          <span class="pv__dim">{{ t('gmData.cardDetail.potential') }}</span>
          <span class="cdd__v m-num">{{ card.potentialRank }}</span>
        </div>
        <div class="cdd__fact">
          <span class="pv__dim">{{ t('gmData.cardDetail.limit') }}</span>
          <span class="cdd__v m-num">{{ card.levelLimitRank }}</span>
        </div>
        <div class="cdd__fact">
          <span class="pv__dim">Prima Stella</span>
          <span class="cdd__v">{{ card.primaStella ? t('gmData.cardDetail.yes') : t('gmData.cardDetail.no') }}</span>
        </div>
        <div class="cdd__fact">
          <span class="pv__dim">{{ t('gmData.cardDetail.acquired') }}</span>
          <span class="cdd__v cdd__v--sm m-num">{{ dateTime(card.createdTime) }}</span>
        </div>
      </div>

      <h4 class="cdd__h">
        {{ t('gmData.cardDetail.skins', { n: skins.length }) }}
        <span v-if="skins.length > 1" class="cdd__hint">{{ t('gmData.cardDetail.skinHint') }}</span>
      </h4>
      <div class="cdd__skins">
        <button
          v-for="s in skins"
          :key="s"
          type="button"
          class="cdd__skin"
          :class="{ 'is-on': s === shownSkin }"
          :aria-pressed="s === shownSkin"
          @click="preview = s"
        >
          <GameImage :src="skinImage(s)" :alt="lookup.name('idolCardSkin', s)" class="cdd__sq" />
          <span class="cdd__small pv__ellipsis">{{ lookup.name('idolCardSkin', s) }}</span>
          <span v-if="s === card.skinId" class="pv__tag is-g">{{ t('gmData.cardDetail.current') }}</span>
        </button>
      </div>
      </div>
      </div>

      <h4 class="cdd__h">{{ t('gmData.cardDetail.records', { n: records.total.value }) }}</h4>
      <div v-if="records.error.value" class="pv__error">{{ records.error.value }}</div>
      <div v-else-if="records.loading.value && !records.items.value.length" class="pv__hint">{{ t('gmData.common.loading') }}</div>
      <p v-else-if="!records.items.value.length" class="pv__hint">{{ t('gmData.cardDetail.noRecords') }}</p>
      <div v-else class="cdd__records">
        <button v-for="m in records.items.value" :key="m.memoryId" type="button" class="cdd__record" @click="emit('memory', m.memoryId)">
          <GradeBadge :grade="m.grade" small />
          <span class="pv__strong m-num">{{ num(m.score) }}</span>
          <span class="pv__dim m-num cdd__stats">Vo {{ num(m.vocal) }} · Da {{ num(m.dance) }} · Vi {{ num(m.visual) }}</span>
          <span v-if="m.lent" class="pv__tag is-g">{{ t('gmData.memories.lent') }}</span>
          <span class="pv__dim m-num cdd__when">{{ when(m.createdTime) }}</span>
        </button>
      </div>
      <Pagination
        v-if="records.total.value > records.pageSize.value"
        v-model:page="records.page.value"
        v-model:page-size="records.pageSize.value"
        :total="records.total.value"
        @change="records.load()"
      />
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { listPlayerMemories } from '@/api/gakumas'
import FormDialog from '@/components/ui/FormDialog.vue'
import type { IdolCard, Memory } from '@/games/gakumas/types'
import { useGmContext } from '../../context'
import { usePlayerPage } from '../context'
import GradeBadge from './GradeBadge.vue'
import RarityBadge from './RarityBadge.vue'
import { catalogImage, idolCardArt } from './game'
import { useGmFormat } from './format'
import { useCatalogLookup } from './lookup'
import { usePagedList, type PageQuery } from './paging'

const props = defineProps<{ card: IdolCard | null }>()
const emit = defineEmits<{ 'update:card': [value: IdolCard | null]; memory: [memoryId: string] }>()

const { t } = useI18n()
const ctx = useGmContext()
const page = usePlayerPage()
const lookup = useCatalogLookup()
const { num, when, dateTime } = useGmFormat()

const skins = computed(() => {
  const c = props.card
  if (!c) return []
  const list = c.skinIds?.length ? [...c.skinIds] : c.skinId ? [c.skinId] : []
  return list.sort((a, b) => Number(b === c.skinId) - Number(a === c.skinId))
})
// 预览的皮肤（只在页面上切换，默认是卡当前的皮肤）
const preview = ref('')
const shownSkin = computed(() => preview.value || props.card?.skinId || '')
watch(
  () => props.card?.idolCardId,
  () => {
    preview.value = ''
  },
)
const cardName = computed(() => {
  const c = props.card
  if (!c) return ''
  return (shownSkin.value && lookup.name('idolCardSkin', shownSkin.value)) || lookup.name('idolCard', c.idolCardId)
})
const charaName = computed(() => (props.card?.characterId ? lookup.name('character', props.card.characterId) : ''))
const art = computed(() => idolCardArt(ctx.game.value, shownSkin.value || props.card?.idolCardId || '', 'full'))
const skinImage = (id: string) => catalogImage(ctx.game.value, 'idolCardSkin', { id })

// 这张卡保留的培育记录：服务端按页读取
const fetchRecords = async (q: PageQuery) => {
  const id = props.card?.idolCardId
  if (!id) return { items: [], nextCursor: '', total: 0 }
  return listPlayerMemories({
    serverId: ctx.serverId.value,
    pid: page.pid.value,
    produced: true,
    q: '',
    rarity: 0,
    grades: [],
    characterId: '',
    idolCardIds: [id],
    cursor: q.cursor,
    limit: q.limit,
  })
}
const records = usePagedList<Memory>(fetchRecords, { pageSize: 20 })
watch(
  () => props.card?.idolCardId || '',
  () => void records.reload(),
  { immediate: true },
)

const onToggle = (open: boolean) => {
  if (!open) emit('update:card', null)
}
</script>

<style scoped lang="scss">
@use './views';

.cdd {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.cdd__top {
  display: flex;
  gap: 16px;
  align-items: flex-start;
}
.cdd__art {
  flex: none;
  width: 300px;
  aspect-ratio: 9 / 16;
  object-fit: cover;
  border-radius: 14px;
}
.cdd__side {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
}
.cdd__head {
  display: flex;
  align-items: center;
  gap: 10px;
}
.cdd__name {
  font-size: 18px;
  font-weight: 800;
  min-width: 0;
}
.cdd__facts {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
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
.cdd__h {
  margin: 8px 0 0;
  font-size: 13px;
  font-weight: 800;
  color: var(--el-text-color-secondary);
}
.cdd__skins {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(92px, 1fr));
  gap: 8px;
}
.cdd__hint {
  margin-left: 8px;
  font-weight: 400;
  font-size: 12px;
}
.cdd__skin {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  min-width: 0;
  padding: 6px;
  border-radius: 12px;
  border: 1px solid var(--el-border-color-light);
  background: transparent;
  color: inherit;
  font: inherit;
  cursor: pointer;
}
.cdd__skin:hover {
  border-color: var(--el-color-primary);
}
.cdd__skin.is-on {
  border-color: var(--el-color-primary);
}
.cdd__sq {
  width: 64px;
  height: 64px;
  border-radius: 10px;
}
.cdd__small {
  max-width: 100%;
  font-size: 12px;
}
.cdd__records {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.cdd__record {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 12px;
  background: var(--el-bg-color);
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.cdd__record:hover {
  border-color: var(--el-color-primary);
}
.cdd__stats {
  flex: 1;
  min-width: 0;
}
.cdd__when {
  flex: none;
}
@media (max-width: 640px) {
  .cdd__top {
    flex-direction: column;
    align-items: stretch;
  }
  .cdd__art {
    width: 100%;
    max-width: 360px;
    align-self: center;
  }
  .cdd__stats {
    display: none;
  }
}
</style>
