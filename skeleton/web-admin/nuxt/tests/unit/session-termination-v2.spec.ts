import { afterEach, describe, expect, it, vi } from 'vitest';
import { PowerXHostSession, createPowerXApiFetchClient, getPowerXHostSession, createPluginSSEClient, createPluginWsClient, createPluginWsBusClient, type PowerXAuthTokenPayload } from '../../../../../framework/frontend/nuxt/framework-client';

const pluginId = 'com.example.termination';
const tenant = '22222222-2222-4222-8222-222222222222';
const member = '33333333-3333-4333-8333-333333333333';
const invalid = 'IAM_CONTEXT_USER_SESSION_INVALID';
const auth = (sequence = 1, changes: Partial<PowerXAuthTokenPayload> = {}): PowerXAuthTokenPayload => ({ source: 'powerx', type: 'auth-token', protocolVersion: 2, pluginId, instanceId: 'frame-1', sequence, sessionId: 'login-1', revision: sequence, accessToken: `fixture-${sequence}`, tokenType: 'Bearer', expiresAt: Date.now() + 60_000, expiresIn: 60, scope: 'powerx', tenantUuid: tenant, ...changes });
const adapter = () => ({ readContext: vi.fn(async () => ({})), validateContext: vi.fn(() => ({ tenantUuid: tenant, memberUuid: member })), commitContext: vi.fn(), clearContext: vi.fn() });
function windowStub() {
  const bridge = { protocolVersion: 2, onAuthToken: vi.fn(), setNavigator: vi.fn(), requestAuth: vi.fn(), requestSync: vi.fn(), getLastSyncAt: () => 0, reportSessionInvalid: vi.fn() };
  vi.stubGlobal('window', { location: { origin: 'https://host.test' }, PowerXBridgeClient: bridge }); return bridge;
}
const deferred = <T>() => { let resolve!: (value: T) => void; let reject!: (error: unknown) => void; const promise = new Promise<T>((yes,no) => { resolve = yes; reject = no; }); return { promise, resolve, reject }; };
function client(session: PowerXHostSession, transport: ReturnType<typeof vi.fn>) { return createPowerXApiFetchClient({ pluginId, hostSession: session, insidePowerX: true, baseURL: `/_p/${pluginId}/api/v1`, fetcher: { create: () => transport } }).client; }
const denial = (code = invalid, status = 401) => ({ response: { status, _data: { error_code: code, trace_id: 'fixture-trace' } } });
afterEach(() => { vi.unstubAllGlobals(); vi.useRealTimers(); });

