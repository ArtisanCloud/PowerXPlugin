import { afterEach, describe, expect, it, vi } from 'vitest';
import { PowerXHostSession, createPowerXApiFetchClient, createPowerXRouterNavigator, resolvePowerXRouterPath, getPowerXHostSession, createPluginWsClient, createPluginWsBusClient, createPluginSSEClient, usePluginApi, readSSEStream, type PowerXAuthTokenPayload } from '../../../../../framework/frontend/nuxt/framework-client';
import { validatePowerXUserContext } from '../../app/utils/host-context';

const pluginId = 'com.example.session';
const tenant = 'a1512a2b-e54f-413d-a957-1721379a3a9a';
const member = '2c062e30-3706-4eb3-a279-6fe4bfcaf6e5';
const message = (sequence = 1, changes: Partial<PowerXAuthTokenPayload> = {}): PowerXAuthTokenPayload => ({ source: 'powerx', type: 'auth-token', protocolVersion: 2, pluginId, instanceId: 'frame-1', sequence, sessionId: 'session-1', revision: sequence, accessToken: `access-${sequence}`, tokenType: 'Bearer', expiresAt: Date.now() + 60_000, expiresIn: 60, scope: 'powerx', tenantUuid: tenant, ...changes });
const deferred = <T>() => { let resolve!: (value: T) => void; let reject!: (cause: unknown) => void; const promise = new Promise<T>((yes, no) => { resolve = yes; reject = no; }); return { promise, resolve, reject }; };
const adapter = () => ({ readContext: vi.fn(async () => ({ user: true })), validateContext: vi.fn(() => ({ tenantUuid: tenant, memberUuid: member })), commitContext: vi.fn(), clearContext: vi.fn() });
const windowStub = () => vi.stubGlobal('window', { location: { origin: 'https://host.test' } });
const requestClient = (session: PowerXHostSession, fetch: ReturnType<typeof vi.fn>, onAuthError = vi.fn()) => {
  fetch.raw = fetch;
  return createPowerXApiFetchClient({ pluginId, insidePowerX: true, hostSession: session, baseURL: `/_p/${pluginId}/api/v1`, fetcher: { create: () => fetch }, getFallbackToken: () => 'obsolete-token', onAuthError }).client;
};
afterEach(() => { vi.useRealTimers(); vi.unstubAllGlobals(); });

