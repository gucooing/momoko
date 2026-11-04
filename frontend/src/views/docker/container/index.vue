<!-- Docker 容器：页头（Docker · 引擎状态 · 任务角标 · 创建容器）→ 子导航 → 进行中的创建/重建任务卡
     → 状态 chips（全部 / 运行中 / 已停止 / 已退出 / 已暂停，带数量）+ 搜索（容器名、镜像或端口）
     → PC 表格（可多选；名称+短 ID / 镜像 / 状态+运行时长或退出信息 / 端口 / CPU / 内存 / 日志·终端·启停 +「⋯」）
       手机卡片（底部端口 + 日志·终端·启停·⋯，⋯ 打开底部操作面板）→ 分页 → 深色批量条（启动 / 停止 / 重启 / 删除）。
     创建 / 重建是本标签里的子视图（?create=1 / ?recreate=<id>），编辑配置、详情、日志、统计、终端走弹窗。 -->
<template>
  <ContainerCreate
    v-if="createMode"
    :key="createState.key"
    :mode="createMode"
    :recreate-id="createState.id"
    :initial-image="createState.image"
    @back="closeCreate"
    @done="onCreateDone"
  />

  <div v-show="!createMode" class="dk-page">
    <DockerHeader @tasks="openTasks">
      <template #actions>
        <UButton v-if="canManage" color="primary" icon="i-lucide-plus" :disabled="!ready" @click="openCreate">
          {{ t('docker.container.createContainer') }}
        </UButton>
      </template>
    </DockerHeader>

    <DockerSubNav />

    <DockerUnavailable v-if="engineLoaded && !ready" @retry="init(true)" />

    <template v-else>
      <DockerPullTasks :types="ACTIVE_TYPES" :can-cancel="false" @open="openTask" />

      <ListToolbar
        v-model:status="queryForm.status"
        v-model:keyword="queryForm.keyword"
        :chips="statusChips"
        :chips-label="t('docker.ctr.filterLabel')"
        :placeholder="t('docker.ctr.searchPlaceholder')"
        @search="search"
      />

      <div class="dk-page__body">
        <!-- PC：表格 -->
        <DataTable
          v-if="!menuStore.isMobile"
          v-model="selected"
          :columns="columns"
          :rows="list"
          row-key="id"
          :selectable="canManage"
          :loading="loading"
          :empty-text="t('docker.container.noContainers')"
        >
          <template #cell-names="{ row }">
            <div class="ct-name">
              <strong>{{ nameOf(row) }}</strong>
              <span class="m-mono">{{ shortId(row.id) }}</span>
            </div>
          </template>
          <template #cell-image="{ row }"><span class="ct-img m-mono" :title="String(row.image)">{{ row.image }}</span></template>
          <template #cell-state="{ row }">
            <div class="ct-state">
              <StatusPill :variant="pillOf(row).variant" :label="pillOf(row).label" />
              <span class="ct-sub m-num">{{ subLineOf(row) }}</span>
            </div>
          </template>
          <template #cell-ports="{ row }">
            <div v-if="portsOf(row).length" class="dk-tags">
              <span v-for="p in portsOf(row)" :key="p" class="ct-port m-mono">{{ p }}</span>
            </div>
            <span v-else class="dk-dim">—</span>
          </template>
          <template #cell-cpu="{ row }">
            <div v-if="usageOf(row)" class="ct-cpu">
              <span class="ct-bar"><span :style="{ width: `${cpuBar(usageOf(row)!.cpu)}%` }" /></span>
              <strong class="m-num">{{ usageOf(row)!.cpu.toFixed(1) }}%</strong>
            </div>
            <span v-else class="dk-dim">—</span>
          </template>
          <template #cell-mem="{ row }">
            <strong v-if="usageOf(row)" class="ct-mem m-num" :title="usageOf(row)!.limit ? t('docker.ctr.memOf', { limit: formatBytes(usageOf(row)!.limit) }) : undefined">
              {{ formatBytes(usageOf(row)!.mem) }}
            </strong>
            <span v-else class="dk-dim">—</span>
          </template>
          <template #cell-operation="{ row }">
            <div class="dk-acts">
              <AppIconButton
                v-for="qa in quickActionsFor(row)"
                :key="qa.key"
                :icon="qa.icon"
                :label="`${qa.label} ${nameOf(row)}`"
                :box="36"
                :size="18"
                :class="qa.tone"
                @click="onRowAction(qa.key, row)"
              />
              <ActionMenu :items="menuActionsFor(row)" :label="t('docker.ctr.moreOf', { name: nameOf(row) })" @select="(key) => onRowAction(key, row)" />
            </div>
          </template>
        </DataTable>

        <!-- 手机：卡片 -->
        <template v-else>
          <div v-if="loading && !list.length" class="dk-cards">
            <div v-for="i in 4" :key="i" class="dk-skeleton ct-skel" />
          </div>
          <EmptyState
            v-else-if="!list.length"
            icon="HOutline:CubeIcon"
            :title="t('docker.container.noContainers')"
            :description="queryForm.keyword || queryForm.status ? t('docker.ctr.noMatch') : t('docker.container.emptyDesc')"
          />
          <div v-else class="dk-cards">
            <article v-for="row in list" :key="row.id" class="dk-card ct-card">
              <header class="dk-card__head">
                <span class="dk-card__name ct-card__name">{{ nameOf(row) }}</span>
                <StatusPill :variant="pillOf(row).variant" :label="pillOf(row).label" />
              </header>
              <div class="ct-card__line">
                <span class="ct-card__img m-mono">{{ row.image }} · {{ shortLineOf(row) }}</span>
                <span class="ct-card__use m-num">{{ usageOf(row) ? `${usageOf(row)!.cpu.toFixed(1)}% · ${formatBytes(usageOf(row)!.mem)}` : '— · —' }}</span>
              </div>
              <footer class="dk-card__foot">
                <div class="dk-card__lead ct-card__ports">
                  <span v-for="p in portsOf(row)" :key="p" class="ct-port m-mono">{{ p }}</span>
                </div>
                <AppIconButton
                  v-for="qa in quickActionsFor(row)"
                  :key="qa.key"
                  :icon="qa.icon"
                  :label="`${qa.label} ${nameOf(row)}`"
                  :box="44"
                  :class="qa.tone"
                  @click="onRowAction(qa.key, row)"
                />
                <AppIconButton icon="HOutline:EllipsisHorizontalIcon" :label="t('docker.ctr.moreOf', { name: nameOf(row) })" :box="44" @click="openSheet(row)" />
              </footer>
            </article>
          </div>
        </template>

        <div v-if="pagination.total" class="dk-pager">
          <span class="dk-pager__info m-num">{{ rangeText }}</span>
          <Pagination
            v-model:page="pagination.page"
            v-model:page-size="pagination.pageSize"
            :total="pagination.total"
            @change="getList"
          />
        </div>

        <Transition name="dk-pop">
          <div v-if="selected.length" class="dk-batch" role="toolbar" :aria-label="t('docker.ctr.batchLabel')">
            <span class="dk-batch__count">{{ t('docker.ctr.selected', { n: selected.length }) }}</span>
            <button type="button" class="dk-batch__btn" :disabled="batchBusy" @click="runBatch('start')">
              <component :is="menuStore.iconComponents['HOutline:PlayIcon']" />{{ t('docker.lifecycle.start') }}
            </button>
            <button type="button" class="dk-batch__btn" :disabled="batchBusy" @click="runBatch('stop')">
              <component :is="menuStore.iconComponents['HOutline:StopIcon']" />{{ t('docker.lifecycle.stop') }}
            </button>
            <button type="button" class="dk-batch__btn" :disabled="batchBusy" @click="runBatch('restart')">
              <component :is="menuStore.iconComponents['HOutline:ArrowPathIcon']" />{{ t('docker.lifecycle.restart') }}
            </button>
            <button type="button" class="dk-batch__btn is-danger" :disabled="batchBusy" @click="runBatch('delete')">
              <component :is="menuStore.iconComponents['HOutline:TrashIcon']" />{{ t('docker.common.delete') }}
            </button>
            <button type="button" class="dk-batch__btn is-icon" :aria-label="t('docker.ctr.clearSelection')" @click="selected = []">
              <component :is="menuStore.iconComponents['HOutline:XMarkIcon']" />
            </button>
          </div>
        </Transition>
      </div>
    </template>

    <!-- 手机：悬浮创建 -->
    <button v-if="menuStore.isMobile && canManage && ready" type="button" class="dk-fab" :aria-label="t('docker.container.createContainer')" @click="openCreate">
      <UIcon name="i-lucide-plus" />
    </button>

    <!-- 手机：行操作面板 -->
    <DockerActionSheet
      v-model="sheet.open"
      :title="sheet.row ? nameOf(sheet.row) : ''"
      :subtitle="sheet.row ? String(sheet.row.image || '') : ''"
      icon="i-lucide-box"
      tone="a2"
      :pill="sheet.row ? pillOf(sheet.row) : null"
      :quick="sheetQuick"
      :items="sheetItems"
      @select="(key) => sheet.row && onRowAction(key, sheet.row)"
    />

    <!-- 编辑配置 -->
    <FormDialog v-model="editVisible" :title="t('docker.ctr.editConfig')" :width="620" :loading="editSubmitting" @confirm="submitEdit">
      <div class="dk-form">
        <p class="dk-hint">{{ t('docker.ctr.editHint') }}</p>
        <div class="app-field">
          <label class="app-label">{{ t('docker.container.namePlaceholder') }}</label>
          <input v-model="editForm.name" class="app-input m-mono" :placeholder="t('docker.container.namePlaceholder')" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('docker.container.restartPolicy') }}</label>
          <div class="ct-seg" role="radiogroup" :aria-label="t('docker.container.restartPolicy')">
            <button
              v-for="opt in restartSeg"
              :key="opt.value"
              type="button"
              role="radio"
              :aria-checked="editForm.restartPolicy === opt.value"
              :class="{ 'is-on': editForm.restartPolicy === opt.value }"
              @click="editForm.restartPolicy = opt.value"
            >{{ opt.label }}</button>
          </div>
        </div>
        <div class="dk-grid">
          <div class="app-field">
            <label class="app-label">{{ t('docker.container.memoryLimitMb') }}</label>
            <div class="ct-num">
              <input v-model.number="editForm.memoryMB" type="number" min="0" step="64" class="app-input" />
              <span>MB</span>
            </div>
            <div class="ct-presets">
              <button v-for="m in MEMORY_PRESETS" :key="m" type="button" :class="{ 'is-on': editForm.memoryMB === m }" @click="editForm.memoryMB = m">{{ m }}</button>
              <button type="button" :class="{ 'is-on': !editForm.memoryMB }" @click="editForm.memoryMB = 0">{{ t('docker.common.unlimited') }}</button>
            </div>
          </div>
          <div class="app-field">
            <label class="app-label">{{ t('docker.container.cpuLimitCores') }}</label>
            <div class="ct-num">
              <input v-model.number="editForm.nanoCpuCores" type="number" min="0" step="0.5" class="app-input" />
              <span>{{ t('docker.common.cpuUnit') }}</span>
            </div>
            <div class="ct-presets">
              <button v-for="c in CPU_PRESETS" :key="c" type="button" :class="{ 'is-on': editForm.nanoCpuCores === c }" @click="editForm.nanoCpuCores = c">{{ c }}</button>
              <button type="button" :class="{ 'is-on': !editForm.nanoCpuCores }" @click="editForm.nanoCpuCores = 0">{{ t('docker.common.unlimited') }}</button>
            </div>
          </div>
        </div>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="ghost" icon="i-lucide-refresh-cw" @click="close(); openRecreate(editId)">{{ t('docker.ctr.recreate') }}</UButton>
        <UButton color="neutral" variant="soft" @click="close">{{ t('docker.common.cancel') }}</UButton>
        <UButton color="primary" :loading="editSubmitting" @click="submitEdit">{{ t('docker.common.save') }}</UButton>
      </template>
    </FormDialog>

    <!-- 容器详情 -->
    <FormDialog v-model="detailVisible" :title="t('docker.container.containerDetail')" :width="860">
      <div v-if="detail" class="dk-detail">
        <div class="dk-detail__hero">
          <span class="dk-detail__sticker m-sticker m-sticker--a2"><UIcon name="i-lucide-box" /></span>
          <div class="dk-detail__head">
            <div class="dk-detail__title">
              {{ detail.name || '-' }}
              <StatusPill :variant="detailPill.variant" :label="detailPill.label" />
            </div>
            <div class="dk-detail__sub">{{ shortId(detail.id) }} · {{ detailImageName }} · {{ detailNetworkMode }}</div>
            <div v-if="detailPortTags.length" class="dk-tags ct-detail-ports">
              <span v-for="p in detailPortTags" :key="p" class="ct-port m-mono">{{ p }}</span>
            </div>
          </div>
        </div>

        <div class="dk-grid">
          <div>
            <div class="dk-detail__label">{{ t('docker.container.runtime') }}</div>
            <dl class="dk-kv ct-kv">
              <div><dt>PID</dt><dd class="m-num">{{ detail.state?.pid || '-' }}</dd></div>
              <div><dt>{{ t('docker.container.restartCount') }}</dt><dd class="m-num">{{ detail.restartCount }}</dd></div>
              <div><dt>{{ t('docker.container.exitCode') }}</dt><dd class="m-num">{{ detail.state?.exitCode ?? '-' }}</dd></div>
              <div><dt>{{ t('docker.common.platform') }}</dt><dd>{{ detail.platform || '-' }}</dd></div>
              <div><dt>{{ t('docker.container.created') }}</dt><dd class="m-num">{{ formatDateTime(detail.created) }}</dd></div>
              <div><dt>{{ t('docker.container.started') }}</dt><dd class="m-num">{{ formatDateTime(detail.state?.startedAt) }}</dd></div>
            </dl>
          </div>
          <div>
            <div class="dk-detail__label">{{ t('docker.container.resource') }}</div>
            <dl class="dk-kv ct-kv">
              <div><dt>{{ t('docker.container.restartPolicy') }}</dt><dd class="m-mono">{{ detailRestartPolicy }}</dd></div>
              <div><dt>{{ t('docker.container.networkMode') }}</dt><dd class="m-mono">{{ detailNetworkMode }}</dd></div>
              <div><dt>{{ t('docker.common.memory') }}</dt><dd class="m-num">{{ detailMemoryLimit }}</dd></div>
              <div><dt>CPU</dt><dd class="m-num">{{ detailCpuLimit }}</dd></div>
              <div><dt>{{ t('docker.container.autoRemove') }}</dt><dd>{{ detail.hostConfig?.autoRemove ? t('docker.common.yes') : t('docker.common.no') }}</dd></div>
              <div><dt>{{ t('docker.container.privileged') }}</dt><dd>{{ detail.hostConfig?.privileged ? t('docker.common.yes') : t('docker.common.no') }}</dd></div>
            </dl>
          </div>
        </div>

        <div v-if="detailNetworks.length">
          <div class="dk-detail__label">{{ t('docker.common.network') }}</div>
          <div class="ct-list">
            <div v-for="item in detailNetworks" :key="item.name" class="ct-list__row ct-list--net">
              <strong>{{ item.name }}</strong>
              <span class="m-mono">{{ item.ipAddress || '-' }}</span>
              <span class="m-mono">{{ item.gateway || '-' }}</span>
              <span class="m-mono">{{ item.macAddress || '-' }}</span>
            </div>
          </div>
        </div>

        <div>
          <div class="dk-detail__label">{{ t('docker.container.startup') }}</div>
          <div class="dk-code">{{ detailCommand }}</div>
        </div>

        <div v-if="detail.mounts?.length">
          <div class="dk-detail__label">{{ t('docker.container.mounts') }}</div>
          <div class="ct-list">
            <div class="ct-list__row ct-list--mounts ct-list__head">
              <span>{{ t('docker.common.type') }}</span><span>{{ t('docker.common.source') }}</span><span>{{ t('docker.common.target') }}</span><span>{{ t('docker.common.mode') }}</span><span>RW</span>
            </div>
            <div v-for="(m, i) in detail.mounts" :key="i" class="ct-list__row ct-list--mounts">
              <span>{{ m.type }}</span>
              <span class="m-mono" :title="m.source">{{ m.source || m.name }}</span>
              <span class="m-mono" :title="m.destination">{{ m.destination }}</span>
              <span>{{ m.mode || '-' }}</span>
              <span><StatusPill :variant="m.rw ? 'success' : 'warning'" :label="m.rw ? 'RW' : 'RO'" :dot="false" /></span>
            </div>
          </div>
        </div>
      </div>
      <div v-else class="dk-detail__loading">{{ t('docker.common.noData') }}</div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" icon="i-lucide-file-text" :disabled="!detail" @click="openDetailLogs">{{ t('docker.common.logs') }}</UButton>
        <UButton color="neutral" variant="soft" icon="i-lucide-bar-chart-3" :disabled="!detail" @click="openDetailStats">{{ t('docker.common.stats') }}</UButton>
        <UButton v-if="canManage" color="neutral" variant="soft" icon="i-lucide-terminal" :disabled="!detail" @click="openDetailExec">{{ t('docker.common.terminal') }}</UButton>
        <UButton color="primary" @click="close">{{ t('docker.common.close') }}</UButton>
      </template>
    </FormDialog>

    <DockerTaskDialogs v-model="taskDialogsVisible" :active-task="activeTask" @finished="handleTaskFinished" />
    <DockerContainerLogsDialog v-model="logsVisible" :container-id="logsContainerId" :container-name="logsContainerName" :ws-path="logsWsPath" />
    <DockerContainerStatsDialog v-model="statsVisible" :container-id="statsContainerId" :container-name="statsContainerName" />
    <DockerContainerExecDialog v-model="execVisible" :container-id="execContainerId" :container-name="execContainerName" :ws-path="execWsPath" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { PAGE_GROUPS, pageTabTo } from '@/config/pageGroups'
