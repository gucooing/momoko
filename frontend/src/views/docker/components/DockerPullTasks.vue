<!-- 进行中的 Docker 后台任务卡（镜像页：拉取；容器页：创建 / 重建）。
     头部：贴纸 + 「正在拉取 / 正在创建」+ 目标 + 进行中 pill + 来源与发起时间；右侧大号百分比、分层展开、查看日志、取消（可选）。
     进度条下：已下载 / 总量 · 速度 · 剩余时间 · N 层 · N 层完成 · N 层下载中；展开后逐层显示进度（PC 两列，手机一列）。 -->
<template>
  <section
    v-for="task in running"
    :key="task.id"
    class="pull"
    :aria-label="headOf(task)"
  >
    <div class="pull__head">
      <span class="pull__sticker m-sticker" :class="isPull(task) ? 'm-sticker--a2' : 'm-sticker--p'">
        <UIcon :name="isPull(task) ? 'i-lucide-download' : 'i-lucide-box'" />
      </span>
      <span class="pull__text">
        <span class="pull__title">
          <span class="pull__label">{{ headOf(task) }}</span>
          <span class="pull__ref m-mono">{{ targetOf(task) }}</span>
          <StatusPill v-if="!menuStore.isMobile" variant="warning" :label="statusText(task)" />
        </span>
        <span class="pull__sub">{{ subOf(task) }}</span>
      </span>
      <span v-if="task.pull" class="pull__pct m-num">{{ task.pull.percent }}<small>%</small></span>
      <AppIconButton
        v-if="task.pull?.layers?.length && !menuStore.isMobile"
        :icon="isOpen(task.id) ? 'HOutline:ChevronUpIcon' : 'HOutline:ChevronDownIcon'"
        :label="isOpen(task.id) ? t('docker.pullCard.hideLayers') : t('docker.pullCard.showLayers')"
        :aria-expanded="isOpen(task.id)"
        @click="toggle(task.id)"
      />
      <AppIconButton icon="HOutline:DocumentTextIcon" :label="t('docker.image.viewPullLog')" @click="emit('open', task)" />
      <AppIconButton
        v-if="canCancel"
        class="pull__cancel"
        icon="HOutline:XMarkIcon"
        :label="t('docker.pullCard.cancel')"
        :disabled="canceling === task.id"
        @click="cancel(task)"
      />
    </div>

    <div class="pull__progress">
      <div class="pull__bar" :class="{ 'is-indeterminate': !task.pull }" role="progressbar" :aria-valuenow="task.pull?.percent || 0" aria-valuemin="0" aria-valuemax="100">
        <span :style="task.pull ? { width: `${task.pull.percent}%` } : undefined" />
      </div>
      <div class="pull__stats m-num">
        <template v-if="task.pull && Number(task.pull.total)">
          <span><strong>{{ formatBytes(task.pull.current) }}</strong> / {{ formatBytes(task.pull.total) }}</span>
          <span v-if="Number(task.pull.bytesPerSecond)">{{ formatRate(task.pull.bytesPerSecond) }}</span>
          <span v-if="etaOf(task)">{{ t('docker.pullCard.eta', { d: etaOf(task) }) }}</span>
        </template>
        <span v-else>{{ task.message?.trim().split('\n').pop() || t('docker.image.pullWaiting') }}</span>
        <span class="pull__grow" />
        <span v-if="task.pull?.layersTotal && !menuStore.isMobile">{{ layersText(task) }}</span>
      </div>
    </div>

    <button
      v-if="task.pull?.layers?.length && menuStore.isMobile"
      type="button"
      class="pull__mtoggle"
      :aria-expanded="isOpen(task.id)"
      @click="toggle(task.id)"
    >
      <span>{{ layersText(task) }}</span>
      <UIcon :name="isOpen(task.id) ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'" />
    </button>

    <div v-if="isOpen(task.id) && task.pull?.layers?.length" class="pull__layers">
      <div v-for="l in task.pull.layers" :key="l.id" class="pull__ly">
        <span class="pull__lyid m-mono">{{ l.id }}</span>
        <span class="pull__lybar"><span :class="`is-${tone(l.status)}`" :style="{ width: `${layerPct(l)}%` }" /></span>
        <span class="pull__lyst" :class="`is-${tone(l.status)}`">{{ layerStatus(l) }}</span>
        <span v-if="!menuStore.isMobile" class="pull__lysz m-num">{{ layerSize(l) }}</span>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { cancelDockerTask } from '@/api/docker'
