<!-- 卡池头部（横幅、状态、名称）与「1 基本信息」：名称、说明、排序、开放时间、背景图。
     只编辑 data/Gasha.json 里真实存在的字段；没有名称可显示的客户端资源名与条件（动画、BGM、解锁条件等）不在界面上出现，
     新建卡池时沿用模板卡池的值。 -->
<template>
  <div v-if="g" class="gb">
    <div class="gb__hero">
      <div class="gb__banner-wrap">
        <GameImage v-if="g.bannerAssetId" :src="gakumas.gashaBanner(g.bannerAssetId)" class="gb__banner" eager />
        <span v-else class="gb__banner gb__banner--empty">{{ t('gmConfig.gasha.noBanner') }}</span>
        <UButton size="sm" color="neutral" variant="solid" class="gmc-btn gb__banner-btn" icon="i-lucide-image" @click="bannerOpen = true">
          {{ t('gmConfig.gasha.changeBanner') }}
        </UButton>
      </div>
      <div class="gb__head">
        <div class="gb__pills">
          <StatusPill :variant="statusVariant(status)" :label="statusText" />
          <span v-if="!ed.publishedGasha.value" class="gmc-tag gmc-tag--g">{{ t('gmConfig.common.new') }}</span>
          <span v-else-if="changed" class="gmc-tag gmc-tag--warn">{{ t('gmConfig.common.changed') }}</span>
        </div>
        <h2 class="gb__name">{{ g.name || t('gmConfig.gasha.unnamed') }}</h2>
        <span class="gmc-hint">{{ t('gmConfig.gasha.bannerHint') }}</span>
      </div>
    </div>

    <section class="gmc-sec">
      <div class="gmc-sh"><span class="gmc-sn">1</span><span class="gmc-sh__title">{{ t('gmConfig.gasha.sec.basic') }}</span></div>
      <div class="gmc-grid">
        <label class="gmc-field gmc-full">
          <span class="gmc-label">{{ t('gmConfig.gasha.name') }}</span>
          <input v-model="g.name" class="gmc-input" :class="{ 'is-invalid': !g.name }" />
        </label>
        <label class="gmc-field gmc-full">
          <span class="gmc-label">{{ t('gmConfig.gasha.description') }}<span class="gmc-label__aside">{{ t('gmConfig.gasha.descriptionHint') }}</span></span>
          <textarea v-model="g.description" class="gmc-textarea" rows="2" />
        </label>
        <div class="gmc-field">
          <span class="gmc-label">{{ t('gmConfig.gasha.type') }}</span>
          <span class="gmc-readonly">{{ typeLabel }}</span>
        </div>
        <label class="gmc-field">
          <span class="gmc-label">{{ t('gmConfig.gasha.order') }}</span>
          <span class="gmc-unit">
            <input class="gmc-input m-num" type="number" step="1" :value="g.order" @change="setOrder" />
            <span class="gmc-unit__u">{{ t('gmConfig.gasha.orderHint') }}</span>
          </span>
        </label>
        <div class="gmc-field">
          <span class="gmc-label">{{ t('gmConfig.common.start') }}</span>
          <DateTimeField v-model="g.startTime" :zone="ed.zone.value" :invalid="!(g.startTime > 0)" :label="t('gmConfig.common.start')" />
        </div>
        <div class="gmc-field">
          <span class="gmc-label">
            {{ t('gmConfig.common.end') }}
            <span class="gb__perm">
              <span>{{ t('gmConfig.gasha.permanent') }}</span>
              <AppSwitch :model-value="!g.endTime" @update:model-value="setPermanent" />
            </span>
          </span>
          <DateTimeField
            v-if="g.endTime"
            v-model="g.endTime"
            :zone="ed.zone.value"
            :invalid="g.endTime <= g.startTime"
            :label="t('gmConfig.common.end')"
          />
          <span v-else class="gmc-readonly">{{ t('gmConfig.gasha.permanentHint') }}</span>
        </div>
        <div class="gmc-field gmc-full">
          <span class="gmc-label">{{ t('gmConfig.gasha.background') }}</span>
          <div class="gb__bg">
            <GameImage v-if="g.screenAssetId" :src="gakumas.gashaBg(g.screenAssetId)" class="gb__bg-img" />
            <span class="gb__bg-text">
              <span class="gb__bg-state">{{ g.screenAssetId ? t('gmConfig.gasha.backgroundSet') : t('gmConfig.gasha.noBackground') }}</span>
              <span class="gmc-hint">{{ t('gmConfig.gasha.backgroundHint', { n: bgPickups }) }}</span>
            </span>
            <UButton class="gmc-btn" size="sm" color="neutral" variant="outline" icon="i-lucide-image" @click="bgOpen = true">{{ t('gmConfig.gasha.chooseAsset') }}</UButton>
            <AppIconButton v-if="g.screenAssetId" icon="HOutline:XMarkIcon" :label="t('gmConfig.gasha.clearBackground')" :box="40" @click="g.screenAssetId = ''" />
          </div>
        </div>
      </div>
    </section>

    <AssetPickerDialog
      v-model="bannerOpen"
      :title="t('gmConfig.gasha.changeBanner')"
      :sources="[gashaBannerSource(t('gmConfig.assets.gashaBanner'))]"
      :selected="g.bannerAssetId"
      @pick="g.bannerAssetId = $event"
    />
    <AssetPickerDialog
      v-model="bgOpen"
      :title="t('gmConfig.gasha.background')"
      :sources="[gashaBackgroundSource(t('gmConfig.assets.gashaBackground'))]"
      :selected="g.screenAssetId"
      @pick="g.screenAssetId = $event"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { gakumas } from '@/games'
