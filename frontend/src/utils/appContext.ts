import type { AppContext } from 'vue'

// 主应用的上下文（包含已安装的插件：i18n、pinia、router、自定义指令等）
let appContext: AppContext | null = null

/**
 * 保存主应用的上下文。
 * 在 main.ts 创建应用后调用，供命令式渲染（如 utils/dialog.ts 通过 render() 挂载组件）复用，
 * 否则命令式渲染出的组件无法访问 i18n 等插件（useI18n 会因取不到实例而报错）。
 */
export const setAppContext = (context: AppContext) => {
  appContext = context
}

/** 获取主应用上下文，用于给命令式渲染的 vnode 注入插件依赖 */
export const getAppContext = () => appContext
