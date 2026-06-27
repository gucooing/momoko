import request from '@/utils/request'
import type {
  ListFileSourcesResponse,
  CreateFileSourceRequest,
  CreateFileSourceResponse,
  UpdateFileSourceRequest,
  UpdateFileSourceResponse,
  DeleteFileSourceResponse,
  TestFileSourceRequest,
  TestFileSourceResponse,
} from '@/types/v1/file'

// 文件来源（OSS/FTP/WebDAV）管理 HTTP 封装。全局/管理员维护；密钥仅写不回显。

// 列表（enabledOnly=true 供文件浏览器下拉只取启用项）
export const listFileSourcesRequest = (params?: { keywords?: string; enabledOnly?: boolean }) =>
  request.get<ListFileSourcesResponse>('/file/source', { params })

// 新增
export const createFileSourceRequest = (data: CreateFileSourceRequest) =>
  request.post<CreateFileSourceResponse>('/file/source/create', data)

// 更新
export const updateFileSourceRequest = (data: UpdateFileSourceRequest) =>
  request.post<UpdateFileSourceResponse>('/file/source/update', data)

// 删除
export const deleteFileSourceRequest = (id: string) =>
  request.post<DeleteFileSourceResponse>('/file/source/delete', { id })

// 测试连通性（传 id 测已存在来源；或传 type+config 测未保存配置）
export const testFileSourceRequest = (data: TestFileSourceRequest) =>
  request.post<TestFileSourceResponse>('/file/source/test', data)
