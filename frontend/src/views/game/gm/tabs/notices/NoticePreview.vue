<!-- 游戏内公告列表预览：按草稿内容排出这个分类在服务器当前时间展示中的公告（新的在前，每页 pageSize 条），
     正在编辑的这条即使没在展示期也排进去并标出来。玩家点开公告后看到的是 detailUrl 网页。 -->
<template>
  <div class="np">
    <span class="np__label">{{ t('gmConfig.notice.preview') }}</span>
    <div class="np__phone">
      <div class="np__screen">
        <div class="np__top">
          <span class="np__top-title">{{ t('gmConfig.notice.previewTitle') }}</span>
        </div>
        <div class="np__tabs">
          <span v-for="c in NOTICE_CATEGORIES" :key="c" class="np__tab" :class="{ 'is-on': c === current.category }">
            {{ t(`gmConfig.notice.categories.${c}`) }}
          </span>
        </div>
        <div class="np__list">
          <div v-for="n in list" :key="n.id" class="np__item" :class="{ 'is-current': n === current }">
            <GameImage v-if="bannerImage(n.bannerAssetId)" :src="bannerImage(n.bannerAssetId)" class="np__banner" />
            <span class="np__title">{{ n.title || t('gmConfig.notice.untitled') }}</span>
            <span class="np__period m-num">{{ period(n) }}</span>
            <span v-if="n === current && hidden" class="np__hidden">{{ hidden }}</span>
          </div>
          <p v-if="!list.length" class="np__empty">{{ t('gmConfig.notice.previewEmpty') }}</p>
        </div>
      </div>
    </div>
    <span class="np__hint">{{ t('gmConfig.notice.previewHint', { n: cfg.pageSize }) }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { NoticeConfig, NoticeEntry } from '@/games/gakumas/types'
import { bannerImage } from '../gasha/shared/assets'
import { formatZoned } from '../gasha/shared/time'
import { NOTICE_CATEGORIES, noticeStatus } from './notice'

const props = defineProps<{
  cfg: NoticeConfig
  current: NoticeEntry
  now: number
  zone: string
}>()

const { t } = useI18n()

const hidden = computed(() => {
  const s = noticeStatus(props.current, props.now)
  return s === 'showing' ? '' : t(`gmConfig.notice.previewHidden.${s}`)
})

const list = computed(() => {
  const items = props.cfg.notices.filter(
    (n) => n.category === props.current.category && (n === props.current || noticeStatus(n, props.now) === 'showing'),
  )
  items.sort((a, b) => b.startTime - a.startTime)
  const size = props.cfg.pageSize > 0 ? props.cfg.pageSize : items.length
  const page = items.slice(0, size)
  if (!page.includes(props.current) && items.includes(props.current)) page.push(props.current)
  return page
})

const period = (n: NoticeEntry) =>
  `${formatZoned(n.startTime, props.zone)} 〜 ${n.endTime ? formatZoned(n.endTime, props.zone) : ''}`
</script>

<style scoped lang="scss">
.np {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
}
.np__label {
  align-self: flex-start;
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.04em;
  color: var(--el-text-color-secondary);
}
.np__phone {
  width: 300px;
  max-width: 100%;
  border-radius: 36px;
  padding: 10px;
  background: var(--m-ink);
  box-shadow: var(--app-shadow-lg);
}
.np__screen {
  border-radius: 28px;
  overflow: hidden;
  background: var(--m-bg);
  height: 540px;
  display: flex;
  flex-direction: column;
}
.np__top {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 16px 14px 10px;
  background: var(--el-bg-color);
}
.np__top-title {
  flex: 1;
  font-size: 15px;
  font-weight: 800;
}
.np__tabs {
  display: flex;
  gap: 4px;
  padding: 0 12px 10px;
  background: var(--el-bg-color);
}
.np__tab {
  flex: 1;
  min-height: 32px;
  border-radius: 999px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 800;
  color: var(--el-text-color-secondary);
}
.np__tab.is-on {
  background: var(--m-g, var(--m-p));
  color: var(--m-gOn, var(--m-pOn));
}
.np__list {
  flex: 1;
  overflow-y: auto;
  scrollbar-width: none;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.np__list::-webkit-scrollbar {
  display: none;
}
.np__item {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px;
  border-radius: 14px;
  background: var(--el-bg-color);
  border: 2px solid transparent;
}
.np__item.is-current {
  border-color: var(--m-g, var(--m-p));
}
.np__item .np__banner {
  width: 100%;
  aspect-ratio: 920 / 320;
  object-fit: cover;
  border-radius: 10px;
}
.np__title {
  font-size: 13px;
  font-weight: 800;
  line-height: 1.4;
  overflow-wrap: anywhere;
}
.np__period {
  font-size: 10px;
  font-weight: 700;
  color: var(--m-gInk, var(--m-pInk));
}
.np__hidden {
  align-self: flex-start;
  padding: 1px 8px;
  border-radius: 999px;
  background: var(--m-warnSoft);
  color: var(--m-warn);
  font-size: 10px;
  font-weight: 800;
}
.np__empty {
  margin: 24px 0;
  text-align: center;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.np__hint {
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
  text-align: center;
}
</style>
