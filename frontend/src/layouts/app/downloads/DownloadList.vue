<!-- 下载列表（顶栏「下载」弹层）：服务器从外网下载的文件，进行中的显示进度 / 速度 / 剩余时间，
     可暂停、继续、取消；结束的可从列表删除或一次清掉。只列出当前用户有权限的下载。 -->
<template>
  <div class="dl">
    <header class="dl__head">
      <span class="dl__title">{{ active > 0 ? t('downloads.titleActive', { n: active }) : t('downloads.title') }}</span>
      <button v-if="hasFinished" type="button" class="dl__action" @click="clearFinished">
        {{ t('downloads.clearFinished') }}
      </button>
    </header>

    <div class="dl__list">
      <EmptyState v-if="downloads.length === 0" icon="HOutline:ArrowDownTrayIcon" :title="t('downloads.empty')" />
      <article v-for="d in downloads" :key="d.id" class="dl-item" :class="`is-${stateKey(d)}`">
        <div class="dl-item__row">
          <span class="dl-item__sticker m-sticker" :class="`m-sticker--${tone(d)}`">
            <UIcon :name="icon(d)" />
          </span>
          <span class="dl-item__text">
            <span class="dl-item__name" :title="d.url">{{ d.title || d.name }}</span>
            <span class="dl-item__sub">
              <template v-if="d.title">{{ d.name }} · </template>{{ sourceLabel(d.source) }}<template v-if="d.operator"> · {{ t('downloads.by', { name: d.operator }) }}</template>
            </span>
          </span>
          <span class="dl-item__acts">
            <AppIconButton
              v-if="canPause(d)"
              icon="HOutline:PauseIcon"
              :label="t('downloads.pause')"
              :box="30"
              :size="16"
              :disabled="busy[d.id]"
              @click="act(d, 'pause')"
            />
            <AppIconButton
              v-if="canResume(d)"
              icon="HOutline:PlayIcon"
              :label="t('downloads.resume')"
              :box="30"
              :size="16"
              :disabled="busy[d.id]"
              @click="act(d, 'resume')"
            />
            <AppIconButton
              v-if="!isFinished(d)"
              icon="HOutline:XMarkIcon"
              :label="t('downloads.cancel')"
              :box="30"
              :size="16"
              :disabled="busy[d.id]"
              @click="confirmCancel(d)"
            />
            <AppIconButton
              v-else
              icon="HOutline:TrashIcon"
              :label="t('downloads.remove')"
              :box="30"
              :size="16"
              :disabled="busy[d.id]"
              @click="act(d, 'remove')"
            />
          </span>
        </div>

        <div v-if="!isFinished(d)" class="dl-item__bar" role="progressbar" :aria-valuenow="percentOf(d.downloaded, d.total) ?? undefined" aria-valuemin="0" aria-valuemax="100">
          <span
            class="dl-item__fill"
            :class="{ 'is-indeterminate': percentOf(d.downloaded, d.total) === null && stateKey(d) === 'downloading' }"
            :style="{ width: `${percentOf(d.downloaded, d.total) ?? (stateKey(d) === 'downloading' ? 30 : 0)}%` }"
          />
        </div>

        <div class="dl-item__meta m-num">
          <span class="dl-item__state">{{ stateLabel(d) }}</span>
          <span class="dl-item__detail" :title="detail(d)">{{ detail(d) }}</span>
        </div>
      </article>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { cancelDownload, clearDownloads, pauseDownload, removeDownload, resumeDownload } from '@/api/system'
import { DownloadState, type DownloadInfo } from '@/types/v1/system'
import { Dialog } from '@/utils/dialog'
import { formatFileSize } from '@/utils/file'
import { showRequestError } from '@/utils/request'
import { isFinished, percentOf, progressText, refreshDownloads, sourceLabel, stateKey, stateLabel, useDownloads } from './useDownloads'

const { t } = useI18n()
const { downloads, active } = useDownloads(true)
const busy = reactive<Record<string, boolean>>({})

const hasFinished = computed(() => downloads.value.some(isFinished))

const canPause = (d: DownloadInfo) =>
  d.state === DownloadState.DownloadState_Downloading || d.state === DownloadState.DownloadState_Retrying
const canResume = (d: DownloadInfo) => d.state === DownloadState.DownloadState_Paused

