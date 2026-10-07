using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;
using PowerXPlugin.Framework.EventBridge;
using PowerXPlugin.Framework.Runtime.Common;

namespace PowerXPlugin.Framework.Runtime.Scheduler;

public static class SchedulerExtensions
{
    public static IServiceCollection AddLocalScheduler(this IServiceCollection services)
    {
        return services.AddPowerXScheduler(ProviderMode.Local, _ => throw new InvalidOperationException(SchedulerErrors.CodeInvalidMode));
    }

    /// <summary>
    /// Registers exactly one scheduler implementation selected at application
    /// startup. Delegated mode never starts the in-process runner.
    /// </summary>
    public static IServiceCollection AddPowerXScheduler(
        this IServiceCollection services,
        ProviderMode mode,
        Func<IServiceProvider, IScheduler> delegatedFactory,
        Func<IServiceProvider, ILocalSchedulerStore>? localStoreFactory = null)
    {
        ArgumentNullException.ThrowIfNull(services);
        ArgumentNullException.ThrowIfNull(delegatedFactory);
        if (mode == ProviderMode.Local)
        {
            services.AddLocalEventBridge();
            services.AddSingleton(sp => new LocalScheduler(localStoreFactory == null
                ? new InMemorySchedulerStore() : localStoreFactory(sp)));
            services.AddHostedService<SchedulerRunner>();
        }
        return services.AddPowerXRuntime<IScheduler>("scheduler", mode,
            sp => sp.GetRequiredService<LocalScheduler>(), delegatedFactory);
    }
}

/// <summary>
/// Background service that ticks the scheduler every minute, invoking due jobs.
/// </summary>
internal class SchedulerRunner : BackgroundService
{
    private readonly LocalScheduler _scheduler;
    private readonly ILogger<SchedulerRunner> _logger;
    private readonly IEventEmitter _eventEmitter;

    public SchedulerRunner(LocalScheduler scheduler, IEventEmitter eventEmitter, ILogger<SchedulerRunner> logger)
    {
        _scheduler = scheduler;
        _eventEmitter = eventEmitter;
        _logger = logger;
    }

    protected override async Task ExecuteAsync(CancellationToken ct)
    {
        try
        {
            while (!ct.IsCancellationRequested)
            {
                try
                {
                    await TickAsync(DateTime.UtcNow, ct);
                }
                catch (Exception ex) when (ex is not OperationCanceledException) { _logger.LogError(ex, "Scheduler tick failed"); }
                await Task.Delay(TimeSpan.FromMinutes(1), ct);
            }
        }
        catch (OperationCanceledException) when (ct.IsCancellationRequested) { }
    }

    internal async Task TickAsync(DateTime now, CancellationToken ct = default)
    {
        var jobs = await _scheduler.ListJobsAsync(status: JobStatus.Active, ct: ct);
        foreach (var job in jobs.Where(j => j.NextRunAt <= now))
        {
            ct.ThrowIfCancellationRequested();
            try
            {
                var scheduledAt = job.NextRunAt;
                var traceId = Guid.NewGuid().ToString();
                var payload = new SchedulerTriggeredPayload(
                    job.JobId, job.Name, job.OwnerType, job.OwnerId, job.TenantUuid,
                    job.ScheduleType, scheduledAt, now, traceId,
                    $"{job.Uuid}:{scheduledAt.Ticks}",
                    TryGetString(job.Payload, "business_action"), job.Payload ?? []);
                await _eventEmitter.EmitAsync(new FrameworkEvent
                {
                    Topic = job.Topic,
                    Meta = new EventMeta
                    {
                        TenantUUID = job.TenantUuid, SourcePlugin = job.OwnerId,
                        TraceID = traceId, OccurredAt = now
                    },
                    Payload = payload
                }, ct);
                job.LastRunAt = now;
                if (job.ScheduleType == ScheduleType.Once) job.Status = JobStatus.Completed;
                else job.NextRunAt = LocalScheduler.ComputeNextRun(job.ScheduleType, job.ScheduleExpr, now);
                job.UpdatedAt = now;
                await _scheduler.SaveAsync(job, ct);
            }
            catch (Exception ex) when (ex is not OperationCanceledException)
            {
                // Keep the due time and idempotency key for retry. One failing
                // job must not prevent another tenant's due jobs from running.
                _logger.LogError(ex, "Scheduler job failed. TenantUuid={TenantUuid} JobId={JobId}", job.TenantUuid, job.JobId);
            }
        }
    }

    private static string? TryGetString(IReadOnlyDictionary<string, object?>? payload, string key) =>
        payload != null && payload.TryGetValue(key, out var value) ? value?.ToString() : null;
}
