# Feature Specification: PowerX Publish Hub Distribution Spec

**Feature Branch**: `004-publish-hub-spec`  
**Created**: 2025-11-07  
**Status**: Draft  
**Input**: User description: "请根据docs/use_cases/_from_hub/SCN-PUBLISH-HUB-001路径下的所有需求用例文档，生成spec相关文档"
**Additional Inputs (2025-11-20)**: docs/use_cases/_from_hub/SCN-DEV-PLUGIN-INIT-001, docs/use_cases/_from_hub/SCN-DEV-PLUGIN-DEBUG-001, docs/use_cases/_from_hub/SCN-DEV-PLUGIN-PUBLISH-001（覆盖 CLI 工程初始化、宿主模拟调试与发布/Marketplace 主场景）

## Context & Scope

- **Problem Statement**: Publish Hub must connect plugin developers, Marketplace 审核员, and 租户管理员 through a single, auditable workflow that spans local 热加载、在线发布、离线导入与生产启用，确保版本同步可控、审核可追、安装可回滚。
- **In Scope**: CLI 打包/发布体验（`px-plugin dev/publish/dist` 系列）、`.pxp` 包定义与签名策略、Marketplace 在线/离线审核队列、租户安装/灰度/回滚入口、Dev API 日志与 SSE 通知、Telemetry & 审计链路、跨渠道 SLA。
- **Out of Scope**: 插件业务功能实现、第三方支付结算、非 PowerX 平台分发、底层运行时改造。
- **Success Signals**: 审核 SLA（≤4h 在线 / ≤1 工作日离线）、热加载 reload ≤2s、版本 30 分钟内同步到订阅租户、离线包 integrity 验证 100%、安装失败 5 分钟内可回滚、Telemetry 覆盖率 ≥98%。

## Assumptions & Constraints

1. Feature Flags `PX_PLUGIN_DEV_MODE`, `PX_PLUGIN_PUBLISH`, `PX_MARKET_PUBLISH_ENABLED`, `PX_MARKET_OFFLINE_UPLOAD`, `PX_PLUGIN_HUB_ENABLED` 已在目标环境正确配置。
2. 插件工程满足 Go/TypeScript 版本要求，且构建产物、manifest、签名材料、`publish.yml` 与 `dist.config` 在执行 CLI 前可用。
3. Marketplace 审核团队具备在线/离线双队列权限，可基于 CLI 提交的 artefact 独立完成签名验证、安全扫描与审批，并对 SLA 负责。
4. 租户管理员只通过 PowerX Web Admin/API 安装插件，具备灰度和回滚上一版本的权限；多租户广播由 Publish Hub 统一调度。
5. 网络受限客户允许人工传递 `.pxp` 包与审计材料；CLI 输出的 integrity 报告是唯一可信库存档。
6. Dev API 网关支持 register/reload/delete 接口、mTLS 与 SSE 日志；CLI 能访问 `px auth configure` 产出的凭据。
7. 离线 `.pxp` 包在存储与传输过程中必须使用临时对称密钥加密，CLI 需生成该密钥并用 Marketplace 公钥封装后随 artefact 上传，审核节点解密后再执行签名与 hash 校验。

## User Scenarios & Testing *(mandatory)*

### User Story 0 - Plugin scaffolding & compliance bootstrap (Priority: P1)

> 来源：docs/use_cases/_from_hub/SCN-DEV-PLUGIN-INIT-001（CLI 初始化、团队协作、第三方导入）

PowerX 需要在 1 分钟内通过 `px-plugin init` 生成标准化工程，串联模板选择、依赖安装、许可证扫描、Git 注册与 `px-plugin doctor` 团队诊断，确保第三方源码导入也能留痕和审批。

**Why this priority**: 没有统一脚手架和合规守护，后续 dev/publish 流程无法保证 artefact 与权限一致性。

**Independent Test**: 执行 `px-plugin init demo --template react-dashboard`，观察模板渲染、依赖安装、SBOM/扫描、Git 注册与 `publish.yml`/`manifest` 生成；随后由团队成员跑 `px-plugin doctor`，第三方源码导入触发 `plugin-import-audit`。

