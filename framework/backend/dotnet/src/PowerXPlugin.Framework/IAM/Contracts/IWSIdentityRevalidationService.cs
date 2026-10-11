using PowerXPlugin.Framework.IAM.Models;
namespace PowerXPlugin.Framework.IAM.Contracts;
public interface IWSIdentityRevalidationService
{
    Task<WSIdentity> RevalidateWSIdentity(WSIdentityQuery query, CancellationToken ct = default);
}
