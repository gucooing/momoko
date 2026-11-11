<!-- 取色器（主题编辑器右栏 / 手机底部面板）：饱和度方块 + 色相条 + HEX 输入 + 推荐色。
     variant=compact（手机）只有色相条、推荐色和 HEX。拖动时连续触发 update:modelValue。 -->
<template>
  <div class="cpk" :class="`cpk--${variant}`">
    <div
      v-if="variant === 'full'"
      ref="svRef"
      class="cpk-sv"
      role="slider"
      tabindex="0"
      :aria-label="t('theme.editor.saturation')"
      :aria-valuetext="hex"
      :style="{ background: `linear-gradient(to top, #000, transparent), linear-gradient(to right, #fff, hsl(${Math.round(hsv.h)}, 100%, 50%))` }"
      @pointerdown="startSv"
      @keydown="onSvKey"
    >
      <span :style="{ left: `${hsv.s * 100}%`, top: `${(1 - hsv.v) * 100}%`, background: hex }" />
    </div>
    <div
      ref="hueRef"
      class="cpk-hue"
      role="slider"
      tabindex="0"
      :aria-label="t('theme.editor.hue')"
      aria-valuemin="0"
      aria-valuemax="360"
      :aria-valuenow="Math.round(hsv.h)"
      @pointerdown="startHue"
      @keydown="onHueKey"
    >
      <span :style="{ left: `${(hsv.h / 360) * 100}%` }" />
    </div>
    <div v-if="variant === 'full'" class="cpk-hexrow">
      <span class="cpk-big" :style="{ background: hex }" />
      <label class="cpk-hex">
        <b>HEX</b>
        <input class="m-mono" type="text" :value="hex" :aria-label="t('theme.editor.hexLabel')" @change="onHex" @keydown.enter="onHex" />
      </label>
    </div>
    <div v-if="suggestions.length" class="cpk-sug">
      <button
        v-for="c in suggestions"
        :key="c"
        type="button"
        class="cpk-sgw"
        :class="{ on: c.toUpperCase() === hex.toUpperCase() }"
        :aria-label="c"
        :style="{ background: c }"
        @click="emit('update:modelValue', c.toUpperCase())"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { hexToHsv, hsvToHex, isHex, normalizeHex, type HSV } from '@/theme/color'

const props = withDefaults(defineProps<{ modelValue: string; suggestions?: string[]; variant?: 'full' | 'compact' }>(), {
  suggestions: () => [],
  variant: 'full',
})
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const { t } = useI18n()

const hex = computed(() => (isHex(props.modelValue) ? props.modelValue.toUpperCase() : '#000000'))
// 拖动时保留色相（纯灰时 hex → hsv 会丢掉色相）
const local = ref<HSV>(hexToHsv(hex.value))
watch(hex, (h) => {
  if (hsvToHex(local.value) !== h) local.value = hexToHsv(h)
})
const hsv = computed(() => local.value)

const set = (next: HSV) => {
  local.value = next
  emit('update:modelValue', hsvToHex(next))
}

const svRef = ref<HTMLElement>()
const hueRef = ref<HTMLElement>()

const drag = (el: HTMLElement | undefined, e: PointerEvent, apply: (x: number, y: number) => void) => {
  if (!el) return
  e.preventDefault()
  const move = (ev: PointerEvent) => {
    const r = el.getBoundingClientRect()
    apply(Math.min(1, Math.max(0, (ev.clientX - r.left) / r.width)), Math.min(1, Math.max(0, (ev.clientY - r.top) / r.height)))
  }
  move(e)
  const up = () => {
    window.removeEventListener('pointermove', move)
    window.removeEventListener('pointerup', up)
  }
  window.addEventListener('pointermove', move)
  window.addEventListener('pointerup', up)
}
const startSv = (e: PointerEvent) => drag(svRef.value, e, (x, y) => set({ ...hsv.value, s: x, v: 1 - y }))
const startHue = (e: PointerEvent) => drag(hueRef.value, e, (x) => set({ ...hsv.value, h: Math.min(359.9, x * 360) }))

const onSvKey = (e: KeyboardEvent) => {
  const step = e.shiftKey ? 0.1 : 0.02
  const h = hsv.value
  if (e.key === 'ArrowLeft') set({ ...h, s: Math.max(0, h.s - step) })
  else if (e.key === 'ArrowRight') set({ ...h, s: Math.min(1, h.s + step) })
  else if (e.key === 'ArrowUp') set({ ...h, v: Math.min(1, h.v + step) })
  else if (e.key === 'ArrowDown') set({ ...h, v: Math.max(0, h.v - step) })
  else return
  e.preventDefault()
}
const onHueKey = (e: KeyboardEvent) => {
  const step = e.shiftKey ? 15 : 3
  const h = hsv.value
  if (e.key === 'ArrowLeft' || e.key === 'ArrowDown') set({ ...h, h: (h.h - step + 360) % 360 })
  else if (e.key === 'ArrowRight' || e.key === 'ArrowUp') set({ ...h, h: (h.h + step) % 360 })
  else return
  e.preventDefault()
}

const onHex = (e: Event) => {
  const input = e.target as HTMLInputElement
  const v = normalizeHex(input.value)
  if (v) emit('update:modelValue', v)
  else input.value = hex.value
}
</script>

<style scoped lang="scss">
.cpk {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.cpk-sv {
  position: relative;
  height: 150px;
  border-radius: 14px;
  cursor: crosshair;
  touch-action: none;
  span {
    position: absolute;
    width: 18px;
    height: 18px;
    margin: -9px 0 0 -9px;
    border-radius: 50%;
    border: 3px solid #ffffff;
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.4);
    pointer-events: none;
  }
}
.cpk-hue {
  position: relative;
  height: 14px;
  border-radius: 999px;
  background: linear-gradient(90deg, #f00, #ff0, #0f0, #0ff, #00f, #f0f, #f00);
  cursor: pointer;
  touch-action: none;
  span {
    position: absolute;
    top: 50%;
    width: 18px;
    height: 18px;
    margin: -9px 0 0 -9px;
    border-radius: 50%;
    background: #ffffff;
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.4);
    pointer-events: none;
  }
}
.cpk--compact .cpk-hue {
  height: 18px;
  span {
    width: 22px;
    height: 22px;
    margin: -11px 0 0 -11px;
  }
}
.cpk-hexrow {
  display: flex;
  align-items: center;
  gap: 8px;
}
.cpk-big {
  flex: none;
  width: 40px;
  height: 40px;
  border-radius: 12px;
  border: 1px solid var(--m-line);
}
.cpk-hex {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 40px;
  padding: 0 14px;
  border-radius: 12px;
  background: var(--m-s);
  border: 1.5px solid var(--m-line);
  b {
    font-size: 0.75rem;
  }
  input {
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
.cpk-sug {
  display: grid;
  grid-template-columns: repeat(8, minmax(0, 1fr));
  gap: 6px;
}
.cpk-sgw {
  aspect-ratio: 1;
  padding: 0;
  border-radius: 9px;
  border: 1px solid var(--m-line);
  cursor: pointer;
  &.on {
    box-shadow: 0 0 0 2px var(--m-s), 0 0 0 4px var(--m-p);
  }
}
</style>
