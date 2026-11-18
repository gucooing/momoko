<!-- GM 运营 · 操作与日志 · 客户端日志（?view=clientLogs）：这台服务器的玩家上传的游戏客户端错误日志，从新到旧分页。
     按原因（chips）、玩家 ID、上传时间筛选，关键字搜摘要、说明与设备。列表与查看器和 GM 玩家详情的「客户端日志」共用：
     点一行读全文、在日志里查找、复制全部；行内复制摘要、下载、删除，并可打开那个玩家的 GM 详情。 -->
<template>
  <div class="gcl">
    <div class="gcl__head">
      <p class="gcl__sub">{{ t('clientLogs.count', { n: total }) }} · {{ t('clientLogs.description', { n: CLIENT_LOG_KEEP }) }}</p>
      <AppIconButton icon="HOutline:ArrowPathIcon" :label="t('gmData.common.refresh')" @click="load" />
    </div>

    <ListToolbar
      v-model:status="reason"
      v-model:keyword="keyword"
      :chips="chips"
      :chips-label="t('clientLogs.filter.reasons')"
      :placeholder="t('clientLogs.filter.search')"
      @search="reload"
    />

    <div class="gcl__filters">
      <UInput v-model="pid" class="m-mono gcl__pid" icon="i-lucide-user" :placeholder="t('clientLogs.filter.pid')" :aria-label="t('clientLogs.filter.pid')" />
      <label class="gcl__date">
        <span>{{ t('clientLogs.filter.since') }}</span>
        <UInput v-model="since" type="date" :aria-label="t('clientLogs.filter.since')" />
      </label>
      <label class="gcl__date">
        <span>{{ t('clientLogs.filter.until') }}</span>
        <UInput v-model="until" type="date" :aria-label="t('clientLogs.filter.until')" />
      </label>
      <UButton v-if="filtered" color="neutral" variant="ghost" icon="i-lucide-x" @click="clearFilters">{{ t('clientLogs.filter.clear') }}</UButton>
    </div>

    <div v-if="error" class="gcl__error">{{ error }}</div>
    <ClientLogTable
      :items="items"
      :loading="loading"
      :empty-text="filtered ? t('clientLogs.emptyFiltered') : t('clientLogs.empty')"
      show-player
      @changed="load"
    />
    <Pagination v-if="total > pageSize" v-model:page="page" v-model:page-size="pageSize" :total="total" class="gcl__pager" @change="load" />
  </div>
</template>

<script setup lang="ts">
import { computed, onActivated, onMounted, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDebounceFn } from '@vueuse/core'
import dayjs from 'dayjs'
import { listGmClientLogs } from '@/api/gameGm'
import type { GmClientLog } from '@/types/v1/game_gm'
import { getRequestErrorMessage } from '@/utils/request'
import { CLIENT_LOG_KEEP, CLIENT_LOG_REASONS } from '@/views/game/clientLogs'
import ClientLogTable from '@/views/game/components/ClientLogTable.vue'
import { useGmContext } from '../../context'

defineOptions({ name: 'GmClientLogsPanel' })

const { t } = useI18n()
const ctx = useGmContext()

// ---------- 筛选 ----------
const reason = ref('')
const keyword = ref('')
const pid = ref('')
const since = ref('')
const until = ref('')

const chips = computed(() => [
  { value: '', label: t('clientLogs.filter.reasonAll') },
  ...CLIENT_LOG_REASONS.map((r) => ({ value: r as string, label: t(`clientLogs.reasons.${r}`) })),
])

const filtered = computed(() => !!(reason.value || keyword.value.trim() || pid.value.trim() || since.value || until.value))

const clearFilters = () => {
  reason.value = ''
  keyword.value = ''
  pid.value = ''
  since.value = ''
  until.value = ''
  reload()
}

// ---------- 列表 ----------
const items = shallowRef<GmClientLog[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const error = ref('')
let seq = 0

/** 日期（本地时间）→ 当天 0 点；until 取下一天 0 点（不含）。 */
const dayStart = (value: string, next = false) => {
  if (!value) return undefined
  const d = dayjs(value)
  if (!d.isValid()) return undefined
  return (next ? d.add(1, 'day') : d).startOf('day').toDate()
}

const load = async () => {
  const id = ++seq
  loading.value = true
  error.value = ''
  try {
    const { data } = await listGmClientLogs({
      serverId: ctx.serverId.value,
      pid: pid.value.trim(),
      reason: reason.value,
      q: keyword.value.trim(),
      since: dayStart(since.value),
      until: dayStart(until.value, true),
      page: page.value,
      pageSize: pageSize.value,
    })
    if (id !== seq) return
    items.value = data.items || []
    total.value = data.total || 0
  } catch (e) {
    if (id !== seq) return
    error.value = getRequestErrorMessage(e)
    items.value = []
    total.value = 0
  } finally {
    if (id === seq) loading.value = false
  }
}

const reload = () => {
  page.value = 1
  void load()
}

const debouncedReload = useDebounceFn(reload, 350)
watch(pid, () => void debouncedReload())
watch([since, until], reload)

let activatedOnce = false
onMounted(() => void load())
onActivated(() => {
  if (activatedOnce) void load()
  activatedOnce = true
})
</script>

<style scoped lang="scss">
.gcl {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}
.gcl__head {
  display: flex;
  align-items: center;
  gap: 12px 16px;
}
.gcl__sub {
  flex: 1 1 auto;
  min-width: 0;
  margin: 0;
  font-size: 14px;
  color: var(--el-text-color-secondary);
}
.gcl__filters {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px 14px;
}
.gcl__pid {
  width: 220px;
}
.gcl__date {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.gcl__error {
  font-size: 0.8125rem;
  color: var(--m-bad);
}
.gcl__pager {
  align-self: flex-end;
}
@media (max-width: 760px) {
  .gcl__pid {
    width: 100%;
  }
  .gcl__date {
    flex: 1 1 100%;
  }
}
</style>
