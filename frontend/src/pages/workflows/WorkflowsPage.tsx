import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Play, ChevronDown, ChevronUp } from 'lucide-react'
import {
  workflowService,
  emptyDefinition,
  TRIGGER_TYPES,
  type Workflow,
  type CreateWorkflowPayload,
} from '@/services/workflow.service'
import WorkflowBuilder from './WorkflowBuilder'
import RunHistory from './RunHistory'

const STATUS_BADGE: Record<string, string> = {
  DRAFT:    'bg-gray-100  text-gray-700',
  ACTIVE:   'bg-green-100 text-green-700',
  PAUSED:   'bg-yellow-100 text-yellow-700',
  ARCHIVED: 'bg-red-100   text-red-700',
}

export default function WorkflowsPage() {
  const qc = useQueryClient()

  // ── List ──────────────────────────────────────────────────────────────────
  const { data: workflows = [], isLoading } = useQuery({
    queryKey: ['workflows'],
    queryFn: () => workflowService.list(),
    select: (res) => (res.data?.data ?? []) as Workflow[],
  })

  // ── Create form state ─────────────────────────────────────────────────────
  const [showCreate, setShowCreate]   = useState(false)
  const [form, setForm] = useState<CreateWorkflowPayload>({
    name: '',
    description: '',
    definition: emptyDefinition(),
    trigger_type: 'MANUAL',
    cron_expr: '',
  })

  const createMutation = useMutation({
    mutationFn: () => workflowService.create(form),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['workflows'] })
      setShowCreate(false)
      setForm({ name: '', description: '', definition: emptyDefinition(), trigger_type: 'MANUAL', cron_expr: '' })
    },
  })

  // ── Run mutation ──────────────────────────────────────────────────────────
  const runMutation = useMutation({
    mutationFn: (id: string) => workflowService.triggerRun(id),
    onSuccess: (_, id) => {
      qc.invalidateQueries({ queryKey: ['workflow-runs', id] })
    },
  })

  // ── Delete ────────────────────────────────────────────────────────────────
  const deleteMutation = useMutation({
    mutationFn: (id: string) => workflowService.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['workflows'] }),
  })

  // ── Expand state for run history ──────────────────────────────────────────
  const [expandedId, setExpandedId] = useState<string | null>(null)

  return (
    <div className="space-y-6 max-w-4xl">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Workflows</h1>
          <p className="text-muted-foreground text-sm mt-1">
            Automate multi-step tasks with AI agents, tools, conditions, and scheduling
          </p>
        </div>
        {!showCreate && (
          <button
            onClick={() => setShowCreate(true)}
            className="px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90"
          >
            + New workflow
          </button>
        )}
      </div>

      {/* Create form */}
      {showCreate && (
        <div className="rounded-lg border bg-card p-5 space-y-5">
          <h2 className="font-semibold">New workflow</h2>

          <div className="flex gap-3">
            <input
              placeholder="Workflow name *"
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              className="flex-1 rounded-md border bg-background px-3 py-2 text-sm"
            />
            <select
              value={form.trigger_type}
              onChange={(e) => setForm((f) => ({ ...f, trigger_type: e.target.value }))}
              className="rounded-md border bg-background px-3 py-2 text-sm"
            >
              {TRIGGER_TYPES.map((t) => (
                <option key={t.value} value={t.value}>{t.label}</option>
              ))}
            </select>
            {form.trigger_type === 'SCHEDULE' && (
              <input
                placeholder="Cron expr e.g. 0 9 * * 1"
                value={form.cron_expr}
                onChange={(e) => setForm((f) => ({ ...f, cron_expr: e.target.value }))}
                className="rounded-md border bg-background px-3 py-2 text-sm font-mono w-44"
              />
            )}
          </div>

          <input
            placeholder="Description (optional)"
            value={form.description}
            onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
            className="w-full rounded-md border bg-background px-3 py-2 text-sm"
          />

          <WorkflowBuilder
            value={form.definition}
            onChange={(def) => setForm((f) => ({ ...f, definition: def }))}
          />

          <div className="flex gap-2 pt-1">
            <button
              disabled={!form.name.trim() || createMutation.isPending}
              onClick={() => createMutation.mutate()}
              className="px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium disabled:opacity-50"
            >
              {createMutation.isPending ? 'Creating…' : 'Create workflow'}
            </button>
            <button
              onClick={() => setShowCreate(false)}
              className="px-4 py-2 rounded-lg border text-sm"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* Loading */}
      {isLoading && <p className="text-sm text-muted-foreground">Loading workflows…</p>}

      {/* Empty state */}
      {!isLoading && workflows.length === 0 && !showCreate && (
        <div className="text-center py-20 text-muted-foreground">
          <p className="text-4xl mb-3">⚙️</p>
          <p className="font-medium">No workflows yet</p>
          <p className="text-sm mt-1">Build automated pipelines with AI agents and tools.</p>
        </div>
      )}

      {/* Workflow list */}
      <div className="space-y-3">
        {workflows.map((wf) => (
          <div key={wf.id} className="rounded-lg border bg-card overflow-hidden">
            {/* Header row */}
            <div className="flex items-center gap-3 p-4">
              <span className={`text-xs font-medium rounded-full px-2.5 py-0.5 shrink-0 ${STATUS_BADGE[wf.status]}`}>
                {wf.status}
              </span>
              <div className="flex-1 min-w-0">
                <p className="font-semibold truncate">{wf.name}</p>
                {wf.description && (
                  <p className="text-xs text-muted-foreground truncate">{wf.description}</p>
                )}
              </div>
              <span className="text-xs text-muted-foreground shrink-0">
                {wf.definition.nodes.length} step{wf.definition.nodes.length !== 1 ? 's' : ''}
                {wf.trigger_type && wf.trigger_type !== 'MANUAL' && ` · ${wf.trigger_type}`}
              </span>
              <button
                onClick={() => runMutation.mutate(wf.id)}
                disabled={runMutation.isPending}
                title="Trigger run"
                className="shrink-0 p-1.5 rounded-md bg-primary/10 text-primary hover:bg-primary/20 transition-colors disabled:opacity-40"
              >
                <Play className="h-4 w-4" />
              </button>
              <button
                onClick={() => deleteMutation.mutate(wf.id)}
                className="shrink-0 text-xs text-muted-foreground hover:text-destructive border rounded px-2 py-1 transition-colors"
              >
                Delete
              </button>
              <button
                onClick={() => setExpandedId(expandedId === wf.id ? null : wf.id)}
                className="shrink-0 text-muted-foreground hover:text-foreground"
              >
                {expandedId === wf.id ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
              </button>
            </div>

            {/* Run history panel */}
            {expandedId === wf.id && (
              <div className="border-t bg-muted/30 p-4">
                <p className="text-xs font-semibold text-muted-foreground mb-3 uppercase tracking-wide">Run history</p>
                <RunHistory workflowId={wf.id} />
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
