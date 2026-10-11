using System.Text.Json.Serialization;

namespace PowerXPlugin.Framework.Runtime.Scheduler;

/// <summary>Optional cleanup contract. Existing Scheduler adapters need not implement deletion.</summary>
public interface ISchedulerCleanup
{
    Task<SchedulerJobDeletion> DeleteJobAsync(SchedulerDeleteRequest request, CancellationToken ct = default);
}

public sealed record SchedulerDeleteRequest(
    [property: JsonPropertyName("job_uuid")] string JobUuid,
    [property: JsonPropertyName("expected_revision")] ulong ExpectedRevision);

public sealed record SchedulerJobDeletion(
    [property: JsonPropertyName("schema")] string Schema,
    [property: JsonPropertyName("job_uuid")] string JobUuid,
    [property: JsonPropertyName("status")] string Status,
    [property: JsonPropertyName("deleted")] bool Deleted,
    [property: JsonPropertyName("already_deleted")] bool AlreadyDeleted,
    [property: JsonPropertyName("revision")] ulong Revision,
    [property: JsonPropertyName("deleted_at")] DateTimeOffset DeletedAt,
    [property: JsonPropertyName("history_retained")] bool HistoryRetained,
    [property: JsonPropertyName("trace_id")] string TraceId);
