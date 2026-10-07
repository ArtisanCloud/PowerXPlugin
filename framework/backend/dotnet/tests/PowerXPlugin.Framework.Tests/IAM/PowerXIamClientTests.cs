using System.Net;
using System.Text;
using System.Text.Json;
using PowerXPlugin.Framework.IAM;
using PowerXPlugin.Framework.IAM.Delegated;
using Xunit;

namespace PowerXPlugin.Framework.Tests.IAM;

public sealed class PowerXIamClientTests
{
    [Theory]
    [InlineData(1)]
    [InlineData(2)]
    public async Task Maps_Core_numeric_status_to_Framework_string(int status)
    {
        var client = Client(status.ToString(), "tenant-1");
        var members = await client.ListMembers("tenant-1");
        var member = Assert.Single(members);
        Assert.Equal(status.ToString(), member.Status);
        Assert.Equal("member-1", member.MemberUUID);
        Assert.Equal("user-1", member.UserUUID);
    }

    [Theory]
    [InlineData("\"1\"", "tenant-1")]
    [InlineData("1", "tenant-2")]
    public async Task Rejects_invalid_wire_status_or_cross_tenant_response(string status, string tenant)
    {
        var error = await Assert.ThrowsAsync<IAMAdapterException>(() => Client(status, tenant).ListMembers("tenant-1"));
        Assert.Equal(IAMErrors.CodeUpstreamDependency, error.Code);
    }

    [Fact]
    public async Task Reads_every_member_page_with_the_same_service_identity()
    {
        var handler = new PagingHandler("complete");
        var members = await PagingClient(handler).ListMembers("tenant-1");

        Assert.Equal(201, members.Count);
        Assert.Equal("member-201", members[^1].MemberUUID);
        Assert.Equal(["?page=1&page_size=200", "?page=2&page_size=200"], handler.Queries);
    }

    [Theory]
    [InlineData("empty-page")]
    [InlineData("duplicate")]
    [InlineData("changed-total")]
    [InlineData("wrong-page")]
    [InlineData("missing-pagination")]
    public async Task Incomplete_or_inconsistent_pagination_never_returns_a_partial_directory(string scenario)
    {
        var error = await Assert.ThrowsAsync<IAMAdapterException>(() => PagingClient(new(scenario)).ListMembers("tenant-1"));
        Assert.Equal(IAMErrors.CodeUpstreamDependency, error.Code);
    }

    [Fact]
    public async Task Revocation_on_a_later_page_preserves_forbidden_instead_of_returning_page_one()
    {
        var error = await Assert.ThrowsAsync<IAMAdapterException>(() => PagingClient(new("forbidden")).ListMembers("tenant-1"));
        Assert.Equal(IAMErrors.CodeForbidden, error.Code);
    }

    private static PowerXIamClient PagingClient(PagingHandler handler) => new(
        new PowerXIamClientOptions { BaseUrl = "https://core.example", Credential = "sts-token" }, new HttpClient(handler));

    private sealed class PagingHandler(string scenario) : HttpMessageHandler
    {
        public List<string> Queries { get; } = [];
        protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken ct)
        {
            Assert.Equal("Bearer sts-token", request.Headers.Authorization?.ToString());
            Queries.Add(request.RequestUri!.Query);
            var page = Queries.Count;
            if (scenario == "forbidden" && page == 2)
                return Task.FromResult(new HttpResponseMessage(HttpStatusCode.Forbidden) { Content = new StringContent("{}") });
            var count = page == 1 ? 200 : scenario == "empty-page" ? 0 : 1;
            var items = Enumerable.Range((page - 1) * 200 + 1, count).Select(id => new
            {
                member_uuid = "member-" + (scenario == "duplicate" && page == 2 ? 1 : id),
                user_uuid = "user-" + id, tenant_uuid = "tenant-1", status = 1
            }).ToArray();
            var json = scenario == "missing-pagination" ? JsonSerializer.Serialize(new { data = new { items } })
                : JsonSerializer.Serialize(new { data = new { items, pagination = new
                {
                    page = scenario == "wrong-page" ? 10 : page, page_size = 200,
                    total = scenario == "changed-total" && page == 2 ? 202 : 201
                } } });
            return Task.FromResult(new HttpResponseMessage(HttpStatusCode.OK)
            {
                Content = new StringContent(json, Encoding.UTF8, "application/json")
            });
        }
    }

    private static PowerXIamClient Client(string status, string tenant) => new(
        new PowerXIamClientOptions { BaseUrl = "https://core.example", Credential = "sts-token" },
        new HttpClient(new Handler(status, tenant)));

    private sealed class Handler(string status, string tenant) : HttpMessageHandler
    {
        protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken ct)
        {
            Assert.Equal("Bearer sts-token", request.Headers.Authorization?.ToString());
            Assert.Equal("/api/v1/tenant/iam/members?page=1&page_size=200", request.RequestUri!.PathAndQuery);
            var item = "{\"member_uuid\":\"member-1\",\"tenant_uuid\":\"" + tenant + "\",\"user_uuid\":\"user-1\",\"status\":" + status + "}";
            return Task.FromResult(new HttpResponseMessage(HttpStatusCode.OK)
            {
                Content = new StringContent("{\"data\":{\"items\":[" + item + "],\"pagination\":{\"page\":1,\"page_size\":200,\"total\":1}}}", Encoding.UTF8, "application/json")
            });
        }
    }
}
