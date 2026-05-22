import request from '@/utils/request'
import type {
  BatchCompressFileSystemRequest,
  BatchCompressFileSystemResponse,
  BatchDeleteFileSystemRequest,
  BatchDeleteFileSystemResponse,
  CancelFileUploadRequest,
  CancelFileUploadResponse,
  CompleteFileUploadRequest,
  CompleteFileUploadResponse,
  CopyFileSystemRequest,
  CopyFileSystemResponse,
  CreateFileSystemRequest,
  CreateFileSystemResponse,
  CutFileSystemRequest,
  CutFileSystemResponse,
  FileSystemPreSignRequest,
  FileSystemPreSignResponse,
  FileSystemPreSignUploadRequest,
  FileSystemPreSignUploadResponse,
  GetFileTaskResponse,
  GetFileUploadStatusRequest,
  GetFileUploadStatusResponse,
  GetFileSystemListRequest,
  GetFileSystemListResponse,
  OpenFileSystemFileRequest,
  OpenFileSystemFileResponse,
  RenameFileSystemRequest,
  RenameFileSystemResponse,
  UnzipFileSystemRequest,
  UnzipFileSystemResponse,
} from '@/types/v1/file'

const encodeBytesToBase64 = (value: Uint8Array) => {
  let binary = ''
  value.forEach((byte) => {
    binary += String.fromCharCode(byte)
  })
  return window.btoa(binary)
}

export const getFileSystemListRequest = (params: GetFileSystemListRequest) => {
  return request.get<GetFileSystemListResponse>('/file/system', { params })
}

export const createFileSystemRequest = (data: CreateFileSystemRequest = { info: undefined }) => {
  const payload = data.info
    ? {
        info: {
          ...data.info,
          content: encodeBytesToBase64(data.info.content),
        },
      }
    : data

  return request.post<CreateFileSystemResponse>('/file/system/create', payload)
}

export const deleteFileSystemEntriesRequest = (
  data: BatchDeleteFileSystemRequest = { paths: [] },
) => {
  return request.post<BatchDeleteFileSystemResponse>('/file/system/deletes', data)
}

export const getFileSystemPreSignRequest = (params: FileSystemPreSignRequest = { path: '' }) => {
  return request.get<FileSystemPreSignResponse>('/file/system/file/pre-sign', { params })
}

export const getFileSystemUploadPreSignRequest = (
  params: FileSystemPreSignUploadRequest = {
    path: '',
    fileName: '',
    fileSize: 0,
    hash: '',
  },
) => {
  return request.get<FileSystemPreSignUploadResponse>('/file/system/file/upload/pre-sign', {
    params,
  })
}

export const openFileSystemFileRequest = (data: OpenFileSystemFileRequest = { path: '' }) => {
  return request.post<OpenFileSystemFileResponse>('/file/system/open/file', data)
}

export const completeFileUploadRequest = (data: CompleteFileUploadRequest = { uploadId: '' }) => {
  return request.post<CompleteFileUploadResponse>('/file/upload/complete', data)
}

export const cancelFileUploadRequest = (data: CancelFileUploadRequest = { uploadId: '' }) => {
  return request.post<CancelFileUploadResponse>('/file/upload/cancel', data)
}

export const getFileUploadStatusRequest = (
  params: GetFileUploadStatusRequest = { uploadId: '' },
) => {
  return request.get<GetFileUploadStatusResponse>('/file/upload/status', { params })
}

export const compressFileSystemRequest = (data: BatchCompressFileSystemRequest) =>
  request.post<BatchCompressFileSystemResponse>('/file/system/compress', data)

export const renameFileSystemRequest = (data: RenameFileSystemRequest) =>
  request.post<RenameFileSystemResponse>('/file/system/rename', data)

export const unzipFileSystemRequest = (data: UnzipFileSystemRequest) =>
  request.post<UnzipFileSystemResponse>('/file/system/unzip', data)

export const copyFileSystemRequest = (data: CopyFileSystemRequest) =>
  request.post<CopyFileSystemResponse>('/file/system/copy', data)

export const cutFileSystemRequest = (data: CutFileSystemRequest) =>
  request.post<CutFileSystemResponse>('/file/system/cut', data)

export const getFileTaskRequest = (taskId: string) =>
  request.get<GetFileTaskResponse>(`/file/task/${taskId}`)
