// Package router 提供最小化的 HTTP 路由能力，包括：
//   - Path 参数解析（如 /templates/:id）
//   - 统一响应助手（RespondSuccess/RespondError）
//   - 与 bootstrap.App 集成的 AttachHTTPServer 注册流程
//
// 业务侧可通过 router.RegisterPluginRoutes + 注册的 middleware.RequestID/TenantContext
// 获得与 Base 插件一致的运行期行为。
package router
