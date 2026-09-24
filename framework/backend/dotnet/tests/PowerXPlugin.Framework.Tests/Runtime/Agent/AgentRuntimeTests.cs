using PowerXPlugin.Framework.Runtime.Agent;
using PowerXPlugin.Framework.Runtime.Common;
using System.Net;
using System.Text;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.Agent;

public sealed class AgentRuntimeTests
{
    private const string Tenant = "00000000-0000-4000-8000-000000000001";
    private const string Agent = "00000000-0000-4000-8000-000000000002";
    private const string Session = "00000000-0000-4000-8000-000000000003";
    private const string Message = "00000000-0000-4000-8000-000000000004";
    private const string Invocation = "00000000-0000-4000-8000-000000000005";
    private const string Trace = "00000000-0000-4000-8000-000000000006";

    [Fact]
    public async Task Local_rejects_cross_session_response()
    {
        var service = new LocalAgentSessionService(new CrossSessionStore());
        var error = await Assert.ThrowsAsync<AgentRuntimeException>(() => service.StartAsync(Scope, new(Message, "invoke-1")));
        Assert.Equal(AgentRuntimeErrors.InvalidResponse, error.Code);
    }

    [Fact]
    public async Task Local_rejects_second_active_invocation()
    {
        var service = new LocalAgentSessionService(new ActiveConflictStore());
        var error = await Assert.ThrowsAsync<AgentRuntimeException>(() => service.StartAsync(Scope, new(Message, "invoke-1")));
        Assert.Equal(AgentRuntimeErrors.SessionActive, error.Code);
    }

    [Fact]
    public async Task Local_accepts_idempotent_replay_for_same_message()
    {
        var service = new LocalAgentSessionService(new ReplayStore());
        var invocation = await service.StartAsync(Scope, new(Message, "invoke-1"));
        Assert.Equal(Invocation, invocation.InvocationUuid);
        Assert.Equal(AgentInvocationState.Running, invocation.State);
    }

    [Fact]
    public async Task Local_accepts_cooperative_cancellation_before_worker_finishes()
    {
        var service = new LocalAgentSessionService(new CancellingStore());
        var invocation = await service.CancelAsync(Scope, Invocation);
        Assert.Equal(AgentInvocationState.Cancelling, invocation.State);
    }

    [Fact]
    public async Task Local_rejects_non_uuid_session_scope_before_store_call()
    {
        var service = new LocalAgentSessionService(new ThrowingStore());
        var error = await Assert.ThrowsAsync<AgentRuntimeException>(() => service.GetSessionAsync(new(Tenant, "not-a-uuid")));
        Assert.Equal(AgentRuntimeErrors.InvalidArgument, error.Code);
    }

    [Fact]
    public async Task Lifecycle_rejects_health_summary_from_a_different_tenant()
    {
        var service = new LocalAgentLifecycleService(new CrossTenantLifecycleStore());
        var error = await Assert.ThrowsAsync<AgentRuntimeException>(() => service.GetHealthSummaryAsync(new(Tenant, Agent)));
        Assert.Equal(AgentRuntimeErrors.InvalidResponse, error.Code);
    }

    [Fact]
    public async Task Delegated_invocation_uses_fixed_session_route_and_idempotency_key()
    {
        HttpRequestMessage? seen = null;
        var client = new PowerXAgentSessionClient("https://core", Tenant, new StaticServiceCredentialProvider(new ServiceCredential("sts")), new HttpClient(new Handler(request =>
        {
            seen = request;
            return Json(HttpStatusCode.Accepted, "{\"code\":202,\"data\":{\"invocation_uuid\":\"" + Invocation + "\",\"session_uuid\":\"" + Session + "\",\"message_uuid\":\"" + Message + "\",\"trace_uuid\":\"" + Trace + "\",\"status\":\"running\",\"reason_code\":\"\",\"output\":\"\",\"created_at\":\"2026-01-01T00:00:00Z\",\"deadline_at\":\"2026-01-01T00:01:00Z\"}}");
        })));
        var invocation = await client.StartAsync(Scope, new(Message, "invoke-1"));
        Assert.Equal(Invocation, invocation.InvocationUuid);
        Assert.Equal($"/api/v1/tenant/agent/sessions/{Session}/invocations", seen!.RequestUri!.AbsolutePath);
        Assert.Equal("invoke-1", seen.Headers.GetValues("Idempotency-Key").Single());
        Assert.Equal("Bearer sts", seen.Headers.Authorization!.ToString());
    }

