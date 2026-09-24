using Microsoft.Extensions.DependencyInjection;
using PowerXPlugin.Framework.IAM.Contracts;
using PowerXPlugin.Framework.IAM.Models;
using PowerXPlugin.Framework.Runtime.Common;

namespace PowerXPlugin.Framework.IAM;

/// <summary>
/// The complete IAM adapter set selected once during application bootstrap.
/// Plugins provide their local implementation; Framework owns delegated
/// transport and the single-selection registration rule.
/// </summary>
public sealed record IAMAdapterBundle(
    IDirectoryService Directory,
    IAuthzService Authz,
    IIdentityContextService IdentityContext);

public static class IAMRuntime
{
    public static IAMRegistry CreateRegistry(ProviderMode mode, IAMAdapterBundle selected)
    {
        ArgumentNullException.ThrowIfNull(selected);

        var registry = new IAMRegistry();
        registry.Bind(mode == ProviderMode.Local ? IAMAdapterMode.Local : IAMAdapterMode.Delegated, selected.Directory, selected.Authz, selected.IdentityContext);
        return registry;
    }

    /// <summary>
    /// Registers exactly one IAM registry. The factories are evaluated once by
    /// DI; request handlers can only consume the already-selected registry.
    /// </summary>
    public static IServiceCollection AddPowerXIamRegistry(
        this IServiceCollection services,
        ProviderMode mode,
        Func<IServiceProvider, IAMAdapterBundle> localFactory,
        Func<IServiceProvider, IAMAdapterBundle> delegatedFactory)
    {
        ArgumentNullException.ThrowIfNull(services);
        ArgumentNullException.ThrowIfNull(localFactory);
        ArgumentNullException.ThrowIfNull(delegatedFactory);

        return services.AddPowerXRuntime<IAMRegistry>("iam", mode,
            sp => CreateRegistry(mode, localFactory(sp)),
            sp => CreateRegistry(mode, delegatedFactory(sp)));
    }
}

public sealed class IAMAdapterException : Exception
{
    public string Code { get; }

    public IAMAdapterException(string code, Exception? innerException = null)
        : base(code, innerException) => Code = code;
}
