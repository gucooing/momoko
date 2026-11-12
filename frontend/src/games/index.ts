// 游戏素材与游戏皮肤的入口：管理端和用户通行证前端共用。
// gameAsset('gakumas', key) 返回游戏图片的 URL（静态文件，分组索引按需加载），查不到或还在加载时返回占位图；
// 学マス 的 ID → key 换算在 gakumas 命名空间里（gakumas.idolCardImage(skinId) 等）。
import './gakumas/theme.css'
import { catalogEntryImage as gakumasCatalogImage } from './gakumas/assets'

export {
  gameAsset,
  gamePlaceholder,
  hasGameAsset,
  listGameAssets,
  loadGameAssetGroup,
  type GameAssetEntry,
} from './core'
export * as gakumas from './gakumas/assets'
export { ensureGameNames, gameName, gameNameVariants, loadGameNames, localizeEntries, lookupGameName, nameLangOf, searchGameNames } from './names'
export { fuzzyMatch, normalizeName } from './search'

/** 游戏目录条目（MGP /catalogs/{kind}）的图片；这款游戏或这一类没有图片时返回空串。 */
export function gameCatalogImage(game: string, kind: string, entry: { id: string; assetId?: string }): string {
  switch (game) {
    case 'gakumas':
      return gakumasCatalogImage(kind, entry)
    default:
      return ''
  }
}