    [Fact]
    public async Task Delegated_sse_eof_without_explicit_end_is_interrupted()
    {
        var client = new PowerXAgentSessionClient("https://core", Tenant, new StaticServiceCredentialProvider(new ServiceCredential("sts")), new HttpClient(new Handler(_ =>
        {
            var response = new HttpResponseMessage(HttpStatusCode.OK) { Content = new StringContent("", Encoding.UTF8) };
            response.Content.Headers.ContentType = new("text/event-stream");
            return response;
        })));
        var error = await Assert.ThrowsAsync<AgentRuntimeException>(() => client.StreamEventsAsync(Scope, Invocation, (_, _) => Task.CompletedTask));
        Assert.Equal(AgentRuntimeErrors.StreamInterrupted, error.Code);
    }

    [Fact]
    public async Task Delegated_sse_requires_final_before_successful_end()
    {
        var stream = "id: 1\nevent: state\ndata: {\"invocation_uuid\":\"" + Invocation + "\",\"session_uuid\":\"" + Session + "\",\"message_uuid\":\"" + Message + "\",\"trace_uuid\":\"" + Trace + "\",\"status\":\"running\",\"created_at\":\"2026-01-01T00:00:00Z\",\"deadline_at\":\"2026-01-01T00:01:00Z\"}\n\n" +
                     "id: 2\nevent: final\ndata: {\"invocation_uuid\":\"" + Invocation + "\",\"session_uuid\":\"" + Session + "\",\"message_uuid\":\"" + Message + "\",\"trace_uuid\":\"" + Trace + "\",\"status\":\"succeeded\",\"created_at\":\"2026-01-01T00:00:00Z\",\"deadline_at\":\"2026-01-01T00:01:00Z\"}\n\n" +
                     "id: 3\nevent: end\ndata: {\"status\":\"succeeded\"}\n\n";
        var client = new PowerXAgentSessionClient("https://core", Tenant, new StaticServiceCredentialProvider(new ServiceCredential("sts")), new HttpClient(new Handler(_ => EventStream(stream))));
        var types = new List<string>();
        await client.StreamEventsAsync(Scope, Invocation, (ev, _) => { types.Add(ev.Type); return Task.CompletedTask; });
        Assert.Equal(["state", "final", "end"], types);
    }

    [Fact]
    public async Task Delegated_sse_reconnects_from_last_confirmed_event_id()
    {
        var calls = 0;
        var first = "id: 1\nevent: state\ndata: {\"invocation_uuid\":\"" + Invocation + "\",\"session_uuid\":\"" + Session + "\",\"message_uuid\":\"" + Message + "\",\"trace_uuid\":\"" + Trace + "\",\"status\":\"running\",\"created_at\":\"2026-01-01T00:00:00Z\",\"deadline_at\":\"2026-01-01T00:01:00Z\"}\n\n";
        var second = "id: 2\nevent: final\ndata: {\"invocation_uuid\":\"" + Invocation + "\",\"session_uuid\":\"" + Session + "\",\"message_uuid\":\"" + Message + "\",\"trace_uuid\":\"" + Trace + "\",\"status\":\"succeeded\",\"created_at\":\"2026-01-01T00:00:00Z\",\"deadline_at\":\"2026-01-01T00:01:00Z\"}\n\nid: 3\nevent: end\ndata: {\"status\":\"succeeded\"}\n\n";
        var client = new PowerXAgentSessionClient("https://core", Tenant, new StaticServiceCredentialProvider(new ServiceCredential("sts")), new HttpClient(new Handler(request =>
        {
            calls++;
            if (calls == 1) return EventStream(first);
            Assert.Equal("1", request.Headers.GetValues("Last-Event-ID").Single());
            return EventStream(second);
        })));
        var types = new List<string>();
        await client.StreamEventsAsync(Scope, Invocation, (ev, _) => { types.Add(ev.Type); return Task.CompletedTask; });
        Assert.Equal(2, calls);
        Assert.Equal(["state", "final", "end"], types);
    }

