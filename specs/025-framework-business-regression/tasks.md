# Tasks: 插件业务场景回归工作台

2026-10-17 补充交付：结果概览／业务校验／测试数据与清理／执行与诊断四分区、插件声明的业务结果展示、显式资源名称与来源及清理读回证据，落在 T017／T020／T027～T031／T042／T051／T056 的既有范围。用户随后明确授权窄屏全局导航的最小适配。8 项 Chromium、197 项 unit、Go race、CLI 与两个生成后端均通过，详见[本次证据](evidence/acceptance.md#结果页分区与可核对证据2026-10-17)。不新增任务编号，仍为 56/57，T045 待专用环境。

**Input**: `/specs/025-framework-business-regression/`
**Prerequisites**: [spec.md](./spec.md)、[plan.md](./plan.md)、[research.md](./research.md)、[data-model.md](./data-model.md)、[contracts](./contracts/scenario-sdk.md)
**Status**: 本地实现与验收已落地；用户确认暂时没有专用 Core 环境，T045 待环境就绪。勾选基于源码与对应本地证据，安装态／已发布模块另验。
**Tests**: spec FR-024 明确要求关键机制、租户隔离、迁移、页面与生成物验收；相关验证先于对应实现，不把纯函数测试列为工作台业务场景。
**Format**: `- [ ] Txxx [P?] [USx?] 具体动作及文件路径`；[P] 只表示不同文件可在其前置完成后并行，不表示当前已授权多 agent 或并行服务操作。

## Phase 1: Setup（公共接口）

- [x] T001 定义 UUID、套件/场景/profile、required_assertion_keys、步骤断言、Run/CaseExecution/Event 与状态契约于 `framework/backend/go/runtime/regression/types.go`，EventMeta 复用 `framework/backend/go/event.Meta`，对齐 OpenAPI 与 data-model。
- [x] T002 定义 Scenario、Scope、Profile factory、Recorder 与 FixtureRegistry 接口于 `framework/backend/go/runtime/regression/scenario.go`，禁止业务 SDK 依赖 testing.T/gin.Context。
- [x] T003 定义原子创建、快照/事件更新、游标补读与恢复 Store 接口于 `framework/backend/go/runtime/regression/store.go`，在 `errors.go` 定义稳定 Meta 无效、必要断言漏报/重复及证据缺失机器码。

## Phase 2: Foundational（阻塞基础）

- [x] T004 [P] 补注册完整性、重复 UUID、空步骤、空/重复必要断言 key、输入 schema、缺失时限与不可用 profile 验证于 `framework/backend/go/runtime/regression/registry_test.go`，先确认不合规注册失败。
- [x] T005 实现启动 Registry、必要断言声明与输入校验于 `framework/backend/go/runtime/regression/registry.go`，冻结 required_assertion_keys，满足 T004。
- [x] T006 [P] 定义三表持久化模型及 models.S 表名常量于 `skeleton/backend/go-gin/internal/entity/models/regression/records.go`、`skeleton/backend/go-gin/internal/entity/models/model.go`，使用 UUID 主键，包含 EventMeta JSON、missing_assertion_keys 和审计字段。
- [x] T007 编写提交去重、租户事务、原子快照/事件、Meta 六项校验/回放保持及游标范围测试于 `framework/backend/go/runtime/regression/storegorm/store_test.go`，验证 schema-qualified 表名与无裸 DB 跨层访问。
- [x] T008 实现注入租户事务函数的 GORM adapter 于 `framework/backend/go/runtime/regression/storegorm/store.go`、`records.go`，仅补必需依赖至 `framework/backend/go/go.mod`。
- [x] T009 实现组合 BaseRepository 的 Store 装配于 `skeleton/backend/go-gin/internal/entity/repository/regression/repository.go`，PostgreSQL 与 SQLite 均显式隔离租户。
- [x] T010 [P] 补 UUID、索引、RLS、幂等与回滚冒烟于 `skeleton/backend/go-gin/cmd/database/migrate/regression_test.go`，真实 PostgreSQL RLS 验证不能用 SQLite 代替。
- [x] T011 注册模型与 RLS/index/显式回滚于 `skeleton/backend/go-gin/cmd/database/migrate/migrate.go`、`regression.go`，保持 POWERX_RUN_MIGRATE 门控。
- [x] T012 [P] 补默认禁用、非法环境、缺失 Store/profile 绑定声明和配置不可变测试于 `skeleton/backend/go-gin/internal/config/regression_test.go`，仅依赖公共接口，不构造后续具体 Runner/场景。
- [x] T013 实现配置加载于 `skeleton/backend/go-gin/internal/config/regression.go`、`config.go`，更新 `skeleton/backend/etc/config.example.yaml`，无真实凭据或固定服务器地址。
- [x] T014 [P] 定义 regression:read/invoke/manage 映射与标准错误于 `skeleton/backend/go-gin/internal/transport/http/admin/regression/rbac.go`、`skeleton/plugin.d/rbac.yaml`，后台校验可信租户/操作者。

**Checkpoint**: 注册契约、存储、迁移、权限与配置就绪。此阶段不装配具体 Service/worker/场景；它们在 US1 的依赖完成后装配，不能以测试 Store 替代持久化。

## Phase 3: User Story 1 - 页面执行业务场景 (Priority: P1)

**Goal**: 从管理端执行 controlled 模板场景，查看步骤、业务断言和清理结果。
**Independent Test**: 模板生命周期正常及断言失败两种情况均可从页面定位；连续三次执行互相独立。

- [x] T015 [P] [US1] 补真实业务读回、空/仅可选/漏报/重复必要断言、必需标记不可降级、blocked/skipped/证据缺失与 completed 非通过测试于 `framework/backend/go/runtime/regression/runner_test.go`。
- [x] T016 [P] [US1] 补 create/list/detail、可信身份、提交去重、忙碌和越权测试于 `skeleton/backend/go-gin/internal/transport/http/admin/regression/handler_test.go`。
- [x] T017 [US1] 实现 Step/Assert/Evidence、必要标记派生、重复/漏报检查、fixture 登记、严格 Meta 校验、脱敏与结论聚合于 `framework/backend/go/runtime/regression/recorder.go`、`report.go`，非空必要断言与清理共同约束通过。
- [x] T018 [US1] 在 Recorder 就绪后实现串行 worker 与 Preflight/Prepare/Execute/Cleanup 闭环于 `framework/backend/go/runtime/regression/runner.go`；按冻结的必要 key 核对结果与 missing_assertion_keys，继续独立场景且跳过依赖失败步骤。
- [x] T019 [P] [US1] 补模板场景重复执行、故障读回与资源清理测试于 `skeleton/backend/go-gin/tests/integration/regression/template_lifecycle_test.go`。
- [x] T020 [US1] 编写模板创建/修改/读取/删除场景于 `skeleton/backend/go-gin/internal/services/admin/regression/scenarios/template_lifecycle.go`，使用正式业务 Service 和独立 fixture。
- [x] T021 [US1] 在 Recorder/Runner 与模板场景完成后创建 controlled profile factory，并装配 Service、Store、worker 生命周期及启动入口于 `skeleton/backend/go-gin/internal/services/admin/regression/profile.go`、`service.go`、`internal/shared/app/deps.go`、`cmd/plugin/main.go`；禁用状态不得开放执行，live 装配留到 US3。
- [x] T022 [US1] 在受控 Service 装配完成后实现套件、场景、profile、创建批次与详情薄 Handler 于 `skeleton/backend/go-gin/internal/transport/http/admin/regression/handler.go`、`handler.go`，注册现有 admin router。
- [x] T023 [US1] 实现基础持久化 SSE 推送于 `skeleton/backend/go-gin/internal/transport/http/admin/regression/events.go`，复用 realtime envelope，内层 EventMeta 完整且与运行/来源一致，提交后广播，回放保留原 Meta。
- [x] T024 [P] [US1] 补客户端字段、必要断言列表、missing_assertion_keys、事件 Meta 缺失/不一致及身份校验测试于 `framework/frontend/nuxt/framework-client/__tests__/regression.test.ts`，不依赖硬编码用户文案。
- [x] T025 [US1] 实现类型化客户端、现有会话/Realtime 接入及导出于 `framework/frontend/nuxt/framework-client/regression.ts`、`index.ts`、`package.json`，确保发布文件 allowlist 包含新增模块。
- [x] T026 [P] [US1] 补表单/业务错误/状态/权限测试于 `skeleton/web-admin/nuxt/tests/unit/regression-workbench.spec.ts`。
- [x] T027 [US1] 实现公共工作台 composable 于 `framework/frontend/nuxt/framework-admin/layer/app/composables/usePowerXRegression.ts`，注入统一 API、locale 和权限。
- [x] T028 [US1] 实现目录、执行表单、步骤和断言详情于 `framework/frontend/nuxt/framework-admin/layer/app/components/powerx/PXRegressionWorkbench.vue`，遵循 Nuxt UI 明暗主题与本地化。
- [x] T029 [US1] 添加域 API 包装与薄页面于 `skeleton/web-admin/nuxt/app/composables/api/useRegression.ts`、`app/pages/admin/regression/index.vue`、`app/pages/_p/com.powerx.plugins.base/admin/regression/index.vue`，复用现有 API base。
- [x] T030 [US1] 补工作台/场景 locale 与菜单权限于 `skeleton/web-admin/nuxt/i18n/locales/en.json`、`zh.json`、`skeleton/plugin.yaml`，导航接入现有 Sidebar 数据源，不直接展示 UUID。
- [x] T031 [US1] 完成真实页面发起、过程查看和失败定位验收于 `skeleton/web-admin/nuxt/tests/e2e/regression-workbench.spec.ts`，记录 API 与业务数据库证据。

**Checkpoint**: US1 最小闭环可用；源码测试与真实页面/业务事实证据分别报告。

## Phase 4: User Story 2 - 历史、恢复与清理 (Priority: P1)

**Goal**: 刷新恢复、取消、中断及清理失败能够准确处理，报告和历史可查看。
**Independent Test**: 关闭/恢复页面、断线补读、取消及重试清理保留完整记录和原始 verdict。

- [x] T032 [P] [US2] 补部分准备失败、panic、超时、协作式取消和清理 context 测试于 `framework/backend/go/runtime/regression/lifecycle_test.go`。
- [x] T033 [US2] 实现取消、超时、独立清理时限和中断恢复于 `framework/backend/go/runtime/regression/lifecycle.go`，在 `skeleton/backend/go-gin/internal/services/admin/regression/service.go` 接入实例生命周期。
- [x] T034 [US2] 实现历史、取消和 fixture 清理重试于 `skeleton/backend/go-gin/internal/transport/http/admin/regression/handler.go`，只处理已登记同租户资源，重试不改原 verdict。
- [x] T035 [P] [US2] 验证 JSON/JUnit 结论映射、版本快照和脱敏于 `framework/backend/go/runtime/regression/report_test.go`，实现导出至 `report.go` 及管理端 report handler。
- [x] T036 [P] [US2] 补事务事件顺序、订阅/补读竞态、游标越权、慢消费者与终结流测试于 `skeleton/backend/go-gin/internal/transport/http/admin/regression/handler_test.go`。
- [x] T037 [US2] 实现游标补读、先订阅后回放与去重于 `skeleton/backend/go-gin/internal/transport/http/admin/regression/events.go`，连接异常明确结束。
- [x] T038 [US2] 实现快照恢复、历史、停止中、清理重试及身份/session revision 切换关闭订阅于 `framework/frontend/nuxt/framework-admin/layer/app/composables/usePowerXRegression.ts`、`components/powerx/PXRegressionWorkbench.vue`。
- [x] T039 [US2] 补取消/中断/阻塞/清理失败 locale 于 `skeleton/web-admin/nuxt/i18n/locales/en.json`、`zh.json`，并更新 `skeleton/web-admin/nuxt/tests/e2e/regression-workbench.spec.ts` 验证刷新、重连及两个租户隔离。

**Checkpoint**: 全部异常准确归类，历史和证据可恢复，清理只触及本次资源。

## Phase 5: User Story 3 - 受控与真实集成 (Priority: P2)

**Goal**: 独立 profile 绑定与真实业务读回证明两类回归，依赖缺失必须阻塞。
**Independent Test**: controlled 和 live 分别运行；缺失 live grant 不得变为模拟成功。

- [x] T040 [P] [US3] 补 profile 选择、缺失授权、actual_mode、无全局配置改变和无降级测试于 `skeleton/backend/go-gin/internal/services/admin/regression/profile_test.go`。
- [x] T041 [US3] 在既有 controlled factory 上扩展独立 live factory 与逐项 Preflight 于 `skeleton/backend/go-gin/internal/services/admin/regression/profile.go`，复用现有 STS/Core SDK；真实场景启动注册在场景实现后完成。
- [x] T042 [P] [US3] 编写场景合同与隔离测试于 `skeleton/backend/go-gin/internal/services/admin/regression/scenarios/metadata_tag_bindings_test.go`，明确 mock transport 测试不能替代真实 Core 验收。
- [x] T043 [US3] 实现后端固定专用资源与既有测试标签的真实 Core 绑定/读回/重复绑定及原标签集合恢复场景于 `skeleton/backend/go-gin/internal/services/admin/regression/scenarios/metadata_tag_bindings.go`，声明必要断言，并在 `cmd/plugin/main.go` 追加 live factory 和场景注册；不创建、停用或删除既有标签。
- [x] T044 [US3] 展示 profile 名称、依赖方式、验证边界和阻塞原因于 `framework/frontend/nuxt/framework-admin/layer/app/components/powerx/PXRegressionWorkbench.vue`，补两类示例的 `skeleton/web-admin/nuxt/i18n/locales/en.json`、`zh.json`。
- [ ] T045 [US3] 在已授权的专用环境验证真实 grant、业务读回、幂等与清理，将证据保存于 `specs/025-framework-business-regression/evidence/live-integration.md`；权限未获授权时准确记录阻塞，不自行扩大操作范围。

**Checkpoint**: 受控、真实授权、真实业务结果各有独立证据，不能互相替代。

## Phase 6: User Story 4 - 统一接入、CLI 与生成物 (Priority: P2)

**Goal**: 页面、CLI 和新生成插件使用相同契约，自动化默认运行受控场景。
**Independent Test**: 参考骨架与新插件分别注册执行，CLI/page 导出的业务字段与判定一致。

- [x] T046 [P] [US4] 补 CLI 请求去重、参数 UUID、SSE 等待/断线和退出码测试于 `tools/cli/internal/regression/client_test.go`。
- [x] T047 [US4] 实现 list/run/report 与 --wait 于 `tools/cli/cmd/regression.go`、`cmd/root.go`、`internal/regression/client.go`，通过 --token-env 读取凭据，用户文案使用 `internal/regression/i18n/en.json`、`zh.json`。
- [x] T048 [US4] 同步标准后端配置、注册、迁移、示例和接口至 `scaffold/templates/backend/go-gin/`、`scaffold/templates/backend/go-gin/etc/`，禁止携带运行数据或真实密钥。
- [x] T049 [US4] 同步 Nuxt 薄页面、locale、权限与菜单至 `scaffold/templates/web-admin/nuxt/`、`scaffold/templates/plugin.yaml.tmpl`、`scaffold/templates/plugin.d/rbac.yaml.tmpl`。
- [x] T050 [US4] 使用现有 `package.json` 的 sync:templates 流程同步 `tools/cli/internal/templates/data/`，验证新增 Framework Client 文件与后端 adapter 均进入 allowlist。
- [x] T051 [P] [US4] 在 `tools/cli/internal/templates/regression_generated_test.go` 验证 px-plugin init 新项目的注册/构建/页面/场景闭环，明确本地依赖验证与已发布 consumer 验证的区别。
- [x] T052 [US4] 将受控场景接入 `scripts/testing/regression.sh`、`.github/workflows/ci.yml` 的既有隔离测试流程并归档报告，live 仅显式配置环境执行，不触发 CI 或发布作为本次文档任务。
- [x] T053 [US4] 完成 CLI/page 报告一致性和两插件接入证据于 `specs/025-framework-business-regression/evidence/adoption.md`。

**Checkpoint**: 新项目可按规范接入；外部插件升级新 SDK 的正式版本须在另行授权发布后验证，不能用本地 replace 声称已发布可用。

## Phase 7: Polish & Cross-Cutting Concerns

- [x] T054 [P] 编写完整接入与使用指南于 `docs/guides/features/025-framework-business-regression/guide.md`，包含业务场景步骤、依赖、流程/泳道关系、预期结果、证据及排障，回链业务计划与 specs。
- [x] T055 核查 UUID、locale、manifest/RBAC、SDK/OpenAPI 与模板一致性于 `specs/025-framework-business-regression/checklists/implementation.md`，不得补 numeric id 或通用 fallback。
- [x] T056 按 `specs/025-framework-business-regression/quickstart.md` 完成源码/迁移/受控/浏览器/真实授权分层验收，保存 `evidence/acceptance.md` 并记录自有临时服务 PID/端口及释放结果。
- [x] T057 回填 `specs/025-framework-business-regression/quickstart.md` 的实际可执行命令与环境结果，更新 `docs/plan/025-framework-business-regression.md` 状态；仅以真实实现和证据勾选任务。

## Dependencies & Execution Order

- Phase 1 → Phase 2 → US1 → US2 → US3 → US4 → Phase 7 为首期推荐开发顺序。
- US1 依赖共享基础；US2 复用 US1 的运行与基本事件流，可通过固定场景单独验证；US3 复用执行器，可通过单独 profile 验证；US4 复用受保护接口，可通过 CLI 与生成项目单独验证。
- 每个故事先补关键测试，再实现对应行为，最后执行 Independent Test；迁移/RLS、真实外部授权和浏览器验收分别完成，不能用其他层的测试结果替代。
- [P] 仅用于不同文件且前置已完成的任务。例如 US1 的 runner/handler 测试、US2 的报告/事件测试、US3 的 profile/场景合同验证；共享 Runner、组件、locale、Handler 的编辑任务依次进行。

### 关键任务依赖

| 任务 | 必须先完成 | 依赖原因 |
| --- | --- | --- |
| T005 | T001、T002、T003、T004 | 注册规则和公共接口/验证先就绪。 |
| T008 | T006、T007 | 持久化模型与事务/Meta 契约先就绪。 |
| T009 | T006、T008 | 插件仓储装配依赖标准模型与 Store adapter。 |
| T011 | T006、T010 | 迁移实现对应模型与 RLS/回滚验证。 |
| T017 | T001、T003、T005、T009、T015 | Recorder 依赖定义、存储、必要断言/Meta 合同和验证。 |
| T018 | T015、T017 | Runner 使用已就绪 Recorder 和必要断言判定。 |
| T020 | T018、T019 | 模板场景实现依赖执行器与场景验收约定。 |
| T021 | T009、T011、T013、T014、T018、T020 | controlled factory 与 Service/启动装配依赖数据库、配置、权限、执行器和具体场景。 |
| T022 | T016、T021 | 薄 Handler 依赖接口验证与可用 Service。 |
| T023 | T007、T021、T022 | SSE 依赖持久化事件、Meta 与已装配服务/路由。 |
| T025 | T023、T024 | Client 对齐可用接口/事件和客户端合同验证。 |
| T027 | T025、T026 | composable 依赖类型化 Client 和 UI 验证约定。 |
| T028 | T027 | 工作台组件依赖域状态与客户端。 |
| T029 | T028 | 插件薄页面接入已完成公共组件。 |
| T031 | T023、T029、T030 | 页面验收依赖事件流、页面、locale 和菜单。 |
| T041 | T021、T040 | live factory 在受控装配基线上扩展，遵循独立 profile 验证。 |
| T043 | T041、T042 | 真实场景及启动注册依赖 live factory 与场景合同；不提前引用未实现场景。 |
| T045 | T043、T044 | 真实验收依赖场景注册与页面显示，不反向阻塞受控 MVP。 |

## Parallel Examples

- US1：T015 与 T016 可独立补机制/接口测试，T024 可独立补 Client 测试；相关类型先由 Phase 1 固定。
- US2：T035 报告验证与 T036 事件恢复验证可独立，生命周期修改与 UI 接入按依赖推进。
- US3：T040 配置边界验证与 T042 场景合同验证可独立；live 环境准备与源码验证分别记录。
- US4：T046 CLI 测试与 T051 生成物测试可独立；模板同步须在源模板修改完成后执行。

## Implementation Strategy

1. MVP 为 Phase 1/2/US1：受控模板场景从页面执行，持久化步骤、断言和清理结果。
2. 增量交付 US2 的恢复与异常，再交付 US3 的真实集成证据，最后交付 US4 的统一接入与 CLI。
3. 每个 checkpoint 评审结果与证据边界；发现需要新增 Core 能力、改其他插件、用户服务操作或发布时先说明范围差异。
4. 提交、推送、合并、发布和触发 CI 均按用户另行授权执行，不因 tasks 模板中的通用建议自动开展。

## 执行记录（2026-10-13）

- 56/57 项完成。本地结果见 [acceptance.md](evidence/acceptance.md)、[adoption.md](evidence/adoption.md)。
- T045：用户明确确认没有专用环境；真实授权、撤权和 Core 业务闭环待环境就绪。
- T051：两个生成插件完成注册、后端及各自 Nuxt 构建、真实业务用例、页面失败／取消／刷新历史及 CLI 闭环；source_plugin 分别等于各自插件 ID。
- T056：已完成本次用户确认的本地分层验收，并明确记录 live／安装态／发布边界。
- 未提交、推送、发布或触发 CI。

### 本地交付续作

- 两个生成插件的安装／构建／页面／CLI 验收已串成 `regression-generated.py`，隔离 npm 11.6.2、独立依赖目录，实际通过。
- CLI 中断可恢复标识与严格事件时间／租户快照校验补齐；Pinia 显式依赖、产物 Git 隔离、CI Chromium 闭环补齐。
- `live-integration.md` 保存只读前置检查与缺失配置；用户再次确认仍无环境，T045 不勾选。

### 管理页面与规范复核（2026-10-14）

- 按既有页面要求补齐目录步骤／边界／时限、最近批次入口、详情版本／耗时／实际依赖／必要断言和测试资源技术详情。
- 新增独立管理页面使用指南并回链计划与总览；统一 SDK／quickstart 的 CLI 退出码、live 既有标签恢复口径及实际代码路径。
- CLI 重复 `--case` 对齐规范并完成真实 HTTP 请求测试；189 项前端测试、CLI race、Nuxt build、双模板同步检查通过。
- Skeleton 与两个独立生成插件的新版页面／CLI 再验通过，临时服务与端口均释放；结果见 [acceptance.md](evidence/acceptance.md)。
- 不新增任务或改变真实验收边界；仍为 56/57，唯一未完成任务为 T045。

### 页面操作流程重做（2026-10-15）

- 对应 T028、T029、T030、T031、T038、T044 的展示与验收修订，补充 FR-025～FR-027：用例列表／执行记录分标签页，详情抽屉说明业务，准备弹窗使用预填业务表单，确认后进入当次结果。
- 公共工作台通过 `RegressionCasePresentation` 接收插件的通过条件、数据影响、字段与 locale 映射；缺少映射明确阻止执行，空必填项显示字段提示，不提供 JSON 输入兜底。
- 结果顶部定位失败步骤，直接展示校验差异及清理；历史重跑带回原批次保存的输入并重新核对。技术快照、版本与证据保留在明确的技术入口。
- 页面、接入指南、SDK 展示契约、双模板及浏览器用例同步更新；本轮本地验收证据见 [acceptance.md](evidence/acceptance.md)。T045 继续等待专用环境。


## 用户确认的 Template 用例扩展验收（2026-10-15）

本次为 T019／T020／T021／T028～T031／T051／T056 的业务覆盖补充，不改变原 57 项任务编号；原任务仍为 56/57，T045 等待真实 Core 环境。

- 已将 controlled 从 1 条扩展到 8 条：lifecycle、search、pagination、deletion、clone、partialClone、invalidInput、validation。每条具备独立输入、业务步骤、必要断言、证据和资源清理，使用正式 TemplateService；输入拒绝经现有进程内 HTTP 路由验证。
- 已补齐插件页面展示映射及中英文说明；表头全选当前可见列表，批量准备保留业务目的和数据影响，预填输入按需展开，输入错误自动展开对应表单。
- 完成正序／倒序两次整批、8 次独立运行（24 次通过的用例执行），额外注入 HTTP 拒绝后意外写入故障，确认判失败且 3 条登记资源全部清理；已有同租户和其他租户对象不变，最终 active_templates=0。
- 浏览器 6 项通过，含页面真实 8 条整批；CLI 重复 `--case` 的 8 条批次通过；Go race 检查通过；两个独立生成插件包含同一目录与故障验收并通过；191 项前端单测、Nuxt build、双模板同步检查通过。完整证据见 [acceptance.md](evidence/acceptance.md)。
- 临时服务与子进程已退出并核对端口释放；日常 8078／3131 服务保持原进程。用户日常后端需按原启动方式重新启动，才能加载这次新增的 Go 场景；不将本地验收等同于日常实例已更新。
