using System.Net;
using System.Text.Json;
using PowerXPlugin.Framework.IAM;
using PowerXPlugin.Framework.IAM.Contracts;
using PowerXPlugin.Framework.IAM.Delegated;
using PowerXPlugin.Framework.IAM.Models;
using PowerXPlugin.Framework.Runtime.WSBus;
using Xunit;

namespace PowerXPlugin.Framework.Tests.IAM;

public sealed class WSRevalidationFailureTests
{
    private const string Plugin = "com.example.ws.failure";
    private const string Handle = "pxwsi_v1_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
    private sealed class Handler(Func<HttpRequestMessage, Task<HttpResponseMessage>> send) : HttpMessageHandler
    {
        protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken ct) => send(request);
    }
    [Theory]
    [InlineData(401, "IAM_CONTEXT_USER_SESSION_INVALID")]
    [InlineData(403, "IAM_CONTEXT_USER_FORBIDDEN")]
    [InlineData(403, "IAM_CONTEXT_SERVICE_FORBIDDEN")]
    [InlineData(409, "IAM_CONTEXT_POLICY_CHANGED")]
    [InlineData(503, "IAM_CONTEXT_UNAVAILABLE")]
    public async Task Errors_preserve_status_and_diagnostics_without_exposing_handle(int status, string code)
    {
        foreach (var scheme in new[] { "ApiKey", "Bearer" })
        {
            var calls = 0;
            var client = new PowerXIamClient(new() { BaseUrl = "https://core.example", PluginId = Plugin, AuthScheme = scheme, Credential = "fixture-service" }, new HttpClient(new Handler(async request =>
            {
                calls++;
                Assert.Equal(scheme + " fixture-service", request.Headers.Authorization!.ToString());
                using var body = JsonDocument.Parse(await request.Content!.ReadAsStringAsync());
                Assert.Equal("revalidate_ws_identity", body.RootElement.GetProperty("payload").GetProperty("body").GetProperty("operation").GetString());
                var response = new HttpResponseMessage((HttpStatusCode)status)
                {
                    Content = new StringContent(JsonSerializer.Serialize(new { code = status, error_code = code, reason_code = code, message = Handle, source_trace_id = "ws-source-trace", source_request_id = "ws-source-request" }))
                };
                response.Headers.Add("X-Trace-ID", "ws-trace");
                response.Headers.Add("X-Request-ID", "ws-request");
                return response;
            })));
            var error = await Assert.ThrowsAsync<IdentityContextException>(() => client.RevalidateWSIdentity(new() { Handle = Handle }));
            Assert.Equal(status, error.Status); Assert.Equal(code, error.Code);
            Assert.Equal("ws-trace", error.TraceId); Assert.Equal("ws-request", error.RequestId);
            Assert.Equal("ws-source-trace", error.SourceTraceId); Assert.Equal("ws-source-request", error.SourceRequestId);
            Assert.DoesNotContain(Handle, error.ToString()); Assert.Equal(1, calls);
        }
    }
    [Fact]
    public void Query_serialization_cannot_expose_server_handle()
    {
        var query = new WSIdentityQuery { Handle = Handle };
        Assert.DoesNotContain(Handle, JsonSerializer.Serialize(query));
        Assert.DoesNotContain(Handle, query.ToString());
    }
    [Fact]
    public async Task Missing_handle_is_rejected_before_service_request()
    {
        var calls = 0;
        var client = new PowerXIamClient(new() { BaseUrl = "https://core.example", PluginId = Plugin, AuthScheme = "ApiKey", Credential = "fixture-service" }, new HttpClient(new Handler(_ =>
        {
            calls++;
            throw new InvalidOperationException("HTTP_MUST_NOT_RUN");
        })));
        var error = await Assert.ThrowsAsync<IAMAdapterException>(() => client.RevalidateWSIdentity(new()));
        Assert.Equal("IAM_CONTEXT_WS_HANDLE_REQUIRED", error.Code); Assert.Equal(0, calls);
    }
    private sealed class EmptyRevalidator : IWSIdentityRevalidationService
    {
        public Task<WSIdentity> RevalidateWSIdentity(WSIdentityQuery query, CancellationToken ct = default) => Task.FromResult<WSIdentity>(null!);
    }
    [Fact]
    public async Task Empty_identity_fails_closed_with_a_machine_error()
    {
        var expires = DateTime.UtcNow.AddMinutes(2).ToString("O");
        var identity = new DelegatedIdentity
        {
            Schema = "powerx.iam.plugin-request-identity/v1", Mode = "delegated", PluginId = Plugin,
            TenantUuid = "11111111-1111-4111-8111-111111111111", UserUuid = "22222222-2222-4222-8222-222222222222", MemberUuid = null,
            IsRoot = true, Roles = [], Permissions = [], PolicyVersion = "fixture-policy", PermsHash = "fixture-hash",
            SourceMethod = "GET", SourcePath = "/ws", SourceRequiredPermission = "runtime.realtime:read", Transport = "ws_upgrade",
            SourceRequestId = "source-request", SourceTraceId = "source-trace", TraceId = "identity-trace",
            ProofExpiresAt = DateTime.UtcNow.AddSeconds(30).ToString("O"), UserSessionExpiresAt = expires,
            WSRevalidationHandle = Handle, WSRevalidationExpiresAt = expires
        };
        using var lifecycle = new IdentityLifecycle();
        string? closedReason = null;
        using var lease = lifecycle.Open(identity, Plugin, TimeSpan.FromMinutes(1), lifecycle.Revision, (_, _) => true, reason => closedReason = reason, new EmptyRevalidator());
        var error = await Assert.ThrowsAsync<IAMAdapterException>(() => lease.AuthorizeTopicAsync("_topic.fixture"));
        Assert.Equal("IAM_CONTEXT_RESPONSE_INVALID", error.Code);
        Assert.Equal(error.Code, closedReason); Assert.True(lease.CancellationToken.IsCancellationRequested);
    }
}
