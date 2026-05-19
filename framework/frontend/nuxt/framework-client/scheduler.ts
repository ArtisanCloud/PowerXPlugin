import type { PluginApi } from './api'

export const SCHEDULER_TRIGGERED_TOPIC = 'powerx.runtime.scheduler.triggered.v1'

export type SchedulerScheduleType = 'once' | 'interval' | 'cron'
export type SchedulerJobStatus = 'active' | 'paused' | string
export type SchedulerProviderMode = 'local' | 'host' | 'dual' | 'proxy' | 'gateway'

export interface SchedulerRetryPolicy {
  max_attempts?: number
  backoff_seconds?: number
}

export interface SchedulerJobSpec {
  job_id?: string
  tenant_uuid?: string
  provider_mode?: SchedulerProviderMode
  force_local?: boolean
  force_host?: boolean
  owner_type?: string
  owner_id?: string
  name: string
  schedule_type: SchedulerScheduleType
  schedule_expr: string
  timezone?: string
  topic?: string
  payload?: Record<string, any>
  idempotency_key?: string
  retry_policy?: SchedulerRetryPolicy
  paused?: boolean
}

export interface SchedulerJob extends SchedulerJobSpec {
  job_id: string
  status: SchedulerJobStatus
  created_at?: string
  updated_at?: string
  next_run_at?: string
  last_run_at?: string
}

export interface ListSchedulerJobsInput {
  tenant_uuid?: string
  provider_mode?: SchedulerProviderMode
  owner_type?: string
  owner_id?: string
  status?: string
}

export interface SchedulerActionOptions {
  tenant_uuid?: string
  provider_mode?: SchedulerProviderMode
  force_local?: boolean
  force_host?: boolean
}

export interface ListSchedulerJobsResult {
  items: SchedulerJob[]
  total: number
}

export interface SchedulerActionResult {
  job_id: string
  status: string
}

export interface SchedulerClient {
  listJobs(input?: ListSchedulerJobsInput): Promise<ListSchedulerJobsResult>
  getJob(jobId: string, tenantUuid?: string | SchedulerActionOptions): Promise<SchedulerJob>
  createJob(job: SchedulerJobSpec): Promise<SchedulerJob>
  updateJob(jobId: string, job: SchedulerJobSpec): Promise<SchedulerJob>
  pauseJob(jobId: string, tenantUuid?: string | SchedulerActionOptions): Promise<SchedulerActionResult>
  resumeJob(jobId: string, tenantUuid?: string | SchedulerActionOptions): Promise<SchedulerActionResult>
  triggerJob(jobId: string, tenantUuid?: string | SchedulerActionOptions): Promise<SchedulerActionResult>
}

type Envelope<T> = {
  success?: boolean
  data?: T
  message?: string
}

const unwrap = <T>(value: T | Envelope<T>): T => {
  if (value && typeof value === 'object' && 'data' in (value as any)) {
    return (value as Envelope<T>).data as T
  }
  return value as T
}

const query = (input?: ListSchedulerJobsInput) => {
  const params = new URLSearchParams()
  if (!input) return ''
  for (const [key, value] of Object.entries(input)) {
    if (value !== undefined && value !== null && String(value).trim() !== '') {
      params.set(key, String(value))
    }
  }
  const text = params.toString()
  return text ? `?${text}` : ''
}

const normalizeActionOptions = (input?: string | SchedulerActionOptions): SchedulerActionOptions => {
  if (!input) return {}
  if (typeof input === 'string') return { tenant_uuid: input }
  return input
}

const tenantInit = (input?: string | SchedulerActionOptions): RequestInit | undefined => {
  const options = normalizeActionOptions(input)
  if (!options.tenant_uuid && !options.provider_mode && !options.force_local && !options.force_host) return undefined
  const headers: Record<string, string> = {}
  if (options.tenant_uuid) headers.tenant_uuid = options.tenant_uuid
  if (options.provider_mode) headers['X-Scheduler-Provider-Mode'] = options.provider_mode
  return { headers }
}

const actionBody = (input?: string | SchedulerActionOptions) => {
  const options = normalizeActionOptions(input)
  const body: Record<string, any> = {}
  if (options.provider_mode) body.provider_mode = options.provider_mode
  if (options.force_local) body.force_local = true
  if (options.force_host) body.force_host = true
  return body
}

export function createSchedulerClient(api: PluginApi): SchedulerClient {
  return {
    async listJobs(input) {
      return unwrap(await api.get<ListSchedulerJobsResult | Envelope<ListSchedulerJobsResult>>(`/admin/runtime/scheduler/jobs${query(input)}`, tenantInit(input)))
    },
    async getJob(jobId, tenantUuid) {
      const options = normalizeActionOptions(tenantUuid)
      const params = new URLSearchParams()
      if (options.tenant_uuid) params.set('tenant_uuid', options.tenant_uuid)
      if (options.provider_mode) params.set('provider_mode', options.provider_mode)
      const suffix = params.toString() ? `?${params.toString()}` : ''
      return unwrap(await api.get<SchedulerJob | Envelope<SchedulerJob>>(`/admin/runtime/scheduler/jobs/${encodeURIComponent(jobId)}${suffix}`, tenantInit(options)))
    },
    async createJob(job) {
      return unwrap(await api.post<SchedulerJob | Envelope<SchedulerJob>>('/admin/runtime/scheduler/jobs', job, tenantInit(job)))
    },
    async updateJob(jobId, job) {
      return unwrap(await api.put<SchedulerJob | Envelope<SchedulerJob>>(`/admin/runtime/scheduler/jobs/${encodeURIComponent(jobId)}`, job, tenantInit(job)))
    },
    async pauseJob(jobId, tenantUuid) {
      return unwrap(await api.post<SchedulerActionResult | Envelope<SchedulerActionResult>>(`/admin/runtime/scheduler/jobs/${encodeURIComponent(jobId)}/pause`, actionBody(tenantUuid), tenantInit(tenantUuid)))
    },
    async resumeJob(jobId, tenantUuid) {
      return unwrap(await api.post<SchedulerActionResult | Envelope<SchedulerActionResult>>(`/admin/runtime/scheduler/jobs/${encodeURIComponent(jobId)}/resume`, actionBody(tenantUuid), tenantInit(tenantUuid)))
    },
    async triggerJob(jobId, tenantUuid) {
      return unwrap(await api.post<SchedulerActionResult | Envelope<SchedulerActionResult>>(`/admin/runtime/scheduler/jobs/${encodeURIComponent(jobId)}/trigger`, actionBody(tenantUuid), tenantInit(tenantUuid)))
    }
  }
}
