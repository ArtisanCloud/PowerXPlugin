# 恢复副本的只读启动

Core 的恢复验证使用 PostgreSQL 只读角色。启动时依据数据库的 `transaction_read_only` 检查 schema：已有结构只读取；缺失返回 `RECOVERY_SCHEMA_MISSING`。不会借恢复启动执行迁移、建表或初始化种子数据。

`db.IsReadOnly()` 表示数据库实际限制；`db.BackgroundWorkersDisabled()` 同时接受宿主明确注入的 `POWERX_DISABLE_BACKGROUND_WORKERS=true`。只读副本和本地隔离演练停用后台消费者、调度、渠道连接和补偿任务，避免演练处理原业务任务或向外部渠道发送消息。该开关不授予写权限；正常可写启动保留原有行为。

AI Craft 的内置回归定义同步和安装初始化在只读验证阶段跳过；需要的结构与配置必须已经存在于快照中。原有 Framework 绑定和授权检查继续运行，不能用跳过鉴权或扩大授权来通过恢复验证。Skeleton 的本地 pgvector 自动建表同样不在只读验证期间执行。

重新构建适配后的插件并通过正常发布／安装流程升级后，再建立新快照。修改模板或 Core 不会更新已安装的插件二进制。对旧包使用明确的程序覆盖进行测试时，要单独注明它不是原产物原样回放。

临时 PostgreSQL 回归测试通过 `POWERX_TEST_RECOVERY_POSTGRES_DSN` 指向测试专用实例后执行 `go test ./internal/db -run Recovery -v`。测试会创建专用 schema，不能指向日常或正式数据库。它检查已有 schema、缺失 schema、只读拒绝 DDL，以及正常可写创建路径。

宿主托管与恢复实例按 `POWERX_PLUGIN_ROOT` 解析插件清单、能力合同和打包的应用配置，避免依赖 Core 工作目录或固定执行镜像的位置。显式 `CONFIG_PATH`／清单路径仍优先；指定宿主根目录后，不再搜索协调器或源码目录。应用配置应随发布包提供 `config/config.yaml`、`backend/etc/config.yaml` 或 `etc/config.yaml`；`host-values.yaml` 的环境绑定不等于应用默认配置。AI provider 目录等新构建所需文件必须与二进制一并升级。

相关 Skeleton 修改同步到 `scaffold/templates` 与 CLI 内嵌模板，后续生成插件同样使用上述规则。

恢复模式只在宿主明确提供 `POWERX_RECOVERY_READ_ONLY` 时重绑定插件 Redis：`POWERX_RECOVERY_REDIS_SOURCE_ADDR` 表示已经采集的原实例，`POWERX_REDIS_ADDR` 表示独立副本。只接受同一来源的 WS、TaskBus、幂等及缓存 URL，保留 DB 编号；额外实例和 TLS Redis 明确拒绝。正常启动不自动改写这些 URL。
