import { readFileSync } from 'node:fs';
import vm from 'node:vm';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { initPowerXBridge, PowerXHostSession, createPowerXRouterNavigator, getPowerXFrameBridge } from '../../../../../framework/frontend/nuxt/framework-client';
import { createRouter, createMemoryHistory } from 'vue-router';

const pluginId = 'com.example.session';
const origin = 'https://host.test';
const mount = `/_p/${pluginId}/admin/`;
const flush = async () => { for (let i = 0; i < 12; i++) await Promise.resolve(); };
function setup() {
  const packets: any[] = []; const listeners: Array<{ fn: any; capture: boolean }> = [];
  const parent = { postMessage: (payload: any, target: string) => packets.push({ ...payload, target }) };
  const win: any = {
    location: new URL(mount + 'intro', origin), parent,
    addEventListener: (_type: string, fn: any, capture = false) => listeners.push({ fn, capture }),
    removeEventListener: (_type: string, fn: any) => { const index = listeners.findIndex(item => item.fn === fn); if (index >= 0) listeners.splice(index, 1); },
    __NUXT__: { config: { app: { baseURL: mount } } },
  };
  const doc = { referrer: origin + '/plugins' };
  vi.stubGlobal('window', win); vi.stubGlobal('document', doc);
  vm.runInNewContext(readFileSync(new URL('../fixtures/core-bridge-v2.js', import.meta.url), 'utf8'), { window: win, document: doc, URL, AbortController, console });
  const receive = (type: string, fields: Record<string, any> = {}, source: any = parent, messageOrigin = origin) => {
    let stopped = false;
    const event = { source, origin: messageOrigin, data: { source: 'powerx', protocolVersion: 2, pluginId, instanceId: 'frame-1', type, ...fields }, stopImmediatePropagation: () => { stopped = true; } };
    for (const listener of [...listeners].sort((a, b) => Number(b.capture) - Number(a.capture))) { if (stopped) break; listener.fn(event); }
  };
  const auth = (sequence = 1) => ({ sequence, sessionId: 'session-1', revision: sequence, accessToken: `access-${sequence}`, tokenType: 'Bearer', expiresAt: Date.now() + 60_000, expiresIn: 60, scope: 'powerx', tenantUuid: null });
  return { win, receive, packets, auth, parent, listeners };
}
afterEach(() => vi.unstubAllGlobals());

