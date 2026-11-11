<!-- 欢迎横幅 · 背景（ThemeEditorPC 默认选中的状态）：主色 / 渐变 / 图片；图片可上传、拖动焦点、压暗，
     文字颜色 自动 / 白字 / 黑字（自动按图片左侧亮度判断）。同一张图也用在登录页插画区和手机首页顶部卡片。 -->
<template>
  <div class="bne">
    <div class="bne-head">
      <span class="bne-lbl">{{ t('theme.editor.bannerBg') }}</span>
      <span class="bne-tag" :class="{ 'is-page': ed.scope.value === 'page' }">{{ ed.scopeTag.value }}</span>
    </div>
    <div class="bne-seg">
      <button v-for="k in BANNER_TYPES" :key="k" type="button" :class="{ on: b.type === k || (k === 'image' && pendingImage) }" @click="setType(k)">
        {{ t(`theme.editor.bannerTypes.${k}`) }}
      </button>
    </div>

    <template v-if="b.type === 'image' || pendingImage">
      <div class="bne-grid">
        <button
          v-for="im in images"
          :key="im.id"
          type="button"
          class="bne-thumb"
          :class="{ on: b.image === im.id }"
          :aria-label="im.name || im.id"
          :style="{ backgroundImage: `url(&quot;${assetUrl(im.id)}&quot;)` }"
          @click="patch({ type: 'image', image: im.id })"
        />
        <label class="bne-thumb bne-up">
          <input type="file" accept="image/jpeg,image/png,image/webp" class="bne-file" @change="onPick" />
          <component :is="uploading ? ArrowPathIcon : ArrowUpTrayIcon" aria-hidden="true" />
          <b>{{ t('theme.uploadShort') }}</b>
        </label>
      </div>
      <p class="bne-hint">{{ t('theme.editor.bannerSize') }}</p>
      <div
        v-if="b.type === 'image' && b.image"
        ref="focusRef"
        class="bne-focus"
        role="slider"
        tabindex="0"
        :aria-label="t('theme.editor.focus')"
        :aria-valuetext="`${b.focusX}%, ${b.focusY}%`"
        :style="{ backgroundImage: `url(&quot;${assetUrl(b.image)}&quot;)` }"
        @pointerdown="startFocus"
        @keydown="onFocusKey"
      >
        <span :style="{ left: `${b.focusX}%`, top: `${b.focusY}%` }" />
      </div>
      <label class="bne-range">
        <span><span>{{ t('theme.editor.bannerDim') }}</span><b class="m-num">{{ b.dim }}%</b></span>
        <input type="range" min="0" max="80" :value="b.dim" :aria-label="t('theme.editor.bannerDim')" @input="patch({ dim: Number(($event.target as HTMLInputElement).value) })" />
      </label>
      <div class="bne-col">
        <span class="bne-lbl">{{ t('theme.editor.inkTitle') }}</span>
        <div class="bne-seg">
          <button v-for="k in BANNER_INKS" :key="k" type="button" :class="{ on: b.ink === k }" @click="patch({ ink: k })">
            {{ t(`theme.editor.inks.${k}`) }}
          </button>
        </div>
      </div>
    </template>

    <div v-else-if="b.type === 'gradient'" class="bne-grads">
      <button
        v-for="(css, i) in BANNER_GRADIENTS"
        :key="i"
        type="button"
        class="bne-gsw"
        :class="{ on: b.gradient === i }"
        :aria-label="t('theme.gradientPreset', { n: i + 1 })"
        :style="{ background: css }"
        @click="patch({ gradient: i })"
      />
    </div>

    <p class="bne-note">{{ t('theme.editor.bannerShared') }}</p>
  </div>
</template>

<script setup lang="ts">
import { ArrowPathIcon, ArrowUpTrayIcon } from '@heroicons/vue/24/outline'
import type { ThemeBanner } from '@/types/v1/theme'
import { BANNER_GRADIENTS, BANNER_INKS, BANNER_TYPES, assetUrl } from '@/theme/content'
import { useThemeAssets } from '@/theme/useThemeAssets'
import { useThemeEditor } from './useThemeEditor'

const ed = useThemeEditor()
const { t } = ed
const { images, upload, uploading } = useThemeAssets()

const b = computed(() => ed.bannerModel.value)
const pendingImage = ref(false)
const patch = (p: Partial<ThemeBanner>) => {
  ed.bannerModel.value = { ...b.value, ...p }
  if (p.type) pendingImage.value = false
}
const setType = (k: string) => {
  if (k === 'image' && !b.value.image) {
    if (images.value[0]) patch({ type: 'image', image: images.value[0].id })
    else pendingImage.value = true
    return
  }
  pendingImage.value = false
  patch({ type: k })
}

