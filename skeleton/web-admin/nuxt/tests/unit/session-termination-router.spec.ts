import { afterEach, expect, it, vi } from 'vitest';
const mocks = vi.hoisted(() => ({ token: { value: 'fixture' as string|null }, fetch: vi.fn() }));
vi.mock('~/composables/useAuth', () => ({ useAuth: () => ({ token: mocks.token, ensureFreshToken: async()=>mocks.token.value }) }));
vi.mock('~/stores/user', () => ({ useUserStore: () => ({ context:null, isLoading:false, fetchUserContext:mocks.fetch }) }));
afterEach(() => { vi.unstubAllGlobals(); vi.clearAllMocks(); });
it('embedded_context_rejection_keeps_route_query_hash_for_host_owned_recovery',async()=>{
  vi.stubGlobal('process', { client:true });vi.stubGlobal('defineNuxtRouteMiddleware',(fn:unknown)=>fn);
  vi.stubGlobal('useRuntimeConfig',()=>({public:{insidePowerX:true}}));const navigate=vi.fn();vi.stubGlobal('navigateTo',navigate);
  mocks.fetch.mockImplementation(async()=>{mocks.token.value=null;throw {code:'IAM_CONTEXT_USER_SESSION_INVALID'};});
  const {default:guard}=await import('../../app/middleware/auth.global');
  await (guard as any)({path:'/orders',fullPath:'/orders?draft=saved#editor',meta:{}});
  expect(navigate).not.toHaveBeenCalled();
});
