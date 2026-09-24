using Microsoft.Extensions.DependencyInjection;

namespace PowerXPlugin.Framework.Runtime.Common;

/// <summary>
/// Holds the single adapter selected at startup for one business contract.
/// No fallback adapter is retained or consulted after construction.
/// </summary>
public sealed class DualModeRuntime<TService>
    where TService : class
{
    public DualModeRuntime(ProviderMode mode, TService service, string module)
    {
        Mode = mode;
        Module = string.IsNullOrWhiteSpace(module) ? typeof(TService).Name : module;
        Service = service ?? throw new FrameworkAdapterException(
            FrameworkRuntimeErrors.AdapterUnavailable,
            Module,
            "bootstrap");
    }

    public ProviderMode Mode { get; }
    public string Module { get; }
    public TService Service { get; }
}

/// <summary>
/// Common DI registration for all local/delegated business contracts.
/// Factories are evaluated only for the selected mode.
/// </summary>
public static class RuntimeRegistrationExtensions
{
    public static IServiceCollection AddPowerXRuntime<TService>(
        this IServiceCollection services,
        string module,
        ProviderMode mode,
        Func<IServiceProvider, TService> localFactory,
        Func<IServiceProvider, TService> delegatedFactory)
        where TService : class
    {
        ArgumentNullException.ThrowIfNull(services);
        ArgumentNullException.ThrowIfNull(localFactory);
        ArgumentNullException.ThrowIfNull(delegatedFactory);

        services.AddSingleton(sp =>
        {
            var selected = mode switch
            {
                ProviderMode.Local => localFactory(sp),
                ProviderMode.Delegated => delegatedFactory(sp),
                _ => throw new FrameworkAdapterException(
                    FrameworkRuntimeErrors.InvalidProviderMode,
                    module,
                    "bootstrap")
            };

            return new DualModeRuntime<TService>(mode, selected, module);
        });
        services.AddSingleton<TService>(sp => sp.GetRequiredService<DualModeRuntime<TService>>().Service);
        return services;
    }
}

