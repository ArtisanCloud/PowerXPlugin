import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { performance } from "node:perf_hooks";

interface SessionClientOptions {
  baseUrl: string;
  certPath?: string;
  keyPath?: string;
  caPath?: string;
  maxRetries?: number;
  retryDelayMs?: number;
  hostSessionId?: string;
}

interface RegisterPayload {
  manifest: any;
  tenant?: string;
}

interface RegisterResponse {
  sessionId: string;
  reloadToken: string;
  adminPreviewUrl?: string;
}

interface ReloadPayload {
  sessionId: string;
  reloadToken: string;
  changedFiles: string[];
}

interface AttachPayload {
  breakpoints: Array<{ file: string; line: number }>;
  variables?: Record<string, string>;
}

interface CertificateConfig {
  cert?: Buffer;
  key?: Buffer;
  ca?: Buffer;
}

export class SessionClient {
  private readonly certConfig: CertificateConfig;
  private readonly maxRetries: number;
  private readonly retryDelayMs: number;
  private hostSessionId?: string;

  constructor(private readonly options: SessionClientOptions) {
    this.maxRetries = options.maxRetries ?? 3;
    this.retryDelayMs = options.retryDelayMs ?? 1000;
    this.hostSessionId = options.hostSessionId;
    this.certConfig = {
      cert: options.certPath ? this.loadCert(options.certPath) : undefined,
      key: options.keyPath ? this.loadCert(options.keyPath) : undefined,
      ca: options.caPath ? this.loadCert(options.caPath) : undefined,
    };
  }

  configureHostSession(hostSessionId: string) {
    this.hostSessionId = hostSessionId;
  }

  private loadCert(certPath: string): Buffer {
    try {
      const resolved = path.resolve(certPath);
      return fs.readFileSync(resolved);
    } catch (error) {
      throw new Error(`Failed to load certificate from ${certPath}: ${error}`);
    }
  }

  async register(payload: RegisterPayload, retryCount = 0): Promise<RegisterResponse> {
    const startedAt = performance.now();
    const requestOptions: RequestInit = {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "User-Agent": "px-plugin-cli/1.0",
      },
      body: JSON.stringify(payload),
      // agent 配置将在后续实现 mTLS 时添加
    };

    try {
      const response = await fetch(`${this.options.baseUrl}/internal/dev/plugins/register`, requestOptions);

      if (!response.ok) {
        if (response.status >= 500 && retryCount < this.maxRetries) {
          console.warn(`Register failed (attempt ${retryCount + 1}), retrying in ${this.retryDelayMs}ms...`);
          await this.delay(this.retryDelayMs * Math.pow(2, retryCount)); // Exponential backoff
          return this.register(payload, retryCount + 1);
        }
        throw new Error(`register failed: ${response.status} ${response.statusText}`);
      }
      const json = (await response.json()) as RegisterResponse;
      this.emitMetric("dev.hotload.register_ms", performance.now() - startedAt);
      return json;
    } catch (error) {
      if (retryCount < this.maxRetries) {
        console.warn(`Register error (attempt ${retryCount + 1}): ${error}, retrying...`);
        await this.delay(this.retryDelayMs * Math.pow(2, retryCount));
        return this.register(payload, retryCount + 1);
      }
      throw error;
    }
  }

  async reload(payload: ReloadPayload, retryCount = 0): Promise<void> {
    const startedAt = performance.now();
    const reloadId = crypto.randomUUID();
    const requestOptions: RequestInit = {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "x-reload-id": reloadId, // 幂等控制
        "User-Agent": "px-plugin-cli/1.0",
      },
      body: JSON.stringify(payload),
    };

    try {
      const response = await fetch(`${this.options.baseUrl}/internal/dev/plugins/reload`, requestOptions);

      if (!response.ok) {
        if (response.status >= 500 && retryCount < this.maxRetries) {
          console.warn(`Reload failed (attempt ${retryCount + 1}), retrying in ${this.retryDelayMs}ms...`);
          await this.delay(this.retryDelayMs * Math.pow(2, retryCount));
          return this.reload(payload, retryCount + 1);
        }
        throw new Error(`reload failed: ${response.status} ${response.statusText}`);
      }
      this.emitMetric("dev.hotload.reload_ms", performance.now() - startedAt);
    } catch (error) {
      if (retryCount < this.maxRetries) {
        console.warn(`Reload error (attempt ${retryCount + 1}): ${error}, retrying...`);
        await this.delay(this.retryDelayMs * Math.pow(2, retryCount));
        return this.reload(payload, retryCount + 1);
      }
      throw error;
    }
  }

  async delete(sessionId: string, retryCount = 0): Promise<void> {
    const requestOptions: RequestInit = {
      method: "DELETE",
      headers: {
        "User-Agent": "px-plugin-cli/1.0",
      },
    };

    try {
      const response = await fetch(
        `${this.options.baseUrl}/internal/dev/plugins/register/${sessionId}`,
        requestOptions
      );

      if (!response.ok) {
        if (response.status >= 500 && retryCount < this.maxRetries) {
          console.warn(`Delete failed (attempt ${retryCount + 1}), retrying in ${this.retryDelayMs}ms...`);
          await this.delay(this.retryDelayMs * Math.pow(2, retryCount));
          return this.delete(sessionId, retryCount + 1);
        }
        throw new Error(`delete failed: ${response.status} ${response.statusText}`);
      }
    } catch (error) {
      if (retryCount < this.maxRetries) {
        console.warn(`Delete error (attempt ${retryCount + 1}): ${error}, retrying...`);
        await this.delay(this.retryDelayMs * Math.pow(2, retryCount));
        return this.delete(sessionId, retryCount + 1);
      }
      throw error;
    }
  }

  async attachBreakpoints(payload: AttachPayload, retryCount = 0): Promise<void> {
    if (!this.hostSessionId) {
      console.warn("host session id not configured, skip attaching breakpoints");
      return;
    }
    const requestOptions: RequestInit = {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "User-Agent": "px-plugin-cli/1.0",
      },
      body: JSON.stringify(payload),
    };
    try {
      const url = `${this.options.baseUrl}/internal/dev/hosts/sessions/${this.hostSessionId}/attach`;
      const response = await fetch(url, requestOptions);
      if (!response.ok) {
        if (response.status >= 500 && retryCount < this.maxRetries) {
          await this.delay(this.retryDelayMs * Math.pow(2, retryCount));
          return this.attachBreakpoints(payload, retryCount + 1);
        }
        throw new Error(`attach failed: ${response.status} ${response.statusText}`);
      }
    } catch (error) {
      if (retryCount < this.maxRetries) {
        await this.delay(this.retryDelayMs * Math.pow(2, retryCount));
        return this.attachBreakpoints(payload, retryCount + 1);
      }
      throw error;
    }
  }

  private delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  private emitMetric(name: string, value: number) {
    const line = `${name}=${value.toFixed(2)}ms`;
    console.debug(`[telemetry] ${line}`);
  }
}
