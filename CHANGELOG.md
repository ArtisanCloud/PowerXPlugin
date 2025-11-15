# Changelog

所有显著变更会记录在该文件中，并遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/) 与 [语义化版本号](https://semver.org/lang/zh-CN/) 规范。

## [Unreleased]

### Added
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

### Changed
- 脚手架与 CLI Manifest 模板默认声明 `iam.user.read/iam.role.read/iam.department.read` Scope，便于宿主在安装期提示所需权限
- `docs/guides/develop/standalone-mode.md` / `specs/005-plugin-auth/quickstart.md` 增补 Local IAM 环境变量与演练步骤
- Quickstart/Runbook 补充观测指标、验收命令与性能参考
- 脚手架模板 README 增加 Release 指引与多语言 TODO 提示
- Manifest Schema 追加菜单 `children` 递归定义，Skeleton/CLI 均可注册嵌套导航
- Quickstart / Standalone 指南补充多租户 CRUD 验证与延迟记录流程
- `px-plugin` 顶级帮助新增 doctor 命令说明，并提示 CLI/FAQ 文档入口；`px-plugin doctor` 输出分步骤进度，便于跟踪卡顿环节
- `px-plugin dev` 会读取 `~/.px-plugin/config.json` 与 `PX_DEV_TENANT` 等环境变量作为默认值（entry/tenant/dev-api/ignore/mTLS）

### Deprecated
- 无

### Removed
- 无

### Fixed
- 无

### Security
- 无
