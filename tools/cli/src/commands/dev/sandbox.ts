export interface SandboxDeployOptions {
  baseUrl: string;
  hostSessionId: string;
  datasetId?: string;
  testPlanId?: string;
  flags?: string[];
}

export interface SandboxDeployResponse {
  validationId: string;
  status: string;
  startedAt: string;
}

export async function deploySandboxValidation(options: SandboxDeployOptions): Promise<SandboxDeployResponse> {
  if (!options.baseUrl) {
    throw new Error("baseUrl is required");
  }
  if (!options.hostSessionId) {
    throw new Error("hostSessionId is required");
  }
  const payload = {
    hostSessionId: options.hostSessionId,
    datasetId: options.datasetId,
    testPlanId: options.testPlanId,
    flags: options.flags ?? [],
  };
  const resp = await fetch(`${options.baseUrl}/internal/dev/sandbox/deploy`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "User-Agent": "px-plugin-cli/1.0 sandbox",
    },
    body: JSON.stringify(payload),
  });
  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(`sandbox deploy failed: ${resp.status} ${resp.statusText} ${text}`);
  }
  return (await resp.json()) as SandboxDeployResponse;
}
