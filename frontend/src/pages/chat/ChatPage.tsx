import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { chatService } from '@/services/chat.service'
import ConversationList from './ConversationList'
import MessageThread from './MessageThread'

export default function ChatPage() {
  const { id } = useParams<{ id?: string }>()
  const navigate = useNavigate()
  const [activeId, setActiveId] = useState<string | undefined>(id)

  const handleSelect = (convId: string) => {
    setActiveId(convId)
    navigate(`/chat/${convId}`, { replace: true })
  }

  return (
    <div className="flex h-full -m-6 overflow-hidden">
      {/* Left panel: conversation list */}
      <div className="w-72 shrink-0 border-r border-border bg-card flex flex-col">
        <ConversationList activeId={activeId} onSelect={handleSelect} />
      </div>

      {/* Right panel: messages */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {activeId ? (
          <MessageThread convId={activeId} />
        ) : (
          <EmptyState onSelect={handleSelect} />
        )}
      </div>
    </div>
  )
}

function EmptyState({ onSelect }: { onSelect: (id: string) => void }) {
  const { mutateAsync, isPending } = useMutation({
    mutationFn: () => chatService.createConversation({}),
  })

  const handleNew = async () => {
    const res = await mutateAsync()
    const conv = res.data.data
    if (conv) onSelect(conv.id)
  }

  return (
    <div className="flex-1 flex flex-col items-center justify-center gap-4 text-center p-8">
      <div className="text-5xl">💬</div>
      <h2 className="text-xl font-semibold">Start a conversation</h2>
      <p className="text-muted-foreground text-sm max-w-xs">
        Ask anything. Jarvas has memory, can search your documents, and gets smarter over time.
      </p>
      <button
        onClick={handleNew}
        disabled={isPending}
        className="px-5 py-2 bg-primary text-primary-foreground rounded-md text-sm font-medium hover:bg-primary/90 transition-colors disabled:opacity-50"
      >
        {isPending ? 'Creating…' : 'New conversation'}
      </button>
    </div>
  )
}
