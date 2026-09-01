using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;

namespace PowerXPlugin.Framework.Runtime.Media;

public class PowerXMediaClient
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web);
    private readonly HttpClient _http;
    private readonly PowerXMediaClientOptions _options;

    public PowerXMediaClient(PowerXMediaClientOptions options, HttpClient? httpClient = null)
    {
        _options = options;
        _http = httpClient ?? new HttpClient();
        if (httpClient == null && _options.Timeout > TimeSpan.Zero) _http.Timeout = _options.Timeout;
    }

    public async Task<MediaAsset> CreateAssetAsync(CreateMediaAssetRequest request, CancellationToken ct = default)
    {
        var body = new
        {
            tenant_uuid = Clean(request.TenantUuid),
            operator_id = Clean(request.OperatorId),
            name = Clean(request.Name),
            description = Clean(request.Description),
            driver = Clean(request.Driver),
            folder = Clean(request.Folder),
            ownerSubjectType = Clean(request.OwnerSubjectType),
            ownerSubjectId = Clean(request.OwnerSubjectId),
            tags = request.Tags.Where(tag => !string.IsNullOrWhiteSpace(tag)).Select(tag => tag.Trim()).ToArray(),
            uploadMethod = FirstNonEmpty(request.UploadMethod, MediaUploadChannels.Presigned),
            externalUrl = Clean(request.ExternalUrl),
            objectKey = Clean(request.ObjectKey),
            sizeBytes = request.SizeBytes,
            mimeType = Clean(request.MimeType),
            contentSha256 = Clean(request.ContentSha256),
            metadata = BuildMetadata(request)
        };

        using var doc = await InvokeRestAsync(MediaCapabilities.AssetsManage, "CreateMediaAsset", HttpMethod.Post, "/api/v1/media/assets", body, request.TenantUuid, ct);
        var asset = DeserializeUnwrapped<MediaAsset>(doc.RootElement);
        if (string.IsNullOrWhiteSpace(asset.Uuid)) throw new InvalidOperationException("media asset response missing uuid");
        return asset;
    }

    public async Task<MediaPresignTicket> PresignAssetAsync(PresignMediaAssetRequest request, CancellationToken ct = default)
    {
        var body = new
        {
            tenant_uuid = Clean(request.TenantUuid),
            uuid = Clean(request.Uuid),
            operator_id = Clean(request.OperatorId),
            action = FirstNonEmpty(request.Action, MediaPresignActions.Download),
            method = Clean(request.Method),
            expiresInSeconds = request.ExpiresInSeconds,
            headers = request.Headers
        };

        using var doc = await InvokeRestAsync(MediaCapabilities.AssetsManage, "PresignMediaAsset", HttpMethod.Post, $"/api/v1/media/assets/{Uri.EscapeDataString(request.Uuid)}/presign", body, request.TenantUuid, ct);
        var ticket = DeserializeUnwrapped<MediaPresignTicket>(doc.RootElement);
        if (string.IsNullOrWhiteSpace(ticket.Url)) throw new InvalidOperationException("media presign response missing url");
        if (ticket.ExpiresInSeconds == 0)
        {
            var root = Unwrap(doc.RootElement);
            ticket.ExpiresInSeconds = (uint)ReadInt(root, "expiresInSeconds", "expires_in_seconds");
        }
        if (string.IsNullOrWhiteSpace(ticket.ObjectKey))
        {
            var root = Unwrap(doc.RootElement);
            ticket.ObjectKey = ReadString(root, "objectKey", "object_key", "StorageKey");
        }
        return ticket;
    }

    public async Task UploadBytesAsync(MediaPresignTicket ticket, Stream body, string contentType, CancellationToken ct = default)
    {
        if (string.IsNullOrWhiteSpace(ticket.Url)) throw new InvalidOperationException("media upload ticket url is required");
        var resolvedUrl = ResolveTicketUrl(ticket.Url);
        using var req = new HttpRequestMessage(new HttpMethod(FirstNonEmpty(ticket.Method, "PUT")), resolvedUrl)
        {
            Content = new StreamContent(body)
        };
        foreach (var item in ticket.Headers ?? new Dictionary<string, string>())
        {
            if (!string.IsNullOrWhiteSpace(item.Key) && !string.IsNullOrWhiteSpace(item.Value)) req.Headers.TryAddWithoutValidation(item.Key, item.Value);
        }
        ApplyGatewayAuthorization(req, resolvedUrl);
        if (!string.IsNullOrWhiteSpace(contentType)) req.Content.Headers.ContentType = new MediaTypeHeaderValue(contentType);
        using var resp = await _http.SendAsync(req, ct);
        if (!resp.IsSuccessStatusCode) throw new InvalidOperationException($"media upload failed: status={(int)resp.StatusCode}");
    }

    public async Task<Stream> DownloadBytesAsync(MediaPresignTicket ticket, CancellationToken ct = default)
    {
        if (string.IsNullOrWhiteSpace(ticket.Url)) throw new InvalidOperationException("media download ticket url is required");
        var resolvedUrl = ResolveTicketUrl(ticket.Url);
        using var req = new HttpRequestMessage(HttpMethod.Get, resolvedUrl);
        foreach (var item in ticket.Headers ?? new Dictionary<string, string>())
        {
            if (!string.IsNullOrWhiteSpace(item.Key) && !string.IsNullOrWhiteSpace(item.Value)) req.Headers.TryAddWithoutValidation(item.Key, item.Value);
        }
        ApplyGatewayAuthorization(req, resolvedUrl);
        var resp = await _http.SendAsync(req, HttpCompletionOption.ResponseHeadersRead, ct);
        if (!resp.IsSuccessStatusCode)
        {
            resp.Dispose();
            throw new InvalidOperationException($"media download failed: status={(int)resp.StatusCode}");
        }
        return await resp.Content.ReadAsStreamAsync(ct);
    }

    private async Task<JsonDocument> InvokeRestAsync(string capabilityId, string action, HttpMethod method, string endpoint, object body, string tenantUuid, CancellationToken ct)
    {
        if (string.IsNullOrWhiteSpace(_options.BaseUrl)) throw new InvalidOperationException("gateway base url is required");
        var url = $"{NormalizeBaseUrl(_options.BaseUrl).TrimEnd('/')}/{FirstNonEmpty(_options.ApiPrefix, "/api/v1").Trim('/')}/tenant/invocations";
        using var req = new HttpRequestMessage(HttpMethod.Post, url);
        req.Headers.TryAddWithoutValidation("Authorization", BuildAuthHeader());
        var tenant = FirstNonEmpty(tenantUuid, _options.TenantUuid);
        if (!string.IsNullOrWhiteSpace(tenant)) req.Headers.TryAddWithoutValidation("tenant_uuid", tenant);
        req.Content = new StringContent(JsonSerializer.Serialize(new
        {
            capability_id = capabilityId,
            action,
            preferred_protocol = "rest",
            payload = new { method = method.Method, endpoint, body }
        }, JsonOptions), Encoding.UTF8, "application/json");

        using var resp = await _http.SendAsync(req, ct);
        var text = await resp.Content.ReadAsStringAsync(ct);
        if (!resp.IsSuccessStatusCode) throw new InvalidOperationException(string.IsNullOrWhiteSpace(text) ? $"gateway invocation failed: status={(int)resp.StatusCode}" : text);
        return JsonDocument.Parse(string.IsNullOrWhiteSpace(text) ? "{}" : text);
    }

    private string BuildAuthHeader()
    {
        if (string.IsNullOrWhiteSpace(_options.Credential)) throw new InvalidOperationException("gateway credential is required");
        return NormalizeAuthScheme(_options.AuthScheme) == "apikey"
            ? $"ApiKey {_options.Credential.Trim()}"
            : $"Bearer {_options.Credential.Trim()}";
    }

    private Uri ResolveTicketUrl(string raw)
    {
        var url = Clean(raw);
        if (Uri.TryCreate(url, UriKind.Absolute, out var absolute) && IsHttpUri(absolute)) return absolute;
        if (string.IsNullOrWhiteSpace(_options.BaseUrl)) throw new InvalidOperationException("gateway base url is required for relative media ticket url");

        var baseUrl = BuildGatewayUploadBaseUrl();
        var originalUrl = url;
        var apiPrefix = NormalizeApiPrefix(_options.ApiPrefix);
        if (baseUrl.EndsWith(apiPrefix, StringComparison.OrdinalIgnoreCase) &&
            url.StartsWith(apiPrefix + "/", StringComparison.OrdinalIgnoreCase))
        {
            url = url[apiPrefix.Length..];
        }

        if (originalUrl.StartsWith("/media/", StringComparison.OrdinalIgnoreCase) &&
            Uri.TryCreate(baseUrl, UriKind.Absolute, out var parsedBase))
        {
            return RequireHttpUri($"{parsedBase.Scheme}://{parsedBase.Authority}{originalUrl}", raw);
        }

        return RequireHttpUri($"{baseUrl.TrimEnd('/')}/{url.TrimStart('/')}", raw);
    }

    private string BuildGatewayUploadBaseUrl()
    {
        var baseUrl = NormalizeBaseUrl(_options.BaseUrl).TrimEnd('/');
        var apiPrefix = NormalizeApiPrefix(_options.ApiPrefix);
        return baseUrl.EndsWith(apiPrefix, StringComparison.OrdinalIgnoreCase)
            ? baseUrl
            : $"{baseUrl}{apiPrefix}";
    }

    private void ApplyGatewayAuthorization(HttpRequestMessage req, Uri resolvedUrl)
    {
        if (req.Headers.Contains("Authorization")) return;
        if (string.IsNullOrWhiteSpace(_options.Credential)) return;
        if (!Uri.TryCreate(NormalizeBaseUrl(_options.BaseUrl), UriKind.Absolute, out var gateway)) return;
        if (!string.Equals(resolvedUrl.Scheme, gateway.Scheme, StringComparison.OrdinalIgnoreCase) ||
            !string.Equals(resolvedUrl.Authority, gateway.Authority, StringComparison.OrdinalIgnoreCase))
        {
            return;
        }
        req.Headers.TryAddWithoutValidation("Authorization", BuildAuthHeader());
    }

    private static Uri RequireHttpUri(string value, string original)
    {
        if (Uri.TryCreate(value, UriKind.Absolute, out var uri) && IsHttpUri(uri)) return uri;
        throw new InvalidOperationException($"media ticket url is invalid: original={Clean(original)}, resolved={Clean(value)}");
    }

    private static bool IsHttpUri(Uri uri) =>
        uri.Scheme.Equals(Uri.UriSchemeHttp, StringComparison.OrdinalIgnoreCase) ||
        uri.Scheme.Equals(Uri.UriSchemeHttps, StringComparison.OrdinalIgnoreCase);

    private static string NormalizeBaseUrl(string? value)
    {
        var baseUrl = Clean(value).TrimEnd('/');
        if (baseUrl.Length == 0) return string.Empty;
        return baseUrl.StartsWith("http://", StringComparison.OrdinalIgnoreCase) ||
            baseUrl.StartsWith("https://", StringComparison.OrdinalIgnoreCase)
            ? baseUrl
            : "http://" + baseUrl;
    }

    private static string NormalizeApiPrefix(string? value)
    {
        var prefix = FirstNonEmpty(value, "/api/v1").Trim('/');
        return "/" + prefix;
    }

    private static T DeserializeUnwrapped<T>(JsonElement root)
    {
        var unwrapped = Unwrap(root);
        var value = JsonSerializer.Deserialize<T>(unwrapped.GetRawText(), JsonOptions);
        return value ?? throw new InvalidOperationException("media response data is empty");
    }

    private static JsonElement Unwrap(JsonElement root)
    {
        if (root.ValueKind == JsonValueKind.Object && root.TryGetProperty("data", out var data))
        {
            if (data.ValueKind == JsonValueKind.Object && data.TryGetProperty("data", out var nested)) return nested;
            if (data.ValueKind == JsonValueKind.Object && data.TryGetProperty("payload", out var payload)) return payload;
            if (data.ValueKind == JsonValueKind.Object && data.TryGetProperty("result", out var result)) return result;
            return data;
        }
        if (root.ValueKind == JsonValueKind.Object && root.TryGetProperty("payload", out var rootPayload)) return rootPayload;
        if (root.ValueKind == JsonValueKind.Object && root.TryGetProperty("result", out var rootResult)) return rootResult;
        return root;
    }

    private static string ReadString(JsonElement element, params string[] names)
    {
        foreach (var name in names)
        {
            if (element.ValueKind == JsonValueKind.Object && element.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String)
            {
                return value.GetString() ?? string.Empty;
            }
        }
        return string.Empty;
    }

    private static int ReadInt(JsonElement element, params string[] names)
    {
        foreach (var name in names)
        {
            if (element.ValueKind != JsonValueKind.Object || !element.TryGetProperty(name, out var value)) continue;
            if (value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var number)) return number;
            if (value.ValueKind == JsonValueKind.String && int.TryParse(value.GetString(), out number)) return number;
        }
        return 0;
    }

    private static string NormalizeAuthScheme(string raw) => raw.Trim().ToLowerInvariant() is "apikey" or "api_key" or "api-key" ? "apikey" : "bearer";
    private static string Clean(string? value) => value?.Trim() ?? string.Empty;
    private static string FirstNonEmpty(params string?[] values) => values.FirstOrDefault(v => !string.IsNullOrWhiteSpace(v))?.Trim() ?? string.Empty;

    private static Dictionary<string, string> BuildMetadata(CreateMediaAssetRequest request)
    {
        var metadata = new Dictionary<string, string>(StringComparer.Ordinal);
        foreach (var item in request.Metadata)
        {
            if (!string.IsNullOrWhiteSpace(item.Key) && !string.IsNullOrWhiteSpace(item.Value))
            {
                metadata[item.Key.Trim()] = item.Value.Trim();
            }
        }
        var contentSha256 = Clean(request.ContentSha256);
        if (!string.IsNullOrWhiteSpace(contentSha256)) metadata["content_sha256"] = contentSha256;
        return metadata;
    }
}
