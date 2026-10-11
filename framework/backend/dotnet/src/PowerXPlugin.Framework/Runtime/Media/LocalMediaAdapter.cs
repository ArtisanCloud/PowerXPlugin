using System.Security.Cryptography;

namespace PowerXPlugin.Framework.Runtime.Media;

/// <summary>Filesystem-backed local adapter selected only in local mode.</summary>
public sealed class LocalMediaAdapter : IMediaService
{
    private readonly string _localRoot;

    public LocalMediaAdapter(LocalMediaAdapterOptions options)
    {
        ArgumentNullException.ThrowIfNull(options);
        _localRoot = string.IsNullOrWhiteSpace(options.LocalRoot)
            ? throw new ArgumentException("Local media root is required.", nameof(options))
            : Path.GetFullPath(options.LocalRoot);
    }

    public PowerXMediaCapabilities Capabilities() => new()
    {
        ProviderMode = "local", StorageProvider = "local", UploadEnabled = true
    };

    public async Task<PowerXMediaStoredAsset> StoreAssetBytesAsync(CreateMediaAssetRequest request, Stream body, string contentType, long sizeBytes, CancellationToken ct = default)
    {
        ArgumentNullException.ThrowIfNull(request);
        await using var prepared = await PreparedMediaUpload.FromAsync(body, sizeBytes, ct);
        MediaUploadIdentity.Apply(request, prepared, contentType);

        var uuid = Guid.NewGuid().ToString("N");
        var extension = Path.GetExtension(Clean(request.Name));
        var tenant = SafePath(FirstNonEmpty(request.TenantUuid, "default"));
        var folder = SafeRelativePath(request.Folder);
        var objectKey = Path.Combine(tenant, folder, uuid + extension);
        var path = ResolveLocalPath(objectKey);
        Directory.CreateDirectory(Path.GetDirectoryName(path)!);
        await using (var output = File.Create(path))
            await prepared.Stream.CopyToAsync(output, ct);

        return new PowerXMediaStoredAsset
        {
            Asset = new MediaAsset
            {
                Uuid = uuid, TenantUuid = Clean(request.TenantUuid), Name = Clean(request.Name),
                Driver = "local", Folder = Clean(request.Folder), ObjectKey = objectKey,
                SizeBytes = prepared.SizeBytes, MimeType = Clean(contentType),
                OwnerSubjectType = Clean(request.OwnerSubjectType), OwnerSubjectId = Clean(request.OwnerSubjectId)
            },
            StorageProvider = "local", StorageKey = objectKey, MediaAssetUuid = uuid
        };
    }

    public Task<Stream> OpenReadAsync(string? mediaAssetUuid, string? storageKey, string? tenantUuid = null, CancellationToken ct = default)
    {
        var key = Clean(storageKey);
        if (string.IsNullOrWhiteSpace(key)) throw new InvalidOperationException("local media storage key is required");
        var path = ResolveLocalPath(key);
        if (!File.Exists(path)) throw new FileNotFoundException("local media file not found", path);
        return Task.FromResult<Stream>(File.OpenRead(path));
    }

    private string ResolveLocalPath(string relative)
    {
        var full = Path.GetFullPath(Path.Combine(_localRoot, relative));
        if (!full.StartsWith(_localRoot + Path.DirectorySeparatorChar, StringComparison.Ordinal) && !string.Equals(full, _localRoot, StringComparison.Ordinal))
            throw new InvalidOperationException("invalid local media storage key");
        return full;
    }

    private static string SafePath(string value) => string.Join("_", Clean(value).Split(Path.GetInvalidFileNameChars(), StringSplitOptions.RemoveEmptyEntries));
    private static string SafeRelativePath(string value) => Path.Combine(Clean(value).Split(new[] { '/', '\\' }, StringSplitOptions.RemoveEmptyEntries).Select(SafePath).Where(part => !string.IsNullOrWhiteSpace(part)).DefaultIfEmpty("assets").ToArray());
    private static string Clean(string? value) => value?.Trim() ?? string.Empty;
    private static string FirstNonEmpty(params string?[] values) => values.FirstOrDefault(v => !string.IsNullOrWhiteSpace(v))?.Trim() ?? string.Empty;
}

public sealed class LocalMediaAdapterOptions { public string LocalRoot { get; set; } = string.Empty; }

internal sealed class PreparedMediaUpload : IAsyncDisposable
{
    private readonly string _path;
    private PreparedMediaUpload(string path, FileStream stream, string hash, long size) { _path = path; Stream = stream; ContentSha256 = hash; SizeBytes = size; }
    public FileStream Stream { get; }
    public string ContentSha256 { get; }
    public long SizeBytes { get; }
    public static async Task<PreparedMediaUpload> FromAsync(Stream source, long declaredSizeBytes, CancellationToken ct)
    {
        ArgumentNullException.ThrowIfNull(source);
        var path = Path.Combine(Path.GetTempPath(), "powerx-media-" + Guid.NewGuid().ToString("N") + ".upload");
        var output = new FileStream(path, FileMode.CreateNew, FileAccess.ReadWrite, FileShare.None, 64 * 1024, FileOptions.Asynchronous | FileOptions.DeleteOnClose);
        try
        {
            using var hash = IncrementalHash.CreateHash(HashAlgorithmName.SHA256);
            var buffer = new byte[64 * 1024]; long total = 0;
            while (true) { var read = await source.ReadAsync(buffer.AsMemory(), ct); if (read == 0) break; await output.WriteAsync(buffer.AsMemory(0, read), ct); hash.AppendData(buffer, 0, read); total += read; }
            if (declaredSizeBytes > 0 && declaredSizeBytes != total) throw new InvalidOperationException($"media upload size mismatch: declared={declaredSizeBytes}, actual={total}");
            output.Position = 0;
            return new PreparedMediaUpload(path, output, Convert.ToHexString(hash.GetHashAndReset()).ToLowerInvariant(), total);
        }
        catch { await output.DisposeAsync(); if (File.Exists(path)) File.Delete(path); throw; }
    }
    public async ValueTask DisposeAsync() { await Stream.DisposeAsync(); if (File.Exists(_path)) File.Delete(_path); }
}

internal static class MediaUploadIdentity
{
    public static void Apply(CreateMediaAssetRequest request, PreparedMediaUpload prepared, string contentType)
    {
        request.SizeBytes = prepared.SizeBytes; request.MimeType = contentType?.Trim() ?? string.Empty; request.ContentSha256 = prepared.ContentSha256;
        request.Metadata["content_sha256"] = prepared.ContentSha256;
    }
}
