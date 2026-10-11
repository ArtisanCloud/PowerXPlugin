import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  classifyPowerXApiAuthError, createPowerXApiFetchClient, getPowerXHostSession,
  PowerXHostSession, usePluginApi, type PowerXAuthTokenPayload,
} from '../../../../../framework/frontend/nuxt/framework-client';

const pluginId = 'com.example.auth-errors';
const tenantUuid = 'ca38f3a7-4073-440e-9b7c-73c5b30b6a09';
const memberUuid = 'd68c130c-d601-4a3b-b31d-9d7597d912e2';
const message = (sequence = 1): PowerXAuthTokenPayload => ({
  source: 'powerx', type: 'auth-token', protocolVersion: 2, pluginId, instanceId: 'frame-1',
  sequence, sessionId: 'session-1', revision: sequence, accessToken: `access-${sequence}`,
  tokenType: 'Bearer', expiresAt: Date.now() + 60_000, expiresIn: 60, scope: 'powerx', tenantUuid,
});
const adapter = () => ({
  readContext: vi.fn(async () => ({})), validateContext: vi.fn(() => ({ tenantUuid, memberUuid })),
  commitContext: vi.fn(), clearContext: vi.fn(),
});
const windowStub = () => vi.stubGlobal('window', { location: { origin: 'https://host.test' } });
const failures = [
  { status: 401, code: 'IAM_CONTEXT_PROOF_REQUIRED', kind: 'identity-proof' },
  { status: 401, code: 'IAM_CONTEXT_PROOF_INVALID', kind: 'identity-proof' },
  { status: 401, code: 'IAM_CONTEXT_PROOF_EXPIRED', kind: 'identity-proof' },
  { status: 401, code: 'IAM_CONTEXT_WS_HANDLE_EXPIRED', kind: 'identity-proof' },
  { status: 401, code: 'AUTH_STS_EXPIRED', kind: 'service-credential' },
  { status: 401, code: 'IAM_CONTEXT_SERVICE_CREDENTIAL_INVALID', kind: 'service-credential' },
  { status: 401, code: 'IAM_CONTEXT_SERVICE_CREDENTIAL_EXPIRED', kind: 'service-credential' },
  { status: 401, code: 'IAM_UNAUTHORIZED', kind: 'unknown' },
  { status: 401, code: 'AUTH_TOKEN_INVALID', kind: 'user-session-invalid' },
  { status: 401, code: 'IAM_CONTEXT_USER_SESSION_INVALID', kind: 'user-session-invalid' },
  { status: 401, code: undefined, kind: 'unknown' },
  { status: 403, code: 'IAM_CONTEXT_FORBIDDEN', kind: 'forbidden' },
  { status: 503, code: 'IAM_CONTEXT_UNAVAILABLE', kind: 'identity-unavailable' },
  { status: 424, code: 'IAM_UPSTREAM_DEPENDENCY', kind: 'identity-unavailable' },
] as const;
const payload = (code?: string) => ({ error: { code }, trace_id: 'trace-fixture', request_id: 'request-fixture' });

afterEach(() => { vi.unstubAllGlobals(); });

describe('structured_auth_error_classification', () => {
  it.each(failures)('status=$status code=$code', ({ status, code, kind }) => {
    expect(classifyPowerXApiAuthError({ status, _data: payload(code) })).toEqual({ code, kind });
  });
  it('accepts_explicit_user_expiry_and_prioritizes_specific_reason_codes', () => {
    for (const code of ['AUTH_TOKEN_EXPIRED', 'TOKEN_EXPIRED', 'IAM_CONTEXT_USER_SESSION_EXPIRED']) {
      expect(classifyPowerXApiAuthError({ status: 401, _data: payload(code) }).kind).toBe('user-session-expired');
    }
    expect(classifyPowerXApiAuthError({ status: 401, _data: {
      code: 'AUTH_TOKEN_EXPIRED', details: { reason_code: 'IAM_CONTEXT_PROOF_EXPIRED' },
    } })).toEqual({ code: 'IAM_CONTEXT_PROOF_EXPIRED', kind: 'identity-proof' });
    expect(classifyPowerXApiAuthError({ status: 401, _data: {
      code: 401, details: { code: 'IAM_CONTEXT_PROOF_INVALID' },
    } })).toEqual({ code: 'IAM_CONTEXT_PROOF_INVALID', kind: 'identity-proof' });
  });
  it('does_not_infer_expiry_from_free_text_or_wrong_http_status', () => {
    for (const _data of [{ message: 'TOKEN_EXPIRED' }, { error: 'TOKEN_EXPIRED' }, 'TOKEN_EXPIRED', null, []]) {
      expect(classifyPowerXApiAuthError({ status: 401, _data }).kind).toBe('unknown');
    }
    expect(classifyPowerXApiAuthError({ status: 502, _data: payload('AUTH_TOKEN_EXPIRED') }).kind).toBe('unknown');
  });
});

