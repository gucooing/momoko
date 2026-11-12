// 结构化对比两份配置（JSON 值）：对象按键对比；带 id 的对象数组按 id 对齐（gashas、lotteries、notices…），
// 基本类型数组按集合对比（增 / 删），其余数组按位置对比。结果用于「查看改动」与版本对比。

export type DiffKind = 'added' | 'removed' | 'changed'

export interface DiffEntry {
  /** 路径：对象键；按 id 对齐的数组元素写作元素 id；按位置对齐写作 #序号（从 1 开始） */
  path: string[]
  kind: DiffKind
  before?: unknown
  after?: unknown
  /** 按 id 对齐的数组只是元素顺序变了（before / after 是元素 id 的顺序） */
  order?: boolean
}

/** 数组元素的身份键（按顺序尝试）。 */
const ID_KEYS = ['id', 'storyEventId', 'pvpRateConfigId', 'eventId']

type Json = Record<string, unknown>

export function isPlainObject(v: unknown): v is Json {
  return !!v && typeof v === 'object' && !Array.isArray(v)
}

const isPrimitive = (v: unknown) => v === null || (typeof v !== 'object' && typeof v !== 'function')
const has = (o: Json, k: string) => Object.prototype.hasOwnProperty.call(o, k)

export function deepEqual(a: unknown, b: unknown): boolean {
  if (a === b) return true
  if (Array.isArray(a)) {
    if (!Array.isArray(b) || a.length !== b.length) return false
    for (let i = 0; i < a.length; i++) if (!deepEqual(a[i], b[i])) return false
    return true
  }
  if (isPlainObject(a)) {
    if (!isPlainObject(b)) return false
    const ka = Object.keys(a)
    const kb = Object.keys(b)
    if (ka.length !== kb.length) return false
    for (const k of ka) if (!has(b, k) || !deepEqual(a[k], b[k])) return false
    return true
  }
  return false
}

function idKeyOf(list: unknown[]): string | undefined {
  if (!list.length) return undefined
  for (const key of ID_KEYS) {
    const seen = new Set<string>()
    let ok = true
    for (const v of list) {
      const id = isPlainObject(v) ? v[key] : undefined
      if (typeof id !== 'string' || !id || seen.has(id)) {
        ok = false
        break
      }
      seen.add(id)
    }
    if (ok) return key
  }
  return undefined
}

/** 对比 a（之前）与 b（之后）；最多返回 limit 条。 */
export function diffJson(a: unknown, b: unknown, limit = 2000): DiffEntry[] {
  const out: DiffEntry[] = []

  const walk = (x: unknown, y: unknown, path: string[]) => {
    if (out.length >= limit || deepEqual(x, y)) return
    if (Array.isArray(x) && Array.isArray(y)) {
      const kx = idKeyOf(x)
      const ky = idKeyOf(y)
      const key = kx && (ky === kx || !y.length) ? kx : !x.length && ky ? ky : undefined
      if (key) {
        const before = new Map(x.map((v) => [String((v as Json)[key]), v]))
        const after = new Map(y.map((v) => [String((v as Json)[key]), v]))
        let moved = false
        for (const [id, v] of before) {
          if (!after.has(id)) out.push({ path: [...path, id], kind: 'removed', before: v })
          else walk(v, after.get(id), [...path, id])
        }
        for (const [id, v] of after) if (!before.has(id)) out.push({ path: [...path, id], kind: 'added', after: v })
        // 元素相同只是顺序变了
        const common = [...before.keys()].filter((id) => after.has(id))
        const commonAfter = [...after.keys()].filter((id) => before.has(id))
        moved = common.some((id, i) => commonAfter[i] !== id)
        if (moved) out.push({ path, kind: 'changed', before: common, after: commonAfter, order: true })
        return
      }
      if (x.every(isPrimitive) && y.every(isPrimitive)) {
        const xs = new Set(x)
        const ys = new Set(y)
        let any = false
        for (const v of y) {
          if (!xs.has(v)) {
            out.push({ path, kind: 'added', after: v })
            any = true
          }
        }
        for (const v of x) {
          if (!ys.has(v)) {
            out.push({ path, kind: 'removed', before: v })
            any = true
          }
        }
        if (!any) out.push({ path, kind: 'changed', before: x, after: y })
        return
      }
      const n = Math.max(x.length, y.length)
      for (let i = 0; i < n; i++) {
        const seg = `#${i + 1}`
        if (i >= x.length) out.push({ path: [...path, seg], kind: 'added', after: y[i] })
        else if (i >= y.length) out.push({ path: [...path, seg], kind: 'removed', before: x[i] })
        else walk(x[i], y[i], [...path, seg])
      }
      return
    }
    if (isPlainObject(x) && isPlainObject(y)) {
      for (const k of Object.keys(x)) {
        if (!has(y, k)) out.push({ path: [...path, k], kind: 'removed', before: x[k] })
        else walk(x[k], y[k], [...path, k])
      }
      for (const k of Object.keys(y)) if (!has(x, k)) out.push({ path: [...path, k], kind: 'added', after: y[k] })
      return
    }
    out.push({ path, kind: 'changed', before: x, after: y })
  }

  walk(a, b, [])
  return out
}

/** 改动所属的顶层条目（例如 gashas › gasha-01055-pickup-001），用于分组显示。 */
export function diffGroupKey(entry: DiffEntry): string {
  return entry.path.slice(0, 2).join(' › ')
}

/** 带名称的对象的显示名（name / title / eventName）；只有 ID 的对象返回空串（ID 不直接显示）。 */
export function objectLabel(v: unknown): string {
  if (!isPlainObject(v)) return ''
  for (const k of ['name', 'title', 'eventName']) {
    const s = v[k]
    if (typeof s === 'string' && s) return s
  }
  return ''
}