import { showRequestError } from '@/utils/request'
import { DockerTaskStatus, DockerTaskType } from '@/types/v1/docker'
import type { DockerPullLayer, DockerTaskInfo } from '@/types/v1/docker'
import { formatAgo, formatBytes, formatDuration, formatRate } from './format'
import { useDockerTasks } from './useDockerTasks'

const props = withDefaults(
  defineProps<{
    /** 展示哪些类型的进行中任务 */
    types?: DockerTaskType[]
    /** 是否提供取消按钮（需镜像管理权限） */
    canCancel?: boolean
  }>(),
  { types: () => [DockerTaskType.DOCKER_TASK_TYPE_IMAGE_PULL], canCancel: false },
)
const emit = defineEmits<{ open: [task: DockerTaskInfo]; finished: [] }>()
const { t } = useI18n()
const menuStore = useMenuStore()
const { active, finishedTick, refresh } = useDockerTasks()

const running = computed(() => active.value.filter((task) => props.types.includes(task.type)))
watch(finishedTick, () => emit('finished'))

const openIds = ref(new Set<string>())
const isOpen = (id: string) => openIds.value.has(id)
const toggle = (id: string) => {
  const next = new Set(openIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  openIds.value = next
}

const isPull = (task: DockerTaskInfo) => task.type === DockerTaskType.DOCKER_TASK_TYPE_IMAGE_PULL
const headOf = (task: DockerTaskInfo) => {
  if (isPull(task)) return t('docker.image.pulling')
  if (task.type === DockerTaskType.DOCKER_TASK_TYPE_CONTAINER_RECREATE) return t('docker.pullCard.recreating')
  return task.pull && task.pull.percent < 100 ? t('docker.pullCard.creatingPull') : t('docker.pullCard.creating')
}
const targetOf = (task: DockerTaskInfo) => task.target || task.title.replace(/^\S+\s*/, '') || task.id.slice(0, 8)
const statusText = (task: DockerTaskInfo) =>
  task.status === DockerTaskStatus.DOCKER_TASK_STATUS_PENDING ? t('docker.task.status.pending') : t('docker.image.inProgress')

/** 镜像所在仓库：第一段带 . 或 : 或为 localhost 时是仓库主机，否则是 Docker Hub */
const registryOf = (ref: string) => {
  const first = ref.split('/')[0] || ''
  return ref.includes('/') && (first.includes('.') || first.includes(':') || first === 'localhost') ? first : 'Docker Hub'
}
const subOf = (task: DockerTaskInfo) => {
  const started = formatAgo(task.startTime, t)
  if (isPull(task)) return t('docker.pullCard.fromAgo', { registry: registryOf(targetOf(task)), ago: started })
  return t('docker.pullCard.startedAgo', { ago: started })
}
const etaOf = (task: DockerTaskInfo) => {
  const p = task.pull
  const bps = Number(p?.bytesPerSecond || 0)
  const left = Number(p?.total || 0) - Number(p?.current || 0)
  if (!bps || left <= 0) return ''
  return formatDuration((left / bps) * 1000, t)
}
const layersText = (task: DockerTaskInfo) =>
  t('docker.pullCard.layers', {
    total: task.pull?.layersTotal || 0,
    done: task.pull?.layersDone || 0,
    dl: task.pull?.layersDownloading || 0,
  })

const tone = (status: string) => (status === 'done' || status === 'exists' ? 'ok' : status === 'waiting' ? 'wait' : 'run')
const layerPct = (l: DockerPullLayer) => {
  if (l.status === 'done' || l.status === 'exists' || l.status === 'extracting') return 100
  const total = Number(l.total)
  return total ? Math.min(100, Math.round((Number(l.current) / total) * 100)) : 0
}
const layerStatus = (l: DockerPullLayer) => {
  if (l.status === 'downloading') return t('docker.pullCard.lyDownloading', { pct: layerPct(l) })
  return t(`docker.pullCard.ly.${l.status || 'waiting'}`)
}
const layerSize = (l: DockerPullLayer) => {
  const total = Number(l.total)
  if (!total) return '—'
  if (l.status === 'downloading') return `${formatBytes(l.current)} / ${formatBytes(total)}`
  return formatBytes(total)
}

const canceling = ref('')
const cancel = async (task: DockerTaskInfo) => {
  canceling.value = task.id
  try {
    await cancelDockerTask({ id: task.id })
    feedback.success(t('docker.pullCard.canceled'))
    await refresh()
  } catch (e) {
    showRequestError(e, t('docker.pullCard.cancelFailed'))
  } finally {
    canceling.value = ''
  }
}

defineExpose({ refresh })
</script>

<style scoped lang="scss">
.pull {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 20px 16px 20px 22px;
  border-radius: var(--app-radius-xl);
  border: 1.5px solid var(--m-a2);
  background: var(--el-bg-color);
  box-shadow: 0 0 0 3px var(--m-a2Soft);
  min-width: 0;
}
.pull__head {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.pull__sticker {
  flex: none;
  width: 48px;
  height: 48px;
  border-radius: 16px;
  transform: rotate(-5deg);
}
.pull__sticker :deep(svg),
.pull__sticker :deep(.iconify) {
  width: 24px;
  height: 24px;
}
.pull__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.pull__title {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  min-width: 0;
}
.pull__label {
  font-family: var(--app-font-display);
  font-size: 1.0625rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.pull__ref {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.9375rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.pull__sub {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.pull__pct {
  flex: none;
  font-family: var(--app-font-display);
  font-size: 2.125rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: var(--el-text-color-primary);
}
.pull__pct small {
  font-size: 1.125rem;
}
.pull__cancel {
  color: var(--m-bad);
}
.pull__cancel:hover {
  background: var(--m-badSoft);
}
.pull__progress {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-right: 6px;
}
.pull__bar {
  height: 10px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  overflow: hidden;
}
.pull__bar span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--m-a2);
  transition: width 0.4s ease;
}
.pull__bar.is-indeterminate span {
  width: 35%;
  animation: pull-slide 1.4s ease-in-out infinite;
}
@keyframes pull-slide {
  from { transform: translateX(-100%); }
  to { transform: translateX(300%); }
}
.pull__stats {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 18px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  min-width: 0;
}
.pull__stats strong {
  color: var(--el-text-color-primary);
}
.pull__grow {
  flex: 1;
}
.pull__mtoggle {
  display: flex;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  margin-right: 6px;
  padding: 0 12px;
  border: none;
  border-radius: 12px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
}
.pull__mtoggle span {
  flex: 1;
  text-align: left;
}
.pull__mtoggle :deep(.iconify),
.pull__mtoggle :deep(svg) {
  width: 16px;
  height: 16px;
}
.pull__layers {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  padding-right: 6px;
}
.pull__ly {
  display: grid;
  grid-template-columns: 110px minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 10px 12px;
  border-radius: 14px;
  background: var(--el-fill-color-light);
}
.pull__lyid {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
}
.pull__lybar {
  height: 6px;
  border-radius: 999px;
  background: var(--el-bg-color);
  overflow: hidden;
}
.pull__lybar span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--m-a2);
  transition: width 0.4s ease;
}
.pull__lybar span.is-ok {
  background: var(--m-ok);
}
.pull__lyst {
  font-size: 0.75rem;
  font-weight: 700;
  white-space: nowrap;
  color: var(--el-text-color-primary);
}
.pull__lyst.is-ok {
  color: var(--m-ok);
}
.pull__lyst.is-wait {
  color: var(--el-text-color-secondary);
}
.pull__lysz {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  text-align: right;
  white-space: nowrap;
}
@media (width < 768px) {
  .pull {
    gap: 10px;
    padding: 14px 6px 12px 14px;
  }
  .pull__head {
    gap: 10px;
  }
  .pull__sticker {
    width: 40px;
    height: 40px;
    border-radius: 13px;
  }
  .pull__label {
    font-size: 0.75rem;
    font-weight: 700;
    font-family: inherit;
    color: var(--el-text-color-secondary);
  }
  .pull__title {
    flex-direction: column;
    align-items: flex-start;
    gap: 1px;
  }
  .pull__ref {
    max-width: 100%;
    font-size: 0.875rem;
  }
  .pull__sub {
    display: none;
  }
  .pull__pct {
    font-size: 1.625rem;
  }
  .pull__pct small {
    font-size: 0.875rem;
  }
  .pull__bar {
    height: 8px;
  }
  .pull__stats {
    gap: 4px 10px;
    font-size: 0.75rem;
  }
  .pull__layers {
    grid-template-columns: minmax(0, 1fr);
  }
  .pull__ly {
    grid-template-columns: 96px minmax(0, 1fr) auto;
    padding: 0;
    background: transparent;
  }
  .pull__lyid {
    font-size: 0.6875rem;
  }
  .pull__lyst {
    font-size: 0.6875rem;
  }
}
</style>
