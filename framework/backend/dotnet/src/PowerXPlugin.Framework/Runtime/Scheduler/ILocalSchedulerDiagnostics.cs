namespace PowerXPlugin.Framework.Runtime.Scheduler;

/// <summary>Explicit local operations. Host adapters do not substitute for these operations.</summary>
public interface ILocalSchedulerDiagnostics
{
    Task<bool> TriggerJobPreviewAsync(string jobId, string tenantUuid, CancellationToken ct = default);
    Task<bool> DeleteJobAsync(string jobId, string tenantUuid, string jobUuid, long expectedRevision, CancellationToken ct = default);
}
