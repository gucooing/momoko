// 学マス 玩家详情的分页：每个分页调用对应的接口，Momoko 自己的「GM 记录」固定排在最后。
import { defineAsyncComponent, type Component } from 'vue'
import type { GakumasMode } from '@/games/gakumas/modes'
import type { PlayerDetail } from '@/games/gakumas/types'

export interface PlayerViewDef {
  id: string
  /** 标签标题（i18n key） */
  titleKey: string
  component: Component
  /** 标签上显示的数量（从玩家概览里取） */
  count?: (p: PlayerDetail) => number | undefined
  /** 只在服务器配置了这个玩法时出现 */
  mode?: GakumasMode
}

export const PLAYER_VIEWS: PlayerViewDef[] = [
  { id: 'overview', titleKey: 'gmData.views.overview', component: defineAsyncComponent(() => import('./OverviewView.vue')) },
  {
    id: 'idolCards',
    titleKey: 'gmData.views.idolCards',
    component: defineAsyncComponent(() => import('./IdolCardsView.vue')),
    count: (p) => p.overview?.idolCardCount,
  },
  {
    id: 'supportCards',
    titleKey: 'gmData.views.supportCards',
    component: defineAsyncComponent(() => import('./SupportCardsView.vue')),
    count: (p) => p.overview?.supportCardCount,
  },
  {
    id: 'items',
    titleKey: 'gmData.views.items',
    component: defineAsyncComponent(() => import('./ItemsView.vue')),
    count: (p) => p.overview?.itemCount,
  },
  {
    id: 'giftBox',
    titleKey: 'gmData.views.giftBox',
    component: defineAsyncComponent(() => import('./GiftBoxView.vue')),
    count: (p) => p.overview?.gifts?.unreceived,
  },
  { id: 'gashaHistory', titleKey: 'gmData.views.gashaHistory', component: defineAsyncComponent(() => import('./GashaHistoryView.vue')) },
  {
    id: 'produceHistory',
    titleKey: 'gmData.views.produceHistory',
    component: defineAsyncComponent(() => import('./ProduceHistoryView.vue')),
    count: (p) => p.overview?.produceCount,
  },
  {
    id: 'memories',
    titleKey: 'gmData.views.memories',
    component: defineAsyncComponent(() => import('./MemoriesView.vue')),
    count: (p) => p.overview?.memoryCount,
  },
  { id: 'pvp', titleKey: 'gmData.views.pvp', component: defineAsyncComponent(() => import('./PvpView.vue')), mode: 'pvp' },
  { id: 'seminar', titleKey: 'gmData.views.seminar', component: defineAsyncComponent(() => import('./SeminarView.vue')), mode: 'seminar' },
  { id: 'sessions', titleKey: 'gmData.views.sessions', component: defineAsyncComponent(() => import('./SessionsView.vue')) },
  { id: 'backups', titleKey: 'gmData.views.backups', component: defineAsyncComponent(() => import('./BackupsView.vue')) },
]

export const gmLogView = defineAsyncComponent(() => import('./GmLogView.vue'))
