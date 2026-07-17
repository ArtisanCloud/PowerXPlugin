using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;

namespace PowerXPlugin.Framework.Runtime.Scheduler;

public static class SchedulerExtensions
{
    public static IServiceCollection AddLocalScheduler(this IServiceCollection services)
    {
        services.AddSingleton<LocalScheduler>();
        services.AddSingleton<IScheduler>(sp => sp.GetRequiredService<LocalScheduler>());
        services.AddHostedService<SchedulerRunner>();
        return services;
    }
}

/// <summary>
/// Background service that ticks the scheduler every minute, invoking due jobs.
/// </summary>
internal class SchedulerRunner : BackgroundService
{
    private readonly LocalScheduler _scheduler;
    private readonly ILogger<SchedulerRunner> _logger;
    private readonly Dictionary<string, SchedulerHandler> _handlers = new();

    public SchedulerRunner(LocalScheduler scheduler, ILogger<SchedulerRunner> logger)
    {
        _scheduler = scheduler;
        _logger = logger;
    }

    public void RegisterHandler(string jobName, SchedulerHandler handler) => _handlers[jobName] = handler;

    protected override async Task ExecuteAsync(CancellationToken ct)
    {
        while (!ct.IsCancellationRequested)
        {
            try
            {
                var now = DateTime.UtcNow;
                var jobs = await _scheduler.ListJobsAsync(status: JobStatus.Active);
                foreach (var job in jobs.Where(j => j.NextRunAt <= now))
                {
                    if (_handlers.TryGetValue(job.Name, out var handler))
                    {
                        await handler(job, ct);
                        job.NextRunAt = LocalScheduler.ComputeNextRun(job.ScheduleType, job.ScheduleExpr, now);
                    }
                }
            }
            catch (Exception ex) { _logger.LogError(ex, "Scheduler tick failed"); }
            await Task.Delay(TimeSpan.FromMinutes(1), ct);
        }
    }
}
