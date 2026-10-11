import type { PowerXAuthMessage, PowerXAuthTokenPayload } from './bridge';

export type HostSessionStatus = 'empty' | 'applying' | 'ready' | 'cleared' | 'rejected';
export interface PowerXHostIdentity { readonly tenantUuid: string | null; readonly memberUuid: string | null; }
export type PowerXHostSessionSnapshot = Readonly<PowerXAuthTokenPayload & { epoch: number; identity?: PowerXHostIdentity }>;
export interface PowerXHostSessionState {
  status: HostSessionStatus;
  snapshot: PowerXHostSessionSnapshot | null;
  error: string | null;
}
export interface PowerXHostSessionAdapter<T> {
  readContext(snapshot: PowerXHostSessionSnapshot, signal: AbortSignal): Promise<T>;
  validateContext(context: T, snapshot: PowerXHostSessionSnapshot): PowerXHostIdentity;
  commitContext(context: T, snapshot: PowerXHostSessionSnapshot): void;
  clearContext(): void;
}
export class PowerXHostSessionError extends Error {
  readonly statusCode: number;
  constructor(public readonly code: string) { super(code); this.name = 'PowerXHostSessionError'; this.statusCode = code === 'AUTH_SESSION_HOST_MANAGED' ? 409 : 401; }
}
export const normalizeBearerToken = (token: string) => /^Bearer\s/i.test(token) ? token : `Bearer ${token}`;
const failure = (code: string) => new PowerXHostSessionError(code);

type Waiter = {
  expected: PowerXHostSessionSnapshot | null; minimumSequence: number;
  resolve: (snapshot: PowerXHostSessionSnapshot) => void;
  reject: (error: Error) => void; timer: ReturnType<typeof setTimeout>;
};

/** One consumer per plugin frame. Never reads or writes host storage. */
export class PowerXHostSession {
  private state: PowerXHostSessionState = { status: 'empty', snapshot: null, error: null };
  private instanceId = '';
  private sequence = 0;
  private epoch = 0;
  private controller: AbortController | null = null;
  private listeners = new Set<(state: Readonly<PowerXHostSessionState>) => void>();
  private waiters = new Set<Waiter>();
  private requester: (() => void) | null = null;
  private requested = false;
  private applying: Promise<void> | null = null;
  private clearIdentityContext: (() => void) | null = null;
  constructor(public readonly pluginId: string) {}

