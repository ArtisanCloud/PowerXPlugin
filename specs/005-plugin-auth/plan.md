# Implementation Plan: Plugin Auth Integration

**Branch**: `005-plugin-auth` | **Date**: 2025-11-14 | **Spec**: `/specs/005-plugin-auth/spec.md`
**Input**: Feature specification from `/specs/005-plugin-auth/spec.md`

## Summary

实现插件侧统一的登录/logout/token 生命周期能力，使其在 PowerX 宿主模式下委托宿主 IAM，在独立/本地模式下可使用本地 IAM 表。前端 Nuxt 复用宿主 `useAuth` 语义（localStorage + cookie +刷新），后端 Go 提供 `IAMDirectory` 接口，包含 Delegated（通过 `POWERX_CORE_ENDPOINT` + `POWERX_AUTH_TOKEN`）与 Local（自持 IAM 模型）双实现，并以 fail-closed 策略处理 Core 不可用场景。

## Technical Context

**Language/Version**: Go 1.24 (backend), TypeScript 5 / Nuxt 4.2 (web admin)
**Primary Dependencies**: Gin, Gorm, `$fetch`/Nitro、Pinia、`@nuxt/ui`, PowerX framework middleware、`@artisan-cloud/plugin-framework-*`
**Storage**: 插件数据库（SQLite/PostgreSQL 由配置决定）中的业务表 + 新增 IAM 表（Local 模式）；Delegated 模式仅读写宿主 API
**Testing**: `go test ./...`, `npm test`, `npm run lint`, Playwright (登录/刷新/登出)
**Target Platform**: Linux (PowerX host pods) + 浏览器（Chrome/Edge/Safari 最新版）
**Project Type**: Web（Go backend + Nuxt admin SPA）
**Performance Goals**: Delegated 登录 p90 ≤ 2s；Token 刷新成功率 ≥98%；登出清理耗时 <1s；本地模式迁移 + 种子 ≤60s
**Constraints**: Must fail closed when Core auth不可用；token 仅存 localStorage+cookie；模板需可通过 `px-plugin init` 即可运行；需要记录 `plugin_provider_mode` 指标
**Scale/Scope**: 预计单租户上千活跃用户、几十插件实例；需要支持多租户 Token / Tenant Header 同步

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. 双重使命仓库**：✅ 方案所有改动都会同步 Skeleton 与模板，并在 Quickstart 中强调 `npm run sync:templates`。
- **II. 契约优先兼容性**：✅ Phase 1 输出 OpenAPI (`/specs/005-plugin-auth/contracts/auth.openapi.yaml`) 并要求后端/前端消费。
- **III. Go + Nuxt 基线**：✅ 技术栈仅使用 Go Gin + Nuxt 4；无额外语言。
- **IV. 脚手架与 CLI 纪律**：✅ 快速上手文档要求 `px-plugin init` 校验，并同步 CLI 模板环境变量。
- **V. 透明交付**：✅ Spec/plan/research/data-model/quickstart 均记录真实状态；无闸门豁免。
> 结论：无宪章违规，可进入 Phase 0。

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)
<!--
  ACTION REQUIRED: Replace the placeholder tree below with the concrete layout
  for this feature. Delete unused options and expand the chosen structure with
  real paths (e.g., apps/admin, packages/something). The delivered plan must
  not include Option labels.
-->

```text
```text
backend/
├── cmd/
│   ├── plugin/                # 主入口，负责装配 router/middleware
│   └── database/              # migrate/seed/refresh
├── internal/
│   ├── config/                # 读取 env + YAML
│   ├── router/                # Gin 注册，含 JWT/RBAC 中间件
│   ├── middleware/            # auth_jwt、request_trace
│   ├── services/
│   │   ├── authproxy/         # 新增 IAMDirectory 实现
│   │   └── ...                # 业务域
│   ├── entity/models/         # marketplace/runtime/... + 新 IAM 模型
│   ├── transport/http/        # public/protected handler（新 auth handler）
│   └── manifestx/             # 菜单/RBAC 元数据
└── tests/                     # Go 单测