import {
  deleteDockerContainer, getDockerContainer, killDockerContainer, listDockerContainerUsage, listDockerContainers,
  pauseDockerContainer, restartDockerContainer, startDockerContainer, stopDockerContainer, unpauseDockerContainer,
  updateDockerContainer,
} from '@/api/docker'
import DockerHeader from '@/views/docker/components/DockerHeader.vue'
import DockerSubNav from '@/views/docker/components/DockerSubNav.vue'
import DockerUnavailable from '@/views/docker/components/DockerUnavailable.vue'
import DockerPullTasks from '@/views/docker/components/DockerPullTasks.vue'
import DockerActionSheet from '@/views/docker/components/DockerActionSheet.vue'
import DockerTaskDialogs from '@/views/docker/components/DockerTaskDialogs.vue'
import DockerContainerLogsDialog from '@/views/docker/components/DockerContainerLogsDialog.vue'
import DockerContainerStatsDialog from '@/views/docker/components/DockerContainerStatsDialog.vue'
import DockerContainerExecDialog from '@/views/docker/components/DockerContainerExecDialog.vue'
import ContainerCreate from './ContainerCreate.vue'
import { useDockerStatus } from '@/views/docker/components/useDockerStatus'
import { useDockerTasks } from '@/views/docker/components/useDockerTasks'
import { formatAgo, formatBytes, formatDateTime, formatDuration, formatMonthDay, shortId } from '@/views/docker/components/format'
import { PERM } from '@/config/permission'
import { useButtonPermission } from '@/composables/useButtonPermission'
import { Dialog } from '@/utils/dialog'
import { showRequestError } from '@/utils/request'
import { DockerTaskType } from '@/types/v1/docker'
import type {
  DockerContainerInfo, DockerContainerListStats, DockerContainerSummary, DockerTaskInfo,
} from '@/types/v1/docker'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'

