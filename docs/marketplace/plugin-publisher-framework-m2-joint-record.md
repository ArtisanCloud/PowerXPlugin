# Framework／Market M2 联合记录

2026-11-11，方案 v0.9，协议 `powerx.plugin-publisher/v1`／`1.0.0-rc.2`。

**本轮已使用独立生成插件的 Framework 开发页面，连接真实隔离 Market 完成送审主链路。** 输入为 Market 实验环境经真实上传接口登记的受控 Linux/amd64 ZIP；完整原生插件构建、Framework 实际构建上传、公开包下载和 PowerX 安装启用仍待验收。环境、凭据及临时进程已关闭并清理，旧地址不再可用。详细证据见[本轮验证](../contracts/evidence/plugin-publisher-m2-page-joint-validation-20261111.json)。

2026-11-16 历史进展：本记录保留上轮受控包 M2 的历史结果。实际 Base 0.1.5 已完成网页构建和包内 Nuxt 容器启动，真实上传被市场 P04 的 Schema ID 路径误判阻塞；市场维护者负责修复交付，Framework 接续自身页面验收。新问题、证据、参考方案及双方清单见[协作交接](plugin-publisher-market-admission-handoff-20261116.md)。

2026-11-17 Framework 接续：Market 已交付 Schema ID 准入修复（协议仍为 rc.2）；Framework 使用同一 Base 0.1.5 冻结 ZIP，在自己的发布页面完成真实上传、系统凭据绑定和送审，取得 Version、Artifact、Publication 回执。模拟浏览器丢失送审响应后恢复原任务与 operation，刷新页面及重启临时执行器后回执保持一致。修正了送审后刷新完整制品集合被草稿状态门禁阻挡的问题：资料读取允许查看审核中及已审核版本，stable/draft/rejected 检查仍在每次真正写入前执行。该包的驳回重提、批准待上架及上架须由市场维护者操作后继续验收，当前不宣称完成。见[本轮页面证据](../contracts/evidence/plugin-publisher-base-page-joint-validation-20261117.json)及[市场协作步骤](plugin-publisher-base-review-coordination-20261117.md)。

## 环境与实际交付物

- 按[市场交接记录](../../../../PowerXPluginMarket/docs/marketplace/plugin-publisher-market-m2-joint-record.md)在 Market 根目录运行 `python3 scripts/publisher-joint-lab.py`；使用本轮新 handoff，不复用历史地址。实际 PostgreSQL、MinIO、Identity/Publisher/Catalog Gin 与生产 Nuxt 均由该脚本隔离启动。
- Framework 当前 CLI 生成 `com.example.publisher-joint` 的独立插件；前端从实际 npm tarball 安装 Admin `0.0.14`／Client `0.0.15`，未用 workspace symlink，也未正式发布 SDK。摘要见[消费记录](../../artifacts/plugin-publisher-m2-real-page-20261111/consumer-provenance.json)。
- 在真实市场网页签发仅上传、M2 送审、只读恢复三组凭据，经 CLI 私密 stdin 绑定到同一个连接，存入本机 macOS 系统凭据库。用户和开发者 UUID 保持相同；令牌未写入源码、记录、截图或日志，三组凭据均已撤销，系统条目删除后查找返回不存在。
- 市场侧使用本次专用实验 root 账号，通过真实后台页面执行驳回、批准和上架；不代表普通发布者取得审核权限或生产人工审查完成。
- 唯一故障注入为市场实验代理丢弃首个已提交的 201 响应，以及 Framework 临时代理暂停原 operation 查询；正式路由、DTO、能力和业务响应未改写。代理随后移除暂停，全部临时服务已关闭。

## 联合结果