describe('host_session_v2', () => {
  it('publishes_only_after_context_validation', async () => {
    const session = new PowerXHostSession(pluginId); const a = adapter(); const read = deferred<any>(); a.readContext.mockReturnValue(read.promise);
    const applied = session.apply(message(), a);
    expect(session.getReadySnapshot()).toBeNull(); expect(a.commitContext).not.toHaveBeenCalled();
    read.resolve({ user: true }); await applied;
    expect(session.getReadySnapshot()?.accessToken).toBe('access-1'); expect(a.validateContext).toHaveBeenCalledOnce(); expect(a.commitContext).toHaveBeenCalledOnce();
  });
  it('old_context_success_cannot_replace_new_login', async () => {
    const session = new PowerXHostSession(pluginId); const a = adapter(); const read = deferred<any>(); a.readContext.mockReturnValueOnce(read.promise);
    const old = session.apply(message(), a);
    await session.apply(message(2, { sessionId: 'session-2' }), a);
    read.resolve({ user: 'obsolete' }); await old;
    expect(session.getReadySnapshot()?.sessionId).toBe('session-2'); expect(a.commitContext).toHaveBeenCalledOnce();
  });
  it('old_context_failure_does_not_clear_current_session', async () => {
    const session = new PowerXHostSession(pluginId); const a = adapter(); const read = deferred<any>(); a.readContext.mockReturnValueOnce(read.promise);
    const old = session.apply(message(), a); await session.apply(message(2, { sessionId: 'session-2' }), a);
    const clears = a.clearContext.mock.calls.length;
    read.reject(new Error('OLD_CONTEXT_FAILED')); await old;
    expect(session.getReadySnapshot()?.sessionId).toBe('session-2'); expect(a.clearContext).toHaveBeenCalledTimes(clears);
  });
  it('rejects_current_context_failure', async () => {
    const session = new PowerXHostSession(pluginId); const a = adapter(); a.validateContext.mockImplementation(() => { throw new Error('AUTH_MEMBER_CONTEXT_INVALID'); });
    await expect(session.apply(message(), a)).rejects.toThrow('AUTH_MEMBER_CONTEXT_INVALID'); expect(session.getState().status).toBe('rejected'); expect(a.commitContext).not.toHaveBeenCalled();
  });
  it('rejects_old_sequence_revision_and_instance', async () => {
    const session = new PowerXHostSession(pluginId); const a = adapter(); await session.apply(message(3), a);
    await session.apply(message(2), a); expect(a.readContext).toHaveBeenCalledOnce();
    await expect(session.apply(message(4, { revision: 2 }), a)).rejects.toMatchObject({ code: 'AUTH_SESSION_REVISION_STALE' });
    await expect(session.apply(message(5, { instanceId: 'old-frame' }), a)).rejects.toMatchObject({ code: 'PLUGIN_BRIDGE_INSTANCE_STALE' });
    expect(session.getReadySnapshot()?.sequence).toBe(3);
  });
  it('merges_auth_requests_and_waits_for_verified_snapshot', async () => {
    const session = new PowerXHostSession(pluginId); const request = vi.fn(); session.setRequester(request);
    const a = adapter(); const read = deferred<any>(); a.readContext.mockReturnValue(read.promise);
    const first = session.ensureSnapshot(); const second = session.ensureSnapshot(); expect(request).toHaveBeenCalledOnce();
    const applied = session.apply(message(), a); const third = session.ensureSnapshot(); expect(request).toHaveBeenCalledOnce();
    read.resolve({ user: true }); await applied;
    expect(await first).toBe(await second); expect(await third).toBe(session.getReadySnapshot());
  });
  it('logout_cancels_refresh_and_rejects_late_auth', async () => {
    const session = new PowerXHostSession(pluginId); session.setRequester(vi.fn()); const a = adapter(); await session.apply(message(), a);
    const waiting = expect(session.ensureSnapshot({ force: true })).rejects.toMatchObject({ code: 'PLUGIN_AUTH_REQUIRED' });
    await session.apply({ ...message(3), type: 'auth-cleared' }, a); await waiting;
    await session.apply(message(2), a); expect(session.getReadySnapshot()).toBeNull();
  });
  it('new_login_rejects_waiters_for_previous_session', async () => {
    const session = new PowerXHostSession(pluginId); session.setRequester(vi.fn()); const a = adapter(); await session.apply(message(), a);
    const waiting = expect(session.ensureSnapshot({ force: true })).rejects.toMatchObject({ code: 'AUTH_SESSION_CHANGED' });
    await session.apply(message(2, { sessionId: 'session-2' }), a); await waiting;
  });
  it('times_out_without_polling', async () => {
    vi.useFakeTimers(); const session = new PowerXHostSession(pluginId); const request = vi.fn(); session.setRequester(request);
    const waiting = expect(session.ensureSnapshot({ timeoutMs: 50 })).rejects.toMatchObject({ code: 'PLUGIN_AUTH_ADAPTER_TIMEOUT' });
    await vi.advanceTimersByTimeAsync(51); await waiting; expect(request).toHaveBeenCalledOnce();
  });
  it('ordinary_requests_cannot_wait_across_a_cleared_session', async () => {
    const session = new PowerXHostSession(pluginId); const request = vi.fn(); session.setRequester(request); const a = adapter(); await session.apply(message(), a);
    await session.apply({ ...message(2), type: 'auth-cleared' }, a);
    await expect(session.ensureSnapshot()).rejects.toMatchObject({ code: 'PLUGIN_AUTH_REQUIRED' }); expect(request).not.toHaveBeenCalled();
    const recovered = session.ensureSnapshot({ recovery: true }); expect(request).toHaveBeenCalledOnce();
    await session.apply(message(3, { sessionId: 'session-2' }), a); expect((await recovered).sessionId).toBe('session-2');
  });
  it('old_adapter_disposal_does_not_invalidate_new_owner', async () => {
    const session = new PowerXHostSession(pluginId); const first = vi.fn(); const second = vi.fn(); const stopFirst = session.setRequester(first); session.setRequester(second);
    await session.apply(message(), adapter()); stopFirst(); expect(session.getReadySnapshot()).not.toBeNull(); expect(session.ownsRequester(second)).toBe(true);
  });
  it('does_not_share_browser_sessions_with_ssr', () => { expect(() => getPowerXHostSession(pluginId)).toThrow('HOST_SESSION_BROWSER_REQUIRED'); });
  it('validates_member_tenant_and_root_rules', () => {
    const snapshot = { ...message(), epoch: 1 };
    const context: any = { is_root: false, permission_codes: [], user: { status: 1 }, current_tenant_uuid: tenant, current_member_uuid: member, members: [{ tenant_uuid: tenant, member_uuid: member }] };
    expect(() => validatePowerXUserContext(context, snapshot)).not.toThrow();
    expect(() => validatePowerXUserContext({ ...context, current_member_uuid: '' }, snapshot)).toThrow('AUTH_MEMBER_CONTEXT_INVALID');
    expect(() => validatePowerXUserContext({ ...context, current_tenant_uuid: '' }, snapshot)).toThrow('AUTH_TENANT_MISMATCH');
    expect(() => validatePowerXUserContext({ ...context, is_root: true, current_member_uuid: '' }, snapshot)).not.toThrow();
    expect(() => validatePowerXUserContext({ is_root: true, permission_codes: [], user: { status: 1 }, current_tenant_uuid: '', members: [] } as any, { ...snapshot, tenantUuid: null })).not.toThrow();
  });
});

