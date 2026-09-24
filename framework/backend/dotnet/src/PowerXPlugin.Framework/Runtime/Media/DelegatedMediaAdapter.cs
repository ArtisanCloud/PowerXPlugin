using PowerXPlugin.Framework.Runtime.Common;

namespace PowerXPlugin.Framework.Runtime.Media;

/// <summary>Core-host Media adapter selected only in delegated mode.</summary>
public sealed class DelegatedMediaAdapter : IMediaService
{
    private readonly PowerXMediaClient _client;
    public DelegatedMediaAdapter(PowerXMediaClient client) => _client = client ?? throw new ArgumentNullException(nameof(client));
    public PowerXMediaCapabilities Capabilities() => new() { ProviderMode = "delegated", StorageProvider = "powerx_media", UploadEnabled = true };
    public async Task<PowerXMediaStoredAsset> StoreAssetBytesAsync(CreateMediaAssetRequest request, Stream body, string contentType, long sizeBytes, CancellationToken ct = default)
    {
        ArgumentNullException.ThrowIfNull(request);
        await using var prepared = await PreparedMediaUpload.FromAsync(body, sizeBytes, ct);
        MediaUploadIdentity.Apply(request, prepared, contentType);
        var asset = await _client.CreateAssetAsync(request, ct);
        var ticket = await _client.PresignAssetAsync(new PresignMediaAssetRequest { Uuid = asset.Uuid, Action = MediaPresignActions.Upload }, ct);
        await _client.UploadBytesAsync(ticket, prepared.Stream, contentType, ct);
        asset = await _client.CompleteUploadAsync(asset.Uuid, prepared.ContentSha256, ct);
        return new PowerXMediaStoredAsset { Asset = asset, StorageProvider = "powerx_media", StorageKey = asset.Uuid, MediaAssetUuid = asset.Uuid, ExternalUrl = asset.ExternalUrl };
    }
    public async Task<Stream> OpenReadAsync(string? mediaAssetUuid, string? storageKey, string? tenantUuid = null, CancellationToken ct = default)
    {
        if (string.IsNullOrWhiteSpace(mediaAssetUuid)) throw new FrameworkAdapterException(FrameworkRuntimeErrors.AdapterUnavailable, "media", "download", "Delegated Media requires an asset UUID.");
        return await _client.DownloadBytesAsync(await _client.PresignAssetAsync(new PresignMediaAssetRequest { Uuid = mediaAssetUuid, Action = MediaPresignActions.Download }, ct), ct);
    }
}
