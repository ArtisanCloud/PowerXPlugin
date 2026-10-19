# Specification Quality Checklist: 插件业务场景回归工作台

**Purpose**: 检查需求文档是否完整、可验证，支持后续实施规划。
**Created**: 2026-10-13
**Feature**: [spec.md](../spec.md)
**Validation Scope**: 本清单仅检查文档质量，不表示实现或运行验收完成。

## Content Quality

- [x] CHK001 spec 聚焦用户价值与业务需求，语言、框架、路由和具体实现留在 plan/contracts。
- [x] CHK002 面向插件开发者，明确目录、执行、定位、恢复和统一接入的业务目标。
- [x] CHK003 必需章节和四个按优先级排序的用户故事完整。
- [x] CHK004 每个故事有明确价值、独立验证方式和 Given/When/Then 场景。

## Requirement Completeness

- [x] CHK005 无未解决的澄清标记；范围选择沿用已确认方案，默认值在 Assumptions 记录。
- [x] CHK006 FR-001 至 FR-024 可验证且编号唯一，明确正常与异常行为。
- [x] CHK007 SC-001 至 SC-008 可衡量，采用数量、比例和可复现结果，未用技术实现作为成功标准。
- [x] CHK008 覆盖断言失败、依赖阻塞、超时、取消、中断、部分准备与清理失败。
- [x] CHK009 明确正式业务入口、受控与真实集成的证据边界；非空必要断言逐项核对，空/漏报/重复/降级/阻塞/跳过/缺证据均不能 passed。
- [x] CHK010 明确 UUID、租户身份、权限、locale、脱敏和测试资源 ownership。
- [x] CHK011 明确仅开发/专用测试环境、串行、无自动降级、无浏览器业务执行器等边界。
- [x] CHK012 明确外部测试环境与授权由操作者准备，不以模拟结果代替真实集成。

## Feature Readiness

- [x] CHK013 四个故事覆盖主要用户流程，FR 可追踪到验收场景与 tasks 的实施阶段；关键依赖无前向引用或循环，controlled 与 live 分阶段装配。
- [x] CHK014 目录、运行、断言、清理、历史与报告的关键实体已定义。
- [x] CHK015 plan、research、data-model、contracts 和 quickstart 对齐范围、接口、必填 EventMeta、必要断言列表及默认值。
- [x] CHK016 实施任务全部保持待实施；文档检查与源码、迁移、浏览器及真实授权验收分开。

## Notes

需求输入和用户已确认的五项选择已纳入 spec。技术细节集中在 plan、data-model 与 contracts；没有将业务计划中的 Go/Nuxt、HTTP/SSE 路由直接写入功能需求。SC-001/003/006/007 分别规定两类场景、三次重复执行、两个项目接入与两个租户隔离，具有明确验证方式。

规范文档检查通过后可进入实施。实际完成状态以 tasks 与 evidence 为准，当前没有执行业务测试、数据库迁移、浏览器验收或真实 Core 授权操作。

初次验证覆盖文档结构、引用和 OpenAPI 格式；实施前语义复查发现的三项问题经本次修订关闭，不能将初次格式通过解释为运行验收通过。

## 实施前复查修正

| 编号 | 原问题 | 已落实的修正 | 文档检查结果 |
| --- | --- | --- | --- |
| DEP-01 | 原 T014 提前引用后续执行器、场景和 profile | 保留 57 项并按执行顺序重新编号：T017 Recorder、T018 Runner、T020 模板场景之后，T021 才装配 controlled Service/worker；T041/T043 扩展并注册 live。新增关键依赖表。 | 18 个依赖节点、45 条边均指向已完成前置，无循环。 |
| META-01 | 事件契约缺少宪章规定的关联字段 | RegressionEvent.meta 复用 event.Meta；六项必填，严格验证、原子持久化和原值回放；OpenAPI/SSE/报告与相应测试任务同步。 | 六项 Meta 均为 required，SSE payload 引用携带 meta 的事件，版本为 v1。 |
| ASSERT-01 | 空声明或漏报必要断言存在误判空间 | 声明非空且唯一的 required_assertion_keys，required 从声明派生，正常执行漏报/重复失败，blocked/skipped/缺证据非通过；输出 missing_assertion_keys。 | 空、空白、首尾空白、重复等 6 类非法声明反例被 schema 约束拒绝；运行行为在 T004/T015/T017/T018 待实现验证。 |

2026-10-13 修订后验证：9 个文件的本地链接、Markdown、24 项 FR、8 项 SC 和 57 项待实施任务格式检查通过；关键依赖顺序检查通过；OpenAPI 3.0 契约通过 swagger-cli 校验，99 个本地引用、9 个路径/10 个操作及 22 个 schema 对齐。AGENTS.md 手工规则完整保留，分支仍为 025-framework-business-regression。以上为文档/契约静态检查，全部代码任务保持待实施。
