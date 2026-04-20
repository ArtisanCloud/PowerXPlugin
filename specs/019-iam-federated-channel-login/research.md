# Research: IAM 联邦渠道扫码登录（企微/钉钉/飞书）

## Decision 1: Provider factory 与默认渠道实现放在 framework

- **Decision**: 在 `framework/backend/go/iam/federated` 提供 provider contracts、registry/factory 及默认渠道实现（wecom/dingtalk/lark）；skeleton 仅做初始化装配与路由接线。  
- **Rationale**: 满足“其他插件可直接复用”的目标，避免每个插件重复实现 SDK 接入与挑战校验逻辑。  
- **Alternatives considered**:
  - 工厂与实现放 skeleton：当前插件可跑通，但无法版本化复用给其他插件。
  - 每个插件各自实现：最灵活但维护和安全一致性成本最高。

## Decision 2: 默认 JIT 策略为“唯一匹配自动绑定”

- **Decision**: 首次扫码时仅当 external identity 可唯一匹配本地成员时自动绑定；否则进入管理员处理流程。  
- **Rationale**: 在体验和安全间取平衡，降低误绑定风险。  
- **Alternatives considered**:
  - 全自动创建与绑定：体验最好但误绑风险高。
  - 完全禁用 JIT：安全高但落地阻力大。

## Decision 3: 映射策略按版本变化重算

- **Decision**: 登录时检查映射版本，仅在版本变化时重算并覆盖角色/部门上下文。  
- **Rationale**: 保证映射变更可生效，同时避免每次登录无差别重算。  
- **Alternatives considered**:
  - 每次登录都重算：一致性高但性能和稳定性压力更大。
  - 仅首次绑定时计算：性能好但权限滞后明显。

## Decision 4: 风控拒绝返回可区分错误码，前端统一文案

- **Decision**: 后端返回可区分风险错误码（expired/replay/cross-tenant/signature）；前端统一展示通用失败提示，详细原因进审计。  
- **Rationale**: 兼顾安全（不泄露细节）和运维排障（机器可读）。  
- **Alternatives considered**:
  - 仅返回模糊错误：安全较高但定位效率差。
  - 向前端返回详细原因：排障快但容易暴露防护细节。

## Decision 5: delegated 模式以宿主会话/令牌为权威

- **Decision**: delegated 登录结果必须依赖宿主权威会话，插件仅做身份上下文适配与最小缓存。  
- **Rationale**: 符合 Host Contract First，避免身份源分裂。  
- **Alternatives considered**:
  - 插件侧建立独立权威会话：实现快但和宿主一致性风险高。
  - 双权威（宿主 + 插件）：复杂且易产生会话漂移。

## Decision 6: 渠道扩展顺序采用“钉钉先行，飞书跟进”

- **Decision**: 在企微稳定后优先实现钉钉，再实现飞书。  
- **Rationale**: 钉钉组织模型（企业、部门、成员）与企微更同构，可最大复用现有同步与绑定逻辑；飞书字段体系差异更大（`tenant_key/open_id/user_id` 并存）。  
- **Alternatives considered**:
  - 飞书优先：Go SDK生态更成熟，但会增加身份字段适配成本。
  - 两者并行：周期更短但联调复杂度与回归面同时上升。

## Decision 7: SDK 采用“可替换传输层”策略

- **Decision**: provider 抽象作为稳定接口，SDK 仅作为具体实现可替换项；不把业务流程绑定到某单一 SDK。  
- **Rationale**: 避免 SDK 维护状态波动影响主链路，保障统一错误语义与风控处理。  
- **Alternatives considered**:
  - 强绑定单一 SDK：上手快但迁移成本高。
  - 纯手写 HTTP：可控但重复造轮子，调试与签名处理成本高。
