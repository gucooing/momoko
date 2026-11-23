<!-- Agent 管理 · 官方账户：Claude Code、Codex（和自己添加的 Agent）在服务器上登录的账户，大家共用。
     用户自己在服务器上登录命令行，这里只「检查」登录了没有、读出能用的模型，再给读到的模型设最长上下文和最长输出。
     大家在 Agent 设置的模型供应商里能看到（只读）；第三方接口每个人在那里加自己的。 -->
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
            <div class="ag-rs">{{ loginSub(t, r, true) }}</div>
          </div>
          <div class="ag-rc">
            <span class="ag-pill ag-ac-pill" :class="loginPill(t, r).cls">{{ loginPill(t, r).text }}</span>
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
            <button
              v-if="r.loggedIn && r.models?.length && agentKindOf(r.kind) !== 'custom'"
              type="button"
              class="ag-ib ag-ib--sm"
              :aria-label="`${t('agent.providers.loginModels.edit')} ${r.name}`"
              :title="t('agent.providers.loginModels.edit')"
              @click="openModels(r)"
            >
              <UIcon name="i-lucide-pencil" />
            </button>
          </div>
        </div>
        <div v-if="!runtimes.length" class="ag-empty">{{ t('agent.common.loading') }}</div>
      </div>
    </div>

    <LoginModelsDialog v-model="modelsOpen" :runtime="modelsTarget" @saved="reload" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { detectAgentRuntime, listAgentRuntimes } from '@/api/agent'
import type { AgentRuntime } from '@/types/v1/agent'
import AgAgentTag from '@/views/agent/components/AgAgentTag.vue'
import AgManageHeader from '@/views/agent/components/AgManageHeader.vue'
import { useAgentModels } from '@/views/agent/composables/useAgentModels'
import { useAgentOverview } from '@/views/agent/composables/useAgentOverview'
import { agentKindOf } from '@/views/agent/utils/format'
import { loginPill, loginSub } from './login'
import LoginModelsDialog from './LoginModelsDialog.vue'

defineOptions({ name: 'AgentManageAccountsView' })

const { t } = useI18n()
const { refresh } = useAgentOverview()
const { refresh: refreshModels } = useAgentModels()

const runtimes = ref<AgentRuntime[]>([])
const loaded = ref(false)

// 已登录（和「已登录」那个标记一样：就绪且登录着）/ 一共几个 Agent
const subtitle = computed(() => {
  if (!loaded.value) return ''
  const logins = runtimes.value.filter((r) => r.ready && r.loggedIn).length
  return t('agent.accounts.subtitle', { n: runtimes.value.length, logins })
})

const load = async () => {
  try {
    const { data } = await listAgentRuntimes()
    runtimes.value = data.runtimes || []
  } catch {
    /* shown */
  } finally {
    loaded.value = true
  }
}
/** 改了数据之后：重新拉列表，刷新标签上的数量和各页面共用的模型清单 */
const reload = () => {
  void refresh()
  void refreshModels()
  return load()
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

// 编辑模型：给检查时读到的模型设最长上下文和最长输出
const modelsOpen = ref(false)
const modelsTarget = ref<AgentRuntime | null>(null)
const openModels = (r: AgentRuntime) => {
  modelsTarget.value = r
  modelsOpen.value = true
}

onMounted(load)
// 标签页在 KeepAlive 里：切回来时静默刷新一次（运行时标签里可能刚检测过，会话里可能发现登录失效了），标签上的数量一起刷新
onActivated(() => {
  if (loaded.value) void reload()
})
</script>

<style lang="scss">
.ag-ac-pill {
  margin-right: 4px;
}
@media (width < 992px) {
  .ag-ac-pill {
    margin-right: auto;
  }
}
</style>
