import { apiClient } from './api'
import type { ApiResponse, PaginationMeta, Workflow, WorkflowRun, WorkflowDefinition, WorkflowNode, WorkflowEdge } from '@/types/api'

// Re-export canonical types so existing imports from this file still work.
export type { Workflow, WorkflowRun, WorkflowDefinition, WorkflowNode, WorkflowEdge }

export interface CreateWorkflowPayload {
  name: string
  description?: string
  definition: WorkflowDefinition
  trigger_type?: string
  cron_expr?: string
}

export const workflowService = {
  create: (payload: CreateWorkflowPayload) =>
    apiClient.post<ApiResponse<Workflow>>('/workflows', payload),

  list: (page = 1, limit = 20) =>
    apiClient.get<ApiResponse<Workflow[]> & { meta: PaginationMeta }>(
      `/workflows?page=${page}&limit=${limit}`,
    ),

  getById: (id: string) =>
    apiClient.get<ApiResponse<Workflow>>(`/workflows/${id}`),

  update: (id: string, payload: Partial<CreateWorkflowPayload> & { status?: string }) =>
    apiClient.patch<ApiResponse<Workflow>>(`/workflows/${id}`, payload),

  delete: (id: string) =>
    apiClient.delete(`/workflows/${id}`),

  triggerRun: (id: string, payload?: Record<string, unknown>) =>
    apiClient.post<ApiResponse<WorkflowRun>>(`/workflows/${id}/run`, { payload }),

  listRuns: (id: string, page = 1, limit = 20) =>
    apiClient.get<ApiResponse<WorkflowRun[]> & { meta: PaginationMeta }>(
      `/workflows/${id}/runs?page=${page}&limit=${limit}`,
    ),

  getRun: (workflowId: string, runId: string) =>
    apiClient.get<ApiResponse<WorkflowRun>>(`/workflows/${workflowId}/runs/${runId}`),
}

// Empty workflow template for the builder
export const emptyDefinition = (): WorkflowDefinition => ({
  nodes: [],
  edges: [{ from: 'START', to: 'END' }],
  trigger: { type: 'MANUAL' },
})

export const NODE_TYPES = [
  { value: 'agent',     label: 'AI Agent',   desc: 'Call an LLM with a prompt' },
  { value: 'tool',      label: 'Tool',        desc: 'Run a registered tool (web_search, calculator, http_request)' },
  { value: 'condition', label: 'Condition',   desc: 'Branch on a value' },
  { value: 'delay',     label: 'Delay',       desc: 'Wait N seconds' },
] as const

export const TRIGGER_TYPES = [
  { value: 'MANUAL',   label: 'Manual trigger' },
  { value: 'SCHEDULE', label: 'Schedule (cron)' },
  { value: 'EVENT',    label: 'Event-based' },
] as const
