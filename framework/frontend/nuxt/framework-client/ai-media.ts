import { PluginSSEHTTPError, type PluginSSEStreamOptions } from './sse';
import { PowerXHostSessionError } from './host-session';

export type MediaCapability = 'vision_understanding' | 'image_generation' | 'video_generation' | 'speech_synthesis';
export type MediaModality = 'image' | 'video' | 'tts';
export interface MediaParameterSchema {
  type: 'object' | 'string' | 'boolean' | 'number' | 'integer';
  properties?: Record<string, MediaParameterSchema>;
  additionalProperties?: boolean;
  enum?: unknown[];
  minimum?: number;
  maximum?: number;
  default?: unknown;
}
export interface MediaModelOperation {
  driver: string;
  input_types: string[];
  max_images: number;
  max_videos: number;
  parameters: Record<string, MediaParameterSchema>;
  parameter_schema: MediaParameterSchema;
  defaults: Record<string, unknown>;
  available: boolean;
  reason_code?: string;
  async: boolean;
}
export interface MediaModel {
  model_key: string;
  provider: string;
  model: string;
  app?: string;
  label: string;
  capabilities: MediaCapability[];
  configured: boolean;
  available: boolean;
  reason_code?: string;
  operations: Partial<Record<MediaCapability, MediaModelOperation>>;
}
export interface MediaModelCatalog { contract_version: 2; items: MediaModel[] }
export interface MediaProviderCatalog { contract_version: 2; providers: Array<{ provider: string; label: string }> }
export interface MediaCatalogFilter { provider?: string; app?: string; capability?: MediaCapability }
export interface MediaInvokeInput {
  model_key: string;
  inputs: Array<{ type: 'text' | 'image_url' | 'image_base64' | 'video_url'; content?: string; url?: string; role?: string }>;
  params?: Record<string, unknown>;
}
export interface MediaGenerationInput extends MediaInvokeInput { idempotency_key: string }
export interface MediaAnalysis {
  contract_version: 2; type: 'analysis'; text: string; provider: string; model: string;
  usage?: Record<string, unknown>; trace_id: string;
}
export interface MediaAssetResult {
  media_asset_uuid: string; type: 'image' | 'audio' | 'video'; mime_type: string;
  size_bytes: number; checksum: string; checksum_algorithm: 'sha256';
}
export interface MediaTask {
  contract_version: 2;
  task_uuid: string;
  status: 'queued' | 'running' | 'succeeded' | 'failed' | 'cancelled';
  phase: 'queued' | 'submitting' | 'waiting_provider' | 'materializing' | 'completed';
  model_key: string;
  capability: Exclude<MediaCapability, 'vision_understanding'>;
  revision: number;
  trace_id: string;
  error_code?: string;
  provider_error_code?: string;
  upstream_status?: number;
  provider_trace_id?: string;
  cancel_status: 'none' | 'requested' | 'confirmed' | 'unsupported';
  assets: MediaAssetResult[];
  created_at: string;
  updated_at: string;
  completed_at?: string;
}
export class AIMediaStreamError extends Error {
  constructor(public readonly reasonCode: string, public readonly status = 0) { super(reasonCode); }
}
export interface AIMediaClientOptions {
  request<T>(path: string, options: { method: 'GET' | 'POST'; query?: MediaCatalogFilter; body?: MediaInvokeInput | MediaGenerationInput }): Promise<{ data: T }>;
  stream(options: PluginSSEStreamOptions): Promise<void>;
  basePath?: string;
}
export interface MediaTaskWatchOptions {
  signal: AbortSignal;
  onTask(task: MediaTask): void;
  lastEventID?: string;
  maxReconnects?: number;
  reconnectDelayMs?: number;
  onState?(state: 'connecting' | 'connected' | 'reconnecting' | 'closed' | 'error'): void;
}
const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/;
function validUUID(id: string) { return uuidPattern.test(id) && id !== '00000000-0000-0000-0000-000000000000'; }
export function isMediaTaskTerminal(task: MediaTask) { return ['succeeded', 'failed', 'cancelled'].includes(task.status); }
export function validateMediaTask(task: MediaTask, taskUUID?: string): MediaTask {
  if (!task || task.contract_version !== 2 || !validUUID(task.task_uuid) || (taskUUID && task.task_uuid !== taskUUID) || !Number.isSafeInteger(task.revision) || task.revision < 1 || !task.model_key || !task.trace_id || !Array.isArray(task.assets) || !['queued','running','succeeded','failed','cancelled'].includes(task.status) || !['queued','submitting','waiting_provider','materializing','completed'].includes(task.phase) || !['image_generation','video_generation','speech_synthesis'].includes(task.capability) || !['none','requested','confirmed','unsupported'].includes(task.cancel_status) || !Number.isFinite(Date.parse(task.created_at)) || !Number.isFinite(Date.parse(task.updated_at))) throw new AIMediaStreamError('AI_TASK_RESPONSE_INVALID');
  for (const asset of task.assets) {
    if (!validUUID(asset.media_asset_uuid) || !['image','audio','video'].includes(asset.type) || !asset.mime_type || asset.size_bytes < 1 || asset.checksum_algorithm !== 'sha256' || !/^[a-f0-9]{64}$/.test(asset.checksum)) throw new AIMediaStreamError('AI_ASSET_RESPONSE_INVALID');
  }
  if (task.status === 'succeeded' && !task.assets.length) throw new AIMediaStreamError('AI_ASSET_RESPONSE_INVALID');
  return task;
}
// Defaults remain keyed by the declared operation. No capability is inferred
// from a model name and false/zero defaults are preserved.
export function mediaOperationDefaults(model: MediaModel, capability: MediaCapability) {
  const operation = model.operations[capability];
  if (!operation) throw new AIMediaStreamError('AI_MODEL_CAPABILITY_UNSUPPORTED');
  return { ...operation.defaults };
}
function delay(ms: number, signal: AbortSignal) {
  return new Promise<void>((resolve, reject) => {
    if (signal.aborted) { reject(signal.reason); return; }
    const abort = () => { clearTimeout(timer); reject(signal.reason); };
    const timer = setTimeout(() => { signal.removeEventListener('abort', abort); resolve(); }, ms);
    signal.addEventListener('abort', abort, { once: true });
  });
}

