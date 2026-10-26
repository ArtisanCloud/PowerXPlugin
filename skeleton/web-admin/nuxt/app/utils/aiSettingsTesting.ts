import type { MediaModelOperation } from '@artisan-cloud/plugin-framework-client';

export const settingsMediaCapabilities: Record<string, string> = {
  vlm: 'vision_understanding', image: 'image_generation', video: 'video_generation', audio_tts: 'speech_synthesis',
};
export function settingsTestBlocked(input: {
  ready: boolean; modality: string; delegated: boolean; source: string; modelKey: string;
  provider: string; endpoint: string; prompt: string; image: string; imageType: string;
  catalogFailed: boolean; available?: boolean; operation?: MediaModelOperation; localVision: boolean;
}): string {
  if (!input.ready) return 'runtimePending';
  if (input.modality !== 'llm' && !settingsMediaCapabilities[input.modality]) return 'unsupported';
  if (!input.delegated && !['llm', 'vlm'].includes(input.modality)) return 'localUnsupported';
  if (!input.modelKey || !input.prompt.trim()) return 'inputRequired';
  if (input.catalogFailed || input.available === false) return 'modelUnavailable';
  if (!input.delegated && input.source === 'powerx' && input.modality === 'llm') return 'powerxRuntimeRequired';
  if (input.delegated) {
    if (input.source !== 'powerx') return 'coreCatalogRequired';
    if (input.modality !== 'llm' && input.operation?.available !== true) return 'modelUnavailable';
    if (input.modality !== 'llm' && !input.operation?.input_types.includes('text')) return 'unsupportedInput';
  } else {
    if (input.modality !== 'llm' && input.modality !== 'vlm') return 'localUnsupported';
    if (input.modality === 'llm' && (input.provider !== 'ollama' || !input.endpoint.trim())) return 'localProfileRequired';
    if (input.modality === 'vlm' && !input.localVision) return 'localProfileRequired';
  }
  if (input.modality === 'vlm') {
    if (!input.image.trim()) return 'imageRequired';
    if (input.delegated ? !input.operation?.input_types.includes(input.imageType) : input.imageType !== 'image_base64') return 'unsupportedInput';
  }
  return '';
}

export interface SettingsAttempt<T> { signature: string; body: T & { idempotency_key: string } }
// An uncertain submission is retried verbatim. Only an explicit new attempt
// clears this record; transport failures never mint a replacement key.
export function settingsTestAttempt<T>(previous: SettingsAttempt<T> | undefined, body: T, key: () => string): SettingsAttempt<T> {
  const signature = JSON.stringify(body);
  if (previous) {
    if (previous.signature !== signature) throw new Error('AI_TEST_ATTEMPT_CHANGED');
    return previous;
  }
  return { signature, body: { ...body, idempotency_key: key() } };
}
