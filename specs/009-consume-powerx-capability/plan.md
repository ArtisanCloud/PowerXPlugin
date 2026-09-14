# Implementation Plan: PowerX 通用能力插件消费

**Branch**: `009-consume-powerx-capability` | **Date**: 2025-12-20 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/009-consume-powerx-capability/spec.md`

**Note**: 依据 `/speckit.plan` 流程撰写，后续 `/speckit.tasks` 将基于本文拆分任务。

## Summary

当前实施围绕 Framework contract → delegated client → 启动 Factory → 插件 local 注入 → 合同与安装验收推进。插件业务层不自行分流；本地 Store/事务由插件实现，Core 权威数据只通过正式 Host Contract 访问。

2026-09-08 消费者审计新增范围：Media Variant 传输、独立组织写 contract，以及 Registry/Gateway 逐操作授权映射。详见 [任务单](../../docs/contracts/framework-consumer-contract-gaps.md) 和 tasks Phase 15。现有 Media Variant 元数据方法不代表支持文件传输；IAM Directory 不承接组织写入或后台登录会话；已有 Registry/Gateway typed client 不等于授权全量验收。

唯一对外入口为 [业务模块接入指南](../../docs/guides/features/009-consume-powerx-capability/guide.md)，[双模式规范](../../docs/guides/develop/framework-dual-mode-business-modules.md)维护共同规则，[覆盖台账](../../docs/contracts/powerx-core-framework-coverage.md)维护状态。示例随接口编译验证，不替代真实安装授权验收。

## Technical Context

| 项目 | 说明 |
| --- | --- |
| Language/Version | Go 1.24 backend、TypeScript 5 + Nuxt 4.2（Node.js 20）frontend/scripts |
| Primary Dependencies | Core Host Contract/OpenAPI、`@artisan-cloud/plugin-framework-*`、px-plugin CLI、STS service actor、Gateway API Key（仅本地调试）、`scripts/capabilities` |
| Storage | 无新增持久层；仅依赖现有 manifest/config |
| Testing | Go `make test`、typed-client/handler contract tests；Nuxt `npm test`/Playwright；manifest required-capability 静态校验 |
| Target Platform | PowerX 插件 backend（Linux 宿主）+ web-admin（Nuxt SSR/static）+ Skeleton 本地环境 |
| Project Type | 全栈插件（backend + web-admin + scripts + skeleton） |
| Performance Goals | Host Contract 调用 P95 < 2s；Trace 覆盖 ≥99%；capability grant 拒绝可诊断 |
| Constraints | 通用调用仅通过 Integration Gateway；正式模块调用仅通过 Framework typed Host Contract；`delegated` 模式只使用宿主 STS service actor；本地 API Key 仅限精确 scope 的 Skeleton 调试；缺失凭证或 grant 必须显式失败；禁止访问宿主内部 API、请求注入 tenant 或 Mock/本地降级 |
| Scale/Scope | 需覆盖 >50 CoreX 能力 ID、百 TPS 调用，支持多租户/多插件部署 |

## Constitution Check

*GATE: Must pass before Phase 0 research. Phase 1 结束后复查。*

- ✅ **Host Contract First**：所有能力通过 Gateway/Registry 契约，无内部 API 依赖。
- ✅ **Tenant Isolation & Zero Trust**：Core 从 STS service actor 或 Gateway API Key 推导 tenant；Framework 保留 `X-Request-ID`/traceId，Skeleton 不接受调用方 tenant。
- ✅ **Service-Centric Architecture**：Gateway Client 位于服务/框架层，Handler/前端仅使用接口。
- ✅ **Observable & Testable Delivery**：设计包含统一日志、指标、限流事件、CLI 与 typed-contract 测试链路。
- ✅ **Minimal Footprint & Versioned Releases**：不增加新项目或持久层，沿用现有 Go/Nuxt 模板。

## Project Structure

### Documentation (this feature)

```text
specs/009-consume-powerx-capability/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
└── checklists/
```

### Source Code（历史目录方案，不作为当前源码路径）

```text
backend/
├── internal/
│   ├── integration/            # Gateway client、凭证注入扩展
│   ├── services/
│   └── transport/http/
├── cmd/
└── tests/

packages/
├── backend/                    # Go framework helpers（新增 Gateway 封装）
└── admin/                      # Nuxt 插件/Composable + runtimeConfig

skeleton/
├── backend/
├── web-admin/
└── scripts/.env templates

