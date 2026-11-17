// 学マス 的目录（选择器、搜索与筛选用的条目）：管理端与用户通行证共用。
// 目录是游戏数据，和名称表、图片一样是前端静态资源：scripts/games/gakumas/export-names.py 导出到 ./catalogs/<kind>.json，
// 名称取名称表（当前语言的译名 → 游戏原文名）；GM 编辑的数据（卡池）由调用方按服务器读取。
// 关键字搜索、ID / 角色 / 稀有度筛选和分页都在浏览器里完成。
import type { CatalogEntry, CatalogQuery, ListPage } from '../mgp'
import { ensureGameNames, localizeEntries, lookupGameName, searchGameNames } from '../names'
import { fuzzyMatch } from '../search'

/** 目录的种类（也是前端名称表与素材包的分类）。 */
export type GakumasCatalogKind =
  | 'currency'
  | 'item'
  | 'idolCardSkin'
  | 'supportCard'
  | 'character'
  | 'costume'
  | 'costumeHead'
  | 'music'
  | 'gasha'
  | 'produceGroup'
  | 'produce'
  | 'seminar'
  | 'tower'
  | 'produceCard'
  | 'produceItem'
  | 'memoryAbility'

/** 目录的显示名。 */
export const catalogTitles: Record<GakumasCatalogKind, string> = {
  currency: '货币',
  item: '道具',
  idolCardSkin: '偶像卡',
  supportCard: '支援卡',
  character: '角色',
  costume: '服装',
  costumeHead: '发型',
  music: '乐曲',
  gasha: '卡池',
  produceGroup: '培育模式',
  produce: '培育剧本',
  seminar: '研修',
  tower: '塔',
  produceCard: '技能卡',
  produceItem: 'P 道具',
  memoryAbility: '记忆能力',
}

/** 可作为奖励发放的目录（按奖励选择器里的顺序）。 */
export const rewardCatalogs: GakumasCatalogKind[] = ['currency', 'item', 'supportCard', 'idolCardSkin', 'music', 'costume', 'costumeHead']

/** 奖励的资源类型（游戏协议 ResourceType）→ 目录；货币作为奖励时 resourceId 为空串，对应 currency 目录里的条目。 */
export const resourceKinds: Record<number, { kind: GakumasCatalogKind; currency?: string }> = {
  2: { kind: 'item' },
  4: { kind: 'supportCard' },
  5: { kind: 'idolCardSkin' },
  8: { kind: 'currency', currency: 'actionPoint' },
  9: { kind: 'currency', currency: 'exp' },
  12: { kind: 'music' },
  13: { kind: 'costumeHead' },
  14: { kind: 'costume' },
  1100: { kind: 'currency', currency: 'jewelFree' },
  1101: { kind: 'currency', currency: 'jewelPaid' },
}

/** 付费钻石的资源类型（邮件、兑换码不能发放）。 */
export const RESOURCE_JEWEL_PAID = 1101

/** 货币 → 资源类型。 */
export const currencyResourceTypes: Record<string, number> = { jewelFree: 1100, jewelPaid: 1101, actionPoint: 8, exp: 9 }

/** 一项奖励在目录中的位置。 */
export const resourceRef = (resourceType: number, resourceId: string): { kind: GakumasCatalogKind; id: string } | undefined => {
  const r = resourceKinds[resourceType]
  if (!r) return undefined
  if (r.currency) return { kind: 'currency', id: r.currency }
  return { kind: r.kind, id: resourceId }
}

/** 静态目录里的一条（export-names.py 导出；名称不在这里，取名称表）。 */
interface StaticEntry {
  id: string
  resourceType?: number
  rarity?: number
  type?: number
  characterId?: string
  parentId?: string
  assetId?: string
  characterAsset?: boolean
  levels?: number[]
}

const staticLoaders = import.meta.glob<StaticEntry[]>('./catalogs/*.json', { import: 'default' })

/** 条目文字所在的名称表（默认与目录同名）：技能卡强化名 `<id>+1`、P 道具效果文字、记忆能力各等级的效果文字 `<id>@<等级>`。 */
const TEXT_TABLES: Partial<Record<string, string[]>> = {
  produceItem: ['produceItem', 'produceItemText'],
  memoryAbility: ['memoryAbilityText'],
}

/** 游戏原文名的语言（names.ts 里没有译名表的语言取原文名）。 */
const ORIGINAL = 'ja'