**Acceptance Scenarios**:

1. **Given** 开发者启用 `PX_PLUGIN_SCAFFOLD_V2`，**When** 运行 `px-plugin init --template react-dashboard --org demo`, **Then** CLI 在 60 秒内生成目录、写入 manifest/权限声明、执行依赖安装并通过 `POST /internal/plugins/bootstrap/validate` 注册 Git 仓与 CI，返回扫描报告。
2. **Given** 团队成员克隆仓库，**When** 运行 `px-plugin doctor --fix`, **Then** 命令校验 Node/Go 版本、依赖与 Feature Flag，输出 ≥95% 的健康检查通过率并将结果写入审计日志。
3. **Given** 需要导入第三方源码包，**When** 调用 `px-plugin import --source tar.gz --policy external`, **Then** 许可证/漏洞扫描在 15 分钟内完成，若发现高危依赖立即阻断并要求审批，审批结果同步到 CLI 重试。

### User Story 1 - Developer submits publish candidate (Priority: P1)

Plugin developers需要从本地热加载产物生成合规 artefact，并将版本与元数据分别推送至在线/离线渠道，使任意交付路径都能进入审核。**Go CLI 版实现使用 `tools/cli` 提供完整的 dev/publish/dist 命令链，与 TypeScript 版功能对齐。**

**Why this priority**: 没有合规 artefact，审核与安装链路无法启动；它直接驱动 Marketplace 审核与租户更新。

**Independent Test**: 通过执行 `px-plugin dev`→`px-plugin publish`/`dist`，验证 CLI 输出 manifest、签名、回执，且 Marketplace 或运维能收到对应版本记录。

**Acceptance Scenarios**:

1. **Given** 开发者已通过 lint/test 且具备 publish 权限，**When** 运行 `px-plugin publish --channel stable`，**Then** CLI 完成预检、签名、上传并返回 `publishId`、审核链接与 Telemetry 事件。
2. **Given** 客户环境无法访问公网，**When** 开发者执行 `px-plugin dist --target offline --sign ./cert.pem`，**Then** CLI 产出 `.pxp`、`integrity.txt`、`manifest.signature`、`report.json` 并写入 `dist/audit.log`。
3. **Given** 开发者在热加载模式下调试（Go CLI），**When** `px-plugin dev --watch` 触发 reload，**Then** CLI 的 FileWatcher 在 250ms 去抖后调用 Dev API reload，Admin 立即获取最新插件入口并保留 session 日志 7 天，reload 耗时 ≤2s。
4. **Given** 开发者使用 Go CLI 进行热加载调试，**When** 运行 `px-plugin dev --watch --tenant <id> --entry <path>`，**Then** CLI 执行以下流程：
   - 解析 `--entry` 参数并加载 `plugin.yaml` manifest
   - 向 Dev API 发送 `POST /internal/dev/plugins/register` 建立会话，返回 `sessionId` + `reloadToken`
   - 启动 `fsnotify` 文件监听器，递归监听 `<path>` 目录（忽略 `.git`, `node_modules`, `dist/**`）
   - 文件变更时生成 SHA256 哈希，聚合到 250ms 去抖窗口
   - 调用 `POST /internal/dev/plugins/reload`（带幂等 `x-reload-id`），成功后在 stdout 输出日志
   - 结束调试时调用 `DELETE /internal/dev/plugins/register/{sessionId}` 清理会话
   - 会话数据持久化到 `~/.px-plugin/sessions/{sessionId}.json`，支持 `px-plugin dev resume <id>` 恢复
   - 所有操作记录到 `~/.px-plugin/logs/audit.log`，格式与 TypeScript 版完全对齐（`dev.hotload.*` metrics）

---

### User Story 2 - Marketplace reviewer processes submissions (Priority: P2)

Marketplace 审核员需要在 Publish Hub 中接收在线发布和离线上传，完成签名/哈希校验、自动化测试、人工审批，并向租户广播版本可用性。

