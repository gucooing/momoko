<!-- 镜像页顶部「正在拉取」卡：列出进行中的拉取任务，解析任务 Progress（如 "68% (32.1MB/47.2MB) 2/6"）画进度条。
     有进行中任务时每 2.5 秒轮询一次；任务结束后通知父级刷新列表。点「查看日志」交给父级打开任务日志。 -->
<template>
  <section v-for="task in running" :key="task.id" class="pull" :aria-label="t('docker.image.pulling')">
    <div class="pull__head">
      <span class="pull__sticker m-sticker m-sticker--a2">
        <component :is="menuStore.iconComponents['HOutline:ArrowDownTrayIcon']" />
      </span>
      <span class="pull__text">
        <span class="pull__title">
          <span class="pull__label">{{ t('docker.image.pulling') }}</span>
          <span class="pull__ref m-mono">{{ refOf(task) }}</span>
          <StatusPill variant="warning" :label="statusText(task)" />
        </span>
        <span class="pull__sub m-num">
          <template v-if="parse(task).total">
            <strong>{{ parse(task).current }}</strong> / {{ parse(task).total }}
          </template>
          <template v-if="parse(task).layers"> · {{ t('docker.image.layersDone', { done: parse(task).done, total: parse(task).layers }) }}</template>
          <template v-if="!parse(task).total && !parse(task).layers">{{ t('docker.image.pullWaiting') }}</template>
        </span>
      </span>
      <span class="pull__pct m-num">{{ parse(task).percent }}<small>%</small></span>
      <AppIconButton icon="HOutline:DocumentTextIcon" :label="t('docker.image.viewPullLog')" @click="emit('open', task)" />
    </div>
    <div class="pull__bar" role="progressbar" :aria-valuenow="parse(task).percent" aria-valuemin="0" aria-valuemax="100">
      <span :style="{ width: `${parse(task).percent}%` }" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { listDockerTasks } from '@/api/docker'
import { DockerTaskStatus, DockerTaskType } from '@/types/v1/docker'
import type { DockerTaskInfo } from '@/types/v1/docker'

const emit = defineEmits<{ open: [task: DockerTaskInfo]; finished: [] }>()
const { t } = useI18n()
const menuStore = useMenuStore()

const tasks = ref<DockerTaskInfo[]>([])
let timer: ReturnType<typeof setTimeout> | null = null
let alive = true

const isActive = (task: DockerTaskInfo) =>
  task.status === DockerTaskStatus.DOCKER_TASK_STATUS_RUNNING ||
  task.status === DockerTaskStatus.DOCKER_TASK_STATUS_PENDING
const running = computed(() =>
  tasks.value.filter((task) => task.type === DockerTaskType.DOCKER_TASK_TYPE_IMAGE_PULL && isActive(task)),
)

const refOf = (task: DockerTaskInfo) => (task.title || '').replace(/^拉取镜像\s*/, '') || task.resultPath || task.id.slice(0, 8)
const statusText = (task: DockerTaskInfo) =>
  task.status === DockerTaskStatus.DOCKER_TASK_STATUS_PENDING ? t('docker.task.status.pending') : t('docker.image.inProgress')

const parse = (task: DockerTaskInfo) => {
  const text = task.progress || ''
  const pct = text.match(/^(\d+)%/)
  const bytes = text.match(/\(([^/)]+)\/([^)]+)\)/)
  const layers = text.match(/(\d+)\/(\d+)\s*$/)
  return {
    percent: pct ? Math.min(100, Number(pct[1])) : 0,
    current: bytes?.[1]?.trim() || '',
    total: bytes?.[2]?.trim() || '',
    done: layers ? Number(layers[1]) : 0,
    layers: layers ? Number(layers[2]) : 0,
  }
}

const schedule = () => {
  if (timer) clearTimeout(timer)
  timer = null
  if (alive && running.value.length) timer = setTimeout(() => void refresh(), 2500)
}

const refresh = async () => {
  const before = running.value.length
  try {
    const { data } = await listDockerTasks({})
    tasks.value = data?.tasks || []
  } catch {
    /* 拉取卡片是辅助信息，失败时静默 */
  }
  if (before > running.value.length) emit('finished')
  schedule()
}

defineExpose({ refresh })
onMounted(() => { void refresh() })
onBeforeUnmount(() => {
  alive = false
  if (timer) clearTimeout(timer)
})
</script>

<style scoped lang="scss">
.pull {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px 12px 16px 18px;
  border-radius: var(--app-radius-xl);
  border: 1.5px solid var(--m-a2);
  background: var(--el-bg-color);
  box-shadow: 0 0 0 3px var(--m-a2Soft);
}
.pull__head {
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 0;
}
.pull__sticker {
  width: 44px;
  height: 44px;
  border-radius: 15px;
  transform: rotate(-5deg);
}
.pull__sticker :deep(svg) {
  width: 22px;
  height: 22px;
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
  gap: 8px;
  min-width: 0;
}
.pull__label {
  font-size: 1rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.pull__ref {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.pull__sub {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.pull__sub strong {
  color: var(--el-text-color-primary);
}
.pull__pct {
  font-family: var(--app-font-display);
  font-size: 2rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: var(--el-text-color-primary);
}
.pull__pct small {
  font-size: 1rem;
}
.pull__bar {
  height: 10px;
  margin-right: 6px;
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
@media (width < 640px) {
  .pull__label {
    display: none;
  }
  .pull__pct {
    font-size: 1.5rem;
  }
}
</style>
