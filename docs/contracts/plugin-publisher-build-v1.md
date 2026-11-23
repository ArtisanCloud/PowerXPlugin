# 开发者发布工具隔离构建契约 v1（F03）

日期：2026-11-16。状态：Base 0.1.5 已通过页面构建 Linux/amd64 完整包并在容器启动其中的 Nuxt；Market 已交付 P04 的 schema ID 修复，2026-11-17 同一原包已在 Framework 页面真实上传并取得送审回执；后续人工审核状态待双方接续。具体以 GUI 验证记录为准；本机模式仍执行原生准入。协议沿用 `powerx.plugin-publisher-local/v1`，数据模型 schema 为 `powerx.plugin-publisher-build/v1`。见[本地 OpenAPI](plugin-publisher-local-v1.openapi.yaml)、[F02 会话契约](plugin-publisher-local-v1.md)和[统一方案](../marketplace/plugin-developer-publishing-center-plan.md)。

市场侧准入问题已由 Market 维护者修复并交付，Framework 已接续自己的客户端与页面复验；详见[问题、方案与协作分工](../marketplace/plugin-publisher-market-admission-handoff-20261116.md)。参考补丁只供市场侧 Review，不由 Framework 代改市场功能源码。

## 1 首个受支持配置

配置标识为 `make-dist-pure-go-js/v1`。安装包目标为 Linux/amd64，包含 `CGO_ENABLED=0` 的 Go 后端和纯 JavaScript 的 Node/Nitro 运行产物。构建工具基线为 Go 1.24+、Node 20+，项目锁定的依赖可以要求更高版本。

发布中心默认使用直接构建。Linux/amd64 使用匹配架构的 Node。Apple 芯片 Mac 使用 arm64 Node 构建前端，并把 Go 后端交叉编译为 Linux/amd64。包内记录真实的 Node 构建平台，市场独立校验包。在市场准入完成对齐前，Apple 芯片 Mac 产物的上传仍待联合验收。[Market 准入交接](../marketplace/plugin-publisher-direct-make-dist-handoff-20261119.md)

开发者可在「更多 → 构建设置」手动选择容器。该方式使用 Linux/amd64 工具链。build_environment 区分直接构建和容器的可用状态。

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
| POST /v1/builds | 接收 release_plan_id UUID 和 executor（native 或 container），202 表示已排入所属引擎 |
| GET /v1/builds | page/size 分页，默认 20，最大 100 |
| GET /v1/builds/{build_id} | 阶段、来源、工具、诊断进程和终态结果 |
| POST /v1/builds/{build_id}/cancel | 无请求体；取消该执行器拥有的任务，不按历史 PID 猜测所有权 |
| POST /v1/builds/{build_id}/verify | 无请求体；重新读取同一 ZIP，核对实际大小和 SHA256，不重新构建或压缩 |
| GET /v1/builds/{build_id}/logs | 最多 4 MiB 的脱敏诊断文本 |

构建 POST 拒绝 command、cwd、可执行文件、环境变量、未知字段和重复键。HTTP 请求结束或浏览器断开不取消已接受任务；停止执行器会取消所属任务。构建、日志及产物引用使用 build_id，不使用 numeric id 或摘要作为业务对象身份。

## 3 源码副本与隔离

任务记录位于 `.px-plugin/publisher/builds/<UUID>.json`；每个任务有独立的 `work/<UUID>/source`、staging、build.log 和 artifact.zip。同工作区通过独立 build.lock 串行化；服务会话的 executor.lock 与构建锁分开，CLI 与 HTTP 之间也不能并发构建。

构建输入固定为 Makefile、make-files、scripts、plugin.yaml、backend、web-admin、plugin.d、contracts、skills、受控配置及单独 README/LICENSE。排除源码仓库、本地工具状态、既有 bin/dist/.output/.nuxt、缓存、上传、日志、数据库、私钥、真实 config.yaml 和 .env。

Linux 直接构建复制已安装的依赖，保留 node_modules 内依赖的 bin/dist。Mac 直接构建使用不含 node_modules 的源码副本，在副本按 package-lock.json 执行 `npm ci`，使用独立 npm 配置。Go 从本机缓存及校验和验证的 Go 公共代理解析锁定模块。已有的三类 Framework 本地 SDK 通过受控复制接入，路径仅在构建副本中改写并记录依赖摘要。

副本以实际文件内容、相对路径及可执行位计算 SHA256，覆盖未提交源码和已安装依赖。复制后、构建后分别核对原输入与副本；改变则阻断产物。Git 仓库记录完整 commit 和 dirty，未建 Git 的目录明确记录 filesystem 来源及 dirty=null，不伪造提交记录。

仅接受受控根内的文件链接，复制时解引用；外部链接、循环链接、特殊文件、反斜杠路径和超出预算的输入明确失败。根路径须满足 Makefile 安全参数约束；不把含 shell/Make 元字符的路径拼入变量。目录身份发生替换时拒绝开始构建。