**Why this priority**: Publish Hub 的可信链依赖审核节点做风控决策；若缺失会阻断上线或放出风险版本。

**Independent Test**: 模拟一个版本进入审核队列，验证审核员可以查看 artefact、运行 checklist、批准或拒绝，并触发通知。

**Acceptance Scenarios**:

1. **Given** CLI 上传的在线版本在审核队列，**When** 审核员完成安全扫描并批准，**Then** 系统在 4 小时内将状态切换为 “Approved” 并广播 `plugin.publish.approved` 事件。
2. **Given** 运维通过离线上传交付 `.pxp` 包，**When** 审核员校验签名失败，**Then** 版本被拒绝且告知具体 hash/证书问题。
3. **Given** Marketplace 配置自动升级，**When** 新版本批准，**Then** 所有订阅租户在 30 分钟内收到通知，可选择立即升级或加入灰度批次。

---

### User Story 3 - Tenant admin installs and manages versions (Priority: P3)

租户管理员在 PowerX Web Admin 需要查看可用插件版本、执行在线拉取或离线上传、监控安装日志，并在失败时回滚。

**Why this priority**: 没有稳定的安装/回滚体验，审核通过的版本无法落地，影响业务连续性。

**Independent Test**: 通过 Admin/API 触发安装、验证日志与 Telemetry、模拟失败后执行回滚并确认上一版本恢复。

**Acceptance Scenarios**:

1. **Given** 新版本通过在线审核，**When** 管理员在 Admin 选择 “从远程安装”，**Then** 后端调用 `install/url` 完成部署、记录事件，并在 5 分钟内支持回滚。
2. **Given** 客户处于隔离网络，**When** 管理员上传 `.pxp` 至 `install/local`，**Then** 系统校验签名并在成功后同步状态到 Publish Hub。
3. **Given** 安装失败或触发回滚策略，**When** 管理员点击 “一键回退”，**Then** 系统在 5 分钟内恢复至前一稳定版本并发出告警。

---

### User Story 4 - Offline distribution steward handles `.pxp` packages (Priority: P2)

运维或 Marketplace 离线运营团队需要上传 `.pxp` 包、验证签名链、登记版本，并在 SLA 内完成人工审核。

**Why this priority**: 离线客户贡献了大量收入，若流程缺失将影响交付并违反合规。

**Independent Test**: 模拟 `.pxp` 包上传到离线审核队列、验证签名、审批、并生成可追溯日志。

**Acceptance Scenarios**:

1. **Given** `.pxp` 与 `integrity.txt` 被上传到 Marketplace 离线入口，**When** 系统校验签名成功，**Then** 在 1 个工作日内给出审核结论并生成内部版本号。
2. **Given** 审核发现包体 hash 不一致，**When** 运营拒绝版本，**Then** CLI 发布者收到失败回执与整改建议。
3. **Given** 审核通过并配置租户白名单，**When** Publish Hub 推送通知，**Then** 指定租户在 Admin 中只能看到授权版本。

---

### User Story 5 - Dev hotload workflow maintains rapid feedback (Priority: P1)

CLI 与 Dev API 必须支撑 `px-plugin dev --watch`、register/reload/delete、BundleBuilder、SessionClient 以及 Admin SSE 日志，以确保本地变更秒级可见。

**Why this priority**: 热加载输出的 manifest 和 artefact 是 publish 的起点；若体验不稳定，后续链路无法保证一致性。

**Independent Test**: 在 sandbox 中运行 `px-plugin dev --watch`，观察文件变动触发 incremental build、Dev API reload、Admin SSE 日志与 CLI Telemetry。

**Acceptance Scenarios**:

1. **Given** 开发者执行 `px-plugin dev --watch --tenant t1`，**When** 新增代码文件，**Then** BundleBuilder 生成差异包并通过 SessionClient 调用 Dev API `reload`，整体耗时 ≤2s。
2. **Given** Dev API 返回错误或证书即将过期，**When** CLI 捕获异常，**Then** 向开发者展示 remediation 并在 Telemetry 中记录 `dev.hotload.cli_error`。
3. **Given** 调试完成，**When** CLI 调用 `px-plugin dev --stop`，**Then** Dev API `DELETE /register` 被调用，Admin SSE 推送 session 结束事件并归档日志 7 天。

