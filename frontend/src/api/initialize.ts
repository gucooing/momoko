import request from '@/utils/request'
import type {
  ConfirmInitializeRequest,
  ConfirmInitializeResponse,
  InitializeStatusRequest,
  InitializeStatusResponse,
  TestInitializeDatabaseRequest,
  TestInitializeDatabaseResponse,
} from '@/types/v1/initialize'

const BASE = '/system/initialize'

export const getInitializeStatus = (params: InitializeStatusRequest = {}) => {
  return request.get<InitializeStatusResponse>(`${BASE}/status`, { params })
}

export const testDatabaseConnection = (params: TestInitializeDatabaseRequest) => {
  return request.post<TestInitializeDatabaseResponse>(`${BASE}/database/test`, params)
}

export const confirmInitialize = (params: ConfirmInitializeRequest) => {
  return request.post<ConfirmInitializeResponse>(`${BASE}/confirm`, params)
}
