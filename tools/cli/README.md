# px-plugin CLI

`px-plugin` 提供 PowerX 插件的脚手架能力，并将必要模板/契约通过 `go:embed` 打包到二进制中，以满足离线初始化的要求。

## 构建

```bash
cd tools/cli
go build -o ../../bin/px-plugin ./cmd/px-plugin
```

## 可用命令

- `px-plugin init <plugin-id> [--module=<module>] [--force]`：渲染 `scaffold/templates` 与契约文件，生成可运行骨架并写入 `plugin.yaml`。
- `px-plugin package`：**Experimental**，当前仅打印计划中的 `go build`/`npm run build` 流程。
- `px-plugin dist`：**Experimental**，当前仅输出预期的打包说明。
- `px-plugin publish [--token=<token>]`：**Experimental**，当前仅提示未来的 Marketplace 发布流程。

## 模板与契约同步

CLI 在 `internal/templates/data/` 与 `internal/contracts/data/` 中保存 `go:embed` 所需的拷贝，请在更新仓库根部的 `scaffold/templates/` 或 `docs/contracts/` 后执行：

```bash
rsync -a scaffold/templates/ tools/cli/internal/templates/data/
rsync -a docs/contracts/{manifest.json,rbac.json,openapi.yaml} tools/cli/internal/contracts/data/
```

随后重新运行 `go build` 以确保最新模板被打包到 CLI。
