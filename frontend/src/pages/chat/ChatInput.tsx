import { useRef, useState } from 'react'
import { Send, Loader2, Zap } from 'lucide-react'
import { clsx } from 'clsx'

interface Props {
  onSend: (content: string, stream: boolean) => void
  disabled?: boolean
}

export default function ChatInput({ onSend, disabled }: Props) {
  const [value, setValue] = useState('')
  const [streaming, setStreaming] = useState(true)
  const ref = useRef<HTMLTextAreaElement>(null)

  const submit = () => {
    const trimmed = value.trim()
    if (!trimmed || disabled) return
    onSend(trimmed, streaming)
    setValue('')
    if (ref.current) ref.current.style.height = 'auto'
  }

  const onKeyDown = (e: React.KeyboardEvent) => {
    // Enter = send; Shift+Enter = newline
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      submit()
    }
  }

  const onInput = () => {
    const el = ref.current
    if (!el) return
    el.style.height = 'auto'
    el.style.height = Math.min(el.scrollHeight, 160) + 'px'
  }

  return (
    <div className="border-t border-border bg-card p-3">
      <div className="flex items-end gap-2 bg-background border border-border rounded-xl px-3 py-2">
        <textarea
          ref={ref}
          rows={1}
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onInput={onInput}
          onKeyDown={onKeyDown}
          disabled={disabled}
          placeholder="Message Jarvas… (Enter to send, Shift+Enter for newline)"
          className="flex-1 resize-none bg-transparent text-sm focus:outline-none placeholder:text-muted-foreground disabled:opacity-50 max-h-40"
        />

        {/* Stream toggle */}
        <button
          type="button"
          onClick={() => setStreaming((s) => !s)}
          title={streaming ? 'Streaming on' : 'Streaming off'}
          className={clsx(
            'shrink-0 p-1.5 rounded-md transition-colors text-xs',
            streaming
              ? 'text-primary bg-primary/10'
              : 'text-muted-foreground hover:bg-muted',
          )}
        >
          <Zap className="h-4 w-4" />
        </button>

        {/* Send button */}
        <button
          type="button"
          onClick={submit}
          disabled={!value.trim() || disabled}
          className="shrink-0 p-1.5 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {disabled ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Send className="h-4 w-4" />
          )}
        </button>
      </div>

      <p className="text-center text-xs text-muted-foreground mt-1.5">
        Jarvas may make mistakes. Verify important information.
      </p>
    </div>
  )
}