import { useGashaEditor } from './editor'
import { gashaStatus, statusVariant } from './gasha'
import AssetPickerDialog from './shared/AssetPickerDialog.vue'
import { gashaBackgroundSource, gashaBannerSource } from './shared/assets'
import DateTimeField from './shared/DateTimeField.vue'
import { deepEqual } from './shared/diff'

const { t, te } = useI18n()
const ed = useGashaEditor()
const g = computed(() => ed.gasha.value)

const bannerOpen = ref(false)
const bgOpen = ref(false)

const status = computed(() => (g.value ? gashaStatus(g.value, ed.now.value) : 'upcoming'))
const DAY = 86_400_000
const statusText = computed(() => {
  const x = g.value
  if (!x) return ''
  const label = t(`gmConfig.gasha.status.${status.value}`)
  if (status.value === 'open') return t('gmConfig.gasha.remaining', { label, n: Math.max(1, Math.ceil((x.endTime - ed.now.value) / DAY)) })
  if (status.value === 'upcoming') return t('gmConfig.gasha.startsIn', { label, n: Math.max(1, Math.ceil((x.startTime - ed.now.value) / DAY)) })
  return label
})
const changed = computed(() => !!g.value && !!ed.publishedGasha.value && !deepEqual(g.value, ed.publishedGasha.value))
const typeLabel = computed(() => {
  const type = g.value?.type
  return type && te(`gmConfig.gasha.types.${type}`) ? t(`gmConfig.gasha.types.${type}`) : '—'
})
const bgPickups = computed(() => (g.value?.backgroundPickupIdolCardSkins?.length || 0) + (g.value?.backgroundPickupSupportCards?.length || 0))

const setOrder = (e: Event) => {
  if (!g.value) return
  const v = (e.target as HTMLInputElement).valueAsNumber
  g.value.order = Number.isFinite(v) ? Math.trunc(v) : 0
}

const setPermanent = (on: boolean) => {
  const x = g.value
  if (!x) return
  if (on) x.endTime = 0
  else {
    const published = ed.publishedGasha.value
    // 关掉常驻：先用已发布的结束时间（有的话），否则从开始时间起算一天，之后自己改
    x.endTime = published?.endTime && published.endTime > x.startTime ? published.endTime : x.startTime + DAY
  }
}
</script>

<style lang="scss" src="./shared/form.scss"></style>
<style scoped lang="scss">
.gb {
  display: flex;
  flex-direction: column;
}
.gb__hero {
  display: flex;
  flex-wrap: wrap;
  gap: 18px;
  align-items: center;
  padding-bottom: 16px;
}
.gb__banner-wrap {
  position: relative;
  flex: none;
  max-width: 100%;
}
.gb__banner-wrap .gb__banner {
  width: 260px;
  max-width: 100%;
  height: 130px;
  object-fit: contain;
  border-radius: 20px;
  background: var(--m-gSoft, var(--el-fill-color-light));
}
.gb__banner--empty {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.gb__banner-btn {
  position: absolute;
  right: 8px;
  bottom: 8px;
}
.gb__head {
  flex: 1 1 260px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.gb__pills {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}
.gb__name {
  margin: 0;
  font-size: 21px;
  font-weight: 800;
  line-height: 1.35;
  overflow-wrap: anywhere;
}
.gb__perm {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.gb__bg {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  padding: 8px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.gb__bg .gb__bg-img {
  width: 120px;
  height: 68px;
  border-radius: 10px;
  object-fit: cover;
}
.gb__bg-text {
  flex: 1 1 200px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.gb__bg-state {
  font-size: 12px;
  overflow-wrap: anywhere;
}
</style>
