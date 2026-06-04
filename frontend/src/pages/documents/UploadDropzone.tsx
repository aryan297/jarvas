import { useCallback } from 'react'
import { useDropzone } from 'react-dropzone'
import { Upload, Loader2 } from 'lucide-react'
import { clsx } from 'clsx'

const ACCEPTED = {
  'application/pdf':  ['.pdf'],
  'text/plain':       ['.txt'],
  'text/markdown':    ['.md'],
  'text/csv':         ['.csv'],
  'text/html':        ['.html', '.htm'],
}

interface Props {
  onUpload: (files: File[]) => void
  loading?: boolean
}

export default function UploadDropzone({ onUpload, loading }: Props) {
  const onDrop = useCallback(
    (accepted: File[]) => {
      if (accepted.length > 0 && !loading) onUpload(accepted)
    },
    [onUpload, loading],
  )

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: ACCEPTED,
    disabled: loading,
    maxSize: 50 * 1024 * 1024, // 50 MB
  })

  return (
    <div
      {...getRootProps()}
      className={clsx(
        'border-2 border-dashed rounded-xl p-8 text-center cursor-pointer transition-colors',
        isDragActive
          ? 'border-primary bg-primary/5'
          : 'border-border hover:border-primary/50 hover:bg-muted/40',
        loading && 'opacity-60 cursor-not-allowed',
      )}
    >
      <input {...getInputProps()} />
      <div className="flex flex-col items-center gap-2">
        {loading ? (
          <Loader2 className="h-8 w-8 text-muted-foreground animate-spin" />
        ) : (
          <Upload className="h-8 w-8 text-muted-foreground" />
        )}
        <p className="font-medium text-sm">
          {loading
            ? 'Uploading…'
            : isDragActive
            ? 'Drop files here'
            : 'Drag & drop files, or click to browse'}
        </p>
        <p className="text-xs text-muted-foreground">
          PDF, TXT, MD, CSV, HTML — up to 50 MB each
        </p>
      </div>
    </div>
  )
}
