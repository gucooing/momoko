// GM 运营的页内标签（/game/gm/:serverId?tab=…）：学マス 的工作台、玩家、卡池、玩法、运营日历、邮件补偿、兑换码、
// 公告、GM 操作（选玩家执行操作、玩家权限）与 GM 记录；玩法标签只在服务端配置了竞赛 / 研修 / 塔等玩法时出现。
// 标签视图放在 ./tabs/<key>/index.vue，按需加载。
import { defineAsyncComponent, type Component } from 'vue'
import { enabledModes } from '@/games/gakumas/modes'
import type { GmContext } from './context'

export interface GmTab {
  key: string
  labelKey: string
  icon: string
  /** 当前服务器是否提供这个标签 */
  visible: (ctx: GmContext) => boolean
}

const views = import.meta.glob('./tabs/*/index.vue')

export const GM_TABS: GmTab[] = [
  { key: 'home', labelKey: 'gm.tabs.home', icon: 'HOutline:PresentationChartLineIcon', visible: () => true },
  { key: 'players', labelKey: 'gm.tabs.players', icon: 'HOutline:UsersIcon', visible: () => true },
  { key: 'gasha', labelKey: 'gm.tabs.gasha', icon: 'HOutline:SparklesIcon', visible: () => true },
  { key: 'modes', labelKey: 'gm.tabs.modes', icon: 'HOutline:TrophyIcon', visible: (c) => enabledModes(c.modes.value).length > 0 },
  { key: 'calendar', labelKey: 'gm.tabs.calendar', icon: 'HOutline:CalendarDaysIcon', visible: () => true },
  { key: 'mail', labelKey: 'gm.tabs.mail', icon: 'HOutline:EnvelopeIcon', visible: () => true },
  { key: 'codes', labelKey: 'gm.tabs.codes', icon: 'HOutline:TicketIcon', visible: () => true },
  { key: 'notices', labelKey: 'gm.tabs.notices', icon: 'HOutline:MegaphoneIcon', visible: () => true },
  {
    key: 'operations',
    labelKey: 'gm.tabs.operations',
    icon: 'HOutline:CommandLineIcon',
    visible: (c) => c.operations.some((op) => op.target === 'player' || op.target === 'players'),
  },
  { key: 'logs', labelKey: 'gm.tabs.logs', icon: 'HOutline:ClockIcon', visible: () => true },
]

const cache = new Map<string, Component>()

/** 标签视图组件（视图文件不存在时返回 undefined）。 */
export function gmTabView(key: string): Component | undefined {
  const loader = views[`./tabs/${key}/index.vue`]
  if (!loader) return undefined
  let c = cache.get(key)
  if (!c) {
    c = defineAsyncComponent(loader as () => Promise<{ default: Component }>)
    cache.set(key, c)
  }
  return c
}
