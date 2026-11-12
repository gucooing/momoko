// 学マス 素材 key 换算：把 MGP 返回的 ID / assetId 换成素材包的 key（见 assets/index.json）。
// 每个 *Key 函数只做字符串换算，同名的 *Image 函数返回图片 URL（查不到时是占位图）。
import { gameAsset } from '../core'

const GAME = 'gakumas'

function after(value: string, prefix: string): string | undefined {
  return value.startsWith(prefix) ? value.slice(prefix.length) : undefined
}

// ---------- 偶像卡 / 支援卡 ----------

export type CardVariant = 'sq' | 'land'

// {角色}-{稀有度}-{编号}，例如 hski-3-000
const IDOL_CARD_RE = /(?:^|[-_])([a-z][a-z0-9]*-\d-\d{3})(?:_|$)/
// {稀有度}-{编号}，例如 2-0001
const SUPPORT_CARD_RE = /(?:^|[-_])(\d-\d{4})(?:_|$)/

/**
 * 偶像卡 key：cards/idol/hski-3-000-sq。
 * 接受偶像卡皮肤 ID（i_card-skin-hski-3-000，换皮卡用皮肤 ID 才是对的图）、偶像卡 ID（i_card-hski-3-000，
 * 即初始皮肤）、assetId（cidol-hski-3-000）、资源名（img_general_cidol-hski-3-000_1-thumb-square）
 * 和偶像卡碎片道具 ID（item-piece-idol_hski-3-000）。
 */
export function idolCardKey(idOrAssetId: string, variant: CardVariant = 'sq'): string {
  const m = IDOL_CARD_RE.exec(idOrAssetId)
  return `cards/idol/${m ? m[1] : idOrAssetId}-${variant}`
}

export function idolCardImage(idOrAssetId: string, variant: CardVariant = 'sq'): string {
  return gameAsset(GAME, idolCardKey(idOrAssetId, variant))
}

/** 支援卡 key：cards/support/2-0001-sq。接受 s_card-2-0001、csprt-2-0001、资源名或 2-0001。 */
export function supportCardKey(idOrAssetId: string, variant: CardVariant = 'sq'): string {
  const m = SUPPORT_CARD_RE.exec(idOrAssetId)
  return `cards/support/${m ? m[1] : idOrAssetId}-${variant}`
}

export function supportCardImage(idOrAssetId: string, variant: CardVariant = 'sq'): string {
  return gameAsset(GAME, supportCardKey(idOrAssetId, variant))
}

// ---------- 道具 / 货币 ----------

/**
 * 道具 key：items/money。优先传道具目录（catalogs/item）的 assetId（item_money）：
 * 道具 ID 和图片名大多不同（item-coin-box-gasha-event-004 → item_event-box-story-009-gasha-coin）。
 * 只有道具 ID 时按 item-xxx → items/xxx 试（item-money 能对上，多数对不上）；
 * 偶像卡碎片（item-piece-idol_*，assetId 为空）用对应偶像卡的方形图。
 */
export function itemKey(idOrAssetId: string): string {
  const piece = after(idOrAssetId, 'item-piece-idol_')
  if (piece) return idolCardKey(piece, 'sq')
  const name =
    after(idOrAssetId, 'img_general_item_') ??
    after(idOrAssetId, 'item_') ??
    after(idOrAssetId, 'item-') ??
    idOrAssetId
  return `items/${name.toLowerCase()}`
}

export function itemImage(idOrAssetId: string): string {
  return gameAsset(GAME, itemKey(idOrAssetId))
}

/** catalogs/currency 的 ID（jewelFree / jewelPaid / actionPoint / exp），加上道具 item-money。 */
export type CurrencyKind = 'jewelFree' | 'jewelPaid' | 'actionPoint' | 'exp' | 'money'

const CURRENCY_KEYS: Record<CurrencyKind, string> = {
  jewelFree: 'items/jewel-free',
  jewelPaid: 'items/jewel-paid',
  actionPoint: 'items/ap',
  exp: 'items/user-exp',
  money: 'items/money',
}

export function currencyKey(kind: CurrencyKind): string {
  return CURRENCY_KEYS[kind]
}

export function currencyImage(kind: CurrencyKind): string {
  return gameAsset(GAME, currencyKey(kind))
}

// ---------- 角色 ----------

export type CharaVariant = 'circle' | 'bust'

/** 角色 key：chara/hski（圆形头像）、chara/hski-bust（半身像）。接受角色 ID 或资源名 img_chr_hski_00-*。 */
export function charaKey(characterId: string, variant: CharaVariant = 'circle'): string {
  const m = /^img_chr_([a-z0-9]+)_/.exec(characterId)
  const id = m ? m[1] : characterId
  return variant === 'bust' ? `chara/${id}-bust` : `chara/${id}`
}

