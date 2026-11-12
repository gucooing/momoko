<!-- GM 记录（Momoko）：涉及这个玩家的指令执行，管理端与玩家自助都在内。点一行看结果（逐个对象的实际变化），不显示请求参数。 -->
<template>
  <div class="pv">
    <div v-if="error" class="pv__error">{{ error }}</div>

    <DataTable
      v-if="!menuStore.isMobile"
      :columns="columns"
      :rows="rows"
      row-key="id"
      :loading="loading"
      :empty-text="t('gmData.gmLog.empty')"
      @row-click="(row) => open(x(row))"
    >
      <template #cell-time="{ row }">
        <span class="pv__dim m-num">{{ when(x(row).createTime) }}</span>
      </template>
      <template #cell-command="{ row }">
        <span class="pv__stack">
          <span class="pv__strong pv__ellipsis">{{ x(row).commandTitle || t('game.unnamed') }}</span>
          <span v-if="x(row).reason" class="pv__dim pv__ellipsis" :title="x(row).reason">{{ x(row).reason }}</span>
        </span>
      </template>
      <template #cell-actor="{ row }">
        <span class="pv__stack">
          <span class="pv__strong pv__ellipsis">{{ x(row).actorName || '—' }}</span>
          <span class="pv__dim">{{ actorKind(x(row)) }}</span>
        </span>
      </template>
      <template #cell-risk="{ row }">
        <GmRiskPill :risk="x(row).risk" />
      </template>
      <template #cell-status="{ row }">
        <StatusPill :variant="executionVariant(x(row).status)" :label="statusLabel(x(row).status)" />
      </template>
    </DataTable>

    <template v-else>
      <EmptyState v-if="!loading && !items.length" icon="HOutline:ClockIcon" :title="t('gmData.gmLog.empty')" />
      <div v-else class="pv__mcards">
        <button v-for="e in items" :key="e.id" type="button" class="pv__mcard gl__mrow" @click="open(e)">
          <div class="pv__mbody">
            <span class="pv__mline">
              <span class="pv__strong">{{ e.commandTitle || t('game.unnamed') }}</span>
              <GmRiskPill :risk="e.risk" />
            </span>
            <span class="pv__dim">{{ e.actorName || '—' }} · {{ actorKind(e) }} · {{ when(e.createTime) }}</span>
          </div>
          <StatusPill :variant="executionVariant(e.status)" :label="statusLabel(e.status)" />
        </button>
      </div>
    </template>

    <div class="pv__foot">
      <RouterLink class="gl__link" :to="logsLink">{{ t('gmData.gmLog.all') }}</RouterLink>
      <Pagination v-if="total" v-model:page="pageNo" v-model:page-size="pageSize" :total="total" @change="load" />
    </div>

    <FormDialog v-model="detailOpen" :title="detail ? detail.commandTitle || t('game.unnamed') : ''" :width="620" :show-footer="false">
      <div v-if="detailLoading && !detail" class="pv__hint">{{ t('gmData.common.loading') }}</div>
      <div v-else-if="detail" class="gl__detail">
        <div class="pv__mline">
          <GmRiskPill :risk="detail.risk" />
          <StatusPill :variant="executionVariant(detail.status)" :label="statusLabel(detail.status)" />
        </div>
        <dl class="gl__facts">
          <div><dt>{{ t('gmData.gmLog.time') }}</dt><dd class="m-num">{{ dateTime(detail.createTime) }}</dd></div>
          <div><dt>{{ t('gmData.gmLog.actor') }}</dt><dd>{{ detail.actorName || '—' }} · {{ actorKind(detail) }}</dd></div>
          <div v-if="detail.reason"><dt>{{ t('gmData.gmLog.reason') }}</dt><dd>{{ detail.reason }}</dd></div>
          <div v-if="detail.reviewerName">
            <dt>{{ t('gmData.gmLog.reviewer') }}</dt>
            <dd>{{ detail.reviewerName }}<template v-if="detail.reviewNote"> · {{ detail.reviewNote }}</template></dd>
          </div>
          <div v-if="detail.errorCode"><dt>{{ t('gmData.gmLog.error') }}</dt><dd>{{ detail.errorMessage || detail.errorCode }}</dd></div>
          <div v-if="detail.durationMs"><dt>{{ t('gmData.gmLog.duration') }}</dt><dd class="m-num">{{ detail.durationMs }} ms</dd></div>
        </dl>
        <GmCommandResult v-if="detailResult" :result="detailResult" />
      </div>
      <div v-if="detailError" class="pv__error">{{ detailError }}</div>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getGmExecution, listGmExecutions } from '@/api/gameGm'
