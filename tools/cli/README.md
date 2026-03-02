# px-plugin CLI

`px-plugin` 提供 PowerX 插件脚手架能力，模板/契约通过 `go:embed` 打包进二进制。

## 构建

在仓库根目录执行：

```bash
make build-px-plugin
./bin/px-plugin --version
```

安装为全局命令：

```bash
make install-px-plugin PX_PLUGIN_CLI_VERSION=v0.0.3.3-alpha
hash -r
px-plugin --version
```

## 常用命令

- `px-plugin init <plugin-id>`：初始化插件项目（支持交互向导）
- `px-plugin package`：experimental
- `px-plugin dist`：experimental
- `px-plugin publish`：experimental
- `px-plugin doctor`：环境诊断

## 主流程文档（推荐）

完整流程（init → DB 配置 → migrate/seed → 启动 → 本地安装到 PowerX）请看：

- `docs/guides/develop/cli-plugin-tutorial.md`

## 模板/契约改动后同步

```bash
rsync -a scaffold/templates/ tools/cli/internal/templates/data/
rsync -a docs/contracts/{manifest.json,rbac.json,openapi.yaml} tools/cli/internal/contracts/data/
make build-px-plugin
```
