// 用户通行证自己的文案（不引用管理后台的 locales）：modules/*.ts 各自默认导出
// { 'zh-CN': {...}, 'zh-HK': {...}, 'en-US': {...} }，这里深合并成一份。
import { createI18n } from 'vue-i18n'

export type PassportLocale = 'zh-CN' | 'zh-HK' | 'en-US'
export const LOCALES: PassportLocale[] = ['zh-CN', 'zh-HK', 'en-US']

type Tree = { [key: string]: string | Tree }

const modules = import.meta.glob<{ default: Partial<Record<PassportLocale, Tree>> }>('./modules/*.ts', { eager: true })

function merge(into: Tree, from: Tree): Tree {
  for (const [k, v] of Object.entries(from)) {
    const cur = into[k]
    if (typeof v === 'object' && typeof cur === 'object') merge(cur, v)
    else into[k] = typeof v === 'object' ? merge({}, v) : v
  }
  return into
}

const messages: Record<PassportLocale, Tree> = { 'zh-CN': {}, 'zh-HK': {}, 'en-US': {} }
for (const mod of Object.values(modules)) {
  for (const locale of LOCALES) {
    const part = mod.default[locale]
    if (part) merge(messages[locale], part)
  }
}

const STORAGE_KEY = 'passport.locale'

/** 浏览器语言 → 通行证支持的语言。 */
export function browserLocale(): PassportLocale {
  const lang = (navigator.language || 'zh-CN').toLowerCase()
  if (lang.startsWith('zh')) return /hk|tw|mo|hant/.test(lang) ? 'zh-HK' : 'zh-CN'
  return 'en-US'
}

function initialLocale(): PassportLocale {
  const saved = localStorage.getItem(STORAGE_KEY) as PassportLocale | null
  return saved && LOCALES.includes(saved) ? saved : browserLocale()
}

export const i18n = createI18n({
  legacy: false,
  locale: initialLocale(),
  fallbackLocale: 'zh-CN',
  messages,
  missingWarn: false,
  fallbackWarn: false,
})

/** 切换界面语言（空串表示跟随浏览器）。 */
export function setLocale(locale: string) {
  const next = LOCALES.includes(locale as PassportLocale) ? (locale as PassportLocale) : browserLocale()
  if (locale) localStorage.setItem(STORAGE_KEY, next)
  else localStorage.removeItem(STORAGE_KEY)
  i18n.global.locale.value = next
  document.documentElement.lang = next
}

export const t = (key: string, params?: Record<string, unknown>) => i18n.global.t(key, params ?? {})
export const te = (key: string) => i18n.global.te(key)
