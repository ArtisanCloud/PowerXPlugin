using PowerXPlugin.Framework.IAM.Contracts;
using PowerXPlugin.Framework.IAM.Models;

namespace PowerXPlugin.Framework.IAM;

public class IAMRegistry
{
    private readonly object _lock = new();
    private IAMMode? _mode;

    public IDirectoryService? Directory { get; private set; }
    public IAuthzService? Authz { get; private set; }
    public IIdentityContextService? IdentityContext { get; private set; }

    public bool IsBound => Directory != null && Authz != null && IdentityContext != null;

    public IAMMode? Mode => _mode;

    public void Bind(IAMMode mode, IDirectoryService directory, IAuthzService authz, IIdentityContextService identityContext)
    {
        ArgumentNullException.ThrowIfNull(directory);
        ArgumentNullException.ThrowIfNull(authz);
        ArgumentNullException.ThrowIfNull(identityContext);

        lock (_lock)
        {
            if (IsBound)
                throw new InvalidOperationException($"{IAMErrors.CodeAdapterAlreadyBound}: IAM adapters are already bound and cannot be re-bound");

            _mode = mode;
            Directory = directory;
            Authz = authz;
            IdentityContext = identityContext;
        }
    }
}
