import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { clsx } from 'clsx'
import type { Message } from '@/types/api'

interface Props {
  message: Message | { id: string; role: string; content: string; created_at: string; streaming?: boolean }
}

export default function MessageBubble({ message }: Props) {
  const isUser = message.role === 'USER'
  const isStreaming = 'streaming' in message && message.streaming

  return (
    <div className={clsx('flex', isUser ? 'justify-end' : 'justify-start')}>
      <div
        className={clsx(
          'max-w-[75%] rounded-2xl px-4 py-3 text-sm',
          isUser
            ? 'bg-primary text-primary-foreground rounded-tr-sm'
            : 'bg-muted text-foreground rounded-tl-sm',
        )}
      >
        {isUser ? (
          <p className="whitespace-pre-wrap">{message.content}</p>
        ) : (
          <div className="prose prose-sm dark:prose-invert max-w-none">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>
              {message.content + (isStreaming ? '▍' : '')}
            </ReactMarkdown>
          </div>
        )}
      </div>
    </div>
  )
}
