# 开发者发布工具本地契约 v1（F02/F03）

状态：2026-11-09 已实现本地计划、原子存储及受保护的回环服务；F03 已增加隔离构建与冻结产物的受控实现，见[构建契约](plugin-publisher-build-v1.md)；实际 Linux/Nuxt 运行验收、上传及送审未完成。协议为 `powerx.plugin-publisher-local/v1`，与市场候选 `powerx.plugin-publisher/v1` 分开。机器结构见[本地 OpenAPI](plugin-publisher-local-v1.openapi.yaml)。

本地协议由 Framework/CLI 维护，当前只注册在 CLI 启动的独立进程中，没有注册到 Skeleton 业务后端、生产 Nuxt 或插件安装包。本文的本地默认值不替代市场 P01 的令牌、请求及上传限制。

## 1 当前命令

在仓库构建 CLI；以下示例不会安装全局命令：

```bash
make build-px-plugin
./bin/px-plugin publisher plan create --entry ./skeleton --version 1.2.0
./bin/px-plugin publisher plan list --entry ./skeleton --page 1 --size 20
./bin/px-plugin publisher plan get --entry ./skeleton --id <release_plan_uuid>
./bin/px-plugin publisher serve --entry ./skeleton --origin http://localhost:3131
```

在独立生成插件中，使用实际 CLI 可执行路径，并将 entry 指向该插件根目录。`create` 可传 `--notes-file`、`--channel stable`、`--os linux`、`--arch amd64`。更新说明最多 10000 个 UTF-8 字节；正式版本不得含 v 前缀或预发布段，可带合法 SemVer 构建元信息，最多 50 字符。

本地计划只接受 stable 和计划目标 linux/amd64；创建计划不证明执行器具备该平台构建能力。当前 info 返回 release-plans/local-builds；通过原生 Linux/amd64 和工具准入才返回可构建目标，macOS 返回空集合。build 命令及接口见 F03；upload/submit 未注册。

CLI 成功输出 `data` JSON。退出码 0 为完成，2 为输入或工作区无效，3 为身份/源码版本变化或执行器冲突，其他错误为 1。帮助和可见诊断读取内嵌 en/zh locale（LC_ALL 优先于 LANG）；错误的底层原因不会原样暴露本机路径。

## 2 存储及对象身份

状态保存在受信工作区 `.px-plugin/publisher/`。计划路径为 `plans/<release_plan_id>.json`，schema 为 `powerx.plugin-publisher-plan/v1`；计划不可变，UUID 是稳定对象引用。当前 create DTO 只包含 version、channel、target、release_notes，插件身份由工作区正式 id 解析，不接受客户端指定归属、工作目录或命令。

Linux/macOS 使用固定目录描述符、相对系统调用和 O_NOFOLLOW；publisher/plans 目录必须为私有目录，记录权限为 0600。临时文件先写入和 fsync，再以不覆盖目标的原子硬链接提交；同目录 fsync 后返回。中断产生的 `.pending-*` 不被当作业务记录，其余非法文件、损坏 JSON、未知 schema、过宽权限和跨插件记录明确报错，不静默跳过或降级。

用户的源码 manifest 不被改写。每次 create 复核启动时的插件 ID 和源码声明版本；变化则要求重新打开工具。计划来源暂不构成构建输入快照，F03 已实现完整输入指纹、构建任务、冻结产物和所属进程组管理，真实 Linux/Nuxt 运行仍待验收。

当前仅支持 Linux/macOS 的安全存储实现；其他平台明确返回不支持，不回退到宽松的文件实现。生成插件已有 `.px-plugin/` 忽略规则，仓库根目录也排除本地工具状态。

## 3 服务启动与会话握手

`serve` 默认监听 `127.0.0.1:0`，实际空闲端口随启动 JSON 返回；可显式传回环 IP 和端口。拒绝通配地址、空 Host、DNS 监听名和公网地址。`--origin` 必填，只接受精确的 http/https 回环 origin（包括 localhost），不接受路径、query、userinfo、null 或通配符。

同一工作区仅允许一个服务持有 executor.lock 的系统锁；进程退出释放锁，不靠删除锁文件决定是否存活。启动 JSON 包含 protocol、instance_id UUID、endpoint、相对 session_file、bootstrap_expires_at 及诊断 PID，不包含凭据或绝对工作区路径。

