using System.Net;
using PowerXPlugin.Framework.IAM;
using PowerXPlugin.Framework.IAM.Delegated;
using PowerXPlugin.Framework.Runtime.Integration;
using PowerXPlugin.Framework.Runtime.Integration.Delegated;
using PowerXPlugin.Framework.Runtime.Scheduler;
using PowerXPlugin.Framework.Runtime.Scheduler.Delegated;
using PowerXPlugin.Framework.Runtime.TaskQueue;
using PowerXPlugin.Framework.Runtime.TaskQueue.Delegated;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.Common;

public sealed class DelegatedFailureTests
{
    [Fact]
    public async Task Scheduler_maps_unauthorized_and_missing_configuration()
    {
        var missing = new PowerXSchedulerClient(new PowerXSchedulerClientOptions(), new HttpClient(new StatusHandler(HttpStatusCode.OK)));
        var unavailable = await Assert.ThrowsAsync<SchedulerAdapterException>(() => missing.GetJobAsync("job-1", "tenant-a"));
        Assert.Equal(SchedulerErrors.CodeUnavailable, unavailable.Code);

        var client = new PowerXSchedulerClient(Options(), new HttpClient(new StatusHandler(HttpStatusCode.Unauthorized)));
        var unauthorized = await Assert.ThrowsAsync<SchedulerAdapterException>(() => client.GetJobAsync("job-1", "tenant-a"));
        Assert.Equal(SchedulerErrors.CodeUnauthorized, unauthorized.Code);
    }

    [Fact]
    public async Task Task_queue_rejects_cross_tenant_and_maps_forbidden()
    {
        var client = new PowerXTaskQueueClient(new PowerXTaskQueueClientOptions { BaseUrl = "https://core.example", Credential = "sts", TenantUuid = "tenant-a" }, new HttpClient(new StatusHandler(HttpStatusCode.Forbidden)));
        var mismatch = await Assert.ThrowsAsync<TaskQueueAdapterException>(() => client.DequeueAsync(new TaskQueueDequeueRequest("tenant-b", "subscriber")));
        Assert.Equal(TaskQueueErrors.CodeTenantMismatch, mismatch.Code);
        var forbidden = await Assert.ThrowsAsync<TaskQueueAdapterException>(() => client.DequeueAsync(new TaskQueueDequeueRequest("tenant-a", "subscriber")));
        Assert.Equal(TaskQueueErrors.CodeForbidden, forbidden.Code);
    }

    [Fact]
    public async Task Integration_maps_forbidden_and_iam_never_forwards_user_bearer()
    {
        var integration = new PowerXIntegrationGatewayClient(new PowerXIntegrationGatewayClientOptions { BaseUrl = "https://core.example" }, new PowerXPlugin.Framework.Runtime.Common.StaticServiceCredentialProvider(new PowerXPlugin.Framework.Runtime.Common.ServiceCredential("sts")), new HttpClient(new StatusHandler(HttpStatusCode.Forbidden)));
        var forbidden = await Assert.ThrowsAsync<IntegrationAdapterException>(() => integration.GetRouteAsync("crm-order"));
        Assert.Equal(IntegrationErrors.CodeForbidden, forbidden.Code);

        var iam = new PowerXIamClient(new PowerXIamClientOptions { BaseUrl = "https://core.example", Credential = "service-sts" }, new HttpClient(new ThrowingHandler()));
        var blocked = await Assert.ThrowsAsync<IAMAdapterException>(() => iam.ResolveIdentity("incoming-user-bearer"));
        Assert.Equal(IAMErrors.CodeIdentityDelegationUnavailable, blocked.Code);

        var deniedIam = new PowerXIamClient(new PowerXIamClientOptions { BaseUrl = "https://core.example", Credential = "service-sts" }, new HttpClient(new StatusHandler(HttpStatusCode.Forbidden)));
        var iamForbidden = await Assert.ThrowsAsync<IAMAdapterException>(() => deniedIam.GetTenant("tenant-a"));
        Assert.Equal(IAMErrors.CodeForbidden, iamForbidden.Code);
    }

    private static PowerXSchedulerClientOptions Options() => new() { BaseUrl = "https://core.example", Credential = "sts" };
    private sealed class StatusHandler(HttpStatusCode status) : HttpMessageHandler { protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken ct) => Task.FromResult(new HttpResponseMessage(status)); }
    private sealed class ThrowingHandler : HttpMessageHandler { protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken ct) => throw new Xunit.Sdk.XunitException("Identity delegation must not send a request."); }
}
