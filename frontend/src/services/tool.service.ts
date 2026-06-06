import { apiClient } from './api'
import type { ApiResponse } from '@/types/api'

export interface ToolInfo {
  id: string
  name: string
  display_name: string
  description: string
  category: string
  is_builtin: boolean
  schema: Record<string, unknown>
}

export const toolService = {
  list: () =>
    apiClient.get<ApiResponse<ToolInfo[]>>('/tools'),

  getConfig: (name: string) =>
    apiClient.get<ApiResponse<{ tool_id: string; is_enabled: boolean; config: Record<string, unknown> }>>(
      `/tools/${name}/config`,
    ),

  configure: (name: string, config: Record<string, unknown>) =>
    apiClient.post(`/tools/${name}/configure`, config),
}
