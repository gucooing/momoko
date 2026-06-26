import { InstanceStatus, type InstanceInfo } from '@/types/v1/instance'
import type { FileManagerAction } from '@/components/fileManager/index.vue'

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

export type FileManagerShell = 'cmd' | 'powershell' | 'bash' | 'sh'

export type FileManagerSource = 'instance' | 'terminal'

export interface FileManagerContext {
  instanceId: string
  name: string
  source: FileManagerSource
  sourceLabel: string
  status: InstanceStatus
  runtime: string
  workdir: string
  shell: FileManagerShell
}

export type FileManagerEntryKind = 'folder' | 'file'

export interface FileManagerEntry {
  id: string
  name: string
  path: string
  kind: FileManagerEntryKind
  permission: string
  ownerGroup: string
  size: string
  bytes: number
  updatedAt: string
  extension?: string
  previewText?: string
}

export type FileManagerEntryMap = Record<string, FileManagerEntry[]>
export type { FileManagerAction } from '@/components/fileManager/index.vue'

export const fileManagerActionTextMap: Record<FileManagerAction, string> = {
  refresh: 'fileManager.refresh',
  createFolder: 'fileManager.createFolder',
  createFile: 'fileManager.createFile',
  upload: 'fileManager.upload',
  download: 'fileManager.download',
  copyTemporaryLink: 'fileManager.copyLink',
  share: 'fileManager.share',
  compress: 'fileManager.compress',
  unzip: 'fileManager.unzip',
  rename: 'fileManager.rename',
  open: 'fileManager.open',
  delete: 'fileManager.delete',
  more: 'fileManager.more',
  copy: 'fileManager.copy',
  cut: 'fileManager.cut',
  paste: 'fileManager.paste',
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
