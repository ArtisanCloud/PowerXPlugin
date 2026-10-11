using System.Net;
using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;
using PowerXPlugin.Framework.Runtime.Common;

namespace PowerXPlugin.Framework.Runtime.WSBus;

public sealed record WSBusScope(string TenantUuid, string? MemberUuid, string TraceId);
public interface ILocalWSBus
{
    Task<int> PublishAsync(string topic, object payload, WSBusScope scope, CancellationToken ct = default);
}

/// <summary>Typed parity for Go WS Bus HostClient and notification test contract. Uses bootstrap service credentials only.</summary>
public sealed class PowerXWSBusClient(string baseUrl, string tenantUuid, IServiceCredentialProvider credentials, HttpClient http)
{
    private readonly string _base = baseUrl.TrimEnd('/').EndsWith("/api/v1", StringComparison.Ordinal) ? baseUrl.TrimEnd('/')[..^7] : baseUrl.TrimEnd('/');

    public async Task GrantAndPublishAsync(string topic, object payload, WSBusScope scope, CancellationToken ct = default)
    {
        var grant = await SendAsync("/api/v1/admin/runtime/ws-bus/grant", new { topics = new[] { topic } }, scope, ct);
        if (!grant.TryGetProperty("topics", out var topics) || topics.ValueKind != JsonValueKind.Array
            || !topics.EnumerateArray().Any(x => x.ValueKind == JsonValueKind.String && x.GetString() == topic))
            throw new WSBusException(424, "FRAMEWORK_WSBUS_INVALID_RESPONSE");
        var result = await SendAsync("/api/v1/admin/runtime/ws-bus/publish", new { topic, payload, trace_id = scope.TraceId }, scope, ct);
        if (!result.TryGetProperty("topic", out var published) || published.GetString() != topic
            || !result.TryGetProperty("tenant_uuid", out var tenant) || tenant.GetString() != scope.TenantUuid)
            throw new WSBusException(424, "FRAMEWORK_WSBUS_INVALID_RESPONSE");
    }

    public Task<JsonElement> TestNotificationAsync(string title, string content, WSBusScope scope, CancellationToken ct = default)
        => SendAsync("/api/v1/admin/notifications/test", new { title, content, member_uuid = scope.MemberUuid,
            type = "info", category = "system", is_important = false, metadata = new { source = "plugin-framework-runtime", trace_id = scope.TraceId } }, scope, ct);

    private async Task<JsonElement> SendAsync(string path, object body, WSBusScope scope, CancellationToken ct)
    {
        if (string.IsNullOrWhiteSpace(_base)) throw new WSBusException(503, "FRAMEWORK_WSBUS_UNAVAILABLE");
        if (!Guid.TryParse(scope.TenantUuid, out var tenant) || tenant == Guid.Empty || scope.TenantUuid != tenantUuid)
            throw new WSBusException(403, "FRAMEWORK_WSBUS_TENANT_MISMATCH");
        var credential = (await credentials.GetCredentialAsync(ct)).Validate();
        using var request = new HttpRequestMessage(HttpMethod.Post, _base + path);
        request.Headers.Authorization = new AuthenticationHeaderValue(credential.NormalizedAuthScheme, credential.Value);
        request.Headers.Add("X-Request-ID", scope.TraceId);
        request.Content = new StringContent(JsonSerializer.Serialize(body), Encoding.UTF8, "application/json");
        try
        {
            using var response = await http.SendAsync(request, ct);
            if (!response.IsSuccessStatusCode) throw new WSBusException(response.StatusCode switch { HttpStatusCode.Unauthorized => 401, HttpStatusCode.Forbidden => 403, _ => 424 }, "FRAMEWORK_WSBUS_UPSTREAM_REJECTED");
            using var json = JsonDocument.Parse(await response.Content.ReadAsStringAsync(ct));
            var root = json.RootElement;
            if (!root.TryGetProperty("code", out var code) || code.GetInt32() != 200
                || !root.TryGetProperty("data", out var data) || data.ValueKind != JsonValueKind.Object)
                throw new WSBusException(424, "FRAMEWORK_WSBUS_INVALID_RESPONSE");
            return data.Clone();
        }
        catch (HttpRequestException) { throw new WSBusException(424, "FRAMEWORK_WSBUS_UPSTREAM_DEPENDENCY"); }
        catch (JsonException) { throw new WSBusException(424, "FRAMEWORK_WSBUS_INVALID_RESPONSE"); }
        catch (OperationCanceledException) when (!ct.IsCancellationRequested) { throw new WSBusException(424, "FRAMEWORK_WSBUS_UPSTREAM_DEPENDENCY"); }
    }
}
public sealed class WSBusException(int status, string code) : InvalidOperationException(code)
{
    public int Status { get; } = status;
    public string Code { get; } = code;
}
