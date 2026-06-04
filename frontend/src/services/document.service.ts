import { apiClient } from './api'
import type { ApiResponse, Document, PaginationMeta } from '@/types/api'

export interface RAGSearchResult {
  document_id: string
  doc_name: string
  content: string
  score: number
}

export interface RAGSearchResponse {
  chunks: RAGSearchResult[]
  total: number
}

export const documentService = {
  // Upload — multipart form, field name must be "file"
  upload: (file: File) => {
    const form = new FormData()
    form.append('file', file)
    return apiClient.post<ApiResponse<Document>>('/documents', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },

  list: (page = 1, limit = 20) =>
    apiClient.get<ApiResponse<Document[]> & { meta: PaginationMeta }>(
      `/documents?page=${page}&limit=${limit}`,
    ),

  getById: (id: string) =>
    apiClient.get<ApiResponse<Document>>(`/documents/${id}`),

  delete: (id: string) =>
    apiClient.delete(`/documents/${id}`),

  getDownloadURL: (id: string) =>
    apiClient.get<ApiResponse<{ url: string }>>(`/documents/${id}/url`),

  search: (query: string, topK = 5, minScore = 0.3) =>
    apiClient.post<ApiResponse<RAGSearchResponse>>('/rag/search', {
      query,
      top_k: topK,
      min_score: minScore,
    }),
}
