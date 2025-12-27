# Feature Specification: PowerX 通用能力插件消费

**Feature Branch**: `009-consume-powerx-capability`  
**Created**: 2025-12-20  
**Status**: Draft  
**Input**: User description: "Spec for plugin consumption of PowerX open capabilities per docs/plan/009-consume-powerx-capability.md"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 宿主插件统一调用核心能力 (Priority: P1)

宿主部署的插件开发者在 Capability Registry 中申领 `source=corex` 能力，框架在后端自动注入 Gateway Client，并通过统一的服务层/HTTP 接口向前端或业务 handler 暴露调用入口；所有请求都由插件后端以清单中的能力 ID 向 `/tenant/invocations` 或 `IntegrationGatewayTenantService` 发起，开发者只需提供业务参数即可获得 Media、事件、Scheduler、Workflow 等服务。

**Why this priority**: 宿主环境是绝大多数线上流量的入口，若无法快速复用核心能力，将继续依赖内部 API，违背开放计划目标。

**Independent Test**: 仅实现宿主端 Gateway Client 与 manifest 校验，即可通过调用 `com.corex.media.assets.manage` 验证真实上传/预签名流程，直接体现价值。

**Acceptance Scenarios**:

1. **Given** 插件 manifest 已声明 `com.corex.media.assets.manage`，**When** 开发者在宿主环境通过插件后端暴露的服务层/API 触发该能力，**Then** Gateway 根据能力 ID 正确路由请求并返回 MediaX 的上传凭证。
2. **Given** 能力调用发生错误或被限流，**When** Gateway 返回错误码与 `traceId`，**Then** 插件后端框架必须将 `traceId` 写入日志并向业务抛出统一的错误对象，同时透传给前端调用者。

---

### User Story 2 - Skeleton 模式复用同一封装 (Priority: P2)

Skeleton 本地开发者通过 `px-plugin login` 获取 Tool Token，把 `PX_GATEWAY_BASE_URL`、`PX_TOOL_TOKEN` 写入 `.env.local`，并使用框架内置的 Go Client 在 Skeleton 后端发起远程调用（前端通过插件后端提供的 API 间接访问 Gateway）或在 Gateway 不可用时切换到 Mock，实现与宿主一致的行为以便预先验证调用链与权限配置。

**Why this priority**: Skeleton 是插件开发调试的默认入口，若无法直连 PowerX 能力，将导致环境差异和额外搬运成本。

**Independent Test**: 只需在本地运行 Skeleton 服务，通过插件后端提供的 `capability invoke` API 向 Dev Gateway 拉取媒资列表，即可独立验证凭证、环境配置和 Mock 回退逻辑。

**Acceptance Scenarios**:

1. **Given** 开发者完成本地登录并写入 `.env.local`，**When** 运行 Skeleton 服务并通过插件后端公开的 REST/gRPC API 请求 `com.corex.media.assets.read` 的列表能力，**Then** 能够在不修改业务代码的情况下获得真实媒资列表。
2. **Given** Dev Gateway 暂不可达，**When** 开发者启动带 `--use-mock=media` 的 Skeleton 服务，**Then** 框架会自动切换到内存 Mock 并提示本次调用未连接到真实 PowerX。

---

### User Story 3 - 能力调用治理与观测 (Priority: P3)

平台运维或 SRE 需要在统一的日志与指标中看到插件调用 PowerX 能力的 `capabilityId`、`tenantUUID`、`traceId`、限流与配额使用情况，发生异常时可快速定位调用链并触发告警或熔断，保障宿主与 Skeleton 两种模式的调用一致受控。

**Why this priority**: 无治理就无法开放，更难排查跨插件的问题；治理指标直接影响上线审批与配额管理。

**Independent Test**: 在沙箱环境中模拟多插件调用和速率突增，验证框架自动写入指标、触发限流并将异常事件推送到审计通道即可验证该故事。

**Acceptance Scenarios**:

