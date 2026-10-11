using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using PowerXPlugin.Framework.Runtime.Common;

namespace PowerXPlugin.Framework.Runtime.Media;

/// <summary>Typed client for Core's /api/v1/tenant/media host contract.</summary>
public sealed class PowerXMediaClient
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web);
    private readonly HttpClient _http;
    private readonly PowerXMediaClientOptions _options;
    private readonly IServiceCredentialProvider _credentials;

    public PowerXMediaClient(PowerXMediaClientOptions options, IServiceCredentialProvider credentials, HttpClient? httpClient = null)
    {
        _options = options ?? throw new ArgumentNullException(nameof(options));
        _credentials = credentials ?? throw new ArgumentNullException(nameof(credentials));
        if (string.IsNullOrWhiteSpace(_options.BaseUrl)) throw new FrameworkAdapterException(FrameworkRuntimeErrors.AdapterUnavailable, "media", "bootstrap", "Delegated Media base URL is required.");
        _http = httpClient ?? new HttpClient();
        if (httpClient is null && _options.Timeout > TimeSpan.Zero) _http.Timeout = _options.Timeout;
    }

    public async Task<MediaAsset> CreateAssetAsync(CreateMediaAssetRequest request, CancellationToken ct = default)
    {
        ArgumentNullException.ThrowIfNull(request);
        var host = await SendAsync<HostAsset>(HttpMethod.Post, "/api/v1/tenant/media/assets", new
        {
            name = request.Name?.Trim() ?? string.Empty,
            mime_type = request.MimeType?.Trim() ?? string.Empty,
            size_bytes = request.SizeBytes,
            checksum = "sha256:" + (request.ContentSha256?.Trim() ?? string.Empty),
            owner_subject_uuid = request.OwnerSubjectId?.Trim() ?? string.Empty
        }, ct);
        return ToMediaAsset(host, request);
    }

    public async Task<MediaPresignTicket> PresignAssetAsync(PresignMediaAssetRequest request, CancellationToken ct = default)
    {
        ArgumentNullException.ThrowIfNull(request);
        if (string.IsNullOrWhiteSpace(request.Uuid)) throw new ArgumentException("Media asset UUID is required.", nameof(request));
        var suffix = request.Action?.Trim().ToLowerInvariant() switch
        {
            MediaPresignActions.Upload => "presign-upload",
            MediaPresignActions.Download => "presign-download",
            _ => throw new ArgumentException("Media presign action must be upload or download.", nameof(request))
        };
        var ticket = await SendAsync<HostTransferTicket>(HttpMethod.Post, $"/api/v1/tenant/media/assets/{Uri.EscapeDataString(request.Uuid)}/{suffix}", new { }, ct);
        return new MediaPresignTicket { Url = ticket.Url, Method = ticket.Method, Headers = ticket.Headers ?? new(), ExpiresAt = ticket.ExpiresAt, ObjectKey = request.Uuid };
    }

    public async Task<MediaAsset> CompleteUploadAsync(string mediaAssetUuid, string contentSha256, CancellationToken ct = default)
    {
        if (string.IsNullOrWhiteSpace(mediaAssetUuid)) throw new ArgumentException("Media asset UUID is required.", nameof(mediaAssetUuid));
        var asset = await SendAsync<HostAsset>(HttpMethod.Post, $"/api/v1/tenant/media/assets/{Uri.EscapeDataString(mediaAssetUuid)}/complete-upload", new { checksum = "sha256:" + contentSha256.Trim() }, ct);
        return ToMediaAsset(asset, null);
    }

    public async Task UploadBytesAsync(MediaPresignTicket ticket, Stream body, string contentType, CancellationToken ct = default)
    {
        ArgumentNullException.ThrowIfNull(ticket); ArgumentNullException.ThrowIfNull(body);
        using var request = new HttpRequestMessage(new HttpMethod(string.IsNullOrWhiteSpace(ticket.Method) ? "PUT" : ticket.Method), RequireTransferUrl(ticket.Url)) { Content = new StreamContent(body) };
        ApplyTransferHeaders(request, ticket.Headers);
        if (!string.IsNullOrWhiteSpace(contentType)) request.Content.Headers.ContentType = new MediaTypeHeaderValue(contentType);
        using var response = await _http.SendAsync(request, ct);
        if (!response.IsSuccessStatusCode) throw new FrameworkAdapterException(FrameworkRuntimeErrors.AdapterUnavailable, "media", "upload", $"Media upload failed with status {(int)response.StatusCode}.");
    }

    public async Task<Stream> DownloadBytesAsync(MediaPresignTicket ticket, CancellationToken ct = default)
    {
        ArgumentNullException.ThrowIfNull(ticket);
        using var request = new HttpRequestMessage(HttpMethod.Get, RequireTransferUrl(ticket.Url));
        ApplyTransferHeaders(request, ticket.Headers);
        var response = await _http.SendAsync(request, HttpCompletionOption.ResponseHeadersRead, ct);
        if (!response.IsSuccessStatusCode) { response.Dispose(); throw new FrameworkAdapterException(FrameworkRuntimeErrors.AdapterUnavailable, "media", "download", $"Media download failed with status {(int)response.StatusCode}."); }
        return await response.Content.ReadAsStreamAsync(ct);
    }

    private async Task<T> SendAsync<T>(HttpMethod method, string path, object? body, CancellationToken ct)
    {
        var credential = (await _credentials.GetCredentialAsync(ct)).Validate();
        using var request = new HttpRequestMessage(method, _options.BaseUrl.TrimEnd('/') + path);
        request.Headers.TryAddWithoutValidation("Authorization", credential.NormalizedAuthScheme + " " + credential.Value.Trim());
        request.Headers.Accept.Add(new MediaTypeWithQualityHeaderValue("application/json"));
        if (body is not null) request.Content = new StringContent(JsonSerializer.Serialize(body, JsonOptions), Encoding.UTF8, "application/json");
        using var response = await _http.SendAsync(request, ct);
        var raw = await response.Content.ReadAsStringAsync(ct);
        if (!response.IsSuccessStatusCode) throw new FrameworkAdapterException(FrameworkRuntimeErrors.AdapterUnavailable, "media", path, $"Core Media request failed with status {(int)response.StatusCode}.");
        using var document = JsonDocument.Parse(string.IsNullOrWhiteSpace(raw) ? "{}" : raw);
        if (!document.RootElement.TryGetProperty("data", out var data) || data.ValueKind is JsonValueKind.Null or JsonValueKind.Undefined) throw new FrameworkAdapterException(FrameworkRuntimeErrors.AdapterUnavailable, "media", path, "Core Media response did not contain data.");
        return JsonSerializer.Deserialize<T>(data.GetRawText(), JsonOptions) ?? throw new FrameworkAdapterException(FrameworkRuntimeErrors.AdapterUnavailable, "media", path, "Core Media response data was invalid.");
    }

    private static Uri RequireTransferUrl(string value) => Uri.TryCreate(value, UriKind.Absolute, out var uri) && (uri.Scheme == Uri.UriSchemeHttp || uri.Scheme == Uri.UriSchemeHttps) ? uri : throw new InvalidOperationException("Media transfer ticket URL is invalid.");
    private static void ApplyTransferHeaders(HttpRequestMessage request, Dictionary<string, string>? headers) { foreach (var (name, value) in headers ?? new()) if (!string.IsNullOrWhiteSpace(name) && !string.IsNullOrWhiteSpace(value)) request.Headers.TryAddWithoutValidation(name, value); }
    private static MediaAsset ToMediaAsset(HostAsset asset, CreateMediaAssetRequest? request) => new() { Uuid = asset.AssetUuid, Name = asset.Name, MimeType = asset.MimeType, SizeBytes = asset.SizeBytes, TenantUuid = request?.TenantUuid ?? string.Empty, Folder = request?.Folder ?? string.Empty, OwnerSubjectType = request?.OwnerSubjectType ?? string.Empty, OwnerSubjectId = request?.OwnerSubjectId ?? string.Empty, Driver = "powerx_media" };
    private sealed class HostAsset { [JsonPropertyName("asset_uuid")] public string AssetUuid { get; set; } = string.Empty; public string Name { get; set; } = string.Empty; [JsonPropertyName("size_bytes")] public long SizeBytes { get; set; } [JsonPropertyName("mime_type")] public string MimeType { get; set; } = string.Empty; }
    private sealed class HostTransferTicket { public string Url { get; set; } = string.Empty; public string Method { get; set; } = string.Empty; [JsonPropertyName("expires_at")] public string ExpiresAt { get; set; } = string.Empty; public Dictionary<string, string>? Headers { get; set; } }
}
