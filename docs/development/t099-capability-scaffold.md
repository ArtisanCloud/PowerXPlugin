# T099 – Capability Scaffold Updates

## 修改内容

1. `scaffold/templates/plugin.yaml.tmpl` 与 `tools/cli/internal/templates/data/plugin.yaml.tmpl` 现已默认包含：
   - `capabilities.provides` 的 `schemas.input/output` 字段；
   - `agent_tools` 列表及 handler 注释；
   - `exposure.channels`（REST + Agent Tool）。
2. `packages/template-registry/index.yaml` 为两个模板声明 `features: [capabilities.v1]`，便于 CLI 判定能力脚手架可用性。
3. 新增 handler stub：`backend/internal/handlers/capabilities/template/create_handler.go.tmpl`。

## 验证步骤

```bash
npm run sync:templates

# （可选）在本地构建 px-plugin CLI 后执行
# px-plugin init demo.capability --template fullstack-go-nuxt
# tree demo.capability/contracts
# grep -n \"capabilities\" demo.capability/plugin.yaml
```

> 注：仓库中未内置 `px-plugin` 可执行文件，需在开发环境编译后运行上述命令。
