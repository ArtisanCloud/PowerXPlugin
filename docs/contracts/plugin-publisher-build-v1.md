# 开发者发布工具隔离构建契约 v1（F03）

日期：2026-11-09。状态：源码及受控构建测试已交付；当前 macOS 执行器拒绝 Linux 构建，实际 Linux/Nuxt 运行验收尚未完成。协议沿用 `powerx.plugin-publisher-local/v1`，数据模型 schema 为 `powerx.plugin-publisher-build/v1`。见[本地 OpenAPI](plugin-publisher-local-v1.openapi.yaml)、[F02 会话契约](plugin-publisher-local-v1.md)和[统一方案](../marketplace/plugin-developer-publishing-center-plan.md)。

## 1 首个受支持配置

配置标识为 `make-dist-pure-go-js/v1`：Linux/amd64 本机执行器、Go 1.24+、Node 20+、CGO_ENABLED=0 的 Go 后端、纯 JavaScript 的 Node/Nitro 运行产物。市场仍未接入；构建不意味着市场登记、审核、签名或宿主启用。

Go 进程与 Node 工具均须匹配 Linux/amd64。macOS 上成功交叉编译 Go 不算完整 Linux 构建；工具不会自动启动 Docker、切换到远程机器或绕过准入。信息接口返回 release-plans/local-builds；只有本机平台和工具检查通过时，build_targets 才包含 linux/amd64。

当前不提供 CGO、Node .node 或其他原生前端运行文件的 ABI 验证，遇到这些文件明确拒绝，不将它们视为已验证运行时。Go ELF 必须为未剥离符号的 ELF64/x86-64 可执行文件，具有正确 Go 构建目标和 trimpath 标记；不是所有 Linux 运行时组合都已开放。

## 2 CLI 与 HTTP

CLI 与 HTTP 复用同一引擎、任务记录和冻结文件。CLI 构建后使用：

```bash
./bin/px-plugin publisher build run --entry <plugin_workspace> --plan-id <release_plan_uuid>
./bin/px-plugin publisher build list --entry <plugin_workspace>
./bin/px-plugin publisher build get --entry <plugin_workspace> --id <build_uuid>
./bin/px-plugin publisher build verify --entry <plugin_workspace> --id <build_uuid>
./bin/px-plugin publisher build logs --entry <plugin_workspace> --id <build_uuid>
```

run 在所属进程内等待终态，输出任务 JSON；失败或取消同时返回非零退出码。Ctrl-C/SIGTERM 取消所属构建。独立 CLI 不凭历史 PID 取消另一进程的任务；服务发起的任务通过同一服务的 cancel 接口取消。

下列路径均沿用 F02 的精确 Host、Origin 和工具会话校验：

| 方法与路径 | 语义 |
| --- | --- |
| POST /v1/builds | 仅接收 release_plan_id UUID；202 表示已排入所属引擎，不表示已构建成功 |
| GET /v1/builds | page/size 分页，默认 20，最大 100 |
| GET /v1/builds/{build_id} | 阶段、来源、工具、诊断进程和终态结果 |
| POST /v1/builds/{build_id}/cancel | 无请求体；取消该执行器拥有的任务，不按历史 PID 猜测所有权 |
| POST /v1/builds/{build_id}/verify | 无请求体；重新读取同一 ZIP，核对实际大小和 SHA256，不重新构建或压缩 |
| GET /v1/builds/{build_id}/logs | 最多 4 MiB 的脱敏诊断文本 |

构建 POST 拒绝 command、cwd、可执行文件、环境变量、未知字段和重复键。HTTP 请求结束或浏览器断开不取消已接受任务；停止执行器会取消所属任务。构建、日志及产物引用使用 build_id，不使用 numeric id 或摘要作为业务对象身份。

## 3 源码副本与隔离

任务记录位于 `.px-plugin/publisher/builds/<UUID>.json`；每个任务有独立的 `work/<UUID>/source`、staging、build.log 和 artifact.zip。同工作区通过独立 build.lock 串行化；服务会话的 executor.lock 与构建锁分开，CLI 与 HTTP 之间也不能并发构建。

构建输入固定为 Makefile、make-files、scripts、plugin.yaml、backend、web-admin、plugin.d、contracts、skills、config/event_fabric.yaml 及单独 README/LICENSE。复制已安装的依赖，不自动安装。排除源码仓库、本地工具状态、既有 bin/dist/.output/.nuxt、缓存、上传、日志、数据库、私钥、真实 config.yaml 和 .env；node_modules 内实际依赖的 bin/dist 目录保留。不能复制整个任意工作区。

副本以实际文件内容、相对路径及可执行位计算 SHA256，覆盖未提交源码和已安装依赖。复制后、构建后分别核对原输入与副本；改变则阻断产物。Git 仓库记录完整 commit 和 dirty，未建 Git 的目录明确记录 filesystem 来源及 dirty=null，不伪造提交记录。

