# Changelog

所有显著变更会记录在该文件中，并遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/) 与 [语义化版本号](https://semver.org/lang/zh-CN/) 规范。

## [Unreleased]

### Added
- `px-plugin iam export/seed` CLI 子命令，可离线导出租户/角色/成员 JSON，并在 Standalone 模式快速重置默认管理员；`Quickstart`/`Runbook` 新增示例与推荐参数。
- Delegated IAM Auth Proxy（Go）与 `/api/v1/auth/*` 公共路由，覆盖 login/refresh/logout/me/context 及 `docs/contracts/manifest.yaml` 示范清单
- `useAuth` Vitest 单测与 `tests/e2e/auth-delegated.spec.ts` Playwright 场景，验证 token fallback/Fail-Closed 交互
- Local IAM Directory（`internal/services/iam/local_store.go`）与 RefreshToken 模型，可在 `POWERX_PROXY=0` 下完成登录/刷新/登出
- 新增 Playwright 本地登录用例 `tests/e2e/auth-local.spec.ts`，并输出 `docs/operations/runbooks/auth-troubleshooting.md`
- 指南 `docs/guides/develop/auth.md`，整理 Delegated/Local 流程、指标、性能与排障
- 初始发布流程文档与 Release Workflow (`docs/release.md`, `.github/workflows/release.yml`)
- CLI 生成物示例 `examples/starter/`，对齐 Phase 6 用户故事
- Skeleton Templates CRUD 栈（后端内存仓储 + 前端页面 + `useTemplateApi` 示例）并同步至 CLI 脚手架模板
- `px-plugin init` 输出 Nuxt 项目新增 `lint` / `test:e2e` 占位脚本，便于后续扩展质量闸门
- Go CLI 故障排查手册 `docs/guides/cli/go-cli-troubleshooting.md`，涵盖 doctor、SSE、跨平台脚本 FAQ
- `px-plugin package` / `px-plugin publish` 完成真实构建与上传：新增 `internal/package` builder + metadata、`internal/publish` 客户端、CLI flags（`--channel`/`--artifact`/`--publish-api`），并在 smoke 流程中校验。
- `~/.px-plugin/config.json` 支持 `publishApi.{baseUrl,apiKey}`，`docs/guides/develop/go-cli-dev-watch.md`、`docs/guides/publish/online.md`、`specs/005-plugin-auth/quickstart.md` 补充 package/publish 步骤与配置示例。
- `tools/cli/internal/package/builder_test.go`、`tools/cli/internal/publish/client_test.go` 以及 `scripts/testing/smoke.sh` 新增测试覆盖，确保缺失 artefact/Registry 异常可检测，并在 CI 中实测 package/publish。
- Capabilities 观测层：新增 `CapabilityMetrics`、`capability.catalog.sync_status`、`capability.workflow.async_duration` 指标，并在安装阶段自动打点；`docs/plan/006-plugin-capability.md`、`docs/guides/publish/capabilities.md`、`docs/guides/quickstart.md` 补充多协议导出、默认 `CAP_MANIFEST` 及验证流程。
- 能力消费治理（[docs/plan/009-consume-powerx-capability.md](docs/plan/009-consume-powerx-capability.md)）：新增 `scripts/capabilities/contract-digest.mjs` 产出 `dist/capability-contracts.json`，并在 Gateway Client 中加载/校验契约哈希与期望版本，警告 Admin UI 需要升级。

### Changed
- Manifest/RBAC schema 统一采用 `com.powerx.plugins.base` ID，并在清单中声明 IAM 菜单路由，便于宿主自动生成导航。
- 脚手架与 CLI Manifest 模板默认声明 `iam.user.read/iam.role.read/iam.department.read` Scope，便于宿主在安装期提示所需权限
- `docs/guides/develop/standalone-mode.md` / `specs/005-plugin-auth/quickstart.md` 增补 Local IAM 环境变量与演练步骤
- Quickstart/Runbook 补充观测指标、验收命令与性能参考
- 脚手架模板 README 增加 Release 指引与多语言 TODO 提示
- Manifest Schema 追加菜单 `children` 递归定义，Skeleton/CLI 均可注册嵌套导航
- Quickstart / Standalone 指南补充多租户 CRUD 验证与延迟记录流程
- `px-plugin` 顶级帮助新增 doctor 命令说明，并提示 CLI/FAQ 文档入口；`px-plugin doctor` 输出分步骤进度，便于跟踪卡顿环节
- `px-plugin dev` 会读取 `~/.px-plugin/config.json` 与 `PX_DEV_TENANT` 等环境变量作为默认值（entry/tenant/dev-api/ignore/mTLS）
- `px-plugin capabilities quota`（[docs/plan/009-consume-powerx-capability.md](docs/plan/009-consume-powerx-capability.md)）支持 `--manifest` 自动解析 `capabilities.provides` 并生成 Postman/HTTP 示例，Quickstart 新增配额配置指引。

### Deprecated
- 无

### Removed
- 无

### Fixed
- 无

### Security
- 无
