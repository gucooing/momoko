import { listDownloads } from '@/api/system'
import { DownloadState, type DownloadInfo } from '@/types/v1/system'
import { translate } from '@/locales'
import { formatFileSize } from '@/utils/file'

// 下载管理（服务器从外网下载的文件）的共享轮询：顶栏按钮、下载列表、设置页共用一份数据。
// 有进行中的下载或列表开着时每秒刷新，平时 15 秒看一眼有没有新下载。
const IDLE_MS = 15_000
const ACTIVE_MS = 1_000

const downloads = ref<DownloadInfo[]>([])
const active = ref(0)
let users = 0
let watchers = 0
let timer: ReturnType<typeof setTimeout> | null = null

export const refreshDownloads = async () => {
  try {
    const { data } = await listDownloads()
    downloads.value = data?.downloads ?? []
    active.value = Number(data?.active ?? 0)
  } catch {
    // 忽略：只是展示
  }
}

const schedule = () => {
  if (timer) clearTimeout(timer)
  timer = null
  if (users <= 0) return
  timer = setTimeout(tick, active.value > 0 || watchers > 0 ? ACTIVE_MS : IDLE_MS)
}

const tick = async () => {
  await refreshDownloads()
  schedule()
}

/** watch=true：调用方正在看列表，按快的节奏刷新 */
export const useDownloads = (watch = false) => {
  onMounted(() => {
    users += 1
    if (watch) watchers += 1
    if (users === 1 || watch) void tick()
  })
  onBeforeUnmount(() => {
    users -= 1
    if (watch) watchers -= 1
    if (users <= 0) {
      users = 0
      watchers = 0
      if (timer) clearTimeout(timer)
      timer = null
    }
  })
  return { downloads, active }
}

/* ───────── 展示 ───────── */

export const isFinished = (d: DownloadInfo) =>
  d.state === DownloadState.DownloadState_Completed ||
  d.state === DownloadState.DownloadState_Failed ||
  d.state === DownloadState.DownloadState_Canceled

const STATE_KEYS: Record<string, string> = {
  [DownloadState.DownloadState_Downloading]: 'downloading',
  [DownloadState.DownloadState_Paused]: 'paused',
  [DownloadState.DownloadState_Retrying]: 'retrying',
  [DownloadState.DownloadState_Completed]: 'completed',
  [DownloadState.DownloadState_Failed]: 'failed',
  [DownloadState.DownloadState_Canceled]: 'canceled',
}

/** 状态的短名，也用作样式类名后缀 */
export const stateKey = (d: DownloadInfo) => STATE_KEYS[d.state] || 'downloading'
export const stateLabel = (d: DownloadInfo) => translate(`downloads.state.${stateKey(d)}`)

export const sourceLabel = (source: string) => {
  const key = `downloads.source.${source.replace(/\./g, '_')}`
  const text = translate(key)
  return text === key ? source : text
}

/** 0–100；总大小未知时为 null */
export const percentOf = (downloaded: unknown, total: unknown) => {
  const t = Number(total)
  if (!Number.isFinite(t) || t <= 0) return null
  return Math.min(100, Math.floor((Number(downloaded) / t) * 100))
}

const fmtDuration = (sec: number) => {
  if (sec < 60) return `${Math.max(1, Math.round(sec))}s`
  if (sec < 3600) return `${Math.floor(sec / 60)}m ${Math.round(sec % 60)}s`
  return `${Math.floor(sec / 3600)}h ${Math.floor((sec % 3600) / 60)}m`
}

/** 「12.3 MB / 45.6 MB · 2.1 MB/s · 剩余 20s」 */
export const progressText = (d: { downloaded: unknown; total: unknown; speed?: unknown }) => {
  const done = Number(d.downloaded) || 0
  const total = Number(d.total) || 0
  const speed = Number(d.speed) || 0
  const parts = [total > 0 ? `${formatFileSize(done)} / ${formatFileSize(total)}` : formatFileSize(done)]
  if (speed > 0) {
    parts.push(`${formatFileSize(speed)}/s`)
    if (total > done) parts.push(translate('downloads.left', { time: fmtDuration((total - done) / speed) }))
  }
  return parts.join(' · ')
}
