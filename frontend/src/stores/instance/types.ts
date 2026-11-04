import { InstanceStatus, type InstanceInfo } from '@/types/v1/instance'

export { InstanceStatus } from '@/types/v1/instance'

export type InstanceRecord = InstanceInfo

export interface StatusMetaItem {
  labelKey: string
}

export interface QueryFormValue {
  keyword: string
  type: string
  status: InstanceStatus | ''
}

export interface InstanceTypeOption {
  label: string
  value: string
  /** 类型模板：新建应用时预填 */
  description?: string
  startCommand?: string
  stopCommand?: string
}

export type InstanceEditorMode = 'create' | 'edit'

export interface InstanceEditorFormValue {
  id: string
  name: string
  remark: string
  tags: string
  type: string
  startCommand: string
  stopCommand: string
  instancePath: string
  autoStart: boolean
  autoRestart: boolean
  /** 停止超时（秒），0 表示默认 */
  stopTimeout: number
  env: InstanceEnvRow[]
}

/** 编辑器里的一条环境变量 */
export interface InstanceEnvRow {
  key: string
  value: string
  /** 按密钥保存：值只写不读，控制台输出里打码 */
  secret: boolean
  /** 服务器上已保存过这个密钥（值不返回，留空表示保持不变） */
  saved: boolean
}

export const statusMeta: Record<InstanceStatus, StatusMetaItem> = {
  [InstanceStatus.INSTANCE_STATUS_RUNNING]: { labelKey: 'instance.running' },
  [InstanceStatus.INSTANCE_STATUS_STOPPED]: { labelKey: 'instance.stopped' },
  [InstanceStatus.INSTANCE_STATUS_MAINTENANCE]: { labelKey: 'instance.maintenance' },
  [InstanceStatus.INSTANCE_STATUS_UNSPECIFIED]: { labelKey: 'instance.unspecified' },
  [InstanceStatus.UNRECOGNIZED]: { labelKey: 'instance.unknownStatus' },
}

export type ConsoleSocketStatus = 'disconnected' | 'connecting' | 'connected' | 'error'

export interface ConsoleFeatureItem {
  key: string
  title?: string
  titleKey?: string
  description?: string
  descriptionKey?: string
  icon: string
}
