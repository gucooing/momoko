import request from '@/utils/request'
import type {
  ListTasksRequest,
  ListTasksResponse,
  GetTaskResponse,
  CancelTaskResponse,
  RetryTaskResponse,
  DeleteTaskResponse,
} from '@/types/v1/task'

// 任务管理：通用任务管理器（pkg/task）的统一面板接口。鉴权由后端 /v1.TaskManager/ 前缀统一要求 task:manage。

export const listTasks = (params: Partial<ListTasksRequest>) =>
  request.get<ListTasksResponse>('/tasks', { params })

export const getTask = (id: string) => request.get<GetTaskResponse>(`/task/${id}`)

export const cancelTask = (id: string) =>
  request.post<CancelTaskResponse>('/task/cancel', { id })

export const retryTask = (id: string) => request.post<RetryTaskResponse>('/task/retry', { id })

export const deleteTask = (id: string) =>
  request.post<DeleteTaskResponse>('/task/delete', { id })
