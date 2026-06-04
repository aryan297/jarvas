import { useState } from 'react'
import { documentService, type RAGSearchResult } from '@/services/document.service'
import { Search, Loader2, FileText } from 'lucide-react'

export default function DocumentSearch() {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<RAGSearchResult[]>([])
  const [loading, setLoading] = useState(false)
  const [searched, setSearched] = useState(false)

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!query.trim()) return
    setLoading(true)
    setSearched(true)
    try {
      const res = await documentService.search(query.trim())
      setResults(res.data.data?.chunks ?? [])
    } catch {
      setResults([])
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-5 max-w-3xl">
      {/* Search input */}
      <form onSubmit={handleSearch} className="flex gap-2">
        <div className="flex-1 relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search across all your documents…"
            className="w-full pl-9 pr-4 py-2 border border-border rounded-lg bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary"
          />
        </div>
        <button
          type="submit"
          disabled={loading || !query.trim()}
          className="px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium disabled:opacity-50 hover:bg-primary/90 transition-colors"
        >
          {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Search'}
        </button>
      </form>

      {/* Results */}
      {loading && (
        <div className="flex justify-center py-8">
          <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
        </div>
      )}

      {!loading && searched && results.length === 0 && (
        <div className="text-center py-12 text-muted-foreground">
          <p className="text-3xl mb-2">🔍</p>
          <p className="font-medium">No results found</p>
          <p className="text-sm">Try a different query, or upload more documents</p>
        </div>
      )}

      {!loading && results.length > 0 && (
        <div className="space-y-3">
          <p className="text-sm text-muted-foreground">{results.length} result{results.length > 1 ? 's' : ''} found</p>
          {results.map((r, i) => (
            <SearchResultCard key={i} result={r} rank={i + 1} />
          ))}
        </div>
      )}

      {!searched && !loading && (
        <div className="text-center py-12 text-muted-foreground">
          <p className="text-4xl mb-3">🧠</p>
          <p className="font-medium">Semantic search</p>
          <p className="text-sm">Search by meaning, not just keywords — across all your indexed documents</p>
        </div>
      )}
    </div>
  )
}

function SearchResultCard({ result, rank }: { result: RAGSearchResult; rank: number }) {
  return (
    <div className="bg-card border border-border rounded-xl p-4 space-y-2">
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2 min-w-0">
          <span className="shrink-0 text-xs font-bold text-muted-foreground w-5">#{rank}</span>
          <FileText className="shrink-0 h-4 w-4 text-muted-foreground" />
          <span className="text-sm font-medium truncate">{result.doc_name}</span>
        </div>
        <span className="shrink-0 text-xs text-muted-foreground bg-muted px-2 py-0.5 rounded-full">
          {(result.score * 100).toFixed(0)}% match
        </span>
      </div>
      <p className="text-sm text-foreground/80 leading-relaxed line-clamp-4">{result.content}</p>
    </div>
  )
}