describe('core_v2_consumer_interoperability', () => {
  it('auth_applied_is_emitted_only_after_returned_context_promise', async () => {
    const f = setup(); const session = new PowerXHostSession(pluginId); let finish!: () => void;
    const pending = new Promise<void>(resolve => { finish = resolve; }); const read = vi.fn(() => pending);
    const bridge = initPowerXBridge({ pluginId, onAuthToken: message => session.apply(message, { readContext: read, validateContext: vi.fn(() => ({ tenantUuid: null, memberUuid: null })), commitContext: vi.fn(), clearContext: vi.fn() }) });
    f.receive('sync', { locale: 'zh', theme: 'dark', hostOrigin: origin }); f.receive('auth-token', f.auth()); await flush();
    expect(read).toHaveBeenCalledOnce(); expect(f.packets.filter(packet => packet.type === 'auth:applied')).toHaveLength(0);
    finish(); await flush(); expect(f.packets.filter(packet => packet.type === 'auth:applied')).toMatchObject([{ protocolVersion: 2, sessionId: 'session-1', revision: 1, sequence: 1, target: origin }]); bridge.stop();
  });
  it('current_validation_rejection_reaches_core', async () => {
    const f = setup(); const bridge = initPowerXBridge({ pluginId, onAuthToken: async () => { throw new Error('AUTH_MEMBER_CONTEXT_INVALID'); } });
    f.receive('sync'); f.receive('auth-token', f.auth()); await flush(); expect(f.packets.some(packet => packet.type === 'auth:rejected')).toBe(true); expect(f.packets.some(packet => packet.type === 'auth:applied')).toBe(false); bridge.stop();
  });
  it('rejects_forged_origin_window_plugin_instance_and_protocol', async () => {
    const f = setup(); const auth = vi.fn(); const sync = vi.fn(); const bridge = initPowerXBridge({ pluginId, onAuthToken: auth, onSync: sync }); f.receive('sync'); sync.mockClear();
    f.receive('auth-token', f.auth(), f.parent, 'https://other.test');
    f.receive('auth-token', f.auth(), f.win); f.receive('auth-token', { ...f.auth(), protocolVersion: 1 });
    f.receive('auth-token', { ...f.auth(), pluginId: 'other' }); f.receive('auth-token', { ...f.auth(), instanceId: 'old' });
    f.receive('sync', { hostOrigin: origin }, f.parent, 'https://other.test'); await flush(); expect(auth).not.toHaveBeenCalled(); expect(sync).not.toHaveBeenCalled(); bridge.stop();
  });
  it('late_registration_receives_the_latest_snapshot_once', async () => {
    const f = setup(); f.receive('sync'); f.receive('auth-token', f.auth()); const auth = vi.fn(); const bridge = initPowerXBridge({ pluginId, onAuthToken: auth }); await flush(); expect(auth).toHaveBeenCalledOnce(); bridge.stop();
    f.receive('auth-token', f.auth(2)); await flush(); expect(auth).toHaveBeenCalledOnce();
  });
  it('late_failure_cannot_reject_new_auth_or_restore_old_token', async () => {
    const f = setup(); const session = new PowerXHostSession(pluginId); let fail!: () => void;
    const pending = new Promise<void>((_resolve, reject) => { fail = () => reject(new Error('OLD_CONTEXT_FAILED')); });
    const read = vi.fn().mockReturnValueOnce(pending).mockResolvedValue(undefined);
    const bridge = initPowerXBridge({ pluginId, onAuthToken: message => session.apply(message, { readContext: read, validateContext: vi.fn(() => ({ tenantUuid: null, memberUuid: null })), commitContext: vi.fn(), clearContext: vi.fn() }) });
    f.receive('sync'); f.receive('auth-token', f.auth()); await flush(); f.receive('auth-token', { ...f.auth(2), sessionId: 'session-2' }); await flush(); fail(); await flush();
    expect(session.getReadySnapshot()?.sessionId).toBe('session-2'); expect(f.packets.filter(packet => packet.type === 'auth:applied')).toHaveLength(1); expect(f.packets.some(packet => packet.type === 'auth:rejected')).toBe(false); bridge.stop();
  });
  it('router_navigation_keeps_the_frame_and_confirms_actual_query_hash', async () => {
    const f = setup(); const router = createRouter({ history: createMemoryHistory(mount), routes: [{ path: '/intro', component: {} }, { path: '/customers', component: {} }] });
    await router.push('/intro'); router.afterEach(to => { f.win.location = new URL(mount.replace(/\/$/, '') + to.fullPath, origin); });
    const bridge = initPowerXBridge({ pluginId, navigator: createPowerXRouterNavigator(router, mount, origin) }); f.receive('sync');
    expect(f.packets.some(packet => packet.type === 'bridge:ready' && packet.navigationReady === true)).toBe(true);
    f.receive('navigate', { path: mount + 'customers?name=hello%20world#details', requestId: 'nav-1' });
    await new Promise(resolve => setTimeout(resolve, 0)); await flush();
    expect(router.currentRoute.value.path).toBe('/customers'); expect(router.currentRoute.value.query).toEqual({ name: 'hello world' }); expect(router.currentRoute.value.hash).toBe('#details');
    expect(f.packets.find(packet => packet.type === 'navigation:result')).toMatchObject({ requestId: 'nav-1', ok: true }); bridge.stop();
  });
  it('retires_the_legacy_auth_receiver_instead_of_running_both', () => {
    const f = setup(); const stop = vi.fn(); f.win.__POWERX_FRAMEWORK_BRIDGE__ = { stop };
    f.win.__POWERX_PLUGIN_HOST_SESSION_CACHE__ = new Map();
    const bridge = initPowerXBridge({ pluginId }); expect(stop).toHaveBeenCalledOnce();
    expect(f.win.__POWERX_FRAMEWORK_BRIDGE__).toBeUndefined(); expect(f.win.__POWERX_PLUGIN_HOST_SESSION_CACHE__).toBeUndefined(); bridge.stop();
  });
  it('protocol_absence_and_wildcard_fail_explicitly', () => {
    const f = setup(); f.win.PowerXBridgeClient.protocolVersion = 1; expect(() => getPowerXFrameBridge()).toThrow('PLUGIN_BRIDGE_V2_REQUIRED'); f.win.PowerXBridgeClient.protocolVersion = 2;
    expect(() => initPowerXBridge({ pluginId, allowedOrigins: ['*'] })).toThrow('PLUGIN_BRIDGE_ORIGIN_INVALID');
  });
});
