import { useEffect, useRef, useState } from 'react'
import { useAuthStore } from '@/store/authStore'
import { useTenantStore } from '@/store/tenantStore'
import { authService } from '@/services/auth.service'
import { tenantService } from '@/services/tenant.service'
import { LogOut, User, ChevronDown, Building2 } from 'lucide-react'
import toast from 'react-hot-toast'
import { useNavigate } from 'react-router-dom'

export default function Header() {
  const { user, logout } = useAuthStore()
  const { tenants, activeTenantId, activeTenant, setTenants, setActiveTenant } = useTenantStore()
  const navigate = useNavigate()
  const [dropdownOpen, setDropdownOpen] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)

  // Load tenants once on mount
  useEffect(() => {
    tenantService.list()
      .then((res) => {
        const data = res.data?.data ?? []
        if (data.length > 0) setTenants(data)
      })
      .catch(() => {/* silently ignore — user may have no tenants yet */})
  }, [])

  // Close dropdown on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setDropdownOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  const handleLogout = async () => {
    try {
      await authService.logout()
    } finally {
      logout()
      navigate('/login')
      toast.success('Logged out')
    }
  }

  const current = activeTenant()

  return (
    <header className="h-16 border-b border-border bg-card flex items-center justify-between px-6">

      {/* Tenant switcher */}
      <div className="relative" ref={dropdownRef}>
        {tenants.length > 0 ? (
          <button
            onClick={() => setDropdownOpen((o) => !o)}
            className="flex items-center gap-2 px-3 py-1.5 rounded-md hover:bg-muted transition-colors text-sm"
          >
            <Building2 className="h-4 w-4 text-muted-foreground" />
            <span className="font-medium max-w-[160px] truncate">
              {current?.name ?? 'Select workspace'}
            </span>
            <ChevronDown className="h-3 w-3 text-muted-foreground" />
          </button>
        ) : (
          <div className="flex items-center gap-2 px-3 py-1.5 text-sm text-muted-foreground">
            <Building2 className="h-4 w-4" />
            <span>Personal</span>
          </div>
        )}

        {dropdownOpen && tenants.length > 0 && (
          <div className="absolute top-full left-0 mt-1 w-56 rounded-lg border bg-card shadow-md z-50 overflow-hidden">
            <div className="px-3 py-2 text-xs font-semibold text-muted-foreground uppercase tracking-wide border-b">
              Workspaces
            </div>
            {tenants.map((t) => (
              <button
                key={t.id}
                onClick={() => { setActiveTenant(t.id); setDropdownOpen(false) }}
                className={`w-full text-left px-3 py-2.5 text-sm flex items-center gap-2 transition-colors ${
                  t.id === activeTenantId
                    ? 'bg-primary/10 text-primary font-medium'
                    : 'hover:bg-muted text-foreground'
                }`}
              >
                <Building2 className="h-3.5 w-3.5 shrink-0" />
                <span className="truncate">{t.name}</span>
                {t.plan !== 'free' && (
                  <span className="ml-auto text-xs bg-primary/10 text-primary rounded px-1.5 py-0.5 shrink-0">
                    {t.plan}
                  </span>
                )}
              </button>
            ))}
          </div>
        )}
      </div>

      {/* User info + logout */}
      <div className="flex items-center gap-4">
        <div className="flex items-center gap-2 text-sm">
          {user?.avatar_url ? (
            <img src={user.avatar_url} alt={user.full_name} className="h-8 w-8 rounded-full" />
          ) : (
            <div className="h-8 w-8 rounded-full bg-primary flex items-center justify-center">
              <User className="h-4 w-4 text-primary-foreground" />
            </div>
          )}
          <span className="font-medium">{user?.full_name}</span>
        </div>
        <button
          onClick={handleLogout}
          className="p-2 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
          title="Logout"
        >
          <LogOut className="h-4 w-4" />
        </button>
      </div>
    </header>
  )
}
