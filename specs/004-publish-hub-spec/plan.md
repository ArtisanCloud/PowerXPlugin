# Implementation Plan: PowerX Publish Hub Distribution Spec

**Branch**: `004-publish-hub-spec` | **Date**: 2025-11-07 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `/specs/004-publish-hub-spec/spec.md`

## Summary

Publish Hub 需要一个端到端方案，将 `px-plugin dev/publish/dist` 产物统一到单一真相，借助 Marketplace 在线/离线审核来保证安全与 SLA，并让租户在 PowerX Admin 中完成安装、灰度与回滚。实现重点：定义 `.pxp` 契约、Dev API register/reload 协议、在线/离线审核流程、RBAC + 180 天审计策略与 Telemetry/SLA 监控，再以 CLI/Go/Nuxt 既有堆栈落地。

## Technical Context

**Language/Version**: Go 1.24 (PowerX Core/Dev API), TypeScript 5 + Node.js 18 (px-plugin CLI & tooling), Nuxt 4.2 (Admin)  
**Primary Dependencies**: Gin、`@artisan-cloud/plugin-framework-*`、px-plugin CLI runtime、S3/MinIO 对象存储、Redis/Kafka 链路、Playwright 1.48+  
**Storage**: Artefacts + integrity 文件放置于对象存储；Marketplace 复用既有 DB 持久化版本/审核记录，本阶段不新增数据库  
**Testing**: `go test ./...`, `go tool cover`, `npm test && npm run lint`, `npm run e2e:dev-hotload`, Playwright Admin 流程、CLI 集成测试  
**Target Platform**: 多租户 PowerX 运行时（Linux/k8s），CLI 面向 macOS/Linux/Windows；Admin 基于 Nuxt 4  
**Project Type**: Go 服务 + Node CLI + Nuxt Admin 的多项目单仓（Monorepo）  
**Performance Goals**: `px-plugin dev` reload P95 ≤ 2s；`px-plugin publish` 预检+入队 ≤ 10min；在线审核 ≤ 4h；离线审核 ≤ 1 工作日（可观测且可告警）；租户通知 ≤ 30min；安装失败 5min 内可回滚；`.pxp` 校验 100% 通过  
**Constraints**: `.pxp` 默认 <300MB；契约优先并同步到 docs/contracts；CLI 模板与脚手架输出一致；Dev API 必须 mTLS/幂等；离线环境零公网依赖；RBAC 守卫覆盖开发者/审核员/租户角色并保留 180 天审计日志；在线/离线 SLA 需具备独立监测与仪表板
**Scale/Scope**: 覆盖数百插件、上千租户；支持 ≥50 并发租户升级以及数十并行开发者热加载

## Constitution Check

- **I. 双重使命仓库** — ✅ CLI 命令、docs/contracts 与脚手架保持一致，确保模板/框架产出统一。
- **II. 契约优先兼容性** — ✅ 所有新 API、manifest、`.pxp` Schema 先落在 `specs/004-publish-hub-spec/contracts/` 并与 `docs/contracts/**` 对齐。
- **III. Go + Nuxt 基线** — ✅ 继续在 Go/Gin 服务 + Nuxt Admin 上扩展，无引入额外运行时。
- **IV. 脚手架与 CLI 纪律** — ✅ `px-plugin dev/publish/dist` 均在 CLI 侧定义参数/示例，并与 `px-plugin init` 模板验证。
- **V. 透明交付与一致性** — ✅ research/data-model/quickstart/contracts 统一沉淀，方便路线图追踪与 docmap 更新。
- **Post-Design Re-check** — ✅ Phase 1 artefacts（research/data-model/contracts/quickstart）未引入新的栈或结构，所有闸门保持通过。

## Project Structure

### Documentation (this feature)

```text
specs/004-publish-hub-spec/
├── plan.md              # 本文件
├── research.md          # Phase 0 输出
├── data-model.md        # Phase 1 输出
├── quickstart.md        # Phase 1 输出
├── contracts/           # Phase 1 输出（OpenAPI/Schema）
└── tasks.md             # Phase 2 (/speckit.tasks) 生成
```

### Source Code (repository root)