scripts/
└── capabilities/               # run-from-package.mjs & helpers

docs/
└── plan/009-consume-powerx-capability.md
```

**Structure Decision**: 采用现有 backend + packages + skeleton + scripts 的多项目结构，针对上述目录补充 Gateway client、凭证管理、Mock、CLI 与文档更新，无需新增仓库或子项目。

## Complexity Tracking

无额外宪章违例需求；若后续新增项目/依赖将单独提 RFC。

## Capability Contract Upgrade & Quota Strategy（历史规划，非交付承诺）

1. **契约版本守护**
   - 在 `scripts/capabilities` 目录新增契约摘要生成脚本（Phase 3/4 依赖），将 `contracts/powerx-capability-invoke.openapi.yaml` 等资产写入 `dist/capability-contracts.json` 并记录版本号。
   - `framework/backend/go/internal/integration/gateway/client.go` 加入契约版本检查 Hook，当检测到最新 Registry 版本与当前版本不一致时，向日志与 Admin UI 抛出提示，并提供 `PX_GATEWAY_CONTRACT_VERSION` 环境变量覆盖。
   - `docs/plan/009-consume-powerx-capability.md` 与 quickstart 补充“契约升级流程”说明，明确兼容窗口与提醒方式。

2. **限流/配额治理**
   - 利用 Capability Registry 中的 `rateLimit/quota` 字段，在 Gateway Client 初始化时加载默认策略，并允许运维通过 `px-plugin capabilities quota --manifest ./skeleton/plugin.yaml`（或新增命令）为租户/插件设置额度。
   - 在 `framework/backend/go/observability` 中暴露限流/配额指标（命中率、被拒绝次数），并在限流事件时写 `audit.capability.invocation.denied`。
   - 在 docs/operations 章节说明如何配置与监控限流/配额。

## Runtime Gateway Policy Alignment（历史 Tool Token 阶段，非当前装配说明）

1. **框架级 Host Capability Client**
   - 在 framework 层提供统一 client，内部完成 `detectRuntimeGatewayMode`、凭证策略分流、请求头注入、错误结构归一化。
   - 插件业务侧仅调用 client 接口，不再手写 `bearer/apikey` 分支。

2. **框架级 HTTP Guard**
   - 提供 `RequireCapabilityGateway` 中间件/辅助函数，统一输出 `503` + 诊断字段（`mode`、`required_credential`、`base_url_configured`）。
   - admin/integration 与通用 invoke 接口统一接入，避免重复判空和不一致报错。

3. **启动期配置校验**
   - 在 bootstrap 阶段执行 gateway preflight，明确打印运行模式、有效凭证来源、缺失配置。
   - 对冲突配置（如 local 模式仍注入 bearer）直接 fail-fast，避免运行期随机失败。

4. **宿主注入机制（安装/启用链路）**
   - PowerX 在插件进程启动环境中注入 `PX_GATEWAY_BASE_URL`、`PX_PLUGIN_TOOL_TOKEN`、`PX_GATEWAY_AUTH_SCHEME=bearer`。
   - PostEnable 执行“进程内凭证探活”（debug health check）；失败标记 `enable_failed_missing_gateway_credential`。
   - 默认 stub 下发路径不再视为成功路径，必须以插件进程可观测到变量并通过探活为准。

## 2026-09-03 实施增量：Core Host Contract Lab

本增量复用既有 Capability Lab、Framework typed client 与 Skeleton 后端代理，不新建第二套 Gateway client。它把“通用 capability 调试”和“正式 Core Host Contract 验收”明确分层：前者继续经 Capability Registry，后者通过具体模块的 Framework adapter。

| 层 | 职责 | 不允许的行为 |
| --- | --- | --- |
| `Capability Lab` | 目录发现、通用 capability invoke、协议元数据诊断 | 充当 IAM/Knowledge 等正式 Host DTO 的替代客户端 |
| `Host Contract Lab` | 强类型模块 probe、稳定错误映射、异步任务状态与 trace 展示 | 前端直连 Core、请求注入 tenant、静默 fallback |
| Framework typed client | STS/API-Key 出站、DTO 映射与错误归一化 | 读取 Core 数据库或调用未声明的内部路由 |

安装态 delegated 使用 STS service actor；Skeleton 的 API-Key 调试仅用于开发验证，不能替代安装态 capability grant 验收。任何 Host 写操作均采用单独的测试对象与显式确认，不作为默认 smoke test。
