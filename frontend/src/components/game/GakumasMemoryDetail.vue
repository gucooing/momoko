<!-- 学マス 一条回忆的详情（管理端与用户通行证共用，只负责展示），版式照游戏的回忆详情与培育记录：
     顶部是回忆照片（没有照片时是游戏图片）、评价与评价值、偶像卡与标记；
     「培育」可获得的技能卡（图标、获得时机）与能力；「舞台」能力值评价、P 道具与技能卡（图标）；
     「培育记录」（保留的培育结果才有）剧本与难度、日期、制作人与 PLv、评价与评价值、偶像卡、能力值评价与成长率、体力，
     编成的支援卡与回忆（点开看详情，自己的回忆可打开完整详情，发出 open-memory），最终持有的 P 道具与技能卡。
     名称与效果文字取前端名称表（当前语言的译名 → 游戏原文），图标与卡的稀有度取调用方的目录查询（lookup）；
     照片由调用方的 photo 读取；文案是组件自己的三种语言。 -->
<template>
  <div class="gmd">
    <header class="gmd__hero">
      <span class="gmd__photo">
        <GameImage :src="memoryImage(detail.imagePath, detail.assetId)" :alt="cardName" eager />
        <span class="gmd__photo-grade">{{ gradeLabel(detail.grade) }}</span>
      </span>
      <div class="gmd__info">
        <div class="gmd__line">
          <span class="gmd__grade">{{ gradeLabel(detail.grade) }}</span>
          <span v-if="detail.score" class="gmd__score">{{ num(detail.score) }}<small>Pt</small></span>
        </div>
        <div class="gmd__idol">
          <GameImage :src="skinImage(detail.skinId || detail.idolCardId)" :alt="cardName" class="gmd__sq" />
          <span class="gmd__idol-text">
            <span class="gmd__name">{{ cardName || '—' }}</span>
            <span v-if="charaName" class="gmd__dim">{{ charaName }}</span>
          </span>
        </div>
        <div class="gmd__tags">
          <span v-if="detail.power" class="gmd__tag">{{ t('power', { n: num(detail.power) }) }}</span>
          <span v-if="detail.lent" class="gmd__tag is-on">{{ t('lent') }}</span>
          <span v-if="detail.protected" class="gmd__tag">{{ t('protected') }}</span>
          <span v-if="!detail.produceId" class="gmd__tag">{{ detail.researchId ? t('research') : t('other') }}</span>
          <span class="gmd__tag">{{ t('potential', { n: detail.potentialRank }) }}</span>
          <span class="gmd__tag">{{ t('limit', { n: detail.levelLimitRank }) }}</span>
          <span v-if="detail.primaStella" class="gmd__tag">Prima Stella</span>
        </div>
        <span class="gmd__dim gmd__small">{{ t('created', { time: time(detail.createdTime) }) }}</span>
      </div>
    </header>

    <nav class="gmd__tabs" role="tablist">
      <button
        v-for="x in tabs"
        :key="x"
        type="button"
        role="tab"
        class="gmd__tab"
        :class="{ 'is-on': tab === x }"
        :aria-selected="tab === x"
        @click="tab = x"
      >
        {{ t(`tab.${x}`) }}
      </button>
    </nav>

    <!-- 培育：可获得的技能卡与能力 -->
    <section v-if="tab === 'produce'" class="gmd__pane">
      <div class="gmd__h">
        <span>{{ t('produceCard') }}</span>
        <span v-if="phaseText" class="gmd__phase">{{ phaseText }}</span>
      </div>
      <div v-if="detail.produceCard?.cardId" class="gmd__pcard">
        <span class="gmd__card-icon is-large">
          <GameImage :src="cardIcon(detail.produceCard.cardId)" :alt="cardTitle(detail.produceCard)" />
          <span v-if="detail.produceCard.upgradeCount > 0" class="gmd__plus">+</span>
        </span>
        <span class="gmd__pcard-name">{{ cardTitle(detail.produceCard) }}</span>
      </div>
      <p v-else class="gmd__none">{{ t('none') }}</p>

      <div class="gmd__h">{{ t('abilities') }}</div>
      <ul v-if="detail.abilities.length" class="gmd__rows">
        <li v-for="(a, i) in detail.abilities" :key="`${a.abilityId}-${i}`" class="gmd__ability">
          <span class="gmd__lv">Lv{{ a.level }}</span>
          <span class="gmd__text">{{ abilityText(a.abilityId, a.level) }}</span>
        </li>
      </ul>
      <p v-else class="gmd__none">{{ t('none') }}</p>
    </section>

    <!-- 舞台：能力值、P 道具、技能卡 -->
    <section v-else-if="tab === 'stage'" class="gmd__pane">
      <div class="gmd__stats">
        <div v-for="s in stageStats" :key="s.key" class="gmd__stat" :class="`is-${s.key}`">
          <span v-if="s.grade" class="gmd__stat-grade">{{ gradeLabel(s.grade) }}</span>
          <span class="gmd__stat-body">
            <span class="gmd__statk">{{ s.label }}</span>
            <span class="gmd__statv">{{ num(s.value) }}</span>
          </span>
        </div>
      </div>

      <div class="gmd__h">{{ t('items') }}</div>
      <div v-if="detail.itemIds.length" class="gmd__items">
        <div v-for="(id, i) in detail.itemIds" :key="`${id}-${i}`" class="gmd__item">
          <GameImage :src="itemIcon(id)" :alt="itemName(id)" class="gmd__item-icon" />
          <span class="gmd__item-text">
            <span class="gmd__strong">{{ itemName(id) }}</span>
            <span v-if="itemText(id)" class="gmd__dim gmd__small">{{ itemText(id) }}</span>
          </span>
        </div>
      </div>
      <p v-else class="gmd__none">{{ t('none') }}</p>

      <div class="gmd__h">{{ t('cards') }}</div>
      <div v-if="detail.cards.length" class="gmd__cards">
        <div v-for="(c, i) in detail.cards" :key="`${c.cardId}-${i}`" class="gmd__card">
          <span class="gmd__card-icon">
            <GameImage :src="cardIcon(c.cardId)" :alt="cardTitle(c)" />
            <span v-if="c.upgradeCount > 0" class="gmd__plus">+</span>
          </span>
          <span class="gmd__card-name">{{ cardTitle(c) }}</span>
        </div>
      </div>
      <p v-else class="gmd__none">{{ t('none') }}</p>
    </section>

    <!-- 培育记录 -->
    <section v-else-if="record" class="gmd__pane gmd__record">
      <div class="gmd__rec-head">
        <span class="gmd__strong">{{ produceGroupName || produceName || '—' }}</span>
        <span v-if="produceGroupName && produceName" class="gmd__badge">{{ produceName }}</span>
        <span class="gmd__spacer" />
        <span class="gmd__dim">{{ time(record.clearedTime) }}</span>
      </div>

      <div class="gmd__rec-result">
        <div class="gmd__rec-score">
          <div class="gmd__line">
            <span class="gmd__strong">{{ record.userName || '—' }}</span>
            <span class="gmd__dim">PLv {{ record.producerLevel }}</span>
          </div>
          <div class="gmd__line">
            <span class="gmd__big-grade">{{ gradeLabel(record.grade) }}</span>
            <span class="gmd__score">{{ num(record.score) }}<small>Pt</small></span>
          </div>
        </div>
        <span class="gmd__rec-photo">
          <GameImage :src="memoryImage(detail.imagePath, detail.assetId)" :alt="cardName" />
          <span class="gmd__photo-grade is-small">{{ gradeLabel(detail.grade) }}</span>
        </span>
      </div>

      <div class="gmd__rec-idol">
        <GameImage :src="skinImage(record.skinId || record.idolCardId)" :alt="recordCardName" class="gmd__rec-sq" />
        <div class="gmd__rec-idol-main">
          <div class="gmd__line">
            <span v-if="recordRarity" class="gmd__rarity" :class="`is-${recordRarity}`">{{ rarityLabel(recordRarity) }}</span>
            <span class="gmd__idol-text">
              <span class="gmd__dim gmd__small">{{ recordCardName || '—' }}</span>
              <span class="gmd__strong">{{ recordCharaName }}</span>
            </span>
          </div>
          <div class="gmd__rec-stats">
            <div v-for="s in recordStats" :key="s.key" class="gmd__rstat" :class="`is-${s.key}`">
              <span class="gmd__statk">{{ s.label }}</span>
              <span class="gmd__line">
                <span v-if="s.grade" class="gmd__stat-grade is-small">{{ gradeLabel(s.grade) }}</span>
                <span class="gmd__statv">{{ num(s.value) }}</span>
              </span>
              <span v-if="s.growth" class="gmd__growth">↑{{ percent(s.growth) }}%</span>
            </div>
          </div>
          <div class="gmd__tags">
            <span class="gmd__tag">{{ t('potential', { n: record.potentialRank }) }}</span>
            <span class="gmd__tag">{{ t('limit', { n: record.levelLimitRank }) }}</span>
            <span v-if="record.primaStella" class="gmd__tag">Prima Stella</span>
            <span v-if="record.voteCount" class="gmd__tag">{{ t('votes', { n: num(record.voteCount) }) }}</span>
            <span v-if="record.star" class="gmd__tag">{{ t('stars', { n: num(record.star) }) }}</span>
          </div>
        </div>
      </div>

      <div class="gmd__h">{{ t('deck') }}</div>
      <div v-if="record.supportCards.length || record.memories.length" class="gmd__deck">
        <div class="gmd__deck-col">
          <span class="gmd__dim gmd__small">{{ t('supportCards') }}</span>
          <div class="gmd__deck-grid">
            <button
              v-for="(s, i) in record.supportCards"
              :key="`s-${s.supportCardId}-${i}`"
              type="button"
              class="gmd__slot"
              @click="peek = { kind: 'support', support: s }"
            >
              <GameImage :src="supportImage(s.supportCardId, 'land')" :alt="lookup.name('supportCard', s.supportCardId)" />
              <span class="gmd__slot-lv">Lv{{ s.level }}</span>
              <span v-if="s.rental" class="gmd__slot-rental">{{ t('rental') }}</span>
            </button>
          </div>
        </div>
        <div v-if="record.memories.length" class="gmd__deck-col is-memories">
          <span class="gmd__dim gmd__small">{{ t('memories') }}</span>
          <div class="gmd__deck-grid">
            <button
              v-for="(m, i) in record.memories"
              :key="`m-${m.memoryId}-${i}`"
              type="button"
              class="gmd__slot"
              @click="peek = { kind: 'memory', memory: m }"
            >
              <GameImage :src="memoryImage(m.imagePath, m.assetId)" :alt="t('memory')" />
              <span class="gmd__slot-grade">{{ gradeLabel(m.grade) }}</span>
              <span v-if="m.rental" class="gmd__slot-rental">{{ t('rental') }}</span>
            </button>
          </div>
        </div>
      </div>
      <p v-else class="gmd__none">{{ t('none') }}</p>

      <div class="gmd__h">{{ t('finalItems') }}</div>
      <div v-if="record.itemIds.length" class="gmd__items">
        <div v-for="(id, i) in record.itemIds" :key="`${id}-${i}`" class="gmd__item">
          <GameImage :src="itemIcon(id)" :alt="itemName(id)" class="gmd__item-icon" />
          <span class="gmd__item-text">
            <span class="gmd__strong">{{ itemName(id) }}</span>
            <span v-if="itemText(id)" class="gmd__dim gmd__small">{{ itemText(id) }}</span>
          </span>
        </div>
      </div>
      <p v-else class="gmd__none">{{ t('none') }}</p>

      <div class="gmd__h">{{ t('finalCards', { n: record.cards.length }) }}</div>
      <div v-if="record.cards.length" class="gmd__cards">
        <div v-for="(c, i) in record.cards" :key="`${c.cardId}-${i}`" class="gmd__card">
          <span class="gmd__card-icon">
            <GameImage :src="cardIcon(c.cardId)" :alt="cardTitle(c)" />
            <span v-if="c.upgradeCount > 0" class="gmd__plus">+</span>
          </span>
          <span class="gmd__card-name">{{ cardTitle(c) }}</span>
          <span v-if="c.fromMemory" class="gmd__from">{{ t('fromMemory') }}</span>
        </div>
      </div>
      <p v-else class="gmd__none">{{ t('none') }}</p>
    </section>

    <!-- 编成的支援卡 / 回忆详情 -->
    <Teleport to="body">
      <div v-if="peek" class="gmd-peek" @mousedown.self="peek = null">
        <div class="gmd-peek__panel" role="dialog" aria-modal="true">
          <button type="button" class="gmd-peek__x" :aria-label="t('close')" @click="peek = null">×</button>
          <template v-if="peek.kind === 'support'">
            <GameImage :src="supportImage(peek.support.supportCardId, 'full')" class="gmd-peek__art" eager />
            <div class="gmd-peek__body">
              <div class="gmd__line">
                <span v-if="supportRarity(peek.support.supportCardId)" class="gmd__rarity" :class="`is-${supportRarity(peek.support.supportCardId)}`">
                  {{ rarityLabel(supportRarity(peek.support.supportCardId)) }}
                </span>
                <span class="gmd__name">{{ lookup.name('supportCard', peek.support.supportCardId) || '—' }}</span>
              </div>
              <div class="gmd__tags">
                <span class="gmd__tag">Lv{{ peek.support.level }}</span>
                <span class="gmd__tag">{{ t('limit', { n: peek.support.levelLimitRank }) }}</span>
                <span v-if="peek.support.rental" class="gmd__tag is-on">{{ t('rental') }}</span>
              </div>
            </div>
          </template>
          <template v-else>
            <span class="gmd__photo is-peek">
              <GameImage :src="memoryImage(peek.memory.imagePath, peek.memory.assetId)" eager />
              <span class="gmd__photo-grade">{{ gradeLabel(peek.memory.grade) }}</span>
            </span>
            <div class="gmd-peek__body">
              <div class="gmd__idol">
                <GameImage :src="skinImage(peek.memory.skinId || peek.memory.idolCardId)" class="gmd__sq" />
                <span class="gmd__idol-text">
                  <span class="gmd__name">{{ skinName(peek.memory.skinId, peek.memory.idolCardId) || '—' }}</span>
                  <span class="gmd__dim">{{ charaOfSkin(peek.memory.skinId) }}</span>
                </span>
              </div>
              <div class="gmd__tags">
                <span v-if="peek.memory.power" class="gmd__tag">{{ t('power', { n: num(peek.memory.power) }) }}</span>
                <span v-if="peek.memory.rental" class="gmd__tag is-on">{{ t('rental') }}</span>
              </div>
              <button v-if="!peek.memory.rental" type="button" class="gmd-peek__open" @click="openMemory(peek.memory.memoryId)">
                {{ t('openMemory') }}
              </button>
            </div>
          </template>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import GameImage from './GameImage.vue'
