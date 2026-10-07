using System.Net;
using System.Text;
using PowerXPlugin.Framework.Runtime.Agent;
using PowerXPlugin.Framework.Runtime.Common;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.Agent;

public sealed class PowerXAgentLifecycleClientTests
{
    private const string Tenant = "00000000-0000-4000-8000-000000000001";
    private const string Other = "00000000-0000-4000-8000-000000000099";
    private const string Agent = "00000000-0000-4000-8000-000000000002";
    private const string Event = "00000000-0000-4000-8000-000000000003";
    private const string Trace = "00000000-0000-4000-8000-000000000004";
    private const string Root = "/api/v1/openapi/agents/" + Agent;
    private const string Health = "{\"status\":\"healthy\",\"health_score\":95,\"updated_at\":\"2026-01-01T00:00:00Z\",\"window_duration_sec\":60,\"metrics\":{\"throughput_per_min\":2.5,\"success_rate\":0.9,\"p95_latency_ms\":100,\"resource_util_pct\":20,\"error_rate\":0.1}}";
    private const string AgentWire = "{\"id\":\"" + Agent + "\",\"tenant_uuid\":\"" + Tenant + "\",\"alias\":\"agent\",\"status\":\"active\",\"capacity\":{\"default\":2,\"current\":3,\"max\":5},\"updated_at\":\"2026-01-01T00:00:00Z\"}";
    private const string EventWire = "{\"id\":\"" + Event + "\",\"type\":\"freeze\",\"from_status\":\"active\",\"to_status\":\"frozen\",\"trace_id\":\"" + Trace + "\",\"occurred_at\":\"2026-01-01T00:00:00Z\"}";

    [Fact]
    public async Task Reads_summary_history_and_state_using_core_contract()
    {
        var seen = new List<string>();
        var client = Client(request =>
        {
            Assert.Equal("Bearer sts", request.Headers.Authorization?.ToString());
            Assert.False(request.Headers.Contains("tenant_uuid"));
            seen.Add(request.RequestUri!.PathAndQuery);
            return Response(request.RequestUri.PathAndQuery.Contains("history") ? "{\"snapshots\":[" + Health + "]}"
                : request.RequestUri.PathAndQuery.Contains("state") ? "{\"agent\":" + AgentWire + ",\"health\":" + Health + ",\"events\":[]}" : Health);
        });
        var scope = new AgentLifecycleScope(Tenant, Agent);
        Assert.Equal(95, (await client.GetHealthSummaryAsync(scope)).HealthScore);
        Assert.Single((await client.ListHealthHistoryAsync(scope, 24, 5)).Snapshots);
        var state = await client.GetBridgeStateAsync(scope, 7);
        Assert.Equal(Agent, state.Agent.AgentUuid);
        Assert.Equal(3, state.Capacity.Current);
        Assert.Equal([Root + "/health/summary", Root + "/health/history?range_hours=24&limit=5", Root + "/bridge/state?limit=7"], seen);
    }

    [Fact]
    public async Task Bridge_state_can_have_no_health_snapshot_yet()
    {
        var client = Client(_ => Response("{\"agent\":" + AgentWire + ",\"health\":{\"status\":\"\",\"updated_at\":\"\",\"metrics\":{}},\"events\":null}"));
        var state = await client.GetBridgeStateAsync(new(Tenant, Agent), 10);
        Assert.Null(state.Health);
        Assert.Empty(state.Events);
    }

    [Theory]
    [InlineData("freeze")]
    [InlineData("recover")]
    [InlineData("rebalance")]
    public async Task Controls_use_sts_and_fixed_core_body(string action)
    {
        var calls = 0;
        var client = Client(request =>
        {
            calls++;
            Assert.Equal(HttpMethod.Post, request.Method);
            Assert.Equal(Root + "/bridge/" + action, request.RequestUri!.AbsolutePath);
            Assert.Equal("Bearer sts", request.Headers.Authorization?.ToString());
            var body = request.Content!.ReadAsStringAsync().Result;
            Assert.Contains("\"trace_id\":\"" + Trace + "\"", body);
            Assert.DoesNotContain("tenant_uuid", body);
            if (action == "rebalance") Assert.Contains("\"target_capacity_instances\":4", body);
            return Response("{\"agent\":" + AgentWire + ",\"event\":" + EventWire + "}");
        });
        var scope = new AgentLifecycleScope(Tenant, Agent);
        var result = action switch
        {
            "freeze" => await client.FreezeAsync(scope, new("reason", Trace)),
            "recover" => await client.RecoverAsync(scope, new("reason", Trace)),
            _ => await client.RebalanceAsync(scope, new(4, "reason", Trace))
        };
        Assert.Equal(Event, result.Event?.Id);
        Assert.Equal(1, calls);
    }

