import { decodeFileContent, encodeTextToBase64 } from '@/utils/fileEncoding'
import { waitFileTask } from '@/utils/fileTask'
import { translate as t } from '@/locales'
import * as systemApi from '@/api/file'
import * as instanceApi from '@/api/instanceFile'
import type { FileTaskInfo, UploadInfo } from '@/types/v1/file'
import type { FileClient, FileScope } from './types'

// 复制/剪切：拿到任务后轮询至终态（系统级与实例级共用系统级任务轮询）。
const finishTask = async (task: FileTaskInfo | undefined): Promise<FileTaskInfo> => {
  if (!task) throw new Error(t('utils.fileTask.missingTask'))
  return waitFileTask(task.taskId, { initialTask: task })
}

const createSystemFileClient = (): FileClient => ({
  scope: { kind: 'system' },
  async list(params) {
    const { data } = await systemApi.getFileSystemList(params)
    return {
      directory: data.directory,
      items: data.items ?? [],
      page: data.page,
      pageSize: data.pageSize,
      total: data.total,
    }
  },
  async tree(path) {
    const { data } = await systemApi.getFileSystemTree(path)
    return data.nodes ?? []
  },
  async create(params) {
    await systemApi.createFileSystem({
      path: params.path,
      isDir: params.isDir,
      content: params.isDir ? undefined : encodeTextToBase64(params.content ?? ''),
    })
  },
  async rename(path, newName) {
    const { data } = await systemApi.renameFileSystem(path, newName)
    return data.path
  },
  async remove(paths) {
    const { data } = await systemApi.batchDeleteFileSystem(paths)
    return data.items ?? []
  },
  async copy(paths, targetPath) {
    const { data } = await systemApi.copyFileSystem(paths, targetPath)
    return finishTask(data.task)
  },
  async move(paths, targetPath) {
    const { data } = await systemApi.cutFileSystem(paths, targetPath)
    return finishTask(data.task)
  },
  async compress(paths, targetPath) {
    const { data } = await systemApi.batchCompressFileSystem(paths, targetPath)
    return data.outputPath
  },
  async unzip(path, targetPath) {
    const { data } = await systemApi.unzipFileSystem(path, targetPath)
    return data.outputPath
  },
  async open(path) {
    const { data } = await systemApi.openFileSystemFile(path)
    return decodeFileContent(data.info)
  },
  async edit(path, content) {
    await systemApi.editFileSystemFile(path, encodeTextToBase64(content))
  },
  async preSignDownload(path) {
    const { data } = await systemApi.fileSystemPreSign(path)
    return data.downloadUrlPath
  },
  async preSignUpload(params) {
    const { data } = await systemApi.fileSystemPreSignUpload(params)
    return data.info as UploadInfo
  },
})

const createInstanceFileClient = (id: string): FileClient => ({
  scope: { kind: 'instance', id },
  async list(params) {
    const { data } = await instanceApi.getInstanceFileList(id, params)
    return {
      directory: data.directory,
      items: data.items ?? [],
      page: data.page,
      pageSize: data.pageSize,
      total: data.total,
    }
  },
  async tree(path) {
    const { data } = await instanceApi.getInstanceFileTree(id, path)
    return data.nodes ?? []
  },
  async create(params) {
    await instanceApi.createInstanceFile(id, {
      path: params.path,
      isDir: params.isDir,
      content: params.isDir ? undefined : encodeTextToBase64(params.content ?? ''),
    })
  },
  async rename(path, newName) {
    const { data } = await instanceApi.renameInstanceFile(id, path, newName)
    return data.path
  },
  async remove(paths) {
    const { data } = await instanceApi.batchDeleteInstanceFile(id, paths)
    return data.items ?? []
  },
  async copy(paths, targetPath) {
    const { data } = await instanceApi.copyInstanceFile(id, paths, targetPath)
    return finishTask(data.task)
  },
  async move(paths, targetPath) {
    const { data } = await instanceApi.cutInstanceFile(id, paths, targetPath)
    return finishTask(data.task)
  },
  async compress(paths, targetPath) {
    const { data } = await instanceApi.batchCompressInstanceFile(id, paths, targetPath)
    return data.outputPath
  },
  async unzip(path, targetPath) {
    const { data } = await instanceApi.unzipInstanceFile(id, path, targetPath)
    return data.outputPath
  },
  async open(path) {
    const { data } = await instanceApi.openInstanceFile(id, path)
    return decodeFileContent(data.info)
  },
  async edit(path, content) {
    await instanceApi.editInstanceFile(id, path, encodeTextToBase64(content))
  },
  async preSignDownload(path) {
    const { data } = await instanceApi.instanceFilePreSign(id, path)
    return data.downloadUrlPath
  },
  async preSignUpload(params) {
    const { data } = await instanceApi.instanceFilePreSignUpload(id, params)
    return data.info as UploadInfo
  },
})

// 工厂：按 scope 返回对应的文件操作客户端，组件层与 scope 无关。
export const createFileClient = (scope: FileScope): FileClient =>
  scope.kind === 'system' ? createSystemFileClient() : createInstanceFileClient(scope.id)
