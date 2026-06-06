import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { useAuthStore } from '@/store/authStore'
import { chatService } from '@/services/chat.service'
import { documentService } from '@/services/document.service'
import { memoryService } from '@/services/memory.service'
import { agentService } from '@/services/agent.service'
import { workflowService } from '@/services/workflow.service'
import {
  MessageSquare, FileText, Brain, Bot, GitBranch, Upload,
  Plus, Mic, ArrowRight,
} from 'lucide-react'

interface StatCard {
  label: string
  value: number | string
  Icon: React.ElementType
  color: string
  to: string
}

export default function DashboardPage() {
  const { user } = useAuthStore()
  const navigate = useNavigate()

  // Fetch counts from all modules in parallel
  const { data: convMeta }     = useQuery({ queryKey: ['dashboard-convs'],      queryFn: () => chatService.listConversations(1, 1),      select: (r) => r.data?.meta })
  const { data: docMeta }      = useQuery({ queryKey: ['dashboard-docs'],       queryFn: () => documentService.list(1, 1),               select: (r) => r.data?.meta })
  const { data: memMeta }      = useQuery({ queryKey: ['dashboard-memories'],   queryFn: () => memoryService.list(1, 1),                  select: (r) => r.data?.meta })
  const { data: agentMeta }    = useQuery({ queryKey: ['dashboard-agents'],     queryFn: () => agentService.list(1, 1),                   select: (r) => r.data?.meta })
  const { data: workflowMeta } = useQuery({ queryKey: ['dashboard-workflows'],  queryFn: () => workflowService.list(1, 1),                select: (r) => r.data?.meta })

  // Recent conversations
  const { data: recentConvs = [] } = useQuery({
    queryKey: ['dashboard-recent-convs'],
    queryFn: () => chatService.listConversations(1, 5),
    select: (r) => r.data?.data ?? [],
  })

  const stats: StatCard[] = [
    { label: 'Conversations', value: convMeta?.total ?? '—',  Icon: MessageSquare, color: 'text-blue-600  bg-blue-50',   to: '/chat'      },
    { label: 'Documents',     value: docMeta?.total ?? '—',   Icon: FileText,      color: 'text-green-600 bg-green-50',  to: '/documents' },
    { label: 'Memories',      value: memMeta?.total ?? '—',   Icon: Brain,         color: 'text-purple-600 bg-purple-50',to: '/memory'    },
    { label: 'Agents',        value: agentMeta?.total ?? '—', Icon: Bot,           color: 'text-orange-600 bg-orange-50',to: '/agents'    },
    { label: 'Workflows',     value: workflowMeta?.total ?? '—', Icon: GitBranch,  color: 'text-pink-600 bg-pink-50',   to: '/workflows' },
  ]

  const quickActions = [
    { label: 'New chat',       Icon: MessageSquare, to: '/chat',      primary: true  },
    { label: 'Upload document',Icon: Upload,        to: '/documents', primary: false },
    { label: 'Create agent',   Icon: Bot,           to: '/agents',    primary: false },
    { label: 'New workflow',   Icon: GitBranch,     to: '/workflows', primary: false },
  ]

  const greeting = () => {
    const h = new Date().getHours()
    if (h < 12) return 'Good morning'
    if (h < 17) return 'Good afternoon'
    return 'Good evening'
  }

  return (
    <div className="space-y-8 max-w-5xl">

      {/* Hero greeting */}
      <div>
        <h1 className="text-3xl font-bold">
          {greeting()}, {user?.full_name?.split(' ')[0] ?? 'there'} 👋
        </h1>
        <p className="text-muted-foreground mt-1">
          Here's what's happening in your Jarvas workspace.
        </p>
      </div>

      {/* Quick actions */}
      <div className="flex flex-wrap gap-3">
        {quickActions.map(({ label, Icon, to, primary }) => (
          <button
            key={to}
            onClick={() => navigate(to)}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
              primary
                ? 'bg-primary text-primary-foreground hover:bg-primary/90'
                : 'border bg-card hover:bg-muted'
            }`}
          >
            <Icon className="h-4 w-4" />
            {label}
          </button>
        ))}
      </div>

      {/* Stats grid */}
      <div>
        <h2 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide mb-3">Overview</h2>
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
          {stats.map(({ label, value, Icon, color, to }) => (
            <button
              key={label}
              onClick={() => navigate(to)}
              className="rounded-xl border bg-card p-4 text-left hover:shadow-sm transition-shadow group"
            >
              <div className={`inline-flex p-2 rounded-lg ${color} mb-3`}>
                <Icon className="h-5 w-5" />
              </div>
              <p className="text-2xl font-bold">{value}</p>
              <p className="text-xs text-muted-foreground mt-0.5 flex items-center gap-1">
                {label}
                <ArrowRight className="h-3 w-3 opacity-0 group-hover:opacity-100 transition-opacity" />
              </p>
            </button>
          ))}
        </div>
      </div>

      {/* Recent conversations */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">Recent conversations</h2>
          <button
            onClick={() => navigate('/chat')}
            className="text-xs text-primary hover:underline flex items-center gap-1"
          >
            View all <ArrowRight className="h-3 w-3" />
          </button>
        </div>

        {recentConvs.length === 0 ? (
          <div className="rounded-xl border bg-card p-8 text-center text-muted-foreground">
            <MessageSquare className="h-10 w-10 mx-auto mb-3 opacity-30" />
            <p className="font-medium">No conversations yet</p>
            <p className="text-sm mt-1">Start a chat to see recent activity here.</p>
            <button
              onClick={() => navigate('/chat')}
              className="mt-4 px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm hover:bg-primary/90"
            >
              <Plus className="h-4 w-4 inline mr-1" /> New chat
            </button>
          </div>
        ) : (
          <div className="space-y-2">
            {recentConvs.map((conv) => (
              <button
                key={conv.id}
                onClick={() => navigate(`/chat/${conv.id}`)}
                className="w-full rounded-lg border bg-card px-4 py-3 flex items-center gap-3 text-left hover:bg-muted/50 transition-colors group"
              >
                <MessageSquare className="h-4 w-4 text-muted-foreground shrink-0" />
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium truncate">
                    {conv.title || 'Untitled conversation'}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {new Date(conv.updated_at).toLocaleDateString()}
                  </p>
                </div>
                <ArrowRight className="h-4 w-4 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity shrink-0" />
              </button>
            ))}
          </div>
        )}
      </div>

      {/* Feature highlights */}
      <div>
        <h2 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide mb-3">What Jarvas can do</h2>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          {[
            { Icon: Brain,      title: 'Long-term memory',    desc: 'Jarvas remembers facts about you across conversations automatically.' },
            { Icon: FileText,   title: 'Document RAG',        desc: 'Upload PDFs and docs — Jarvas cites them when answering questions.'  },
            { Icon: Bot,        title: 'Custom AI agents',    desc: 'Build agents with custom prompts, tools, and model settings.'         },
            { Icon: GitBranch,  title: 'Workflows',           desc: 'Automate multi-step tasks on a schedule or with a single click.'     },
            { Icon: Mic,        title: 'Voice input',         desc: 'Record audio and Whisper transcribes it into a chat message.'        },
            { Icon: MessageSquare, title: 'SSE streaming',    desc: 'Real-time streaming responses — no waiting for the full reply.'     },
          ].map(({ Icon, title, desc }) => (
            <div key={title} className="rounded-xl border bg-card p-4 space-y-2">
              <div className="flex items-center gap-2">
                <Icon className="h-4 w-4 text-primary" />
                <p className="text-sm font-semibold">{title}</p>
              </div>
              <p className="text-xs text-muted-foreground leading-relaxed">{desc}</p>
            </div>
          ))}
        </div>
      </div>

    </div>
  )
}
