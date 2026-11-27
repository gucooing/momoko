// 游戏素材与游戏皮肤的入口：管理端和用户通行证前端共用。
// gameAsset('gakumas', key) 返回游戏图片的 URL（静态文件，分组索引按需加载），查不到或还在加载时返回占位图；
// 学マス 的 ID → key 换算在 gakumas 命名空间里（gakumas.idolCardImage(skinId) 等）。
import './gakumas/theme.css'
import './azurpromilia/theme.css'
import { gameAsset, listGameAssets, type GameAssetEntry } from './core'
import {
  catalogEntryImage as gakumasCatalogImage,
  coverKey as gakumasCoverKey,
  idolCardImage as gakumasIdolCardImage,
  idolCardKey as gakumasIdolCardKey,
} from './gakumas/assets'
import * as azurAssets from './azurpromilia/assets'

export {
  gameAsset,
  gamePlaceholder,
  hasGameAsset,
  listGameAssets,
  loadGameAssetGroup,
  probeGameAssets,
  setGameAssetBase,
  type GameAssetEntry,
  type GameAssetProbe,
} from './core'
export * as gakumas from './gakumas/assets'
export * as azurpromilia from './azurpromilia/assets'
export {
  ensureGameNames,
  gameName,
  gameNameVariants,
  loadGameNames,
  localizeEntries,
  lookupGameName,
  nameLangOf,
  searchGameNames,
} from './names'
export { fuzzyMatch, normalizeName } from './search'
export { isVideoLink } from './serverMedia'

/**
 * 游戏的默认封面图 URL（游戏卡片、没设封面的服务器卡片用）：横图 2:1；
 * seed（例如服务器 ID）不同时在几张里稳定地挑一张。这款游戏没有素材时返回空串。
 */
export function gameCover(game: string, seed = ''): string {
  switch (game) {
    case 'gakumas':
      return gameAsset(game, gakumasCoverKey(seed))
    case 'azurpromilia':
      return azurAssets.coverImage(seed)
    default:
      return ''
  }
}

/** 游戏图标 URL（服务器没有上传应用图标时的游戏标记）：蓝色星原为游戏客户端的应用图标；这款游戏没有图标时返回空串。 */
export function gameIcon(game: string): string {
  switch (game) {
    case 'azurpromilia':
      return azurAssets.appIconImage()
    default:
      return ''
  }
}

/** 服务器可选的封面（编辑服务器时选）：学マス 为偶像卡横图，蓝色星原为游戏的主视觉与卡池横幅。 */
export async function gameCoverAssets(game: string): Promise<GameAssetEntry[]> {
  switch (game) {
    case 'gakumas':
      return (await listGameAssets(game, 'cards/idol')).filter((a) => a.key.endsWith('-land'))
    case 'azurpromilia':
      return listGameAssets(game, 'covers')
    default:
      return []
  }
}

// 蓝色星原的头像：通行证绑定记录与玩家条目里是「名片头像部件:性别」（如 10102:2）
const azurHead = (asset: string): [string, number] => {
  const [head = '', sex = ''] = asset.split(':')
  return [head, Number(sex)]
}

/** 游戏账号的头像 URL（通行证绑定记录的 avatarAsset）：学マス 为偶像卡方图，蓝色星原为名片头像；没有时为空串。 */
export function gameLinkAvatar(game: string, asset?: string): string {
  if (!asset) return ''
  switch (game) {
    case 'gakumas':
      return gakumasIdolCardImage(asset, 'sq')
    case 'azurpromilia':
      return azurAssets.headImage(...azurHead(asset))
    default:
      return ''
  }
}

/** 游戏账号头像对应的素材（<游戏>/<key>，通行证资料里可选为头像）；没有时为空串。 */
export function gameLinkAvatarAsset(game: string, asset?: string): string {
  if (!asset) return ''
  switch (game) {
    case 'gakumas':
      return `gakumas/${gakumasIdolCardKey(asset, 'sq')}`
    case 'azurpromilia':
      return `azurpromilia/${azurAssets.headKey(...azurHead(asset))}`
    default:
      return ''
  }
}

/** 游戏目录条目（MGP /catalogs/{kind}）的图片；这款游戏或这一类没有图片时返回空串。 */
export function gameCatalogImage(game: string, kind: string, entry: { id: string; assetId?: string; characterId?: string }): string {
  switch (game) {
    case 'gakumas':
      return gakumasCatalogImage(kind, entry)
    default:
      return ''
  }
}
