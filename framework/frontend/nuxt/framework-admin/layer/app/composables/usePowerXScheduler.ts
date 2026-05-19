import { computed, ref } from 'vue'
import { usePluginApi } from '../../../../framework-client/api'
import {
  createSchedulerClient,
  type ListSchedulerJobsInput,
  type SchedulerClient,
  type SchedulerJob,
  type SchedulerJobSpec
} from '../../../../framework-client/scheduler'

export interface UsePowerXSchedulerOptions {
  pluginId: string
  baseURL?: string
  tenantUuid?: string
}

export const usePowerXScheduler = (options: UsePowerXSchedulerOptions) => {
  const api = usePluginApi({
    pluginId: options.pluginId,
    baseURL: options.baseURL,
    tenantUuid: options.tenantUuid
  })
  const client: SchedulerClient = createSchedulerClient(api)
  const jobs = ref<SchedulerJob[]>([])
  const loading = ref(false)
  const error = ref<Error | null>(null)

  const total = computed(() => jobs.value.length)

  const listJobs = async (input?: ListSchedulerJobsInput) => {
    loading.value = true
    error.value = null
    try {
      const result = await client.listJobs({
        tenant_uuid: options.tenantUuid,
        ...input
      })
      jobs.value = result.items || []
      return result
    } catch (err: any) {
      error.value = err
      throw err
    } finally {
      loading.value = false
    }
  }

  const createJob = async (job: SchedulerJobSpec) => {
    const created = await client.createJob(job)
    await listJobs()
    return created
  }

  const updateJob = async (jobId: string, job: SchedulerJobSpec) => {
    const updated = await client.updateJob(jobId, job)
    await listJobs()
    return updated
  }

  const pauseJob = async (jobId: string) => {
    const result = await client.pauseJob(jobId, options.tenantUuid)
    await listJobs()
    return result
  }

  const resumeJob = async (jobId: string) => {
    const result = await client.resumeJob(jobId, options.tenantUuid)
    await listJobs()
    return result
  }

  const triggerJob = async (jobId: string) => {
    return client.triggerJob(jobId, options.tenantUuid)
  }

  return {
    client,
    jobs,
    total,
    loading,
    error,
    listJobs,
    createJob,
    updateJob,
    pauseJob,
    resumeJob,
    triggerJob
  }
}

