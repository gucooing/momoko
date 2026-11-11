<!-- 上传队列卡（原型：文件管理左栏目录树下方 / 手机列表上方）：当前文件的分片格子进度 + 速度 / 剩余时间 + 暂停续传，
     其余文件一行一个（已完成 / 排队中 / 失败…）。compact=true 为手机版：只展示当前文件。 -->
<template>
  <section class="fuq" :class="{ 'is-compact': compact }" :aria-label="t('fileManager.uploadQueue')">
    <!-- 手机：单卡 -->
    <template v-if="compact">
      <div v-if="cur" class="fuq-m">
        <div class="fuq-m__row">
          <span class="fuq-sticker m-sticker m-sticker--p" style="transform: rotate(-4deg)">
            <UIcon name="i-lucide-upload" />
          </span>
          <span class="fuq-m__text">
            <span class="fuq-m__name">{{ cur.name }}</span>
            <span class="fuq-m__meta m-num">{{ compactMeta }}</span>
          </span>
          <span class="fuq-m__pct m-num">{{ cur.progress }}%</span>
          <AppIconButton
            v-if="canToggle(cur)"
            :icon="cur.status === 'paused' ? 'HOutline:PlayIcon' : 'HOutline:PauseIcon'"
            :label="cur.status === 'paused' ? t('fileManager.resumeUpload') : t('fileManager.pauseUpload')"
            :box="44"
            :size="20"
            @click="togglePause(cur)"
          />
        </div>
        <div class="fuq-cells fuq-cells--m" role="img" :aria-label="partsLabel">
          <span v-for="(c, i) in cells" :key="i" class="fuq-cell" :class="`is-${c}`" />
        </div>
        <div v-if="others.length" class="fuq-m__more m-num">{{ t('fileManager.uploadMore', { n: others.length }) }}</div>
      </div>
    </template>

    <!-- PC：队列卡 -->
    <template v-else>
      <header class="fuq-head">
        <span class="fuq-sticker m-sticker m-sticker--p" style="transform: rotate(-4deg)">
          <UIcon name="i-lucide-upload" />
        </span>
        <!-- 左栏只有 236px：标题区本身就是展开/收起按钮，省下一个图标按钮的宽度，标题不再折行 -->
        <button
          type="button"
          class="fuq-head__text"
          :aria-expanded="open"
          :title="open ? t('fileManager.collapseQueue') : t('fileManager.expandQueue')"
          @click="open = !open"
        >
          <span class="fuq-head__title">
            {{ title }}
            <UIcon :name="open ? 'i-lucide-chevron-down' : 'i-lucide-chevron-up'" class="fuq-head__chev" />
          </span>
          <span class="fuq-head__sub m-num">{{ subtitle }}</span>
        </button>
        <ActionMenu :items="menuItems" :label="t('fileManager.uploadQueue')" @select="onMenu" />
      </header>

      <template v-if="open">
        <div v-if="cur" class="fuq-cur">
          <div class="fuq-cur__row">
            <span class="fuq-sticker fuq-sticker--sm m-sticker" :class="`m-sticker--${visualOf(cur.name).tone}`">
              <UIcon :name="visualOf(cur.name).icon" />
            </span>
            <span class="fuq-cur__name" :title="cur.name">{{ cur.name }}</span>
            <AppIconButton
              v-if="canToggle(cur)"
              :icon="cur.status === 'paused' ? 'HOutline:PlayIcon' : 'HOutline:PauseIcon'"
              :label="cur.status === 'paused' ? t('fileManager.resumeUpload') : t('fileManager.pauseUpload')"
              :box="32"
              :size="16"
              @click="togglePause(cur)"
            />
            <AppIconButton
              v-if="cur.status !== 'finishing'"
              icon="HOutline:XMarkIcon"
              :label="t('system.common.cancel')"
              :box="32"
              :size="16"
              @click="upload.cancel(cur)"
            />
          </div>
          <div class="fuq-cells" role="img" :aria-label="partsLabel">
            <span v-for="(c, i) in cells" :key="i" class="fuq-cell" :class="`is-${c}`" />
          </div>
          <div class="fuq-cur__line m-num">
            <span><b>{{ cur.progress }}%</b><template v-if="cur.totalParts"> · {{ t('fileManager.partsDone', { done: doneParts, total: cur.totalParts }) }}</template></span>
            <span>{{ speedText }}</span>
          </div>
          <div class="fuq-cur__line m-num">
            <span>{{ formatFileSize(cur.uploadedBytes) }} / {{ formatFileSize(cur.size) }} · {{ etaText }}</span>
          </div>
        </div>

        <ul v-if="others.length" class="fuq-list">
          <li v-for="item in others" :key="item.id" class="fuq-item">
            <UIcon :name="statusIcon(item)" class="fuq-item__ico" :class="`is-${item.status}`" />
            <span class="fuq-item__name" :title="item.error || item.name">{{ item.name }}</span>
            <span class="fuq-item__status m-num" :class="`is-${item.status}`">{{ statusLabel(item) }}</span>
            <button
              v-if="item.status === 'failed' || item.status === 'canceled'"
              type="button"
              class="fuq-item__act"
              :aria-label="t('fileManager.retryUpload', { name: item.name })"
              :title="t('fileManager.retry')"
              @click="upload.retry(item)"
            >
              <UIcon name="i-lucide-rotate-ccw" />
            </button>
            <button
              v-else-if="item.status === 'paused'"
              type="button"
              class="fuq-item__act"
              :aria-label="t('fileManager.resumeUpload')"
              :title="t('fileManager.resumeUpload')"
              @click="upload.resume(item)"
            >
              <UIcon name="i-lucide-play" />
            </button>
            <button
              type="button"
              class="fuq-item__act"
              :aria-label="t('fileManager.removeUpload', { name: item.name })"
              :title="t('fileManager.remove')"
              @click="item.status === 'waiting' ? upload.cancel(item) : upload.removeItem(item.id)"
            >
              <UIcon name="i-lucide-x" />
            </button>
          </li>
        </ul>
      </template>
    </template>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { formatFileSize, resolveFilePreviewKind } from '@/utils/file'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import type { FileUploadQueue, UploadItem } from './useFileUpload'

