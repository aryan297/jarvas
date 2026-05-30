import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '@/store/authStore'
import { authService } from '@/services/auth.service'
import toast from 'react-hot-toast'
import { clsx } from 'clsx'

type Tab = 'login' | 'register'

const GoogleIcon = () => (
  <svg className="h-4 w-4 shrink-0" viewBox="0 0 24 24">
    <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
    <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
    <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
    <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
  </svg>
)

const Divider = () => (
  <div className="relative my-5">
    <div className="absolute inset-0 flex items-center">
      <div className="w-full border-t border-border" />
    </div>
    <div className="relative flex justify-center text-xs uppercase">
      <span className="bg-card px-2 text-muted-foreground">or</span>
    </div>
  </div>
)

function extractError(err: unknown) {
  return (err as { response?: { data?: { message?: string } } })
    ?.response?.data?.message
}

export default function LoginPage() {
  const [tab, setTab] = useState<Tab>('login')
  const setAuth = useAuthStore((s) => s.setAuth)
  const navigate = useNavigate()

  // ── Login state ───────────────────────────────────────────────────────────
  const [loginEmail, setLoginEmail]       = useState('')
  const [loginPassword, setLoginPassword] = useState('')
  const [loginLoading, setLoginLoading]   = useState(false)

  // ── Register state ────────────────────────────────────────────────────────
  const [regName, setRegName]         = useState('')
  const [regEmail, setRegEmail]       = useState('')
  const [regPassword, setRegPassword] = useState('')
  const [regLoading, setRegLoading]   = useState(false)

  // ── Handlers ──────────────────────────────────────────────────────────────
  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoginLoading(true)
    try {
      const res = await authService.login({ email: loginEmail, password: loginPassword })
      const { user, access_token } = res.data.data!
      setAuth(user, access_token)
      navigate('/dashboard')
    } catch (err) {
      toast.error(extractError(err) ?? 'Login failed')
    } finally {
      setLoginLoading(false)
    }
  }

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault()
    setRegLoading(true)
    try {
      const res = await authService.register({
        email: regEmail,
        password: regPassword,
        full_name: regName,
      })
      const { user, access_token } = res.data.data!
      setAuth(user, access_token)
      toast.success('Welcome to Jarvas!')
      navigate('/dashboard')
    } catch (err) {
      toast.error(extractError(err) ?? 'Registration failed')
    } finally {
      setRegLoading(false)
    }
  }

  const handleGoogle = async () => {
    try {
      const res = await authService.googleLoginUrl()
      window.location.href = res.data.data!.url
    } catch {
      toast.error('Could not reach Google OAuth')
    }
  }

  // ── Shared input style ────────────────────────────────────────────────────
  const input =
    'w-full px-3 py-2 border border-border rounded-md bg-background text-sm ' +
    'focus:outline-none focus:ring-2 focus:ring-primary placeholder:text-muted-foreground'

  return (
    <div className="min-h-screen flex items-center justify-center bg-background px-4">
      <div className="w-full max-w-md">

        {/* Logo */}
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-primary">Jarvas</h1>
          <p className="text-muted-foreground mt-1 text-sm">Your personal AI platform</p>
        </div>

        <div className="bg-card rounded-xl border border-border shadow-sm overflow-hidden">

          {/* Tab switcher */}
          <div className="flex border-b border-border">
            {(['login', 'register'] as Tab[]).map((t) => (
              <button
                key={t}
                onClick={() => setTab(t)}
                className={clsx(
                  'flex-1 py-3 text-sm font-medium transition-colors',
                  tab === t
                    ? 'border-b-2 border-primary text-primary bg-card'
                    : 'text-muted-foreground hover:text-foreground bg-muted/40',
                )}
              >
                {t === 'login' ? 'Sign in' : 'Create account'}
              </button>
            ))}
          </div>

          <div className="p-8">

            {/* ── Login form ───────────────────────────────────────────── */}
            {tab === 'login' && (
              <form onSubmit={handleLogin} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium mb-1">Email</label>
                  <input
                    type="email"
                    required
                    autoComplete="email"
                    placeholder="you@example.com"
                    value={loginEmail}
                    onChange={(e) => setLoginEmail(e.target.value)}
                    className={input}
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Password</label>
                  <input
                    type="password"
                    required
                    autoComplete="current-password"
                    placeholder="••••••••"
                    value={loginPassword}
                    onChange={(e) => setLoginPassword(e.target.value)}
                    className={input}
                  />
                </div>
                <button
                  type="submit"
                  disabled={loginLoading}
                  className="w-full py-2 px-4 bg-primary text-primary-foreground rounded-md text-sm font-medium disabled:opacity-50 hover:bg-primary/90 transition-colors"
                >
                  {loginLoading ? 'Signing in…' : 'Sign in'}
                </button>

                <Divider />

                <button
                  type="button"
                  onClick={handleGoogle}
                  className="w-full py-2 px-4 border border-border rounded-md text-sm font-medium hover:bg-muted transition-colors flex items-center justify-center gap-2"
                >
                  <GoogleIcon />
                  Continue with Google
                </button>

                <p className="text-center text-sm text-muted-foreground pt-1">
                  No account?{' '}
                  <button
                    type="button"
                    onClick={() => setTab('register')}
                    className="text-primary hover:underline font-medium"
                  >
                    Create one
                  </button>
                </p>
              </form>
            )}

            {/* ── Register form ─────────────────────────────────────────── */}
            {tab === 'register' && (
              <form onSubmit={handleRegister} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium mb-1">Full name</label>
                  <input
                    type="text"
                    required
                    autoComplete="name"
                    placeholder="Jane Smith"
                    value={regName}
                    onChange={(e) => setRegName(e.target.value)}
                    className={input}
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Email</label>
                  <input
                    type="email"
                    required
                    autoComplete="email"
                    placeholder="you@example.com"
                    value={regEmail}
                    onChange={(e) => setRegEmail(e.target.value)}
                    className={input}
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Password</label>
                  <input
                    type="password"
                    required
                    minLength={8}
                    autoComplete="new-password"
                    placeholder="At least 8 characters"
                    value={regPassword}
                    onChange={(e) => setRegPassword(e.target.value)}
                    className={input}
                  />
                </div>
                <button
                  type="submit"
                  disabled={regLoading}
                  className="w-full py-2 px-4 bg-primary text-primary-foreground rounded-md text-sm font-medium disabled:opacity-50 hover:bg-primary/90 transition-colors"
                >
                  {regLoading ? 'Creating account…' : 'Create account'}
                </button>

                <Divider />

                <button
                  type="button"
                  onClick={handleGoogle}
                  className="w-full py-2 px-4 border border-border rounded-md text-sm font-medium hover:bg-muted transition-colors flex items-center justify-center gap-2"
                >
                  <GoogleIcon />
                  Continue with Google
                </button>

                <p className="text-center text-sm text-muted-foreground pt-1">
                  Already have an account?{' '}
                  <button
                    type="button"
                    onClick={() => setTab('login')}
                    className="text-primary hover:underline font-medium"
                  >
                    Sign in
                  </button>
                </p>
              </form>
            )}

          </div>
        </div>
      </div>
    </div>
  )
}
