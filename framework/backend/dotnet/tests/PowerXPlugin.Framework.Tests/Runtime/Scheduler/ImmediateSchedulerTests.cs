using Microsoft.Extensions.Logging.Abstractions;
using PowerXPlugin.Framework.EventBridge;
using PowerXPlugin.Framework.Runtime.Scheduler;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.Scheduler;

public sealed class ImmediateSchedulerTests
{
    private const string Tenant = "00000000-0000-4000-8000-000000000001";

    [Fact]
    public async Task Manual_trigger_dispatches_and_saves_once_completion_before_return_without_a_tick()
    {
        var bridge = new LocalEventBridge(); var scheduler = new LocalScheduler(new InMemorySchedulerStore(), bridge);
        var payloads = new List<SchedulerTriggeredPayload>();
        await bridge.RegisterHandlerAsync("topic", (evt, _) => { payloads.Add(Assert.IsType<SchedulerTriggeredPayload>(evt.Payload)); return Task.CompletedTask; });
        await scheduler.CreateJobAsync(Spec());
        Assert.True(await scheduler.TriggerJobAsync("job", Tenant));
        var saved = (await scheduler.GetJobAsync("job", Tenant))!;
        Assert.Equal(JobStatus.Completed, saved.Status);
        Assert.Equal(saved.LastRunAt, Assert.Single(payloads).FiredAt);
        Assert.Equal("manual", payloads[0].TriggerSource);
        using var runner = new SchedulerRunner(scheduler, NullLogger<SchedulerRunner>.Instance);
        await runner.TickAsync(DateTime.UtcNow.AddHours(2));
        Assert.Single(payloads);
    }

    [Fact]
    public async Task Manual_failure_returns_error_and_persisted_retry_reuses_the_same_occurrence()
    {
        var store = new InMemorySchedulerStore(); var bridge = new LocalEventBridge(); var scheduler = new LocalScheduler(store, bridge);
        var payloads = new List<SchedulerTriggeredPayload>(); var fail = true;
        await bridge.RegisterHandlerAsync("topic", (evt, _) =>
        {
            payloads.Add(Assert.IsType<SchedulerTriggeredPayload>(evt.Payload));
            if (fail) throw new InvalidOperationException("CONSUMER_FAILED");
            return Task.CompletedTask;
        });
        await scheduler.CreateJobAsync(Spec());
        var error = await Assert.ThrowsAsync<SchedulerAdapterException>(() => scheduler.TriggerJobAsync("job", Tenant));
        Assert.Equal(SchedulerErrors.CodeUpstreamDependency, error.Code);
        var due = (await scheduler.GetJobAsync("job", Tenant))!;
        Assert.Null(due.LastRunAt); Assert.Equal(JobStatus.Active, due.Status);
        fail = false;
        // A new runtime using the same store must retry the durable occurrence.
        var restarted = new LocalScheduler(store, bridge);
        using var runner = new SchedulerRunner(restarted, NullLogger<SchedulerRunner>.Instance);
        await runner.TickAsync(due.NextRunAt.AddSeconds(1));
        Assert.Equal(JobStatus.Completed, (await restarted.GetJobAsync("job", Tenant))!.Status);
        Assert.Equal(2, payloads.Count);
        Assert.Equal(payloads[0].IdempotencyKey, payloads[1].IdempotencyKey);
        Assert.Equal(payloads[0].ScheduledAt, payloads[1].ScheduledAt);
    }

    [Fact]
    public async Task Overlapping_manual_and_scheduled_execution_emit_a_once_job_only_once()
    {
        var bridge = new LocalEventBridge(); var scheduler = new LocalScheduler(new InMemorySchedulerStore(), bridge);
        var entered = new TaskCompletionSource(TaskCreationOptions.RunContinuationsAsynchronously);
        var release = new TaskCompletionSource(TaskCreationOptions.RunContinuationsAsynchronously);
        var count = 0;
        await bridge.RegisterHandlerAsync("topic", async (_, _) => { Interlocked.Increment(ref count); entered.SetResult(); await release.Task; });
        await scheduler.CreateJobAsync(Spec());
        var manual = scheduler.TriggerJobAsync("job", Tenant);
        await entered.Task.WaitAsync(TimeSpan.FromSeconds(5));
        using var runner = new SchedulerRunner(scheduler, NullLogger<SchedulerRunner>.Instance);
        var scheduled = runner.TickAsync(DateTime.UtcNow.AddSeconds(1));
        release.SetResult();
        await Task.WhenAll(manual, scheduled).WaitAsync(TimeSpan.FromSeconds(5));
        Assert.Equal(1, count);
        Assert.Equal(JobStatus.Completed, (await scheduler.GetJobAsync("job", Tenant))!.Status);
    }

    [Fact]
    public async Task Recurring_manual_execution_keeps_a_future_schedule_and_inactive_jobs_fail_explicitly()
    {
        var scheduler = new LocalScheduler();
        await scheduler.CreateJobAsync(Spec() with { ScheduleType = ScheduleType.Interval, ScheduleExpr = "01:00:00" });
        Assert.True(await scheduler.TriggerJobAsync("job", Tenant));
        var saved = (await scheduler.GetJobAsync("job", Tenant))!;
        Assert.Equal(JobStatus.Active, saved.Status); Assert.True(saved.NextRunAt > saved.LastRunAt);
        await scheduler.PauseJobAsync("job", Tenant);
        var error = await Assert.ThrowsAsync<SchedulerAdapterException>(() => scheduler.TriggerJobAsync("job", Tenant));
        Assert.Equal(SchedulerErrors.CodeNotActive, error.Code);
        Assert.Equal(saved.LastRunAt, (await scheduler.GetJobAsync("job", Tenant))!.LastRunAt);
    }

    private static JobSpec Spec() => new("job", "plugin", "crm", ScheduleType.Once, DateTime.UtcNow.AddHours(1).ToString("O"),
        JobId: "job", TenantUuid: Tenant, Topic: "topic");
}
