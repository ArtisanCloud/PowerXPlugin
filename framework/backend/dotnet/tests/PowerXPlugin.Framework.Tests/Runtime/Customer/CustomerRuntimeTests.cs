using System.Net;
using System.Text;
using PowerXPlugin.Framework.Runtime.Common;
using PowerXPlugin.Framework.Runtime.Customer;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.Customer;
public sealed class CustomerRuntimeTests
{
    private const string Tenant="00000000-0000-4000-8000-000000000001", Customer="00000000-0000-4000-8000-000000000002";
    [Fact] public async Task Membership_uses_two_credentials_and_rejects_tenant_override()
    {
        HttpRequestMessage? seen=null; var client=new PowerXCustomerClient("https://core.example",Tenant,new StaticServiceCredentialProvider(new ServiceCredential("sts")),new HttpClient(new Handler(r=>{seen=r;return new HttpResponseMessage(HttpStatusCode.OK){Content=new StringContent($"{{\"data\":{{\"tenant_uuid\":\"{Tenant}\",\"customer_uuid\":\"{Customer}\",\"membership_uuid\":\"m\",\"status\":\"active\",\"roles\":[],\"scopes\":[]}}}}",Encoding.UTF8)};})));
        await client.ResolveAsync(new CustomerSession(Tenant,Customer,"customer-jwt")); Assert.Equal("/api/v1/tenant/customer/memberships:resolve",seen!.RequestUri!.AbsolutePath); Assert.Equal("Bearer sts",seen.Headers.Authorization!.ToString()); Assert.Equal("Bearer customer-jwt",seen.Headers.GetValues("X-PowerX-Customer-Authorization").Single());
        var e=await Assert.ThrowsAsync<CustomerException>(()=>client.ResolveAsync(new CustomerSession("00000000-0000-4000-8000-000000000003",Customer,"jwt")));Assert.Equal(CustomerErrors.TenantMismatch,e.Code);
    }
    private sealed class Handler(Func<HttpRequestMessage,HttpResponseMessage> f):HttpMessageHandler { protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage r,CancellationToken ct)=>Task.FromResult(f(r)); }
}
