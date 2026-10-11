using System.Net;
using System.Net.Http.Json;
using System.Text.Json;
using PowerXPlugin.Framework.Runtime.Scheduler;
using PowerXPlugin.Framework.Runtime.Scheduler.Delegated;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.Scheduler;

public sealed class SchedulerHostCleanupTests
{
    private const string Id = "00000000-0000-4000-8000-000000000010";
    private const string Tenant = "00000000-0000-4000-8000-000000000001";
    private const string Owner = "com.powerx.plugins.crm";

    [Theory]
    [InlineData("apikey")]
    [InlineData("Bearer")]
    public async Task Service_reads_and_deletion_use_the_official_plane_and_keep_receipt_and_history(string scheme)
    {
        var operations = new List<string>();
        var handler = new Handler(async request =>
        {
            Assert.Equal(scheme == "apikey" ? "ApiKey" : "Bearer", request.Headers.Authorization!.Scheme);
            Assert.Equal("service-fixture", request.Headers.Authorization.Parameter);
            Assert.DoesNotContain("/admin/", request.RequestUri!.AbsolutePath);
            string operation;
            if (scheme == "apikey")
            {
                Assert.Null(request.Content);
                Assert.DoesNotContain("tenant_uuid", request.RequestUri.Query);
                operation = request.Method == HttpMethod.Delete ? "delete_job" : request.RequestUri.AbsolutePath.EndsWith("/runs") ? "list_runs"
                    : request.RequestUri.AbsolutePath.EndsWith("/jobs") ? "list_jobs" : "get_job";
                if (operation == "delete_job") Assert.Equal("?expected_revision=7", request.RequestUri.Query);
            }
            else
            {
                Assert.Equal(HttpMethod.Post, request.Method);
                Assert.Equal("/api/v1/tenant/invocations", request.RequestUri.AbsolutePath);
                using var json = JsonDocument.Parse(await request.Content!.ReadAsStringAsync());
                var root = json.RootElement;
                Assert.Equal("core_internal", root.GetProperty("preferred_protocol").GetString());
                var payload = root.GetProperty("payload");
                Assert.Equal("INVOKE", payload.GetProperty("method").GetString());
                Assert.Equal("core://scheduler/jobs", payload.GetProperty("endpoint").GetString());
                Assert.False(payload.TryGetProperty("query", out _)); Assert.False(payload.TryGetProperty("headers", out _));
                var body = payload.GetProperty("body");
                Assert.False(body.TryGetProperty("tenant_uuid", out _));
                operation = body.GetProperty("operation").GetString()!;
                Assert.Equal("com.corex.scheduler.jobs.service_" + (operation == "delete_job" ? "delete" : "read"), root.GetProperty("capability_id").GetString());
                if (operation == "delete_job") Assert.Equal(7UL, body.GetProperty("expected_revision").GetUInt64());
            }
            operations.Add(operation);
            object data = operation switch
            {
                "list_jobs" => new { items = new[] { Job() }, pagination = new { total = 1 } },
                "get_job" => new { item = Job() },
                "list_runs" => new { items = new[] { new { uuid = Tenant, job_uuid = Id, status = "triggered", trigger_source = "manual", fired_at = "2026-10-08T00:00:00Z", event_id = "event-fixture" } } },
                _ => scheme == "apikey" ? Receipt() : new { result = Receipt() }
            };
            return Response(scheme == "apikey" ? data : new { status = "completed", protocol_used = "core_internal", fallback_used = false, payload = data });
        });
        var client = Client(handler, scheme);
        var listed = await client.ListJobsAsync("browser-tenant-must-not-forward", "plugin", Owner, "completed");
        Assert.Equal(Id, listed.Single().JobId); Assert.Equal(7, listed[0].Revision);
        Assert.Equal("framework_lab_scheduler_probe", listed[0].Payload!["business_action"]!.ToString());
        Assert.Equal(Id, (await client.GetJobAsync(Id, "browser-tenant"))!.Uuid);
        Assert.Single(await client.ListRunsAsync(Id));
        var deletion = await ((ISchedulerCleanup)client).DeleteJobAsync(new(Id, 7));
        Assert.True(deletion.Deleted); Assert.True(deletion.HistoryRetained); Assert.Equal("core-trace", deletion.TraceId);
        Assert.Equal(new[] { "list_jobs", "get_job", "list_runs", "delete_job" }, operations);
    }

    [Theory]
    [InlineData(409, "SCHEDULER_JOB_VERSION_CONFLICT")]
    [InlineData(403, "SCHEDULER_CAPABILITY_FORBIDDEN")]
    [InlineData(404, "SCHEDULER_JOB_NOT_FOUND")]
    public async Task Denial_conflict_and_trace_are_preserved_without_retry_or_refresh(int status, string code)
    {
        var calls = 0;
        var client = Client(new Handler(_ =>
        {
            calls++;
            return Task.FromResult(new HttpResponseMessage((HttpStatusCode)status) { Content = JsonContent.Create(new { code = status, reason_code = code, request_id = "request-fixture", trace_id = "trace-fixture" }) });
        }));
        var error = await Assert.ThrowsAsync<SchedulerAdapterException>(() => client.DeleteJobAsync(new(Id, 7)));
        Assert.Equal(code, error.Code); Assert.Equal(status, error.UpstreamStatusCode);
        Assert.Equal("request-fixture", error.UpstreamRequestId); Assert.Equal("trace-fixture", error.UpstreamTraceId); Assert.Equal(1, calls);
    }

