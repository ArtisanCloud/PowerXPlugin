using System.Net;
using System.Text.Json;
using System.Text.Json.Nodes;
using PowerXPlugin.Framework.IAM;
using PowerXPlugin.Framework.IAM.Contracts;
using PowerXPlugin.Framework.IAM.Delegated;
using PowerXPlugin.Framework.IAM.Models;
using PowerXPlugin.Framework.Runtime.WSBus;
using Xunit;

namespace PowerXPlugin.Framework.Tests.IAM;
public sealed class WSRevalidationTests
{
    private const string Plugin="com.example.ws";
    private const string Handle="pxwsi_v1_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
    private static DelegatedIdentity Initial()
    {
        var expiry=DateTime.UtcNow.AddMinutes(5).ToString("O");
        return new(){Schema="powerx.iam.plugin-request-identity/v1",Mode="delegated",PluginId=Plugin,TenantUuid="11111111-1111-4111-8111-111111111111",UserUuid="22222222-2222-4222-8222-222222222222",MemberUuid="33333333-3333-4333-8333-333333333333",IsRoot=false,Roles=[],Permissions=["topic:read"],PolicyVersion="p1",PermsHash="hash",SourceMethod="GET",SourcePath="/ws",SourceRequiredPermission="runtime.realtime:read",Transport="ws_upgrade",SourceRequestId="original",SourceTraceId="original-trace",TraceId="resolve",ProofExpiresAt=DateTime.UtcNow.AddSeconds(30).ToString("O"),UserSessionExpiresAt=expiry,WSRevalidationHandle=Handle,WSRevalidationExpiresAt=expiry};
    }
    private sealed class Revalidator(DelegatedIdentity initial):IWSIdentityRevalidationService
    {
        public int Calls{get;private set;}
        public Func<WSIdentity,Task<WSIdentity>>? Change{get;set;}
        public Task<WSIdentity> RevalidateWSIdentity(WSIdentityQuery query,CancellationToken ct=default)
        {
            Calls++;Assert.Equal(Handle,query.Handle);
            var result=new WSIdentity(initial with {Schema="powerx.iam.plugin-ws-identity/v1",ProofExpiresAt=initial.WSRevalidationExpiresAt!,TraceId="fresh"});
            return Change?.Invoke(result) ?? Task.FromResult(result);
        }
    }
    private sealed class Handler(Func<HttpRequestMessage,Task<HttpResponseMessage>> call):HttpMessageHandler
    {protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request,CancellationToken ct)=>call(request);}
    [Theory]
    [InlineData("ApiKey")]
    [InlineData("Bearer")]
    public async Task Typed_client_maps_handle_and_keeps_it_server_only(string scheme)
    {
        var initial=Initial();var output=initial with {Schema="powerx.iam.plugin-ws-identity/v1",ProofExpiresAt=initial.WSRevalidationExpiresAt!};
        var client=new PowerXIamClient(new(){BaseUrl="https://core.example",PluginId=Plugin,AuthScheme=scheme,Credential="server-owned"},new HttpClient(new Handler(async request=>{
            Assert.Equal(scheme+" server-owned",request.Headers.Authorization!.ToString());Assert.Equal("/api/v1/tenant/invocations",request.RequestUri!.AbsolutePath);
            using var document=JsonDocument.Parse(await request.Content!.ReadAsStringAsync());var body=document.RootElement.GetProperty("payload").GetProperty("body");
            Assert.Equal("revalidate_ws_identity",body.GetProperty("operation").GetString());var query=body.GetProperty("request");Assert.Equal(2,query.EnumerateObject().Count());Assert.Equal(Handle,query.GetProperty("ws_revalidation_handle").GetString());Assert.Equal("powerx.iam.plugin-ws-identity-query/v1",query.GetProperty("schema").GetString());
            var fields=JsonSerializer.SerializeToNode(output)!.AsObject();fields["ws_revalidation_handle"]=Handle;fields["ws_revalidation_expires_at"]=output.WSRevalidationExpiresAt;
            return new(HttpStatusCode.OK){Content=new StringContent(JsonSerializer.Serialize(new{code=200,data=new{result=new{result=fields}}}))};
        })));
        var result=await client.RevalidateWSIdentity(new(){Handle=Handle});Assert.Equal(Handle,result.Identity.WSRevalidationHandle);
        Assert.Equal(initial.UserSessionExpiresAt,result.Identity.UserSessionExpiresAt);Assert.DoesNotContain(Handle,JsonSerializer.Serialize(result));Assert.DoesNotContain(Handle,result.ToString());
    }
    [Fact]
    public async Task Each_topic_message_uses_current_permissions_without_polling_or_expiry_extension()
    {
        using var lifecycle=new IdentityLifecycle();var initial=Initial() with {ProofExpiresAt=DateTime.UtcNow.AddMilliseconds(20).ToString("O")};var revalidator=new Revalidator(initial);var closes=0;
        using var lease=lifecycle.Open(initial,Plugin,TimeSpan.FromSeconds(1),lifecycle.Revision,(identity,topic)=>identity.HasPermission("topic:read"),_=>closes++,revalidator);
        await Task.Delay(30);Assert.Equal(0,revalidator.Calls);await lease.AuthorizeTopicAsync("_topic.allowed");
        revalidator.Change=result=>Task.FromResult(new WSIdentity(result.Identity with {Permissions=[],PolicyVersion="p2"}));
        var error=await Assert.ThrowsAsync<IAMAdapterException>(()=>lease.AuthorizeTopicAsync("_topic.allowed"));Assert.Equal("IAM_WS_TOPIC_FORBIDDEN",error.Code);Assert.Equal(1,closes);Assert.Equal(2,revalidator.Calls);Assert.True(lease.ExpiresAt<=initial.UserSessionExpiry);
    }
    [Theory]
    [InlineData("IAM_CONTEXT_USER_SESSION_INVALID")]
    [InlineData("IAM_CONTEXT_USER_FORBIDDEN")]
    [InlineData("IAM_CONTEXT_SERVICE_FORBIDDEN")]
    [InlineData("IAM_CONTEXT_POLICY_CHANGED")]
    [InlineData("IAM_CONTEXT_UNAVAILABLE")]
    [InlineData("IAM_WS_IDENTITY_MISMATCH")]
    [InlineData("LATE_RESULT")]
    public async Task Rejected_or_late_revalidation_closes_without_authorizing(string code)
    {
        using var lifecycle=new IdentityLifecycle();var initial=Initial();var revalidator=new Revalidator(initial);var closes=0;var authorized=false;
        using var lease=lifecycle.Open(initial,Plugin,TimeSpan.FromMinutes(1),lifecycle.Revision,(identity,topic)=>{authorized=true;return true;},_=>closes++,revalidator);
        revalidator.Change=result=>{
            if(code=="IAM_WS_IDENTITY_MISMATCH")return Task.FromResult(new WSIdentity(result.Identity with {UserUuid="44444444-4444-4444-8444-444444444444"}));
            if(code=="LATE_RESULT"){lease.Close("IAM_WS_IDENTITY_REVOKED");return Task.FromResult(result);}
            throw new IAMAdapterException(code);
        };
        await Assert.ThrowsAsync<IAMAdapterException>(()=>lease.AuthorizeTopicAsync("_topic.allowed"));Assert.False(authorized);Assert.Equal(1,closes);
    }

    [Fact]
    public async Task Cancellation_during_topic_authorization_cannot_commit_success()
    {
        using var lifecycle=new IdentityLifecycle();var initial=Initial();var revalidator=new Revalidator(initial);var closes=0;
        using var cancellation=new CancellationTokenSource();
        using var lease=lifecycle.Open(initial,Plugin,TimeSpan.FromMinutes(1),lifecycle.Revision,(identity,topic)=>{cancellation.Cancel();return true;},_=>closes++,revalidator);
        var error=await Assert.ThrowsAsync<IAMAdapterException>(()=>lease.AuthorizeTopicAsync("_topic.allowed",cancellation.Token));
        Assert.Equal("IAM_WS_IDENTITY_REVOKED",error.Code);Assert.Equal(1,closes);
    }
}