    [Fact]
    public async Task Delegated_transport_cancellation_is_not_reclassified_as_upstream_failure()
    {
        var client = new PowerXAgentSessionClient("https://core", Tenant, new StaticServiceCredentialProvider(new ServiceCredential("sts")), new HttpClient(new CancellableHandler()));
        using var cancellation = new CancellationTokenSource();
        cancellation.Cancel();
        await Assert.ThrowsAnyAsync<OperationCanceledException>(() => client.ListSessionsAsync(new(Tenant), new(), cancellation.Token));
    }

    [Fact]
    public async Task Delegated_rejects_tenant_override_before_transport()
    {
        var client = new PowerXAgentSessionClient("https://core", Tenant, new StaticServiceCredentialProvider(new ServiceCredential("sts")), new HttpClient(new Handler(_ => throw new Xunit.Sdk.XunitException("transport must not be called"))));
        var error = await Assert.ThrowsAsync<AgentRuntimeException>(() => client.ListSessionsAsync(new("00000000-0000-4000-8000-000000000099"), new()));
        Assert.Equal("FRAMEWORK_AGENT_TENANT_MISMATCH", error.Code);
    }

    private static AgentSessionScope Scope => new(Tenant, Session);
    private static AgentInvocation ValidInvocation(AgentInvocationState state, string session = Session) => new(Invocation, Tenant, session, Message, Trace, state, "invoke-1", DateTimeOffset.UtcNow, DateTimeOffset.UtcNow.AddMinutes(1));

    private abstract class StoreBase : ILocalAgentStore
    {
        public virtual Task<AgentSession> CreateSessionAsync(AgentTenantScope s, CreateAgentSessionRequest r, CancellationToken c = default) => throw new NotImplementedException();
        public virtual Task<AgentPage<AgentSession>> ListSessionsAsync(AgentTenantScope s, AgentPageRequest p, CancellationToken c = default) => throw new NotImplementedException();
        public virtual Task<AgentSession> GetSessionAsync(AgentSessionScope s, CancellationToken c = default) => throw new NotImplementedException();
        public virtual Task<AgentSession> RenameSessionAsync(AgentSessionScope s, RenameAgentSessionRequest r, CancellationToken c = default) => throw new NotImplementedException();
        public virtual Task<AgentSession> ArchiveSessionAsync(AgentSessionScope s, CancellationToken c = default) => throw new NotImplementedException();
        public virtual Task<AgentSession> DeleteSessionAsync(AgentSessionScope s, CancellationToken c = default) => throw new NotImplementedException();
        public virtual Task<AgentMessage> AppendMessageAsync(AgentSessionScope s, AppendAgentMessageRequest r, CancellationToken c = default) => throw new NotImplementedException();
        public virtual Task<AgentPage<AgentMessage>> ListMessagesAsync(AgentSessionScope s, AgentPageRequest p, CancellationToken c = default) => throw new NotImplementedException();
        public virtual Task<AgentStartResult> StartAsync(AgentSessionScope s, StartAgentInvocationRequest r, CancellationToken c = default) => throw new NotImplementedException();
        public virtual Task<AgentInvocation> GetInvocationAsync(AgentSessionScope s, string i, CancellationToken c = default) => throw new NotImplementedException();
        public virtual Task<AgentInvocation> CancelAsync(AgentSessionScope s, string i, CancellationToken c = default) => throw new NotImplementedException();
        public virtual Task StreamEventsAsync(AgentSessionScope s, string i, Func<AgentSessionEvent, CancellationToken, Task> f, CancellationToken c = default) => throw new NotImplementedException();
    }

