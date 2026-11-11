// 组件尺寸注册表：每个组件在这里登记自己的尺寸变量（CSS 变量 --cmp-<key>），组件样式里用 var(--cmp-<key>) 取值。
// 主题编辑器的「组件尺寸」完全由这张表生成：新组件只要在这里加一组（并在样式里用上变量）就会出现在编辑器里，
// 编辑器本身不登记任何组件。selector 用来在预览里「点选元素」时认出是哪个组件，并用虚线框标出它在哪。
// 名称走 i18n：theme.sizeGroups.<group>、theme.sizes.<key>（没有翻译时显示键名）。
//
// unit：px（长度）/ font（字号，按 px 编辑，输出为 rem，跟随「字号 小 / 标准 / 大」缩放）/ scale（百分比缩放，输出为倍数）

export type SizeUnit = 'px' | 'font' | 'scale'

export interface SizeToken {
  key: string
  default: number
  min: number
  max: number
  step?: number
  unit: SizeUnit
}

export interface SizeGroup {
  key: string
  /** 预览里点到哪些元素算这个组件（closest 匹配） */
  selector: string
  /**
   * 还把「看起来是卡片」的元素也算进来：背景是卡片色（--m-s，含毛玻璃）、圆角 ≥ 12px 的块。
   * 各页面自己写的卡片不用登记，padding 用 calc(var(--cmp-card-unit) * N) 就会跟着「卡片 · 内边距」变。
   */
  surface?: boolean
  tokens: SizeToken[]
}

const px = (key: string, def: number, min: number, max: number, step = 1): SizeToken => ({ key, default: def, min, max, step, unit: 'px' })
const font = (key: string, def: number, min: number, max: number): SizeToken => ({ key, default: def, min, max, step: 0.5, unit: 'font' })
const scale = (key: string, def: number, min: number, max: number): SizeToken => ({ key, default: def, min, max, step: 5, unit: 'scale' })

export const SIZE_GROUPS: SizeGroup[] = [
  {
    key: 'layout',
    selector: '.app-content',
    tokens: [px('page-padding-x', 32, 8, 80), px('page-padding-top', 24, 0, 64)],
  },
  {
    key: 'sidebar',
    selector: '.app-sidebar',
    tokens: [px('sidebar-width', 252, 200, 360), px('nav-item-height', 42, 32, 60), font('nav-font-size', 14, 12, 18), px('nav-icon-size', 20, 14, 28)],
  },
  {
    key: 'topbar',
    selector: '.app-topbar',
    tokens: [px('topbar-height', 68, 48, 100), px('search-width', 520, 240, 800, 10)],
  },
  {
    key: 'card',
    selector: '.app-panel, .entity-card',
    surface: true,
    tokens: [px('card-padding', 22, 8, 44), font('card-title-size', 17, 13, 26), px('card-border-width', 1, 0, 4)],
  },
  {
    key: 'table',
    selector: '.data-table',
    tokens: [px('table-row-padding', 12, 4, 24), font('table-font-size', 14, 11, 18), font('table-head-size', 12, 10, 16)],
  },
  {
    key: 'button',
    selector: '.app-ui-btn, .dash-hero__btn',
    tokens: [scale('button-scale', 100, 80, 140)],
  },
  {
    key: 'iconButton',
    selector: '.app-icon-btn, .action-menu__trigger',
    tokens: [scale('icon-button-scale', 100, 80, 140)],
  },
  {
    key: 'input',
    selector: '.app-input, .app-select',
    tokens: [px('input-height', 38, 30, 56), font('input-font-size', 14, 12, 18)],
  },
  {
    key: 'pill',
    selector: '.status-pill',
    tokens: [font('pill-font-size', 12, 10, 16), px('pill-padding-x', 10, 4, 20)],
  },
  {
    key: 'subnav',
    selector: '.subnav',
    tokens: [px('subnav-height', 44, 32, 60), font('subnav-font-size', 15, 12, 20)],
  },
  {
    key: 'pageHeader',
    selector: '.page-header',
    tokens: [font('page-title-size', 30, 20, 44)],
  },
  {
    key: 'hero',
    selector: '.dash-hero',
    tokens: [px('hero-min-height', 248, 120, 420, 4), px('hero-padding', 36, 16, 64), font('hero-title-size', 38, 24, 60)],
  },
  {
    key: 'tabbar',
    selector: '.tabbar',
    tokens: [px('tabbar-height', 56, 46, 76), px('tabbar-icon-size', 20, 16, 28)],
  },
]

export const SIZE_TOKENS: SizeToken[] = SIZE_GROUPS.flatMap((g) => g.tokens)
const BY_KEY = new Map(SIZE_TOKENS.map((t) => [t.key, t]))
export const sizeToken = (key: string) => BY_KEY.get(key)
export const groupOfSize = (key: string) => SIZE_GROUPS.find((g) => g.tokens.some((t) => t.key === key))
export const sizeGroup = (key: string) => SIZE_GROUPS.find((g) => g.key === key)

export const SIZE_KEY_RE = /^[a-z][a-z0-9-]{0,47}$/

/** 编辑值 → CSS 值 */
export const sizeCssValue = (t: SizeToken, v: number) =>
  t.unit === 'font' ? `${Math.round((v / 16) * 10000) / 10000}rem` : t.unit === 'scale' ? `${v / 100}` : `${v}px`

/** 显示用的值（px / %） */
export const sizeDisplay = (t: SizeToken, v: number) => (t.unit === 'scale' ? `${v}%` : `${v} px`)

export const clampSize = (t: SizeToken, v: number) => Math.min(t.max, Math.max(t.min, v))

/** 全部默认值（engine 在启动时写到 :root 上） */
export const sizeDefaultsCss = () =>
  `:root { ${SIZE_TOKENS.map((t) => `--cmp-${t.key}: ${sizeCssValue(t, t.default)};`).join(' ')} }`

/** 覆盖值 → CSS 变量声明（未登记的键忽略：可能来自更新版本的主题包） */
export const sizesToCssVars = (sizes: Record<string, number>) =>
  Object.entries(sizes)
    .map(([k, v]) => {
      const t = BY_KEY.get(k)
      return t && Number.isFinite(v) ? `--cmp-${k}: ${sizeCssValue(t, clampSize(t, v))};` : ''
    })
    .filter(Boolean)
    .join(' ')
