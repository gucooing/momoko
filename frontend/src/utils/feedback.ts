// 统一反馈（toast）：封装 Nuxt UI 的 useToast，替代 Element Plus 的 ElMessage（见 docs/redesign/07 约定 §3）。
//
// 两种用法：
//  1) 组件 setup 内：`const fb = useFeedback()`（依赖注入，必须在 <UApp> 子树的 setup 里执行）。
//  2) 任意上下文（axios 拦截器 / store action / util 等非组件）：`import { feedback } from '@/utils/feedback'`
//     使用模块级单例代理。单例由 <UApp> 内的 FeedbackBridge 在挂载时 initFeedbackSingleton() 捕获。
import type { Toast } from '@nuxt/ui/runtime/composables/useToast.js'

type FeedbackColor = 'success' | 'error' | 'warning' | 'info'

export function useFeedback() {
  const toast = useToast()

  const push = (color: FeedbackColor, message: string, extra?: Partial<Toast>) =>
    toast.add({ title: message, color, ...extra })

  return {
    success: (message: string, extra?: Partial<Toast>) => push('success', message, extra),
    error: (message: string, extra?: Partial<Toast>) => push('error', message, extra),
    warning: (message: string, extra?: Partial<Toast>) => push('warning', message, extra),
    info: (message: string, extra?: Partial<Toast>) => push('info', message, extra),
    /** 底层 add：需要持久（duration:0）/后续 remove 的场景用（如粘贴进行中提示）。 */
    add: toast.add,
    remove: toast.remove,
  }
}

type FeedbackApi = ReturnType<typeof useFeedback>
let singleton: FeedbackApi | null = null

/** 在 <UApp> 内的桥组件 setup 里调用一次，捕获 toast 实例供非组件上下文使用。 */
export function initFeedbackSingleton() {
  singleton = useFeedback()
}

/** 全局代理：组件与非组件上下文统一使用（App 挂载后可用；未就绪时静默丢弃，不抛错）。 */
export const feedback = {
  success: (message: string, extra?: Partial<Toast>) => singleton?.success(message, extra),
  error: (message: string, extra?: Partial<Toast>) => singleton?.error(message, extra),
  warning: (message: string, extra?: Partial<Toast>) => singleton?.warning(message, extra),
  info: (message: string, extra?: Partial<Toast>) => singleton?.info(message, extra),
  add: (options: Partial<Toast>) => singleton?.add(options),
  remove: (id: string | number) => singleton?.remove(id),
}
