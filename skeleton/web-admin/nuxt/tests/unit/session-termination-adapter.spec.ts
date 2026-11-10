import { ref } from 'vue';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { getPowerXHostSession } from '../../../../../framework/frontend/nuxt/framework-client';
const mocks = vi.hoisted(() => ({ user: { clearUserState: vi.fn(), commitHostContext: vi.fn() }, dispose: vi.fn(), app: {} as any, me: vi.fn() }));
vi.mock('#imports', () => ({ useNuxtApp: () => mocks.app, useRuntimeConfig: () => ({ public: { powerxPluginId: 'com.example.adapter' }, app: { baseURL: '/_p/com.example.adapter/admin/' } }), useRouter: () => ({ beforeEach: ()=>()=>{}, afterEach: ()=>()=>{}, currentRoute: { value: { fullPath: '/intro' } }, replace: vi.fn() }) }));
vi.mock('~/composables/useTheme', () => ({ useTheme: () => ({ setTheme: vi.fn() }) }));
vi.mock('~/stores/user', () => ({ useUserStore: () => mocks.user }));
vi.mock('~/composables/useMe', () => ({ useMe: () => ({ getUserContext: mocks.me }) }));
vi.mock('~/utils/host-context', () => ({ validatePowerXUserContext: () => ({ tenantUuid: null, memberUuid: null }) }));
afterEach(() => { vi.unstubAllGlobals(); vi.clearAllMocks(); });
it('identity_suspension_preserves_business_store_actual_logout_disposes_it', async () => {
  const pluginId = 'com.example.adapter'; let receive!: (message: any)=>Promise<void>;
  const business = { $id: 'drafts', $dispose: mocks.dispose };
  mocks.app = { $pinia: { _s: new Map([['drafts', business]]), state: { value: { drafts: { saved: true } } } }, $i18n: { locale: { value: 'en' }, setLocale: vi.fn() } };
  mocks.me.mockResolvedValue({});
  vi.stubGlobal('document', { referrer: 'https://host.test/plugins' });
  vi.stubGlobal('window', { location: { origin: 'https://host.test' }, addEventListener: vi.fn(), removeEventListener: vi.fn(), PowerXBridgeClient: { protocolVersion: 2, onAuthToken: (handler: typeof receive) => { receive=handler; return ()=>{}; }, setNavigator: ()=>()=>{}, requestAuth: vi.fn(), requestSync: vi.fn(), getLastSyncAt: ()=>0, reportSessionInvalid: vi.fn() } });
  const { setupHostBridgeAdapter } = await import('../../app/composables/useHostBridgeAdapter');
  const adapter = setupHostBridgeAdapter();
  await receive({ source:'powerx', type:'auth-token', protocolVersion:2,pluginId,instanceId:'frame',sequence:1,sessionId:'login',revision:1,accessToken:'fixture',tokenType:'Bearer',scope:'powerx',tenantUuid:null,expiresAt:Date.now()+60000,expiresIn:60 });
  // Initial binding starts an identity scope. Only assert the suspended login.
  mocks.dispose.mockClear(); mocks.app.$pinia.state.value.drafts={ saved:true };
  const session=getPowerXHostSession(pluginId);session.reportSessionInvalid(session.getReadySnapshot()!,'IAM_CONTEXT_USER_SESSION_INVALID');
  expect(mocks.dispose).not.toHaveBeenCalled();expect(mocks.app.$pinia.state.value.drafts).toEqual({saved:true});
  await receive({source:'powerx',type:'auth-cleared',protocolVersion:2,pluginId,instanceId:'frame',sequence:2});
  expect(mocks.dispose).toHaveBeenCalled();expect(mocks.app.$pinia.state.value.drafts).toBeUndefined();adapter.dispose();
});

it.each(['failClosed','clearAuth'] as const)('embedded_%s_only_suspends_identity',async method=>{
  vi.stubGlobal('window',{location:{origin:'https://host.test'}});vi.stubGlobal('useState',(_key:string,init:()=>unknown)=>ref(init()));
  mocks.app={$i18n:{t:(key:string)=>key},vueApp:{onUnmount:vi.fn()}};
  const session=getPowerXHostSession('com.example.adapter');const clear=vi.fn();
  await session.apply({source:'powerx',type:'auth-token',protocolVersion:2,pluginId:'com.example.adapter',instanceId:'frame',sequence:1,sessionId:'login',revision:1,accessToken:'fixture',tokenType:'Bearer',scope:'powerx',tenantUuid:null,expiresAt:Date.now()+60000,expiresIn:60},{readContext:async()=>({}),validateContext:()=>({tenantUuid:null,memberUuid:null}),commitContext:vi.fn(),clearContext:clear});clear.mockClear();
  const {useEmbeddedAuth}=await import('../../app/composables/useEmbeddedAuth');useEmbeddedAuth()[method]();
  expect(clear).toHaveBeenCalledExactlyOnceWith('identity');expect(session.getState().clearScope).toBe('identity');
});