1. **Given** 插件调用 `com.corex.eventfabric.publish`，**When** Gateway 返回成功响应，**Then** 观测系统可查到一条包含能力 ID、租户、traceId、耗时的记录。
2. **Given** 插件在 1 分钟内连续调用超过额度，**When** 限流器生效，**Then** Gateway 返回标准化错误并触发框架记录 `rateLimitExceeded` 事件供运维订阅。
3. **Given** Skeleton 开发者在 web-admin 的 Capability Lab 页面填写 `capabilityId/action/payload`，**When** 点击 Invoke，**Then** 插件后端收到请求并调用 Gateway 或 Mock，页面需展示响应/TraceId/耗时，并在后端返回 `warnings`（如契约版本过期或 Mock 提示）时显式告警。

### Edge Cases

- 当 manifest 未声明任何 `source=corex` 能力却尝试调用 Gateway 时，框架必须阻止请求并提示缺失的能力 ID。
- 当 Tool Token 已过期或缺失 `X-Tenant-UUID`，系统需阻止调用并提供刷新指引，避免出现匿名请求。
- 当 Gateway 网络不可达时，应在 2 秒内超时并回落至 Mock（若已开启），否则向用户明确提示“无法连接底座”。
- 当同一能力在 Skeleton 与宿主使用不同的 `PX_GATEWAY_BASE_URL` 时，配置对比工具需提示差异以避免环境偏差。
- 当插件调用结果体积过大（>50MB）或媒资路径受限时，框架需要自动启用分片或预签名路径，并提示开发者后续处理方式。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 插件 manifest 与 `skeleton/plugin.yaml` 必须支持声明 `requiredCapabilities`，并在 CI 中通过 `px-plugin capabilities plan|apply --manifest ./skeleton/plugin.yaml` 进行校验，未声明即调用时需阻断。
- **FR-002**: 框架需提供宿主与 Skeleton 共享的 Gateway Client（Go SDK），默认读取 `PX_GATEWAY_BASE_URL`、`PX_PLUGIN_TOOL_TOKEN`、`X-Tenant-UUID` 等环境变量，并通过插件后端对前端暴露统一 API（禁止前端直连 Gateway）。
- **FR-003**: 在宿主模式，部署系统需自动注入 Tool Token；在 Skeleton 模式，开发者通过 `px-plugin login` 与脚本生成本地凭证，并由框架自动刷新或提醒即将过期。
- **FR-004**: 所有调用必须由插件后端统一走 `/tenant/invocations` REST 或 `IntegrationGatewayTenantService.InvokeCapability` gRPC，框架不得允许前端或其他组件直接访问底座内部 API。
- **FR-005**: Gateway Client 需对每次调用自动附带 `capabilityId`、`action`、`payload`、`X-Request-ID`，并把 Gateway 返回的 `traceId` 注入日志与指标。
- **FR-006**: 当 Gateway 返回限流、鉴权失败或 5xx 错误时，框架需提供标准化错误对象，包含能力 ID、traceId、错误类别，方便业务捕获与重试。
- **FR-007**: Skeleton 模式必须支持 `--use-mock=<module>` 参数，将指定能力映射到内存实现，并清晰标记响应来源。
- **FR-008**: 提供 `scripts/capabilities/run-from-package.mjs --manifest ./skeleton/plugin.yaml --cap <capabilityId>` 命令，在宿主/本地模式均可复用，输出完整的请求/响应日志。
- **FR-009**: 观测体系需记录调用耗时、成功率、限流事件，并允许按 `capabilityId`、租户、插件版本进行聚合查询，确保 FR-005 数据有处可查。
- **FR-010**: 当能力契约（OpenAPI/Proto）升级时，框架需提供版本提示与向后兼容策略，至少支持一次向后兼容窗口并提醒开发者更新依赖。
- **FR-011**: 文档与教程需覆盖如何在宿主与 Skeleton 环境配置凭证、调用示例、错误排查以及能力速查表，保证新人在 1 天内可完成首次调用。
- **FR-012**: 平台需提供限流/配额管理接口，允许管理员针对插件或租户设置 `rateLimit`、`quota`，并确保调用链实时尊重这些配置。
- **FR-013**: Skeleton web-admin 必须提供“Capability Lab” 调试页面，覆盖 Capability 选择、Action/Payload 编辑、请求预览、响应/Trace 可视化、Mock 切换与告警提示，仅 `IsRoot` 或系统管理员可访问，便于本地验证 PowerX 能力且避免普通用户误用。
- **FR-014**: `/tenant/invocations` REST 调用需要显式传入 `preferred_protocol + method + endpoint` 等字段；插件后端在 API 层要校验并拒绝缺失字段，Capability Lab/文档需给出模板或构建器提示，防止开发者误以为 `action` 可以自动拼接 URL。
- **FR-015**: Skeleton `/api/v1/admin/capabilities` 在未指定 `source` 时返回本地 manifest，而当 `source=corex` 时必须通过 Gateway 调用 PowerX `/tenant/capabilities` 并把底座能力转换为 CatalogEntry 数据，供 Capability Lab、CLI 与文档使用。

