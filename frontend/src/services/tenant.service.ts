import { apiClient } from './api'
import type { ApiResponse, Tenant, TenantMember } from '@/types/api'

// Re-export canonical types so existing imports from this file still work.
export type { Tenant, TenantMember }

export const tenantService = {
  create: (name: string) =>
    apiClient.post<ApiResponse<Tenant>>('/tenants', { name }),

  list: () =>
    apiClient.get<ApiResponse<Tenant[]>>('/tenants'),

  getById: (id: string) =>
    apiClient.get<ApiResponse<Tenant>>(`/tenants/${id}`),

  listMembers: (tenantId: string) =>
    apiClient.get<ApiResponse<TenantMember[]>>(`/tenants/${tenantId}/members`),

  invite: (tenantId: string, email: string, role: 'ADMIN' | 'MEMBER' = 'MEMBER') =>
    apiClient.post(`/tenants/${tenantId}/invite`, { email, role }),

  removeMember: (tenantId: string, userId: string) =>
    apiClient.delete(`/tenants/${tenantId}/members/${userId}`),
}
