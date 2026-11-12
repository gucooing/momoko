// 游戏里的名称（角色、道具、偶像卡、支援卡、服装、乐曲、培育、塔、抽卡按钮、货币）：
// 和图片一样是前端资源（scripts/games/<game>/export-names.py 导出到 ./<game>/names/<lang>/<kind>.json）。
// 界面上只显示名称、不显示 ID：当前语言有译名用译名，没有时用游戏原文名（ja 表），都没有时返回空串，
// 由使用方显示「未命名」之类的文案。
// 按需加载：每个「语言 × 目录类型」一份小文件，第一次用到时才加载。管理端与用户通行证共用。
import { reactive } from 'vue'
import { fuzzyMatch } from './search'

type NameTable = Record<string, string>

const loaders = import.meta.glob<NameTable>('./*/names/*/*.json', { import: 'default' })

const loaded = reactive(new Map<string, NameTable>())
const pending = new Map<string, Promise<NameTable | undefined>>()

/** 界面语言 → 译名语言（zh-CN / zh-HK 用中文译名，en-US 用英文译名）；没有对应译名时为空串。 */
export const nameLangOf = (locale: string): string => {
  const l = locale.toLowerCase()
  if (l.startsWith('zh')) return 'zh'
  if (l.startsWith('en')) return 'en'
  return ''
}

const keyOf = (game: string, lang: string, kind: string) => `${game}:${lang}:${kind}`

/** 原文名所在的表（游戏主数据里的名称）。 */
const ORIGINAL = 'ja'

const loadTable = (game: string, lang: string, kind: string): Promise<NameTable | undefined> => {
  const key = keyOf(game, lang, kind)
  const done = loaded.get(key)
  if (done) return Promise.resolve(done)
  let p = pending.get(key)
  if (!p) {
    const loader = loaders[`./${game}/names/${lang}/${kind}.json`]
    p = loader
      ? loader()
          .then((table) => {
            loaded.set(key, table)
            return table
          })
          .catch(() => {
            pending.delete(key)
            return undefined
          })
      : Promise.resolve(undefined)
    pending.set(key, p)
  }
  return p
}

/** 加载一类译名（只加载一次；当前语言没有这份译名时返回 undefined）。 */
export function loadGameNames(game: string, locale: string, kind: string): Promise<NameTable | undefined> {
  const lang = nameLangOf(locale)
  return lang ? loadTable(game, lang, kind) : Promise.resolve(undefined)
}

/** 一张名称表里的名称：undefined = 表还在加载；'' = 没有这张表或表里没有这一项。 */
const lookupSync = (game: string, lang: string, kind: string, id: string): string | undefined => {
  if (!loaders[`./${game}/names/${lang}/${kind}.json`]) return ''
  const table = loaded.get(keyOf(game, lang, kind))
  if (!table) {
    void loadTable(game, lang, kind)
    return undefined
  }
  return table[id] || ''
}

/**
 * 前端名称表里的名称（同步、响应式）：当前语言的译名 → 游戏原文名。
 * undefined = 名称表还在加载（加载完成后用到它的组件自动重新渲染）；'' = 名称表里没有（调用方再用服务端目录的名称）。
 */
export function lookupGameName(game: string, locale: string, kind: string, id: string): string | undefined {
  if (!game || !kind || !id) return ''
  const lang = nameLangOf(locale)
  let waiting = false
  if (lang && lang !== ORIGINAL) {
    const name = lookupSync(game, lang, kind, id)
    if (name) return name
    waiting = name === undefined
  }
  const original = lookupSync(game, ORIGINAL, kind, id)
  if (original) return original
  return waiting || original === undefined ? undefined : ''
}

/**
 * 名称（同步、响应式），从不返回 ID：当前语言的译名 → 游戏原文名 → fallback（服务端给的名称）→ 空串。
 * 名称表还没加载时先返回 fallback 并开始加载，加载完成后用到它的组件自动重新渲染。
 */
export function gameName(game: string, locale: string, kind: string, id: string, fallback = ''): string {
  return lookupGameName(game, locale, kind, id) || fallback
}

/**
 * 一个条目可用于搜索的全部名称（同步、响应式）：当前语言的译名与游戏原文名（去重）。
 * 名称表还没加载时先返回已有的，加载完成后用到它的 computed 自动重新计算。
 */
export function gameNameVariants(game: string, locale: string, kind: string, id: string): string[] {
  if (!game || !kind || !id) return []
  const lang = nameLangOf(locale)
  const out: string[] = []
  const add = (n: string | undefined) => {
    if (n && !out.includes(n)) out.push(n)
  }
  if (lang && lang !== ORIGINAL) add(lookupSync(game, lang, kind, id))
  add(lookupSync(game, ORIGINAL, kind, id))
  return out
}

/** 等某一类名称表（当前语言与原文）加载完，用于需要一次性拿到名称的场景（导出、排序）。 */
export async function ensureGameNames(game: string, locale: string, kind: string): Promise<void> {
  const lang = nameLangOf(locale)
  await Promise.all([lang && lang !== ORIGINAL ? loadTable(game, lang, kind) : undefined, loadTable(game, ORIGINAL, kind)])
}

/** 一类条目的全部 ID（游戏原文名表就是主数据的完整列表）：没有 MGP 目录的类型（例如抽卡按钮）用它列出可选项。 */
export async function gameNameIds(game: string, kind: string): Promise<string[]> {
  return Object.keys((await loadTable(game, ORIGINAL, kind)) || {})
}

/** 把一批目录条目的 name 换成当前语言的译名（等译名表加载完；没有译名的保留原文）。 */
export async function localizeEntries<T extends { id: string; name?: string }>(
  game: string,
  locale: string,
  kind: string,
  entries: T[],
): Promise<T[]> {
  await ensureGameNames(game, locale, kind)
  return entries.map((e) => {
    const name = gameName(game, locale, kind, e.id, e.name)
    return name && name !== e.name ? { ...e, name, originalName: e.name } : e
  })
}

/** 在名称表（当前语言的译名与游戏原文名）里模糊搜索关键字（search.ts 的 fuzzyMatch）；没有名称表时返回空。 */
export async function searchGameNames(game: string, locale: string, kind: string, q: string): Promise<Set<string>> {
  const lang = nameLangOf(locale)
  const tables = await Promise.all([
    lang && lang !== ORIGINAL ? loadTable(game, lang, kind) : undefined,
    loadTable(game, ORIGINAL, kind),
  ])
  const out = new Set<string>()
  if (!q.trim()) return out
  for (const table of tables) {
    for (const [id, name] of Object.entries(table ?? {})) {
      if (fuzzyMatch(q, [name])) out.add(id)
    }
  }
  return out
}
