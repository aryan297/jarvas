import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { documentService } from '@/services/document.service'
import UploadDropzone from './UploadDropzone'
import DocumentCard from './DocumentCard'
import DocumentSearch from './DocumentSearch'
import { Search, Upload } from 'lucide-react'
import { clsx } from 'clsx'
import toast from 'react-hot-toast'
import type { Document } from '@/types/api'

type Tab = 'documents' | 'search'

export default function DocumentsPage() {
  const [tab, setTab] = useState<Tab>('documents')
  const [uploading, setUploading] = useState(false)
  const qc = useQueryClient()

  const { data: docs = [], isLoading } = useQuery({
    queryKey: ['documents'],
    queryFn: async () => {
      const res = await documentService.list()
      return res.data.data ?? []
    },
  })

  const { mutate: remove } = useMutation({
    mutationFn: (id: string) => documentService.delete(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['documents'] })
      toast.success('Document deleted')
    },
    onError: () => toast.error('Failed to delete document'),
  })

  const handleUpload = async (files: File[]) => {
    setUploading(true)
    let ok = 0
    for (const f of files) {
      try {
        await documentService.upload(f)
        ok++
      } catch {
        toast.error(`Failed to upload ${f.name}`)
      }
    }
    if (ok > 0) {
      toast.success(`${ok} file${ok > 1 ? 's' : ''} uploaded — indexing in background`)
      qc.invalidateQueries({ queryKey: ['documents'] })
    }
    setUploading(false)
  }

  // Poll for status changes every 5s when any doc is processing
  const hasProcessing = docs.some((d: Document) => d.status === 'UPLOADED' || d.status === 'PROCESSING')
  useQuery({
    queryKey: ['documents-poll'],
    queryFn: () => documentService.list(),
    refetchInterval: hasProcessing ? 5000 : false,
    enabled: hasProcessing,
    select: (res) => {
      qc.setQueryData(['documents'], res.data.data ?? [])
      return res
    },
  })

  return (
    <div className="space-y-6 max-w-5xl">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Documents</h1>
          <p className="text-muted-foreground text-sm mt-0.5">
            Upload files · AI indexes them · Chat with your knowledge base
          </p>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 border-b border-border">
        {([
          { id: 'documents', label: 'My Documents', Icon: Upload },
          { id: 'search', label: 'Search Knowledge', Icon: Search },
        ] as const).map(({ id, label, Icon }) => (
          <button
            key={id}
            onClick={() => setTab(id)}
            className={clsx(
              'flex items-center gap-2 px-4 py-2 text-sm font-medium border-b-2 transition-colors -mb-px',
              tab === id
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground',
            )}
          >
            <Icon className="h-4 w-4" />
            {label}
          </button>
        ))}
      </div>

      {/* Documents tab */}
      {tab === 'documents' && (
        <div className="space-y-6">
          <UploadDropzone onUpload={handleUpload} loading={uploading} />

          {isLoading && (
            <div className="text-sm text-muted-foreground">Loading documents…</div>
          )}

          {!isLoading && docs.length === 0 && (
            <div className="text-center py-16 text-muted-foreground">
              <p className="text-4xl mb-3">📄</p>
              <p className="font-medium">No documents yet</p>
              <p className="text-sm">Drop a PDF or text file above to get started</p>
            </div>
          )}

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {docs.map((doc: Document) => (
              <DocumentCard key={doc.id} doc={doc} onDelete={() => remove(doc.id)} />
            ))}
          </div>
        </div>
      )}

      {/* Search tab */}
      {tab === 'search' && <DocumentSearch />}
    </div>
  )
}
