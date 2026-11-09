using System.Net;
using System.Text;
using System.Text.Json;
using PowerXPlugin.Framework.Runtime.Common;
using PowerXPlugin.Framework.Runtime.Customer;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.Customer;

public sealed class CustomerMiniAppTests
{
    private const string Tenant = "00000000-0000-4000-8000-000000000001";
    private const string Customer = "00000000-0000-4000-8000-000000000002";
    private static readonly CustomerSession Session = new(Tenant, Customer, "customer-jwt");
    private static readonly CustomerPhoneVerifyRequest PhoneRequest = new(
        "wechat_miniapp", " wx-contract ", new("wechat_miniapp_phone_code", "phone-code"));

    [Theory]
    [InlineData(true, "/api/v1/tenant/customer/auth/register")]
    [InlineData(false, "/api/v1/tenant/customer/auth/login")]
    public async Task Auth_forwards_app_and_code_without_tenant(bool register, string path)
    {
        var client = Client(async request =>
        {
            Assert.Equal(path, request.RequestUri!.AbsolutePath);
            Assert.Equal("Bearer sts", request.Headers.Authorization!.ToString());
            using var body = JsonDocument.Parse(await request.Content!.ReadAsStringAsync());
            Assert.Equal("wechat_miniapp", body.RootElement.GetProperty("channel").GetString());
            Assert.Equal("wx-contract", body.RootElement.GetProperty("app_id").GetString());
            Assert.False(body.RootElement.TryGetProperty("tenant_uuid", out _));
            var credential = body.RootElement.GetProperty("credential");
            Assert.Equal("wechat_miniapp_code", credential.GetProperty("type").GetString());
            Assert.Equal("login-code", credential.GetProperty("value").GetString());
            return Response(HttpStatusCode.OK, JsonSerializer.Serialize(new
            {
                data = new
                {
                    access_token = "customer-jwt", expires_in = 60, phone_verified = true,
                    context = new { tenant_uuid = Tenant, customer_uuid = Customer, membership_uuid = "00000000-0000-4000-8000-000000000003", authenticated = true }
                }
            }));
        });
        var input = new CustomerAuthRequest("wechat_miniapp", new("wechat_miniapp_code", "login-code"), " wx-contract ");
        var result = register ? await client.RegisterAsync(input) : await client.LoginAsync(input);
        Assert.True(result.PhoneVerified);
        Assert.Equal("customer-jwt", result.AccessToken);
    }

    [Fact]
    public async Task Phone_verification_forwards_both_credentials_and_reads_result()
    {
        var client = Client(async request =>
        {
            Assert.Equal("/api/v1/tenant/customer/phone:verify", request.RequestUri!.AbsolutePath);
            Assert.Equal("Bearer sts", request.Headers.Authorization!.ToString());
            Assert.Equal("Bearer customer-jwt", request.Headers.GetValues("X-PowerX-Customer-Authorization").Single());
            using var body = JsonDocument.Parse(await request.Content!.ReadAsStringAsync());
            Assert.Equal("wx-contract", body.RootElement.GetProperty("app_id").GetString());
            Assert.False(body.RootElement.TryGetProperty("tenant_uuid", out _));
            Assert.Equal("wechat_miniapp_phone_code", body.RootElement.GetProperty("credential").GetProperty("type").GetString());
            return Response(HttpStatusCode.OK, "{\"data\":{\"phone_masked\":\"138****1234\",\"verified_at\":\"2026-11-09T12:00:00Z\"}}");
        });
        var result = await client.VerifyPhoneAsync(Session, PhoneRequest);
        Assert.Equal("138****1234", result.PhoneMasked);
        Assert.Equal(DateTimeOffset.Parse("2026-11-09T12:00:00Z"), result.VerifiedAt);
    }

    [Theory]
    [InlineData(400, CustomerErrors.ClientAppNotFound)]
    [InlineData(403, CustomerErrors.ClientAppForbidden)]
    [InlineData(400, CustomerErrors.ClientPlatformUnsupported)]
    [InlineData(503, CustomerErrors.ClientAppNotConfigured)]
    [InlineData(401, CustomerErrors.PhoneCodeInvalid)]
    [InlineData(409, CustomerErrors.PhoneConflict)]
    public async Task Phone_verification_preserves_Core_code_status_and_reason(int status, string reason)
    {
        var client = Client(_ => Task.FromResult(Response((HttpStatusCode)status,
            JsonSerializer.Serialize(new { error = new { code = reason, reason_code = reason } }))));
        var error = await Assert.ThrowsAsync<CustomerException>(() => client.VerifyPhoneAsync(Session, PhoneRequest));
        Assert.Equal(reason, error.Code);
        Assert.Equal(status, error.StatusCode);
        Assert.Equal(reason, error.ReasonCode);
    }

