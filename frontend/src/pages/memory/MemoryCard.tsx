import type { Memory } from '@/types/api'

const TYPE_COLORS: Record<string, string> = {
  FACT:         'bg-blue-100 text-blue-800',
  PREFERENCE:   'bg-purple-100 text-purple-800',
  EVENT:        'bg-green-100 text-green-800',
  SKILL:        'bg-yellow-100 text-yellow-800',
  RELATIONSHIP: 'bg-pink-100 text-pink-800',
}

interface Props {
  memory: Memory
  onDelete: (id: string) => void
}

export default function MemoryCard({ memory, onDelete }: Props) {
  const importancePct = Math.round(memory.importance * 100)
  const badgeClass = TYPE_COLORS[memory.type] ?? 'bg-gray-100 text-gray-800'

  return (
    <div className="rounded-lg border bg-card p-4 flex flex-col gap-3">
      <div className="flex items-start justify-between gap-2">
        <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${badgeClass}`}>
          {memory.type}
        </span>
        <button
          onClick={() => onDelete(memory.id)}
          className="text-muted-foreground hover:text-destructive transition-colors text-sm leading-none"
          aria-label="Delete memory"
        >
          ×
        </button>
      </div>

      <p className="text-sm text-foreground leading-relaxed">{memory.content}</p>

      <div className="flex items-center gap-2">
        <span className="text-xs text-muted-foreground">Importance</span>
        <div className="flex-1 h-1.5 rounded-full bg-muted overflow-hidden">
          <div
            className="h-full rounded-full bg-primary transition-all"
            style={{ width: `${importancePct}%` }}
          />
        </div>
        <span className="text-xs text-muted-foreground w-8 text-right">{importancePct}%</span>
      </div>

      <p className="text-xs text-muted-foreground">
        {new Date(memory.created_at).toLocaleDateString()}
      </p>
    </div>
  )
}
