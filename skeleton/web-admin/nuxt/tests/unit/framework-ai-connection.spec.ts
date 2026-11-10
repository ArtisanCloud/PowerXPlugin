import { describe, expect, it, vi } from 'vitest';
import { createAIConnectionClient, validateAIConnectionTestResult } from '../../../../../framework/frontend/nuxt/framework-client';
import fixture from '../../../../../docs/contracts/fixtures/ai-connection-test.v1.json';

const input = { source: 'powerx', modality: 'llm', provider: 'ollama' } as const;
describe('AI connection probe', () => {
  it('sends only diagnostic fields without requiring a model, prompt or attempt key', async () => {
    const request = vi.fn(async () => ({ data: structuredClone(fixture.expected_result) }));
    const client = createAIConnectionClient({ request: request as any });
    const result = await client.testConnection(input);
    expect(request).toHaveBeenCalledExactlyOnceWith('admin/ai-settings/test-connection', { method: 'POST', body: input });
    expect(result.verification).toMatchObject({ generation_submitted: false, model_access_verified: false, api_key_verified: false, authentication: 'not_required' });
  });
  it('rejects generation fields and endpoint overrides before issuing a request', async () => {
    const request = vi.fn(); const client = createAIConnectionClient({ request });
    for (const extra of [{ inputs: [] }, { prompt: 'fixture' }, { params: {} }, { endpoint: 'https://fixture.invalid' }, { environment: 'production' }, { idempotency_key: 'fixture' }, { api_key: 'fixture' }]) {
      await expect(client.testConnection({ ...input, ...extra } as any)).rejects.toMatchObject({ reasonCode: 'AI_REQUEST_INVALID' });
    }
    expect(request).not.toHaveBeenCalled();
  });
  it('rejects incomplete verification and any claim of model execution', () => {
    for (const key of ['api_key_verified', 'generation_submitted', 'model_access_verified']) {
      const result: any = structuredClone(fixture.expected_result); delete result.verification[key];
      expect(() => validateAIConnectionTestResult(result, input)).toThrow('AI_CONNECTION_CONTRACT_INVALID');
    }
    for (const change of [{ generation_submitted: true }, { model_access_verified: true }, { provider: 'other' }, { authentication: 'api_key' }]) {
      const result: any = structuredClone(fixture.expected_result); Object.assign(result.verification, change);
      expect(() => validateAIConnectionTestResult(result, input)).toThrow('AI_CONNECTION_CONTRACT_INVALID');
    }
  });
  it('preserves transport diagnostics and does not retry or call another endpoint', async () => {
    const error = { statusCode: 403, data: { error_code: 'fixture.denied', trace_id: 'fixture.trace', request_id: 'fixture.request' } };
    const request = vi.fn(async () => { throw error; });
    await expect(createAIConnectionClient({ request }).testConnection(input)).rejects.toBe(error);
    expect(request).toHaveBeenCalledTimes(1);
  });
});
