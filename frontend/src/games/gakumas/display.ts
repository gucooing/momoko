// 学マス 详情展示的公共部分（管理端与用户通行证共用）：协议数值的标签、记忆列表的评价分类，
// 以及详情组件需要调用方提供的目录查询。
import type { CatalogEntry } from '../mgp'

// ResultGrade：1 F … 8 A、10 S、12 SS、14 SSS、16 SSSS、18 SSSSS，奇数为对应的 +
const GRADES = ['', 'F', 'E', 'D', 'C', 'C+', 'B', 'B+', 'A', 'A+', 'S', 'S+', 'SS', 'SS+', 'SSS', 'SSS+', 'SSSS', 'SSSS+', 'SSSSS', 'SSSSS+']

export const gradeLabel = (grade: number | null | undefined) => (grade ? GRADES[grade] || String(grade) : '—')

// Rarity：1 R、2 SR、3 SSR
const RARITIES = ['', 'R', 'SR', 'SSR']

export const rarityLabel = (rarity: number | null | undefined) => (rarity ? RARITIES[rarity] || String(rarity) : '')

/** 列表筛选里的稀有度（从高到低）。 */
export const RARITY_VALUES = [3, 2, 1]

/** 记忆、培育记录按评价筛选的分类：S 起每一档一类（含 +），A+ 及以下一类；grades 为交给服务端的 ResultGrade。 */
export interface GradeGroup {
  key: string
  label: string
  grades: number[]
}

export const GRADE_GROUPS: GradeGroup[] = [
  { key: 'SSSSS', label: 'SSSSS', grades: [18, 19] },
  { key: 'SSSS', label: 'SSSS', grades: [16, 17] },
  { key: 'SSS', label: 'SSS', grades: [14, 15] },
  { key: 'SS', label: 'SS', grades: [12, 13] },
  { key: 'S', label: 'S', grades: [10, 11] },
  { key: 'A', label: '≤ A+', grades: [1, 2, 3, 4, 5, 6, 7, 8, 9] },
]

/** 一个评价分类的 ResultGrade（空串或未知分类为全部）。 */
export const gradeGroupGrades = (key: string) => GRADE_GROUPS.find((g) => g.key === key)?.grades ?? []

/** 按评价的条数（列表 summary.grades）合计到分类。 */
export const gradeGroupCount = (counts: Record<number, number> | undefined, group: GradeGroup) =>
  group.grades.reduce((n, g) => n + (counts?.[g] || 0), 0)

/** 按稀有度或评价的条数合计。 */
export const countTotal = (counts: Record<number, number> | undefined) => Object.values(counts || {}).reduce((n, v) => n + v, 0)

/**
 * 回忆照片、名片图片（接口 MemoryPhoto / MeishiImage）→ 图片地址：接口 JSON 里 bytes 字段是 base64 字符串（生成的类型写作 Uint8Array）。
 */
export const imageDataUrl = (p: { contentType: string; data: unknown }) => `data:${p.contentType || 'image/jpeg'};base64,${String(p.data || '')}`

/** 记忆中技能卡的获得时机（ProduceMemoryProduceCardPhaseType）：1 培育开始时、2 期中考试结束后。 */
export type MemoryCardPhase = 1 | 2

/** 详情组件用到的目录查询：两端各自的目录缓存（管理端 useCatalogLookup，通行证 GameContext）。 */
export interface GakumasDetailLookup {
  /** 名称（前端名称表 → 服务端目录）：查不到或还在查询时为空串 */
  name: (kind: string, id: string) => string
  /** 目录条目：查询中为 undefined，查不到为 null */
  entry: (kind: string, id: string) => CatalogEntry | null | undefined
}
