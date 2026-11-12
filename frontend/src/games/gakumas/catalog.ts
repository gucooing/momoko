// 学マス 的目录（名称与素材 ID）：每类目录对应一个接口，结果整理成页面统一使用的 CatalogEntry。
// 管理端与用户通行证各自传入自己的接口函数（src/api/gakumas.ts、passport/src/api/gakumas.ts）。
import type { CatalogEntry, CatalogQuery, ListPage } from '../mgp'
import type {
  CharacterPage,
  CostumePage,
  CurrencyPage,
  GashaEntryPage,
  IdolCardSkinPage,
  ItemEntryPage,
  MemoryAbilityPage,
  MusicPage,
  ProduceCardEntryPage,
  ProduceEntryPage,
  ProduceGroupPage,
  ProduceItemEntryPage,
  SeminarEntryPage,
  SupportCardEntryPage,
  TowerEntryPage,
} from '@/types/v1/gakumas/types'

/** 目录查询的公共参数（两端的目录接口都接受这些字段；各自的请求类型在 gm.ts / passport.ts）。 */
export interface CatalogParams {
  serverId: string
  q: string
  ids: string[]
  cursor: string
  limit: number
}
export interface ItemCatalogParams extends CatalogParams {
  type: number
  characterId: string
  rarity: number
}
export interface IdolCardSkinCatalogParams extends CatalogParams {
  characterId: string
  rarity: number
  planType: number
  idolCardIds: string[]
}
export interface SupportCardCatalogParams extends CatalogParams {
  rarity: number
  type: number
}
export interface CostumeCatalogParams extends CatalogParams {
  characterId: string
}
export interface GashaCatalogParams extends CatalogParams {
  open: boolean
}
export interface ProduceCatalogParams extends CatalogParams {
  produceGroupId: string
}

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

/** 目录接口；没有提供的种类不可查询。 */
export interface GakumasCatalogApi {
  currencies?: (req: CatalogParams) => Promise<CurrencyPage>
  items?: (req: ItemCatalogParams) => Promise<ItemEntryPage>
  idolCardSkins?: (req: IdolCardSkinCatalogParams) => Promise<IdolCardSkinPage>
  supportCards?: (req: SupportCardCatalogParams) => Promise<SupportCardEntryPage>
  characters?: (req: CatalogParams) => Promise<CharacterPage>
  costumes?: (req: CostumeCatalogParams) => Promise<CostumePage>
  costumeHeads?: (req: CostumeCatalogParams) => Promise<CostumePage>
  music?: (req: CatalogParams) => Promise<MusicPage>
  gashas?: (req: GashaCatalogParams) => Promise<GashaEntryPage>
  produceGroups?: (req: CatalogParams) => Promise<ProduceGroupPage>
  produces?: (req: ProduceCatalogParams) => Promise<ProduceEntryPage>
  seminars?: (req: CatalogParams) => Promise<SeminarEntryPage>
  towers?: (req: CatalogParams) => Promise<TowerEntryPage>
  produceCards?: (req: CatalogParams) => Promise<ProduceCardEntryPage>
  produceItems?: (req: CatalogParams) => Promise<ProduceItemEntryPage>
  memoryAbilities?: (req: CatalogParams) => Promise<MemoryAbilityPage>
}

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

const page = <T>(p: { items: T[]; nextCursor: string; total: number }, map: (x: T) => CatalogEntry[]): ListPage<CatalogEntry> => ({
  items: p.items.flatMap(map),
  nextCursor: p.nextCursor,
  total: p.total,
})

const empty: ListPage<CatalogEntry> = { items: [], nextCursor: '', total: 0 }

