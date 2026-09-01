import { useApiClient } from "../_client";
import { getTenantUuid } from "../_base";
import type { ApiResponse } from "../types/types";
import { resolveTenantUUIDForRequest } from "~/utils/tenant-context";

export interface Department {
	 uuid: string;
  tenant_uuid: string;
  name: string;
  code: string;
  key?: string;
	 parent_department_uuid?: string | null;
  description?: string;
  path?: string;
  sort?: number;
  sort_order?: number;
  status?: number;
  meta?: any;
  children?: Department[];
}

export interface DepartmentCreateParams {
	name: string;
	key?: string;
	parent_department_uuid?: string | null;
  description?: string;
  sort?: number;
  status?: number;
  meta?: any;
}

export type DepartmentUpdateParams = {
  name?: string;
  key?: string;
	parent_department_uuid?: string | null;
  sort?: number;
  description?: string;
  status?: number;
  meta?: any;
};

const STORAGE_KEYS = ["px_current_tenant_uuid", "tenant_uuid"];

const ensureTenantUuid = (): string => {
  const fromResolver = resolveTenantUUIDForRequest();
  if (fromResolver) return fromResolver;

  const fromCookie = getTenantUuid();
  if (fromCookie) return fromCookie;

  if (process.client) {
    for (const key of STORAGE_KEYS) {
      const value = window.localStorage?.getItem(key);
      if (value && value.trim()) {
        return value.trim();
      }
    }
  }

  throw new Error("缺少租户上下文，请先选择租户或重新登录");
};

const buildTenantQuery = () => ({ tenant_uuid: ensureTenantUuid() });

const slugify = (value: string) =>
  value
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "") || `dept-${Date.now()}`;

const unwrapList = (resp: any): any[] => {
  if (!resp) return [];
  const direct = resp?.data ?? resp;
  if (Array.isArray(direct)) return direct;
  if (Array.isArray(direct?.items)) return direct.items;
  if (Array.isArray(direct?.data)) return direct.data;
  if (Array.isArray(direct?.list)) return direct.list;
  return [];
};

const normalizeDepartmentEntry = (dept: any): Department => {
  const childNodes = Array.isArray(dept?.children)
    ? dept.children.map(normalizeDepartmentEntry)
    : undefined;
	const parentCandidate = dept?.parent_department_uuid;
  const sortCandidate =
    dept?.sort ?? dept?.sort_order ?? dept?.sortOrder ?? undefined;

  const normalized: Department = {
		uuid: String(dept?.uuid ?? ""),
    tenant_uuid: String(
      dept?.tenant_uuid || dept?.tenantUuid || dept?.tenantUUID || ""
    ),
    name: String(dept?.name ?? ""),
    code: String(dept?.code || dept?.key || ""),
    key: dept?.code || dept?.key || undefined,
		parent_department_uuid: parentCandidate === null ? null : (typeof parentCandidate === "string" ? parentCandidate : undefined),
    description: dept?.description,
    path: typeof dept?.path === "string" ? dept.path : undefined,
		sort_order: typeof sortCandidate === "number" ? sortCandidate : undefined,
		sort: typeof sortCandidate === "number" ? sortCandidate : undefined,
    status:
      dept?.status === null || dept?.status === undefined
        ? undefined
        : Number.isFinite(Number(dept.status))
        ? Number(dept.status)
        : undefined,
    meta: dept?.meta ?? null,
    children: childNodes,
  };

  if (!normalized.code) {
		normalized.code = slugify(normalized.name || "department");
  }
  if (!normalized.key) {
    normalized.key = normalized.code;
  }

  return normalized;
};

const normalizeDepartmentTree = (departments: any[]): Department[] => {
  return departments
    .filter((dept) => dept && typeof dept === "object")
    .map((dept) => normalizeDepartmentEntry(dept));
};

const parseDepartmentsFromResponse = (resp: any): Department[] => {
  const list = unwrapList(resp);
  return normalizeDepartmentTree(list);
};

const extractDepartmentFromResponse = (resp: any): Department | null => {
  const direct = resp?.data ?? resp;
  const payload =
    direct && typeof direct === "object" && !Array.isArray(direct?.data)
      ? direct
      : direct?.data;
  if (payload && typeof payload === "object" && !Array.isArray(payload)) {
    return normalizeDepartmentEntry(payload);
  }
  return null;
};

const buildCreatePayload = (
  params: DepartmentCreateParams,
  tenantUuid: string
) => {
  const payload: Record<string, any> = {
    tenant_uuid: tenantUuid,
    name: params.name?.trim(),
    code: params.key?.trim() || slugify(params.name || ""),
  };
	if (params.parent_department_uuid !== undefined) {
		payload.parent_department_uuid = params.parent_department_uuid;
  }
  if (params.description) {
    payload.description = params.description;
  }
  if (typeof params.sort === "number") {
    payload.sort_order = params.sort;
  }
  return payload;
};

const buildUpdatePayload = (params: DepartmentUpdateParams) => {
  const payload: Record<string, any> = {};
  if (params.name) {
    payload.name = params.name.trim();
  }
  if (typeof params.sort === "number") {
    payload.sort_order = params.sort;
  }
  if (params.description) {
    payload.description = params.description;
  }
	if (Object.prototype.hasOwnProperty.call(params, "parent_department_uuid")) {
		payload.parent_department_uuid = params.parent_department_uuid;
  }
  return payload;
};

export function useDepartmentService() {
  const apiClient = useApiClient();
  const baseUrl = "/admin/iam/departments";

  const fetchAll = async () => {
    const res = await apiClient.get<ApiResponse<any>>(baseUrl, {
      params: buildTenantQuery(),
    });
    return parseDepartmentsFromResponse(res);
  };

  return {
    getDepartmentTree: async (): Promise<Department[]> => {
      const query = { params: buildTenantQuery() };
		const res = await apiClient.get<ApiResponse<any>>(`${baseUrl}/tree`, query);
      return parseDepartmentsFromResponse(res);
    },

    createDepartment: async (
      data: DepartmentCreateParams
    ): Promise<Department | null> => {
      const tenantUuid = ensureTenantUuid();
      const payload = buildCreatePayload(data, tenantUuid);
      const res = await apiClient.post<ApiResponse<any>>(baseUrl, payload);
      return extractDepartmentFromResponse(res);
    },

    updateDepartment: async (
		departmentUUID: string,
      data: DepartmentUpdateParams
    ): Promise<Department | null> => {
      const payload = buildUpdatePayload(data);
      const res = await apiClient.patch<ApiResponse<any>>(
		`${baseUrl}/${departmentUUID}`,
        payload
      );
      return extractDepartmentFromResponse(res);
    },

	deleteDepartment: async (departmentUUID: string): Promise<boolean> => {
		await apiClient.delete<ApiResponse<null>>(`${baseUrl}/${departmentUUID}`);
      return true;
    },

	getDepartment: async (departmentUUID: string): Promise<Department | null> => {
		const list = await fetchAll();
		return list.find((dept) => dept.uuid === departmentUUID) ?? null;
    },
  };
}
