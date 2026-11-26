<!-- 媒体查看层（图片 / 视频 / 音频）：铺满窗口的暗底，顶栏是名称、缩放、在新标签页打开、下载、关闭。
     图片默认适应窗口；点图片在适应窗口和原始大小之间切换，滚轮以指针为中心缩放，放大后按住拖动平移。
     点空白处或按 Esc 关闭。给了 downloadUrl 才有下载按钮（外站图片下载不了，只能新标签页打开）。 -->
<template>
  <Teleport to="body">
    <Transition name="fmd">
      <div v-if="modelValue" class="fmd" role="dialog" aria-modal="true" :aria-label="name">
        <header class="fmd__bar" @click.self="zoomMenu = false">
          <span class="fmd__pill fmd__title">
            <UIcon :name="kindIcon" class="fmd__ico" />
            <span class="fmd__name">{{ name }}</span>
          </span>
          <span class="fmd__gap" @click="zoomMenu = false" />
          <div v-if="kind === 'image'" class="fmd__zoom">
            <button
              type="button"
              class="fmd__pill fmd__btn"
              aria-haspopup="menu"
              :aria-expanded="zoomMenu"
              :aria-label="t('fileManager.mediaZoom')"
              @click="zoomMenu = !zoomMenu"
            >
              <span class="fmd__num">{{ pct }}%</span>
              <UIcon name="i-lucide-chevron-down" class="fmd__ico" />
            </button>
            <div v-if="zoomMenu" class="fmd__menu" role="menu">
              <button type="button" class="fmd__mi" role="menuitemradio" :aria-checked="fit" @click="setFit">
                {{ t('fileManager.mediaFit') }}
              </button>
              <button
                v-for="z in ZOOMS"
                :key="z"
                type="button"
                class="fmd__mi"
                role="menuitemradio"
                :aria-checked="!fit && zoom === z"
                @click="pickZoom(z)"
              >
                {{ z * 100 }}%
              </button>
            </div>
          </div>
          <button type="button" class="fmd__pill fmd__btn" @click="openNew">
            <UIcon name="i-lucide-external-link" class="fmd__ico" />
            <span class="fmd__label">{{ t('fileManager.mediaOpen') }}</span>
          </button>
          <button
            v-if="downloadUrl"
            type="button"
            class="fmd__pill fmd__btn fmd__btn--icon"
            :aria-label="t('fileManager.download')"
            :title="t('fileManager.download')"
            @click="download"
          >
            <UIcon name="i-lucide-download" class="fmd__ico" />
          </button>
          <button
            type="button"
            class="fmd__pill fmd__btn fmd__btn--icon"
            :aria-label="t('common.close')"
            :title="t('common.close')"
            @click="close"
          >
            <UIcon name="i-lucide-x" class="fmd__ico" />
          </button>
        </header>

        <div
          ref="stage"
          class="fmd__stage"
          :class="{ 'is-pan': pannable, 'is-drag': dragging }"
          @click="onStageClick"
          @wheel="onWheel"
          @pointerdown="onDown"
          @pointermove="onMove"
          @pointerup="onUp"
          @pointercancel="onUp"
        >
          <img
            v-if="kind === 'image'"
            ref="img"
            :src="url"
            :alt="name"
            class="fmd__img"
            :class="{ 'is-sized': nat.w > 0, 'can-in': fit && fitScale < 1, 'can-out': !fit }"
            :style="imgStyle"
            draggable="false"
            @load="onLoad"
          />
          <video v-else-if="kind === 'video'" :src="url" class="fmd__video" controls autoplay />
          <audio v-else-if="kind === 'audio'" :src="url" class="fmd__audio" controls autoplay />
          <p v-else class="fmd__empty">{{ t('fileManager.cannotEditBinary') }}</p>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { downloadFileFromUrl, type FilePreviewKind } from '@/utils/file'

const props = defineProps<{
  modelValue: boolean
  name: string
  kind: FilePreviewKind
  url: string
  /** 下载地址；不给就没有下载按钮 */
  downloadUrl?: string
}>()

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const { t } = useI18n()

const ZOOMS = [0.5, 1, 2] as const
const MIN = 0.05
const MAX = 8
/** 适应窗口时四周留的空 */
const PAD = 48

const stage = ref<HTMLElement | null>(null)
const img = ref<HTMLImageElement | null>(null)
const nat = reactive({ w: 0, h: 0 })
const box = reactive({ w: 0, h: 0 })
const fit = ref(true)
const zoom = ref(1)
const zoomMenu = ref(false)
const dragging = ref(false)

