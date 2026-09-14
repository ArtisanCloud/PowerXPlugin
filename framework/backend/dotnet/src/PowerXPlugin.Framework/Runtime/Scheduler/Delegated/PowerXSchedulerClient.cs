using System.Net;
using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;

namespace PowerXPlugin.Framework.Runtime.Scheduler.Delegated;

/// <summary>
/// Framework-owned delegated Scheduler transport. Calls are made as the
/// configured plugin service actor; an incoming administrator/user bearer is
/// never accepted or forwarded by this client.
/// </summary>
public sealed class PowerXSchedulerClient : IScheduler
{
    private const string JobsPath = "/admin/scheduler/jobs";
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
        return await SendJobAsync(HttpMethod.Patch, $"{JobsPath}/{Uri.EscapeDataString(spec.JobId!.Trim())}", ToHostPayload(spec), ct);
    }

    public async Task<bool> PauseJobAsync(string jobId, string tenantUuid, CancellationToken ct = default)
        => await SendActionAsync(jobId, "pause", ct);

    public async Task<bool> ResumeJobAsync(string jobId, string tenantUuid, CancellationToken ct = default)
        => await SendActionAsync(jobId, "resume", ct);

    public async Task<bool> TriggerJobAsync(string jobId, string tenantUuid, CancellationToken ct = default)
        => await SendActionAsync(jobId, "trigger", ct);

    public async Task<Job?> GetJobAsync(string jobId, string tenantUuid, CancellationToken ct = default)
    {
        if (string.IsNullOrWhiteSpace(jobId)) throw new SchedulerAdapterException(SchedulerErrors.CodeInvalidJob);
        using var response = await SendAsync(HttpMethod.Get, $"{JobsPath}/{Uri.EscapeDataString(jobId.Trim())}", null, ct);
        if (response.StatusCode == HttpStatusCode.NotFound) return null;
        return DeserializeJob(await ReadBodyAsync(response, ct));
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
        var path = query.Count == 0 ? JobsPath : JobsPath + "?" + string.Join("&", query);
        using var response = await SendAsync(HttpMethod.Get, path, null, ct);
        return DeserializeJobs(await ReadBodyAsync(response, ct));
    }

    private async Task<Job> SendJobAsync(HttpMethod method, string path, object payload, CancellationToken ct)
    {
        using var response = await SendAsync(method, path, payload, ct);
        var job = DeserializeJob(await ReadBodyAsync(response, ct));
        if (string.IsNullOrWhiteSpace(job.JobId)) job.JobId = job.Uuid;
        if (string.IsNullOrWhiteSpace(job.JobId)) throw new SchedulerAdapterException(SchedulerErrors.CodeUpstreamDependency);
        return job;
    }

    private async Task<bool> SendActionAsync(string jobId, string action, CancellationToken ct)
    {
        if (string.IsNullOrWhiteSpace(jobId)) throw new SchedulerAdapterException(SchedulerErrors.CodeInvalidJob);
        using var response = await SendAsync(HttpMethod.Post, $"{JobsPath}/{Uri.EscapeDataString(jobId.Trim())}/{action}", null, ct);
        return response.IsSuccessStatusCode;
    }

    private async Task<HttpResponseMessage> SendAsync(HttpMethod method, string path, object? body, CancellationToken ct)
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
            if (response.IsSuccessStatusCode || response.StatusCode == HttpStatusCode.NotFound) return response;
            var status = response.StatusCode;
            response.Dispose();
            throw new SchedulerAdapterException(MapStatus(status), (int)status);
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
        using var document = JsonDocument.Parse(text);
        var root = document.RootElement;
        if (root.ValueKind == JsonValueKind.Object && root.TryGetProperty("success", out var success) && success.ValueKind == JsonValueKind.False)
            throw new SchedulerAdapterException(SchedulerErrors.CodeUpstreamDependency);
        return text;
    }

    private static Job DeserializeJob(string raw)
    {
        using var document = JsonDocument.Parse(raw);
        var payload = Unwrap(document.RootElement);
        if (payload.ValueKind == JsonValueKind.Object && payload.TryGetProperty("job", out var job)) payload = job;
        return JsonSerializer.Deserialize<Job>(payload.GetRawText(), JsonOptions)
            ?? throw new SchedulerAdapterException(SchedulerErrors.CodeUpstreamDependency);
    }

    private static List<Job> DeserializeJobs(string raw)
    {
        using var document = JsonDocument.Parse(raw);
        var payload = Unwrap(document.RootElement);
        if (payload.ValueKind == JsonValueKind.Object && payload.TryGetProperty("items", out var items)) payload = items;
        return JsonSerializer.Deserialize<List<Job>>(payload.GetRawText(), JsonOptions)
            ?? throw new SchedulerAdapterException(SchedulerErrors.CodeUpstreamDependency);
    }

    private static JsonElement Unwrap(JsonElement root) =>
        root.ValueKind == JsonValueKind.Object && root.TryGetProperty("data", out var data) ? data : root;

    private static object ToHostPayload(JobSpec spec) => new
    {
        job_id = Clean(spec.JobId),
        owner_type = Clean(spec.OwnerType),
        owner_id = Clean(spec.OwnerId),
        name = Clean(spec.Name),
        schedule_type = Clean(spec.ScheduleType).ToLowerInvariant(),
        schedule_expr = Clean(spec.ScheduleExpr),
        timezone = Clean(spec.Timezone),
        topic = string.IsNullOrWhiteSpace(spec.Topic) ? "powerx.runtime.scheduler.triggered.v1" : spec.Topic.Trim(),
        payload = spec.Payload ?? new Dictionary<string, object?>(),
        paused = spec.Paused,
        retry_policy = spec.Retry
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
        _ => SchedulerErrors.CodeUpstreamDependency
    };
}

public sealed class PowerXSchedulerClientOptions
{
    public string BaseUrl { get; init; } = string.Empty;
    public string ApiPrefix { get; init; } = "/api/v1";
    public string Credential { get; init; } = string.Empty;
    public string AuthScheme { get; init; } = "Bearer";
    public string? UserAgent { get; init; }
    public TimeSpan Timeout { get; init; } = TimeSpan.FromSeconds(10);
}
