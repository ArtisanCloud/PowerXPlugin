import fetch from "node-fetch";
import crypto from "node:crypto";

interface SessionClientOptions {
  baseUrl: string;
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

export class SessionClient {
  constructor(private readonly options: SessionClientOptions) {}

  async register(payload: RegisterPayload): Promise<RegisterResponse> {
    const response = await fetch(`${this.options.baseUrl}/internal/dev/plugins/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    if (!response.ok) {
      throw new Error(`register failed: ${response.status}`);
    }
    return (await response.json()) as RegisterResponse;
  }

  async reload(payload: ReloadPayload): Promise<void> {
    const reloadId = crypto.randomUUID();
    const response = await fetch(`${this.options.baseUrl}/internal/dev/plugins/reload`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "x-reload-id": reloadId,
      },
      body: JSON.stringify(payload),
    });
    if (!response.ok) {
      throw new Error(`reload failed: ${response.status}`);
    }
  }
}