defineOptions({ name: 'DockerContainerView' })

const menuStore = useMenuStore()
const route = useRoute()
const router = useRouter()
const { t, locale } = useI18n()
const canManage = useButtonPermission([PERM.DOCKER_CONTAINER_MANAGE], [])
const { ready, loaded: engineLoaded, refresh: refreshEngine } = useDockerStatus()
const { finishedTick, refresh: refreshTasks } = useDockerTasks()

const ACTIVE_TYPES = [DockerTaskType.DOCKER_TASK_TYPE_CONTAINER_CREATE, DockerTaskType.DOCKER_TASK_TYPE_CONTAINER_RECREATE]

type Row = Record<string, unknown>
type Category = '' | 'running' | 'stopped' | 'exited' | 'paused'
const list = ref<DockerContainerSummary[]>([])
const stats = ref<DockerContainerListStats | null>(null)
const loading = ref(false)
const selected = ref<string[]>([])
const queryForm = reactive({ keyword: '', status: '' as Category })
const pagination = ref({ page: 1, pageSize: 10, total: 0 })

const statusChips = computed(() => {
  const s = stats.value
  const n = (v: unknown) => (s ? Number(v || 0) : null)
  const chips = [
    { label: t('docker.common.all'), value: '' as Category, count: n(s?.total) },
    { label: t('docker.states.running'), value: 'running' as Category, count: n(s?.running) },
    { label: t('docker.ctr.catStopped'), value: 'stopped' as Category, count: n(s?.stopped) },
    { label: t('docker.ctr.catExited'), value: 'exited' as Category, count: n(s?.exited) },
  ]
  if (Number(s?.paused || 0) || queryForm.status === 'paused') chips.push({ label: t('docker.ctr.catPaused'), value: 'paused', count: n(s?.paused) })
  return chips
})