    [Theory]
    [InlineData("{\"reason_code\":\"CUSTOMER_CLIENT_APP_FORBIDDEN\",\"error\":\"diagnostic\"}")]
    [InlineData("{\"details\":{\"reason_code\":\"CUSTOMER_CLIENT_APP_FORBIDDEN\"}}")]
    [InlineData("{\"error\":{\"code\":\"CUSTOMER_CLIENT_APP_FORBIDDEN\"}}")]
    public async Task Login_preserves_structured_Core_reasons(string body)
    {
        var client = Client(_ => Task.FromResult(Response(HttpStatusCode.Forbidden, body)));
        var error = await Assert.ThrowsAsync<CustomerException>(() => client.LoginAsync(
            new("wechat_miniapp", new("wechat_miniapp_code", "code"), "wx-contract")));
        Assert.Equal(CustomerErrors.ClientAppForbidden, error.Code);
        Assert.Equal(CustomerErrors.ClientAppForbidden, error.ReasonCode);
        Assert.Equal(403, error.StatusCode);
    }

    [Fact]
    public async Task Unknown_Core_reason_and_status_remain_available()
    {
        const string reason = "CUSTOMER_TEST_UPSTREAM_REJECTED";
        var client = Client(_ => Task.FromResult(Response(HttpStatusCode.TooManyRequests,
            JsonSerializer.Serialize(new { reason_code = reason }))));
        var error = await Assert.ThrowsAsync<CustomerException>(() => client.VerifyPhoneAsync(Session, PhoneRequest));
        Assert.Equal(CustomerErrors.Upstream, error.Code);
        Assert.Equal(reason, error.ReasonCode);
        Assert.Equal(429, error.StatusCode);
    }

    [Theory]
    [InlineData("[]")]
    [InlineData("not-json")]
    [InlineData("{\"error\":\"diagnostic\"}")]
    public async Task Invalid_Core_error_envelopes_fail_explicitly(string body)
    {
        var client = Client(_ => Task.FromResult(Response(HttpStatusCode.BadGateway, body)));
        var error = await Assert.ThrowsAsync<CustomerException>(() => client.VerifyPhoneAsync(Session, PhoneRequest));
        Assert.Equal(CustomerErrors.InvalidResponse, error.Code);
        Assert.Equal(502, error.StatusCode);
        Assert.Null(error.ReasonCode);
    }

    [Fact]
    public async Task Invalid_inputs_are_rejected_before_transport()
    {
        var calls = 0;
        var client = Client(_ =>
        {
            calls++;
            return Task.FromResult(Response(HttpStatusCode.OK, "{}"));
        });
        await Assert.ThrowsAsync<CustomerException>(() => client.RegisterAsync(new("wechat_miniapp", new("wechat_miniapp_code", "code"))));
        await Assert.ThrowsAsync<CustomerException>(() => client.LoginAsync(new("wechat_miniapp", new("shopify_customer_access_token", "code"), "wx-contract")));
        await Assert.ThrowsAsync<CustomerException>(() => client.VerifyPhoneAsync(Session, PhoneRequest with { AppId = "" }));
        await Assert.ThrowsAsync<CustomerException>(() => client.VerifyPhoneAsync(Session, PhoneRequest with { Credential = new("wechat_miniapp_code", "code") }));
        await Assert.ThrowsAsync<CustomerException>(() => client.VerifyPhoneAsync(Session with { CustomerBearer = "" }, PhoneRequest));
        await Assert.ThrowsAsync<CustomerException>(() => client.VerifyPhoneAsync(Session with { TenantUuid = "00000000-0000-4000-8000-000000000004" }, PhoneRequest));
        Assert.Equal(0, calls);
    }

    private static PowerXCustomerClient Client(Func<HttpRequestMessage, Task<HttpResponseMessage>> handler) =>
        new("https://core.example", Tenant, new StaticServiceCredentialProvider(new ServiceCredential("sts")), new HttpClient(new Handler(handler)));

    private static HttpResponseMessage Response(HttpStatusCode status, string body) =>
        new(status) { Content = new StringContent(body, Encoding.UTF8, "application/json") };

    private sealed class Handler(Func<HttpRequestMessage, Task<HttpResponseMessage>> handler) : HttpMessageHandler
    {
        protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken ct) => handler(request);
    }
}
