export type PowerXThemeKey = 'light' | 'dark' | 'system';
export interface PowerXBridgeEnvelope { protocolVersion: 2; pluginId: string; instanceId: string; }
export interface PowerXSyncPayload extends PowerXBridgeEnvelope { source: 'powerx'; type: 'sync'; locale: string; theme: PowerXThemeKey; hostOrigin: string; }
export interface PowerXAuthTokenPayload extends PowerXBridgeEnvelope {
  source: 'powerx'; type: 'auth-token'; sequence: number; sessionId: string; revision: number;
  accessToken: string; tokenType: string; expiresAt: number; expiresIn: number; scope: string; tenantUuid: string | null;
}
export interface PowerXAuthClearedPayload extends PowerXBridgeEnvelope { source: 'powerx'; type: 'auth-cleared'; sequence: number; sessionId?: string; revision?: number; }
export type PowerXAuthMessage = PowerXAuthTokenPayload | PowerXAuthClearedPayload;
export interface PowerXNavigationRequest { signal: AbortSignal; requestId: string; instanceId: string; }
export interface PowerXFrameBridgeClient {
  readonly protocolVersion: 2;
  onAuthToken(handler: (message: PowerXAuthMessage) => void | Promise<void>): () => void;
  setNavigator(handler: (path: string, request: PowerXNavigationRequest) => void | Promise<void>): () => void;
  requestAuth(): void; requestSync(): void; getLastSyncAt(): number;
}
export type PowerXHostMessage = PowerXSyncPayload | PowerXAuthMessage | (PowerXBridgeEnvelope & { source: 'powerx'; type: 'navigate'; path: string; requestId: string });
export type PowerXFullscreenAction = 'enter' | 'exit' | 'toggle';
export interface PowerXFullscreenRequestOptions { pluginId?: string; instanceId?: string; route?: string; reason?: string; }
export type PowerXPluginFullscreenPayload = PowerXBridgeEnvelope & { source: 'powerx-plugin'; type: 'fullscreen:request'; action: PowerXFullscreenAction; route?: string; reason?: string; };
export type PowerXPluginToHost = PowerXBridgeEnvelope & { source: 'plugin' | 'powerx-plugin'; type: string };
export interface PowerXBridgeLogger { info?: (...args: any[]) => void; warn?: (...args: any[]) => void; error?: (...args: any[]) => void; }
export interface PowerXBridgeOptions {
  pluginId: string; debug?: boolean; allowedOrigins?: string[]; logger?: PowerXBridgeLogger;
  onTheme?: (theme: PowerXThemeKey) => void; onLocale?: (locale: string) => void | Promise<void>;
  onSync?: (payload: PowerXSyncPayload) => void | Promise<void>;
  onAuthToken?: (payload: PowerXAuthMessage) => void | Promise<void>;
  navigator?: (path: string, request: PowerXNavigationRequest) => void | Promise<void>;
}
export function getPowerXFrameBridge(): PowerXFrameBridgeClient {
  const bridge = typeof window === 'undefined' ? undefined : (window as Window & { PowerXBridgeClient?: PowerXFrameBridgeClient }).PowerXBridgeClient;
  if (!bridge || bridge.protocolVersion !== 2 || typeof bridge.onAuthToken !== 'function' || typeof bridge.setNavigator !== 'function' || typeof bridge.requestAuth !== 'function' || typeof bridge.requestSync !== 'function' || typeof bridge.getLastSyncAt !== 'function') throw new Error('PLUGIN_BRIDGE_V2_REQUIRED');
  return bridge;
}

function retireLegacyReceiver(): void {
  if (typeof window === 'undefined') return;
  const holder = window as unknown as Record<string, any>;
  const legacy = holder.__POWERX_FRAMEWORK_BRIDGE__;
  if (legacy) {
    if (typeof legacy.stop !== 'function') throw new Error('PLUGIN_BRIDGE_LEGACY_DISPOSAL_REQUIRED');
    legacy.stop(); delete holder.__POWERX_FRAMEWORK_BRIDGE__;
  }
  delete holder.__POWERX_PLUGIN_HOST_SESSION_CACHE__;
}

