<!-- 客户端日志（Momoko）：玩家的游戏客户端上传的错误日志，每个游戏账号保留最新 5 份（从新到旧）：
     原因、摘要与说明、设备与客户端版本、大小、时间。点一行在等宽查看器里看全文（关键字筛选行并标出），
     可下载为 .log 或删除。 -->
<template>
  <div class="pv">
    <div class="pv__line">
      <span>{{ t('gmData.clientLogs.hint', { n: KEEP }) }}</span>
      <span class="pv__spacer" />
      <AppIconButton icon="HOutline:ArrowPathIcon" :label="t('gmData.common.refresh')" @click="load" />
    </div>
    <div v-if="error" class="pv__error">{{ error }}</div>

    <DataTable
      v-if="!menuStore.isMobile"
      :columns="columns"
      :rows="rows"
      row-key="id"
      :loading="loading"
      :empty-text="t('gmData.clientLogs.empty')"
      @row-click="(row) => open(x(row))"
    >
      <template #cell-time="{ row }">
        <span class="pv__stack">
          <span class="pv__strong m-num">{{ dateTime(x(row).occurredAt) }}</span>
          <span class="pv__dim">{{ t('gmData.clientLogs.uploaded', { time: when(x(row).createdAt) }) }}</span>
        </span>
      </template>
      <template #cell-reason="{ row }">
        <StatusPill :variant="reasonVariant(x(row).reason)" :label="reasonLabel(x(row).reason)" />
      </template>
      <template #cell-summary="{ row }">
        <span class="pv__stack">
          <span class="pv__ellipsis" :title="x(row).summary">{{ x(row).summary || '—' }}</span>
          <span v-if="x(row).note" class="pv__dim pv__ellipsis" :title="x(row).note">{{ x(row).note }}</span>
        </span>
      </template>
      <template #cell-device="{ row }">
        <span class="pv__stack">
          <span class="pv__ellipsis" :title="x(row).device">{{ x(row).device || '—' }}</span>
          <span class="pv__dim m-mono">{{ x(row).clientVersion || '—' }}</span>
        </span>
      </template>
      <template #cell-size="{ row }">
        <span class="pv__dim m-num">{{ formatFileSize(x(row).size) }}</span>
      </template>
      <template #cell-op="{ row }">
        <div class="pv__acts">
          <AppIconButton
            icon="HOutline:ArrowDownTrayIcon"
            :label="t('gmData.clientLogs.download')"
            :box="40"
            :size="18"
            @click.stop="download(x(row))"
          />
          <AppIconButton
            icon="HOutline:TrashIcon"
            :label="t('gmData.clientLogs.delete')"
            :box="40"
            :size="18"
            class="cl__del"
            @click.stop="remove(x(row))"
          />
        </div>
      </template>
    </DataTable>

    <template v-else>
      <EmptyState v-if="!loading && !items.length" icon="HOutline:BugAntIcon" :title="t('gmData.clientLogs.empty')" />
      <div v-else class="pv__mcards">
        <article v-for="l in items" :key="l.id" class="pv__mcard cl__mcard">
          <button type="button" class="pv__mbody cl__mopen" @click="open(l)">
            <span class="pv__mline">
              <StatusPill :variant="reasonVariant(l.reason)" :label="reasonLabel(l.reason)" />
              <span class="pv__dim m-num">{{ dateTime(l.occurredAt) }}</span>
            </span>
            <span class="pv__strong cl__clamp">{{ l.summary || '—' }}</span>
            <span class="pv__dim">{{ [l.device, l.clientVersion, formatFileSize(l.size)].filter(Boolean).join(' · ') }}</span>
          </button>
          <div class="cl__macts">
            <AppIconButton icon="HOutline:ArrowDownTrayIcon" :label="t('gmData.clientLogs.download')" :box="44" @click="download(l)" />
            <AppIconButton icon="HOutline:TrashIcon" :label="t('gmData.clientLogs.delete')" :box="44" class="cl__del" @click="remove(l)" />
          </div>
        </article>
      </div>
    </template>

    <FormDialog v-model="viewerOpen" :title="viewerTitle" :width="1000" :show-footer="false">
      <div v-if="current" class="cl__viewer">
        <dl class="cl__facts">
          <div><dt>{{ t('gmData.clientLogs.occurredAt') }}</dt><dd class="m-num">{{ dateTime(current.occurredAt) }}</dd></div>
          <div><dt>{{ t('gmData.clientLogs.createdAt') }}</dt><dd class="m-num">{{ dateTime(current.createdAt) }}</dd></div>
          <div><dt>{{ t('gmData.clientLogs.device') }}</dt><dd>{{ current.device || '—' }}</dd></div>
          <div><dt>{{ t('gmData.clientLogs.version') }}</dt><dd class="m-mono">{{ current.clientVersion || '—' }}</dd></div>
          <div><dt>{{ t('gmData.clientLogs.size') }}</dt><dd class="m-num">{{ formatFileSize(current.size) }}</dd></div>
          <div v-if="current.summary" class="is-wide"><dt>{{ t('gmData.clientLogs.summary') }}</dt><dd>{{ current.summary }}</dd></div>
          <div v-if="current.note" class="is-wide"><dt>{{ t('gmData.clientLogs.note') }}</dt><dd>{{ current.note }}</dd></div>
        </dl>

        <ListToolbar v-model:keyword="query" :placeholder="t('gmData.clientLogs.search')" @search="applied = query">
          <span v-if="needle" class="pv__dim m-num">{{ matchLine }}</span>
          <UButton color="neutral" variant="outline" icon="i-lucide-download" :disabled="!content" @click="download(current)">
            {{ t('gmData.clientLogs.download') }}
          </UButton>
        </ListToolbar>

        <div v-if="contentLoading" class="pv__hint">{{ t('gmData.common.loading') }}</div>
        <div v-else-if="contentError" class="pv__error">{{ contentError }}</div>
        <pre v-else-if="!needle" class="cl__pre m-mono">{{ content }}</pre>
        <div v-else-if="!matches.length" class="pv__hint">{{ t('gmData.common.noMatch') }}</div>
        <pre v-else class="cl__pre m-mono"><span
          v-for="m in shownMatches"
          :key="m.n"
          class="cl__hit"
        ><span class="cl__ln">{{ m.n }}</span><template v-for="(s, i) in m.parts" :key="i"><mark v-if="s.hit">{{ s.text }}</mark><template v-else>{{ s.text }}</template></template>
