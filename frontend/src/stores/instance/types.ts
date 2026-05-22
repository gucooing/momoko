import { InstanceStatus, type InstanceInfo } from '@/types/v1/instance'
import type { FileManagerAction } from '@/components/fileManager/index.vue'

export { InstanceStatus } from '@/types/v1/instance'

export type InstanceRecord = InstanceInfo

export interface StatusMetaItem {
  label: string
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
  [InstanceStatus.INSTANCE_STATUS_RUNNING]: { label: '运行中' },
  [InstanceStatus.INSTANCE_STATUS_STOPPED]: { label: '已停止' },
  [InstanceStatus.INSTANCE_STATUS_MAINTENANCE]: { label: '维护中' },
  [InstanceStatus.INSTANCE_STATUS_UNSPECIFIED]: { label: '未指定' },
  [InstanceStatus.UNRECOGNIZED]: { label: '未知状态' },
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
  refresh: '刷新目录',
  createFolder: '新建文件夹',
  createFile: '新建文件',
  upload: '上传文件',
  download: '下载选中文件',
  copyTemporaryLink: '复制链接',
  compress: '压缩',
  unzip: '解压',
  rename: '重命名',
  open: '打开',
  delete: '删除',
  more: '更多操作',
  copy: '复制',
  cut: '剪切',
  paste: '粘贴',
}

export type ConsoleSocketStatus = 'disconnected' | 'connecting' | 'connected' | 'error'

export interface ConsoleFeatureItem {
  key: string
  title: string
  description: string
  icon: string
}
