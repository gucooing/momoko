import request from '@/utils/request'
import type { GetMcpOverviewResponse } from '@/types/v1/mcp'

/** Momoko MCP 管理页：服务路径、工具和各自要的权限、当前用户能当凭证的 API Key */
export const getMcpOverview = () => {
  return request.get<GetMcpOverviewResponse>('/mcp/overview')
}
