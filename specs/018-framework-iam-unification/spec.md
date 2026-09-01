# Feature Specification: Framework IAM 统一封装（Standalone/Delegated）

**Feature Branch**: `018-framework-iam-unification`  
**Created**: 2026-04-11  
**Status**: Draft  
**Input**: User description: "把 local/delegated 的 IAM 统一接口上提到 framework（组织、成员、角色、权限、token、上下文解析），输出 framework iam 契约 + adapter 机制；skeleton 只做实现适配。"

## Clarifications

### Session 2026-04-11

- Q: Framework IAM 的模式判定优先级如何定义？ → A: `config.context.provider_mode` 最高优先，其次环境变量；冲突即启动失败（fail-fast）。
- Q: Delegated 模式下组织架构写操作边界如何定义？ → A: Delegated 下插件只读组织数据，所有写操作走宿主接口，插件不本地落库。
- Q: Standalone(local) 模式下组织架构的最小必备范围是什么？ → A: 租户、部门、成员、角色、权限为完整最小集。
- Q: Framework IAM adapter 切换策略如何定义？ → A: 启动期一次性绑定单一 adapter，运行中不自动切换。

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
- **FR-004**: 系统 MUST 定义统一模式解析与优先级规则：`config.context.provider_mode` 高于环境变量信号（如 `POWERX_PROVIDER_MODE`、`POWERX_PROXY`），并在冲突时 fail-fast。
- **FR-005**: 系统 MUST 定义统一错误语义（认证失败、权限不足、上下文缺失、上游不可用）并跨模式保持一致。
- **FR-006**: 系统 MUST 提供统一审计与观测字段，至少包含 tenant、user、role、permission、mode、trace。
- **FR-007**: skeleton MUST 作为 framework IAM 契约的适配实现存在，不再承担契约定义职责。
- **FR-008**: 系统 MUST 发布破坏性变更说明与迁移步骤；新契约生效后不为废弃接口、废弃数据格式或旧 adapter 提供静默兼容或 fallback。
- **FR-009**: 在 delegated 模式下，组织/成员/角色/权限等写操作 MUST 由宿主接口承载；插件侧只读，不本地持久化写入。
- **FR-010**: 在 standalone(local) 模式下，系统 MUST 至少提供租户、部门、成员、角色、权限五类实体能力，作为可运行最小集。
- **FR-011**: 系统 MUST 在启动阶段完成 IAM adapter 单选绑定（local 或 delegated），运行期不得自动切换 adapter。
- **FR-012**: `DirectoryService` MUST 提供 `GetMember(ctx, tenantUUID, memberUUID)` 与 `BatchGetMembers(ctx, tenantUUID, memberUUIDs)`；按 UUID 的业务查询不得通过拉取全量成员列表替代。
- **FR-013**: 所有跨插件、跨服务、API、事件和审计中的成员引用 MUST 使用 `member_uuid`；`user_uuid` 与 `member_uuid` 语义独立，不得互相替代。
- **FR-014**: 成员不存在、跨租户、无权限与 delegated 上游不可用 MUST 返回稳定错误；不得返回空成功结果掩盖异常。
- **FR-015**: `display_name` 仅承载人类可读名称。解析失败时必须为空或返回明确错误，MUST NOT 回填成员 UUID。
- **FR-016**: delegated adapter MUST 仅调用 PowerX Core 已发布的 Host IAM Contract；MUST NOT 直连 Core 数据库、调用 Core 内部 service 或解析 Gateway 原始 `map[string]any`。

### Key Entities *(include if feature involves data)*

- **IAM Adapter**: framework 约束的模式实现单元（local/delegated）。
- **Identity Context**: 统一身份上下文（tenant/user/roles/permissions/policy_version）。
- **Organization Node**: 组织结构节点（tenant、department、member 的层级关系）。
- **Tenant**: 本地 IAM 租户主体，作为组织与权限边界。
- **Department**: 租户内部门层级节点，承载组织树关系。
- **Member**: 租户内成员主体，以 `member_uuid` 为稳定跨边界标识；可选关联独立的 `user_uuid`。
- **Role**: 权限集合载体，定义可授予能力范围。
- **Permission**: 原子权限项（resource/action），用于统一鉴权判定。
- **Authorization Decision**: 权限判定结果，包含结果、原因和审计信息。
- **Mode Resolution Record**: 模式解析输出，记录来源、优先级与冲突信息。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 2 个以上插件在不改业务 handler 的前提下完成 local/delegated 双模式切换验证，通过率 100%。
- **SC-002**: 组织/成员/角色/权限核心接口在两种模式下契约一致性测试通过率 100%。
- **SC-003**: 身份上下文解析相关线上故障（模式不一致、上下文缺失）较改造前 30 天下降至少 60%。
- **SC-004**: 新插件接入 IAM 的开发步骤较现状减少至少 40%（以模板接入 checklist 统计）。
