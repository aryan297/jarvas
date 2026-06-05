import type { Agent } from '@/types/api'

const TYPE_COLORS: Record<string, string> = {
  CUSTOM:     'bg-gray-100 text-gray-800',
  RESEARCH:   'bg-blue-100 text-blue-800',
  CODING:     'bg-green-100 text-green-800',
  PLANNING:   'bg-yellow-100 text-yellow-800',
  SUPERVISOR: 'bg-purple-100 text-purple-800',
  WORKFLOW:   'bg-orange-100 text-orange-800',
}

const TOOL_LABELS: Record<string, string> = {
  web_search: 'Web Search',
  calculator: 'Calculator',
}

interface Props {
  agent: Agent
  onEdit: (agent: Agent) => void
  onDelete: (id: string) => void
  onChat: (agent: Agent) => void
}

export default function AgentCard({ agent, onEdit, onDelete, onChat }: Props) {
  const badgeClass = TYPE_COLORS[agent.type] ?? TYPE_COLORS.CUSTOM

  return (
    <div className="rounded-lg border bg-card p-4 flex flex-col gap-3">
      {/* Header */}
      <div className="flex items-start justify-between gap-2">
        <div className="flex items-center gap-2 min-w-0">
          <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium whitespace-nowrap ${badgeClass}`}>
            {agent.type}
          </span>
          <h3 className="font-semibold text-sm truncate">{agent.name}</h3>
        </div>
        <div className="flex gap-1 shrink-0">
          <button
            onClick={() => onEdit(agent)}
            className="text-xs text-muted-foreground hover:text-foreground px-2 py-1 rounded border transition-colors"
          >
            Edit
          </button>
          <button
            onClick={() => onDelete(agent.id)}
            className="text-xs text-muted-foreground hover:text-destructive px-2 py-1 rounded border transition-colors"
          >
            Delete
          </button>
        </div>
      </div>

      {/* Description */}
      {agent.description && (
        <p className="text-sm text-muted-foreground line-clamp-2">{agent.description}</p>
      )}

      {/* Meta */}
      <div className="flex flex-wrap gap-1.5 text-xs text-muted-foreground">
        <span className="bg-muted rounded px-1.5 py-0.5">{agent.model}</span>
        <span className="bg-muted rounded px-1.5 py-0.5">t={agent.temperature}</span>
        {agent.memory_enabled && (
          <span className="bg-muted rounded px-1.5 py-0.5">Memory</span>
        )}
        {agent.rag_enabled && (
          <span className="bg-muted rounded px-1.5 py-0.5">RAG</span>
        )}
      </div>

      {/* Tools */}
      {agent.tools_enabled.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {agent.tools_enabled.map((t) => (
            <span key={t} className="text-xs bg-primary/10 text-primary rounded px-1.5 py-0.5">
              {TOOL_LABELS[t] ?? t}
            </span>
          ))}
        </div>
      )}

      {/* Chat button */}
      <button
        onClick={() => onChat(agent)}
        className="mt-auto w-full py-1.5 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors"
      >
        Chat with agent
      </button>
    </div>
  )
}
