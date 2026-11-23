// 模型清单只有一个来源：模型供应商页「检查」服务器上的登录时同步进模型表的，和第三方接口里配置的模型目录，
// 由后端 /agent/options 一起给出。页面上所有选模型的地方用这里的清单，所有显示模型的地方用这里的 modelName，
// 不自己拼名字，也不直接显示存下来的模型 id（清单里找不到时才退回 id，比如供应商已删、登录失效）。
import { getAgentOptions } from '@/api/agent'
import type { AgentModelOption, AgentProviderOptions, AgentRuntimeOptions } from '@/types/v1/agent'

const runtimes = ref<AgentRuntimeOptions[]>([])
const loaded = ref(false)
let seq = 0
let pending: Promise<void> | null = null

/** 重读清单：每次都发请求（配置刚改过时不能用还没回来的旧请求），只采用最后一次的结果 */
const refresh = (): Promise<void> => {
  const id = ++seq
  const p = getAgentOptions()
    .then(({ data }) => {
      if (id === seq) runtimes.value = data.runtimes ?? []
    })
    .catch(() => {
      /* 请求层已经提示过 */
    })
    .finally(() => {
      if (id === seq) {
        loaded.value = true
        pending = null
      }
    })
  pending = p
  return p
}

/** 选的是哪个供应商：空的 providerId 是服务器上的登录 */
export const findProvider = (runtimeId?: string, providerId?: string): AgentProviderOptions | undefined =>
  runtimes.value
    .find((r) => r.runtimeId === runtimeId)
    ?.providers.find((p) => (p.providerId || '') === (providerId || ''))

/** 清单里的这个模型 */
export const findModel = (runtimeId?: string, providerId?: string, model?: string): AgentModelOption | undefined =>
  model ? findProvider(runtimeId, providerId)?.models.find((m) => m.id === model) : undefined

/** 模型的显示名：清单里的名称，没有名称用 id */
export const modelLabel = (m?: Pick<AgentModelOption, 'id' | 'name'> | null, fallback = '') => m?.name || m?.id || fallback

/** 会话、工作区存的模型 id 显示成什么：按它的 Agent 和供应商在清单里找 */
export const modelName = (runtimeId?: string, providerId?: string, model?: string) =>
  modelLabel(findModel(runtimeId, providerId, model), model || '')

/** 推理强度的显示名：这个模型在清单里声明的名称 */
export const effortName = (runtimeId?: string, providerId?: string, model?: string, effort?: string) =>
  (effort && findModel(runtimeId, providerId, model)?.efforts.find((e) => e.id === effort)?.name) || effort || ''

/** 在组件里用：还没读过就读一次。改了配置的页面（或收到配置变化推送）调用 refresh() */
export const useAgentModels = () => {
  if (!loaded.value && !pending) void refresh()
  return { runtimes, loaded, refresh }
}
