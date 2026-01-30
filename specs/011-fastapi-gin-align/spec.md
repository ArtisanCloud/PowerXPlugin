# Feature Specification: Python Backend Parity for Nuxt Integration

**Feature Branch**: `011-fastapi-gin-align`  
**Created**: 2026-01-24  
**Status**: In Progress (Handlers/DDD aligned)  
**Input**: User description: "FastAPI 对齐 Go Gin（以 Nuxt 联调为第一目标）"

## Clarifications

### Session 2026-01-24

- Q: 最小联调 API 里哪些必须纳入 P1（第一阶段必须可用）的范围？ → A: 认证 + IAM 核心 + 模板 CRUD + capability 生命周期/曝光/配额
- Q: 宿主模式的反代路径规范，FastAPI 需要确保哪些路径必定可用并与 Gin 一致？ → A: 业务接口 + 管理端点
- Q: 数据库结构一致性第一阶段范围？ → A: 先对齐联调所需实体
- Q: 是否需要明确性能/可用性目标用于联调验收？ → A: 设轻量目标（P95 < 1s，错误率 < 1%）
- Q: 当出现契约不一致时，优先以哪一侧为权威进行修正？ → A: 以 Go Gin 实现为权威

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Nuxt 管理端可直接联调 (Priority: P1)

作为前端使用者，我希望在不改动 Nuxt 代码的情况下，Python 后端能够提供与现有 Go 后端一致的管理端能力，从而完成核心管理流程的联调验证。

**Why this priority**: 这是联调第一目标，能直接验证 Python 后端与前端的契约一致性。

**Independent Test**: 使用现有 Nuxt 管理端页面访问核心管理流程，验证主要功能可用且响应结构一致。

**Acceptance Scenarios**:

1. **Given** Nuxt 管理端不做修改，**When** 用户完成登录、权限查询与基础数据浏览，**Then** 页面正常工作且无契约错误。
2. **Given** Python 后端启动，**When** 触发模板与权限相关的 CRUD 流程，**Then** 返回结构与现有后端一致。

**Current State**: Handler 已按 Gin 行为对齐（Auth/IAM/Templates/Capabilities/Runtime Sessions/Manifest/RBAC），可进入联调验收阶段。

---

### User Story 2 - 宿主模式接口一致 (Priority: P2)

作为插件在宿主环境运行时，我希望 Python 后端在反代路径、权限校验与租户上下文上与现有后端一致，以避免宿主模式下的行为偏差。

**Why this priority**: 宿主模式是实际部署形态，接口规范不一致会导致上线失败。

**Independent Test**: 通过宿主反代路径请求管理端 API，验证权限与租户上下文的行为与现有后端一致。

**Acceptance Scenarios**:

1. **Given** 宿主反代路径已启用，**When** 发起管理端请求，**Then** 请求按同样的路径前缀与权限策略处理。

---

### User Story 3 - 数据结构一致性 (Priority: P3)

作为运维与开发者，我希望 Python 后端的表结构与现有后端保持一致，确保数据可互通与迁移可复用。

**Why this priority**: 数据结构不一致会直接导致业务数据不可用或迁移失败。

**Independent Test**: 通过对比关键实体的表结构与字段，验证一致性。

**Acceptance Scenarios**:

1. **Given** 关键实体的数据库结构已定义，**When** 对比 Python 与 Go 后端的模型/迁移结果，**Then** 表名与字段保持一致。

---

### Edge Cases

- 当配置中缺失 API 前缀时，系统如何保证默认行为与现有后端一致？
- 当租户上下文缺失或无效时，系统如何返回一致的错误响应？

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统必须在不修改现有前端代码的前提下完成核心管理流程的联调。
- **FR-002**: 系统必须遵循与现有后端一致的路径前缀策略与管理端点范围。
- **FR-003**: 系统必须输出与现有后端一致的响应封装与错误语义。
- **FR-004**: 系统必须遵循相同的权限与租户上下文规则。
- **FR-005**: 系统必须覆盖 Nuxt 当前依赖的管理端功能域（认证、IAM、能力管理、模板与运行时会话）。
- **FR-006**: 系统必须保持与现有后端一致的数据库表结构与字段定义。
- **FR-007**: 系统不得修改现有 Go 后端或 Nuxt 前端实现。
- **FR-008**: 第一阶段必须覆盖认证、IAM 核心、模板 CRUD 与 capability 生命周期/曝光/配额的管理端接口。
- **FR-009**: 宿主模式下必须同时支持业务接口反代路径与管理端点规范。
- **FR-010**: 第一阶段数据库结构仅需对齐联调所需实体，其余实体分阶段补齐。
- **FR-011**: 发生契约不一致时，以 Go Gin 实现作为最终权威进行修正。

### Key Entities *(include if feature involves data)*

- **Tenant**: 租户与隔离上下文，包含租户标识与状态。
- **User**: 用户信息与认证状态。
- **Role**: 权限角色与权限集合。
- **Permission**: 资源与动作的权限定义。
- **Department**: 组织结构与成员归属。
- **Template**: 可由前端管理的模板资源。
- **Capability**: 能力清单与生命周期信息。
- **Runtime Session**: 运行时会话与调用状态。

## Dependencies & Assumptions

- 现有 Go 后端实现与其契约被视为权威基线。
- Nuxt 前端保持不变，联调基于现有调用行为。
- 宿主反代路径与租户上下文已可用，用于一致性验证。
- 关键实体的数据库结构已有明确基线。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 在不改动 Nuxt 代码的前提下，核心管理流程可完成率达到 100%。
- **SC-002**: 管理端 API 的响应结构与错误语义在联调验证中保持一致（无结构性不匹配）。
- **SC-003**: 宿主反代路径下的管理端请求成功率达到 99% 以上。
- **SC-004**: 关键实体表结构对比一致率达到 100%。
- **SC-005**: 关键联调 API 的 P95 响应时间小于 1s，错误率低于 1%。