| 场景 | 本轮结果 |
| --- | --- |
| 授权范围 | 真实仅上传凭据不能启用送审；本人另外签发 publications:read/submit 后，页面读取真实选项和完整集合 |
| 预览与显式确认 | 页面选择权威 MIT／Core 1.0.0 测试快照，规范化资料并确认提交；未使用页面/业务响应 fixture |
| 丢响应、刷新及执行器重启 | 首次 POST 已真实提交但响应丢失；页面刷新，执行器重启后沿用原 operation 恢复，写入次数仍为 1，仅一份原申请 |
| 驳回重提 | 市场后台页面驳回；Framework 页面读取 rejected 后修订并再次确认，生成新任务、新 operation、新 Publication UUID |
| 不可变历史 | 原资料、完整包集合和提交回执保持；旧可信链接仍显示原申请原资料；真实申请分页总数为 2 |
| 批准待上架 | 市场后台页面完成四项人工检查并批准；Framework 刷新后显示待上架，真实公开目录查询仍为 404 |
| 显式上架 | 市场后台另行上架；Framework 当前申请状态变为 published，公开目录查询变为 200；未执行公开包下载 |
| 授权撤销与换发 | 撤销 M2 凭据后，页面显示规范市场请求 UUID，清除旧的当前状态并保留原回执；同身份只读凭据可恢复旧 rejected 申请及当前 published 申请，无新增送审写入 |
| 请求证据 | 任务技术详情按需读取真实客户端／市场 request UUID；无响应的 POST／查询与已确认响应分开记录 |
| 页面及收尾 | 1440 桌面浅色、390 窄屏手动深色且系统浅色下实际交互；页面错误、焦点警告均为 0；凭据、浏览器、服务、schema/bucket 与临时根均已清理 |

## 关键对象

| 对象 | UUID／摘要 |
| --- | --- |
| Version UUID | `6eb94ef9-1f20-4a9a-93bf-b70a721381e7` |
| Artifact UUID | `016cd120-91dc-470e-ad55-31e9e88ab453` |
| ZIP SHA256／字节数 | `743cdbae1308c23f78136030769a0a41570651b27bd6f7d5cb59b68c87ba742c`／`4415873` |
| 原 operation | `524c50c3-b0d2-4b70-adff-4577d434cc46` |
| 原 Publication UUID | `7b8a61da-841a-4816-b393-dcaf0bc476f0` |
| 重提 operation | `95085481-6a6c-4938-84cb-45525dc3af56` |
| 重提 Publication UUID | `6efcf61c-1232-4c08-aff7-a88b398abe3e` |

完整任务、指纹及逐次请求 UUID 见[页面联合记录](../../artifacts/plugin-publisher-m2-real-page-20261111/page-record.json)。这些是已关闭的隔离环境历史证据，不能用旧地址继续请求。

## 仍待验收

- 实际独立插件在原生 Linux/amd64 完整 `make dist`、启动、冻结包及 Framework 页面上传；本轮只有市场提供的受控已上传草稿。
- A15 选项服务不可用等剩余失败分支，以及 A16 集合增删/篡改/主包替换的当前双方源码联合回放；不把市场独立测试自动并入本轮页面结果。
- 公开包下载、License/购买及 PowerX 安装启用；本轮只验证公开目录可见性。
- 正式 SDK/CLI 注册表发布、远程 CI、正式部署代理与生产开放；本机 Keychain 结果不代替 Linux Secret Service。

此前 F08 与 P07 对齐证据保留其原时点边界；本轮真实页面结果由本文件和新证据单独记录。下一次仍在 Market 根目录启动新的前台实验环境，读取新 handoff；默认存活 20 分钟，结束时 Ctrl+C 清理。

## 2026-11-12：送审制品集合状态同步修复

此前卡片读取准备送审时的版本状态，驳回重提后即使新申请已批准或上架，仍会显示缓存的 rejected。现改为读取同一市场连接、插件与版本的最新送审任务所确认的市场申请状态；旧任务仍显示自己的申请状态，原资料、完整集合和回执不被覆盖。未核实当前状态时明确显示待核对。

使用重新启动的真实隔离 Market、CLI 生成的独立插件与修正后的 SDK tarball，验证待审核、已驳回、通过待上架、已上架四种状态；重提准备快照确实保留 rejected，批准与上架后卡片仍正确更新。两次送审各写入一次，取得不同 operation 和 Publication UUID；旧申请资料、集合、回执及旧控制台链接保持。批准后目录仍为 404，显式上架后为 200。

发布模块 54 项测试、严格 Vue/TypeScript 检查及模板同步通过。四态各有桌面浅色／窄屏深色截图，另补桌面深色／窄屏浅色；覆盖手动主题与系统相反、导航切换、默认／选中／悬停／聚焦／禁用文字，最低实测对比度 5.91:1。页面错误和焦点警告均为 0。见[截图索引](../../artifacts/plugin-publisher-review-status-20261112/README.md)及[独立验证记录](../contracts/evidence/plugin-publisher-review-status-validation-20261112.json)。本轮凭据已撤销、系统凭据条目已删除，临时服务及子进程、schema/bucket 和临时目录均已清理；受控包与原生包、公开下载、宿主安装的验收边界继续保留。
