import {
  createSchedulerClient,
  type ListSchedulerJobsInput,
  type SchedulerJobSpec
} from "@artisan-cloud/plugin-framework-client";
import { useApiClient } from "./_client";

const unwrapApi = <T>(value: any): T => {
  if (value && typeof value === "object" && "data" in value) {
    return value.data as T;
  }
  return value as T;
};

export function useSchedulerApi() {
  const { get, post, put } = useApiClient();
  const api = {
    get: async <T>(path: string, init?: RequestInit) => {
      return unwrapApi<T>(await get(path, init as any));
    },
    post: async <T>(path: string, body: unknown, init?: RequestInit) => {
      return unwrapApi<T>(await post(path, body, init as any));
    },
    put: async <T>(path: string, body: unknown, init?: RequestInit) => {
      return unwrapApi<T>(await put(path, body, init as any));
    },
    delete: async <T>(path: string, init?: RequestInit) => {
      return unwrapApi<T>(await useApiClient().delete(path, init as any));
    },
  };
  return createSchedulerClient(api);
}

export function listSchedulerJobs(input?: ListSchedulerJobsInput) {
  return useSchedulerApi().listJobs(input);
}

export function createSchedulerJob(job: SchedulerJobSpec) {
  return useSchedulerApi().createJob(job);
}

export function updateSchedulerJob(jobId: string, job: SchedulerJobSpec) {
  return useSchedulerApi().updateJob(jobId, job);
}

export function pauseSchedulerJob(
  jobId: string,
  tenantUuid?: string | { tenant_uuid?: string; provider_mode?: "local" | "host" | "dual" | "proxy" | "gateway"; force_local?: boolean; force_host?: boolean }
) {
  return useSchedulerApi().pauseJob(jobId, tenantUuid);
}

export function resumeSchedulerJob(
  jobId: string,
  tenantUuid?: string | { tenant_uuid?: string; provider_mode?: "local" | "host" | "dual" | "proxy" | "gateway"; force_local?: boolean; force_host?: boolean }
) {
  return useSchedulerApi().resumeJob(jobId, tenantUuid);
}

export function triggerSchedulerJob(
  jobId: string,
  tenantUuid?: string | { tenant_uuid?: string; provider_mode?: "local" | "host" | "dual" | "proxy" | "gateway"; force_local?: boolean; force_host?: boolean }
) {
  return useSchedulerApi().triggerJob(jobId, tenantUuid);
}

export type {
  ListSchedulerJobsInput,
  ListSchedulerJobsResult,
  SchedulerActionResult,
  SchedulerClient,
  SchedulerJob,
  SchedulerJobSpec,
  SchedulerScheduleType
} from "@artisan-cloud/plugin-framework-client";

export type SchedulerProviderMode = "local" | "host" | "dual" | "proxy" | "gateway";
