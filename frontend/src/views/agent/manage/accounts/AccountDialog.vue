<!-- Agent 管理 · 官方账户 · 配置：每个账户都能打开，登没登录都一样。
     分两页：代理（只有 Linux 有这一页）和模型（检查时读到的模型设推理等级；Claude 系、Codex 系还能设最长上下文，
     Claude 系还有最长输出；没登录时说明登录后才有）。
     保存时两页一起存：代理没改就不提交（提交会让正在跑的会话进程重启）。 -->
<template>
  <AgDialog
    :model-value="modelValue"
    :title="t('agent.accounts.configTitle', { name: runtime?.name || '' })"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div v-if="proxySupported" class="ag-seg ag-acd-tabs" role="tablist">
      <button
        v-for="p in pages"
        :key="p.key"
        type="button"
        role="tab"
        :class="{ 'is-on': page === p.key }"
        :aria-selected="page === p.key"
        @click="page = p.key"
      >
        <UIcon :name="p.icon" />{{ p.label }}
      </button>
    </div>

    <AgProxyField
      v-if="proxySupported && page === 'proxy'"
      v-model="proxy"
      :unavailable="proxyUnavailable"
      :note="t('agent.proxy.noteAccount', { name: runtime?.name || '' })"
      :runtime-id="runtime?.id"
    />

    <template v-else>
      <AgModelCatalog
        v-if="modelsShown"
        v-model="rows"
        levels
        :context="withContext"
        :output="withOutput"
        :default-label="t('agent.providers.loginModels.agentDefault')"
        :limits-hint="t(withOutput ? 'agent.providers.loginModels.hint' : 'agent.providers.loginModels.hintCodex')"
        :value-label="t('agent.accounts.levelValue')"
        :levels-hint="t('agent.accounts.levelsHint')"
        :fetch-levels="fetchLevels"
      />
      <div v-else class="ag-acd-empty">
        <UIcon name="i-lucide-boxes" />
        <span>{{ t('agent.accounts.modelsEmpty', { name: runtime?.name || '' }) }}</span>
      </div>
    </template>

    <template #footer>
      <button type="button" class="ag-btn" @click="emit('update:modelValue', false)">{{ t('agent.common.cancel') }}</button>
      <button type="button" class="ag-btn ag-btn--p" :disabled="saving || (!modelsShown && !proxySupported)" @click="save">
        <UIcon v-if="saving" name="i-lucide-loader-circle" class="ag-spin" />{{ t('agent.common.save') }}
      </button>
    </template>
  </AgDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { getAgentRuntimeModelLevels, setAgentRuntimeModels, setAgentRuntimeProxy } from '@/api/agent'
import type { AgentRuntime } from '@/types/v1/agent'
import AgDialog from '@/views/agent/components/AgDialog.vue'
import AgModelCatalog from '@/views/agent/components/AgModelCatalog.vue'
import AgProxyField from '@/views/agent/components/AgProxyField.vue'
import { modelLabel } from '@/views/agent/composables/useAgentModels'
import { type CatalogRow, levelRow } from '@/views/agent/utils/catalog'
import { agentKindOf } from '@/views/agent/utils/format'
import { proxyForm, proxyPayload, proxyReady, type ProxyForm } from '@/views/agent/utils/proxy'

type Page = 'proxy' | 'models'

const props = defineProps<{
  modelValue: boolean
  runtime?: AgentRuntime | null
  /** 这台服务器能配代理（只有 Linux）；proxyUnavailable 是现在用不了的原因 */
  proxySupported?: boolean
  proxyUnavailable?: string
}>()
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  saved: []
}>()
const { t } = useI18n()

const pages = computed<{ key: Page; label: string; icon: string }[]>(() => [
  { key: 'proxy', label: t('agent.accounts.proxy'), icon: 'i-lucide-shield' },
  { key: 'models', label: t('agent.accounts.models'), icon: 'i-lucide-boxes' },
])
const page = ref<Page>('proxy')
// 检查时读到的模型：只能改最长上下文和最长输出
const rows = ref<CatalogRow[]>([])
const proxy = ref<ProxyForm>(proxyForm())
let proxyBefore = ''
const saving = ref(false)

// 最长上下文和最长输出交得给哪一系：Claude 系两项都有，Codex 系只有上下文，别的都没有
const kind = computed(() => agentKindOf(props.runtime?.kind))
const withContext = computed(() => kind.value === 'claude' || kind.value === 'codex')
const withOutput = computed(() => kind.value === 'claude')
// 登录并检查过、读到了模型才有得设
const modelsShown = computed(() => !!props.runtime?.loggedIn && !!props.runtime?.models?.length)
// 「同步」：开一次 Agent 切到这个模型，读它现在给的推理等级
const fetchLevels = (model: string) =>
  getAgentRuntimeModelLevels(props.runtime?.id || '', model).then(({ data }) => data.levels)

const limit = (v: number | string | undefined): number | '' => (Number(v) > 0 ? Number(v) : '')
watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    // 能配代理时先看代理（没登录也要能先设好）；登录了、读到模型的先看模型
    page.value = props.proxySupported && !modelsShown.value ? 'proxy' : 'models'
    rows.value = (props.runtime?.models || []).map((m) => ({
      id: m.id,
      name: modelLabel(m),
      levels: (m.efforts || []).map((e) => levelRow(e.name || e.id, e.id)),
      contextWindow: limit(m.contextWindow),
      maxOutput: limit(m.maxOutput),
    }))
    proxy.value = proxyForm(props.runtime?.proxy)
    proxyBefore = JSON.stringify(proxyPayload(proxy.value))
    saving.value = false
  },
  { immediate: true },
)

const save = async () => {
  const rt = props.runtime
  if (!rt || saving.value) return
  if (props.proxySupported && !proxyReady(proxy.value)) {
    page.value = 'proxy'
    feedback.warning(t('agent.proxy.notReady'))
    return
  }
  saving.value = true
  try {
    const nextProxy = proxyPayload(proxy.value)
    if (props.proxySupported && JSON.stringify(nextProxy) !== proxyBefore) {
      await setAgentRuntimeProxy(rt.id, nextProxy)
    }
    if (modelsShown.value) {
      await setAgentRuntimeModels(
        rt.id,
        rows.value.map((m) => ({
          model: m.id,
          contextWindow: withContext.value ? Math.max(0, Math.round(Number(m.contextWindow) || 0)) : 0,
          maxOutput: withOutput.value ? Math.max(0, Math.round(Number(m.maxOutput) || 0)) : 0,
          levels: m.levels.map((r) => ({ name: r.name.trim(), value: r.value.trim() })).filter((r) => r.name),
        })),
      )
    }
    feedback.success(t('agent.accounts.saved', { name: rt.name }))
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
.ag-acd-tabs {
  align-self: flex-start;
  min-width: 220px;
  margin-bottom: 4px;
}
.ag-acd-tabs .iconify {
  width: 14px;
  height: 14px;
}
.ag-acd-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 28px 16px;
  border-radius: 12px;
  background: var(--m-s2);
  color: var(--m-ink2);
  font-size: 12.5px;
  line-height: 1.6;
  text-align: center;
}
.ag-acd-empty .iconify {
  width: 22px;
  height: 22px;
  opacity: 0.7;
}
@media (width < 560px) {
  .ag-acd-tabs {
    align-self: stretch;
    min-width: 0;
  }
}
</style>
