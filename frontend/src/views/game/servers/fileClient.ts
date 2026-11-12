// 托管游戏服务器的文件客户端：服务端目录的文件接口与应用实例同构（路径为 /game/servers/{id}/files/*），
// 交给通用的 FileBrowser / useFileUpload 使用。
import * as api from '@/api/gameServer'
import type { FileClient } from '@/components/file/types'
import type { FileTaskInfo, UploadInfo } from '@/types/v1/file'
import { translate as t } from '@/locales'
import { decodeFileContent, encodeTextToBase64 } from '@/utils/fileEncoding'
import { isFileTooLargeForEditor, resolvePreSignedFileUrl } from '@/utils/file'
import { waitFileTask } from '@/utils/fileTask'

const finishTask = async (task: FileTaskInfo | undefined): Promise<FileTaskInfo> => {
  if (!task) throw new Error(t('utils.fileTask.missingTask'))
  return waitFileTask(task.taskId, { initialTask: task })
}

const openViaDownload = async (downloadUrlPath: string, path: string) => {
  const res = await fetch(resolvePreSignedFileUrl(downloadUrlPath))
  if (!res.ok) throw new Error(t('fileManager.openFailed'))
  if (isFileTooLargeForEditor(path, res.headers.get('content-length'))) throw new Error(t('fileManager.fileTooLargeForEditor'))
  const bytes = new Uint8Array(await res.arrayBuffer())
  if (isFileTooLargeForEditor(path, bytes.byteLength)) throw new Error(t('fileManager.fileTooLargeForEditor'))
  return decodeFileContent(bytes)
}

export const createGameServerFileClient = (id: string): FileClient => ({
  scope: { kind: 'instance', id },
  async list(params) {
    const { data } = await api.gameServerFileList(id, params)
    return {
      directory: data.directory,
      items: data.items ?? [],
      page: data.page,
      pageSize: data.pageSize,
      total: data.total,
    }
  },
  async tree(path) {
    const { data } = await api.gameServerFileTree(id, path)
    return data.nodes ?? []
  },
  async create(params) {
    await api.createGameServerFile(id, {
      path: params.path,
      isDir: params.isDir,
      content: params.isDir ? undefined : encodeTextToBase64(params.content ?? ''),
    })
  },
  async rename(path, newName) {
    const { data } = await api.renameGameServerFile(id, path, newName)
    return data.path
  },
  async remove(paths) {
    const { data } = await api.deleteGameServerFiles(id, paths)
    return data.items ?? []
  },
  async copy(paths, targetPath) {
    const { data } = await api.copyGameServerFiles(id, paths, targetPath)
    return finishTask(data.task)
  },
  async move(paths, targetPath) {
    const { data } = await api.cutGameServerFiles(id, paths, targetPath)
    return finishTask(data.task)
  },
  async compress(paths, targetPath) {
    const { data } = await api.compressGameServerFiles(id, paths, targetPath)
    return data.outputPath
  },
  async unzip(path, targetPath) {
    const { data } = await api.unzipGameServerFile(id, path, targetPath)
    return data.outputPath
  },
  async open(path) {
    const { data } = await api.gameServerFilePreSign(id, path)
    return openViaDownload(data.downloadUrlPath, path)
  },
  async edit(path, content) {
    await api.editGameServerFile(id, path, encodeTextToBase64(content))
  },
  async preSignDownload(path) {
    const { data } = await api.gameServerFilePreSign(id, path)
    return data.downloadUrlPath
  },
  async preSignUpload(params) {
    const { path, fileName, fileSize, hash, partSize } = params
    const { data } = await api.gameServerFilePreSignUpload(id, { path, fileName, fileSize, hash, partSize })
    return data.info as UploadInfo
  },
})
