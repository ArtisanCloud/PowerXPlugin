using Microsoft.Extensions.Logging.Abstractions;
using PowerXPlugin.Framework.EventBridge;
using PowerXPlugin.Framework.Runtime.Scheduler;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime;

public sealed class SchedulerRunnerTests
{
    private const string Tenant = "00000000-0000-4000-8000-000000000001";
    private const string Other = "00000000-0000-4000-8000-000000000002";

    [Fact]
    public async Task Once_completes_after_success_and_is_not_emitted_again()
    {
        var scheduler = new LocalScheduler();
        var bridge = new LocalEventBridge();
        var payloads = new List<SchedulerTriggeredPayload>();
        await bridge.RegisterHandlerAsync("topic", (evt, _) =>
        {
            payloads.Add(Assert.IsType<SchedulerTriggeredPayload>(evt.Payload));
            return Task.CompletedTask;
        });
        var now = DateTime.UtcNow;
        var job = await scheduler.CreateJobAsync(Spec(Tenant, now));
        using var runner = new SchedulerRunner(scheduler, bridge, NullLogger<SchedulerRunner>.Instance);
        await runner.TickAsync(now.AddSeconds(1));
        await runner.TickAsync(now.AddMinutes(1));
        job = (await scheduler.GetJobAsync(job.JobId, Tenant))!;
        Assert.Equal(JobStatus.Completed, job.Status);
        Assert.Equal(now.AddSeconds(1), job.LastRunAt);
        Assert.Equal(ScheduleType.Once, Assert.Single(payloads).TriggerSource);
    }

    [Fact]
    public async Task Failing_tenant_does_not_block_other_jobs_and_retry_keeps_same_key()
    {
        var scheduler = new LocalScheduler();
        var bridge = new LocalEventBridge();
        var attempts = new List<SchedulerTriggeredPayload>();
        var fail = true;
        await bridge.RegisterHandlerAsync("topic", (evt, _) =>
        {
            var payload = Assert.IsType<SchedulerTriggeredPayload>(evt.Payload);
            attempts.Add(payload);
            if (payload.TenantUuid == Tenant && fail) throw new InvalidOperationException("consumer failed");
            return Task.CompletedTask;
        });
        var now = DateTime.UtcNow;
        var failed = await scheduler.CreateJobAsync(Spec(Tenant, now));
        var succeeded = await scheduler.CreateJobAsync(Spec(Other, now));
        var dueAt = failed.NextRunAt;
        using var runner = new SchedulerRunner(scheduler, bridge, NullLogger<SchedulerRunner>.Instance);
        await runner.TickAsync(now.AddSeconds(1));
        failed = (await scheduler.GetJobAsync(failed.JobId, Tenant))!;
        succeeded = (await scheduler.GetJobAsync(succeeded.JobId, Other))!;
        Assert.Equal(JobStatus.Active, failed.Status);
        Assert.Null(failed.LastRunAt);
        Assert.Equal(dueAt, failed.NextRunAt);
        Assert.Equal(JobStatus.Completed, succeeded.Status);
        fail = false;
        await runner.TickAsync(now.AddSeconds(2));
        failed = (await scheduler.GetJobAsync(failed.JobId, Tenant))!;
        Assert.Equal(JobStatus.Completed, failed.Status);
        var retries = attempts.Where(x => x.TenantUuid == Tenant).ToArray();
        Assert.Equal(2, retries.Length);
        Assert.Equal(retries[0].IdempotencyKey, retries[1].IdempotencyKey);
        Assert.Single(attempts, x => x.TenantUuid == Other);
    }

    [Fact]
    public async Task Cancellation_stops_tick_without_completing_job()
    {
        var scheduler = new LocalScheduler();
        var bridge = new LocalEventBridge();
        using var cts = new CancellationTokenSource();
        await bridge.RegisterHandlerAsync("topic", (_, _) =>
        {
            cts.Cancel();
            throw new OperationCanceledException(cts.Token);
        });
        var now = DateTime.UtcNow;
        var job = await scheduler.CreateJobAsync(Spec(Tenant, now));
        using var runner = new SchedulerRunner(scheduler, bridge, NullLogger<SchedulerRunner>.Instance);
        await Assert.ThrowsAsync<OperationCanceledException>(() => runner.TickAsync(now.AddSeconds(1), cts.Token));
        Assert.Null(job.LastRunAt);
        Assert.Equal(JobStatus.Active, job.Status);
    }

    private static JobSpec Spec(string tenant, DateTime now) => new("once", "plugin", "crm",
        ScheduleType.Once, now.AddSeconds(-1).ToString("O"), JobId: "once", TenantUuid: tenant, Topic: "topic");

    [Fact]
    public void Once_offset_is_normalized_to_utc_for_runner_comparison()
    {
        var due = LocalScheduler.ComputeNextRun(ScheduleType.Once, "2026-10-03T08:00:00+08:00", DateTime.UtcNow);
        Assert.Equal(new DateTime(2026, 10, 3, 0, 0, 0, DateTimeKind.Utc), due);
        Assert.Equal(DateTimeKind.Utc, due.Kind);
    }
}