---

### User Story 6 - Host simulator & sandbox validation close the debug loop (Priority: P1)

> 来源：docs/use_cases/_from_hub/SCN-DEV-PLUGIN-DEBUG-001（宿主模拟、沙箱验证、错误诊断）

调试阶段需要 `px-plugin host start --mock`、`px-plugin debug attach`、沙箱租户与脱敏数据，加上错误诊断报告和工单闭环，确保 1 分钟复现场景、10 分钟完成沙箱验证并输出可追踪报告。

**Why this priority**: 调试体验是 Publish Hub artefact 质量的起点，缺少受控宿主与沙箱验证将导致上线合规风险。

**Independent Test**: 启动宿主模拟器→执行 `px-plugin dev --watch`→触发 `POST /internal/sandbox/deploy`→收集 `POST /internal/debug/report` 结果并在 Admin/工单中查看诊断。

**Acceptance Scenarios**:

1. **Given** 开发者执行 `px-plugin host start --mock --plugin order-sync`, **When** CLI 校验宿主镜像与 manifest 匹配, **Then** 宿主在 30 秒内启动，自动阻断访问生产资源并将 sessionId 写入 Telemetry (`debug.host.version_mismatch_total`=0)。
2. **Given** 需要挂载断点与日志，**When** 运行 `px-plugin debug attach --session <id>` 并使用 `px-plugin dev --watch`, **Then** SessionClient 将断点/变量同步到宿主，日志脱敏并通过 `debug-observability-v2` 上报。
3. **Given** 沙箱租户执行自动化验证，**When** 触发 `POST /internal/sandbox/deploy` 并完成 `plugin-sandbox-suite`, **Then** `POST /internal/debug/report` 生成错误诊断，自动推送至工单系统并保留 ≥180 天审计。

---

### User Story 7 - Release pipeline & multi-channel Marketplace orchestration (Priority: P1)

> 来源：docs/use_cases/_from_hub/SCN-DEV-PLUGIN-PUBLISH-001（发布审批、灰度、在线/离线上架）

发布经理需要通过 `px-plugin publish create/deploy`, `px-plugin pack`, `px-plugin import --offline` 与 Marketplace 审核入口，贯穿测试租户门禁、灰度策略、自动回滚与上架同步。

**Why this priority**: 发布与 Marketplace 上架是业务交付的最终环节，需统一 artefact、审批、灰度和离线送审，确保 SLA（在线 4h / 离线 1d / 审核 3d）。

**Independent Test**: 使用 `px-plugin publish create` → `px-plugin publish deploy --strategy canary` → `px-plugin pack` → `px-plugin import --offline` → `POST /marketplace/listing/apply`，验证版本可追踪、灰度/回滚策略触发、线上/离线上架同步。

**Acceptance Scenarios**:

1. **Given** 新版本在测试租户通过质量门禁，**When** 运维执行 `px-plugin publish create --channel stable` 并 `px-plugin publish deploy --strategy canary`, **Then** 流水线自动生成发布计划、5 分钟内可触发回滚、指标写入 `publish.gray.error_rate`。
2. **Given** 需要向隔离环境交付，**When** 运行 `px-plugin pack --mode release` 并在运维端调用 `px-plugin import --offline --pkg dist/app.pxp`, **Then** 离线校验与签名报告和在线 artefact 一致，导入成功率 ≥98%。
3. **Given** 发布通过审批，**When** 调用 `POST /marketplace/listing/apply` 同步元数据, **Then** `marketplace.listing.sla_hours` 跟踪 3 个工作日 SLA，`plugin.publish.approved` 与 `marketplace.listing.status` 事件双向关联。

---

### Edge Cases