import { getRequestErrorMessage } from '@/utils/request'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { CommandResult, ManifestView } from '@/games/mgp'
import type { GmExecution } from '@/types/v1/game_gm'
import { useGmContext } from '../../context'
import GmCommandResult from '../../components/GmCommandResult.vue'
import GmRiskPill from '../../components/GmRiskPill.vue'
import { usePlayerPage } from '../context'
import { executionVariant, gmExecutionsQuery } from '../shared/executions'
import { useGmFormat } from '../shared/format'

defineProps<{ view?: ManifestView }>()

const { t } = useI18n()
const ctx = useGmContext()
const page = usePlayerPage()
const menuStore = useMenuStore()
const { when, dateTime } = useGmFormat()

const items = shallowRef<GmExecution[]>([])
const total = ref(0)
const pageNo = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const error = ref('')

const load = async () => {
  loading.value = true
  error.value = ''
  try {
    const { data } = await listGmExecutions(
      gmExecutionsQuery(ctx.serverId.value, { pid: page.pid.value, page: pageNo.value, pageSize: pageSize.value }),
    )
    items.value = data.items || []
    total.value = data.total || 0
  } catch (e) {
    error.value = getRequestErrorMessage(e)
  } finally {
    loading.value = false
  }
}
watch(page.version, () => void load(), { immediate: true })

const columns = computed<DataTableColumn[]>(() => [
  { key: 'time', title: t('gmData.gmLog.time'), width: 130 },
  { key: 'command', title: t('gmData.gmLog.command'), minWidth: 240 },
  { key: 'actor', title: t('gmData.gmLog.actor'), minWidth: 160 },
  { key: 'risk', title: t('gmData.gmLog.risk'), width: 90 },
  { key: 'status', title: t('gmData.gmLog.status'), width: 110 },
])
const rows = computed(() => items.value as unknown as Record<string, unknown>[])
const x = (row: Record<string, unknown>) => row as unknown as GmExecution

const actorKind = (e: GmExecution) => (e.actorKind === 'player' ? t('gmData.gmLog.kindPlayer') : t('gmData.gmLog.kindAdmin'))
const statusLabel = (status: string) => t(`gmData.exec.${status || 'unknown'}`)

const logsLink = computed(() => ({ path: `/game/gm/${encodeURIComponent(ctx.serverId.value)}`, query: { tab: 'logs' } }))

// ---------- 详情 ----------
const detailOpen = ref(false)
const detail = shallowRef<GmExecution>()
const detailLoading = ref(false)
const detailError = ref('')
const detailResult = computed(() => {
  const r = detail.value?.result as CommandResult | undefined
  return r && (r.results?.length || r.summary !== undefined) ? r : undefined
})

const open = async (e: GmExecution) => {
  detail.value = e
  detailOpen.value = true
  detailLoading.value = true
  detailError.value = ''
  try {
    const { data } = await getGmExecution(ctx.serverId.value, e.id)
    if (detail.value?.id === e.id) detail.value = data
  } catch (err) {
    detailError.value = getRequestErrorMessage(err)
  } finally {
    detailLoading.value = false
  }
}
</script>

<style scoped lang="scss">
@use '../shared/views';
.gl__mrow {
  width: 100%;
  align-items: center;
  font: inherit;
  color: inherit;
  text-align: left;
  cursor: pointer;
}
.gl__link {
  font-size: 13px;
  font-weight: 700;
  color: var(--m-pInk);
  text-decoration: none;
}
.gl__detail {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.gl__facts {
  margin: 0;
  display: flex;
  flex-direction: column;
}
.gl__facts > div {
  display: flex;
  gap: 12px;
  padding: 8px 0;
  border-top: 1px solid var(--el-border-color-lighter);
  font-size: 13px;
}
.gl__facts > div:first-child {
  border-top: 0;
}
.gl__facts dt {
  flex: none;
  width: 84px;
  color: var(--el-text-color-secondary);
}
.gl__facts dd {
  margin: 0;
  min-width: 0;
  font-weight: 600;
  word-break: break-all;
}
</style>
