import { describe, it, expect, beforeEach, vi } from "vitest";
import { setActivePinia, createPinia } from "pinia";

const getUserContextMock = vi.fn();
const switchTenantMock = vi.fn();
const persistTenantUUIDMock = vi.fn();
const getStoredTenantUUIDMock = vi.fn(() => null);

vi.mock("~/composables/useMe", () => ({
  useMe: () => ({
    getUserContext: getUserContextMock,
    switchTenant: switchTenantMock,
  }),
}));

vi.mock("~/utils/tenant-context", () => ({
  persistTenantUUID: persistTenantUUIDMock,
  getStoredTenantUUID: getStoredTenantUUIDMock,
}));

const createSampleContext = () => ({
  is_root: true,
  current_tenant_uuid: "tenant-root",
  current_member_id: 101,
  user: {
    id: 1,
    email: "admin@local.test",
    phone: "+86-123456",
    display_name: "Standalone Admin",
    avatar_url: "/avatar.png",
    status: 1,
  },
  members: [
    {
      tenant_uuid: "tenant-root",
      tenant_name: "Root Org",
      member_id: 101,
      is_admin: true,
    },
    {
      tenant_uuid: "tenant-beta",
      tenant_name: "Beta Org",
      member_id: 202,
      is_admin: false,
    },
  ],
});

describe("useUserStore", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setActivePinia(createPinia());
    getUserContextMock.mockReset();
    switchTenantMock.mockReset();
    persistTenantUUIDMock.mockReset();
    getStoredTenantUUIDMock.mockReset();
    getStoredTenantUUIDMock.mockReturnValue(null);
  });

  it("fetches user context and persists tenant uuid", async () => {
    const sample = createSampleContext();
    getUserContextMock.mockResolvedValueOnce(sample);
    const { useUserStore } = await import("~/stores/user");
    const store = useUserStore();

    await store.fetchUserContext();

    expect(store.context).toEqual(sample);
    expect(store.isRoot).toBe(true);
    expect(persistTenantUUIDMock).toHaveBeenCalledWith("tenant-root");
  });

  it("skips refetch when cached and not forced", async () => {
    const sample = createSampleContext();
    getUserContextMock.mockResolvedValue(sample);
    const { useUserStore } = await import("~/stores/user");
    const store = useUserStore();

    await store.fetchUserContext();
    await store.fetchUserContext();

    expect(getUserContextMock).toHaveBeenCalledTimes(1);
  });

  it("switchTenant updates context and persists new uuid", async () => {
    const initial = createSampleContext();
    getUserContextMock.mockResolvedValue(initial);
    const switched = {
      ...initial,
      current_tenant_uuid: "tenant-beta",
      current_member_id: 202,
    };
    switchTenantMock.mockResolvedValue(switched);
    const { useUserStore } = await import("~/stores/user");
    const store = useUserStore();

    await store.fetchUserContext();
    persistTenantUUIDMock.mockClear();

    await store.switchTenant("tenant-beta");

    expect(switchTenantMock).toHaveBeenCalledWith("tenant-beta");
    expect(store.context?.current_tenant_uuid).toBe("tenant-beta");
    expect(store.currentMemberId).toBe(202);
    expect(persistTenantUUIDMock).toHaveBeenCalledWith("tenant-beta");
  });

  it("clearUserState resets context and removes stored tenant uuid", async () => {
    const sample = createSampleContext();
    getUserContextMock.mockResolvedValue(sample);
    const { useUserStore } = await import("~/stores/user");
    const store = useUserStore();

    await store.fetchUserContext();
    persistTenantUUIDMock.mockClear();

    store.clearUserState();

    expect(store.context).toBeNull();
    expect(store.isLoading).toBe(false);
    expect(persistTenantUUIDMock).toHaveBeenCalledWith(null);
  });
});
