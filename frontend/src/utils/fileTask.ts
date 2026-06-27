import request from '@/utils/request'
import { translate as t } from '@/locales'
import { FileTaskStatus, type FileTaskInfo, type GetFileTaskResponse } from '@/types/v1/file'

// 复制/剪切是后端异步任务（系统级与实例级共用同一 user 维度任务存储），
// 统一轮询系统级 `GET /file/task/{id}` 直到 SUCCESS/FAILED。

const POLL_INTERVAL_MS = 800

const isTerminal = (status: FileTaskStatus) =>
  status === FileTaskStatus.FILE_TASK_STATUS_SUCCESS ||
  status === FileTaskStatus.FILE_TASK_STATUS_FAILED

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

export interface WaitFileTaskOptions {
  // 进度回调（每次轮询到新状态时触发）
  onProgress?: (task: FileTaskInfo) => void
  // 已知的初始任务（来自 copy/cut 响应）：若已是终态则无需轮询
  initialTask?: FileTaskInfo
}

// 轮询文件任务直至完成；失败抛错（消息取任务 message 或兜底文案）。
export const waitFileTask = async (
  taskId: string,
  options: WaitFileTaskOptions = {},
): Promise<FileTaskInfo> => {
  if (!taskId) throw new Error(t('utils.fileTask.missingTask'))

  const { onProgress, initialTask } = options

  if (initialTask && isTerminal(initialTask.status)) {
    onProgress?.(initialTask)
    if (initialTask.status === FileTaskStatus.FILE_TASK_STATUS_FAILED) {
      throw new Error(initialTask.message || t('utils.fileTask.failed'))
    }
    return initialTask
  }

  for (;;) {
    const { data } = await request.get<GetFileTaskResponse>(`/file/task/${taskId}`)
    const task = data?.task
    if (!task) throw new Error(t('utils.fileTask.notFound'))

    onProgress?.(task)

    if (task.status === FileTaskStatus.FILE_TASK_STATUS_SUCCESS) return task
    if (task.status === FileTaskStatus.FILE_TASK_STATUS_FAILED) {
      throw new Error(task.message || t('utils.fileTask.failed'))
    }

    await sleep(POLL_INTERVAL_MS)
  }
}
