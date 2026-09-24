using System.Net;
using System.Text;
using System.Text.Json;
using PowerXPlugin.Framework.Runtime.Common;
using PowerXPlugin.Framework.Runtime.Skills;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.Skills;
public sealed class SkillRuntimeTests
{
    private const string Tenant = "00000000-0000-4000-8000-000000000001"; private const string User = "00000000-0000-4000-8000-000000000002";
    private static SkillScope Scope => new(Tenant, User, "agent-1", "session-1", "trace-1", Capability: "crm.read");
    private static SkillInvokeRequest Request => new("summarize", "v1", JsonDocument.Parse("{\"text\":\"hello\"}").RootElement.Clone(), new Dictionary<string, JsonElement> { ["locale"] = JsonDocument.Parse("\"zh-CN\"").RootElement.Clone() });
    [Fact] public async Task Local_refuses_unregistered_or_capability_mismatched_skill_before_execution()
    {
        var invoker = new LocalSkillInvoker(new Registry(null), new Executor()); var error = await Assert.ThrowsAsync<SkillException>(() => invoker.InvokeAsync(Scope, Request)); Assert.Equal(SkillErrors.NotAuthorized, error.Code);
    }
    [Fact] public async Task Delegated_uses_tenant_contract_and_does_not_serialize_trusted_scope()
    {
        string? body = null; HttpRequestMessage? seen = null; var client = new PowerXSkillClient("https://core.example", Tenant, new StaticServiceCredentialProvider(new ServiceCredential("sts")), new HttpClient(new Handler(r => { seen = r; body = r.Content!.ReadAsStringAsync().GetAwaiter().GetResult(); return new HttpResponseMessage(HttpStatusCode.OK) { Content = new StringContent("{\"data\":{\"trace_id\":\"trace-host\",\"status\":\"succeeded\",\"protocol_used\":\"skill\",\"fallback_used\":false,\"result\":{\"summary\":\"ok\"}}}", Encoding.UTF8) }; })));
        await client.InvokeAsync(Scope, Request); Assert.Equal("/api/v1/tenant/skills/invoke", seen!.RequestUri!.AbsolutePath); Assert.Equal("Bearer sts", seen.Headers.Authorization!.ToString()); Assert.DoesNotContain("tenant_uuid", body); Assert.DoesNotContain("user_uuid", body);
    }
    private sealed class Registry(LocalSkillManifest? manifest) : ILocalSkillAuthorizationRegistry { public Task<LocalSkillManifest?> ResolveAuthorizedAsync(SkillScope scope, string id, string version, CancellationToken ct = default) => Task.FromResult(manifest); }
    private sealed class Executor : ILocalSkillExecutor { public Task<SkillInvokeResult> ExecuteAsync(SkillScope s, SkillInvokeRequest r, LocalSkillManifest m, CancellationToken ct = default) => throw new Xunit.Sdk.XunitException("must not execute"); }
    private sealed class Handler(Func<HttpRequestMessage, HttpResponseMessage> f) : HttpMessageHandler { protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage r, CancellationToken ct) => Task.FromResult(f(r)); }
}