仅接受受控根内的文件链接，复制时解引用；外部链接、循环链接、特殊文件、反斜杠路径和超出预算的输入明确失败。根路径须满足 Makefile 安全参数约束；不把含 shell/Make 元字符的路径拼入变量。目录身份发生替换时拒绝开始构建。

Go 依赖通过 `go mod edit -json` 只读核对，拒绝本地 replace；前端拒绝 file:/link:/workspace: 依赖和逃出根目录的依赖链接。需要预先安装确定版本或可追踪包，不自动沿用外部本地链接。GOWORK/GOENV 关闭，GOPROXY/GOSUMDB 关闭网络获取，GOTOOLCHAIN=local；缺少预备依赖或工具链明确失败。

子进程只继承工具路径、现有 HOME、临时目录、语言及 Go 缓存相关的允许项，不继承业务令牌或部署配置；禁用构建遥测与 npm 通知。源码是用户明确启动的受信构建输入，此工具不提供任意 shell 接口，也不把执行受信 Makefile 声称为操作系统级沙箱。

## 4 固定构建、版本与冻结

引擎用参数数组执行固定 make dist，传入计划版本、linux/amd64 及副本内独立 BUILD_DIR、DIST_ROOT、DIST_DIR。前端 .output 写在副本内，源目录 manifest、后端 bin 和前端输出不改写；不创建分支或 git worktree。

Skeleton、scaffold 和 CLI 内嵌模板把 app.PluginVersion 定义为变量，并在 Go plugin/migrate 构建中通过 -X 注入 VERSION，开启 trimpath。源 manifest 默认版本不自动改写。旧插件需显式迁移此构建契约；不自动改写用户源码，不对常量、strip 符号或旧布局增加兼容回退。

校验直接读取 plugin ELF 中 Go 字符串变量 PluginVersion，核对实际链接值与计划版本；不以链接参数文本或声明元数据代替，也不执行插件二进制。迁移程序没有独立业务版本接口，固定按同任务包版本归属，检查 Go/ELF 目标和 trimpath，不宣称读取到独立迁移业务版本。

冻结 staging 只接收安装布局的 plugin.yaml、后端 bin、前端 .output，以及受控 plugin.d、contracts、skills、事件声明、语言/公开资源、供应商目录和单独说明文件。核对唯一正式 manifest、版本、target、后端入口、前端 Node 入口、迁移入口、目录声明和契约引用；敏感配置值、危险路径及未验证原生文件阻断。前端写入 powerx-publisher.json，记录本次版本、目标、工具版本和配置身份；它不是签名或任意业务运行测试。

ZIP 使用单一版本根、规范路径、固定条目时间和可执行权限。压缩同一 staging 一次，最终文件记录真实字节大小与 SHA256，最大 2 GiB，设为只读后才保存 ready。verify 只读原文件，缺失、链接替换或字节变化返回明确错误；不会重新压缩来伪装同一次重试。

## 5 状态、进程和预算

状态为 queued → prechecking → building → packing → verifying → ready，异常为 failed/cancelled/interrupted。没有真实完成的阶段不标成功；只有 ready 附带可校验的冻结产物。

记录当前 PID，以及所属父 PID、进程组、源副本相对目录和开始/退出时间。make 与正常子进程进入独立进程组；取消先发送 TERM，最多两秒等待后终止所属组的剩余进程。执行器退出也等待所属构建收敛，不按端口或通用进程名清理。

重启仅在能取得构建锁时把遗留活动记录改为 interrupted，清除当前 PID，不根据过期 PID 杀进程、不自动重放、不把遗留 ZIP 认作成功。新构建使用新 UUID；失败任务和脱敏诊断保留供核对。

源码及单次受控树复制上限 8 GiB/200000 文件，YAML 单文件 2 MiB，构建时限 30 分钟。诊断日志最大 4 MiB，单行过长及总预算耗尽均写机器标记；过滤令牌、私钥、认证、Cookie、URL 和本机路径。它不是逐字完整的原始工具输出。上述是复制、任务和产物预算，不声称任意受信 Makefile 已受磁盘配额或网络沙箱隔离。

## 6 验证边界

受控测试运行真实 make/Go/Node 工具，生成 Linux ELF 和静态 JS 文件，验证快照隔离、源变化、版本值、ZIP、引用资源、原生文件拒绝、篡改、会话 HTTP、取消进程组及中断恢复。测试内部注入平台探针只用于这个 fixture，生产无开关可绕过 macOS 准入；这个 fixture 不是 Nuxt 真实 Linux 运行验收。

当前机器为 macOS，Docker 守护进程不可用，未自行启动。因此实际 Linux 执行器、完整 Nuxt 产物、市场上传、宿主安装和启用仍分别待验收；不能把本次受控构建标为 M1 或 A03 的真实运行验收通过。

受控测试、进程清理、模板消费和 CLI 准入检查见[本次验证记录](evidence/plugin-publisher-build-validation-20261109.json)。源码及临时编译产物未正式发布，用户全局 CLI 未更新。
