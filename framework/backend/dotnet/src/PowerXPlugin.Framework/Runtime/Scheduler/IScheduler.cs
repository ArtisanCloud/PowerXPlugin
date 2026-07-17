namespace PowerXPlugin.Framework.Runtime.Scheduler;

public interface IScheduler
{
    Task<Job> CreateJobAsync(JobSpec spec, CancellationToken ct = default);
    Task<Job?> UpdateJobAsync(JobSpec spec, CancellationToken ct = default);
    Task<bool> PauseJobAsync(string jobId, string tenantUuid, CancellationToken ct = default);
    Task<bool> ResumeJobAsync(string jobId, string tenantUuid, CancellationToken ct = default);
    Task<bool> TriggerJobAsync(string jobId, string tenantUuid, CancellationToken ct = default);
    Task<Job?> GetJobAsync(string jobId, string tenantUuid, CancellationToken ct = default);
    Task<List<Job>> ListJobsAsync(string? tenantUuid = null, string? ownerType = null, string? ownerId = null, string? status = null, CancellationToken ct = default);
}

public delegate Task SchedulerHandler(Job job, CancellationToken ct);