Linux 直接构建通过 `go mod edit -json` 只读核对 Go 依赖，拒绝本地 replace。前端拒绝 file:/link:/workspace: 依赖和逃出根目录的依赖链接。Linux 直接构建关闭 GOPROXY/GOSUMDB 网络获取。直接构建均关闭 GOWORK/GOENV，使用 GOTOOLCHAIN=local。

子进程只继承工具路径、现有 HOME、临时目录、语言及 Go 缓存相关的允许项，不继承业务令牌或部署配置；禁用构建遥测与 npm 通知。源码是用户明确启动的受信构建输入，此工具不提供任意 shell 接口，也不把执行受信 Makefile 声称为操作系统级沙箱。

## 4 固定构建、版本与冻结

引擎用参数数组执行固定 make dist，传入计划版本、linux/amd64 及副本内独立 BUILD_DIR、DIST_ROOT、DIST_DIR。前端 .output 写在副本内，源目录 manifest、后端 bin 和前端输出不改写；不创建分支或 git worktree。

Skeleton、scaffold 和 CLI 内嵌模板把 app.PluginVersion 定义为变量，并在 Go plugin/migrate 构建中通过 -X 注入 VERSION，开启 trimpath。源 manifest 默认版本不自动改写。旧插件需显式迁移此构建契约；不自动改写用户源码，不对常量、strip 符号或旧布局增加兼容回退。

校验直接读取 plugin ELF 中 Go 字符串变量 PluginVersion，核对实际链接值与计划版本；不以链接参数文本或声明元数据代替，也不执行插件二进制。迁移程序没有独立业务版本接口，固定按同任务包版本归属，检查 Go/ELF 目标和 trimpath，不宣称读取到独立迁移业务版本。

冻结 staging 只接收安装布局的 plugin.yaml、后端 bin、前端 .output，以及受控 plugin.d、contracts、skills、事件声明、语言/公开资源、供应商目录和单独说明文件。核对唯一正式 manifest、版本、target、后端入口、前端 Node 入口、迁移入口、目录声明和契约引用；敏感配置值、危险路径及未验证原生文件阻断。前端写入 powerx-publisher.json，记录本次版本、目标、工具版本和配置身份；它不是签名或任意业务运行测试。

ZIP 使用单一版本根、规范路径、固定条目时间和可执行权限。压缩同一 staging 一次，最终文件记录真实字节大小与 SHA256，最大 2 GiB，设为只读后才保存 ready。verify 只读原文件，缺失、链接替换或字节变化返回明确错误；不会重新压缩来伪装同一次重试。

### 4.1 冻结发布包准入

下列限制落实市场 P01 Review 已确认的包规则，与源码复制预算分别执行。市场 P04 必须独立复核；Framework 本地通过不代替市场验收，也不表示已冻结或消费 rc.2 协议包。

| 项目 | 发布包限制 |
| --- | --- |
| ZIP 实际字节 | 最多 2 GiB；写入时即限制，包含 ZIP 目录和元数据 |
| 正式 manifest | 最多 262144 字节，UTF-8、单一 mapping 文档；拒绝重复键、alias 和自定义 tag |
| ZIP 条目 | 最多 50000，实际存在的目录条目也计数 |
| 解压总大小 | 最多 8 GiB；先检查目录声明，再有界读取每项，核对实际尺寸及 CRC |
| ZIP 条目路径 | 最多 1024 UTF-8 字节，包含版本根；拒绝绝对路径、空段、`.`、`..`、反斜线、NUL、冒号和百分号 |
| 路径唯一性 | 拒绝重复路径、文件/目录冲突和 Unicode 大小写折叠碰撞；隐式父目录也参与校验 |
| 压缩比 | 实际解压总字节 / 完整 ZIP 实际字节不大于 200，含等于边界 |
| 类型和权限 | 普通文件/空目录及必要执行位；目录不携带数据或 CRC；拒绝 ZIP 链接、设备、setuid/setgid/sticky 和加密条目 |

本配置只生成单一 `<version>/` 根，根外不能夹带文件；唯一 `plugin.yaml` 必须位于该根，并再次核对正式 id、版本和 target。冻结 staging 内需要的依赖链接已展开为普通文件，ZIP 中不保留链接。其他资源 YAML 的既有 2 MiB 限制与 manifest 256 KiB 限制分别维护。

打包完成后从同一已打开的 ZIP 文件重新读取目录和完整内容，全部校验通过后才登记摘要、只读文件和 ready 回执。拒绝的本次 ZIP 会删除，失败任务保留错误，artifact 为空。verify 在校对原文件摘要后也重新执行发布包准入；即使历史记录的摘要与文件相符，不合格文件仍明确失败，原文件和记录不改写。

