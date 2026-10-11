using PowerXPlugin.Framework.IAM;
using PowerXPlugin.Framework.IAM.Models;
using PowerXPlugin.Framework.IAM.Contracts;

namespace PowerXPlugin.Framework.Runtime.WSBus;

public sealed record IdentityRevocation(string? PluginId = null, string? TenantUuid = null, string? UserUuid = null, string? MemberUuid = null, string? PolicyVersion = null);
public sealed class IdentityLifecycle : IDisposable
{
    private readonly object _gate = new();
    private readonly Dictionary<Guid, IdentityLease> _leases = [];
    private long _revision;
    private bool _disposed;
    public long Revision { get { lock (_gate) return _revision; } }
    public IdentityLease Open(DelegatedIdentity identity, string pluginId, TimeSpan ttl, long observedRevision,
        Func<DelegatedIdentity,string,bool> authorizeTopic, Action<string> close,IWSIdentityRevalidationService revalidator)
    {
        ArgumentNullException.ThrowIfNull(authorizeTopic); ArgumentNullException.ThrowIfNull(close);
        ArgumentNullException.ThrowIfNull(revalidator);
        identity.Validate(pluginId, DateTimeOffset.UtcNow);
        var expiry = BoundExpiry(identity, ttl, DateTimeOffset.UtcNow);
        var frozen = identity with { Roles = [..identity.Roles], Permissions = [..identity.Permissions] };
        lock (_gate) {
            if (_disposed) throw new IAMAdapterException(IAMErrors.CodeAdapterNotBound);
            if (_revision != observedRevision) throw new IAMAdapterException("IAM_CONTEXT_POLICY_CHANGED");
            var lease = new IdentityLease(this, frozen, expiry, authorizeTopic, close,revalidator);
            _leases.Add(lease.Id, lease); lease.Start(); return lease;
        }
    }
    public static DateTimeOffset BoundExpiry(DelegatedIdentity identity, TimeSpan ttl, DateTimeOffset now)
    {
        if (ttl <= TimeSpan.Zero) throw new IAMAdapterException("IAM_CONTEXT_INVALID_ARGUMENT");
        if (now >= identity.UserSessionExpiry) throw new IAMAdapterException("IAM_CONTEXT_USER_SESSION_EXPIRED");
        var expiry = now + ttl;
        if(expiry>identity.UserSessionExpiry) expiry=identity.UserSessionExpiry;
        if(expiry>identity.WSRevalidationExpiry) expiry=identity.WSRevalidationExpiry;
        if(now>=expiry) throw new IAMAdapterException("IAM_CONTEXT_WS_HANDLE_EXPIRED");return expiry;
    }
    internal void Remove(Guid id) { lock (_gate) _leases.Remove(id); }
    public int Revoke(IdentityRevocation selector)
    {
        if (selector == new IdentityRevocation()) return 0;
        IdentityLease[] selected;
        lock (_gate) {
            _revision++;
            selected = _leases.Values.Where(l => (selector.PluginId == null || selector.PluginId == l.Identity.PluginId)
                && (selector.TenantUuid == null || selector.TenantUuid == l.Identity.TenantUuid)
                && (selector.UserUuid == null || selector.UserUuid == l.Identity.UserUuid)
                && (selector.MemberUuid == null || selector.MemberUuid == l.Identity.MemberUuid)
                && (selector.PolicyVersion == null || selector.PolicyVersion == l.Identity.PolicyVersion)).ToArray();
        }
        foreach (var lease in selected) lease.Close("IAM_WS_IDENTITY_REVOKED"); return selected.Length;
    }
    public void Dispose()
    {
        IdentityLease[] selected;
        lock (_gate) { _disposed = true; _revision++; selected = _leases.Values.ToArray(); _leases.Clear(); }
        foreach (var lease in selected) lease.Close("IAM_WS_IDENTITY_REVOKED");
    }
}