import { gakumas, gameCatalogImage, lookupGameName } from '@/games'
import { gradeLabel, rarityLabel, type GakumasDetailLookup } from '@/games/gakumas/display'
import type { MemoryDetail, RecordMemory, RecordSupportCard } from '@/types/v1/gakumas/types'

const props = defineProps<{
  detail: MemoryDetail
  game: string
  lookup: GakumasDetailLookup
  /** Unix 毫秒 → 显示的时间（各端按自己的时区与格式） */
  time: (ms: number) => string
  /** 读取一张回忆照片（image_path）→ 可直接用作图片地址的 URL */
  photo: (path: string) => Promise<string>
}>()
const emit = defineEmits<{ 'open-memory': [memoryId: string] }>()

type Tab = 'produce' | 'stage' | 'record'
type Peek = { kind: 'support'; support: RecordSupportCard } | { kind: 'memory'; memory: RecordMemory }

const messages = {
  'zh-CN': {
    tab: { produce: '培育', stage: '舞台', record: '培育记录' },
    power: '编成力 {n}',
    lent: '援助中',
    protected: '保护',
    research: '研究活动',
    other: '非培育',
    potential: '才能开花 {n}',
    limit: '上限解放 {n}',
    created: '获得于 {time}',
    produceCard: '可获得技能卡',
    phase1: '培育开始时获得',
    phase2: '通过期中考试后获得',
    abilities: '能力',
    items: 'P道具',
    cards: '技能卡',
    none: '无',
    unknown: '未知效果',
    vocal: '歌唱',
    dance: '舞蹈',
    visual: '视觉',
    stamina: '体力',
    votes: '票数 {n}',
    stars: '星 {n}',
    deck: '编成支援卡／回忆',
    supportCards: '支援卡',
    memories: '回忆',
    rental: '租借',
    memory: '回忆',
    finalItems: '最终持有P道具',
    finalCards: '最终持有技能卡（{n}）',
    fromMemory: '回忆',
    openMemory: '查看回忆详情',
    close: '关闭',
  },
  'zh-HK': {
    tab: { produce: '培育', stage: '舞台', record: '培育記錄' },
    power: '編成力 {n}',
    lent: '援助中',
    protected: '保護',
    research: '研究活動',
    other: '非培育',
    potential: '才能開花 {n}',
    limit: '上限解放 {n}',
    created: '獲得於 {time}',
    produceCard: '可獲得技能卡',
    phase1: '培育開始時獲得',
    phase2: '通過期中考試後獲得',
    abilities: '能力',
    items: 'P道具',
    cards: '技能卡',
    none: '無',
    unknown: '未知效果',
    vocal: '歌唱',
    dance: '舞蹈',
    visual: '視覺',
    stamina: '體力',
    votes: '票數 {n}',
    stars: '星 {n}',
    deck: '編成支援卡／回憶',
    supportCards: '支援卡',
    memories: '回憶',
    rental: '租借',
    memory: '回憶',
    finalItems: '最終持有P道具',
    finalCards: '最終持有技能卡（{n}）',
    fromMemory: '回憶',
    openMemory: '查看回憶詳情',
    close: '關閉',
  },
  'en-US': {
    tab: { produce: 'Produce', stage: 'Stage', record: 'Produce record' },
    power: 'Power {n}',
    lent: 'Support',
    protected: 'Protected',
    research: 'Research',
    other: 'Not produced',
    potential: 'Bloom {n}',
    limit: 'Limit break {n}',
    created: 'Obtained {time}',
    produceCard: 'Obtainable skill card',
    phase1: 'Obtained when producing starts',
    phase2: 'Obtained after the midterm exam',
    abilities: 'Abilities',
    items: 'P items',
    cards: 'Skill cards',
    none: 'None',
    unknown: 'Unknown effect',
    vocal: 'Vocal',
    dance: 'Dance',
    visual: 'Visual',
    stamina: 'Stamina',
    votes: 'Votes {n}',
    stars: 'Stars {n}',
    deck: 'Support cards / memories',
    supportCards: 'Support cards',
    memories: 'Memories',
    rental: 'Rental',
    memory: 'Memory',
    finalItems: 'Final P items',
    finalCards: 'Final skill cards ({n})',
    fromMemory: 'Memory',
    openMemory: 'Open memory details',
    close: 'Close',
  },
}