  getState(): Readonly<PowerXHostSessionState> { return { ...this.state }; }
  getReadySnapshot(): PowerXHostSessionSnapshot | null {
    return this.state.status === 'ready' && this.state.snapshot!.expiresAt > Date.now() ? this.state.snapshot : null;
  }
  isCurrent(snapshot: PowerXHostSessionSnapshot): boolean { return this.state.snapshot === snapshot; }
  isSameSession(snapshot: PowerXHostSessionSnapshot): boolean {
    return snapshot.epoch === this.epoch && this.state.snapshot?.sessionId === snapshot.sessionId && snapshot.instanceId === this.instanceId;
  }
  isContextSnapshot(snapshot: PowerXHostSessionSnapshot): boolean {
    return this.state.status === 'applying' && this.isCurrent(snapshot) && snapshot.expiresAt > Date.now();
  }
  subscribe(listener: (state: Readonly<PowerXHostSessionState>) => void): () => void {
    this.listeners.add(listener); listener(this.getState());
    return () => { this.listeners.delete(listener); };
  }
  setRequester(requester: () => void): () => void {
    this.requester = requester;
    return () => { if (this.requester === requester) { this.requester = null; this.invalidate('PLUGIN_BRIDGE_DISPOSED'); } };
  }
  ownsRequester(requester: () => void): boolean { return this.requester === requester; }
  bindInstance(instanceId: string): void {
    if (!instanceId || instanceId === this.instanceId) return;
    if (this.instanceId) this.invalidate('AUTH_SESSION_CHANGED');
    this.instanceId = instanceId; this.sequence = 0;
  }
  invalidate(code = 'AUTH_SESSION_CHANGED'): void {
    this.epoch++; this.controller?.abort(); this.controller = null;
    this.state = { status: 'cleared', snapshot: null, error: code };
    const clear = this.clearIdentityContext; this.clearIdentityContext = null;
    try { clear?.(); } finally { this.rejectWaiters(failure(code)); this.emit(); }
  }
  private emit(): void { this.listeners.forEach(listener => listener(this.getState())); }
  private rejectWaiters(error: Error): void {
    for (const waiter of this.waiters) { clearTimeout(waiter.timer); waiter.reject(error); }
    this.waiters.clear(); this.requested = false;
  }
  private settle(): void {
    const snapshot = this.getReadySnapshot();
    if (!snapshot) return;
    for (const waiter of this.waiters) {
      if (waiter.expected && !this.isSameSession(waiter.expected)) {
        clearTimeout(waiter.timer); waiter.reject(failure('AUTH_SESSION_CHANGED')); this.waiters.delete(waiter);
      } else if (snapshot.sequence > waiter.minimumSequence) {
        clearTimeout(waiter.timer); waiter.resolve(snapshot); this.waiters.delete(waiter);
      }
    }
    if (!this.waiters.size) this.requested = false;
  }
  ensureSnapshot(options: { force?: boolean; expected?: PowerXHostSessionSnapshot; timeoutMs?: number; signal?: AbortSignal; recovery?: boolean } = {}): Promise<PowerXHostSessionSnapshot> {
    if (options.signal?.aborted) return Promise.reject(failure('AUTH_REQUEST_ABORTED'));
    if (options.expected && !this.isSameSession(options.expected)) return Promise.reject(failure('AUTH_SESSION_CHANGED'));
    const ready = this.getReadySnapshot();
    if (ready && !options.force) return Promise.resolve(ready);
    if (this.state.status === 'cleared' && !options.recovery) return Promise.reject(failure('PLUGIN_AUTH_REQUIRED'));
    if (!this.requester) return Promise.reject(failure('PLUGIN_BRIDGE_UNAVAILABLE'));
    // Join a validation already in progress instead of requesting another copy.
    const minimumSequence = this.state.status === 'applying' ? this.sequence - 1 : this.sequence;
    return new Promise<PowerXHostSessionSnapshot>((resolve, reject) => {
      let waiter: Waiter;
      const finish = (error?: Error, snapshot?: PowerXHostSessionSnapshot) => {
        clearTimeout(waiter.timer); this.waiters.delete(waiter);
        options.signal?.removeEventListener('abort', aborted);
        if (!this.waiters.size) this.requested = false;
        if (error) reject(error); else resolve(snapshot!);
      };
      const aborted = () => finish(failure('AUTH_REQUEST_ABORTED'));
      waiter = {
        expected: options.expected || this.state.snapshot, minimumSequence,
        resolve: snapshot => finish(undefined, snapshot), reject: error => finish(error),
        timer: setTimeout(() => finish(failure('PLUGIN_AUTH_ADAPTER_TIMEOUT')), options.timeoutMs ?? 8000),
      };
      this.waiters.add(waiter); options.signal?.addEventListener('abort', aborted, { once: true });
      if (!this.requested && this.state.status !== 'applying') {
        this.requested = true;
        try { this.requester!(); } catch { this.rejectWaiters(failure('PLUGIN_BRIDGE_UNAVAILABLE')); }
      }
    });
  }
  apply<T>(message: PowerXAuthMessage, adapter: PowerXHostSessionAdapter<T>): Promise<void> {
    if (message.protocolVersion !== 2 || message.pluginId !== this.pluginId || !message.instanceId) return Promise.reject(failure('PLUGIN_BRIDGE_PROTOCOL_INVALID'));
    if (!this.instanceId) this.bindInstance(message.instanceId);
    else if (message.instanceId !== this.instanceId) return Promise.reject(failure('PLUGIN_BRIDGE_INSTANCE_STALE'));
    if (!Number.isSafeInteger(message.sequence) || message.sequence <= 0) return Promise.reject(failure('PLUGIN_BRIDGE_SEQUENCE_INVALID'));
    if (message.sequence < this.sequence) return Promise.resolve();
    if (message.sequence === this.sequence) {
      if (this.applying) return this.applying;
      if (this.state.status === 'ready') return Promise.resolve();
      return Promise.reject(failure('AUTH_SESSION_CHANGED'));
    }
    if (message.type === 'auth-token' && this.state.snapshot?.sessionId === message.sessionId && message.revision < this.state.snapshot.revision) return Promise.reject(failure('AUTH_SESSION_REVISION_STALE'));
    this.sequence = message.sequence;
    this.controller?.abort();
    if (message.type === 'auth-cleared') {
      this.clearIdentityContext = () => adapter.clearContext();
      this.invalidate('PLUGIN_AUTH_REQUIRED'); this.applying = null;
      return Promise.resolve();
    }
    if (typeof message.sessionId !== 'string' || !message.sessionId || !Number.isSafeInteger(message.revision) || message.revision < 1 || typeof message.accessToken !== 'string' || !message.accessToken || message.tokenType !== 'Bearer' || typeof message.scope !== 'string' || !(message.tenantUuid === null || typeof message.tenantUuid === 'string') || !Number.isFinite(message.expiresAt) || message.expiresAt <= Date.now()) {
      this.clearIdentityContext = () => adapter.clearContext();
      this.invalidate('PLUGIN_AUTH_TOKEN_INVALID');
      return Promise.reject(failure('PLUGIN_AUTH_TOKEN_INVALID'));
    }
    if (this.state.snapshot?.sessionId !== message.sessionId) {
      this.epoch++; adapter.clearContext();
      for (const waiter of this.waiters) if (waiter.expected) { clearTimeout(waiter.timer); waiter.reject(failure('AUTH_SESSION_CHANGED')); this.waiters.delete(waiter); }
    }
    const snapshot: PowerXHostSessionSnapshot = Object.freeze({
      source: 'powerx', type: 'auth-token', protocolVersion: 2, pluginId: this.pluginId,
      instanceId: message.instanceId, sequence: message.sequence, sessionId: message.sessionId, revision: message.revision,
      accessToken: message.accessToken, tokenType: message.tokenType, expiresAt: message.expiresAt,
      expiresIn: message.expiresIn, scope: message.scope, tenantUuid: message.tenantUuid, epoch: this.epoch,
    });
    this.clearIdentityContext = () => adapter.clearContext();
    const controller = new AbortController(); this.controller = controller;
    this.state = { status: 'applying', snapshot, error: null }; this.emit();
    const applying = (async () => {
      try {
        const context = await adapter.readContext(snapshot, controller.signal);
        if (!this.isCurrent(snapshot) || controller.signal.aborted) return;
        const identity = adapter.validateContext(context, snapshot);
        if (!identity || !(identity.tenantUuid === null || typeof identity.tenantUuid === 'string') || !(identity.memberUuid === null || typeof identity.memberUuid === 'string') || (snapshot.tenantUuid && identity.tenantUuid !== snapshot.tenantUuid)) throw failure('AUTH_CONTEXT_IDENTITY_INVALID');
        if (!this.isCurrent(snapshot) || controller.signal.aborted) return;
        if (snapshot.expiresAt <= Date.now()) throw failure('PLUGIN_AUTH_TOKEN_EXPIRED');
        const ready: PowerXHostSessionSnapshot = Object.freeze({ ...snapshot, identity: Object.freeze({ ...identity }) });
        adapter.commitContext(context, ready);
        this.state = { status: 'ready', snapshot: ready, error: null }; this.emit(); this.settle();
      } catch (error) {
        if (!this.isCurrent(snapshot) || controller.signal.aborted) return;
        this.state = { status: 'rejected', snapshot, error: 'PLUGIN_AUTH_CONTEXT_FAILED' };
        adapter.clearContext(); this.rejectWaiters(failure('PLUGIN_AUTH_CONTEXT_FAILED')); this.emit();
        throw error;
      } finally { if (this.controller === controller) this.applying = null; }
    })();
    this.applying = applying; return applying;
  }
}

const REGISTRY = '__POWERX_V2_CONSUMER_SESSIONS__';
export function getPowerXHostSession(pluginId: string): PowerXHostSession {
  if (typeof window === 'undefined') throw failure('HOST_SESSION_BROWSER_REQUIRED');
  const holder = window as unknown as Record<string, unknown>;
  const registry = (holder[REGISTRY] ||= new Map<string, PowerXHostSession>()) as Map<string, PowerXHostSession>;
  if (!registry.has(pluginId)) registry.set(pluginId, new PowerXHostSession(pluginId));
  return registry.get(pluginId)!;
}