frontend/ (skeleton/web-admin/nuxt)
├── app/
│   ├── composables/
│   │   ├── useAuth.ts         # 新增
│   │   └── api/services/      # authService.ts
│   ├── middleware/auth.global.ts
│   ├── pages/users/**         # login/register/forgot
│   ├── plugins/auth.client.ts # initAuth
│   └── stores/user.ts
└── tests/
    └── playwright/auth.spec.ts

scaffold/templates/** + tools/cli/** mirror skeleton
```

**Structure Decision**: 采用仓库现有 “backend + frontend + scaffold/cli” 结构；Auth 能力主要落在 `skeleton/backend/go-gin` 与 `skeleton/web-admin/nuxt`，再通过 `npm run sync:templates` 复制到 `scaffold/templates` 与 `tools/cli`.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| _None_ | - | - |

## Phase 0 – Research & Unknown Resolution

| Task | Owner | Notes |
|------|-------|-------|
| 明确 Delegated 模式下 Core 故障策略（fail-closed vs fallback） | Backend | 参考 PowerX IAM 要求 → fail-closed |
| 确认 Local 模式管理员凭证注入方式 | Backend | 使用 `PLUGIN_IAM_ADMIN_*` env / config，缺失则报错 |
| 统一模式切换环境变量（`POWERX_PROVIDER_MODE` + `POWERX_PROXY`） | Backend | 记录速记表，供 Resolver 使用 |
| Token 存储 & 同步策略 | Frontend | 沿用宿主 localStorage + cookie + storage event |
| Observability 指标与日志字段 | Backend | `plugin_provider_mode`、`plugin_auth_login_total` 等 |

> 产物：`/specs/005-plugin-auth/research.md`

## Phase 1 – Design, Contracts & Templates

| Deliverable | Description |
|-------------|-------------|
| `data-model.md` | 描述 AuthTokens、TenantContext、User、Tenant、Role、Department、ProviderModeSetting 等实体及约束 |
| `contracts/auth.openapi.yaml` | 定义 `/api/v1/auth/{login,refresh,logout,me/context}` 请求/响应、header、错误模型 |
| `quickstart.md` | 引导开发者在 Delegated 与 Local 模式间切换、配置 env、运行测试（Go & Nuxt） |
| Skeleton 变更 | `useAuth`、`authService`、middleware、后端 `IAMDirectory`、router/public handler、migrate 拆分 |
| Templates/CLI | 执行 `npm run sync:templates`，更新 `plugin.yaml.tmpl`（注入 `POWERX_CORE_ENDPOINT`、`POWERX_AUTH_TOKEN` 等） |
| Agent Context | `.specify/scripts/bash/update-agent-context.sh codex` 记录新增技术/约束 |

完成后需重新检查宪章要求（尤其是契约 + CLI 同步）。

## Phase 2 – Implementation Preparation

| Task | Details |
|------|---------|
| 细化任务拆解 | 交给 `/speckit.tasks`：前端、后端、模板、测试、文档、CI |
| 定义验收测试 | Playwright login/refresh/logout，Go 单测覆盖 ProviderResolver、AuthProxy |
| 集成计划 | 先在 Skeleton 验证，再同步模板 + CLI，最后运行 `px-plugin init` 生成样例并 smoke test |

---

## Phase 3 – CLI Packaging & Publish Enablement

**Goal**：补齐 `px-plugin package`/`px-plugin publish`，让开发者可以一键打包 artefacts 并上传到自建 PowerX Registry，形成从 Auth → 发布 → 安装 → Dev 热加载的闭环。

### Deliverables

| Deliverable | Description |
|-------------|-------------|
| CLI Package Pipeline | 在 `tools/cli/cmd/package.go` 等处实现真实打包：读取 `package.json`、`go.mod`，执行 `npm --prefix <frontend> run build` 与 `go build ./backend/cmd/plugin`（可用 `--frontend-dir`、`--backend-dir` 覆盖），收集 dist、后端二进制、manifest/RBAC/metadata，并输出到 `.px-plugin/build/<timestamp>/package.tar.gz`。 |
| Package Metadata & Validation | 生成 `metadata.json`（版本、channel、hash、commit、CLI 版本、artefact 概览）与 `manifest.json`、`rbac.json`，并对输出文件做 SHA256 校验，供 publish/Registry 校验。 |
| CLI Publish Client | 在 `tools/cli/cmd/publish.go` 实现 `POST {publishApi.baseUrl}/internal/plugins/releases` 上传 package + metadata；支持 `--channel`、`--notes`、`--artifact <path>`、`--publish-api`、`--publish-token`，解析 PowerX envelope 返回 `publishId` 与审核链接。失败时输出 remediation。 |
| Config/Docs 更新 | `~/.px-plugin/config.json` 增加 `publishApi.{baseUrl,apiKey}`，`docs/guides/develop/go-cli-dev-watch.md`、`docs/guides/publish/online.md`、`specs/005-plugin-auth/quickstart.md` 补充 package/publish 步骤与排障；`CHANGELOG.md` 记录 CLI 新能力。 |
| Tests & CI | `tools/cli/internal/package/builder_test.go`、`publish_client_test.go` 等单测；在 CI 中新增 smoke job（可用 httptest/mock server）验证 package/publish。 |

### Exit Criteria

1. `px-plugin package --entry examples/com.powerx.demo` 生成 `.px-plugin/build/<timestamp>/package.tar.gz`、`metadata.json` 并列出 artefacts/hashes。
2. `px-plugin publish --entry examples/com.powerx.demo --channel dev --notes "feat"` 在带有 mock Registry 的环境中返回 `publishId`，PowerX Admin 可看到待审核版本；缺少配置时 CLI 给出可操作提示。
3. Quickstart/Go CLI 文档加入完整“package → publish → install → dev”步骤，运行 `px-plugin doctor --check-devapi` 验证配置后能顺利热加载。

---

## Phase 4 – Delegated UX & Template RBAC Hardening

**Context**：在 Spec 中新增的两个需求需要单独跟进：

1. **Delegated Token 失效体验**：`insidePowerX` 环境下 token 失效不可再跳 `/users/login`，而是展示宿主模式专属提示，并在宿主重新注入 token 后恢复。
2. **Template CRUD RBAC**：模板示例必须声明/输出 RBAC 资源；Standalone 模式 enforce，Delegated 模式暴露给宿主 IAM 并让前端根据权限/模式调整 UI。

### Deliverables

| Deliverable | Description |
|-------------|-------------|
| Delegated Token UX | `useAuth` / `auth.global.ts` 分支逻辑：Delegated 模式触发 `failClosed` 时只清理 token + 记录错误提示，不调用 `navigateTo('/users/login')`；提供 Banner 组件/Store，允许“重试”触发宿主 token 注入；Playwright delegated 用例覆盖；文档更新。 |
| Token Rehydration Flow | 与 `px-bridge` 集成：当宿主 `postMessage` 发送新 token 时重新调用 `initAuth()`，关闭 Banner。 |
| Template RBAC Server | 新增 `internal/transport/http/admin/templates/rbac.go` + registry 注入；manifest/RBAC 输出 `base.templates.read/manage`；Go 单测覆盖。 |
| Template RBAC Frontend | `pages/templates/**` 根据 `useAuth`/`useRuntimeConfig` 检查权限/模式，隐藏或禁用 CRUD 操作，Delegated 模式显示“宿主控制权限”提示；文档记录差异。 |

### Exit Criteria

1. Delegated 模式下刻意删除 token 或让 refresh 返回 401/503，页面不跳登录且出现提示；宿主重新注入 token 后无需刷新即可恢复。
2. `GET /api/v1/admin/rbac`、manifest 和文档中均包含模板资源；Standalone 模式未授权时模板 CRUD 返回 403；Delegated 模式 manifest 依旧暴露资源但由宿主判定。
