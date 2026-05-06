import { useApiClient } from "../_client";
import type { ApiResponse } from "../types/types";
import { useAuth } from "~/composables/useAuth";

/**
 * 用户上下文相关类型定义
 */
export interface UserContextData {
  is_root: boolean;
  current_tenant_uuid: string;
  current_member_id?: number | null;
  user: ContextUser;
  members: ContextMember[];
  roles?: string[];
  permissions?: string[];
  capabilities?: UserContextCapabilities;
}

export interface UserContextCapabilities {
  templates?: {
    can_create?: boolean;
    can_update?: boolean;
    can_delete?: boolean;
  };
}

export interface ContextUser {
  id: number;
  email: string;
  phone: string;
  display_name: string;
  avatar_url: string;
  status: number; // 1=active, 0=disabled
}

export interface ContextMember {
  tenant_uuid: string;
  tenant_name: string;
  member_id: number;
  is_admin: boolean;
}

/**
 * 用户上下文服务 API
 */
export const useMeService = () => {
  const apiClient = useApiClient();
  const auth = useAuth();
  const runtimeConfig = useRuntimeConfig();
  const delegatedMode = (() => {
    const raw = String(runtimeConfig.public?.iamMode || "").trim().toLowerCase();
    if (raw) return raw === "delegated";
    return Boolean(runtimeConfig.public?.delegatedMode || runtimeConfig.public?.insidePowerX);
  })();
  const contextBaseCandidates = delegatedMode
    ? ["/admin/user/auth"]
    : ["/admin/user/auth"];

  const withAuthToken = async (options: Record<string, any> = {}) => {
    if (!process.client) return options;
    const refreshed = await auth.ensureFreshToken?.();
    const token = String(refreshed || auth.getToken?.() || "").trim();
    if (!token) {
      const err: any = new Error("登录状态缺失，请重新登录");
      err.status = 401;
      err.statusCode = 401;
      throw err;
    }
    return {
      ...options,
      token,
    };
  };

  const requestWithFallback = async <T>(
    method: "get" | "post" | "put",
    suffix: string,
    body?: any
  ): Promise<T> => {
    let lastError: any = null;
    for (const base of contextBaseCandidates) {
      try {
        if (method === "get") {
          return await apiClient.get<T>(`${base}${suffix}`, await withAuthToken());
        }
        if (method === "post") {
          return await apiClient.post<T>(`${base}${suffix}`, body, await withAuthToken());
        }
        return await apiClient.put<T>(`${base}${suffix}`, body, await withAuthToken());
      } catch (error: any) {
        lastError = error;
        const status = Number(
          error?.statusCode || error?.status || error?.response?.status || 0
        );
        if (
          status === 404 ||
          (delegatedMode && suffix === "/me/context" && status === 403)
        ) {
          continue;
        }
        throw error;
      }
    }
    throw lastError || new Error(`未找到可用的上下文接口: ${suffix}`);
  };

  const adminBaseUrl = "/admin/user/auth";

  return {
    /**
     * 获取当前用户的上下文信息
     * 包含用户基本信息、当前租户、所属成员信息等
     */
    getMyContext: () => {
      return requestWithFallback<ApiResponse<UserContextData>>(
        "get",
        "/me/context"
      );
    },

    /**
     * 切换当前租户上下文
     * @param tenantUuid 要切换到的租户 UUID
     */
    switchTenant: (tenantUuid: string) => {
      return requestWithFallback<ApiResponse<UserContextData>>(
        "post",
        "/me/switch-tenant",
        { tenant_uuid: tenantUuid }
      );
    },

    /**
     * 获取我的租户列表（简化版，只返回基本信息）
     */
    getMyTenants: () => {
      return requestWithFallback<ApiResponse<ContextMember[]>>(
        "get",
        "/me/tenants"
      );
    },

    /**
     * 更新用户资料
     * @param data 要更新的用户信息
     */
    updateMyProfile: (data: Partial<Omit<ContextUser, "id" | "status">>) => {
      return apiClient.put<ApiResponse<ContextUser>>(
        `${adminBaseUrl}/me/profile`,
        data
      );
    },

    /**
     * 上传用户头像
     * @param file 头像文件
     */
    uploadAvatar: (file: File) => {
      const formData = new FormData();
      formData.append("avatar", file);

      return apiClient.post<ApiResponse<{ avatar_url: string }>>(
        `${adminBaseUrl}/me/avatar`,
        formData,
        {
          headers: {
            "Content-Type": "multipart/form-data",
          },
        }
      );
    },

    /**
     * 检查用户权限
     * @param permission 权限代码
     * @param resource 资源标识（可选）
     */
    checkPermission: (permission: string, resource?: string) => {
      return apiClient.post<ApiResponse<{ has_permission: boolean }>>(
        `${adminBaseUrl}/me/check-permission`,
        {
          permission,
          resource,
        }
      );
    },

    /**
     * 获取用户在当前租户下的角色信息
     */
    getMyRoles: () => {
      return apiClient.get<
        ApiResponse<
          Array<{
            id: number;
            name: string;
            code: string;
            permissions: string[];
          }>
        >
      >(`${adminBaseUrl}/me/roles`);
    },

    /**
     * 获取用户在当前租户下的部门信息
     */
    getMyDepartments: () => {
      return apiClient.get<
        ApiResponse<
          Array<{
            id: number;
            name: string;
            code: string;
            parent_id?: number;
          }>
        >
      >(`${adminBaseUrl}/me/departments`);
    },
  };
};