describe('session_scoped_http', () => {
  it('uses_memory_session_and_preserves_http_options', async () => {
    windowStub(); const session = new PowerXHostSession(pluginId); await session.apply(message(), adapter()); const fetch = vi.fn(async () => ({ success: true }));
    const api = requestClient(session, fetch); await api.post('/records', { uuid: 'record' }, { headers: { Authorization: 'Bearer obsolete', 'X-PowerX-Identity-Context': 'browser-spoof' }, params: new URLSearchParams({ page: '2' }) });
    const options = fetch.mock.calls[0][1]; expect(options.headers.get('Authorization')).toBe('Bearer access-1'); expect(options.headers.get('tenant_uuid')).toBe(tenant); expect(options.headers.get('X-PowerX-Identity-Context')).toBeNull(); expect(options.query).toEqual({ page: '2' }); expect(options.body).toBe('{"uuid":"record"}'); expect(options.retry).toBe(0);
  });
  it('401_waits_for_actual_verified_renewal_and_retries_once', async () => {
    vi.useFakeTimers(); windowStub(); const session = new PowerXHostSession(pluginId); const a = adapter(); await session.apply(message(), a); const requested = vi.fn(); session.setRequester(requested);
    const fetch = vi.fn().mockRejectedValueOnce({ response: { status: 401, _data: { error: { code: 'AUTH_TOKEN_EXPIRED' } } } }).mockResolvedValue({ ok: true }); const api = requestClient(session, fetch);
    const response = api.get('/records'); await vi.advanceTimersByTimeAsync(301);
    expect(fetch).toHaveBeenCalledOnce(); expect(requested).toHaveBeenCalledOnce();
    const read = deferred<any>(); a.readContext.mockReturnValueOnce(read.promise); const apply = session.apply(message(2), a); await vi.advanceTimersByTimeAsync(0); expect(fetch).toHaveBeenCalledOnce();
    read.resolve({ user: true }); await apply; expect(await response).toEqual({ ok: true }); expect(fetch).toHaveBeenCalledTimes(2); expect(fetch.mock.calls[1][1].headers.get('Authorization')).toBe('Bearer access-2');
  });
  it('root_request_scope_comes_from_verified_core_context', async () => {
    windowStub(); const session = new PowerXHostSession(pluginId); const a = adapter();
    await session.apply(message(1, { tenantUuid: null }), a);
    const ready = session.getReadySnapshot()!; expect(ready.tenantUuid).toBeNull(); expect(ready.identity?.tenantUuid).toBe(tenant);
    const fetch = vi.fn(async () => ({})); await requestClient(session, fetch).get('/records', { tenantUuid: tenant });
    expect(fetch.mock.calls[0][1].headers.get('tenant_uuid')).toBe(tenant);
  });
  it('403_does_not_request_auth', async () => {
    windowStub(); const session = new PowerXHostSession(pluginId); await session.apply(message(), adapter()); const request = vi.fn(); session.setRequester(request);
    const fetch = vi.fn().mockRejectedValue({ response: { status: 403 } }); await expect(requestClient(session, fetch).get('/records')).rejects.toMatchObject({ response: { status: 403 } }); expect(request).not.toHaveBeenCalled(); expect(fetch).toHaveBeenCalledOnce();
  });
  it('second_401_has_no_refresh_loop', async () => {
    windowStub(); const session = new PowerXHostSession(pluginId); const a = adapter(); await session.apply(message(), a); session.setRequester(() => { void session.apply(message(2), a); });
    const fetch = vi.fn().mockRejectedValue({ response: { status: 401, _data: { error: { code: 'AUTH_TOKEN_EXPIRED' } } } }); await expect(requestClient(session, fetch).get('/records')).rejects.toMatchObject({ response: { status: 401 } }); expect(fetch).toHaveBeenCalledTimes(2);
  });
  it('new_login_prevents_401_retry', async () => {
    windowStub(); const session = new PowerXHostSession(pluginId); const a = adapter(); await session.apply(message(), a); const requested = vi.fn(); session.setRequester(requested); const read = deferred<any>(); const fetch = vi.fn().mockReturnValue(read.promise);
    const rejected = expect(requestClient(session, fetch).get('/records')).rejects.toMatchObject({ code: 'AUTH_SESSION_CHANGED' }); await Promise.resolve(); await Promise.resolve();
    await session.apply(message(2, { sessionId: 'session-2' }), a); read.reject({ response: { status: 401, _data: { error: { code: 'AUTH_TOKEN_EXPIRED' } } } }); await rejected; expect(requested).not.toHaveBeenCalled(); expect(fetch).toHaveBeenCalledOnce();
  });
  it.each(['success', 'failure'])('discards_late_%s_after_logout', async kind => {
    windowStub(); const session = new PowerXHostSession(pluginId); const a = adapter(); await session.apply(message(), a); const read = deferred<any>(); const fetch = vi.fn().mockReturnValue(read.promise); const onError = vi.fn();
    const result = expect(requestClient(session, fetch, onError).get('/records')).rejects.toMatchObject({ code: 'AUTH_SESSION_CHANGED' }); await Promise.resolve(); await Promise.resolve();
    const signal = fetch.mock.calls[0][1].signal;
    await session.apply({ ...message(2), type: 'auth-cleared' }, a); expect(signal.aborted).toBe(true);
    if (kind === 'success') read.resolve({ sensitive: true }); else read.reject({ response: { status: 403 } }); await result; expect(onError).not.toHaveBeenCalled();
  });
  it('context_request_uses_only_current_candidate_without_recursion', async () => {
    windowStub(); const session = new PowerXHostSession(pluginId); const requested = vi.fn(); session.setRequester(requested); const fetch = vi.fn(async () => ({ user: true })); const api = requestClient(session, fetch); const a = adapter();
    a.readContext.mockImplementation(snapshot => api.get('/admin/user/auth/me/context', { hostSnapshot: snapshot })); await session.apply(message(), a);
    expect(requested).not.toHaveBeenCalled(); expect(fetch.mock.calls[0][1].headers.get('Authorization')).toBe('Bearer access-1');
  });
  it('refuses_managed_auth_writes_and_other_scopes', async () => {
    windowStub(); const session = new PowerXHostSession(pluginId); await session.apply(message(), adapter()); const fetch = vi.fn(); const api = requestClient(session, fetch);
    for (const action of ['login', 'refresh', 'logout']) await expect(api.post(`/admin/user/auth/${action}`, {}, { skipAuth: true })).rejects.toMatchObject({ code: 'AUTH_SESSION_HOST_MANAGED', statusCode: 409 });
    await expect(api.get('https://other.test/records')).rejects.toMatchObject({ code: 'PLUGIN_API_SCOPE_INVALID' });
    await expect(api.get('//other.test/records')).rejects.toMatchObject({ code: 'PLUGIN_API_SCOPE_INVALID' }); await expect(api.get('/records', { tenantUuid: 'other' })).rejects.toMatchObject({ code: 'AUTH_TENANT_MISMATCH' }); expect(fetch).not.toHaveBeenCalled();
  });
  it('public_native_fetch_entrypoint_uses_shared_host_session', async () => {
    windowStub(); const session = getPowerXHostSession(pluginId); await session.apply(message(), adapter());
    const fetch = vi.fn(async () => new Response(JSON.stringify({ uuid: 'record' }), { headers: { 'Content-Type': 'application/json' } })); vi.stubGlobal('fetch', fetch);
    const api = usePluginApi({ pluginId, insidePowerX: true }); expect(await api.get('/records')).toEqual({ uuid: 'record' });
    expect(fetch.mock.calls[0][1].headers.get('Authorization')).toBe('Bearer access-1');
    await session.apply({ ...message(2), type: 'auth-cleared' }, adapter()); await expect(api.get('/records')).rejects.toMatchObject({ code: 'PLUGIN_AUTH_REQUIRED' }); expect(fetch).toHaveBeenCalledOnce();
  });
  it('standalone_keeps_its_own_auth_flow', async () => {
    const fetch = vi.fn(async () => ({ ok: true })); const ownAuth = { getToken: () => 'standalone-token' }; const client = createPowerXApiFetchClient({ pluginId, baseURL: '/api/v1', fetcher: { create: () => fetch }, getAuth: () => ownAuth }).client;
    await client.get('/records'); expect(fetch.mock.calls[0][1].headers.get('Authorization')).toBe('Bearer standalone-token');
  });
});

