// 官方账户（服务器上的登录）一行的说明和状态。Agent 管理的「官方账户」里能检查、编辑模型（manage），
// Agent 设置的模型供应商里大家只能看，说明里不叫人去点检查。
import type { AgentRuntime } from '@/types/v1/agent'

type Translate = (key: string, named?: Record<string, unknown>) => string

export const loginSub = (t: Translate, r: AgentRuntime, manage: boolean) => {
  if (!r.ready) return t(manage ? 'agent.providers.notReadySub' : 'agent.providers.view.notReady')
  if (!r.detectedAt) return t(manage ? 'agent.providers.uncheckedSub' : 'agent.providers.view.unchecked')
  if (!r.loggedIn) {
    if (!manage) return t('agent.providers.view.notLoggedIn')
    // 有登录命令的：点「登录」在服务器上打开登录终端
    return t(r.canLogin ? 'agent.accounts.loginSub' : 'agent.providers.notLoggedInSub', { agent: r.name })
  }
  const n = (r.models || []).length
  const account = [r.authLabel, r.authDetail].filter(Boolean).join(' · ')
  return account ? t('agent.providers.serverLoginSub', { account, n }) : t('agent.providers.serverLoginSubAnon', { n })
}

export const loginPill = (t: Translate, r: AgentRuntime) => {
  if (!r.ready) return { cls: 'ag-pill--bad', text: t('agent.providers.status.notReady') }
  if (!r.detectedAt) return { cls: '', text: t('agent.providers.status.unchecked') }
  if (r.loggedIn) return { cls: 'ag-pill--ok', text: t('agent.providers.status.loggedIn') }
  return { cls: 'ag-pill--warn', text: t('agent.providers.status.notLoggedIn') }
}