</span></pre>
      </div>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import { deleteGmClientLog, getGmClientLog, listGmClientLogs } from '@/api/gameGm'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { GmClientLog } from '@/types/v1/game_gm'
import { Dialog } from '@/utils/dialog'
import { feedback } from '@/utils/feedback'
import { formatFileSize } from '@/utils/file'
import { getRequestErrorMessage, showRequestError } from '@/utils/request'
import { useGmContext } from '../../context'
import { usePlayerPage } from '../context'
import { useGmFormat } from '../shared/format'

/** 每个游戏账号保留的份数（服务端规则）。 */
const KEEP = 5
/** 关键字筛选时最多列出的行数。 */
const MAX_MATCHES = 2000
const REASONS = ['exception', 'error', 'crash', 'fatal', 'login', 'sdk', 'manual']

const { t, te } = useI18n()
const ctx = useGmContext()
const page = usePlayerPage()
const menuStore = useMenuStore()
const { when, dateTime } = useGmFormat()

const items = shallowRef<GmClientLog[]>([])
const loading = ref(false)
const error = ref('')
let seq = 0

const load = async () => {
  const id = ++seq
  loading.value = true
  error.value = ''
  try {
    const { data } = await listGmClientLogs(ctx.serverId.value, page.pid.value)
    if (id === seq) items.value = data.items || []
  } catch (e) {
    if (id === seq) {
      error.value = getRequestErrorMessage(e)
      items.value = []
    }
  } finally {
    if (id === seq) loading.value = false
  }
}
watch(page.version, () => void load(), { immediate: true })

const columns = computed<DataTableColumn[]>(() => [
  { key: 'time', title: t('gmData.clientLogs.time'), width: 170 },
  { key: 'reason', title: t('gmData.clientLogs.reason'), width: 120 },
  { key: 'summary', title: t('gmData.clientLogs.summary'), minWidth: 260 },
  { key: 'device', title: t('gmData.clientLogs.deviceVersion'), minWidth: 180 },
  { key: 'size', title: t('gmData.clientLogs.size'), width: 100, align: 'right' },
  { key: 'op', title: '', width: 104, align: 'right' },
])
const rows = computed(() => items.value as unknown as Record<string, unknown>[])
const x = (row: Record<string, unknown>) => row as unknown as GmClientLog

const reasonLabel = (reason: string) =>
  REASONS.includes(reason) && te(`gmData.clientLogs.reasons.${reason}`) ? t(`gmData.clientLogs.reasons.${reason}`) : reason || '—'
const reasonVariant = (reason: string) => {
  switch (reason) {
    case 'crash':
    case 'fatal':
      return 'error' as const
    case 'exception':
    case 'error':
      return 'warning' as const
    case 'manual':
      return 'primary' as const
    default:
      return 'neutral' as const
  }
}

// ---------- 查看器 ----------
const viewerOpen = ref(false)
const current = shallowRef<GmClientLog>()
const content = ref('')
const contentLoading = ref(false)
const contentError = ref('')
const query = ref('')
const applied = ref('')

const viewerTitle = computed(() =>
  current.value ? `${reasonLabel(current.value.reason)} · ${dateTime(current.value.occurredAt)}` : '',
)

/** 读取日志原文（查看与下载共用；同一份只读一次）。 */
const fetchContent = async (log: GmClientLog) => {
  if (current.value?.id === log.id && content.value) return content.value
  const { data } = await getGmClientLog(ctx.serverId.value, page.pid.value, log.id)
  return data.content
}

