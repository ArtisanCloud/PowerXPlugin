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
- `px-plugin package`：experimental，当前独立生成 TAR payload，与 Makefile dist/ZIP 尚未统一
- `px-plugin dist`：experimental 提示，当前不执行打包
- `px-plugin publish`：旧内部 JSON 提交，包通过本地 URI 表达，不等于市场文件上传
- `px-plugin publisher plan create/list/get`：创建、分页和读取本地版本计划
- `px-plugin publisher serve`：启动受保护的回环工具服务
- `px-plugin publisher build run/get/list/verify/logs`：隔离构建、任务查询、冻结文件复核及脱敏日志
- `px-plugin doctor`：环境诊断

## 开发者发布中心与命令迁移

统一基线为 [开发者发布中心对接方案](../../docs/marketplace/plugin-developer-publishing-center-plan.md)及[实现设计](../../docs/marketplace/plugin-publisher-implementation.md)，两仓库维护同版文档。本地计划及会话服务基础已实现，见[本地工具契约与使用说明](../../docs/contracts/plugin-publisher-local-v1.md)。F03 首个隔离构建配置已实现，见[构建契约](../../docs/contracts/plugin-publisher-build-v1.md)；仅原生 Linux/amd64、无 CGO Go 后端和纯 JS 前端，真实 Linux/Nuxt 运行未验收，F05 本地页面及开发代理已实现，使用方式见[页面操作指南](../../docs/guides/features/plugin-developer-publisher/guide.md)；市场上传、恢复和送审已接通 rc.2，实际实例仍须开放对应能力及授权。

目标是 Go CLI 与回环执行器复用一套引擎，以 `make dist` 的安装布局构建、冻结并生成 ZIP，使用独立发布令牌向 `/publisher` 上传真实文件。页面仅在开发态挂载；M1 单平台上传，M2 送审，M3 多平台/beta，M4 CI。

TypeScript publish 当前本地生成 ID、地址和回执，旧 Plan/Deploy 及内存记录不作为真实发布证据。新工具不回退 `/internal/*`、旧配置中的 API key 或 Host 凭据；plan/serve 的命令、退出码、会话和请求预算已固定在 F02 本地契约；构建命令已补齐，市场与送审命令已实现。任务状态统一保存于 `.px-plugin/publisher/` 并排除安装包；首期只恢复任务/结果，必要时整包重传，不支持字节断点续传。

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

### Publisher 市场适配（rc.2）

`publisher market add/list/get/bind/verify/disconnect/versions` 管理当前插件市场连接；`bind --token-stdin` 仅从标准输入向系统凭据库绑定令牌。`publisher upload run/list/get/resume/cancel` 使用同一冻结包及持久化 operation UUID；跨进程取消需通过所属执行器，恢复不分配新 operation。

页面与 CLI 共用 Go 引擎；普通 HTTP 30 秒、上传 1860 秒、最多三次完整上传和 7200 秒恢复预算。市场 discovery 未开放能力时拒绝写入。详见[适配契约](../../docs/contracts/plugin-publisher-market-v1.md)和[操作指南](../../docs/guides/features/plugin-developer-publisher/guide.md)。本地快照尚未发布，独立构建需消费对应 Framework SDK 快照；已发布旧 SDK 不代表包含本轮 manifest 接口。

### Publisher 送审（F08，受控实现）

`publisher review prepare/preview/submit/list/get/resume/cancel/history` 消费 rc.2 的 M2 保留契约。prepare 读取完整集合及市场选项，preview 不写市场，submit 显式确认后持久化原 operation；驳回修订另建操作。P07 已交付正式路由及 M2 scopes，默认关闭；实际目标以 discovery 与当前凭据权限为准。详见[送审契约](../../docs/contracts/plugin-publisher-submission-v1.md)。

请求记录：`publisher upload requests --entry . --id <upload_uuid> --page 1 --size 20`；送审使用 `publisher review requests --entry . --id <submission_uuid>`。仅输出规范 UUID、固定请求类别、状态码和时间，不输出令牌或正文。

### 开发页面自动管理（2026-11-16）

日常开发页面自动识别插件并管理同版发布执行器，无需另开终端运行 serve 或填写配对变量。Base 页面支持明确选择的 Linux 容器构建，以及私密绑定发布令牌到宿主系统凭据库。CLI 命令仍供自动化和 CI 使用，不是页面操作的前置步骤。详见页面指南和本地 OpenAPI 1.1.0；市场继续使用 rc.2。
