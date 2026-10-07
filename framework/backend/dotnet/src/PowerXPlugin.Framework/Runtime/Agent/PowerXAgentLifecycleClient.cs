using System.Net;
using System.Net.Http.Headers;
using System.Globalization;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using PowerXPlugin.Framework.Runtime.Common;

namespace PowerXPlugin.Framework.Runtime.Agent;

/// <summary>Core's six tenant-scoped Agent Lifecycle operations, using a composition-time service identity.</summary>
public sealed class PowerXAgentLifecycleClient(string baseUrl, string tenantUuid, IServiceCredentialProvider credentials, HttpClient? http = null) : IAgentLifecycleService
{
    private readonly string _baseUrl = baseUrl.TrimEnd('/');
    private readonly string _tenantUuid = tenantUuid;
    private readonly IServiceCredentialProvider _credentials = credentials ?? throw new ArgumentNullException(nameof(credentials));
    private readonly HttpClient _http = http ?? new HttpClient();
    private static readonly JsonSerializerOptions Json = new(JsonSerializerDefaults.Web);

    public async Task<AgentHealthSummary> GetHealthSummaryAsync(AgentLifecycleScope scope, CancellationToken ct = default) =>
        Summary(await SendAsync<HealthWire>(scope, HttpMethod.Get, Path(scope) + "/health/summary", null, ct), scope);

    public async Task<AgentHealthHistory> ListHealthHistoryAsync(AgentLifecycleScope scope, int rangeHours, int limit, CancellationToken ct = default)
    {
        Validate(scope);
        if (rangeHours is < 0 or > 8784 || limit is < 0 or > 1000) throw InvalidArgument();
        var wire = await SendAsync<HistoryWire>(scope, HttpMethod.Get, $"{Path(scope)}/health/history?range_hours={rangeHours}&limit={limit}", null, ct);
        return new AgentHealthHistory(wire.Snapshots?.Select(item => Summary(item, scope)).ToArray() ?? throw InvalidResponse());
    }

    public async Task<AgentBridgeState> GetBridgeStateAsync(AgentLifecycleScope scope, int limit, CancellationToken ct = default)
    {
        Validate(scope);
        if (limit is < 0 or > 1000) throw InvalidArgument();
        var wire = await SendAsync<StateWire>(scope, HttpMethod.Get, $"{Path(scope)}/bridge/state?limit={limit}", null, ct);
        var agent = Identity(wire.Agent, scope);
        return new AgentBridgeState(agent, Capacity(wire.Agent!.Capacity), string.IsNullOrEmpty(wire.Health?.Status) ? null : Summary(wire.Health, scope),
            wire.Events?.Select(Event).ToArray() ?? []);
    }

    public Task<AgentBridgeLifecycleResult> FreezeAsync(AgentLifecycleScope scope, AgentBridgeControlRequest request, CancellationToken ct = default) =>
        ControlAsync(scope, "freeze", request, ct);

    public Task<AgentBridgeLifecycleResult> RecoverAsync(AgentLifecycleScope scope, AgentBridgeControlRequest request, CancellationToken ct = default) =>
        ControlAsync(scope, "recover", request, ct);

    public async Task<AgentBridgeLifecycleResult> RebalanceAsync(AgentLifecycleScope scope, AgentBridgeRebalanceRequest request, CancellationToken ct = default)
    {
        Validate(scope); ValidateTrace(request?.TraceUuid);
        if (request is null || request.TargetCapacityInstances is < 1 or > 100_000) throw InvalidArgument();
        return Result(await SendAsync<ResultWire>(scope, HttpMethod.Post, Path(scope) + "/bridge/rebalance",
            new { target_capacity_instances = request.TargetCapacityInstances, reason = request.Reason, trace_id = request.TraceUuid }, ct), scope);
    }

    private async Task<AgentBridgeLifecycleResult> ControlAsync(AgentLifecycleScope scope, string action, AgentBridgeControlRequest request, CancellationToken ct)
    {
        Validate(scope); ValidateTrace(request?.TraceUuid);
        if (request is null) throw InvalidArgument();
        return Result(await SendAsync<ResultWire>(scope, HttpMethod.Post, Path(scope) + "/bridge/" + action,
            new { reason = request.Reason, trace_id = request.TraceUuid }, ct), scope);
    }

    private async Task<T> SendAsync<T>(AgentLifecycleScope scope, HttpMethod method, string path, object? body, CancellationToken ct)
    {
        Validate(scope);
        var credential = (await _credentials.GetCredentialAsync(ct)).Validate();
        if (credential.NormalizedAuthScheme != "Bearer") throw new AgentRuntimeException("FRAMEWORK_AGENT_SERVICE_ACTOR_REQUIRED");
        using var request = new HttpRequestMessage(method, _baseUrl + path);
        request.Headers.Authorization = new AuthenticationHeaderValue("Bearer", credential.Value);
        if (body is not null) request.Content = new StringContent(JsonSerializer.Serialize(body, Json), Encoding.UTF8, "application/json");
        using var response = await _http.SendAsync(request, ct);
        var raw = await response.Content.ReadAsStringAsync(ct);
        if (!response.IsSuccessStatusCode) throw new AgentRuntimeException(response.StatusCode == HttpStatusCode.Forbidden
            ? "FRAMEWORK_AGENT_FORBIDDEN" : "FRAMEWORK_AGENT_UPSTREAM_DEPENDENCY");
        if (response.StatusCode != HttpStatusCode.OK) throw InvalidResponse();
        try
        {
            using var document = JsonDocument.Parse(raw);
            var root = document.RootElement;
            if (!root.TryGetProperty("code", out var code) || code.ValueKind != JsonValueKind.Number || code.GetInt32() != 200
                || !root.TryGetProperty("data", out var data) || data.ValueKind != JsonValueKind.Object) throw InvalidResponse();
            return JsonSerializer.Deserialize<T>(data.GetRawText(), Json) ?? throw InvalidResponse();
        }
        catch (JsonException) { throw InvalidResponse(); }
    }

