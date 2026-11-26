# Framework 送审适配（F08／M2）

2026-11-11，方案 v0.9；使用未修改的 `powerx.plugin-publisher/v1`／`1.0.0-rc.2` 便携包。市场 P07 已实现机器送审、权威资料选项、申请分页与 M2 显式授权，默认关闭；驳回重提返回新的 Publication UUID，旧资料、制品集合和回执保持不变。本端已完成源码与受控检查，本轮独立生成插件的开发页面已在真实隔离实例验证送审主链路（受控预上传包），其余联合项目单列。

## 1 入口与职责

开发页面在已有版本、构建、上传、连接分区之外增加“资料与送审”；桌面保持左侧导航，窄屏使用五项紧凑导航。市场审核记录按需进入弹层，填写资料使用“发布资料／安全与隐私”分区弹窗，确认时展示完整平台集合和规范化资料。所有草稿仅保存在 Vue 内存，切换页面分区保留；已创建任务从本地私有记录恢复。

CLI 与页面调用相同 Go 引擎：

```bash
px-plugin publisher review prepare --entry . --connection-id <connection_uuid> --version-id <version_uuid>
px-plugin publisher review preview --entry . --request-file <submission.json>
px-plugin publisher review submit --entry . --request-file <submission.json>
px-plugin publisher review list --entry .
px-plugin publisher review get --entry . --id <submission_uuid>
px-plugin publisher review resume --entry . --id <submission_uuid>
px-plugin publisher review history --entry . --id <connection_uuid> --page 1 --size 20
```

`prepare` 从市场读取当前版本的完整制品集合与 publication options，允许查看 submitted/approved/published 版本，便于刷新或重启后的只读展示。可送审状态仅在真正写入前要求 stable 渠道及 draft/rejected，不能因放开读取而绕过写入检查。`preview` 仅本地规范化与校验，不发市场写请求；`submit` 必须经用户显式确认。`cancel` 仅停止所属执行器请求并只读核对，不能撤销市场申请、批准或公开上架。

市场继续负责凭据签发、scope、逐包字节复核、不可变申请快照、审核和公开上架。Framework 不添加市场功能路由，不兼容旧浏览器送审体，不用上传成功回执冒充送审结果。

## 2 能力、身份和资料

创建前使用已绑定的稳定用户、开发者与连接 UUID；每次网络写入前重新校验身份、插件归属及权限。需要 discovery 同时包含 `publication-submit/idempotent-writes`，凭据同时具有 `plugins:read/publications:read/publications:submit`。上传 scope 不授予送审权限。能力关闭时明确拒绝新写入；已有回执仍可用 publications:read 查询现状。

资料由 `description/license/min_core/support_url/icon_url/security_statement/privacy_statement` 组成，拒绝未知字段。三段正文各至少 10 个 Unicode 字符、最多 10000 UTF-8 字节；URL 至多 2000 字节。许可证与最低 Core 版本必须在市场当前返回的选项内，不制造默认值或替代版本目录。图标可为空；文本按 rc.2 去除首尾 ASCII 空白。原始 JSON 先检查 64 KiB 字节预算，拒绝 BOM、重复键、非法 Unicode、未知字段和尾随文档。

## 3 完整集合与快照

客户端使用 `version_id`，每个制品使用 `artifact_id/os/arch/sha256/size`。页面展示当前版本的全部 1—6 个平台包，不提供只选主制品或静默排除其他包的操作。普通界面显示平台、大小与版本；完整 UUID、SHA256 和请求指纹放在明确的技术详情中。

发送前重新读取版本，核对 stable 渠道、draft/rejected 状态、真实版本名及完整制品集合。集合比较包含 UUID、平台、摘要和大小，顺序不影响结果；任何缺失、替换或多出制品都停止本次写入，要求刷新后新建任务。该检查不代替市场逐包读取实际存储字节；“第二个平台包损坏”的真实门禁必须由 P07 验证。

任务在网络写入前原子保存：任务 UUID、operation UUID、用户/开发者、插件、版本 UUID、规范化资料、完整集合和 rc.2 指纹。记录位于 `.px-plugin/publisher/submissions/<submission_uuid>.json`，目录 0700、记录 0600，使用固定目录描述符及锁，单条记录最多 1 MiB。该预算覆盖原始提交快照与当前申请 DTO，不改变市场请求预算。

## 4 重试、恢复及驳回

JSON 写请求总预算 60 秒，连接 10 秒、响应头 30 秒；不使用文件上传的 1860 秒预算。禁止 HTTP 重定向和底层请求体自动重放。明确收到成功、202、可重试错误或丢失响应后，按 rc.2 查询原 operation；同 key、同资料、同集合最多三次写入，7200 秒总预算、120 秒轮询窗口、至少 2 秒间隔及不截断的 Retry-After 共同约束等待。

任务重启后不自动重发，显式恢复复用原 operation。未知结果禁止通过新 operation 绕过；取消保留待核对记录并执行一次有界只读核对，不删除或撤回市场申请。新凭据必须保持原用户和开发者身份；只读凭据可查询原结果，不能重传。

送审回执与当前申请状态分开保存。按回执 Publication UUID 在有界分页接口中读取申请，核对插件、版本和原始资料／集合指纹；查询失败清除旧的当前状态，保留回执。得到 submitted 表示已登记申请，approved 仍不意味着公开上架。本地任务完成后停止自动状态查询，后续人工审核使用显式刷新或分页记录查询。

