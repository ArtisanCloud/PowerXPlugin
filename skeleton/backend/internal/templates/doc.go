// Package templates 提供 Skeleton 模板域的占位实现。
//
// 该包的结构需遵循 `.specify/memory/constitution.md` 中对仓储/服务/传输层的要求：
//   1. 仓储必须内嵌 repository.BaseRepository[T]，并在 BeginTenantTx 中执行 SET LOCAL app.tenant_id；
//   2. Service 层负责业务编排，HTTP/gRPC Handler 保持薄层；
//   3. 所有示例实现以内存为主，但接口需与真实多租户数据库保持一致，便于后续替换。
//
// 后续任务会在该包下补充 model、repository、service、handler 等文件。
package templates
