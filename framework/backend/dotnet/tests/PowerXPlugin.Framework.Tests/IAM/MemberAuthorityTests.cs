using System.Net;
using System.Text;
using System.Text.Json;
using PowerXPlugin.Framework.IAM;
using PowerXPlugin.Framework.IAM.Delegated;
using PowerXPlugin.Framework.IAM.Models;
using Xunit;

namespace PowerXPlugin.Framework.Tests.IAM;

public sealed class MemberAuthorityTests
{
    private const string Tenant = "00000000-0000-4000-8000-000000000001";
    private const string Member = "00000000-0000-4000-8000-000000000002";
    private const string User = "00000000-0000-4000-8000-000000000003";
    private const string Department = "00000000-0000-4000-8000-000000000004";
    private static MemberAuthorityQuery Query => new([Member]);
    private static MemberAuthorityBatch Batch => new() { TenantUuid = Tenant, EvaluatedAt = DateTimeOffset.UtcNow,
        Members = [new() { MemberUuid = Member, UserUuid = User, Status = "active", ManagerStatus = "unset", UpdatedAt = DateTimeOffset.UtcNow, RoleCodes = ["supervisor"], DepartmentUuids = [Department] }],
        Departments = [new() { DepartmentUuid = Department, Name = "fixture_department", LeaderStatus = "unset" }] };
    private static PowerXIamClient Client(Func<HttpRequestMessage, HttpResponseMessage> handler) => new(new PowerXIamClientOptions { BaseUrl = "https://core.example", PluginId = "plugin.fixture", Credential = "fixture_credential" }, new HttpClient(new Handler(handler)));
    private static HttpResponseMessage Ok(MemberAuthorityBatch batch) => new(HttpStatusCode.OK) { Content = new StringContent(JsonSerializer.Serialize(new { code = 200, data = new { result = new { result = batch } } }), Encoding.UTF8, "application/json") };

    [Fact]
    public async Task Uses_fixed_typed_service_contract_and_checks_exact_member_coverage()
    {
        var client = Client(request => {
            Assert.Equal("/api/v1/tenant/invocations", request.RequestUri!.AbsolutePath); Assert.Equal("Bearer fixture_credential", request.Headers.Authorization!.ToString());
            using var json = JsonDocument.Parse(request.Content!.ReadAsStringAsync().GetAwaiter().GetResult()); var root = json.RootElement;
            Assert.Equal(MemberAuthorityBatch.CapabilityId, root.GetProperty("capability_id").GetString()); Assert.Equal("core_internal", root.GetProperty("preferred_protocol").GetString());
            var body = root.GetProperty("payload").GetProperty("body"); Assert.Equal("read_member_authority", body.GetProperty("operation").GetString()); Assert.False(body.GetProperty("request").TryGetProperty("tenant_uuid", out _));
            return Ok(Batch);
        });
        var result = await client.ReadMemberAuthority(Tenant, Query); Assert.Equal(Member, Assert.Single(result.Members).MemberUuid); Assert.Equal(Department, Assert.Single(result.Departments).DepartmentUuid);
    }

    [Theory]
    [InlineData("schema")] [InlineData("tenant")] [InlineData("missing_member")] [InlineData("duplicate_member")] [InlineData("manager")] [InlineData("parent")] [InlineData("cycle")] [InlineData("user")] [InlineData("not_found_manager")]
    public async Task Mismatched_or_incomplete_authority_never_becomes_local_or_empty_authority(string mismatch)
    {
        var batch = Batch;
        batch = mismatch switch {
            "schema" => batch with { Schema = "old" }, "tenant" => batch with { TenantUuid = User },
            "missing_member" => batch with { Members = [] }, "duplicate_member" => batch with { Members = [batch.Members[0], batch.Members[0]] },
            "manager" => batch with { Members = [batch.Members[0] with { ManagerStatus = "active", DirectManagerMemberUuid = Member }] },
            "parent" => batch with { Departments = [batch.Departments[0] with { ParentDepartmentUuid = User }] },
            "cycle" => batch with { Departments = [batch.Departments[0] with { ParentDepartmentUuid = Department }] },
            "user" => batch with { Members = [batch.Members[0] with { UserUuid = "invalid" }] },
            "not_found_manager" => batch with { Members = [new() { MemberUuid = Member, Status = "not_found", ManagerStatus = "active", DirectManagerMemberUuid = User }] }, _ => batch
        };
        var error = await Assert.ThrowsAsync<MemberAuthorityException>(() => Client(_ => Ok(batch)).ReadMemberAuthority(Tenant, Query)); Assert.Equal(502, error.Status); Assert.Equal("IAM_ORGANIZATION_RESPONSE_INVALID", error.Code);
    }

