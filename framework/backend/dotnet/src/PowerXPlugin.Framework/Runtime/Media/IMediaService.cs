namespace PowerXPlugin.Framework.Runtime.Media;

/// <summary>
/// Framework boundary for binary assets. Applications receive one selected
/// implementation at bootstrap and must not choose storage per request.
/// </summary>
public interface IMediaService
{
    PowerXMediaCapabilities Capabilities();

    Task<PowerXMediaStoredAsset> StoreAssetBytesAsync(
        CreateMediaAssetRequest request,
        Stream body,
        string contentType,
        long sizeBytes,
        CancellationToken ct = default);

    Task<Stream> OpenReadAsync(
        string? mediaAssetUuid,
        string? storageKey,
        string? tenantUuid = null,
        CancellationToken ct = default);
}
