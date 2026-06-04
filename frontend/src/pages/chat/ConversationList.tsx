import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { chatService } from '@/services/chat.service'
import { clsx } from 'clsx'
import { Plus, Trash2, MessageSquare } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import type { Conversation } from '@/types/api'
import toast from 'react-hot-toast'

interface Props {
  activeId?: string
  onSelect: (id: string) => void
}

export default function ConversationList({ activeId, onSelect }: Props) {
  const qc = useQueryClient()

  const { data: convs = [], isLoading } = useQuery({
    queryKey: ['conversations'],
    queryFn: async () => {
      const res = await chatService.listConversations()
      return res.data.data ?? []
    },
  })

  const { mutateAsync: create, isPending: creating } = useMutation({
    mutationFn: () => chatService.createConversation({}),
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: ['conversations'] })
      const conv = res.data.data
      if (conv) onSelect(conv.id)
    },
    onError: () => toast.error('Failed to create conversation'),
  })

  const { mutate: remove } = useMutation({
    mutationFn: (id: string) => chatService.deleteConversation(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['conversations'] }),
    onError: () => toast.error('Failed to delete conversation'),
  })

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-border">
        <span className="text-sm font-semibold">Conversations</span>
        <button
          onClick={() => create()}
          disabled={creating}
          title="New conversation"
          className="p-1.5 rounded-md hover:bg-muted transition-colors text-muted-foreground hover:text-foreground disabled:opacity-50"
        >
          <Plus className="h-4 w-4" />
        </button>
      </div>

      {/* List */}
      <div className="flex-1 overflow-y-auto py-1">
        {isLoading && (
          <div className="px-4 py-3 text-sm text-muted-foreground">Loading…</div>
        )}
        {!isLoading && convs.length === 0 && (
          <div className="px-4 py-8 text-center">
            <MessageSquare className="h-8 w-8 text-muted-foreground mx-auto mb-2" />
            <p className="text-sm text-muted-foreground">No conversations yet</p>
          </div>
        )}
        {convs.map((conv) => (
          <ConvItem
            key={conv.id}
            conv={conv}
            isActive={conv.id === activeId}
            onSelect={() => onSelect(conv.id)}
            onDelete={() => remove(conv.id)}
          />
        ))}
      </div>
    </div>
  )
}

function ConvItem({
  conv,
  isActive,
  onSelect,
  onDelete,
}: {
  conv: Conversation
  isActive: boolean
  onSelect: () => void
  onDelete: () => void
}) {
  const label = conv.title || 'New conversation'
  const ago = formatDistanceToNow(new Date(conv.updated_at), { addSuffix: true })

  return (
    <div
      role="button"
      onClick={onSelect}
      className={clsx(
        'group flex items-start justify-between gap-2 px-4 py-2.5 cursor-pointer rounded-md mx-1 my-0.5',
        isActive
          ? 'bg-primary/10 text-primary'
          : 'hover:bg-muted text-foreground',
      )}
    >
      <div className="min-w-0">
        <p className="text-sm font-medium truncate">{label}</p>
        <p className="text-xs text-muted-foreground truncate">{ago}</p>
      </div>
      <button
        onClick={(e) => { e.stopPropagation(); onDelete() }}
        title="Delete"
        className="shrink-0 opacity-0 group-hover:opacity-100 p-1 rounded hover:text-destructive transition-all"
      >
        <Trash2 className="h-3.5 w-3.5" />
      </button>
    </div>
  )
}
