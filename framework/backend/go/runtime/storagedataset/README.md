# 插件数据集 Go SDK

`runtime/storagedataset` 对齐 Core `feat/storage-datasets` 的 `a2690886` 提交和 T175–T179。合同来源为 Core 的 `docs/contracts/plugin-dataset-execution.md` 和 `specs/027-monitor-center/contracts/plugin-dataset-{host,execution}.v1.schema.json`。SDK 内嵌的两个 Schema 与该提交逐字节相同。

## 接入顺序

1. 在插件安装包的 `plugin.yaml` 声明 `data_lifecycle.<dataset_key>`，填写原文件 SHA-256 和 `observation_tables`。把声明文件打包到 `contracts/storage-datasets/<dataset_key>.json`。从可信的安装根目录调用 `LoadAssets`。在 Core 为该租户的插件服务授予登记能力后，用 `Client.Register` 登记。
2. 给同一服务主体授予 `com.corex.runtime.taskqueue.service_consume`、`com.corex.storage.datasets.preview_worker` 及所需的 `com.corex.storage.datasets.execution_worker`。用同一租户的 STS TokenProvider 配置 Framework Gateway 和 Host TaskQueue。SDK 按通知版本、租户、原请求、租约凭据和 fence 调用对应的 Core worker 能力。
3. 将源库的 `SourceLedgerMigration()` 纳入插件自身的正式数据库升级。由数据库所有者在迁移事务内设置 `SET LOCAL search_path TO <可信插件schema>, pg_catalog` 后执行一次。Worker 使用专用登录角色和独立连接池。该角色需要插件 schema 的 `USAGE` 和已声明业务表的 `SELECT`，执行删除时另授所需业务表的 `DELETE`。执行账本授 `SELECT`、`INSERT` 和状态列的 `UPDATE`，包括 `outcome`、`source_receipt_uuid`、`affected_count`、`archive_bytes`、`archive_sha256`、`sealed_at`、`delivered`。账本由迁移角色持有。归档工作时，一个批次需要账本事务和只读快照各占一条连接。
4. 由宿主提供安装代次 `POWERX_PLUGIN_INSTALLATION_REVISION`、稳定部署标识 `POWERX_ENVIRONMENT_ID`、绑定 UUID、源 schema 和受限角色。托管绑定 UUID 位于 `POWERX_PLUGIN_HOST_VALUES` 指向的文件内，集群标识由部署配置提供，与 Core 的登记配置一致。调用 `NewWorker` 绑定插件编译进来的 `Adapter`。进程启动及传输恢复时可调用 `RecoverPending`，运行期间由既有服务监管循环调用 `ReceiveAndHandle`；该循环可按运行需要触发有界的待交付恢复。按已授权动作消费 `PreviewSubscriber` 或 `ExecutionSubscriber`。Core 策略由 Root 启用，符合条件的预览由 Root 用于逐批派发。

## 业务适配器和回执

- `Scan` 使用授权租户、保留期限和稳定主记录 UUID，只返回已检查对象及合同定义的保护原因。业务插件负责确认终态、引用、可重试性、最新有效记录、明细完整性及保留资格。扫描使用确切的 schema 和结构版本。预览与归档在 PostgreSQL 只读快照内执行。
- 删除扫描在源事务内取得与业务引用及重试写入共用的事务保护。删除时再次检查租户、期限和业务保护，只返回实际删除的主记录 UUID。明细变更与主记录变更在同一事务完成。Framework 按主记录计数。
- 源账本按部署和原请求 UUID 锁定，绑定插件安装、租户、源绑定及原授权。业务更改、终态和待交付回执同事务提交。摘要以 Go JSON 固定结构序列化授权、回执和可选归档 SHA-256，其中用于上报的回执摘要字段在计算时为空。数据库触发器保护已确定的回执。Core 接受回执后才确认 TaskBus。
- 归档保留源记录，导出限制为 128 个 JSON 对象、256 KiB。Core 接受前，待交付账本保存完整 JSON Lines。恢复时原样上报。Core 确认后，源账本标记交付并释放副本，保留源回执和内容摘要。声明了 `archive_before_delete` 的数据集在 v1 可执行归档。
- `RecoverPending` 先重放已封存的回执，也处理超过期限但还没有终态的源账本预约。保留 Core 尚未确认的账本记录，包含待重传归档字节。

## 暂停与授权响应丢失

- Worker 的普通 TaskBus 领取可能得到 `reconcile_only`。存在终态回执时可按原批次重传。原租约已失效或不可用时，可用同一租户和原 STS 主体调用 `Worker.ReconcileExecution(ctx, requestUUID)`。请求 UUID 来自原 TaskBus 通知、待核对记录或 Core 批次详情。Core 只在原授权到期后允许无租约对账。
- 源端没有预约时，SDK 要求 Core 返回原 `expired_grant`，并核对绑定、源指纹、声明摘要、适配器及结构版本。源 PostgreSQL 时钟达到原截止时间后，SDK 按原请求 UUID 创建或锁定账本记录。已有终态就重放，没有终态才封存 `aborted`、零变更的回执。业务记录保持原状。
- 快照缺失时只重放已有终态；授权、时钟、来源或锁状态无法确认时留待后续对账。Core 确认同一持久回执之后，SDK 才标记交付。

## 源端回执时间

SDK 将 PostgreSQL 延迟约束触发器检查授权并封存回执时的 `clock_timestamp()` 作为 `source_sealed_at` 上报。业务变更与回执同事务提交。对 `committed`，封存检查必须早于授权截止时间，物理提交和 WAL 刷盘可能稍后完成。零变更的 `aborted` 回执可以在截止时间后封存。Core、Worker 与源 PostgreSQL 需保持时钟同步。

## 隔离验证

在 Framework 仓库根目录运行以下命令。Python 环境需有 `boto3`，并可调用 PostgreSQL、Redis、MinIO 和 Go。传入的 Core Checkout 需含 `scripts/dev/check-storage-datasets.py` 与 `backend/internal/bootstrap/storage_datasets_live_test.go`。

```sh
python3 scripts/testing/storage-dataset-framework.py /path/to/PowerX-storage-datasets
```

联合测试通过临时 Go 工作区和测试 overlay 复用 Core 的隔离启动脚本。合成插件使用测试安装目录和真实受限数据库角色。在托管和独立 PostgreSQL 中，测试在 Core 落账后丢弃首个预览响应，检查 SDK 重报的原始载荷、Core 回执以及三张源表的逐行内容。其余场景覆盖限时删除、保护、源库回滚、过期预约恢复、全局暂停后按原批次并发对账及版本化 MinIO 字节取回。

另加 `--with-package` 可用正式安装链路验收托管来源。脚本调用 Framework 的 `make package`，核对 ZIP 中声明原始字节及可执行文件，再将同一份 `dist` 上传为管理端接受的 `.tar.gz`。Core 安装任务通过 SSE 返回结果，并运行包内迁移和 worker。随后检查实际 schema、部署和安装代次、预览、原归档重传及 Root 取回。该合成包用于本地隔离环境。

实验性的 `px-plugin package` 在独立的 TAR 构建中复制 `data_lifecycle` 索引点名的声明原始字节，并把大小和 SHA-256 写入构建元数据。缺少声明、摘要不符或文件路径越界会导致打包失败。