const { t, locale } = useI18n({ useScope: 'local', inheritLocale: true, messages })

const tab = ref<Tab>('produce')
const peek = ref<Peek | null>(null)
const record = computed(() => props.detail.record)
const tabs = computed<Tab[]>(() => (record.value ? ['produce', 'stage', 'record'] : ['produce', 'stage']))
watch(
  () => props.detail.memoryId,
  () => {
    tab.value = 'produce'
    peek.value = null
  },
)

const lang = () => String(locale.value)
const num = (n: number) => new Intl.NumberFormat(lang()).format(n || 0)
const percent = (permil: number) => (permil / 10).toFixed(1)

// ---------- 图片 ----------
const skinImage = (id: string) => (id ? gameCatalogImage(props.game, 'idolCardSkin', { id }) : '')
const supportImage = (id: string, variant: 'sq' | 'land' | 'full' = 'sq') =>
  id && props.game === 'gakumas' ? gakumas.supportCardImage(id, variant) : ''
// 技能卡图标：按偶像区分图片的卡用这条回忆的偶像（角色）
const cardIcon = (cardId: string) => {
  if (props.game !== 'gakumas') return ''
  const e = props.lookup.entry('produceCard', cardId)
  return gakumas.skillCardImage(e?.assetId || '', e?.characterAsset ? props.detail.characterId : '')
}
const itemIcon = (id: string) => (props.game === 'gakumas' ? gakumas.produceItemImage(props.lookup.entry('produceItem', id)?.assetId || '') : '')

