import { createI18n } from 'vue-i18n'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import zhTw from 'element-plus/es/locale/lang/zh-tw'
import en from 'element-plus/es/locale/lang/en'
import { APP_LOCALES, type AppLocale } from '@/locales/types'
import { knownTextMessages, messages } from '@/locales/messages'

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

export const elementPlusLocales = {
  'zh-CN': zhCn,
  'zh-HK': zhTw,
  'en-US': en,
} satisfies Record<AppLocale, typeof zhCn>

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
