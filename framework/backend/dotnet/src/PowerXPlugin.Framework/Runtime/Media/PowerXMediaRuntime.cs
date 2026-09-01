using System.Security.Cryptography;

namespace PowerXPlugin.Framework.Runtime.Media;

public class PowerXMediaRuntime
{
    private readonly HttpClient _http;
    private readonly PowerXMediaRuntimeOptions _options;

    public PowerXMediaRuntime(PowerXMediaRuntimeOptions? options = null, HttpClient? httpClient = null)
    {
        _options = Normalize(options ?? FromEnvironment());
        _http = httpClient ?? new HttpClient();
        if (httpClient == null && _options.Timeout > TimeSpan.Zero) _http.Timeout = _options.Timeout;
    }

    public static PowerXMediaRuntimeOptions FromEnvironment() => new()
    {
        ProviderMode = FirstNonEmpty(
            Environment.GetEnvironmentVariable("POWERX_PROVIDER_MODE"),
            Environment.GetEnvironmentVariable("PROVIDER_MODE"),
            "local"),
        Proxy = FirstNonEmpty(Environment.GetEnvironmentVariable("POWERX_PROXY"), "0"),
        BaseUrl = FirstNonEmpty(
            Environment.GetEnvironmentVariable("PX_GATEWAY_BASE_URL"),
            Environment.GetEnvironmentVariable("GATEWAY_BASE_URL")),
        ApiPrefix = FirstNonEmpty(Environment.GetEnvironmentVariable("PX_GATEWAY_API_PREFIX"), "/api/v1"),
        AuthScheme = FirstNonEmpty(Environment.GetEnvironmentVariable("PX_GATEWAY_AUTH_SCHEME"), "apikey"),
        Credential = FirstNonEmpty(
            Environment.GetEnvironmentVariable("PX_GATEWAY_API_KEY"),
            Environment.GetEnvironmentVariable("PX_PLUGIN_API_KEY"),
            Environment.GetEnvironmentVariable("PX_GATEWAY_BEARER_TOKEN"),
            Environment.GetEnvironmentVariable("GATEWAY_API_KEY"),
            Environment.GetEnvironmentVariable("GATEWAY_BEARER_TOKEN")),
        LocalRoot = FirstNonEmpty(
            Environment.GetEnvironmentVariable("POWERX_MEDIA_LOCAL_ROOT"),
            Environment.GetEnvironmentVariable("PX_MEDIA_LOCAL_ROOT"),
            Path.Combine(Directory.GetCurrentDirectory(), "tmp", "media"))
    };

    public PowerXMediaCapabilities Capabilities() => new()
    {
        ProviderMode = ProviderMode,
        StorageProvider = IsDelegated ? "powerx_media" : "local",
        UploadEnabled = !IsDelegated || GatewayConfigured,
        Message = IsDelegated && !GatewayConfigured
            ? "Delegated media requires PX_GATEWAY_BASE_URL and gateway credentials."
            : null
    };

    public async Task<PowerXMediaStoredAsset> StoreAssetBytesAsync(
        CreateMediaAssetRequest request,
        Stream body,
        string contentType,
        long sizeBytes,
        CancellationToken ct = default)
    {
        await using var prepared = await PreparedUpload.FromAsync(body, sizeBytes, ct);
        ApplyContentIdentity(request, prepared, contentType);
        return IsDelegated
            ? await StoreDelegatedAsync(request, prepared.Stream, contentType, prepared.SizeBytes, ct)
            : await StoreLocalAsync(request, prepared.Stream, contentType, prepared.SizeBytes, ct);
    }

    public async Task<Stream> OpenReadAsync(string? mediaAssetUuid, string? storageKey, string? tenantUuid = null, CancellationToken ct = default)
    {
        if (IsDelegated)
        {
            if (string.IsNullOrWhiteSpace(mediaAssetUuid)) throw new InvalidOperationException("media asset uuid is required");
            var client = CreateDelegatedClient();
            var ticket = await client.PresignAssetAsync(new PresignMediaAssetRequest
            {
                TenantUuid = tenantUuid ?? string.Empty,
                Uuid = mediaAssetUuid,
                Action = MediaPresignActions.Download,
                Method = "GET",
                ExpiresInSeconds = 300
            }, ct);
            return await client.DownloadBytesAsync(ticket, ct);
        }

        var key = Clean(storageKey);
        if (string.IsNullOrWhiteSpace(key)) throw new InvalidOperationException("local media storage key is required");
        var path = ResolveLocalPath(key);
        if (!File.Exists(path)) throw new FileNotFoundException("local media file not found", path);
        return File.OpenRead(path);
    }

