import path from "node:path";
import fs from "node:fs";

export interface HostStartOptions {
  baseUrl: string;
  pluginId: string;
  runtimeVersion?: string;
  tenant?: string;
  mock?: boolean;
  manifestPath?: string;
  certPath?: string;
  keyPath?: string;
  caPath?: string;
}

export interface HostAttachOptions {
  baseUrl: string;
  sessionId: string;
  breakpoints: Array<{ file: string; line: number }>;
  variables?: Record<string, string>;
}

export interface HostStatusOptions {
  baseUrl: string;
  sessionId: string;
}

export interface HostStopOptions {
  baseUrl: string;
  sessionId: string;
}

interface HostSessionResponse {
  sessionId: string;
  status: string;
  endpoint: string;
  mock: boolean;
  startedAt: string;
}

async function readManifest(manifestPath?: string) {
  if (!manifestPath) {
    return undefined;
  }
  const resolved = path.resolve(manifestPath);
  const content = await fs.promises.readFile(resolved, "utf-8");
  return JSON.parse(content);
}

async function request<T>(url: string, init: RequestInit): Promise<T> {
  const resp = await fetch(url, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      "User-Agent": "px-plugin-cli/1.0 host",
      ...(init.headers ?? {}),
    },
  });
  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(`host API ${url} failed: ${resp.status} ${resp.statusText} ${text}`);
  }
  if (resp.status === 204) {
    return undefined as T;
  }
  return (await resp.json()) as T;
}

export async function startHostSession(options: HostStartOptions): Promise<HostSessionResponse> {
  if (!options.baseUrl) {
    throw new Error("baseUrl is required");
  }
  if (!options.pluginId) {
    throw new Error("pluginId is required");
  }
  const manifest = await readManifest(options.manifestPath);
  const payload = {
    pluginId: options.pluginId,
    runtimeVersion: options.runtimeVersion ?? "latest",
    tenant: options.tenant,
    mock: options.mock ?? true,
    manifest,
  };
  const url = `${options.baseUrl}/internal/dev/hosts/sessions`;
  return request<HostSessionResponse>(url, {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function stopHostSession(options: HostStopOptions): Promise<void> {
  const url = `${options.baseUrl}/internal/dev/hosts/sessions/${options.sessionId}`;
  await request<void>(url, { method: "DELETE" });
}

export async function fetchHostStatus(options: HostStatusOptions): Promise<HostSessionResponse> {
  const url = `${options.baseUrl}/internal/dev/hosts/sessions/${options.sessionId}`;
  return request<HostSessionResponse>(url, { method: "GET" });
}

export async function attachHostDebugger(options: HostAttachOptions): Promise<{ attached: boolean }> {
  const url = `${options.baseUrl}/internal/dev/hosts/sessions/${options.sessionId}/attach`;
  const payload = {
    breakpoints: options.breakpoints,
    variables: options.variables ?? {},
  };
  return request<{ attached: boolean }>(url, {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function syncHostLogs(baseUrl: string, sessionId: string): Promise<string[]> {
  const url = `${baseUrl}/internal/dev/hosts/sessions/${sessionId}/logs`;
  return request<string[]>(url, { method: "GET" });
}