/** Facade over Core's consumer. No second authentication message listener. */
export class PowerXBridgeClient {
  private cleanup: Array<() => void> = [];
  private origin: string;
  private instanceId = '';
  private started = false;
  readonly consumer: PowerXFrameBridgeClient;
  constructor(private options: PowerXBridgeOptions) {
    retireLegacyReceiver();
    this.consumer = getPowerXFrameBridge();
    this.origin = '';
    try { this.origin = new URL(document.referrer).origin; } catch { /* Missing trusted origin is a contract error. */ }
    if (!this.origin || options.allowedOrigins?.includes('*') || (options.allowedOrigins?.length && !options.allowedOrigins.includes(this.origin))) throw new Error('PLUGIN_BRIDGE_ORIGIN_INVALID');
  }
  start(): void {
    if (this.started) return;
    this.started = true;
    window.addEventListener('message', this.metadata);
    this.cleanup.push(() => window.removeEventListener('message', this.metadata));
    if (this.options.onAuthToken) this.cleanup.push(this.consumer.onAuthToken(message => {
      if (message.protocolVersion !== 2 || message.pluginId !== this.options.pluginId || !message.instanceId) throw new Error('PLUGIN_BRIDGE_PROTOCOL_INVALID');
      this.instanceId = message.instanceId;
      return this.options.onAuthToken!(message);
    }));
    if (this.options.navigator) this.cleanup.push(this.consumer.setNavigator(this.options.navigator));
    this.consumer.requestSync();
  }
  stop(): void { this.cleanup.splice(0).reverse().forEach(dispose => dispose()); this.started = false; }
  requestSync(): void { this.consumer.requestSync(); }
  requestAuthToken(): void { this.consumer.requestAuth(); }
  private metadata = (event: MessageEvent) => {
    const payload = event.data;
    if (event.source !== window.parent || event.origin !== this.origin || payload?.source !== 'powerx' || payload.protocolVersion !== 2 || payload.pluginId !== this.options.pluginId || !payload.instanceId || payload.type !== 'sync') return;
    this.instanceId = payload.instanceId;
    const result = Promise.all([this.options.onSync?.(payload as PowerXSyncPayload), this.options.onTheme?.(payload.theme), this.options.onLocale?.(payload.locale)]);
    result.catch(error => this.options.logger?.error?.('PLUGIN_BRIDGE_METADATA_FAILED', { code: String(error?.code || 'PLUGIN_BRIDGE_METADATA_FAILED') }));
  };
  requestFullscreen(options: PowerXFullscreenRequestOptions = {}): boolean { return this.fullscreen('enter', options); }
  exitFullscreen(options: PowerXFullscreenRequestOptions = {}): boolean { return this.fullscreen('exit', options); }
  toggleFullscreen(options: PowerXFullscreenRequestOptions = {}): boolean { return this.fullscreen('toggle', options); }
  private fullscreen(action: PowerXFullscreenAction, options: PowerXFullscreenRequestOptions): boolean {
    if (!this.instanceId || window.parent === window) return false;
    window.parent.postMessage({ ...options, source: 'powerx-plugin', type: 'fullscreen:request', action, protocolVersion: 2, pluginId: this.options.pluginId, instanceId: this.instanceId }, this.origin); return true;
  }
}
const clients = new WeakMap<Window, Map<string, PowerXBridgeClient>>();
export function initPowerXBridge(options: PowerXBridgeOptions): PowerXBridgeClient {
  if (!clients.has(window)) clients.set(window, new Map());
  const registry = clients.get(window)!;
  registry.get(options.pluginId)?.stop();
  const client = new PowerXBridgeClient(options); client.start(); registry.set(options.pluginId, client);
  return client;
}
export function requestPowerXHostAuthToken(): void { getPowerXFrameBridge().requestAuth(); }
export function requestPowerXHostFullscreen(pluginId: string, options: PowerXFullscreenRequestOptions = {}): boolean { return clients.get(window)?.get(pluginId)?.requestFullscreen(options) || false; }
export function exitPowerXHostFullscreen(pluginId: string, options: PowerXFullscreenRequestOptions = {}): boolean { return clients.get(window)?.get(pluginId)?.exitFullscreen(options) || false; }
export function togglePowerXHostFullscreen(pluginId: string, options: PowerXFullscreenRequestOptions = {}): boolean { return clients.get(window)?.get(pluginId)?.toggleFullscreen(options) || false; }

export function isPowerXEmbedded(value?: unknown): boolean {
  return value === true || value === 'true' || value === '1' || (typeof window !== 'undefined' && Boolean((window as Window & { PowerXBridgeClient?: unknown }).PowerXBridgeClient));
}
