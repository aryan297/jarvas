import axios, { AxiosError } from 'axios'
import { useAuthStore } from '@/store/authStore'
import { useTenantStore } from '@/store/tenantStore'

const BASE_URL = import.meta.env.VITE_API_URL ?? '/api/v1'

export const apiClient = axios.create({
  baseURL: BASE_URL,
  withCredentials: true, // send HttpOnly refresh cookie automatically
  headers: { 'Content-Type': 'application/json' },
})

// Attach access token + active workspace on every request.
apiClient.interceptors.request.use((config) => {
  const token = useAuthStore.getState().accessToken
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  const tenantId = useTenantStore.getState().activeTenantId
  if (tenantId) {
    config.headers['X-Tenant-ID'] = tenantId
  }
  return config
})

// Silently refresh the access token on 401, retry once.
let isRefreshing = false
let pendingQueue: Array<(token: string) => void> = []

apiClient.interceptors.response.use(
  (res) => res,
  async (error: AxiosError) => {
    const original = error.config as typeof error.config & { _retry?: boolean }

    if (error.response?.status !== 401 || original._retry) {
      return Promise.reject(error)
    }

    if (isRefreshing) {
      return new Promise((resolve) => {
        pendingQueue.push((token) => {
          original!.headers!.Authorization = `Bearer ${token}`
          resolve(apiClient(original!))
        })
      })
    }

    original._retry = true
    isRefreshing = true

    try {
      const res = await apiClient.post<{ data: { access_token: string } }>('/auth/refresh')
      const newToken = res.data.data!.access_token
      useAuthStore.getState().setAccessToken(newToken)
      pendingQueue.forEach((cb) => cb(newToken))
      pendingQueue = []
      original!.headers!.Authorization = `Bearer ${newToken}`
      return apiClient(original!)
    } catch {
      useAuthStore.getState().logout()
      return Promise.reject(error)
    } finally {
      isRefreshing = false
    }
  },
)