const onPick = async (e: Event) => {
  const input = e.target as HTMLInputElement
  const f = input.files?.[0]
  input.value = ''
  if (!f) return
  const a = await upload(f, 'image')
  if (a) patch({ type: 'image', image: a.id })
}

const focusRef = ref<HTMLElement>()
const startFocus = (e: PointerEvent) => {
  const el = focusRef.value
  if (!el) return
  e.preventDefault()
  const move = (ev: PointerEvent) => {
    const r = el.getBoundingClientRect()
    patch({
      focusX: Math.round(Math.min(1, Math.max(0, (ev.clientX - r.left) / r.width)) * 100),
      focusY: Math.round(Math.min(1, Math.max(0, (ev.clientY - r.top) / r.height)) * 100),
    })
  }
  move(e)
  const up = () => {
    window.removeEventListener('pointermove', move)
    window.removeEventListener('pointerup', up)
  }
  window.addEventListener('pointermove', move)
  window.addEventListener('pointerup', up)
}
const onFocusKey = (e: KeyboardEvent) => {
  const step = e.shiftKey ? 10 : 2
  const clamp = (n: number) => Math.max(0, Math.min(100, n))
  if (e.key === 'ArrowLeft') patch({ focusX: clamp(b.value.focusX - step) })
  else if (e.key === 'ArrowRight') patch({ focusX: clamp(b.value.focusX + step) })
  else if (e.key === 'ArrowUp') patch({ focusY: clamp(b.value.focusY - step) })
  else if (e.key === 'ArrowDown') patch({ focusY: clamp(b.value.focusY + step) })
  else return
  e.preventDefault()
}
</script>

<style scoped lang="scss">
.bne {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.bne-head {
  display: flex;
  align-items: center;
  gap: 8px;
  .bne-lbl {
    flex: 1;
  }
}
.bne-lbl {
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--m-ink2);
}
.bne-tag {
  display: inline-flex;
  align-items: center;
  height: 18px;
  padding: 0 7px;
  border-radius: 999px;
  background: var(--m-s2);
  color: var(--m-ink2);
  font-size: 0.625rem;
  font-weight: 800;
  white-space: nowrap;
  &.is-page {
    background: var(--m-aSoft);
    color: var(--m-aOn);
  }
}
.bne-seg {
  display: flex;
  gap: 2px;
  padding: 4px;
  border-radius: 999px;
  background: var(--m-s2);
  button {
    flex: 1;
    min-height: 34px;
    padding: 0 10px;
    border: 0;
    border-radius: 999px;
    background: transparent;
    font: inherit;
    font-size: 0.8125rem;
    font-weight: 700;
    color: var(--m-ink2);
    cursor: pointer;
  }
  .on {
    background: var(--m-s);
    color: var(--m-ink);
    box-shadow: 0 1px 3px rgba(20, 10, 30, 0.12);
  }
}
.bne-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}
.bne-thumb {
  position: relative;
  aspect-ratio: 16 / 10;
  padding: 0;
  border-radius: 12px;
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
.bne-up {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  border-style: dashed;
  color: var(--m-ink2);
  svg {
    width: 16px;
    height: 16px;
  }
  b {
    font-size: 0.6875rem;
    color: var(--m-ink);
  }
  &:focus-within {
    box-shadow: var(--app-focus-ring);
  }
}
.bne-file {
  position: absolute;
  width: 1px;
  height: 1px;
  opacity: 0;
  pointer-events: none;
}
.bne-hint {
  margin: 0;
  font-size: 0.6875rem;
  color: var(--m-ink2);
}
.bne-focus {
  position: relative;
  height: 90px;
  border-radius: 12px;
  border: 1px solid var(--m-line);
  background-size: cover;
  background-position: center;
  cursor: crosshair;
  touch-action: none;
  span {
    position: absolute;
    width: 22px;
    height: 22px;
    margin: -11px 0 0 -11px;
    border-radius: 50%;
    border: 3px solid #ffffff;
    box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.35), 0 2px 6px rgba(0, 0, 0, 0.4);
    pointer-events: none;
  }
}
.bne-range {
  display: flex;
  flex-direction: column;
  gap: 4px;
  > span {
    display: flex;
    font-size: 0.75rem;
    font-weight: 700;
    span {
      flex: 1;
    }
    b {
      color: var(--m-pInk);
    }
  }
  input {
    width: 100%;
    height: 24px;
    margin: 0;
    accent-color: var(--m-p);
    cursor: pointer;
  }
}
.bne-col {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.bne-grads {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}
.bne-gsw {
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
.bne-note {
  margin: 0;
  padding: 10px 12px;
  border-radius: 14px;
  background: var(--m-s2);
  font-size: 0.75rem;
  line-height: 1.6;
  color: var(--m-ink2);
}
</style>
