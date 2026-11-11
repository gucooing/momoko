import { defineStore } from 'pinia'
import { useDark, usePreferredDark, useToggle } from '@vueuse/core'
import type { SiteTheme, Theme, ThemeBackground, ThemeContent } from '@/types/v1/theme'
import { PALETTE_OPTIONS, isPaletteKey, type PaletteKey, type SchemeKey } from '@/theme/palettes'
import {
  emptyContent,
  normalizeBackground,
  normalizeContent,
  resolveBackground,
  resolveBanner,
  resolveTokens,
  type AssetResolver,
} from '@/theme/content'
import { applyThemeToDocument } from '@/theme/engine'
import { IS_THEME_PREVIEW_FRAME } from '@/theme/frame'
import * as themeApi from '@/api/theme'

export type ThemeMode = 'light' | 'dark' | 'auto'
/** 圆角：round 圆润（默认） / standard 标准（整体收小一档） */
export type RadiusMode = 'round' | 'standard'
export type { PaletteKey }

/** 一份完整的外观偏好（跟随账号同步，见 stores/theme/sync.ts） */
export interface AppearancePrefs {
  /** 内置主题为配色键，「我的主题」为主题 id */
  theme: string
  mode: ThemeMode
  compact: boolean
  reduceMotion: boolean
  radius: RadiusMode
  showTabs: boolean
  /** 外观页上改过、还没存进主题的背景 */
  background: ThemeBackground | null
}

/** 临时套用的主题（导入前预览 / 预览「我的主题」）：只在当前会话内存里，刷新即恢复 */
export interface ThemePreview {
  name: string
  content: ThemeContent
  scheme: SchemeKey
  /** import：导入前的主题包；mine：我的主题 */
  from: 'import' | 'mine'
  themeId?: string
  /** 导入前的主题包里的图片（包内路径 → data URL） */
  resolver?: AssetResolver
  /** 悬浮条上的主按钮（导入并使用 / 使用这个主题） */
  accept?: () => Promise<void> | void
}

export { PALETTE_OPTIONS, isPaletteKey }
export const isThemeMode = (v: unknown): v is ThemeMode => v === 'light' || v === 'dark' || v === 'auto'
export const isRadiusMode = (v: unknown): v is RadiusMode => v === 'round' || v === 'standard'

const DEFAULT_THEME: PaletteKey = 'sakura'
const LS = {
  mode: 'themeMode',
  theme: 'theme-current',
  themeData: 'theme-current-data',
  bg: 'theme-bg-override',
  site: 'theme-site',
  followSite: 'theme-follow-site',
  compact: 'theme-compact',
  reduceMotion: 'theme-reduce-motion',
  showTabs: 'showTabs',
  radius: 'theme-radius',
}

const readJSON = <T>(key: string): T | null => {
  try {
    const raw = localStorage.getItem(key)
    return raw ? (JSON.parse(raw) as T) : null
  } catch {
    return null
  }
}
const readBool = (key: string, fallback: boolean) => {
  const v = readJSON<unknown>(key)
  return typeof v === 'boolean' ? v : fallback
}
/** 预览框里只套用草稿，绝不写本机存储 */
const write = (key: string, value: unknown) => {
  if (IS_THEME_PREVIEW_FRAME) return
  try {
    if (value === null || value === undefined) localStorage.removeItem(key)
    else localStorage.setItem(key, typeof value === 'string' ? value : JSON.stringify(value))
  } catch {
    /* 隐私模式 / 配额 */
  }
}

