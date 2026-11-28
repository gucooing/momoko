// Agent 页面上传文件（消息附件、技能压缩包）：Momoko 标准的分片上传（和文件管理同一套队列、分片、校验、收尾），
// 先传进自己的暂存目录，用的时候只带 uploadId，内容不经 JSON。
import { watch } from 'vue'
import { preSignAgentUpload } from '@/api/agent'
import { useFileUpload, type FileUploadQueue } from '@/components/file/useFileUpload'
import { translate as t } from '@/locales'

const agentUploadClient = {
  preSignUpload: (p: { fileName: string; fileSize: number; hash: string }) =>
    preSignAgentUpload({ fileName: p.fileName, fileSize: p.fileSize, hash: p.hash }).then(({ data }) => {
      if (!data.info) throw new Error(t('agent.code.composer.uploadFailed'))
      return data.info
    }),
}

/** 一个上传队列：加进去就开始传 */
export const useAgentUploads = () => useFileUpload(() => agentUploadClient, () => '')

/** 传一个文件，等它传完：给 uploadId；失败、取消时抛错。队列里的这一项传完就去掉 */
export const uploadAgentFile = (queue: FileUploadQueue, file: File) =>
  new Promise<string>((resolve, reject) => {
    queue.addFiles([file])
    const item = queue.items.value.find((i) => i.file === file)
    if (!item) {
      reject(new Error(t('agent.code.composer.uploadFailed')))
      return
    }
    const stop = watch(
      () => item.status,
      (status) => {
        if (status !== 'done' && status !== 'failed' && status !== 'canceled') return
        stop()
        queue.removeItem(item.id)
        if (status === 'done' && item.uploadId) resolve(item.uploadId)
        else reject(new Error(item.error || t('agent.code.composer.uploadFailed')))
      },
    )
  })