- 模板索引缺少依赖或扫描失败时，`px-plugin init` 必须回滚输出并提示补齐镜像/豁免。
- 宿主模拟器版本落后或访问生产资源时必须阻断 `px-plugin host start` 并上报 `debug.host.version_mismatch_total` 告警。
- 离线包体积超过阈值或缺少签名时必须阻断导入并给出可操作提示。
- 发布版本号回退或重复提交需拒绝并提示开发者校正 manifest。
- 审核 SLA 超时（>4h 在线 / >1 工作日离线）时，系统需通知发布者与运营，避免版本滞留。
- 多租户并发安装同一版本时若部分失败，需支持分批回滚并保持审计日志一致。
- Dev API 会话断联或 mTLS 证书失效时，`px-plugin dev --watch` 需自动重试并在 3 次失败后降级为手动 reload。
- `.pxp` 包含的敏感配置文件需要脱敏，若校验发现未脱敏则拒绝并附带字段列表。
- 离线 `.pxp` 解密失败或密钥封装不匹配时必须阻断安装，提示重新生成包体或密钥；连续两次失败触发安全告警。
- 非授权角色（如普通租户）尝试执行 publish/审核/安装操作时必须立即阻断、写入审计日志并触发告警。

## High-Level Architecture & CLI Contracts

### 模块分解

| 模块 | 负责人 | 主要职责 |
|------|--------|----------|
| TemplateRegistry & ScaffoldExecutor | PowerX Plugin CLI | 管理模板索引、脚手架渲染、post-hook 与 Git 初始化 |
| BundleBuilder | PowerX Plugin CLI | 收集 artefact、计算 diff、控制包体大小（默认 <300MB）、生成 `.pxp` |
| SessionClient | PowerX Plugin CLI | 管理 Dev API register/reload/delete、mTLS、重试、Backoff |
| PublishPipeline | PowerX Plugin CLI | 预检、签名、上传、生成 `publish-receipt.json` |
| OfflinePackager / `px-plugin pack` | PowerX Plugin CLI | 生成 `.pxp`、`integrity.txt`、`manifest.signature`、`dist/audit.log`、密钥封装 |
| HostSimulator Controller | PowerX Plugin CLI + Core | `px-plugin host start --mock`、断点/日志挂载、版本兼容守护 |
| SandboxValidation Service | PowerX Core | `POST /internal/sandbox/deploy`、脱敏数据、测试结果与 `POST /internal/debug/report` |
| Marketplace Reviewer Console | PowerX Marketplace | 在线/离线审核队列、签名校验、SLA 计时、操作记录 |
| ReleasePipeline Orchestrator | PowerX Core | `px-plugin publish create/deploy`、灰度策略、自动回滚与指标采集 |
| Dev API Gateway | PowerX Core | `POST /internal/dev/plugins/{register|reload}`、SSE 日志、异常回滚 |
| Admin Installer | PowerX Web Admin | `install/url`, `install/local`, 灰度、回滚、日志/告警 |
| RBACGuard Middleware | PowerX Core/Admin | 统一校验开发者/审核员/租户权限，写入审计日志 |
| AuditRetention Worker | PowerX Core | 归档操作日志并确保 ≥180 天保留、定期清理 |
| SLAWatcher | Marketplace Ops | 计算在线/离线审核 & 安装 SLA，触发告警/仪表板 |

### CLI 命令与参数（摘要）

- `px-plugin dev --watch [--tenant <id>] [--entry <dir>] [--max-bundle-size <MB>] [--no-telemetry]`
  - 输出 `sessionId`, `reloadToken`, Admin 调试链接，支持 `--resume` 和 `--stop`.
- `px-plugin host start --mock [--plugin <id>] [--runtime <version>]`
  - 启动宿主模拟器、校验 manifest/运行时、隔离资源并生成 sessionId。
- `px-plugin debug attach --session <id> [--breakpoints ./launch.json]`
  - 将断点/变量快照同步给宿主模拟器，聚合日志并可触发 `POST /internal/debug/report`。
- `px-plugin publish create --channel <stable|beta> [--window <ts>]`
  - 生成发布计划、触发测试租户门禁、记录审批链。
