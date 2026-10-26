import { describe, expect, it, vi } from 'vitest';
import { settingsTestAttempt, settingsTestBlocked } from '../../app/utils/aiSettingsTesting';
import fixture from '../../../../../docs/contracts/fixtures/ai-media-v2.json';

const ready = () => ({ ready: true, modality: 'image', delegated: true, source: 'powerx', modelKey: 'fixture.model', provider: 'fixture.provider', endpoint: '', prompt: 'fixture.prompt', image: '', imageType: 'image_url', catalogFailed: false, available: true, operation: fixture.catalog_result.items[0]!.operations.image_generation, localVision: false });
describe('settings capability tests', () => {
  it('disables unavailable runtime, models, operations and undeclared modalities', () => {
    expect(settingsTestBlocked(ready())).toBe('');
    for (const input of [{ ready: false }, { modelKey: '' }, { available: false }, { catalogFailed: true }, { operation: undefined }, { modality: 'embedding' }, { modality: 'audio_asr' }, { modality: 'model3d' }, { modality: 'rerank' }, { source: 'local' }]) {
      expect(settingsTestBlocked({ ...ready(), ...input })).not.toBe('');
    }
    expect(settingsTestBlocked({ ...ready(), delegated: false })).toBe('localUnsupported');
    expect(settingsTestBlocked({ ...ready(), delegated: false, modality: 'llm' })).toBe('powerxRuntimeRequired');
  });
  it('requires a supported image input for vision and a configured local vision model', () => {
    const input = { ...ready(), modality: 'vlm', operation: { ...ready().operation, input_types: ['text', 'image_url'] } };
    expect(settingsTestBlocked(input)).toBe('imageRequired');
    expect(settingsTestBlocked({ ...input, image: 'https://fixture.invalid/image' })).toBe('');
    expect(settingsTestBlocked({ ...input, image: 'fixture.base64', imageType: 'image_base64' })).toBe('unsupportedInput');
    expect(settingsTestBlocked({ ...input, delegated: false, image: 'fixture.base64', imageType: 'image_base64' })).toBe('localProfileRequired');
    expect(settingsTestBlocked({ ...input, delegated: false, image: 'fixture.base64', imageType: 'image_base64', localVision: true })).toBe('');
  });
  it('retries the exact generation request with its original key until explicitly cleared', () => {
    const key = vi.fn().mockReturnValueOnce('fixture.attempt.1').mockReturnValueOnce('fixture.attempt.2');
    const body = { modality: 'image', inputs: [{ type: 'text', content: 'fixture.prompt' }], parameters: { seed: 0, watermark: false } };
    const first = settingsTestAttempt(undefined, body, key);
    const retry = settingsTestAttempt(first, structuredClone(body), key);
    expect(retry).toBe(first); expect(key).toHaveBeenCalledTimes(1);
    expect(retry.body.idempotency_key).toBe('fixture.attempt.1');
    expect(retry.body.parameters).toEqual({ seed: 0, watermark: false });
    expect(() => settingsTestAttempt(first, { ...body, modality: 'video' }, key)).toThrow('AI_TEST_ATTEMPT_CHANGED');
    expect(settingsTestAttempt(undefined, body, key).body.idempotency_key).toBe('fixture.attempt.2');
  });
});
