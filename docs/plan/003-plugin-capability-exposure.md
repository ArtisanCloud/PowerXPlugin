# 插件能力暴露实现计划（PowerXPlugin → PowerX）

> 目标：让 PowerXPlugin 侧可以用规范化的 manifest / contracts / API 通道向 PowerX 暴露可调用能力，并满足 SCN-INT-PLUGIN-CAPABILITY-001 的建模、审核、暴露与生命周期治理要求。

---

## 1. 背景与目标

- **现状**：Local install 已验证菜单 & dist 发布，但未覆盖能力声明 (`capabilities.provides`)、能力模型 Schema、租户授权/额度、暴露通道等核心场景。
- **需求来源**：
  - `SCN-INT-PLUGIN-CAPABILITY-001` 及子场景（Modeling / Review / Exposure / Lifecycle）。
  - 标准文档：`docs/standards/powerx-plugin/integration/02_capabilities_and_schema/*`。
  - 用户诉求：插件必须向 PowerX 暴露可消费的 API/Agent 能力，并在 manifest 中准确描述。
- **目标**：在 PowerXPlugin 仓库提供一套端到端方案，使插件开发者能：
  1. 定义能力契约（命名、Schema、示例、RBAC）；
  2. 在构建 & 发布流程中校验能力；
  3. 通过 CLI/工具向 PowerX 注册能力并生成 `.pxp`；
  4. 配合宿主完成暴露配置、租户授权与生命周期治理。

成功度量（对齐场景验收）：
- 能力注册表单/CLI 5 分钟内完成建模并生成能力 ID；
- 审核状态驱动本地/CI 流水（阻断未审核能力发布）；
- 暴露配置后的 API/Agent 通道 3 分钟内可调用；
- 版本变更具备差异报告 + 灰度通知。

---

## 2. 范围与交付

| 模块 | 范围 | Out of Scope |
|------|------|--------------|
| Manifest & Contracts | 提供脚手架模板、Schema/descriptor 骨架、生成脚本、lint/验证 | PowerX 主站的审批 UI |
| CLI & Tooling | `px-plugin` 子命令：能力校验、注册、暴露同步、租户授权辅助 | PowerX Core 的内部 API 实现 |
| Docs & Samples | 指南/示例（Go/Nuxt 模板）覆盖能力声明、Schema、Agent Tool Integration | Marketplace 计费流程 |
| Tests & CI | Lint/Schema/manifest 校验、capability 合同测试、`.pxp` 集成 | PowerX 主仓集成测试 |

---

## 3. 实施阶段

### Phase 1 — Manifest & Contracts 基线（Week 1-2）
1. **模板增强**（完成 ✅ 部分已在 `scaffold/templates/plugin.yaml.tmpl` 更新菜单 & RBAC）：
   - 为 `capabilities.provides`、`tools`、`agent_tools` 引入真实示例（API + Agent tool）。
   - 自动生成 `contracts/capabilities/*.yaml` 与 `contracts/schema/*`（输入/输出 JSON Schema）。
2. **Schema 校验脚本**：
   - 在 `scripts/capabilities/` 添加 `validate-capabilities.mjs`（或 Go 工具）；
   - `make dist` / `make validate` 时运行 Schema/命名冲突检测。
3. **Docs 更新**：
   - 在 `docs/guides/develop/cli-plugin-tutorial.md`、`docs/guides/publish/local-install.md` 补充「能力声明」步骤；
   - 新增 FAQ：能力命名、Schema 路径、RBAC 对齐。

### Phase 2 — 能力注册与审核衔接（Week 3-4）
1. **CLI 命令**：
   - `px-plugin capabilities init`：基于模板生成能力描述文件。
   - `px-plugin capabilities lint`：校验 manifest/Schema/descriptor，生成报告。
   - `px-plugin capabilities submit`：调用 PowerX Dev API 的 `/internal/plugins/capabilities`（mock/接口描述），写入 `.audit`.
2. **审计/状态管理**：
   - 在 `.px-plugin/` 记录能力 ID、版本、审核状态；
   - `make dist`/`make pack` 阶段若状态≠approved，则警告/阻断。