const columns = computed<DataTableColumn[]>(() => [
  { key: 'names', title: t('docker.common.name'), minWidth: 160 },
  { key: 'image', title: t('docker.common.image'), minWidth: 170 },
  { key: 'state', title: t('docker.common.status'), width: 170 },
  { key: 'ports', title: t('docker.ctr.ports'), minWidth: 150, ellipsis: false },
  { key: 'cpu', title: 'CPU', width: 120 },
  { key: 'mem', title: t('docker.common.memory'), width: 100 },
  { key: 'operation', title: t('docker.common.operation'), width: 190, align: 'right' },
])

/* ───────── 行展示 ───────── */
const asSummary = (row: Row) => row as unknown as DockerContainerSummary
const nameOf = (row: Row) => (asSummary(row).names || []).map((n) => n.replace(/^\//, '')).join(', ') || shortId(row.id)
const catOf = (row: Row): Category => {
  const r = asSummary(row)
  if (r.state === 'running' || r.state === 'restarting') return 'running'
  if (r.state === 'paused') return 'paused'
  if (r.state === 'dead' || (r.state === 'exited' && Number(r.exitCode))) return 'exited'
  return 'stopped'
}
type PillVariant = 'success' | 'warning' | 'error' | 'neutral' | 'info'
const pillOf = (row: Row): { variant: PillVariant; label: string } => {
  const r = asSummary(row)
  if (r.state === 'restarting') return { variant: 'warning', label: t('docker.states.restarting') }
  if (r.state === 'removing') return { variant: 'neutral', label: t('docker.states.removing') }
  if (r.state === 'created') return { variant: 'neutral', label: t('docker.states.created') }
  const cat = catOf(row)
  if (cat === 'running') return { variant: 'success', label: t('docker.states.running') }
  if (cat === 'paused') return { variant: 'warning', label: t('docker.ctr.catPaused') }
  if (cat === 'exited') return { variant: 'error', label: t('docker.ctr.catExited') }
  return { variant: 'neutral', label: t('docker.ctr.catStopped') }
}
const now = ref(Date.now())
const sinceMs = (value: unknown) => {
  const d = value ? new Date(String(value)) : null
  return d && !Number.isNaN(d.getTime()) && d.getFullYear() > 1971 ? now.value - d.getTime() : -1
}
const subLineOf = (row: Row) => {
  const r = asSummary(row)
  const cat = catOf(row)
  if (cat === 'running' || cat === 'paused') {
    const ms = sinceMs(r.startedAt)
    return ms >= 0 ? t('docker.ctr.upFor', { d: formatDuration(ms, t) }) : '—'
  }
  if (sinceMs(r.finishedAt) < 0) return t('docker.ctr.neverStarted')
  if (cat === 'exited') return t('docker.ctr.exitAgo', { code: r.exitCode, ago: formatAgo(r.finishedAt, t) })
  return t('docker.ctr.stoppedOn', { date: formatMonthDay(r.finishedAt, locale.value) })
}
const shortLineOf = (row: Row) => {
  const r = asSummary(row)
  const cat = catOf(row)
  if (cat === 'running' || cat === 'paused') {
    const ms = sinceMs(r.startedAt)
    return ms >= 0 ? formatDuration(ms, t) : '—'
  }
  if (sinceMs(r.finishedAt) < 0) return t('docker.ctr.neverStarted')
  if (cat === 'exited') return t('docker.ctr.exitCode', { code: r.exitCode })
  return formatMonthDay(r.finishedAt, locale.value)
}
const portsOf = (row: Row) => {
  const r = asSummary(row)
  if (r.networkMode === 'host') return ['host']
  const out = new Set<string>()
  for (const p of r.ports || []) {
    const proto = (p.type || 'tcp').toLowerCase() === 'udp' ? '/udp' : ''
    const ip = p.ip && p.ip !== '0.0.0.0' && p.ip !== '::' ? `${p.ip.includes(':') ? `[${p.ip}]` : p.ip}:` : ''
    if (p.publicPort) out.add(`${ip}${p.publicPort}→${p.privatePort}${proto}`)
    else if (p.privatePort) out.add(`${p.privatePort}${proto}`)
  }
  return [...out]
}

/* ───────── 资源占用（异步补齐，可见时每 10 秒刷新） ───────── */
const usage = ref<Record<string, { cpu: number; mem: number; limit: number }>>({})
const usageOf = (row: Row) => (catOf(row) === 'running' || catOf(row) === 'paused' ? usage.value[String(row.id)] : undefined)
const cpuBar = (cpu: number) => (cpu > 0 ? Math.max(4, Math.min(100, cpu)) : 0)
let usageTimer: ReturnType<typeof setInterval> | null = null
let usageBusy = false
const loadUsage = async () => {
  const ids = list.value.filter((r) => r.state === 'running').map((r) => r.id)
  if (!ids.length || usageBusy || document.hidden) return
  usageBusy = true
  try {
    const { data } = await listDockerContainerUsage({ ids })
    const next = { ...usage.value }
    for (const item of data?.items || []) {
      next[item.id] = { cpu: Number(item.cpuPercent) || 0, mem: Number(item.memoryUsage) || 0, limit: Number(item.memoryLimit) || 0 }
    }
    usage.value = next
  } catch { /* 资源占用是辅助信息 */ } finally {
    usageBusy = false
  }
}
const startUsageTimer = () => {
  stopUsageTimer()
  usageTimer = setInterval(() => {
    now.value = Date.now()
    if (!createState.mode) void loadUsage()
  }, 10000)
}
const stopUsageTimer = () => {
  if (usageTimer) clearInterval(usageTimer)
  usageTimer = null
}

/* ───────── 列表 ───────── */
const getList = async () => {
  if (!ready.value) return
  loading.value = true
  try {
    const { data } = await listDockerContainers({
      page: pagination.value.page, pageSize: pagination.value.pageSize,
      status: queryForm.status, keyword: queryForm.keyword.trim(), network: '', labels: {},
    })
    list.value = data?.items || []
    stats.value = data?.stats || null
    pagination.value.total = Number(data?.total || 0)
    const ids = new Set(list.value.map((i) => i.id))
    selected.value = selected.value.filter((id) => ids.has(id))
    now.value = Date.now()
    void loadUsage()
  } catch {
    list.value = []
    pagination.value.total = 0
  } finally {
    loading.value = false
  }
}
const search = () => { pagination.value.page = 1; void getList() }
const rangeText = computed(() => {
  const { page, pageSize, total } = pagination.value
  const from = total ? (page - 1) * pageSize + 1 : 0
  return t('docker.page.range', { from, to: Math.min(total, page * pageSize), total })
})
const init = async (force = false) => {
  await refreshEngine(force)
  if (ready.value) await getList()
}

/* ───────── 动作 ───────── */
type LifecycleAction = 'start' | 'stop' | 'restart' | 'kill' | 'pause' | 'unpause' | 'delete'
const actionMap: Record<LifecycleAction, { labelKey: string; fn: (id: string) => Promise<unknown>; confirm: boolean }> = {
  start: { labelKey: 'docker.lifecycle.start', fn: (id) => startDockerContainer({ id }), confirm: false },
  stop: { labelKey: 'docker.lifecycle.stop', fn: (id) => stopDockerContainer({ id, timeoutSeconds: 10 }), confirm: true },
  restart: { labelKey: 'docker.lifecycle.restart', fn: (id) => restartDockerContainer({ id, timeoutSeconds: 10 }), confirm: true },
  kill: { labelKey: 'docker.lifecycle.kill', fn: (id) => killDockerContainer({ id, signal: 'SIGKILL' }), confirm: true },
  pause: { labelKey: 'docker.lifecycle.pause', fn: (id) => pauseDockerContainer({ id }), confirm: false },
  unpause: { labelKey: 'docker.lifecycle.unpause', fn: (id) => unpauseDockerContainer({ id }), confirm: false },
  delete: { labelKey: 'docker.lifecycle.delete', fn: (id) => deleteDockerContainer({ id, force: false, removeVolumes: false }), confirm: true },
}

/** 行内常用：日志 / 终端 / 启停（停止红、启动绿；异常退出的给「重启」） */
const quickActionsFor = (row: Row) => {
  const out: { key: string; icon: string; label: string; tone: string }[] = [
    { key: 'logs', icon: 'HOutline:DocumentTextIcon', label: t('docker.common.logs'), tone: '' },
  ]
  if (!canManage.value) return out
  const cat = catOf(row)
  out.push({ key: 'exec', icon: 'HOutline:CommandLineIcon', label: t('docker.common.terminal'), tone: '' })
  if (row.state === 'running') out.push({ key: 'stop', icon: 'HOutline:StopIcon', label: t('docker.lifecycle.stop'), tone: 'is-stop' })
  else if (cat === 'paused') out.push({ key: 'unpause', icon: 'HOutline:PlayIcon', label: t('docker.lifecycle.unpause'), tone: 'is-start' })
  else if (cat === 'exited') out.push({ key: 'start', icon: 'HOutline:ArrowPathIcon', label: t('docker.lifecycle.restart'), tone: 'is-start' })
  else if (cat === 'stopped' && row.state !== 'removing') out.push({ key: 'start', icon: 'HOutline:PlayIcon', label: t('docker.lifecycle.start'), tone: 'is-start' })
  return out
}
const menuActionsFor = (row: Row): ActionMenuItem[] => {
  const manage = canManage.value
  const running = row.state === 'running'
  return [
    { key: 'pause', label: t('docker.lifecycle.pause'), icon: 'HOutline:PauseIcon', hidden: !manage || !running },
    { key: 'restart', label: t('docker.lifecycle.restart'), icon: 'HOutline:ArrowPathIcon', hidden: !manage || !running },
    { key: 'stats', label: t('docker.common.stats'), icon: 'HOutline:ChartBarIcon', hidden: !running },
    { key: 'edit', label: t('docker.ctr.editConfig'), icon: 'HOutline:PencilSquareIcon', hidden: !manage },
    { key: 'recreate', label: t('docker.ctr.recreate'), icon: 'HOutline:ArrowPathRoundedSquareIcon', hidden: !manage },
    { key: 'detail', label: t('docker.common.detail'), icon: 'HOutline:InformationCircleIcon' },
    { key: 'kill', label: t('docker.lifecycle.kill'), icon: 'HOutline:BoltSlashIcon', hidden: !manage || !running },
    { key: 'delete', label: t('docker.ctr.deleteContainer'), icon: 'HOutline:TrashIcon', danger: true, hidden: !manage },
  ]
}

/* 手机操作面板 */
const sheet = reactive({ open: false, row: null as Row | null })
const openSheet = (row: Row) => {
  sheet.row = row
  sheet.open = true
}
const sheetQuick = computed<ActionMenuItem[]>(() => {
  const row = sheet.row
  if (!row) return []
  const running = row.state === 'running'
  return [
    { key: 'logs', label: t('docker.common.logs'), icon: 'i-lucide-file-text' },
    { key: 'exec', label: t('docker.common.terminal'), icon: 'i-lucide-terminal', hidden: !canManage.value, disabled: !running },
    { key: catOf(row) === 'running' ? 'restart' : 'start', label: t('docker.lifecycle.restart'), icon: 'i-lucide-rotate-cw', hidden: !canManage.value || catOf(row) === 'paused' },
    { key: 'stats', label: t('docker.common.stats'), icon: 'i-lucide-bar-chart-3', disabled: !running },
  ]
})
const sheetItems = computed<ActionMenuItem[]>(() => {
  const row = sheet.row
  if (!row) return []
  const manage = canManage.value
  const cat = catOf(row)
  return [
    { key: 'edit', label: t('docker.ctr.editConfig'), icon: 'i-lucide-pencil', hidden: !manage },
    { key: 'recreate', label: t('docker.ctr.recreate'), icon: 'i-lucide-refresh-cw', hidden: !manage },
    { key: 'pause', label: t('docker.lifecycle.pause'), icon: 'i-lucide-pause', hidden: !manage || row.state !== 'running' },
    { key: 'unpause', label: t('docker.lifecycle.unpause'), icon: 'i-lucide-play', hidden: !manage || cat !== 'paused' },
    { key: 'stop', label: t('docker.lifecycle.stop'), icon: 'i-lucide-square', hidden: !manage || row.state !== 'running' },
    { key: 'detail', label: t('docker.ctr.detailInspect'), icon: 'i-lucide-info' },
    { key: 'delete', label: t('docker.ctr.deleteContainer'), icon: 'i-lucide-trash-2', danger: true, hidden: !manage },
  ]
})

const findRow = (id: string) => list.value.find((x) => x.id === id)
const onRowAction = (key: string, row: Row) => {
  const record = findRow(String(row.id))
  if (!record) return
  if (key in actionMap) { void handleAction(record, key as LifecycleAction); return }
  if (key === 'logs') void openLogs(record)
  else if (key === 'stats') openStats(record)
  else if (key === 'exec') void openExec(record)
  else if (key === 'edit') void openEdit(record)
  else if (key === 'recreate') openRecreate(record.id)
  else if (key === 'detail') void openDetail(record)
}

const handleAction = async (row: DockerContainerSummary, action: LifecycleAction) => {
  const def = actionMap[action]
  const name = nameOf(row as unknown as Row)
  const actionLabel = t(def.labelKey)
  const run = async () => {
    try {
      await def.fn(row.id)
      feedback.success(t('docker.container.actionSuccess', { action: actionLabel }))
      await getList()
      void refreshEngine(true)
    } catch (e) { showRequestError(e, t('docker.container.actionFailed', { action: actionLabel })) }
  }
  if (def.confirm) {
    Dialog.confirm({
      title: t('docker.container.confirmActionTitle', { action: actionLabel }),
      content: t('docker.container.confirmActionContent', { action: actionLabel, name }),
      confirmText: t('docker.container.confirmActionText', { action: actionLabel }),
      cancelText: t('docker.common.cancel'),
      onConfirm: run,
    })
    return
  }
  await run()
}

/* 批量 */
const batchBusy = ref(false)
const runBatch = (action: 'start' | 'stop' | 'restart' | 'delete') => {
  const rows = selected.value.map(findRow).filter((r): r is DockerContainerSummary => !!r)
  if (!rows.length) return
  const def = actionMap[action]
  const actionLabel = t(def.labelKey)
  const run = async () => {
    batchBusy.value = true
    let ok = 0
    for (const row of rows) {
      try {
        await def.fn(row.id)
        ok++
      } catch (e) { showRequestError(e, `${nameOf(row as unknown as Row)}：${t('docker.container.actionFailed', { action: actionLabel })}`) }
    }
    batchBusy.value = false
    if (ok) feedback.success(t('docker.ctr.batchDone', { action: actionLabel, n: ok }))
    selected.value = []
    await getList()
    void refreshEngine(true)
  }
  if (action === 'start') { void run(); return }
  Dialog.confirm({
    title: t('docker.ctr.batchConfirmTitle', { action: actionLabel }),
    content: t('docker.ctr.batchConfirmContent', { action: actionLabel, n: rows.length }),
    confirmText: t('docker.container.confirmActionText', { action: actionLabel }),
    cancelText: t('docker.common.cancel'),
    onConfirm: run,
  })
}

/* ───────── 创建 / 重建子视图（?create=1 / ?recreate=<id>） ───────── */
const createState = reactive({ mode: '' as '' | 'create' | 'recreate', id: '', image: '', key: 0 })
watch(
  () => [route.path, route.query.tab, route.query.create, route.query.recreate, route.query.image] as const,
  ([path, tab, create, recreate, image]) => {
    if (path !== PAGE_GROUPS.docker.path || (tab && tab !== 'container')) return
    const id = typeof recreate === 'string' ? recreate : ''
    const mode = id ? 'recreate' : create === '1' ? 'create' : ''
    if (mode && (mode !== createState.mode || id !== createState.id)) createState.key++
    createState.mode = mode
    createState.id = id
    createState.image = typeof image === 'string' ? image : ''
  },
  { immediate: true },
)
// 引擎不可用时（直接打开 ?create=1 链接）停在列表页显示不可用状态，不挂载创建页去请求镜像/网络/容器
const createMode = computed(() => (ready.value ? createState.mode : ''))
const openCreate = () => { void router.push(pageTabTo('docker', 'container', { create: '1' })) }
const openRecreate = (id: string) => { if (id) void router.push(pageTabTo('docker', 'container', { recreate: id })) }
const closeCreate = () => {
  const back = (window.history.state as { back?: string } | null)?.back || ''
  if (back.startsWith(PAGE_GROUPS.docker.path) && !back.includes('create=') && !back.includes('recreate=')) router.back()
  else void router.replace(pageTabTo('docker', 'container'))
}
const onCreateDone = (result: { id?: string; task?: DockerTaskInfo }) => {
  // 创建完回到容器列表（从镜像页「用它建容器」进来时也一样），新容器 / 创建任务就在列表里
  void router.replace(pageTabTo('docker', 'container'))
  if (result.task?.id) void refreshTasks()
  void getList()
  void refreshEngine(true)
}

/* ───────── 任务 ───────── */
const taskDialogsVisible = ref(false)
const activeTask = ref<DockerTaskInfo | null>(null)
const openTasks = () => { activeTask.value = null; taskDialogsVisible.value = true }
const openTask = (task: DockerTaskInfo | undefined) => {
  if (!task?.id) return
  activeTask.value = task
  taskDialogsVisible.value = true
}
const handleTaskFinished = async () => { await getList() }
watch(finishedTick, () => { void getList(); void refreshEngine(true) })

/* ───────── 编辑配置（可在线修改的：名称 / 重启策略 / 资源限制） ───────── */
const restartSeg = computed(() => [
  { label: t('docker.container.restartNo'), value: '' },
  { label: t('docker.container.restartAlways'), value: 'always' },
  { label: t('docker.container.restartOnFailure'), value: 'on-failure' },
  { label: t('docker.container.restartUnlessStopped'), value: 'unless-stopped' },
])
const MEMORY_PRESETS = [512, 1024, 2048, 4096]
const CPU_PRESETS = [0.5, 1, 2, 4]
const editVisible = ref(false)
const editSubmitting = ref(false)
const editId = ref('')
const editForm = reactive({ name: '', restartPolicy: '', memoryMB: 0, nanoCpuCores: 0 })
const openEdit = async (row: DockerContainerSummary) => {
  editId.value = row.id
  Object.assign(editForm, { name: nameOf(row as unknown as Row), restartPolicy: '', memoryMB: 0, nanoCpuCores: 0 })
  editVisible.value = true
  try {
    const { data } = await getDockerContainer({ id: row.id })
    const info = data?.info
    if (!info) return
    const host = info.hostConfig
    editForm.name = info.name || editForm.name
    editForm.restartPolicy = host?.restartPolicy === 'no' ? '' : host?.restartPolicy || ''
    if (host?.memory) editForm.memoryMB = Math.round(Number(host.memory) / 1024 / 1024)
    if (host?.nanoCpus) editForm.nanoCpuCores = Math.round((Number(host.nanoCpus) / 1e9) * 10) / 10
  } catch { /* 用列表里的值 */ }
}
const submitEdit = async () => {
  editSubmitting.value = true
  try {
    await updateDockerContainer({
      id: editId.value, name: editForm.name.trim(), restartPolicy: editForm.restartPolicy,
      memory: (Number(editForm.memoryMB) || 0) * 1024 * 1024, memorySwap: 0, cpuShares: 0, cpuQuota: 0, cpuPeriod: 0,
      nanoCpus: Math.round((Number(editForm.nanoCpuCores) || 0) * 1e9), recreate: false, force: false, removeVolumes: false, options: undefined,
    })
    feedback.success(t('docker.container.updateSuccess'))
    editVisible.value = false
    await getList()
  } catch (e) { showRequestError(e, t('docker.container.updateFailed')) }
  finally { editSubmitting.value = false }
}

/* ───────── 详情 ───────── */
const detailVisible = ref(false)
const detail = ref<DockerContainerInfo | null>(null)
const detailPill = computed(() => {
  const s = detail.value?.state
  if (!s) return { variant: 'neutral' as PillVariant, label: '-' }
  return pillOf({ state: s.status, exitCode: s.exitCode } as unknown as Row)
})
const detailCommand = computed(() => {
  const item = detail.value
  if (!item) return '-'
  return [item.path, ...(item.args || [])].filter(Boolean).join(' ') || '-'
})
const detailImageName = computed(() => detail.value?.config?.image || detail.value?.image || '-')
const detailRestartPolicy = computed(() => detail.value?.hostConfig?.restartPolicy || '-')
const detailNetworkMode = computed(() => detail.value?.hostConfig?.networkMode || '-')
const detailMemoryLimit = computed(() => {
  const memory = Number(detail.value?.hostConfig?.memory || 0)
  return memory ? formatBytes(memory) : t('docker.common.unlimited')
})
const detailCpuLimit = computed(() => {
  const host = detail.value?.hostConfig
  const nano = Number(host?.nanoCpus || 0)
  if (nano) return `${(nano / 1e9).toFixed(2)} ${t('docker.common.cpuUnit')}`
  const quota = Number(host?.cpuQuota || 0)
  const period = Number(host?.cpuPeriod || 0)
  if (quota && period) return `${(quota / period).toFixed(2)} ${t('docker.common.cpuUnit')}`
  return t('docker.common.unlimited')
})
const detailNetworks = computed(() =>
  Object.entries(detail.value?.network?.networks || {}).map(([name, item]) => ({
    name, ipAddress: item?.ipAddress || '', gateway: item?.gateway || '', macAddress: item?.macAddress || '',
  })),
)
const detailPortTags = computed(() =>
  (detail.value?.hostConfig?.portBindings || []).map((b) => {
    const cp = b.containerPort || ''
    return b.hostPort ? `${b.hostIp ? `${b.hostIp}:` : ''}${b.hostPort}→${cp}` : cp
  }).filter(Boolean),
)
const openDetail = async (row: DockerContainerSummary) => {
  detailVisible.value = true
  detail.value = null
  try { const { data } = await getDockerContainer({ id: row.id }); detail.value = data?.info || null }
  catch (e) { showRequestError(e, t('docker.container.getDetailFailed')) }
}
const openDetailLogs = () => {
  if (!detail.value?.logsWsPath) { feedback.warning(t('docker.container.missingLogsWsPath')); return }
  logsContainerId.value = detail.value.id
  logsContainerName.value = detail.value.name || detail.value.id
  logsWsPath.value = detail.value.logsWsPath
  logsVisible.value = true
}
const openDetailStats = () => {
  if (!detail.value) return
  statsContainerId.value = detail.value.id
  statsContainerName.value = detail.value.name || detail.value.id
  statsVisible.value = true
}
const openDetailExec = () => {
  if (!detail.value?.execWsPath) { feedback.warning(t('docker.container.missingExecWsPath')); return }
  execContainerId.value = detail.value.id
  execContainerName.value = detail.value.name || detail.value.id
  execWsPath.value = detail.value.execWsPath
  execVisible.value = true
}

/* ───────── 日志 / 统计 / 终端 ───────── */
const logsVisible = ref(false)
const logsContainerId = ref('')
const logsContainerName = ref('')
const logsWsPath = ref('')
const openLogs = async (row: DockerContainerSummary) => {
  try {
    const { data } = await getDockerContainer({ id: row.id })
    const info = data?.info
    if (!info?.logsWsPath) { feedback.warning(t('docker.container.missingLogsWsPath')); return }
    logsContainerId.value = info.id || row.id
    logsContainerName.value = info.name || nameOf(row as unknown as Row)
    logsWsPath.value = info.logsWsPath
    logsVisible.value = true
  } catch (e) { showRequestError(e, t('docker.container.getDetailFailed')) }
}
const statsVisible = ref(false)
const statsContainerId = ref('')
const statsContainerName = ref('')
const openStats = (row: DockerContainerSummary) => {
  statsContainerId.value = row.id
  statsContainerName.value = nameOf(row as unknown as Row)
  statsVisible.value = true
}
const execVisible = ref(false)
const execContainerId = ref('')
const execContainerName = ref('')
const execWsPath = ref('')
const openExec = async (row: DockerContainerSummary) => {
  try {
    const { data } = await getDockerContainer({ id: row.id })
    const info = data?.info
    if (!info?.execWsPath) { feedback.warning(t('docker.container.missingExecWsPath')); return }
    execContainerId.value = info.id || row.id
    execContainerName.value = info.name || nameOf(row as unknown as Row)
    execWsPath.value = info.execWsPath
    execVisible.value = true
  } catch (e) { showRequestError(e, t('docker.container.getDetailFailed')) }
}

onMounted(() => { void init() })
onActivated(() => {
  startUsageTimer()
  if (ready.value && !loading.value) void loadUsage()
})
onDeactivated(stopUsageTimer)
onBeforeUnmount(stopUsageTimer)
</script>

<style scoped lang="scss">
@use '../components/dk-page.scss';

.ct-name {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
}
.ct-name strong {
  font-weight: 800;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ct-name span,
.ct-sub {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.ct-img {
  display: block;
  max-width: 220px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ct-state {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
}
.ct-port {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 8px;
  border-radius: 8px;
  background: var(--m-a2Soft);
  color: var(--m-ink);
  font-size: 0.75rem;
  font-weight: 700;
  white-space: nowrap;
}
.ct-cpu {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.8125rem;
}
.ct-bar {
  flex: none;
  width: 48px;
  height: 6px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  overflow: hidden;
}
.ct-bar span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--m-p);
}
.ct-mem {
  font-size: 0.8125rem;
}
.dk-acts :deep(.is-stop),
.dk-card__foot :deep(.is-stop) {
  color: var(--m-bad);
}
.dk-acts :deep(.is-start),
.dk-card__foot :deep(.is-start) {
  color: var(--m-ok);
}
.dk-acts :deep(.is-stop:hover),
.dk-card__foot :deep(.is-stop:hover) {
  background: var(--m-badSoft);
}
.dk-acts :deep(.is-start:hover),
.dk-card__foot :deep(.is-start:hover) {
  background: var(--m-okSoft);
}

/* 手机卡片 */
.ct-skel {
  height: 132px;
}
.ct-card {
  gap: 10px;
}
.ct-card__name {
  font-size: 1rem;
}
.ct-card__line {
  display: flex;
  align-items: baseline;
  gap: 10px;
  margin-top: -4px;
  padding-right: 6px;
  min-width: 0;
}
.ct-card__img {
  flex: 1;
  min-width: 0;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ct-card__use {
  flex: none;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}
.ct-card__ports {
  align-items: center;
  flex-wrap: wrap;
  overflow: hidden;
}

/* 编辑配置 */
.ct-seg {
  display: flex;
  flex-wrap: wrap;
  gap: 2px;
  padding: 3px;
  border-radius: 12px;
  background: var(--el-fill-color-light);
}
.ct-seg button {
  flex: 1;
  min-width: 0;
  min-height: 38px;
  padding: 0 10px;
  border: none;
  border-radius: 9px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
}
.ct-seg button.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}
.ct-num {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.ct-num .app-input {
  flex: 1;
  min-width: 0;
}
.ct-presets {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 6px;
}
.ct-presets button {
  min-height: 32px;
  padding: 0 12px;
  border: 1.5px solid var(--el-border-color-light);
  border-radius: 999px;
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  font-weight: 700;
  cursor: pointer;
}
.ct-presets button.is-on {
  border-color: var(--m-p);
  background: var(--m-pSoft);
  color: var(--m-pInk);
}

/* 详情 */
.ct-detail-ports {
  margin-top: 6px;
}
.ct-kv {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}
.ct-list {
  border-radius: var(--app-radius);
  border: 1px solid var(--el-border-color-lighter);
  overflow: hidden;
  font-size: 0.78125rem;
}
.ct-list__row {
  display: grid;
  gap: 8px;
  align-items: center;
  padding: 8px 12px;
  border-top: 1px solid var(--el-border-color-lighter);
  color: var(--el-text-color-regular);
}
.ct-list__row:first-child {
  border-top: none;
}
.ct-list__row > * {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ct-list__head {
  background: var(--el-fill-color-light);
  font-size: 0.71875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.ct-list--net {
  grid-template-columns: repeat(4, minmax(0, 1fr));
}
.ct-list--mounts {
  grid-template-columns: 0.7fr 1.6fr 1.4fr 0.7fr 0.6fr;
}
.dk-pop-enter-active,
.dk-pop-leave-active {
  transition: opacity 0.18s, transform 0.18s;
}
.dk-pop-enter-from,
.dk-pop-leave-to {
  opacity: 0;
  transform: translateY(8px);
}
@media (width <= 768px) {
  .ct-kv {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .ct-list--mounts {
    grid-template-columns: 0.8fr 1.4fr 1.4fr 0.6fr;
  }
  .ct-list--mounts > :nth-child(4) {
    display: none;
  }
}
</style>
