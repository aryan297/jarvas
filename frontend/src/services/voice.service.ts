import { apiClient } from './api'
import { useAuthStore } from '@/store/authStore'
import type { ApiResponse } from '@/types/api'

export interface VoiceSession {
  id: string
  conversation_id: string
  status: 'PENDING' | 'PROCESSING' | 'COMPLETED' | 'FAILED'
  transcript?: string
  duration_seconds?: number
  language_code?: string
  created_at: string
}

export interface UploadVoiceResponse {
  session_id: string
  status: string
}

const BASE = import.meta.env.VITE_API_URL ?? '/api/v1'

export const voiceService = {
  /**
   * Upload an audio blob. Returns session_id to poll.
   * Uses native fetch so we can send FormData with the binary blob.
   */
  upload: async (
    blob: Blob,
    conversationId: string,
    language = '',
  ): Promise<UploadVoiceResponse> => {
    const token = useAuthStore.getState().accessToken
    const form = new FormData()
    form.append('audio', blob, `audio.webm`)
    form.append('conversation_id', conversationId)
    if (language) form.append('language', language)

    const res = await fetch(`${BASE}/voice/upload`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
      body: form,
    })
    if (!res.ok) throw new Error(`voice upload failed: ${res.status}`)
    const json = await res.json()
    return json.data as UploadVoiceResponse
  },

  getSession: (id: string) =>
    apiClient.get<ApiResponse<VoiceSession>>(`/voice/sessions/${id}`),

  listSessions: () =>
    apiClient.get<ApiResponse<VoiceSession[]>>('/voice/sessions'),

  /** Poll until status is COMPLETED or FAILED, or timeout (30s). */
  pollUntilDone: async (
    sessionId: string,
    onUpdate?: (s: VoiceSession) => void,
  ): Promise<VoiceSession> => {
    const maxAttempts = 15 // 15 × 2s = 30s
    for (let i = 0; i < maxAttempts; i++) {
      await new Promise((r) => setTimeout(r, 2000))
      const res = await voiceService.getSession(sessionId)
      const session = res.data.data!
      onUpdate?.(session)
      if (session.status === 'COMPLETED' || session.status === 'FAILED') {
        return session
      }
    }
    throw new Error('Transcription timed out')
  },
}
