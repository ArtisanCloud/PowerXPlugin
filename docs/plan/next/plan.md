# Next 对齐 Nuxt 计划（以页面与交互完整迁移为第一目标）

## 背景

当前仓库的管理端事实标准是 `skeleton/web-admin/nuxt`。`skeleton/web-admin/next` 目前仅有最小骨架（`app/layout.tsx`、`app/page.tsx`），尚未承载现有 Nuxt 的页面、交互、鉴权、状态管理与 E2E 场景。

本计划目标是将 Nuxt 管理端能力完整翻译到 Next（App Router），并保持对后端 API 契约与运行模式（Standalone / Host Proxy）的一致性。

## 原则

- **不修改** Nuxt 现有页面行为作为对齐基线（Nuxt 是权威实现）。
- Next 按 Nuxt 的路由、交互、字段与错误语义对齐，作为新增实现而非替换。
- API 契约以现有后端（Gin/FastAPI 已对齐）为准，不新增“仅 Next 可用”的私有接口。
- 产物路径遵循现有模板链路：`skeleton/` 为唯一真源，后续通过模板同步下发。
- 本轮实现默认对齐 **Go-Gin 后端**；除非定位为 Gin 侧缺陷，否则不改 Gin 业务代码。

## Go-Gin 协作约束（新增）

- 后端以 `skeleton/backend`（Gin）现有接口与中间件行为作为权威契约来源。
- Next 迁移优先通过前端路由、请求参数、错误解析与适配层完成兼容，不通过修改 Gin 逻辑“反向适配”。
- 若联调出现差异，按以下顺序处理：
  1. 先校验 Next 是否偏离 Nuxt 既有行为（URL、header、payload、鉴权时序）。
  2. 再校验 Nuxt 与 Gin 的既有联调结果是否稳定可复现。
  3. 仅当确认 Gin 存在真实缺陷（与既有契约不一致、返回语义错误、明显 bug）时，最小化修复 Gin。
- Gin 修复约束：
  - 仅修复缺陷，不做顺手重构或接口扩展。
  - 变更需附带回归用例（至少覆盖该缺陷路径）。
  - 修复后需验证 Nuxt 与 Next 双端均不回归。

## 目标

- Next 管理端具备与 Nuxt 等价的核心页面与流程（登录、模板 CRUD、IAM、Capabilities、Integration、Operations、Security）。
- Next 在 `insidePowerX` 与非代理模式下行为一致（路径、鉴权、API base、Bridge 交互）。
- Next E2E 覆盖 Nuxt 关键回归场景，确保迁移可验证、可发布。

## 非目标

- 不在本阶段重写后端业务逻辑与接口契约。
- 不在本阶段引入新的设计系统或替换既有产品视觉语义。
- 不在本阶段推动 Nuxt 下线；Nuxt 仍保留为对照实现与回归基线。

## 当前对齐进度（2026-03-13）

- Nuxt 页面存量：已覆盖 IAM、Templates、Capabilities、Integration、Operations、Security、Auth、测试页。
- Next 现状：仅最小骨架，可启动但无业务页面与状态层。
- 结论：Next 对齐尚未开始业务迁移，处于 Phase 0（基线盘点）阶段。

## 现状基线（Nuxt 对齐对象）

Nuxt 页面基线（`skeleton/web-admin/nuxt/app/pages`）：

- 基础与鉴权：`index`、`intro`、`users/login|register|forgot-password`
- 模板：`templates/index|crud|develop`
- IAM：`admin/iam/overview|members|roles|settings`
- 能力：`capabilities/Lifecycle|RegisterForm`、`powerx/capability-lab`
- 集成/运营/安全：`_p/com.powerx.plugins.base/admin/integration/*`、`operations/*`、`security/*`
- 调试与测试：`bridge-dev/*`、`tests/capability`

Nuxt 关键实现层：

- 状态层：`app/stores/**`（Pinia）
- API 层：`app/composables/api/**`
- 鉴权与桥接：`useAuth.ts`、`auth.global.ts`、`layouts/*`、`useHostBridgeAdapter.ts`
- E2E：`tests/e2e/**`（Playwright）

## Next 对齐清单（从 Nuxt 完整翻译）

### 1) 路由与页面结构对齐（Nuxt Pages → Next App Router）

建议目录（目标）：

```text
skeleton/web-admin/next/
  app/
    (public)/
      users/login/page.tsx
      users/register/page.tsx
      users/forgot-password/page.tsx
    (admin)/
      intro/page.tsx
      templates/page.tsx
      templates/crud/page.tsx
      templates/develop/page.tsx
      admin/iam/overview/page.tsx
      admin/iam/members/page.tsx
      admin/iam/roles/page.tsx
      admin/iam/settings/page.tsx
      capabilities/lifecycle/page.tsx
      capabilities/register/page.tsx
      powerx/capability-lab/page.tsx
      tests/capability/page.tsx
    _p/[pluginId]/admin/[...internal]/page.tsx
  components/
  hooks/
  lib/
    api/
    auth/
    bridge/
    stores/
```

对齐要求：

- 页面 URL、参数、入口可见性与 Nuxt 一致。
- `/_p/{pluginId}/admin/*` 与非代理模式路由均可访问。
- 测试页（`/tests/*`）保持独立，不依赖真实登录后端。

### 2) 布局与运行模式对齐（Default/Embedded）