const props = withDefaults(defineProps<{ upload: FileUploadQueue; compact?: boolean }>(), { compact: false })

const { t } = useI18n()
const open = ref(true)

const cur = computed(() => props.upload.current.value)
const others = computed(() => props.upload.items.value.filter((item) => item !== cur.value))
const doneParts = computed(() => cur.value?.parts.filter((p) => p === 2).length ?? 0)

// 分片格子：分片过多时合并显示（一格代表若干片：全部完成=完成、有在传=传输中）。
const MAX_CELLS = computed(() => (props.compact ? 36 : 48))
type Cell = 'done' | 'active' | 'idle'
const cells = computed<Cell[]>(() => {
  const item = cur.value
  if (!item) return []
  const parts = item.parts
  if (!parts.length) {
    const n = Math.min(MAX_CELLS.value, 12)
    const filled = Math.round((item.progress / 100) * n)
    return Array.from({ length: n }, (_, i) => (i < filled ? 'done' : 'idle'))
  }
  const count = Math.min(MAX_CELLS.value, parts.length)
  const per = parts.length / count
  return Array.from({ length: count }, (_, i) => {
    const slice = parts.slice(Math.floor(i * per), Math.max(Math.floor(i * per) + 1, Math.floor((i + 1) * per)))
    if (slice.every((p) => p === 2)) return 'done'
    if (slice.some((p) => p === 1)) return 'active'
    return 'idle'
  })
})
const partsLabel = computed(() =>
  cur.value?.totalParts
    ? t('fileManager.partsAria', { done: doneParts.value, total: cur.value.totalParts })
    : t('fileManager.uploadQueue'),
)

