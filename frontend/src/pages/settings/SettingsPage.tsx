import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Building2, Users, Plus, Trash2 } from 'lucide-react'
import { tenantService, type Tenant, type TenantMember } from '@/services/tenant.service'
import { useTenantStore } from '@/store/tenantStore'
import { useAuthStore } from '@/store/authStore'
import toast from 'react-hot-toast'

type Tab = 'workspaces' | 'members'

export default function SettingsPage() {
  const qc = useQueryClient()
  const { user } = useAuthStore()
  const { tenants, activeTenantId, setTenants, setActiveTenant } = useTenantStore()

  const [tab, setTab] = useState<Tab>('workspaces')
  const [newWorkspaceName, setNewWorkspaceName] = useState('')
  const [showCreate, setShowCreate] = useState(false)
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteRole, setInviteRole] = useState<'ADMIN' | 'MEMBER'>('MEMBER')

  const activeTenant = tenants.find((t) => t.id === activeTenantId) ?? null

  // ── Members ────────────────────────────────────────────────────────────────
  const { data: members = [] } = useQuery<TenantMember[]>({
    queryKey: ['tenant-members', activeTenantId],
    queryFn: async () => {
      if (!activeTenantId) return []
      const res = await tenantService.listMembers(activeTenantId)
      return res.data?.data ?? []
    },
    enabled: !!activeTenantId && tab === 'members',
  })

  // ── Create workspace ──────────────────────────────────────────────────────
  const createMutation = useMutation({
    mutationFn: () => tenantService.create(newWorkspaceName),
    onSuccess: (res) => {
      const t = res.data?.data as Tenant
      const next = [...tenants, t]
      setTenants(next)
      setActiveTenant(t.id)
      setNewWorkspaceName('')
      setShowCreate(false)
      toast.success('Workspace created')
    },
    onError: () => toast.error('Failed to create workspace'),
  })

  // ── Invite member ─────────────────────────────────────────────────────────
  const inviteMutation = useMutation({
    mutationFn: () => tenantService.invite(activeTenantId!, inviteEmail, inviteRole),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['tenant-members', activeTenantId] })
      setInviteEmail('')
      toast.success(`${inviteEmail} added to workspace`)
    },
    onError: () => toast.error('User not found or already a member'),
  })

  // ── Remove member ─────────────────────────────────────────────────────────
  const removeMutation = useMutation({
    mutationFn: (userId: string) => tenantService.removeMember(activeTenantId!, userId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['tenant-members', activeTenantId] })
      toast.success('Member removed')
    },
  })

  const ROLE_BADGE: Record<string, string> = {
    OWNER:  'bg-purple-100 text-purple-800',
    ADMIN:  'bg-blue-100  text-blue-800',
    MEMBER: 'bg-gray-100  text-gray-700',
  }

  return (
    <div className="space-y-6 max-w-3xl">
      <h1 className="text-2xl font-bold">Settings</h1>

      {/* Tabs */}
      <div className="flex gap-1 border-b">
        {([
          { key: 'workspaces', label: 'Workspaces', Icon: Building2 },
          { key: 'members',    label: 'Members',    Icon: Users },
        ] as { key: Tab; label: string; Icon: React.ElementType }[]).map(({ key, label, Icon }) => (
          <button
            key={key}
            onClick={() => setTab(key)}
            className={`flex items-center gap-2 px-4 py-2 text-sm font-medium transition-colors ${
              tab === key
                ? 'border-b-2 border-primary text-primary'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            <Icon className="h-4 w-4" />
            {label}
          </button>
        ))}
      </div>

      {/* ── Workspaces tab ── */}
      {tab === 'workspaces' && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <p className="text-sm text-muted-foreground">
              {tenants.length} workspace{tenants.length !== 1 ? 's' : ''}
            </p>
            {!showCreate && (
              <button
                onClick={() => setShowCreate(true)}
                className="flex items-center gap-1.5 px-3 py-1.5 text-sm bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
              >
                <Plus className="h-4 w-4" /> New workspace
              </button>
            )}
          </div>

          {showCreate && (
            <div className="rounded-lg border bg-card p-4 flex gap-2">
              <input
                value={newWorkspaceName}
                onChange={(e) => setNewWorkspaceName(e.target.value)}
                placeholder="Workspace name"
                className="flex-1 rounded-md border bg-background px-3 py-2 text-sm"
                onKeyDown={(e) => { if (e.key === 'Enter' && newWorkspaceName.trim()) createMutation.mutate() }}
                autoFocus
              />
              <button
                disabled={!newWorkspaceName.trim() || createMutation.isPending}
                onClick={() => createMutation.mutate()}
                className="px-4 py-2 rounded-md bg-primary text-primary-foreground text-sm font-medium disabled:opacity-50"
              >
                {createMutation.isPending ? 'Creating…' : 'Create'}
              </button>
              <button onClick={() => setShowCreate(false)} className="px-3 py-2 rounded-md border text-sm">
                Cancel
              </button>
            </div>
          )}

          <div className="space-y-2">
            {tenants.map((t) => (
              <div
                key={t.id}
                onClick={() => setActiveTenant(t.id)}
                className={`rounded-lg border p-4 cursor-pointer transition-colors ${
                  t.id === activeTenantId ? 'border-primary bg-primary/5' : 'bg-card hover:bg-muted/50'
                }`}
              >
                <div className="flex items-center justify-between">
                  <div>
                    <p className="font-semibold">{t.name}</p>
                    <p className="text-xs text-muted-foreground mt-0.5">/{t.slug}</p>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-xs bg-muted rounded-full px-2.5 py-0.5 font-medium capitalize">
                      {t.plan}
                    </span>
                    {t.id === activeTenantId && (
                      <span className="text-xs text-primary font-medium">Active</span>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* ── Members tab ── */}
      {tab === 'members' && (
        <div className="space-y-4">
          {!activeTenant ? (
            <p className="text-sm text-muted-foreground">Select a workspace first.</p>
          ) : (
            <>
              <div className="flex items-center justify-between">
                <p className="text-sm font-medium">{activeTenant.name}</p>
                <p className="text-sm text-muted-foreground">{members.length} member{members.length !== 1 ? 's' : ''}</p>
              </div>

              {/* Invite form — only shown to owner/admin */}
              {members.some((m) => m.user_id === user?.id && (m.role === 'OWNER' || m.role === 'ADMIN')) && (
                <div className="rounded-lg border bg-card p-4 space-y-3">
                  <p className="text-sm font-medium">Invite member</p>
                  <div className="flex gap-2">
                    <input
                      type="email"
                      value={inviteEmail}
                      onChange={(e) => setInviteEmail(e.target.value)}
                      placeholder="user@example.com"
                      className="flex-1 rounded-md border bg-background px-3 py-2 text-sm"
                    />
                    <select
                      value={inviteRole}
                      onChange={(e) => setInviteRole(e.target.value as 'ADMIN' | 'MEMBER')}
                      className="rounded-md border bg-background px-3 py-2 text-sm"
                    >
                      <option value="MEMBER">Member</option>
                      <option value="ADMIN">Admin</option>
                    </select>
                    <button
                      disabled={!inviteEmail.trim() || inviteMutation.isPending}
                      onClick={() => inviteMutation.mutate()}
                      className="px-4 py-2 rounded-md bg-primary text-primary-foreground text-sm font-medium disabled:opacity-50"
                    >
                      {inviteMutation.isPending ? 'Adding…' : 'Add'}
                    </button>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    The user must already have a Jarvas account.
                  </p>
                </div>
              )}

              {/* Member list */}
              <div className="space-y-2">
                {members.map((m) => (
                  <div key={m.id} className="rounded-lg border bg-card p-3 flex items-center gap-3">
                    <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
                      <span className="text-sm font-medium text-primary">
                        {(m.user_full_name || m.user_email)[0].toUpperCase()}
                      </span>
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium truncate">{m.user_full_name || m.user_email}</p>
                      <p className="text-xs text-muted-foreground truncate">{m.user_email}</p>
                    </div>
                    <span className={`text-xs font-medium rounded-full px-2.5 py-0.5 shrink-0 ${ROLE_BADGE[m.role]}`}>
                      {m.role}
                    </span>
                    {m.role !== 'OWNER' && m.user_id !== user?.id && (
                      <button
                        onClick={() => removeMutation.mutate(m.user_id)}
                        className="shrink-0 p-1 text-muted-foreground hover:text-destructive transition-colors"
                        title="Remove member"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    )}
                  </div>
                ))}
              </div>
            </>
          )}
        </div>
      )}
    </div>
  )
}
