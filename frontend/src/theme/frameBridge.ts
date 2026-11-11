// 预览框里的一端：接收编辑器送来的草稿主题并套用；「点选元素」开着时拦截点击，
// 找出被点元素用到的 token 回传给编辑器；并给当前 token 用到的元素加虚线框。
import type { Router } from 'vue-router'
import { useThemeStore } from '@/stores/theme'
import { IS_THEME_PREVIEW_FRAME, isToFrameMessage, type FromFrameMessage } from './frame'
import { hexToRgb, parseCssColor, rgbDistance, type RGB } from './color'
import type { TokenKey } from './palettes'
import { pageKeyOf } from './pages'
import { SIZE_GROUPS, sizeGroup } from './sizeTokens'

/** 派生 token 归到编辑器里列出的 token */
const EDITOR_KEY: Partial<Record<TokenKey, string>> = {
  pInk: 'p',
  pSoft: 'p',
  aOn: 'a',
  aSoft: 'a',
  a2Soft: 'a2',
  okSoft: 'ok',
  warnSoft: 'warn',
  badSoft: 'bad',
}
const BG_ORDER: TokenKey[] = ['p', 'a', 'a2', 's', 's2', 'bg', 'termBg', 'pSoft', 'aSoft', 'a2Soft', 'okSoft', 'warnSoft', 'badSoft', 'ok', 'warn', 'bad', 'line', 'ink']
const TEXT_ORDER: TokenKey[] = ['ink', 'ink2', 'pInk', 'p', 'pOn', 'termFg', 'termCursor', 'ok', 'warn', 'bad', 'a', 'a2', 'aOn']
const BORDER_ORDER: TokenKey[] = ['line', 'p', 'a2']

const HOT = 'theme-pick-hot'
/** 页面里有终端（应用控制台、SSH、容器终端…）时编辑器显示终端专用 token；按真实 DOM 判断，不按页面登记 */
const TERMINAL_SELECTOR = '.xterm, .term-shell'
const MAX_MARKS = 240

const post = (msg: FromFrameMessage) => window.parent.postMessage(msg, window.location.origin)

