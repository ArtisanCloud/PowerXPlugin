import { describe, expect, it, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { createLocalModelCatalogClient, validateLocalOnlineCatalog } from '../../../../../framework/frontend/nuxt/framework-client/ai-model-catalog';
const fixture = JSON.parse(readFileSync(new URL('../../../../../docs/contracts/fixtures/ai-local-model-catalog.v1.json', import.meta.url), 'utf8'));

describe('local online model catalog client', () => {
  it('keeps native identifiers and uses the fixed plugin-local route without generation inputs', async () => {
    const request = vi.fn().mockResolvedValue({ data: fixture.result });
    const client = createLocalModelCatalogClient({ request });
    const result = await client.sync(fixture.input);
    expect(result.models[0]!.id).toBe('private-chat:14b');
    expect(request).toHaveBeenCalledExactlyOnceWith('admin/ai-settings/catalog/local-online', { method: 'POST', body: { ...fixture.input, action: 'refresh' } });
  });
  it('retains the explicit assignment revision and does not retry conflicts', async () => {
    const error = { status: 409, data: { reason_code: 'AI_MODEL_SYNC_STALE', trace_id: 'fixture.trace', request_id: 'fixture.request' } };
    const request = vi.fn().mockRejectedValue(error);
    const client = createLocalModelCatalogClient({ request });
    await expect(client.sync(fixture.input, { revision: fixture.result.revision, models: ['private-chat:14b'] })).rejects.toBe(error);
    expect(request).toHaveBeenCalledOnce();
    expect(request.mock.calls[0]![1].body).toEqual({ ...fixture.input, action: 'assign', revision: fixture.result.revision, models: ['private-chat:14b'] });
  });
  it('rejects caller identity, credentials, Core source and free-form transport fields before sending a request', async () => {
    const request = vi.fn();
    const client = createLocalModelCatalogClient({ request });
    for (const key of ['tenant_uuid', 'api_key', 'source', 'headers', 'prompt', 'method']) await expect(client.sync({ ...fixture.input, [key]: 'fixture.forbidden' })).rejects.toMatchObject({ reasonCode: 'AI_REQUEST_INVALID' });
    await expect(client.sync(fixture.input, { revision: fixture.result.revision, models: ['duplicate', 'duplicate'] })).rejects.toMatchObject({ reasonCode: 'AI_REQUEST_INVALID' });
    expect(request).not.toHaveBeenCalled();
  });
  it('rejects incomplete, wrong-scope or malformed results without manufacturing revisions or models', () => {
    for (const patch of [{ revision: '' }, { environment: 'production' }, { provider: 'other' }, { models: null }, { source: 'https://user:secret@fixture.invalid/models' }, { fingerprint: 'fixture.private' }, { models: [{ ...fixture.result.models[0], id: 'trimmed ' }] }]) expect(() => validateLocalOnlineCatalog({ ...fixture.result, ...patch }, fixture.input)).toThrow();
  });
});
