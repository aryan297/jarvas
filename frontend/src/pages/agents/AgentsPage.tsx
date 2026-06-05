import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { agentService } from '@/services/agent.service'
import type { Agent } from '@/types/api'
import AgentCard from './AgentCard'
import AgentForm from './AgentForm'

export default function AgentsPage() {
  const qc = useQueryClient()
  const navigate = useNavigate()

  const [showForm, setShowForm]       = useState(false)
  const [editingAgent, setEditingAgent] = useState<Agent | null>(null)

  // ── List ──────────────────────────────────────────────────────────────────
  const { data, isLoading } = useQuery({
    queryKey: ['agents'],
    queryFn: () => agentService.list(),
    select: (res) => (res.data?.data ?? []) as Agent[],
  })

  // ── Create ────────────────────────────────────────────────────────────────
  const createMutation = useMutation({
    mutationFn: (payload: Parameters<typeof agentService.create>[0]) =>
      agentService.create(payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['agents'] })
      setShowForm(false)
    },
  })

  // ── Update ────────────────────────────────────────────────────────────────
  const updateMutation = useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: Parameters<typeof agentService.update>[1] }) =>
      agentService.update(id, payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['agents'] })
      setEditingAgent(null)
    },
  })

  // ── Delete ────────────────────────────────────────────────────────────────
  const deleteMutation = useMutation({
    mutationFn: (id: string) => agentService.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['agents'] }),
  })

  const handleSave = (form: Parameters<typeof createMutation.mutate>[0]) => {
    if (editingAgent) {
      updateMutation.mutate({ id: editingAgent.id, payload: form })
    } else {
      createMutation.mutate(form)
    }
  }

  const handleEdit = (agent: Agent) => {
    setEditingAgent(agent)
    setShowForm(true)
  }

  const handleChat = (agent: Agent) => {
    // Navigate to chat page with agent_id query param — ChatPage picks it up.
    navigate(`/chat?agent_id=${agent.id}&agent_name=${encodeURIComponent(agent.name)}`)
  }

  const agents = data ?? []

  return (
    <div className="space-y-6 max-w-5xl">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Agents</h1>
          <p className="text-muted-foreground text-sm mt-1">
            Custom AI agents with tools, memory, and RAG
          </p>
        </div>
        {!showForm && (
          <button
            onClick={() => { setEditingAgent(null); setShowForm(true) }}
            className="px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors"
          >
            + New agent
          </button>
        )}
      </div>

      {/* Form */}
      {showForm && (
        <AgentForm
          agent={editingAgent}
          onSave={handleSave}
          onCancel={() => { setShowForm(false); setEditingAgent(null) }}
          isPending={createMutation.isPending || updateMutation.isPending}
        />
      )}

      {/* Loading */}
      {isLoading && <p className="text-sm text-muted-foreground">Loading agents...</p>}

      {/* Empty state */}
      {!isLoading && agents.length === 0 && !showForm && (
        <div className="text-center py-20 text-muted-foreground">
          <p className="text-4xl mb-3">🤖</p>
          <p className="font-medium">No agents yet</p>
          <p className="text-sm mt-1">Create your first agent with custom tools and prompts.</p>
        </div>
      )}

      {/* Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {agents.map((agent) => (
          <AgentCard
            key={agent.id}
            agent={agent}
            onEdit={handleEdit}
            onDelete={(id) => deleteMutation.mutate(id)}
            onChat={handleChat}
          />
        ))}
      </div>
    </div>
  )
}