    private sealed class CrossSessionStore : StoreBase { public override Task<AgentStartResult> StartAsync(AgentSessionScope s, StartAgentInvocationRequest r, CancellationToken c = default) => Task.FromResult(new AgentStartResult(AgentStartDisposition.Started, ValidInvocation(AgentInvocationState.Running, Agent))); }
    private sealed class ActiveConflictStore : StoreBase { public override Task<AgentStartResult> StartAsync(AgentSessionScope s, StartAgentInvocationRequest r, CancellationToken c = default) => Task.FromResult(new AgentStartResult(AgentStartDisposition.ActiveInvocationConflict, null)); }
    private sealed class ReplayStore : StoreBase { public override Task<AgentStartResult> StartAsync(AgentSessionScope s, StartAgentInvocationRequest r, CancellationToken c = default) => Task.FromResult(new AgentStartResult(AgentStartDisposition.IdempotentReplay, ValidInvocation(AgentInvocationState.Running))); }
    private sealed class CancellingStore : StoreBase { public override Task<AgentInvocation> CancelAsync(AgentSessionScope s, string i, CancellationToken c = default) => Task.FromResult(ValidInvocation(AgentInvocationState.Cancelling)); }
    private sealed class ThrowingStore : StoreBase { public override Task<AgentSession> GetSessionAsync(AgentSessionScope s, CancellationToken c = default) => throw new Xunit.Sdk.XunitException("store should not be called"); }

    private sealed class CrossTenantLifecycleStore : ILocalAgentLifecycleStore
    {
        public Task<AgentHealthSummary> GetHealthSummaryAsync(AgentLifecycleScope s, CancellationToken c = default) => Task.FromResult(new AgentHealthSummary(Agent, Tenant, "healthy", 100, DateTimeOffset.UtcNow, 60, new(1, 1, 1, 1, 0), [], []));
        public Task<AgentHealthHistory> ListHealthHistoryAsync(AgentLifecycleScope s, int h, int l, CancellationToken c = default) => throw new NotImplementedException();
        public Task<object> GetBridgeStateAsync(AgentLifecycleScope s, int l, CancellationToken c = default) => throw new NotImplementedException();
        public Task<AgentBridgeLifecycleResult> FreezeAsync(AgentLifecycleScope s, AgentBridgeControlRequest r, CancellationToken c = default) => throw new NotImplementedException();
        public Task<AgentBridgeLifecycleResult> RecoverAsync(AgentLifecycleScope s, AgentBridgeControlRequest r, CancellationToken c = default) => throw new NotImplementedException();
        public Task<AgentBridgeLifecycleResult> RebalanceAsync(AgentLifecycleScope s, AgentBridgeRebalanceRequest r, CancellationToken c = default) => throw new NotImplementedException();
    }

    private sealed class Handler(Func<HttpRequestMessage, HttpResponseMessage> handle) : HttpMessageHandler { protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken) => Task.FromResult(handle(request)); }
    private sealed class CancellableHandler : HttpMessageHandler { protected override async Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken) { await Task.Delay(Timeout.InfiniteTimeSpan, cancellationToken); throw new InvalidOperationException(); } }
    private static HttpResponseMessage Json(HttpStatusCode status, string value) => new(status) { Content = new StringContent(value, Encoding.UTF8, "application/json") };
    private static HttpResponseMessage EventStream(string value) { var response = new HttpResponseMessage(HttpStatusCode.OK) { Content = new StringContent(value, Encoding.UTF8) }; response.Content.Headers.ContentType = new("text/event-stream"); return response; }
}