const kindIcon = computed(
  () => ({ image: 'i-lucide-image', video: 'i-lucide-film', audio: 'i-lucide-music' })[props.kind] ?? 'i-lucide-file',
)
const fitScale = computed(() => {
  if (!nat.w || !nat.h || !box.w || !box.h) return 1
  return Math.max(MIN, Math.min(1, (box.w - PAD) / nat.w, (box.h - PAD) / nat.h))
})
const scale = computed(() => (fit.value ? fitScale.value : zoom.value))
const pct = computed(() => Math.round(scale.value * 100))
const imgStyle = computed(() =>
  nat.w ? { width: `${Math.round(nat.w * scale.value)}px`, height: `${Math.round(nat.h * scale.value)}px` } : undefined,
)
const pannable = computed(() => nat.w * scale.value > box.w - PAD || nat.h * scale.value > box.h - PAD)

const close = () => emit('update:modelValue', false)
// 浏览器不让直接打开 data: 地址：先换成 blob: 地址再开（同步做完，免得被当成弹窗拦掉）。
const openNew = () => {
  if (!props.url.startsWith('data:')) {
    window.open(props.url, '_blank', 'noopener')
    return
  }
  const comma = props.url.indexOf(',')
  const head = props.url.slice(5, comma)
  const body = props.url.slice(comma + 1)
  const type = head.split(';')[0] || 'application/octet-stream'
  let blob: Blob
  if (head.endsWith(';base64')) {
    const raw = atob(body)
    const bytes = new Uint8Array(raw.length)
    for (let i = 0; i < raw.length; i++) bytes[i] = raw.charCodeAt(i)
    blob = new Blob([bytes], { type })
  } else {
    blob = new Blob([decodeURIComponent(body)], { type })
  }
  const blobUrl = URL.createObjectURL(blob)
  window.open(blobUrl, '_blank', 'noopener')
  setTimeout(() => URL.revokeObjectURL(blobUrl), 60_000)
}
const download = () => {
  if (props.downloadUrl) downloadFileFromUrl(props.downloadUrl, props.name)
}

const measure = () => {
  const el = stage.value
  if (!el) return
  box.w = el.clientWidth
  box.h = el.clientHeight
}
const onLoad = () => {
  const el = img.value
  if (!el) return
  nat.w = el.naturalWidth
  nat.h = el.naturalHeight
  measure()
}

/** 缩放到 next，并让 at（屏幕坐标，默认画面中心）下的那一点留在原处。 */
const zoomTo = async (next: number, at?: { x: number; y: number }) => {
  const el = stage.value
  const im = img.value
  if (!el || !im || !nat.w) return
  const r = el.getBoundingClientRect()
  const x = at?.x ?? r.left + r.width / 2
  const y = at?.y ?? r.top + r.height / 2
  const before = im.getBoundingClientRect()
  const rx = before.width ? Math.min(1, Math.max(0, (x - before.left) / before.width)) : 0.5
  const ry = before.height ? Math.min(1, Math.max(0, (y - before.top) / before.height)) : 0.5
  fit.value = false
  zoom.value = Math.min(MAX, Math.max(MIN, next))
  await nextTick()
  const after = im.getBoundingClientRect()
  el.scrollLeft += after.left + rx * after.width - x
  el.scrollTop += after.top + ry * after.height - y
}
const setFit = () => {
  fit.value = true
  zoomMenu.value = false
}
const pickZoom = (z: number) => {
  zoomMenu.value = false
  void zoomTo(z)
}
const onWheel = (e: WheelEvent) => {
  if (props.kind !== 'image' || !nat.w) return
  e.preventDefault()
  void zoomTo(scale.value * Math.exp(-e.deltaY * 0.002), { x: e.clientX, y: e.clientY })
}
/** 点图片：适应窗口且图被缩小了就放到原始大小（点哪放大哪），已经放大过就回到适应窗口。 */
const toggleZoom = (e: MouseEvent) => {
  if (!fit.value) setFit()
  else if (fitScale.value < 1) void zoomTo(1, { x: e.clientX, y: e.clientY })
}

// 拖动平移：动了几像素才算拖，这时才捕获指针；拖完松手的那次点击不算点空白。
let drag: { id: number; x: number; y: number; left: number; top: number; moved: boolean } | null = null
let skipClick = false
const onDown = (e: PointerEvent) => {
  const el = stage.value
  if (e.button !== 0 || !el || props.kind !== 'image' || !pannable.value) return
  drag = { id: e.pointerId, x: e.clientX, y: e.clientY, left: el.scrollLeft, top: el.scrollTop, moved: false }
}
const onMove = (e: PointerEvent) => {
  const el = stage.value
  if (!drag || !el) return
  const dx = e.clientX - drag.x
  const dy = e.clientY - drag.y
  if (!drag.moved) {
    if (Math.hypot(dx, dy) < 4) return
    drag.moved = true
    dragging.value = true
    el.setPointerCapture(drag.id)
  }
  el.scrollLeft = drag.left - dx
  el.scrollTop = drag.top - dy
}
const onUp = () => {
  skipClick = drag?.moved ?? false
  drag = null
  dragging.value = false
}
const onStageClick = (e: MouseEvent) => {
  if (skipClick) {
    skipClick = false
    return
  }
  if (zoomMenu.value) {
    zoomMenu.value = false
    return
  }
  if (e.target === img.value) toggleZoom(e)
  else if (e.target === e.currentTarget) close()
}

