# Research: Framework 统一日志适配

## Decision 1: 宿主模式默认仅 stdout，由 PowerX 汇聚

- **Decision**: 在 `POWERX_PROXY=1` 场景下默认输出 `stdout + json`，由 PowerX 统一采集与转发；插件侧直连 `file/loki` 仅在显式授权后启用。  
- **Rationale**: 保证平台级治理一致性、降低链路分叉与审计盲区。  
- **Alternatives considered**:
  - 插件默认多 sink 直连：灵活但治理成本高，配置漂移风险大。
  - 宿主强制禁止所有扩展 sink：安全高但排障与特殊合规场景受限。

## Decision 2: 多 sink 路由采用“独立降级 + 重试”

- **Decision**: sink 失败不阻塞主链路；失败 sink 记录告警并进入重试。  
- **Rationale**: 满足“主业务不中断”与“失败可观测”双目标。  
- **Alternatives considered**:
  - 任一 sink 失败即失败：一致性强但影响业务可用性。
  - 失败即丢弃不重试：实现简单但易形成隐性数据缺口。

## Decision 3: 固定低基数标签基线

- **Decision**: 统一标签固定为 `plugin_id, tenant_uuid, component, level`；其余字段作为日志内容字段。  
- **Rationale**: 防止高基数标签导致索引膨胀与查询成本失控。  
- **Alternatives considered**:
  - 包含 `channel`：可提升部分检索效率，但在多渠道场景会增加基数风险。
  - 允许插件自定义标签：灵活但跨插件检索语义不一致。

## Decision 4: 业务日期口径统一为 UTC + biz_date + biz_tz

- **Decision**: 每条日志保留标准 UTC 时间戳，同时携带 `biz_date` 与 `biz_tz`。  
- **Rationale**: 兼顾跨区域一致性与业务统计可读性。  
- **Alternatives considered**:
  - 仅记录 biz_date：时区语义丢失，跨区对账困难。
  - 仅记录 UTC：查询侧换算复杂，统计口径易不一致。

## Decision 5: 遗留直写日志采用分阶段治理

- **Decision**: 第一阶段告警+审计，第二阶段截止版本后强制阻断。  
- **Rationale**: 在不中断现有插件迭代的前提下实现收口治理。  
- **Alternatives considered**:
  - 立即强制：短期改造风险高，影响交付节奏。
  - 永久兼容：长期无法达成统一门面目标。