const finished = computed(() => props.upload.items.value.filter((i) => i.status === 'done').length)
const title = computed(() => {
  const total = props.upload.items.value.length
  if (props.upload.unfinishedCount.value) {
    return t('fileManager.uploadingCount', { done: finished.value + 1 > total ? total : finished.value + 1, total })
  }
  return t('fileManager.uploadFinishedCount', { done: finished.value, total })
})
const subtitle = computed(() => {
  const size = cur.value?.partSize
  if (size) return t('fileManager.chunkHint', { size: formatFileSize(size), threads: props.upload.threads.value })
  return t('fileManager.threadsHint', { threads: props.upload.threads.value })
})

const formatDuration = (sec: number) => {
  if (!Number.isFinite(sec) || sec <= 0) return ''
  if (sec < 60) return t('fileManager.etaSeconds', { n: Math.ceil(sec) })
  if (sec < 3600) return t('fileManager.etaMinutes', { n: Math.ceil(sec / 60) })
  return t('fileManager.etaHours', { n: Math.round((sec / 3600) * 10) / 10 })
}
const speedText = computed(() => {
  const item = cur.value
  if (!item) return ''
  if (item.status === 'paused') return t('fileManager.uploadPaused')
  if (item.status !== 'uploading') return item.statusText
  return item.speed ? `${formatFileSize(item.speed)}/s` : item.statusText
})
const etaText = computed(() => {
  const item = cur.value
  if (!item) return ''
  if (item.status === 'paused') return t('fileManager.resumeAnytime')
  if (item.status === 'waiting') return t('fileManager.waitUpload')
  return formatDuration(item.eta) || item.statusText
})
const compactMeta = computed(() => {
  const item = cur.value
  if (!item) return ''
  const parts = item.totalParts ? `${t('fileManager.chunkShort')} ${doneParts.value}/${item.totalParts}` : item.statusText
  return `${parts} · ${speedText.value}`
})

const canToggle = (item: UploadItem) =>
  item.status === 'paused' || item.status === 'waiting' || item.status === 'hashing' || item.status === 'uploading' || item.status === 'verifying'
const togglePause = (item: UploadItem) => {
  if (item.status === 'paused') props.upload.resume(item)
  else props.upload.pause(item)
}

const statusIcon = (item: UploadItem) =>
  ({
    done: 'i-lucide-circle-check',
    failed: 'i-lucide-circle-alert',
    canceled: 'i-lucide-circle-x',
    paused: 'i-lucide-circle-pause',
    waiting: 'i-lucide-clock',
  })[item.status as 'done'] ?? 'i-lucide-loader-circle'
const statusLabel = (item: UploadItem) => {
  if (item.status === 'done') return t('fileManager.uploadDoneShort')
  if (item.status === 'waiting') return t('fileManager.queued')
  return item.statusText
}

const visualOf = (name: string) => {
  const kind = resolveFilePreviewKind(name)
  if (kind === 'image') return { icon: 'i-lucide-image', tone: 'ok' }
  if (kind === 'video') return { icon: 'i-lucide-film', tone: 'p' }
  if (kind === 'audio') return { icon: 'i-lucide-music', tone: 'a2' }
  if (/\.(zip|tar|gz|tgz|rar|7z|bz2|xz)$/i.test(name)) return { icon: 'i-lucide-file-archive', tone: 'warn' }
  return { icon: 'i-lucide-file', tone: 'a2' }
}

const THREADS = [1, 2, 3, 4, 6]
const menuItems = computed<ActionMenuItem[]>(() => [
  ...THREADS.map((n) => ({
    key: `threads:${n}`,
    label: t('fileManager.threadsOption', { n }),
    icon: props.upload.threads.value === n ? 'i-lucide-check' : 'i-lucide-gauge',
  })),
  { key: 'retry', label: t('fileManager.retryFailedItems'), icon: 'i-lucide-rotate-ccw', hidden: !props.upload.hasFailed.value },
  { key: 'clear', label: t('fileManager.clearFinished'), icon: 'i-lucide-list-x' },
  { key: 'cancelAll', label: t('fileManager.cancelAllUploads'), icon: 'i-lucide-circle-x', danger: true, hidden: !props.upload.unfinishedCount.value },
])
// 并发数是上传队列自己的状态（ref），由队列菜单直接调整
const threads = props.upload.threads
const onMenu = (key: string) => {
  if (key.startsWith('threads:')) threads.value = Number(key.slice(8))
  else if (key === 'retry') void props.upload.retryFailed()
  else if (key === 'clear') props.upload.clearFinished()
  else if (key === 'cancelAll') props.upload.cancelAll()
}
</script>

