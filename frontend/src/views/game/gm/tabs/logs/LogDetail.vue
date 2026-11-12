<!-- 一条 GM 记录的详情：时间、执行者、来源、对象、风险、原因、审批，以及逐个对象的结果。
     记录详情从不显示请求参数（接口也不返回）。 -->
<template>
  <div class="ld">
    <div v-if="loading && !exec" class="ld__hint">{{ t('gmOps.common.loading') }}</div>
    <template v-else-if="exec">
      <div class="ld__head">
        <span class="ld__head-text">
          <span class="ld__title">{{ commandTitle(exec) }}</span>
        </span>
        <StatusPill :variant="statusPill(exec.status)" :label="statusText(exec.status)" />
      </div>
      <div v-if="exec.dryRun" class="ld__banner">{{ t('gmOps.logs.dryRunNote') }}</div>

      <dl class="ld__kv">
        <dt>{{ t('gmOps.logs.kv.time') }}</dt>
        <dd class="m-num">{{ fmtDateTime(exec.createTime, true) }}</dd>
        <dt>{{ t('gmOps.logs.kv.actor') }}</dt>
        <dd>{{ actorName(exec) }} · {{ kindText(exec.actorKind) }}</dd>
        <dt>{{ t('gmOps.logs.kv.channel') }}</dt>
        <dd>{{ channelLine }}</dd>
        <dt>{{ t('gmOps.logs.kv.source') }}</dt>
        <dd>{{ sourceText(exec) }}</dd>
        <dt>{{ t('gmOps.logs.kv.object') }}</dt>
        <dd>{{ objectText(exec) || '—' }}</dd>
        <dt>{{ t('gmOps.logs.kv.risk') }}</dt>
        <dd><GmRiskPill :risk="exec.risk" /></dd>
        <template v-if="exec.actorKind === 'player'">
          <dt>{{ t('gmOps.logs.kv.reason') }}</dt>
          <dd>{{ exec.reason || '—' }}</dd>
        </template>
        <dt>{{ t('gmOps.logs.kv.review') }}</dt>
        <dd>{{ reviewLine }}</dd>
        <template v-if="exec.confirmed">
          <dt>{{ t('gmOps.logs.kv.confirm') }}</dt>
          <dd>{{ t('gmOps.logs.confirmed') }}</dd>
        </template>
        <template v-if="exec.backupRef">
          <dt>{{ t('gmOps.logs.kv.backup') }}</dt>
          <dd>{{ t('gmOps.logs.backupDone') }}</dd>
        </template>
        <template v-if="exec.finishedAt">
          <dt>{{ t('gmOps.logs.kv.finished') }}</dt>
          <dd class="m-num">{{ fmtDateTime(exec.finishedAt, true) }} · {{ t('gmOps.logs.duration', { ms: exec.durationMs }) }}</dd>
        </template>
        <template v-if="exec.errorCode || exec.errorMessage">
          <dt>{{ t('gmOps.logs.kv.error') }}</dt>
          <dd class="ld__bad">{{ exec.errorMessage || exec.errorCode }}</dd>
        </template>
      </dl>

      <div class="ld__results">
        <span class="ld__label">{{ t('gmOps.logs.results') }}</span>
        <div v-if="exec.status === 'pending'" class="ld__row">
          <span class="ld__row-name">{{ targetNames }}</span>
          <span class="ld__hint ld__row-detail">{{ t('gmOps.logs.pendingResult') }}</span>
          <StatusPill variant="warning" :label="statusText('pending')" />
        </div>
        <div v-else-if="exec.status === 'rejected'" class="ld__row">
          <span class="ld__row-name">{{ targetNames }}</span>
          <span class="ld__hint ld__row-detail">{{ t('gmOps.logs.rejectedResult') }}</span>
          <StatusPill variant="error" :label="statusText('rejected')" />
        </div>
        <template v-else>
          <div v-for="r in results" :key="r.pid" class="ld__row ld__row--wrap">
            <span class="ld__row-name">{{ r.name || nameOf(r.pid) }}</span>
            <span class="ld__hint ld__row-detail">
              <template v-if="r.status !== 'success'">{{ r.message || r.code }}</template>
              <template v-else-if="r.changes?.length">
                <span v-for="(c, i) in r.changes" :key="i" class="ld__change">
                  {{ ctx.changeLabel(c.kind, c.resourceId) }}
                  <span class="m-num">{{ num(c.before) }} → {{ num(c.after) }}</span>
                </span>
              </template>
              <template v-else-if="r.backupId">{{ t('gmOps.logs.backupDone') }}</template>
            </span>
            <StatusPill :variant="r.status === 'success' ? 'success' : 'error'" :label="statusText(r.status === 'success' ? 'success' : 'failed')" />
          </div>
          <dl v-if="summaryEntries.length" class="ld__kv ld__summary">
            <template v-for="[k, v] in summaryEntries" :key="k">
              <dt>{{ k }}</dt>
              <dd class="m-num">{{ v }}</dd>
            </template>
          </dl>
          <div v-if="!results.length && !summaryEntries.length" class="ld__hint">{{ t('gmOps.logs.noResults') }}</div>
        </template>
      </div>

    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getGmExecution } from '@/api/gameGm'
