<!-- Docker 容器（重写 · P1 列表）：PageHeader(创建容器/任务) + FilterBar(名称/状态/镜像) + DataTable/移动卡 + Pagination。
     行内 ActionMenu 按状态给生命周期动作 + 日志/统计/终端 + 编辑/详情/删除（管理权限门控）。
     创建/详情/编辑走 FormDialog；日志/终端复用 TerminalConsole，统计令牌壳（见 components/*）。 -->
<template>
  <div class="dk-page">
    <PageHeader :title="t('docker.container.title')" :description="t('docker.container.pageDesc')">
      <template #actions>
        <UButton color="neutral" variant="soft" icon="i-lucide-clock" @click="openTasks">
          {{ t('docker.common.tasks') }}
        </UButton>
        <UButton v-if="canManage" color="primary" icon="i-lucide-plus" @click="openCreate">
          {{ t('docker.container.createContainer') }}
        </UButton>
      </template>
    </PageHeader>

    <FilterBar @search="search" @reset="reset">
      <template #fields>
        <div class="app-field">
          <label class="app-label">{{ t('docker.common.name') }}</label>
          <input v-model="queryForm.name" class="app-input" :placeholder="t('docker.container.namePlaceholder')" @keyup.enter="search" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('docker.common.status') }}</label>
          <AppSelect v-model="queryForm.status" :options="statusFilterOptions" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('docker.common.image') }}</label>
          <input v-model="queryForm.image" class="app-input" :placeholder="t('docker.container.imagePlaceholder')" @keyup.enter="search" />
        </div>
      </template>
    </FilterBar>

    <div class="dk-page__body">
      <div class="dk-page__bar">
        <span class="dk-page__hint">{{ t('system.common.total', { total: pagination.total }) }}</span>
      </div>

      <!-- 桌面：表格 -->
      <DataTable
        v-if="!menuStore.isMobile"
        :columns="columns"
        :rows="list"
        row-key="id"
        seq
        :loading="loading"
        :empty-text="t('docker.container.noContainers')"
      >
        <template #cell-names="{ row }">{{ displayNames(row) }}</template>
        <template #cell-state="{ row }">
          <StatusPill :variant="stateVariant(row.state)" :label="stateLabel(String(row.state || ''))" />
        </template>
        <template #cell-status="{ row }">{{ formatContainerRuntime(row) }}</template>
        <template #cell-network="{ row }">
          <div v-if="networkDisplay(row).ips.length || networkDisplay(row).ports.length" class="dk-net">
            <span v-for="ip in networkDisplay(row).ips" :key="ip" class="dk-tag">{{ ip }}</span>
            <span v-for="port in networkDisplay(row).ports" :key="port" class="dk-tag dk-tag--port">{{ port }}</span>
          </div>
          <span v-else class="dk-dim">—</span>
        </template>
        <template #cell-operation="{ row }">
          <ActionMenu :items="rowActionsFor(row)" @select="(key) => onRowAction(key, row)" />
        </template>
      </DataTable>

      <!-- 移动：卡片 -->
      <template v-else>
        <div v-if="loading" class="dk-cards">
          <div v-for="i in 4" :key="i" class="dk-skeleton" />
        </div>
        <EmptyState
          v-else-if="!list.length"
          icon="HOutline:CubeIcon"
          :title="t('docker.container.noContainers')"
          :description="t('docker.container.emptyDesc')"
        />
        <div v-else class="dk-cards">
          <EntityCard v-for="row in list" :key="row.id">
            <template #title>{{ displayNames(row) }}</template>
            <template #status><StatusPill :variant="stateVariant(row.state)" :label="stateLabel(String(row.state || ''))" /></template>
            <template #meta>
              <span class="dk-card__full">{{ t('docker.container.imageMeta', { image: row.image }) }}</span>
              <span>{{ t('docker.container.runtimeMeta', { runtime: formatContainerRuntime(row) }) }}</span>
              <span class="dk-card__full">{{ t('docker.container.networkMeta', { network: formatNetworkText(row) }) }}</span>
            </template>
            <template #footer>
              <span />
              <ActionMenu :items="rowActionsFor(row)" @select="(key) => onRowAction(key, row)" />
            </template>
          </EntityCard>
        </div>
      </template>

      <Pagination
        v-model:page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        @change="getList"
      />
    </div>

    <!-- 创建容器 -->
    <FormDialog v-model="createVisible" :title="t('docker.container.createContainer')" :width="700" :loading="createSubmitting" @confirm="submitCreate">
      <div class="dk-form">
        <div class="dk-grid">
          <div class="app-field">
            <label class="app-label">{{ t('docker.container.namePlaceholder') }}</label>
            <input v-model="createForm.name" class="app-input" :placeholder="t('docker.common.optional')" />
          </div>
          <div class="app-field">
            <label class="app-label app-label--required">{{ t('docker.common.image') }}</label>
            <AppCombobox v-model="createForm.image" :options="imageOptions" :placeholder="t('docker.container.imageInputPlaceholder')" />
          </div>
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('docker.container.startCommand') }}</label>
          <input v-model="createForm.cmdText" class="app-input dk-mono" :placeholder="t('docker.container.commandPlaceholder')" />
        </div>
        <div class="dk-grid">
          <div class="app-field">
            <label class="app-label">{{ t('docker.common.network') }}</label>
            <AppSelect v-model="createForm.network" :options="networkSelectOptions" searchable :placeholder="t('docker.container.defaultBridge')" />
          </div>
          <div class="app-field">
            <label class="app-label">{{ t('docker.container.restartPolicy') }}</label>
            <div class="seg seg--wide">
              <button
                v-for="opt in restartSeg"
                :key="opt.value"
                type="button"
                class="seg__btn"
                :class="{ 'is-active': createForm.restartPolicy === opt.value }"
                @click="createForm.restartPolicy = opt.value"
              >{{ opt.label }}</button>
            </div>
          </div>
        </div>
        <ContainerEnvEditor v-model="createEnv" />
        <ContainerPortEditor v-model="createPorts" />
        <ContainerMountEditor v-model="createMounts" />
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('docker.common.cancel') }}</UButton>
        <UButton color="primary" :loading="createSubmitting" @click="submitCreate">{{ t('docker.common.create') }}</UButton>
      </template>
    </FormDialog>

    <!-- 容器详情 -->
    <FormDialog v-model="detailVisible" :title="t('docker.container.containerDetail')" :width="860">
      <div v-if="detail" class="dk-detail">
        <div class="dk-detail__hero">
          <div class="dk-detail__title">
            <span>{{ detail.name || '-' }}</span>
            <StatusPill :variant="stateVariant(detail.state?.status || '')" :label="stateLabel(detail.state?.status || '')" />
          </div>
          <div class="dk-detail__sub">{{ shortId(detail.id) }} · {{ detailImageName }} · {{ detailNetworkMode }}</div>
          <div v-if="detailPortTags.length" class="dk-tags dk-detail__ports">
            <span v-for="p in detailPortTags" :key="p" class="dk-tag dk-tag--port">{{ p }}</span>
          </div>
        </div>

        <div class="dk-detail__grid2">
          <div class="dk-detail__block">
            <div class="dk-detail__label">{{ t('docker.container.runtime') }}</div>
            <div class="dk-kv">
              <div><span>PID</span><strong>{{ detail.state?.pid || '-' }}</strong></div>
              <div><span>{{ t('docker.container.restartCount') }}</span><strong>{{ detail.restartCount }}</strong></div>
              <div><span>{{ t('docker.container.exitCode') }}</span><strong>{{ detail.state?.exitCode ?? '-' }}</strong></div>
              <div><span>{{ t('docker.common.platform') }}</span><strong>{{ detail.platform || '-' }}</strong></div>
              <div><span>{{ t('docker.container.created') }}</span><strong>{{ formatDateTime(detail.created) }}</strong></div>
              <div><span>{{ t('docker.container.started') }}</span><strong>{{ formatDateTime(detail.state?.startedAt) }}</strong></div>
            </div>
          </div>
          <div class="dk-detail__block">
            <div class="dk-detail__label">{{ t('docker.container.resource') }}</div>
            <div class="dk-kv">
              <div><span>{{ t('docker.container.restartPolicy') }}</span><strong>{{ detailRestartPolicy }}</strong></div>
              <div><span>{{ t('docker.container.networkMode') }}</span><strong>{{ detailNetworkMode }}</strong></div>
              <div><span>{{ t('docker.common.memory') }}</span><strong>{{ detailMemoryLimit }}</strong></div>
              <div><span>CPU</span><strong>{{ detailCpuLimit }}</strong></div>
              <div><span>{{ t('docker.container.autoRemove') }}</span><strong>{{ detailAutoRemove }}</strong></div>
              <div><span>{{ t('docker.container.privileged') }}</span><strong>{{ detailPrivileged }}</strong></div>
            </div>
          </div>
        </div>

        <div v-if="detailNetworks.length" class="dk-detail__block">
          <div class="dk-detail__label">{{ t('docker.common.network') }}</div>
          <div class="dk-list dk-list--net4">
            <div v-for="item in detailNetworks" :key="item.name" class="dk-list__row">
              <span class="dk-list__name">{{ item.name }}</span>
              <span>{{ item.ipAddress || '-' }}</span>
              <span>{{ item.gateway || '-' }}</span>
              <span>{{ item.macAddress || '-' }}</span>
            </div>
          </div>
        </div>

        <div class="dk-detail__block">
          <div class="dk-detail__label">{{ t('docker.container.startup') }}</div>
          <div class="dk-code">{{ detailCommand }}</div>
        </div>

        <div v-if="detail.mounts?.length" class="dk-detail__block">
          <div class="dk-detail__label">{{ t('docker.container.mounts') }}</div>
          <div class="dk-list dk-list--mounts">
            <div class="dk-list__head"><span>{{ t('docker.common.type') }}</span><span>{{ t('docker.common.source') }}</span><span>{{ t('docker.common.target') }}</span><span>{{ t('docker.common.mode') }}</span><span>RW</span></div>
            <div v-for="(m, i) in detail.mounts" :key="i" class="dk-list__row">
              <span>{{ m.type }}</span><span :title="m.source">{{ m.source }}</span><span :title="m.destination">{{ m.destination }}</span><span>{{ m.mode }}</span>
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

    <!-- 编辑容器 -->
    <FormDialog v-model="editVisible" :title="t('docker.container.editContainer')" :width="760" :loading="editSubmitting" @confirm="submitEdit">
      <div class="dk-form">
        <div class="dk-grid">
          <div class="app-field">
            <label class="app-label">{{ t('docker.container.namePlaceholder') }}</label>
            <input v-model="editForm.name" class="app-input" :placeholder="t('docker.container.namePlaceholder')" />
          </div>
          <div class="app-field">
            <label class="app-label">{{ t('docker.container.restartPolicy') }}</label>
            <div class="seg seg--wide">
              <button
                v-for="opt in restartSeg"
                :key="opt.value"
                type="button"
                class="seg__btn"
                :class="{ 'is-active': editForm.restartPolicy === opt.value }"
                @click="editForm.restartPolicy = opt.value"
              >{{ opt.label }}</button>
            </div>
          </div>
          <div class="app-field" />
          <div class="app-field">
            <label class="app-label">{{ t('docker.container.memoryLimitMb') }}</label>
            <div class="dk-num">
              <input v-model.number="editForm.memoryMB" type="number" min="0" step="64" class="app-input" />
              <span class="dk-num__unit">MB</span>
            </div>
            <div class="dk-chips">
              <button v-for="m in MEMORY_PRESETS" :key="m" type="button" class="dk-chip" :class="{ 'is-on': editForm.memoryMB === m }" @click="editForm.memoryMB = m">{{ m }}</button>
              <button type="button" class="dk-chip" :class="{ 'is-on': !editForm.memoryMB }" @click="editForm.memoryMB = 0">{{ t('docker.common.unlimited') }}</button>
            </div>
          </div>
          <div class="app-field">
            <label class="app-label">{{ t('docker.container.cpuLimitCores') }}</label>
            <div class="dk-num">
              <input v-model.number="editForm.nanoCpuCores" type="number" min="0" step="0.5" class="app-input" />
              <span class="dk-num__unit">{{ t('docker.common.cpuUnit') }}</span>
            </div>
            <div class="dk-chips">
              <button v-for="c in CPU_PRESETS" :key="c" type="button" class="dk-chip" :class="{ 'is-on': editForm.nanoCpuCores === c }" @click="editForm.nanoCpuCores = c">{{ c }}</button>
              <button type="button" class="dk-chip" :class="{ 'is-on': !editForm.nanoCpuCores }" @click="editForm.nanoCpuCores = 0">{{ t('docker.common.unlimited') }}</button>
            </div>
          </div>
        </div>

        <div class="app-field dk-switch">
          <label class="app-label">{{ t('docker.container.recreateTip') }}</label>
          <AppSwitch v-model="editForm.recreate" />
        </div>

        <template v-if="editForm.recreate">
          <div class="dk-grid">
            <div class="app-field dk-grid__full">
              <label class="app-label app-label--required">{{ t('docker.common.image') }}</label>
              <AppCombobox v-model="editForm.image" :options="imageOptions" :placeholder="t('docker.container.imageInputPlaceholder')" />
            </div>
            <div class="app-field dk-grid__full">
              <label class="app-label">{{ t('docker.common.network') }}</label>
              <AppSelect v-model="editForm.network" :options="networkSelectOptions" searchable :placeholder="t('docker.container.defaultBridge')" />
            </div>
          </div>

          <ContainerPortEditor v-model="editPorts" />
          <ContainerMountEditor v-model="editMounts" />
          <ContainerEnvEditor v-model="editEnv" />

          <div class="app-field dk-switch">
            <label class="app-label">{{ t('docker.container.forceDeleteOld') }}</label>
            <AppSwitch v-model="editForm.force" />
          </div>
        </template>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('docker.common.cancel') }}</UButton>
        <UButton color="primary" :loading="editSubmitting" @click="submitEdit">{{ t('docker.common.save') }}</UButton>
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
import {
  createDockerContainer, deleteDockerContainer, getDockerContainer,
  killDockerContainer, listDockerContainers, listDockerImages, listDockerNetworks,
  pauseDockerContainer, restartDockerContainer, startDockerContainer,
  stopDockerContainer, unpauseDockerContainer, updateDockerContainer,
} from '@/api/docker'
import DockerTaskDialogs from '@/views/docker/components/DockerTaskDialogs.vue'
import DockerContainerLogsDialog from '@/views/docker/components/DockerContainerLogsDialog.vue'
import DockerContainerStatsDialog from '@/views/docker/components/DockerContainerStatsDialog.vue'
import DockerContainerExecDialog from '@/views/docker/components/DockerContainerExecDialog.vue'
import { PERM } from '@/config/permission'
import { useButtonPermission } from '@/composables/useButtonPermission'
import { Dialog } from '@/utils/dialog'
import { showRequestError } from '@/utils/request'
import type { DockerContainerInfo, DockerContainerSummary, DockerPortBinding, DockerMount, DockerTaskInfo } from '@/types/v1/docker'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import ContainerPortEditor from '@/views/docker/components/ContainerPortEditor.vue'
import ContainerMountEditor from '@/views/docker/components/ContainerMountEditor.vue'
import ContainerEnvEditor from '@/views/docker/components/ContainerEnvEditor.vue'
import type { PortRow, MountRow, EnvRow } from '@/views/docker/components/specTypes'

defineOptions({ name: 'DockerContainerView' })

const menuStore = useMenuStore()
const { t } = useI18n()
const canManage = useButtonPermission([PERM.DOCKER_CONTAINER_MANAGE], [])

const list = ref<DockerContainerSummary[]>([])
const loading = ref(false)
/** docker 引擎不可用时为 true（列表/选项接口返回失败后置位），避免创建弹窗再打镜像/网络选项接口刷 toast */
const dockerUnavailable = ref(false)
const queryForm = reactive({ name: '', status: '', image: '' })
const pagination = ref({ page: 1, pageSize: 10, total: 0 })

