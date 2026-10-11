using System.Net;
using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;
using PowerXPlugin.Framework.Runtime.Common;

namespace PowerXPlugin.Framework.Runtime.Integration.Delegated;

/// <summary>.NET parity for Go runtime/powerx/integration client. Do not expose this generic gateway to CRM controllers.</summary>
public sealed class PowerXIntegrationGatewayClient : IIntegrationGateway
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web);
    private readonly PowerXIntegrationGatewayClientOptions _options;
    private readonly IServiceCredentialProvider _credentials;
    private readonly HttpClient _http;

    public PowerXIntegrationGatewayClient(PowerXIntegrationGatewayClientOptions options, IServiceCredentialProvider credentials, HttpClient? http = null)
    {
        _options = options ?? throw new ArgumentNullException(nameof(options));
        if (string.IsNullOrWhiteSpace(options.BaseUrl)) throw new IntegrationAdapterException(IntegrationErrors.CodeUnavailable);
        _credentials = credentials ?? throw new ArgumentNullException(nameof(credentials));
        _http = http ?? new HttpClient { Timeout = options.Timeout > TimeSpan.Zero ? options.Timeout : TimeSpan.FromSeconds(30) };
    }

    public async Task<IReadOnlyList<IntegrationRouteSummary>> ListRoutesAsync(IntegrationRouteQuery query, CancellationToken ct = default)
    {
        var path = "/api/v1/tenant/integration/routes";
        var terms = new List<string>();
        if (!string.IsNullOrWhiteSpace(query.CapabilityId)) terms.Add("capability_id=" + Uri.EscapeDataString(query.CapabilityId.Trim()));
        if (!string.IsNullOrWhiteSpace(query.Channel)) terms.Add("channel=" + Uri.EscapeDataString(query.Channel.Trim()));
        if (terms.Count > 0) path += "?" + string.Join("&", terms);
        var root = await SendAsync(HttpMethod.Get, path, null, ct);
        return JsonSerializer.Deserialize<List<IntegrationRouteSummary>>(root.GetProperty("items").GetRawText(), JsonOptions) ?? [];
    }

    public async Task<IntegrationRouteDetail> GetRouteAsync(string routeSlug, CancellationToken ct = default)
    {
        RequireRoute(routeSlug);
        var root = await SendAsync(HttpMethod.Get, "/api/v1/tenant/integration/routes/" + Uri.EscapeDataString(routeSlug.Trim()), null, ct);
        return JsonSerializer.Deserialize<IntegrationRouteDetail>(root.GetRawText(), JsonOptions) ?? throw new IntegrationAdapterException(IntegrationErrors.CodeUpstreamDependency);
    }

    public async Task<IntegrationRouteResult> InvokeRouteAsync(string routeSlug, IntegrationRouteInvocation invocation, CancellationToken ct = default)
    {
        RequireRoute(routeSlug);
        if (invocation.Payload == null || invocation.Payload.Count == 0) throw new IntegrationAdapterException(IntegrationErrors.CodeInvalidRequest);
        var root = await SendAsync(HttpMethod.Post, "/api/v1/tenant/integration/routes/" + Uri.EscapeDataString(routeSlug.Trim()) + "/invoke", new { payload = invocation.Payload, idempotency_key = invocation.IdempotencyKey, context = invocation.Context }, ct);
        return JsonSerializer.Deserialize<IntegrationRouteResult>(root.GetRawText(), JsonOptions) ?? throw new IntegrationAdapterException(IntegrationErrors.CodeUpstreamDependency);
    }

    private async Task<JsonElement> SendAsync(HttpMethod method, string path, object? body, CancellationToken ct)
    {
        using var request = new HttpRequestMessage(method, NormalizeBaseUrl(_options.BaseUrl) + path);
        var credential = (await _credentials.GetCredentialAsync(ct)).Validate();
        request.Headers.Authorization = new AuthenticationHeaderValue(credential.NormalizedAuthScheme, credential.Value);
        if (body != null) request.Content = new StringContent(JsonSerializer.Serialize(body, JsonOptions), Encoding.UTF8, "application/json");
        try
        {
            using var response = await _http.SendAsync(request, ct);
            var raw = await response.Content.ReadAsStringAsync(ct);
            if (!response.IsSuccessStatusCode) throw new IntegrationAdapterException(MapStatus(response.StatusCode), (int)response.StatusCode);
            using var document = JsonDocument.Parse(string.IsNullOrWhiteSpace(raw) ? "{}" : raw);
            var root = document.RootElement;
            return (root.TryGetProperty("data", out var data) ? data : root).Clone();
        }
        catch (IntegrationAdapterException) { throw; }
        catch (HttpRequestException ex) { throw new IntegrationAdapterException(IntegrationErrors.CodeUpstreamDependency, innerException: ex); }
        catch (OperationCanceledException) when (!ct.IsCancellationRequested) { throw new IntegrationAdapterException(IntegrationErrors.CodeUpstreamDependency); }
    }

    private static void RequireRoute(string value) { if (string.IsNullOrWhiteSpace(value)) throw new IntegrationAdapterException(IntegrationErrors.CodeInvalidRequest); }
    private static string NormalizeBaseUrl(string value) => value.Trim().TrimEnd('/').StartsWith("http", StringComparison.OrdinalIgnoreCase) ? value.Trim().TrimEnd('/') : "http://" + value.Trim().TrimEnd('/');
    private static string MapStatus(HttpStatusCode status) => status switch { HttpStatusCode.Unauthorized => IntegrationErrors.CodeUnauthorized, HttpStatusCode.Forbidden => IntegrationErrors.CodeForbidden, HttpStatusCode.NotFound => IntegrationErrors.CodeNotFound, _ => IntegrationErrors.CodeUpstreamDependency };
}

public sealed class PowerXIntegrationGatewayClientOptions
{
    public string BaseUrl { get; init; } = string.Empty;
    public TimeSpan Timeout { get; init; } = TimeSpan.FromSeconds(30);
}
