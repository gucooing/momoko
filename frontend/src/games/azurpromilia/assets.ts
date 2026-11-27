// 蓝色星原素材 key 换算：把游戏表 ID 换成素材包的 key（图片由 scripts/games/azurpromilia/extract-assets.py 从客户端导出）。
// 分组：heroes/<英雄 ID>、pets/<奇波 ID>、items/<种类表名>/<ID>（表名同 reward-types.json 的 kind，例如 items/common_item/9000）、
// heads/<名片头像部件 ID>、heads/default-<性别>、covers/loading-<ID>（横幅大图）、app/icon（游戏的应用图标）。
// 每个 *Key 函数只做字符串换算，同名的 *Image 函数返回图片 URL（没有图片或索引还在加载时是占位图）。
// 只依赖 vue、core.ts 与 reward-types.json：管理端和用户通行证共用。
import { shallowRef } from 'vue'
import { gameAsset, hasGameAsset, loadGameAssetGroup } from '../core'
import rewardTypes from './reward-types.json'

const GAME = 'azurpromilia'

type Id = number | string

const kindByType = new Map<number, string>(rewardTypes.map((k) => [k.type, k.kind]))

// 不按 items/<表名> 分组的种类：英雄、奇波单独分组。
// 声望（表名 groups）的 ID 是势力 ID（服务端按势力校验），没有导出图标，显示占位图。
const GROUP_OF_KIND: Record<string, string> = {
  hero: 'heroes',
  pet: 'pets',
}

/** 英雄头像 key：heroes/101003（英雄列表用的小头像）。 */
export function heroKey(heroId: Id): string {
  return `heroes/${heroId}`
}

export function heroImage(heroId: Id): string {
  return gameAsset(GAME, heroKey(heroId))
}

/** 奇波图标 key：pets/500001。 */
export function petKey(petId: Id): string {
  return `pets/${petId}`
}

export function petImage(petId: Id): string {
  return gameAsset(GAME, petKey(petId))
}

/** 某一种类表里的图标 key：items/common_item/9000。kind 为种类表名（reward-types.json 的 kind）。 */
export function itemKey(kind: string, id: Id): string {
  return `items/${kind}/${id}`
}

export function itemImage(kind: string, id: Id): string {
  return gameAsset(GAME, itemKey(kind, id))
}

/** 奖励图标 key：type 为游戏 reward_type 的数值；没有图标分组的种类返回空串。 */
export function rewardKey(type: number, id: Id): string {
  const kind = kindByType.get(type)
  if (!kind) return ''
  const group = GROUP_OF_KIND[kind]
  return group ? `${group}/${id}` : itemKey(kind, id)
}

/** 奖励图标 URL（邮件附件、道具、货币等）；这一种类或这个 ID 没有图标时是占位图。 */
export function rewardImage(type: number, id: Id): string {
  return gameAsset(GAME, rewardKey(type, id))
}

// ---------- 玩家头像 ----------

/**
 * 玩家头像 key：名片的头像部件（playercard_dress 的头像类）有图时为 heads/<部件 ID>，
 * 否则（默认头像部件、没有部件、头像索引还在加载）为按性别的默认头像 heads/default-<sex>：sex 为 1 时是男，其余按女（2）。
 * 读的是 heads 分组索引，索引加载完成后用到它的组件自动重新渲染。
 */
export function headKey(head: Id, sex: number): string {
  const own = `heads/${head}`
  return hasGameAsset(GAME, own) ? own : `heads/default-${sex === 1 ? 1 : 2}`
}

export function headImage(head: Id, sex: number): string {
  return gameAsset(GAME, headKey(head, sex))
}

// ---------- 应用图标 ----------

/** 游戏的应用图标（游戏客户端 exe 的图标，256 px 方图）。 */
export function appIconImage(): string {
  return gameAsset(GAME, 'app/icon')
}

// ---------- 封面大图 ----------

// covers 分组的名称（已排序）；undefined = 还没加载
const coverNames = shallowRef<string[] | undefined>()
let coverLoading = false

/** covers 分组的全部 key（covers/loading-1 …）；索引加载前为空数组并开始加载，加载完成后用到它的组件自动重新渲染。 */
export function coverKeys(): string[] {
  const names = coverNames.value
  if (!names && !coverLoading) {
    coverLoading = true
    void loadGameAssetGroup(GAME, 'covers').then((index) => {
      coverLoading = false
      if (index) coverNames.value = Object.keys(index).sort()
    })
  }
  return (names ?? []).map((name) => `covers/${name}`)
}

/** 封面 key：按 seed（例如服务器 ID）稳定地挑一张，同一个 seed 每次都是同一张；索引加载前为空串。 */
export function coverKey(seed = ''): string {
  const keys = coverKeys()
  if (!keys.length) return ''
  let h = 0
  for (const c of seed) h = (h * 31 + c.charCodeAt(0)) >>> 0
  return keys[h % keys.length] ?? ''
}

/** 封面 URL（横图，宽 1280 以内）；索引加载前或没有封面时是占位图。 */
export function coverImage(seed = ''): string {
  return gameAsset(GAME, coverKey(seed))
}