    private void Validate(AgentLifecycleScope? scope)
    {
        if (scope?.TenantUuid != _tenantUuid) throw new AgentRuntimeException("FRAMEWORK_AGENT_TENANT_MISMATCH");
        if (!IsUuid(scope.TenantUuid) || !IsUuid(scope.AgentUuid)) throw InvalidArgument();
    }

    private static void ValidateTrace(string? trace) { if (!string.IsNullOrEmpty(trace) && !IsUuid(trace)) throw InvalidArgument(); }
    private static string Path(AgentLifecycleScope scope) => "/api/v1/openapi/agents/" + scope.AgentUuid;
    private static AgentHealthSummary Summary(HealthWire? wire, AgentLifecycleScope scope)
    {
        if (wire?.Metrics is null || string.IsNullOrWhiteSpace(wire.Status) || wire.HealthScore is < 0 or > 100
            || !DateTimeOffset.TryParse(wire.UpdatedAt, CultureInfo.InvariantCulture, DateTimeStyles.None, out var updatedAt) || wire.WindowDurationSec < 0
            || (wire.AnomalyTraceIds?.Any(id => !IsUuid(id)) ?? false)) throw InvalidResponse();
        return new(scope.TenantUuid, scope.AgentUuid, wire.Status, wire.HealthScore, updatedAt, wire.WindowDurationSec,
            new(wire.Metrics.ThroughputPerMin, wire.Metrics.SuccessRate, wire.Metrics.P95LatencyMs, wire.Metrics.ResourceUtilPct, wire.Metrics.ErrorRate),
            wire.Recommendations ?? [], wire.AnomalyTraceIds ?? []);
    }

    private static AgentBridgeIdentity Identity(AgentWire? wire, AgentLifecycleScope scope)
    {
        if (wire is null || wire.Id != scope.AgentUuid || wire.TenantUuid != scope.TenantUuid || wire.UpdatedAt == default
            || string.IsNullOrWhiteSpace(wire.Status) || wire.Capacity is null) throw InvalidResponse();
        return new(wire.Id, wire.TenantUuid, wire.Alias ?? "", wire.Status);
    }

    private static AgentBridgeCapacity Capacity(CapacityWire? wire)
    {
        if (wire is null || wire.Default < 0 || wire.Current < 0 || wire.Max is < 0) throw InvalidResponse();
        return new(wire.Default, wire.Current, wire.Max);
    }

    private static AgentBridgeEvent Event(EventWire? wire)
    {
        if (wire is null || !IsUuid(wire.Id) || wire.OccurredAt == default || string.IsNullOrWhiteSpace(wire.Type)) throw InvalidResponse();
        return new(wire.Id!, wire.Type, wire.FromStatus ?? "", wire.ToStatus ?? "", wire.Reason ?? "", wire.TriggeredBy ?? "", wire.TraceId ?? "", wire.OccurredAt);
    }

    private static AgentBridgeLifecycleResult Result(ResultWire? wire, AgentLifecycleScope scope) =>
        wire is null ? throw InvalidResponse() : new(Identity(wire.Agent, scope), Event(wire.Event));

    private static bool IsUuid(string? value) => Guid.TryParse(value, out var uuid) && uuid != Guid.Empty && uuid.ToString() == value;
    private static AgentRuntimeException InvalidArgument() => new(AgentRuntimeErrors.InvalidArgument);
    private static AgentRuntimeException InvalidResponse() => new(AgentRuntimeErrors.InvalidResponse);

    private sealed record MetricsWire([property: JsonPropertyName("throughput_per_min")] double ThroughputPerMin, [property: JsonPropertyName("success_rate")] double SuccessRate,
        [property: JsonPropertyName("p95_latency_ms")] int P95LatencyMs, [property: JsonPropertyName("resource_util_pct")] double ResourceUtilPct, [property: JsonPropertyName("error_rate")] double ErrorRate);
    private sealed record HealthWire(string? Status, [property: JsonPropertyName("health_score")] int HealthScore, [property: JsonPropertyName("updated_at")] string? UpdatedAt,
        [property: JsonPropertyName("window_duration_sec")] int WindowDurationSec, MetricsWire? Metrics, IReadOnlyList<string>? Recommendations,
        [property: JsonPropertyName("anomaly_trace_ids")] IReadOnlyList<string>? AnomalyTraceIds);
    private sealed record HistoryWire(IReadOnlyList<HealthWire>? Snapshots);
    private sealed record CapacityWire(int Default, int Current, int? Max);
    private sealed record AgentWire(string? Id, [property: JsonPropertyName("tenant_uuid")] string? TenantUuid, string? Alias, string? Status, CapacityWire? Capacity,
        [property: JsonPropertyName("updated_at")] DateTimeOffset UpdatedAt);
    private sealed record EventWire(string? Id, string? Type, [property: JsonPropertyName("from_status")] string? FromStatus,
        [property: JsonPropertyName("to_status")] string? ToStatus, string? Reason, [property: JsonPropertyName("triggered_by")] string? TriggeredBy,
        [property: JsonPropertyName("trace_id")] string? TraceId, [property: JsonPropertyName("occurred_at")] DateTimeOffset OccurredAt);
    private sealed record StateWire(AgentWire? Agent, HealthWire? Health, IReadOnlyList<EventWire>? Events);
    private sealed record ResultWire(AgentWire? Agent, EventWire? Event);
}
