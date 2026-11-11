// 主题编辑器的预览框：编辑器用 <iframe> 加载真实页面（带 ?__theme_preview=1），
// 通过 postMessage 把草稿主题送进去；框内的应用只套用草稿、不保存任何外观设置，
// 「点选元素」时拦截点击，把元素用到的 token 回传给编辑器，并用虚线框标出这个 token 用在哪。
import type { ThemeContent } from '@/types/v1/theme'

export const THEME_PREVIEW_PARAM = '__theme_preview'

/** 当前页面是否运行在编辑器的预览框里（启动时判断一次，框内跳转后仍然有效） */
export const IS_THEME_PREVIEW_FRAME = (() => {
  if (typeof window === 'undefined') return false
  try {
    return window.self !== window.top && new URLSearchParams(window.location.search).has(THEME_PREVIEW_PARAM)
  } catch {
    return false
  }
})()

/** 编辑器 → 预览框 */
export type ToFrameMessage =
  | {
      type: 'momoko-theme:draft'
      content: ThemeContent
      scheme: 'light' | 'dark'
      /** 草稿以哪个页面的身份预览（页面单独设置按它取） */
      pageKey: string
      picking: boolean
      /** color：点选颜色 token；part：点选组件（组件尺寸） */
      pickMode: 'color' | 'part'
      /** 需要虚线框标出的 token（hero = 横幅） */
      highlight: string | null
      /** 需要虚线框标出的组件（theme/sizeTokens.ts 里的分组键） */
      highlightPart: string | null
    }
  | { type: 'momoko-theme:navigate'; path: string }

/** 预览框 → 编辑器 */
export type FromFrameMessage =
  | { type: 'momoko-theme:ready' }
  | { type: 'momoko-theme:pick'; token?: string; part?: string }
  /** 预览框里的页面变了（编辑器在预览里点进其它页面时跟过去）；terminal = 页面里有终端，编辑器显示终端专用 token */
  | { type: 'momoko-theme:route'; path: string; key: string; terminal: boolean }

export const isToFrameMessage = (d: unknown): d is ToFrameMessage =>
  !!d && typeof d === 'object' && typeof (d as { type?: unknown }).type === 'string' && (d as { type: string }).type.startsWith('momoko-theme:')
