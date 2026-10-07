using PowerXPlugin.Framework.Runtime.Scheduler;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.Scheduler;

public sealed class LocalSchedulerTests
{
    private const string Tenant = "00000000-0000-4000-8000-000000000001";
    private const string OtherTenant = "00000000-0000-4000-8000-000000000002";

    [Fact]
    public async Task Another_tenant_cannot_read_update_pause_resume_or_trigger_a_job()
    {
        var scheduler = new LocalScheduler();
        var job = await scheduler.CreateJobAsync(Spec(Tenant));
        var scheduledAt = job.NextRunAt;

        Assert.Null(await scheduler.GetJobAsync(job.JobId, OtherTenant));
        Assert.Null(await scheduler.UpdateJobAsync(Spec(OtherTenant) with { Name = "changed" }));
        Assert.False(await scheduler.PauseJobAsync(job.JobId, OtherTenant));
        Assert.False(await scheduler.ResumeJobAsync(job.JobId, OtherTenant));
        Assert.False(await scheduler.TriggerJobAsync(job.JobId, OtherTenant));
        Assert.Empty(await scheduler.ListJobsAsync(OtherTenant));
        Assert.Equal("reclaim", job.Name);
        Assert.Equal(JobStatus.Active, job.Status);
        Assert.Equal(scheduledAt, job.NextRunAt);
        Assert.Equal(job.Uuid, (await scheduler.GetJobAsync(job.JobId, Tenant))!.Uuid);
    }

    [Fact]
    public async Task Same_local_job_name_can_exist_independently_for_two_tenants()
    {
        var scheduler = new LocalScheduler();
        var first = await scheduler.CreateJobAsync(Spec(Tenant));
        var second = await scheduler.CreateJobAsync(Spec(OtherTenant));

        Assert.NotEqual(first.Uuid, second.Uuid);
        Assert.Equal(first.Uuid, (await scheduler.GetJobAsync(first.JobId, Tenant))!.Uuid);
        Assert.Equal(second.Uuid, (await scheduler.GetJobAsync(second.JobId, OtherTenant))!.Uuid);
        Assert.True(await scheduler.PauseJobAsync(first.JobId, Tenant));
        first = (await scheduler.GetJobAsync(first.JobId, Tenant))!;
        Assert.Equal(JobStatus.Paused, first.Status);
        Assert.Equal(JobStatus.Active, second.Status);
        Assert.NotNull(await scheduler.UpdateJobAsync(Spec(OtherTenant) with { Name = "other-reclaim" }));
        second = (await scheduler.GetJobAsync(second.JobId, OtherTenant))!;
        Assert.Equal("reclaim", first.Name);
        Assert.Equal("other-reclaim", second.Name);
        Assert.Single(await scheduler.ListJobsAsync(Tenant));
        Assert.Single(await scheduler.ListJobsAsync(OtherTenant));
        Assert.Equal(2, (await scheduler.ListJobsAsync()).Count);
    }

    [Theory]
    [InlineData(null)]
    [InlineData("")]
    [InlineData("invalid")]
    [InlineData("00000000-0000-0000-0000-000000000000")]
    public async Task Missing_or_invalid_tenant_never_becomes_an_unscoped_job_operation(string? tenant)
    {
        var scheduler = new LocalScheduler();
        var job = await scheduler.CreateJobAsync(Spec(Tenant));
        Func<Task>[] operations =
        [
            () => scheduler.CreateJobAsync(Spec(tenant)),
            () => scheduler.UpdateJobAsync(Spec(tenant)),
            () => scheduler.GetJobAsync(job.JobId, tenant!),
            () => scheduler.PauseJobAsync(job.JobId, tenant!),
            () => scheduler.ResumeJobAsync(job.JobId, tenant!),
            () => scheduler.TriggerJobAsync(job.JobId, tenant!)
        ];
        foreach (var operation in operations)
        {
            var error = await Assert.ThrowsAsync<SchedulerAdapterException>(operation);
            Assert.Equal(SchedulerErrors.CodeInvalidJob, error.Code);
        }
        if (tenant != null)
        {
            var error = await Assert.ThrowsAsync<SchedulerAdapterException>(() => scheduler.ListJobsAsync(tenant));
            Assert.Equal(SchedulerErrors.CodeInvalidJob, error.Code);
        }
        Assert.Single(await scheduler.ListJobsAsync(Tenant));
    }

    private static JobSpec Spec(string? tenant) => new(
        "reclaim", "plugin", "plugin-a", ScheduleType.Interval, "01:00:00",
        JobId: "pool-reclaim", TenantUuid: tenant);
}