describe('terminal_session_http', () => {
  it('reports_once_for_parallel_invalid_and_never_replays_a_write_after_login', async () => {
    const bridge = windowStub(); const session = new PowerXHostSession(pluginId); const a = adapter(); await session.apply(auth(), a); a.clearContext.mockClear();
    const requestAuth = vi.fn(); session.setRequester(requestAuth);
    const transport = vi.fn().mockRejectedValue(denial()); const api = client(session, transport);
    await Promise.allSettled([api.post('/orders', { amount: 1 }), api.get('/orders'), api.get('/orders')]);
    expect(bridge.reportSessionInvalid).toHaveBeenCalledExactlyOnceWith({ sessionId: 'login-1', revision: 1, code: invalid });
    expect(requestAuth).not.toHaveBeenCalled(); expect(a.clearContext).toHaveBeenCalledWith('identity'); expect(session.getReadySnapshot()).toBeNull();
    await expect(session.ensureSnapshot({ recovery: true })).rejects.toMatchObject({ code: invalid }); expect(requestAuth).not.toHaveBeenCalled();
    await session.apply(auth(2, { sessionId: 'login-2' }), a); expect(transport).toHaveBeenCalledTimes(3);
  });
  it.each(['IAM_CONTEXT_PROOF_INVALID','AUTH_STS_EXPIRED','AUTH_TOKEN_INVALID','IAM_UNAUTHORIZED'])('does_not_report_other_machine_codes_%s', async code => {
    const bridge = windowStub(); const session = new PowerXHostSession(pluginId); await session.apply(auth(), adapter()); const transport = vi.fn().mockRejectedValue(denial(code));
    await expect(client(session, transport).get('/records')).rejects.toBeDefined(); expect(bridge.reportSessionInvalid).not.toHaveBeenCalled();
  });
  it.each([403,503])('does_not_report_wrong_status_%s', async status => {
    const bridge = windowStub(); const session = new PowerXHostSession(pluginId); await session.apply(auth(), adapter()); const snapshot = session.getReadySnapshot();
    await expect(client(session, vi.fn().mockRejectedValue(denial(invalid,status))).get('/records')).rejects.toBeDefined();
    expect(bridge.reportSessionInvalid).not.toHaveBeenCalled(); expect(session.getReadySnapshot()).toBe(snapshot);
  });
  it('ignores_late_failure_from_previous_revision', async () => {
    const bridge = windowStub(); const session = new PowerXHostSession(pluginId); const a = adapter(); await session.apply(auth(),a);
    const pending = deferred<any>(); const transport = vi.fn(() => pending.promise); const result = client(session,transport).get('/records'); const rejected = expect(result).rejects.toMatchObject({ code: 'AUTH_SESSION_CHANGED' });
    await vi.waitFor(() => expect(transport).toHaveBeenCalledOnce()); await session.apply(auth(2),a); pending.reject(denial()); await rejected;
    expect(bridge.reportSessionInvalid).not.toHaveBeenCalled(); expect(session.getReadySnapshot()?.revision).toBe(2);
  });
  it('aborted_request_cannot_report_a_late_invalid', async () => {
    const bridge = windowStub(); const session = new PowerXHostSession(pluginId); await session.apply(auth(),adapter());
    const pending = deferred<any>(); const transport = vi.fn(() => pending.promise); const controller = new AbortController();
    const result = client(session,transport).get('/records',{ signal: controller.signal }); const rejected = expect(result).rejects.toBeDefined();
    await vi.waitFor(() => expect(transport).toHaveBeenCalledOnce()); controller.abort(); pending.reject(denial()); await rejected;
    expect(bridge.reportSessionInvalid).not.toHaveBeenCalled();
  });
  it('old_core_feature_absence_is_explicit_and_does_not_refresh', async () => {
    const bridge = windowStub(); delete (bridge as any).reportSessionInvalid;
    const session = new PowerXHostSession(pluginId); await session.apply(auth(),adapter()); const request = vi.fn(); session.setRequester(request);
    await expect(client(session,vi.fn().mockRejectedValue(denial())).get('/records')).rejects.toMatchObject({ code: 'PLUGIN_BRIDGE_SESSION_TERMINATION_REQUIRED', statusCode: 409 });
    expect(session.getState().error).toBe('PLUGIN_BRIDGE_SESSION_TERMINATION_REQUIRED'); expect(request).not.toHaveBeenCalled(); expect(bridge.requestAuth).not.toHaveBeenCalled();
  });
  it('synchronous_new_login_during_report_is_not_cleared', async () => {
    const bridge = windowStub(); const session = new PowerXHostSession(pluginId); const a=adapter(); await session.apply(auth(),a);
    bridge.reportSessionInvalid.mockImplementation(() => { void session.apply(auth(2,{sessionId:'login-2'}),a); });
    await expect(client(session,vi.fn().mockRejectedValue(denial())).get('/records')).rejects.toBeDefined();
    await vi.waitFor(() => expect(session.getReadySnapshot()?.sessionId).toBe('login-2'));
  });
  it('same_session_verified_resync_preserves_business_scope_but_new_identity_does_not', async () => {
    windowStub(); const session=new PowerXHostSession(pluginId);const a=adapter();await session.apply(auth(),a);a.clearContext.mockClear();
    session.reportSessionInvalid(session.getReadySnapshot()!,invalid);await session.apply(auth(2,{revision:1}),a);
    expect(a.clearContext.mock.calls.every(([scope])=>scope==='identity')).toBe(true);
    a.validateContext.mockReturnValue({tenantUuid:tenant,memberUuid:'44444444-4444-4444-8444-444444444444'});await session.apply(auth(3),a);expect(a.clearContext).toHaveBeenCalledWith('session');
  });
  it('expiry_retry_cannot_replay_a_write_into_another_member', async()=>{
    windowStub();const session=new PowerXHostSession(pluginId);const a=adapter();await session.apply(auth(),a);
    session.setRequester(()=>{a.validateContext.mockReturnValue({tenantUuid:tenant,memberUuid:'44444444-4444-4444-8444-444444444444'});void session.apply(auth(2),a);});
    const transport=vi.fn().mockRejectedValue(denial('IAM_CONTEXT_USER_SESSION_EXPIRED'));
    await expect(client(session,transport).post('/orders',{})).rejects.toMatchObject({code:'AUTH_SESSION_CHANGED'});expect(transport).toHaveBeenCalledOnce();
  });
});

