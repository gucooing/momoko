// 在线主题编辑器的状态（PC / 手机两种布局共用，provide / inject）：
// 草稿（名称 + 主题内容）、改动作用于「全局 / 仅本页」、撤销 / 重做、草稿自动保存到本机、对比度检查与一键修正、保存并使用。
import { computed, inject, provide, ref, watch, type InjectionKey, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import type { Theme, ThemeBackground, ThemeBanner, ThemeContent, ThemePageOverride, ThemeShape } from '@/types/v1/theme'
import { createTheme, getTheme, updateTheme } from '@/api/theme'
import { useThemeStore } from '@/stores/theme'
import { feedback } from '@/utils/feedback'
import {
  GLASS_TYPES,
  assetUrl,
  backdropBrightColor,
  brightestStop,
  cloneContent,
  contrastChecks,
  dimForContrast,
  effectiveDim,
  emptyContent,
  emptyPage,
  fixContrast,
  isPageEmpty,
  normalizeBackground,
  normalizeBanner,
  normalizeContent,
  pageOverrideCount,
  resolveBackground,
  resolveBanner,
  resolveTokens,
  type BgType,
} from '@/theme/content'
import { EDITOR_TOKENS, PALETTE_TOKENS, TERM_DEFAULTS, isPaletteKey, type SchemeKey, type TokenKey } from '@/theme/palettes'
import { contrastRatio, ensureContrast, hexToHsv, hsvToHex, luminance, mixHex } from '@/theme/color'
import { imageStats } from '@/theme/imageStats'
import { useThemePages } from '@/theme/pages'
import { THEME_PREVIEW_PARAM } from '@/theme/frame'
import { clampSize, sizeDisplay, sizeGroup, sizeToken, groupOfSize } from '@/theme/sizeTokens'

export type EditorTab = 'color' | 'bg' | 'shape' | 'comp'
type TokensField = 'tokensLight' | 'tokensDark'

const SUGGEST = {
  accent: ['#D6245F', '#FF4F8B', '#E2502A', '#FFC93C', '#7B5CFF', '#3450FF', '#00A884', '#0F7F72'],
  surfL: ['#FFFFFF', '#FFF6F9', '#FFF0F5', '#F6F1FF', '#F2F7F5', '#F6DCE6', '#ECE6F5', '#FAF7F2'],
  surfD: ['#140E17', '#1E1622', '#271D2C', '#33263A', '#0E1116', '#171B22', '#1B1430', '#22160E'],
  text: ['#2A1630', '#1B2333', '#11163A', '#6E5A72', '#5F6878', '#FFFFFF', '#F8EEF4', '#B9A6BE'],
}
const ACCENT_KEYS = ['p', 'a', 'a2', 'ok', 'warn', 'bad', 'termCursor']
const TEXT_KEYS = ['ink', 'ink2', 'pOn', 'termFg']

const DRAFT_PREFIX = 'theme-editor-draft:'
const HISTORY_LIMIT = 120
const MERGE_MS = 600

interface Snapshot {
  name: string
  content: ThemeContent
}

/** 两份主题内容之间有多少处不同（「N 处改动未保存」） */
const diffCount = (a: ThemeContent, b: ThemeContent) => {
  let n = 0
  const tok = (x: Record<string, string>, y: Record<string, string>) => {
    for (const k of new Set([...Object.keys(x), ...Object.keys(y)])) if (x[k] !== y[k]) n++
  }
  const same = (x: unknown, y: unknown) => JSON.stringify(x ?? null) === JSON.stringify(y ?? null)
  const num = (x: Record<string, number> = {}, y: Record<string, number> = {}) => {
    for (const k of new Set([...Object.keys(x), ...Object.keys(y)])) if (x[k] !== y[k]) n++
  }
  if (a.base !== b.base) n++
  num(a.sizes, b.sizes)
  tok(a.tokensLight, b.tokensLight)
  tok(a.tokensDark, b.tokensDark)
  if (!same(a.background, b.background)) n++
  if (!same(a.banner, b.banner)) n++
  for (const k of Object.keys({ ...a.shape, ...b.shape }) as (keyof ThemeShape)[]) if (!same(a.shape?.[k], b.shape?.[k])) n++
  for (const key of new Set([...Object.keys(a.pages), ...Object.keys(b.pages)])) {
    const pa = a.pages[key] || emptyPage()
    const pb = b.pages[key] || emptyPage()
    tok(pa.tokensLight, pb.tokensLight)
    tok(pa.tokensDark, pb.tokensDark)
    if (!same(pa.background, pb.background)) n++
    if (!same(pa.cardOpacity, pb.cardOpacity)) n++
    if (!same(pa.banner, pb.banner)) n++
    num(pa.sizes, pb.sizes)
  }
  return n
}

const readDraft = (key: string): Snapshot | null => {
  try {
    const raw = localStorage.getItem(DRAFT_PREFIX + key)
    if (!raw) return null
    const d = JSON.parse(raw) as Snapshot
    return d?.content ? { name: d.name || '', content: normalizeContent(d.content) } : null
  } catch {
    return null
  }
}
const writeDraft = (key: string, snap: Snapshot | null) => {
  try {
    if (snap) localStorage.setItem(DRAFT_PREFIX + key, JSON.stringify(snap))
    else localStorage.removeItem(DRAFT_PREFIX + key)
  } catch {
    /* 配额 / 隐私模式 */
  }
}

const createThemeEditor = (routeId: Ref<string>) => {
  const { t, te } = useI18n()
  const router = useRouter()
  const themeStore = useThemeStore()
  const { pages, labelOf } = useThemePages()

  const loading = ref(true)
  const notFound = ref(false)
  const saving = ref(false)
  const themeId = ref('')
  const name = ref('')
  const content = ref<ThemeContent>(emptyContent())
  const saved = ref<Snapshot>({ name: '', content: emptyContent() })

  const scheme = ref<SchemeKey>(themeStore.scheme)
  const device = ref<'pc' | 'mobile'>('pc')
  const tab = ref<EditorTab>('color')
  // 页面：清单来自路由表（theme/pages.ts），默认第一个菜单页面；预览里点进别的页面时编辑器跟过去
  const firstPage = () => pages.value.find((p) => !p.params)?.key ?? '/'
  const pageKey = ref(firstPage())
  /** 预览框实际打开的地址（带参数的页面，如应用详情，要在预览里点进去） */
  const pagePath = ref(pageKey.value)
  /** 预览里的页面含终端（按真实页面判断） */
  const frameTerminal = ref(false)
  const scopeChoice = ref<'all' | 'page' | null>(null)
  const sel = ref<string>('p')
  const picking = ref(true)
  /** 这次打开过的页面（显示在页面标签里） */
  const extraPages = ref<string[]>([])

  const tokensField = computed<TokensField>(() => (scheme.value === 'dark' ? 'tokensDark' : 'tokensLight'))
  const pageName = computed(() => labelOf(pageKey.value))

  // ---- 撤销 / 重做 ----
  const undoStack = ref<string[]>([])
  const redoStack = ref<string[]>([])
  let lastPush = 0
  let lastMergeKey = ''
  const snapshot = (): string => JSON.stringify({ name: name.value, content: content.value })
  const restore = (json: string) => {
    const s = JSON.parse(json) as Snapshot
    name.value = s.name
    content.value = normalizeContent(s.content)
    scheduleDraft()
  }
  const pushHistory = (before: string, mergeKey?: string) => {
    const now = Date.now()
    const merge = !!mergeKey && mergeKey === lastMergeKey && now - lastPush < MERGE_MS
    if (!merge) {
      undoStack.value.push(before)
      if (undoStack.value.length > HISTORY_LIMIT) undoStack.value.shift()
    }
    lastPush = now
    lastMergeKey = mergeKey || ''
    redoStack.value = []
  }
  /** 改一次内容（mergeKey 相同且间隔很短的连续改动，如拖动滑块，合并成一步撤销） */
  const commit = (mutate: (c: ThemeContent) => void, mergeKey?: string) => {
    const before = snapshot()
    const next = cloneContent(content.value)
    mutate(next)
    for (const k of Object.keys(next.pages)) if (isPageEmpty(next.pages[k])) delete next.pages[k]
    content.value = normalizeContent(next)
    pushHistory(before, mergeKey)
    scheduleDraft()
  }
  const setName = (v: string) => {
    const before = snapshot()
    name.value = v.slice(0, 40)
    pushHistory(before, 'name')
    scheduleDraft()
  }
  const undo = () => {
    const prev = undoStack.value.pop()
    if (!prev) return
    redoStack.value.push(snapshot())
    lastMergeKey = ''
    restore(prev)
  }
  const redo = () => {
    const next = redoStack.value.pop()
    if (!next) return
    undoStack.value.push(snapshot())
    lastMergeKey = ''
    restore(next)
  }
  const canUndo = computed(() => undoStack.value.length > 0)
  const canRedo = computed(() => redoStack.value.length > 0)

  // ---- 草稿自动保存（本机） ----
  let draftTimer: ReturnType<typeof setTimeout> | null = null
  const draftKey = () => themeId.value || 'new'
  const scheduleDraft = () => {
    if (draftTimer) clearTimeout(draftTimer)
    draftTimer = setTimeout(() => {
      draftTimer = null
      if (loading.value) return
      writeDraft(draftKey(), changeCount.value ? { name: name.value, content: content.value } : null)
    }, 500)
  }

  const changeCount = computed(
    () => diffCount(saved.value.content, content.value) + (name.value.trim() !== saved.value.name ? 1 : 0),
  )

  // ---- 色值 ----
  const page = computed<ThemePageOverride | undefined>(() => content.value.pages[pageKey.value])
  /** 已有单独设置的页面，默认「仅本页」，否则「全局」 */
  const scope = computed<'all' | 'page'>(() => scopeChoice.value ?? (page.value ? 'page' : 'all'))
  const isTermPage = computed(() => {
    if (frameTerminal.value) return true
    const p = page.value
    return !!p && [...Object.keys(p.tokensLight), ...Object.keys(p.tokensDark)].some((k) => k.startsWith('term') || k.startsWith('ansi'))
  })
  const baseTokens = computed(() => ({ ...PALETTE_TOKENS[isPaletteKey(content.value.base) ? content.value.base : 'sakura'][scheme.value], ...TERM_DEFAULTS }))
  const globalTokens = computed(() => resolveTokens(content.value, scheme.value, null))
  const tokens = computed(() => resolveTokens(content.value, scheme.value, pageKey.value))
  const tokenValue = (key: string) => tokens.value[key as TokenKey] || '#000000'
  const isPageToken = (key: string) => !!page.value?.[tokensField.value][key]
  const isEdited = (key: string) => tokenValue(key).toUpperCase() !== (baseTokens.value[key as TokenKey] || '').toUpperCase()

  const setToken = (key: string, hex: string, mergeKey = `tok:${key}`) =>
    commit((c) => {
      const field = tokensField.value
      if (scope.value === 'page') {
        const p = (c.pages[pageKey.value] ||= emptyPage())
        p[field][key] = hex
      } else if (hex.toUpperCase() === (baseTokens.value[key as TokenKey] || '').toUpperCase()) {
        delete c[field][key]
      } else {
        c[field][key] = hex
      }
    }, mergeKey)

  /** 恢复原色：有本页覆盖先去掉本页的，否则去掉全局的 */
  const revertToken = (key: string) =>
    commit((c) => {
      const field = tokensField.value
      const p = c.pages[pageKey.value]
      if (p?.[field][key]) delete p[field][key]
      else delete c[field][key]
    })

  const isSizeSel = computed(() => sel.value.startsWith('size:'))
  const selName = computed(() => {
    if (sel.value === 'hero') return t('theme.tokens.hero.name')
    if (isSizeSel.value) return t('theme.editor.sizeOf', { name: sizeGroupLabel(sel.value.slice(5)) })
    if (sel.value.startsWith('ansi')) return `ANSI ${sel.value.slice(4)}`
    return t(`theme.tokens.${sel.value}.name`)
  })
  const selHex = computed(() => (sel.value === 'hero' || isSizeSel.value ? tokenValue('p') : tokenValue(sel.value)))
  const selBase = computed(() => baseTokens.value[sel.value as TokenKey] || '')
  const selEdited = computed(() => sel.value !== 'hero' && !isSizeSel.value && isEdited(sel.value))
  const scopeTag = computed(() =>
    sel.value.startsWith('term') || sel.value.startsWith('ansi')
      ? t('theme.editor.scopeTagTerm')
      : scope.value === 'page'
        ? t('theme.editor.scopeTagPage')
        : t('theme.editor.scopeTagAll'),
  )

  // ---- 背景 / 横幅 / 细节 ----
  const bgModel = computed<ThemeBackground>({
    get: () => (scope.value === 'page' ? (page.value?.background ?? content.value.background!) : content.value.background!),
    set: (bg) =>
      commit((c) => {
        const next = normalizeBackground(bg)
        if (scope.value === 'page') (c.pages[pageKey.value] ||= emptyPage()).background = next
        else c.background = next
      }, 'bg'),
  })
  const bannerModel = computed<ThemeBanner>({
    get: () => (scope.value === 'page' ? (page.value?.banner ?? content.value.banner!) : content.value.banner!),
    set: (b) =>
      commit((c) => {
        const next = normalizeBanner(b)
        if (scope.value === 'page') (c.pages[pageKey.value] ||= emptyPage()).banner = next
        else c.banner = next
      }, 'banner'),
  })
  const setShape = (patch: Partial<ThemeShape>, mergeKey = 'shape') =>
    commit((c) => {
      c.shape = { ...c.shape!, ...patch }
    }, mergeKey)

  // ---- 组件尺寸（theme/sizeTokens.ts 注册表；未翻译的组件 / 尺寸显示键名） ----
  const labelOr = (key: string, fallback: string) => (te(key) ? t(key) : fallback)
  const sizeGroupLabel = (g: string) => labelOr(`theme.sizeGroups.${g}`, g)
  const sizeLabel = (k: string) => labelOr(`theme.sizes.${k}`, k)
  const sizeValue = (key: string) => {
    const tk = sizeToken(key)
    const v = page.value?.sizes?.[key] ?? content.value.sizes?.[key] ?? tk?.default ?? 0
    return tk ? clampSize(tk, v) : v
  }
  const globalSizeValue = (key: string) => content.value.sizes?.[key] ?? sizeToken(key)?.default ?? 0
  const isSizeEdited = (key: string) => sizeValue(key) !== (sizeToken(key)?.default ?? sizeValue(key))
  const isPageSize = (key: string) => page.value?.sizes?.[key] != null
  const setSize = (key: string, v: number) =>
    commit((c) => {
      const tk = sizeToken(key)
      const val = tk ? clampSize(tk, v) : v
      if (scope.value === 'page') {
        const p = (c.pages[pageKey.value] ||= emptyPage())
        p.sizes = { ...(p.sizes || {}), [key]: val }
      } else if (tk && val === tk.default) {
        const next = { ...(c.sizes || {}) }
        delete next[key]
        c.sizes = next
      } else {
        c.sizes = { ...(c.sizes || {}), [key]: val }
      }
    }, `size:${key}`)
  /** 恢复：有本页覆盖先去掉本页的，否则去掉全局的 */
  const revertSize = (key: string) =>
    commit((c) => {
      const p = c.pages[pageKey.value]
      if (p?.sizes?.[key] != null) {
        const next = { ...p.sizes }
        delete next[key]
        p.sizes = next
      } else if (c.sizes) {
        const next = { ...c.sizes }
        delete next[key]
        c.sizes = next
      }
    })
  /** 某个组件改过几项尺寸（当前页面生效值与默认值不同） */
  const sizeGroupEdits = (g: string) => (sizeGroup(g)?.tokens || []).filter((tk) => isSizeEdited(tk.key)).length

  // ---- 本页单独设置 ----
  const pageOvCount = computed(() => pageOverrideCount(page.value, scheme.value))
  const pageOvItems = computed(() => {
    const p = page.value
    if (!p) return []
    const items: { id: string; title: string; value: string; swatch: string; drop: () => void }[] = []
    if (p.background) {
      const b = p.background
      items.push({
        id: 'bg',
        title: t('theme.editor.ovBackground'),
        value: t(`theme.bgTypes.${b.type}`),
        swatch: b.type === 'image' ? `url("${assetUrl(b.image)}") center / cover` : b.type === 'solid' ? b.color : 'var(--m-s2)',
        drop: () => commit((c) => void delete c.pages[pageKey.value]!.background),
      })
    }
    if (p.cardOpacity != null) {
      items.push({
        id: 'cardOpacity',
        title: t('theme.editor.ovCardOpacity'),
        value: `${p.cardOpacity}%${t('theme.editor.ovGlobal', { v: `${content.value.background!.cardOpacity}%` })}`,
        swatch: 'var(--m-s)',
        drop: () => commit((c) => void delete c.pages[pageKey.value]!.cardOpacity),
      })
    }
    if (p.banner) {
      items.push({
        id: 'banner',
        title: t('theme.editor.ovBanner'),
        value: t(`theme.editor.bannerTypes.${p.banner.type}`),
        swatch: p.banner.type === 'image' ? `url("${assetUrl(p.banner.image)}") center / cover` : 'var(--m-p)',
        drop: () => commit((c) => void delete c.pages[pageKey.value]!.banner),
      })
    }
    for (const [key, v] of Object.entries(p.sizes || {})) {
      const tk = sizeToken(key)
      const g = groupOfSize(key)
      items.push({
        id: `size:${key}`,
        title: `${g ? sizeGroupLabel(g.key) : ''} · ${sizeLabel(key)}`,
        value: tk
          ? `${sizeDisplay(tk, v)}${t('theme.editor.ovGlobal', { v: sizeDisplay(tk, globalSizeValue(key)) })}`
          : String(v),
        swatch: 'var(--m-s2)',
        drop: () =>
          commit((c) => {
            const pg = c.pages[pageKey.value]
            if (!pg?.sizes) return
            const next = { ...pg.sizes }
            delete next[key]
            pg.sizes = next
          }),
      })
    }
    for (const [key, v] of Object.entries(p[tokensField.value])) {
      items.push({
        id: `tok:${key}`,
        title: key.startsWith('ansi') ? `ANSI ${key.slice(4)}` : t(`theme.tokens.${key}.name`),
        value: `${v}${t('theme.editor.ovGlobal', { v: globalTokens.value[key as TokenKey] })}`,
        swatch: v,
        drop: () => commit((c) => void delete c.pages[pageKey.value]![tokensField.value][key]),
      })
    }
    return items
  })
  const pagesWithOv = computed(() => new Set(Object.keys(content.value.pages)))

  // ---- 检查 ----
  const pageBg = computed(() => resolveBackground(content.value, pageKey.value, null).bg)
  const backdrop = computed(() => {
    const bg = pageBg.value
    if (!GLASS_TYPES.includes(bg.type as BgType)) return null
    const bright = bg.type === 'gradient' ? brightestStop(bg.gradientStops) : (imageStats(assetUrl(bg.image))?.bright ?? null)
    return bright ? { bright, mixed: backdropBrightColor(bright, tokens.value.bg, effectiveDim(bg, scheme.value)) } : null
  })
  const checks = computed(() =>
    contrastChecks(tokens.value, { backdropBright: backdrop.value?.mixed ?? null, terminal: isTermPage.value }),
  )
  const hasWarn = computed(() => checks.value.some((c) => !c.ok))
  const fixAll = () => {
    const bad = checks.value.filter((c) => !c.ok)
    if (!bad.length) return
    commit((c) => {
      const field = tokensField.value
      const target = scope.value === 'page' ? (c.pages[pageKey.value] ||= emptyPage())[field] : c[field]
      for (const ck of bad) {
        if (ck.fix === 'dim' && backdrop.value) {
          const dim = dimForContrast(tokens.value.ink, backdrop.value.bright, tokens.value.bg)
          const pg = c.pages[pageKey.value]
          if (pg?.background) pg.background.dim = dim
          else c.background!.dim = dim
          continue
        }
        for (const f of fixContrast(ck, tokens.value)) target[f.key] = f.value
      }
    })
    feedback.success(t('theme.editor.fixed'))
  }

  // ---- 推荐色 ----
  const suggestions = computed(() => {
    const key = sel.value
    let list = ACCENT_KEYS.includes(key) || key.startsWith('ansi')
      ? SUGGEST.accent
      : TEXT_KEYS.includes(key)
        ? SUGGEST.text
        : scheme.value === 'dark' || key === 'termBg'
          ? SUGGEST.surfD
          : SUGGEST.surfL
    const img = pageBg.value.type === 'image' ? imageStats(assetUrl(pageBg.value.image))?.palette : null
    if (img?.length && (ACCENT_KEYS.includes(key) || key.startsWith('ansi'))) list = [...img.slice(0, 4), ...list].slice(0, 8)
    return [...new Set(list.map((c) => c.toUpperCase()))].slice(0, 8)
  })

  // ---- 从背景图取色 / 生成深色版 ----
  const pickFromImage = () => {
    const bg = pageBg.value
    const pal = bg.type === 'image' ? imageStats(assetUrl(bg.image))?.palette : null
    if (!pal?.length) {
      feedback.warning(t('theme.editor.noImageToPick'))
      return
    }
    const sat = (hex: string) => hexToHsv(hex).s
    const vivid = [...pal].sort((x, y) => sat(y) - sat(x))
    const s = tokens.value.s
    const p = ensureContrast(vivid[0]!, s, 3)
    const a2 = ensureContrast(vivid[1] || mixHex(p, '#5B6CFF', 0.5), s, 3)
    const pOn = contrastRatio('#FFFFFF', p) >= contrastRatio('#140A1E', p) ? '#FFFFFF' : '#140A1E'
    commit((c) => {
      const field = tokensField.value
      const target = scope.value === 'page' ? (c.pages[pageKey.value] ||= emptyPage())[field] : c[field]
      target.p = p
      target.pOn = pOn
      target.a2 = a2
      if (vivid[2]) target.a = vivid[2]
    })
    feedback.success(t('theme.editor.pickedFromImage'))
  }

  const genDark = () => {
    const light = resolveTokens(content.value, 'light', null)
    const tint = (hex: string, base: string, w: number) => mixHex(hex, base, w)
    const lift = (hex: string) => {
      const hsv = hexToHsv(hex)
      return hsvToHex({ h: hsv.h, s: Math.min(1, hsv.s * 0.85), v: Math.max(0.82, hsv.v) })
    }
    const bg = tint(light.p, '#0C0A10', 0.08)
    const s = tint(light.p, '#16131B', 0.08)
    const s2 = tint(light.p, '#1E1A24', 0.1)
    const line = tint(light.p, '#2A2531', 0.14)
    const ink = tint(light.p, '#F6F2F8', 0.04)
    const ink2 = mixHex(ink, s, 0.66)
    const p = ensureContrast(lift(light.p), s, 3)
    const out: Record<string, string> = {
      bg, s, s2, line, ink, ink2, p,
      pOn: luminance(p) > 0.4 ? bg : '#FFFFFF',
      a: lift(light.a),
      a2: ensureContrast(lift(light.a2), s, 3),
      ok: ensureContrast(lift(light.ok), s, 3),
      warn: ensureContrast(lift(light.warn), s, 3),
      bad: ensureContrast(lift(light.bad), s, 3),
    }
    commit((c) => {
      c.tokensDark = { ...c.tokensDark, ...out }
    })
    scheme.value = 'dark'
    feedback.success(t('theme.editor.genDarkDone'))
  }

  // ---- 页面 ----
  const rememberPage = (key: string) => {
    if (!extraPages.value.includes(key)) extraPages.value.push(key)
  }
  /** 点页面标签：预览跳到这一页（带参数的页面只能在预览里点进去） */
  const setPage = (key: string) => {
    if (key === pageKey.value) return
    const def = pages.value.find((p) => p.key === key)
    if (def?.params) return
    pageKey.value = key
    pagePath.value = key
    scopeChoice.value = null
    frameTerminal.value = false
    rememberPage(key)
  }
  /** 预览框里换了页面（用户在预览里点进去）：编辑器跟过去 */
  const followFrame = (key: string, path: string, terminal: boolean) => {
    const clean = path.replace(new RegExp(`[?&]${THEME_PREVIEW_PARAM}=[^&]*`), '').replace(/^([^?]*)&/, '$1?')
    if (key !== pageKey.value) {
      pageKey.value = key
      scopeChoice.value = null
      rememberPage(key)
    }
    pagePath.value = clean
    frameTerminal.value = terminal
    if (!terminal && (sel.value.startsWith('term') || sel.value.startsWith('ansi'))) sel.value = 'p'
  }
  const select = (token: string) => {
    sel.value = token
    if (token === 'hero' || token.startsWith('size:')) tab.value = 'comp'
    else if (tab.value !== 'color') tab.value = 'color'
  }

  // ---- 读取 / 保存 ----
  const paletteName = (key: string) => t(`theme.palettes.${isPaletteKey(key) ? key : 'sakura'}`)

  const load = async () => {
    loading.value = true
    notFound.value = false
    undoStack.value = []
    redoStack.value = []
    try {
      let snap: Snapshot
      const id = routeId.value
      if (id === 'new') {
        themeId.value = ''
        const base = themeStore.usingSite ? normalizeContent(themeStore.siteTheme?.content) : themeStore.currentEffectiveContent
        snap = {
          name: t('theme.editor.newName', { name: themeStore.activeName || paletteName(base.base) }).slice(0, 40),
          content: cloneContent(base),
        }
        saved.value = { name: '', content: cloneContent(base) }
      } else {
        let th: Theme | undefined = themeStore.myThemes.find((x) => x.id === id)
        if (!th) th = (await getTheme(id)).data?.theme
        if (!th) {
          notFound.value = true
          return
        }
        themeId.value = th.id
        const c = normalizeContent(th.content)
        saved.value = { name: th.name, content: cloneContent(c) }
        snap = { name: th.name, content: cloneContent(c) }
        // 外观页上还没存进这个主题的背景修改，带进编辑器（保存后就存进主题了）
        if (th.id === themeStore.currentTheme && themeStore.bgModified && themeStore.bgOverride) {
          snap.content.background = normalizeBackground(themeStore.bgOverride)
        }
      }
      const draft = readDraft(themeId.value || 'new')
      if (draft && JSON.stringify(draft) !== JSON.stringify(saved.value)) {
        snap = draft
        feedback.info(t('theme.editor.draftRestored'))
      }
      name.value = snap.name
      content.value = normalizeContent(snap.content)
    } catch {
      notFound.value = true
    } finally {
      loading.value = false
    }
  }

  const discardDraft = () => {
    writeDraft(themeId.value || 'new', null)
    name.value = saved.value.name || name.value
    content.value = cloneContent(saved.value.content)
    undoStack.value = []
    redoStack.value = []
  }

  const save = async () => {
    const nm = name.value.trim() || t('theme.editor.defaultName')
    if (saving.value) return
    saving.value = true
    try {
      const body = { name: nm, description: '', content: content.value }
      const res = themeId.value ? await updateTheme({ id: themeId.value, ...body }) : await createTheme(body)
      const th = res.data?.theme
      if (!th) return
      const wasNew = !themeId.value
      writeDraft(themeId.value || 'new', null)
      themeId.value = th.id
      writeDraft(th.id, null)
      name.value = th.name
      content.value = normalizeContent(th.content)
      saved.value = { name: th.name, content: cloneContent(content.value) }
      themeStore.upsertTheme(th)
      themeStore.setTheme(th.id, th)
      feedback.success(t('theme.editor.saved'))
      if (wasNew) void router.replace(`/profile/theme/editor/${th.id}`)
    } finally {
      saving.value = false
    }
  }

  watch(routeId, (id, old) => {
    // 新建后 replace 到正式 id 时不必重新加载
    if (old === 'new' && id === themeId.value) return
    void load()
  })

  return {
    t,
    loading,
    notFound,
    saving,
    themeId,
    name,
    content,
    scheme,
    device,
    tab,
    pageKey,
    pagePath,
    pages,
    labelOf,
    pageName,
    isTermPage,
    scope,
    scopeChoice,
    sel,
    picking,
    extraPages,
    changeCount,
    canUndo,
    canRedo,
    undo,
    redo,
    setName,
    commit,
    tokens,
    globalTokens,
    tokenValue,
    isPageToken,
    isEdited,
    setToken,
    revertToken,
    selName,
    selHex,
    selBase,
    selEdited,
    isSizeSel,
    scopeTag,
    sizeGroupLabel,
    sizeLabel,
    sizeValue,
    isSizeEdited,
    isPageSize,
    setSize,
    revertSize,
    sizeGroupEdits,
    bgModel,
    bannerModel,
    setShape,
    page,
    pageOvCount,
    pageOvItems,
    pagesWithOv,
    pageBg,
    banner: computed(() => resolveBanner(content.value, pageKey.value)),
    checks,
    hasWarn,
    fixAll,
    suggestions,
    pickFromImage,
    genDark,
    setPage,
    followFrame,
    select,
    load,
    save,
    discardDraft,
    editorTokens: EDITOR_TOKENS,
  }
}

export type ThemeEditor = ReturnType<typeof createThemeEditor>
const KEY: InjectionKey<ThemeEditor> = Symbol('theme-editor')

export const provideThemeEditor = (routeId: Ref<string>) => {
  const editor = createThemeEditor(routeId)
  provide(KEY, editor)
  return editor
}

export const useThemeEditor = () => {
  const editor = inject(KEY)
  if (!editor) throw new Error('useThemeEditor() must be used inside the theme editor')
  return editor
}
