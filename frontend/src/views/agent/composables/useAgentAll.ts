// 「Agent 管理」页（/agent/admin）和「Agent 设置」页用同一套会话、工作区、MCP、技能列表：
// Agent 管理页在外层 provideAgentAll()，里面的列表就看所有人的，只留清理（归档、删除、导出、看进程），不能打开或修改别人的。
import type { InjectionKey } from 'vue'

const KEY: InjectionKey<boolean> = Symbol('agentAll')

/** Agent 管理页的外层调用：下面的列表都看所有人的 */
export const provideAgentAll = () => provide(KEY, true)

/** 当前列表是不是 Agent 管理页里的（要在组件的 setup 里调用） */
export const useAgentAll = () => inject(KEY, false)
