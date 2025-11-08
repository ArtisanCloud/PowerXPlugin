export interface PublishDeployOptions {
  baseUrl: string;
  planId: string;
  strategy?: "canary" | "blue-green" | "direct";
  batches?: Array<{ percentage: number; wait?: string }>;
  commit?: string;
  notes?: string;
  dryRun?: boolean;
}

export interface PublishDeployResponse {
  deploymentId: string;
  planId: string;
  state: string;
  batches: Array<{ percentage: number; status: string }>;
  rollbackToken?: string;
}

export async function runPublishDeployCommand(options: PublishDeployOptions): Promise<PublishDeployResponse> {
  if (!options.baseUrl) {
    throw new Error("baseUrl is required");
  }
  if (!options.planId) {
    throw new Error("planId is required");
  }

  const payload = {
    planId: options.planId,
    strategy: options.strategy ?? "canary",
    batches: options.batches ?? [],
    commit: options.commit,
    notes: options.notes,
    dryRun: options.dryRun ?? false,
  };

  const response = await fetch(`${options.baseUrl}/internal/publish/deploy`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "User-Agent": "px-plugin-cli/1.0 publish-deploy",
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(`publish deploy failed: ${response.status} ${response.statusText} ${text}`);
  }

  return (await response.json()) as PublishDeployResponse;
}
