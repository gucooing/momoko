// 游戏素材与游戏皮肤的入口：管理端和用户通行证前端共用。
// gameAsset('gakumas', key) 返回游戏图片的 URL（静态文件，分组索引按需加载），查不到或还在加载时返回占位图；
// 学マス 的 ID → key 换算在 gakumas 命名空间里（gakumas.idolCardImage(skinId) 等）。
import './gakumas/theme.css'
import { gameAsset } from './core'
import { catalogEntryImage as gakumasCatalogImage, coverKey as gakumasCoverKey } from './gakumas/assets'

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
