<!-- 客户端日志列表（「客户端日志」页与 GM 玩家详情共用）：一份日志一行（手机端一张卡片），点一行或「查看」打开查看器读全文；
     行内可一键复制摘要、下载 .log、删除（先确认）。show-owner 时多出服务器、玩家与绑定的通行证，并可打开那个玩家的 GM 详情。 -->
<template>
  <div class="clt">
    <DataTable
      v-if="!menuStore.isMobile"
      :columns="columns"
      :rows="rows"
      row-key="id"
      :loading="loading"
      :empty-text="emptyText"
      @row-click="(row) => open(x(row))"
    >
      <template #cell-time="{ row }">
        <span class="clt__stack">
          <span class="clt__strong m-num">{{ dateTime(x(row).occurredAt) }}</span>
          <span class="clt__dim">{{ t('clientLogs.uploaded', { time: when(x(row).createdAt) }) }}</span>
        </span>
      </template>
      <template #cell-server="{ row }">
        <span class="clt__stack">
          <span class="clt__ellipsis" :title="clientLogServer(x(row))">{{ x(row).server?.name || x(row).server?.id || '—' }}</span>
          <span class="clt__dim clt__ellipsis">{{ x(row).server?.gameName || x(row).server?.game || '' }}</span>
        </span>
      </template>
      <template #cell-player="{ row }">
        <span class="clt__stack">
          <span class="clt__strong m-mono">{{ x(row).playerId }}</span>
          <span class="clt__dim clt__ellipsis">{{ clientLogPassport(x(row)) || t('clientLogs.noLink') }}</span>
        </span>
      </template>
      <template #cell-reason="{ row }">
        <StatusPill :variant="clientLogReasonVariant(x(row).reason)" :label="clientLogReasonLabel(t, x(row).reason)" />
      </template>
      <template #cell-summary="{ row }">
        <span class="clt__summary">
          <span class="clt__stack">
            <span class="clt__ellipsis" :title="x(row).summary">{{ x(row).summary || '—' }}</span>
            <span v-if="x(row).note" class="clt__dim clt__ellipsis" :title="x(row).note">{{ x(row).note }}</span>
          </span>
          <AppIconButton
            v-if="x(row).summary"
            icon="HOutline:ClipboardDocumentIcon"
            :label="t('clientLogs.actions.copySummary')"
            :box="32"
            :size="16"
            class="clt__copy"
            @click.stop="copySummary(x(row))"
          />
        </span>
      </template>
      <template #cell-device="{ row }">
        <span class="clt__stack">
          <span class="clt__ellipsis" :title="x(row).device">{{ x(row).device || '—' }}</span>
          <span class="clt__dim m-mono">{{ x(row).clientVersion || '—' }}</span>
        </span>
      </template>
      <template #cell-size="{ row }">
        <span class="clt__dim m-num">{{ formatFileSize(x(row).size) }}</span>
      </template>
      <template #cell-op="{ row }">
        <div class="clt__acts">
          <AppIconButton icon="HOutline:EyeIcon" :label="t('clientLogs.actions.view')" :box="36" :size="18" @click.stop="open(x(row))" />
          <AppIconButton
            v-if="showOwner"
            icon="HOutline:UserIcon"
            :label="t('clientLogs.actions.player')"
            :box="36"
            :size="18"
            @click.stop="openPlayer(x(row))"
          />
          <AppIconButton
            icon="HOutline:ArrowDownTrayIcon"
            :label="t('clientLogs.actions.download')"
            :box="36"
            :size="18"
            @click.stop="download(x(row))"
          />
          <AppIconButton
            icon="HOutline:TrashIcon"
            :label="t('clientLogs.actions.delete')"
            :box="36"
            :size="18"
            class="clt__del"
            @click.stop="remove(x(row))"
          />
        </div>
      </template>
    </DataTable>

    <template v-else>
      <EmptyState v-if="!loading && !items.length" icon="HOutline:BugAntIcon" :title="emptyText" />
      <div v-else class="clt__cards">
        <article v-for="l in items" :key="l.id" class="clt__card">
          <button type="button" class="clt__open" @click="open(l)">
            <span class="clt__line">
              <StatusPill :variant="clientLogReasonVariant(l.reason)" :label="clientLogReasonLabel(t, l.reason)" />
              <span class="clt__dim m-num">{{ dateTime(l.occurredAt) }}</span>
            </span>
            <span v-if="showOwner" class="clt__dim">{{ [clientLogServer(l), l.playerId, clientLogPassport(l)].filter(Boolean).join(' · ') }}</span>
            <span class="clt__strong clt__clamp">{{ l.summary || '—' }}</span>
            <span class="clt__dim">{{ [l.device, l.clientVersion, formatFileSize(l.size)].filter(Boolean).join(' · ') }}</span>
          </button>
          <div class="clt__card-acts">
            <AppIconButton v-if="l.summary" icon="HOutline:ClipboardDocumentIcon" :label="t('clientLogs.actions.copySummary')" :box="44" @click="copySummary(l)" />
            <AppIconButton v-if="showOwner" icon="HOutline:UserIcon" :label="t('clientLogs.actions.player')" :box="44" @click="openPlayer(l)" />
            <AppIconButton icon="HOutline:ArrowDownTrayIcon" :label="t('clientLogs.actions.download')" :box="44" @click="download(l)" />
            <AppIconButton icon="HOutline:TrashIcon" :label="t('clientLogs.actions.delete')" :box="44" class="clt__del" @click="remove(l)" />
          </div>
        </article>
      </div>
    </template>

    <ClientLogViewer v-model="viewerOpen" :log="current" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, shallowRef } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { deleteGmClientLog } from '@/api/gameGm'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { GmClientLog } from '@/types/v1/game_gm'
