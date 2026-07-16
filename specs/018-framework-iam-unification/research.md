# Research: Framework IAM 统一封装（Standalone/Delegated）

## Decision 1: 模式解析优先级采用“配置优先 + 冲突失败”

- **Decision**: `config.context.provider_mode` 作为最高优先级；环境变量（如 `POWERX_PROVIDER_MODE`、`POWERX_PROXY`）仅作为次级来源；冲突直接 fail-fast。
- **Rationale**: 配置是显式声明，最可控且可审计；冲突自动兜底会造成跨环境行为漂移。
- **Alternatives considered**:
  - 仅环境变量：部署方便但不可控，误注入风险高。
  - 仅配置文件：运维临时切换成本高，不利于容器化。
  - 冲突自动降级 local：会掩盖配置错误并扩大排障成本。

## Decision 2: adapter 切换策略采用“启动期单选绑定”

- **Decision**: 在启动阶段完成 IAM adapter 单选绑定（local 或 delegated），运行期不自动切换。
- **Rationale**: 保证单一执行语义，避免请求级漂移和上下文混乱。
- **Alternatives considered**:
  - 请求期动态切换：灵活但复杂，容易导致不可预测行为。
  - 双 adapter 回退链：看似高可用，实则引入隐式授权路径与安全风险。

## Decision 3: delegated 模式组织写操作边界固定为宿主写

- **Decision**: delegated 模式下插件只读组织数据；组织/成员/角色/权限写操作必须走宿主接口。
- **Rationale**: 宿主是权威写源，避免插件侧双写和最终一致性冲突。
- **Alternatives considered**:
  - 插件本地写 + 异步同步：一致性复杂，失败补偿成本高。
  - 宿主/插件双写：权限边界模糊，审计难统一。

## Decision 4: local 模式最小可运行实体集固定为五类

- **Decision**: local 模式至少提供 `tenant/department/member/role/permission` 五类实体能力。
- **Rationale**: 满足组织架构 + 权限闭环的最小完整集，避免后续反复补模型。
- **Alternatives considered**:
  - 只做 user+role：无法表达组织层级。
  - 不做 department：短期简单但后续 SCRM/审批链路必然返工。

## Decision 5: skeleton 角色定位调整为“默认 adapter 实现”

- **Decision**: framework 负责 IAM 契约定义；skeleton 负责 local/delegated 默认实现和迁移兼容。
- **Rationale**: 可被其他插件复用，避免契约散落在 skeleton 私有目录。
- **Alternatives considered**:
  - 继续由 skeleton 定义契约：复用成本高，跨插件一致性差。

