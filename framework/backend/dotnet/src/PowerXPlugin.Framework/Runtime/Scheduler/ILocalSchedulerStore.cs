using System.Text.Json;

namespace PowerXPlugin.Framework.Runtime.Scheduler;

/// <summary>Plugin-owned durable storage. Reads return detached snapshots; writes are atomic by revision.</summary>
public interface ILocalSchedulerStore
{
    Task CreateAsync(Job job, CancellationToken ct = default);
    Task<Job?> GetAsync(string tenantUuid, string jobId, CancellationToken ct = default);
    Task<List<Job>> ListAsync(CancellationToken ct = default);
    Task<bool> SaveAsync(Job job, long expectedRevision, CancellationToken ct = default);
}

/// <summary>Explicit development/test storage; CRM production injects its PostgreSQL store.</summary>
public sealed class InMemorySchedulerStore : ILocalSchedulerStore
{
    private readonly object _gate = new();
    private readonly Dictionary<(string, string), Job> _jobs = [];
    private static Job Copy(Job job) => JsonSerializer.Deserialize<Job>(JsonSerializer.Serialize(job))! with { Revision = job.Revision };
    public Task CreateAsync(Job job, CancellationToken ct = default)
    {
        ct.ThrowIfCancellationRequested();
        lock (_gate)
            if (!_jobs.TryAdd((job.TenantUuid, job.JobId), Copy(job)))
                throw new SchedulerAdapterException(SchedulerErrors.CodeConflict);
        return Task.CompletedTask;
    }
    public Task<Job?> GetAsync(string tenantUuid, string jobId, CancellationToken ct = default)
    {
        ct.ThrowIfCancellationRequested();
        lock (_gate) return Task.FromResult(_jobs.TryGetValue((tenantUuid, jobId), out var job) ? Copy(job) : null);
    }
    public Task<List<Job>> ListAsync(CancellationToken ct = default)
    {
        ct.ThrowIfCancellationRequested();
        lock (_gate) return Task.FromResult(_jobs.Values.Select(Copy).ToList());
    }
    public Task<bool> SaveAsync(Job job, long expectedRevision, CancellationToken ct = default)
    {
        ct.ThrowIfCancellationRequested();
        lock (_gate)
        {
            var key = (job.TenantUuid, job.JobId);
            if (!_jobs.TryGetValue(key, out var current) || current.Revision != expectedRevision) return Task.FromResult(false);
            _jobs[key] = Copy(job) with { Revision = expectedRevision + 1 };
            return Task.FromResult(true);
        }
    }
}
