# Implementation Plan: PowerX 通用能力插件消费

**Branch**: `009-consume-powerx-capability` | **Date**: 2025-12-20 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/009-consume-powerx-capability/spec.md`

**Note**: 依据 `/speckit.plan` 流程撰写，后续 `/speckit.tasks` 将基于本文拆分任务。

## Summary

插件在宿主与 Skeleton 模式需要统一消费 PowerX 核心能力。我们将在 framework/backend/frontend/脚本层封装 Integration Gateway 调用（REST/gRPC）、Tool Grant 凭证、Mock 降级与观测，配合文档/CLI 引导 manifest 申领与速查，并确保调用链满足零信任、Tenant UUID、限流与 trace 要求。

关键交付：
- Gateway Client + 前后端封装：统一 `/tenant/invocations`/gRPC 调用、凭证注入、错误/trace 处理；
- Skeleton dev 体验：`px-plugin login`、`.env.local`、Mock 降级、CLI 辅助工具；
- 契约升级守护：生成能力契约版本摘要、比对并提醒插件开发者在兼容窗口内升级；
- 限流/配额治理：暴露配置入口、命令及观测指标，确保调用链尊重 Registry/租户额度。

## Technical Context

| 项目 | 说明 |
| --- | --- |
| Language/Version | Go 1.24 backend、TypeScript 5 + Nuxt 4.2（Node.js 20）frontend/scripts |
| Primary Dependencies | Integration Gateway `/tenant/invocations` & gRPC、`@artisan-cloud/plugin-framework-*`、px-plugin CLI、STS/Tool Grant、`scripts/capabilities` |
| Storage | 无新增持久层；仅依赖现有 manifest/config，Mock 数据存内存 |
| Testing | Go `make test`、integration stub；Nuxt `npm test`/Playwright；`px-plugin capabilities plan|apply`；`scripts/capabilities/run-from-package.mjs` |
| Target Platform | PowerX 插件 backend（Linux 宿主）+ web-admin（Nuxt SSR/static）+ Skeleton 本地环境 |
| Project Type | 全栈插件（backend + web-admin + scripts + skeleton） |
| Performance Goals | Gateway 调用 P95 < 2s；降级检测 < 5s；Trace 覆盖 ≥99%；能力申领失败率 <2% |
| Constraints | 仅允许通过 Integration Gateway；强制 Tool Token + `X-Tenant-UUID`；Skeleton Mock 必须提示；禁止访问宿主内部 API；契约升级需保留兼容窗口并发布提示 |
| Scale/Scope | 需覆盖 >50 CoreX 能力 ID、百 TPS 调用，支持多租户/多插件部署 |

## Constitution Check

*GATE: Must pass before Phase 0 research. Phase 1 结束后复查。*

- ✅ **Host Contract First**：所有能力通过 Gateway/Registry 契约，无内部 API 依赖。
- ✅ **Tenant Isolation & Zero Trust**：框架注入 `tenant_uuid`、Tool Token、`X-Request-ID`/traceId，Skeleton 亦遵循。
- ✅ **Service-Centric Architecture**：Gateway Client 位于服务/框架层，Handler/前端仅使用接口。
- ✅ **Observable & Testable Delivery**：设计包含统一日志、指标、限流事件、CLI/Mock 测试链路。
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

### Source Code (repository root)

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

## Capability Contract Upgrade & Quota Strategy

1. **契约版本守护**
   - 在 `scripts/capabilities` 目录新增契约摘要生成脚本（Phase 3/4 依赖），将 `contracts/powerx-capability-invoke.openapi.yaml` 等资产写入 `dist/capability-contracts.json` 并记录版本号。
   - `framework/backend/go/internal/integration/gateway/client.go` 加入契约版本检查 Hook，当检测到最新 Registry 版本与当前版本不一致时，向日志与 Admin UI 抛出提示，并提供 `PX_GATEWAY_CONTRACT_VERSION` 环境变量覆盖。
   - `docs/plan/009-consume-powerx-capability.md` 与 quickstart 补充“契约升级流程”说明，明确兼容窗口与提醒方式。

2. **限流/配额治理**
   - 利用 Capability Registry 中的 `rateLimit/quota` 字段，在 Gateway Client 初始化时加载默认策略，并允许运维通过 `px-plugin capabilities quota --manifest ./skeleton/plugin.yaml`（或新增命令）为租户/插件设置额度。
   - 在 `framework/backend/go/observability` 中暴露限流/配额指标（命中率、被拒绝次数），并在限流事件时写 `audit.capability.invocation.denied`。
   - 在 docs/operations 章节说明如何配置与监控限流/配额。