```text
framework/
├── runtime/
├── plugins/
└── observability/

tools/
└── cli/                      # px-plugin CLI (dev/publish/dist)

sdk/workspace/
├── packages/framework-admin/
└── packages/framework-client/

docs/
├── contracts/
└── use_cases/_from_hub/SCN-PUBLISH-HUB-001/

specs/
└── 004-publish-hub-spec/

make-files/
└── targets/cli.mk            # 统一 npm/go 目标

tests/
├── cli/
└── e2e/
```

**Structure Decision**: 保持现有多语言 Monorepo，不新增子项目；CLI 改动集中在 `tools/cli`，Go/Nuxt 侧按既有目录拓展，所有设计 artefact 均存放在 `specs/004-publish-hub-spec/`，方便链路追踪。

## Complexity Tracking

No constitution violations identified — 不需要额外复杂度豁免。

## Additional Scope from SCN-DEV-PLUGIN-INIT/DEBUG/PUBLISH

### Workstream A – CLI Scaffold & Compliance Bootstrap (SCN-DEV-PLUGIN-INIT-001)
- **Deliverables**: `px-plugin init` + `px-plugin doctor` CLI（`tools/cli/src/commands/plugin/*.ts`）、模板索引+执行器（`scaffold/templates/**` 或 `packages/template-registry/index.yaml`）、第三方导入策略（`config/compliance/external_source_policy.yaml`）、bootstrap 服务（`framework/backend/go/runtime/bootstrap/service/bootstrap_handler.go`）与许可证扫描接入。
- **Acceptance**: 初始化 ≤60s、依赖安装成功率 ≥98%、Git/CI 自动注册、SBOM+许可证扫描报告可追踪，团队克隆运行 `plugin doctor` 后健康项 ≥95%，第三方导入 15 分钟内完成审批。
- **Testing**: `scripts/workflows/scaffold-smoke.mjs`, CLI 集成测试覆盖模板渲染/导入/doctor, Go 单测覆盖 bootstrap handler、scanner 适配。

### Workstream B – Host Simulator & Sandbox Validation (SCN-DEV-PLUGIN-DEBUG-001)
- **Deliverables**: `px-plugin host start --mock` + `px-plugin debug attach`（`tools/cli/src/commands/dev/host.ts`, `tools/cli/src/runtime/hotreload/session.ts` 扩展）、宿主模拟器控制器（`framework/backend/go/runtime/devapi/handlers/host_simulator.go`）、沙箱验证与报告 API（`framework/backend/go/runtime/devapi/handlers/sandbox_validation.go`, `.../telemetry/debug_reports.go`）、调试报告文档/工单集成（`docs/guides/publish/marketplace-review.md`, `docs/guides/bootstrap-context.md` 更新）。
- **Acceptance**: 宿主启动 ≤30s、热更新延迟 ≤2s、沙箱验证 ≤10min、错误诊断 1min 生成并自动关联工单，敏感字段全部脱敏。
- **Testing**: `npm run e2e:dev-hotload` 扩展 host simulator 路径、Go integration 覆盖 sandbox deploy/report、Playwright/Admin 验证调试日志面板。

### Workstream C – Release Pipeline & Marketplace Orchestration (SCN-DEV-PLUGIN-PUBLISH-001)
- **Deliverables**: `px-plugin publish create/deploy` CLI（`tools/cli/src/commands/publish/create.ts`, `publish/deploy.ts`）、`px-plugin pack` + `px-plugin import --offline`（`tools/cli/src/commands/dist.ts` & 新 pack 执行器）、发布流水线 orchestrator + canary 策略（`framework/backend/go/runtime/publish/pipeline_handler.go`, `config/publish/approval_flows.yaml`）、Marketplace listing UI/API（`examples/starter/web-admin/app/pages/publish/pipelines.vue`, `framework/backend/go/runtime/marketplace/services/listing.go`）。
- **Acceptance**: 测试租户门禁 + 审批在 24h 内闭环，灰度可 5 分钟回滚，离线导入成功率 ≥98%，Marketplace 审核 ≤3 个工作日且事件同步率 ≥99%。
- **Testing**: CI pipeline smoke tests、Playwright 发布/回滚/Marketplace 审核剧本、离线导入 fixture、Grafana SLA 仪表板验证。
