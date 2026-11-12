<!-- 执行一个 GM 操作的表单（games/<游戏>/operations.ts 里的定义，各自调用对应接口）：参数按 JSON Schema 渲染；
     支持试运行的操作可先预览变化；危险操作执行前服务端自动备份存档。
     执行结果逐个对象显示实际变化；每次执行都写进 GM 记录。操作对话框与「GM 操作」页的执行面板共用。 -->
<template>
  <div v-if="operation" class="gcd">
    <div class="gcd__meta">
      <GmRiskPill :risk="operation.risk" />
    </div>

    <div v-if="operation.target !== 'server'" class="gcd__targets">
      <span class="gcd__label">{{ t('gm.operation.targets', { n: targets.length }) }}</span>
      <div class="gcd__chips">
        <span v-for="tg in targets.slice(0, 20)" :key="tg.pid" class="gcd__chip">{{ tg.name || t('gm.player.noName') }}</span>
        <span v-if="targets.length > 20" class="gcd__chip">+{{ targets.length - 20 }}</span>
      </div>
    </div>

    <GmSchemaForm
      v-if="hasParams"
      v-model="params"
      :schema="operation.params"
      :pid="targets.length === 1 ? targets[0]?.pid : undefined"
      :exclude-rewards="excludeRewards"
      :locked="lockedKeys"
      :disabled="running"
    />

    <p v-if="operation.risk === 'danger'" class="gcd__hint">{{ t('gm.command.dangerHint') }}</p>

    <div v-if="error" class="gcd__error">{{ error }}</div>
    <GmOperationResult v-if="result" :outcome="result" />

    <div class="gcd__actions">
      <UButton v-if="closable" color="neutral" variant="ghost" :disabled="running" @click="emit('close')">{{ t('gm.command.close') }}</UButton>
      <UButton v-if="operation.dryRun" color="neutral" variant="outline" :loading="running && dryRunning" :disabled="!canRun" @click="run(true)">
        {{ t('gm.command.dryRun') }}
      </UButton>
      <UButton :color="operation.risk === 'danger' ? 'error' : 'primary'" :loading="running && !dryRunning" :disabled="!canRun" @click="run(false)">
        {{ t('gm.command.execute') }}
      </UButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getRequestErrorMessage } from '@/utils/request'
import type { OperationDef, OperationOutcome } from '@/games/mgp'
import { useGmContext } from '../context'
import GmOperationResult from './GmOperationResult.vue'
import GmRiskPill from './GmRiskPill.vue'
import GmSchemaForm from './GmSchemaForm.vue'
import type { GmTargetRef } from './types'

const props = withDefaults(
  defineProps<{
    operation?: OperationDef
    targets?: GmTargetRef[]
    /** 预填参数（已经定好，表单里不能再改） */
    initialParams?: Record<string, unknown>
    /** 写进 GM 记录的可展示摘要 */
    summary?: Record<string, string>
    /** 奖励里不允许的资源类型 */
    excludeRewards?: number[]
    /** 变化时清空表单与结果（对话框每次打开加一） */
    resetKey?: number
    /** 显示「关闭」按钮（对话框里） */
    closable?: boolean
  }>(),
  {
    operation: undefined,
    targets: () => [],
    initialParams: () => ({}),
    summary: () => ({}),
    excludeRewards: () => [],
    resetKey: 0,
    closable: false,
  },
)
const emit = defineEmits<{
  /** 真实执行完成（不含试运行） */
  done: [result: OperationOutcome | undefined, status: string]
  /** 正在执行（对话框据此禁止关闭） */
  busy: [running: boolean]
  close: []
}>()

const { t } = useI18n()
const ctx = useGmContext()

const params = ref<Record<string, unknown>>({})
const running = ref(false)
const dryRunning = ref(false)
const result = ref<OperationOutcome>()
const error = ref('')

watch(
  () => [props.operation?.id, props.resetKey] as const,
  () => {
    params.value = { ...props.initialParams }
    result.value = undefined
    error.value = ''
  },
  { immediate: true },
)
watch(running, (v) => emit('busy', v))

const lockedKeys = computed(() => Object.keys(props.initialParams).filter((k) => props.initialParams[k] !== undefined && props.initialParams[k] !== ''))
const hasParams = computed(() => Object.keys(props.operation?.params?.properties || {}).length > 0)
const canRun = computed(() => {
  if (!props.operation || running.value) return false
  return props.operation.target === 'server' || props.targets.length > 0
})

const run = async (dryRun: boolean) => {
  const op = props.operation
  if (!op) return
  running.value = true
  dryRunning.value = dryRun
  error.value = ''
  try {
    const out = await op.run(
      ctx.serverId.value,
      op.target === 'server' ? [] : props.targets.map((x) => x.pid),
      params.value,
      { dryRun, summary: props.summary },
    )
    result.value = out
    if (out.status === 'failed' && out.errorCode) error.value = `${out.errorCode} ${out.errorMessage || ''}`
    if (!dryRun) emit('done', out, out.status)
  } catch (e) {
    error.value = getRequestErrorMessage(e)
  } finally {
    running.value = false
    dryRunning.value = false
  }
}
</script>

<style scoped>
.gcd {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.gcd__meta {
  display: flex;
  align-items: center;
  gap: 10px;
}
.gcd__targets {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.gcd__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.gcd__chip {
  padding: 4px 10px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  font-size: 13px;
  font-weight: 700;
}
.gcd__label {
  font-size: 13px;
  font-weight: 700;
}
.gcd__hint {
  margin: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.gcd__error {
  font-size: 13px;
  color: var(--m-bad);
}
.gcd__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}
</style>
