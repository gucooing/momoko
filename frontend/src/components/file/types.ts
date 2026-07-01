// 文件管理「scope 化」契约：同一套 UI 同时驱动系统级与实例级两套后端 API。
// 系统级与实例级 RPC 近乎镜像（实例多一个 {id}），底层共用 FileOper 引擎、
// 任务存储与消息类型，故在前端用一个 FileClient 抽象屏蔽差异，组件与 scope 无关。
import type { InjectionKey, Ref } from 'vue'
import type {
  FileDirectoryInfo,
  FileEntryInfo,
  FileOperationResult,
  FileSortField,
  FileTaskInfo,
  FileTreeNode,
  UploadInfo,
} from '@/types/v1/file'

// 作用域：系统级 或 某实例。
export type FileScope = { kind: 'system' } | { kind: 'instance'; id: string }

export const isInstanceScope = (scope: FileScope): scope is { kind: 'instance'; id: string } =>
  scope.kind === 'instance'

// 列表查询参数（scope 无关，id 由 client 内部补齐）。
export interface FileListParams {
  path: string
  page: number
  pageSize: number
  keywords?: string
  includeSubDir?: boolean
  sortField: FileSortField
  isDesc: boolean
}

export interface FileListResult {
  directory?: FileDirectoryInfo
  items: FileEntryInfo[]
  page: number
  pageSize: number
  total: number
}

// 创建文件/目录的友好参数（content 为 UTF-8 文本，client 内部编码为 base64）。
export interface FileCreateParams {
  path: string
  isDir: boolean
  content?: string
}

// 上传预签名的友好参数（id 由 client 补齐）。
export interface FileUploadSignParams {
  path: string
  fileName: string
  fileSize: number
  hash: string
  partSize?: number
}

// 打开文件的解码结果。
export interface FileOpenResult {
  text: string
  isBinary: boolean
}

// scope 无关的文件操作客户端：base64 编解码、id 注入、任务轮询都封装在内部。
export interface FileClient {
  readonly scope: FileScope
  list(params: FileListParams): Promise<FileListResult>
  tree(path: string): Promise<FileTreeNode[]>
  create(params: FileCreateParams): Promise<void>
  rename(path: string, newName: string): Promise<string>
  remove(paths: string[]): Promise<FileOperationResult[]>
  // 复制/剪切返回最终完成的任务（内部已轮询至终态）
  copy(paths: string[], targetPath: string): Promise<FileTaskInfo>
  move(paths: string[], targetPath: string): Promise<FileTaskInfo>
  compress(paths: string[], targetPath?: string): Promise<string>
  unzip(path: string, targetPath?: string): Promise<string>
  // 打开文本文件，返回解码文本（二进制时 isBinary=true）
  open(path: string): Promise<FileOpenResult>
  edit(path: string, content: string): Promise<void>
  // 下载/预览预签名路径（配 resolvePreSignedFileUrl 使用）；inline=true 走内联预览
  preSignDownload(path: string, inline?: boolean): Promise<string>
  preSignUpload(params: FileUploadSignParams): Promise<UploadInfo>
}

// 文件选择器选中的文件/文件夹：自带来源 id（空=本地磁盘），支持在一次选择里跨来源累积。
export interface PickedFile {
  sourceId: string
  path: string
}

// 剪贴板（复制/剪切缓冲）。
export type ClipboardMode = 'copy' | 'cut'

export interface FileClipboard {
  mode: ClipboardMode
  paths: string[]
}

// 工具栏「更多操作」动作标识（与 i18n fileManager.* 对应）。
export type FileManagerAction =
  | 'refresh'
  | 'createFolder'
  | 'createFile'
  | 'upload'
  | 'download'
  | 'copyTemporaryLink'
  | 'share'
  | 'compress'
  | 'unzip'
  | 'rename'
  | 'open'
  | 'delete'
  | 'copy'
  | 'cut'
  | 'paste'

// 编辑器文件树的 provide/inject 上下文（懒加载树通过它直连客户端与选中态，免逐层 emit）。
export interface FileTreeContext {
  client: FileClient
  activePath: Ref<string>
  selectedPaths?: Ref<Set<string>>
  multiple: boolean
  // 'file'=仅文件可选（编辑器）；'all'=文件与文件夹均可选（分享/选择器）
  selectable: 'file' | 'all'
  select: (path: string, name: string, isDir: boolean) => void
}

export const FILE_TREE_CONTEXT: InjectionKey<FileTreeContext> = Symbol('file-tree-context')

// 列表行的展示用视图模型（在 FileEntryInfo 之上补充派生展示字段）。
export interface FileRow extends FileEntryInfo {
  // 小写扩展名（无则空）
  extension: string
  // 类型展示文案（“文件夹” / “{EXT} 图片|视频|音频|文件”），由组件用 i18n 生成
  // 这里只放派生原料，文案在组件内本地化
  previewKind: 'image' | 'video' | 'audio' | null
}
