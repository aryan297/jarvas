import { useState } from 'react'
import { useParams, useNavigate, useSearchParams } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { chatService } from '@/services/chat.service'
import ConversationList from './ConversationList'
import MessageThread from './MessageThread'

export default function ChatPage() {
  const { id } = useParams<{ id?: string }>()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const [activeId, setActiveId] = useState<string | undefined>(id)

  // Agent pre-selection — set when navigating from /agents
  const agentId   = searchParams.get('agent_id')   ?? undefined
  const agentName = searchParams.get('agent_name')
    ? decodeURIComponent(searchParams.get('agent_name')!)
    : undefined

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
          <EmptyState
            agentId={agentId}
            agentName={agentName}
            onSelect={handleSelect}
          />
        )}
      </div>
    </div>
  )
}

function EmptyState({
  agentId,
  agentName,
  onSelect,
}: {
  agentId?: string
  agentName?: string
  onSelect: (id: string) => void
}) {
  const { mutateAsync, isPending } = useMutation({
    mutationFn: (aId?: string) =>
      chatService.createConversation({ agent_id: aId }),
  })

  const handleNew = async (useAgent?: boolean) => {
    const res = await mutateAsync(useAgent && agentId ? agentId : undefined)
    const conv = res.data.data
    if (conv) onSelect(conv.id)
  }

  return (
    <div className="flex-1 flex flex-col items-center justify-center gap-4 text-center p-8">
      <div className="text-5xl">💬</div>
      <h2 className="text-xl font-semibold">Start a conversation</h2>

      {agentId && agentName ? (
        <>
          <div className="rounded-lg border bg-card px-4 py-3 text-sm max-w-xs">
            <p className="text-muted-foreground">Agent selected:</p>
            <p className="font-semibold mt-0.5">{agentName}</p>
          </div>
          <div className="flex gap-2">
            <button
              onClick={() => handleNew(true)}
              disabled={isPending}
              className="px-5 py-2 bg-primary text-primary-foreground rounded-md text-sm font-medium hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              {isPending ? 'Creating…' : `Chat with ${agentName}`}
            </button>
            <button
              onClick={() => handleNew(false)}
              disabled={isPending}
              className="px-5 py-2 border rounded-md text-sm font-medium hover:bg-accent transition-colors disabled:opacity-50"
            >
              Default chat
            </button>
          </div>
        </>
      ) : (
        <>
          <p className="text-muted-foreground text-sm max-w-xs">
            Ask anything. Jarvas has memory, can search your documents, and gets smarter over time.
          </p>
          <button
            onClick={() => handleNew(false)}
            disabled={isPending}
            className="px-5 py-2 bg-primary text-primary-foreground rounded-md text-sm font-medium hover:bg-primary/90 transition-colors disabled:opacity-50"
          >
            {isPending ? 'Creating…' : 'New conversation'}
          </button>
        </>
      )}
    </div>
  )
}
