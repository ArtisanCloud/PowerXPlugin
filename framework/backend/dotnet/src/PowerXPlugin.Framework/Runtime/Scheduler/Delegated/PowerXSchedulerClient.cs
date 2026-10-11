using System.Net;
using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;

namespace PowerXPlugin.Framework.Runtime.Scheduler.Delegated;

/// <summary>
/// Framework-owned delegated Scheduler transport. Calls are made as the
/// configured plugin service actor; an incoming administrator/user bearer is
/// never accepted or forwarded by this client.
/// </summary>
public sealed class PowerXSchedulerClient : IScheduler, ISchedulerCleanup
{
    private const string JobsPath = "/admin/scheduler/jobs";
    private const string ServiceJobsPath = "/tenant/scheduler/jobs";
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web);
    private readonly HttpClient _http;
    private readonly PowerXSchedulerClientOptions _options;

    public PowerXSchedulerClient(PowerXSchedulerClientOptions options, HttpClient? httpClient = null)
    {
        _options = options ?? throw new ArgumentNullException(nameof(options));
        _http = httpClient ?? new HttpClient();
        if (httpClient == null && options.Timeout > TimeSpan.Zero) _http.Timeout = options.Timeout;
    }

    public async Task<Job> CreateJobAsync(JobSpec spec, CancellationToken ct = default)
    {
        ValidateSpec(spec, requireJobId: false);
        return await SendJobAsync(HttpMethod.Post, JobsPath, ToHostPayload(spec), ct);
    }

    public async Task<Job?> UpdateJobAsync(JobSpec spec, CancellationToken ct = default)
    {
        ValidateSpec(spec, requireJobId: true);
        return await SendJobAsync(HttpMethod.Patch, $"{JobsPath}/{RequireJobId(spec.JobId!)}", ToHostPayload(spec), ct);
    }

    public async Task<bool> PauseJobAsync(string jobId, string tenantUuid, CancellationToken ct = default)
        => await SendActionAsync(jobId, "pause", ct);

    public async Task<bool> ResumeJobAsync(string jobId, string tenantUuid, CancellationToken ct = default)
        => await SendActionAsync(jobId, "resume", ct);

    public async Task<bool> TriggerJobAsync(string jobId, string tenantUuid, CancellationToken ct = default)
        => (await TriggerWithRunAsync(jobId, ct)).Run.Status == "triggered";

    public async Task<PowerXSchedulerTriggerResult> TriggerWithRunAsync(string jobId, CancellationToken ct = default)
    {
        jobId = RequireJobId(jobId);
        using var response = await SendAsync(HttpMethod.Post, $"{JobsPath}/{Uri.EscapeDataString(jobId)}/trigger", null, ct);
        using var document = JsonDocument.Parse(await ReadBodyAsync(response, ct));
        var data = Unwrap(document.RootElement);
        if (!data.TryGetProperty("job", out var job) || !data.TryGetProperty("run", out var run)) throw InvalidResponse();
        var normalized = DeserializeJob(job);
        if (normalized.Uuid != jobId) throw InvalidResponse();
        return new(normalized, DeserializeRun(run, jobId));
    }

    public async Task<List<PowerXSchedulerRun>> ListRunsAsync(string jobId, CancellationToken ct = default)
    {
        jobId = RequireJobId(jobId);
        var data = await ServiceAsync("list_runs", $"{ServiceJobsPath}/{jobId}/runs?page=1&page_size=50",
            new { operation = "list_runs", job_uuid = jobId, page = 1, page_size = 50 }, ct);
        if (!data.TryGetProperty("items", out var items) || items.ValueKind != JsonValueKind.Array) throw InvalidResponse();
        return items.EnumerateArray().Select(item => DeserializeRun(item, jobId)).ToList();
    }

    public async Task<Job?> GetJobAsync(string jobId, string tenantUuid, CancellationToken ct = default)
        => await GetCleanupJobAsync(jobId, false, ct);

    public async Task<Job?> GetCleanupJobAsync(string jobId, bool includeDeleted = false, CancellationToken ct = default)
    {
        jobId = RequireJobId(jobId);
        try
        {
            var data = await ServiceAsync("get_job", $"{ServiceJobsPath}/{jobId}?include_deleted={includeDeleted.ToString().ToLowerInvariant()}",
                new { operation = "get_job", job_uuid = jobId, include_deleted = includeDeleted }, ct);
            if (!data.TryGetProperty("item", out var item)) throw InvalidResponse();
            var job = DeserializeServiceJob(item);
            if (job.JobId != jobId || !includeDeleted && job.Status == "deleted") throw InvalidResponse();
            return job;
        }
        catch (SchedulerAdapterException ex) when (ex.UpstreamStatusCode == 404 && ex.Code != SchedulerErrors.CodeServiceRouteUnavailable) { return null; }
    }

    public async Task<List<Job>> ListJobsAsync(string? tenantUuid = null, string? ownerType = null, string? ownerId = null, string? status = null, CancellationToken ct = default)
    {
        // tenantUuid is deliberately not sent. The host derives tenant scope
        // from the trusted configured service actor, as it does for all job
        // operations; a plugin cannot query another tenant through a parameter.
        var query = new List<string>();
        AddQuery(query, "owner_type", ownerType);
        AddQuery(query, "owner_id", ownerId);
        AddQuery(query, "status", status);
        var jobs = new List<Job>();
        var page = 1;
        while (true)
        {
            var path = ServiceJobsPath + "?" + string.Join("&", query.Append($"page={page}&page_size=100"));
            var body = new Dictionary<string, object?> { ["operation"] = "list_jobs", ["owner_type"] = ownerType ?? "plugin", ["owner_id"] = ownerId, ["page"] = page, ["page_size"] = 100 };
            if (!string.IsNullOrWhiteSpace(status)) body["status"] = status;
            var data = await ServiceAsync("list_jobs", path, body, ct);
            if (!data.TryGetProperty("items", out var items) || items.ValueKind != JsonValueKind.Array
                || !data.TryGetProperty("pagination", out var pagination) || pagination.ValueKind != JsonValueKind.Object || !pagination.TryGetProperty("total", out var totalValue)
                || totalValue.ValueKind != JsonValueKind.Number || !totalValue.TryGetInt32(out var total) || total < 0) throw InvalidResponse();
            var batch = items.EnumerateArray().Select(DeserializeServiceJob).ToArray();
            if (jobs.Count + batch.Length > total || batch.Any(job => ownerType != null && job.OwnerType != ownerType
                || ownerId != null && job.OwnerId != ownerId || status != null && job.Status != status)) throw InvalidResponse();
            if (batch.Any(job => jobs.Any(existing => existing.Uuid == job.Uuid)) || batch.Select(job => job.Uuid).Distinct().Count() != batch.Length) throw InvalidResponse();
            jobs.AddRange(batch);
            if (jobs.Count >= total) return jobs;
            if (batch.Length == 0) throw InvalidResponse();
            page++;
        }
    }

    public async Task<SchedulerJobDeletion> DeleteJobAsync(SchedulerDeleteRequest request, CancellationToken ct = default)
    {
        var jobId = RequireJobId(request.JobUuid);
        if (request.ExpectedRevision == 0) throw new SchedulerAdapterException(SchedulerErrors.CodeInvalidJob);
        var data = await ServiceAsync("delete_job", $"{ServiceJobsPath}/{jobId}?expected_revision={request.ExpectedRevision}",
            new { operation = "delete_job", job_uuid = jobId, expected_revision = request.ExpectedRevision }, ct);
        // REST returns the receipt directly; the typed operation wraps it in result.
        if (data.TryGetProperty("result", out var receipt)) data = receipt;
        try
        {
            var result = data.Deserialize<SchedulerJobDeletion>(JsonOptions) ?? throw InvalidResponse();
            if (result.Schema != "powerx.scheduler.job-deletion/v1" || result.JobUuid != jobId || result.Status != "deleted"
                || !result.Deleted || !result.HistoryRetained || result.Revision == 0 || result.DeletedAt == default
                || string.IsNullOrWhiteSpace(result.TraceId) || !result.AlreadyDeleted && result.Revision <= request.ExpectedRevision) throw InvalidResponse();
            return result;
        }
        catch (JsonException ex) { throw new SchedulerAdapterException(SchedulerErrors.CodeUpstreamDependency, innerException: ex); }
    }

    private async Task<JsonElement> ServiceAsync(string operation, string restPath, object body, CancellationToken ct)
    {
        var simulation = _options.AuthScheme.Trim().Equals("apikey", StringComparison.OrdinalIgnoreCase);
        // API Key Host simulation uses tenant REST. STS uses only the fixed typed binding.
        using var response = await SendAsync(simulation ? operation == "delete_job" ? HttpMethod.Delete : HttpMethod.Get : HttpMethod.Post,
            simulation ? restPath : "/tenant/invocations", simulation ? null : new
            {
                capability_id = operation == "delete_job" ? "com.corex.scheduler.jobs.service_delete" : "com.corex.scheduler.jobs.service_read",
                preferred_protocol = "core_internal",
                payload = new { method = "INVOKE", endpoint = "core://scheduler/jobs", body }
            }, ct);
        using var document = JsonDocument.Parse(await ReadBodyAsync(response, ct));
        var data = Unwrap(document.RootElement);
        if (simulation) return data.Clone();
        if (!data.TryGetProperty("status", out var state) || state.ValueKind != JsonValueKind.String || state.GetString() != "completed"
            || !data.TryGetProperty("protocol_used", out var protocol) || protocol.ValueKind != JsonValueKind.String || protocol.GetString() != "core_internal"
            || !data.TryGetProperty("fallback_used", out var fallback) || fallback.ValueKind != JsonValueKind.False
            || !data.TryGetProperty("payload", out var payload) || payload.ValueKind != JsonValueKind.Object) throw InvalidResponse();
        return payload.Clone();
    }

    private static Job DeserializeServiceJob(JsonElement payload)
    {
        try
        {
            if (!payload.TryGetProperty("job_uuid", out var id) || !Guid.TryParse(id.GetString(), out var uuid) || uuid == Guid.Empty
                || !payload.TryGetProperty("revision", out var revision) || !revision.TryGetInt64(out var version) || version <= 0
                || !payload.TryGetProperty("payload", out var body) || body.ValueKind != JsonValueKind.Object
                || !payload.TryGetProperty("created_at", out var created) || created.ValueKind != JsonValueKind.String
                || !payload.TryGetProperty("updated_at", out var updated) || updated.ValueKind != JsonValueKind.String) throw InvalidResponse();
            var job = payload.Deserialize<Job>(JsonOptions) ?? throw InvalidResponse();
            if (!Guid.TryParse(job.TenantUuid, out var tenant) || tenant == Guid.Empty || string.IsNullOrWhiteSpace(job.OwnerId)
                || job.OwnerType is not ("plugin" or "core") || !payload.TryGetProperty("status", out _)
                || job.Status is not ("active" or "paused" or "completed" or "deleted") || string.IsNullOrWhiteSpace(job.Name)
                || job.CreatedAt == default || job.UpdatedAt == default) throw InvalidResponse();
            job.JobId = job.Uuid = uuid.ToString("D"); job.Revision = version;
            return job;
        }
        catch (Exception ex) when (ex is JsonException or InvalidOperationException)
        { throw new SchedulerAdapterException(SchedulerErrors.CodeUpstreamDependency, innerException: ex); }
    }

    private async Task<Job> SendJobAsync(HttpMethod method, string path, object payload, CancellationToken ct)
    {
        using var response = await SendAsync(method, path, payload, ct);
        var job = DeserializeJob(await ReadBodyAsync(response, ct));
        return job;
    }

    private async Task<bool> SendActionAsync(string jobId, string action, CancellationToken ct)
    {
        if (string.IsNullOrWhiteSpace(jobId)) throw new SchedulerAdapterException(SchedulerErrors.CodeInvalidJob);
        jobId = RequireJobId(jobId);
        using var response = await SendAsync(HttpMethod.Post, $"{JobsPath}/{Uri.EscapeDataString(jobId.Trim())}/{action}", null, ct);
        var job = DeserializeJob(await ReadBodyAsync(response, ct));
        if (job.Uuid != jobId) throw InvalidResponse();
        return true;
    }

    private async Task<HttpResponseMessage> SendAsync(HttpMethod method, string path, object? body, CancellationToken ct, bool allowNotFound = false)
    {
        var baseUrl = NormalizeBaseUrl(_options.BaseUrl);
        if (string.IsNullOrWhiteSpace(baseUrl) || string.IsNullOrWhiteSpace(_options.Credential))
            throw new SchedulerAdapterException(SchedulerErrors.CodeUnavailable);
        using var request = new HttpRequestMessage(method, baseUrl + NormalizeApiPrefix(_options.ApiPrefix) + path);
        request.Headers.Accept.Add(new MediaTypeWithQualityHeaderValue("application/json"));
        request.Headers.Authorization = new AuthenticationHeaderValue(NormalizeAuthScheme(_options.AuthScheme), _options.Credential.Trim());
        if (!string.IsNullOrWhiteSpace(_options.UserAgent)) request.Headers.UserAgent.ParseAdd(_options.UserAgent.Trim());
        if (body != null) request.Content = new StringContent(JsonSerializer.Serialize(body, JsonOptions), Encoding.UTF8, "application/json");

        try
        {
            var response = await _http.SendAsync(request, ct);
            if (response.IsSuccessStatusCode || allowNotFound && response.StatusCode == HttpStatusCode.NotFound) return response;
            var status = response.StatusCode;
            var error = await response.Content.ReadAsStringAsync(ct);
            response.Dispose();
            string? code = null, requestId = null, traceId = null;
            try
            {
                using var failure = JsonDocument.Parse(error);
                var root = failure.RootElement;
                foreach (var field in new[] { "reason_code", "error_code" })
                    if (root.TryGetProperty(field, out var value) && value.ValueKind == JsonValueKind.String) { code = value.GetString(); break; }
                if (root.TryGetProperty("request_id", out var upstreamRequest)) requestId = upstreamRequest.GetString();
                if (root.TryGetProperty("trace_id", out var trace)) traceId = trace.GetString();
            }
            catch (Exception ex) when (ex is JsonException or InvalidOperationException) { }
            if (status == HttpStatusCode.NotFound && string.IsNullOrWhiteSpace(code)
                && (path.StartsWith(ServiceJobsPath, StringComparison.Ordinal) || path == "/tenant/invocations"))
                code = SchedulerErrors.CodeServiceRouteUnavailable;
            throw new SchedulerAdapterException(string.IsNullOrWhiteSpace(code) ? MapStatus(status) : code, (int)status,
                upstreamRequestId: requestId, upstreamTraceId: traceId);
        }
        catch (SchedulerAdapterException) { throw; }
        catch (OperationCanceledException) when (!ct.IsCancellationRequested)
        {
            throw new SchedulerAdapterException(SchedulerErrors.CodeUpstreamDependency);
        }
        catch (HttpRequestException exception)
        {
            throw new SchedulerAdapterException(SchedulerErrors.CodeUpstreamDependency, innerException: exception);
        }
    }

    private static async Task<string> ReadBodyAsync(HttpResponseMessage response, CancellationToken ct)
    {
        var text = await response.Content.ReadAsStringAsync(ct);
        if (string.IsNullOrWhiteSpace(text)) throw new SchedulerAdapterException(SchedulerErrors.CodeUpstreamDependency);
        try
        {
            using var document = JsonDocument.Parse(text);
            var root = document.RootElement;
            if (root.ValueKind != JsonValueKind.Object || !root.TryGetProperty("code", out var code) || code.ValueKind != JsonValueKind.Number || !code.TryGetInt32(out var value)) throw InvalidResponse();
            if (value != 200) throw new SchedulerAdapterException(MapStatus((HttpStatusCode)value), value);
            if (!root.TryGetProperty("data", out var data) || data.ValueKind != JsonValueKind.Object) throw InvalidResponse();
        }
        catch (JsonException ex) { throw new SchedulerAdapterException(SchedulerErrors.CodeUpstreamDependency, innerException: ex); }
        return text;
    }

    private static Job DeserializeJob(string raw)
    {
        using var document = JsonDocument.Parse(raw);
        var payload = Unwrap(document.RootElement);
        if (payload.ValueKind == JsonValueKind.Object && payload.TryGetProperty("job", out var job)) payload = job;
        return DeserializeJob(payload);
    }

    private static Job DeserializeJob(JsonElement payload)
    {
        try
        {
            var job = payload.Deserialize<Job>(JsonOptions) ?? throw InvalidResponse();
            if (!Guid.TryParse(job.Uuid, out var uuid) || uuid == Guid.Empty || !payload.TryGetProperty("uuid", out _)
                || !Guid.TryParse(job.TenantUuid, out var tenant) || tenant == Guid.Empty
                || string.IsNullOrWhiteSpace(job.OwnerId) || !payload.TryGetProperty("owner_type", out _) || string.IsNullOrWhiteSpace(job.OwnerType)
                || !payload.TryGetProperty("status", out _) || job.Status is not ("active" or "paused" or "completed" or "deleted")
                || !payload.TryGetProperty("payload_json", out var body) || body.ValueKind != JsonValueKind.Object) throw InvalidResponse();
            job.JobId = job.Uuid = uuid.ToString("D");
            job.Payload = body.Deserialize<Dictionary<string, object?>>(JsonOptions);
            return job;
        }
        catch (JsonException ex) { throw new SchedulerAdapterException(SchedulerErrors.CodeUpstreamDependency, innerException: ex); }
    }

    private static PowerXSchedulerRun DeserializeRun(JsonElement value, string jobId)
    {
        try
        {
            var run = value.Deserialize<PowerXSchedulerRun>(JsonOptions) ?? throw InvalidResponse();
            if (!Guid.TryParse(run.Uuid, out var uuid) || uuid == Guid.Empty || run.JobUuid != jobId
                || run.Status is not ("triggered" or "failed" or "skipped")
                || run.Status == "triggered" && (run.FiredAt == null || string.IsNullOrWhiteSpace(run.EventId))) throw InvalidResponse();
            return run;
        }
        catch (JsonException ex) { throw new SchedulerAdapterException(SchedulerErrors.CodeUpstreamDependency, innerException: ex); }
    }

    private static SchedulerAdapterException InvalidResponse() => new(SchedulerErrors.CodeUpstreamDependency);
    private static string RequireJobId(string jobId)
    {
        if (!Guid.TryParse(jobId, out var uuid) || uuid == Guid.Empty) throw new SchedulerAdapterException(SchedulerErrors.CodeInvalidJob);
        return uuid.ToString("D");
    }

    private static JsonElement Unwrap(JsonElement root) =>
        root.ValueKind == JsonValueKind.Object && root.TryGetProperty("data", out var data) ? data : root;

    private static object ToHostPayload(JobSpec spec) => new
    {
        owner_type = Clean(spec.OwnerType),
        owner_id = Clean(spec.OwnerId),
        name = Clean(spec.Name),
        schedule_type = Clean(spec.ScheduleType).ToLowerInvariant(),
        schedule_expr = Clean(spec.ScheduleExpr),
        timezone = Clean(spec.Timezone),
        payload = spec.Payload ?? new Dictionary<string, object?>(),
        idempotency_key = Clean(spec.IdempotencyKey)
    };

    private static void ValidateSpec(JobSpec spec, bool requireJobId)
    {
        if (string.IsNullOrWhiteSpace(spec.Name)
            || string.IsNullOrWhiteSpace(spec.OwnerType)
            || string.IsNullOrWhiteSpace(spec.OwnerId)
            || string.IsNullOrWhiteSpace(spec.ScheduleType)
            || string.IsNullOrWhiteSpace(spec.ScheduleExpr)
            || requireJobId && string.IsNullOrWhiteSpace(spec.JobId))
            throw new SchedulerAdapterException(SchedulerErrors.CodeInvalidJob);
        if (spec.ScheduleType.Trim().ToLowerInvariant() is not (ScheduleType.Once or ScheduleType.Interval or ScheduleType.Cron))
            throw new SchedulerAdapterException(SchedulerErrors.CodeInvalidJob);
        if (spec.Paused || spec.Retry != null || spec.Topic is not (null or "powerx.runtime.scheduler.triggered.v1"))
            throw new SchedulerAdapterException(SchedulerErrors.CodeInvalidJob);
    }

    private static void AddQuery(ICollection<string> query, string key, string? value)
    {
        if (!string.IsNullOrWhiteSpace(value)) query.Add($"{key}={Uri.EscapeDataString(value.Trim())}");
    }

    private static string NormalizeBaseUrl(string value)
    {
        var baseUrl = Clean(value).TrimEnd('/');
        if (string.IsNullOrWhiteSpace(baseUrl)) return string.Empty;
        return baseUrl.StartsWith("http://", StringComparison.OrdinalIgnoreCase) || baseUrl.StartsWith("https://", StringComparison.OrdinalIgnoreCase)
            ? baseUrl
            : "http://" + baseUrl;
    }

    private static string NormalizeApiPrefix(string value) => "/" + (string.IsNullOrWhiteSpace(value) ? "api/v1" : value.Trim('/'));
    private static string NormalizeAuthScheme(string value) => value.Trim().Equals("apikey", StringComparison.OrdinalIgnoreCase) ? "ApiKey" : "Bearer";
    private static string Clean(string? value) => value?.Trim() ?? string.Empty;
    private static string MapStatus(HttpStatusCode status) => status switch
    {
        HttpStatusCode.Unauthorized => SchedulerErrors.CodeUnauthorized,
        HttpStatusCode.Forbidden => SchedulerErrors.CodeForbidden,
        HttpStatusCode.NotFound => SchedulerErrors.CodeNotFound,
        HttpStatusCode.Conflict => SchedulerErrors.CodeConflict,
        HttpStatusCode.BadRequest => SchedulerErrors.CodeInvalidJob,
        _ => SchedulerErrors.CodeUpstreamDependency
    };
}

public sealed record PowerXSchedulerTriggerResult(Job Job, PowerXSchedulerRun Run);
public sealed record PowerXSchedulerRun(
    [property: JsonPropertyName("uuid")] string Uuid,
    [property: JsonPropertyName("job_uuid")] string JobUuid,
    [property: JsonPropertyName("status")] string Status,
    [property: JsonPropertyName("trigger_source")] string TriggerSource,
    [property: JsonPropertyName("scheduled_at")] DateTime? ScheduledAt,
    [property: JsonPropertyName("fired_at")] DateTime? FiredAt,
    [property: JsonPropertyName("event_id")] string? EventId,
    [property: JsonPropertyName("trace_id")] string? TraceId,
    [property: JsonPropertyName("error_code")] string? ErrorCode,
    [property: JsonPropertyName("error_message")] string? ErrorMessage);

public sealed class PowerXSchedulerClientOptions
{
    public string BaseUrl { get; init; } = string.Empty;
    public string ApiPrefix { get; init; } = "/api/v1";
    public string Credential { get; init; } = string.Empty;
    public string AuthScheme { get; init; } = "Bearer";
    public string? UserAgent { get; init; }
    public TimeSpan Timeout { get; init; } = TimeSpan.FromSeconds(10);
}
