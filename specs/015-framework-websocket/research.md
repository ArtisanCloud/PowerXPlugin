# Research: Framework WS Bus Adapter

## Decision 1: Topic 命名与兼容策略

- **Decision**: 支持 `org_sync.progress` 作为兼容别名，同时引入规范化 topic `powerx.org_sync.progress.v1` 作为推荐新值。
- **Rationale**: 现有前端/业务已使用 `org_sync.progress`，强制替换会破坏宿主/standalone 的对齐与验收。
- **Alternatives considered**:
  - 仅保留规范化 topic：会导致存量订阅失效。
  - 仅保留旧 topic：违反宪章的事件命名规范，影响长期治理。

## Decision 2: 宿主发布入口调用方式

- **Decision**: 宿主模式通过框架 SDK 调用底座发布入口 `POST /api/v1/admin/runtime/internal/ws-bus/publish`。
- **Rationale**: 与现有 WS-NOTIFY 文档契约一致，且便于宿主统一鉴权与 topic 白名单治理。
- **Alternatives considered**:
  - 直接访问宿主 WS Hub 内部方法：跨进程不可行且破坏契约。
  - 直接写宿主 Redis：绕过授权/审计，不符合规范。

## Decision 3: Topic 注册机制

- **Decision**: 宿主模式启动后调用 `POST /api/v1/admin/runtime/internal/ws-bus/register` 注册 topic。
- **Rationale**: 底座可维护动态注册表并明确可订阅范围。
- **Alternatives considered**:
  - 不注册：需要底座全量放行，难以治理。

## Decision 4: 鉴权方式

- **Decision**: 通过 STS 获取短期凭证调用宿主发布入口。
- **Rationale**: 宿主模式对外调用必须走 STS 短期凭证，符合宪章要求。
- **Alternatives considered**:
  - 长期凭证或匿名调用：违反安全要求。

## Decision 5: Standalone 调试入口

- **Decision**: standalone 提供 `POST /api/v1/admin/runtime/internal/ws-bus/publish` 作为调试入口（仅 dev mode）。
- **Rationale**: 便于本地联调 `/ws` 订阅，避免业务端尚未接入时无发布通道。
- **Alternatives considered**:
  - 不提供调试入口：本地验证成本高。
