import { apiClient } from './api'
import type { ApiResponse, Agent, PaginationMeta } from '@/types/api'

export interface CreateAgentPayload {
  name: string
  description?: string
  type: string
  system_prompt?: string
  model?: string
  temperature?: number
  max_tokens?: number
  tools_enabled?: string[]
  memory_enabled?: boolean
  rag_enabled?: boolean
}

export interface UpdateAgentPayload extends Partial<CreateAgentPayload> {}

export const agentService = {
  create: (payload: CreateAgentPayload) =>
    apiClient.post<ApiResponse<Agent>>('/agents', payload),

  list: (page = 1, limit = 20) =>
    apiClient.get<ApiResponse<Agent[]> & { meta: PaginationMeta }>(
      `/agents?page=${page}&limit=${limit}`,
    ),

  getById: (id: string) =>
    apiClient.get<ApiResponse<Agent>>(`/agents/${id}`),

  update: (id: string, payload: UpdateAgentPayload) =>
    apiClient.patch<ApiResponse<Agent>>(`/agents/${id}`, payload),

  delete: (id: string) =>
    apiClient.delete(`/agents/${id}`),
}

export const AGENT_TYPES = [
  { value: 'CUSTOM',     label: 'Custom' },
  { value: 'RESEARCH',   label: 'Research' },
  { value: 'CODING',     label: 'Coding' },
  { value: 'PLANNING',   label: 'Planning' },
  { value: 'SUPERVISOR', label: 'Supervisor' },
  { value: 'WORKFLOW',   label: 'Workflow' },
] as const

export const AVAILABLE_TOOLS = [
  { value: 'web_search', label: 'Web Search', desc: 'Search the web via DuckDuckGo' },
  { value: 'calculator', label: 'Calculator',  desc: 'Arithmetic: add, subtract, multiply, divide' },
] as const

export const AVAILABLE_MODELS = [
  'gpt-4o',
  'gpt-4o-mini',
  'gpt-4-turbo',
  'gpt-3.5-turbo',
] as const
