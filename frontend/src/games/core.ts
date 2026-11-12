// 游戏素材：图片是前端自带的静态文件（scripts/games/<game>/ 导出到 public/games/<game>/），从不向游戏服务端取图。
//
// 按需加载，不一次性全部加载：
// - 图片不进 Vite 的模块图，文件名带内容哈希（<name>.<hash>.webp），可以长期缓存；
// - 每个分组（chara、cards/idol、gasha…）有一份很小的「名称 → 哈希」索引，页面第一次用到这个分组时才加载；
// - gameAsset() 是同步的：索引还没加载时先返回占位图并触发加载，加载完成后用到它的组件自动重新渲染；
// - 浏览器只请求真正显示出来的图片（配合 <GameImage> 的 loading="lazy"）。
import { reactive } from 'vue'
import placeholder from './placeholder.svg?url'

type GroupIndex = Record<string, string>

interface AssetPack {
  /** 图片目录（带部署前缀），例如 /games/gakumas/ */
  base: string
  /** 分组索引的加载函数，键为 ./<game>/assets/<group 中的 / 换成 ->.json */
  loaders: Record<string, () => Promise<GroupIndex>>
}

const indexLoaders = import.meta.glob<GroupIndex>('./*/assets/*.json', { import: 'default' })

const base = `${import.meta.env.BASE_URL || '/'}games/`.replace(/\/{2,}/g, '/')

const packs: Record<string, AssetPack> = {}
for (const [path, loader] of Object.entries(indexLoaders)) {
  const m = /^\.\/([^/]+)\/assets\/([^/]+)\.json$/.exec(path)
  if (!m) continue
  const [, game, file] = m as unknown as [string, string, string]
  packs[game] ??= { base: `${base}${game}/`, loaders: {} }
  packs[game].loaders[file] = loader
}

// 已加载的分组索引（响应式：加载完成后，渲染时读过它的组件会重新渲染）。
const loaded = reactive(new Map<string, GroupIndex>())
const pending = new Map<string, Promise<GroupIndex | undefined>>()

const groupFile = (group: string) => group.replace(/\//g, '-')

function splitKey(key: string): [string, string] | undefined {
  const i = key.lastIndexOf('/')
  if (i <= 0) return undefined
  return [key.slice(0, i), key.slice(i + 1)]
}

/** 加载某个分组的索引（只加载一次；分组不存在时返回 undefined）。 */
export function loadGameAssetGroup(game: string, group: string): Promise<GroupIndex | undefined> {
  const id = `${game}:${group}`
  const done = loaded.get(id)
  if (done) return Promise.resolve(done)
  let p = pending.get(id)
  if (!p) {
    const loader = packs[game]?.loaders[groupFile(group)]
    p = loader
      ? loader()
          .then((index) => {
            loaded.set(id, index)
            return index
          })
          .catch(() => {
            pending.delete(id)
            return undefined
          })
      : Promise.resolve(undefined)
    pending.set(id, p)
  }
  return p
}

function lookup(game: string, key: string): string | undefined | null {
  const parts = splitKey(key)
  const pack = packs[game]
  if (!parts || !pack) return undefined
  const [group, name] = parts
  const index = loaded.get(`${game}:${group}`)
  if (!index) {
    void loadGameAssetGroup(game, group)
    return null // 索引加载中
  }
  const hash = index[name]
  return hash ? `${pack.base}${group}/${name}.${hash}.webp` : undefined
}

/** 占位图 URL（素材包里没有的 key，或分组索引还在加载）。 */
export const gamePlaceholder: string = placeholder

/** 素材包里是否有这张图（分组索引未加载时为 false，并开始加载）。 */
export function hasGameAsset(game: string, key: string): boolean {
  return typeof lookup(game, key) === 'string'
}

/** 图片 URL；游戏或 key 不存在、或分组索引还在加载时返回占位图。 */
export function gameAsset(game: string, key: string): string {
  return lookup(game, key) || placeholder
}

export interface GameAssetEntry {
  key: string
  url: string
}

/**
 * 按分组列出素材（例如 'gasha'、'events'），供「从游戏素材选」之类的选择器用。
 * 只加载这个分组的索引；prefix 可进一步按名称前缀过滤。图片本身由选择器按需显示。
 */
export async function listGameAssets(game: string, group: string, prefix = ''): Promise<GameAssetEntry[]> {
  const pack = packs[game]
  const index = await loadGameAssetGroup(game, group)
  if (!pack || !index) return []
  return Object.keys(index)
    .filter((name) => name.startsWith(prefix))
    .sort()
    .map((name) => ({ key: `${group}/${name}`, url: `${pack.base}${group}/${name}.${index[name]}.webp` }))
}
