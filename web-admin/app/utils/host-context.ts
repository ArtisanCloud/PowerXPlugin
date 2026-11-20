import type { PowerXHostSessionSnapshot, PowerXHostIdentity } from '@artisan-cloud/plugin-framework-client';
import type { UserContextData } from '~/composables/api/services/meService';
const uuid = (value: unknown) => typeof value === 'string' && value !== '00000000-0000-0000-0000-000000000000' && /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(value);
export function validatePowerXUserContext(context: UserContextData, snapshot: PowerXHostSessionSnapshot): PowerXHostIdentity {
  if (!context?.user || context.user.status !== 1 || typeof context.is_root !== 'boolean' || !Array.isArray(context.permission_codes) || context.permission_codes.some(code => typeof code !== 'string')) throw new Error('AUTH_USER_CONTEXT_INVALID');
  const tenant = context.current_tenant_uuid || null;
  if ((snapshot.tenantUuid && tenant !== snapshot.tenantUuid) || (tenant && !uuid(tenant))) throw new Error('AUTH_TENANT_MISMATCH');
  if (context.is_root === true && context.current_member_uuid && !uuid(context.current_member_uuid)) throw new Error('AUTH_MEMBER_CONTEXT_INVALID');
  if (context.is_root === true) return { tenantUuid: tenant, memberUuid: context.current_member_uuid || null };
  if (!tenant || !snapshot.tenantUuid || !uuid(context.current_member_uuid) || !Array.isArray(context.members) || !context.members.some(member => member.tenant_uuid === tenant && member.member_uuid === context.current_member_uuid)) throw new Error('AUTH_MEMBER_CONTEXT_INVALID');
  return { tenantUuid: tenant, memberUuid: context.current_member_uuid! };
}
