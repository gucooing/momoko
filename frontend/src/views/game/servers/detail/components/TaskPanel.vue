<!-- 托管任务进度（更新 / 资源 / 备份 / 回档）：步骤时间线 + 任务日志，进行中每 2 秒刷新，结束时通知父组件。 -->
<template>
  <div v-if="task" class="tp">
    <div class="tp__head">
      <b>{{ t(`gameServers.task.kind.${task.kind}`, task.kind) }}</b>
      <StatusPill :variant="variant" :label="t(`gameServers.task.status.${task.status}`, task.status)" />
      <span class="tp__meta">
        <span v-if="task.userName">{{ t('gameServers.task.by', { name: task.userName }) }}</span>
        <span class="m-num">{{ fmtTime(task.startTime || task.createTime) }}</span>
      </span>
      <UButton v-if="active" size="sm" color="neutral" variant="ghost" icon="i-lucide-x" :loading="canceling" @click="cancel">
        {{ t('gameServers.task.cancel') }}
      </UButton>
    </div>
    <ol v-if="task.steps.length" class="tp__steps">
      <li v-for="s in task.steps" :key="s.key" :class="`is-${s.state}`">
        <UIcon :name="stepIcon(s.state)" :class="{ 'tp__spin': s.state === 'running' }" />
        <span>{{ t(`gameServers.task.step.${s.key}`, s.key) }}</span>
        <small v-if="s.detail" class="m-mono">{{ s.detail }}</small>
      </li>
    </ol>
    <p v-if="task.error" class="tp__error">{{ task.error }}</p>
    <details v-if="task.logs.length" class="tp__logs" :open="active">
      <summary>{{ t('gameServers.task.logs') }} · {{ task.logs.length }}</summary>
      <pre class="m-mono">{{ logText }}</pre>
    </details>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import { cancelGameServerTask, getGameServerTask } from '@/api/gameServer'
import { getRequestErrorMessage } from '@/utils/request'
import type { GameServerTask } from '@/types/v1/game_server'
import { fmtTime, type PillVariant } from '../../shared'

const props = defineProps<{ serverId: string; task: GameServerTask | null | undefined }>()
const emit = defineEmits<{ finished: [task: GameServerTask] }>()
const { t } = useI18n()
const fb = useFeedback()

const task = ref<GameServerTask | null>(props.task ?? null)
const canceling = ref(false)
const active = computed(() => task.value?.status === 'pending' || task.value?.status === 'running')
const variant = computed<PillVariant>(() => {
  switch (task.value?.status) {
    case 'success':
      return 'success'
    case 'failed':
      return 'error'
    case 'canceled':
      return 'neutral'
    default:
      return 'info'
  }
})
const logText = computed(() =>
  (task.value?.logs || [])
    .slice(-200)
    .map((l) => `${dayjs(l.time as unknown as string).format('HH:mm:ss')} ${l.level.padEnd(7)} ${l.message}`)
    .join('\n'),
)
const stepIcon = (state: string) => {
  switch (state) {
    case 'success':
      return 'i-lucide-circle-check'
    case 'failed':
      return 'i-lucide-circle-x'
    case 'running':
      return 'i-lucide-loader-circle'
    case 'skipped':
      return 'i-lucide-circle-minus'
    default:
      return 'i-lucide-circle'
  }
}

let timer: ReturnType<typeof setTimeout> | undefined
const poll = async () => {
  clearTimeout(timer)
  const cur = task.value
  if (!cur || !props.serverId) return
  try {
    const { data } = await getGameServerTask(props.serverId, cur.id)
    const wasActive = active.value
    task.value = data
    if (wasActive && !active.value) emit('finished', data)
  } catch {
    // 任务记录可能已被清理：停止轮询
    return
  }
  if (active.value) timer = setTimeout(poll, 2000)
}

watch(
  () => props.task,
  (next) => {
    task.value = next ?? null
    if (active.value) timer = setTimeout(poll, 1000)
  },
  { immediate: true },
)
onBeforeUnmount(() => clearTimeout(timer))

const cancel = async () => {
  if (!task.value) return
  canceling.value = true
  try {
    task.value = (await cancelGameServerTask(props.serverId, task.value.id)).data
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    canceling.value = false
  }
}
</script>

<style scoped lang="scss">
.tp {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px 16px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.tp__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 10px;
  font-size: 0.875rem;
}
.tp__meta {
  flex: 1;
  display: flex;
  flex-wrap: wrap;
  gap: 4px 10px;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.tp__steps {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-wrap: wrap;
  gap: 6px 16px;
  font-size: 0.8125rem;
}
.tp__steps li {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--el-text-color-secondary);
  min-width: 0;
}
.tp__steps li small {
  max-width: 260px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.tp__steps li.is-success {
  color: var(--m-ok);
}
.tp__steps li.is-failed {
  color: var(--m-bad);
}
.tp__steps li.is-running {
  color: var(--m-pInk);
  font-weight: 700;
}
.tp__spin {
  animation: tp-spin 1s linear infinite;
}
@keyframes tp-spin {
  to {
    transform: rotate(360deg);
  }
}
.tp__error {
  margin: 0;
  font-size: 0.8125rem;
  color: var(--m-bad);
  overflow-wrap: anywhere;
}
.tp__logs summary {
  cursor: pointer;
  min-height: 32px;
  display: flex;
  align-items: center;
  font-size: 0.8125rem;
  font-weight: 700;
}
.tp__logs pre {
  margin: 6px 0 0;
  max-height: 240px;
  overflow: auto;
  padding: 10px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--el-bg-color);
  font-size: 0.75rem;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
