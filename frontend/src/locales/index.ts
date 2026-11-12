import { createI18n } from 'vue-i18n'
import { APP_LOCALES, type AppLocale } from '@/locales/types'
import { knownTextMessages, messages as baseMessages } from '@/locales/messages'

type LocaleTree = { [key: string]: string | LocaleTree }

// 各模块的补充文案放在 locales/extra/*.ts（默认导出 { 'zh-CN': {...}, 'zh-HK': {...}, 'en-US': {...} }），
// 启动时深合并进主文案表，同名键以补充文案为准。
const deepMerge = (target: LocaleTree, source: LocaleTree): LocaleTree => {
  const out: LocaleTree = { ...target }
  for (const [key, value] of Object.entries(source)) {
    const prev = out[key]
    out[key] =
      typeof value === 'object' && typeof prev === 'object' ? deepMerge(prev, value) : value
  }
  return out
}

const extraModules = import.meta.glob<{ default: Partial<Record<AppLocale, LocaleTree>> }>(
  './extra/*.ts',
  { eager: true },
)

const messages = Object.values(extraModules).reduce(
  (acc, mod) => {
    for (const [locale, tree] of Object.entries(mod.default)) {
      const key = locale as AppLocale
      acc[key] = deepMerge(acc[key] as LocaleTree, tree as LocaleTree) as (typeof acc)[AppLocale]
    }
    return acc
  },
  { ...baseMessages } as Record<AppLocale, (typeof baseMessages)[AppLocale]>,
) as typeof baseMessages

const LOCALE_STORAGE_KEY = 'app-locale'

const isAppLocale = (value: unknown): value is AppLocale =>
  typeof value === 'string' && APP_LOCALES.includes(value as AppLocale)

const normalizeNavigatorLocale = (value: string): AppLocale | null => {
  const normalized = value.toLowerCase()

  if (normalized === 'zh-hk' || normalized === 'zh-tw' || normalized === 'zh-mo') {
    return 'zh-HK'
  }

  if (normalized === 'zh' || normalized.startsWith('zh-')) {
    return 'zh-CN'
  }

  if (normalized === 'en' || normalized.startsWith('en-')) {
    return 'en-US'
  }

  return null
}

const resolveSystemLocale = (): AppLocale => {
  if (typeof navigator === 'undefined') return 'zh-CN'

  const locales = navigator.languages?.length ? navigator.languages : [navigator.language]
  for (const locale of locales) {
    const resolvedLocale = normalizeNavigatorLocale(locale)
    if (resolvedLocale) return resolvedLocale
  }

  return 'zh-CN'
}

const resolveInitialLocale = (): AppLocale => {
  if (typeof localStorage !== 'undefined') {
    const storedLocale = localStorage.getItem(LOCALE_STORAGE_KEY)
    if (isAppLocale(storedLocale)) return storedLocale
  }

  return resolveSystemLocale()
}

export const i18n = createI18n({
  legacy: false,
  globalInjection: true,
  locale: resolveInitialLocale(),
  fallbackLocale: 'zh-CN',
  messages,
  missingWarn: false,
  fallbackWarn: false,
})

export const languageOptions: Array<{ code: AppLocale; shortLabel: string; labelKey: string }> = [
  { code: 'zh-CN', shortLabel: 'CN', labelKey: 'language.simplifiedChinese' },
  { code: 'zh-HK', shortLabel: 'HK', labelKey: 'language.traditionalChinese' },
  { code: 'en-US', shortLabel: 'EN', labelKey: 'language.english' },
]

// 页面专用文案按路由按需加载：locales/route/<路由第一段>/*.ts 只在进入 /<路由第一段>/… 时加载并合并，
// 不打进首屏（例如 route/game/ 只在游戏服务器、GM 运营、用户通行证后台页面里用到）。
const routeModules = import.meta.glob<{ default: Partial<Record<AppLocale, LocaleTree>> }>('./route/*/*.ts')
const routeLoads = new Map<string, Promise<void>>()

/** 加载某个路由路径需要的页面文案（只加载一次；没有对应目录时立即完成）。 */
export const ensureRouteLocales = (path: string): Promise<void> => {
  const segment = path.split('/').filter(Boolean)[0] || ''
  if (!segment) return Promise.resolve()
  let p = routeLoads.get(segment)
  if (!p) {
    const loaders = Object.entries(routeModules)
      .filter(([file]) => file.startsWith(`./route/${segment}/`))
      .map(([, load]) => load)
    p = Promise.all(loaders.map((load) => load())).then((mods) => {
      for (const mod of mods) {
        for (const [locale, tree] of Object.entries(mod.default)) {
          i18n.global.mergeLocaleMessage(locale, tree as LocaleTree)
        }
      }
    })
    p.catch(() => routeLoads.delete(segment))
    routeLoads.set(segment, p)
  }
  return p
}

export const getCurrentLocale = () => i18n.global.locale.value as AppLocale

export const setAppLocale = (locale: AppLocale) => {
  i18n.global.locale.value = locale
  localStorage.setItem(LOCALE_STORAGE_KEY, locale)
  document.documentElement.lang = locale
}

export const translate = (
  key: string,
  params?: Record<string, string | number | boolean | null | undefined>,
) => i18n.global.t(key, params || {})

export const translateKnownText = (text?: string | null) => {
  if (!text) return ''
  const locale = getCurrentLocale()
  return knownTextMessages[locale]?.[text] || text
}

document.documentElement.lang = getCurrentLocale()
