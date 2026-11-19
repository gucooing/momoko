<!-- Agent 管理 · 模型供应商：「服务器上的登录」（用户自己在服务器上登录命令行，这里只检查登录了没有、读出能用的模型）
     和「第三方接口」两组。 -->
<template>
  <div class="ag ag-page">
    <AgManageHeader>{{ subtitle }}</AgManageHeader>

    <div class="ag-body">
      <h2 class="ag-sh">{{ t('agent.providers.serverLogin') }}</h2>
      <div class="ag-grp">
        <div v-for="r in runtimes" :key="r.id" class="ag-rw">
          <div class="ag-rm">
            <div class="ag-rt">
              <span>{{ r.name }}</span>
              <AgAgentTag :kind="agentKindOf(r.kind)" :name="r.name" />
            </div>
            <div class="ag-rs">{{ loginSub(r) }}</div>
          </div>
          <div class="ag-rc">
            <span class="ag-pill ag-pv-pill" :class="loginPill(r).cls">{{ loginPill(r).text }}</span>
            <button
              type="button"
              class="ag-btn ag-btn--sm"
              :disabled="!r.ready || checking.has(r.id)"
              @click="check(r)"
            >
              <UIcon
                :name="checking.has(r.id) ? 'i-lucide-loader-circle' : 'i-lucide-scan-search'"
                :class="{ 'ag-spin': checking.has(r.id) }"
              />{{ t('agent.providers.check') }}
            </button>
          </div>
        </div>
        <div v-if="!runtimes.length" class="ag-empty">{{ t('agent.common.loading') }}</div>
      </div>

      <div class="ag-sh-row ag-gap">
        <h2 class="ag-sh">{{ t('agent.providers.apis') }}</h2>
        <button type="button" class="ag-btn ag-btn--sm" @click="openForm(null)">
          <UIcon name="i-lucide-plus" />{{ t('agent.providers.addApi') }}
        </button>
      </div>
      <div class="ag-grp">
        <div v-for="p in providers" :key="p.id" class="ag-rw">
          <div class="ag-rm">
            <div class="ag-rt">
              <span>{{ p.name }}</span>
              <AgAgentTag :kind="agentKind(p)" :name="agentName(p)" />
              <span class="ag-tag">{{ protocolTag(p) }}</span>
            </div>
            <div class="ag-rs ag-pv-sub">
              <span class="ag-mono">{{ p.baseUrl }}</span> · {{ modelsSummary(p) }}
              <template v-if="p.apiKeyTail"> · <span class="ag-mono">sk-••••{{ p.apiKeyTail }}</span></template>
            </div>
          </div>
          <div class="ag-rc">
            <span class="ag-pill ag-pv-pill" :class="apiPill(p).cls">{{ apiPill(p).text }}</span>
            <button
              type="button"
              class="ag-ib ag-ib--sm"
              :aria-label="`${t('agent.common.testConnection')} ${p.name}`"
              :title="t('agent.common.testConnection')"
              :disabled="testing.has(p.id)"
              @click="runTest(p)"
            >
              <UIcon
                :name="testing.has(p.id) ? 'i-lucide-loader-circle' : 'i-lucide-zap'"
                :class="{ 'ag-spin': testing.has(p.id) }"
              />
            </button>
            <button
              type="button"
              class="ag-ib ag-ib--sm"
              :aria-label="`${t('agent.common.edit')} ${p.name}`"
              :title="t('agent.common.edit')"
              @click="openForm(p)"
            >
              <UIcon name="i-lucide-pencil" />
            </button>
            <button
              type="button"
              class="ag-ib ag-ib--sm"
              :aria-label="`${t('agent.common.delete')} ${p.name}`"
              :title="t('agent.common.delete')"
              @click="askDelete(p)"
            >
              <UIcon name="i-lucide-trash-2" />
            </button>
          </div>
        </div>
        <div v-if="!providers.length" class="ag-empty">
          {{ loaded ? t('agent.providers.noApis') : t('agent.common.loading') }}
        </div>
      </div>
    </div>

    <ProviderDialog v-model="formOpen" :provider="formTarget" @saved="reload" />
    <AgConfirm
      v-model="deleteOpen"
      danger
      :title="t('agent.providers.deleteTitle', { name: deleteTarget?.name || '' })"
      :message="t('agent.providers.deleteApiMsg')"
      :confirm-text="t('agent.common.delete')"
      :on-confirm="doDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { deleteAgentProvider, detectAgentRuntime, listAgentProviders, listAgentRuntimes, testAgentProvider } from '@/api/agent'
import type { AgentProvider, AgentRuntime } from '@/types/v1/agent'
import AgAgentTag from '@/views/agent/components/AgAgentTag.vue'
import AgConfirm from '@/views/agent/components/AgConfirm.vue'
import AgManageHeader from '@/views/agent/components/AgManageHeader.vue'
import { useAgentOverview } from '@/views/agent/composables/useAgentOverview'
import { agentKindOf } from '@/views/agent/utils/format'
import ProviderDialog from './ProviderDialog.vue'

defineOptions({ name: 'AgentManageProvidersView' })

const { t } = useI18n()
const { overview, refresh } = useAgentOverview()

const runtimes = ref<AgentRuntime[]>([])
const providers = ref<AgentProvider[]>([])
const loaded = ref(false)

const subtitle = computed(() => {
  const o = overview.value
  return o ? t('agent.providers.subtitle', { n: o.providers }) : ''
})