/** 一类静态目录（名称换成当前语言的译名，原文名在 originalName）。 */
async function staticCatalog(game: string, locale: string, kind: string): Promise<CatalogEntry[]> {
  const load = staticLoaders[`./catalogs/${kind}.json`]
  if (!load) return []
  const tables = TEXT_TABLES[kind] ?? [kind]
  const [list] = await Promise.all([load(), Promise.all(tables.map((t) => ensureGameNames(game, locale, t)))])
  const text = (table: string, key: string) => lookupGameName(game, locale, table, key) || ''
  return list.map((e) => {
    const name = text(kind, e.id) || e.id
    const original = lookupGameName(game, ORIGINAL, kind, e.id) || ''
    const out: CatalogEntry = {
      id: e.id,
      name,
      resourceType: e.resourceType || 0,
      rarity: e.rarity,
      type: e.type,
      characterId: e.characterId,
      parentId: e.parentId,
      assetId: e.assetId,
      characterAsset: e.characterAsset,
    }
    if (original && original !== name) out.originalName = original
    if (kind === 'produceCard') out.upgradedName = text(kind, `${e.id}+1`) || undefined
    if (kind === 'produceItem') out.description = text('produceItemText', e.id) || undefined
    if (kind === 'memoryAbility') {
      out.levels = (e.levels || []).map((level) => ({ level, description: text('memoryAbilityText', `${e.id}@${level}`) }))
    }
    return out
  })
}

/** 服务器上的目录：不是静态游戏数据的种类（GM 编辑的卡池），按服务器读取整份。 */
export type ServerCatalogs = Partial<Record<string, (serverId: string) => Promise<CatalogEntry[]>>>

/** 目录查询（管理端与用户通行证共用）。 */
export interface CatalogStore {
  /** 按查询筛选分页（cursor 为偏移量）。 */
  query: (serverId: string, kind: string, query?: CatalogQuery) => Promise<ListPage<CatalogEntry>>
  /** 重读一类服务器目录（GM 发布卡池配置后）。 */
  refresh: (kind: string) => void
}

/**
 * 目录查询：静态目录每类读一次（浏览器按静态文件缓存），服务器目录每台服务器每类读一次，之后都在内存里筛选分页。
 * 关键字匹配：名称表（当前语言的译名与原文名，含技能卡强化名与记忆能力效果文字）、条目的名称与效果文字、ID 片段。
 */
export function createCatalogStore(game: () => string, locale: () => string, server: ServerCatalogs = {}): CatalogStore {
  const lists = new Map<string, Promise<CatalogEntry[]>>()

  const entries = (serverId: string, kind: string, lc: string): Promise<CatalogEntry[]> => {
    const load = server[kind]
    const key = load ? `${serverId}|${kind}|${lc}` : `|${kind}|${lc}`
    let list = lists.get(key)
    if (!list) {
      const value = load ? load(serverId).then((items) => localizeEntries(game(), lc, kind, items)) : staticCatalog(game(), lc, kind)
      lists.set(key, value)
      value.catch(() => lists.get(key) === value && lists.delete(key))
      list = value
    }
    return list
  }

  const query = async (serverId: string, kind: string, q: CatalogQuery = {}): Promise<ListPage<CatalogEntry>> => {
    const lc = locale()
    const list = await entries(serverId, kind, lc)
    const keyword = q.q?.trim() || ''
    const hits = keyword ? await searchGameNames(game(), lc, kind, keyword) : undefined
    const needle = keyword.toLowerCase()
    const ids = q.ids?.length ? new Set(q.ids) : undefined
    const rarity = Number(q.rarity) || 0
    const matched = list.filter(
      (e) =>
        (!ids || ids.has(e.id)) &&
        (!q.characterId || e.characterId === q.characterId) &&
        (!rarity || e.rarity === rarity) &&
        (!keyword ||
          hits?.has(e.id) ||
          e.id.toLowerCase().includes(needle) ||
          fuzzyMatch(keyword, [e.name, e.originalName, e.upgradedName, e.description, ...(e.levels || []).map((l) => l.description)])),
    )
    const offset = Number(q.cursor) || 0
    const limit = q.limit || 50
    return {
      items: matched.slice(offset, offset + limit),
      nextCursor: offset + limit < matched.length ? String(offset + limit) : '',
      total: matched.length,
    }
  }

  const refresh = (kind: string) => {
    for (const key of [...lists.keys()]) if (key.split('|')[1] === kind) lists.delete(key)
  }

  return { query, refresh }
}