import type { Change } from '@/games/mgp'
import type { GmExecution } from '@/types/v1/game_gm'
import { useGmContext } from '../../context'
import GmRiskPill from '../../components/GmRiskPill.vue'
import { fmtDateTime } from '../mail/shared'
import { useExecText } from './useExecText'

/** 记录结果里逐个对象的部分（不含 detail 原文）。 */
interface ResultRow {
  pid: string
  name?: string
  status: string
  code?: string
  message?: string
  changes?: Change[]
  backupId?: number | string
}

const props = defineProps<{ id: string; preview?: GmExecution }>()
const emit = defineEmits<{ loaded: [exec: GmExecution] }>()

const { t } = useI18n()
const ctx = useGmContext()
const { statusPill, statusText, kindText, channelText, sourceText, objectText, actorName, commandTitle } = useExecText()

const exec = ref<GmExecution>()
const loading = ref(false)

watch(
  () => props.id,
  async (id) => {
    exec.value = props.preview
    if (!id) return
    loading.value = true
    try {
      const { data } = await getGmExecution(ctx.serverId.value, id)
      if (props.id !== id) return
      exec.value = data
      emit('loaded', data)
    } catch {
      // 错误已由请求层提示
    } finally {
      loading.value = false
    }
  },
  { immediate: true },
)

const results = computed<ResultRow[]>(() => {
  const r = exec.value?.result as { results?: unknown } | undefined
  return Array.isArray(r?.results) ? (r.results as ResultRow[]) : []
})
const summaryEntries = computed<Array<[string, string]>>(() => {
  const r = exec.value?.result as { summary?: unknown } | undefined
  const s = r?.summary
  if (!s || typeof s !== 'object' || Array.isArray(s)) return []
  return Object.entries(s as Record<string, unknown>)
    .filter(([k]) => !/(^id$|^pid$|Id$|Ids$)/.test(k))
    .filter(([, v]) => v === null || ['string', 'number', 'boolean'].includes(typeof v) || Array.isArray(v))
    .map(([k, v]) => [k, Array.isArray(v) ? v.join(', ') : String(v ?? '')] as [string, string])
})

const nameOf = (pid: string) => {
  const hit = exec.value?.targets?.find((x) => x.pid === pid)
  return hit?.name || t('gm.player.noName')
}
const targetNames = computed(() => (exec.value?.targets || []).map((x) => x.name || t('gm.player.noName')).join('、') || '—')

const channelLine = computed(() => {
  const e = exec.value
  if (!e) return ''
  return [channelText(e.channel), e.platform, e.clientIp].filter(Boolean).join(' · ')
})
const reviewLine = computed(() => {
  const e = exec.value
  if (!e) return ''
  if (e.status === 'pending') return t('gmOps.logs.waiting')
  if (e.reviewerName || e.reviewedAt) {
    const key = e.status === 'rejected' ? 'gmOps.logs.rejectedBy' : 'gmOps.logs.approvedBy'
    const line = t(key, { name: e.reviewerName || '—', time: fmtDateTime(e.reviewedAt) })
    return e.reviewNote ? `${line} · ${e.reviewNote}` : line
  }
  return e.actorKind === 'player' ? t('gmOps.logs.noReview') : t('gmOps.logs.adminDirect')
})

const num = (v: unknown) => (typeof v === 'number' ? v.toLocaleString() : String(v ?? '—'))

</script>

<style scoped>
.ld {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.ld__head {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}
.ld__head-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.ld__title {
  font-size: 18px;
  font-weight: 800;
}
.ld__hint {
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
.ld__banner {
  padding: 8px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--m-a2Soft);
  font-size: 13px;
  font-weight: 700;
}
.ld__kv {
  display: grid;
  grid-template-columns: 84px minmax(0, 1fr);
  gap: 9px 12px;
  margin: 0;
  font-size: 13px;
}
.ld__kv dt {
  color: var(--el-text-color-secondary);
}
.ld__kv dd {
  margin: 0;
  font-weight: 600;
  min-width: 0;
  overflow-wrap: anywhere;
}
.ld__bad {
  color: var(--m-bad);
}
.ld__results {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.ld__label {
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.04em;
  color: var(--el-text-color-secondary);
}
.ld__row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 0;
  border-top: 1px solid var(--el-border-color-light);
  font-size: 13px;
}
.ld__row--wrap {
  flex-wrap: wrap;
}
.ld__row-name {
  flex: 1 1 120px;
  min-width: 0;
  font-weight: 700;
  overflow-wrap: anywhere;
}
.ld__row-detail {
  flex: 2 1 160px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  overflow-wrap: anywhere;
}
.ld__change {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 6px;
}
.ld__summary {
  padding: 10px 12px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
</style>
