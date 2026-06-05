import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { memoryService } from '@/services/memory.service'
import MemoryCard from './MemoryCard'
import type { Memory } from '@/types/api'

const MEMORY_TYPES = ['FACT', 'PREFERENCE', 'EVENT', 'SKILL', 'RELATIONSHIP'] as const

export default function MemoryPage() {
  const qc = useQueryClient()

  // ── List ──────────────────────────────────────────────────────────────────
  const { data, isLoading } = useQuery({
    queryKey: ['memories'],
    queryFn: () => memoryService.list(),
    select: (res) => (res.data?.data ?? []) as Memory[],
  })

  // ── Create ────────────────────────────────────────────────────────────────
  const [form, setForm] = useState({ type: 'FACT', content: '', importance: 0.5 })
  const [showForm, setShowForm] = useState(false)

  const createMutation = useMutation({
    mutationFn: () => memoryService.create(form.type, form.content, form.importance),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['memories'] })
      setForm({ type: 'FACT', content: '', importance: 0.5 })
      setShowForm(false)
    },
  })

  // ── Delete ────────────────────────────────────────────────────────────────
  const deleteMutation = useMutation({
    mutationFn: (id: string) => memoryService.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['memories'] }),
  })

  // ── Search ────────────────────────────────────────────────────────────────
  const [searchQuery, setSearchQuery] = useState('')
  const [activeTab, setActiveTab] = useState<'list' | 'search'>('list')

  const searchMutation = useMutation({
    mutationFn: () => memoryService.search(searchQuery, 10, 0.3),
  })

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    if (searchQuery.trim()) searchMutation.mutate()
  }

  const memories = data ?? []

  return (
    <div className="space-y-6 max-w-4xl">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Memory</h1>
          <p className="text-muted-foreground text-sm mt-1">
            Long-term facts Jarvas remembers about you
          </p>
        </div>
        <button
          onClick={() => setShowForm(!showForm)}
          className="px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors"
        >
          + Add memory
        </button>
      </div>

      {/* Create form */}
      {showForm && (
        <div className="rounded-lg border bg-card p-4 space-y-4">
          <h2 className="font-semibold text-sm">New memory</h2>
          <div className="flex gap-3">
            <select
              value={form.type}
              onChange={(e) => setForm((f) => ({ ...f, type: e.target.value }))}
              className="rounded-md border bg-background px-3 py-2 text-sm"
            >
              {MEMORY_TYPES.map((t) => (
                <option key={t} value={t}>{t}</option>
              ))}
            </select>
            <input
              type="number"
              min={0} max={1} step={0.1}
              value={form.importance}
              onChange={(e) => setForm((f) => ({ ...f, importance: parseFloat(e.target.value) }))}
              className="rounded-md border bg-background px-3 py-2 text-sm w-24"
              placeholder="0.5"
            />
          </div>
          <textarea
            rows={3}
            placeholder="e.g. User prefers Python over Java"
            value={form.content}
            onChange={(e) => setForm((f) => ({ ...f, content: e.target.value }))}
            className="w-full rounded-md border bg-background px-3 py-2 text-sm resize-none"
          />
          <div className="flex gap-2">
            <button
              disabled={!form.content.trim() || createMutation.isPending}
              onClick={() => createMutation.mutate()}
              className="px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium disabled:opacity-50"
            >
              {createMutation.isPending ? 'Saving...' : 'Save'}
            </button>
            <button
              onClick={() => setShowForm(false)}
              className="px-4 py-2 rounded-lg border text-sm"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* Tabs */}
      <div className="flex gap-1 border-b">
        {(['list', 'search'] as const).map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 text-sm font-medium capitalize transition-colors ${
              activeTab === tab
                ? 'border-b-2 border-primary text-primary'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            {tab === 'list' ? `My memories (${memories.length})` : 'Search'}
          </button>
        ))}
      </div>

      {/* List tab */}
      {activeTab === 'list' && (
        <>
          {isLoading && (
            <p className="text-sm text-muted-foreground">Loading...</p>
          )}
          {!isLoading && memories.length === 0 && (
            <div className="text-center py-16 text-muted-foreground">
              <p className="text-4xl mb-3">🧠</p>
              <p className="font-medium">No memories yet</p>
              <p className="text-sm mt-1">Chat with Jarvas or add memories manually.</p>
            </div>
          )}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {memories.map((m) => (
              <MemoryCard
                key={m.id}
                memory={m}
                onDelete={(id) => deleteMutation.mutate(id)}
              />
            ))}
          </div>
        </>
      )}

      {/* Search tab */}
      {activeTab === 'search' && (
        <div className="space-y-4">
          <form onSubmit={handleSearch} className="flex gap-2">
            <input
              type="text"
              placeholder="Search your memories semantically..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="flex-1 rounded-md border bg-background px-3 py-2 text-sm"
            />
            <button
              type="submit"
              disabled={!searchQuery.trim() || searchMutation.isPending}
              className="px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium disabled:opacity-50"
            >
              {searchMutation.isPending ? 'Searching...' : 'Search'}
            </button>
          </form>

          {searchMutation.data && (
            <div className="space-y-2">
              {(searchMutation.data.data?.data ?? []).length === 0 && (
                <p className="text-sm text-muted-foreground">No matching memories found.</p>
              )}
              {(searchMutation.data.data?.data ?? []).map((r) => (
                <div key={r.id} className="rounded-lg border bg-card p-3 flex items-start gap-3">
                  <span className="text-xs font-medium bg-muted px-2 py-0.5 rounded-full whitespace-nowrap">
                    {r.type}
                  </span>
                  <p className="text-sm flex-1">{r.content}</p>
                  <span className="text-xs text-muted-foreground whitespace-nowrap">
                    {Math.round(r.score * 100)}% match
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
