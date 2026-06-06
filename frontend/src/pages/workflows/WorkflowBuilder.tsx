import { useState } from 'react'
import { Plus, Trash2 } from 'lucide-react'
import type { WorkflowDefinition, WorkflowNode } from '@/services/workflow.service'
import { NODE_TYPES, TRIGGER_TYPES } from '@/services/workflow.service'

interface Props {
  value: WorkflowDefinition
  onChange: (def: WorkflowDefinition) => void
}

export default function WorkflowBuilder({ value, onChange }: Props) {
  const [jsonMode, setJsonMode] = useState(false)
  const [jsonText, setJsonText] = useState('')
  const [jsonError, setJsonError] = useState('')

  const setDef = (def: WorkflowDefinition) => onChange(def)

  // ── JSON editor mode ───────────────────────────────────────────────────────
  const enterJsonMode = () => {
    setJsonText(JSON.stringify(value, null, 2))
    setJsonError('')
    setJsonMode(true)
  }

  const applyJson = () => {
    try {
      const parsed = JSON.parse(jsonText)
      setDef(parsed)
      setJsonMode(false)
      setJsonError('')
    } catch (e) {
      setJsonError('Invalid JSON: ' + (e as Error).message)
    }
  }

  // ── Node helpers ───────────────────────────────────────────────────────────
  const addNode = () => {
    const id = `node_${Date.now()}`
    const node: WorkflowNode = { id, type: 'agent', config: { prompt: 'Describe what this step should do.' } }
    const newNodes = [...value.nodes, node]
    // Auto-connect: if there's a START→END edge, replace it with START→node→END
    let newEdges = value.edges.filter((e) => !(e.from === 'START' && e.to === 'END'))
    if (newNodes.length === 1) {
      newEdges = [{ from: 'START', to: id }, { from: id, to: 'END' }]
    } else {
      // Connect previous last node to this one
      const prev = newNodes[newNodes.length - 2]
      newEdges = newEdges.filter((e) => e.from !== prev.id || e.to !== 'END')
      newEdges.push({ from: prev.id, to: id }, { from: id, to: 'END' })
    }
    setDef({ ...value, nodes: newNodes, edges: newEdges })
  }

  const removeNode = (id: string) => {
    const newNodes = value.nodes.filter((n) => n.id !== id)
    const newEdges = value.edges.filter((e) => e.from !== id && e.to !== id)
    setDef({ ...value, nodes: newNodes, edges: newEdges })
  }

  const updateNode = (id: string, patch: Partial<WorkflowNode>) => {
    setDef({
      ...value,
      nodes: value.nodes.map((n) => n.id === id ? { ...n, ...patch } : n),
    })
  }

  const updateConfig = (nodeId: string, key: string, val: string) => {
    const node = value.nodes.find((n) => n.id === nodeId)!
    updateNode(nodeId, { config: { ...node.config, [key]: val } })
  }

  // ── Trigger ────────────────────────────────────────────────────────────────
  const updateTrigger = (patch: Partial<WorkflowDefinition['trigger']>) => {
    setDef({ ...value, trigger: { ...value.trigger, ...patch } })
  }

  if (jsonMode) {
    return (
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium">JSON Definition</span>
          <div className="flex gap-2">
            <button onClick={applyJson} className="text-xs px-3 py-1.5 bg-primary text-primary-foreground rounded-md">Apply</button>
            <button onClick={() => setJsonMode(false)} className="text-xs px-3 py-1.5 border rounded-md">Cancel</button>
          </div>
        </div>
        {jsonError && <p className="text-xs text-destructive">{jsonError}</p>}
        <textarea
          rows={20}
          value={jsonText}
          onChange={(e) => setJsonText(e.target.value)}
          className="w-full font-mono text-xs rounded-md border bg-background px-3 py-2 resize-y"
          spellCheck={false}
        />
      </div>
    )
  }

  return (
    <div className="space-y-5">
      {/* Trigger */}
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold">Trigger</h3>
          <button onClick={enterJsonMode} className="text-xs text-muted-foreground hover:text-foreground border rounded px-2 py-1">
            Edit JSON
          </button>
        </div>
        <div className="flex gap-3">
          <select
            value={value.trigger.type}
            onChange={(e) => updateTrigger({ type: e.target.value })}
            className="rounded-md border bg-background px-3 py-2 text-sm"
          >
            {TRIGGER_TYPES.map((t) => (
              <option key={t.value} value={t.value}>{t.label}</option>
            ))}
          </select>
          {value.trigger.type === 'SCHEDULE' && (
            <input
              placeholder="Cron e.g. 0 9 * * 1"
              value={value.trigger.cron_expr ?? ''}
              onChange={(e) => updateTrigger({ cron_expr: e.target.value })}
              className="flex-1 rounded-md border bg-background px-3 py-2 text-sm font-mono"
            />
          )}
        </div>
      </div>

      {/* Nodes */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold">Steps ({value.nodes.length})</h3>
          <button
            onClick={addNode}
            className="flex items-center gap-1 text-xs px-3 py-1.5 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
          >
            <Plus className="h-3 w-3" /> Add step
          </button>
        </div>

        {value.nodes.length === 0 && (
          <p className="text-sm text-muted-foreground italic">
            No steps yet. Click "Add step" to build your workflow.
          </p>
        )}

        {value.nodes.map((node, idx) => (
          <NodeCard
            key={node.id}
            node={node}
            index={idx}
            onUpdate={(patch) => updateNode(node.id, patch)}
            onConfigChange={(k, v) => updateConfig(node.id, k, v)}
            onRemove={() => removeNode(node.id)}
          />
        ))}
      </div>
    </div>
  )
}

// ── NodeCard ──────────────────────────────────────────────────────────────────

interface NodeCardProps {
  node: WorkflowNode
  index: number
  onUpdate: (patch: Partial<WorkflowNode>) => void
  onConfigChange: (key: string, val: string) => void
  onRemove: () => void
}

function NodeCard({ node, index, onUpdate, onConfigChange, onRemove }: NodeCardProps) {
  const typeInfo = NODE_TYPES.find((t) => t.value === node.type)

  return (
    <div className="rounded-lg border bg-card p-4 space-y-3">
      <div className="flex items-center gap-2">
        <span className="text-xs text-muted-foreground font-mono w-6 text-right">{index + 1}</span>
        <select
          value={node.type}
          onChange={(e) => onUpdate({ type: e.target.value as WorkflowNode['type'], config: defaultConfig(e.target.value) })}
          className="rounded-md border bg-background px-2 py-1.5 text-sm font-medium"
        >
          {NODE_TYPES.map((t) => (
            <option key={t.value} value={t.value}>{t.label}</option>
          ))}
        </select>
        <input
          placeholder="Node ID"
          value={node.id}
          readOnly
          className="flex-1 rounded-md border bg-muted px-2 py-1.5 text-xs font-mono text-muted-foreground"
        />
        <button onClick={onRemove} className="text-muted-foreground hover:text-destructive p-1">
          <Trash2 className="h-4 w-4" />
        </button>
      </div>

      {typeInfo && (
        <p className="text-xs text-muted-foreground">{typeInfo.desc}</p>
      )}

      <NodeConfigFields node={node} onConfigChange={onConfigChange} />
    </div>
  )
}

function NodeConfigFields({ node, onConfigChange }: { node: WorkflowNode; onConfigChange: (k: string, v: string) => void }) {
  const get = (k: string) => String(node.config[k] ?? '')

  switch (node.type) {
    case 'agent':
      return (
        <div className="space-y-2">
          <ConfigField label="Prompt" value={get('prompt')} onChange={(v) => onConfigChange('prompt', v)} multiline
            placeholder="You are a research assistant. Summarise: {node_1_output}" />
          <ConfigField label="Model (optional)" value={get('model')} onChange={(v) => onConfigChange('model', v)}
            placeholder="gpt-4o" />
        </div>
      )
    case 'tool':
      return (
        <div className="space-y-2">
          <ConfigField label="Tool name" value={get('tool')} onChange={(v) => onConfigChange('tool', v)}
            placeholder="web_search / calculator / http_request" />
          <ConfigField label="Query / args" value={get('query')} onChange={(v) => onConfigChange('query', v)}
            placeholder="Use {previous_node_id_output} to reference prior output" />
        </div>
      )
    case 'condition':
      return (
        <div className="space-y-2">
          <ConfigField label="If expression" value={get('if')} onChange={(v) => onConfigChange('if', v)}
            placeholder="{node_1_output}" />
          <div className="grid grid-cols-2 gap-2">
            <ConfigField label="Then (true path)" value={get('then')} onChange={(v) => onConfigChange('then', v)} placeholder="continue" />
            <ConfigField label="Else (false path)" value={get('else')} onChange={(v) => onConfigChange('else', v)} placeholder="skip" />
          </div>
        </div>
      )
    case 'delay':
      return (
        <ConfigField label="Delay (seconds)" value={get('seconds')} onChange={(v) => onConfigChange('seconds', v)} placeholder="5" />
      )
    default:
      return null
  }
}

function ConfigField({ label, value, onChange, placeholder, multiline }: {
  label: string; value: string; onChange: (v: string) => void; placeholder?: string; multiline?: boolean
}) {
  return (
    <div className="space-y-1">
      <label className="text-xs text-muted-foreground font-medium">{label}</label>
      {multiline ? (
        <textarea
          rows={3}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          className="w-full rounded-md border bg-background px-2 py-1.5 text-sm resize-none"
        />
      ) : (
        <input
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          className="w-full rounded-md border bg-background px-2 py-1.5 text-sm"
        />
      )}
    </div>
  )
}

function defaultConfig(type: string): Record<string, unknown> {
  switch (type) {
    case 'agent':     return { prompt: '' }
    case 'tool':      return { tool: '', query: '' }
    case 'condition': return { if: '', then: '', else: '' }
    case 'delay':     return { seconds: '5' }
    default:          return {}
  }
}