/**
 * 用户上下文 Composable
 * 提供响应式的用户上下文状态管理
 */
export const useUserContext = () => {
  const meService = useMeService();

  // 响应式状态
  const userContext = ref<UserContextData | null>(null);
  const isLoading = ref(false);
  const error = ref<string | null>(null);

  // 计算属性
  const isRoot = computed(() => userContext.value?.is_root ?? false);
  const currentUser = computed(() => userContext.value?.user ?? null);
  const currentTenantUuid = computed(
    () => userContext.value?.current_tenant_uuid || null
  );
  const currentMemberId = computed(
    () => userContext.value?.current_member_id ?? null
  );
  const myTenants = computed(() => userContext.value?.members ?? []);

  // 当前租户信息
  const currentTenant = computed(() => {
    if (!userContext.value?.current_tenant_uuid) return null;
    return (
      myTenants.value.find(
        (m) => m.tenant_uuid === userContext.value!.current_tenant_uuid
      ) ?? null
    );
  });

  // 是否为当前租户的管理员
  const isCurrentTenantAdmin = computed(() => {
    return currentTenant.value?.is_admin ?? false;
  });

  /**
   * 加载用户上下文
   */
  const loadUserContext = async () => {
    isLoading.value = true;
    error.value = null;

    try {
      const response = await meService.getMyContext();
      userContext.value = response.data;
    } catch (err: any) {
      error.value = err.message || "加载用户上下文失败";
      console.error("Failed to load user context:", err);
    } finally {
      isLoading.value = false;
    }
  };

  /**
   * 切换租户
   */
  const switchTenant = async (tenantUuid: string) => {
    if (tenantUuid === currentTenantUuid.value) return;

    isLoading.value = true;
    error.value = null;

    try {
      const response = await meService.switchTenant(tenantUuid);
      userContext.value = response.data;

      // 可以在这里触发页面刷新或路由跳转
      await navigateTo("/dashboard");
    } catch (err: any) {
      error.value = err.message || "切换租户失败";
      console.error("Failed to switch tenant:", err);
    } finally {
      isLoading.value = false;
    }
  };

  /**
   * 更新用户资料
   */
  const updateProfile = async (
    data: Partial<Omit<ContextUser, "id" | "status">>
  ) => {
    isLoading.value = true;
    error.value = null;

    try {
      const response = await meService.updateMyProfile(data);
      if (userContext.value) {
        userContext.value.user = response.data;
      }
    } catch (err: any) {
      error.value = err.message || "更新资料失败";
      console.error("Failed to update profile:", err);
      throw err;
    } finally {
      isLoading.value = false;
    }
  };

  /**
   * 上传头像
   */
  const uploadAvatar = async (file: File) => {
    isLoading.value = true;
    error.value = null;

    try {
      const response = await meService.uploadAvatar(file);
      if (userContext.value) {
        userContext.value.user.avatar_url = response.data.avatar_url;
      }
      return response.data.avatar_url;
    } catch (err: any) {
      error.value = err.message || "上传头像失败";
      console.error("Failed to upload avatar:", err);
      throw err;
    } finally {
      isLoading.value = false;
    }
  };

  /**
   * 检查权限
   */
  const hasPermission = async (permission: string, resource?: string) => {
    try {
      const response = await meService.checkPermission(permission, resource);
      return response.data.has_permission;
    } catch (err) {
      console.error("Failed to check permission:", err);
      return false;
    }
  };

  /**
   * 清空用户上下文（用于登出）
   */
  const clearUserContext = () => {
    userContext.value = null;
    error.value = null;
  };

  return {
    // 状态
    userContext: readonly(userContext),
    isLoading: readonly(isLoading),
    error: readonly(error),

    // 计算属性
    isRoot,
    currentUser,
    currentTenantUuid,
    currentMemberId,
    myTenants,
    currentTenant,
    isCurrentTenantAdmin,

    // 方法
    loadUserContext,
    switchTenant,
    updateProfile,
    uploadAvatar,
    hasPermission,
    clearUserContext,
  };
};
