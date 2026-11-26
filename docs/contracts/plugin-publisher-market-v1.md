# Framework 市场发布适配（F04—F07）

2026-11-10，消费市场 `powerx.plugin-publisher/v1`、`1.0.0-rc.2`。

权威为市场正式 OpenAPI 与便携包；本仓库的[完整快照](../../contracts/plugin-publisher/v1/README.md)保持原文件，`bundle_sha256` 为 `89141c00e0b99b6a803312119e4cbcb3e64224bf458e234e35fc5ffbdeacd8fa`。`python3 scripts/contracts/check-plugin-publisher-bundle.py` 验证 24 个原文件摘要、JSON 副本及仅改 package 名的 Go 参考实现。rc.1 的历史证据不覆盖、不替换。

## 1 连接与凭据

`publisher market add` 创建不可变连接 UUID，保存名称、API root、独立批准的 console base、契约版本及摘要。地址必须规范、无凭据/查询/fragment，生产使用 HTTPS；HTTP 仅接受显式回环 IP。修改信任边界须新增连接，不能复用旧凭据。

开发页面通过受保护的 `POST /v1/markets/{connection_id}/bind` 一次性提交发布令牌；CLI 自动化仍可用 `publisher market bind --id <connection_uuid> --token-stdin`。先验证 discovery、me、插件归属和权限，再写系统凭据库。macOS 使用 Keychain，Linux 使用 Secret Service 的 `secret-tool`；不可用明确失败，没有明文文件、环境变量或旧登录凭据兜底。令牌不进入命令参数、URL、持久浏览器状态、日志、任务记录或容器；绑定弹窗在提交、关闭和卸载时清空输入。

连接只保存稳定用户/开发者身份、令牌 UUID、scope、插件范围、到期时间及验证时间。更新令牌必须匹配同一用户和开发者 UUID。`market disconnect` 只移除本机凭据，不替代市场撤销。市场令牌管理仍由市场 Cookie/CSRF 页面及其 API 负责。并发绑定/校验通过私有目录锁串行处理。

## 2 上传与回执

本地 `POST /v1/uploads` 仅接收 `connection_id/build_id`，立即返回异步任务；不让长上传经过本地 HTTP 的 10 秒预算。引擎将上传 UUID、市场 operation UUID、用户/开发者/插件、计划、原冻结包摘要/大小、规范化请求摘要和尝试次数在网络写入前原子落盘。

恢复先读取原 operation；已有成功结果不依赖本地 ZIP 仍存在。只有确需发送文件时才读取原计划及原文件。每次传输重新核对冻结 ZIP、manifest、ELF/纯 JS、路径和预算，不重建、不重新压缩。multipart 恰好八个字段，文件为 `application/zip`；发送期间计算实际文件摘要并记录字节进度。字节发送完成显示市场处理中。只有上传回执的 operation/plugin/version/target/SHA256/size 均匹配、可信链接通过校验，且按回执 Version UUID 读取到当前版本并包含对应 Artifact，才显示上传完成。不可变回执与当前版本状态分开保存。

普通连接/请求预算为 10/30 秒，上传写空闲 60 秒、响应头 1800 秒、客户端总计 1860 秒。所有请求禁止重定向和环境代理，不发送 Cookie、Host API key 或业务 JWT。返回体最多 8 MiB，经 rc.2 严格 JSON/schema/语义校验；不把远端自由文本或原始异常显示给用户。

## 3 断网、取消与进程恢复

丢失响应、可重试错误和 202 都先读取原 operation。恢复使用便携包 `RecoveryAction`，保留同一 operation、请求指纹及文件，最多三次整包传输；总恢复预算 7200 秒、每轮轮询窗口 120 秒、至少间隔 2 秒、Retry-After 最多 300 秒，等待必须同时装入剩余预算。退避采用 full jitter。预算耗尽保存 pending，不能换 operation 绕过。

同一构建和连接已有未解决或成功记录时，创建新上传返回冲突。任务与市场连接均使用跨进程锁；重启不自动重发，原活跃任务在读取时呈 pending，显式恢复后查询原操作。恢复授权以稳定用户/开发者身份为准，允许同身份换发令牌。缺少写 scope 仍可读取成功回执和处理状态，但不能重传。

取消只停止本进程拥有的传输；随后进行一次有界只读核对，保存成功/确定失败或 pending，不删除市场草稿。服务退出停止任务并保留恢复记录，不按旧 PID 结束其他进程。其他 CLI 进程不能冒充任务所有者执行取消；应通过启动任务的服务取消。

市场 discovery 缺少 `package-upload` 或 `idempotent-writes` 时明确阻断写入。当前市场源码仍返回空 features；生产不得用测试包装器、请求头或本地开关绕过。

## 4 页面与独立消费

开发态 `/powerx/publisher` 的连接分区配置市场、展示身份/权限/到期与私密绑定弹窗；上传分区确认目标和冻结版本、显示传输/市场处理/恢复状态、读取当前版本并打开可信详情链接；市场版本列表按 page/size 分页并展示总数，不把当前页当作全部历史。绑定请求只在本机同源开发代理和工具会话校验后转发；响应仅含脱敏身份与权限。任务和错误在分区外可见，选择与表单在切换分区时保留。写响应不确定后禁止再次写入，须显式读取已有记录。

Nuxt 包内页面显式导入 Nuxt composable 和 Nuxt UI 组件；页面 meta 由开发模块注册，避免依赖源码工作区才有的自动注入。生产不注册页面、代理和凭据配置。模板通过 `sync-templates --check`；独立 Go 消费用内容寻址模块代理快照、`GOWORK=off`、无 replace，前端消费实际 npm tarball，无工作区链接。

## 5 验证边界

[本轮证据](evidence/plugin-publisher-m1-validation-20261110.json)记录实际命令、隔离市场源码、MinIO、浏览器及清理。已核对市场原始关闭能力；正向联合测试只在隔离副本的测试 HTTP 包装器中显式开放 capabilities，业务上传/恢复/版本路由及对象存储使用市场源码。

这证明适配和真实字节链路，不能代替原生 Linux/amd64 上完整 Nuxt 插件构建与运行验收、生产系统钥匙串验收、市场正式开放/P08、SDK 正式发布、公开下载或 PowerX 安装启用。M2 送审、多平台和 CI 工作流触发不属于本轮 M1 实现。
