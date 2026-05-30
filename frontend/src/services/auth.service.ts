import { apiClient } from './api'
import type { ApiResponse, AuthResponse, User } from '@/types/api'

interface RegisterPayload { email: string; password: string; full_name: string }
interface LoginPayload    { email: string; password: string }

export const authService = {
  register: (data: RegisterPayload) =>
    apiClient.post<ApiResponse<AuthResponse>>('/auth/register', data),

  login: (data: LoginPayload) =>
    apiClient.post<ApiResponse<AuthResponse>>('/auth/login', data),

  logout: () =>
    apiClient.post('/auth/logout'),

  refresh: () =>
    apiClient.post<ApiResponse<{ access_token: string }>>('/auth/refresh'),

  me: () =>
    apiClient.get<ApiResponse<User>>('/auth/me'),

  googleLoginUrl: () =>
    apiClient.get<ApiResponse<{ url: string }>>('/auth/google/login'),
}
