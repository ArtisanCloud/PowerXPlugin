import { afterEach, describe, expect, it, vi } from 'vitest';
import { createAIMediaClient, createPluginSSEClient, mediaOperationDefaults, PluginSSEHTTPError, readSSEStream, type MediaTask, type PluginSSEStreamEvent } from '../../../../../framework/frontend/nuxt/framework-client';
import fixture from '../../../../../docs/contracts/fixtures/ai-media-v2.json';

const task = (): MediaTask => structuredClone(fixture.task_result) as MediaTask;
const status = (value: MediaTask): PluginSSEStreamEvent => ({ id: `${value.task_uuid}:${value.revision}`, event: 'status', data: JSON.stringify(value), payload: value, raw: '' });
const running = () => ({ ...task(), status: 'running', phase: 'waiting_provider', revision: 2 } as MediaTask);
afterEach(() => vi.unstubAllGlobals());

describe('AI media v2', () => {
  it('uses typed catalog, declared defaults and stable attempt keys', async () => {
    const request = vi.fn(async (path: string, _options?: unknown) => ({ data: path.endsWith('/models') ? fixture.catalog_result : task() }));
    const client = createAIMediaClient({ request: request as any, stream: vi.fn() });
    const catalog = await client.listModels({ provider: 'qwen-cn', capability: 'image_generation' });
    expect(mediaOperationDefaults(catalog.items[0]!, 'image_generation')).toEqual(fixture.catalog_result.items[0]!.operations.image_generation.defaults);
    const zero = structuredClone(catalog.items[0]!); zero.operations.image_generation!.defaults = { seed: 0, watermark: false };
    expect(mediaOperationDefaults(zero, 'image_generation')).toEqual({ seed: 0, watermark: false });
    const input = fixture.invoke_operation;
    await client.generate('image', input as any); await client.generate('image', input as any);
    expect(request.mock.calls[1]).toEqual(request.mock.calls[2]);
    expect(request.mock.calls[1]![1]).toMatchObject({ method: 'POST', body: { idempotency_key: input.idempotency_key } });
    await expect(client.generate('image', { ...input, idempotency_key: '' } as any)).rejects.toMatchObject({ reasonCode: 'AI_IDEMPOTENCY_KEY_REQUIRED' });
    expect(request).toHaveBeenCalledTimes(3);
  });

  it('recovers the same task with a header cursor and drops duplicate revisions', async () => {
    const request = vi.fn(); const stream = vi.fn(async (options: any) => {
      options.onEvent(status(running()));
      if (stream.mock.calls.length === 1) options.onEvent({ event: 'reconnect', payload: { reason_code: 'AI_EVENT_REAUTHORIZE' } });
      else options.onEvent(status(task()));
    });
    const client = createAIMediaClient({ request, stream });
    const updates = vi.fn();
    await client.watchTask(task().task_uuid, { signal: new AbortController().signal, onTask: updates, reconnectDelayMs: 0 });
    expect(updates.mock.calls.map(([value]) => value.revision)).toEqual([2, 4]);
    expect(stream).toHaveBeenCalledTimes(2);
    expect(stream.mock.calls[1]![0].headers).toEqual({ 'Last-Event-ID': `${task().task_uuid}:2` });
    expect(stream.mock.calls[0]![0].path).toBe(stream.mock.calls[1]![0].path);
    expect(request).not.toHaveBeenCalled();
  });

  it('stops on revocation and rejects an event for another task', async () => {
    for (const event of [ { event: 'error', payload: { reason_code: 'AI_EVENT_ACCESS_REVOKED' } }, status({ ...task(), task_uuid: 'aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee' }) ]) {
      const stream = vi.fn(async (options: any) => options.onEvent(event));
      const client = createAIMediaClient({ request: vi.fn(), stream });
      await expect(client.watchTask(task().task_uuid, { signal: new AbortController().signal, onTask: vi.fn() })).rejects.toBeInstanceOf(Error);
      expect(stream).toHaveBeenCalledTimes(1);
    }
  });

  it('bounds reconnects and preserves the final HTTP diagnostics', async () => {
    const upstream = new PluginSSEHTTPError(503, JSON.stringify(fixture.parameter_error), 'trace', 'request');
    const stream = vi.fn(async () => { throw upstream; });
    const client = createAIMediaClient({ request: vi.fn(), stream });
    await expect(client.watchTask(task().task_uuid, { signal: new AbortController().signal, onTask: vi.fn(), maxReconnects: 1, reconnectDelayMs: 0 })).rejects.toBe(upstream);
    expect(stream).toHaveBeenCalledTimes(2);
    const denied = new PluginSSEHTTPError(403, JSON.stringify(fixture.parameter_error), 'trace', 'request');
    const access = createAIMediaClient({ request: vi.fn(), stream: vi.fn(async () => { throw denied; }) });
    await expect(access.watchTask(task().task_uuid, { signal: new AbortController().signal, onTask: vi.fn() })).rejects.toBe(denied);
  });

  it('aborts reads on leaving the page without reconnecting', async () => {
    const controller = new AbortController();
    const stream = vi.fn(async (options: any) => new Promise<void>((_, reject) => { options.signal.addEventListener('abort', () => reject(new Error('AUTH_REQUEST_ABORTED')), { once: true }); controller.abort(); }));
    const client = createAIMediaClient({ request: vi.fn(), stream });
    await client.watchTask(task().task_uuid, { signal: controller.signal, onTask: vi.fn() });
    expect(stream).toHaveBeenCalledTimes(1);
  });

  it('uses header authentication for fetch SSE and exposes the event ID', async () => {
    vi.stubGlobal('window', { location: { origin: 'https://plugin.test' } });
    const raw = `: keepalive\n\nid: ${task().task_uuid}:4\nevent: status\ndata: ${JSON.stringify(task())}\n\n`;
    const fetcher = vi.fn(async (_url: unknown, _options?: RequestInit) => new Response(raw, { headers: { 'Content-Type': 'text/event-stream' } }));
    vi.stubGlobal('fetch', fetcher);
    const sse = createPluginSSEClient({ pluginId: 'fixture', apiBaseURL: '/api/v1', token: 'token', tenantUuid: 'tenant' });
    const events: PluginSSEStreamEvent[] = [];
    await sse.stream({ path: 'admin/ai-settings/media/tasks/task/events', headers: { 'Last-Event-ID': `${task().task_uuid}:2` }, onEvent: event => events.push(event) });
    expect(fetcher.mock.calls[0]![0]).toBe('https://plugin.test/api/v1/admin/ai-settings/media/tasks/task/events');
    expect(fetcher.mock.calls[0]![1]!.headers).toMatchObject({ Authorization: 'Bearer token', tenant_uuid: 'tenant', 'Last-Event-ID': `${task().task_uuid}:2` });
    expect(events).toHaveLength(1); expect(events[0]!.id).toBe(`${task().task_uuid}:4`);
    const chunks: PluginSSEStreamEvent[] = [];
    await readSSEStream(new Response(raw.replaceAll('\n', '\r\n')).body!, event => chunks.push(event));
    expect(chunks[0]!.id).toBe(events[0]!.id);
  });
});