export function createAIMediaClient(options: AIMediaClientOptions) {
  const base = options.basePath || 'admin/ai-settings/media';
  const request = async <T>(path: string, input: Parameters<AIMediaClientOptions['request']>[1]): Promise<T> => {
    const response = await options.request<T>(`${base}/${path}`, input);
    if (!response || response.data === undefined || response.data === null) throw new AIMediaStreamError('AI_RESPONSE_INVALID');
    return response.data;
  };
  return {
    async listModels(filter: MediaCatalogFilter = {}) {
      const result = await request<MediaModelCatalog>('models', { method: 'GET', query: filter });
      if (result.contract_version !== 2 || !Array.isArray(result.items)) throw new AIMediaStreamError('AI_CATALOG_RESPONSE_INVALID');
      return result;
    },
    async listProviders(capability?: MediaCapability) {
      const result = await request<MediaProviderCatalog>('providers', { method: 'GET', query: { capability } });
      if (result.contract_version !== 2 || !Array.isArray(result.providers)) throw new AIMediaStreamError('AI_CATALOG_RESPONSE_INVALID');
      return result;
    },
    async analyze(input: MediaInvokeInput) {
      const result = await request<MediaAnalysis>('vlm/invoke', { method: 'POST', body: input });
      if (result.contract_version !== 2 || result.type !== 'analysis' || typeof result.text !== 'string' || !result.trace_id || !result.provider || !result.model) throw new AIMediaStreamError('AI_ANALYSIS_RESPONSE_INVALID');
      return result;
    },
    async generate(modality: MediaModality, input: MediaGenerationInput) {
      if (!/^[A-Za-z0-9._:-]{1,128}$/.test(input.idempotency_key || '')) throw new AIMediaStreamError('AI_IDEMPOTENCY_KEY_REQUIRED');
      return validateMediaTask(await request<MediaTask>(`${modality}/invoke`, { method: 'POST', body: input }));
    },
    async getTask(taskUUID: string) {
      if (!validUUID(taskUUID)) throw new AIMediaStreamError('AI_TASK_UUID_INVALID');
      return validateMediaTask(await request<MediaTask>(`tasks/${taskUUID}`, { method: 'GET' }), taskUUID);
    },
    async cancelTask(taskUUID: string) {
      if (!validUUID(taskUUID)) throw new AIMediaStreamError('AI_TASK_UUID_INVALID');
      return validateMediaTask(await request<MediaTask>(`tasks/${taskUUID}/cancel`, { method: 'POST' }), taskUUID);
    },
    async watchTask(taskUUID: string, watch: MediaTaskWatchOptions) {
      if (!validUUID(taskUUID)) throw new AIMediaStreamError('AI_TASK_UUID_INVALID');
      let cursor = watch.lastEventID || '';
      let revision = 0;
      if (cursor) {
        const parts = cursor.split(':');
        revision = Number(parts[1]);
        if (parts.length !== 2 || parts[0] !== taskUUID || !/^[1-9][0-9]*$/.test(parts[1] || '') || !Number.isSafeInteger(revision)) throw new AIMediaStreamError('AI_EVENT_CURSOR_INVALID');
      }
      let failures = 0;
      let failed = false;
      try {
        while (!watch.signal.aborted) {
          const controller = new AbortController();
          const abort = () => controller.abort(watch.signal.reason);
          watch.signal.addEventListener('abort', abort, { once: true });
          const connectedAt = Date.now();
          let terminal = false, received = false, reconnect = false;
          let permanent: unknown;
          let lastFailure: unknown;
          watch.onState?.(failures ? 'reconnecting' : 'connecting');
          try {
            await options.stream({ path: `${base}/tasks/${taskUUID}/events`, signal: controller.signal,
              headers: cursor ? { 'Last-Event-ID': cursor } : {},
              onEvent(event) {
                if (watch.signal.aborted) return;
                try {
                  if (event.event === 'status') {
                    const task = validateMediaTask(event.payload as MediaTask, taskUUID);
                    if (event.id !== `${taskUUID}:${task.revision}`) throw new AIMediaStreamError('AI_EVENT_CURSOR_INVALID');
                    terminal = isMediaTaskTerminal(task);
                    watch.onState?.('connected');
                    if (task.revision > revision) { received = true; revision = task.revision; cursor = event.id; watch.onTask(task); }
                    if (terminal) controller.abort();
                  } else if (event.event === 'error') {
                    const diagnostic = event.payload as { reason_code?: string; http_status?: number; trace_id?: string; request_id?: string };
                    if (!diagnostic?.reason_code) throw new AIMediaStreamError('AI_EVENT_RESPONSE_INVALID');
                    throw new PluginSSEHTTPError(diagnostic.http_status || 403, JSON.stringify(diagnostic), diagnostic.trace_id || '', diagnostic.request_id || '');
                  } else if (event.event === 'reconnect') {
                    const diagnostic = event.payload as { http_status?: number; trace_id?: string; request_id?: string };
                    if (diagnostic?.http_status) lastFailure = new PluginSSEHTTPError(diagnostic.http_status, JSON.stringify(diagnostic), diagnostic.trace_id || '', diagnostic.request_id || '');
                    reconnect = true; controller.abort();
                  }
                  else throw new AIMediaStreamError('AI_EVENT_RESPONSE_INVALID');
                } catch (error) { permanent = error; controller.abort(); throw error; }
              },
            });
          } catch (error) {
            if (!reconnect) lastFailure = error;
            if (permanent) throw permanent;
            if (watch.signal.aborted || terminal) return;
            if (!reconnect && (error instanceof PowerXHostSessionError || error instanceof AIMediaStreamError || (error instanceof PluginSSEHTTPError && ![429,502,503,504].includes(error.status)))) throw error;
          } finally { watch.signal.removeEventListener('abort', abort); controller.abort(); }
          if (permanent) throw permanent;
          if (terminal || watch.signal.aborted) return;
          if (received || Date.now() - connectedAt >= 30000) failures = 0;
          if (++failures > (watch.maxReconnects ?? 5)) {
            if (lastFailure instanceof PluginSSEHTTPError) throw lastFailure;
            throw new AIMediaStreamError('AI_EVENT_RECONNECT_EXHAUSTED');
          }
          watch.onState?.('reconnecting');
          await delay(Math.min((watch.reconnectDelayMs ?? 250) * 2 ** (failures - 1), 5000), watch.signal);
        }
      } catch (error) { if (!watch.signal.aborted) { failed = true; watch.onState?.('error'); throw error; } }
      finally { if (!failed) watch.onState?.('closed'); }
    },
  };
}