### Key Entities *(include if feature involves data)*

- **Capability Registry Entry**：包含 `capabilityId`、来源（corex/plugin）、描述、协议入口与限流/配额策略，是 manifest 与授权的依据。
- **Tool Grant Token**：承载租户、插件、权限范围与过期时间的凭证，由宿主或 `px-plugin login` 生成，是调用 Gateway 的唯一凭证。
- **Invocation Request**：框架封装的请求对象，至少含 `capabilityId`、`action`、`payload`、`tenantUUID`、`requestId`，用于发送到 Gateway。
- **Invocation Telemetry**：观测与审计记录，包括 traceId、耗时、结果、错误码、限流状态，为治理与排障提供数据。
- **Mock Capability Adapter**：Skeleton 模式下的内存实现，保存预设响应或录制数据，用于 Gateway 不可达时的回放。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% 的插件能力调用必须通过 Gateway Client 完成，且开发者在 30 分钟内即可完成首个核心能力调用（含宿主和 Skeleton 指南）。
- **SC-002**: Skeleton 环境调用真实 Gateway 的成功率 ≥95%，若降级至 Mock 需在 5 秒内提示，并记录降级原因。
- **SC-003**: 所有核心能力调用都写入观测指标，Trace 覆盖率达到 99%，且限流告警可在 1 分钟内到达运维群。
- **SC-004**: 能力速查与 CLI 校验使 manifest 申领错误率低于 2%，能力契约升级后 3 个工作日内完成 100% 插件的兼容性验证。

## Assumptions

- Integration Gateway 已在宿主环境提供统一的 HTTP/gRPC 接口，并遵守 `IntegrationGatewayTenantService` 契约。
- 插件框架可在宿主部署时获取环境变量并安全注入到后端/前端进程。
- Skeleton 模式允许开发者访问 Dev Gateway，且 `px-plugin login` 能获取可调用核心能力的临时凭证。
- 平台观测系统已具备聚合能力，新增指标/日志只需提供结构化字段即可接入。

## Manifest / Docs Consistency（2025-12-22）

- `skeleton/plugin.yaml` → `capabilities.required` 默认示例保持与 Quickstart/Plan 中一致的 CoreX 能力（`com.corex.media.assets.manage`、`com.corex.eventfabric.publish`），`capabilities.provides` 指向 `contracts/capabilities/com.powerx.plugins.base.template.*`，方便 docs/plan/009 引用。
- `docs/plan/009-consume-powerx-capability.md` 与本 spec 均引用同一套环境变量（`PX_GATEWAY_BASE_URL/PX_PLUGIN_TOOL_TOKEN/PX_TENANT_UUID/NUXT_PUBLIC_POWERX_*`），并对应 manifest 注释，确保读者可在三个入口间互相对照。