const statusFilterOptions = computed(() => [
  { label: t('docker.common.all'), value: '' },
  { label: t('docker.states.running'), value: 'running' },
  { label: t('docker.states.exited'), value: 'exited' },
  { label: t('docker.states.paused'), value: 'paused' },
])
const restartSeg = computed(() => [
  { label: t('docker.container.restartNo'), value: '' },
  { label: t('docker.container.restartAlways'), value: 'always' },
  { label: t('docker.container.restartOnFailure'), value: 'on-failure' },
  { label: t('docker.container.restartUnlessStopped'), value: 'unless-stopped' },
])
const MEMORY_PRESETS = [512, 1024, 2048, 4096]
const CPU_PRESETS = [0.5, 1, 2, 4]

// —— 状态/运行时/网络 ——
const STATE_KEY_MAP: Record<string, string> = {
  running: 'docker.states.running', exited: 'docker.states.exited', paused: 'docker.states.paused',
  restarting: 'docker.states.restarting', created: 'docker.states.created', removing: 'docker.states.removing', dead: 'docker.states.dead',
}
type PillVariant = 'success' | 'info' | 'warning' | 'error' | 'neutral'
const STATE_VARIANT: Record<string, PillVariant> = { running: 'success', exited: 'info', paused: 'warning', restarting: 'warning', created: 'info', dead: 'error', removing: 'neutral' }
const stateLabel = (s: string) => (STATE_KEY_MAP[s] ? t(STATE_KEY_MAP[s]) : s || '-')
const stateVariant = (s: unknown) => STATE_VARIANT[String(s || '')] || 'neutral'
const displayNames = (row: Record<string, unknown>) => ((row.names as string[]) || []).map((n) => n.replace(/^\//, '')).join(', ') || '-'

const DURATION_UNIT_KEY_MAP: Record<string, string> = {
  second: 'docker.container.duration.second', seconds: 'docker.container.duration.second',
  minute: 'docker.container.duration.minute', minutes: 'docker.container.duration.minute',
  hour: 'docker.container.duration.hour', hours: 'docker.container.duration.hour',
  day: 'docker.container.duration.day', days: 'docker.container.duration.day',
  week: 'docker.container.duration.week', weeks: 'docker.container.duration.week',
  month: 'docker.container.duration.month', months: 'docker.container.duration.month',
  year: 'docker.container.duration.year', years: 'docker.container.duration.year',
}
const formatDockerDuration = (value: string) => {
  const text = value.trim()
  const lower = text.toLowerCase()
  if (lower === 'less than a second') return t('docker.container.duration.lessThanSecond')
  if (lower === 'about a minute') return t('docker.container.duration.aboutMinute')
  if (lower === 'about an hour') return t('docker.container.duration.aboutHour')
  const parts: string[] = []
  for (const match of text.matchAll(/(\d+)\s+([a-z]+)/gi)) {
    const amount = match[1]
    const unit = match[2]
    if (!amount || !unit) continue
    const unitKey = DURATION_UNIT_KEY_MAP[unit.toLowerCase()]
    parts.push(`${amount} ${unitKey ? t(unitKey) : unit}`)
  }
  return parts.length ? parts.join(' ') : text
}
const formatContainerRuntime = (row: Record<string, unknown>) => {
  if (row.state !== 'running') return '—'
  const match = String(row.status || '').match(/^Up\s+(.+?)(?:\s+\(.+\))?$/i)
  return match?.[1] ? formatDockerDuration(match[1]) : '—'
}

const uniqueItems = (items: string[]) => Array.from(new Set(items))
const formatPortItem = (port: DockerContainerSummary['ports'][number]) => {
  const protocol = (port.type || 'tcp').toLowerCase()
  const privatePort = port.privatePort || 0
  const publicPort = port.publicPort || 0
  if (!privatePort && !publicPort) return ''
  if (publicPort && privatePort) return `${port.ip ? `${port.ip}:` : ''}${publicPort}->${privatePort}/${protocol}`
  return `${privatePort || publicPort}/${protocol}`
}
const networkDisplay = (row: Record<string, unknown>) => {
  const r = row as unknown as DockerContainerSummary
  if (r.networkMode === 'host' || r.networks?.includes('host')) return { ips: ['host'], ports: [] as string[] }
  const ips = uniqueItems((r.networkEndpoints || []).map((e) => e.ipAddress).filter(Boolean))
  const ports = uniqueItems((r.ports || []).map((p) => formatPortItem(p)).filter(Boolean))
  return { ips, ports }
}
const formatNetworkText = (row: Record<string, unknown>) => {
  const n = networkDisplay(row)
  return [...n.ips, ...n.ports].join(' ') || '—'
}

const columns = computed<DataTableColumn[]>(() => [
  { key: 'names', title: t('docker.common.name'), minWidth: 150 },
  { key: 'image', title: t('docker.common.image'), minWidth: 170 },
  { key: 'state', title: t('docker.common.status'), width: 90 },
  { key: 'status', title: t('docker.container.runtime'), width: 120 },
  { key: 'network', title: t('docker.common.network'), minWidth: 240 },
  { key: 'operation', title: t('docker.common.operation'), width: 80, align: 'center' },
])

// —— 行动作（按状态） ——
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

const rowActionsFor = (row: Record<string, unknown>): ActionMenuItem[] => {
  if (!canManage.value) return [{ key: 'detail', label: t('docker.common.detail'), icon: 'HOutline:InformationCircleIcon' }]
  const items: ActionMenuItem[] = []
  const state = String(row.state || '')
  if (state === 'running') {
    items.push(
      { key: 'pause', label: t('docker.lifecycle.pause'), icon: 'HOutline:PauseIcon' },
      { key: 'stop', label: t('docker.lifecycle.stop'), icon: 'HOutline:StopIcon' },
      { key: 'restart', label: t('docker.lifecycle.restart'), icon: 'HOutline:ArrowPathIcon' },
      { key: 'logs', label: t('docker.common.logs'), icon: 'HOutline:DocumentTextIcon' },
      { key: 'stats', label: t('docker.common.stats'), icon: 'HOutline:ChartBarIcon' },
      { key: 'exec', label: t('docker.common.terminal'), icon: 'HOutline:CommandLineIcon' },
    )
  } else if (state === 'paused') {
    items.push({ key: 'unpause', label: t('docker.lifecycle.unpause'), icon: 'HOutline:PlayIcon' })
  } else if (state === 'exited' || state === 'created') {
    items.push({ key: 'start', label: t('docker.lifecycle.start'), icon: 'HOutline:PlayIcon' })
  }
  items.push(
    { key: 'edit', label: t('docker.common.edit'), icon: 'HOutline:PencilSquareIcon' },
    { key: 'detail', label: t('docker.common.detail'), icon: 'HOutline:InformationCircleIcon' },
    { key: 'delete', label: t('docker.common.delete'), icon: 'HOutline:TrashIcon', danger: true },
  )
  return items
}

const findRow = (id: string) => list.value.find((x) => x.id === id)
const onRowAction = (key: string, row: Record<string, unknown>) => {
  const record = findRow(String(row.id))
  if (!record) return
  if (key in actionMap) { void handleAction(record, key as LifecycleAction); return }
  if (key === 'logs') void openLogs(record)
  else if (key === 'stats') openStats(record)
  else if (key === 'exec') void openExec(record)
  else if (key === 'edit') void openEdit(record)
  else if (key === 'detail') void openDetail(record)
}

const handleAction = async (row: DockerContainerSummary, action: LifecycleAction) => {
  const def = actionMap[action]
  const name = displayNames(row as unknown as Record<string, unknown>)
  const actionLabel = t(def.labelKey)
  const run = async () => {
    try {
      await def.fn(row.id)
      ElMessage.success(t('docker.container.actionSuccess', { action: actionLabel }))
      await getList()
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

const getList = async () => {
  loading.value = true
  try {
    const { data } = await listDockerContainers({
      all: true, page: pagination.value.page, pageSize: pagination.value.pageSize,
      status: queryForm.status || '', name: queryForm.name || '', image: queryForm.image || '', network: '', labels: {},
    })
    list.value = data?.items || []
    pagination.value.total = Number(data?.total || 0)
    dockerUnavailable.value = false
  } catch {
    list.value = []
    pagination.value.total = 0
    dockerUnavailable.value = true
  } finally { loading.value = false }
}
const search = () => { pagination.value.page = 1; getList() }
const reset = () => { queryForm.name = ''; queryForm.status = ''; queryForm.image = ''; search() }

// —— 任务 ——
const taskDialogsVisible = ref(false)
const activeTask = ref<DockerTaskInfo | null>(null)
const openTasks = () => { taskDialogsVisible.value = true }
const openTask = (task: DockerTaskInfo | undefined) => {
  if (!task?.id) return
  activeTask.value = task
  taskDialogsVisible.value = true
}
const handleTaskFinished = async () => { await getList() }

// —— 镜像/网络 选项 ——
const imageOptions = ref<string[]>([])
const networkOptions = ref<string[]>([])
const networkSelectOptions = computed(() => [
  { label: t('docker.container.defaultBridge'), value: '' },
  ...networkOptions.value.map((n) => ({ label: n, value: n })),
])
const loadImageOptions = async () => {
  if (dockerUnavailable.value || imageOptions.value.length) return
  try {
    const { data } = await listDockerImages({ all: true, keyword: '', labels: {}, page: 1, pageSize: 200 })
    imageOptions.value = Array.from(new Set((data?.items || []).flatMap((i) => i.repoTags || []).filter((i) => i && i !== '<none>:<none>')))
  } catch { /* interceptor 已 toast；选项为空时允许手输镜像 */ }
}
const loadNetworkOptions = async () => {
  if (dockerUnavailable.value || networkOptions.value.length) return
  try {
    const { data } = await listDockerNetworks({ name: '', driver: '', scope: '', labels: {}, page: 1, pageSize: 200 })
    networkOptions.value = Array.from(new Set((data?.items || []).map((i) => i.name).filter(Boolean)))
  } catch { /* ignore */ }
}

// —— 创建 ——
const createVisible = ref(false)
const createSubmitting = ref(false)
const createForm = reactive({ name: '', image: '', cmdText: '', network: '', restartPolicy: '' })
const createEnv = ref<EnvRow[]>([])
const createPorts = ref<PortRow[]>([])
const createMounts = ref<MountRow[]>([])
const openCreate = () => {
  Object.assign(createForm, { name: '', image: '', cmdText: '', network: '', restartPolicy: '' })
  createEnv.value = []
  createPorts.value = []
  createMounts.value = []
  createVisible.value = true
  void loadImageOptions()
  void loadNetworkOptions()
}
const buildPorts = (rows: PortRow[]): DockerPortBinding[] =>
  rows
    .map((item) => ({
      hostIp: item.hostIp.trim(),
      hostPort: item.hostPort.trim(),
      containerPort: item.containerPort.trim()
        ? `${item.containerPort.trim()}/${item.protocol || 'tcp'}`
        : '',
    }))
    .filter((item) => item.containerPort)
const buildMounts = (rows: MountRow[]): DockerMount[] =>
  rows
    .map((item) => ({
      type: item.type || 'bind',
      source: item.source.trim(),
      target: item.target.trim(),
      readOnly: item.readOnly,
    }))
    .filter((item) => item.target)
const buildEnv = (rows: EnvRow[]) =>
  rows.filter((item) => item.key.trim()).map((item) => `${item.key.trim()}=${item.value}`)
const submitCreate = async () => {
  const image = createForm.image.trim()
  if (!image) { ElMessage.error(t('docker.container.enterImage')); return }
  createSubmitting.value = true
  try {
    await createDockerContainer({
      options: {
        name: (createForm.name.trim() || undefined) as unknown as string,
        image,
        cmd: createForm.cmdText.trim() ? createForm.cmdText.trim().split(/\s+/) : [],
        env: buildEnv(createEnv.value),
        ports: buildPorts(createPorts.value),
        mounts: buildMounts(createMounts.value),
        network: (createForm.network.trim() || undefined) as unknown as string,
        restartPolicy: (createForm.restartPolicy || undefined) as unknown as string,
        hostname: '', user: '', entrypoint: [], workingDir: '', labels: {},
        tty: false, openStdin: false, autoRemove: false, privileged: false,
        memory: 0, memorySwap: 0, cpuShares: 0, cpuQuota: 0, cpuPeriod: 0, nanoCpus: 0, platform: '',
      },
    })
    ElMessage.success(t('docker.container.createSuccess'))
    createVisible.value = false
    await getList()
  } catch (e) { showRequestError(e, t('docker.container.createFailed')) }
  finally { createSubmitting.value = false }
}

// —— 详情 ——
const detailVisible = ref(false)
const detail = ref<DockerContainerInfo | null>(null)
const shortId = (id?: unknown) => { const s = String(id || ''); return s ? s.slice(0, 12) : '-' }
const formatDateTime = (value?: unknown) => {
  const s = value ? String(value) : ''
  if (!s || s.startsWith('0001-')) return '-'
  const date = new Date(s)
  if (Number.isNaN(date.getTime())) return s
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}
const formatBytes = (bytes?: number | string) => {
  const n = Number(bytes)
  if (!n) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = n
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
  return `${v.toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}
const detailCommand = computed(() => {
  const item = detail.value
  if (!item) return '-'
  return [item.path, ...(item.args || [])].filter(Boolean).join(' ') || '-'
})
const detailHostConfig = computed(() => detail.value?.hostConfig)
const detailImageName = computed(() => detail.value?.config?.image || detail.value?.image || '-')
const detailRestartPolicy = computed(() => detailHostConfig.value?.restartPolicy || '-')
const detailNetworkMode = computed(() => detailHostConfig.value?.networkMode || '-')
const detailAutoRemove = computed(() => (detailHostConfig.value?.autoRemove ? t('docker.common.yes') : t('docker.common.no')))
const detailPrivileged = computed(() => (detailHostConfig.value?.privileged ? t('docker.common.yes') : t('docker.common.no')))
const detailMemoryLimit = computed(() => {
  const memory = detailHostConfig.value?.memory || 0
  return memory ? formatBytes(memory) : t('docker.common.unlimited')
})
const detailCpuLimit = computed(() => {
  const nanoCpus = detailHostConfig.value?.nanoCpus || 0
  if (nanoCpus) return `${(nanoCpus / 1e9).toFixed(2)} ${t('docker.common.cpuUnit')}`
  const cpuQuota = detailHostConfig.value?.cpuQuota || 0
  const cpuPeriod = detailHostConfig.value?.cpuPeriod || 0
  if (cpuQuota && cpuPeriod) return `${(cpuQuota / cpuPeriod).toFixed(2)} ${t('docker.common.cpuUnit')}`
  return t('docker.common.unlimited')
})
const detailNetworks = computed(() => {
  const networks = detail.value?.network?.networks || {}
  return Object.entries(networks).map(([name, item]) => ({
    name, ipAddress: item?.ipAddress || '', gateway: item?.gateway || '', macAddress: item?.macAddress || '',
  }))
})
const detailPortTags = computed(() => {
  const bindings = detailHostConfig.value?.portBindings || []
  return bindings.map((b) => {
    const containerPort = b.containerPort || ''
    const hostIp = b.hostIp || ''
    const hostPort = b.hostPort || ''
    return hostPort ? `${hostIp ? `${hostIp}:` : ''}${hostPort}->${containerPort}` : containerPort
  }).filter(Boolean)
})
const openDetail = async (row: DockerContainerSummary) => {
  detailVisible.value = true
  detail.value = null
  try { const { data } = await getDockerContainer({ id: row.id }); detail.value = data?.info || null }
  catch (e) { showRequestError(e, t('docker.container.getDetailFailed')) }
}
const openDetailLogs = () => {
  if (!detail.value?.logsWsPath) { ElMessage.warning(t('docker.container.missingLogsWsPath')); return }
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
  if (!detail.value?.execWsPath) { ElMessage.warning(t('docker.container.missingExecWsPath')); return }
  execContainerId.value = detail.value.id
  execContainerName.value = detail.value.name || detail.value.id
  execWsPath.value = detail.value.execWsPath
  execVisible.value = true
}

// —— 日志 / 统计 / 终端 ——
const logsVisible = ref(false)
const logsContainerId = ref('')
const logsContainerName = ref('')
const logsWsPath = ref('')
const openLogs = async (row: DockerContainerSummary) => {
  try {
    const { data } = await getDockerContainer({ id: row.id })
    const info = data?.info
    if (!info?.logsWsPath) { ElMessage.warning(t('docker.container.missingLogsWsPath')); return }
    logsContainerId.value = info.id || row.id
    logsContainerName.value = info.name || displayNames(row as unknown as Record<string, unknown>)
    logsWsPath.value = info.logsWsPath
    logsVisible.value = true
  } catch (e) { showRequestError(e, t('docker.container.getDetailFailed')) }
}
const statsVisible = ref(false)
const statsContainerId = ref('')
const statsContainerName = ref('')
const openStats = (row: DockerContainerSummary) => {
  statsContainerId.value = row.id
  statsContainerName.value = displayNames(row as unknown as Record<string, unknown>)
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
    if (!info?.execWsPath) { ElMessage.warning(t('docker.container.missingExecWsPath')); return }
    execContainerId.value = info.id || row.id
    execContainerName.value = info.name || displayNames(row as unknown as Record<string, unknown>)
    execWsPath.value = info.execWsPath
    execVisible.value = true
  } catch (e) { showRequestError(e, t('docker.container.getDetailFailed')) }
}

// —— 编辑 ——
const editVisible = ref(false)
const editSubmitting = ref(false)
const editId = ref('')
type LooseRecord = Record<string, unknown>
const editForm = reactive({ name: '', restartPolicy: '', memoryMB: 0, nanoCpuCores: 0, recreate: false, image: '', network: '', force: false })
const editPorts = ref<PortRow[]>([])
const editMounts = ref<MountRow[]>([])
const editEnv = ref<EnvRow[]>([])

const asRecord = (value: unknown): LooseRecord => (value && typeof value === 'object' && !Array.isArray(value) ? value as LooseRecord : {})
const readString = (source: LooseRecord, ...keys: string[]) => {
  for (const key of keys) {
    const value = source[key]
    if (value !== undefined && value !== null && value !== '') return String(value)
  }
  return ''
}
const resetEditForm = () => {
  Object.assign(editForm, { name: '', restartPolicy: '', memoryMB: 0, nanoCpuCores: 0, recreate: false, image: '', network: '', force: false })
  editPorts.value = []
  editMounts.value = []
  editEnv.value = []
}
const splitPortProtocol = (value: string): Pick<PortRow, 'containerPort' | 'protocol'> => {
  const [port, protocol] = value.trim().split('/')
  return { containerPort: port || '', protocol: protocol?.toLowerCase() === 'udp' ? 'udp' : 'tcp' }
}
const fillEditPorts = (items: unknown) => {
  if (!Array.isArray(items)) return
  editPorts.value = items.map((raw) => {
    const item = asRecord(raw)
    const port = splitPortProtocol(readString(item, 'ContainerPort', 'containerPort'))
    return { hostIp: readString(item, 'HostIP', 'HostIp', 'hostIp', 'hostIP'), hostPort: readString(item, 'HostPort', 'hostPort'), containerPort: port.containerPort, protocol: port.protocol }
  }).filter((item) => item.containerPort)
}
const fillEditMounts = (hostMounts: unknown, points: DockerContainerInfo['mounts']) => {
  if (Array.isArray(hostMounts) && hostMounts.length) {
    editMounts.value = hostMounts.map((raw) => {
      const item = asRecord(raw)
      return { type: readString(item, 'Type', 'type') || 'bind', source: readString(item, 'Source', 'source'), target: readString(item, 'Target', 'target'), readOnly: !!(item.ReadOnly ?? item.readOnly) }
    }).filter((item) => item.target)
    return
  }
  editMounts.value = (points || []).map((item) => ({ type: item.type || 'bind', source: item.source || item.name || '', target: item.destination || '', readOnly: !item.rw })).filter((item) => item.target)
}
const fillEditEnv = (items: unknown) => {
  if (!Array.isArray(items)) return
  editEnv.value = items.map((raw) => {
    const text = String(raw)
    const index = text.indexOf('=')
    return index >= 0 ? { key: text.slice(0, index), value: text.slice(index + 1) } : { key: text, value: '' }
  }).filter((item) => item.key)
}

const openEdit = async (row: DockerContainerSummary) => {
  editId.value = row.id
  resetEditForm()
  editForm.name = displayNames(row as unknown as Record<string, unknown>)
  editForm.image = row.image
  editVisible.value = true
  void loadImageOptions()
  void loadNetworkOptions()
  try {
    const { data } = await getDockerContainer({ id: row.id })
    const info = data?.info
    if (info) {
      editForm.name = info.name || editForm.name
      const config = info.config
      const hostConfig = info.hostConfig
      editForm.image = config?.image || info.image || editForm.image
      editForm.restartPolicy = hostConfig?.restartPolicy || ''
      editForm.network = hostConfig?.networkMode || ''
      const memory = hostConfig?.memory || 0
      const nanoCpus = hostConfig?.nanoCpus || 0
      if (memory) editForm.memoryMB = Math.round(memory / 1024 / 1024)
      if (nanoCpus) editForm.nanoCpuCores = Math.round((nanoCpus / 1e9) * 10) / 10
      fillEditPorts(hostConfig?.portBindings)
      fillEditMounts(hostConfig?.mounts, info.mounts)
      fillEditEnv(config?.env)
    }
  } catch { /* use defaults */ }
}
const submitEdit = async () => {
  editSubmitting.value = true
  try {
    if (editForm.recreate) {
      if (!editForm.image.trim()) { ElMessage.error(t('docker.container.enterImage')); editSubmitting.value = false; return }
      const { data } = await updateDockerContainer({
        id: editId.value, name: editForm.name.trim(), restartPolicy: editForm.restartPolicy,
        memory: editForm.memoryMB * 1024 * 1024, memorySwap: 0, cpuShares: 0, cpuQuota: 0, cpuPeriod: 0,
        nanoCpus: Math.round(editForm.nanoCpuCores * 1e9), recreate: true, force: editForm.force, removeVolumes: false,
        options: {
          name: (editForm.name.trim() || undefined) as unknown as string, image: editForm.image.trim(),
          cmd: [], env: buildEnv(editEnv.value), ports: buildPorts(editPorts.value), mounts: buildMounts(editMounts.value),
          network: (editForm.network.trim() || undefined) as unknown as string,
          restartPolicy: (editForm.restartPolicy || undefined) as unknown as string,
          hostname: '', user: '', entrypoint: [], workingDir: '', labels: {},
          tty: false, openStdin: false, autoRemove: false, privileged: false,
          memory: editForm.memoryMB * 1024 * 1024, memorySwap: 0, cpuShares: 0, cpuQuota: 0, cpuPeriod: 0,
          nanoCpus: Math.round(editForm.nanoCpuCores * 1e9), platform: '',
        },
      })
      openTask(data?.task)
    } else {
      await updateDockerContainer({
        id: editId.value, name: editForm.name.trim(), restartPolicy: editForm.restartPolicy,
        memory: editForm.memoryMB * 1024 * 1024, memorySwap: 0, cpuShares: 0, cpuQuota: 0, cpuPeriod: 0,
        nanoCpus: Math.round(editForm.nanoCpuCores * 1e9), recreate: false, force: false, removeVolumes: false, options: undefined,
      })
    }
    ElMessage.success(editForm.recreate ? t('docker.container.recreateTaskCreated') : t('docker.container.updateSuccess'))
    editVisible.value = false
    if (!editForm.recreate) await getList()
  } catch (e) { showRequestError(e, t('docker.container.updateFailed')) }
  finally { editSubmitting.value = false }
}

onMounted(() => { getList() })
</script>

<style scoped lang="scss">
.dk-page { display: flex; flex-direction: column; gap: 12px; }
.dk-page__body { display: flex; flex-direction: column; gap: 10px; }
.dk-page__bar { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.dk-page__hint { font-size: 0.8125rem; color: var(--el-text-color-secondary); font-variant-numeric: tabular-nums; }
.dk-dim { color: var(--el-text-color-placeholder); }
.dk-net { display: flex; flex-wrap: wrap; gap: 4px; }
.dk-tags { display: flex; flex-wrap: wrap; align-items: center; gap: 4px; }
.dk-tag {
  display: inline-block; max-width: 240px; padding: 1px 8px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  border-radius: var(--app-radius-xs); background: var(--el-fill-color-light); color: var(--el-text-color-regular); font-size: 0.72rem;
}
.dk-tag--port { background: color-mix(in srgb, var(--el-color-primary) 12%, transparent); color: var(--el-color-primary); }

/* 表单 */
.dk-form { display: flex; flex-direction: column; gap: 14px; }
.dk-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
.dk-grid__full { grid-column: 1 / -1; }
.dk-switch { flex-direction: row; align-items: center; justify-content: space-between; gap: 12px; }
.dk-mono { font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', Consolas, monospace; font-size: 0.8125rem; }

/* 分段 / 数字 / 预设 chips */
.seg {
  display: inline-flex;
  padding: 2px;
  gap: 2px;
  background: var(--el-fill-color-light);
  border-radius: var(--app-radius);
  max-width: 100%;
  flex-wrap: wrap;
}
.seg--wide { width: 100%; }
.seg__btn {
  flex: 1;
  min-width: 0;
  padding: 5px 8px;
  border: none;
  background: transparent;
  border-radius: var(--app-radius-sm);
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: background 0.15s, color 0.15s, box-shadow 0.15s;
}
.seg__btn.is-active {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}
.dk-num {
  display: flex;
  align-items: center;
  gap: 6px;
}
.dk-num .app-input { flex: 1; min-width: 0; }
.dk-num__unit {
  flex-shrink: 0;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.dk-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 6px;
}
.dk-chip {
  padding: 2px 8px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 999px;
  background: var(--el-fill-color-blank, var(--el-bg-color));
  color: var(--el-text-color-secondary);
  font-size: 0.72rem;
  cursor: pointer;
  transition: border-color 0.15s, color 0.15s, background 0.15s;
}
.dk-chip.is-on {
  border-color: color-mix(in srgb, var(--el-color-primary) 45%, transparent);
  background: color-mix(in srgb, var(--el-color-primary) 12%, transparent);
  color: var(--el-color-primary);
}

/* 详情 */
.dk-detail { display: flex; flex-direction: column; gap: 14px; }
.dk-detail__hero { padding: 12px; border: 1px solid var(--el-border-color-lighter); border-radius: var(--app-radius); background: var(--el-fill-color-lighter); }
.dk-detail__title { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; font-weight: 700; color: var(--el-text-color-primary); }
.dk-detail__sub { margin-top: 4px; font-size: 0.78rem; color: var(--el-text-color-secondary); word-break: break-all; }
.dk-detail__ports { margin-top: 8px; }
.dk-detail__grid2 { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
.dk-detail__label { margin-bottom: 6px; font-size: 0.8125rem; font-weight: 600; color: var(--el-text-color-primary); }
.dk-kv { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); border: 1px solid var(--el-border-color-lighter); border-radius: var(--app-radius); overflow: hidden; }
.dk-kv > div { padding: 8px 10px; border-right: 1px solid var(--el-border-color-lighter); border-bottom: 1px solid var(--el-border-color-lighter); }
.dk-kv > div:nth-child(3n) { border-right: 0; }
.dk-kv > div:nth-last-child(-n + 3) { border-bottom: 0; }
.dk-kv span { display: block; color: var(--el-text-color-secondary); font-size: 0.72rem; }
.dk-kv strong { display: block; margin-top: 2px; color: var(--el-text-color-primary); font-size: 0.8rem; font-weight: 600; word-break: break-word; }
.dk-code { padding: 8px 10px; border: 1px solid var(--el-border-color-lighter); border-radius: var(--app-radius); background: var(--el-fill-color-lighter); color: var(--el-text-color-regular); font-size: 0.75rem; line-height: 1.6; word-break: break-all; }
.dk-list { border: 1px solid var(--el-border-color-lighter); border-radius: var(--app-radius); overflow: hidden; font-size: 0.78rem; }
.dk-list__head, .dk-list__row { display: grid; gap: 8px; align-items: center; padding: 7px 10px; }
.dk-list__head { color: var(--el-text-color-secondary); font-size: 0.72rem; background: var(--el-fill-color-lighter); }
.dk-list__row { border-top: 1px solid var(--el-border-color-lighter); color: var(--el-text-color-regular); }
.dk-list--net4 .dk-list__row { grid-template-columns: 1fr 1fr 1fr 1fr; }
.dk-list--mounts .dk-list__head, .dk-list--mounts .dk-list__row { grid-template-columns: 0.7fr 1.6fr 1.4fr 0.7fr 0.6fr; }
.dk-list__name { font-weight: 600; color: var(--el-text-color-primary); }
.dk-list span { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dk-detail__loading { padding: 40px 0; text-align: center; color: var(--el-text-color-secondary); }

/* 移动卡片 */
.dk-cards { display: flex; flex-direction: column; gap: 10px; }
.dk-card__full { flex-basis: 100%; }
.dk-skeleton {
  height: 132px; border-radius: var(--app-radius);
  background: linear-gradient(100deg, var(--el-fill-color-light) 30%, var(--el-fill-color) 50%, var(--el-fill-color-light) 70%);
  background-size: 200% 100%; animation: dk-shimmer 1.4s ease-in-out infinite;
}
@keyframes dk-shimmer { from { background-position: 200% 0; } to { background-position: -200% 0; } }
@media (width <= 768px) {
  .dk-grid, .dk-detail__grid2 { grid-template-columns: 1fr; }
  .dk-grid__full { grid-column: 1; }
}
</style>