import { Dialog } from '@/utils/dialog'
import { feedback } from '@/utils/feedback'
import { formatFileSize } from '@/utils/file'
import { showRequestError } from '@/utils/request'
import { useGmFormat } from '@/views/game/gm/player/shared/format'
import { CLIENT_LOG_TAB } from '@/views/game/gm/player/context'
import {
  clientLogPassport,
  clientLogReasonLabel,
  clientLogReasonVariant,
  clientLogServer,
  copyClientLogText,
  fetchClientLogContent,
  saveClientLog,
} from '../clientLogs'
import ClientLogViewer from './ClientLogViewer.vue'

const props = defineProps<{
  items: GmClientLog[]
  loading: boolean
  emptyText: string
  /** 显示服务器、玩家与通行证，并可打开玩家的 GM 详情（全部服务器的列表用） */
  showOwner?: boolean
}>()
const emit = defineEmits<{ changed: [] }>()

const { t } = useI18n()
const router = useRouter()
const menuStore = useMenuStore()
const { when, dateTime } = useGmFormat()

const columns = computed<DataTableColumn[]>(() => [
  { key: 'time', title: t('clientLogs.col.time'), width: 170 },
  ...(props.showOwner
    ? [
        { key: 'server', title: t('clientLogs.col.server'), minWidth: 150 },
        { key: 'player', title: t('clientLogs.col.player'), minWidth: 160 },
      ]
    : []),
  { key: 'reason', title: t('clientLogs.col.reason'), width: 130 },
  { key: 'summary', title: t('clientLogs.col.summary'), minWidth: 260 },
  { key: 'device', title: t('clientLogs.col.deviceVersion'), minWidth: 170 },
  { key: 'size', title: t('clientLogs.col.size'), width: 96, align: 'right' },
  { key: 'op', title: '', width: props.showOwner ? 168 : 132, align: 'right' },
])
const rows = computed(() => props.items as unknown as Record<string, unknown>[])
const x = (row: Record<string, unknown>) => row as unknown as GmClientLog

// ---------- 查看 ----------
const viewerOpen = ref(false)
const current = shallowRef<GmClientLog | null>(null)

const open = (log: GmClientLog) => {
  current.value = log
  viewerOpen.value = true
}

const openPlayer = (log: GmClientLog) => {
  if (!log.server?.id) return
  void router.push({
    path: `/game/gm/${encodeURIComponent(log.server.id)}/players/${encodeURIComponent(log.playerId)}`,
    query: { tab: CLIENT_LOG_TAB },
  })
}

// ---------- 复制、下载与删除 ----------
const copySummary = (log: GmClientLog) => void copyClientLogText(t, log.summary, t('clientLogs.copied'))

const download = async (log: GmClientLog) => {
  try {
    saveClientLog(log, await fetchClientLogContent(log.id))
  } catch (e) {
    showRequestError(e)
  }
}

const remove = (log: GmClientLog) => {
  Dialog.confirm({
    title: t('clientLogs.deleteTitle'),
    content: t('clientLogs.deleteContent', { time: dateTime(log.occurredAt) }),
    cancelText: t('clientLogs.cancel'),
    confirmText: t('clientLogs.actions.delete'),
    onConfirm: async () => {
      try {
        await deleteGmClientLog(log.id)
        feedback.success(t('clientLogs.deleted'))
        if (current.value?.id === log.id) viewerOpen.value = false
        emit('changed')
      } catch {
        // 错误已由请求层提示
      }
    },
  })
}
</script>

<style scoped lang="scss">
.clt__stack {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.clt__strong {
  font-weight: 700;
}
.clt__dim {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.clt__ellipsis {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.clt__summary {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.clt__summary > .clt__stack {
  flex: 1;
}
.clt__copy {
  flex: none;
  opacity: 0.6;
}
.clt__copy:hover {
  opacity: 1;
}
.clt__acts {
  display: flex;
  justify-content: flex-end;
  gap: 2px;
}
.clt__del:hover {
  color: var(--m-bad);
}
.clt__cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.clt__card {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 12px 12px 14px;
  border-radius: var(--app-radius);
  background: var(--el-bg-color);
  box-shadow: var(--app-shadow-sm);
}
.clt__open {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
  border: 0;
  padding: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.clt__line {
  display: flex;
  align-items: center;
  gap: 8px;
}
.clt__clamp {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  word-break: break-all;
}
.clt__card-acts {
  flex: none;
  display: grid;
  grid-template-columns: repeat(2, 44px);
  gap: 2px;
}
</style>
