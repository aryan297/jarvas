import { useEffect, useRef, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { chatService } from '@/services/chat.service'
import MessageBubble from './MessageBubble'
import ChatInput from './ChatInput'
import { Loader2 } from 'lucide-react'
import toast from 'react-hot-toast'
import type { Message } from '@/types/api'

interface StreamingMsg {
  id: string
  role: string
  content: string
  created_at: string
  streaming: boolean
}

interface Props {
  convId: string
}

export default function MessageThread({ convId }: Props) {
  const qc = useQueryClient()
  const bottomRef = useRef<HTMLDivElement>(null)
  const [sending, setSending] = useState(false)
  const [streamingMsg, setStreamingMsg] = useState<StreamingMsg | null>(null)

  const { data: messages = [], isLoading } = useQuery({
    queryKey: ['messages', convId],
    queryFn: async () => {
      const res = await chatService.listMessages(convId, 1, 100)
      return res.data.data ?? []
    },
    enabled: !!convId,
  })

  // Auto-scroll to bottom whenever messages change or streaming updates.
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, streamingMsg?.content])

  const handleSend = async (content: string, stream: boolean) => {
    if (sending) return
    setSending(true)

    // Optimistic user bubble
    const optimisticUser: Message = {
      id: `opt-${Date.now()}`,
      role: 'USER',
      content,
      created_at: new Date().toISOString(),
    }
    qc.setQueryData<Message[]>(['messages', convId], (prev = []) => [
      ...prev,
      optimisticUser,
    ])

    if (stream) {
      // Streaming path
      const placeholder: StreamingMsg = {
        id: `stream-${Date.now()}`,
        role: 'ASSISTANT',
        content: '',
        created_at: new Date().toISOString(),
        streaming: true,
      }
      setStreamingMsg(placeholder)

      await chatService.streamMessage(
        convId,
        content,
        (delta) => setStreamingMsg((prev) => prev ? { ...prev, content: prev.content + delta } : prev),
        () => {
          // Done — refresh messages from server
          setStreamingMsg(null)
          qc.invalidateQueries({ queryKey: ['messages', convId] })
          qc.invalidateQueries({ queryKey: ['conversations'] })
          setSending(false)
        },
        (err) => {
          setStreamingMsg(null)
          toast.error(err.message || 'Stream failed')
          setSending(false)
        },
      )
    } else {
      // Non-streaming path
      try {
        await chatService.sendMessage(convId, content)
        qc.invalidateQueries({ queryKey: ['messages', convId] })
        qc.invalidateQueries({ queryKey: ['conversations'] })
      } catch {
        toast.error('Failed to send message')
      } finally {
        setSending(false)
      }
    }
  }

  return (
    <div className="flex flex-col h-full">
      {/* Messages area */}
      <div className="flex-1 overflow-y-auto px-6 py-4 space-y-4">
        {isLoading && (
          <div className="flex justify-center py-8">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
        )}

        {!isLoading && messages.length === 0 && !streamingMsg && (
          <div className="flex flex-col items-center justify-center h-full text-center gap-2">
            <p className="text-muted-foreground text-sm">Send a message to start the conversation.</p>
          </div>
        )}

        {messages.map((msg) => (
          <MessageBubble key={msg.id} message={msg} />
        ))}

        {streamingMsg && <MessageBubble message={streamingMsg} />}

        <div ref={bottomRef} />
      </div>

      {/* Input */}
      <ChatInput onSend={handleSend} disabled={sending} conversationId={convId} />
    </div>
  )
}