const open = async (log: GmClientLog) => {
  current.value = log
  content.value = ''
  contentError.value = ''
  query.value = ''
  applied.value = ''
  viewerOpen.value = true
  contentLoading.value = true
  try {
    const text = await fetchContent(log)
    if (current.value?.id === log.id) content.value = text
  } catch (e) {
    if (current.value?.id === log.id) contentError.value = getRequestErrorMessage(e)
  } finally {
    if (current.value?.id === log.id) contentLoading.value = false
  }
}

interface Match {
  n: number
  parts: { text: string; hit: boolean }[]
}

const needle = computed(() => applied.value.trim().toLowerCase())
const lines = computed(() => content.value.split(/\r?\n/))

/** 一行按关键字切成片段（不区分大小写；大小写转换改变长度的行不标出）。 */
const highlightParts = (text: string, q: string): Match['parts'] => {
  const lower = text.toLowerCase()
  if (lower.length !== text.length) return [{ text, hit: false }]
  const parts: Match['parts'] = []
  let i = 0
  while (i < text.length) {
    const j = lower.indexOf(q, i)
    if (j < 0) {
      parts.push({ text: text.slice(i), hit: false })
      break
    }
    if (j > i) parts.push({ text: text.slice(i, j), hit: false })
    parts.push({ text: text.slice(j, j + q.length), hit: true })
    i = j + q.length
  }
  return parts
}

const matches = computed(() => {
  const q = needle.value
  if (!q) return [] as number[]
  const out: number[] = []
  lines.value.forEach((line, i) => {
    if (line.toLowerCase().includes(q)) out.push(i)
  })
  return out
})
const shownMatches = computed<Match[]>(() =>
  matches.value.slice(0, MAX_MATCHES).map((i) => ({ n: i + 1, parts: highlightParts(lines.value[i] ?? '', needle.value) })),
)
const matchLine = computed(() =>
  matches.value.length > MAX_MATCHES
    ? t('gmData.clientLogs.matchedFirst', { n: matches.value.length, shown: MAX_MATCHES })
    : t('gmData.clientLogs.matched', { n: matches.value.length }),
)

// ---------- 下载与删除 ----------
const fileName = (log: GmClientLog) =>
  `client-${page.pid.value}-${dayjs(log.occurredAt).format('YYYYMMDD-HHmmss')}-${log.reason || 'log'}.log`

const download = async (log: GmClientLog | undefined) => {
  if (!log) return
  try {
    const text = await fetchContent(log)
    const blob = new Blob([text], { type: 'text/plain;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = fileName(log)
    a.click()
    setTimeout(() => URL.revokeObjectURL(url), 1000)
  } catch (e) {
    showRequestError(e)
  }
}

const remove = (log: GmClientLog) => {
  Dialog.confirm({
    title: t('gmData.clientLogs.deleteTitle'),
    content: t('gmData.clientLogs.deleteContent', { time: dateTime(log.occurredAt) }),
    cancelText: t('gmData.clientLogs.cancel'),
    confirmText: t('gmData.clientLogs.delete'),
    onConfirm: async () => {
      try {
        await deleteGmClientLog(ctx.serverId.value, page.pid.value, log.id)
        feedback.success(t('gmData.clientLogs.deleted'))
        if (current.value?.id === log.id) viewerOpen.value = false
        await load()
      } catch {
        // 错误已由请求层提示
      }
    },
  })
}
</script>

<style scoped lang="scss">
@use '../shared/views';
.cl__del:hover {
  color: var(--m-bad);
}
.cl__mcard {
  align-items: center;
}
.cl__mopen {
  border: 0;
  padding: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.cl__clamp {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  word-break: break-all;
}
.cl__macts {
  flex: none;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.cl__viewer {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
}
.cl__facts {
  margin: 0;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 10px 16px;
  padding: 12px 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.cl__facts > div {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.cl__facts > div.is-wide {
  grid-column: 1 / -1;
}
.cl__facts dt {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.cl__facts dd {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  word-break: break-all;
}
.cl__pre {
  margin: 0;
  max-height: 60vh;
  overflow: auto;
  padding: 12px 14px;
  border-radius: var(--app-radius);
  border: 1px solid var(--el-border-color-light);
  background: var(--m-s2);
  color: var(--el-text-color-primary);
  font-size: 12px;
  line-height: 1.55;
  white-space: pre;
  tab-size: 4;
}
.cl__ln {
  display: inline-block;
  min-width: 5ch;
  margin-right: 12px;
  color: var(--el-text-color-secondary);
  text-align: right;
  user-select: none;
}
.cl__pre mark {
  border-radius: 3px;
  background: var(--m-warnSoft);
  color: var(--m-ink);
}
@media (max-width: 760px) {
  .cl__pre {
    max-height: 55vh;
    font-size: 11px;
  }
}
</style>
