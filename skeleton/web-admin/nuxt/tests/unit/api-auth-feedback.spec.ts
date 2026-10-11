import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({ transport: vi.fn(), toast: vi.fn(), clearAuth: vi.fn(), failClosed: vi.fn(), log: vi.fn(), inside: true }));
vi.mock('#imports', () => ({
  useRuntimeConfig: () => ({ public: { insidePowerX: mocks.inside, powerxPluginId: 'com.powerx.plugins.base' } }),
  useToast: () => ({ add: mocks.toast }),
}));
vi.mock('~/composables/api/_base', () => ({
  resolveApiBase: () => mocks.inside ? '/_p/com.powerx.plugins.base/api/v1' : '/api/v1',
  getAuthToken: () => 'standalone-token', getTenantUuid: () => undefined,
}));
vi.mock('~/utils/tenant-context', () => ({ resolveTenantUUIDForRequest: () => undefined }));
vi.mock('~/composables/useAuth', () => ({
  useAuth: () => ({ getToken: () => 'standalone-token', clearAuth: mocks.clearAuth, failClosed: mocks.failClosed }),
}));

beforeEach(() => {
  vi.resetModules(); vi.clearAllMocks(); mocks.inside = true;
  vi.stubGlobal('window', { location: { origin: 'https://host.test', pathname: '/admin/records', search: '' } });
  vi.stubGlobal('process', { client: true, env: {} });
  vi.stubGlobal('$fetch', { create: () => mocks.transport });
  vi.stubGlobal('useNuxtApp', () => ({ $i18n: { t: (key: string) => key } }));
  vi.spyOn(console, 'error').mockImplementation(mocks.log);
});
afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals(); });

const cases = [
  { status: 401, code: 'IAM_CONTEXT_PROOF_REQUIRED', key: 'hostSession.identityProofRejected' },
  { status: 401, code: 'IAM_CONTEXT_PROOF_EXPIRED', key: 'hostSession.identityProofRejected' },
  { status: 401, code: 'IAM_CONTEXT_WS_HANDLE_EXPIRED', key: 'hostSession.identityProofRejected' },
  { status: 401, code: 'AUTH_STS_EXPIRED', key: 'hostSession.serviceCredentialRejected' },
  { status: 401, code: 'IAM_CONTEXT_SERVICE_CREDENTIAL_INVALID', key: 'hostSession.serviceCredentialRejected' },
  { status: 401, code: 'IAM_CONTEXT_SERVICE_CREDENTIAL_EXPIRED', key: 'hostSession.serviceCredentialRejected' },
  { status: 401, code: 'IAM_UNAUTHORIZED', key: 'hostSession.requestFailed' },
  { status: 503, code: 'IAM_CONTEXT_UNAVAILABLE', key: 'hostSession.upstreamUnavailable' },
  { status: 424, code: 'IAM_UPSTREAM_DEPENDENCY', key: 'hostSession.upstreamUnavailable' },
];

describe.each([true, false])('api_auth_feedback_embedded_%s', inside => {
  it.each(cases)('status=$status code=$code', async ({ status, code, key }) => {
    mocks.inside = inside;
    const framework = await import('../../../../../framework/frontend/nuxt/framework-client');
    const session = framework.getPowerXHostSession('com.powerx.plugins.base');
    const requestAuth = vi.fn(); session.setRequester(requestAuth);
    if (inside) await session.apply({
      source: 'powerx', type: 'auth-token', protocolVersion: 2, pluginId: 'com.powerx.plugins.base', instanceId: 'frame-1',
      sequence: 1, sessionId: 'session-1', revision: 1, accessToken: 'user-token', tokenType: 'Bearer',
      expiresAt: Date.now() + 60_000, expiresIn: 60, scope: 'powerx', tenantUuid: null,
    }, { readContext: async () => ({}), validateContext: () => ({ tenantUuid: null, memberUuid: null }), commitContext: vi.fn(), clearContext: vi.fn() });
    const snapshot = session.getReadySnapshot();
    const error = { response: { status, _data: { error: { code }, trace_id: 'trace-fixture' } } };
    mocks.transport.mockRejectedValue(error);
    const { useApiClient } = await import('../../app/composables/api/_client');
    await expect(useApiClient().get('/records')).rejects.toBe(error);
    expect(mocks.transport).toHaveBeenCalledOnce(); expect(requestAuth).not.toHaveBeenCalled();
    expect(session.getReadySnapshot()).toBe(snapshot);
    expect(mocks.clearAuth).not.toHaveBeenCalled(); expect(mocks.failClosed).not.toHaveBeenCalled();
    expect(mocks.toast).toHaveBeenCalledWith(expect.objectContaining({ description: key }));
    expect(JSON.stringify(mocks.log.mock.calls)).not.toContain('user-token');
  });
});
