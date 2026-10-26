import type { PluginSSEStreamOptions } from './sse';

export type ImageDerivativeStatus = 'not_requested' | 'waiting_upload' | 'waiting_source' | 'queued' | 'processing' | 'ready' | 'failed' | 'scheduling_failed' | 'unsupported' | 'unavailable';
export interface ImageDerivativeItem {
  variant: string; status: ImageDerivativeStatus; variant_uuid?: string;
  error_code?: string; retryable?: boolean;
}
export interface ImageDerivatives {
  asset_uuid: string; upload_state: string; status: ImageDerivativeStatus;
  recipe: string; job_uuid?: string; attempts: number; error_code?: string;
  retryable: boolean; items: ImageDerivativeItem[];
}
export class ImageDerivativesError extends Error {
  constructor(public readonly reasonCode: string) { super(reasonCode); }
}
const validUUID = (id: string) => /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/.test(id) && id !== '00000000-0000-0000-0000-000000000000';
const statuses: ImageDerivativeStatus[] = ['not_requested','waiting_upload','waiting_source','queued','processing','ready','failed','scheduling_failed','unsupported','unavailable'];

export function validateImageDerivatives(value: ImageDerivatives, assetUUID?: string): ImageDerivatives {
  const invalid = () => { throw new ImageDerivativesError('MEDIA_DERIVATIVES_RESPONSE_INVALID'); };
  if (!value || !validUUID(value.asset_uuid) || (assetUUID && value.asset_uuid !== assetUUID) || typeof value.upload_state !== 'string' || !value.upload_state || !['standard-image-v1','standard-image-v2'].includes(value.recipe) || !statuses.includes(value.status) || !Number.isSafeInteger(value.attempts) || value.attempts < 0 || typeof value.retryable !== 'boolean' || !Array.isArray(value.items)) invalid();
  if (value.recipe === "standard-image-v2" && value.items.length < 3) invalid();
  const names = new Set<string>();
  for (const item of value.items) {
    if (!item || typeof item.variant !== 'string' || !item.variant || names.has(item.variant) || !statuses.includes(item.status) || (item.status === 'ready' ? !validUUID(item.variant_uuid || '') : item.variant_uuid !== undefined) || (item.retryable !== undefined && typeof item.retryable !== 'boolean') || (item.error_code !== undefined && typeof item.error_code !== 'string')) invalid();
    names.add(item.variant);
  }
  if (value.status === 'ready') {
    const expected = value.recipe === 'standard-image-v2' ? ['preview','thumbnail','thumbnail_64'] : ['preview','thumbnail'];
    if (expected.some(name => !value.items.some(item => item.variant === name && item.status === 'ready'))) invalid();
  }
  return value;
}

/** Selects one spec independently of the aggregate status; never substitutes
 * a preview, origin or an item at a fixed array position. */
export function findImageDerivative(value: ImageDerivatives, variant: string): ImageDerivativeItem | undefined {
  return validateImageDerivatives(value).items.find(item => item.variant === variant);
}

