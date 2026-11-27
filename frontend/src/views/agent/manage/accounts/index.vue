<!-- Agent 管理 · 官方账户：Claude Code、Codex（和自己添加的 Agent）在服务器上登录的账户，大家共用。
     没登录的点「登录」：在悬浮终端里打开一个标签，在服务器上跑这一系的登录命令（配了代理就只经代理联网），
     命令结束后服务器自动检查一次，这里收到「配置变了」就刷新。也可以自己点「检查」。
     配置（每个账户都有，登没登录都能开）：分「代理」（只有 Linux 有这一页）和「模型」两页。
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
              <span v-if="r.proxy?.protocol" class="ag-tag ag-px-tag" :title="t('agent.proxy.via', { addr: proxyLabel(r.proxy) })">
                <UIcon name="i-lucide-shield" />{{ proxyLabel(r.proxy) }}
              </span>
            </div>
            <div class="ag-rs">{{ loginSub(t, r, true) }}</div>
          </div>
          <div class="ag-rc">
            <span class="ag-pill ag-ac-pill" :class="loginPill(t, r).cls">{{ loginPill(t, r).text }}</span>
            <button
              v-if="canLogin(r)"
              type="button"
              class="ag-btn ag-btn--sm ag-btn--p"
              :aria-label="t('agent.accounts.loginOf', { name: r.name })"
              @click="openLogin(r)"
            >
              <UIcon name="i-lucide-log-in" />{{ t('agent.accounts.login') }}
            </button>
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
              type="button"
              class="ag-ib ag-ib--sm"
              :aria-label="t('agent.accounts.configTitle', { name: r.name })"
              :title="t('agent.accounts.config')"
              @click="openConfig(r)"
            >
              <UIcon name="i-lucide-pencil" />
            </button>
          </div>
        </div>
        <div v-if="!runtimes.length" class="ag-empty">{{ t('agent.common.loading') }}</div>
      </div>
    </div>

    <AccountDialog
      v-model="configOpen"
      :runtime="configTarget"
      :proxy-supported="proxySupported"
      :proxy-unavailable="proxyUnavailable"
      @saved="reload"
    />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { detectAgentRuntime, listAgentRuntimes } from '@/api/agent'
import { addTab } from '@/layouts/app/termDock/state'
import type { AgentRuntime } from '@/types/v1/agent'
import AgAgentTag from '@/views/agent/components/AgAgentTag.vue'
import AgManageHeader from '@/views/agent/components/AgManageHeader.vue'
import { useAgentConn } from '@/views/agent/composables/agentConn'
import { useAgentModels } from '@/views/agent/composables/useAgentModels'
import { useAgentOverview } from '@/views/agent/composables/useAgentOverview'
import { agentKindOf } from '@/views/agent/utils/format'
import { proxyLabel, sandboxProblem } from '@/views/agent/utils/proxy'
import AccountDialog from './AccountDialog.vue'
import { loginPill, loginSub } from './login'

defineOptions({ name: 'AgentManageAccountsView' })

const { t } = useI18n()
const { refresh } = useAgentOverview()
const { refresh: refreshModels } = useAgentModels()

const runtimes = ref<AgentRuntime[]>([])
const loaded = ref(false)
/** 这台服务器能配代理（只有 Linux）；proxyUnavailable 是沙箱现在用不了的原因（建不了、sing-box 没下载） */
const proxySupported = ref(false)
const proxyUnavailable = ref('')

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
    proxySupported.value = !!data.sandbox?.supported
    proxyUnavailable.value = sandboxProblem(t, data.sandbox)
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

// 登录：就绪、没登录、有登录命令的才有
const canLogin = (r: AgentRuntime) => r.ready && !r.loggedIn && r.canLogin
const openLogin = (r: AgentRuntime) => {
  addTab('login', { runtimeId: r.id, label: t('termDock.login', { name: r.name }) })
}

// 配置：每个账户都能开（代理要在登录之前就能设好）
const configOpen = ref(false)
const configTarget = ref<AgentRuntime | null>(null)
const openConfig = (r: AgentRuntime) => {
  configTarget.value = r
  configOpen.value = true
}

// 登录终端结束后服务器会自动检查一次，检查完推「配置变了」：收到就刷新（连着发来的合成一次）
let pending: ReturnType<typeof setTimeout> | undefined
const conn = useAgentConn(
  (msg) => {
    if (msg.t !== 'config') return
    if (pending) clearTimeout(pending)
    pending = setTimeout(() => void load(), 300)
  },
  () => {},
)

onMounted(() => {
  void load()
  conn.start()
})
// 标签页在 KeepAlive 里：切回来时静默刷新一次（运行时标签里可能刚检测过，会话里可能发现登录失效了），标签上的数量一起刷新
onActivated(() => {
  conn.start()
  if (loaded.value) void reload()
})
onDeactivated(() => conn.stop())
</script>

<style lang="scss">
.ag-ac-pill {
  margin-right: 4px;
}
// 走代理的标记
.ag-px-tag {
  gap: 4px;
}
.ag-px-tag .iconify {
  width: 12px;
  height: 12px;
}
@media (width < 992px) {
  .ag-ac-pill {
    margin-right: auto;
  }
}
</style>