// 回忆照片按 image_path 读取一次；没有照片的回忆显示游戏图片（asset_id）
const photos = reactive(new Map<string, string>())
const memoryImage = (imagePath: string, assetId: string) => {
  if (imagePath) {
    if (!photos.has(imagePath)) {
      photos.set(imagePath, '')
      props
        .photo(imagePath)
        .then((url) => photos.set(imagePath, url))
        .catch(() => undefined)
    }
    return photos.get(imagePath) || ''
  }
  return assetId && props.game === 'gakumas' ? gakumas.memoryAssetImage(assetId) : ''
}

// ---------- 名称与文字（名称表：当前语言的译名 → 游戏原文；名称表没有时用目录） ----------
const tableText = (kind: string, key: string) => lookupGameName(props.game, lang(), kind, key)

const skinName = (skinId: string, idolCardId: string) =>
  (skinId && props.lookup.name('idolCardSkin', skinId)) || props.lookup.name('idolCard', idolCardId)
const charaOfSkin = (skinId: string) => {
  const chara = skinId ? props.lookup.entry('idolCardSkin', skinId)?.characterId : ''
  return chara ? props.lookup.name('character', chara) : ''
}
const cardName = computed(() => skinName(props.detail.skinId, props.detail.idolCardId))
const charaName = computed(() => (props.detail.characterId ? props.lookup.name('character', props.detail.characterId) : ''))
const recordCardName = computed(() => (record.value ? skinName(record.value.skinId, record.value.idolCardId) : ''))
const recordCharaName = computed(() => charaName.value)
const recordRarity = computed(() => (record.value ? props.lookup.entry('idolCardSkin', record.value.skinId)?.rarity || 0 : 0))
const supportRarity = (id: string) => props.lookup.entry('supportCard', id)?.rarity || 0