describe('router_navigation', () => {
  const base = `/_p/${pluginId}/admin/`;
  it('strips_exact_mount_preserving_query_hash_and_unicode', () => {
    expect(resolvePowerXRouterPath(base + 'customers?tab=identities#details', base, 'https://host.test')).toBe('/customers?tab=identities#details');
    expect(resolvePowerXRouterPath(base.slice(0, -1), base, 'https://host.test')).toBe('/');
    expect(() => resolvePowerXRouterPath('/_p/another/admin/items', base, 'https://host.test')).toThrow('PLUGIN_NAVIGATION_INVALID');
    expect(() => resolvePowerXRouterPath('//other.test/x', base, 'https://host.test')).toThrow('PLUGIN_NAVIGATION_INVALID');
  });
  it('waits_for_router_completion_and_honors_abort_during_guards', async () => {
    const pushed = deferred<void>(); const dispose = vi.fn(); let guard: any; const router = { push: vi.fn(() => pushed.promise), beforeResolve: vi.fn((fn: any) => { guard = fn; return dispose; }) }; const controller = new AbortController();
    const result = expect(createPowerXRouterNavigator(router, base, 'https://host.test')(base + 'deep?x=1#part', { signal: controller.signal, instanceId: 'frame-1', requestId: 'nav-1' })).rejects.toThrow('PLUGIN_NAVIGATION_ABORTED');
    expect(router.push).toHaveBeenCalledWith('/deep?x=1#part'); controller.abort(); expect(guard({ fullPath: '/deep?x=1#part' })).toBe(false); pushed.resolve(); await result; expect(dispose).toHaveBeenCalledOnce();
  });
  it('does_not_start_aborted_navigation', async () => {
    const controller = new AbortController(); controller.abort(); const router = { push: vi.fn(), beforeResolve: vi.fn() };
    await expect(createPowerXRouterNavigator(router, base, 'https://host.test')(base, { signal: controller.signal, instanceId: 'frame-1', requestId: 'nav-1' })).rejects.toThrow('PLUGIN_NAVIGATION_ABORTED'); expect(router.push).not.toHaveBeenCalled();
  });
});

