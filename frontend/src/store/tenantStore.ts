import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { Tenant } from '@/types/api'

interface TenantState {
  tenants: Tenant[]
  activeTenantId: string | null
  setTenants: (tenants: Tenant[]) => void
  setActiveTenant: (id: string | null) => void
  activeTenant: () => Tenant | null
}

export const useTenantStore = create<TenantState>()(
  persist(
    (set, get) => ({
      tenants: [],
      activeTenantId: null,

      setTenants: (tenants) =>
        set((state) => ({
          tenants,
          // Auto-select first tenant if none selected yet
          activeTenantId: state.activeTenantId ?? tenants[0]?.id ?? null,
        })),

      setActiveTenant: (id) => set({ activeTenantId: id }),

      activeTenant: () => {
        const { tenants, activeTenantId } = get()
        return tenants.find((t) => t.id === activeTenantId) ?? null
      },
    }),
    {
      name: 'jarvas-tenant',
      partialize: (state) => ({
        activeTenantId: state.activeTenantId,
      }),
    },
  ),
)