“送审制品集合”的状态取同一市场连接、插件和准备版本的最新本地送审任务，经市场查询确认的当前申请状态；不再沿用准备快照中的 draft/rejected。查看旧任务不会把当前版本卡片切回旧申请状态。最新任务仍在核对或查询失败时显示“状态待核对”，不借用旧状态；尚无匹配任务时才显示准备时读取的版本状态。准备资料、制品集合与旧申请回执保持原值，不增加完成任务后的自动轮询。2026-11-12 的四态真实页面回归和截图见[状态同步验证](evidence/plugin-publisher-review-status-validation-20261112.json)。

驳回后先刷新原申请状态，再读取当前完整集合并编辑资料；只有用户再次确认，才创建新的任务和 operation。原任务、指纹、资料、集合及回执保持不变。若新操作返回已知旧任务的同一 Publication UUID，本端按协议异常保留待核对，拒绝覆盖旧回执。旧任务仍待核对或尚未驳回时拒绝新建重复送审。

## 5 本地 API

所有接口复用本地 Host／Origin／一次性会话边界，生产不注册开发代理；不接受浏览器令牌。

| 资源 | 行为 |
| --- | --- |
| POST `/v1/submissions/prepare` | 当前市场版本、完整集合及资料选项 |
| POST `/v1/submissions/preview` | 规范化输入和指纹，不执行市场写入 |
| GET/POST `/v1/submissions` | 列表／持久化异步送审任务 |
| GET `/v1/submissions/{submission_id}` | 读取本地阶段、回执及当前申请 |
| POST `/v1/submissions/{submission_id}/resume` | 恢复原操作或读取已知回执当前状态 |
| POST `/v1/submissions/{submission_id}/cancel` | 停止所属请求并保留结果核对 |
| GET `/v1/markets/{connection_id}/publications?page=1&size=20` | 当前插件的市场申请分页 |

## 6 市场 P07 对齐结果及联合验收

- P07 已实现上述正式路由和 M2 scopes，实例仍须显式启用 submission、discovery 等开关并通过数据库／存储／维护健康检查；Framework 消费真实 discovery，不修改响应。
- P07 已实现逐包字节验证、事务重查及不可变申请快照；当前准入仍限完整 Linux/amd64 profile，1—6 集合结构不代表 M3 多平台准入。
- 驳回后新 key 取得新的 Publication UUID，旧 key 恢复旧申请。页面按当前申请查询显示“审核通过，待上架”；管理员显式上架后才显示已公开上架。
- 联合验收仍需真实提交、驳回修订、第二个平台包损坏、批准仍私有、显式公开上架及下载。相关证据不能由本轮 HTTP fixture 代替。

本轮结果见[验证记录](evidence/plugin-publisher-f08-validation-20261111.json)。M1 的 Linux／系统凭据／正式开放及 SDK 发布边界继续保留，未因 F08 源码交付而完成。

## 7 HTTP 请求记录与诊断

任务运行、恢复和取消后核对中的每次市场 HTTP 请求在发送前保存意图，在结束后原子更新。记录位于 `.px-plugin/publisher/requests/{uploads|submissions}/{task_uuid}/{client_request_uuid}.json`；目录 0700、文件 0600，单条上限 4 KiB，不嵌入原任务预算。只保存任务和 operation UUID、客户端和规范市场请求 UUID、固定响应类型、GET/POST、状态码及起止时间；不保存 URL、令牌、请求正文、响应正文或私有存储引用。

缺少确认响应时保留本次客户端请求 UUID，市场 UUID 留空，不制造市场已接受的证据。意图没有完成时间也不能当作业务失败；继续恢复原 operation。记录持久化失败明确停止本次请求或保留原操作待核对，不静默省略诊断。

GET `/v1/{uploads|submissions}/{task_uuid}/requests?page=1&size=20` 使用本地工具会话及既有 Host/Origin 门禁，默认 20、上限 100；拒绝未知或重复 query。页面在任务技术详情中按需分页读取，显示 Publication/Version UUID 及可信申请链接。CLI 使用 `publisher upload requests --id <upload_uuid>` 或 `publisher review requests --id <submission_uuid>`，均支持 page/size。

远端固定错误通过 `params.request_id` 保留已验证的市场请求 UUID，代理只转发此项诊断，不转发远端正文。HTTP 状态与请求 UUID 用于关联尝试，不能代替当前申请状态或业务成功回执。

本轮修正、真实系统凭据库的合成写读删除、独立包消费及环境阻塞见[当前验证记录](evidence/plugin-publisher-p07-alignment-20261111.json)。此前 F08 测试不升级为真实联合验收。

2026-11-11 实际页面联合结果见[本轮验证](evidence/plugin-publisher-m2-page-joint-validation-20261111.json)及[联合记录](../marketplace/plugin-publisher-framework-m2-joint-record.md)。原 operation 在丢响应、刷新及执行器重启后恢复；驳回重提返回新 Publication UUID；批准仍私有、显式上架及同身份只读换发恢复均通过。输入为受控草稿，未验证原生完整包、Framework 实际构建上传或公开下载。所有本轮环境和凭据均已清理。

2026-11-17：真实 Base 0.1.5 原包已完成页面上传、送审和丢响应／页面刷新／执行器重启恢复。修正送审后 prepare 只读资料被状态门禁拦截的问题，保留写入门禁。四态人工审核尚待市场维护者操作；见[本轮证据](evidence/plugin-publisher-base-page-joint-validation-20261117.json)。
