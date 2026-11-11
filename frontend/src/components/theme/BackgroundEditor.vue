<!-- 背景编辑（ThemePC「背景」卡片 / ThemeMobile「背景」/ 主题编辑器「背景」标签）：
     跟随主题 / 纯色 / 图案 / 渐变 / 图片。图片可上传（JPG / PNG / WebP ≤ 10 MB，服务端转 WebP，可直接拖进来），
     模糊、遮罩、卡片不透明度、填充方式、深色模式自动加深遮罩；渐变有预设、色标、角度、线性 / 径向与缓慢流动。 -->
<template>
  <div class="bge" :class="`bge--${variant}`">
    <!-- 类型 -->
    <div v-if="variant === 'page'" class="bge-seg" role="tablist" :aria-label="t('theme.background')">
      <button
        v-for="k in BG_TYPES"
        :key="k"
        type="button"
        role="tab"
        :class="{ on: viewType === k }"
        :aria-selected="viewType === k"
        @click="setType(k)"
      >
        {{ t(`theme.bgTypes.${k}`) }}
      </button>
    </div>
    <div v-else class="bge-chips" :class="{ 'is-scroll': variant === 'mobile' }" role="tablist" :aria-label="t('theme.background')">
      <button
        v-for="k in BG_TYPES"
        :key="k"
        type="button"
        role="tab"
        class="bge-chip"
        :class="{ on: viewType === k }"
        :aria-selected="viewType === k"
        @click="setType(k)"
      >
        {{ t(`theme.bgTypes.${k}`) }}
      </button>
    </div>

    <div :class="{ 'bge-card': variant === 'mobile' }">
      <!-- 图片 -->
      <template v-if="viewType === 'image'">
        <div class="bge-img">
          <div v-if="variant === 'page'" class="bge-img__main">
            <div
              class="bge-img__pv"
              :class="{ 'is-drop': dragging }"
              @dragover.prevent="dragging = true"
              @dragleave="dragging = false"
              @drop.prevent="onDrop"
            >
              <span class="bge-img__bg" :style="{ background: css, filter: bg.blur ? `blur(${bg.blur}px)` : undefined }" />
              <span class="bge-img__dim" :style="{ background: `color-mix(in srgb, var(--m-bg) ${dimNow}%, transparent)` }" />
              <span class="bge-img__cards">
                <span :style="glassCard"><b>blog-api</b><small>{{ t('theme.scene.sampleRunning') }}</small></span>
                <span :style="glassCard"><b class="m-num">CPU 16.5%</b><small>{{ t('theme.scene.sampleMem') }}</small></span>
              </span>
            </div>
            <div class="bge-img__meta">
              <span class="m-mono">{{ currentAsset?.name || (bg.image ? bg.image : t('theme.noImage')) }}</span>
              <span class="m-num">{{ assetMeta(currentAsset) }}</span>
            </div>
          </div>
          <div class="bge-thumbs">
            <button
              v-for="im in images"
              :key="im.id"
              type="button"
              class="bge-thumb"
              :class="{ on: bg.image === im.id }"
              :aria-pressed="bg.image === im.id"
              :aria-label="im.name || im.id"
              :style="{ backgroundImage: `url(&quot;${assetUrl(im.id)}&quot;)` }"
              @click="patch({ image: im.id })"
            />
            <label
              class="bge-thumb bge-up"
              :class="{ 'is-drop': dragging, 'is-wide': variant === 'page' }"
              @dragover.prevent="dragging = true"
              @dragleave="dragging = false"
              @drop.prevent="onDrop"
            >
              <input type="file" accept="image/jpeg,image/png,image/webp" class="bge-file" @change="onPick" />
              <component :is="uploading ? ArrowPathIcon : ArrowUpTrayIcon" :class="{ 'is-spin': uploading }" aria-hidden="true" />
              <span v-if="variant === 'page'" class="bge-up__text">
                <b>{{ uploading ? t('theme.uploading') : t('theme.upload') }}</b>
                <small>{{ t('theme.uploadHint') }}</small>
              </span>
              <b v-else>{{ uploading ? t('theme.uploading') : t('theme.uploadShort') }}</b>
            </label>
          </div>
        </div>
        <div class="bge-sliders">
          <label v-for="s in sliders" :key="s.key" class="bge-range">
            <span class="bge-range__head"><span>{{ t(`theme.${s.key}`) }}</span><b class="m-num">{{ s.value }}{{ s.unit }}</b></span>
            <input
              class="bge-rng"
              type="range"
              :min="s.min"
              :max="s.max"
              :value="s.value"
              :aria-label="t(`theme.${s.key}`)"
              @input="patch({ [s.key]: Number(($event.target as HTMLInputElement).value) })"
            />
            <small v-if="variant === 'page'">{{ t(`theme.${s.key}Hint`) }}</small>
          </label>
        </div>
        <div class="bge-row">
          <span class="bge-row__label">{{ t('theme.fit') }}</span>
          <div class="bge-seg bge-seg--sm">
            <button v-for="f in FITS" :key="f" type="button" :class="{ on: bg.fit === f }" @click="patch({ fit: f })">
              {{ t(`theme.fits.${f}`) }}
            </button>
          </div>
          <span class="bge-row__spacer" />
          <label class="bge-switch">
            <AppSwitch :model-value="bg.autoDarkDim" :aria-label="t('theme.autoDarkDim')" @update:model-value="patch({ autoDarkDim: $event })" />
            {{ t('theme.autoDarkDim') }}
          </label>
        </div>
      </template>

      <!-- 渐变 -->
      <template v-else-if="viewType === 'gradient'">
        <div class="bge-grads">
          <button
            v-for="(g, i) in GRADIENT_PRESETS"
            :key="g.key"
            type="button"
            class="bge-gsw"
            :class="{ on: isPreset(g) }"
            :aria-pressed="isPreset(g)"
            :aria-label="t('theme.gradientPreset', { n: i + 1 })"
            :style="{ background: gradientCss(g.stops, g.angle, 'linear') }"
            @click="patch({ gradientStops: [...g.stops], gradientAngle: g.angle })"
          />
        </div>
        <div class="bge-gbox">
          <div v-if="variant !== 'mobile'" class="bge-gbar" :style="{ background: `linear-gradient(90deg, ${bg.gradientStops.join(', ')})` }">
            <span v-for="(c, i) in bg.gradientStops" :key="i" :style="{ left: stopPos(i), background: c }" />
          </div>
          <div class="bge-stops">
            <span v-for="(c, i) in bg.gradientStops" :key="i" class="bge-stop">
              <label class="bge-stop__sw" :style="{ background: c }">
                <input type="color" :value="c" :aria-label="t('theme.stopColor')" @input="setStop(i, ($event.target as HTMLInputElement).value)" />
              </label>
              <span class="m-mono">{{ c }}</span>
              <span v-if="variant !== 'mobile'" class="m-num bge-stop__pos">{{ stopPos(i) }}</span>
              <button
                v-if="bg.gradientStops.length > 2"
                type="button"
                class="bge-stop__x"
                :aria-label="t('theme.removeStop')"
                @click="removeStop(i)"
              >
                <XMarkIcon aria-hidden="true" />
              </button>
            </span>
            <button v-if="bg.gradientStops.length < 5" type="button" class="bge-addstop" @click="addStop">
              <PlusIcon aria-hidden="true" />{{ t('theme.addStop') }}
            </button>
            <span class="bge-row__spacer" />
            <div class="bge-seg bge-seg--sm">
              <button
                v-for="k in ['linear', 'radial']"
                :key="k"
                type="button"
                :class="{ on: bg.gradientKind === k }"
                @click="patch({ gradientKind: k })"
              >
                {{ t(`theme.gradientKinds.${k}`) }}
              </button>
            </div>
          </div>
          <label v-if="bg.gradientKind !== 'radial'" class="bge-angle">
            <span>{{ t('theme.angle') }}</span>
            <input
              class="bge-rng"
              type="range"
              min="0"
              max="360"
              :value="bg.gradientAngle"
              :aria-label="t('theme.gradientAngle')"
              @input="patch({ gradientAngle: Number(($event.target as HTMLInputElement).value) })"
            />
            <b class="m-num">{{ bg.gradientAngle }}°</b>
          </label>
        </div>
        <label class="bge-switch">
          <AppSwitch :model-value="bg.gradientFlow" :aria-label="t('theme.flow')" @update:model-value="patch({ gradientFlow: $event })" />
          {{ t('theme.flow') }}<small>{{ t('theme.flowHint') }}</small>
        </label>
      </template>

      <!-- 图案 -->
      <template v-else-if="viewType === 'pattern'">
        <div class="bge-pats">
          <button
            v-for="p in PATTERNS"
            :key="p"
            type="button"
            class="bge-pat"
            :class="{ on: bg.pattern === p }"
            :aria-pressed="bg.pattern === p"
            @click="patch({ pattern: p })"
          >
            <span :style="{ background: `${patternCss(p, 'color-mix(in srgb, var(--m-ink) 22%, transparent)')}, var(--m-bg)` }" />
            <b>{{ t(`theme.patterns.${p}`) }}</b>
          </button>
        </div>
        <div class="bge-row">
          <span class="bge-row__label">{{ t('theme.patternColor') }}</span>
          <div class="bge-seg bge-seg--sm">
            <button v-for="c in PATTERN_COLORS" :key="c" type="button" :class="{ on: bg.patternColor === c }" @click="patch({ patternColor: c })">
              {{ t(`theme.patternColors.${c}`) }}
            </button>
          </div>
        </div>
      </template>

      <!-- 纯色 -->
      <template v-else-if="viewType === 'solid'">
        <div class="bge-solids">
          <button
            v-for="c in SOLID_PRESETS"
            :key="c"
            type="button"
            class="bge-gsw bge-gsw--sq"
            :class="{ on: bg.color === c }"
            :aria-pressed="bg.color === c"
            :aria-label="c"
            :style="{ background: c }"
            @click="patch({ color: c })"
          />
          <label class="bge-hex">
            <span class="bge-hex__sw" :style="{ background: bg.color }">
              <input type="color" :value="bg.color" :aria-label="t('theme.customColor')" @input="patch({ color: ($event.target as HTMLInputElement).value.toUpperCase() })" />
            </span>
            <input class="m-mono" type="text" :value="bg.color" :aria-label="t('theme.customColor')" @change="onHex" />
          </label>
        </div>
      </template>

      <div v-else class="bge-note">
        {{ variant === 'page' ? t('theme.bgThemeHint') : t('theme.bgThemeHintShort') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ArrowPathIcon, ArrowUpTrayIcon, PlusIcon, XMarkIcon } from '@heroicons/vue/24/outline'
import type { ThemeBackground } from '@/types/v1/theme'
import {
  BG_TYPES,
  FITS,
  GRADIENT_PRESETS,
  PATTERNS,
  PATTERN_COLORS,
  SOLID_PRESETS,
  assetUrl,
  backgroundCss,
  backgroundOfType,
  effectiveDim,
  gradientCss,
  normalizeBackground,
  patternCss,
  type BgType,
} from '@/theme/content'
import { mixHex, normalizeHex } from '@/theme/color'
import { assetMeta, useThemeAssets } from '@/theme/useThemeAssets'

const props = withDefaults(
  defineProps<{
    modelValue: ThemeBackground
    variant?: 'page' | 'mobile' | 'editor'
    /** 预览用的深浅色（遮罩的自动加深按它算） */
    scheme?: 'light' | 'dark'
  }>(),
  { variant: 'page', scheme: 'light' },
)
const emit = defineEmits<{ 'update:modelValue': [value: ThemeBackground] }>()

const { t } = useI18n()
const { images, findAsset, upload, uploading } = useThemeAssets()

const bg = computed(() => normalizeBackground(props.modelValue))
const css = computed(() => backgroundCss(bg.value))
const dimNow = computed(() => effectiveDim(bg.value, props.scheme))
const currentAsset = computed(() => (bg.value.image ? findAsset(bg.value.image) : null))
const glassCard = computed(() => ({ background: `color-mix(in srgb, var(--m-s) ${bg.value.cardOpacity}%, transparent)` }))
const dragging = ref(false)
/** 还没有图片时先切到「图片」页签等用户上传（此时不改背景） */
const pendingImage = ref(false)
const viewType = computed<BgType>(() => (pendingImage.value ? 'image' : (bg.value.type as BgType)))
watch(
  () => props.modelValue,
  (v) => {
    if (v?.type && v.type !== 'theme') pendingImage.value = false
  },
)

const sliders = computed(() => [
  { key: 'blur', value: bg.value.blur, min: 0, max: 40, unit: ' px' },
  { key: 'dim', value: bg.value.dim, min: 0, max: 80, unit: '%' },
  { key: 'cardOpacity', value: bg.value.cardOpacity, min: 40, max: 100, unit: '%' },
])

const emitBg = (next: ThemeBackground) => emit('update:modelValue', next)
const patch = (p: Partial<ThemeBackground>) => emitBg({ ...bg.value, ...p })
const setType = (type: BgType) => {
  if (type === viewType.value) return
  pendingImage.value = false
  const next = backgroundOfType(type, bg.value)
  if (type === 'image' && !next.image) {
    if (!images.value[0]) {
      pendingImage.value = true
      return
    }
    next.image = images.value[0].id
  }
  emitBg(next)
}

const isPreset = (g: { stops: string[]; angle: number }) =>
  g.stops.join() === bg.value.gradientStops.join() && g.angle === bg.value.gradientAngle
const stopPos = (i: number) => `${Math.round((i / Math.max(1, bg.value.gradientStops.length - 1)) * 100)}%`
const setStop = (i: number, c: string) => {
  const stops = [...bg.value.gradientStops]
  stops[i] = c.toUpperCase()
  patch({ gradientStops: stops })
}
const removeStop = (i: number) => patch({ gradientStops: bg.value.gradientStops.filter((_, k) => k !== i) })
const addStop = () => {
  const s = bg.value.gradientStops
  const last = s[s.length - 1] || '#FFFFFF'
  patch({ gradientStops: [...s, mixHex(last, s[s.length - 2] || '#FFFFFF', 0.5)] })
}

const onHex = (e: Event) => {
  const hex = normalizeHex((e.target as HTMLInputElement).value)
  if (hex) patch({ color: hex })
  else (e.target as HTMLInputElement).value = bg.value.color
}

const useFile = async (file?: File | null) => {
  dragging.value = false
  if (!file) return
  const asset = await upload(file, 'image')
  if (asset) emitBg({ ...backgroundOfType('image', bg.value), image: asset.id })
}
const onPick = (e: Event) => {
  const input = e.target as HTMLInputElement
  void useFile(input.files?.[0])
  input.value = ''
}
const onDrop = (e: DragEvent) => void useFile(e.dataTransfer?.files?.[0])
</script>

<style scoped lang="scss">
.bge {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}
.bge--editor {
  gap: 12px;
}
.bge-card {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 14px;
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
}
.bge--page > div:not(.bge-seg),
.bge--editor > div:not(.bge-chips) {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.bge--editor > div:not(.bge-chips) {
  gap: 12px;
}

/* 分段 */
.bge-seg {
  display: inline-flex;
  align-self: flex-start;
  gap: 2px;
  padding: 4px;
  border-radius: 999px;
  background: var(--m-s2);
  max-width: 100%;
  overflow-x: auto;
  scrollbar-width: none;
  &::-webkit-scrollbar {
    display: none;
  }
  button {
    min-height: 36px;
    padding: 0 16px;
    border: 0;
    border-radius: 999px;
    background: transparent;
    font: inherit;
    font-size: 0.8125rem;
    font-weight: 700;
    color: var(--m-ink2);
    cursor: pointer;
    white-space: nowrap;
  }
  .on {
    background: var(--m-s);
    color: var(--m-ink);
    box-shadow: 0 1px 3px rgba(20, 10, 30, 0.12);
  }
}
.bge-seg--sm button {
  min-height: 32px;
  padding: 0 12px;
  font-size: 0.75rem;
}
.bge-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  &.is-scroll {
    flex-wrap: nowrap;
    overflow-x: auto;
    margin: 0 -16px;
    padding: 0 16px;
    scrollbar-width: none;
    &::-webkit-scrollbar {
      display: none;
    }
  }
}
.bge-chip {
  display: inline-flex;
  align-items: center;
  flex: none;
  min-height: 40px;
  padding: 0 16px;
  border-radius: 999px;
  border: 1.5px solid var(--m-line);
  background: var(--m-s);
  color: var(--m-ink2);
  font: inherit;
  font-size: 0.875rem;
  font-weight: 600;
  cursor: pointer;
  &:hover {
    border-color: var(--m-p);
    color: var(--m-pInk);
  }
  &.on {
    background: var(--m-ink);
    border-color: var(--m-ink);
    color: var(--m-bg);
  }
}
.bge--editor .bge-chip {
  min-height: 34px;
  padding: 0 12px;
  font-size: 0.8125rem;
}

/* 图片 */
.bge-img {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
}
.bge-img__main {
  flex: 1 1 300px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.bge-img__pv {
  position: relative;
  aspect-ratio: 16 / 9;
  overflow: hidden;
  border-radius: 18px;
  border: 1px solid var(--m-line);
  &.is-drop {
    outline: 2px dashed var(--m-p);
    outline-offset: 2px;
  }
}
.bge-img__bg {
  position: absolute;
  inset: -20px;
}
.bge-img__dim {
  position: absolute;
  inset: 0;
}
.bge-img__cards {
  position: absolute;
  left: 14px;
  right: 14px;
  bottom: 14px;
  display: flex;
  gap: 8px;
  > span {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 10px 12px;
    border-radius: 14px;
    -webkit-backdrop-filter: blur(14px);
    backdrop-filter: blur(14px);
  }
  b {
    font-size: 0.8125rem;
    color: var(--m-ink);
  }
  small {
    font-size: 0.6875rem;
    color: var(--m-ink2);
  }
}
.bge-img__meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.75rem;
  color: var(--m-ink2);
  .m-mono {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--m-ink);
    font-weight: 600;
  }
}
.bge-thumbs {
  flex: 1 1 200px;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  align-content: start;
}
.bge--mobile .bge-thumbs {
  flex-basis: 100%;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}
