<!-- 操作结果：逐个对象的状态、实际变化（变化项的名称与前后数值）、失败原因、其他返回内容与自动备份；服务器操作显示 summary。 -->
<template>
  <div class="gcr">
    <div v-if="outcome.dryRun" class="gcr__banner">{{ t('gm.result.dryRun') }}</div>
    <div v-for="r in outcome.results" :key="r.pid" class="gcr__row">
      <div class="gcr__head">
        <span class="gcr__who">{{ r.name || t('gm.player.noName') }}</span>
        <StatusPill :variant="r.status === 'success' ? 'success' : 'error'" :label="t(`gm.result.status.${r.status}`)" />
      </div>
      <div v-if="r.status !== 'success'" class="gcr__error">
        {{ r.message || r.code }}
      </div>
      <ul v-if="r.changes?.length" class="gcr__changes">
        <li v-for="(c, i) in r.changes" :key="i">
          <span class="gcr__kind">{{ ctx.changeLabel(c.kind, c.resourceId) }}</span>
          <span class="gcr__delta m-num">{{ fmt(c.kind, c.before) }} → {{ fmt(c.kind, c.after) }}</span>
        </li>
      </ul>
      <button v-if="exportable(r.detail)" type="button" class="gcr__link" @click="download(r.name, r.detail)">
        {{ t('gm.result.downloadExport') }}
      </button>
      <GenericData v-else-if="r.detail !== undefined" class="gcr__summary" :data="r.detail" />
    </div>
    <div v-if="outcome.backupRef" class="gcr__backup">{{ t('gm.result.backup') }}</div>
    <GenericData v-if="hasSummary" class="gcr__summary" :data="outcome.summary" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import type { OperationOutcome } from '@/games/mgp'
import { useGmContext } from '../context'
import GenericData from '../player/shared/GenericData.vue'

const props = defineProps<{ outcome: OperationOutcome }>()
const { t } = useI18n()
const ctx = useGmContext()

// 封禁到期时间是毫秒时间戳（0 = 未封禁），其余是数值
const fmt = (kind: string, v: unknown) => {
  if (kind === 'bannedUntil') return typeof v === 'number' && v > 0 ? dayjs(v).format('YYYY-MM-DD HH:mm') : t('gm.change.notBanned')
  return typeof v === 'number' ? v.toLocaleString() : String(v ?? '—')
}
const hasSummary = computed(() => props.outcome.summary !== undefined && props.outcome.summary !== null)

// archive.export 的结果在 detail.export 里是完整存档 JSON，提供下载；其他返回内容原样显示。
const exportable = (detail: unknown): detail is { export: unknown } =>
  !!detail && typeof detail === 'object' && 'export' in (detail as Record<string, unknown>)

const download = (name: string | undefined, detail: { export: unknown }) => {
  const blob = new Blob([JSON.stringify(detail.export, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${name || 'player'}-${dayjs().format('YYYYMMDD-HHmm')}.json`
  a.click()
  URL.revokeObjectURL(url)
}
</script>

<style scoped>
.gcr {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.gcr__banner {
  padding: 8px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--m-a2Soft);
  font-size: 13px;
  font-weight: 700;
}
.gcr__row {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
}
.gcr__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.gcr__who {
  font-weight: 800;
}
.gcr__error {
  font-size: 13px;
  color: var(--m-bad);
}
.gcr__changes {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 13px;
}
.gcr__changes li {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.gcr__kind {
  font-weight: 700;
}
.gcr__backup {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.gcr__link {
  align-self: flex-start;
  border: 0;
  background: transparent;
  color: var(--m-pInk);
  font: inherit;
  font-weight: 700;
  cursor: pointer;
  padding: 0;
}
.gcr__summary {
  margin: 0;
  padding: 10px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