public sealed class IdentityLease : IDisposable
{
    private readonly IdentityLifecycle _owner;
    private readonly Func<DelegatedIdentity,string,bool> _authorize;
    private readonly Action<string> _close;
    private readonly IWSIdentityRevalidationService _revalidator;
    private readonly CancellationTokenSource _stop = new();
    private Timer? _timer;
    private int _closed;
    internal DelegatedIdentity Identity { get; }
    public Guid Id { get; } = Guid.NewGuid();
    public DateTimeOffset ExpiresAt { get; }
    public CancellationToken CancellationToken => _stop.Token;
    internal IdentityLease(IdentityLifecycle owner, DelegatedIdentity identity, DateTimeOffset expiry,
        Func<DelegatedIdentity,string,bool> authorize, Action<string> close,IWSIdentityRevalidationService revalidator)
    { _owner=owner; Identity=identity; ExpiresAt=expiry; _authorize=authorize; _close=close;_revalidator=revalidator; }
    internal void Start()
    {
        var due = ExpiresAt - DateTimeOffset.UtcNow;
        _timer = new Timer(_ => Close("IAM_WS_CREDENTIAL_EXPIRED"), null, due > TimeSpan.Zero ? due : TimeSpan.Zero, Timeout.InfiniteTimeSpan);
    }
    public async Task<WSIdentity> RevalidateAsync(CancellationToken ct=default)
    {
        if (_stop.IsCancellationRequested) throw new IAMAdapterException("IAM_WS_IDENTITY_REVOKED");
        if (DateTimeOffset.UtcNow >= ExpiresAt) { Close("IAM_WS_CREDENTIAL_EXPIRED"); throw new IAMAdapterException("IAM_WS_CREDENTIAL_EXPIRED"); }
        using var linked=CancellationTokenSource.CreateLinkedTokenSource(ct,_stop.Token);
        try {
            var current=await _revalidator.RevalidateWSIdentity(new(){Handle=Identity.WSRevalidationHandle!},linked.Token);
            if (current is null) throw new IAMAdapterException("IAM_CONTEXT_RESPONSE_INVALID");
            current.Validate(Identity.PluginId,DateTimeOffset.UtcNow);var v=current.Identity;var previous=Identity;
            if(v.UserUuid!=previous.UserUuid || v.TenantUuid!=previous.TenantUuid || v.MemberUuid!=previous.MemberUuid || v.IsRoot!=previous.IsRoot
                || v.WSRevalidationHandle!=previous.WSRevalidationHandle || v.UserSessionExpiresAt!=previous.UserSessionExpiresAt
                || v.WSRevalidationExpiresAt!=previous.WSRevalidationExpiresAt || v.SourcePath!=previous.SourcePath || v.SourceMethod!=previous.SourceMethod
                || v.Transport!=previous.Transport || v.SourceRequestId!=previous.SourceRequestId || v.SourceTraceId!=previous.SourceTraceId || v.SourceRequiredPermission!=previous.SourceRequiredPermission)
                throw new IAMAdapterException("IAM_WS_IDENTITY_MISMATCH");
            if(_stop.IsCancellationRequested || ct.IsCancellationRequested) throw new IAMAdapterException("IAM_WS_IDENTITY_REVOKED");
            if(DateTimeOffset.UtcNow>=ExpiresAt) throw new IAMAdapterException("IAM_WS_CREDENTIAL_EXPIRED");
            return current;
        } catch(Exception error) {
            Close(error is IAMAdapterException iam ? iam.Code : "IAM_CONTEXT_UNAVAILABLE");throw;
        }
    }
    public async Task AuthorizeTopicAsync(string topic,CancellationToken ct=default)
    {
        var current=await RevalidateAsync(ct);var snapshot=current.Identity with {Roles=[..current.Identity.Roles],Permissions=[..current.Identity.Permissions]};
        if(string.IsNullOrWhiteSpace(topic) || !_authorize(snapshot,topic)){Close("IAM_WS_TOPIC_FORBIDDEN");throw new IAMAdapterException("IAM_WS_TOPIC_FORBIDDEN");}
        if(_stop.IsCancellationRequested || ct.IsCancellationRequested) {Close("IAM_WS_IDENTITY_REVOKED");throw new IAMAdapterException("IAM_WS_IDENTITY_REVOKED");}
        if(DateTimeOffset.UtcNow>=ExpiresAt) {Close("IAM_WS_CREDENTIAL_EXPIRED");throw new IAMAdapterException("IAM_WS_CREDENTIAL_EXPIRED");}
    }
    public void Close(string reason)
    {
        if (Interlocked.Exchange(ref _closed,1) != 0) return;
        _stop.Cancel(); _timer?.Dispose(); _owner.Remove(Id); _close(reason);
    }
    public void Dispose() => Close("IAM_WS_IDENTITY_REVOKED");
}