    [Fact]
    public async Task Tenant_mismatch_is_rejected_before_credential_or_http()
    {
        var client = Client(_ => throw new Xunit.Sdk.XunitException("HTTP must not be called"));
        var error = await Assert.ThrowsAsync<AgentRuntimeException>(() => client.GetHealthSummaryAsync(new(Other, Agent)));
        Assert.Equal("FRAMEWORK_AGENT_TENANT_MISMATCH", error.Code);
    }

    [Fact]
    public async Task Cross_tenant_core_response_is_rejected()
    {
        var client = Client(_ => Response("{\"agent\":" + AgentWire.Replace(Tenant, Other) + ",\"event\":" + EventWire + "}"));
        var error = await Assert.ThrowsAsync<AgentRuntimeException>(() => client.FreezeAsync(new(Tenant, Agent), new()));
        Assert.Equal(AgentRuntimeErrors.InvalidResponse, error.Code);
    }

    [Fact]
    public async Task Core_forbidden_fails_without_local_fallback()
    {
        var client = Client(_ => new HttpResponseMessage(HttpStatusCode.Forbidden));
        var error = await Assert.ThrowsAsync<AgentRuntimeException>(() => client.GetHealthSummaryAsync(new(Tenant, Agent)));
        Assert.Equal("FRAMEWORK_AGENT_FORBIDDEN", error.Code);
    }

    [Fact]
    public async Task Api_key_is_rejected_for_sts_only_contract()
    {
        var client = new PowerXAgentLifecycleClient("https://core", Tenant,
            new StaticServiceCredentialProvider(new ServiceCredential("key", "ApiKey")),
            new HttpClient(new Handler(_ => throw new Xunit.Sdk.XunitException("HTTP must not be called"))));
        var error = await Assert.ThrowsAsync<AgentRuntimeException>(() => client.GetHealthSummaryAsync(new(Tenant, Agent)));
        Assert.Equal("FRAMEWORK_AGENT_SERVICE_ACTOR_REQUIRED", error.Code);
    }

    [Fact]
    public async Task Invalid_arguments_and_malformed_payload_fail_closed()
    {
        var client = Client(_ => Response("{\"snapshots\":[{}]}"));
        var scope = new AgentLifecycleScope(Tenant, Agent);
        var invalid = await Assert.ThrowsAsync<AgentRuntimeException>(() => client.GetBridgeStateAsync(scope, 1001));
        Assert.Equal(AgentRuntimeErrors.InvalidArgument, invalid.Code);
        var malformed = await Assert.ThrowsAsync<AgentRuntimeException>(() => client.ListHealthHistoryAsync(scope, 24, 5));
        Assert.Equal(AgentRuntimeErrors.InvalidResponse, malformed.Code);
    }

    [Fact]
    public async Task Cancellation_remains_operation_cancelled()
    {
        var client = new PowerXAgentLifecycleClient("https://core", Tenant,
            new StaticServiceCredentialProvider(new ServiceCredential("sts")), new HttpClient(new CancellingHandler()));
        using var cancellation = new CancellationTokenSource();
        cancellation.Cancel();
        await Assert.ThrowsAnyAsync<OperationCanceledException>(() => client.GetHealthSummaryAsync(new(Tenant, Agent), cancellation.Token));
    }

    private static PowerXAgentLifecycleClient Client(Func<HttpRequestMessage, HttpResponseMessage> handle) =>
        new("https://core", Tenant, new StaticServiceCredentialProvider(new ServiceCredential("sts")), new HttpClient(new Handler(handle)));
    private static HttpResponseMessage Response(string data) => new(HttpStatusCode.OK)
    {
        Content = new StringContent("{\"code\":200,\"data\":" + data + "}", Encoding.UTF8, "application/json")
    };
    private sealed class Handler(Func<HttpRequestMessage, HttpResponseMessage> handle) : HttpMessageHandler
    {
        protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken) => Task.FromResult(handle(request));
    }
    private sealed class CancellingHandler : HttpMessageHandler
    {
        protected override async Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken)
        {
            await Task.Delay(Timeout.InfiniteTimeSpan, cancellationToken);
            throw new InvalidOperationException();
        }
    }
}