3. **文档**：
   - `docs/guides/publish/capabilities.md`：描述 CLI 使用、PowerX 审核流对接。

### Phase 3 — 暴露通道 & 租户授权（Week 5-6）
1. **通道描述**：
   - 在 manifest 中新增 `exposure.channels`（REST/GraphQL/Webhook/Workflow/SDK）草案；
   - CLI 读取该配置并调用 `PATCH /internal/plugins/capabilities/{id}/exposure`。
2. **租户授权辅助脚本**：
   - `px-plugin capabilities quota`：为 Dev/Stage 环境快速分配示例租户额度；
   - 生成 Postman/SDK bundle（参考 `docs/use_cases/.../SCN-INT-PLUGIN-CAPABILITY-EXPOSURE-001.md`）。
3. **E2E Demo**：
   - 在 `examples/com.powerx.demo` 中实现一个可调用的 REST + Agent Tool 能力；
   - 提供 Playwright/Go tests 调用 `/_p/<plugin>/api/v1/...` 并验证 Schema。

### Phase 4 — 生命周期与灰度支持（Week 7-8）
1. **差异报告工具**：
   - `px-plugin capabilities diff --from manifest.old --to manifest.new` 输出 Schema/RBAC 变化；
   - 生成 `release/capabilities-change-report.md` 用于提交到 PowerX。
2. **通知/灰度规划**：
   - 在 CLI 中生成灰度计划模板（订阅方、渠道、窗口、回滚策略）；
   - 文档说明如何配合 PowerX API（`POST /capabilities/{id}/versions/.../plan`）。
3. **Telemetry Hook**：
   - `make local-install`/`pack` 记录 capability 版本入 `audit.capability.registry.updated`；
   - 提供脚本读取 PowerX logs 以验证通知覆盖率。

---

## 4. 技术细节与关键决策

- **Manifest 扩展**：
  - `frontend.admin` 与 `rbac.resources` 已对齐；下一步在 `plugin.yaml` 增加 `capabilities`, `agent_tools`, `dependencies`.
  - 设计 `exposure` 段落（示例）：
    ```yaml
    exposure:
      channels:
        - type: rest
          entrypoint: ${POWERX_PLUGIN_HTTP_BASE:-/api/v1}/templates
          auth: jwt
          rate_limit: { rps: 50, burst: 100 }
        - type: agent_tool
          capability: {{ .PluginID }}.template.create
    ```
- **Contracts 布局**（对齐标准文档）：
  ```
  contracts/
    capabilities/<capability>.yaml
    schema/input|output/<capability>.json
  ```
  CLI 根据 manifest 自动生成 stub 并同步版本号。
- **验证栈**：
  - Node scripts + Go validator（`gojsonschema`）双向校验；
  - Git Hooks / CI job `make capabilities-verify`.
- **与 PowerX Dev API 的接口**：
  - CLI 仅模拟/调用接口契约；若后端未就绪，先实现 mock server 或录制 fixture；
  - 所有 API 调用写入 `~/.px-plugin/audit/capability-*.log`。

### 4.1 能力分层抽象（从声明到执行）

| 层级 | 作用 | 产物/文件 |
|------|------|-----------|
| **能力层（Capability）** | PowerX 可调用的最小语义单元，带命名空间与 RBAC。 | `plugin.yaml` `capabilities.provides` |
| **通道层（Exposure Channel）** | 说明能力通过 REST / Agent Tool / Workflow / SDK 等何种方式暴露。 | `plugin.yaml.exposure.channels` |
| **契约层（Contract Schema）** | 输入/输出结构、错误码与示例，供 CLI 校验与 PowerX 生成 SDK。 | `contracts/capabilities/*.yaml` + `contracts/schema/input|output/*.json` |
| **执行层（Handler）** | 插件内部的最小执行单元（Go handler、Nuxt API route 等），真正处理请求。 | `backend/internal/handlers/**` |
| **宿主层（PowerX Integration）** | 能力注册、审批、授权、灰度与审计的统一入口。 | PowerX 能力中心 + `/internal/plugins/capabilities/**` |