    [Fact]
    public async Task Missing_version_or_mismatched_receipt_cannot_become_success()
    {
        var calls = 0;
        var client = Client(new Handler(_ => { calls++; return Task.FromResult(Response(Receipt(jobId: Tenant))); }));
        await Assert.ThrowsAsync<SchedulerAdapterException>(() => client.DeleteJobAsync(new(Id, 0)));
        Assert.Equal(0, calls);
        var error = await Assert.ThrowsAsync<SchedulerAdapterException>(() => client.DeleteJobAsync(new(Id, 7)));
        Assert.Equal(SchedulerErrors.CodeUpstreamDependency, error.Code);
    }

    [Fact]
    public async Task Repeated_delete_preserves_already_deleted_receipt()
    {
        var client = Client(new Handler(_ => Task.FromResult(Response(Receipt(alreadyDeleted: true)))));
        Assert.True((await client.DeleteJobAsync(new(Id, 7))).AlreadyDeleted);
    }

    [Fact]
    public async Task Preview_collects_every_page_and_rejects_duplicate_or_unversioned_records()
    {
        var pages = new List<string>();
        var client = Client(new Handler(request =>
        {
            pages.Add(request.RequestUri!.Query);
            return Task.FromResult(Response(new { items = new[] { Job(pages.Count == 1 ? Id : Tenant) }, pagination = new { total = 2 } }));
        }));
        Assert.Equal(2, (await client.ListJobsAsync(ownerType: "plugin", ownerId: Owner)).Count);
        Assert.Contains("page=2", pages[1]);
        client = Client(new Handler(_ => Task.FromResult(Response(new { items = new[] { Job() }, pagination = new { total = 2 } }))));
        await Assert.ThrowsAsync<SchedulerAdapterException>(() => client.ListJobsAsync(ownerType: "plugin", ownerId: Owner));
        client = Client(new Handler(_ => Task.FromResult(Response(new { item = Job(revision: 0) }))));
        await Assert.ThrowsAsync<SchedulerAdapterException>(() => client.GetJobAsync(Id, ""));
    }

    [Fact]
    public async Task Typed_operation_rejects_fallback_even_with_a_valid_receipt()
    {
        var client = Client(new Handler(_ => Task.FromResult(Response(new { status = "completed", protocol_used = "http", fallback_used = true, payload = new { result = Receipt() } }))), "Bearer");
        await Assert.ThrowsAsync<SchedulerAdapterException>(() => client.DeleteJobAsync(new(Id, 7)));
    }

    [Fact]
    public async Task A_missing_service_route_is_not_reported_as_a_missing_job_or_retried_on_admin()
    {
        var calls = 0;
        var client = Client(new Handler(request =>
        {
            calls++; Assert.Contains("/tenant/scheduler/", request.RequestUri!.AbsolutePath);
            return Task.FromResult(new HttpResponseMessage(HttpStatusCode.NotFound) { Content = new StringContent("404 page not found") });
        }));
        var error = await Assert.ThrowsAsync<SchedulerAdapterException>(() => client.GetJobAsync(Id, ""));
        Assert.Equal(SchedulerErrors.CodeServiceRouteUnavailable, error.Code); Assert.Equal(404, error.UpstreamStatusCode); Assert.Equal(1, calls);
        client = Client(new Handler(_ => Task.FromResult(new HttpResponseMessage(HttpStatusCode.NotFound)
            { Content = JsonContent.Create(new { code = 404, reason_code = "SCHEDULER_JOB_NOT_FOUND", request_id = "core-request" }) })));
        Assert.Null(await client.GetJobAsync(Id, ""));
    }

    private static object Job(string id = Id, long revision = 7) => new { job_uuid = id, tenant_uuid = Tenant, owner_type = "plugin", owner_id = Owner,
        name = "framework_lab_once_test", status = "completed", revision, schedule_type = "once", schedule_expr = "2030-01-01T00:00:00Z",
        created_at = "2026-10-08T00:00:00Z", updated_at = "2026-10-08T00:00:00Z", payload = new { business_action = "framework_lab_scheduler_probe", source = "framework-lab" } };
    private static object Receipt(bool alreadyDeleted = false, string jobId = Id) => new { schema = "powerx.scheduler.job-deletion/v1", job_uuid = jobId,
        status = "deleted", deleted = true, already_deleted = alreadyDeleted, revision = 8, deleted_at = "2026-10-08T00:00:00Z", history_retained = true, trace_id = "core-trace" };
    private static HttpResponseMessage Response(object data) => new(HttpStatusCode.OK) { Content = JsonContent.Create(new { code = 200, data }) };
    private static PowerXSchedulerClient Client(HttpMessageHandler handler, string scheme = "apikey") => new(new() { BaseUrl = "https://core.example.test", Credential = "service-fixture", AuthScheme = scheme }, new HttpClient(handler));
    private sealed class Handler(Func<HttpRequestMessage, Task<HttpResponseMessage>> send) : HttpMessageHandler
    { protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken ct) => send(request); }
}