export const useThemeStore = defineStore('theme', () => {
  const colorSchemeIsDark = useDark()
  const toggleDark = useToggle(colorSchemeIsDark)
  const preferredDark = usePreferredDark()

  // ---- 偏好 ----
  const storedMode = localStorage.getItem(LS.mode)
  const themeMode = ref<ThemeMode>(isThemeMode(storedMode) ? storedMode : 'auto')
  const isDarkTheme = computed(() => (themeMode.value === 'auto' ? preferredDark.value : themeMode.value === 'dark'))
  const compact = ref(readBool(LS.compact, false))
  const reduceMotion = ref(readBool(LS.reduceMotion, false))
  const showTabs = ref(readBool(LS.showTabs, false))
  const storedRadius = localStorage.getItem(LS.radius)
  const radius = ref<RadiusMode>(isRadiusMode(storedRadius) ? storedRadius : 'round')

  // ---- 当前主题 ----
  const currentTheme = ref<string>(localStorage.getItem(LS.theme) || DEFAULT_THEME)
  /** 当前主题是「我的主题」时缓存它（刷新后先用缓存立即套上，避免闪一下默认配色） */
  const currentCustom = ref<Theme | null>(isPaletteKey(currentTheme.value) ? null : readJSON<Theme>(LS.themeData))
  const bgOverride = ref<ThemeBackground | null>(readJSON<ThemeBackground>(LS.bg))
  const myThemes = ref<Theme[]>([])
  const themesLoaded = ref(false)

  // ---- 全站默认 ----
  const siteTheme = ref<SiteTheme | null>(readJSON<SiteTheme>(LS.site))
  /** 账号从没设置过外观：跟随全站默认 */
  const followSite = ref(readBool(LS.followSite, false))

  // ---- 路由（页面单独设置、登录页用全站默认）----
  const pageKey = ref<string | null>(null)
  const onAuthPage = ref(false)

  // ---- 预览 ----
  const preview = shallowRef<ThemePreview | null>(null)
  /** 编辑器预览框里送进来的草稿 */
  const frameDraft = shallowRef<{ content: ThemeContent; scheme: SchemeKey; pageKey: string } | null>(null)

  const isBuiltin = computed(() => isPaletteKey(currentTheme.value))
  const currentContent = computed<ThemeContent>(() => {
    if (isPaletteKey(currentTheme.value)) return emptyContent(currentTheme.value)
    const c = currentCustom.value?.content
    return c ? normalizeContent(c) : emptyContent(DEFAULT_THEME)
  })
  const currentName = computed(() => currentCustom.value?.name || '')
  /** 当前主题 + 外观页未保存的背景修改（另存为 / 导出 / 设为全站默认用它） */
  const currentEffectiveContent = computed<ThemeContent>(() =>
    bgOverride.value ? { ...currentContent.value, background: normalizeBackground(bgOverride.value) } : currentContent.value,
  )
  const bgModified = computed(
    () => !!bgOverride.value && JSON.stringify(normalizeBackground(bgOverride.value)) !== JSON.stringify(currentContent.value.background),
  )

  const usingSite = computed(() => !!siteTheme.value?.content && (onAuthPage.value || followSite.value))

  /** 现在实际套用的主题内容 */
  const activeContent = computed<ThemeContent>(() => {
    if (frameDraft.value) return frameDraft.value.content
    if (preview.value) return preview.value.content
    if (usingSite.value) return normalizeContent(siteTheme.value!.content)
    return currentContent.value
  })
  const activeOverride = computed(() =>
    frameDraft.value || preview.value || usingSite.value ? null : bgOverride.value,
  )
  const activePageKey = computed(() => frameDraft.value?.pageKey ?? pageKey.value)
  const activeResolver = computed(() => preview.value?.resolver)
  const effectiveDark = computed(() => {
    if (frameDraft.value) return frameDraft.value.scheme === 'dark'
    if (preview.value) return preview.value.scheme === 'dark'
    return isDarkTheme.value
  })
  const scheme = computed<SchemeKey>(() => (effectiveDark.value ? 'dark' : 'light'))

  /** 现在套用的主题名（内置主题为空，界面上显示配色名） */
  const activeName = computed(() => {
    if (preview.value) return preview.value.name
    if (usingSite.value) return siteTheme.value?.name || ''
    return isPaletteKey(currentTheme.value) ? '' : currentName.value
  })

  /** 所基于的内置配色（兼容老代码：配色键） */
  const palette = computed<PaletteKey>(() => (isPaletteKey(activeContent.value.base) ? activeContent.value.base : DEFAULT_THEME))
  const currentPalette = computed(() => PALETTE_OPTIONS.find((p) => p.key === palette.value) ?? PALETTE_OPTIONS[0]!)

  /** 当前页面实际生效的完整色表（终端、图表等要具体色值的地方用） */
  const tokens = computed(() => resolveTokens(activeContent.value, scheme.value, activePageKey.value))
  const background = computed(() => resolveBackground(activeContent.value, activePageKey.value, activeOverride.value))
  const banner = computed(() => resolveBanner(activeContent.value, activePageKey.value))
  /** 主题里有终端 token 覆盖时，深色终端使用它们 */
  const hasTerminalTokens = computed(() => {
    const c = activeContent.value
    const page = activePageKey.value ? c.pages[activePageKey.value] : undefined
    const keys = [
      ...Object.keys(scheme.value === 'dark' ? c.tokensDark : c.tokensLight),
      ...Object.keys(page ? (scheme.value === 'dark' ? page.tokensDark : page.tokensLight) : {}),
    ]
    return keys.some((k) => k.startsWith('term') || k.startsWith('ansi'))
  })

  // ---- 套用到文档 ----
  const applyAttributes = () => {
    const root = document.documentElement
    root.classList.toggle('is-compact', compact.value)
    root.classList.toggle('is-reduce-motion', reduceMotion.value)
    applyThemeToDocument({
      content: activeContent.value,
      pageKey: activePageKey.value,
      override: activeOverride.value,
      radiusPref: radius.value,
      resolver: activeResolver.value,
    })
  }
  const applyThemeMode = () => {
    toggleDark(effectiveDark.value)
    applyAttributes()
  }

  watch([activeContent, activeOverride, activePageKey, activeResolver, compact, reduceMotion, radius], applyAttributes)
  watch(effectiveDark, applyThemeMode)

  // ---- 偏好设置 ----
  const toggleThemeMode = (newVal: ThemeMode) => {
    themeMode.value = newVal
    write(LS.mode, newVal)
    applyThemeMode()
  }
  /** 顶栏按钮：在浅色/深色之间快速切换 */
  const toggleDarkQuick = () => toggleThemeMode(isDarkTheme.value ? 'light' : 'dark')

  const setCompact = (v: boolean) => {
    compact.value = v
    write(LS.compact, v)
  }
  const setReduceMotion = (v: boolean) => {
    reduceMotion.value = v
    write(LS.reduceMotion, v)
  }
  const toggleShowTabs = (v: boolean) => {
    showTabs.value = v
    write(LS.showTabs, v)
  }
  const setRadius = (v: RadiusMode) => {
    radius.value = v
    write(LS.radius, v)
  }
  const setFollowSite = (v: boolean) => {
    followSite.value = v
    write(LS.followSite, v)
  }

  const cacheCustom = (t: Theme | null) => {
    currentCustom.value = t
    write(LS.themeData, t)
  }

  /** 选主题（内置为配色键，我的主题为 id；会清掉外观页未保存的背景修改） */
  const setTheme = (key: string, theme?: Theme | null) => {
    setFollowSite(false)
    bgOverride.value = null
    write(LS.bg, null)
    currentTheme.value = key
    write(LS.theme, key)
    if (isPaletteKey(key)) cacheCustom(null)
    else cacheCustom(theme ?? myThemes.value.find((t) => t.id === key) ?? currentCustom.value)
  }
  /** 兼容老入口：选内置配色 */
  const setPalette = (key: PaletteKey) => setTheme(key)

  const setBackground = (bg: ThemeBackground | null) => {
    setFollowSite(false)
    bgOverride.value = bg ? normalizeBackground(bg) : null
    write(LS.bg, bgOverride.value)
  }

  const resetAppearance = () => {
    setTheme(DEFAULT_THEME)
    setCompact(false)
    setReduceMotion(false)
    setRadius('round')
    toggleThemeMode('auto')
  }

  /** 当前外观偏好快照（同步到账号） */
  const appearance = computed<AppearancePrefs>(() => ({
    // 空字符串 = 跟随全站默认（账号从没选过主题）
    theme: followSite.value ? '' : currentTheme.value,
    mode: themeMode.value,
    compact: compact.value,
    reduceMotion: reduceMotion.value,
    radius: radius.value,
    showTabs: showTabs.value,
    background: bgOverride.value,
  }))

  /** 一次性套用账号里的外观偏好（非法值保持当前值） */
  const applyAppearance = (prefs: Partial<AppearancePrefs>, theme?: Theme | null) => {
    setFollowSite(prefs.theme === '' && !!siteTheme.value)
    if (typeof prefs.theme === 'string' && prefs.theme) {
      currentTheme.value = prefs.theme
      write(LS.theme, prefs.theme)
      if (isPaletteKey(prefs.theme)) cacheCustom(null)
      else if (theme) cacheCustom(theme)
      else if (currentCustom.value?.id !== prefs.theme) cacheCustom(null)
    }
    bgOverride.value = prefs.background ? normalizeBackground(prefs.background) : null
    write(LS.bg, bgOverride.value)
    if (typeof prefs.compact === 'boolean') setCompact(prefs.compact)
    if (typeof prefs.reduceMotion === 'boolean') setReduceMotion(prefs.reduceMotion)
    if (isRadiusMode(prefs.radius)) setRadius(prefs.radius)
    if (typeof prefs.showTabs === 'boolean') toggleShowTabs(prefs.showTabs)
    if (isThemeMode(prefs.mode)) toggleThemeMode(prefs.mode)
  }

  const setSiteThemeData = (t: SiteTheme | null | undefined) => {
    siteTheme.value = t?.content ? t : null
    write(LS.site, siteTheme.value)
  }

  /** 登录页等未登录页面：拉一次全站默认主题 */
  const loadSiteTheme = async () => {
    try {
      const { data } = await themeApi.getSiteTheme()
      setSiteThemeData(data?.theme)
    } catch {
      /* 拿不到就用本机外观 */
    }
  }

  const setRoute = (key: string | null, authPage: boolean) => {
    pageKey.value = key
    onAuthPage.value = authPage
  }

  // ---- 我的主题 ----
  const upsertTheme = (t: Theme) => {
    const i = myThemes.value.findIndex((x) => x.id === t.id)
    if (i >= 0) myThemes.value.splice(i, 1, t)
    else myThemes.value.unshift(t)
    if (t.id === currentTheme.value) cacheCustom(t)
  }

  const loadThemes = async () => {
    const { data } = await themeApi.listThemes()
    myThemes.value = data?.themes || []
    themesLoaded.value = true
    const cur = myThemes.value.find((t) => t.id === currentTheme.value)
    if (cur) cacheCustom(cur)
    else if (!isPaletteKey(currentTheme.value)) setTheme(currentContent.value.base || DEFAULT_THEME)
    // 管理员把「我的主题」设为全站默认后，卡片上的标记以服务端为准
    return myThemes.value
  }

  const removeTheme = async (id: string) => {
    const wasCurrent = id === currentTheme.value
    const base = myThemes.value.find((t) => t.id === id)?.content?.base
    await themeApi.deleteTheme(id)
    myThemes.value = myThemes.value.filter((t) => t.id !== id)
    if (wasCurrent) setTheme(isPaletteKey(base) ? base : DEFAULT_THEME)
  }

  const duplicateThemeById = async (id: string) => {
    const { data } = await themeApi.duplicateTheme({ id, name: '' })
    if (data?.theme) upsertTheme(data.theme)
    return data?.theme
  }

  /** 另存为：当前外观（含未保存的背景修改）存成一个新主题，并切换过去 */
  const saveCurrentAs = async (name: string) => {
    const { data } = await themeApi.createTheme({ name, description: '', content: currentEffectiveContent.value })
    if (!data?.theme) return null
    upsertTheme(data.theme)
    setTheme(data.theme.id, data.theme)
    return data.theme
  }

  /** 设为 / 取消全站默认（默认用当前外观，也可以指定「我的主题」里的某一个） */
  const setSiteDefault = async (enabled: boolean, target?: { id: string; name: string; content: ThemeContent }) => {
    const id = target?.id ?? currentTheme.value
    const { data } = await themeApi.setSiteTheme({
      enabled,
      themeId: id,
      name: target?.name ?? currentName.value,
      content: target?.content ?? currentEffectiveContent.value,
    })
    setSiteThemeData(enabled ? data?.theme : null)
    myThemes.value = myThemes.value.map((t) => ({ ...t, siteDefault: enabled && t.id === id }))
  }

  /** 当前主题就是全站默认 */
  const isSiteDefault = computed(() => !!siteTheme.value && siteTheme.value.themeId === currentTheme.value)

  // ---- 预览 ----
  const startPreview = (p: ThemePreview) => {
    preview.value = p
  }
  const setPreviewScheme = (s: SchemeKey) => {
    if (preview.value) preview.value = { ...preview.value, scheme: s }
  }
  const endPreview = () => {
    preview.value = null
  }
  const setFrameDraft = (d: { content: ThemeContent; scheme: SchemeKey; pageKey: string } | null) => {
    frameDraft.value = d
  }

  const themeConfigDrawerOpen = ref(false)

  watch(preferredDark, () => {
    if (themeMode.value === 'auto') applyThemeMode()
  })

  applyThemeMode()

  return {
    // 偏好
    themeMode,
    isDarkTheme,
    effectiveDark,
    scheme,
    compact,
    reduceMotion,
    showTabs,
    radius,
    appearance,
    themeConfigDrawerOpen,
    toggleThemeMode,
    toggleDarkQuick,
    setCompact,
    setReduceMotion,
    toggleShowTabs,
    setRadius,
    applyAppearance,
    resetAppearance,
    // 主题
    palette,
    currentPalette,
    paletteOptions: PALETTE_OPTIONS,
    currentTheme,
    currentCustom,
    currentName,
    currentContent,
    currentEffectiveContent,
    isBuiltin,
    bgOverride,
    bgModified,
    myThemes,
    themesLoaded,
    activeContent,
    activeName,
    activePageKey,
    activeResolver,
    tokens,
    background,
    banner,
    hasTerminalTokens,
    setTheme,
    setPalette,
    setBackground,
    loadThemes,
    upsertTheme,
    removeTheme,
    duplicateThemeById,
    saveCurrentAs,
    // 全站默认
    siteTheme,
    followSite,
    usingSite,
    isSiteDefault,
    setSiteThemeData,
    setFollowSite,
    loadSiteTheme,
    setSiteDefault,
    // 路由
    pageKey,
    setRoute,
    // 预览
    preview,
    startPreview,
    setPreviewScheme,
    endPreview,
    frameDraft,
    setFrameDraft,
  }
})
