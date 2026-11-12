// 学マス 的玩法：竞赛、培育、研修、塔。服务端 manifest 的 modes 给出这台服务器配置了哪些。
import type { Modes } from '@/types/v1/gakumas/types'

export type GakumasMode = keyof Modes

export interface ModeDef {
  id: GakumasMode
  title: string
}

export const gakumasModes: ModeDef[] = [
  { id: 'pvp', title: '竞赛' },
  { id: 'produce', title: '培育' },
  { id: 'seminar', title: '研修' },
  { id: 'tower', title: '塔' },
]

/** 这台服务器配置了的玩法（按固定顺序）。 */
export const enabledModes = (modes: Modes | undefined): ModeDef[] => gakumasModes.filter((m) => !!modes?.[m.id])