> 关键结论：**所有能力最终都归结为插件 handler**。Manifest/contract/CLI 只是把 handler 的存在、Schema、通道告知 PowerX。

### 4.2 Handler 注册策略

- **目录约定**：建议放在 `backend/internal/handlers/capabilities/<domain>/<action>_handler.go`，使能力 ID 可以通过路径自动推导。
- **绑定方式**：
  - 早期使用显式注册：`framework.RegisterCapabilityHandler("com.powerx.demo.template.create", template.CreateHandler)`;
  - 后续通过 `framework.AutoRegisterHandlers("backend/internal/handlers/capabilities", pluginID)` 扫描 `_handler.go` 文件并生成 ID → handler 的映射。
- **最小示例**：
  ```go
  func CreateTemplateHandler(ctx context.Context, req *framework.Request[Input]) (*framework.Response[Output], error) {
      // TODO: 业务逻辑
      return framework.NewResponse(Output{ID: "tpl_x"}), nil
  }
  ```
  对应 manifest:
  ```yaml
  capabilities:
    provides:
      - id: com.powerx.demo.template.create
        type: rest
        entrypoint: /api/v1/templates
        input_schema: contracts/schema/input/template.create.json
        output_schema: contracts/schema/output/template.create.json
  ```

### 4.3 能力注册文件与 CLI

| 层级 | 存储位置 | CLI 触发 |
|------|----------|----------|
| 声明（Manifest） | `plugin.yaml` | `px-plugin capabilities lint` 校验 ID/通道 |
| 契约（Contracts） | `contracts/capabilities/*.yaml` + `contracts/schema/*.json` | `capabilities init` 生成、`lint` 校验 |
| 状态（Audit/Registry） | `.px-plugin/capabilities.json`, `.px-plugin/audit/*` | `capabilities submit` / `quota` 写入 |

CLI 职责：
- `init`：生成 manifest & contract stub、对应 handler 模板。
- `lint`：校验命名、Schema、RBAC 对齐。
+- `submit`：把契约推送至 PowerX 能力注册 API，写入 `.px-plugin` 状态。
- `quota`/`diff`：辅助租户授权与版本灰度。

---

## 5. 依赖与风险

| 风险 | 影响 | 缓解 |
|------|------|------|
| PowerX 侧 API/流程未就绪 | CLI 提交/暴露命令无法联调 | 提供 mock server + 配置开关；先落地 manifest/Schema 校验 |
| 能力命名/Schema 未统一 | 宿主注册失败或冲突 | 引入 lint 规则、命名占位符，CI 强校验 |
| 文档/示例滞后 | 开发者无法照搬 | 在 `docs/guides` 建立 step-by-step；提供 demo repo |
| 审核/暴露过程耗时 | 开发环境卡住 | 支持 Dev 模式“跳过审核”的 flag，并在 CLI 警告 |

---

## 6. 里程碑 & 输出

| 时间 | 里程碑 | 主要输出 |
|------|--------|---------|
| Week 2 | Phase 1 完成 | 新模板、Schema 生成工具、文档更新 |
| Week 4 | Phase 2 完成 | CLI lint/submit、审计记录、指南 |
| Week 6 | Phase 3 完成 | 暴露通道配置、租户授权脚本、示例插件 |
| Week 8 | Phase 4 完成 | 差异报告工具、灰度计划模板、Telemetry Hook |

---

## 7. 后续工作

- 与 PowerX 主仓对齐接口契约与 Feature Flags；
- 在 QA / DevRel 流水中加入 `px-plugin capabilities <cmd>`；
- 为 Marketplace/Agent 场景撰写复用指南；
- 跟踪 Open Issues（多语言字段、敏感数据加密通道等）并回填到计划表。

> 若计划需要扩展到特定插件（例如 `com.powerx.helloworld`），可在 Phase 3 Demo 基础上复制落地；文档与工具应覆盖 Go/Nuxt 模板并提供清晰的落地步骤。
