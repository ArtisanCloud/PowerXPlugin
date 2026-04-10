---
name: ci-manifest-align
description: 自动检查 contracts/capabilities 与 plugin.yaml/catalog（plugin.d）的同步对齐，适用于功能迭代后快速发现路由、RBAC、清单漂移问题。
---

# CI Manifest Align（清单对齐守卫）

## 目标

- 将 `contracts/capabilities/*.yaml` 作为单一事实源。
- 自动校验 capability -> `plugin.d/{capabilities,exposure,rbac}.yaml` 的映射一致性。
- 在提交前阻止“代码改了但清单没同步”的漂移。

## 一键命令

- 自动同步并校验（推荐，本地开发）：
  - `node .codex/skills/ci/manifest-align/scripts/manifest-align-check.mjs --fix`
- 严格检查（推荐，CI gate）：
  - `node .codex/skills/ci/manifest-align/scripts/manifest-align-check.mjs`
- 自动同步并暂存产物（可选）：
  - `node .codex/skills/ci/manifest-align/scripts/manifest-align-check.mjs --fix --stage`

## 检查内容

1) 脚本内部自动执行基础清单校验
- 内部会调用 `plugin-yaml-check`（包含 `manifestcheck --sync-catalogs`），重建 `plugin.d` 产物。

2) 检查清单漂移（支持 auto-fix）
- 对以下文件做 `git diff` 检查：
  - `plugin.d/capabilities.yaml`
  - `plugin.d/exposure.yaml`
  - `plugin.d/rbac.yaml`
- 若有变更：
  - 默认模式：报错并阻止继续（适合 CI）。
  - `--fix`：自动接受同步结果并继续做映射校验。
  - `--fix --stage`：在 `--fix` 基础上自动 `git add` 变更文件。

3) 检查 capability -> exposure 的 RBAC 映射
- 读取每个 capability 的：
  - `id`
  - `rbac.resource`
  - `rbac.actions`
- 断言 `plugin.d/exposure.yaml` 中存在对应 `capability` 且 `rbac=resource:firstAction`。

4) 检查 capability -> rbac catalog 的资源动作覆盖
- 聚合 capability 声明的 `resource/actions`。
- 断言 `plugin.d/rbac.yaml` 的 `rbac.resources` 覆盖全部资源与动作。

## 建议接入

- 本地开发：每次改 `contracts/capabilities/*.yaml` 后执行 `--fix`。
- CI：使用严格模式（不带 `--fix`），作为 merge gate。
