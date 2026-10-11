using PowerXPlugin.Framework.IAM.Models;
namespace PowerXPlugin.Framework.IAM.Contracts;
public interface IDelegatedIdentityService
{
    Task<DelegatedIdentity> ResolveDelegatedIdentity(DelegatedIdentityQuery query, CancellationToken ct = default);
}
