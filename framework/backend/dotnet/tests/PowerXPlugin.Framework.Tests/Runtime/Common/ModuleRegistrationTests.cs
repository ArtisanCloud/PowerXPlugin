using Microsoft.Extensions.DependencyInjection;
using PowerXPlugin.Framework.IAM;
using PowerXPlugin.Framework.IAM.Contracts;
using PowerXPlugin.Framework.IAM.Models;
using PowerXPlugin.Framework.Runtime.Common;
using PowerXPlugin.Framework.Runtime.Integration;
using PowerXPlugin.Framework.Runtime.Scheduler;
using PowerXPlugin.Framework.Runtime.TaskQueue;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.Common;

public sealed class ModuleRegistrationTests
{
    [Fact]
    public void Scheduler_local_does_not_evaluate_delegated_factory()
    {
        var called = false;
        var services = new ServiceCollection().AddLogging();
        services.AddPowerXScheduler(ProviderMode.Local, _ => { called = true; throw new InvalidOperationException(); });
        using var provider = services.BuildServiceProvider();
        Assert.IsType<LocalScheduler>(provider.GetRequiredService<IScheduler>());
        Assert.False(called);
    }

    [Fact]
    public void Task_queue_delegated_does_not_construct_local_queue()
    {
        var delegated = new StubTaskQueue();
        var services = new ServiceCollection();
        services.AddPowerXTaskQueue(ProviderMode.Delegated, _ => new LocalTaskQueueAdapter(new InMemoryTaskQueue()), _ => delegated);
        using var provider = services.BuildServiceProvider();
        Assert.Same(delegated, provider.GetRequiredService<ITaskQueue>());
        Assert.Null(provider.GetService<InMemoryTaskQueue>());
    }

    [Fact]
    public void Integration_local_does_not_evaluate_delegated_factory()
    {
        var called = false;
        var local = new StubIntegrationGateway();
        var services = new ServiceCollection();
        services.AddPowerXIntegrationRuntime(ProviderMode.Local, _ => local, _ => { called = true; return new StubIntegrationGateway(); });
        using var provider = services.BuildServiceProvider();
        Assert.Same(local, provider.GetRequiredService<IIntegrationGateway>());
        Assert.False(called);
    }

    [Fact]
    public void Iam_delegated_does_not_evaluate_local_bundle()
    {
        var localCalled = false;
        var delegated = Bundle();
        var services = new ServiceCollection();
        services.AddPowerXIamRegistry(ProviderMode.Delegated, _ => { localCalled = true; return Bundle(); }, _ => delegated);
        using var provider = services.BuildServiceProvider();
        Assert.Equal(IAMAdapterMode.Delegated, provider.GetRequiredService<IAMRegistry>().Mode);
        Assert.False(localCalled);
    }

    private static IAMAdapterBundle Bundle() => new(new DirectoryStub(), new AuthzStub(), new IdentityStub());
    private sealed class DirectoryStub : IDirectoryService { public Task<Tenant?> GetTenant(string tenantUuid, CancellationToken ct = default) => Task.FromResult<Tenant?>(null); public Task<IReadOnlyList<Department>> ListDepartments(string tenantUuid, CancellationToken ct = default) => Task.FromResult<IReadOnlyList<Department>>([]); public Task<IReadOnlyList<Member>> ListMembers(string tenantUuid, CancellationToken ct = default) => Task.FromResult<IReadOnlyList<Member>>([]); public Task<IReadOnlyList<Role>> ListRoles(string tenantUuid, CancellationToken ct = default) => Task.FromResult<IReadOnlyList<Role>>([]); public Task<IReadOnlyList<Permission>> ListPermissions(string tenantUuid, CancellationToken ct = default) => Task.FromResult<IReadOnlyList<Permission>>([]); }
    private sealed class AuthzStub : IAuthzService { public Task<AuthorizationDecision?> AuthorizeAsync(AuthorizationRequest request, CancellationToken ct = default) => Task.FromResult<AuthorizationDecision?>(null); }
    private sealed class IdentityStub : IIdentityContextService { public Task<IdentityContext?> ResolveIdentity(string? bearerToken, CancellationToken ct = default) => Task.FromResult<IdentityContext?>(null); }
    private sealed class StubTaskQueue : ITaskQueue { public Task EnqueueAsync(TaskQueueMessage message, CancellationToken ct = default) => Task.CompletedTask; public Task<IReadOnlyList<TaskQueueMessage>> DequeueAsync(TaskQueueDequeueRequest request, CancellationToken ct = default) => Task.FromResult<IReadOnlyList<TaskQueueMessage>>([]); public Task AckAsync(TaskQueueAckRequest request, CancellationToken ct = default) => Task.CompletedTask; public Task NackAsync(TaskQueueNackRequest request, CancellationToken ct = default) => Task.CompletedTask; public Task RetryAsync(TaskQueueRetryRequest request, CancellationToken ct = default) => Task.CompletedTask; }
    private sealed class StubIntegrationGateway : IIntegrationGateway { public Task<IReadOnlyList<IntegrationRouteSummary>> ListRoutesAsync(IntegrationRouteQuery query, CancellationToken ct = default) => Task.FromResult<IReadOnlyList<IntegrationRouteSummary>>([]); public Task<IntegrationRouteDetail> GetRouteAsync(string routeSlug, CancellationToken ct = default) => throw new NotImplementedException(); public Task<IntegrationRouteResult> InvokeRouteAsync(string routeSlug, IntegrationRouteInvocation invocation, CancellationToken ct = default) => throw new NotImplementedException(); }
}
