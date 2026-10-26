import { describe, expect, it, vi } from 'vitest';
import { createImageDerivativesClient, findImageDerivative, validateImageDerivatives, type ImageDerivatives } from '../../../../../framework/frontend/nuxt/framework-client/image-derivatives';
import { validateBusinessCardPresentation, upsertNotificationByRevision, type BusinessCardPresentationV2, type NotificationPayload } from '../../../../../framework/frontend/nuxt/framework-client/notifications';
import fixture from '../../../../../docs/contracts/fixtures/media-image-derivatives.v2.json';
import card from '../../../../../docs/contracts/fixtures/business-notification-card.v2.json';
import v1 from '../../../../../docs/contracts/fixtures/business-notification-card.v1.json';

const state = (name: 'ready' | 'partial_processing' | 'partial_failed' = 'ready') => structuredClone(fixture[name]) as ImageDerivatives;
const paths = (id: string) => ({ status: `admin/consumer-media/${id}/derivatives`, ensure: `admin/consumer-media/${id}/derivatives/ensure`, events: `admin/consumer-media/${id}/derivatives/events` });
const status = (value: ImageDerivatives) => ({ event: 'status', id: '', data: JSON.stringify(value), payload: value, raw: '' });

describe('standard-image-v2 and business_notification_card.v2', () => {
  it('reads named specs independently of aggregate failure and array order', () => {
    const value = state('partial_failed'); value.items.reverse();
    value.items.push({ variant: 'custom_extra', status: 'processing' });
    expect(findImageDerivative(value, 'preview')?.status).toBe('ready');
    expect(findImageDerivative(value, 'thumbnail_64')).toMatchObject({ status: 'failed', error_code: 'MEDIA_THUMBNAIL_64_BYTE_LIMIT', retryable: true });
    value.items = value.items.filter(item => item.variant !== 'thumbnail_64');
    expect(findImageDerivative(value, 'thumbnail_64')).toBeUndefined();
  });

  it('rejects missing ready UUIDs, pending UUIDs and duplicate variants', () => {
    for (const mutate of [(v: ImageDerivatives) => { delete v.items[2]!.variant_uuid; }, (v: ImageDerivatives) => { v.status = 'processing'; v.items[2]!.status = 'queued'; }, (v: ImageDerivatives) => v.items.push(v.items[0]!)]) {
      const value = state(); mutate(value);
      expect(() => validateImageDerivatives(value)).toThrow('MEDIA_DERIVATIVES_RESPONSE_INVALID');
    }
  });

  it('uses only declared plugin bridge paths and preserves explicit retry input', async () => {
    const request = vi.fn(async () => ({ data: state('partial_processing') }));
    const client = createImageDerivativesClient({ paths, request: request as any, stream: vi.fn() });
    await client.get(state().asset_uuid); await client.ensure(state().asset_uuid, true);
    expect(request.mock.calls).toEqual([[paths(state().asset_uuid).status, { method: 'GET' }], [paths(state().asset_uuid).ensure, { method: 'POST', body: { retry_failed: true } }]]);
  });

  it('renews SSE snapshots without a cursor or idle GETs and stops on invalidation', async () => {
    const request = vi.fn(); let late: (() => void) | undefined;
    const stream = vi.fn(async (options: any) => {
      options.onEvent(status(state('partial_processing')));
      if (stream.mock.calls.length === 1) {
        late = () => options.onEvent(status(state()));
        options.onEvent({ event: 'reconnect', payload: { reason: 'reauthorize' } });
      } else {
        late?.();
        options.onEvent({ event: 'invalidate', payload: { asset_uuid: state().asset_uuid, reason: 'access_or_state_changed' } });
      }
    });
    const client = createImageDerivativesClient({ paths, request, stream });
    const onSnapshot = vi.fn(); const onInvalidate = vi.fn();
    await expect(client.watch(state().asset_uuid, { signal: new AbortController().signal, onSnapshot, onInvalidate, reconnectDelayMs: 0 })).rejects.toMatchObject({ reasonCode: 'MEDIA_DERIVATIVES_INVALIDATED' });
    expect(onSnapshot).toHaveBeenCalledTimes(2);
    expect(onInvalidate).toHaveBeenCalledWith('access_or_state_changed');
    expect(stream).toHaveBeenCalledTimes(2); expect(request).not.toHaveBeenCalled();
    expect(stream.mock.calls[0]![0].path).toBe(stream.mock.calls[1]![0].path);
    expect(stream.mock.calls[1]![0].headers).toBeUndefined();
  });

  it('cancels the stream and ignores a late response', async () => {
    const controller = new AbortController(); const callback = vi.fn(() => controller.abort());
    const stream = vi.fn(async (options: any) => { options.onEvent(status(state())); options.onEvent(status(state('partial_failed'))); });
    const client = createImageDerivativesClient({ paths, request: vi.fn(), stream });
    await client.watch(state().asset_uuid, { signal: controller.signal, onSnapshot: callback });
    expect(callback).toHaveBeenCalledTimes(1); expect(stream).toHaveBeenCalledTimes(1);
  });

  it('retains upstream failures, bounds reconnects and never retries forbidden access', async () => {
    for (const code of [403, 503]) {
      const error = Object.assign(new Error('fixture.upstream'), { status: code, traceID: 'fixture-trace' });
      const stream = vi.fn(async () => { throw error; });
      const client = createImageDerivativesClient({ paths, request: vi.fn(), stream });
      await expect(client.watch(state().asset_uuid, { signal: new AbortController().signal, onSnapshot: vi.fn(), maxReconnects: 2, reconnectDelayMs: 0 })).rejects.toBe(error);
      expect(stream).toHaveBeenCalledTimes(code === 403 ? 1 : 3);
    }
  });

  it('keeps v1 separate and accepts independent v2 thumbnail and preview states', () => {
    expect(validateBusinessCardPresentation(v1.presentation as any)).toEqual(v1.presentation);
    expect(validateBusinessCardPresentation(card.presentation as BusinessCardPresentationV2)).toEqual(card.presentation);
    expect(validateBusinessCardPresentation(card.presentation_pending as BusinessCardPresentationV2).image?.thumbnail.status).toBe('queued');
    for (const mutate of [(v: any) => { v.image.thumbnail.image_key = 'https://external.invalid/image'; }, (v: any) => { v.image.thumbnail.status = 'processing'; }, (v: any) => { v.image.preview.variant = 'origin'; }, (v: any) => { delete v.image.thumbnail.image_key; }]) {
      const value = structuredClone(card.presentation); mutate(value);
      expect(() => validateBusinessCardPresentation(value as BusinessCardPresentationV2)).toThrow('NOTIFICATION_PRESENTATION_INVALID');
    }
    const wrong = { ...structuredClone(v1.presentation), image: structuredClone(card.presentation.image) };
    expect(() => validateBusinessCardPresentation(wrong as any)).toThrow('NOTIFICATION_PRESENTATION_INVALID');
  });

  it('preserves v2 reference and notification read state while dropping old revisions', () => {
    const first = { ...card.notification, isRead: true } as NotificationPayload;
    const updated = { ...card.updated_event, isRead: true } as NotificationPayload;
    let items = upsertNotificationByRevision([first], updated);
    items = upsertNotificationByRevision(items, first); items = upsertNotificationByRevision(items, updated);
    expect(items).toEqual([updated]);
    expect(items[0]?.metadata?.business_card?.contract).toBe('business_notification_card.v2');
    expect(items[0]?.isRead).toBe(true);
    expect(card.upgrade_v1_reference.reason).toBe('presentation_changed');
  });
});
