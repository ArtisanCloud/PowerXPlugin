import type { HostIndexJob } from '../api/knowledgeHostTypes'

export type TaskPhase = 'ready' | 'reading' | 'submitting' | 'accepted' | 'queued' | 'running' | 'succeeded' | 'failed' | 'blocked' | 'unknown'
export type TaskProcessItem = { id: string; title: string; phase: TaskPhase; progress?: number; chunks?: number; durationMs?: number; durationKind?: 'request' | 'execution'; error?: string; traceId?: string; indexMode?: string }

export function taskPhase(status: string): TaskPhase {
 const phases: Record<string, TaskPhase> = { ready:'ready', reading:'reading', submitting:'submitting', accepted:'accepted', queued:'queued', pending:'queued', retrying:'queued', running:'running', processing:'running', succeeded:'succeeded', completed:'succeeded', failed:'failed', blocked:'blocked' }
 return phases[status] || 'unknown'
}
export function taskDurationMs(start?: string, finish?: string): number | undefined {
 if (!start || !finish) return undefined
 const value = Date.parse(finish) - Date.parse(start)
 return Number.isFinite(value) && value >= 0 ? value : undefined
}
export function hostTaskItem(job: HostIndexJob, title: string): TaskProcessItem {
 return { id:job.job_id, title, phase:taskPhase(job.status), chunks:job.chunk_count, durationMs:['succeeded','failed'].includes(job.status) ? taskDurationMs(job.started_at,job.finished_at) : undefined, durationKind:'execution', error:job.error_code, traceId:job.trace_id, indexMode:job.effective_config?.index_mode }
}
export function taskSummary(items: TaskProcessItem[]) {
 const counts = {total:items.length,ready:0,waiting:0,running:0,succeeded:0,failed:0,unknown:0}
 for (const item of items) {
  if (item.phase==='ready') counts.ready++
  else if (['accepted','queued','blocked'].includes(item.phase)) counts.waiting++
  else if (['reading','submitting','running'].includes(item.phase)) counts.running++
  else if (item.phase==='succeeded') counts.succeeded++
  else if (item.phase==='failed') counts.failed++
  else counts.unknown++
 }
 const ended = counts.succeeded + counts.failed
 return {...counts,ended,endedPercent:counts.total ? Math.round(100*ended/counts.total) : 0}
}
export function taskProgress(item: TaskProcessItem): number | null {
 if (item.phase==='succeeded') return 100
 if (['ready','accepted','queued','blocked'].includes(item.phase)) return 0
 if (typeof item.progress==='number' && Number.isFinite(item.progress) && item.progress>=0 && item.progress<=100) return item.progress
 return null
}
