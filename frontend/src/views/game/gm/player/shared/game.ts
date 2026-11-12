// 学マス 协议里固定的枚举值 → 显示名（评价、稀有度、礼物来源），以及素材图片的换算。
// 只是协议数值的标签；卡、道具、角色等名称一律来自 MGP 目录（见 lookup.ts）。
import { gakumas, gameCatalogImage } from '@/games'

// ResultGrade：1 F … 8 A、10 S、12 SS、14 SSS、16 SSSS、18 SSSSS，奇数为对应的 +
const GRADES = ['', 'F', 'E', 'D', 'C', 'C+', 'B', 'B+', 'A', 'A+', 'S', 'S+', 'SS', 'SS+', 'SSS', 'SSS+', 'SSSS', 'SSSS+', 'SSSSS', 'SSSSS+']

export const gradeLabel = (grade: number | null | undefined) => (grade ? GRADES[grade] || String(grade) : '—')

// Rarity：1 R、2 SR、3 SSR
const RARITIES = ['', 'R', 'SR', 'SSR']

export const rarityLabel = (rarity: number | null | undefined) => (rarity ? RARITIES[rarity] || String(rarity) : '')

/** 礼物来源（sourceKind）→ 指令参数里的来源名；0 游戏内没有来源名。 */
const GIFT_SOURCE_NAMES: Record<number, 'gm' | 'mail' | 'code'> = { 1: 'gm', 2: 'mail', 3: 'code' }

export const giftSourceName = (kind: number) => GIFT_SOURCE_NAMES[kind]

/** 永久封禁的 bannedUntil（9223372036854775807）在 JS 里超出安全整数。 */
export const isPermanentBan = (until: number) => until >= Number.MAX_SAFE_INTEGER

/** MGP 目录条目的图片（没有图片的游戏 / 目录返回空串）。 */
export const catalogImage = (game: string, kind: string, entry: { id: string; assetId?: string }) =>
  entry.id ? gameCatalogImage(game, kind, entry) : ''

/** 偶像卡横图（名片、阵容大图）；只有学マス 有。 */
/** 偶像卡大图：land 横图（名片、阵容），full 原画（9:16，详情）；只有学マス 有。 */
export const idolCardArt = (game: string, idOrSkinId: string, variant: 'land' | 'full' = 'land') =>
  game === 'gakumas' && idOrSkinId ? gakumas.idolCardImage(idOrSkinId, variant) : ''

/** 支援卡原画（16:9，详情）；只有学マス 有。 */
export const supportCardArt = (game: string, id: string) => (game === 'gakumas' && id ? gakumas.supportCardImage(id, 'full') : '')

/** 角色半身像；只有学マス 有。 */
export const charaBust = (game: string, characterId: string) =>
  game === 'gakumas' && characterId ? gakumas.charaImage(characterId, 'bust') : ''