一次性随机 256 位引导凭据写入 0600 的 `.px-plugin/publisher/tool-session.json`；该文件只能由受信本地代理/工具读取，不通过公开静态目录、URL、日志或 public runtime config 传递。后续 F05 代理尚未实现，不能宣称页面已经接入。

握手步骤：

1. 请求 `POST /v1/session`，携带精确 Origin 和 `Authorization: Bearer <bootstrap_token>`，无请求体。
2. 服务检查 Host 等于实际监听 IP:port，并检查 Origin、凭据和五分钟有效期。
3. 成功返回一次 session_token 和 expires_at，引导凭据立即失效；重放返回 401。
4. 其余路由使用这个会话令牌，有效期一小时。进程内只保存令牌 SHA256，使用固定时间比较；不接收 Cookie、旧配置、Host Key 或市场令牌作为替代授权。
5. 到期后返回 401；当前不自动续期，需要显式重启工具。服务重启生成新的 instance UUID 和凭据，旧令牌不复用。

没有 CORS 放行或缺失 Origin 的兜底；同源代理需要传递已配置的浏览器 Origin，并把上游 Host 设置为实际执行器地址。开发工具会话不包含市场发布权限。

## 4 本地接口与预算

所有接口均检查 Host/Origin 和适用的 Bearer；错误为 `{error: {code, message_key, params?}}`，不携带人类错误正文或底层异常，界面按 locale 映射。成功为 `{data: ...}`。响应设置 no-store 和 nosniff。

| 方法与路径 | 输入 | 成功结果 |
| --- | --- | --- |
| POST /v1/session | 一次性引导 Bearer，无请求体 | 201，短期会话及到期时间 |
| GET /v1/info | 会话 Bearer | 200，插件身份、源码声明版本、真实支持能力 |
| POST /v1/plans | JSON CreatePlan，会话 Bearer | 201，已落盘的本地计划 |
| GET /v1/plans | page/size，会话 Bearer | 200，items/total/page/size |
| GET /v1/plans/{release_plan_id} | canonical UUID，会话 Bearer | 200，已存在计划 |

page 默认 1，size 默认 20、最大 100；拒绝重复或未知 query。读出全部有效计划后排序分页，不把 200 条当作全部历史。空列表返回 items 空数组。路径 ID 不接受 numeric id、非规范 UUID 或路径表达式。

JSON 请求最大 64 KiB；拒绝未知字段、重复键、无效 UTF-8、多份 JSON 和非 application/json。读记录预算 128 KiB。读请求头 5 秒、完整请求及响应写入 10 秒、空闲连接 30 秒、请求头最大 16 KiB；这些预算仅服务本地计划接口，不适用于未来市场文件上传。

主要错误：400 输入/工作区无效，401 会话或引导凭据无效，403 Host/Origin 不符，404 未找到，405 方法不支持，409 执行器冲突或工作区身份变化，413 请求过大，415 媒体类型不符，500 记录损坏/安全存储失败，501 平台不支持。底层 cause 只在进程内保留。

## 5 退出与验收边界

Ctrl-C、SIGTERM 或调用方 context 取消触发关闭监听、最多五秒等待所属 HTTP 请求结束、清理属于当前 instance UUID 的引导文件，再释放工作区锁。不会按端口或进程名清理其他服务，不会删除本地计划；F03 会取消并等待所属构建进程组，不根据旧 PID 结束其他进程。

受控测试覆盖并发及 200 条以上分页、重启读取、来源变化、损坏/过宽权限记录、目录及记录 symlink、固定目录被替换、Host/Origin、引导重放、会话到期、请求结构、服务互斥和退出后端口释放。实际 CLI 验证使用临时独立生成插件，不启动该插件业务服务或连接市场。

F02 的后续页面代理消费在 F05 验收；F03 构建/进程取消已通过受控测试；实际 Linux/Nuxt 运行、F04 市场写操作恢复尚未验收。本地计划创建不等于构建成功或市场草稿，不能据此标记 M1 或主方案 A01—A14 整体通过。

本次受控测试、实际 CLI 独立生成插件验证及服务清理结果见[验证记录](evidence/plugin-publisher-local-validation-20261109.json)。源码及临时构建未正式发布，未更新用户全局 CLI。
