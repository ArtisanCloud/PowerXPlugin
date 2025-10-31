# Tasks: PowerXPlugin Testing Strategy

**Input**: Design documents from `/specs/002-testing-strategy/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: 本特性强调“文档 + 自动化脚本”双交付；各用户故事均需验证脚本执行结果。

**Organization**: 任务按用户故事分组，确保每个故事可以独立实现与验证。

## Phase 1: Setup（共享基础）

**Purpose**: 准备脚本与文档的工作目录、基础依赖以及临时目录约定。

- [ ] T001 建立 `scripts/testing/` 目录结构并创建占位 README (`scripts/testing/README.md`)
- [ ] T002 在仓库根级 Makefile 中添加 `test-smoke` / `test-regression` 目标占位（注释说明待实现）
- [ ] T003 [P] 更新 `.gitignore` / `tmp/` 目录约定（确保覆盖率/报告文件不被提交）
- [ ] T004 [P] 校验 Playwright/Go/Node 版本检测命令可在当前环境输出（记录示例命令）

---

## Phase 2: Foundational（阻塞性前置）

**Purpose**: 编写共用脚本、测试契约和文档交叉引用，所有用户故事必须在此基础上开展。

- [ ] T005 编写 `scripts/testing/validate-contracts.sh`（含 JSON 校验、OpenAPI 校验、临时项目生成）
- [ ] T006 完成 `docs/test/testing_strategy.md` 中的脚本目录/执行说明更新（引用新脚本路径）
- [ ] T007 [P] 在 `docs/test/testing_usage.md` 中补充脚本执行示例与常见问题链接
- [ ] T008 [P] 在 plan.md Phase 2 中列出的 CI/CD 集成步骤准备文档（脚本落地前的 TODO 标注）

**Checkpoint**: 基础契约脚本与文档同步完成，可进入各用户故事自动化实现。

---

## Phase 3: 用户故事 1 - Run Core Smoke Tests (Priority: P1) 🎯 MVP

**Goal**: 提供 `smoke` 自动化脚本 + Makefile 入口，覆盖后端单测、契约校验、CLI 生成。

**Independent Test**: 执行 `./scripts/testing/smoke.sh` 或 `make test-smoke`，确认每一步输出符合 `contracts/testing-workflows.md` 的定义且≤5 分钟。

### 实现

- [ ] T009 [US1] 实现 `scripts/testing/smoke.sh`（包含 Go 测试、契约校验、CLI scaffolding、产物输出）
- [ ] T010 [US1] 在 Makefile 中实现 `test-smoke` 目标并调用脚本（含 5 分钟超时、失败命令回显、返回非零码）
- [ ] T011 [US1] 更新 `docs/test/testing_usage.md` 的冒烟段落（加入脚本与 Makefile 使用说明）
- [ ] T012 [US1] 在 `docs/test/testing_strategy.md` 中记录冒烟流程的脚本细节/产物位置
- [ ] T013 [US1] 添加 `tmp/coverage.html` 自动生成/清理逻辑（脚本内体现并在文档注明）

**Checkpoint**: `smoke.sh` 与 `make test-smoke` 可独立运行并生成约定产物。

---

## Phase 4: 用户故事 2 - Execute Comprehensive Regression (Priority: P1)

**Goal**: 交付 `regression` 脚本 + Makefile 入口，串联后端全量测试、Playwright E2E、产物归档。

**Independent Test**: 执行 `./scripts/testing/regression.sh` 或 `make test-regression`，确认四个测试层依次运行并输出报告。

### 实现

- [ ] T014 [US2] 实现 `scripts/testing/regression.sh`（复用 smoke 流程 + Playwright 启停 + 报告归档；标注 Playwright 稳定性风险与重试策略）
- [ ] T015 [US2] 在 Makefile 中实现 `test-regression` 目标（依赖 `test-smoke`，处理环境变量）
- [ ] T016 [US2] 更新 `docs/test/testing_usage.md` 回归段落（描述服务等待、Playwright 环境变量）
- [ ] T017 [US2] 在 `docs/test/testing_strategy.md` 中补充回归工作流时序与重试策略
- [ ] T018 [US2] 配置 `.github/workflows/test.yml`：并行 backend/contracts/cli job，串行执行 Playwright E2E，缓存 `~/.cache/ms-playwright`，并暴露 `make test-smoke`/`make test-regression` 日志

**Checkpoint**: `regression.sh` 与 CI 集成验证通过，形成发布前必跑流程。

---

## Phase 5: 用户故事 3 - Extend Coverage with New Tests (Priority: P2)

**Goal**: 提供扩展指南，帮助开发者为新功能添加 Go/Playwright/CLI 测试并正确接入脚本。

**Independent Test**: 按文档指引新增一条示例测试（Go 或 Playwright），并通过 `make test-smoke`/`make test-regression` 验证被覆盖。

### 实现

- [ ] T019 [US3] 扩展 `docs/test/testing_usage.md` 的“添加新测试”章节（给出 Go/Playwright 目录示例）
- [ ] T020 [US3] 在 `docs/test/testing_strategy.md` 中补充 TestLayer/TestWorkflow 映射样例（含具体 Go 单测与 Playwright spec 代码片段）
- [ ] T021 [US3] 在 `scripts/testing/README.md` 记录新增测试如何挂接到现有脚本
- [ ] T022 [US3] 在文档中提供 Go 测试/Playwright spec 示例代码块（更新 `docs/test/testing_usage.md` 或相关说明）

**Checkpoint**: 开发者按照文档新增测试后可被现有脚本识别并执行。

---

## Phase 6: Polish & Cross-Cutting

**Purpose**: 收尾、质量提升与遵循宪章的检查。

- [ ] T023 [P] `docs/test/testing_strategy.md` 与 `testing_usage.md` 进行终审校对（同步新指令、产物路径）
- [ ] T024 [P] 完善 `scripts/testing/README.md`，列出所有脚本/Makefile 目标与常见环境变量
- [ ] T025 运行 `make test-smoke`、`make test-regression` 并更新 Quickstart 中的预计耗时数据
- [ ] T026 在 `research.md` 添加实施后回顾（记录任何变更决策）
- [ ] T027 在 `scripts/testing/smoke.sh`、`regression.sh` 中记录总执行耗时并在 `docs/test/testing_usage.md` 说明如何读取/对照 SC-001、SC-002
- [ ] T028 编写 `scripts/testing/audit-test-adoption.sh` 并在 `docs/test/testing_strategy.md` 追加“测试采纳率”章节，用于统计近期 PR 是否附带测试（支撑 SC-003）

---

## Dependencies & Execution Order

1. **Phase 1 → Phase 2**：必须先建立目录/Makefile占位，再编写基础脚本与文档。
2. **Phase 2 → User Stories**：所有用户故事依赖 `validate-contracts` 和文档同步。
3. **用户故事优先级**：US1、US2 为 P1，需先完成；US3 在 P1 落地后进行。
4. **CI 集成**：US2 完成后触发，Polish 阶段验证最终流程。

---

## Parallel Opportunities

- Phase 1 中 T001、T002 需顺序完成；T003、T004 可并行（不同文件/命令验证）。
- Phase 2 中 T006、T007、T008 可并行处理文档与计划更新。
- US1/US2 实现脚本时谨慎并行，建议一个开发者负责一个脚本以避免冲突。
- Polish 阶段 T023/T024 可并行，T025 需待脚本实现完成。

---

## Implementation Strategy

1. **MVP**：完成 Phase 1 → Phase 2 → US1（smoke 脚本可跑通），即可提供最小可用流程。
2. **增量**：US2 引入回归脚本及 CI，US3 扩充覆盖指南，最终进入 Polish。
3. **验证**：每阶段完成后运行对应脚本，确保产物存放正确并更新文档。