const tone = (d: DownloadInfo) => {
  switch (stateKey(d)) {
    case 'completed':
      return 'ok'
    case 'failed':
      return 'bad'
    case 'paused':
    case 'canceled':
      return 'a2'
    default:
      return 'p'
  }
}
const icon = (d: DownloadInfo) => {
  switch (stateKey(d)) {
    case 'completed':
      return 'i-lucide-check'
    case 'failed':
      return 'i-lucide-triangle-alert'
    case 'paused':
      return 'i-lucide-pause'
    case 'canceled':
      return 'i-lucide-ban'
    case 'retrying':
      return 'i-lucide-rotate-cw'
    default:
      return 'i-lucide-download'
  }
}

const detail = (d: DownloadInfo) => {
  switch (stateKey(d)) {
    case 'completed':
      return formatFileSize(d.downloaded)
    case 'failed':
      return d.error
    case 'canceled':
      return ''
    case 'retrying':
      return `${t('downloads.retrying', { n: d.retries })}${d.error ? `：${d.error}` : ''}`
    case 'paused':
      return d.resumable ? progressText({ downloaded: d.downloaded, total: d.total }) : t('downloads.noResume')
    default:
      return progressText(d)
  }
}

const act = async (d: DownloadInfo, action: 'pause' | 'resume' | 'cancel' | 'remove') => {
  busy[d.id] = true
  try {
    if (action === 'pause') await pauseDownload(d.id)
    else if (action === 'resume') await resumeDownload(d.id)
    else if (action === 'cancel') await cancelDownload(d.id)
    else await removeDownload(d.id)
    await refreshDownloads()
  } catch (e) {
    showRequestError(e)
  } finally {
    delete busy[d.id]
  }
}

const confirmCancel = (d: DownloadInfo) => {
  Dialog.confirm({
    title: t('downloads.cancelTitle'),
    content: t('downloads.cancelContent', { name: d.title || d.name }),
    confirmText: t('downloads.cancelConfirm'),
    cancelText: t('downloads.keep'),
    onConfirm: () => act(d, 'cancel'),
  })
}

const clearFinished = async () => {
  try {
    await clearDownloads()
    await refreshDownloads()
  } catch (e) {
    showRequestError(e)
  }
}
</script>

<style scoped lang="scss">
.dl__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 13px 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.dl__title {
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.dl__action {
  border: none;
  background: transparent;
  color: var(--el-color-primary);
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
}
.dl__list {
  max-height: 26rem;
  overflow-y: auto;
}
.dl-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 12px 12px 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.dl-item:last-child {
  border-bottom: none;
}
.dl-item__row {
  display: flex;
  align-items: center;
  gap: 10px;
}
.dl-item__sticker {
  flex: none;
  width: 30px;
  height: 30px;
  border-radius: 9px;
}
.dl-item__sticker :deep(svg),
.dl-item__sticker :deep(.iconify) {
  width: 15px;
  height: 15px;
}
.is-retrying .dl-item__sticker :deep(svg) {
  animation: dl-spin 1.2s linear infinite;
}
.dl-item__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.dl-item__name {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.dl-item__sub {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.dl-item__acts {
  display: flex;
  flex: none;
  gap: 2px;
}
.dl-item__bar {
  position: relative;
  height: 6px;
  overflow: hidden;
  border-radius: 999px;
  background: var(--m-line);
}
.dl-item__fill {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--m-p);
  transition: width 0.4s ease;
}
.is-paused .dl-item__fill {
  background: var(--el-text-color-placeholder);
}
.is-retrying .dl-item__fill {
  background: var(--m-a);
}
.dl-item__fill.is-indeterminate {
  animation: dl-slide 1.4s ease-in-out infinite;
}
.dl-item__meta {
  display: flex;
  gap: 8px;
  min-width: 0;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.dl-item__state {
  flex: none;
  font-weight: 600;
}
.is-completed .dl-item__state {
  color: var(--m-ok);
}
.is-failed .dl-item__state,
.is-failed .dl-item__detail {
  color: var(--m-bad);
}
.dl-item__detail {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
@keyframes dl-spin {
  to {
    transform: rotate(360deg);
  }
}
@keyframes dl-slide {
  from {
    transform: translateX(-100%);
  }
  to {
    transform: translateX(340%);
  }
}
@media (prefers-reduced-motion: reduce) {
  .is-retrying .dl-item__sticker :deep(svg),
  .dl-item__fill.is-indeterminate {
    animation: none;
  }
}
</style>
