<!-- 编辑一个活动（运营管理里点日历上的活动直接打开）：活动日程里的活动（Home.json）的全部设置，剧情活动再加上它的内容
     （StoryEvent.json：画面、规则、加成、积分奖励、活动页角色、社团任务）；或研究活动（Research.json）的全部设置。 -->
<template>
  <ItemDialog
    :model-value="modelValue"
    :title="title"
    :width="900"
    :state="item ? 'ok' : before ? 'removed' : 'missing'"
    :status="item ? periodStatus(item.startTime, item.endTime, clock.now.value) : undefined"
    :tags="event ? [typeLabel(event.eventType)] : []"
    :draft="draft"
    :copy-label="t('gmConfig.events.rerun')"
    @update:model-value="emit('update:modelValue', $event)"
    @copy="rerun"
    @delete="remove"
    @restore="restore"
  >
    <template v-if="event">
      <label class="gmc-switch-row">
        <AppSwitch :model-value="permanent" @update:model-value="setPermanent" />
        <span class="gmc-switch-row__text">
          <strong>{{ t('gmConfig.ops.status.permanent') }}</strong>
          <span class="gmc-hint">{{ t('gmConfig.events.permanentHint') }}</span>
        </span>
      </label>
      <GmSchemaForm :schema="homeEventSchema(t, te, event)" :model-value="event" keep-empty :image-sources="images" @update:model-value="updateEvent" />
      <template v-if="story">
        <h3 class="edl__section">{{ t('gmConfig.events.storyContent') }}</h3>
        <GmSchemaForm :schema="storySchema" :model-value="story" keep-empty :image-sources="images" @update:model-value="assignInPlace(story, $event)" />
      </template>
    </template>
    <GmSchemaForm v-else-if="research" :schema="researchSchemaValue" :model-value="research" keep-empty :image-sources="images" @update:model-value="assignInPlace(research, $event)" />
    <ul v-if="problems.length" class="gmc-problems">
      <li v-for="p in problems" :key="p.key" :class="`is-${p.level}`">
        <UIcon :name="p.level === 'bad' ? 'i-lucide-circle-x' : 'i-lucide-triangle-alert'" />{{ t(`gmConfig.events.problems.${p.key}`, p.params || {}) }}
      </li>
    </ul>
  </ItemDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { HomeEvent, ResearchEvent } from '@/games/gakumas/types'
import GmSchemaForm from '../../../components/GmSchemaForm.vue'
import { useGmContext } from '../../../context'
import type { ItemRef } from '../calendar'
import { useOpsDomains } from '../domains'
import { homeEventSchema, researchSchema, storyEventSchema } from '../schemas'
import { usedSource } from '../shared/assets'
import { assignInPlace, cloneJson, draftOf, FAR_FUTURE, freeId, nextHour, PERMANENT_END, periodStatus, useTracked } from '../shared/entry'
import ItemDialog from '../shared/ItemDialog.vue'
import { useServerClock } from '../shared/serverClock'
import { homeEventProblems, researchProblems, storyProblems } from './events'

const props = defineProps<{ modelValue: boolean; kind: 'event' | 'research'; id: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; open: [ref: ItemRef] }>()

const { t, te } = useI18n()
const ctx = useGmContext()
const clock = useServerClock(ctx)
const domains = useOpsDomains()
const home = domains.home.state
const researchState = domains.research.state
const storyState = domains.storyEvent.state

const storySchema = computed(() => storyEventSchema(t, te))
const researchSchemaValue = computed(() => researchSchema(t, te))

const event = useTracked(() => (props.kind === 'event' ? home.working.value?.events : undefined), (e) => e.eventId, props.id)
const research = useTracked(() => (props.kind === 'research' ? researchState.working.value?.researches : undefined), (r) => r.id, props.id)
// 剧情活动的内容和活动日程里的活动同 ID
const story = computed(() => (event.value ? storyState.working.value?.storyEvents.find((s) => s.storyEventId === event.value!.eventId) : undefined))
const item = computed(() => event.value || research.value)
const before = computed(() =>
  props.kind === 'event' ? home.published.value?.events.find((e) => e.eventId === props.id) : researchState.published.value?.researches.find((r) => r.id === props.id),
)
const storyBefore = computed(() => storyState.published.value?.storyEvents.find((s) => s.storyEventId === props.id))
const draft = computed(() => draftOf(before.value, item.value) || (item.value && draftOf(storyBefore.value, story.value) ? 'changed' : undefined))

const typeLabel = (type: number) => (te(`gmConfig.calendar.eventTypes.${type}`) ? t(`gmConfig.calendar.eventTypes.${type}`) : t('gmConfig.events.typeN', { n: type }))
const nameOf = (id: string) => home.working.value?.events.find((e) => e.eventId === id)?.eventName || ''
const title = computed(() => (props.kind === 'research' ? nameOf(research.value?.id || props.id) || t('gmConfig.events.researchName') : (event.value || (before.value as HomeEvent | undefined))?.eventName || t('game.unnamed')))

// 改了活动 ID 时，剧情活动内容跟着改，两边保持对应
const updateEvent = (next: Record<string, unknown>) => {
  const e = event.value
  if (!e) return
  const s = story.value
  const oldId = e.eventId
  assignInPlace(e, next)
  if (s && e.eventId !== oldId) s.storyEventId = e.eventId
}

