// GM 运营的页内标签（/game/gm/:serverId?tab=…）。标签是否出现由服务端 manifest 决定：
// 有 dashboard 报表才有工作台，有 gasha 配置域才有卡池，有 mail.broadcast 指令才有邮件补偿……
// 标签视图放在 ./tabs/<key>/index.vue，按需加载。
import { defineAsyncComponent, type Component } from 'vue'
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
  { key: 'home', labelKey: 'gm.tabs.home', icon: 'HOutline:PresentationChartLineIcon', visible: (c) => c.hasReport('dashboard') },
  { key: 'players', labelKey: 'gm.tabs.players', icon: 'HOutline:UsersIcon', visible: (c) => !!c.manifest.value?.capabilities?.includes('players') },
  { key: 'gasha', labelKey: 'gm.tabs.gasha', icon: 'HOutline:SparklesIcon', visible: (c) => c.hasConfig('gasha') },
  { key: 'modes', labelKey: 'gm.tabs.modes', icon: 'HOutline:TrophyIcon', visible: (c) => (c.manifest.value?.modes?.length || 0) > 0 },
  { key: 'calendar', labelKey: 'gm.tabs.calendar', icon: 'HOutline:CalendarDaysIcon', visible: (c) => (c.manifest.value?.configs?.length || 0) > 0 },
  { key: 'mail', labelKey: 'gm.tabs.mail', icon: 'HOutline:EnvelopeIcon', visible: (c) => c.hasCommand('mail.broadcast') },
  { key: 'codes', labelKey: 'gm.tabs.codes', icon: 'HOutline:TicketIcon', visible: (c) => c.hasCommand('gift.send') },
  { key: 'notices', labelKey: 'gm.tabs.notices', icon: 'HOutline:MegaphoneIcon', visible: (c) => c.hasConfig('notice') },
  { key: 'self-service', labelKey: 'gm.tabs.selfService', icon: 'HOutline:IdentificationIcon', visible: (c) => !!c.manifest.value?.commands?.some((cmd) => cmd.selfService) },
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
