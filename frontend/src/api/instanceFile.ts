import request from '@/utils/request'
import type {
  GetInstanceFileListRequest,
  GetInstanceFileListResponse,
  GetInstanceFileTreeResponse,
  CreateInstanceFileResponse,
  RenameInstanceFileResponse,
  BatchDeleteInstanceFileResponse,
  CopyInstanceFileResponse,
  CutInstanceFileResponse,
  BatchCompressInstanceFileResponse,
  UnzipInstanceFileResponse,
  OpenInstanceFileResponse,
  EditInstanceFileResponse,
  InstanceFilePreSignResponse,
  InstanceFilePreSignUploadResponse,
} from '@/types/v1/instance'

// 实例级文件管理 HTTP 封装：与系统级镜像，路径末尾多一个 {id}。
// 底层共用 file.proto 的消息类型与任务存储（复制/剪切任务可经系统级 /file/task 轮询）。

// 列表（id 在路径，其余作为 query）
export const getInstanceFileList = (id: string, params: Omit<GetInstanceFileListRequest, 'id'>) =>
  request.get<GetInstanceFileListResponse>(`/instance/file/${id}`, { params })

// 目录树（懒加载）
export const getInstanceFileTree = (id: string, path: string) =>
  request.get<GetInstanceFileTreeResponse>(`/instance/file/tree/${id}`, { params: { path } })

// 创建文件/目录
export const createInstanceFile = (
  id: string,
  info: { path: string; isDir: boolean; content?: string },
) => request.post<CreateInstanceFileResponse>(`/instance/file/create/${id}`, { id, info })

// 重命名
export const renameInstanceFile = (id: string, path: string, newName: string) =>
  request.post<RenameInstanceFileResponse>(`/instance/file/rename/${id}`, { id, path, newName })

// 批量删除
export const batchDeleteInstanceFile = (id: string, paths: string[]) =>
  request.post<BatchDeleteInstanceFileResponse>(`/instance/file/deletes/${id}`, { id, paths })

// 复制（异步任务）
export const copyInstanceFile = (id: string, paths: string[], targetPath: string) =>
  request.post<CopyInstanceFileResponse>(`/instance/file/copy/${id}`, { id, paths, targetPath })

// 剪切（异步任务）
export const cutInstanceFile = (id: string, paths: string[], targetPath: string) =>
  request.post<CutInstanceFileResponse>(`/instance/file/cut/${id}`, { id, paths, targetPath })

// 压缩
export const batchCompressInstanceFile = (id: string, paths: string[], targetPath?: string) =>
  request.post<BatchCompressInstanceFileResponse>(`/instance/file/compress/${id}`, {
    id,
    paths,
    targetPath,
  })

// 解压
export const unzipInstanceFile = (id: string, path: string, targetPath?: string) =>
  request.post<UnzipInstanceFileResponse>(`/instance/file/unzip/${id}`, { id, path, targetPath })

// 打开文件（响应 info 为 base64 字符串）
export const openInstanceFile = (id: string, path: string) =>
  request.post<OpenInstanceFileResponse>(`/instance/file/open/${id}`, { id, path })

// 编辑文件（content 为 base64 字符串）
export const editInstanceFile = (id: string, path: string, content: string) =>
  request.post<EditInstanceFileResponse>(`/instance/file/edit/${id}`, { id, path, content })

// 下载预签名
export const instanceFilePreSign = (id: string, path: string) =>
  request.get<InstanceFilePreSignResponse>(`/instance/file/pre-sign/${id}`, { params: { id, path } })

// 上传预签名（分片）
export const instanceFilePreSignUpload = (
  id: string,
  params: { path: string; fileName: string; fileSize: number; hash: string; partSize?: number },
) =>
  request.get<InstanceFilePreSignUploadResponse>(`/instance/file/upload/pre-sign/${id}`, {
    params: { id, ...params },
  })
