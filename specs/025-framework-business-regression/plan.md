# Implementation Plan: 插件业务场景回归工作台

**Branch**: `025-framework-business-regression` | **Date**: 2026-10-13 | **Spec**: [spec.md](./spec.md)
**Input**: `/specs/025-framework-business-regression/spec.md`
**Source Plan**: [业务方案](../../docs/plan/025-framework-business-regression.md)
**Status**: 本地实现与验收已完成；真实 Core 授权验收按用户确认待专用环境。

## Summary

Framework 提供业务场景注册、执行、断言、报告、存储与实时更新公共机制，各插件提供业务场景及启动装配。Nuxt 工作台嵌入插件管理端，CLI 调用同一受保护接口。受控与真实集成绑定独立配置，不切换正常业务的全局 ProviderMode。

执行以业务事实和清理结果为准；执行结束、任务完成或接口接受不能代替必要断言。首期覆盖模板生命周期和真实 Core 标签绑定示例，并同步 Skeleton、scaffold 与 CLI 内嵌模板。

## Technical Context

**Language/Version**: Go 1.24, TypeScript 5.x, Node.js 20, Nuxt 4.2
**Primary Dependencies**: Gin, GORM, Nuxt UI 3.3.x, Pinia, Framework Client/Admin Layer, Framework realtime/common logging 与现有 IAM/STS
**Storage**: 插件 PostgreSQL schema powerx_plugin_base；隔离本地测试可使用 SQLite；三类回归记录表，Framework Store 接口与可选 GORM adapter
**Testing**: Go contract/integration、PostgreSQL RLS 与迁移冒烟、Vitest、Playwright、模板同步及生成项目验证；真实 Core 授权与业务读回单列验收
**Target Platform**: 插件 standalone 与 PowerX host/proxy 下显式启用的开发或测试实例
**Project Type**: web 管理端 + backend framework runtime + skeleton adapters + CLI/templates
**Performance Goals**: 首期单实例一个活动批次或清理作业；执行与清理使用场景声明时限，记录实时更新延迟，不设跨外部依赖的统一完成时限
**Constraints**: UUID 跨边界引用、租户隔离、用户文案本地化、证据脱敏、无静默降级、无任意脚本执行、无运行中全局模式修改
**Scale/Scope**: Go/Nuxt 首期；不开发 Core 测试平台、其他业务插件实现、浏览器业务执行器或多实例调度

## Constitution Check

### Pre-Design Gate

| Gate | 设计结论 |
| --- | --- |
| Host Contract First | 复用 server.api_prefix、宿主反代、管理端鉴权与既有 Core STS 客户端，不耦合 Core 内部实现。 |
| Tenant Isolation & Zero Trust | 从验证身份取得 tenant_uuid/actor_uuid；记录、事件游标、fixture 清理均隔离；PostgreSQL RLS 与租户事务必需。 |
| Service-Centric Architecture | Framework 执行器只依赖接口；HTTP handler 薄；插件 Service 注入场景、配置和 Store，Repository 组合 BaseRepository。 |
| Observable & Testable Delivery | 结果与事件先持久化；结构化证据、版本、Trace、异常及清理可检查；迁移幂等并提供回滚，受 POWERX_RUN_MIGRATE 控制。 |
| Event Contracts & TaskBus | 复用 realtime SSE envelope；RegressionEvent.meta 复用 event.Meta，六项必填字段严格校验并持久化，补读保留原值；不新增 TaskBus topic 或宿主回归调度。 |
| Minimal Footprint & Releases | 复用现有 Go/Nuxt 栈；GORM adapter 独立包，不向纯场景接口引入 ORM；发布/tag/CI 触发不属于本次文档任务。 |
| Frontend Gates | 统一 Framework Client/Realtime；接口基址与会话绑定继承现有机制；菜单与按钮受 RBAC 控制；全部文案 i18n，遵循当前 Nuxt UI 与输出目录。 |

适用规则以用户明确的 UUID、i18n、无 fallback 要求为先。宪章通用段的旧 TypeScript 4.x 与 domain 路径说明，同当前 TypeScript 5.x 和 entity 目录存在基线差异；本功能沿用实际工具链与 crud-model/current repository 的 entity 分层，不升级依赖、不改造已有公共目录。该差异记入 research.md，不创建第二套层级。

