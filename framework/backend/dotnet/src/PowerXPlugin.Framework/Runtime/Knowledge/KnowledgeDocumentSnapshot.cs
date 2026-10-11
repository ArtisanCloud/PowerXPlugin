using System.Text.Json;
using System.Text.Json.Serialization;

namespace PowerXPlugin.Framework.Runtime.Knowledge;

public sealed record DocumentProfileRef(string Uuid, int Version);
public sealed record DocumentStrategyRef(string? Uuid, string? Key, int Version);
public sealed class DocumentIngestionSettings
{
    [JsonPropertyName("schema"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string Schema { get; init; } = string.Empty;
    [JsonPropertyName("strategy"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public DocumentStrategyRef? Strategy { get; init; }
    [JsonPropertyName("ingestion_profile"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public DocumentProfileRef? IngestionProfile { get; init; }
    [JsonPropertyName("processor_profile"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public DocumentProfileRef? ProcessorProfile { get; init; }
    [JsonPropertyName("masking_profile"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public DocumentProfileRef? MaskingProfile { get; init; }
    [JsonPropertyName("priority"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string? Priority { get; init; }
    [JsonPropertyName("segment_mode"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string? SegmentMode { get; init; }
    [JsonPropertyName("segment_size_policy"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string? SegmentSizePolicy { get; init; }
    [JsonPropertyName("chunk_size"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public int? ChunkSize { get; init; }
    [JsonPropertyName("chunk_overlap"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public int? ChunkOverlap { get; init; }
    [JsonPropertyName("segment_order"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public IReadOnlyList<string>? SegmentOrder { get; init; }
    [JsonPropertyName("separators"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public IReadOnlyList<string>? Separators { get; init; }
    [JsonPropertyName("page_priority"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public bool? PagePriority { get; init; }
    [JsonPropertyName("anchor_heading_path"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public bool? AnchorHeadingPath { get; init; }
    [JsonPropertyName("anchor_clause_id"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public bool? AnchorClauseId { get; init; }
    [JsonPropertyName("anchor_row_number"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public bool? AnchorRowNumber { get; init; }
    [JsonPropertyName("anchor_speaker"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public bool? AnchorSpeaker { get; init; }
    [JsonPropertyName("anchor_sentence_index"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public bool? AnchorSentenceIndex { get; init; }
}
public sealed class DocumentFrozenProfile
{
    [JsonPropertyName("uuid"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string Uuid { get; init; } = string.Empty;
    [JsonPropertyName("version")] public int Version { get; init; }
    [JsonPropertyName("key"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string Key { get; init; } = string.Empty;
    [JsonPropertyName("config_checksum"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string ConfigChecksum { get; init; } = string.Empty;
}
public sealed class DocumentIngestionSnapshot
{
    [JsonPropertyName("schema"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string Schema { get; init; } = string.Empty;
    [JsonPropertyName("strategy"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public DocumentStrategyRef? Strategy { get; init; }
    [JsonPropertyName("ingestion_profile"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public DocumentFrozenProfile? IngestionProfile { get; init; }
    [JsonPropertyName("priority"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string? Priority { get; init; }
    [JsonPropertyName("segment_mode"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string? SegmentMode { get; init; }
    [JsonPropertyName("segment_size_policy"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string? SegmentSizePolicy { get; init; }
    [JsonPropertyName("chunk_size"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public int? ChunkSize { get; init; }
    [JsonPropertyName("chunk_overlap"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public int? ChunkOverlap { get; init; }
    [JsonPropertyName("segment_order"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public IReadOnlyList<string>? SegmentOrder { get; init; }
    [JsonPropertyName("separators"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public IReadOnlyList<string>? Separators { get; init; }
    [JsonPropertyName("page_priority"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public bool? PagePriority { get; init; }
    [JsonPropertyName("anchor_heading_path"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public bool? AnchorHeadingPath { get; init; }
    [JsonPropertyName("anchor_clause_id"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public bool? AnchorClauseId { get; init; }
    [JsonPropertyName("anchor_row_number"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public bool? AnchorRowNumber { get; init; }
    [JsonPropertyName("anchor_speaker"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public bool? AnchorSpeaker { get; init; }
    [JsonPropertyName("anchor_sentence_index"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public bool? AnchorSentenceIndex { get; init; }
    [JsonPropertyName("index_profile"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public DocumentFrozenProfile? IndexProfile { get; init; }
    [JsonPropertyName("rag_profile"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public DocumentFrozenProfile? RagProfile { get; init; }
    [JsonPropertyName("index_mode"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string IndexMode { get; init; } = string.Empty;
    [JsonPropertyName("length_unit"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string LengthUnit { get; init; } = string.Empty;
    [JsonPropertyName("source_checksum"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string SourceChecksum { get; init; } = string.Empty;
    [JsonPropertyName("source_version"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string SourceVersion { get; init; } = string.Empty;
}
public sealed class DocumentIngestionCapabilities
{
    [JsonPropertyName("schema"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string Schema { get; init; } = string.Empty;
    [JsonPropertyName("index_mode"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string IndexMode { get; init; } = string.Empty;
    [JsonPropertyName("content_types"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public IReadOnlyList<string>? ContentTypes { get; init; }
    [JsonPropertyName("length_unit"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string LengthUnit { get; init; } = string.Empty;
    [JsonPropertyName("chunk_size_min")] public int ChunkSizeMin { get; init; }
    [JsonPropertyName("chunk_size_max")] public int ChunkSizeMax { get; init; }
    [JsonPropertyName("priorities"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public IReadOnlyList<string>? Priorities { get; init; }
    [JsonPropertyName("priority_semantics"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string PrioritySemantics { get; init; } = string.Empty;
    [JsonPropertyName("segment_modes"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public IReadOnlyList<string>? SegmentModes { get; init; }
    [JsonPropertyName("segment_size_policies"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public IReadOnlyList<string>? SegmentSizePolicies { get; init; }
    [JsonPropertyName("segment_order_items"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public IReadOnlyList<string>? SegmentOrderItems { get; init; }
    [JsonPropertyName("processor_profile_supported")] public bool ProcessorProfileSupported { get; init; }
    [JsonPropertyName("masking_profile_supported")] public bool MaskingProfileSupported { get; init; }
    [JsonPropertyName("page_priority_supported")] public bool PagePrioritySupported { get; init; }
    [JsonPropertyName("strategy_override_supported")] public bool StrategyOverrideSupported { get; init; }
    [JsonPropertyName("anchor_mode_requirements"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public IReadOnlyDictionary<string,string>? AnchorModeRequirements { get; init; }
    [JsonPropertyName("defaults"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public DocumentIngestionSettings? Defaults { get; init; }
    [JsonPropertyName("max_content_chunks")] public int MaxContentChunks { get; init; }
    [JsonPropertyName("profile_precedence"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public IReadOnlyList<string>? ProfilePrecedence { get; init; }
}
public sealed class KnowledgeIndexJobChunk
{
    [JsonPropertyName("uuid"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string Uuid { get; init; } = string.Empty;
    [JsonPropertyName("space_uuid"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string SpaceUuid { get; init; } = string.Empty;
    [JsonPropertyName("job_uuid"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string JobUuid { get; init; } = string.Empty;
    [JsonPropertyName("document_uuid"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string DocumentUuid { get; init; } = string.Empty;
    [JsonPropertyName("ordinal")] public int Ordinal { get; init; }
    [JsonPropertyName("kind"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string Kind { get; init; } = string.Empty;
    [JsonPropertyName("content"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string Content { get; init; } = string.Empty;
    [JsonPropertyName("checksum"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] public string Checksum { get; init; } = string.Empty;
    [JsonPropertyName("metadata")] public JsonElement Metadata { get; init; }
}
public interface IKnowledgeJobChunksService
{
 Task<IReadOnlyList<KnowledgeIndexJobChunk>> GetIndexJobChunksAsync(GetKnowledgeIndexJobRequest request, CancellationToken ct = default);
}
public sealed partial class PowerXKnowledgeClient : IKnowledgeJobChunksService
{
 public async Task<IReadOnlyList<KnowledgeIndexJobChunk>> GetIndexJobChunksAsync(GetKnowledgeIndexJobRequest request, CancellationToken ct = default)
 {
  CheckTenant(request.TenantUuid);
  if (!ValidUuid(request.JobId)) throw new KnowledgeException(KnowledgeErrors.InvalidArgument);
  var response = await SendAsync<JobChunksResponse>(HttpMethod.Get, "/api/v1/tenant/knowledge/index-jobs/" + Uri.EscapeDataString(request.JobId) + "/chunks", null, ct);
  if (response.Items is null || response.Items.Any(item => !ValidUuid(item.Uuid) || item.JobUuid != request.JobId || !ValidUuid(item.SpaceUuid) || !ValidUuid(item.DocumentUuid) || string.IsNullOrEmpty(item.Checksum))) throw new KnowledgeException(KnowledgeErrors.InvalidResponse);
  return response.Items;
 }
 private sealed record JobChunksResponse(IReadOnlyList<KnowledgeIndexJobChunk>? Items);
 private static void ValidateSnapshot(DocumentIngestionSettings? settings)
 {
  if (settings is null) return;
  if (settings.Schema != "powerx.knowledge.document-ingestion/v1") throw new KnowledgeException(KnowledgeErrors.InvalidArgument);
  foreach (var profile in new[] { settings.IngestionProfile, settings.ProcessorProfile, settings.MaskingProfile })
   if (profile is not null && (!ValidUuid(profile.Uuid) || profile.Version < 1)) throw new KnowledgeException(KnowledgeErrors.InvalidArgument);
  if (settings.Strategy is { } strategy && (strategy.Version < 1 || string.IsNullOrEmpty(strategy.Uuid) && string.IsNullOrEmpty(strategy.Key) || strategy.Uuid is not null && !ValidUuid(strategy.Uuid))) throw new KnowledgeException(KnowledgeErrors.InvalidArgument);
  if (settings.ChunkSize is < 0 or > 65536 || settings.ChunkOverlap is < 0 || settings.ChunkSize is int size && settings.ChunkOverlap is int overlap && (size == 0 && overlap != 0 || size > 0 && overlap >= size)) throw new KnowledgeException(KnowledgeErrors.InvalidArgument);
 }
}