- `px-plugin publish deploy --strategy canary [--batch 20%]`
  - 编排灰度/扩容/回滚，并推送 `publish.gray.*` 指标。
- `px-plugin dist --target offline [--sign <pem>] [--kms-key-id <id>] [--artifact-dir dist]`
  - 生成 `.pxp`、`integrity.txt`、`manifest.signature`、`report.json`、`dist/audit.log`，并可输出上传指令。
- `px-plugin pack --mode release [--channel offline]`
  - 在 `dist/` 输出 release artefact、签名材料与 `px-plugin import` 所需 metadata。
- `px-plugin import --offline --pkg dist/app.pxp [--tenant <id>]`
  - 调用离线导入 API，执行签名/hash 验证、白名单校验与审计记录。
- `px-plugin init <name> --template <id> [--org <org>] [--lang <lang>]`
  - 渲染模板、安装依赖、生成 manifest/publish.yml、触发许可证扫描与 Git 注册。
- `px-plugin doctor [--fix]`
  - 校验多语言运行时/依赖/Flag、填充 `.doctor/report.json` 并输出 remediation。

### 配置文件与 `.pxp` 结构

- `publish.yml`: `channels[]`, `rollout.batches[]`, `tenantFilters`, `autoUpgrade`, `rollbackPlan`.
- `dist.config.(yml|ts)`: `artifacts`, `signing`, `compression`, `integrity.ignore`.
- `.pxp` 结构：`/manifest.json`, `/backend/**`, `/web-admin/**`, `/migrations/**`, `/assets/**`, `/integrity.txt`, `/manifest.signature`, `/docs/release-notes.md`.

### Dev API 合同（摘要）

- `POST /internal/dev/plugins/register`
  - Body: `pluginId`, `version`, `manifest`, `bundleMeta`, `cliVersion`, `tenantId`.
  - Response: `sessionId`, `reloadToken`, `adminPreviewUrl`.
- `POST /internal/dev/plugins/reload`
  - Body: `sessionId`, `reloadToken`, `changedFiles[]`, `diagnostics`, `bundleUrl`.
  - Headers: `x-reload-id`（幂等）。
- `DELETE /internal/dev/plugins/register/{sessionId}`
  - 清理沙盒、释放资源、记录审计日志。
- `POST /internal/sandbox/deploy`
  - Body: `sessionId`, `datasetId`, `testPlanId`, `flags[]`; 负责加载脱敏数据、执行 `plugin-sandbox-suite` 并返回进度。
- `POST /internal/debug/report`
  - Body: `sessionId`, `logs`, `metrics`, `ticketRef`; 生成可脱敏的错误诊断报告并同步工单系统。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Publish Hub MUST allow developers to produce single-source manifests与 artefact（热加载、在线、离线）并保持版本号/依赖一致。