const permanent = computed(() => !!event.value && event.value.endTime >= FAR_FUTURE)
const setPermanent = (on: boolean) => {
  const e = event.value
  if (!e) return
  const end = on ? PERMANENT_END : Math.max(e.startTime, clock.now.value) + 14 * 86_400_000
  e.endTime = end
  e.fixRankTime = end
  e.closeTime = end
}

// 图片：从配置里已经用过的图里选（确定是客户端里有的资源），显示是哪个活动在用
const images = computed(() => {
  const events = home.working.value?.events || []
  const stories = storyState.working.value?.storyEvents || []
  const researches = researchState.working.value?.researches || []
  const storyOwner = (id: string) => nameOf(id) || t('gmConfig.events.researchName')
  return {
    eventBanner: [usedSource('eventBanner', t('gmConfig.schema.home.bannerAssetId.title'), events.map((e) => ({ assetId: e.bannerAssetId, owner: e.eventName })))],
    homeIcon: [usedSource('homeIcon', t('gmConfig.schema.home.homeIconAssetId.title'), events.map((e) => ({ assetId: e.homeIconAssetId, owner: e.eventName })))],
    researchTitle: [usedSource('researchTitle', t('gmConfig.schema.research.titleAssetId.title'), researches.map((r) => ({ assetId: r.titleAssetId, owner: storyOwner(r.id) })))],
    storyBackground: [usedSource('storyBackground', t('gmConfig.schema.story.backgroundAssetId.title'), stories.map((s) => ({ assetId: s.backgroundAssetId as string, owner: storyOwner(s.storyEventId) })))],
    storyAppeal: [usedSource('storyAppeal', t('gmConfig.schema.story.appealImageAssetId.title'), stories.map((s) => ({ assetId: s.appealImageAssetId as string, owner: storyOwner(s.storyEventId) })))],
    storyRewardBanner: [usedSource('storyRewardBanner', t('gmConfig.schema.story.pointRewardBannerAssetId.title'), stories.map((s) => ({ assetId: s.pointRewardBannerAssetId, owner: storyOwner(s.storyEventId) })))],
  }
})

const problems = computed(() => {
  if (event.value && home.working.value) {
    const out = homeEventProblems(home.working.value, event.value)
    if (story.value && storyState.working.value) out.push(...storyProblems(storyState.working.value, story.value, home.working.value.events))
    return out
  }
  if (research.value && researchState.working.value) return researchProblems(researchState.working.value, research.value)
  return []
})

// 复刻：复制全部字段（同一套任务、剧情与奖励），新 ID，从下一个整点开始，期间长度不变（常驻的仍是常驻）
const shifted = <T extends { startTime: number; endTime: number; fixRankTime?: number; closeTime?: number }>(src: T): T => {
  const copy = cloneJson(src)
  if (src.endTime >= FAR_FUTURE) return copy
  const shift = nextHour(clock.now.value) - src.startTime
  copy.startTime += shift
  copy.endTime += shift
  if (copy.fixRankTime) copy.fixRankTime += shift
  if (copy.closeTime) copy.closeTime += shift
  return copy
}
const rerun = () => {
  if (event.value && home.working.value) {
    const c = home.working.value
    const copy = shifted(event.value)
    copy.eventId = freeId((v) => c.events.some((e) => e.eventId === v), `${event.value.eventId}-rerun`)
    copy.priority = Math.max(0, ...c.events.map((e) => e.priority || 0)) + 1
    c.events.push(copy)
    if (story.value && storyState.working.value) storyState.working.value.storyEvents.push({ ...cloneJson(story.value), storyEventId: copy.eventId })
    emit('open', { kind: 'event', id: copy.eventId })
  } else if (research.value && researchState.working.value) {
    const c = researchState.working.value
    const copy = shifted(research.value)
    copy.id = freeId((v) => c.researches.some((r) => r.id === v), `${research.value.id}-rerun`)
    c.researches.push(copy)
    emit('open', { kind: 'research', id: copy.id })
  }
}
// 删除活动日程里的活动时，它的剧情活动内容一起删
const remove = () => {
  const drop = <T>(list: T[] | undefined, x: T | undefined) => {
    const i = list && x ? list.indexOf(x) : -1
    if (list && i >= 0) list.splice(i, 1)
  }
  drop(storyState.working.value?.storyEvents, story.value)
  drop(home.working.value?.events, event.value)
  drop(researchState.working.value?.researches, research.value)
}
const restore = () => {
  if (props.kind === 'event' && before.value && home.working.value) {
    home.working.value.events.push(cloneJson(before.value as HomeEvent))
    if (storyBefore.value && storyState.working.value && !storyState.working.value.storyEvents.some((s) => s.storyEventId === props.id)) {
      storyState.working.value.storyEvents.push(cloneJson(storyBefore.value))
    }
  }
  if (props.kind === 'research' && before.value && researchState.working.value) researchState.working.value.researches.push(cloneJson(before.value as ResearchEvent))
}
</script>

<style lang="scss" src="../shared/form.scss"></style>
<style scoped lang="scss">
.edl__section {
  margin: 8px 0 0;
  padding-top: 14px;
  border-top: 2px solid var(--el-border-color);
  font-size: 16px;
  font-weight: 800;
}
</style>
