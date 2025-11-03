export interface PluginApiOptions {
  pluginId: string;
  baseURL?: string;
  tenantId?: string | number;
}

export interface PluginApi {
  get<T>(path: string, init?: RequestInit): Promise<T>;
  post<T>(path: string, body: unknown, init?: RequestInit): Promise<T>;
  put<T>(path: string, body: unknown, init?: RequestInit): Promise<T>;
  delete<T>(path: string, init?: RequestInit): Promise<T>;
}

const tenantHeaderName = "X-Tenant-ID";

export function usePluginApi(options: PluginApiOptions): PluginApi {
  const prefix = (options.baseURL ?? `/_p/${options.pluginId}/api/v1`).replace(
    /\/$/,
    "",
  );

  const request = async <T>(
    method: string,
    route: string,
    init?: RequestInit,
  ): Promise<T> => {
    const headers = new Headers(init?.headers as HeadersInit);
    headers.set("Accept", "application/json");

    if (init?.body !== undefined && !(init.body instanceof FormData)) {
      if (!headers.has("Content-Type")) {
        headers.set("Content-Type", "application/json");
      }
    }

    if (options.tenantId !== undefined && !headers.has(tenantHeaderName)) {
      headers.set(tenantHeaderName, String(options.tenantId));
    }

    const response = await fetch(`${prefix}${route}`, {
      method,
      ...init,
      headers,
    });

    if (!response.ok) {
      let message = `request failed: ${response.status}`;
      try {
        const body = await response.clone().json();
        if (typeof body.message === "string") {
          message = body.message;
        } else if (body?.error) {
          if (typeof body.error === "string") {
            message = body.error;
          } else if (typeof body.error.message === "string") {
            message = body.error.message;
          }
        }
      } catch {
        // ignore parsing errors, fall back to default message
      }
      const error = new Error(message);
      (error as any).status = response.status;
      throw error;
    }

    if (response.status === 204) {
      return undefined as T;
    }
    return (await response.json()) as T;
  };

  const serializeBody = (body: unknown): BodyInit | undefined => {
    if (body === undefined || body === null) return undefined;
    if (
      body instanceof FormData ||
      body instanceof Blob ||
      body instanceof ArrayBuffer
    ) {
      return body as BodyInit;
    }
    return JSON.stringify(body);
  };

  return {
    get: (route, init) => request("GET", route, init),
    post: (route, body, init) =>
      request("POST", route, { ...init, body: serializeBody(body) }),
    put: (route, body, init) =>
      request("PUT", route, { ...init, body: serializeBody(body) }),
    delete: (route, init) => request("DELETE", route, init),
  };
}
