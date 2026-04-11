# Tasks: Framework IAM 统一封装（Standalone/Delegated）

**Input**: Design documents from `/specs/018-framework-iam-unification/`  
**Prerequisites**: spec.md, plan.md

## Phase 1: Contracts & Mode Resolution

- [ ] T001 定义 framework IAM 契约接口（组织、成员、角色、权限、token、context）。
- [ ] T002 定义模式解析策略与冲突 fail-fast 规则（local/delegated）。
- [ ] T003 定义统一错误码与错误分类（auth/permission/context/upstream）。

## Phase 2: Adapter Abstractions

- [ ] T004 在 framework 增加 IAM adapter 注册与获取机制。
- [ ] T005 实现 local adapter 接口适配层（对接 skeleton 现有 IAM 服务）。
- [ ] T006 实现 delegated adapter 接口适配层（对接 auth proxy 与宿主上下文）。

## Phase 3: Skeleton Migration

- [ ] T007 将 skeleton IAM 路由层改为调用 framework IAM 契约。
- [ ] T008 将 middleware/rbac/context 读取统一迁移到 framework context API。
- [ ] T009 清理 skeleton 中与契约重复定义的接口与错误类型（保留兼容层）。

## Phase 4: Validation & Rollout

- [ ] T010 补齐 contract tests（local/delegated 一致性）。
- [ ] T011 补齐 integration tests（模式切换、权限边界、错误语义）。
- [ ] T012 输出迁移文档（旧接口映射、新接入范式、回滚方案）。

