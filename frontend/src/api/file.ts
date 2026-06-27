import request from '@/utils/request'
import type {
  GetFileSystemListRequest,
  GetFileSystemListResponse,
  GetFileSystemTreeResponse,
  CreateFileSystemResponse,
  RenameFileSystemResponse,
  BatchDeleteFileSystemResponse,
  CopyFileSystemResponse,
  CutFileSystemResponse,
  BatchCompressFileSystemResponse,
  UnzipFileSystemResponse,
  OpenFileSystemFileResponse,
  EditFileSystemFileResponse,
  FileSystemPreSignResponse,
  FileSystemPreSignUploadResponse,
  GetFileUploadStatusResponse,
  CompleteFileUploadResponse,
  CancelFileUploadResponse,
} from '@/types/v1/file'

// 系统级文件管理 HTTP 封装（ts_proto 仅出类型，请求函数手写）。
// bytes 字段（open.info / edit.content / create.info.content）以 base64 字符串传输，
// 由调用方（fileClient）负责编解码，这里只负责路由与透传。
// sourceId 为文件来源（空=本地磁盘），由 fileClient 透传。

// 列表
export const getFileSystemList = (params: GetFileSystemListRequest) =>
  request.get<GetFileSystemListResponse>('/file/system', { params })

// 目录树（懒加载）
export const getFileSystemTree = (path: string, sourceId = '') =>
  request.get<GetFileSystemTreeResponse>('/file/system/tree', { params: { path, sourceId } })

// 创建文件/目录（content 为 base64 文本，目录时省略）
export const createFileSystem = (
  info: { path: string; isDir: boolean; content?: string },
  sourceId = '',
) => request.post<CreateFileSystemResponse>('/file/system/create', { info, sourceId })

// 重命名
export const renameFileSystem = (path: string, newName: string, sourceId = '') =>
  request.post<RenameFileSystemResponse>('/file/system/rename', { path, newName, sourceId })

// 批量删除
export const batchDeleteFileSystem = (paths: string[], sourceId = '') =>
  request.post<BatchDeleteFileSystemResponse>('/file/system/deletes', { paths, sourceId })

// 复制（异步任务）
export const copyFileSystem = (paths: string[], targetPath: string, sourceId = '') =>
  request.post<CopyFileSystemResponse>('/file/system/copy', { paths, targetPath, sourceId })

// 剪切（异步任务）
export const cutFileSystem = (paths: string[], targetPath: string, sourceId = '') =>
  request.post<CutFileSystemResponse>('/file/system/cut', { paths, targetPath, sourceId })

// 压缩
export const batchCompressFileSystem = (paths: string[], targetPath?: string, sourceId = '') =>
  request.post<BatchCompressFileSystemResponse>('/file/system/compress', { paths, targetPath, sourceId })

// 解压
export const unzipFileSystem = (path: string, targetPath?: string, sourceId = '') =>
  request.post<UnzipFileSystemResponse>('/file/system/unzip', { path, targetPath, sourceId })

// 打开文件（响应 info 为 base64 字符串）
export const openFileSystemFile = (path: string, sourceId = '') =>
  request.post<OpenFileSystemFileResponse>('/file/system/open/file', { path, sourceId })

// 编辑文件（content 为 base64 字符串）
export const editFileSystemFile = (path: string, content: string, sourceId = '') =>
  request.post<EditFileSystemFileResponse>('/file/system/edit/file', { path, content, sourceId })

// 下载/预览预签名（inline=true 走内联预览）
export const fileSystemPreSign = (path: string, sourceId = '', inline = false) =>
  request.get<FileSystemPreSignResponse>('/file/system/file/pre-sign', {
    params: { path, sourceId, inline },
  })

// 上传预签名（分片）
export const fileSystemPreSignUpload = (params: {
  path: string
  fileName: string
  fileSize: number
  hash: string
  partSize?: number
  sourceId?: string
}) => request.get<FileSystemPreSignUploadResponse>('/file/system/file/upload/pre-sign', { params })

// 上传状态
export const getFileUploadStatus = (uploadId: string) =>
  request.get<GetFileUploadStatusResponse>('/file/upload/status', { params: { uploadId } })

// 完成上传（合并分片）
export const completeFileUpload = (uploadId: string) =>
  request.post<CompleteFileUploadResponse>('/file/upload/complete', { uploadId })

// 取消上传
export const cancelFileUpload = (uploadId: string) =>
  request.post<CancelFileUploadResponse>('/file/upload/cancel', { uploadId })