    [Fact]
    public async Task Missing_required_collection_is_rejected_even_when_an_empty_collection_would_be_valid()
    {
        var response = new { code = 200, data = new { result = new { result = new { schema = "powerx.iam.member-authority.v1", tenant_uuid = Tenant, evaluated_at = DateTimeOffset.UtcNow, members = new[] { new { member_uuid = Member, user_uuid = User, status = "active", manager_status = "unset", direct_manager_member_uuid = (string?)null, updated_at = DateTimeOffset.UtcNow, department_uuids = Array.Empty<string>() } }, departments = Array.Empty<object>() } } } };
        var error = await Assert.ThrowsAsync<MemberAuthorityException>(() => Client(_ => new(HttpStatusCode.OK) { Content = new StringContent(JsonSerializer.Serialize(response)) }).ReadMemberAuthority(Tenant, Query)); Assert.Equal("IAM_ORGANIZATION_RESPONSE_INVALID", error.Code);
    }

    [Theory]
    [InlineData(400, "IAM_ORGANIZATION_INVALID")]
    [InlineData(401, "IAM_ORGANIZATION_SERVICE_REQUIRED")]
    [InlineData(403, "IAM_ORGANIZATION_TENANT_UNAVAILABLE")]
    [InlineData(409, "IAM_ORGANIZATION_CHANGED")]
    [InlineData(502, "IAM_ORGANIZATION_SCHEMA_NOT_READY")]
    [InlineData(503, "IAM_ORGANIZATION_SCHEMA_NOT_READY")]
    public async Task Preserves_formal_core_error_and_received_status_without_retry_or_local_fallback(int status, string code)
    {
        var count = 0;
        var client = Client(_ => {
            count++;
            var response = Failure(status, new {
                code = status, message = "integration.invoke_failed", error = "fixture_private_error",
                error_code = code, reason_code = code, request_id = "fixture_body_request",
                details = new { code = "integration.invoke_failed", trace_id = "fixture_body_trace" }
            });
            response.Headers.Add("X-Trace-ID", "fixture_header_trace");
            response.Headers.Add("X-Request-ID", "fixture_header_request");
            return response;
        });
        var error = await Assert.ThrowsAsync<MemberAuthorityException>(() => client.ReadMemberAuthority(Tenant, Query));
        Assert.Equal(status, error.Status);
        Assert.Equal(code, error.Code);
        Assert.Equal(code, error.OriginalCode);
        Assert.Equal(code, error.ReasonCode);
        Assert.Equal("fixture_header_trace", error.TraceId);
        Assert.Equal("fixture_header_request", error.RequestId);
        Assert.DoesNotContain("fixture_private_error", error.ToString());
        Assert.Equal(1, count);
    }

    [Fact]
    public async Task Reads_request_and_invocation_trace_from_formal_body_when_headers_are_absent()
    {
        var client = Client(_ => Failure(403, new {
            error = "fixture_private_error", error_code = "IAM_ORGANIZATION_TENANT_UNAVAILABLE",
            request_id = "fixture_body_request", details = new { trace_id = "fixture_invocation_trace" }
        }));
        var error = await Assert.ThrowsAsync<MemberAuthorityException>(() => client.ReadMemberAuthority(Tenant, Query));
        Assert.Equal("IAM_ORGANIZATION_TENANT_UNAVAILABLE", error.Code);
        Assert.Equal("fixture_body_request", error.RequestId);
        Assert.Equal("fixture_invocation_trace", error.TraceId);
    }

