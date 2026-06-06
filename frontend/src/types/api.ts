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

// ── Auth ──────────────────────────────────────────────────────────────────────

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

// ── Chat ──────────────────────────────────────────────────────────────────────

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

// ── Agents ────────────────────────────────────────────────────────────────────

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

// ── Documents ─────────────────────────────────────────────────────────────────

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

// ── Memory ────────────────────────────────────────────────────────────────────

export interface Memory {
  id: string
  type: string
  content: string
  importance: number
  access_count: number
  created_at: string
}

// ── Voice ─────────────────────────────────────────────────────────────────────

export interface VoiceSession {
  id: string
  conversation_id: string
  status: 'PENDING' | 'PROCESSING' | 'COMPLETED' | 'FAILED'
  transcript?: string
  duration_seconds?: number
  language_code?: string
  created_at: string
}

// ── Workflows ─────────────────────────────────────────────────────────────────

export interface WorkflowNode {
  id: string
  type: 'agent' | 'tool' | 'condition' | 'delay'
  config: Record<string, unknown>
}

export interface WorkflowEdge {
  from: string
  to: string
  condition?: string
}

export interface WorkflowDefinition {
  nodes: WorkflowNode[]
  edges: WorkflowEdge[]
  trigger: { type: string; cron_expr?: string; event_name?: string }
}

export interface Workflow {
  id: string
  name: string
  description?: string
  status: 'DRAFT' | 'ACTIVE' | 'PAUSED' | 'ARCHIVED'
  definition: WorkflowDefinition
  trigger_type?: string
  cron_expr?: string
  created_at: string
  updated_at: string
}

export interface WorkflowRun {
  id: string
  workflow_id: string
  status: 'PENDING' | 'RUNNING' | 'COMPLETED' | 'FAILED' | 'CANCELLED'
  result?: Record<string, unknown>
  error_msg?: string
  started_at?: string
  completed_at?: string
  created_at: string
}

// ── Tenants ───────────────────────────────────────────────────────────────────

export interface Tenant {
  id: string
  name: string
  slug: string
  plan: 'free' | 'pro' | 'enterprise'
  owner_id: string
  is_active: boolean
  created_at: string
}

export interface TenantMember {
  id: string
  user_id: string
  user_email: string
  user_full_name: string
  role: 'OWNER' | 'ADMIN' | 'MEMBER'
  joined_at: string
}
