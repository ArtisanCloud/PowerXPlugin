# Tasks: Next 管理端与 Nuxt 对齐

**Input**: Design documents from `/specs/012-next-nuxt-align/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/openapi.yaml, quickstart.md

**Tests**: 本特性在规格中明确要求 E2E 回归与双模式一致性验证，因此包含测试任务。

**Organization**: 任务按用户故事分组，确保每个故事可独立实现与独立验证。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行执行（不同文件、无未完成前置依赖）
- **[Story]**: 所属用户故事（US1/US2/US3）
- 每个任务描述均包含明确文件路径

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 初始化 Next 迁移所需基础目录、脚本与测试壳

- [x] T001 创建 Next 迁移目录骨架于 skeleton/web-admin/next/app/(public)/users/
- [x] T002 [P] 创建管理端目录骨架于 skeleton/web-admin/next/app/(admin)/
- [x] T003 [P] 创建宿主代理路由目录于 skeleton/web-admin/next/app/_p/[pluginId]/admin/[...internal]/
- [x] T004 [P] 创建 API 与鉴权目录于 skeleton/web-admin/next/lib/api/
- [x] T005 [P] 创建 bridge 与运行模式目录于 skeleton/web-admin/next/lib/bridge/
- [x] T006 [P] 创建 store 目录于 skeleton/web-admin/next/lib/stores/
- [x] T007 [P] 创建组件目录于 skeleton/web-admin/next/components/
- [x] T008 [P] 创建 E2E 测试目录于 skeleton/web-admin/next/tests/e2e/
- [x] T009 配置 Playwright 基础配置于 skeleton/web-admin/next/playwright.config.ts
- [x] T010 更新 Next 脚本命令（含 e2e）于 skeleton/web-admin/next/package.json

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 任何用户故事实现前必须完成的共用能力

**⚠️ CRITICAL**: 本阶段完成前不得开始 US1/US2/US3

- [x] T011 建立 API 基础客户端与 envelope 解析于 skeleton/web-admin/next/lib/api/client.ts
- [x] T012 [P] 实现统一 API 错误标准化于 skeleton/web-admin/next/lib/api/normalizeApiError.ts
- [x] T013 [P] 实现 API 基址解析（独立/宿主）于 skeleton/web-admin/next/lib/api/baseUrl.ts
- [x] T014 实现会话存储协议（access/refresh/expires）于 skeleton/web-admin/next/lib/auth/session.ts
- [x] T015 [P] 实现鉴权守卫与重定向策略于 skeleton/web-admin/next/middleware.ts
- [x] T016 [P] 实现 Host Bridge 适配与 insidePowerX 判定于 skeleton/web-admin/next/lib/bridge/hostBridge.ts
- [x] T017 实现默认与嵌入布局路由组装于 skeleton/web-admin/next/app/layout.tsx
- [x] T018 [P] 建立 Nuxt->Next 逐页路由对照清单（含 Integration/Ops/Security）于 specs/012-next-nuxt-align/parity-map.md
- [x] T019 [P] 建立联调差异记录模板于 specs/012-next-nuxt-align/parity-gap-log.md
- [x] T020 建立 E2E 通用工具（登录、模式切换、断言）于 skeleton/web-admin/next/tests/e2e/_utils.ts
- [x] T076 [P] 建立错误语义矩阵（code/message/envelope）于 specs/012-next-nuxt-align/error-semantics-matrix.md

**Checkpoint**: 基础设施就绪，可开始用户故事开发

---

## Phase 3: User Story 1 - 核心管理流程可迁移可联调 (Priority: P1) 🎯 MVP

**Goal**: 完成登录、首页、模板主链路，并与现有 Gin 契约联通

**Independent Test**: 独立执行登录 + 进入首页 + 模板 CRUD 全流程，验证行为与 Nuxt 基线一致

### Tests for User Story 1

- [x] T021 [P] [US1] 新增本地鉴权回归用例于 skeleton/web-admin/next/tests/e2e/auth-local.spec.ts
- [x] T022 [P] [US1] 新增模板 CRUD 回归用例于 skeleton/web-admin/next/tests/e2e/templates-crud.spec.ts

### Implementation for User Story 1

- [x] T023 [P] [US1] 实现登录页面于 skeleton/web-admin/next/app/(public)/users/login/page.tsx
- [x] T024 [P] [US1] 实现注册页面于 skeleton/web-admin/next/app/(public)/users/register/page.tsx
- [x] T025 [P] [US1] 实现忘记密码页面于 skeleton/web-admin/next/app/(public)/users/forgot-password/page.tsx
- [x] T026 [US1] 实现管理首页 intro 页面于 skeleton/web-admin/next/app/(admin)/intro/page.tsx
- [x] T027 [P] [US1] 实现模板列表页面于 skeleton/web-admin/next/app/(admin)/templates/page.tsx
- [x] T028 [P] [US1] 实现模板 CRUD 页面于 skeleton/web-admin/next/app/(admin)/templates/crud/page.tsx
- [x] T029 [P] [US1] 实现模板 develop 页面于 skeleton/web-admin/next/app/(admin)/templates/develop/page.tsx
- [x] T030 [P] [US1] 实现模板 API 调用于 skeleton/web-admin/next/lib/api/template.ts
- [x] T031 [P] [US1] 实现 auth API 调用于 skeleton/web-admin/next/lib/api/auth.ts
- [x] T032 [US1] 实现模板表单弹窗组件于 skeleton/web-admin/next/components/templates/TemplateFormModal.tsx
- [x] T033 [US1] 实现模板与登录状态 store 于 skeleton/web-admin/next/lib/stores/templates.ts
- [x] T034 [US1] 对齐 US1 页面 testid 与基线映射于 specs/012-next-nuxt-align/parity-map.md

**Checkpoint**: US1 可独立联调与回归通过

---

## Phase 4: User Story 2 - 双运行模式与权限语义一致 (Priority: P2)

**Goal**: 完成宿主/独立双模式下一致的 IAM、Capabilities、Integration、Operations、Security 主链路

**Independent Test**: 在双模式下执行 IAM + Capabilities + Integration/Ops/Security 主链路与关键异常场景，结果一致

### Tests for User Story 2

- [x] T035 [P] [US2] 新增委托鉴权回归用例于 skeleton/web-admin/next/tests/e2e/auth-delegated.spec.ts
- [x] T036 [P] [US2] 新增 IAM 主链路回归用例于 skeleton/web-admin/next/tests/e2e/iam-local.spec.ts
- [x] T037 [P] [US2] 新增能力调用回归用例于 skeleton/web-admin/next/tests/e2e/capability-invocation.spec.ts
- [x] T038 [P] [US2] 新增双模式异常场景回归用例于 skeleton/web-admin/next/tests/e2e/mode-parity-edge.spec.ts
- [x] T068 [P] [US2] 增加逐页路由可访问与入口可见性断言于 skeleton/web-admin/next/tests/e2e/route-parity.spec.ts
- [x] T077 [P] [US2] 增加错误语义矩阵断言回归于 skeleton/web-admin/next/tests/e2e/error-semantics.spec.ts

### Implementation for User Story 2

- [x] T039 [P] [US2] 实现 IAM 概览页面于 skeleton/web-admin/next/app/(admin)/admin/iam/overview/page.tsx
- [x] T040 [P] [US2] 实现 IAM 成员页面于 skeleton/web-admin/next/app/(admin)/admin/iam/members/page.tsx
- [x] T041 [P] [US2] 实现 IAM 角色页面于 skeleton/web-admin/next/app/(admin)/admin/iam/roles/page.tsx
- [x] T042 [P] [US2] 实现 IAM 设置页面于 skeleton/web-admin/next/app/(admin)/admin/iam/settings/page.tsx
- [x] T043 [P] [US2] 实现 Capabilities 生命周期页面于 skeleton/web-admin/next/app/(admin)/capabilities/lifecycle/page.tsx
- [x] T044 [P] [US2] 实现 Capabilities 注册页面于 skeleton/web-admin/next/app/(admin)/capabilities/register/page.tsx
- [x] T045 [P] [US2] 实现 Capability Lab 页面于 skeleton/web-admin/next/app/(admin)/powerx/capability-lab/page.tsx
- [x] T046 [P] [US2] 实现测试能力页面于 skeleton/web-admin/next/app/(admin)/tests/capability/page.tsx
- [x] T047 [P] [US2] 实现 Integration 页面于 skeleton/web-admin/next/app/(admin)/integration/page.tsx
- [x] T048 [P] [US2] 实现 Operations 页面于 skeleton/web-admin/next/app/(admin)/operations/page.tsx
- [x] T049 [P] [US2] 实现 Security 页面于 skeleton/web-admin/next/app/(admin)/security/page.tsx
- [x] T050 [P] [US2] 实现 IAM API 调用于 skeleton/web-admin/next/lib/api/iam.ts
- [x] T051 [P] [US2] 实现 Capability API 调用于 skeleton/web-admin/next/lib/api/capabilities.ts
- [x] T052 [P] [US2] 实现 Integration/Ops/Security API 调用于 skeleton/web-admin/next/lib/api/operations.ts
- [x] T053 [US2] 实现宿主路径透传页于 skeleton/web-admin/next/app/_p/[pluginId]/admin/[...internal]/page.tsx
- [x] T054 [US2] 实现双模式权限可见性规则 store 于 skeleton/web-admin/next/lib/stores/permissions.ts
- [x] T055 [US2] 更新 US2 页面与异常场景映射于 specs/012-next-nuxt-align/parity-map.md
- [x] T069 [US2] 统计 IAM/Capabilities 一次通过率并输出于 specs/012-next-nuxt-align/e2e-report.md

**Checkpoint**: US2 双模式一致性与关键异常场景可独立验证

---

## Phase 5: User Story 3 - 迁移过程可验证且不破坏既有后端基线 (Priority: P3)

**Goal**: 固化差异归因、Gin 缺陷准入、发布门禁与回归证据链

**Independent Test**: 对任一联调差异可在 2 个工作日内完成归因并产出可审计结论，发布门禁可执行

### Tests for User Story 3

- [x] T056 [P] [US3] 新增差异归因流程演练脚本于 specs/012-next-nuxt-align/scripts/parity-triage-drill.md
- [x] T057 [P] [US3] 新增发布门禁核查清单于 specs/012-next-nuxt-align/checklists/release-gate.md

### Implementation for User Story 3

- [x] T058 [US3] 编写联调差异归因 SOP 于 specs/012-next-nuxt-align/parity-triage-sop.md
- [x] T059 [US3] 编写 Gin 缺陷准入与最小修复规范于 specs/012-next-nuxt-align/gin-defect-policy.md
- [x] T060 [US3] 编写双端回归证据模板于 specs/012-next-nuxt-align/regression-evidence-template.md
- [x] T061 [US3] 在 quickstart 增补差异 SLA 与门禁执行步骤于 specs/012-next-nuxt-align/quickstart.md
- [x] T062 [US3] 在 plan 中补充门禁验收追踪点于 specs/012-next-nuxt-align/plan.md
- [x] T070 [US3] 增加联调差异 SLA 统计模板于 specs/012-next-nuxt-align/parity-gap-log.md
- [x] T071 [US3] 输出 2 工作日归因达成率报告于 specs/012-next-nuxt-align/sla-report.md
- [x] T072 [US3] 增加“禁止 Next 私有接口漂移”校验清单于 specs/012-next-nuxt-align/checklists/contract-drift.md
- [x] T078 [US3] 增加 contract diff 自动化校验脚本于 specs/012-next-nuxt-align/scripts/check-contract-drift.sh

**Checkpoint**: US3 治理流程可独立执行并用于发布评审

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: 跨故事收尾、统一验收与质量固化

- [ ] T063 [P] 统一补齐 API/鉴权注释与使用说明于 skeleton/web-admin/next/lib/api/README.md
- [ ] T064 [P] 更新 Next 迁移说明文档于 skeleton/web-admin/next/README.md
- [ ] T073 配置 Next 构建产物对齐发布路径于 skeleton/web-admin/next/next.config.js
- [ ] T074 [P] 增加产物路径一致性校验脚本于 skeleton/web-admin/next/scripts/verify-artifacts.mjs
- [ ] T065 执行全量 E2E 并记录结果于 specs/012-next-nuxt-align/e2e-report.md
- [ ] T066 执行 lint/build 并记录结果于 specs/012-next-nuxt-align/verification-report.md
- [ ] T067 汇总最终页面对齐状态与遗留风险于 specs/012-next-nuxt-align/parity-map.md
- [ ] T075 在发布验证报告中记录产物路径校验结果于 specs/012-next-nuxt-align/verification-report.md
- [ ] T079 执行发布包产物树校验并输出于 specs/012-next-nuxt-align/package-artifact-report.md

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: 无依赖，可立即开始。
- **Phase 2 (Foundational)**: 依赖 Phase 1，且阻塞全部用户故事。
- **Phase 3 (US1)**: 依赖 Phase 2 完成。
- **Phase 4 (US2)**: 依赖 Phase 2 完成；可与 US1 并行但建议在 US1 首轮联调通过后推进。
- **Phase 5 (US3)**: 依赖 US1/US2 的实施结果与联调记录。
- **Phase 6 (Polish)**: 依赖目标用户故事完成。

### User Story Dependencies

- **US1 (P1)**: 无用户故事前置依赖，是 MVP。
- **US2 (P2)**: 依赖基础设施，业务上可独立于 US1 验证。
- **US3 (P3)**: 依赖 US1/US2 的输出用于归因与门禁固化。

### Within Each User Story

- 先完成该故事测试任务，再完成实现任务，再更新对照/证据文档。
- 先 API 与状态层，再页面与交互层，最后做回归闭环。

## Parallel Opportunities

- Phase 1 的 T002-T009 可并行。
- Phase 2 的 T012、T013、T015、T016、T018、T019、T076 可并行。
- US1 的 T023-T025、T027-T031 可并行。
- US2 的 T035-T038、T068、T077、T039-T052 可高并行。
- US3 的 T056-T057、T078 可并行。
- Polish 的 T063-T064、T074、T079 可并行。

## Parallel Example: User Story 1

```bash
Task: "T021 [US1] auth-local E2E in skeleton/web-admin/next/tests/e2e/auth-local.spec.ts"
Task: "T022 [US1] templates CRUD E2E in skeleton/web-admin/next/tests/e2e/templates-crud.spec.ts"
Task: "T023 [US1] login page in skeleton/web-admin/next/app/(public)/users/login/page.tsx"
Task: "T030 [US1] template API in skeleton/web-admin/next/lib/api/template.ts"
```

## Parallel Example: User Story 2

```bash
Task: "T039 [US2] IAM overview in skeleton/web-admin/next/app/(admin)/admin/iam/overview/page.tsx"
Task: "T043 [US2] capability lifecycle in skeleton/web-admin/next/app/(admin)/capabilities/lifecycle/page.tsx"
Task: "T047 [US2] integration page in skeleton/web-admin/next/app/(admin)/integration/page.tsx"
Task: "T050 [US2] IAM API in skeleton/web-admin/next/lib/api/iam.ts"
```

## Implementation Strategy

### MVP First (US1)

1. 完成 Phase 1 + Phase 2。
2. 完成 US1（T021-T034）。
3. 仅验证 US1 独立链路并形成首个可演示版本。

### Incremental Delivery

1. 先交付 US1（核心联调闭环）。
2. 再交付 US2（双模式与全域页面）。
3. 最后交付 US3（治理门禁与发布证据）。
4. 每阶段都可单独回归、单独评审。
5. 里程碑一（US1）完成后进入里程碑二（US2 + US3 + Polish）收敛。

### Parallel Team Strategy

1. 一组负责 Foundational（T011-T020）。
2. 一组负责 US1 页面与 API（T023-T033）。
3. 一组负责 US2 页面与双模式（T035-T055、T068、T069、T077）。
4. 质量/发布组负责 US3 与 Polish（T056-T067、T070-T075、T078、T079）。

## Notes

- 所有任务遵循严格 checklist 格式，便于自动化追踪。
- 标记 [P] 的任务应避免修改同一文件以减少冲突。
- 若发现 Gin 缺陷，必须先完成差异归因记录再进入修复流程。
- 任务编号用于追踪而非严格时间顺序，实际执行以依赖关系与阶段约束为准。
- `.output` 等价映射判定标准：需同时满足“构建输出目录可稳定产出”“发布包内存在可被宿主路由消费的管理端静态产物”“校验脚本返回成功并有产物清单记录”。
