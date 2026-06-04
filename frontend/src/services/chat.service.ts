import { apiClient } from './api'
import { useAuthStore } from '@/store/authStore'
import type { ApiResponse, Conversation, Message, PaginationMeta } from '@/types/api'

const BASE = import.meta.env.VITE_API_URL ?? '/api/v1'

export const chatService = {
  createConversation: (data: { title?: string; agent_id?: string }) =>
    apiClient.post<ApiResponse<Conversation>>('/conversations', data),

  listConversations: (page = 1, limit = 20) =>
    apiClient.get<ApiResponse<Conversation[]> & { meta: PaginationMeta }>(
      `/conversations?page=${page}&limit=${limit}`,
    ),

  getConversation: (id: string) =>
    apiClient.get<ApiResponse<Conversation & { messages: Message[] }>>(`/conversations/${id}`),

  deleteConversation: (id: string) =>
    apiClient.delete(`/conversations/${id}`),

  sendMessage: (convId: string, content: string) =>
    apiClient.post<ApiResponse<Message>>(`/conversations/${convId}/messages`, { content }),

  listMessages: (convId: string, page = 1, limit = 50) =>
    apiClient.get<ApiResponse<Message[]> & { meta: PaginationMeta }>(
      `/conversations/${convId}/messages?page=${page}&limit=${limit}`,
    ),

  // Streaming via native fetch — Axios doesn't support ReadableStream.
  streamMessage: async (
    convId: string,
    content: string,
    onChunk: (text: string) => void,
    onDone: () => void,
    onError: (err: Error) => void,
  ): Promise<void> => {
    const token = useAuthStore.getState().accessToken
    try {
      const res = await fetch(`${BASE}/conversations/${convId}/messages`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ content, stream: true }),
      })

      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        throw new Error(err.message ?? `HTTP ${res.status}`)
      }

      const reader = res.body!.getReader()
      const decoder = new TextDecoder()

      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        const text = decoder.decode(value, { stream: true })
        text.split('\n').forEach((line) => {
          if (!line.startsWith('data:')) return
          const data = line.slice(5).trim()
          if (data === '[DONE]') return
          if (data) onChunk(data)
        })
      }
      onDone()
    } catch (err) {
      onError(err instanceof Error ? err : new Error(String(err)))
    }
  },
}
