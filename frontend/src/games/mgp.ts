/**
 * GM 页面通用的前端类型：分页、目录条目（名称与素材解析用）、操作的表单与结果、MGP 事件流。
 * 各游戏的数据类型由 api/proto 生成（src/types/v1/<游戏>），游戏特有的前端定义见 games/<游戏>。
 */

/** 操作风险等级：danger 需输入玩家昵称确认。 */
export type MgpRisk = 'low' | 'medium' | 'high' | 'danger'

/** 操作目标：单个玩家、1–200 个玩家，或无目标的服务器操作。 */
export type MgpTarget = 'player' | 'players' | 'server'

/** 参数表单的选择器提示：奖励列表、道具、玩家存档备份。 */
export type MgpPicker = 'reward' | 'item' | 'backup'

/** 操作参数表单使用的 JSON Schema 子集。 */
export interface JSONSchema {
  type?: 'object' | 'array' | 'string' | 'integer' | 'number' | 'boolean'
  title?: string
  description?: string
  properties?: Record<string, JSONSchema>
  required?: string[]
  additionalProperties?: boolean
  items?: JSONSchema
  enum?: Array<string | number>
  minimum?: number
  maximum?: number
  minItems?: number
  maxItems?: number
  maxLength?: number
  /** 字符串格式，例如 password。 */
  format?: string
  default?: unknown
  /** 选择器提示：表单用对应的选择器代替原始输入。 */
  'x-picker'?: MgpPicker
}

/** 列表的一页：nextCursor 为空串表示没有下一页。 */
export interface ListPage<T, S = undefined> {
  items: T[]
  nextCursor: string
  total: number
  summary?: S
}

/** 目录条目（名称与素材解析用）；零值字段省略。 */
export interface CatalogEntry {
  id: string
  /** 名称：前端换成当前语言的译名（有的话），原文在 originalName。 */
  name: string
  /** 服务端给的原文名（name 被换成译名时才有）。 */
  originalName?: string
  /** 可作为礼物发放时的 ResourceType，否则为 0。 */
  resourceType: number
  rarity?: number
  type?: number
  characterId?: string
  parentId?: string
  assetId?: string
  startTime?: number
  endTime?: number
  /** 效果文字（道具等） */
  description?: string
  /** 强化后的名称（技能卡） */
  upgradedName?: string
  /** 图片按角色区分（技能卡：图片为 assetId 加「-」和角色 ID） */
  characterAsset?: boolean
  /** 各等级的效果文字（记忆能力） */
  levels?: { level: number; description: string }[]
}

/** 目录查询。 */
export interface CatalogQuery {
  /** 匹配 ID 或名称的一部分。 */
  q?: string
  /** 只保留这些 ID。 */
  ids?: string[]
  characterId?: string
  rarity?: number
  cursor?: string
  limit?: number
}

/** 操作对一项数值的修改。 */
export interface Change {
  kind: string
  resourceId?: string
  before: number
  after: number
}

/** 操作对一个对象的结果。 */
export interface OperationRow {
  pid: string
  name: string
  /** success / failed */
  status: string
  code?: string
  message?: string
  changes?: Change[]
  /** 操作返回的其他内容（例如导出的存档 export）。 */
  detail?: unknown
}

/** 一次操作的结果：GM 记录 id 与状态、逐个对象的结果，服务器操作的结果在 summary。 */
export interface OperationOutcome {
  executionId: string
  status: string
  errorCode?: string
  errorMessage?: string
  dryRun: boolean
  /** 操作前自动生成的存档备份 */
  backupRef?: string
  results: OperationRow[]
  summary?: unknown
}

/** 一个操作的定义（GM 页面的菜单、表单与执行）。 */
export interface OperationDef {
  id: string
  title: string
  group: string
  risk: MgpRisk
  target: MgpTarget
  /** 是否允许开放给玩家自助。 */
  selfService: boolean
  /** 是否支持试运行（只校验并预览变化，不写入）。 */
  dryRun: boolean
  /** 参数表单（总是 type: object）。 */
  params: JSONSchema
  run: (serverId: string, pids: string[], params: Record<string, unknown>, opt: OperationOptions) => Promise<OperationOutcome>
}

/** 执行选项：试运行、写进 GM 记录的摘要。 */
export interface OperationOptions {
  dryRun: boolean
  summary: Record<string, string>
}

/** SSE 事件 data；accountId 是数字账号 ID（不是 pid 字符串），空值字段省略。 */
export interface MgpEvent<D = Record<string, unknown>> {
  kind: string
  time: number
  accountId?: number
  publicUserId?: string
  name?: string
  data?: D
}
