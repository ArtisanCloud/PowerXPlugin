using System.Net;
using System.Text.Json;
using System.Text.Json.Nodes;
using PowerXPlugin.Framework.IAM;
using PowerXPlugin.Framework.IAM.Contracts;
using PowerXPlugin.Framework.IAM.Delegated;
using PowerXPlugin.Framework.IAM.Models;
using PowerXPlugin.Framework.Runtime.Capability;
using PowerXPlugin.Framework.Runtime.Common;
using Xunit;

namespace PowerXPlugin.Framework.Tests.IAM;

public sealed class AuthorizationBaselineTests
{
    private const string Plugin = "com.powerx.fixture";
    private static string Fixture() => File.ReadAllText(Path.Combine(AppContext.BaseDirectory, "Fixtures/authorization_baseline_result.json"));
    private static AuthorizationBaselineQuery Query() => new("20000000-0000-4000-8000-000000000001",
        ["10000000-0000-4000-8000-000000000001"], ["orders:read", "cost:read", "scope:read"]);
    private sealed class Handler(Func<HttpRequestMessage, CancellationToken, Task<HttpResponseMessage>> call) : HttpMessageHandler
    {
        protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken ct) => call(request, ct);
    }

    [Fact]
    public void Same_Go_fixture_preserves_nullable_branches_and_microseconds()
    {
        using var document = JsonDocument.Parse(Fixture());
        var result = AuthorizationBaselineContract.Decode(document.RootElement, Query(), Plugin);
        Assert.Null(result.Items[0].GrantedAt);
        Assert.True(result.Items[0].Allowed);
        Assert.False(result.Items[1].Allowed);
        Assert.Null(result.Items[2].Allowed);
        Assert.Equal(10, result.Items[0].VisibleSince!.Value.Ticks % TimeSpan.TicksPerSecond);
        using var roundtrip = JsonDocument.Parse(JsonSerializer.Serialize(result));
        _ = AuthorizationBaselineContract.Decode(roundtrip.RootElement, Query(), Plugin);
    }

    [Theory]
    [InlineData("schema")]
    [InlineData("tenant_uuid")]
    [InlineData("plugin_id")]
    [InlineData("missing")]
    [InlineData("duplicate")]
    [InlineData("error_allowed")]
    [InlineData("missing_allowed")]
    [InlineData("null_allowed")]
    [InlineData("native_null")]
    [InlineData("milliseconds")]
    [InlineData("offset")]
    [InlineData("extra")]
    [InlineData("missing_nullable")]
    public void Invalid_results_fail_as_whole_contract(string scenario)
    {
        var root = JsonNode.Parse(Fixture())!.AsObject();
        var items = root["items"]!.AsArray();
        switch (scenario)
        {
            case "schema": case "tenant_uuid": case "plugin_id": root[scenario] = "invalid"; break;
            case "missing": items.RemoveAt(2); break;
            case "duplicate": items[2] = items[0]!.DeepClone(); break;
            case "error_allowed": items[2]!["allowed"] = false; break;
            case "missing_allowed": items[1]!.AsObject().Remove("allowed"); break;
            case "null_allowed": items[1]!["allowed"] = null; break;
            case "native_null": items[0]!["baseline_kind"] = "native"; break;
            case "milliseconds": root["evaluated_at"] = "2026-10-15T02:15:00.001Z"; break;
            case "offset": root["evaluated_at"] = "2026-10-15T10:15:00.000001+08:00"; break;
            case "extra": root["extra"] = false; break;
            case "missing_nullable": items[0]!.AsObject().Remove("valid_until"); break;
        }
        using var document = JsonDocument.Parse(root.ToJsonString());
        Assert.Equal("IAM_BASELINE_RESPONSE_INVALID", Assert.Throws<IAMAdapterException>(() => AuthorizationBaselineContract.Decode(document.RootElement, Query(), Plugin)).Code);
    }

    [Theory]
    [InlineData("ApiKey")]
    [InlineData("Bearer")]
    public async Task Typed_transport_uses_only_service_scope_and_optional_registry(string scheme)
    {
        var client = new PowerXIamClient(new() { BaseUrl = "https://core.example", PluginId = Plugin, Credential = "server-owned", AuthScheme = scheme },
            new HttpClient(new Handler(async (request, ct) => {
                Assert.Equal(scheme + " server-owned", request.Headers.Authorization!.ToString());
                Assert.Equal("/api/v1/tenant/invocations", request.RequestUri!.AbsolutePath);
                using var input = JsonDocument.Parse(await request.Content!.ReadAsStringAsync(ct));
                var root = input.RootElement;
                Assert.Equal(AuthorizationBaselineContract.CapabilityId, root.GetProperty("capability_id").GetString());
                var body = root.GetProperty("payload").GetProperty("body");
                Assert.Equal(AuthorizationBaselineContract.Operation, body.GetProperty("operation").GetString());
                Assert.Equal(3, body.GetProperty("request").EnumerateObject().Count());
                using var result = JsonDocument.Parse(Fixture());
                return new(HttpStatusCode.OK) { Content = new StringContent(JsonSerializer.Serialize(new { code = 200, data = new { result = new { result = result.RootElement } } })) };
            })));
        var existing = IAMRuntime.CreateRegistry(ProviderMode.Delegated, new(client, client, client));
        Assert.Throws<IAMAdapterException>(() => existing.GetAuthorizationBaselines());
        await existing.RequireAuthorizationBaselineFeatureAsync(false, [], null);
        var registry = IAMRuntime.CreateRegistry(ProviderMode.Delegated, new(client, client, client, client, Plugin));
        var result = await registry.GetAuthorizationBaselines().ReadMemberBaselines(Query());
        Assert.Equal(3, result.Items.Count);
        string[] required = [AuthorizationBaselineContract.CapabilityId, "com.corex.iam.directory.read", "com.corex.iam.identity_context.service_read", "com.corex.capabilities.grant_status.read"];
        await registry.RequireAuthorizationBaselineFeatureAsync(true, required, new Grants("granted"));
        await Assert.ThrowsAsync<CapabilityGrantException>(() => registry.RequireAuthorizationBaselineFeatureAsync(true, required, new Grants("not_granted")));
        Assert.Equal("IAM_BASELINE_REQUIREMENTS_INVALID", (await Assert.ThrowsAsync<IAMAdapterException>(() => registry.RequireAuthorizationBaselineFeatureAsync(true, [AuthorizationBaselineContract.CapabilityId], new Grants("granted")))).Code);
        Assert.Equal("IAM_BASELINE_REQUIREMENTS_INVALID", (await Assert.ThrowsAsync<IAMAdapterException>(() => registry.RequireAuthorizationBaselineFeatureAsync(true, [], new Grants("granted")))).Code);
    }

    private sealed class Grants(string status) : ICapabilityGrantRegistry
    {
        public Task<IReadOnlyList<CapabilityGrantStatus>> GetGrantStatusAsync(IReadOnlyList<string> ids, CancellationToken ct = default)
            => Task.FromResult<IReadOnlyList<CapabilityGrantStatus>>(ids.Select(id => new CapabilityGrantStatus(id, status, "CAPABILITY_GRANT_STATUS")).ToArray());
    }

    [Theory]
    [InlineData(400)] [InlineData(401)] [InlineData(403)] [InlineData(503)] [InlineData(200)]
    public async Task Errors_preserve_status_code_and_trace(int status)
    {
        var client = new PowerXIamClient(new() { BaseUrl = "https://core.example", PluginId = Plugin, Credential = "server-owned" },
            new HttpClient(new Handler((_, _) => {
                var response = new HttpResponseMessage((HttpStatusCode)status) { Content = new StringContent("{\"error_code\":\"IAM_BASELINE_SERVICE_FORBIDDEN\"}") };
                response.Headers.Add("X-Trace-ID", "core-trace");
                return Task.FromResult(response);
            })));
        var error = await Assert.ThrowsAsync<AuthorizationBaselineException>(() => client.ReadMemberBaselines(Query()));
        Assert.Equal(status, error.Status);
        Assert.Equal("core-trace", error.TraceId);
        Assert.Equal(status == 200 ? "IAM_BASELINE_RESPONSE_INVALID" : "IAM_BASELINE_SERVICE_FORBIDDEN", error.Code);
    }

    [Fact]
    public async Task Cancellation_timeout_and_canonical_duplicates_fail()
    {
        var client = new PowerXIamClient(new() { BaseUrl = "https://core.example", PluginId = Plugin, Credential = "server-owned", Timeout = TimeSpan.FromMilliseconds(40) },
            new HttpClient(new Handler(async (_, ct) => { await Task.Delay(Timeout.Infinite, ct); return new(HttpStatusCode.OK); })));
        Assert.Equal("IAM_BASELINE_DEPENDENCY_UNAVAILABLE", (await Assert.ThrowsAsync<IAMAdapterException>(() => client.ReadMemberBaselines(Query()))).Code);
        using var cancelled = new CancellationTokenSource(); cancelled.Cancel();
        await Assert.ThrowsAnyAsync<OperationCanceledException>(() => client.ReadMemberBaselines(Query(), cancelled.Token));
        Assert.Equal("IAM_BASELINE_INVALID_ARGUMENT", Assert.Throws<IAMAdapterException>(() =>
            (Query() with { MemberUuids = ["10000000-0000-4000-8000-000000000001", "{10000000-0000-4000-8000-000000000001}"] }).Normalize()).Code);
    }
}
