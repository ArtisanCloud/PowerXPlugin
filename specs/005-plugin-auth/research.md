# Research – Plugin Auth Integration

## 1. Delegated 模式故障策略
- **Decision**: 登录/刷新在 Delegated 模式下必须 fail-closed，若 `POWERX_CORE_ENDPOINT` 不可用则立即提示“宿主认证不可用”并终止流程。
- **Rationale**: 避免插件在未授权状态下继续使用过期 Token，满足 PowerX IAM 审计要求。Fail-open 会绕过宿主风控。 
- **Alternatives considered**: 
  - *缓存 token 继续使用* ➜ 无法满足强一致安全；被拒绝。
  - *自动降级到 Local IAM* ➜ 会产生影子帐户且无同步；被拒绝。

## 2. Local IAM 管理员凭证注入
- **Decision**: 仅当设置 `PLUGIN_IAM_ADMIN_EMAIL` + `PLUGIN_IAM_ADMIN_PASSWORD`（或 config.yaml 同名字段）时才创建默认管理员；缺失则向控制台报错并终止 migrate/seed。
- **Rationale**: 避免硬编码弱口令；允许 CI/开发灵活传参。符合“显式即配置”原则。
- **Alternatives considered**: 
  - *固定 admin/Admin123!* ➜ 高风险被滥用。
  - *交互式输入* ➜ 不适合 CI；被拒。
  - *随机生成并打印* ➜ 不便复现、难以自动化；被拒。

## 3. 模式判定环境变量
- **Decision**: 解析依据收敛为 `IAMMode`（`local`/`delegated`）与 `POWERX_PROXY`（`0`/`1`）。resolver 缓存结果并在日志中打印。
- **Rationale**: 与宿主注入模型保持一致，可在特殊调试时强制 override。
- **Alternatives considered**:
  - *仅依赖 POWERX_PROXY* ➜ 无法在宿主环境下临时启用本地模式；被拒。

## 4. Local IAM 数据存储
- **Decision**: 在 `skeleton/backend/go-gin/internal/entity/models/iam` 下新增 `Tenant`, `User`, `Role`, `Department`, `RolePermission` 等模型，沿用 Gorm + plugin schema；AutoMigrate 仅在 Resolver=local 时运行。
- **Rationale**: 与现有骨架风格一致，可通过 SQLite/PG 切换；保持与 Delegated 结构兼容。
- **Alternatives considered**: 
  - *使用内存/JSON 存储* ➜ 无法支持 CI、示例测试；被拒。

## 5. 观测指标与日志
- **Decision**: 暴露 `plugin_auth_login_total{mode}`、`plugin_auth_refresh_total{result}`、`plugin_auth_logout_total`、`plugin_iam_mode{mode}`、`plugin_iam_delegate_errors_total{type}`，并在 request_trace 中打印 `auth_mode`、`tenant_uuid`、`user_id`（遮蔽 PII）。
- **Rationale**: 与 PowerX 观测规范兼容，便于排障/审计。
- **Alternatives considered**: 仅在日志中附带信息但不出指标 ➜ 难以做实时告警；被拒。
