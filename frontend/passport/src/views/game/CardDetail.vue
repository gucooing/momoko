<!-- 一张卡的详情（只读）：大图、名称、角色、养成数值、获得时间。 -->
<template>
  <div class="cd">
    <GameImage v-if="art" :src="art" :alt="name" class="cd__img" />
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
      <span class="pp-hint">{{ t('game.cards.readonly') }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import GameImage from '@/components/game/GameImage.vue'
import { fmtNum, fmtTime } from '@passport/utils/format'
import type { IdolCard, SupportCard } from '@/games/gakumas/types'
import { idolCardArt, rarityLabel, supportCardArt, useGameContext } from './context'

const props = defineProps<{ idol?: IdolCard | null; support?: SupportCard | null }>()

const { t } = useI18n()
const ctx = useGameContext()

const id = computed(() => (props.idol ? props.idol.skinId || props.idol.idolCardId : props.support?.supportCardId || ''))
const rarity = computed(() => props.idol?.rarity ?? props.support?.rarity ?? 0)
const createdTime = computed(() => props.idol?.createdTime ?? props.support?.createdTime ?? 0)

const art = computed(() =>
  props.idol ? idolCardArt(ctx.game.value, id.value, 'land') : supportCardArt(ctx.game.value, id.value, 'land'),
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
      { k: t('game.cards.pieces'), v: fmtNum(props.idol.pieces) },
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
  object-position: 50% 25%;
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
</style>
