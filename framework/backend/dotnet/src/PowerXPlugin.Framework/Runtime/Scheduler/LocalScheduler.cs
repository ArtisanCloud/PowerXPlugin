using System.Globalization;
using Cronos;

namespace PowerXPlugin.Framework.Runtime.Scheduler;

public class LocalScheduler : IScheduler
{
    private readonly ILocalSchedulerStore _store;
    public LocalScheduler() : this(new InMemorySchedulerStore()) { }
    public LocalScheduler(ILocalSchedulerStore store) => _store = store ?? throw new ArgumentNullException(nameof(store));

    public async Task<Job> CreateJobAsync(JobSpec spec, CancellationToken ct = default)
    {
        var tenantUuid = RequireTenantUuid(spec.TenantUuid);
        var job = new Job
        {
            Uuid = Guid.NewGuid().ToString(),
            JobId = spec.JobId ?? $"sch-{Guid.NewGuid():N}"[..12],
            Name = spec.Name,
            TenantUuid = tenantUuid,
            OwnerType = spec.OwnerType,
            OwnerId = spec.OwnerId,
            ScheduleType = spec.ScheduleType,
            ScheduleExpr = spec.ScheduleExpr,
            Topic = spec.Topic ?? "powerx.runtime.scheduler.triggered.v1",
            Status = spec.Paused ? JobStatus.Paused : JobStatus.Active,
            Payload = spec.Payload,
            IdempotencyKey = spec.IdempotencyKey,
            Timezone = spec.Timezone,
            NextRunAt = ComputeNextRun(spec.ScheduleType, spec.ScheduleExpr, DateTime.UtcNow)
        };
        await _store.CreateAsync(job, ct);
        return job;
    }

    public async Task<Job?> UpdateJobAsync(JobSpec spec, CancellationToken ct = default)
    {
        var key = JobKey(spec.JobId, spec.TenantUuid);
        var job = await _store.GetAsync(key.TenantUuid, key.JobId, ct);
        if (job == null) return null;
        job.Name = spec.Name;
        job.ScheduleType = spec.ScheduleType;
        job.ScheduleExpr = spec.ScheduleExpr;
        job.Payload = spec.Payload;
        job.IdempotencyKey = spec.IdempotencyKey;
        job.Topic = spec.Topic ?? "powerx.runtime.scheduler.triggered.v1";
        job.Timezone = spec.Timezone;
        job.Status = spec.Paused ? JobStatus.Paused : JobStatus.Active;
        job.NextRunAt = ComputeNextRun(spec.ScheduleType, spec.ScheduleExpr, DateTime.UtcNow);
        job.UpdatedAt = DateTime.UtcNow;
        await SaveAsync(job, ct);
        return job;
    }

    public Task<bool> PauseJobAsync(string jobId, string tenantUuid, CancellationToken ct = default)
        => ChangeAsync(jobId, tenantUuid, j => j.Status = JobStatus.Paused, ct);

    public Task<bool> ResumeJobAsync(string jobId, string tenantUuid, CancellationToken ct = default)
        => ChangeAsync(jobId, tenantUuid, j => { j.Status = JobStatus.Active; j.NextRunAt = ComputeNextRun(j.ScheduleType, j.ScheduleExpr, DateTime.UtcNow); }, ct);

    public Task<bool> TriggerJobAsync(string jobId, string tenantUuid, CancellationToken ct = default)
        => ChangeAsync(jobId, tenantUuid, j => j.NextRunAt = DateTime.UtcNow, ct);

    public Task<Job?> GetJobAsync(string jobId, string tenantUuid, CancellationToken ct = default)
    {
        var key = JobKey(jobId, tenantUuid);
        return _store.GetAsync(key.TenantUuid, key.JobId, ct);
    }

    public async Task<List<Job>> ListJobsAsync(string? tenantUuid = null, string? ownerType = null, string? ownerId = null, string? status = null, CancellationToken ct = default)
    {
        var tenant = tenantUuid == null ? null : RequireTenantUuid(tenantUuid);
        var q = (await _store.ListAsync(ct)).AsEnumerable();
        if (tenant != null) q = q.Where(j => j.TenantUuid == tenant);
        if (ownerType != null) q = q.Where(j => j.OwnerType == ownerType);
        if (ownerId != null) q = q.Where(j => j.OwnerId == ownerId);
        if (status != null) q = q.Where(j => j.Status == status);
        return q.ToList();
    }

    private async Task<bool> ChangeAsync(string jobId, string tenantUuid, Action<Job> change, CancellationToken ct)
    {
        var job = await GetJobAsync(jobId, tenantUuid, ct);
        if (job == null) return false;
        change(job);
        job.UpdatedAt = DateTime.UtcNow;
        await SaveAsync(job, ct);
        return true;
    }

    internal async Task SaveAsync(Job job, CancellationToken ct)
    {
        if (!await _store.SaveAsync(job, job.Revision, ct)) throw new SchedulerAdapterException(SchedulerErrors.CodeConflict);
        job.Revision++;
    }

    private static (string TenantUuid, string JobId) JobKey(string? jobId, string? tenantUuid)
    {
        if (string.IsNullOrWhiteSpace(jobId)) throw new SchedulerAdapterException(SchedulerErrors.CodeInvalidJob);
        return (RequireTenantUuid(tenantUuid), jobId);
    }

    private static string RequireTenantUuid(string? tenantUuid)
    {
        if (!Guid.TryParse(tenantUuid, out var tenant) || tenant == Guid.Empty)
            throw new SchedulerAdapterException(SchedulerErrors.CodeInvalidJob);
        return tenant.ToString("D");
    }

    public static DateTime ComputeNextRun(string scheduleType, string expr, DateTime from)
    {
        return scheduleType switch
        {
            ScheduleType.Once => DateTime.TryParse(expr, CultureInfo.InvariantCulture,
                DateTimeStyles.AssumeUniversal | DateTimeStyles.AdjustToUniversal, out var dt) ? dt : from.AddMinutes(1),
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