const load = async () => {
  try {
    const [rts, ps] = await Promise.all([listAgentRuntimes(), listAgentProviders()])
    runtimes.value = rts.data.runtimes || []
    providers.value = ps.data.providers || []
  } catch {
    /* shown */
  } finally {
    loaded.value = true
  }
}
/** 改了数据之后：重新拉列表，并刷新标签上的数量 */
const reload = () => {
  void refresh()
  return load()
}

// —— 服务器上的登录 ——
const loginSub = (r: AgentRuntime) => {
  if (!r.ready) return t('agent.providers.notReadySub')
  if (!r.detectedAt) return t('agent.providers.uncheckedSub')
  if (!r.loggedIn) return t('agent.providers.notLoggedInSub', { agent: r.name })
  const n = (r.models || []).length
  const account = [r.authLabel, r.authDetail].filter(Boolean).join(' · ')
  return account ? t('agent.providers.serverLoginSub', { account, n }) : t('agent.providers.serverLoginSubAnon', { n })
}
const loginPill = (r: AgentRuntime) => {
  if (!r.ready) return { cls: 'ag-pill--bad', text: t('agent.providers.status.notReady') }
  if (!r.detectedAt) return { cls: '', text: t('agent.providers.status.unchecked') }
  if (r.loggedIn) return { cls: 'ag-pill--ok', text: t('agent.providers.status.loggedIn') }
  return { cls: 'ag-pill--warn', text: t('agent.providers.status.notLoggedIn') }
}

// 检查：启动一次这个 Agent，看服务器上登录了没有，登录了就读它给的模型和每个模型的推理强度
const checking = ref(new Set<string>())
const check = async (r: AgentRuntime) => {
  checking.value = new Set(checking.value).add(r.id)
  try {
    const { data } = await detectAgentRuntime(r.id)
    const rt = data.runtime
    if (rt?.loggedIn) feedback.success(t('agent.providers.checkOk', { name: r.name, n: (rt.models || []).length }))
    else if (rt?.ready) feedback.error(t('agent.providers.checkNotLoggedIn', { name: r.name }))
    else feedback.error(rt?.error || t('agent.providers.status.notReady'))
  } catch {
    /* shown */
  } finally {
    const next = new Set(checking.value)
    next.delete(r.id)
    checking.value = next
    await reload()
  }
}

// —— 第三方接口 ——
// 供应商不绑哪一条 Agent：按接口协议给哪一系用（自己添加的 Agent 检测出是哪一系也能用）
const agentKind = (p: AgentProvider) => (p.vendor === 'openai' ? 'codex' : 'claude')
const agentName = (p: AgentProvider) => t(`agent.family.${agentKind(p)}`)
const apiPill = (p: AgentProvider) => {
  if (p.status === 'ok') {
    const ms = Number(p.testLatencyMs) || 0
    return { cls: 'ag-pill--ok', text: ms > 0 ? t('agent.providers.status.okMs', { ms }) : t('agent.providers.status.ok') }
  }
  if (p.status === 'bad') return { cls: 'ag-pill--bad', text: p.statusText || t('agent.providers.status.failed') }
  return { cls: '', text: t('agent.common.notTested') }
}
const protocolTag = (p: AgentProvider) =>
  t(p.vendor === 'openai' ? 'agent.providers.protocolTag.openai' : 'agent.providers.protocolTag.anthropic')
const modelsSummary = (p: AgentProvider) => {
  const ids = (p.models || []).map((m) => m.id)
  if (!ids.length) return t('agent.providers.noModels')
  if (ids.length <= 2) return ids.join(t('agent.providers.listSep'))
  return t('agent.providers.modelsMore', { first: ids[0] ?? '', n: ids.length })
}

const testing = ref(new Set<string>())
const runTest = async (p: AgentProvider) => {
  if (testing.value.has(p.id)) return
  testing.value = new Set(testing.value).add(p.id)
  try {
    const { data } = await testAgentProvider({ id: p.id })
    if (data.ok) feedback.success(t('agent.providers.testOk', { name: p.name }))
    else feedback.error(t('agent.providers.withMessage', { name: p.name, message: data.message || t('agent.providers.status.failed') }))
  } catch {
    /* shown */
  } finally {
    const next = new Set(testing.value)
    next.delete(p.id)
    testing.value = next
    await load()
  }
}

// —— 添加 / 编辑 ——
const formOpen = ref(false)
const formTarget = ref<AgentProvider | null>(null)
const openForm = (p: AgentProvider | null) => {
  formTarget.value = p
  formOpen.value = true
}

// —— 删除 ——
const deleteOpen = ref(false)
const deleteTarget = ref<AgentProvider | null>(null)
const askDelete = (p: AgentProvider) => {
  deleteTarget.value = p
  deleteOpen.value = true
}
const doDelete = async () => {
  const p = deleteTarget.value
  if (!p) return
  await deleteAgentProvider(p.id)
  feedback.success(t('agent.providers.deleted', { name: p.name }))
  await reload()
}

onMounted(load)
// 标签页在 KeepAlive 里：切回来时静默刷新一次（运行时页面里可能刚检测过）
onActivated(() => {
  if (loaded.value) void load()
})
</script>

<style lang="scss">
.ag-pv-pill {
  margin-right: 4px;
}
// 供应商弹窗里：两列的格子、单选卡片里的「标题 + 一行给哪个 Agent 用」
.ag-pv-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}
.ag-pv-opt__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.ag-pv-opt__t {
  font-size: 13px;
  font-weight: 600;
}
.ag-pv-opt__d {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--m-ink2);
}
@media (width < 992px) {
  .ag-pv-pill {
    margin-right: auto;
  }
  // 接口地址很长时在卡片里折行
  .ag-pv-sub {
    overflow-wrap: anywhere;
  }
}
</style>