- Nuxt `default` / `embedded` 布局语义迁移到 Next `layout.tsx` + route groups。
- `insidePowerX` 判定、Bridge token 请求、delegated banner 行为一致。
- Host 内嵌时只在插件管理路径使用 embedded 布局。

### 3) 鉴权与会话对齐（useAuth）

- 迁移 `access_token` / `refresh_token` / `expires_at` 存储协议。
- 迁移 fail-closed 逻辑：token 过期、refresh 失败、delegated 错误提示。
- 迁移路由守卫：公开路由、root-only 路由、登录跳转与 redirect 参数策略一致。

### 4) API Client 与错误语义对齐

- 迁移 `_client.ts`、`_base.ts`、`normalizeApiError.ts` 到 Next `lib/api`。
- 保持请求头注入、tenant 上下文注入、错误 envelope 解析一致。
- API base 解析规则对齐 Nuxt `runtimeConfig.public.apiBaseUrl` 语义。

### 5) 状态管理对齐（Pinia → Zustand/Redux Toolkit）

推荐：Zustand（轻量，迁移成本低）。

- 一对一迁移核心 store：`user`、`role`、`permission`、`iam`、`department`、`operations`。
- 保留“列表 + 详情 + loading + error + pagination”状态结构。
- 保留 `fetch* / create* / update* / delete*` 命名语义，减少 API 层重写风险。

### 6) 组件与交互对齐（Nuxt UI → React UI）

- 先以行为一致为第一优先：筛选、分页、弹窗、抽屉、表格、提示。
- 关键交互不变：
  - 模板 CRUD 的创建/编辑/克隆/删除流。
  - IAM 角色权限与成员抽屉流。
  - Capability 调用与本地调试流（含 trace/status/error 展示）。
- 保留 `data-testid` 标识策略，保障 E2E 可平移。

### 7) 国际化对齐（i18n）

- 迁移 `nuxt/i18n/locales/{zh,en}.json` 到 Next i18n 方案（如 `next-intl`）。
- key 命名保持一致，避免页面迁移时文本绑定重写。

### 8) 测试与质量门禁对齐

- Playwright 场景从 Nuxt 复制并适配到 Next（路径与 locator 优先复用）。
- 最小回归集：
  - Auth 登录态与重定向
  - 模板 CRUD 主链路
  - IAM 成员/角色主链路
  - Capability invocation playground
- CI 要求：`lint + unit + e2e + build` 全通过。

## 分阶段实施

### Phase A — 基线冻结与脚手架加固

- 固化 Nuxt 对齐清单（页面、接口、状态、E2E）。
- 搭建 Next 运行时基础：布局、鉴权骨架、API client、i18n、store 框架。
- 建立 `data-testid` 与组件编码规范。

### Phase B — P1 页面迁移（业务可用最小面）

- 迁移 `users/login`、`intro`、`templates/index|crud`。
- 打通 token 生命周期与模板 CRUD API。
- 建立第一批 E2E（登录 + 模板 CRUD）。
- 联调策略：优先修正 Next 请求与状态处理，不改 Gin；仅对已确认 Gin 缺陷打补丁。

### Phase C — IAM 与能力中心迁移

- 迁移 `admin/iam/*` 与 `powerx/capability-lab`、`capabilities/*`。
- 对齐权限可见性、抽屉/弹窗交互与错误提示语义。
- 补齐对应 E2E 场景。
- 若出现权限链路差异，先核对 Nuxt 既有行为与请求，再决定是否进入 Gin 缺陷修复流程。

### Phase D — 集成/运营/安全页面迁移

- 迁移 `integration/*`、`operations/*`、`security/*` 页面。
- 对齐复杂表格、时序信息、详情面板与空态逻辑。

### Phase E — Host 路径与发布收敛

- 完成 `/_p/{pluginId}/admin/*` 反代路径适配与嵌入模式验证。
- 完成构建产物、模板同步与文档更新。
- 执行全量回归并形成切换建议。

## 验证清单

- Next 本地启动成功，首页/登录页/模板页可访问。
- `insidePowerX=true/false` 两种模式路由与鉴权行为一致。
- 与后端联调时，Next 请求路径、请求体、响应解析与 Nuxt 一致。
- Nuxt 关键 E2E 用例在 Next 侧可复现并通过。
- 构建产物可用于后续模板同步与 `px-plugin init` 生成。

## 风险与约束

- **最大风险**：UI 框架替换导致“视觉像、语义不一致”（权限、错误码、空态、分页）。
- **技术风险**：SSR/CSR 边界与 token 存储策略不一致引发鉴权回归。
- **流程风险**：Nuxt 与 Next 双栈并行期间，接口变更可能导致双端漂移。
- **协作风险**：联调期若直接修改 Gin 以迎合 Next，可能破坏 Nuxt 既有稳定行为。

缓解策略：

- 以 Nuxt 行为回归用例为硬约束，先行为后视觉。
- 对关键页面保留“迁移对照表”（Nuxt 文件 → Next 文件 → 验证用例）。
- 在 CI 中同时跑 Nuxt 与 Next 的核心 E2E 子集，防止单端回归。
- 新增“Gin 缺陷准入门槛”：必须可复现、可最小修复、可双端回归验证。

## 实施状态（Next 对齐）

- Phase 0（基线盘点）：进行中
- Phase A（基础骨架）：未开始
- Phase B（P1 页面迁移）：未开始
- Phase C（IAM/Capabilities）：未开始
- Phase D（Integration/Ops/Security）：未开始
- Phase E（发布收敛）：未开始
