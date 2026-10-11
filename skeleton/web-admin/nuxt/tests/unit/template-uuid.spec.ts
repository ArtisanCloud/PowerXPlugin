import { describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({ apiGet: vi.fn(), apiPost: vi.fn(), apiPut: vi.fn(), apiDel: vi.fn(), useApiClient: () => ({ baseURL: "/api/v1/admin" }) }));
vi.mock("~/composables/api/_client", () => mocks);
import { useTemplateApi, type Template } from "~/composables/api/useTemplate";

describe("template UUID contract", () => {
  it("addresses reads, updates and deletes with the returned UUID", async () => {
    const item: Template = { uuid: "11111111-1111-4111-8111-111111111111", name: "fixture", description: "fixture", content: "fixture" };
    const api = useTemplateApi();
    await api.getTemplate(item.uuid);
    await api.updateTemplate(item.uuid, { content: "changed" });
    await api.deleteTemplate(item.uuid);
    expect(mocks.apiGet).toHaveBeenCalledWith(`templates/${item.uuid}`, undefined, undefined);
    expect(mocks.apiPut).toHaveBeenCalledWith(`templates/${item.uuid}`, { content: "changed" }, undefined);
    expect(mocks.apiDel).toHaveBeenCalledWith(`templates/${item.uuid}`, undefined);
  });
});
