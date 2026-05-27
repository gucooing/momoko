<script setup lang="ts">
import { RefreshLeft, RefreshRight, ZoomIn, ZoomOut } from '@element-plus/icons-vue'

defineOptions({ name: 'FileMediaDialogContent' })

export type FileMediaKind = 'image' | 'audio' | 'video'

const props = defineProps<{
  kind: FileMediaKind
  objectUrl: string
}>()

const isImage = computed(() => props.kind === 'image')
const isAudio = computed(() => props.kind === 'audio')

const scale = ref(1)
const rotation = ref(0)
const panX = ref(0)
const panY = ref(0)
const isDragging = ref(false)
const dragStartX = ref(0)
const dragStartY = ref(0)
const lastPanX = ref(0)
const lastPanY = ref(0)

const MIN_SCALE = 0.1
const MAX_SCALE = 10
const ZOOM_STEP = 0.25

const transformStyle = computed(() => ({
  transform: `translate(${panX.value}px, ${panY.value}px) scale(${scale.value}) rotate(${rotation.value}deg)`,
  cursor: isDragging.value ? 'grabbing' : scale.value > 1 ? 'grab' : undefined,
  transition: isDragging.value ? 'none' : 'transform 0.15s ease',
}))

function zoomIn() {
  scale.value = Math.min(MAX_SCALE, +(scale.value + ZOOM_STEP).toFixed(2))
}

function zoomOut() {
  scale.value = Math.max(MIN_SCALE, +(scale.value - ZOOM_STEP).toFixed(2))
}

function rotateLeft() {
  rotation.value = (rotation.value - 90 + 360) % 360
}

function rotateRight() {
  rotation.value = (rotation.value + 90) % 360
}

function reset() {
  scale.value = 1
  rotation.value = 0
  panX.value = 0
  panY.value = 0
}

function onWheel(e: WheelEvent) {
  e.preventDefault()
  const delta = e.deltaY > 0 ? -ZOOM_STEP : ZOOM_STEP
  scale.value = Math.max(MIN_SCALE, Math.min(MAX_SCALE, +(scale.value + delta).toFixed(2)))
}

function onMouseDown(e: MouseEvent) {
  if (scale.value <= 1 && rotation.value % 360 === 0) return
  isDragging.value = true
  dragStartX.value = e.clientX
  dragStartY.value = e.clientY
  lastPanX.value = panX.value
  lastPanY.value = panY.value
}

function onMouseMove(e: MouseEvent) {
  if (!isDragging.value) return
  panX.value = lastPanX.value + (e.clientX - dragStartX.value)
  panY.value = lastPanY.value + (e.clientY - dragStartY.value)
}

function onMouseUp() {
  isDragging.value = false
}

watch(() => props.objectUrl, () => {
  reset()
})
</script>

<template>
  <section class="file-media-content" :class="[`is-${props.kind}`]">
    <template v-if="isImage">
      <div class="image-toolbar">
        <button class="image-toolbar-btn" title="放大" @click="zoomIn">
          <el-icon size="18"><ZoomIn /></el-icon>
        </button>
        <button class="image-toolbar-btn" title="缩小" @click="zoomOut">
          <el-icon size="18"><ZoomOut /></el-icon>
        </button>
        <button class="image-toolbar-btn" title="逆时针旋转" @click="rotateLeft">
          <el-icon size="18"><RefreshLeft /></el-icon>
        </button>
        <button class="image-toolbar-btn" title="顺时针旋转" @click="rotateRight">
          <el-icon size="18"><RefreshRight /></el-icon>
        </button>
        <span class="image-toolbar-info">{{ Math.round(scale * 100) }}% | {{ rotation }}°</span>
        <button class="image-toolbar-btn image-toolbar-reset" title="重置" @click="reset">重置</button>
      </div>
      <div
        class="image-viewer"
        @wheel.prevent="onWheel"
        @mousedown="onMouseDown"
        @mousemove="onMouseMove"
        @mouseup="onMouseUp"
        @mouseleave="onMouseUp"
      >
        <img
          :src="props.objectUrl"
          :style="transformStyle"
          class="image-viewer-img"
          draggable="false"
          @dragstart.prevent
        />
      </div>
    </template>

    <audio
      v-else-if="isAudio"
      class="file-media-player file-media-audio"
      :src="props.objectUrl"
      controls
      preload="metadata"
    />

    <video
      v-else
      class="file-media-player file-media-video"
      :src="props.objectUrl"
      controls
      playsinline
      preload="metadata"
    />
  </section>
</template>

<style scoped lang="scss">
.file-media-content {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  user-select: none;
}

.file-media-content.is-image {
  width: 100%;
  height: 100%;
}

.file-media-content.is-audio {
  width: 100%;
}

.image-toolbar {
  position: absolute;
  bottom: 1.25rem;
  left: 50%;
  transform: translateX(-50%);
  z-index: 10;
  display: flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.5rem 0.75rem;
  border-radius: 999px;
  background: rgb(0 0 0 / 65%);
  backdrop-filter: blur(8px);
}

.image-toolbar-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2.25rem;
  height: 2.25rem;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: #fff;
  cursor: pointer;
  transition: background 0.15s;

  &:hover {
    background: rgb(255 255 255 / 18%);
  }
}

.image-toolbar-reset {
  width: auto;
  padding: 0 0.85rem;
  border-radius: 999px;
  font-size: 0.8rem;
}

.image-toolbar-info {
  padding: 0 0.4rem;
  color: rgb(255 255 255 / 70%);
  font-size: 0.75rem;
  font-variant-numeric: tabular-nums;
  min-width: 6.5rem;
  text-align: center;
}

.image-viewer {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.image-viewer-img {
  max-width: 92vw;
  max-height: 92vh;
  object-fit: contain;
  pointer-events: auto;
}

.file-media-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20rem;
  height: 16rem;
  color: rgb(255 255 255 / 55%);
  font-size: 0.9rem;
}

.file-media-player {
  display: block;
}

.file-media-video {
  max-width: 92vw;
  max-height: 92vh;
  background: #000;
}

.file-media-audio {
  width: min(100%, 42rem);
}
</style>