class Socket { static OPEN = 1; static CONNECTING = 0; static CLOSED = 3; readyState = 0; closed = false; constructor(readonly url: string) {} addEventListener() {} send() {} close() { this.closed = true; this.readyState = 3; } }
class Source { closed = false; constructor(readonly url: string) {} addEventListener() {} close() { this.closed = true; } }
describe('host_realtime', () => {
  it('ws_and_sse_use_shared_session_and_close_on_logout', async () => {
    windowStub(); vi.stubGlobal('WebSocket', Socket); vi.stubGlobal('EventSource', Source); const session = getPowerXHostSession(pluginId); const a = adapter(); await session.apply(message(), a);
    const ws = createPluginWsClient({ pluginId, insidePowerX: true, token: 'obsolete', tenantUuid: 'obsolete' }).connect() as unknown as Socket;
    const sse = createPluginSSEClient({ pluginId, insidePowerX: true, token: 'obsolete' }).connect({ path: '/events' }) as unknown as Source;
    expect(new URL(ws.url).searchParams.get('authorization')).toBe('Bearer access-1'); expect(new URL(sse.url).searchParams.get('tenant_uuid')).toBe(tenant);
    await session.apply({ ...message(2), type: 'auth-cleared' }, a); expect(ws.closed).toBe(true); expect(sse.closed).toBe(true);
    expect(() => createPluginWsClient({ pluginId, insidePowerX: true }).connect()).toThrow('PLUGIN_AUTH_REQUIRED');
  });
  it('stream_reader_cancels_on_abort', async () => {
    const cancelled = vi.fn(); const body = new ReadableStream<Uint8Array>({ cancel: cancelled }); const controller = new AbortController();
    const result = expect(readSSEStream(body, vi.fn(), controller.signal)).rejects.toMatchObject({ code: 'AUTH_REQUEST_ABORTED' }); controller.abort(); await result; expect(cancelled).toHaveBeenCalledOnce();
  });
  it('bus_stops_old_events_and_reconnect_timer_when_session_changes', async () => {
    windowStub(); vi.stubGlobal('WebSocket', Socket); const session = getPowerXHostSession(pluginId); const a = adapter(); await session.apply(message(), a); const events = vi.fn();
    const bus = createPluginWsBusClient({ pluginId, insidePowerX: true, tenantUuid: tenant, onEvent: events }); bus.connect(); expect(bus.getState().status).toBe('connecting');
    await session.apply(message(2, { sessionId: 'session-2' }), a); expect(bus.getState().connected).toBe(false); expect(bus.getState().status).toBe('idle'); bus.disconnect();
  });
});