<style scoped lang="scss">
.fuq {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: calc(var(--cmp-card-unit) * 14);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
  box-shadow: var(--app-shadow-sm);
}
.fuq.is-compact {
  padding: 12px 14px;
  border-radius: var(--app-radius-lg);
}
.fuq-sticker {
  flex: none;
  width: 32px;
  height: 32px;
  border-radius: 10px;
}
.fuq-sticker :deep(svg),
.fuq-sticker :deep(.iconify) {
  width: 16px;
  height: 16px;
}
.fuq-sticker--sm {
  width: 28px;
  height: 28px;
  border-radius: 9px;
}
.fuq-head {
  display: flex;
  align-items: center;
  gap: 8px;
}
.fuq-head__text {
  flex: 1;
  min-width: 0;
  min-height: 40px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.fuq-head__title {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 0.875rem;
  font-weight: 800;
  white-space: nowrap;
}
.fuq-head__chev {
  width: 14px;
  height: 14px;
  flex: none;
  color: var(--el-text-color-secondary);
}
.fuq-head__sub {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fuq-cur {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px;
  border-radius: var(--app-radius);
  background: var(--m-s2);
}
.fuq-cur__row {
  display: flex;
  align-items: center;
  gap: 6px;
}
.fuq-cur__name {
  flex: 1;
  min-width: 0;
  font-size: 0.8125rem;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fuq-cur__line {
  display: flex;
  justify-content: space-between;
  gap: 6px;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.fuq-cur__line b {
  color: var(--el-text-color-primary);
}
.fuq-cells {
  display: grid;
  grid-template-columns: repeat(12, minmax(0, 1fr));
  gap: 3px;
}
.fuq-cells--m {
  grid-template-columns: repeat(18, minmax(0, 1fr));
}
.fuq-cell {
  height: 8px;
  border-radius: 3px;
  background: var(--m-line);
}
.fuq-cells--m .fuq-cell {
  height: 7px;
}
.fuq-cell.is-done {
  background: var(--m-p);
}
.fuq-cell.is-active {
  background: var(--m-a);
}
.fuq-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 0;
  padding: 0;
  list-style: none;
}
.fuq-item {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 32px;
  padding: 0 0 0 4px;
}
.fuq-item__ico {
  flex: none;
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.fuq-item__ico.is-done {
  color: var(--m-ok);
}
.fuq-item__ico.is-failed {
  color: var(--m-bad);
}
.fuq-item__name {
  flex: 1;
  min-width: 0;
  font-size: 0.8125rem;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fuq-item__status {
  flex: none;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.fuq-item__status.is-done {
  color: var(--m-ok);
}
.fuq-item__status.is-failed {
  color: var(--m-bad);
}
.fuq-item__act {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 9px;
  background: transparent;
  color: var(--el-text-color-secondary);
  cursor: pointer;
}
.fuq-item__act:hover {
  background: var(--m-s2);
  color: var(--el-text-color-primary);
}
.fuq-item__act :deep(svg),
.fuq-item__act :deep(.iconify) {
  width: 14px;
  height: 14px;
}

/* 手机 */
.fuq-m {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.fuq-m__row {
  display: flex;
  align-items: center;
  gap: 10px;
}
.fuq-m .fuq-sticker {
  width: 34px;
  height: 34px;
  border-radius: 11px;
}
.fuq-m__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.fuq-m__name {
  font-size: 0.875rem;
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fuq-m__meta {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fuq-m__pct {
  font-family: var(--app-font-display);
  font-size: 1.125rem;
  font-weight: 800;
}
.fuq-m__more {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
</style>
