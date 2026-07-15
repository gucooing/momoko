// 统一反馈（toast）：封装 Nuxt UI 的 useToast，替代 Element Plus 的 ElMessage（见 docs/redesign/07 约定 §3）。
//
// 约束：useToast 依赖注入，必须在组件 setup（或其中调用的 composable）里执行 useFeedback()；
// 返回的方法（success/error/warning/info/add/remove）已绑定 toast 实例，可在事件处理器中安全调用。
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
