// 学マス 素材包 key ↔ 配置里的客户端资源名（assetId）。
// 素材包只按规则导出了部分资源；这里只列出能从 key 准确还原成 assetId 的分组，选出来写进配置的一定是客户端真实存在的资源名。
import { gakumas } from '@/games'

export interface AssetSource {
  /** 标签 id */
  key: string
  /** 标签名 */
  label: string
  /** 素材分组（listGameAssets 的 group） */
  group: string
  /** 分组里哪些名称属于这一类 */
  filter: (name: string) => boolean
  /** 素材名 → 配置里的 assetId */
  toAssetId: (name: string) => string
  /** 素材属于游戏里的哪一项（用它的名称显示素材；素材名本身不显示） */
  owner?: { kind: string; id: (name: string) => string }
  /** 固定的候选（不读素材包），例如配置里已经用过的活动横幅 */
  items?: { assetId: string; url: string; label: string }[]
}

/** 固定候选的来源：配置里已经用过、确定存在的图片。 */
export const fixedSource = (key: string, label: string, items: { assetId: string; url: string; label: string }[]): AssetSource => ({
  key,
  label,
  group: '',
  filter: () => true,
  toAssetId: (name) => name,
  items,
})

/** 登录奖励标题图：ui/login-<名称> ← img_login_title_<名称> */
export const loginTitleSource = (label: string): AssetSource => ({
  key: 'loginTitle',
  label,
  group: 'ui',
  filter: (name) => name.startsWith('login-'),
  toAssetId: (name) => `img_login_title_${name.slice('login-'.length)}`,
})

/** 卡池横幅：gasha/<卡池 ID> ← img_gasha_banner_<卡池 ID>-select */
export const gashaBannerSource = (label: string): AssetSource => ({
  key: 'gashaBanner',
  label,
  group: 'gasha',
  filter: (name) => !name.startsWith('bg-'),
  toAssetId: (name) => `img_gasha_banner_${name}-select`,
  owner: { kind: 'gasha', id: (name) => name },
})

/** 卡池背景：gasha/bg-<名称> ← img_general_gasha_bg_<名称> */
export const gashaBackgroundSource = (label: string): AssetSource => ({
  key: 'gashaBackground',
  label,
  group: 'gasha',
  filter: (name) => name.startsWith('bg-'),
  toAssetId: (name) => `img_general_gasha_bg_${name.slice(3)}`,
})

/** 公告横幅：events/notice-<卡池 ID> ← img_notice_gasha_<卡池 ID>-banner-01-large */
export const noticeBannerSource = (label: string): AssetSource => ({
  key: 'noticeBanner',
  label,
  group: 'events',
  filter: (name) => name.startsWith('notice-'),
  toAssetId: (name) => `img_notice_gasha_${name.slice('notice-'.length)}-banner-01-large`,
  owner: { kind: 'gasha', id: (name) => name.slice('notice-'.length) },
})

/** 公告横幅 / 活动图的显示图片；不认识的资源名返回空串（不显示图片）。 */
export function bannerImage(assetId: string): string {
  if (!assetId) return ''
  if (assetId.startsWith('img_notice_gasha_')) return gakumas.noticeGashaBanner(assetId)
  if (assetId.startsWith('img_gasha_banner_')) return gakumas.gashaBanner(assetId)
  if (assetId.startsWith('img_general_event_')) return gakumas.eventImage(assetId)
  if (assetId.startsWith('img_general_gasha_bg_')) return gakumas.gashaBg(assetId)
  if (assetId.startsWith('img_login_title_')) return gakumas.loginTitle(assetId)
  return ''
}

/** 配置里已经用过的图（去重）：显示图，素材包里没有的显示用在哪一项上。 */
export const usedSource = (key: string, label: string, entries: { assetId?: string; owner: string }[]): AssetSource => {
  const seen = new Map<string, string>()
  for (const e of entries) if (e.assetId && !seen.has(e.assetId)) seen.set(e.assetId, e.owner)
  return fixedSource(key, label, [...seen].map(([assetId, owner]) => ({ assetId, url: bannerImage(assetId), label: owner })))
}
