<!-- 后台任务：最近的一次性任务（拉取镜像、压缩、复制……）与进度条；颜色随状态变化。 -->
<template>
  <section class="tk" aria-labelledby="tk-h">
    <div class="tk__head">
      <h2 id="tk-h" class="tk__title">{{ t('dashboard.home.tasks') }}</h2>
      <span class="tk__spacer" />
      <RouterLink :to="{ path: '/system/logs', query: { tab: 'task' } }" class="tk__link">
        {{ t('dashboard.home.viewAll') }} →
      </RouterLink>
    </div>

    <p v-if="!tasks.length" class="tk__empty">{{ t('dashboard.home.noTasks') }}</p>

    <div v-for="task in rows" :key="task.id" class="tk__item">
      <div class="tk__line">
        <span class="tk__name" :title="task.title">{{ task.title }}</span>
        <span class="tk__state m-num" :style="{ color: task.color }">{{ task.label }}</span>
      </div>
      <div
        class="tk__track"
        role="progressbar"
        :aria-label="task.title"
        :aria-valuenow="task.indeterminate ? undefined : task.percent"
        aria-valuemin="0"
        aria-valuemax="100"
      >
        <div
          class="tk__fill"
          :class="{ 'is-indeterminate': task.indeterminate }"
          :style="{ width: task.indeterminate ? '40%' : `${task.percent}%`, background: task.color }"
        />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { TaskStatus, type TaskInfo } from '@/types/v1/task'

const props = defineProps<{ tasks: TaskInfo[] }>()
const { t } = useI18n()

const rows = computed(() =>
  props.tasks.map((task) => {
    const total = Number(task.total) || 0
    const finished = Number(task.finished) || 0
    const pct = total > 0 ? Math.min(100, Math.round((finished / total) * 100)) : 0
    const status = task.status
    if (status === TaskStatus.TASK_STATUS_SUCCESS) {
      return { id: task.id, title: task.title || task.type, label: t('dashboard.home.taskDone'), color: 'var(--m-ok)', percent: 100, indeterminate: false }
    }
    if (status === TaskStatus.TASK_STATUS_FAILED) {
      return { id: task.id, title: task.title || task.type, label: t('dashboard.home.taskFailed'), color: 'var(--m-bad)', percent: total > 0 ? pct : 100, indeterminate: false }
    }
    if (status === TaskStatus.TASK_STATUS_CANCELED) {
      return { id: task.id, title: task.title || task.type, label: t('dashboard.home.taskCanceled'), color: 'var(--el-text-color-secondary)', percent: total > 0 ? pct : 100, indeterminate: false }
    }
    if (status === TaskStatus.TASK_STATUS_PENDING) {
      return { id: task.id, title: task.title || task.type, label: t('dashboard.home.taskPending'), color: 'var(--m-a2)', percent: 0, indeterminate: false }
    }
    return {
      id: task.id,
      title: task.title || task.type,
      label: total > 0 ? `${pct}%` : t('dashboard.home.taskRunning'),
      color: 'var(--m-p)',
      percent: pct,
      indeterminate: total <= 0,
    }
  }),
)
</script>

<style scoped lang="scss">
.tk {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
  padding: 22px;
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
}
.tk__head {
  display: flex;
  align-items: center;
}
.tk__title {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 1.125rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.tk__spacer {
  flex: 1;
}
.tk__link {
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--m-pInk);
  text-decoration: none;
}
.tk__link:hover {
  color: var(--el-text-color-primary);
}
.tk__empty {
  margin: 0;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.tk__item {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.tk__line {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.8125rem;
}
.tk__name {
  flex: 1;
  min-width: 0;
  font-weight: 700;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.tk__state {
  flex: none;
  font-weight: 700;
}
.tk__track {
  height: 8px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  overflow: hidden;
}
.tk__fill {
  height: 100%;
  border-radius: 999px;
  transition: width 0.4s ease;
}
.tk__fill.is-indeterminate {
  animation: tk-slide 1.4s ease-in-out infinite;
}
@keyframes tk-slide {
  from {
    transform: translateX(-100%);
  }
  to {
    transform: translateX(250%);
  }
}
</style>
