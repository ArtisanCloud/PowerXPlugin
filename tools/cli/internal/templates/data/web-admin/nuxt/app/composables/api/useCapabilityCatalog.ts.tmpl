import { apiGet } from "./_client";
import type { ApiResponse } from "./_base";

export type CapabilityCatalogEntry = {
  id: string;
  version: string;
  descriptor: string;
  module?: string;
  kind?: string;
  tags: string[];
  checksum: string;
  execution: {
    mode: string;
    callback_url?: string;
    sse_channel?: string;
    status_endpoint?: string;
  };
  protocols?: Record<string, any>;
};

export function useCapabilityCatalogApi() {
  const list = () =>
    apiGet<ApiResponse<CapabilityCatalogEntry[]>>(
      "admin/capabilities",
    ).then((res) => res.data);

  return {
    list,
  };
}