    private async Task<PowerXMediaStoredAsset> StoreLocalAsync(
        CreateMediaAssetRequest request,
        Stream body,
        string contentType,
        long sizeBytes,
        CancellationToken ct)
    {
        var uuid = Guid.NewGuid().ToString("N");
        var extension = Path.GetExtension(Clean(request.Name));
        var tenant = SafePath(FirstNonEmpty(request.TenantUuid, "default"));
        var folder = SafeRelativePath(request.Folder);
        var objectKey = Path.Combine(tenant, folder, uuid + extension);
        var path = ResolveLocalPath(objectKey);
        Directory.CreateDirectory(Path.GetDirectoryName(path)!);
        await using (var output = File.Create(path))
        {
            await body.CopyToAsync(output, ct);
        }

        var asset = new MediaAsset
        {
            Uuid = uuid,
            TenantUuid = Clean(request.TenantUuid),
            Name = Clean(request.Name),
            Driver = "local",
            Folder = Clean(request.Folder),
            ObjectKey = objectKey,
            SizeBytes = sizeBytes,
            MimeType = Clean(contentType),
            OwnerSubjectType = Clean(request.OwnerSubjectType),
            OwnerSubjectId = Clean(request.OwnerSubjectId)
        };

        return new PowerXMediaStoredAsset
        {
            Asset = asset,
            StorageProvider = "local",
            StorageKey = objectKey,
            MediaAssetUuid = uuid
        };
    }

    private async Task<PowerXMediaStoredAsset> StoreDelegatedAsync(
        CreateMediaAssetRequest request,
        Stream body,
        string contentType,
        long sizeBytes,
        CancellationToken ct)
    {
        var client = CreateDelegatedClient();
        var asset = await client.CreateAssetAsync(request, ct);
        var ticket = await client.PresignAssetAsync(new PresignMediaAssetRequest
        {
            TenantUuid = request.TenantUuid,
            Uuid = asset.Uuid,
            OperatorId = request.OperatorId,
            Action = MediaPresignActions.Upload,
            Method = "PUT",
            ExpiresInSeconds = 900,
            Headers = new Dictionary<string, string> { ["Content-Type"] = contentType }
        }, ct);
        await client.UploadBytesAsync(ticket, body, contentType, ct);
        if (asset.SizeBytes == 0) asset.SizeBytes = sizeBytes;
        if (string.IsNullOrWhiteSpace(asset.MimeType)) asset.MimeType = contentType;

        return new PowerXMediaStoredAsset
        {
            Asset = asset,
            StorageProvider = "powerx_media",
            StorageKey = ticket.ObjectKey,
            MediaAssetUuid = asset.Uuid,
            ExternalUrl = asset.ExternalUrl
        };
    }

    private PowerXMediaClient CreateDelegatedClient()
    {
        if (!GatewayConfigured) throw new InvalidOperationException("gateway media is not configured");
        return new PowerXMediaClient(_options, _http);
    }

    private string ResolveLocalPath(string relative)
    {
        var root = Path.GetFullPath(_options.LocalRoot);
        var full = Path.GetFullPath(Path.Combine(root, relative));
        if (!full.StartsWith(root, StringComparison.Ordinal)) throw new InvalidOperationException("invalid local media storage key");
        return full;
    }

    private bool IsDelegated => ProviderMode == "delegated";
    private bool GatewayConfigured => !string.IsNullOrWhiteSpace(_options.BaseUrl) && !string.IsNullOrWhiteSpace(_options.Credential);
    private string ProviderMode => IsTruthy(_options.Proxy) || Clean(_options.ProviderMode).Equals("delegated", StringComparison.OrdinalIgnoreCase) ? "delegated" : "local";

    private static PowerXMediaRuntimeOptions Normalize(PowerXMediaRuntimeOptions options)
    {
        options.Proxy = FirstNonEmpty(options.Proxy, "0");
        options.ProviderMode = IsTruthy(options.Proxy) || Clean(options.ProviderMode).Equals("delegated", StringComparison.OrdinalIgnoreCase) ? "delegated" : "local";
        options.ApiPrefix = FirstNonEmpty(options.ApiPrefix, "/api/v1");
        options.AuthScheme = FirstNonEmpty(options.AuthScheme, "apikey");
        options.LocalRoot = FirstNonEmpty(options.LocalRoot, Path.Combine(Directory.GetCurrentDirectory(), "tmp", "media"));
        return options;
    }

