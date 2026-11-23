using PowerXPlugin.Framework.IAM.Models;

namespace PowerXPlugin.Framework.IAM.Contracts;

// 组织事实不等于权限决定；业务调用方仍须核对对象范围和独立角色。
public interface IMemberAuthorityDirectory
{
    Task<MemberAuthorityBatch> ReadMemberAuthority(string tenantUuid, MemberAuthorityQuery query, CancellationToken ct = default);
}