.bge--editor .bge-thumbs {
  flex-basis: 100%;
  gap: 8px;
}
.bge-thumb {
  position: relative;
  aspect-ratio: 16 / 10;
  padding: 0;
  border-radius: 14px;
  border: 2px solid var(--m-line);
  background-color: var(--m-s2);
  background-size: cover;
  background-position: center;
  cursor: pointer;
  &.on {
    border-color: var(--m-p);
    box-shadow: 0 0 0 3px var(--m-pSoft);
  }
}
.bge--mobile .bge-thumb {
  aspect-ratio: 3 / 4;
}
.bge-up {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  border-style: dashed;
  color: var(--m-ink2);
  font-size: 0.75rem;
  svg {
    width: 20px;
    height: 20px;
  }
  b {
    color: var(--m-ink);
  }
  &.is-wide {
    grid-column: 1 / -1;
    aspect-ratio: auto;
    min-height: 64px;
    flex-direction: row;
    justify-content: flex-start;
    gap: 10px;
    padding: 0 14px;
  }
  &.is-drop,
  &:hover {
    border-color: var(--m-p);
  }
  &:focus-within {
    box-shadow: var(--app-focus-ring);
  }
}
.bge-up__text {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  small {
    font-size: 0.6875rem;
  }
}
.bge-file {
  position: absolute;
  width: 1px;
  height: 1px;
  opacity: 0;
  pointer-events: none;
}
.is-spin {
  animation: bge-spin 1s linear infinite;
}
@keyframes bge-spin {
  to {
    rotate: 360deg;
  }
}

