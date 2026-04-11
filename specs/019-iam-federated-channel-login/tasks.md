# Tasks: IAM 联邦渠道扫码登录（企微/钉钉/飞书）

**Input**: Design documents from `/specs/019-iam-federated-channel-login/`  
**Prerequisites**: spec.md, plan.md

## Phase 1: Provider Contracts & Security Baseline

- [ ] T001 定义 federated provider 抽象接口与统一错误语义。
- [ ] T002 设计扫码 challenge（state/nonce/ttl）与回调校验流程。
- [ ] T003 定义回调风控策略（replay、cross-tenant、expired state）与审计字段。

## Phase 2: Identity Binding & JIT

- [ ] T004 设计并实现 external identity / binding 数据模型。
- [ ] T005 实现绑定、解绑、查询 API 与租户隔离校验。
- [ ] T006 实现首次扫码 JIT 入库/绑定策略与可配置开关。

## Phase 3: Authorization Mapping

- [ ] T007 实现外部身份到本地 member 的角色/部门映射策略。
- [ ] T008 实现登录后统一身份上下文输出（tenant/user/roles/permissions）。
- [ ] T009 对齐 standalone/delegated 模式下的登录态与上下文行为一致性。

## Phase 4: Observability, Risk, and Validation

- [ ] T010 增加登录审计日志与风控事件指标。
- [ ] T011 补齐安全回归测试（重放、过期、签名异常、跨租户）。
- [ ] T012 补齐业务回归测试（扫码成功、绑定更新生效、渠道故障降级）。
- [ ] T013 输出接入文档（渠道配置、绑定策略、排障手册）。

