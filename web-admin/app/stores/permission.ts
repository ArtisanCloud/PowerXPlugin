import { computed, readonly, ref } from "vue";
import { defineStore } from "pinia";
import { useApiClient } from "~/composables/api";

export type PermissionMeta = {
  label?: string;
  module?: string;
  type?: "menu" | "action" | "data" | "api";
  api_endpoint?: string;
  http_method?: "GET" | "POST" | "PUT" | "DELETE" | "PATCH";
};

export interface Permission {
  permission_uuid: string;
  plugin: string;
  resource: string;
  action: string;
  description?: string;
  status: "active" | "deprecated";
  meta?: PermissionMeta;
}

export interface PermissionListQuery {
  page?: number;
  size?: number;
  status?: "active" | "deprecated";
  keyword?: string;
}

export interface PermissionListResponse {
  items: Permission[];
  pagination: { total: number; page: number; page_size: number; pages: number };
}

export const usePermissionStore = defineStore("permission", () => {
  const { get, put } = useApiClient();
  const baseUrl = "/admin/iam";
  const listData = ref<PermissionListResponse>({
    items: [],
    pagination: { total: 0, page: 1, page_size: 20, pages: 0 },
  });
  const isLoading = ref(false);
  const error = ref<string | null>(null);
  const roleSelection = ref<Record<string, string[]>>({});
  const roleInitialSelection = ref<Record<string, string[]>>({});

  const normalizedList = computed(() =>
    listData.value.items.map((permission) => {
      const meta = permission.meta || {};
      const code = `${permission.resource || ""}.${permission.action || ""}`.replace(/^\./, "");
      return {
        permission_uuid: permission.permission_uuid,
        name: meta.label || code,
        code,
        module: meta.module || permission.plugin || "",
        description: permission.description || "",
        type: meta.type || "action",
        apiEndpoint: meta.api_endpoint,
        httpMethod: meta.http_method,
        __raw: permission,
      };
    })
  );

  const unwrapList = (response: any): PermissionListResponse => {
    const payload = response?.data?.data ?? response?.data ?? response ?? {};
    const items = Array.isArray(payload?.items) ? payload.items : [];
    return {
      items,
      pagination: payload?.pagination || {
        total: items.length,
        page: 1,
        page_size: items.length || 1,
        pages: 1,
      },
    };
  };

  const fetchList = async (query: PermissionListQuery = {}) => {
    isLoading.value = true;
    error.value = null;
    try {
      listData.value = unwrapList(await get<any>(`${baseUrl}/permissions`, { params: query }));
      return listData.value;
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : "permission_catalog_request_failed";
      throw cause;
    } finally {
      isLoading.value = false;
    }
  };

  const fetchAllActive = () => fetchList({ size: 500, status: "active" });

  const fetchRolePermissionUUIDs = async (roleUUID: string) => {
    const response = await get<any>(`${baseUrl}/roles/${roleUUID}`);
    const payload = response?.data?.data ?? response?.data ?? response;
    const values = Array.isArray(payload?.permission_uuids)
      ? payload.permission_uuids.map((value: unknown) => String(value))
      : [];
    roleSelection.value[roleUUID] = values;
    roleInitialSelection.value[roleUUID] = [...values];
    return values;
  };

  const setRolePermissionUUIDs = async (roleUUID: string, tenantUUID: string, permissionUUIDs: string[]) => {
    if (!tenantUUID) throw new Error("tenant_uuid_required");
    const response = await put<any>(`${baseUrl}/roles/${roleUUID}/permissions`, {
      tenant_uuid: tenantUUID,
      permission_uuids: permissionUUIDs,
    });
    const payload = response?.data?.data ?? response?.data ?? response;
    const values = Array.isArray(payload?.permission_uuids)
      ? payload.permission_uuids.map((value: unknown) => String(value))
      : permissionUUIDs;
    roleSelection.value[roleUUID] = values;
    roleInitialSelection.value[roleUUID] = [...values];
    return payload;
  };

  return {
    listData: readonly(listData),
    isLoading: readonly(isLoading),
    error: readonly(error),
    normalizedList,
    roleSelection,
    roleInitialSelection,
    fetchList,
    fetchAllActive,
    fetchRolePermissionUUIDs,
    setRolePermissionUUIDs,
  };
});
