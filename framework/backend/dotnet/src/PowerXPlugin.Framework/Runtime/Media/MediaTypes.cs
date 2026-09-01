using System.Text.Json.Serialization;

namespace PowerXPlugin.Framework.Runtime.Media;

public static class MediaCapabilities
{
    public const string AssetsRead = "com.corex.media.assets.read";
    public const string AssetsManage = "com.corex.media.assets.manage";
}

public static class MediaUploadChannels
{
    public const string Presigned = "presign_upload";
    public const string Direct = "direct_upload";
    public const string External = "external_link";
}

public static class MediaPresignActions
{
    public const string Upload = "upload";
    public const string Download = "download";
}

public class PowerXMediaClientOptions
{
    public string BaseUrl { get; set; } = string.Empty;
    public string ApiPrefix { get; set; } = "/api/v1";
    public string AuthScheme { get; set; } = "apikey";
    public string Credential { get; set; } = string.Empty;
    public string? TenantUuid { get; set; }
    public TimeSpan Timeout { get; set; } = TimeSpan.FromSeconds(30);
}

public class PowerXMediaRuntimeOptions : PowerXMediaClientOptions
{
    public string ProviderMode { get; set; } = string.Empty;
    public string Proxy { get; set; } = string.Empty;
    public string LocalRoot { get; set; } = string.Empty;
}

public class PowerXMediaCapabilities
{
    public string ProviderMode { get; set; } = "local";
    public string StorageProvider { get; set; } = "local";
    public bool UploadEnabled { get; set; } = true;
    public string? Message { get; set; }
}

public class PowerXMediaStoredAsset
{
    public MediaAsset Asset { get; set; } = new();
    public string StorageProvider { get; set; } = "local";
    public string StorageKey { get; set; } = string.Empty;
    public string? MediaAssetUuid { get; set; }
    public string? ExternalUrl { get; set; }
}

public class MediaAsset
{
    public string Uuid { get; set; } = string.Empty;
    [JsonPropertyName("tenant_uuid")]
    public string TenantUuid { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public string Driver { get; set; } = string.Empty;
    public string Folder { get; set; } = string.Empty;
    [JsonPropertyName("object_key")]
    public string ObjectKey { get; set; } = string.Empty;
    [JsonPropertyName("external_url")]
    public string ExternalUrl { get; set; } = string.Empty;
    [JsonPropertyName("size_bytes")]
    public long SizeBytes { get; set; }
    [JsonPropertyName("mime_type")]
    public string MimeType { get; set; } = string.Empty;
    [JsonPropertyName("owner_subject_type")]
    public string OwnerSubjectType { get; set; } = string.Empty;
    [JsonPropertyName("owner_subject_id")]
    public string OwnerSubjectId { get; set; } = string.Empty;
    [JsonPropertyName("download_url")]
    public string DownloadUrl { get; set; } = string.Empty;
}

public class CreateMediaAssetRequest
{
    public string TenantUuid { get; set; } = string.Empty;
    public string OperatorId { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public string Description { get; set; } = string.Empty;
    public string Driver { get; set; } = string.Empty;
    public string Folder { get; set; } = string.Empty;
    public string OwnerSubjectType { get; set; } = string.Empty;
    public string OwnerSubjectId { get; set; } = string.Empty;
    public string[] Tags { get; set; } = Array.Empty<string>();
    public string UploadMethod { get; set; } = MediaUploadChannels.Presigned;
    public string ExternalUrl { get; set; } = string.Empty;
    public string ObjectKey { get; set; } = string.Empty;
    public long SizeBytes { get; set; }
    public string MimeType { get; set; } = string.Empty;
    public string ContentSha256 { get; set; } = string.Empty;
    public Dictionary<string, string> Metadata { get; set; } = new();
}

public class PresignMediaAssetRequest
{
    public string TenantUuid { get; set; } = string.Empty;
    public string Uuid { get; set; } = string.Empty;
    public string OperatorId { get; set; } = string.Empty;
    public string Action { get; set; } = MediaPresignActions.Download;
    public string Method { get; set; } = "GET";
    public uint ExpiresInSeconds { get; set; } = 300;
    public Dictionary<string, string>? Headers { get; set; }
}

public class MediaPresignTicket
{
    public string Url { get; set; } = string.Empty;
    public string Method { get; set; } = "GET";
    [JsonPropertyName("expires_in_seconds")]
    public uint ExpiresInSeconds { get; set; }
    public Dictionary<string, string> Headers { get; set; } = new();
    public Dictionary<string, string> Fields { get; set; } = new();
    [JsonPropertyName("object_key")]
    public string ObjectKey { get; set; } = string.Empty;
}
