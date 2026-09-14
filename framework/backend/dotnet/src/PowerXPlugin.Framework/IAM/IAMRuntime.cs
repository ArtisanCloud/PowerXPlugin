using Microsoft.Extensions.DependencyInjection;
using PowerXPlugin.Framework.IAM.Contracts;
using PowerXPlugin.Framework.IAM.Models;

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
    public static IAMRegistry CreateRegistry(IAMAdapterMode mode, IAMAdapterBundle local, IAMAdapterBundle delegated)
    {
        ArgumentNullException.ThrowIfNull(local);
        ArgumentNullException.ThrowIfNull(delegated);

        var selected = mode switch
        {
            IAMAdapterMode.Local => local,
            IAMAdapterMode.Delegated => delegated,
            _ => throw new IAMAdapterException(IAMErrors.CodeInvalidMode)
        };

        var registry = new IAMRegistry();
        registry.Bind(mode, selected.Directory, selected.Authz, selected.IdentityContext);
        return registry;
    }

    /// <summary>
    /// Registers exactly one IAM registry. The factories are evaluated once by
    /// DI; request handlers can only consume the already-selected registry.
    /// </summary>
    public static IServiceCollection AddPowerXIamRegistry(
        this IServiceCollection services,
        IAMAdapterMode mode,
        Func<IServiceProvider, IAMAdapterBundle> localFactory,
        Func<IServiceProvider, IAMAdapterBundle> delegatedFactory)
    {
        ArgumentNullException.ThrowIfNull(services);
        ArgumentNullException.ThrowIfNull(localFactory);
        ArgumentNullException.ThrowIfNull(delegatedFactory);

        services.AddSingleton(sp => CreateRegistry(mode, localFactory(sp), delegatedFactory(sp)));
        return services;
    }
}

public sealed class IAMAdapterException : Exception
{
    public string Code { get; }

    public IAMAdapterException(string code, Exception? innerException = null)
        : base(code, innerException) => Code = code;
}
