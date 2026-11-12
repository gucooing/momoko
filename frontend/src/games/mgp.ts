/**
 * MGP v1（Momoko Game Protocol）通用数据类型：游戏服务端响应信封 `{ code, message, data, requestId }`
 * 中 `data` 的形状（信封本身由后端代理处理）。约定：时间均为 Unix 毫秒；pid 为账号 ID 的十进制字符串；
 * 接口只返回资源 ID，不提供图片。具体游戏的视图 / 报表 / 指令结果形状见 `games/<game>/types.ts`。
 */

/** 指令风险等级：high 起要求填写原因，danger 需二次确认且服务端会先自动备份目标存档。 */
export type MgpRisk = 'low' | 'medium' | 'high' | 'danger'

/** 指令目标：单个玩家、1–200 个玩家，或无目标的服务器指令。 */
export type MgpTarget = 'player' | 'players' | 'server'

/** 玩家视图的渲染方式。 */
export type MgpViewKind = 'keyValue' | 'cardGrid' | 'table'

/** 参数表单的选择器提示：奖励列表、道具、玩家存档备份。 */
export type MgpPicker = 'reward' | 'item' | 'backup'

/** 指令参数使用的 JSON Schema 子集（游戏服务端 object / field 生成）。 */
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

export interface ManifestGame {
  id: string
  name: string
  /** 主题色，例如 #F07A1E。 */
  theme: string
  /** 图标资源名（不是 URL）。 */
  icon: string
}

/** 客户端资源版本。 */
export interface ResourceVersion {
  app: string
  master: string
  octo: number
}

export interface ManifestServer {
  version: string
  resource: ResourceVersion
}

export interface ManifestView {
  id: string
  title: string
  kind: MgpViewKind
}

export interface ManifestCommand {
  id: string
  title: string
  /** 分组名，用于菜单归类。 */
  group: string
  risk: MgpRisk
  target: MgpTarget
  /** 是否允许开放给玩家自助执行。 */
  selfService: boolean
  /** 参数的 JSON Schema（总是 type: object）。 */
  params: JSONSchema
}

/** manifest 中只有 ID 与标题的条目（报表、玩法、目录）。 */
export interface ManifestEntry {
  id: string
  title: string
}

export interface ManifestConfig {
  id: string
  title: string
  /** 数据文件路径，例如 data/Gasha.json。 */
  file: string
  hotReload: boolean
}

export interface Manifest {
  /** 协议版本，目前为 1。 */
  protocol: number
  game: ManifestGame
  server: ManifestServer
  capabilities: string[]
  views: ManifestView[]
  commands: ManifestCommand[]
  reports: ManifestEntry[]
  /** 实际配置了的玩法；每个玩法有同名报表。 */
  modes: ManifestEntry[]
  catalogs: ManifestEntry[]
  configs: ManifestConfig[] | null
  /** 奖励资源类型名 → 协议 ResourceType 值。 */
  resourceTypes: Record<string, number>
}

export interface Health {
  name: string
  ok: boolean
  detail: string
  latencyMs: number
}

export interface MaintenanceStatus {
  enabled: boolean
  /** 维护期间可登录的玩家 pid；试运行时原样回显请求，可能为 null。 */
  whitelist: string[] | null
}

export interface Status {
  online: number
  onlinePeak: number
  onlinePeakAt: number
  registered: number
  newToday: number
  uptimeSeconds: number
  now: number
  version: string
  resource: ResourceVersion
  database: { migration: number }
  maintenance: MaintenanceStatus
  /** 存档队列占用。 */
  saves: { waiting: number; capacity: number }
  /** 指令数。 */
  commands: number
  health: Health[]
}

/** 列表的一页；cursor 为偏移量，nextCursor 为空串表示没有下一页。 */
export interface ListPage<T, S = undefined> {
  items: T[]
  nextCursor: string
  total: number
  /** 没有汇总的列表不带此字段。 */
  summary?: S
}

/** 列表请求参数。 */
export interface ListQuery {
  cursor?: string
  limit?: number
}

/** 选择器目录条目；零值字段省略。 */
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
}

export interface CatalogQuery extends ListQuery {
  /** 匹配 ID 或名称的一部分。 */
  q?: string
  characterId?: string
  rarity?: number
}

