// 玩家分页：标签来自服务端 manifest 的 views（顺序与标题由服务端给出），这里只把已知的视图 id
// 对应到专门的界面；不认识的视图用通用渲染（表格 / 键值）。Momoko 自己的「GM 记录」固定排在最后。
import { defineAsyncComponent, type Component } from 'vue'
import type { PlayerDetail } from '@/games/gakumas/types'

export interface PlayerViewDef {
  component: Component
  /** 标签上显示的数量（从玩家概览里取） */
  count?: (p: PlayerDetail) => number | undefined
  /** 这个视图的内容已包含在另一个视图里：两个都有时不单独成标签 */
  mergedInto?: string
}

const pvp = defineAsyncComponent(() => import('./PvpView.vue'))

const DEFS: Record<string, PlayerViewDef> = {
  overview: { component: defineAsyncComponent(() => import('./OverviewView.vue')) },
  idolCards: {
    component: defineAsyncComponent(() => import('./IdolCardsView.vue')),
    count: (p) => p.overview?.idolCardCount,
  },
  supportCards: {
    component: defineAsyncComponent(() => import('./SupportCardsView.vue')),
    count: (p) => p.overview?.supportCardCount,
  },
  items: {
    component: defineAsyncComponent(() => import('./ItemsView.vue')),
    count: (p) => p.overview?.itemCount,
  },
  giftBox: {
    component: defineAsyncComponent(() => import('./GiftBoxView.vue')),
    count: (p) => p.overview?.gifts?.unreceived,
  },
  gashaHistory: { component: defineAsyncComponent(() => import('./GashaHistoryView.vue')) },
  produceHistory: {
    component: defineAsyncComponent(() => import('./ProduceHistoryView.vue')),
    count: (p) => p.overview?.produceCount,
  },
  memories: {
    component: defineAsyncComponent(() => import('./MemoriesView.vue')),
    count: (p) => p.overview?.memoryCount,
  },
  pvp: { component: pvp },
  'pvp-unit': { component: pvp, mergedInto: 'pvp' },
  seminar: { component: defineAsyncComponent(() => import('./SeminarView.vue')) },
  sessions: { component: defineAsyncComponent(() => import('./SessionsView.vue')) },
  backups: { component: defineAsyncComponent(() => import('./BackupsView.vue')) },
}

const generic = defineAsyncComponent(() => import('./GenericView.vue'))

export const gmLogView = defineAsyncComponent(() => import('./GmLogView.vue'))

export function playerViewDef(id: string): PlayerViewDef {
  return DEFS[id] || { component: generic }
}
