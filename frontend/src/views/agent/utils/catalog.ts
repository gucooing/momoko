// 模型目录（AgModelCatalog）里的一行：第三方接口的模型目录和官方账户读到的模型都用它。
export interface CatalogLevel {
  uid: number
  /** 推理等级的名字（会话里能选的） */
  name: string
  /** 选中它时发给接口的参数值（空 = 不带这个参数） */
  value: string
}

export interface CatalogRow {
  id: string
  /** 显示名（官方账户读到的模型有；没有就显示 id） */
  name?: string
  levels: CatalogLevel[]
  /** 最长上下文、最长输出（token）；空串 = 没设 */
  contextWindow: number | ''
  maxOutput: number | ''
}

// 推理等级从低到高；「添加一级」按这个顺序带出下一档
export const LADDER = ['off', 'low', 'medium', 'high', 'xhigh', 'max']
const DEFAULT_LEVELS = ['off', 'low', 'medium', 'high']

let uidSeed = 0
export const levelRow = (name: string, value: string): CatalogLevel => ({ uid: ++uidSeed, name, value })
// off 的参数值留空 = 不带这个参数，其它默认和等级名相同
export const presetLevel = (name: string) => levelRow(name, name === 'off' ? '' : name)
export const defaultLevels = () => DEFAULT_LEVELS.map(presetLevel)

// 上下文、输出的读数：1000000 → 1M，128000 → 128K（多余的 0 不显示）
export const fmtLimit = (value: number | '') => {
  const n = Number(value) || 0
  if (n >= 1_000_000) return `${+(n / 1_000_000).toFixed(2)}M`
  if (n >= 1000) return `${+(n / 1000).toFixed(1)}K`
  return String(n)
}
