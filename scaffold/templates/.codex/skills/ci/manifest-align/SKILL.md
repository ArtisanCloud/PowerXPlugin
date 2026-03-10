# CI Manifest Align

用于初始化插件仓库后的清单对齐检查，避免 `plugin.yaml`、`plugin.d`、`contracts/capabilities` 漂移。

## 路径规范（项目根目录）

- `plugin.yaml`
- `plugin.d/capabilities.yaml`
- `plugin.d/exposure.yaml`
- `plugin.d/rbac.yaml`
- `contracts/capabilities/*.yaml`

禁止使用历史模板目录路径作为运行命令输入。

## 统一命令

- 本地修复并对齐：`make manifest-align-fix`
- CI 校验一致性：`make manifest-align-check`

## 期望结果

- `manifest-align-fix` 会基于 `contracts/capabilities` 重建 `plugin.d` 产物并执行一轮校验。
- `manifest-align-check` 仅检查，不改文件；适合 CI 阶段做门禁。
