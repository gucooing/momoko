<!-- 一张卡的详情：大图、名称、角色、养成数值、获得时间。
     偶像卡另列持有的皮肤（点一套皮肤，大图与卡名换成这套皮肤预览；只在页面上预览）与这张卡保留的培育记录
     （页码分页；点开看回忆详情，发出 memory）；
     支援卡可设为援助（与游戏里的资料设置相同，立即生效），成功后发出 changed。 -->
<template>
  <div class="cd" :class="{ 'cd--idol': idol }">
    <GameImage v-if="art" :src="art" :alt="name" class="cd__img" eager />
    <div class="cd__body">
      <div class="cd__head">
        <span class="cd__rar" :class="`cd__rar--${rarity}`">{{ rarityLabel(rarity) }}</span>
        <span class="cd__name">{{ name }}</span>
      </div>
      <span v-if="chara" class="pp-hint cd__sub">{{ chara }}</span>
      <div v-if="tags.length" class="cd__tags">
        <span v-for="x in tags" :key="x" class="pp-tag pp-tag--g">{{ x }}</span>
      </div>
      <dl class="cd__dl">
        <div v-for="f in facts" :key="f.k" class="cd__fact">
          <dt class="pp-hint">{{ f.k }}</dt>
          <dd class="pp-num">{{ f.v }}</dd>
        </div>
      </dl>
      <span class="cd__acq">{{ t('game.cards.acquired') }} · {{ fmtTime(createdTime) }}</span>

      <template v-if="idol">
        <h3 class="cd__h">
          {{ t('game.cards.skins', { n: skins.length }) }}
          <span v-if="skins.length > 1" class="pp-hint cd__hint">{{ t('game.cards.skinHint') }}</span>
        </h3>
        <div class="cd__skins">
          <button
            v-for="s in skins"
            :key="s"
            type="button"
            class="cd__skin"
            :class="{ 'is-on': s === shownSkin }"
            :aria-pressed="s === shownSkin"
            @click="preview = s"
          >
            <GameImage :src="idolCardArt(ctx.game.value, s, 'sq')" :alt="ctx.nameOf('idolCardSkin', s)" />
            <span class="cd__skin-name">{{ ctx.nameOf('idolCardSkin', s) || t('game.unnamed') }}</span>
          </button>
        </div>

        <h3 class="cd__h">{{ t('game.cards.records', { n: records.total }) }}</h3>
        <p v-if="records.error" class="pp-hint cd__err">{{ records.error }}</p>
        <p v-else-if="records.loading && !records.items.length" class="pp-hint">{{ t('game.memory.loading') }}</p>
        <p v-else-if="!records.items.length" class="pp-hint">{{ t('game.cards.noRecords') }}</p>
        <div v-else class="cd__records">
          <button v-for="m in records.items" :key="m.memoryId" type="button" class="cd__record" @click="emit('memory', m.memoryId)">
            <span class="pp-tag pp-tag--g pp-num">{{ gradeLabel(m.grade) }}</span>
            <span class="pp-num cd__score">{{ fmtNum(m.score) }}</span>
            <span class="pp-hint pp-num cd__stats">Vo {{ fmtNum(m.vocal) }} · Da {{ fmtNum(m.dance) }} · Vi {{ fmtNum(m.visual) }}</span>
            <span v-if="m.lent" class="pp-tag">{{ t('game.cards.lent') }}</span>
          </button>
        </div>
        <PPager
          v-if="records.loaded && records.total"
          :page="records.page"
          :total="records.total"
          :page-size="RECORDS"
          :disabled="records.loading"
          @update:page="(p) => void recordPages.load(p)"
        />
        <span class="pp-hint">{{ t('game.cards.idolReadonly') }}</span>
      </template>

      <PButton v-if="support" :disabled="support.lent" :loading="saving" @click="setRental">
        {{ support.lent ? t('game.cards.lent') : t('game.cards.setRental') }}
      </PButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import GameImage from '@/components/game/GameImage.vue'
import PButton from '@passport/components/PButton.vue'
import PPager from '@passport/components/PPager.vue'
import { listMyMemories, setMyRentalSupportCard } from '@passport/api/gakumas'
import { useUiStore } from '@passport/stores/ui'
import { errorText, fmtNum, fmtTime } from '@passport/utils/format'
import { offsetCursor, usePagedList } from '@passport/utils/paged'
import type { IdolCard, Memory, SupportCard } from '@/games/gakumas/types'
import type { MemoryCounts } from '@/types/v1/gakumas/types'
import { gradeLabel, idolCardArt, rarityLabel, supportCardArt, useGameContext } from './context'

const RECORDS = 20

const props = defineProps<{ idol?: IdolCard | null; support?: SupportCard | null }>()
const emit = defineEmits<{ memory: [memoryId: string]; changed: [] }>()

const { t } = useI18n()
const ctx = useGameContext()
const ui = useUiStore()

// 预览的皮肤（只在页面上切换，默认是卡当前的皮肤）
const preview = ref('')
const shownSkin = computed(() => preview.value || props.idol?.skinId || '')
watch(
  () => props.idol?.idolCardId,
  () => {
    preview.value = ''
  },
)
const id = computed(() => (props.idol ? shownSkin.value || props.idol.idolCardId : props.support?.supportCardId || ''))
const rarity = computed(() => props.idol?.rarity ?? props.support?.rarity ?? 0)
const createdTime = computed(() => props.idol?.createdTime ?? props.support?.createdTime ?? 0)

// 原画：偶像卡 9:16、支援卡 16:9（高清）
const art = computed(() =>
  props.idol ? idolCardArt(ctx.game.value, id.value, 'full') : supportCardArt(ctx.game.value, id.value, 'full'),
)

