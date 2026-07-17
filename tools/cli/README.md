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

- `docs/guides/develop/cli-plugin/cli-plugin-tutorial.md`

## 模板/契约改动后同步

```bash
rsync -a scaffold/templates/ tools/cli/internal/templates/data/
rsync -a docs/contracts/{manifest.json,rbac.json,openapi.yaml} tools/cli/internal/contracts/data/
make build-px-plugin
```

## Web Admin 关键环境变量（Nuxt）

`px-plugin init` 生成的 Nuxt 工程默认支持 Standalone 与宿主代理双模式，常用变量如下：

- `POWERX_PROXY`：`0`（Standalone）/`1`（宿主代理）
- `NUXT_PUBLIC_POWERX_PROVIDER_MODE`：可选 `local`/`delegated`，未指定时默认 `local`
- `NUXT_PUBLIC_API_BASE`：API 基址（默认 `http://localhost:<backend-port>`）
- `NUXT_PUBLIC_API_PREFIX`：默认 `/api/v1`
- `NUXT_DEV_API_PROXY`：开发态 HTTP 代理目标
- `NUXT_DEV_WS_PROXY`：开发态 WS 代理目标
- `NUXT_PUBLIC_POWERX_CORE_BASE`：PowerX Core 地址（默认 `http://localhost:8077`）

建议：

1. Standalone 开发：`POWERX_PROVIDER_MODE=local POWERX_PROXY=0`，前端直连本地 backend。
2. 本地联调宿主链路：`POWERX_PROVIDER_MODE=local POWERX_PROXY=1`，并确认插件网关 ApiKey 与 backend 配置一致。
3. 宿主委派：`POWERX_PROVIDER_MODE=delegated POWERX_PROXY=1`，并确认 STS/gRPC 契约变量齐全。
