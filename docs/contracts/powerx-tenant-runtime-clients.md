# PowerX tenant runtime clients

本文件记录 Framework 面向 PowerX Core tenant OpenAPI 的稳定客户端边界。它们只在 delegated 模式装配，并共享 Skeleton 已建立的 STS service-token provider；本地模式不构造伪造的宿主客户端。

| Core 开放面 | Framework 客户端 | 认证 | 固定 DTO 边界 |
|---|---|---|---|
| `GET /tenant/capabilities` | `runtime/powerx/capability.Client.List` | tenant-scoped STS service token | `Capability`、`Protocol` |
| `GET /tenant/capabilities/resolve` | `runtime/powerx/capability.Client.Resolve` | tenant-scoped STS service token | `ResolveResult` |
| `POST /tenant/invocations` | `runtime/powerx/capability.Client.Invoke` | tenant-scoped STS service token | `InvokeInput`、`InvokeResult` |
| `GET /tenant/invocations/{traceId}` | `runtime/powerx/capability.Client.GetInvocation` | tenant-scoped STS service token | `Invocation` |
| `POST /tenant/skills/invoke` | `runtime/powerx/skills.Client.Invoke` | tenant-scoped STS service token | `InvokeInput`、`InvokeOutput` |
| `POST /tenant/notifications` | `runtime/powerx/notifications.Client.Create` | tenant-scoped STS service token | `CreateInput`、`Notification` |
| `POST /openapi/knowledge-spaces/qa/retrieval-plan` | `runtime/powerx/knowledge.Client.RetrievalPlan` | tenant-scoped STS service token | `RetrievalPlanInput`、`RetrievalPlan` |
| `POST /openapi/knowledge-spaces/qa/memory-snapshot` | `runtime/powerx/knowledge.Client.UpsertMemorySnapshot` | tenant-scoped STS service token | `MemorySnapshotInput`、`MemorySnapshot` |

`payload`、`context` 和 `result` 是各 capability/skill 版本自行声明的 JSON 对象，Framework 只负责保真传输，不能将其转换为自由文本或在失败时静默降级。所有 HTTP 非 2xx 响应均保留为对应客户端的 `HTTPError`，调用方应按 Core 的稳定错误码处理。