describe('request_identity_failure_isolation', () => {
  it.each(failures)('ofetch_status=$status code=$code', async ({ status, code }) => {
    windowStub(); const session = new PowerXHostSession(pluginId); const a = adapter();
    await session.apply(message(), a); const snapshot = session.getReadySnapshot(); a.clearContext.mockClear();
    const requestAuth = vi.fn(); session.setRequester(requestAuth);
    const error = { response: { status, _data: payload(code), headers: new Headers({ 'X-Trace-ID': 'trace-fixture' }) } };
    const transport = vi.fn().mockRejectedValue(error); const onAuthError = vi.fn();
    const client = createPowerXApiFetchClient({
      pluginId, insidePowerX: true, hostSession: session, baseURL: `/_p/${pluginId}/api/v1`,
      fetcher: { create: () => transport }, onAuthError,
    }).client;
    await expect(client.get('/records')).rejects.toBe(error);
    expect(transport).toHaveBeenCalledOnce(); expect(requestAuth).not.toHaveBeenCalled();
    expect(session.getReadySnapshot()).toBe(code === 'AUTH_TOKEN_INVALID' || code === 'IAM_CONTEXT_USER_SESSION_INVALID' ? null : snapshot);
    if (code === 'AUTH_TOKEN_INVALID' || code === 'IAM_CONTEXT_USER_SESSION_INVALID') expect(a.clearContext).toHaveBeenCalledOnce(); else expect(a.clearContext).not.toHaveBeenCalled();
    expect(onAuthError).toHaveBeenCalledWith(expect.objectContaining({ response: error.response }));
  });
  it.each(failures)('native_status=$status code=$code', async ({ status, code }) => {
    windowStub(); const session = getPowerXHostSession(pluginId); const a = adapter();
    await session.apply(message(), a); const snapshot = session.getReadySnapshot(); a.clearContext.mockClear();
    const requestAuth = vi.fn(); session.setRequester(requestAuth);
    const body = payload(code); const response = new Response(JSON.stringify(body), {
      status, headers: { 'Content-Type': 'application/json', 'X-Trace-ID': 'trace-fixture', 'X-Request-ID': 'request-fixture' },
    });
    const transport = vi.fn().mockResolvedValue(response); vi.stubGlobal('fetch', transport);
    const api = usePluginApi({ pluginId, insidePowerX: true });
    const error = await api.get('/records').catch(cause => cause);
    expect(error.response).toBe(response); expect(error.response.status).toBe(status);
    expect(error.response._data).toEqual(body);
    expect(error.response.headers.get('X-Trace-ID')).toBe('trace-fixture');
    expect(error.response.headers.get('X-Request-ID')).toBe('request-fixture');
    expect(transport).toHaveBeenCalledOnce(); expect(requestAuth).not.toHaveBeenCalled();
    expect(session.getReadySnapshot()).toBe(code === 'AUTH_TOKEN_INVALID' || code === 'IAM_CONTEXT_USER_SESSION_INVALID' ? null : snapshot);
    if (code === 'AUTH_TOKEN_INVALID' || code === 'IAM_CONTEXT_USER_SESSION_INVALID') expect(a.clearContext).toHaveBeenCalledOnce(); else expect(a.clearContext).not.toHaveBeenCalled();
  });
  it('native_user_expiry_waits_for_a_verified_snapshot_before_retrying', async () => {
    windowStub(); const session = getPowerXHostSession(pluginId); const a = adapter();
    await session.apply(message(), a); const requestAuth = vi.fn(async () => { await session.apply(message(2), a); });
    session.setRequester(requestAuth);
    const transport = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ code: 401, error_code: 'IAM_CONTEXT_USER_SESSION_EXPIRED' }), { status: 401, headers: { 'Content-Type': 'application/json' } }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ uuid: 'record-fixture' }), { headers: { 'Content-Type': 'application/json' } }));
    vi.stubGlobal('fetch', transport);
    expect(await usePluginApi({ pluginId, insidePowerX: true }).get('/records')).toEqual({ uuid: 'record-fixture' });
    expect(requestAuth).toHaveBeenCalledOnce(); expect(transport).toHaveBeenCalledTimes(2);
    expect(transport.mock.calls[1][1].headers.get('Authorization')).toBe('Bearer access-2');
  });
  it('malformed_json_denial_preserves_http_trace_and_does_not_request_auth', async () => {
    windowStub(); const session = getPowerXHostSession(pluginId); await session.apply(message(), adapter());
    const snapshot = session.getReadySnapshot(); const requestAuth = vi.fn(); session.setRequester(requestAuth);
    const response = new Response('{', { status: 401, headers: { 'Content-Type': 'application/json', 'X-Trace-ID': 'trace-fixture' } });
    const transport = vi.fn().mockResolvedValue(response); vi.stubGlobal('fetch', transport);
    const error = await usePluginApi({ pluginId, insidePowerX: true }).get('/records').catch(cause => cause);
    expect(error.response).toBe(response); expect(error.status).toBe(401); expect(error.cause).toBeInstanceOf(SyntaxError);
    expect(error.response.headers.get('X-Trace-ID')).toBe('trace-fixture'); expect(await error.response.text()).toBe('{');
    expect(requestAuth).not.toHaveBeenCalled(); expect(transport).toHaveBeenCalledOnce(); expect(session.getReadySnapshot()).toBe(snapshot);
  });
  it('unauthenticated_browser_requests_cannot_forward_identity_proof_headers', async () => {
    windowStub(); const session = new PowerXHostSession(pluginId); const transport = vi.fn().mockResolvedValue({});
    const client = createPowerXApiFetchClient({ pluginId, insidePowerX: true, hostSession: session,
      baseURL: `/_p/${pluginId}/api/v1`, fetcher: { create: () => transport } }).client;
    await client.get('/health', { skipAuth: true, headers: { Authorization: 'Bearer obsolete',
      'X-PowerX-Identity-Context': 'browser-spoof', 'X-PowerX-CTX': 'browser-spoof', 'X-PowerX-CTX-SIG': 'browser-spoof' } });
    const headers = transport.mock.calls[0][1].headers;
    for (const header of ['Authorization', 'X-PowerX-Identity-Context', 'X-PowerX-CTX', 'X-PowerX-CTX-SIG']) expect(headers.get(header)).toBeNull();
  });
});
