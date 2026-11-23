<!-- Agent 管理 · 官方账户 · 编辑模型：检查时读到的每个模型设最长上下文（Claude 系还有最长输出），留空用 Agent 自己的。 -->
<template>
  <AgDialog
    :model-value="modelValue"
    :title="t('agent.providers.loginModels.title', { name: runtime?.name || '' })"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div class="ag-lm" :class="{ 'ag-lm--ctx': !withOutput }">
      <div class="ag-lm__row ag-lm__head" aria-hidden="true">
        <span>{{ t('agent.providers.loginModels.model') }}</span>
        <span>{{ t('agent.providers.form.contextWindow') }}</span>
        <span v-if="withOutput">{{ t('agent.providers.form.maxOutput') }}</span>
      </div>
      <div v-for="m in rows" :key="m.id" class="ag-lm__row" role="group" :aria-label="t('agent.providers.form.limitsOf', { name: m.name })">
        <span class="ag-lm__n" :title="m.id">{{ m.name }}</span>
        <span class="ag-fld ag-fld--plain ag-pv-lv">
          <input
            v-model.number="m.contextWindow"
            class="ag-mono"
            type="number"
            min="1"
            step="1000"
            :placeholder="t('agent.providers.loginModels.agentDefault')"
            :aria-label="t('agent.providers.form.contextWindow')"
          />
          <span v-if="Number(m.contextWindow) > 0" class="ag-pv-unit ag-num">{{ fmtTokens(m.contextWindow) }}</span>
        </span>
        <span v-if="withOutput" class="ag-fld ag-fld--plain ag-pv-lv">
          <input
            v-model.number="m.maxOutput"
            class="ag-mono"
            type="number"
            min="1"
            step="1000"
            :placeholder="t('agent.providers.loginModels.agentDefault')"
            :aria-label="t('agent.providers.form.maxOutput')"
          />
          <span v-if="Number(m.maxOutput) > 0" class="ag-pv-unit ag-num">{{ fmtTokens(m.maxOutput) }}</span>
        </span>
      </div>
      <div v-if="!rows.length" class="ag-empty">{{ t('agent.providers.noModels') }}</div>
    </div>
    <span class="ag-fh">{{ t(withOutput ? 'agent.providers.loginModels.hint' : 'agent.providers.loginModels.hintCodex') }}</span>

    <template #footer>
      <button type="button" class="ag-btn" @click="emit('update:modelValue', false)">{{ t('agent.common.cancel') }}</button>
      <button type="button" class="ag-btn ag-btn--p" :disabled="saving || !rows.length" @click="save">
        <UIcon v-if="saving" name="i-lucide-loader-circle" class="ag-spin" />{{ t('agent.common.save') }}
      </button>
    </template>
  </AgDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { setAgentRuntimeModels } from '@/api/agent'
import type { AgentRuntime } from '@/types/v1/agent'
import AgDialog from '@/views/agent/components/AgDialog.vue'
import { modelLabel } from '@/views/agent/composables/useAgentModels'
import { agentKindOf, fmtTokens } from '@/views/agent/utils/format'

interface Row {
  id: string
  name: string
  /** 空串 = 没设，用 Agent 自己的 */
  contextWindow: number | ''
  maxOutput: number | ''
}

const props = defineProps<{ modelValue: boolean; runtime?: AgentRuntime | null }>()
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  saved: []
}>()
const { t } = useI18n()

const rows = ref<Row[]>([])
const saving = ref(false)
// Codex 没有设定最长输出的配置，只有 Claude 系有这一列
const withOutput = computed(() => agentKindOf(props.runtime?.kind) === 'claude')

const limit = (v: number | string | undefined): number | '' => (Number(v) > 0 ? Number(v) : '')
watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    rows.value = (props.runtime?.models || []).map((m) => ({
      id: m.id,
      name: modelLabel(m),
      contextWindow: limit(m.contextWindow),
      maxOutput: limit(m.maxOutput),
    }))
    saving.value = false
  },
  { immediate: true },
)

const save = async () => {
  const rt = props.runtime
  if (!rt) return
  saving.value = true
  try {
    await setAgentRuntimeModels(
      rt.id,
      rows.value.map((m) => ({
        model: m.id,
        contextWindow: Math.max(0, Math.round(Number(m.contextWindow) || 0)),
        maxOutput: withOutput.value ? Math.max(0, Math.round(Number(m.maxOutput) || 0)) : 0,
      })),
    )
    feedback.success(t('agent.providers.loginModels.saved', { name: rt.name }))
    emit('saved')
    emit('update:modelValue', false)
  } catch {
    /* shown */
  } finally {
    saving.value = false
  }
}
</script>

<style lang="scss">
// 模型 + 最长上下文 + 最长输出（Codex 没有最长输出这一列）
.ag-lm {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.ag-lm__row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) minmax(0, 1fr);
  gap: 6px;
  align-items: center;
}
.ag-lm--ctx .ag-lm__row {
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.2fr);
}
.ag-lm__head {
  padding: 0 2px;
  font-size: 12px;
  font-weight: 500;
  color: var(--m-ink2);
}
.ag-lm__n {
  min-width: 0;
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