const produceName = computed(() => (record.value?.produceId ? props.lookup.name('produce', record.value.produceId) : ''))
const produceGroupName = computed(() => {
  const group = record.value?.produceId ? props.lookup.entry('produce', record.value.produceId)?.parentId : ''
  return group ? props.lookup.name('produceGroup', group) : ''
})

const phaseText = computed(() => {
  const p = props.detail.produceCardPhaseType
  return p === 1 || p === 2 ? t(`phase${p}`) : ''
})

/** 技能卡名：强化过的取强化后的名称（名称表 <id>+<次数>，名称里带 +）。 */
const cardTitle = (c: { cardId: string; upgradeCount: number }) => {
  const key = c.upgradeCount > 0 ? `${c.cardId}+${c.upgradeCount}` : c.cardId
  const local = tableText('produceCard', key)
  if (local) return local
  const e = props.lookup.entry('produceCard', c.cardId)
  if (local === undefined || e === undefined) return '…'
  if (c.upgradeCount <= 0) return e?.name || '—'
  return e?.upgradedName || `${e?.name || '—'}${'+'.repeat(c.upgradeCount)}`
}

const itemName = (id: string) => {
  const local = tableText('produceItem', id)
  if (local) return local
  const e = props.lookup.entry('produceItem', id)
  return local === undefined || e === undefined ? '…' : e?.name || '—'
}
const itemText = (id: string) => tableText('produceItemText', id) || props.lookup.entry('produceItem', id)?.description || ''