### Post-Design Gate

数据模型、HTTP/SSE 与接入契约落实上述约束；设计检查通过。此结论仅表示设计文档满足约束，源码、迁移、浏览器及真实授权验收仍待实施。

## Project Structure

### Documentation (this feature)

```text
specs/025-framework-business-regression/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── tasks.md
├── checklists/requirements.md
└── contracts/
    ├── regression.openapi.yaml
    └── scenario-sdk.md
```

### Source Code (planned; not created by this task)

```text
framework/backend/go/runtime/regression/
├── types.go, registry.go, scenario.go, runner.go
├── recorder.go, store.go, report.go, errors.go
└── storegorm/store.go, records.go
framework/frontend/nuxt/framework-client/
└── regression.ts, index.ts, package.json
framework/frontend/nuxt/framework-admin/layer/app/
├── components/powerx/PXRegressionWorkbench.vue
└── composables/usePowerXRegression.ts
skeleton/backend/go-gin/internal/
├── config/regression.go
├── entity/models/regression/records.go
├── entity/repository/regression/repository.go
├── services/admin/regression/service.go
├── services/admin/regression/scenarios/template_lifecycle.go, metadata_tag_bindings.go
└── transport/http/admin/regression/handler.go, events.go, rbac.go
skeleton/backend/go-gin/cmd/
├── plugin/main.go
└── database/migrate/migrate.go, regression.go
skeleton/web-admin/nuxt/app/
├── pages/admin/regression/index.vue
├── pages/_p/com.powerx.plugins.base/admin/regression/index.vue
└── composables/api/useRegression.ts
skeleton/web-admin/nuxt/i18n/locales/
└── en.json, zh.json
skeleton/plugin.yaml, skeleton/plugin.d/rbac.yaml
scaffold/templates/, tools/cli/internal/templates/data/
tools/cli/cmd/regression.go, tools/cli/internal/regression/client.go
scripts/testing/regression.sh, .github/workflows/ci.yml
```

**Structure Decision**: 公共能力位于 runtime/regression，不借用业务 TaskBus 或 TaskCenter 的 completed 状态。插件持久化模型沿用实际 internal/entity 层级；Store/GORM adapter 通过注入租户事务函数和 schema-qualified 表名接入，不引用 Skeleton。页面包装公共组件，声明 locale 依赖；模板镜像由现有同步流程更新。

## Phase 0: Research Output

[research.md](./research.md) 固定以下决策：场景代码为事实来源、独立 profile、三表持久化、串行执行、协作式取消、SSE 补读、正式入口验证、CLI/API 共用和新生成插件验收。

## Phase 1: Design & Contracts

### Runtime and SDK

- `Scenario` 提供 Definition、Preflight、Prepare、Execute、Cleanup；通过 Recorder 记录 Step/Assert/Evidence，不接受自由文本脚本或从浏览器上传执行代码。
- Registry 在启动时校验套件、场景和 profile 的稳定 UUID、i18n key、输入声明及时间约束。场景步骤与 required_assertion_keys 均非空，必要 key 唯一且随定义冻结；重复或缺失注册明确失败。
- Profile UUID 对应独立依赖 factory，类型为 controlled/live。每次执行构造自己的业务依赖与资源清单，不修改生产共享依赖；未经声明的外部依赖不执行。
- 首期同一插件测试实例仅一个活动批次或清理作业；忙碌返回冲突，不维护隐藏队列。接受批次先保存 queued，再由实例内 worker 串行执行选定场景。
- 批次 context 独立于浏览器请求；身份与测试凭据只保留在受控执行上下文中，既有认证链路仍负责验证。请求结束不取消批次，测试凭据失效则明确阻塞。

### Results and Recovery

