import { FileText, Trash2, ExternalLink, Loader2, CheckCircle, XCircle } from 'lucide-react'
import { clsx } from 'clsx'
import { formatDistanceToNow } from 'date-fns'
import { documentService } from '@/services/document.service'
import type { Document } from '@/types/api'

const STATUS_CONFIG = {
  UPLOADED:   { label: 'Queued',    color: 'text-muted-foreground', Icon: Loader2, spin: false },
  PROCESSING: { label: 'Indexing…', color: 'text-blue-500',         Icon: Loader2, spin: true  },
  INDEXED:    { label: 'Ready',     color: 'text-green-600',        Icon: CheckCircle, spin: false },
  FAILED:     { label: 'Failed',    color: 'text-destructive',      Icon: XCircle, spin: false },
} as const

const SIZE_LABELS = ['B', 'KB', 'MB', 'GB']
function formatSize(bytes: number) {
  let n = bytes
  let i = 0
  while (n >= 1024 && i < SIZE_LABELS.length - 1) { n /= 1024; i++ }
  return `${n.toFixed(1)} ${SIZE_LABELS[i]}`
}

interface Props {
  doc: Document
  onDelete: () => void
}

export default function DocumentCard({ doc, onDelete }: Props) {
  const cfg = STATUS_CONFIG[doc.status as keyof typeof STATUS_CONFIG] ?? STATUS_CONFIG.UPLOADED
  const { Icon } = cfg
  const ago = formatDistanceToNow(new Date(doc.created_at), { addSuffix: true })

  const handleDownload = async () => {
    const res = await documentService.getDownloadURL(doc.id)
    const url = res.data.data?.url
    if (url) window.open(url, '_blank')
  }

  return (
    <div className="bg-card border border-border rounded-xl p-4 flex flex-col gap-3 hover:shadow-sm transition-shadow">
      {/* Icon + name */}
      <div className="flex items-start gap-3">
        <div className="p-2 bg-muted rounded-lg shrink-0">
          <FileText className="h-5 w-5 text-muted-foreground" />
        </div>
        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium truncate" title={doc.name}>{doc.name}</p>
          <p className="text-xs text-muted-foreground">{formatSize(doc.size_bytes)} · {ago}</p>
        </div>
      </div>

      {/* Status + chunk count */}
      <div className="flex items-center justify-between text-xs">
        <span className={clsx('flex items-center gap-1 font-medium', cfg.color)}>
          <Icon className={clsx('h-3.5 w-3.5', cfg.spin && 'animate-spin')} />
          {cfg.label}
        </span>
        {doc.status === 'INDEXED' && (
          <span className="text-muted-foreground">{doc.chunk_count} chunks</span>
        )}
      </div>

      {/* Actions */}
      <div className="flex gap-2 pt-1 border-t border-border">
        <button
          onClick={handleDownload}
          className="flex-1 flex items-center justify-center gap-1 py-1.5 text-xs text-muted-foreground hover:text-foreground hover:bg-muted rounded-md transition-colors"
        >
          <ExternalLink className="h-3.5 w-3.5" />
          Download
        </button>
        <button
          onClick={onDelete}
          className="flex-1 flex items-center justify-center gap-1 py-1.5 text-xs text-muted-foreground hover:text-destructive hover:bg-muted rounded-md transition-colors"
        >
          <Trash2 className="h-3.5 w-3.5" />
          Delete
        </button>
      </div>
    </div>
  )
}
