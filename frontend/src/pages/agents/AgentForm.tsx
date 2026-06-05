import { useState, useEffect } from 'react'
import type { Agent } from '@/types/api'
import { AGENT_TYPES, AVAILABLE_TOOLS, AVAILABLE_MODELS } from '@/services/agent.service'

interface FormState {
  name: string
  description: string
  type: string
  system_prompt: string
  model: string
  temperature: number
  max_tokens: number
  tools_enabled: string[]
  memory_enabled: boolean
  rag_enabled: boolean
}

const defaults: FormState = {
  name: '',
  description: '',
  type: 'CUSTOM',
  system_prompt: '',
  model: 'gpt-4o',
  temperature: 0.7,
  max_tokens: 4096,
  tools_enabled: [],
  memory_enabled: true,
  rag_enabled: false,
}

interface Props {
  agent?: Agent | null
  onSave: (data: FormState) => void
  onCancel: () => void
  isPending?: boolean
}

export default function AgentForm({ agent, onSave, onCancel, isPending }: Props) {
  const [form, setForm] = useState<FormState>(defaults)

  useEffect(() => {
    if (agent) {
      setForm({
        name:           agent.name,
        description:    agent.description ?? '',
        type:           agent.type,
        system_prompt:  '',
        model:          agent.model,
        temperature:    agent.temperature,
        max_tokens:     agent.max_tokens,
        tools_enabled:  agent.tools_enabled,
        memory_enabled: agent.memory_enabled,
        rag_enabled:    agent.rag_enabled,
      })
    } else {
      setForm(defaults)
    }
  }, [agent])

  const set = <K extends keyof FormState>(key: K, value: FormState[K]) =>
    setForm((f) => ({ ...f, [key]: value }))

  const toggleTool = (tool: string) => {
    set('tools_enabled',
      form.tools_enabled.includes(tool)
        ? form.tools_enabled.filter((t) => t !== tool)
        : [...form.tools_enabled, tool],
    )
  }

  return (
    <div className="rounded-lg border bg-card p-5 space-y-4">
      <h2 className="font-semibold">{agent ? 'Edit agent' : 'New agent'}</h2>

      {/* Name + Type row */}
      <div className="flex gap-3">
        <div className="flex-1 space-y-1">
          <label className="text-xs font-medium text-muted-foreground">Name *</label>
          <input
            value={form.name}
            onChange={(e) => set('name', e.target.value)}
            placeholder="My Research Agent"
            className="w-full rounded-md border bg-background px-3 py-2 text-sm"
          />
        </div>
        <div className="space-y-1">
          <label className="text-xs font-medium text-muted-foreground">Type</label>
          <select
            value={form.type}
            onChange={(e) => set('type', e.target.value)}
            className="rounded-md border bg-background px-3 py-2 text-sm"
          >
            {AGENT_TYPES.map((t) => (
              <option key={t.value} value={t.value}>{t.label}</option>
            ))}
          </select>
        </div>
      </div>

      {/* Description */}
      <div className="space-y-1">
        <label className="text-xs font-medium text-muted-foreground">Description</label>
        <input
          value={form.description}
          onChange={(e) => set('description', e.target.value)}
          placeholder="What does this agent do?"
          className="w-full rounded-md border bg-background px-3 py-2 text-sm"
        />
      </div>

      {/* System prompt */}
      <div className="space-y-1">
        <label className="text-xs font-medium text-muted-foreground">
          System prompt <span className="text-muted-foreground/60">(leave blank for default)</span>
        </label>
        <textarea
          rows={4}
          value={form.system_prompt}
          onChange={(e) => set('system_prompt', e.target.value)}
          placeholder="You are a helpful research assistant specialised in..."
          className="w-full rounded-md border bg-background px-3 py-2 text-sm resize-none"
        />
      </div>

      {/* Model + Temperature row */}
      <div className="flex gap-3">
        <div className="flex-1 space-y-1">
          <label className="text-xs font-medium text-muted-foreground">Model</label>
          <select
            value={form.model}
            onChange={(e) => set('model', e.target.value)}
            className="w-full rounded-md border bg-background px-3 py-2 text-sm"
          >
            {AVAILABLE_MODELS.map((m) => (
              <option key={m} value={m}>{m}</option>
            ))}
          </select>
        </div>
        <div className="space-y-1">
          <label className="text-xs font-medium text-muted-foreground">Temp</label>
          <input
            type="number" min={0} max={2} step={0.1}
            value={form.temperature}
            onChange={(e) => set('temperature', parseFloat(e.target.value))}
            className="w-20 rounded-md border bg-background px-3 py-2 text-sm"
          />
        </div>
        <div className="space-y-1">
          <label className="text-xs font-medium text-muted-foreground">Max tokens</label>
          <input
            type="number" min={256} max={128000} step={256}
            value={form.max_tokens}
            onChange={(e) => set('max_tokens', parseInt(e.target.value))}
            className="w-28 rounded-md border bg-background px-3 py-2 text-sm"
          />
        </div>
      </div>

      {/* Tools */}
      <div className="space-y-2">
        <label className="text-xs font-medium text-muted-foreground">Tools</label>
        <div className="flex flex-wrap gap-2">
          {AVAILABLE_TOOLS.map((tool) => {
            const active = form.tools_enabled.includes(tool.value)
            return (
              <button
                key={tool.value}
                type="button"
                onClick={() => toggleTool(tool.value)}
                title={tool.desc}
                className={`text-xs px-3 py-1.5 rounded-full border transition-colors ${
                  active
                    ? 'bg-primary text-primary-foreground border-primary'
                    : 'bg-background text-muted-foreground hover:border-primary/50'
                }`}
              >
                {tool.label}
              </button>
            )
          })}
        </div>
      </div>

      {/* Toggles */}
      <div className="flex gap-6">
        {[
          { key: 'memory_enabled' as const, label: 'Long-term memory' },
          { key: 'rag_enabled' as const, label: 'Document RAG' },
        ].map(({ key, label }) => (
          <label key={key} className="flex items-center gap-2 cursor-pointer text-sm">
            <input
              type="checkbox"
              checked={form[key] as boolean}
              onChange={(e) => set(key, e.target.checked)}
              className="h-4 w-4 rounded"
            />
            {label}
          </label>
        ))}
      </div>

      {/* Actions */}
      <div className="flex gap-2 pt-1">
        <button
          disabled={!form.name.trim() || isPending}
          onClick={() => onSave(form)}
          className="px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium disabled:opacity-50"
        >
          {isPending ? 'Saving...' : agent ? 'Save changes' : 'Create agent'}
        </button>
        <button
          onClick={onCancel}
          className="px-4 py-2 rounded-lg border text-sm"
        >
          Cancel
        </button>
      </div>
    </div>
  )
}