const abilityText = (id: string, level: number) => {
  const local = tableText('memoryAbilityText', `${id}@${level}`)
  if (local) return local
  const e = props.lookup.entry('memoryAbility', id)
  if (local === undefined || e === undefined) return '…'
  return e?.levels?.find((l) => l.level === level)?.description || t('unknown')
}

const stageStats = computed(() => [
  { key: 'vo', label: t('vocal'), value: props.detail.vocal, grade: props.detail.vocalGrade },
  { key: 'da', label: t('dance'), value: props.detail.dance, grade: props.detail.danceGrade },
  { key: 'vi', label: t('visual'), value: props.detail.visual, grade: props.detail.visualGrade },
  { key: 'st', label: t('stamina'), value: props.detail.stamina, grade: 0 },
])

const recordStats = computed(() => {
  const r = record.value
  if (!r) return []
  return [
    { key: 'vo', label: t('vocal'), value: r.vocal, grade: r.vocalGrade, growth: r.vocalGrowthRatePermil },
    { key: 'da', label: t('dance'), value: r.dance, grade: r.danceGrade, growth: r.danceGrowthRatePermil },
    { key: 'vi', label: t('visual'), value: r.visual, grade: r.visualGrade, growth: r.visualGrowthRatePermil },
    { key: 'st', label: t('stamina'), value: r.maxStamina, grade: 0, growth: 0 },
  ]
})

const openMemory = (id: string) => {
  peek.value = null
  emit('open-memory', id)
}
</script>