describe('initial_context_failure',()=>{
  it.each([401,503])('preserves_code_status_cause_and_waiters_%s',async status=>{
    const bridge=windowStub();const session=new PowerXHostSession(pluginId);session.setRequester(vi.fn());const a=adapter();const pending=deferred<any>();a.readContext.mockImplementation(()=>pending.promise);
    const original=denial(status===401?invalid:'AUTH_SESSION_UNAVAILABLE',status);const apply=session.apply(auth(),a);const rejected=expect(apply).rejects.toMatchObject({statusCode:status,cause:original});
    const waiter=expect(session.ensureSnapshot()).rejects.toMatchObject({statusCode:status,cause:original});pending.reject(original);await rejected;await waiter;
    expect(session.getState().error).toBe(status===401?invalid:'AUTH_SESSION_UNAVAILABLE');expect(a.clearContext).toHaveBeenLastCalledWith('identity');expect(bridge.reportSessionInvalid).not.toHaveBeenCalled();
  });
});

class Socket extends EventTarget {
  static instances: Socket[]=[];static OPEN=1;static CONNECTING=0;static CLOSED=3;readyState=0;closed=false;onmessage:((event:MessageEvent)=>void)|null=null;onclose:((event:any)=>void)|null=null;onopen:any;onerror:any;
  constructor(readonly url:string){super();Socket.instances.push(this);}
  close(){if(this.closed)return;this.closed=true;this.readyState=3;this.dispatchEvent(new Event('close'));this.onclose?.({code:1000});}
  send(){}
  receive(payload:unknown){const event=new MessageEvent('message',{data:JSON.stringify(payload)});this.dispatchEvent(event);if(!this.closed)this.onmessage?.(event);}
}
class Source extends EventTarget {closed=false;onopen:any;onerror:any;onmessage:any;constructor(readonly url:string){super();}close(){this.closed=true;}}
describe('terminal_realtime',()=>{
  it.each(['ws','bus','eventsource','fetch-sse'])('reports_only_current_structured_error_%s',async mode=>{
    const bridge=windowStub();vi.stubGlobal('WebSocket',Socket);vi.stubGlobal('EventSource',Source);const session=getPowerXHostSession(pluginId);await session.apply(auth(),adapter());
    if(mode==='ws'){const socket=createPluginWsClient({pluginId,insidePowerX:true}).connect() as unknown as Socket;socket.receive({type:'error',payload:{code:invalid}});expect(socket.closed).toBe(true);}
    if(mode==='bus'){const bus=createPluginWsBusClient({pluginId,insidePowerX:true});bus.connect();Socket.instances.at(-1)!.receive({type:'error',payload:{code:invalid}});expect(bus.getState().connected).toBe(false);bus.disconnect();}
    if(mode==='eventsource'){const source=createPluginSSEClient({pluginId,insidePowerX:true}).connect({path:'/events'}) as unknown as Source;source.dispatchEvent(new MessageEvent('error',{data:JSON.stringify({error_code:invalid})}));expect(source.closed).toBe(true);}
    if(mode==='fetch-sse'){vi.stubGlobal('fetch',vi.fn(async()=>new Response(JSON.stringify({error_code:invalid}),{status:401})));await expect(createPluginSSEClient({pluginId,insidePowerX:true}).stream({path:'/events'})).rejects.toMatchObject({status:401});}
    expect(bridge.reportSessionInvalid).toHaveBeenCalledOnce();expect(session.getReadySnapshot()).toBeNull();
  });
  it('fetch_sse_expiry_renews_once_without_replaying_streamed_events',async()=>{
    const bridge=windowStub();const session=getPowerXHostSession(pluginId);const a=adapter();await session.apply(auth(),a);
    session.setRequester(()=>{void session.apply(auth(2),a);});
    const fetcher=vi.fn().mockResolvedValueOnce(new Response(JSON.stringify({error_code:'IAM_CONTEXT_USER_SESSION_EXPIRED'}),{status:401})).mockResolvedValueOnce(new Response('data: {"ready":true}\n\n',{status:200,headers:{'Content-Type':'text/event-stream'}}));vi.stubGlobal('fetch',fetcher);
    const events=vi.fn();await createPluginSSEClient({pluginId,insidePowerX:true}).stream({path:'/events',onEvent:events});
    expect(fetcher).toHaveBeenCalledTimes(2);expect(fetcher.mock.calls[1][1].headers.Authorization).toBe('Bearer fixture-2');expect(events).toHaveBeenCalledOnce();expect(bridge.reportSessionInvalid).not.toHaveBeenCalled();
  });
  it.each([403,503])('fetch_sse_service_failure_never_reports_%s',async status=>{
    const bridge=windowStub();const session=getPowerXHostSession(pluginId);await session.apply(auth(),adapter());
    vi.stubGlobal('fetch',vi.fn(async()=>new Response(JSON.stringify({error_code:invalid}),{status})));
    await expect(createPluginSSEClient({pluginId,insidePowerX:true}).stream({path:'/events'})).rejects.toMatchObject({status});
    expect(bridge.reportSessionInvalid).not.toHaveBeenCalled();expect(session.getReadySnapshot()).not.toBeNull();
  });
  it('fetch_sse_error_frame_stops_before_business_delivery',async()=>{
    const bridge=windowStub();const session=getPowerXHostSession(pluginId);await session.apply(auth(),adapter());
    vi.stubGlobal('fetch',vi.fn(async()=>new Response(`event: error\ndata: {"error_code":"${invalid}"}\n\ndata: {"private":true}\n\n`,{status:200})));
    const events=vi.fn();await expect(createPluginSSEClient({pluginId,insidePowerX:true}).stream({path:'/events',onEvent:events})).rejects.toMatchObject({code:invalid});
    expect(events).not.toHaveBeenCalled();expect(bridge.reportSessionInvalid).toHaveBeenCalledOnce();
  });
  it('opaque_network_event_and_free_text_do_not_report',async()=>{
    const bridge=windowStub();vi.stubGlobal('WebSocket',Socket);vi.stubGlobal('EventSource',Source);const session=getPowerXHostSession(pluginId);await session.apply(auth(),adapter());
    const socket=createPluginWsClient({pluginId,insidePowerX:true}).connect() as unknown as Socket;socket.receive({type:'error',message:invalid});
    const source=createPluginSSEClient({pluginId,insidePowerX:true}).connect({path:'/events'}) as unknown as Source;source.dispatchEvent(new Event('error'));
    expect(bridge.reportSessionInvalid).not.toHaveBeenCalled();expect(session.getReadySnapshot()).not.toBeNull();socket.close();source.close();
  });
});
