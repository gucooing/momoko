// 卡池编辑器的共享状态（卡池页提供，各表单分区注入后直接改草稿对象）。
import { inject, provide, type ComputedRef, type InjectionKey, type Ref } from 'vue'
import type { GashaConfig, GashaDef } from '@/games/gakumas/types'
import type { CatalogLookup } from './shared/catalogLookup'
import type { ConfigDomainState } from './shared/useConfigDomain'

export interface GashaEditor {
  state: ConfigDomainState<GashaConfig>
  /** 草稿内容 */
  cfg: ComputedRef<GashaConfig | undefined>
  /** 正在编辑的卡池 */
  gasha: ComputedRef<GashaDef | undefined>
  /** 服务端当前（已发布）的同一个卡池，新建的为 undefined */
  publishedGasha: ComputedRef<GashaDef | undefined>
  zone: ComputedRef<string>
  now: Ref<number>
  game: ComputedRef<string>
  lookup: CatalogLookup
  /** 卡的稀有度：先看卡组，再看目录 */
  rarityOf: (resourceType: number, id: string) => number | undefined
  /** 卡在游戏目录里的核对状态 */
  cardStatus: (resourceType: number, id: string) => 'ok' | 'missing' | 'pending' | 'error'
}

const KEY: InjectionKey<GashaEditor> = Symbol('gasha-editor')

export function provideGashaEditor(editor: GashaEditor) {
  provide(KEY, editor)
}

export function useGashaEditor(): GashaEditor {
  const editor = inject(KEY)
  if (!editor) throw new Error('useGashaEditor() must be used inside the gasha tab')
  return editor
}
