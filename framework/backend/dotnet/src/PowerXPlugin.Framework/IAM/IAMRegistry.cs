using PowerXPlugin.Framework.IAM.Contracts;
using PowerXPlugin.Framework.IAM.Models;

namespace PowerXPlugin.Framework.IAM;

public class IAMRegistry
{
    private readonly object _lock = new();
    private IAMAdapterMode? _mode;

    public IDirectoryService? Directory { get; private set; }
    public IAuthzService? Authz { get; private set; }
    public IIdentityContextService? IdentityContext { get; private set; }
    public IDelegatedIdentityService? DelegatedIdentity => _mode == IAMAdapterMode.Delegated ? IdentityContext as IDelegatedIdentityService : null;
    public IWSIdentityRevalidationService? WSIdentityRevalidator => _mode==IAMAdapterMode.Delegated ? IdentityContext as IWSIdentityRevalidationService : null;

    public bool IsBound => Directory != null && Authz != null && IdentityContext != null;

    public IAMAdapterMode? Mode => _mode;

    public void Bind(IAMAdapterMode mode, IDirectoryService directory, IAuthzService authz, IIdentityContextService identityContext)
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
        }
    }
}
