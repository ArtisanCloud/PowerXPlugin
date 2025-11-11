export interface DebugReportOptions {
  baseUrl: string;
  sessionId: string;
  pluginId: string;
  tenant?: string;
  findings?: string[];
  metrics?: Record<string, number>;
  attachments?: Record<string, unknown>;
}

export async function submitDebugReport(options: DebugReportOptions): Promise<{ reportId: string }> {
  if (!options.baseUrl) {
    throw new Error("baseUrl is required");
  }
  if (!options.sessionId || !options.pluginId) {
    throw new Error("sessionId and pluginId are required");
  }
  const payload = {
    sessionId: options.sessionId,
    pluginId: options.pluginId,
    tenant: options.tenant,
    findings: options.findings ?? [],
    metrics: options.metrics ?? {},
    attachments: options.attachments ?? {},
  };
  const resp = await fetch(`${options.baseUrl}/internal/dev/debug/report`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "User-Agent": "px-plugin-cli/1.0 debug-report",
    },
    body: JSON.stringify(payload),
  });
  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(`debug report failed: ${resp.status} ${resp.statusText} ${text}`);
  }
  return (await resp.json()) as { reportId: string };
}
