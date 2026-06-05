import { useState, useRef, useCallback } from 'react'
import { Mic, Square, Loader2 } from 'lucide-react'
import { clsx } from 'clsx'
import { voiceService, type VoiceSession } from '@/services/voice.service'

type RecorderState = 'idle' | 'recording' | 'uploading' | 'transcribing'

interface Props {
  conversationId: string
  onTranscript: (text: string) => void
  disabled?: boolean
}

export default function VoiceRecorder({ conversationId, onTranscript, disabled }: Props) {
  const [state, setState] = useState<RecorderState>('idle')
  const [error, setError] = useState<string | null>(null)
  const [statusText, setStatusText] = useState('')

  const mediaRecorder = useRef<MediaRecorder | null>(null)
  const chunks = useRef<Blob[]>([])
  const stream = useRef<MediaStream | null>(null)

  const startRecording = useCallback(async () => {
    setError(null)
    try {
      const s = await navigator.mediaDevices.getUserMedia({ audio: true })
      stream.current = s
      chunks.current = []

      const recorder = new MediaRecorder(s, { mimeType: 'audio/webm' })
      mediaRecorder.current = recorder

      recorder.ondataavailable = (e) => {
        if (e.data.size > 0) chunks.current.push(e.data)
      }

      recorder.onstop = async () => {
        s.getTracks().forEach((t) => t.stop())
        const blob = new Blob(chunks.current, { type: 'audio/webm' })
        await handleUpload(blob)
      }

      recorder.start()
      setState('recording')
      setStatusText('Recording…')
    } catch (err) {
      setError('Microphone access denied')
    }
  }, [conversationId])

  const stopRecording = useCallback(() => {
    if (mediaRecorder.current?.state === 'recording') {
      mediaRecorder.current.stop()
      setState('uploading')
      setStatusText('Uploading…')
    }
  }, [])

  const handleUpload = async (blob: Blob) => {
    try {
      const { session_id } = await voiceService.upload(blob, conversationId)

      setState('transcribing')
      setStatusText('Transcribing…')

      const session = await voiceService.pollUntilDone(
        session_id,
        (s: VoiceSession) => {
          if (s.status === 'PROCESSING') setStatusText('Transcribing…')
        },
      )

      if (session.status === 'COMPLETED' && session.transcript) {
        onTranscript(session.transcript)
      } else {
        setError('Transcription failed. Please try again.')
      }
    } catch (err) {
      setError('Upload or transcription failed.')
    } finally {
      setState('idle')
      setStatusText('')
    }
  }

  const isRecording = state === 'recording'
  const isBusy = state === 'uploading' || state === 'transcribing'

  const handleClick = () => {
    if (disabled || isBusy) return
    if (isRecording) stopRecording()
    else startRecording()
  }

  return (
    <div className="relative flex items-center">
      <button
        type="button"
        onClick={handleClick}
        disabled={disabled || isBusy}
        title={isRecording ? 'Stop recording' : 'Record voice message'}
        className={clsx(
          'shrink-0 p-1.5 rounded-md transition-colors',
          isRecording
            ? 'bg-red-500 text-white animate-pulse'
            : isBusy
            ? 'text-muted-foreground opacity-50 cursor-not-allowed'
            : 'text-muted-foreground hover:bg-muted hover:text-foreground',
        )}
      >
        {isBusy ? (
          <Loader2 className="h-4 w-4 animate-spin" />
        ) : isRecording ? (
          <Square className="h-4 w-4" />
        ) : (
          <Mic className="h-4 w-4" />
        )}
      </button>

      {/* Status tooltip */}
      {(statusText || error) && (
        <span className={clsx(
          'absolute bottom-full left-1/2 -translate-x-1/2 mb-1 whitespace-nowrap rounded px-2 py-0.5 text-xs',
          error ? 'bg-destructive text-destructive-foreground' : 'bg-muted text-muted-foreground',
        )}>
          {error ?? statusText}
        </span>
      )}
    </div>
  )
}
