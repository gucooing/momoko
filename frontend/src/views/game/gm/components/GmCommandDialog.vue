<!-- 执行一条 GM 指令（manifest 里的任何指令都能用它执行）：参数按 JSON Schema 渲染；
     风险只决定交互——中、高：可先试运行预览；危险：执行前输入玩家昵称确认（服务端自动备份）。
     执行结果逐个对象显示实际变化；每次执行都写进 GM 记录。 -->
<template>
  <FormDialog
    :model-value="modelValue"
    :title="command?.title"
    :width="640"
    :show-footer="false"
    @update:model-value="close"
  >
    <div v-if="command" class="gcd">
      <div class="gcd__meta">
        <GmRiskPill :risk="command.risk" />
      </div>

      <div v-if="command.target !== 'server'" class="gcd__targets">
        <span class="gcd__label">{{ t('gm.command.targets', { n: targets.length }) }}</span>
        <div class="gcd__chips">
          <span v-for="tg in targets.slice(0, 20)" :key="tg.pid" class="gcd__chip">{{ tg.name || t('gm.player.noName') }}</span>
          <span v-if="targets.length > 20" class="gcd__chip">+{{ targets.length - 20 }}</span>
        </div>
      </div>

      <GmSchemaForm
        v-if="hasParams"
        v-model="params"
        :schema="command.params"
        :pid="targets.length === 1 ? targets[0]?.pid : undefined"
        :exclude-rewards="excludeRewards"
        :locked="lockedKeys"
        :disabled="running"
      />

      <label v-if="command.risk === 'danger'" class="gcd__field gcd__danger">
        <span class="gcd__label">{{ t('gm.command.confirmName', { name: targets[0]?.name || '' }) }}</span>
        <input v-model="confirmName" class="gcd__input" :disabled="running" autocomplete="off" />
        <span class="gcd__hint">{{ t('gm.command.dangerHint') }}</span>
      </label>

      <div v-if="error" class="gcd__error">{{ error }}</div>
      <GmCommandResult v-if="result" :result="result" />

      <div class="gcd__actions">
        <UButton color="neutral" variant="ghost" :disabled="running" @click="close(false)">{{ t('gm.command.close') }}</UButton>
        <UButton color="neutral" variant="outline" :loading="running && dryRunning" :disabled="running" @click="run(true)">
          {{ t('gm.command.dryRun') }}
        </UButton>
        <UButton :color="command.risk === 'danger' ? 'error' : 'primary'" :loading="running && !dryRunning" :disabled="!canRun" @click="run(false)">
          {{ t('gm.command.execute') }}
        </UButton>
      </div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { executeGmCommand } from '@/api/gameGm'
import { getRequestErrorMessage } from '@/utils/request'
import type { CommandResult, ManifestCommand } from '@/games/mgp'
import { useGmContext } from '../context'
import GmCommandResult from './GmCommandResult.vue'
import GmRiskPill from './GmRiskPill.vue'
import GmSchemaForm from './GmSchemaForm.vue'
import type { GmTargetRef } from './types'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    command?: ManifestCommand
    targets?: GmTargetRef[]
    /** 预填参数 */
    initialParams?: Record<string, unknown>
    /** 写进 GM 记录的可展示摘要 */
    summary?: Record<string, string>
    /** 奖励里不允许的资源类型（manifest resourceTypes 的键） */
    excludeRewards?: string[]
  }>(),
  { command: undefined, targets: () => [], initialParams: () => ({}), summary: () => ({}), excludeRewards: () => [] },
)
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  /** 真实执行完成（不含试运行） */
  done: [result: CommandResult | undefined, status: string]
}>()

const { t } = useI18n()
const ctx = useGmContext()

const params = ref<Record<string, unknown>>({})
const confirmName = ref('')
const running = ref(false)
const dryRunning = ref(false)
const result = ref<CommandResult>()
const error = ref('')

watch(
  () => [props.modelValue, props.command?.id] as const,
  ([open]) => {
    if (!open) return
    params.value = { ...props.initialParams }
    confirmName.value = ''
    result.value = undefined
    error.value = ''
  },
  { immediate: true },
)

// 打开时带进来的参数（从道具、备份的某一行点进来）已经定好，表单里不能再改
const lockedKeys = computed(() => Object.keys(props.initialParams).filter((k) => props.initialParams[k] !== undefined && props.initialParams[k] !== ''))
const hasParams = computed(() => Object.keys(props.command?.params?.properties || {}).length > 0)
const canRun = computed(() => {
  if (!props.command || running.value) return false
  if (props.command.target !== 'server' && !props.targets.length) return false
  if (props.command.risk === 'danger' && !confirmName.value.trim()) return false
  return true
})

const run = async (dryRun: boolean) => {
  const cmd = props.command
  if (!cmd) return
  running.value = true
  dryRunning.value = dryRun
  error.value = ''
  try {
    const { data } = await executeGmCommand({
      serverId: ctx.serverId.value,
      command: cmd.id,
      targets: cmd.target === 'server' ? [] : props.targets.map((x) => x.pid),
      params: params.value,
      dryRun,
      confirmName: confirmName.value.trim(),
      summary: props.summary,
    })
    result.value = (data.result as CommandResult | undefined) || undefined
    if (data.status === 'failed' && data.errorCode) error.value = `${data.errorCode} ${data.errorMessage}`
    if (!dryRun) emit('done', result.value, data.status)
  } catch (e) {
    error.value = getRequestErrorMessage(e)
  } finally {
    running.value = false
    dryRunning.value = false
  }
}

const close = (open: boolean) => {
  if (!open && !running.value) emit('update:modelValue', false)
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
.gcd__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.gcd__label {
  font-size: 13px;
  font-weight: 700;
}
.gcd__req {
  margin-left: 2px;
  color: var(--m-bad);
}
.gcd__input {
  min-height: 40px;
  padding: 0 12px;
  border-radius: var(--app-radius-sm);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
}
.gcd__danger {
  padding: 12px;
  border-radius: var(--app-radius-sm);
  background: var(--m-badSoft);
}
.gcd__hint {
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
