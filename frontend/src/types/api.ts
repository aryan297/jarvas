export interface ApiResponse<T> {
  success: boolean
  data?: T
  meta?: PaginationMeta
  code?: string
  message?: string
}

export interface PaginationMeta {
  page: number
  limit: number
  total: number
  total_pages: number
}

export interface User {
  id: string
  email: string
  full_name: string
  avatar_url?: string
  role: 'ADMIN' | 'USER' | 'PREMIUM_USER'
  email_verified: boolean
}

export interface TokenPair {
  access_token: string
  refresh_token: string
  expires_in: number
  token_type: string
}

export interface AuthResponse extends TokenPair {
  user: User
}

export interface Conversation {
  id: string
  agent_id?: string
  title: string
  status: 'ACTIVE' | 'ARCHIVED' | 'DELETED'
  created_at: string
  updated_at: string
}

export interface Message {
  id: string
  role: 'USER' | 'ASSISTANT' | 'SYSTEM' | 'TOOL'
  content: string
  model?: string
  created_at: string
}

export interface Agent {
  id: string
  name: string
  description?: string
  type: string
  model: string
  temperature: number
  max_tokens: number
  tools_enabled: string[]
  memory_enabled: boolean
  rag_enabled: boolean
  is_active: boolean
  created_at: string
}

export interface Document {
  id: string
  name: string
  type: string
  mime_type: string
  size_bytes: number
  status: 'UPLOADED' | 'PROCESSING' | 'INDEXED' | 'FAILED'
  chunk_count: number
  created_at: string
}

export interface Memory {
  id: string
  type: string
  content: string
  importance: number
  created_at: string
}
