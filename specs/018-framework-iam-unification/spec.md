# Feature Specification: Framework IAM 统一封装（Standalone/Delegated）

**Feature Branch**: `018-framework-iam-unification`  
**Created**: 2026-04-11  
**Status**: Draft  
**Input**: User description: "把 local/delegated 的 IAM 统一接口上提到 framework（组织、成员、角色、权限、token、上下文解析），输出 framework iam 契约 + adapter 机制；skeleton 只做实现适配。"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 插件无感切换 IAM 模式 (Priority: P1)

作为插件开发者，我希望仅依赖 framework 的 IAM 接口，就能在 standalone 与 delegated 模式间切换，而不改业务 handler。

**Why this priority**: 这是统一封装的核心价值，决定其他插件能否真正复用 framework 能力。

**Independent Test**: 使用同一业务接口在 local/delegated 两种模式启动运行，验证代码无分支改动且鉴权结果一致。

**Acceptance Scenarios**:

1. **Given** 插件业务层仅调用 framework IAM 接口，**When** 配置切换为 delegated，**Then** 鉴权与用户上下文由 delegated adapter 生效，业务层无需改代码。
2. **Given** 插件业务层仅调用 framework IAM 接口，**When** 配置切换为 standalone，**Then** 鉴权与用户上下文由 local adapter 生效，业务层无需改代码。

---

### User Story 2 - 组织架构与权限能力统一暴露 (Priority: P2)

作为插件维护者，我希望组织、成员、角色、权限查询与变更能力通过统一契约暴露，避免每个插件重复实现 IAM CRUD 接口。

**Why this priority**: 组织架构是 IAM 主体能力，缺少统一接口会导致重复建设和不一致行为。

**Independent Test**: 对 tenant/department/member/role/permission 关键操作进行契约测试，确保两种模式均有明确行为定义与错误语义。

**Acceptance Scenarios**:

1. **Given** 插件调用 framework IAM 组织接口，**When** 请求租户组织树与成员列表，**Then** 返回结构和错误码符合统一契约。
2. **Given** 插件调用 framework IAM 权限接口，**When** 执行角色授权与成员绑定，**Then** 结果可被审计并符合权限边界。

---

### User Story 3 - Token 与上下文解析规则统一 (Priority: P3)

作为平台运维，我希望 token 解析、tenant/user 上下文提取、权限判定都由 framework 统一处理，避免插件实现漂移。

**Why this priority**: 统一解析与审计是多租户安全底线，属于稳定性与安全性保障。

**Independent Test**: 注入不同 token/上下文组合进行回归，验证 tenant、user、roles、permissions 解析一致且可追踪。

**Acceptance Scenarios**:

1. **Given** 请求携带合法身份信息，**When** framework 解析上下文，**Then** tenant/user/roles/permissions 可被统一读取。
2. **Given** 请求缺失关键上下文或 token 不合法，**When** framework 处理请求，**Then** 返回统一错误并记录审计日志。

---

### Edge Cases

- 模式信号冲突（配置声明 local，但运行环境声明 delegated）时如何 fail-fast。
- delegated 模式下宿主身份服务不可用时，如何返回统一降级错误而不是业务 panic。
- local 模式数据不完整（成员存在但角色映射缺失）时，权限判定如何定义默认行为。
- 同一请求存在多个 tenant 来源（header、claims、signed context）时，优先级如何统一。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统 MUST 在 framework 层提供统一 IAM 契约，覆盖 tenant、department、member、role、permission、token、context 相关能力。
- **FR-002**: 系统 MUST 支持 adapter 机制以接入至少两种实现：`standalone(local)` 与 `delegated`。
- **FR-003**: 插件业务层 MUST 可仅依赖 framework IAM 契约完成鉴权与用户上下文访问，不直接依赖 skeleton 私有实现。
- **FR-004**: 系统 MUST 定义统一模式解析与优先级规则，并在冲突时 fail-fast。
- **FR-005**: 系统 MUST 定义统一错误语义（认证失败、权限不足、上下文缺失、上游不可用）并跨模式保持一致。
- **FR-006**: 系统 MUST 提供统一审计与观测字段，至少包含 tenant、user、role、permission、mode、trace。
- **FR-007**: skeleton MUST 作为 framework IAM 契约的适配实现存在，不再承担契约定义职责。
- **FR-008**: 系统 MUST 提供向后兼容迁移路径，使现有插件可分阶段迁移到 framework IAM 接口。

### Key Entities *(include if feature involves data)*

- **IAM Adapter**: framework 约束的模式实现单元（local/delegated）。
- **Identity Context**: 统一身份上下文（tenant/user/roles/permissions/policy_version）。
- **Organization Node**: 组织结构节点（tenant、department、member 的层级关系）。
- **Authorization Decision**: 权限判定结果，包含结果、原因和审计信息。
- **Mode Resolution Record**: 模式解析输出，记录来源、优先级与冲突信息。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 2 个以上插件在不改业务 handler 的前提下完成 local/delegated 双模式切换验证，通过率 100%。
- **SC-002**: 组织/成员/角色/权限核心接口在两种模式下契约一致性测试通过率 100%。
- **SC-003**: 身份上下文解析相关线上故障（模式不一致、上下文缺失）较改造前 30 天下降至少 60%。
- **SC-004**: 新插件接入 IAM 的开发步骤较现状减少至少 40%（以模板接入 checklist 统计）。

