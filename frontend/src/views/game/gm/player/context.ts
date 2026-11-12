// 玩家详情页的上下文：当前玩家、通行证绑定、执行操作与切换分页的入口。各分页视图用 usePlayerPage() 取用。
import { inject, provide, type InjectionKey, type Ref } from 'vue'
import type { PlayerDetail } from '@/games/gakumas/types'
import type { GmPlayerLink } from '@/types/v1/game_gm'

export interface PlayerPage {
  pid: Ref<string>
  /** 玩家条目、概览与登录次数 */
  player: Ref<PlayerDetail | undefined>
  link: Ref<GmPlayerLink | undefined>
  /** 涉及该玩家的 GM 记录数 */
  gmCount: Ref<number>
  /** 每次真实执行操作后加一，分页视图据此重新读取 */
  version: Ref<number>
  /** 用操作对话框执行一个操作（目标为当前玩家） */
  run: (operation: string, initialParams?: Record<string, unknown>, summary?: Record<string, string>) => void
  /** 打开「发送礼物」抽屉 */
  openGift: () => void
  /** 切换到某个分页（分页 id 或 gmlog） */
  goto: (tab: string) => void
  hasTab: (tab: string) => boolean
  /** 重新读取玩家摘要 */
  refresh: () => Promise<void>
}

const KEY: InjectionKey<PlayerPage> = Symbol('gm-player-page')

export function providePlayerPage(page: PlayerPage) {
  provide(KEY, page)
  return page
}

export function usePlayerPage(): PlayerPage {
  const page = inject(KEY)
  if (!page) throw new Error('usePlayerPage() must be used inside the player page')
  return page
}

/** 玩家分页的固定标签：Momoko 自己的 GM 记录。 */
export const GM_LOG_TAB = 'gmlog'