<style scoped>
.gmd {
  --gmd-vo: #ec4f8c;
  --gmd-da: #3d8fe8;
  --gmd-vi: #f0a323;
  --gmd-st: #3fbf7f;
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
  color: var(--m-ink);
}
.gmd__hero {
  display: flex;
  gap: 16px;
  align-items: flex-start;
}
.gmd__photo {
  position: relative;
  flex: none;
  width: 46%;
  max-width: 320px;
  aspect-ratio: 16 / 9;
  border-radius: 12px;
  overflow: hidden;
  background: var(--m-s2);
}
.gmd__photo :deep(img) {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.gmd__photo.is-peek {
  width: 100%;
  max-width: none;
  border-radius: 0;
}
.gmd__photo-grade {
  position: absolute;
  left: 8px;
  top: 6px;
  font-size: 26px;
  font-weight: 900;
  font-style: italic;
  color: #fff;
  -webkit-text-stroke: 1px #e8447a;
  text-shadow: 0 2px 6px rgb(0 0 0 / 35%);
}
.gmd__photo-grade.is-small {
  font-size: 18px;
}
.gmd__info {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}
.gmd__line {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.gmd__grade {
  padding: 1px 10px;
  border-radius: 999px;
  background: var(--m-p);
  color: var(--m-pOn);
  font-weight: 800;
  font-size: 14px;
}
.gmd__score {
  font-size: 22px;
  font-weight: 900;
  font-variant-numeric: tabular-nums;
}
.gmd__score small {
  margin-left: 2px;
  font-size: 13px;
  font-weight: 700;
}
.gmd__idol {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.gmd__idol-text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.gmd__sq {
  flex: none;
  width: 48px;
  height: 48px;
  border-radius: 10px;
  object-fit: cover;
  background: var(--m-s2);
}
.gmd__name {
  font-size: 16px;
  font-weight: 800;
  word-break: break-word;
}
.gmd__strong {
  font-weight: 700;
}
.gmd__dim {
  color: var(--m-ink2);
}
.gmd__small {
  font-size: 12px;
}
.gmd__spacer {
  flex: 1;
}
.gmd__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.gmd__tag {
  padding: 1px 8px;
  border-radius: 999px;
  background: var(--m-s2);
  font-size: 12px;
  white-space: nowrap;
}
.gmd__tag.is-on {
  background: var(--m-pSoft);
  color: var(--m-pInk);
  font-weight: 700;
}
.gmd__tabs {
  display: flex;
  gap: 4px;
  padding: 4px;
  border-radius: 999px;
  background: var(--m-s2);
}
.gmd__tab {
  flex: 1;
  padding: 6px 10px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--m-ink2);
  font: inherit;
  font-weight: 700;
  cursor: pointer;
}
.gmd__tab.is-on {
  background: var(--m-p);
  color: var(--m-pOn);
}
.gmd__pane {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
}
.gmd__h {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 6px;
  font-size: 14px;
  font-weight: 800;
  color: var(--m-ink2);
}
.gmd__phase {
  padding: 2px 10px;
  border-radius: 999px;
  background: #e8447a;
  color: #fff;
  font-size: 12px;
}
.gmd__none {
  margin: 0;
  color: var(--m-ink2);
  font-size: 13px;
}
.gmd__pcard {
  display: flex;
  align-items: center;
  gap: 14px;
}
.gmd__pcard-name {
  font-size: 17px;
  font-weight: 800;
}
.gmd__card-icon {
  position: relative;
  display: block;
  width: 64px;
  height: 64px;
  border-radius: 10px;
  overflow: hidden;
  background: var(--m-s2);
}
.gmd__card-icon.is-large {
  flex: none;
  width: 96px;
  height: 96px;
}
.gmd__card-icon :deep(img) {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.gmd__plus {
  position: absolute;
  right: 3px;
  top: 1px;
  color: #ff5a5a;
  font-size: 18px;
  font-weight: 900;
  text-shadow: 0 0 3px #fff;
}
.gmd__rows {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin: 0;
  padding: 0;
  list-style: none;
}
.gmd__ability {
  display: flex;
  gap: 10px;
  align-items: baseline;
  padding: 10px 12px;
  border-radius: 12px;
  background: var(--m-s2);
}
.gmd__lv {
  flex: none;
  font-weight: 800;
  color: var(--m-pInk);
  font-variant-numeric: tabular-nums;
}
.gmd__text {
  min-width: 0;
  font-size: 13px;
  white-space: pre-line;
}
.gmd__stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
}
.gmd__stat {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-radius: 12px;
  background: var(--m-s2);
}
.gmd__stat-body {
  display: flex;
  flex-direction: column;
}
.gmd__stat-grade {
  font-size: 26px;
  font-weight: 900;
  font-style: italic;
  line-height: 1;
  color: var(--gmd-c, var(--m-ink));
}
.gmd__stat-grade.is-small {
  font-size: 20px;
}
.gmd__statk {
  font-size: 12px;
  font-weight: 700;
  color: var(--gmd-c, var(--m-ink2));
}
.gmd__statv {
  font-size: 18px;
  font-weight: 800;
  font-variant-numeric: tabular-nums;
}
.is-vo {
  --gmd-c: var(--gmd-vo);
}
.is-da {
  --gmd-c: var(--gmd-da);
}
.is-vi {
  --gmd-c: var(--gmd-vi);
}
.is-st {
  --gmd-c: var(--gmd-st);
}
.gmd__items {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.gmd__item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  border-radius: 12px;
  background: var(--m-s2);
}
.gmd__item-icon {
  flex: none;
  width: 48px;
  height: 48px;
  border-radius: 10px;
  object-fit: contain;
}
.gmd__item-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.gmd__cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(84px, 1fr));
  gap: 10px;
}
.gmd__card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  min-width: 0;
}
.gmd__card-name {
  max-width: 100%;
  overflow: hidden;
  font-size: 12px;
  text-align: center;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gmd__from {
  padding: 0 6px;
  border-radius: 999px;
  background: var(--m-pSoft);
  color: var(--m-pInk);
  font-size: 11px;
}
.gmd__record {
  gap: 12px;
}
.gmd__rec-head {
  display: flex;
  align-items: center;
  gap: 10px;
  padding-bottom: 8px;
  border-bottom: 2px solid var(--m-line);
}
.gmd__badge {
  padding: 1px 12px;
  border-radius: 999px;
  background: linear-gradient(90deg, #7aa2ff, #c48cff);
  color: #fff;
  font-size: 12px;
  font-weight: 700;
}
.gmd__rec-result {
  display: flex;
  align-items: center;
  gap: 14px;
}
.gmd__rec-score {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}
.gmd__big-grade {
  font-size: 34px;
  font-weight: 900;
  font-style: italic;
  line-height: 1;
  color: #e8447a;
}
.gmd__rec-photo {
  position: relative;
  flex: none;
  width: 42%;
  max-width: 260px;
  aspect-ratio: 16 / 9;
  border-radius: 10px;
  overflow: hidden;
  background: var(--m-s2);
}
.gmd__rec-photo :deep(img) {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.gmd__rec-idol {
  display: flex;
  gap: 14px;
  padding: 12px;
  border-radius: 14px;
  background: var(--m-s2);
}
.gmd__rec-sq {
  flex: none;
  width: 92px;
  height: 92px;
  border-radius: 10px;
  object-fit: cover;
}
.gmd__rec-idol-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}
.gmd__rarity {
  flex: none;
  padding: 0 8px;
  border-radius: 6px;
  background: var(--m-s);
  font-size: 15px;
  font-weight: 900;
  font-style: italic;
}
.gmd__rarity.is-3 {
  background: linear-gradient(90deg, #5ad1ff, #ff8ad1, #ffd36b);
  color: #fff;
}
.gmd__rarity.is-2 {
  background: #f3c544;
  color: #3b2a00;
}
.gmd__rec-stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
}
.gmd__rstat {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.gmd__rstat.is-st {
  padding-left: 10px;
  border-left: 2px solid var(--m-line);
}
.gmd__growth {
  font-size: 12px;
  font-weight: 700;
  color: var(--gmd-c, var(--m-ink2));
}
.gmd__deck {
  display: flex;
  flex-wrap: wrap;
  gap: 12px 14px;
}
/* 编成的卡与回忆是固定大小的小图块，一行排满再换行（不随弹窗宽度放大） */
.gmd__deck-col {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
  flex: 1 1 300px;
}
.gmd__deck-col.is-memories {
  flex: 0 1 auto;
  padding-left: 14px;
  border-left: 2px solid var(--m-line);
}
.gmd__deck-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, 136px);
  gap: 8px;
}
.gmd__slot {
  position: relative;
  display: block;
  padding: 0;
  border: 0;
  border-radius: 10px;
  overflow: hidden;
  aspect-ratio: 16 / 9;
  background: var(--m-s2);
  cursor: pointer;
}
.gmd__slot :deep(img) {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.gmd__slot:hover {
  box-shadow: 0 0 0 2px var(--m-p);
}
.gmd__slot-lv {
  position: absolute;
  left: 6px;
  bottom: 2px;
  color: #fff;
  font-size: 14px;
  font-weight: 900;
  text-shadow: 0 1px 3px rgb(0 0 0 / 70%);
}
.gmd__slot-grade {
  position: absolute;
  left: 6px;
  top: 2px;
  color: #fff;
  font-size: 14px;
  font-weight: 900;
  font-style: italic;
  text-shadow: 0 1px 3px rgb(0 0 0 / 70%);
}
.gmd__slot-rental {
  position: absolute;
  right: 0;
  top: 0;
  padding: 0 8px;
  border-bottom-left-radius: 8px;
  background: #ec4f8c;
  color: #fff;
  font-size: 11px;
  font-weight: 800;
}
.gmd-peek {
  position: fixed;
  inset: 0;
  z-index: 2400;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: rgb(11 18 32 / 45%);
}
.gmd-peek__panel {
  position: relative;
  width: 100%;
  max-width: 640px;
  border-radius: 16px;
  overflow: hidden;
  background: var(--m-s);
  color: var(--m-ink);
  box-shadow: 0 12px 40px rgb(0 0 0 / 35%);
}
.gmd-peek__x {
  position: absolute;
  right: 8px;
  top: 6px;
  z-index: 1;
  width: 30px;
  height: 30px;
  border: 0;
  border-radius: 50%;
  background: rgb(0 0 0 / 45%);
  color: #fff;
  font-size: 18px;
  cursor: pointer;
}
.gmd-peek__art {
  display: block;
  width: 100%;
  aspect-ratio: 16 / 9;
  object-fit: cover;
}
.gmd-peek__body {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px 16px 16px;
}
.gmd-peek__open {
  align-self: flex-start;
  padding: 6px 14px;
  border: 0;
  border-radius: 999px;
  background: var(--m-p);
  color: var(--m-pOn);
  font: inherit;
  font-weight: 700;
  cursor: pointer;
}
@media (max-width: 600px) {
  .gmd__hero,
  .gmd__rec-result {
    flex-direction: column;
    align-items: stretch;
  }
  .gmd__photo,
  .gmd__rec-photo {
    width: 100%;
    max-width: none;
  }
  .gmd__stats,
  .gmd__rec-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .gmd__deck-grid {
    grid-template-columns: repeat(auto-fill, minmax(104px, 1fr));
  }
  .gmd__deck-col.is-memories {
    flex-basis: 100%;
    padding-left: 0;
    border-left: 0;
  }
}
</style>
