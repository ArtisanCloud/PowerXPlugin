using System.Collections.Concurrent;
using Cronos;

namespace PowerXPlugin.Framework.Runtime.Scheduler;

public class LocalScheduler : IScheduler
{
    private readonly ConcurrentDictionary<string, Job> _jobs = new();

    public Task<Job> CreateJobAsync(JobSpec spec, CancellationToken ct = default)
    {
        var job = new Job
        {
            Uuid = Guid.NewGuid().ToString(),
            JobId = spec.JobId ?? $"sch-{Guid.NewGuid():N}"[..12],
            Name = spec.Name,
            TenantUuid = spec.TenantUuid ?? "",
            OwnerType = spec.OwnerType,
            OwnerId = spec.OwnerId,
            ScheduleType = spec.ScheduleType,
            ScheduleExpr = spec.ScheduleExpr,
            Topic = spec.Topic ?? "powerx.runtime.scheduler.triggered.v1",
            Status = spec.Paused ? JobStatus.Paused : JobStatus.Active,
            Payload = spec.Payload,
            NextRunAt = ComputeNextRun(spec.ScheduleType, spec.ScheduleExpr, DateTime.UtcNow)
        };
        _jobs[job.JobId] = job;
        return Task.FromResult(job);
    }

    public Task<Job?> UpdateJobAsync(JobSpec spec, CancellationToken ct = default)
    {
        if (!_jobs.TryGetValue(spec.JobId!, out var job)) return Task.FromResult<Job?>(null);
        job.Name = spec.Name;
        job.ScheduleType = spec.ScheduleType;
        job.ScheduleExpr = spec.ScheduleExpr;
        job.Status = spec.Paused ? JobStatus.Paused : JobStatus.Active;
        job.NextRunAt = ComputeNextRun(spec.ScheduleType, spec.ScheduleExpr, DateTime.UtcNow);
        job.UpdatedAt = DateTime.UtcNow;
        return Task.FromResult<Job?>(job);
    }

    public Task<bool> PauseJobAsync(string jobId, string tenantUuid, CancellationToken ct = default)
    {
        if (!_jobs.TryGetValue(jobId, out var j)) return Task.FromResult(false);
        j.Status = JobStatus.Paused; return Task.FromResult(true);
    }

    public Task<bool> ResumeJobAsync(string jobId, string tenantUuid, CancellationToken ct = default)
    {
        if (!_jobs.TryGetValue(jobId, out var j)) return Task.FromResult(false);
        j.Status = JobStatus.Active; j.NextRunAt = ComputeNextRun(j.ScheduleType, j.ScheduleExpr, DateTime.UtcNow);
        return Task.FromResult(true);
    }

    public Task<bool> TriggerJobAsync(string jobId, string tenantUuid, CancellationToken ct = default)
    {
        if (!_jobs.TryGetValue(jobId, out var j)) return Task.FromResult(false);
        j.NextRunAt = DateTime.UtcNow; return Task.FromResult(true);
    }

    public Task<Job?> GetJobAsync(string jobId, string tenantUuid, CancellationToken ct = default)
        => Task.FromResult(_jobs.TryGetValue(jobId, out var j) ? j : null);

    public Task<List<Job>> ListJobsAsync(string? tenantUuid = null, string? ownerType = null, string? ownerId = null, string? status = null, CancellationToken ct = default)
    {
        var q = _jobs.Values.AsEnumerable();
        if (tenantUuid != null) q = q.Where(j => j.TenantUuid == tenantUuid);
        if (ownerType != null) q = q.Where(j => j.OwnerType == ownerType);
        if (ownerId != null) q = q.Where(j => j.OwnerId == ownerId);
        if (status != null) q = q.Where(j => j.Status == status);
        return Task.FromResult(q.ToList());
    }

    public static DateTime ComputeNextRun(string scheduleType, string expr, DateTime from)
    {
        return scheduleType switch
        {
            ScheduleType.Once => DateTime.TryParse(expr, out var dt) ? dt : from.AddMinutes(1),
            ScheduleType.Interval => TimeSpan.TryParse(expr, out var ts) ? from.Add(ts) : from.AddMinutes(5),
            ScheduleType.Cron => ParseCron(expr, from),
            _ => from.AddMinutes(5)
        };
    }

    private static DateTime ParseCron(string expr, DateTime from)
    {
        try { var exp = CronExpression.Parse(expr); return exp.GetNextOccurrence(from, TimeZoneInfo.Utc) ?? from.AddMinutes(5); }
        catch { return from.AddMinutes(5); }
    }
}