- **FR-002**: The system MUST enforce pre-flight validation（版本递增、依赖兼容、权限声明、测试/覆盖率、签名材料）before any在线 publish 进入审核队列。
- **FR-003**: Offline packaging MUST emit `.pxp`、完整性列表、签名/证书链以及审计报告，并在导入前验证 hash 与签名。
- **FR-004**: Marketplace workflow MUST accept在线发布与离线上传请求，支持审批、拒绝、补充资料，并在每次动作中写入可追溯审计日志。
- **FR-005**: PowerX Admin/API MUST expose `install/url` 与 `install/local`，返回安装日志、Telemetry，以及 ≤5 分钟的自动回滚能力。
- **FR-006**: Publish Hub MUST broadcast版本可用事件并在 30 分钟内向所有订阅租户展示升级入口或灰度配置。
- **FR-007**: Telemetry & Ops MUST记录 `publish.*`, `install.*`, `rollback.*` 指标，支持审批超时、安装失败率、连续回滚等阈值告警。
- **FR-008**: Access control MUST enforce开发者、审核员、租户管理员的权限边界，并保留至少 180 天的操作日志。
- **FR-009**: CLI MUST提供 `px-plugin dist --target offline`、`.pxp` 包结构、签名配置校验、上传指引，并拒绝缺失签名的产物。
- **FR-010**: CLI MUST提供 `px-plugin dev --watch/--stop/--resume`，并通过 BundleBuilder + SessionClient 实现 register/reload、差异打包、失败回退。
- **FR-011**: Dev API MUST对 register/reload/delete 请求执行权限校验、mTLS、幂等控制，失败时返回可操作错误并触发 Admin SSE。
- **FR-012**: Marketplace MUST区分在线/离线审核 SLA（≤4h / ≤1 工作日），并在超时后自动通知发布者与运营。
- **FR-013**: Offline packaging MUST encrypt `.pxp` artefacts with short-lived symmetric keys，并用 Marketplace 公钥封装该密钥（或其引用），以便审核节点解密后再执行签名/完整性校验。
- **FR-014**: CLI MUST提供 `px-plugin init`、模板版本治理与 Git 注册 API，对应 SBOM/许可证扫描与审计留痕。
- **FR-015**: CLI MUST提供 `px-plugin doctor` 用于团队克隆/第三方导入场景，输出环境/依赖/Flag 诊断与自动修复建议。
- **FR-016**: Host simulator & sandbox validation MUST支持 `px-plugin host start --mock`、`px-plugin debug attach`、`POST /internal/sandbox/deploy`、`POST /internal/debug/report`，在 10 分钟内完成验证并沉淀报告。
- **FR-017**: Release pipeline MUST公开 `px-plugin publish create/deploy --strategy canary`、`px-plugin pack`、`px-plugin import --offline`，实现测试租户门禁、灰度扩容、自动回滚与 Marketplace 上架同步。

### Key Entities *(include if feature involves data)*

- **Plugin Artefact (.pxp)**: 打包后的插件内容与元数据，包含 manifest、版本、渠道、依赖、完整性列表、签名信息。
- **Publish Request**: 记录 `pluginId`, `version`, `channel`, `submitter`, `approvalStatus`, `reviewQueueId`, `telemetryRefs`，驱动在线审核或离线上传流程。
- **Tenant Deployment Record**: 追踪租户侧的安装/升级/回滚，字段含 `tenantId`, `pluginId`, `version`, `installMethod`, `status`, `rollbackLink`, `timestamp`。
- **Offline Integrity Report**: `integrity.txt`, `manifest.signature`, `report.json`, `audit.log`，用于证明 `.pxp` 完整性与审批链路。
- **Dev Hotload Session**: `sessionId`, `tenantId`, `bundleHash`, `reloadSeq`, `logs`, `status`, `expiresAt`，用于追踪 watch 流程和回滚。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: ≥95% 的 `px-plugin publish` 在 10 分钟内完成预检并入队审核，失败需输出可操作的 remediation。
- **SC-002**: 在线审核平均时长 ≤ 4 小时、离线上传审核 ≤ 1 个工作日，超时案例触发运营告警。
- **SC-003**: 99% 的订阅租户在 30 分钟内收到新版本通知；≥80% 的租户能在 2 小时内完成升级或明确拒绝。
- **SC-004**: 插件安装成功率 ≥ 98%，若失败可在 5 分钟内自动回滚且完整日志同步至 Publish Hub；连续两次失败即时告警。
- **SC-005**: 离线 `.pxp` 包完整性/签名验证通过率 100%，离线上传到审核完成平均时长 ≤ 6 小时（工作时间衡量）。
- **SC-006**: `px-plugin dev --watch` reload P95 ≤ 2s，`dev.hotload.cli_error` 单日率 < 2%；日志与 Telemetry 延迟 ≤ 1s。

## Clarifications

### Session 2025-11-07

- Q: 离线 `.pxp` 在传输/存储时是否必须加密？ → A: 必须使用临时对称密钥对 `.pxp` 加密，并由 Marketplace 解密后再校验签名。
- Q: 临时密钥如何交付给 Marketplace？ → A: CLI 生成本地对称密钥并用 Marketplace 公钥加密后随请求上传，由 Marketplace 使用私钥解封。
