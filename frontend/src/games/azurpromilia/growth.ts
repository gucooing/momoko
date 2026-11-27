// 蓝色星原的养成范围（growth.json 由 scripts/games/azurpromilia/export-growth.py 从游戏服务端 res 导出，规则与服务端 gdconf 一致）：
// GM 修改角色、奇波时把超出范围的值改成最大或最小值；突破由等级推出（等级允许的最高一档）。
import growth from './growth.json'

type Range = [number, number]

const clamp = (v: number, [lo, hi]: Range) => Math.min(hi, Math.max(lo, v))

export const heroLevelRange = growth.heroLevel as Range
export const petLevelRange = growth.petLevel as Range
export const favorRange = growth.favorLevel as Range

export const clampHeroLevel = (v: number) => clamp(v, heroLevelRange)
export const clampPetLevel = (v: number) => clamp(v, petLevelRange)
export const clampFavor = (v: number) => clamp(v, favorRange)

export interface HeroGrowth {
  /** 最低突破 */
  rankMin: number
  /** rankNeeds[i]：升到 rankMin + i 需要的等级 */
  rankNeeds: number[]
  grade: Range
}

const heroes = growth.heroes as unknown as Record<string, [number, number[], number, number]>

export function heroGrowth(heroId: number): HeroGrowth | undefined {
  const h = heroes[heroId]
  return h && { rankMin: h[0], rankNeeds: h[1], grade: [h[2], h[3]] }
}

/** 等级 level 允许的最高突破。 */
export function rankForLevel(h: HeroGrowth, level: number): number {
  let rank = h.rankMin
  h.rankNeeds.forEach((need, i) => {
    if (need <= level) rank = h.rankMin + i
  })
  return rank
}

export const clampGrade = (h: HeroGrowth, v: number) => clamp(v, h.grade)

// ---------- 奇波星魂 ----------

export interface StarState {
  stage: number
  node: number
}

export interface Starsoul {
  /** 按推进顺序的状态：各阶的全部节点，最后一阶只有第 1 节点 */
  states: StarState[]
  /** needs[i]：到达 states[i] 需要的奇波等级 */
  needs: number[]
}

const templates = growth.starsoul as unknown as Record<string, [number[], number[]]>
const petTemplates = growth.pets as Record<string, number>

/** 奇波的星魂模板；没有星魂的奇波为 undefined。 */
export function petStarsoul(petId: number): Starsoul | undefined {
  const tpl = templates[petTemplates[petId] ?? '']
  if (!tpl) return undefined
  const [counts, needs] = tpl
  const states: StarState[] = []
  counts.forEach((n, i) => {
    const last = i === counts.length - 1
    for (let node = 1; node <= (last ? 1 : n); node++) states.push({ stage: i + 1, node })
  })
  return { states, needs }
}

/** 把星魂 (stage, node) 改成模板里能到达、且等级 level 够得着的状态：太大取最远的，太小取第 1 阶第 1 节点。 */
export function clampStar(s: Starsoul, stage: number, node: number, level: number): StarState {
  const last = s.states.length - 1
  let i = s.states.findIndex((x) => x.stage > stage || (x.stage === stage && x.node >= node))
  if (i < 0) i = last
  else if (s.states[i]!.stage > stage && i > 0 && stage >= 1) i -= 1
  let reach = 0
  s.needs.forEach((need, j) => {
    if (need <= level) reach = j
  })
  return s.states[Math.min(i, reach)]!
}
