import { apiClient } from './api'
import type { ApiResponse, Memory, PaginationMeta } from '@/types/api'

export interface MemorySearchResult {
  id: string
  type: string
  content: string
  importance: number
  score: number
}

export const memoryService = {
  create: (type: string, content: string, importance = 0.5) =>
    apiClient.post<ApiResponse<Memory>>('/memories', { type, content, importance }),

  list: (page = 1, limit = 20) =>
    apiClient.get<ApiResponse<Memory[]> & { meta: PaginationMeta }>(
      `/memories?page=${page}&limit=${limit}`,
    ),

  delete: (id: string) =>
    apiClient.delete(`/memories/${id}`),

  search: (query: string, topK = 5, minScore = 0.3) =>
    apiClient.post<ApiResponse<MemorySearchResult[]>>('/memories/search', {
      query,
      top_k: topK,
      min_score: minScore,
    }),
}
