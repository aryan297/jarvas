import { useQuery } from '@tanstack/react-query'
import { workflowService, type WorkflowRun } from '@/services/workflow.service'

const STATUS_STYLE: Record<string, string> = {
  PENDING:   'bg-gray-100  text-gray-700',
  RUNNING:   'bg-blue-100  text-blue-700 animate-pulse',
  COMPLETED: 'bg-green-100 text-green-700',
  FAILED:    'bg-red-100   text-red-700',
  CANCELLED: 'bg-yellow-100 text-yellow-700',
}

interface Props {
  workflowId: string
}

export default function RunHistory({ workflowId }: Props) {
  const { data, isLoading } = useQuery({
    queryKey: ['workflow-runs', workflowId],
    queryFn: () => workflowService.listRuns(workflowId),
    select: (res) => (res.data?.data ?? []) as WorkflowRun[],
    refetchInterval: (query) => {
      const runs = query.state.data
      const hasActive = runs?.some((r) => r.status === 'PENDING' || r.status === 'RUNNING')
      return hasActive ? 3000 : false
    },
  })

  if (isLoading) return <p className="text-sm text-muted-foreground">Loading runs…</p>

  const runs = data ?? []
  if (runs.length === 0) {
    return <p className="text-sm text-muted-foreground italic">No runs yet. Trigger the workflow to see history here.</p>
  }

  return (
    <div className="space-y-2">
      {runs.map((run) => (
        <div key={run.id} className="rounded-lg border bg-card p-3 flex items-start gap-3 text-sm">
          <span className={`shrink-0 rounded-full px-2.5 py-0.5 text-xs font-medium ${STATUS_STYLE[run.status] ?? STATUS_STYLE.PENDING}`}>
            {run.status}
          </span>
          <div className="flex-1 min-w-0">
            <p className="text-xs text-muted-foreground font-mono truncate">{run.id}</p>
            {run.error_msg && (
              <p className="text-red-600 text-xs mt-0.5 truncate">{run.error_msg}</p>
            )}
            {run.completed_at && run.started_at && (
              <p className="text-muted-foreground text-xs mt-0.5">
                Duration: {Math.round((new Date(run.completed_at).getTime() - new Date(run.started_at).getTime()) / 1000)}s
              </p>
            )}
          </div>
          <p className="shrink-0 text-xs text-muted-foreground">
            {new Date(run.created_at).toLocaleString()}
          </p>
        </div>
      ))}
    </div>
  )
}
