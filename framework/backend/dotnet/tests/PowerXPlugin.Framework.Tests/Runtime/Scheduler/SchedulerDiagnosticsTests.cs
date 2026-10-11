using Microsoft.Extensions.Logging.Abstractions;
using PowerXPlugin.Framework.EventBridge;
using PowerXPlugin.Framework.Runtime.Scheduler;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.Scheduler;

public sealed class SchedulerDiagnosticsTests
{
    private const string Tenant = "00000000-0000-4000-8000-000000000001";
    private const string Other = "00000000-0000-4000-8000-000000000002";

    [Fact]
    public async Task Preview_repeats_without_consuming_schedule_and_completed_jobs_remain_testable()
    {
        var bridge = new LocalEventBridge(); var scheduler = new LocalScheduler(new InMemorySchedulerStore(), bridge);
        var events = new List<SchedulerTriggeredPayload>();
        await bridge.RegisterHandlerAsync("topic", (evt, _) => { events.Add(Assert.IsType<SchedulerTriggeredPayload>(evt.Payload)); return Task.CompletedTask; });
        var initial = await scheduler.CreateJobAsync(Spec());
        Assert.True(await scheduler.TriggerJobPreviewAsync("job", Tenant));
        Assert.True(await scheduler.TriggerJobPreviewAsync("job", Tenant));
        var beforeDue = (await scheduler.GetJobAsync("job", Tenant))!;
        Assert.Equal(JobStatus.Active, beforeDue.Status); Assert.Equal(initial.NextRunAt, beforeDue.NextRunAt);
        Assert.Equal(2, events.Count); Assert.NotEqual(events[0].IdempotencyKey, events[1].IdempotencyKey);
        Assert.NotEqual(events[0].TraceId, events[1].TraceId); Assert.Equal(events[1].IdempotencyKey, beforeDue.LastExecutionId);
        using var runner = new SchedulerRunner(scheduler, NullLogger<SchedulerRunner>.Instance);
        await runner.TickAsync(initial.NextRunAt.AddSeconds(1));
        Assert.Equal(JobStatus.Completed, (await scheduler.GetJobAsync("job", Tenant))!.Status);
        Assert.True(await scheduler.TriggerJobPreviewAsync("job", Tenant));
        Assert.Equal(JobStatus.Completed, (await scheduler.GetJobAsync("job", Tenant))!.Status);
        Assert.Equal(4, events.Count); Assert.Equal(4, events.Select(x => x.IdempotencyKey).Distinct().Count());
    }

    [Fact]
    public async Task Failed_and_paused_previews_do_not_rewrite_the_schedule_or_success_record()
    {
        var bridge = new LocalEventBridge(); var scheduler = new LocalScheduler(new InMemorySchedulerStore(), bridge);
        var initial = await scheduler.CreateJobAsync(Spec());
        await bridge.RegisterHandlerAsync("topic", (_, _) => throw new InvalidOperationException("PREVIEW_FAILED"));
        var error = await Assert.ThrowsAsync<SchedulerAdapterException>(() => scheduler.TriggerJobPreviewAsync("job", Tenant));
        Assert.Equal(SchedulerErrors.CodeUpstreamDependency, error.Code);
        var saved = (await scheduler.GetJobAsync("job", Tenant))!;
        Assert.Equal(initial.NextRunAt, saved.NextRunAt); Assert.Null(saved.LastExecutionId); Assert.Null(saved.LastRunAt);
        await scheduler.PauseJobAsync("job", Tenant);
        Assert.Equal(SchedulerErrors.CodeNotActive, (await Assert.ThrowsAsync<SchedulerAdapterException>(() => scheduler.TriggerJobPreviewAsync("job", Tenant))).Code);
        Assert.False(await scheduler.TriggerJobPreviewAsync("job", Other));
    }

    [Fact]
    public async Task Delete_checks_tenant_uuid_and_revision_and_prevents_a_stale_due_snapshot_from_dispatching()
    {
        var bridge = new LocalEventBridge(); var scheduler = new LocalScheduler(new InMemorySchedulerStore(), bridge);
        var count = 0; await bridge.RegisterHandlerAsync("topic", (_, _) => { count++; return Task.CompletedTask; });
        var initial = await scheduler.CreateJobAsync(Spec());
        Assert.False(await scheduler.DeleteJobAsync("job", Other, initial.Uuid, initial.Revision));
        await scheduler.TriggerJobPreviewAsync("job", Tenant);
        Assert.Equal(SchedulerErrors.CodeConflict, (await Assert.ThrowsAsync<SchedulerAdapterException>(() => scheduler.DeleteJobAsync("job", Tenant, initial.Uuid, initial.Revision))).Code);
        var current = (await scheduler.GetJobAsync("job", Tenant))!;
        Assert.True(await scheduler.DeleteJobAsync("job", Tenant, current.Uuid, current.Revision));
        await scheduler.ExecuteDueAsync(current, initial.NextRunAt.AddSeconds(1), CancellationToken.None);
        Assert.Equal(1, count); Assert.Null(await scheduler.GetJobAsync("job", Tenant));
        Assert.False(await scheduler.TriggerJobPreviewAsync("job", Tenant)); Assert.Empty(await scheduler.ListJobsAsync(Tenant));
    }

    private static JobSpec Spec() => new("job", "plugin", "crm", ScheduleType.Once, DateTime.UtcNow.AddHours(1).ToString("O"), JobId: "job", TenantUuid: Tenant, Topic: "topic");
}
