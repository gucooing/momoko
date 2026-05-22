import { getFileTaskRequest } from '@/api/file'
import { FileTaskStatus, type FileTaskInfo } from '@/types/v1/file'

export const waitFileTask = async (taskId: string): Promise<FileTaskInfo> => {
  for (;;) {
    const { data } = await getFileTaskRequest(taskId)
    const task = data.task
    if (!task) throw new Error('文件任务不存在')

    if (task.status === FileTaskStatus.FILE_TASK_STATUS_SUCCESS) return task
    if (task.status === FileTaskStatus.FILE_TASK_STATUS_FAILED) {
      throw new Error(task.message || '文件任务失败')
    }

    await new Promise((resolve) => window.setTimeout(resolve, 1000))
  }
}