/** 查询一类目录（服务器 serverId，q / ids / characterId / rarity 由服务端筛选分页）。 */
export async function queryCatalog(api: GakumasCatalogApi, serverId: string, kind: string, query: CatalogQuery = {}): Promise<ListPage<CatalogEntry>> {
  const base = {
    serverId,
    q: query.q || '',
    ids: query.ids || [],
    cursor: query.cursor || '',
    limit: query.limit || 0,
  }
  const rarity = Number(query.rarity) || 0
  const characterId = query.characterId || ''
  switch (kind) {
    case 'currency':
      if (!api.currencies) return empty
      return page(await api.currencies(base), (e) => [{ id: e.id, name: e.name, resourceType: e.resourceType }])
    case 'item':
      if (!api.items) return empty
      return page(await api.items({ ...base, type: 0, characterId, rarity }), (e) => [
        { id: e.id, name: e.name, resourceType: e.resourceType, rarity: e.rarity, type: e.type, characterId: e.characterId, assetId: e.assetId },
      ])
    case 'idolCardSkin':
      if (!api.idolCardSkins) return empty
      return page(await api.idolCardSkins({ ...base, characterId, rarity, planType: 0, idolCardIds: [] }), (e) => [
        {
          id: e.id,
          name: e.name,
          resourceType: e.resourceType,
          rarity: e.rarity,
          type: e.planType,
          characterId: e.characterId,
          parentId: e.idolCardId,
          assetId: e.assetId,
        },
      ])
    case 'supportCard':
      if (!api.supportCards) return empty
      return page(await api.supportCards({ ...base, rarity, type: 0 }), (e) => [
        { id: e.id, name: e.name, resourceType: e.resourceType, rarity: e.rarity, type: e.type, assetId: e.assetId },
      ])
    case 'character':
      if (!api.characters) return empty
      return page(await api.characters(base), (e) => [{ id: e.id, name: e.name, resourceType: 0 }])
    case 'costume':
    case 'costumeHead': {
      const fn = kind === 'costume' ? api.costumes : api.costumeHeads
      if (!fn) return empty
      return page(await fn({ ...base, characterId }), (e) => [{ id: e.id, name: e.name, resourceType: e.resourceType, characterId: e.characterId }])
    }
    case 'music':
      if (!api.music) return empty
      return page(await api.music(base), (e) => [{ id: e.id, name: e.name, resourceType: e.resourceType, assetId: e.jacketAssetId }])
    case 'gasha':
      if (!api.gashas) return empty
      return page(await api.gashas({ ...base, open: false }), (e) => [
        { id: e.id, name: e.name, resourceType: 0, type: e.type, assetId: e.bannerAssetId, startTime: e.startTime, endTime: e.endTime },
      ])
    case 'produceGroup':
      if (!api.produceGroups) return empty
      return page(await api.produceGroups(base), (e) => [{ id: e.id, name: e.name, resourceType: 0, type: e.type, assetId: e.assetId }])
    case 'produce':
      if (!api.produces) return empty
      return page(await api.produces({ ...base, produceGroupId: '' }), (e) => [
        { id: e.id, name: e.name, resourceType: 0, parentId: e.produceGroupId, assetId: e.assetId },
      ])
    case 'seminar':
      if (!api.seminars) return empty
      return page(await api.seminars(base), (g) => [
        { id: g.id, name: g.name, resourceType: 0 },
        ...g.exams.map((x) => ({ id: x.id, name: x.name, resourceType: 0, parentId: g.id })),
      ])
    case 'tower':
      if (!api.towers) return empty
      return page(await api.towers(base), (e) => [{ id: e.id, name: e.id, resourceType: 0, type: e.layers }])
    case 'produceCard':
      if (!api.produceCards) return empty
      return page(await api.produceCards(base), (e) => [
        {
          id: e.id,
          name: e.name,
          upgradedName: e.upgradedName,
          resourceType: 0,
          rarity: e.rarity,
          type: e.planType,
          assetId: e.assetId,
          characterAsset: e.characterAsset,
        },
      ])
    case 'produceItem':
      if (!api.produceItems) return empty
      return page(await api.produceItems(base), (e) => [
        { id: e.id, name: e.name, resourceType: 0, rarity: e.rarity, assetId: e.assetId, description: e.description },
      ])
    case 'memoryAbility':
      if (!api.memoryAbilities) return empty
      return page(await api.memoryAbilities(base), (e) => [{ id: e.id, name: e.id, resourceType: 0, rarity: e.rarity, levels: e.levels }])
  }
  return empty
}
