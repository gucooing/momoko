import request from '@/utils/request'
import type {
  ListTasksRequest,
  ListTasksResponse,
  GetTaskResponse,
  TaskBatchResponse,
  ClearTasksRequest,
  ClearTasksResponse,
} from '@/types/v1/task'

// 任务管理：通用任务管理器（pkg/task）的统一面板接口。鉴权由后端 /v1.TaskManager/ 前缀统一要求 task:manage。

export const listTasks = (params: Partial<ListTasksRequest>) =>
  request.get<ListTasksResponse>('/tasks', { params })

/** 单个任务（带执行步骤与完整日志）。 */
export const getTask = (id: string) => request.get<GetTaskResponse>(`/tasks/${id}`)

/** 取消若干运行中的任务（后端记录取消人）。 */
export const cancelTasks = (ids: string[]) => request.post<TaskBatchResponse>('/tasks/cancel', { ids })

/** 重试若干已结束的任务。 */
export const retryTasks = (ids: string[]) => request.post<TaskBatchResponse>('/tasks/retry', { ids })

/** 删除若干已结束的任务记录。 */
export const deleteTasks = (ids: string[]) => request.post<TaskBatchResponse>('/tasks/delete', { ids })

export const clearTasks = (data: Partial<ClearTasksRequest>) =>
  request.post<ClearTasksResponse>('/tasks/clear', data)