/* 滑块 */
.bge-sliders {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
  gap: 14px 20px;
}
.bge--mobile .bge-sliders,
.bge--editor .bge-sliders {
  grid-template-columns: 1fr;
  gap: 10px;
}
.bge-range {
  display: flex;
  flex-direction: column;
  gap: 6px;
  small {
    font-size: 0.6875rem;
    color: var(--m-ink2);
  }
}
.bge-range__head {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.8125rem;
  font-weight: 700;
  span {
    flex: 1;
  }
  b {
    color: var(--m-pInk);
  }
}
.bge-rng {
  width: 100%;
  height: 24px;
  margin: 0;
  accent-color: var(--m-p);
  cursor: pointer;
}
.bge--mobile .bge-rng {
  height: 30px;
}
.bge-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px 16px;
}
.bge-row__label {
  font-size: 0.8125rem;
  font-weight: 700;
}
.bge-row__spacer {
  flex: 1;
}
.bge-switch {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
  small {
    font-weight: 500;
    color: var(--m-ink2);
  }
}

/* 渐变 */
.bge-grads,
.bge-solids {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.bge--mobile .bge-grads,
.bge--editor .bge-grads {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}
.bge-gsw {
  width: 64px;
  height: 44px;
  padding: 0;
  border-radius: 14px;
  border: 2px solid var(--m-line);
  cursor: pointer;
  &.on {
    border-color: var(--m-p);
    box-shadow: 0 0 0 3px var(--m-pSoft);
  }
}
.bge-gsw--sq {
  width: 44px;
}
.bge--mobile .bge-grads .bge-gsw,
.bge--editor .bge-grads .bge-gsw {
  width: auto;
  height: 56px;
}
.bge-gbox {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 16px;
  border-radius: 18px;
  background: var(--m-s2);
}
.bge--mobile .bge-gbox,
.bge--editor .bge-gbox {
  padding: 12px;
}
.bge-gbar {
  position: relative;
  height: 34px;
  border-radius: 12px;
  border: 1px solid var(--m-line);
  span {
    position: absolute;
    top: 50%;
    width: 22px;
    height: 22px;
    margin: -11px 0 0 -11px;
    border-radius: 50%;
    border: 3px solid #ffffff;
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.3);
  }
}
.bge-stops {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.bge-stop {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 36px;
  padding: 0 6px 0 7px;
  border-radius: 999px;
  background: var(--m-s);
  border: 1px solid var(--m-line);
  font-size: 0.75rem;
  font-weight: 600;
}
.bge-stop__sw,
.bge-hex__sw {
  position: relative;
  width: 22px;
  height: 22px;
  flex: none;
  overflow: hidden;
  border-radius: 8px;
  border: 1px solid var(--m-line);
  cursor: pointer;
  input {
    position: absolute;
    inset: -8px;
    width: 40px;
    height: 40px;
    opacity: 0;
    cursor: pointer;
  }
}
.bge-stop__pos {
  font-size: 0.6875rem;
  color: var(--m-ink2);
}
.bge-stop__x {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--m-ink2);
  cursor: pointer;
  svg {
    width: 14px;
    height: 14px;
  }
  &:hover {
    background: var(--m-badSoft);
    color: var(--m-bad);
  }
}
.bge-addstop {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 36px;
  padding: 0 12px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--m-pInk);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 600;
  cursor: pointer;
  svg {
    width: 16px;
    height: 16px;
  }
}
.bge-angle {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 0.8125rem;
  font-weight: 700;
  span {
    width: 40px;
  }
  .bge-rng {
    flex: 1;
  }
  b {
    width: 44px;
    text-align: right;
  }
}

