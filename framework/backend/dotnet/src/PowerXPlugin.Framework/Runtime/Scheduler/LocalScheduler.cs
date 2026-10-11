using System.Globalization;
using System.Collections.Concurrent;
using Cronos;
using PowerXPlugin.Framework.EventBridge;

namespace PowerXPlugin.Framework.Runtime.Scheduler;

public class LocalScheduler : IScheduler, ILocalSchedulerDiagnostics
{
    private readonly ILocalSchedulerStore _store;
    private readonly IEventEmitter _emitter;
    private readonly ConcurrentDictionary<(string Tenant, string Job), SemaphoreSlim> _executionGates = new();
    public LocalScheduler() : this(new InMemorySchedulerStore()) { }
    public LocalScheduler(ILocalSchedulerStore store) : this(store, new LocalEventBridge()) { }
    public LocalScheduler(ILocalSchedulerStore store, IEventEmitter emitter)
    {
        _store = store ?? throw new ArgumentNullException(nameof(store));
        _emitter = emitter ?? throw new ArgumentNullException(nameof(emitter));
    }

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

    /// <summary>Persists a due time, dispatches immediately, and returns only after completion is saved.</summary>
    public async Task<bool> TriggerJobAsync(string jobId, string tenantUuid, CancellationToken ct = default)
    {
        var key = JobKey(jobId, tenantUuid);
        var gate = _executionGates.GetOrAdd((key.TenantUuid, key.JobId), _ => new SemaphoreSlim(1, 1));
        await gate.WaitAsync(ct);
        try
        {
            var job = await GetJobAsync(jobId, tenantUuid, ct);
            if (job == null) return false;
            if (job.Status != JobStatus.Active) throw new SchedulerAdapterException(SchedulerErrors.CodeNotActive);
            var now = DateTime.UtcNow;
            // Keep an already-due occurrence's key on retry. A failed consumer
            // must not become a different occurrence on the next manual attempt.
            if (job.NextRunAt > now)
            {
                job.NextRunAt = now;
                job.UpdatedAt = now;
                await SaveAsync(job, ct);
            }
            await DispatchAsync(job, now, "manual", ct);
            return true;
        }
        catch (Exception ex) when (ex is not OperationCanceledException and not SchedulerAdapterException)
        { throw new SchedulerAdapterException(SchedulerErrors.CodeUpstreamDependency, innerException: ex); }
        finally { gate.Release(); }
    }

    internal async Task ExecuteDueAsync(Job snapshot, DateTime now, CancellationToken ct)
    {
        var gate = _executionGates.GetOrAdd((snapshot.TenantUuid, snapshot.JobId), _ => new SemaphoreSlim(1, 1));
        await gate.WaitAsync(ct);
        try
        {
            // A manual trigger may have completed the snapshot while the runner
            // was waiting. Reload under the same gate before emitting anything.
            var job = await GetJobAsync(snapshot.JobId, snapshot.TenantUuid, ct);
            if (job == null || job.Uuid != snapshot.Uuid || job.Status != JobStatus.Active || job.Revision != snapshot.Revision || job.NextRunAt > now) return;
            await DispatchAsync(job, now, job.ScheduleType, ct);
        }
        finally { gate.Release(); }
    }

    public async Task<bool> TriggerJobPreviewAsync(string jobId, string tenantUuid, CancellationToken ct = default)
    {
        var key = JobKey(jobId, tenantUuid);
        var gate = _executionGates.GetOrAdd((key.TenantUuid, key.JobId), _ => new SemaphoreSlim(1, 1));
        await gate.WaitAsync(ct);
        try
        {
            var job = await GetJobAsync(jobId, tenantUuid, ct);
            if (job == null) return false;
            if (job.Status == JobStatus.Paused) throw new SchedulerAdapterException(SchedulerErrors.CodeNotActive);
            await DispatchAsync(job, DateTime.UtcNow, "manual_preview", ct, preview: true);
            return true;
        }
        catch (Exception ex) when (ex is not OperationCanceledException and not SchedulerAdapterException)
        { throw new SchedulerAdapterException(SchedulerErrors.CodeUpstreamDependency, innerException: ex); }
        finally { gate.Release(); }
    }

    public async Task<bool> DeleteJobAsync(string jobId, string tenantUuid, string jobUuid, long expectedRevision, CancellationToken ct = default)
    {
        var key = JobKey(jobId, tenantUuid);
        var gate = _executionGates.GetOrAdd((key.TenantUuid, key.JobId), _ => new SemaphoreSlim(1, 1));
        await gate.WaitAsync(ct);
        try
        {
            var job = await GetJobAsync(jobId, tenantUuid, ct);
            if (job == null) return false;
            if (job.Uuid != jobUuid || job.Revision != expectedRevision) throw new SchedulerAdapterException(SchedulerErrors.CodeConflict);
            if (!await _store.DeleteAsync(key.TenantUuid, key.JobId, jobUuid, expectedRevision, ct))
                throw new SchedulerAdapterException(SchedulerErrors.CodeConflict);
            return true;
        }
        finally { gate.Release(); }
    }

    private async Task DispatchAsync(Job job, DateTime now, string source, CancellationToken ct, bool preview = false)
    {
        var traceId = Guid.NewGuid().ToString();
        var executionId = preview ? $"preview:{job.Uuid}:{Guid.NewGuid():N}" : $"{job.Uuid}:{job.NextRunAt.Ticks}";
        var payload = new SchedulerTriggeredPayload(
            job.JobId, job.Name, job.OwnerType, job.OwnerId, job.TenantUuid,
            source, preview ? now : job.NextRunAt, now, traceId, executionId,
            job.Payload?.TryGetValue("business_action", out var action) == true ? action?.ToString() : null, job.Payload ?? []);
        await _emitter.EmitAsync(new FrameworkEvent
        {
            Topic = job.Topic,
            Meta = new EventMeta { TenantUUID = job.TenantUuid, SourcePlugin = job.OwnerId, TraceID = traceId, OccurredAt = now },
            Payload = payload
        }, ct);
        job.LastRunAt = now;
        job.LastExecutionId = executionId;
        if (!preview)
        {
            if (job.ScheduleType == ScheduleType.Once) job.Status = JobStatus.Completed;
            else job.NextRunAt = ComputeNextRun(job.ScheduleType, job.ScheduleExpr, now);
        }
        job.UpdatedAt = now;
        await SaveAsync(job, ct);
    }

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
