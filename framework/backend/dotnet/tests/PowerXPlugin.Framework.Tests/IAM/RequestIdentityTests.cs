using System.Net;
using System.Text;
using System.Text.Json;
using System.Text.Json.Nodes;
using Microsoft.AspNetCore.Builder;
using Microsoft.AspNetCore.Http;
using Microsoft.Extensions.DependencyInjection;
using PowerXPlugin.Framework.IAM;
using PowerXPlugin.Framework.IAM.Delegated;
using PowerXPlugin.Framework.IAM.Models;
using PowerXPlugin.Framework.IAM.Contracts;
using PowerXPlugin.Framework.Runtime.WSBus;
using PowerXPlugin.Framework.Middleware;
using Xunit;

namespace PowerXPlugin.Framework.Tests.IAM;

public sealed class RequestIdentityTests
{
    private const string Plugin = "com.example.identity";
    private const string Ticket = "pxctx_v1_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
    private static DelegatedIdentity Identity() => new() {
        Schema="powerx.iam.plugin-request-identity/v1", Mode="delegated", PluginId=Plugin,
        TenantUuid="11111111-1111-4111-8111-111111111111", UserUuid="22222222-2222-4222-8222-222222222222",
        MemberUuid=null, IsRoot=true, Roles=[], Permissions=[], PolicyVersion="p1", PermsHash="hash",
        SourceMethod="GET", SourcePath="/admin/records", SourceRequiredPermission="records:read", Transport="http",
        SourceRequestId="original-request", SourceTraceId="original-trace", TraceId="resolve-trace",
        ProofExpiresAt=DateTime.UtcNow.AddSeconds(30).ToString("O"), UserSessionExpiresAt=DateTime.UtcNow.AddMinutes(5).ToString("O")
    };
    private const string WSHandle="pxwsi_v1_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
    private static DelegatedIdentity WSHandshake() { var identity=Identity();return identity with {Transport="ws_upgrade",SourcePath="/ws",WSRevalidationHandle=WSHandle,WSRevalidationExpiresAt=identity.UserSessionExpiresAt}; }
    private sealed class LeaseRevalidator(DelegatedIdentity identity) : IWSIdentityRevalidationService {
        public Func<WSIdentity,Task<WSIdentity>>? Process {get;set;}
        public int Calls {get;private set;}
        public Task<WSIdentity> RevalidateWSIdentity(WSIdentityQuery query,CancellationToken ct=default) {
            Calls++;Assert.Equal(WSHandle,query.Handle);
            var current=new WSIdentity(identity with {Schema="powerx.iam.plugin-ws-identity/v1",ProofExpiresAt=identity.WSRevalidationExpiresAt!,TraceId="fresh-resolve"});
            return Process?.Invoke(current) ?? Task.FromResult(current);
        }
    }
    private sealed class Handler(Func<HttpRequestMessage,Task<HttpResponseMessage>> send) : HttpMessageHandler
    { protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request,CancellationToken ct) => send(request); }
    private static PowerXIamClient Client(Handler handler,string scheme="Bearer") => new(
        new PowerXIamClientOptions { BaseUrl="https://core.example",PluginId=Plugin,AuthScheme=scheme,Credential="server-owned" },new HttpClient(handler));
    private static HttpResponseMessage Success(object identity) => new(HttpStatusCode.OK) {
        Content=new StringContent(JsonSerializer.Serialize(new { code=200,data=new { result=new { result=identity } } }),Encoding.UTF8,"application/json")
    };

    [Theory]
    [InlineData("Bearer")]
    [InlineData("ApiKey")]
    public async Task Uses_fixed_typed_binding_and_server_credential(string scheme)
    {
        var calls=0;
        var client=Client(new Handler(async request => {
            calls++;
            Assert.Equal(scheme+" server-owned",request.Headers.Authorization?.ToString());
            Assert.Equal("/api/v1/tenant/invocations",request.RequestUri!.AbsolutePath); Assert.Equal("",request.RequestUri.Query);
            var raw=await request.Content!.ReadAsStringAsync(); using var json=JsonDocument.Parse(raw);
            var root=json.RootElement; Assert.Equal(3,root.EnumerateObject().Count());
            Assert.Equal("com.corex.iam.identity_context.service_read",root.GetProperty("capability_id").GetString());
            Assert.Equal("core_internal",root.GetProperty("preferred_protocol").GetString());
            var body=root.GetProperty("payload").GetProperty("body"); Assert.Equal("resolve_request_identity",body.GetProperty("operation").GetString());
            var query=body.GetProperty("request"); Assert.Equal(2,query.EnumerateObject().Count());
            Assert.Equal("powerx.iam.plugin-request-identity-query/v1",query.GetProperty("schema").GetString()); Assert.Equal(Ticket,query.GetProperty("context_ticket").GetString());
            return Success(Identity());
        }),scheme);
        var identity=await client.ResolveDelegatedIdentity(new() { ContextTicket=Ticket });
        Assert.True(identity.IsRoot); Assert.Null(identity.MemberUuid); Assert.Equal("original-trace",identity.SourceTraceId);
        Assert.Equal("resolve-trace",identity.TraceId); Assert.True(identity.ProofExpiry<identity.UserSessionExpiry);
        var error=await Assert.ThrowsAsync<IAMAdapterException>(()=>client.ResolveIdentity("inbound-plugin-token"));
        Assert.Equal("IAM_IDENTITY_MIGRATION_REQUIRED",error.Code); Assert.Equal(1,calls);
        Assert.DoesNotContain(Ticket,new DelegatedIdentityQuery { ContextTicket=Ticket }.ToString());
    }
    [Theory]
    [InlineData(400,"IAM_CONTEXT_INVALID_ARGUMENT")]
    [InlineData(401,"IAM_CONTEXT_SERVICE_CREDENTIAL_EXPIRED")]
    [InlineData(401,"IAM_CONTEXT_PROOF_EXPIRED")]
    [InlineData(401,"IAM_CONTEXT_USER_SESSION_INVALID")]
    [InlineData(403,"IAM_CONTEXT_PLUGIN_FORBIDDEN")]
    [InlineData(409,"IAM_CONTEXT_POLICY_CHANGED")]
    [InlineData(424,"CAPABILITY_MANUAL_UPGRADE_REQUIRED")]
    [InlineData(503,"IAM_CONTEXT_UNAVAILABLE")]
    public async Task Preserves_error_plane_status_and_safe_diagnostics(int status,string code)
    {
        var client=Client(new Handler(request=> {
            var response=new HttpResponseMessage((HttpStatusCode)status) { Content=new StringContent(JsonSerializer.Serialize(new {
                code=status,error_code=code,reason_code=code,message=Ticket,source_trace_id="original-trace",source_request_id="original-request"
            })) };
            response.Headers.Add("X-Trace-ID","resolve-trace");response.Headers.Add("X-Request-ID","upstream-request");return Task.FromResult(response);
        }));
        var error=await Assert.ThrowsAsync<IdentityContextException>(()=>client.ResolveDelegatedIdentity(new() { ContextTicket=Ticket }));
        Assert.Equal(status,error.Status); Assert.Equal(code,error.Code); Assert.Equal("resolve-trace",error.TraceId);
        Assert.Equal("upstream-request",error.RequestId); Assert.Equal("original-trace",error.SourceTraceId);
        Assert.DoesNotContain(Ticket,error.ToString());
    }
    [Theory]
    [InlineData("is_root")]
    [InlineData("member_uuid")]
    [InlineData("proof_expires_at")]
    [InlineData("source_trace_id")]
    public async Task Rejects_omitted_required_fields(string field)
    {
        var node=JsonSerializer.SerializeToNode(Identity())!.AsObject();node.Remove(field);
        var error=await Assert.ThrowsAsync<IAMAdapterException>(()=>Client(new Handler(_=>Task.FromResult(Success(node)))).ResolveDelegatedIdentity(new() { ContextTicket=Ticket }));
        Assert.Equal("IAM_CONTEXT_RESPONSE_INVALID",error.Code);
    }
    [Theory]
    [InlineData("member")]
    [InlineData("tenant")]
    [InlineData("plugin")]
    [InlineData("permissions")]
    public async Task Rejects_invalid_identity_and_cross_scope(string field)
    {
        var node=JsonSerializer.SerializeToNode(Identity())!.AsObject();
        if (field=="member") node["is_root"]=false;
        if (field=="tenant") node["tenant_uuid"]="42";
        if (field=="plugin") node["plugin_id"]="another-plugin";
        if (field=="permissions") node["permissions"]=null;
        var error=await Assert.ThrowsAsync<IAMAdapterException>(()=>Client(new Handler(_=>Task.FromResult(Success(node)))).ResolveDelegatedIdentity(new() { ContextTicket=Ticket }));
        Assert.Equal("IAM_CONTEXT_RESPONSE_INVALID",error.Code);
    }
    [Fact]
    public async Task Missing_or_malformed_proof_never_makes_http_request()
    {
        var client=Client(new Handler(_=>throw new InvalidOperationException("HTTP_MUST_NOT_RUN")));
        foreach(var ticket in new[] { "", "inbound-plugin-token" }) await Assert.ThrowsAsync<IAMAdapterException>(()=>client.ResolveDelegatedIdentity(new() { ContextTicket=ticket }));
    }
    [Fact]
    public void Nanosecond_times_are_preserved_exactly()
    {
        var identity=Identity() with { ProofExpiresAt="2030-01-01T00:00:00.123456789Z",UserSessionExpiresAt="2030-01-01T00:10:00.987654321Z" };
        identity.Validate(Plugin,DateTimeOffset.UtcNow);
        Assert.Equal("2030-01-01T00:00:00.123456789Z",identity.ProofExpiresAt);
    }
    [Fact]
    public async Task Idle_lease_expires_and_closes_without_traffic()
    {
        using var registry=new IdentityLifecycle();var completion=new TaskCompletionSource<string>(TaskCreationOptions.RunContinuationsAsynchronously);
        var identity=WSHandshake();var revalidator=new LeaseRevalidator(identity);using var lease=registry.Open(identity,Plugin,TimeSpan.FromMilliseconds(40),registry.Revision,(v,topic)=>topic=="_topic.allowed",reason=>completion.TrySetResult(reason),revalidator);
        Assert.True(lease.ExpiresAt<=identity.UserSessionExpiry);
        Assert.Equal("IAM_WS_CREDENTIAL_EXPIRED",await completion.Task.WaitAsync(TimeSpan.FromSeconds(2)));
        Assert.True(lease.CancellationToken.IsCancellationRequested);Assert.Equal(0,revalidator.Calls);await Assert.ThrowsAsync<IAMAdapterException>(()=>lease.AuthorizeTopicAsync("_topic.allowed"));
    }
    [Fact]
    public async Task Topic_scope_revocation_and_hydration_fence_are_enforced()
    {
        using var registry=new IdentityLifecycle();var identity=WSHandshake();var revalidator=new LeaseRevalidator(identity);var revision=registry.Revision;var closes=0;
        using var lease=registry.Open(identity,Plugin,TimeSpan.FromMinutes(2),revision,(v,topic)=>topic=="_topic.allowed",_=>closes++,revalidator);
        await lease.AuthorizeTopicAsync("_topic.allowed");
        Assert.Equal(0,registry.Revoke(new(TenantUuid:"33333333-3333-4333-8333-333333333333")));
        Assert.Equal(1,registry.Revoke(new(TenantUuid:identity.TenantUuid,UserUuid:identity.UserUuid)));Assert.Equal(1,closes);
        await Assert.ThrowsAsync<IAMAdapterException>(()=>lease.AuthorizeTopicAsync("_topic.allowed"));
        var error=Assert.Throws<IAMAdapterException>(()=>registry.Open(identity,Plugin,TimeSpan.FromMinutes(2),revision,(v,topic)=>true,_=>{},revalidator));Assert.Equal("IAM_CONTEXT_POLICY_CHANGED",error.Code);
    }
    [Theory]
    [InlineData("root",200)]
    [InlineData("missing-proof",401)]
    [InlineData("cross-path",403)]
    [InlineData("cross-user",403)]
    [InlineData("role-without-permission",403)]
    public async Task Middleware_validates_authority_before_business_permission(string scenario,int status)
    {
        var identity=Identity();
        if (scenario=="cross-path") identity=identity with { SourcePath="/admin/another" };
        if (scenario=="role-without-permission") identity=identity with { IsRoot=false, MemberUuid="33333333-3333-4333-8333-333333333333",Roles=["super_admin"],Permissions=[] };
        var client=Client(new Handler(_=>Task.FromResult(Success(identity))));
        var registry=new IAMRegistry();registry.Bind(IAMAdapterMode.Delegated,client,client,client);
        using var services=new ServiceCollection().AddSingleton(registry).BuildServiceProvider();
        var builder=new ApplicationBuilder(services);
        builder.UseDelegatedIdentity(new() { PluginId=Plugin,ApiPrefix="/api/v1",
            UserAssertion=_=>scenario=="cross-user" ? "44444444-4444-4444-8444-444444444444" : identity.UserUuid });
        builder.UseAuthGuard(new() { Resource="records",Action="read" });
        var called=false;builder.Run(context=> { called=true;context.Response.StatusCode=200;return Task.CompletedTask; });
        var context=new DefaultHttpContext();context.Request.Method="GET";context.Request.Path="/api/v1/admin/records";
        context.Response.Body=new MemoryStream();if (scenario!="missing-proof") context.Request.Headers["X-PowerX-Identity-Context"]=Ticket;
        await builder.Build()(context);
        Assert.Equal(status,context.Response.StatusCode);Assert.Equal(status==200,called);
        if (status==200) {Assert.Null(context.GetDelegatedIdentity()!.MemberUuid);Assert.Equal(identity.TenantUuid,context.GetTenantUUID());}
    }
}
