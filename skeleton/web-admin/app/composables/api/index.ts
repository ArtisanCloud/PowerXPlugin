// 统一导出所有 API 相关功能

export * from "./useTemplate";
export * from "./useStream";
export {
  useApiClient,
  apiGet,
  apiPost,
  apiPut,
  apiPatch,
  apiDel,
} from "./_client";
export { resolveApiBase, getAuthToken, getTenantId } from "./_base";
export type { Page, ApiResponse, ListQuery } from "./_base";