- [data-model.md](./data-model.md) 定义运行状态、结论、清理状态及三类表；必要证据缺失、场景阻塞或中断不能聚合为 passed。
- Recorder 的 required 标记由定义派生。正常 Execute 返回后逐项核对必要 key，漏报保存 missing_assertion_keys 并判 failed；重复报告失败，blocked/skipped/缺失证据均不通过，至少一个必要断言全部通过且清理成功才可通过。
- 场景依赖步骤失败后跳过后续业务步骤，但仍进入清理；下一独立场景继续执行。预期拒绝场景根据结构化错误码断言，不因非成功 HTTP 状态自动判失败。
- 超时和取消必须等待场景协作式停止，不能只改变 UI 状态后继续写入；不使用强制终止 goroutine 或自动撤销业务副作用的承诺。
- Cleanup 使用独立、有时限的 context，处理已登记资源；panic 和部分 Prepare 失败同样尝试清理。清理重试必须匹配原 profile 的 binding_revision，追加事件并更新资源状态，不改写原 verdict。
- 启动恢复将本实例遗留的未完成批次标为 interrupted；保留 fixture 清单，操作者显式清理，禁止自动重放。首期部署限定单实例。

### Persistence and Interfaces

- Framework 的 Store 提供原子批次创建、事件追加与结果更新；GORM adapter 接受插件注入的 `WithTenantTx` 与表名，禁止 Framework 依赖插件内部仓储。
- 插件模型声明 gorm/json 标签及 models.S 常量；Repository 组合 BaseRepository，为 adapter 提供租户事务。新增记录表显式注册 AutoMigrate；PostgreSQL 使用/检查 RLS，SQLite 保持显式 tenant_uuid 条件。
- [OpenAPI](./contracts/regression.openapi.yaml) 定义目录、profile、批次、取消、清理、报告和 SSE。普通响应使用现有 success/data/error/timestamp/request_id envelope。
- 使用 `client_request_uuid` 保护提交重试；相同身份与请求 UUID、相同输入返回原批次，不同输入冲突。主动重跑提供新 UUID 与 rerun_of_run_uuid。
- SSE 复用 Framework realtime envelope，RegressionEvent.meta 携带 tenant_uuid/request_id/trace_id/source_plugin/occurred_at/payload_version（固定 v1），从可信操作 Scope/manifest 构造并校验，事务保存后再推送；补读保留原 Meta。游标是事件 UUID，顺序是运行内序号，先订阅再补读并按序去重。

### Workbench, CLI and Examples

- Nuxt 提供目录、执行详情和历史；统一 Client/Realtime，切换租户、身份或宿主会话 revision 时关闭旧订阅。失败显示恢复动作，不增加轮询。
- `regression:read` 用于目录、记录、SSE、报告；`regression:invoke` 用于执行；`regression:manage` 用于取消及清理。使用已验证身份校验，不能只依赖前端可见性。
- CLI 提供 list、run、report；run --wait 使用 SSE。凭据通过显式环境变量名称读取，不写入报告或输出；成功退出码仅用于 verdict=passed 且清理成功。
- Skeleton controlled 示例：模板创建、修改、读取、删除；live 示例：后端绑定的专用资源与既有测试标签，通过正式 Core 接口绑定、读回、重复绑定验证，清理恢复原标签集合，不修改既有标签自身生命周期。使用真实既有客户/管理身份入口；必需授权缺失记为 blocked。
- CI 默认执行受控场景。真实集成由显式配置的测试环境运行，结果与真实授权证据单独归档；不创建新权限授予或修改 Core。

## Implementation Sequence and Acceptance

按 [tasks.md](./tasks.md) 执行共享基础 → US1 页面最小闭环 → US2 恢复/异常/清理 → US3 两类回归 → US4 CLI/生成项目 → 文档与全链路验收。测试明确包含在 spec FR-024 中，任务按关键机制先补验证再实现。

共享基础只提供注册、存储、迁移、配置及权限；US1 先完成 Recorder/Runner 和模板场景，再创建 controlled profile factory 并装配 Service、worker 和启动入口，然后实现 Handler/SSE。US3 才扩展 live factory，并在真实集成场景实现后追加启动注册；基础阶段不引用后续才出现的具体执行器或场景。

[quickstart.md](./quickstart.md) 提供实施后的页面步骤与 CLI 目标命令，标明当前尚不可执行的新增能力。源码/合同、迁移、页面、真实授权与业务读回证据分别记录，任务未完成前不得提前勾选。

## Complexity Tracking

没有新增独立应用或宿主持久化平台。新增三类插件记录表、一个可选 GORM adapter 和现有前后端模块扩展，均直接支持本功能；无额外复杂度豁免。