export interface ImageDerivativesClientOptions {
  // Paths belong to an authenticated plugin bridge supplied by the consumer.
  // Service credentials and Core invocation bindings remain on the backend.
  paths(assetUUID: string): { status: string; ensure: string; events: string };
  request<T>(path: string, options: { method: 'GET' | 'POST'; body?: { retry_failed: boolean } }): Promise<{ data: T }>;
  stream(options: PluginSSEStreamOptions): Promise<void>;
}
export interface ImageDerivativeWatchOptions {
  signal: AbortSignal;
  onSnapshot(snapshot: ImageDerivatives): void;
  onInvalidate?(reason: string): void;
  maxReconnects?: number;
  reconnectDelayMs?: number;
}
function wait(ms: number, signal: AbortSignal) {
  return new Promise<void>((resolve, reject) => {
    if (signal.aborted) { reject(signal.reason); return; }
    const abort = () => { clearTimeout(timer); reject(signal.reason); };
    const timer = setTimeout(() => { signal.removeEventListener('abort', abort); resolve(); }, ms);
    signal.addEventListener('abort', abort, { once: true });
  });
}
export function createImageDerivativesClient(options: ImageDerivativesClientOptions) {
  const paths = (assetUUID: string) => {
    if (!validUUID(assetUUID)) throw new ImageDerivativesError('MEDIA_ASSET_UUID_INVALID');
    const result = options.paths(assetUUID);
    if (!result || ![result.status,result.ensure,result.events].every(path => typeof path === "string")) throw new ImageDerivativesError("MEDIA_BRIDGE_PATH_INVALID");
    for (const path of Object.values(result)) if (!path || /[:%?#\\]/.test(path) || path.includes('..') || path.includes('//')) throw new ImageDerivativesError('MEDIA_BRIDGE_PATH_INVALID');
    return result;
  };
  return {
    async get(assetUUID: string) {
      const response = await options.request<ImageDerivatives>(paths(assetUUID).status, { method: 'GET' });
      return validateImageDerivatives(response?.data, assetUUID);
    },
    async ensure(assetUUID: string, retryFailed = false) {
      const response = await options.request<ImageDerivatives>(paths(assetUUID).ensure, { method: 'POST', body: { retry_failed: retryFailed } });
      return validateImageDerivatives(response?.data, assetUUID);
    },
    async watch(assetUUID: string, watch: ImageDerivativeWatchOptions) {
      const path = paths(assetUUID).events;
      const max = watch.maxReconnects ?? 3;
      if (!Number.isSafeInteger(max) || max < 0 || max > 100) throw new ImageDerivativesError("MEDIA_DERIVATIVES_REQUEST_INVALID");
      let failures = 0;
      while (!watch.signal.aborted) {
        const controller = new AbortController();
        const abort = () => controller.abort(watch.signal.reason);
        watch.signal.addEventListener('abort', abort, { once: true });
        let permanent: unknown; let upstream: unknown;
        const started = Date.now();
        try {
          await options.stream({ path, signal: controller.signal, onEvent(event) {
            if (controller.signal.aborted || watch.signal.aborted) return;
            try {
              if (event.event === 'status') watch.onSnapshot(validateImageDerivatives(event.payload as ImageDerivatives, assetUUID));
              else if (event.event === 'reconnect') {
                if (typeof (event.payload as any)?.reason !== 'string' || !(event.payload as any).reason) throw new ImageDerivativesError('MEDIA_DERIVATIVES_EVENT_INVALID');
                controller.abort();
              } else if (event.event === 'invalidate') {
                const payload = event.payload as { asset_uuid: string; reason: string };
                if (payload?.asset_uuid !== assetUUID || !payload.reason) throw new ImageDerivativesError('MEDIA_DERIVATIVES_EVENT_INVALID');
                permanent = new ImageDerivativesError('MEDIA_DERIVATIVES_INVALIDATED');
                watch.onInvalidate?.(payload.reason); controller.abort();
              } else if (event.event !== 'message' || event.data) throw new ImageDerivativesError('MEDIA_DERIVATIVES_EVENT_INVALID');
            } catch (error) { permanent = error; controller.abort(); }
          } });
        } catch (error) { upstream = error; }
        finally { controller.abort(); watch.signal.removeEventListener('abort', abort); }
        if (watch.signal.aborted) return;
        if (permanent) throw permanent;
        const status = Number((upstream as any)?.status || (upstream as any)?.statusCode || (upstream as any)?.response?.status || 0);
        if (upstream && ((status && ![429,502,503,504].includes(status)) || upstream instanceof ImageDerivativesError || (upstream as Error).name === 'PowerXHostSessionError')) throw upstream;
        if (Date.now() - started >= 30_000) failures = 0;
        if (failures++ >= max) throw upstream || new ImageDerivativesError('MEDIA_DERIVATIVES_RECONNECT_EXHAUSTED');
        try { await wait(watch.reconnectDelayMs ?? 250, watch.signal); } catch (error) { if (watch.signal.aborted) return; throw error; }
      }
    },
  };
}
