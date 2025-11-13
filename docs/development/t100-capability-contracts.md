# T100 – Capability Contracts Validator

## 新增脚本

`scripts/capabilities/validate-capabilities.mjs` 提供下列功能：

- 从 `plugin.yaml` 读取 `capabilities.provides`；
- 自动补齐 `descriptor`、`schemas.input/output` 字段，并在缺失时生成 stub；
- 生成/更新 `contracts/capabilities/*.yaml` 与 `contracts/schema/input|output/*.json`；
- 在校验阶段若发现缺失文件或 ID 不一致会抛出错误。

## 使用方法

```bash
# 验证当前插件（MANIFEST 默认为 plugin.yaml，可通过 --capabilities-dir/--schemas-dir 定制）
node scripts/capabilities/validate-capabilities.mjs --manifest ./plugin.yaml

# 在 CI 中
VALIDATE_MANIFEST=./plugin.yaml make validate

# NPM 流水线（需要显式设置 CAP_MANIFEST）
CAP_MANIFEST=./plugin.yaml npm test
```

脚本只会在指定 manifest 存在时执行；否则会提示缺少文件并退出。
