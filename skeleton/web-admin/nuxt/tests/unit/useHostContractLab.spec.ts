import { describe, expect, it, vi } from "vitest";

const { apiPost } = vi.hoisted(() => ({ apiPost: vi.fn() }));

vi.mock("~/composables/api/_client", () => ({ apiPost }));

import { useHostContractLabApi } from "~/composables/api/useHostContractLab";

describe("useHostContractLabApi", () => {
  it("posts only the allowlisted module operation and explicit confirmation", async () => {
    apiPost.mockResolvedValueOnce({ data: { module: "knowledge" } });
    const api = useHostContractLabApi();

    await api.probe("knowledge", "index.rebuild", { space_uuid: "space-uuid" }, true);

    expect(apiPost).toHaveBeenCalledWith("/admin/host-contract/probe", {
      module: "knowledge",
      operation: "index.rebuild",
      input: { space_uuid: "space-uuid" },
      confirm: true,
    });
  });

  it("preserves stable reason and trace from the Host Contract envelope", () => {
    const api = useHostContractLabApi();
    expect(api.normalizeError({
      status: 403,
      data: { error: { reason_code: "HOST_CONTRACT_FORBIDDEN", trace_id: "trace-1" } },
    })).toEqual({ status: 403, reasonCode: "HOST_CONTRACT_FORBIDDEN", traceID: "trace-1" });
  });
});