/* 图案 */
.bge-pats {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(96px, 1fr));
  gap: 10px;
}
.bge--mobile .bge-pats,
.bge--editor .bge-pats {
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}
.bge-pat {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 6px 6px 8px;
  border-radius: 16px;
  border: 2px solid var(--m-line);
  background: var(--m-s);
  color: var(--m-ink);
  font: inherit;
  text-align: center;
  cursor: pointer;
  span {
    display: block;
    height: 64px;
    border-radius: 12px;
  }
  b {
    font-size: 0.8125rem;
  }
  &.on {
    border-color: var(--m-p);
    box-shadow: 0 0 0 3px var(--m-pSoft);
  }
}
.bge--mobile .bge-pat span {
  height: 52px;
}

/* 纯色 */
.bge-hex {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 170px;
  min-height: 44px;
  padding: 0 14px;
  border-radius: 999px;
  background: var(--m-s);
  border: 1.5px solid var(--m-line);
  input[type='text'] {
    flex: 1;
    min-width: 0;
    border: 0;
    background: transparent;
    font-size: 0.875rem;
    color: var(--m-ink);
    outline: none;
  }
  &:focus-within {
    border-color: var(--m-p);
  }
}
.bge-note {
  padding: 14px 16px;
  border-radius: 16px;
  background: var(--m-s2);
  font-size: 0.8125rem;
  color: var(--m-ink2);
}
.bge--mobile .bge-note {
  padding: 0;
  background: transparent;
}
</style>
