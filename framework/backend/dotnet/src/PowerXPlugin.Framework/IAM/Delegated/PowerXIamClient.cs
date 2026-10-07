using System.Globalization;
using System.Net;
using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using PowerXPlugin.Framework.IAM.Contracts;
using PowerXPlugin.Framework.IAM.Models;

namespace PowerXPlugin.Framework.IAM.Delegated;

public sealed class PowerXIamClientOptions
{
    public string BaseUrl { get; init; } = string.Empty;
    public string Credential { get; init; } = string.Empty;
    public string AuthScheme { get; init; } = "Bearer";
    public TimeSpan Timeout { get; init; } = TimeSpan.FromSeconds(3);
}

/// <summary>
/// Framework-owned delegated transport for the IAM contracts. It uses only the
/// configured service credential for directory and authorization operations;
/// it never promotes an incoming administrator bearer into that credential.
/// </summary>
public sealed class PowerXIamClient : IDirectoryService, IAuthzService, IIdentityContextService
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web);
    private readonly HttpClient _http;
    private readonly PowerXIamClientOptions _options;

    public PowerXIamClient(PowerXIamClientOptions options, HttpClient? httpClient = null)
    {
        _options = options ?? throw new ArgumentNullException(nameof(options));
        _http = httpClient ?? new HttpClient();
        if (httpClient == null && options.Timeout > TimeSpan.Zero) _http.Timeout = options.Timeout;
    }

    public async Task<Tenant?> GetTenant(string tenantUuid, CancellationToken ct = default)
    {
        var response = await SendAsync(HttpMethod.Get, "/tenant/iam/tenant", null, null, ct);
        var result = Deserialize<Tenant>(response);
        AssertTenant(tenantUuid, result.TenantUUID);
        return result;
    }

    public async Task<IReadOnlyList<Department>> ListDepartments(string tenantUuid, CancellationToken ct = default)
    {
        var items = await GetItemsAsync<Department>("/tenant/iam/departments", ct);
        AssertTenants(tenantUuid, items.Select(item => item.TenantUUID));
        AssertTenants(tenantUuid, items.Select(item => item.TenantUUID));
        return items;
    }

    public async Task<IReadOnlyList<Member>> ListMembers(string tenantUuid, CancellationToken ct = default)
    {
        try
        {
            var result = new List<Member>();
            var seen = new HashSet<string>(StringComparer.Ordinal);
            long? expectedTotal = null;
            for (var page = 1; ; page++)
            {
                using var response = await SendAsync(HttpMethod.Get, $"/tenant/iam/members?page={page}&page_size=200", null, null, ct);
                var data = Unwrap(response);
                var pagination = data.GetProperty("pagination");
                var total = pagination.GetProperty("total").GetInt64();
                if (pagination.GetProperty("page").GetInt32() != page || pagination.GetProperty("page_size").GetInt32() != 200
                    || total < 0 || (expectedTotal.HasValue && expectedTotal != total))
                    throw new IAMAdapterException(IAMErrors.CodeUpstreamDependency);
                expectedTotal = total;
                var items = JsonSerializer.Deserialize<List<CoreMember>>(data.GetProperty("items").GetRawText(), JsonOptions)
                    ?? throw new IAMAdapterException(IAMErrors.CodeUpstreamDependency);
                if (items.Count != Math.Min(200L, total - result.Count))
                    throw new IAMAdapterException(IAMErrors.CodeUpstreamDependency);
                AssertTenants(tenantUuid, items.Select(item => item.TenantUUID));
                foreach (var item in items)
                {
                    if (string.IsNullOrWhiteSpace(item.MemberUUID) || !seen.Add(item.MemberUUID))
                        throw new IAMAdapterException(IAMErrors.CodeUpstreamDependency);
                    result.Add(item.ToMember());
                }
                if (result.Count == total) return result;
            }
        }
        catch (Exception exception) when (exception is JsonException or KeyNotFoundException or InvalidOperationException or FormatException)
        {
            throw new IAMAdapterException(IAMErrors.CodeUpstreamDependency, exception);
        }
    }

    public async Task<IReadOnlyList<Role>> ListRoles(string tenantUuid, CancellationToken ct = default)
    {
        var items = await GetItemsAsync<Role>("/tenant/iam/roles", ct);
        return items;
    }

    public async Task<IReadOnlyList<Permission>> ListPermissions(string tenantUuid, CancellationToken ct = default)
        => await GetItemsAsync<Permission>("/tenant/iam/permissions", ct);

    public async Task<AuthorizationDecision?> AuthorizeAsync(AuthorizationRequest request, CancellationToken ct = default)
    {
        var response = await SendAsync(HttpMethod.Post, "/tenant/iam/authorization:check", new
        {
            user_uuid = request.UserUUID,
            member_uuid = request.MemberUUID,
            resource = request.Resource,
            action = request.Action,
            trace_id = request.TraceID
        }, null, ct);
        var result = Deserialize<AuthorizationDecision>(response);
        return result with { Mode = IAMAdapterMode.Delegated.ToString().ToLowerInvariant() };
    }

    public Task<IdentityContext?> ResolveIdentity(string? bearerToken, CancellationToken ct = default)
    {
        // The Framework never forwards an inbound user bearer through its
        // service-to-host client. Core must expose a dedicated delegated
        // identity-exchange contract before this operation can be enabled.
        return Task.FromException<IdentityContext?>(new IAMAdapterException(IAMErrors.CodeIdentityDelegationUnavailable));
    }

    // Core publishes numeric member status; the public Framework model retains
    // the same string representation as the local and Skeleton adapters.
    private sealed record CoreMember
    {
        [JsonPropertyName("member_uuid")] public string MemberUUID { get; init; } = "";
        [JsonPropertyName("tenant_uuid")] public string TenantUUID { get; init; } = "";
        [JsonPropertyName("user_uuid")] public string UserUUID { get; init; } = "";
        [JsonPropertyName("display_name")] public string? DisplayName { get; init; }
        [JsonPropertyName("status"), JsonNumberHandling(JsonNumberHandling.Strict)] public short Status { get; init; }

        public Member ToMember() => new()
        {
            MemberUUID = MemberUUID, TenantUUID = TenantUUID, UserUUID = UserUUID,
            DisplayName = DisplayName, Status = Status.ToString(CultureInfo.InvariantCulture)
        };
    }

    private async Task<List<T>> GetItemsAsync<T>(string path, CancellationToken ct)
    {
        var response = await SendAsync(HttpMethod.Get, path, null, null, ct);
        var root = Unwrap(response);
        if (root.ValueKind == JsonValueKind.Object && root.TryGetProperty("items", out var items))
            return JsonSerializer.Deserialize<List<T>>(items.GetRawText(), JsonOptions) ?? [];
        return JsonSerializer.Deserialize<List<T>>(root.GetRawText(), JsonOptions) ?? [];
    }

    private async Task<JsonDocument> SendAsync(HttpMethod method, string path, object? body, string? explicitBearer, CancellationToken ct)
    {
        var baseUrl = NormalizeBaseUrl(_options.BaseUrl);
        if (string.IsNullOrWhiteSpace(baseUrl)) throw new IAMAdapterException(IAMErrors.CodeUpstreamDependency);
        var credential = string.IsNullOrWhiteSpace(explicitBearer) ? _options.Credential.Trim() : explicitBearer.Trim();
        if (string.IsNullOrWhiteSpace(credential)) throw new IAMAdapterException(IAMErrors.CodeUpstreamDependency);

        using var request = new HttpRequestMessage(method, baseUrl + path);
        request.Headers.Authorization = new AuthenticationHeaderValue(
            string.IsNullOrWhiteSpace(explicitBearer) ? NormalizeAuthScheme(_options.AuthScheme) : "Bearer",
            credential);
        if (body != null)
        {
            request.Content = new StringContent(JsonSerializer.Serialize(body, JsonOptions), Encoding.UTF8, "application/json");
        }

        try
        {
            using var response = await _http.SendAsync(request, ct);
            var payload = await response.Content.ReadAsStringAsync(ct);
            if (!response.IsSuccessStatusCode) throw new IAMAdapterException(MapStatus(response.StatusCode));
            return JsonDocument.Parse(string.IsNullOrWhiteSpace(payload) ? "{}" : payload);
        }
        catch (IAMAdapterException) { throw; }
        catch (OperationCanceledException) when (!ct.IsCancellationRequested)
        {
            throw new IAMAdapterException(IAMErrors.CodeUpstreamDependency);
        }
        catch (HttpRequestException exception)
        {
            throw new IAMAdapterException(IAMErrors.CodeUpstreamDependency, exception);
        }
    }

    private static T Deserialize<T>(JsonDocument response)
    {
        var value = JsonSerializer.Deserialize<T>(Unwrap(response).GetRawText(), JsonOptions);
        return value ?? throw new IAMAdapterException(IAMErrors.CodeUpstreamDependency);
    }

    private static JsonElement Unwrap(JsonDocument response)
    {
        var root = response.RootElement;
        return root.ValueKind == JsonValueKind.Object && root.TryGetProperty("data", out var data) ? data : root;
    }

    private static void AssertTenant(string asserted, string actual)
    {
        if (!string.IsNullOrWhiteSpace(asserted) && !string.Equals(asserted, actual, StringComparison.Ordinal))
            throw new IAMAdapterException(IAMErrors.CodeUpstreamDependency);
    }

    private static void AssertTenants(string asserted, IEnumerable<string> actual)
    {
        foreach (var tenant in actual) AssertTenant(asserted, tenant);
    }

    private static string NormalizeBaseUrl(string value)
    {
        var baseUrl = value.Trim().TrimEnd('/');
        if (string.IsNullOrWhiteSpace(baseUrl)) return string.Empty;
        if (!baseUrl.StartsWith("http://", StringComparison.OrdinalIgnoreCase) && !baseUrl.StartsWith("https://", StringComparison.OrdinalIgnoreCase)) baseUrl = "http://" + baseUrl;
        return baseUrl.Contains("/api/", StringComparison.OrdinalIgnoreCase) ? baseUrl : baseUrl + "/api/v1";
    }

    private static string NormalizeAuthScheme(string scheme) => scheme.Trim().Equals("apikey", StringComparison.OrdinalIgnoreCase) ? "ApiKey" : "Bearer";
    private static string MapStatus(HttpStatusCode status) => status switch
    {
        HttpStatusCode.Unauthorized => IAMErrors.CodeUnauthorized,
        HttpStatusCode.Forbidden => IAMErrors.CodeForbidden,
        _ => IAMErrors.CodeUpstreamDependency
    };
}
