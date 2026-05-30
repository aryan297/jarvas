import { NavLink } from 'react-router-dom'
import {
  LayoutDashboard,
  MessageSquare,
  FileText,
  Bot,
  Brain,
  GitBranch,
  Settings,
} from 'lucide-react'
import { clsx } from 'clsx'

const navItems = [
  { to: '/dashboard',  label: 'Dashboard',  Icon: LayoutDashboard },
  { to: '/chat',       label: 'Chat',        Icon: MessageSquare },
  { to: '/documents',  label: 'Documents',   Icon: FileText },
  { to: '/agents',     label: 'Agents',      Icon: Bot },
  { to: '/memory',     label: 'Memory',      Icon: Brain },
  { to: '/workflows',  label: 'Workflows',   Icon: GitBranch },
  { to: '/settings',   label: 'Settings',    Icon: Settings },
]

export default function Sidebar() {
  return (
    <aside className="w-64 bg-card border-r border-border flex flex-col">
      {/* Logo */}
      <div className="h-16 flex items-center px-6 border-b border-border">
        <span className="text-xl font-bold text-primary">Jarvas</span>
        <span className="ml-2 text-xs text-muted-foreground font-medium">AI Platform</span>
      </div>

      {/* Navigation */}
      <nav className="flex-1 px-3 py-4 space-y-1">
        {navItems.map(({ to, label, Icon }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) =>
              clsx(
                'flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium transition-colors',
                isActive
                  ? 'bg-primary text-primary-foreground'
                  : 'text-muted-foreground hover:bg-muted hover:text-foreground',
              )
            }
          >
            <Icon className="h-4 w-4" />
            {label}
          </NavLink>
        ))}
      </nav>
    </aside>
  )
}