    private static string SafePath(string value) => string.Join("_", Clean(value).Split(Path.GetInvalidFileNameChars(), StringSplitOptions.RemoveEmptyEntries));

    private static string SafeRelativePath(string value)
    {
        var parts = Clean(value).Split(new[] { '/', '\\' }, StringSplitOptions.RemoveEmptyEntries)
            .Select(SafePath)
            .Where(part => !string.IsNullOrWhiteSpace(part));
        return Path.Combine(parts.DefaultIfEmpty("assets").ToArray());
    }

    private static string Clean(string? value) => value?.Trim() ?? string.Empty;
    private static string FirstNonEmpty(params string?[] values) => values.FirstOrDefault(v => !string.IsNullOrWhiteSpace(v))?.Trim() ?? string.Empty;
    private static bool IsTruthy(string? value) => Clean(value).Equals("1", StringComparison.OrdinalIgnoreCase) ||
        Clean(value).Equals("true", StringComparison.OrdinalIgnoreCase) ||
        Clean(value).Equals("yes", StringComparison.OrdinalIgnoreCase);

    private static void ApplyContentIdentity(CreateMediaAssetRequest request, PreparedUpload prepared, string contentType)
    {
        request.SizeBytes = prepared.SizeBytes;
        request.MimeType = Clean(contentType);
        request.ContentSha256 = prepared.ContentSha256;
        request.ObjectKey = DeterministicAssetUuid("content_sha256:" + prepared.ContentSha256);
        request.Metadata["content_sha256"] = prepared.ContentSha256;
    }

    private static string DeterministicAssetUuid(string seed)
    {
        var hash = SHA256.HashData(System.Text.Encoding.UTF8.GetBytes(seed));
        hash[6] = (byte)((hash[6] & 0x0f) | 0x50);
        hash[8] = (byte)((hash[8] & 0x3f) | 0x80);
        return string.Create(36, hash, static (chars, bytes) =>
        {
            const string hex = "0123456789abcdef";
            var source = new[] { 0, 1, 2, 3, -1, 4, 5, -1, 6, 7, -1, 8, 9, -1, 10, 11, 12, 13, 14, 15 };
            var index = 0;
            foreach (var item in source)
            {
                if (item < 0)
                {
                    chars[index++] = '-';
                    continue;
                }
                var value = bytes[item];
                chars[index++] = hex[value >> 4];
                chars[index++] = hex[value & 0x0f];
            }
        });
    }

    private sealed class PreparedUpload : IAsyncDisposable
    {
        private readonly string _path;

        private PreparedUpload(string path, FileStream stream, string contentSha256, long sizeBytes)
        {
            _path = path;
            Stream = stream;
            ContentSha256 = contentSha256;
            SizeBytes = sizeBytes;
        }

        public FileStream Stream { get; }
        public string ContentSha256 { get; }
        public long SizeBytes { get; }

        public static async Task<PreparedUpload> FromAsync(Stream source, long declaredSizeBytes, CancellationToken ct)
        {
            if (source == null) throw new InvalidOperationException("media upload body is required");

            var path = Path.Combine(Path.GetTempPath(), "powerx-media-" + Guid.NewGuid().ToString("N") + ".upload");
            var output = new FileStream(path, FileMode.CreateNew, FileAccess.ReadWrite, FileShare.None, 1024 * 64, FileOptions.Asynchronous | FileOptions.DeleteOnClose);
            try
            {
                using var hash = IncrementalHash.CreateHash(HashAlgorithmName.SHA256);
                var buffer = new byte[1024 * 64];
                long total = 0;
                while (true)
                {
                    var read = await source.ReadAsync(buffer.AsMemory(0, buffer.Length), ct);
                    if (read == 0) break;
                    await output.WriteAsync(buffer.AsMemory(0, read), ct);
                    hash.AppendData(buffer, 0, read);
                    total += read;
                }
                if (declaredSizeBytes > 0 && declaredSizeBytes != total)
                {
                    throw new InvalidOperationException($"media upload size mismatch: declared={declaredSizeBytes}, actual={total}");
                }
                output.Position = 0;
                return new PreparedUpload(path, output, Convert.ToHexString(hash.GetHashAndReset()).ToLowerInvariant(), total);
            }
            catch
            {
                await output.DisposeAsync();
                if (File.Exists(path)) File.Delete(path);
                throw;
            }
        }

        public async ValueTask DisposeAsync()
        {
            await Stream.DisposeAsync();
            if (File.Exists(_path)) File.Delete(_path);
        }
    }
}
