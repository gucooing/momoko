import { getFileTaskRequest } from '@/api/file'
import { FileTaskStatus, type FileTaskInfo } from '@/types/v1/file'
import { translate as t } from '@/locales'

export const waitFileTask = async (taskId: string): Promise<FileTaskInfo> => {
  for (;;) {
    const { data } = await getFileTaskRequest(taskId)
    const task = data.task
    if (!task) throw new Error(t('utils.fileTask.notFound'))

    if (task.status === FileTaskStatus.FILE_TASK_STATUS_SUCCESS) return task
    if (task.status === FileTaskStatus.FILE_TASK_STATUS_FAILED) {
      throw new Error(task.message || t('utils.fileTask.failed'))
    }

    await new Promise((resolve) => window.setTimeout(resolve, 1000))
  }
}
