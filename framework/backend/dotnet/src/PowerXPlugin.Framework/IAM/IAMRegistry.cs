using PowerXPlugin.Framework.IAM.Contracts;
using PowerXPlugin.Framework.IAM.Models;
using PowerXPlugin.Framework.Runtime.Capability;

namespace PowerXPlugin.Framework.IAM;

public class IAMRegistry
{
    private readonly object _lock = new();
    private IAMAdapterMode? _mode;

    public IDirectoryService? Directory { get; private set; }
    public IAuthzService? Authz { get; private set; }
    public IIdentityContextService? IdentityContext { get; private set; }
    public IAuthorizationBaselineService? AuthorizationBaselines { get; private set; }
    public IAuthorizationBaselineService GetAuthorizationBaselines() => AuthorizationBaselines
        ?? throw new IAMAdapterException(IAMErrors.CodeAdapterNotBound);
    public IDelegatedIdentityService? DelegatedIdentity => _mode == IAMAdapterMode.Delegated ? IdentityContext as IDelegatedIdentityService : null;
    public IWSIdentityRevalidationService? WSIdentityRevalidator => _mode==IAMAdapterMode.Delegated ? IdentityContext as IWSIdentityRevalidationService : null;

    public bool IsBound => Directory != null && Authz != null && IdentityContext != null;

    public IAMAdapterMode? Mode => _mode;

    public async Task RequireAuthorizationBaselineFeatureAsync(bool enabled, IReadOnlyList<string> required,
        ICapabilityGrantRegistry? grants, CancellationToken ct = default)
    {
        if (!enabled) return;
        if (required.Any(id => string.IsNullOrWhiteSpace(id) || id.Trim() != id) || required.Distinct(StringComparer.Ordinal).Count() != required.Count)
            throw new IAMAdapterException("IAM_BASELINE_REQUIREMENTS_INVALID");
        if (!required.Contains(AuthorizationBaselineContract.CapabilityId, StringComparer.Ordinal))
            throw new IAMAdapterException("IAM_BASELINE_REQUIREMENTS_INVALID");
        _ = GetAuthorizationBaselines();
        if (!IsBound) throw new IAMAdapterException(IAMErrors.CodeAdapterNotBound);
        if (_mode == IAMAdapterMode.Local) return;
        string[] dependencies = [AuthorizationBaselineContract.CapabilityId, "com.corex.iam.directory.read",
            "com.corex.iam.identity_context.service_read", "com.corex.capabilities.grant_status.read"];
        if (dependencies.Any(id => !required.Contains(id, StringComparer.Ordinal)))
            throw new IAMAdapterException("IAM_BASELINE_REQUIREMENTS_INVALID");
        if (WSIdentityRevalidator is null || grants is null) throw new IAMAdapterException(IAMErrors.CodeAdapterNotBound);
        await PowerXCapabilityGrantClient.RequireGrantsAsync(grants, dependencies, ct);
    }

    public void Bind(IAMAdapterMode mode, IDirectoryService directory, IAuthzService authz, IIdentityContextService identityContext,
        IAuthorizationBaselineService? authorizationBaselines = null)
    {
        ArgumentNullException.ThrowIfNull(directory);
        ArgumentNullException.ThrowIfNull(authz);
        ArgumentNullException.ThrowIfNull(identityContext);

        lock (_lock)
        {
            if (IsBound)
                throw new InvalidOperationException($"{IAMErrors.CodeAdapterAlreadyBound}: IAM adapters are already bound and cannot be re-bound");
            if (mode == IAMAdapterMode.Delegated && identityContext is not IDelegatedIdentityService)
                throw new IAMAdapterException(IAMErrors.CodeAdapterNotBound);

            _mode = mode;
            Directory = directory;
            Authz = authz;
            IdentityContext = identityContext;
            AuthorizationBaselines = authorizationBaselines;
        }
    }
}
