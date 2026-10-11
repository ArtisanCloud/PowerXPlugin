using System.Net;
using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;

namespace PowerXPlugin.Framework.Runtime.TaskQueue.Delegated;

/// <summary>Typed .NET parity implementation of Go runtime/taskqueue HostProvider.</summary>
public sealed class PowerXTaskQueueClient : ITaskQueue
{
    private const string BasePath = "/admin/runtime/task-queue";
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web);
    private readonly PowerXTaskQueueClientOptions _options;
    private readonly HttpClient _http;

    public PowerXTaskQueueClient(PowerXTaskQueueClientOptions options, HttpClient? httpClient = null)
    {
        _options = options ?? throw new ArgumentNullException(nameof(options));
        _http = httpClient ?? new HttpClient { Timeout = options.Timeout > TimeSpan.Zero ? options.Timeout : TimeSpan.FromSeconds(30) };
    }

    public Task EnqueueAsync(TaskQueueMessage message, CancellationToken ct = default)
    {
        ValidateMessage(message);
        return SendAsync("enqueue", new { message = ToHostMessage(message) }, ct);
    }

    public async Task<IReadOnlyList<TaskQueueMessage>> DequeueAsync(TaskQueueDequeueRequest request, CancellationToken ct = default)
    {
        ValidateTenant(request.TenantKey);
        if (string.IsNullOrWhiteSpace(request.SubscriberId)) throw new TaskQueueAdapterException(TaskQueueErrors.CodeInvalidRequest);
        var payload = await SendForDataAsync("dequeue", new
        {
            tenant_key = TrustedTenant(), subscriber_id = request.SubscriberId.Trim(),
            max_items = Math.Clamp(request.MaxItems, 1, 200),
            wait_timeout_ms = Math.Max(0, (long)request.WaitTimeout.TotalMilliseconds)
        }, ct);
        if (!payload.TryGetProperty("messages", out var messages) || messages.ValueKind != JsonValueKind.Array) return [];
        return messages.EnumerateArray().Select(ParseMessage).ToList();
    }

    public Task AckAsync(TaskQueueAckRequest request, CancellationToken ct = default)
    {
        ValidateAck(request.TenantKey, request.SubscriberId, request.MessageId);
        return SendAsync("ack", new { tenant_key = TrustedTenant(), subscriber_id = request.SubscriberId.Trim(), message_id = request.MessageId.Trim(), metadata = request.Metadata }, ct);
    }

    public Task NackAsync(TaskQueueNackRequest request, CancellationToken ct = default)
    {
        ValidateAck(request.TenantKey, request.SubscriberId, request.MessageId);
        return SendAsync("nack", new { tenant_key = TrustedTenant(), subscriber_id = request.SubscriberId.Trim(), message_id = request.MessageId.Trim(), reason = request.Reason?.Trim(), retry_at = request.RetryAt?.ToUniversalTime(), metadata = request.Metadata }, ct);
    }

    public Task RetryAsync(TaskQueueRetryRequest request, CancellationToken ct = default)
    {
        ValidateMessage(request.Message);
        return SendAsync("retry", new { message = ToHostMessage(request.Message), retry_at = request.RetryAt?.ToUniversalTime(), reason = request.Reason?.Trim() }, ct);
    }

    private async Task SendAsync(string action, object body, CancellationToken ct) => _ = await SendForDataAsync(action, body, ct);

    private async Task<JsonElement> SendForDataAsync(string action, object body, CancellationToken ct)
    {
        EnsureConfigured();
        using var request = new HttpRequestMessage(HttpMethod.Post, Endpoint(action));
        request.Headers.Accept.Add(new MediaTypeWithQualityHeaderValue("application/json"));
        request.Headers.Authorization = new AuthenticationHeaderValue(NormalizeAuthScheme(_options.AuthScheme), _options.Credential.Trim());
        request.Headers.TryAddWithoutValidation("tenant_uuid", TrustedTenant());
        if (!string.IsNullOrWhiteSpace(_options.UserAgent)) request.Headers.UserAgent.ParseAdd(_options.UserAgent.Trim());
        request.Content = new StringContent(JsonSerializer.Serialize(body, JsonOptions), Encoding.UTF8, "application/json");
        try
        {
            using var response = await _http.SendAsync(request, ct);
            var raw = await response.Content.ReadAsStringAsync(ct);
            if (!response.IsSuccessStatusCode) throw new TaskQueueAdapterException(MapStatus(response.StatusCode), (int)response.StatusCode);
            using var document = JsonDocument.Parse(string.IsNullOrWhiteSpace(raw) ? "{}" : raw);
            var root = document.RootElement;
            if (root.TryGetProperty("success", out var success) && success.ValueKind == JsonValueKind.False)
                throw new TaskQueueAdapterException(TaskQueueErrors.CodeUpstreamDependency);
            return (root.TryGetProperty("data", out var data) ? data : root).Clone();
        }
        catch (TaskQueueAdapterException) { throw; }
        catch (OperationCanceledException) when (!ct.IsCancellationRequested) { throw new TaskQueueAdapterException(TaskQueueErrors.CodeUpstreamDependency); }
        catch (HttpRequestException exception) { throw new TaskQueueAdapterException(TaskQueueErrors.CodeUpstreamDependency, innerException: exception); }
    }

    private static object ToHostMessage(TaskQueueMessage message) => new
    {
        id = message.Id.Trim(), tenant_key = message.TenantKey.Trim(), subscriber_id = message.SubscriberId.Trim(), topic = message.Topic.Trim(),
        payload_base64 = Convert.ToBase64String(message.Payload), headers = message.Headers, attempt = message.Attempt,
        trace_id = message.TraceId?.Trim(), visible_at = message.VisibleAt?.ToUniversalTime(), metadata = message.Metadata,
        idempotency_key = message.IdempotencyKey?.Trim()
    };

    private TaskQueueMessage ParseMessage(JsonElement item)
    {
        var payload = item.TryGetProperty("payload_base64", out var encoded) && encoded.ValueKind == JsonValueKind.String
            ? Convert.FromBase64String(encoded.GetString() ?? string.Empty) : [];
        return new TaskQueueMessage(
            String(item, "id"), String(item, "tenant_key"), String(item, "subscriber_id"), String(item, "topic"), payload,
            Strings(item, "headers"), Integer(item, "attempt"), NullableString(item, "trace_id"), NullableTime(item, "visible_at"),
            Strings(item, "metadata"), NullableString(item, "idempotency_key"));
    }

    private void ValidateMessage(TaskQueueMessage message)
    {
        ValidateTenant(message.TenantKey);
        if (string.IsNullOrWhiteSpace(message.Id) || string.IsNullOrWhiteSpace(message.SubscriberId) || string.IsNullOrWhiteSpace(message.Topic))
            throw new TaskQueueAdapterException(TaskQueueErrors.CodeInvalidRequest);
    }
    private void ValidateAck(string tenantKey, string subscriberId, string messageId)
    {
        ValidateTenant(tenantKey);
        if (string.IsNullOrWhiteSpace(subscriberId) || string.IsNullOrWhiteSpace(messageId)) throw new TaskQueueAdapterException(TaskQueueErrors.CodeInvalidRequest);
    }
    private void ValidateTenant(string tenantKey)
    {
        if (string.IsNullOrWhiteSpace(tenantKey) || !string.Equals(tenantKey.Trim(), TrustedTenant(), StringComparison.Ordinal))
            throw new TaskQueueAdapterException(TaskQueueErrors.CodeTenantMismatch);
    }
    private void EnsureConfigured()
    {
        if (string.IsNullOrWhiteSpace(_options.BaseUrl) || string.IsNullOrWhiteSpace(_options.Credential) || string.IsNullOrWhiteSpace(_options.TenantUuid))
            throw new TaskQueueAdapterException(TaskQueueErrors.CodeUnavailable);
    }
    private string TrustedTenant() => _options.TenantUuid.Trim();
    private string Endpoint(string action) => NormalizeBaseUrl(_options.BaseUrl) + "/" + (_options.ApiPrefix ?? "api/v1").Trim('/') + BasePath + "/" + action;
    private static string NormalizeBaseUrl(string value) => value.Trim().TrimEnd('/').StartsWith("http", StringComparison.OrdinalIgnoreCase) ? value.Trim().TrimEnd('/') : "http://" + value.Trim().TrimEnd('/');
    private static string NormalizeAuthScheme(string value) => value.Trim().Equals("apikey", StringComparison.OrdinalIgnoreCase) ? "ApiKey" : "Bearer";
    private static string MapStatus(HttpStatusCode status) => status switch { HttpStatusCode.Unauthorized => TaskQueueErrors.CodeUnauthorized, HttpStatusCode.Forbidden => TaskQueueErrors.CodeForbidden, _ => TaskQueueErrors.CodeUpstreamDependency };
    private static string String(JsonElement item, string key) => NullableString(item, key) ?? string.Empty;
    private static string? NullableString(JsonElement item, string key) => item.TryGetProperty(key, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static int Integer(JsonElement item, string key) => item.TryGetProperty(key, out var value) && value.TryGetInt32(out var result) ? result : 0;
    private static DateTime? NullableTime(JsonElement item, string key) => item.TryGetProperty(key, out var value) && value.ValueKind == JsonValueKind.String && DateTime.TryParse(value.GetString(), out var result) ? result : null;
    private static Dictionary<string, string>? Strings(JsonElement item, string key) => item.TryGetProperty(key, out var value) && value.ValueKind == JsonValueKind.Object ? value.EnumerateObject().ToDictionary(x => x.Name, x => x.Value.GetString() ?? string.Empty) : null;
}

public sealed class PowerXTaskQueueClientOptions
{
    public string BaseUrl { get; init; } = string.Empty;
    public string ApiPrefix { get; init; } = "/api/v1";
    public string Credential { get; init; } = string.Empty;
    public string AuthScheme { get; init; } = "Bearer";
    public string TenantUuid { get; init; } = string.Empty;
    public string? UserAgent { get; init; }
    public TimeSpan Timeout { get; init; } = TimeSpan.FromSeconds(30);
}