export function charaImage(characterId: string, variant: CharaVariant = 'circle'): string {
  return gameAsset(GAME, charaKey(characterId, variant))
}

// ---------- 卡池 ----------

/**
 * 卡池横幅 key：gasha/gasha-01055-pickup-001。优先传卡池的 assetId / bannerAssetId
 * （img_gasha_banner_gasha-00000-standard-001-select）：有的卡池复用别的横幅
 * （gasha-00000-standard-004 用 standard-001），ID 写法也可能和图片名不同。
 */
export function gashaBannerKey(gashaIdOrAssetId: string): string {
  const m = /^img_gasha_banner_(.+?)-select(?:-[ms])?$/.exec(gashaIdOrAssetId)
  return `gasha/${m ? m[1] : gashaIdOrAssetId}`
}

export function gashaBanner(gashaIdOrAssetId: string): string {
  return gameAsset(GAME, gashaBannerKey(gashaIdOrAssetId))
}

/** 卡池背景 key：gasha/bg-gasha_wednesday-001。接受卡池配置的 screenAssetId（img_general_gasha_bg_*）或去掉前缀的部分。 */
export function gashaBgKey(screenAssetId: string): string {
  const id =
    after(screenAssetId, 'img_general_gasha_bg_') ?? after(screenAssetId, 'bg-') ?? screenAssetId
  return `gasha/bg-${id}`
}

export function gashaBg(screenAssetId: string): string {
  return gameAsset(GAME, gashaBgKey(screenAssetId))
}

/** 卡池公告图 key：events/notice-gasha-01055-pickup-001。接受卡池 ID、卡池公告 ID（notice-<卡池 ID>）或资源名。 */
export function noticeGashaBannerKey(gashaOrNoticeId: string): string {
  const m = /^img_notice_gasha_(.+)-banner-01-large$/.exec(gashaOrNoticeId)
  if (m) return `events/notice-${m[1]}`
  return gashaOrNoticeId.startsWith('notice-')
    ? `events/${gashaOrNoticeId}`
    : `events/notice-${gashaOrNoticeId}`
}

export function noticeGashaBanner(gashaOrNoticeId: string): string {
  return gameAsset(GAME, noticeGashaBannerKey(gashaOrNoticeId))
}

// ---------- 活动 / 登录标题 ----------

export type EventPart = 'banner' | 'logo'

/**
 * 活动图 key：events/event-story-012-banner。资源名是 img_general_event_{类型}_{活动素材 ID}-{banner|logo}，
 * 配置里的 bannerAssetId / titleAssetId 直接传进来即可（自带 -banner / -logo，忽略 part）；
 * 也可以传活动素材 ID（event-story-012）加 part。注意活动 ID 和素材 ID 不一定相同
 * （panel_mission-event-007 的横幅是 event-panel-mission-007），有 assetId 时用 assetId。
 */
export function eventKey(eventIdOrAssetId: string, part: EventPart = 'banner'): string {
  const m = /^img_general_event_[a-z-]+_(.+)$/.exec(eventIdOrAssetId)
  if (m) return `events/${m[1]}`
  return /-(?:banner|logo)$/.test(eventIdOrAssetId)
    ? `events/${eventIdOrAssetId}`
    : `events/${eventIdOrAssetId}-${part}`
}

export function eventImage(eventIdOrAssetId: string, part: EventPart = 'banner'): string {
  return gameAsset(GAME, eventKey(eventIdOrAssetId, part))
}

/** 登录标题 key：ui/login-event2d-000。接受 event2d-000、login-event2d-000 或 img_login_title_event2d-000。 */
export function loginTitleKey(id: string): string {
  const name = after(id, 'img_login_title_') ?? after(id, 'login-') ?? id
  return `ui/login-${name}`
}

export function loginTitle(id: string): string {
  return gameAsset(GAME, loginTitleKey(id))
}

// ---------- 目录条目 ----------

/** MGP 目录条目的图片（GET /catalogs/{kind} 的条目）；这一类没有图片时返回空串。 */
export function catalogEntryImage(kind: string, entry: { id: string; assetId?: string }): string {
  const ref = entry.assetId || entry.id
  switch (kind) {
    case 'idolCardSkin':
      return idolCardImage(ref)
    case 'supportCard':
      return supportCardImage(entry.id)
    case 'item':
      return itemImage(ref)
    case 'currency':
      return entry.id in CURRENCY_KEYS ? currencyImage(entry.id as CurrencyKind) : ''
    case 'character':
      return charaImage(entry.id)
    case 'gasha':
      return gashaBanner(ref)
    default:
      return ''
  }
}
