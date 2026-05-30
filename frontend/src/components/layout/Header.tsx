import { useAuthStore } from '@/store/authStore'
import { authService } from '@/services/auth.service'
import { LogOut, User } from 'lucide-react'
import toast from 'react-hot-toast'
import { useNavigate } from 'react-router-dom'

export default function Header() {
  const { user, logout } = useAuthStore()
  const navigate = useNavigate()

  const handleLogout = async () => {
    try {
      await authService.logout()
    } finally {
      logout()
      navigate('/login')
      toast.success('Logged out')
    }
  }

  return (
    <header className="h-16 border-b border-border bg-card flex items-center justify-between px-6">
      <div />
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
