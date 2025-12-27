# Capabilities Workspace

该目录用于存放插件侧 `capabilities/*.yaml` 能力目录片段。`px-plugin` CLI、`scripts/capabilities` 工具会在此生成或扫描能力声明，以便 Capabilities Manager 导出协议资产并同步至 PowerX。

## Required Capabilities 申领说明

从 2025-12 起，插件在宿主或 Skeleton 环境中调用 PowerX Core 能力时，必须在 `skeleton/plugin.yaml` 中的 `capabilities.required` 显式声明依赖 ID。操作步骤：

1. 在 `skeleton/plugin.yaml` 中添加示例（`capabilities.required` 字段即文档中的 `requiredCapabilities`）：
   ```yaml
   capabilities:
     required:
       - com.corex.media.assets.manage
       - com.corex.eventfabric.publish
   ```
   根据业务需要替换为真实的 Capability Registry ID。
2. 执行下列命令校验并落库，务必使用 `./skeleton/plugin.yaml` 作为 manifest：
   ```bash
   px-plugin capabilities plan --manifest ./skeleton/plugin.yaml
   px-plugin capabilities apply --manifest ./skeleton/plugin.yaml
   ```
   - `plan`：确保填写的 capabilityId/action 已在 Registry 中公布并获得租户授权；
   - `apply`（或 `lint|submit`）：在本地/CI 中阻止遗漏声明，输出需要审批的 diff。
3. 若需实际申请新的能力，请在宿主控制台或 CLI 中提交审批，再次运行 `plan/apply` 直至通过。

> Tips：`scripts/capabilities/run-from-package.mjs --mode host` 会自动读取 `skeleton/plugin.yaml` 并校验 `requiredCapabilities`；若缺失，脚本会直接报错并提醒补充。

## Provides 与 Contracts

- `capabilities/provides` 下的 YAML 片段仍负责任务能力的原子描述；在申领宿主能力时无需修改这些文件。
- 若能力外露路径、Workflow 模板或 Agent 工具有更新，请在 `contracts/exposure/**` 及 `contracts/schema/**` 中同步，随后执行 `npm --prefix scripts/capabilities run export` 生成最新多协议资产。