const onKey = (e: KeyboardEvent) => {
  if (!props.modelValue || e.key !== 'Escape') return
  if (zoomMenu.value) zoomMenu.value = false
  else close()
}

watch(
  () => [props.modelValue, props.url] as const,
  ([open]) => {
    if (!open) return
    fit.value = true
    zoom.value = 1
    nat.w = 0
    nat.h = 0
    zoomMenu.value = false
    void nextTick(measure)
  },
  { immediate: true },
)

onMounted(() => {
  window.addEventListener('keydown', onKey)
  window.addEventListener('resize', measure)
})
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKey)
  window.removeEventListener('resize', measure)
})
</script>

<style scoped>
.fmd {
  position: fixed;
  inset: 0;
  z-index: 3000;
  background: rgba(12, 10, 16, 0.9);
  backdrop-filter: blur(6px);
  color: #f2f0f5;
}
.fmd__bar {
  position: absolute;
  inset: 0 0 auto;
  z-index: 1;
  display: flex;
  align-items: center;
  gap: 8px;
  height: 60px;
  padding: 0 14px;
}
.fmd__gap {
  flex: 1;
  align-self: stretch;
}
.fmd__pill {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  height: 30px;
  padding: 0 13px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 12px;
  background: rgba(38, 36, 44, 0.92);
  color: inherit;
  font: inherit;
  font-size: 13px;
  white-space: nowrap;
}
.fmd__title {
  min-width: 0;
  max-width: 42vw;
}
.fmd__name {
  overflow: hidden;
  text-overflow: ellipsis;
}
.fmd__ico {
  flex: none;
  width: 16px;
  height: 16px;
}
.fmd__btn {
  cursor: pointer;
}
.fmd__btn:hover,
.fmd__btn[aria-expanded='true'] {
  background: rgba(58, 55, 66, 0.96);
}
.fmd__btn--icon {
  width: 36px;
  padding: 0;
  justify-content: center;
}
.fmd__num {
  font-variant-numeric: tabular-nums;
}
.fmd__zoom {
  position: relative;
}
.fmd__menu {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  min-width: 140px;
  padding: 4px;
  display: flex;
  flex-direction: column;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 12px;
  background: rgba(38, 36, 44, 0.98);
  box-shadow: 0 16px 40px rgba(0, 0, 0, 0.45);
}
.fmd__mi {
  height: 32px;
  padding: 0 10px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: inherit;
  font: inherit;
  font-size: 13px;
  text-align: left;
  cursor: pointer;
}
.fmd__mi:hover {
  background: rgba(255, 255, 255, 0.08);
}
.fmd__mi[aria-checked='true'] {
  font-weight: 600;
}
.fmd__stage {
  position: absolute;
  inset: 60px 0 0;
  display: flex;
  overflow: auto;
  padding: 12px 16px 16px;
  scrollbar-width: thin;
  scrollbar-color: rgba(255, 255, 255, 0.25) transparent;
}
.fmd__stage.is-pan {
  cursor: grab;
}
.fmd__stage.is-drag {
  cursor: grabbing;
}
.fmd__img {
  flex: none;
  margin: auto;
  display: block;
  max-width: 100%;
  max-height: 100%;
  border-radius: 4px;
  box-shadow: 0 24px 60px rgba(0, 0, 0, 0.5);
  user-select: none;
}
.fmd__img.is-sized {
  max-width: none;
  max-height: none;
}
.fmd__stage:not(.is-pan) .fmd__img.can-in {
  cursor: zoom-in;
}
.fmd__stage:not(.is-pan) .fmd__img.can-out {
  cursor: zoom-out;
}
.fmd__video {
  margin: auto;
  max-width: 100%;
  max-height: 100%;
  border-radius: 8px;
}
.fmd__audio {
  margin: auto;
  width: min(560px, 90%);
}
.fmd__empty {
  margin: auto;
  color: rgba(255, 255, 255, 0.6);
  font-size: 13px;
}
.fmd-enter-active,
.fmd-leave-active {
  transition: opacity 0.15s ease;
}
.fmd-enter-from,
.fmd-leave-to {
  opacity: 0;
}
@media (width < 640px) {
  .fmd__label {
    display: none;
  }
  .fmd__title {
    max-width: 36vw;
  }
}
</style>