    [Theory]
    [InlineData("IAM_ORGANIZATION_UNAVAILABLE", "IAM_ORGANIZATION_SCHEMA_NOT_READY")]
    [InlineData("IAM_ORGANIZATION_INVALID", null)]
    [InlineData(null, "IAM_ORGANIZATION_TENANT_UNAVAILABLE")]
    public async Task Keeps_error_and_reason_codes_separate_from_generic_gateway_codes(string? originalCode, string? reasonCode)
    {
        var body = new Dictionary<string, object> {
            ["code"] = 502,
            ["error"] = new { code = "fixture_ignored_nested_code" },
            ["details"] = new { code = "integration.invoke_failed", reason_code = "fixture_ignored_nested_reason" }
        };
        if (originalCode is not null) body["error_code"] = originalCode;
        if (reasonCode is not null) body["reason_code"] = reasonCode;
        var error = await Assert.ThrowsAsync<MemberAuthorityException>(() => Client(_ => Failure(502, body)).ReadMemberAuthority(Tenant, Query));
        Assert.Equal(originalCode, error.OriginalCode);
        Assert.Equal(reasonCode, error.ReasonCode);
        Assert.Equal(reasonCode ?? originalCode, error.Code);
        Assert.Equal(502, error.Status);
    }

    [Theory]
    [InlineData("")]
    [InlineData("{")]
    [InlineData("null")]
    [InlineData("[]")]
    [InlineData("42")]
    [InlineData("\"fixture_scalar\"")]
    [InlineData("{}")]
    [InlineData("{\"error\":{\"code\":\"IAM_ORGANIZATION_INVALID\"}}")]
    [InlineData("{\"error_code\":400}")]
    [InlineData("{\"error_code\":\"\"}")]
    [InlineData("{\"reason_code\":true}")]
    [InlineData("{\"error_code\":\"IAM_ORGANIZATION_INVALID\",\"reason_code\":[]}")]
    [InlineData("{\"error_code\":\"IAM_ORGANIZATION_INVALID\",\"details\":[]}")]
    [InlineData("{\"error_code\":\"IAM_ORGANIZATION_INVALID\",\"request_id\":42}")]
    public async Task Invalid_error_envelope_remains_typed_and_keeps_received_status_and_headers(string body)
    {
        var client = Client(_ => {
            var response = new HttpResponseMessage(HttpStatusCode.ServiceUnavailable) { Content = new StringContent(body) };
            response.Headers.Add("X-Trace-ID", "fixture_trace");
            response.Headers.Add("X-Request-ID", "fixture_request");
            return response;
        });
        var error = await Assert.ThrowsAsync<MemberAuthorityException>(() => client.ReadMemberAuthority(Tenant, Query));
        Assert.Equal("IAM_ORGANIZATION_RESPONSE_INVALID", error.Code);
        Assert.Equal(503, error.Status);
        Assert.Equal("fixture_trace", error.TraceId);
        Assert.Equal("fixture_request", error.RequestId);
    }

    [Fact]
    public async Task Oversized_error_envelope_cannot_be_accepted_as_a_valid_truncated_document()
    {
        var client = Client(_ => {
            var body = "{\"error_code\":\"IAM_ORGANIZATION_INVALID\"}" + new string(' ', 1 << 20);
            var response = new HttpResponseMessage(HttpStatusCode.BadRequest) { Content = new StringContent(body) };
            response.Headers.Add("X-Request-ID", "fixture_request");
            return response;
        });
        var error = await Assert.ThrowsAsync<MemberAuthorityException>(() => client.ReadMemberAuthority(Tenant, Query));
        Assert.Equal("IAM_ORGANIZATION_RESPONSE_INVALID", error.Code);
        Assert.Equal(400, error.Status);
        Assert.Equal("fixture_request", error.RequestId);
    }

    private static HttpResponseMessage Failure(int status, object body) => new((HttpStatusCode)status)
    { Content = new StringContent(JsonSerializer.Serialize(body), Encoding.UTF8, "application/json") };

    [Fact]
    public async Task Invalid_request_is_rejected_before_transport()
    {
        var client = Client(_ => throw new Xunit.Sdk.XunitException("fixture_unexpected_network"));
        await Assert.ThrowsAsync<IAMAdapterException>(() => client.ReadMemberAuthority(Tenant, new([Member, Member])));
        await Assert.ThrowsAsync<IAMAdapterException>(() => client.ReadMemberAuthority(Tenant, new([])));
    }
    private sealed class Handler(Func<HttpRequestMessage, HttpResponseMessage> handler) : HttpMessageHandler
    { protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken ct) => Task.FromResult(handler(request)); }
}
