import { InstanceStatus, type InstanceInfo } from '@/types/v1/instance'

export { InstanceStatus } from '@/types/v1/instance'

export type InstanceRecord = InstanceInfo

export interface StatusMetaItem {
  labelKey: string
}

export interface OverviewCardItem {
  label: string
  value: number
  note: string
  icon: string
  skin: 'tone-a' | 'tone-b' | 'tone-c'
}

export interface QueryFormValue {
  keyword: string
  type: string
  status: InstanceStatus | ''
}

export interface InstanceTypeOption {
  label: string
  value: string
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
  envText: string
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
