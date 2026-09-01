import { useIAMService, type MemberRecord } from "./iamService";
import type { ApiResponse } from "../types/types";

export interface User {
  uuid: string;
  createdAt: string;
  updatedAt: string;
  email?: string;
  phone?: string;
  display_name: string;
  avatar_url?: string;
  status: number;
  meta?: Record<string, any>;
}

export interface Member {
  uuid: string;
  tenant_uuid: string;
  username: string;
  display_name: string;
  avatar_url?: string;
  status: number;
  meta?: Record<string, any>;
  createdAt: string;
  updatedAt: string;
}

export interface MemberWithProfile {
  Member: Member;
  User: User;
	DeptUUIDs: string[] | null;
  RoleCodes: string[];
}

export interface UserListParams {
  tenant_uuid: string;
  q?: string;
  page?: number;
  page_size?: number;
  status?: number | string;
}

type MemberListResponse = {
  items: MemberWithProfile[];
  pagination: {
    total: number;
    page: number;
    page_size: number;
    pages: number;
  };
};

const toMemberWithProfile = (record: MemberRecord): MemberWithProfile => {
	const memberUUID = record.member_uuid;
  const createdAt = (record as any).created_at || (record as any).createdAt || "";
  const updatedAt = (record as any).updated_at || (record as any).updatedAt || createdAt;
  const status =
    record.status === "disabled" || record.status === "locked" ? 0 : 1;
  return {
    Member: {
		uuid: memberUUID,
      tenant_uuid: record.tenant_uuid,
      username: record.username,
      display_name: record.display_name || record.email || record.username,
      avatar_url: (record as any).avatar_url,
      status,
      meta: {
		department_uuids: record.department_uuids,
        phone: record.phone,
      },
      createdAt,
      updatedAt,
    },
    User: {
		uuid: memberUUID,
      createdAt,
      updatedAt,
      email: record.email,
      phone: record.phone,
      display_name: record.display_name || record.email || record.username,
      avatar_url: (record as any).avatar_url,
      status,
      meta: {},
    },
		DeptUUIDs: Array.isArray(record.department_uuids) ? record.department_uuids : [],
    RoleCodes: Array.isArray(record.roles) ? record.roles : [],
  };
};

export const useUserService = () => {
  const iamService = useIAMService();

  const getUsers = async (
    params: UserListParams
  ): Promise<ApiResponse<MemberListResponse>> => {
    if (!params?.tenant_uuid) {
      throw new Error("需要提供租户 UUID");
    }
    const response = await iamService.listMembers({
      tenantUuid: params.tenant_uuid,
      query: params.q,
      status:
        typeof params.status === "number"
          ? params.status === 1
            ? "active"
            : "disabled"
          : params.status,
      page: params.page,
      pageSize: params.page_size,
    });
    const payload = (response as any)?.data ?? response ?? {};
    const list =
      payload?.data?.items ??
      payload?.items ??
      payload?.data ??
      payload ??
      [];
    const items = Array.isArray(list)
      ? (list as MemberRecord[]).map(toMemberWithProfile)
      : [];
    const total =
      payload?.data?.pagination?.total ??
      payload?.pagination?.total ??
      payload?.data?.total ??
      payload?.total ??
      items.length;
    const page = params.page ?? payload?.data?.pagination?.page ?? 1;
    const pageSize =
      params.page_size ??
      payload?.data?.pagination?.page_size ??
      payload?.pagination?.page_size ??
      (items.length || 1);
    const pages =
      payload?.data?.pagination?.pages ??
      payload?.pagination?.pages ??
      Math.max(1, Math.ceil(total / Math.max(1, pageSize)));

    return {
      code: (response as any)?.code ?? 200,
      message: (response as any)?.message ?? "",
      timestamp: Date.now(),
      data: {
        items,
        pagination: {
          total,
          page,
          page_size: pageSize,
          pages,
        },
      },
    };
  };

  const createSystemUser = async (data: Record<string, any>) => {
    const payload = {
      tenant_uuid: data.tenant_uuid,
      email: data.email,
      display_name: data.display_name ?? data.name,
      username: data.username,
      phone: data.phone,
		department_uuids: data.departmentUUIDs ?? [],
		role_uuids: data.roleUUIDs ?? [],
      status: typeof data.status === "number" ? (data.status === 1 ? "active" : "disabled") : data.status,
    };
    const response = await iamService.createMember(payload);
    return (response as any)?.data ?? response;
  };

	const updateUser = async (memberUUID: string, data: Record<string, any>) => {
    const payload = {
      email: data.email,
      display_name: data.display_name ?? data.name,
      username: data.username,
      phone: data.phone,
		department_uuids: data.departmentUUIDs ?? [],
		role_uuids: data.roleUUIDs ?? [],
      replace_roles: data.replaceRoles ?? true,
      status: typeof data.status === "number" ? (data.status === 1 ? "active" : "disabled") : data.status,
    };
		const response = await iamService.updateMember(memberUUID, payload);
    return (response as any)?.data ?? response;
  };

	const setUserStatus = async (memberUUID: string, data: { status: number }) => {
    const status = data.status === 1 ? "active" : "disabled";
		return updateUser(memberUUID, { status });
  };

	const deleteUser = async (memberUUID: string) => {
		await setUserStatus(memberUUID, { status: 0 });
    return { ok: true };
  };

  return {
    getUsers,
    createSystemUser,
    updateUser,
    deleteUser,
    setUserStatus,
    addUserToTenant: async () => ({ member_uuid: "" }),
    forceLogout: async () => ({ ok: true }),
  };
};