const name = computed(() => {
  if (props.idol) {
    return ctx.nameOf('idolCardSkin', id.value) || ctx.nameOf('idolCard', props.idol.idolCardId) || t('game.unnamed')
  }
  return ctx.nameOf('supportCard', id.value) || t('game.unnamed')
})
const chara = computed(() => (props.idol?.characterId ? ctx.nameOf('character', props.idol.characterId) : ''))

const tags = computed(() => {
  const out: string[] = []
  if (props.idol?.primaStella) out.push(t('game.cards.primaStella'))
  if (props.support?.lent) out.push(t('game.cards.lent'))
  return out
})

const facts = computed(() => {
  if (props.idol) {
    return [
      { k: t('game.cards.potential'), v: fmtNum(props.idol.potentialRank) },
      { k: t('game.cards.limit'), v: fmtNum(props.idol.levelLimitRank) },
      { k: t('game.cards.skinCount'), v: fmtNum(skins.value.length) },
    ]
  }
  if (props.support) {
    return [
      { k: t('game.cards.level'), v: fmtNum(props.support.level) },
      { k: t('game.cards.limit'), v: fmtNum(props.support.levelLimitRank) },
      { k: t('game.cards.stock'), v: fmtNum(props.support.stockQuantity) },
    ]
  }
  return []
})

// 持有的皮肤：当前皮肤在前
const skins = computed(() => {
  const c = props.idol
  if (!c) return []
  const list = c.skinIds?.length ? [...c.skinIds] : c.skinId ? [c.skinId] : []
  return list.sort((a, b) => Number(b === c.skinId) - Number(a === c.skinId))
})

// 这张偶像卡保留的培育记录（从新到旧，页码分页）
const recordPages = usePagedList<Memory, MemoryCounts>(async (page, size) => {
  const cardId = props.idol?.idolCardId || ''
  if (!cardId) return { items: [], total: 0 }
  return listMyMemories({
    serverId: ctx.serverId.value,
    produced: true,
    q: '',
    rarity: 0,
    grades: [],
    idolCardIds: [cardId],
    cursor: offsetCursor(page, size),
    limit: size,
  })
}, RECORDS)
const records = recordPages.list
watch(
  () => props.idol?.idolCardId || '',
  () => void recordPages.reload(),
  { immediate: true },
)

const saving = ref(false)
const setRental = async () => {
  const c = props.support
  if (!c || saving.value) return
  saving.value = true
  try {
    await setMyRentalSupportCard({ serverId: ctx.serverId.value, supportCardId: c.supportCardId })
    ui.toast(t('game.cards.rentalDone'), 'ok')
    emit('changed')
  } catch (e) {
    ui.toast(errorText(e), 'bad')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.cd {
  display: flex;
  flex-direction: column;
}
.cd__img {
  width: 100%;
  aspect-ratio: 16 / 9;
  object-fit: cover;
}
.cd--idol {
  flex-direction: row;
  align-items: flex-start;
}
.cd--idol .cd__img {
  flex: none;
  width: 42%;
  max-width: 300px;
  aspect-ratio: 9 / 16;
}
.cd--idol .cd__body {
  flex: 1;
  min-width: 0;
}
@media (max-width: 640px) {
  .cd--idol {
    flex-direction: column;
    align-items: stretch;
  }
  .cd--idol .cd__img {
    width: 100%;
    max-width: none;
    max-height: 56vh;
    object-position: 50% 20%;
  }
}
.cd__body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 18px 20px;
}
.cd__head {
  display: flex;
  align-items: center;
  gap: 10px;
}
.cd__name {
  flex: 1;
  min-width: 0;
  font-size: 20px;
  font-weight: 800;
}
.cd__rar {
  flex: none;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--m-s2);
  font-size: 12px;
  font-weight: 800;
}
.cd__rar--3 {
  background: var(--m-a);
  color: var(--m-aOn);
}
.cd__rar--2 {
  background: var(--m-a2Soft);
}
.cd__sub {
  word-break: break-all;
}
.cd__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.cd__dl {
  margin: 0;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}
.cd__fact {
  padding: 10px 12px;
  border-radius: 14px;
  background: var(--m-s2);
}
.cd__fact dd {
  margin: 0;
  font-size: 18px;
  font-weight: 800;
}
.cd__acq {
  font-size: 13px;
  color: var(--m-ink2);
}
.cd__h {
  margin: 6px 0 0;
  font-size: 14px;
  font-weight: 800;
}
.cd__skins {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(84px, 1fr));
  gap: 8px;
}
.cd__hint {
  margin-left: 8px;
  font-weight: 400;
}
.cd__skin {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  min-width: 0;
  padding: 6px;
  border-radius: 14px;
  border: 2px solid transparent;
  background: var(--m-s2);
  color: var(--m-ink);
  font: inherit;
  cursor: pointer;
}
.cd__skin:hover {
  border-color: var(--m-gLine);
}
.cd__skin.is-on {
  border-color: var(--m-g);
}
.cd__skin :deep(img) {
  width: 64px;
  height: 64px;
  border-radius: 10px;
}
.cd__skin-name {
  max-width: 100%;
  overflow: hidden;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.cd__records {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.cd__record {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border: 0;
  border-radius: 14px;
  background: var(--m-s2);
  color: var(--m-ink);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.cd__record:hover {
  box-shadow: inset 0 0 0 2px var(--m-gLine);
}
.cd__score {
  font-weight: 800;
}
.cd__stats {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.cd__err {
  color: var(--m-bad);
}
</style>