## 5 状态、进程和预算

状态为 queued → prechecking → building → packing → verifying → ready，异常为 failed/cancelled/interrupted。没有真实完成的阶段不标成功；只有 ready 附带可校验的冻结产物。

记录当前 PID，以及所属父 PID、进程组、源副本相对目录和开始/退出时间。make 与正常子进程进入独立进程组；取消先发送 TERM，最多两秒等待后终止所属组的剩余进程。执行器退出也等待所属构建收敛，不按端口或通用进程名清理。

重启仅在能取得构建锁时把遗留活动记录改为 interrupted，清除当前 PID，不根据过期 PID 杀进程、不自动重放、不把遗留 ZIP 认作成功。新构建使用新 UUID；失败任务和脱敏诊断保留供核对。

源码及单次受控树复制上限 8 GiB/200000 文件，资源 YAML 单文件 2 MiB；冻结发布包额外遵守第 4.1 节，正式 manifest 上限为 256 KiB。构建时限 30 分钟。诊断日志最大 4 MiB，单行过长及总预算耗尽均写机器标记；过滤令牌、私钥、认证、Cookie、URL 和本机路径。它不是逐字完整的原始工具输出。上述是复制、任务和产物预算，不声称任意受信 Makefile 已受磁盘配额或网络沙箱隔离。

## 6 验证边界

受控测试运行真实 make/Go/Node 工具，生成 Linux ELF 和静态 JS 文件，验证快照隔离、源变化、版本值、ZIP、引用资源、原生文件拒绝、篡改、会话 HTTP、取消进程组及中断恢复。Base 真实 Nuxt 在 Linux 上的启动证据见上述 Market 准入交接。

2026-11-09 的历史验证机器为 macOS，当时 Docker 守护进程不可用，未启动。该批受控构建证据不代表实际 Linux/Nuxt、市场上传或宿主安装通过。后续容器与页面验证按独立记录给出，不追改历史结果。

受控测试、进程清理、模板消费和 CLI 准入检查见[本次验证记录](evidence/plugin-publisher-build-validation-20261109.json)。源码及临时编译产物未正式发布，用户全局 CLI 未更新。

2026-11-10 的发布包准入修复及边界回归单独记录于[发布包准入验证记录](evidence/plugin-publisher-package-admission-validation-20261110.json)，保留上述 F03 历史证据。修复仅涉及 Framework 当前仓库，市场 P01/P04、真实 Linux/Nuxt 构建、市场上传和安装启用仍须分别验收。

## 页面管理的 Linux 容器构建（2026-11-15）

- 新 HTTP 构建请求显式携带 executor=native/container，记录所用执行方式。macOS 默认页面显示“Linux 容器”，环境未就绪时禁用构建，并提供启动及重新检查动作。
- 容器仅绑定当前任务的受控副本，不绑定宿主源码、home、钥匙串、dotenv 或 Docker socket；只允许本地 Unix Docker context。镜像通过宿主受信网络获取并导入；现有无凭据网络代理可用于工具链及锁定依赖下载，回环代理明确映射为容器 host gateway。
- 源码快照排除 node_modules、IDE 状态及秘密文件。只对 Base 源码仓库中正式的 `contracts -> ../contracts` 增加显式目录授权，其余跨边界链接仍拒绝。
- 本地依赖仅允许具备正式模块／包身份的 Framework Go、Admin、Client SDK；单独复制并记录 SHA256，重写仅发生于构建副本。Go 下载缓存仅复制源模块锁定版本对应的 mod/info/zip/hash 文件；不复制 VCS 凭据或用户配置。
- 完成后复核原始源码及 SDK 摘要，继续通过既有 ELF、CGO、Nuxt 运行时、清单、路径、ZIP 条目与摘要准入；取消和退出清理本次容器，保留原任务及回执。容器镜像 ID 和 SDK 摘要保存在技术记录。

历史 macOS 拒绝构建及 fixture 记录保留其当时边界；本轮结果见 GUI 发布验证记录，不能替代正式市场开放、公开下载或 PowerX 安装验收。

本轮容器构建在 Base／Framework 源码仓库解析同源 CLI，并交叉编译 Linux 校验工具到私有构建目录，记录 cli_sha256。独立插件的 Linux 校验工具分发随 SDK/CLI 工件另行交付；本轮不发布这些版本，也不将 Base 验收当作独立已发布 SDK 消费证明。

冻结资源按已识别的前端源码布局（web-admin 或 Base 的 web-admin/nuxt）读取 public 与 i18n，统一写入安装路径 web-admin/public 与 web-admin/i18n。契约文件的资源存在性只校验 path/schema_ref/imports/descriptor/input/output/entrypoint 等声明的引用字段；schema ID 不按文件路径解释，缺失的真实引用仍拒绝。