/** 指令请求体。 */
export interface CommandRequest<P = Record<string, unknown>> {
  /** 玩家 pid；服务器指令必须为空。 */
  targets?: string[]
  params?: P
  /** high / danger 指令非试运行时必填。 */
  reason?: string
  /** 非试运行时必填；重复提交返回首次结果并带 replayed。 */
  requestId?: string
  dryRun?: boolean
  operator?: string
}

/** 指令对一项数值的修改。 */
export interface Change {
  kind: string
  resourceId?: string
  before: number
  after: number
}

export interface TargetResult<D = unknown> {
  pid: string
  /** 失败的目标为空串。 */
  name: string
  status: 'success' | 'failed'
  code?: string
  message?: string
  changes?: Change[]
  detail?: D
  /** danger 指令执行前自动备份的 ID。 */
  backupId?: number
}

export interface CommandResult<D = unknown, S = unknown> {
  command: string
  requestId: string
  dryRun: boolean
  replayed: boolean
  /** 玩家指令每个目标一项；服务器指令为空数组。 */
  results: TargetResult<D>[]
  /** 服务器指令的结果。 */
  summary?: S
}

/** 配置域（列表中不带 content）。 */
export interface ConfigDomainInfo {
  id: string
  title: string
  file: string
  /** 文件内容的 SHA-256 十六进制。 */
  revision: string
  hotReload: boolean
}

export interface ConfigDocument<C = unknown> extends ConfigDomainInfo {
  content: C
}

/** PUT /configs/{domain} 请求体；revision 也可用 If-Match 头给出。 */
export interface ConfigPutRequest<C = unknown> {
  content: C
  revision?: string
  dryRun?: boolean
  operator?: string
  comment?: string
}

export interface ConfigPutResult {
  id: string
  file: string
  dryRun: boolean
  valid: boolean
  /** 校验失败的原因（只在试运行时返回；发布失败为 CONFIG_INVALID 错误）。 */
  error?: string
  /** 当前 revision；发布成功后为新 revision。 */
  revision: string
  /** 新内容（两空格缩进后）的 revision。 */
  newRevision?: string
  published: boolean
}

/** 事件流中的事件类型。 */
export type MgpEventKind =
  | 'session.login'
  | 'session.disconnect'
  | 'session.kick'
  | 'gasha.draw'
  | 'produce.finish'
  | 'gm.command'
  | 'config.publish'

/** SSE 事件 data；accountId 是数字账号 ID（不是 pid 字符串），空值字段省略。 */
export interface MgpEvent<D = Record<string, unknown>> {
  kind: MgpEventKind | string
  time: number
  accountId?: number
  publicUserId?: string
  name?: string
  data?: D
}

/** gm.command 事件的 data。 */
export interface GmCommandEventData {
  command: string
  requestId: string
  operator: string
  targets: string[] | null
  failed: number
}

/** config.publish 事件的 data。 */
export interface ConfigPublishEventData {
  domain: string
  file: string
  revision: string
  operator: string
}

/** MGP 错误码。 */
export type MgpErrorCode =
  | 'INVALID_ARGUMENT'
  | 'SIGNATURE_MISSING'
  | 'SIGNATURE_INVALID'
  | 'CLOCK_SKEW'
  | 'REPLAY'
  | 'PLAYER_NOT_FOUND'
  | 'VIEW_NOT_FOUND'
  | 'COMMAND_NOT_FOUND'
  | 'CATALOG_NOT_FOUND'
  | 'CONFIG_NOT_FOUND'
  | 'REPORT_NOT_FOUND'
  | 'ITEM_NOT_FOUND'
  | 'BACKUP_NOT_FOUND'
  | 'REVISION_MISMATCH'
  | 'CONFIG_INVALID'
  | 'REQUEST_ID_REUSED'
  | 'REQUEST_INCOMPLETE'
  | 'MAIL_ALREADY_SENT'
  | 'REWARD_INVALID'
  | 'CONDITION_INVALID'
  | 'CREDENTIALS_INVALID'
  | 'BODY_TOO_LARGE'
  | 'GAME_ERROR'
  | 'TIMEOUT'
  | 'SHUTTING_DOWN'
  | 'INTERNAL'