export const installThemeFrameBridge = (router: Router) => {
  if (!IS_THEME_PREVIEW_FRAME) return
  const themeStore = useThemeStore()
  let picking = false
  let pickMode: 'color' | 'part' = 'color'
  let highlight: string | null = null
  let highlightPart: string | null = null
  let timer: ReturnType<typeof setTimeout> | null = null

  const tokenRgb = (key: TokenKey): RGB | null => {
    const hex = themeStore.tokens[key]
    return hex ? hexToRgb(hex) : null
  }
  const match = (value: string, order: TokenKey[]): string | null => {
    const c = parseCssColor(value)
    if (!c || c.alpha < 0.05) return null
    for (const key of order) {
      const rgb = tokenRgb(key)
      if (rgb && rgbDistance(rgb, c.rgb) <= 3) return EDITOR_KEY[key] || key
    }
    return null
  }
  const hasOwnText = (el: Element) =>
    Array.from(el.childNodes).some((n) => n.nodeType === Node.TEXT_NODE && (n.textContent || '').trim())

  const pickToken = (target: Element): string | null => {
    const special = target.closest('[data-theme-pick]')
    if (special) return special.getAttribute('data-theme-pick')
    const own = getComputedStyle(target)
    if (hasOwnText(target)) {
      const t = match(own.color, TEXT_ORDER)
      if (t) return t
    }
    let el: Element | null = target
    for (let depth = 0; el && depth < 8; depth++, el = el.parentElement) {
      const cs = getComputedStyle(el)
      const bg = match(cs.backgroundColor, BG_ORDER)
      if (bg) return bg
    }
    return match(own.color, TEXT_ORDER) || match(own.borderTopColor, BORDER_ORDER)
  }

  /** 「看起来是卡片」：背景是卡片色（忽略透明度，毛玻璃也算）、圆角够大、不是很小的块 */
  const isSurface = (el: Element) => {
    const cs = getComputedStyle(el)
    const c = parseCssColor(cs.backgroundColor)
    const s = tokenRgb('s')
    if (!c || c.alpha < 0.3 || !s || rgbDistance(c.rgb, s) > 3) return false
    if (parseFloat(cs.borderTopLeftRadius) < 12) return false
    const r = el.getBoundingClientRect()
    return r.width >= 120 && r.height >= 48
  }
  const SURFACE_GROUP = SIZE_GROUPS.find((g) => g.surface)

  /** 点到的是哪个组件：从被点元素往上找，最近一个匹配注册表 selector（或卡片外观）的分组 */
  const pickPart = (target: Element): string | null => {
    for (let el: Element | null = target; el && el !== document.body; el = el.parentElement) {
      const g = SIZE_GROUPS.find((x) => el!.matches(x.selector))
      if (g) return g.key
      if (SURFACE_GROUP && isSurface(el)) return SURFACE_GROUP.key
    }
    return null
  }

  const clearMarks = () => document.querySelectorAll(`.${HOT}`).forEach((n) => n.classList.remove(HOT))

  const markUsages = () => {
    clearMarks()
    if (highlightPart) {
      const g = sizeGroup(highlightPart)
      if (!g) return
      let n = 0
      document.querySelectorAll(g.selector).forEach((el) => n++ < MAX_MARKS && el.classList.add(HOT))
      if (g.surface) {
        for (const el of document.body.querySelectorAll('body *')) {
          if (n >= MAX_MARKS) break
          if (!el.classList.contains(HOT) && isSurface(el) && !el.parentElement?.closest(`.${HOT}`)) {
            el.classList.add(HOT)
            n++
          }
        }
      }
      return
    }
    if (!highlight) return
    if (highlight === 'hero') {
      document.querySelectorAll('[data-theme-pick="hero"]').forEach((n) => n.classList.add(HOT))
      return
    }
    const keys = (Object.keys(themeStore.tokens) as TokenKey[]).filter((k) => k === highlight)
    const rgb = keys.length ? tokenRgb(keys[0]!) : null
    if (!rgb) return
    let n = 0
    const all = document.body.querySelectorAll<HTMLElement>('body *')
    for (const el of all) {
      if (n >= MAX_MARKS) break
      if (el.closest('svg') && el.tagName.toLowerCase() !== 'svg') continue
      const rect = el.getBoundingClientRect()
      if (rect.width < 4 || rect.height < 4 || rect.bottom < 0 || rect.top > window.innerHeight) continue
      const cs = getComputedStyle(el)
      const near = (v: string) => {
        const c = parseCssColor(v)
        return !!c && c.alpha >= 0.05 && rgbDistance(c.rgb, rgb) <= 3
      }
      const parentBg = el.parentElement ? getComputedStyle(el.parentElement).backgroundColor : ''
      if (
        (near(cs.backgroundColor) && cs.backgroundColor !== parentBg) ||
        (hasOwnText(el) && near(cs.color)) ||
        (cs.borderTopStyle !== 'none' && parseFloat(cs.borderTopWidth) > 0 && near(cs.borderTopColor) && !near(parentBg))
      ) {
        el.classList.add(HOT)
        n++
      }
    }
  }
  const scheduleMarks = (delay = 160) => {
    if (timer) clearTimeout(timer)
    timer = setTimeout(markUsages, delay)
  }

  window.addEventListener('message', (e) => {
    if (e.origin !== window.location.origin || !isToFrameMessage(e.data)) return
    const d = e.data
    if (d.type === 'momoko-theme:draft') {
      themeStore.setFrameDraft({ content: d.content, scheme: d.scheme, pageKey: d.pageKey })
      picking = d.picking
      pickMode = d.pickMode
      highlight = d.highlight
      highlightPart = d.highlightPart
      document.documentElement.classList.toggle('theme-picking', picking)
      scheduleMarks()
    } else if (d.type === 'momoko-theme:navigate') {
      void router.replace(d.path)
    }
  })

  const swallow = (e: Event) => {
    if (!picking) return
    e.preventDefault()
    e.stopPropagation()
  }
  document.addEventListener(
    'click',
    (e) => {
      if (!picking) return
      e.preventDefault()
      e.stopPropagation()
      const target = e.target instanceof Element ? e.target : null
      if (!target) return
      if (pickMode === 'part') {
        const part = pickPart(target)
        if (part) post({ type: 'momoko-theme:pick', part })
        return
      }
      const token = pickToken(target)
      if (token) post({ type: 'momoko-theme:pick', token })
    },
    true,
  )
  for (const type of ['mousedown', 'pointerdown', 'submit', 'dblclick', 'contextmenu']) document.addEventListener(type, swallow, true)

  // 当前页面（路由模板）与是否含终端；变化时告诉编辑器
  let lastRoute = ''
  const reportRoute = () => {
    const to = router.currentRoute.value
    const key = pageKeyOf(to.matched[to.matched.length - 1]?.path)
    const terminal = !!document.querySelector(TERMINAL_SELECTOR)
    const sig = `${to.fullPath}|${key}|${terminal}`
    if (sig === lastRoute) return
    lastRoute = sig
    post({ type: 'momoko-theme:route', path: to.fullPath, key, terminal })
  }
  let routeTimer: ReturnType<typeof setTimeout> | null = null
  const scheduleRoute = (delay = 300) => {
    if (routeTimer) clearTimeout(routeTimer)
    routeTimer = setTimeout(reportRoute, delay)
  }

  new MutationObserver(() => {
    if (highlight || highlightPart) scheduleMarks(400)
    scheduleRoute(500)
  }).observe(document.body, { childList: true, subtree: true })
  window.addEventListener('scroll', () => (highlight || highlightPart) && scheduleMarks(200), true)

  router.afterEach(() => {
    scheduleRoute(0)
    scheduleMarks(500)
  })
  post({ type: 'momoko-theme:ready' })
  void router.isReady().then(() => scheduleRoute(0))
}
