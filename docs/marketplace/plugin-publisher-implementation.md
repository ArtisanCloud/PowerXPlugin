# 开发者发布中心实现设计与交付范围

版本：0.10（Framework 接续）。日期：2026-11-17。状态：Market 已交付 Base 原包准入修复；Framework 使用同一 0.1.5 Linux/amd64 包完成真实网页上传、送审和原操作恢复。该包的驳回重提、批准待上架及上架尚待双方继续；此前受控包 M2 结果单列。正式开放、Linux 系统凭据、公开下载和 PowerX 安装分别验收。

本文与[开发者发布中心与市场对接方案](plugin-developer-publishing-center-plan.md)共同使用，并在 PowerXPlugin 与 PowerXPluginMarket 的 `docs/marketplace/` 下维护同一正文。本文落实架构、命令迁移、文件范围和分阶段依赖；机器路径、字段、状态及验收编号以主方案为统一索引，正式 OpenAPI 在 P01 冻结。

F05 本地开发入口为 `/powerx/publisher`，沿用后台主布局，采用桌面纵向分区、窄屏紧凑导航和独立工具会话；使用方式见[页面操作指南](../guides/features/plugin-developer-publisher/guide.md)。生产不注册页面及代理。真实 Go 计划操作与受控构建页面样例分别记录，不将其等同市场上传或真实 Linux/Nuxt 运行验收。

2026-11-16 GUI 更新：开发服务自动识别受信插件工作区、准备并清理同版执行器；页面自动连接，不再要求手工运行 CLI 或填写配对环境变量。构建分区明确选择本机 Linux／Linux 容器，macOS 可在页面启动已安装的本地容器引擎。市场连接增加私密令牌绑定弹窗，令牌经本地受保护接口验证后只保存于系统凭据库，不进入日志、任务、持久浏览器状态或容器。此调整属于 Framework 本地工具契约，市场 rc.2、插件归属、scopes、原 operation 恢复和审核／上架语义保持不变。具体运行证据与验收边界以 Framework GUI 发布验证记录为准。

2026-11-17 Framework 接续：Market 已交付 Schema ID 准入修复（协议仍为 rc.2）；Framework 使用同一 Base 0.1.5 冻结 ZIP，在自己的发布页面完成真实上传、系统凭据绑定和送审，取得 Version、Artifact、Publication 回执。模拟浏览器丢失送审响应后恢复原任务与 operation，刷新页面及重启临时执行器后回执保持一致。修正了送审后刷新完整制品集合被草稿状态门禁阻挡的问题：资料读取允许查看审核中及已审核版本，stable/draft/rejected 检查仍在每次真正写入前执行。该包的驳回重提、批准待上架及上架须由市场维护者操作后继续验收，当前不宣称完成。见[本轮页面证据](../contracts/evidence/plugin-publisher-base-page-joint-validation-20261117.json)及[市场协作步骤](plugin-publisher-base-review-coordination-20261117.md)。

本轮职责澄清只更新 Framework 仓库的文档副本；Market 功能源码、日常服务及对应文档由市场维护者处理。市场侧按协作交接同步自己的两份对接文档，双方以交付回执确认同步结果。

## 1 当前交付与后续执行边界

首次文档对齐仅修改两仓库文档；随后按用户执行指令在 PowerXPlugin 当前分支实现 F02 本地工具及 F03 首个构建配置。市场功能代码、Core、数据库及日常服务未修改；用于受控验证的临时 CLI 服务和独立生成插件已清理。未提交、推送或切换分支。

当前可用命令是 `px-plugin publisher plan create/list/get`、`px-plugin publisher serve` 和 `publisher build run/get/list/verify/logs`，正式本地契约见[本地工具 v1](../contracts/plugin-publisher-local-v1.md)；已注册本地计划、构建、日志、复核和取消接口。F03 配置为 make-dist-pure-go-js/v1，接受 Linux/amd64、CGO_ENABLED=0 和纯 JS 前端；本机执行验证原生平台，macOS 页面可明确使用本地 Linux 容器。详见[构建契约](../contracts/plugin-publisher-build-v1.md)，真实 Linux/Nuxt 运行待验，不代表完整 M1。

本地构建、市场上传、送审、批准、上架、下载、宿主安装及启用分别记录证据。文档中的文件范围是后续实现清单，不表示已经修改或授予真实环境执行权限。

## 2 统一架构与调用顺序

```text
开发态 Nuxt 发布页面 ── 同源开发代理 ── Go CLI 回环执行器
                                           │
非交互 CLI ──────────────────────────────── Go 共享引擎
                                           ├─ 本地计划与任务记录
                                           ├─ make dist → 冻结 staging → ZIP → 校验
                                           ├─ 系统凭据存储中的市场令牌引用
                                           └─ /publisher 机器协议 → 市场现有业务服务
                                                                       │
                                                               数据库 + 对象存储
```

1. 用户打开受信插件源码工作区的开发页面，由 Nuxt 自动准备工具；执行器固定该工作区、监听回环地址，建立短期本地工具会话。
2. 页面通过同源开发代理取得脱敏配置及任务 DTO；执行器检查 Host、Origin 和工具会话。F02 必须补齐握手、失效、代理边界及端口配置，不能只依赖 localhost。
3. 创建计划时锁定 `plugin_id/version/channel/target/market_connection`，生成计划 UUID。构建、上传及送审分别记录稳定 UUID。
4. “仅构建”执行本地预检查。“构建并上传”先验证 discovery、发布凭据及插件归属，再执行构建。
5. 执行器用参数数组启动固定的 `make dist` 任务，记录 PID、子进程和阶段；页面不得传 shell、任意命令或工作目录。
6. 构建完成后从本次输出形成冻结 staging，校验身份和完整运行时，压缩一次，保存文件摘要及大小。
7. 上传前先持久化 operation UUID 和规范化请求摘要，再流式发送同一冻结文件。市场独立核对、登记并可靠保存操作结果后才返回成功。
8. 页面按有界任务查询显示结果；传输达到 100% 后仍是市场校验中。收到可靠回执并核对版本才显示已上传草稿。
9. M2 才开放送审；资料及完整 Artifact UUID 集合由市场事务绑定，批准和公开上架仍由市场人员完成。

生产 Nuxt 构建不注册开发发布页面、菜单和代理；插件业务 HTTP 服务不注册构建命令接口。安装包排除执行器、工具状态和市场凭据。A01 需要直接访问验证，不能只检查菜单。

## 3 共享引擎与命令迁移

引擎由 Go 实现，位于 CLI 可复用的内部模块；CLI 命令和回环执行器调用同一套计划、构建、打包、上传与恢复逻辑。Framework 前端提供开发态界面及结构化客户端，TypeScript 不再独立生成发布成功结果。

下表“目标行为”仍是后续构建/发布实现要求。新增的 publisher plan/serve/build 提供独立工具和首个隔离构建配置，不改变旧 package/publish 命令行为。

| 入口 | 当前行为 | 目标行为与迁移边界 |
| --- | --- | --- |
| `make dist` | 构建后端和前端，组织安装目录，写产物版本/target，检查 Go 目标 | 作为新引擎的构建基础，补隔离输出和完整运行时校验 |
| `make package` | 依赖 dist，再压缩整个版本目录为 ZIP | 保留本地构建用法；新引擎只压缩冻结 staging，不调用该入口重新构建 |
| `make release/package-release` | 另一套 target 目录及 ZIP 规则 | 不作为 M1 的统一引擎；不得宣称自动生成签名、扫描报告或市场回执 |
| Go `px-plugin package` | 独立 Builder，输出 payload、metadata 和 `package.tar.gz` | 后续迁移到统一引擎；不维护第二套新发布包布局，不静默转换旧产物 |
| Go `px-plugin dist` | experimental 提示，无实际打包 | 不作为已交付构建能力；正式命令由 F02 明确 |
| Go `px-plugin publish` | 向旧内部接口发送 JSON，本地包 URI 由 metadata 提供 | 与新 publisher 明确区分并迁移；新协议不可用时失败，不回退内部接口或旧凭据配置 |
| TypeScript `publish` | 本地生成 ID、固定上传地址和回执 | 不作为真实发布或签名证明；新入口使用 Go 引擎，不保留模拟成功路径 |
| TypeScript `publish create/deploy` 与 Framework Plan Handler | 旧 Plan/Deploy 样例，Framework Handler 使用进程内 map | 不承载新市场任务恢复、审核或客户部署；对应样例测试不等于真实持久化链路 |
| 市场 `px plugin ...` 与 `/internal/*` | 市场自身的内部开发及治理入口 | 保持职责独立，不接入插件发布页面，不充当发布令牌接口 |
| 市场 `/dev/*` | 现有浏览器登录、CSRF 和开发者控制台入口 | 保持现有入口；共用业务服务及审核权威，不因此给新工具添加旧 DTO 或鉴权回退 |

已交付命令、会话握手、DTO、退出码、请求预算和输出 schema 已列入 F02 本地契约；build 命令及本地契约已交付；上传及送审命令仍待各阶段交付。不得自动读取旧 `publishApi.apiKey` 或将 Host API Key 当作发布令牌。迁移说明提示用户显式绑定市场、选择冻结产物、重新取得适用凭据；不静默迁移秘密或猜测市场地址。

## 4 工作区、构建和状态记录

本地状态统一保存到 `.px-plugin/publisher/`，纳入生成插件忽略规则并从安装包排除。配置名称及细分目录由 F02 固定，记录契约如下：

| 记录 | 最低字段与约束 |
| --- | --- |
| 连接 | 连接 UUID、市场 API root、凭据引用及脱敏身份；切换 origin 不沿用凭据 |
| 计划 | 计划 UUID、插件 ID、SemVer、stable、目标集合、更新说明及连接 UUID |
| 构建 | 构建 UUID、计划 UUID、源码提交、dirty 标记、构建输入指纹、工具版本、阶段、日志引用 |
| 冻结产物 | 构建 UUID、相对路径、插件/版本/平台、实际字节数、SHA256、校验报告 |
| 写操作 | operation UUID、类型、身份及连接绑定、规范化业务请求指纹、状态、市场 UUID 回执及请求 ID |

所有跨对象引用使用 UUID；`plugin_id` 沿用现有正式插件标识契约，不能把显示名称当作身份。时间戳只记录时间，不能代替任务 UUID。

首期同一工作区构建串行化，同时为每个任务分配独立 staging。共享的 backend/bin、Nuxt .output 和打包路径必须纳入锁及输出隔离设计，不能只设置 `DIST_DIR`。不创建或切换用户 Git 分支，不自动提交、暂存或清理改动。构建输入指纹应覆盖实际使用的受控源码、未提交内容和工具配置；只记录 HEAD 与 dirty 布尔值不足以证明输入未变。

版本只写入产物 manifest 和对应运行元数据，源码 manifest 不改写。Linux 首期使用能构建完整 Linux 运行时的执行器；验证 Go plugin、migrate 与 Nuxt/Nitro 中原生 Node 依赖的目标。macOS 上 Go 跨编译成功不能使 Linux 完整包校验通过；当前开发页面新增本地 Linux 容器构建；容器引擎通过页面明确启动，不自动切换远程执行器。

打包只接受受控文件清单，包含有效的唯一 manifest、plugin.d、被引用的契约/资源、后端及前端运行时。排除凭据、`.env`、数据库、上传文件、源码仓库和本地工具状态。链接、路径、重复条目与可执行权限按主方案 A09 及宿主解包规则验证。签名文件占位和哈希文件不能当作有效签名。

## 5 授权、协议与恢复

市场发布令牌只由市场网页会话创建、列出、撤销，绑定真实用户、开发者、插件范围、scope 和有效期。执行器保存系统凭据引用；页面只收脱敏 DTO。CI secret 在 M4 接入。未实现系统凭据存储的环境明确报错，不自动改成明文文件。

机器端统一使用 `<market_api_root>/publisher` 和 `powerx.plugin-publisher/v1`，新包采用正式 `id` 和显式 `target`，不添加旧字段解析或共享令牌回退。正式路径、DTO、错误结构和共享 fixture 以 P01 rc.2 契约为准，M2 送审消费 P07 交付。

恢复顺序固定为：

1. 启动时原子加载本地记录；中断构建标为 interrupted，不推断成功。
2. 已发出但结果未知的上传或送审进入 reconciling，以有效授权查询原 operation。
3. 成功操作返回原业务回执，再读取市场当前版本或申请，避免拿旧回执误判当前可写状态。
4. 未找到操作仍不能证明旧请求不会到达；必要重试仅复用原 operation ID、同一冻结文件和同一业务内容。
5. 凭据更换后核对稳定市场身份、插件范围和操作权限；撤销凭据不能继续查询或写入。
6. 超过保留期或重试预算时保留任务，展示需要核对和手工恢复动作，不自动换 key、覆盖包或重新送审。

取消构建只终止执行器拥有的进程树。取消上传停止客户端传输并核对服务端结果，不隐含删除草稿或撤回送审。M1 不支持字节级断点续传；必要时从头重传同一文件。

市场对象写入、业务登记和操作状态需要租约/接管、唯一约束及可重试清理共同收敛。清理只删除该操作拥有、尚未引用的对象。上传完成前再次核对授权及归属；已完成事务不因之后撤销而回滚。

## 6 页面与交互实现要求

顶部固定显示当前插件、选中版本、目标市场、任务阶段和市场状态；默认版本管理。桌面左侧纵向 Tab、右侧当前内容，窄屏紧凑导航。版本管理、构建打包、上传与审核、市场连接四个分区互不堆叠，详情通过抽屉或弹层进入。

选中版本、表单、运行任务和错误随分区切换保留，刷新从执行器恢复。授权到期、维护、失败和结果待核对始终可发现。“上传并送审”在 M2 提供，默认关闭；M1 不呈现不可用的 beta 或送审能力。

所有可见文案、错误映射、空状态、提示及依赖文案的测试断言使用 locale。对象显示优先名称和业务标识；UUID 仅作内部值或明确诊断信息。两端按各自安装版本使用 Nuxt UI，不跨版本复制 props 或插槽。

主题使用 useColorMode 语义色。桌面、窄屏、浅深色、手动/系统主题不一致，以及默认/悬停/选中/聚焦/禁用和弹层均实际渲染；普通文字至少 4.5:1，大字至少 3:1。检查控制台、SSR 和 hydration，不能只以构建成功代替 A02。

## 7 两仓库拟变更范围

以下是实现任务的目标区域。当前 CLI 已实现本地工具、构建准入、系统凭据、市场连接、异步字节上传与恢复；Framework Admin Layer 页面及外层私有代理已接通同一引擎。F06 模板同步，F07 采用真实 tarball/模块代理快照独立消费。业务 HTTP 路由与市场功能源码未修改；详见[市场适配契约](../contracts/plugin-publisher-market-v1.md)。

| 仓库 | 目标区域 | 职责与限制 |
| --- | --- | --- |
| PowerXPlugin | `tools/cli/cmd/`、`tools/cli/internal/` | Go 命令、回环服务、共享引擎、状态、凭据引用和 HTTP 客户端；正式模块名由 F02 固定 |
| PowerXPlugin | `framework/frontend/nuxt/` | 开发态公共页面/客户端与 locale；生产不注册开发路由及代理 |
| PowerXPlugin | `skeleton/make-files/`、`skeleton/scripts/manifest/` | 统一构建契约、受控 staging、目标及打包校验 |
| PowerXPlugin | `skeleton/web-admin/nuxt/`、忽略配置 | 开发入口、代理与必要配置，不向业务后端添加构建接口 |
| PowerXPlugin | `scaffold/templates/`、`tools/cli/internal/templates/data/` | 同步实际生成能力、忽略规则及必要示例，独立生成插件消费 |
| PowerXPlugin | `docs/guides/publish/`、CLI README、打包规范及契约目录 | 现状、迁移、接口和验收证据 |
| PowerXPluginMarket | `backend/api/openapi/` | P01 正式 publisher 契约；现有浏览器 OpenAPI 继续约束已实现入口 |
| PowerXPluginMarket | `backend/internal/http/identity/` 或独立 publisher 路由组 | 独立机器认证与严格 DTO，不开放内部治理接口 |
| PowerXPluginMarket | `backend/internal/services/identity/`、现有版本/制品服务 | 复用归属和业务规则，逐包验证、操作恢复及集合快照 |
| PowerXPluginMarket | 现有模型、仓储和增量迁移目录 | 凭据、scope 关联、操作记录、租约及清理；新业务对象和引用使用 UUID |
| PowerXPluginMarket | `web-admin/app/` 及现有 locale | 本人发布令牌管理、现有审核页面所需集合信息，不复制 Framework 构建执行器 |

Core、其他业务插件和用户日常服务不在本轮变更范围。后续发现宿主安装契约需要改动时，先列明证据和必要范围，再单独安排。

## 8 阶段任务、依赖及验收

| 阶段 | Framework 任务 | 市场任务 | 依赖与退出证据 |
| --- | --- | --- | --- |
| M0 文档与契约 | F01 现状及迁移清单；F02 本地工具契约 | P01 正式 OpenAPI、错误、预算和 fixture | P01 rc.2 已交付，Framework 同版消费、共享 Go/Node 验证和隔离对接通过；正式能力开放和运行验收另列 |
| M1 单平台上传 | F02 执行器；F03 构建/冻结；F04 凭据/上传/恢复；F05 页面；F06 模板；F07 独立生成消费 | P02 凭据；P03 机器路由；P04 字节接收；P05 幂等恢复；P06 查询；P08 本阶段审计 | 冻结 M0 后，受控实现可并行；隔离市场源码 + MinIO 字节上传/恢复通过；原始 discovery 拦截通过。正式开放与原生 Linux/Nuxt 的联合 A01—A14 仍保留 |
| M2 送审 | F08 资料选项、完整集合及状态回显 | P07 快照贯通浏览器、送审、审核、查看和上架；P08 审计 | F08 源码、页面和受控契约验证已增加；P07 已交付且默认关闭。本轮已验证真实页面主链路；A15/A16 剩余失败分支、原生完整包和公开下载等仍待联合验收 |
| M3 多平台与渠道 | 批量目标、同源码来源、补传和渠道选择 | beta、排序、发现、promotion 契约及逐包规则 | 单独冻结渠道规则；A16/A19/A20，不覆盖既有包 |
| M4 CI | 非交互命令、secret、同引擎回执和独立包消费 | 同版机器接口及操作审计 | A21，记录 CI/CLI/Framework/生成插件版本和 SHA256；不自动触发用户 workflow |

F07 使用新的独立插件和隔离市场资源，登记维护人、源码状态、实际消费包版本和摘要，排除意外 replace、链接或工作区依赖。本地可追踪快照、正式包发布、真实上传、公开下载与 PowerX 安装启用分别报告。首个测试插件 ID、账号及证据位置在联调前登记，不能在文档中假设已有授权资源。

## 9 交付状态、剩余冻结清单与文档维护

| 项目 | 当前状态及剩余工作 | 负责方 |
| --- | --- | --- |
| 本地工具 | 本地契约及计划/会话服务已交付；构建任务/进程取消的首个配置已由 F03 受控验证；F04/F05 市场链路及页面已交付，F06 模板与 F07 独立快照消费通过；真实 Linux/Nuxt 运行待验 | F02/F03/F05 |
| 发布凭据 | rc.2 DTO/TTL/scope 已消费；F04 系统存储和同身份更新已实现；P02 网页已交付；本机 macOS 已完成真实 M2 绑定与同身份只读换发恢复；Linux Secret Service 及正式环境另验 | P01/P02 |
| 机器 schema | rc.2 原包 24 文件与摘要保持，Go/Node 共享验证通过，客户端执行严格 schema 与可信控制台校验 | P01 |
| 指纹及集合 | 上传和送审规范化算法、字段上限、集合排序、Artifact UUID 和冲突 fixture | P01/P07 |
| 预算 | 幂等保留期、整包及 multipart 上限、解压预算、请求/空闲超时、轮询和重试上限 | P01/F02 |
| 联调资源 | 独立插件 ID、维护人、隔离账号/数据库/存储、证据位置及清理归属 | F07 与市场维护人 |

七天终态保留期、分页 20/最大 100 和预算已由 P01 rc.2 定义并同版消费。验证与清理见[本轮证据](../contracts/evidence/plugin-publisher-m1-validation-20261110.json)。源码交付、隔离联合验证、正式包发布和生产联合验收分别报告；不因本地快照或测试包装器宣布市场已开放。

两仓库主方案和本设计保持相同版本及正文，允许相对链接因仓库位置不同而变化。市场侧是机器 API 正式 OpenAPI 的维护方，Framework 消费同版契约和 fixture；本地工具契约由 Framework 维护。协议语义发生变更时同步两份方案、指南及测试输入，不只改一端。

F08 细节见[送审适配契约](../contracts/plugin-publisher-submission-v1.md)与[受控证据](../contracts/evidence/plugin-publisher-f08-validation-20261111.json)。原始 rc.2 工件未修改；本端不绕过 publication-submit/idempotent-writes 或 publications scopes，不用旧浏览器路由兼容。P07 已明确驳回重提生成新的 Publication UUID，旧回执及其资料／集合保持不变。

市场 P08/M1 审计、P02 网页及默认关闭的 M1 能力公告门禁已交付（市场文档保持各阶段历史基线）；M1 新门禁仍待 Framework 客户端联合重放、目标运行时和真实系统凭据验收，M2/P07 默认关闭；本轮受控包的真实开发页面主链路已验证。配对方案 v0.9 不改写既有历史证据。

2026-11-11／v0.8 对齐：[市场 P07 交付](../../../../PowerXPluginMarket/docs/marketplace/plugin-publisher-p07-delivery.md)及[M2 联调清单](../../../../PowerXPluginMarket/docs/marketplace/plugin-publisher-m2-joint-acceptance.md)已核对。协议继续 rc.2，24 文件与 bundle 摘要未变。本端增加按任务持久化、分页读取的脱敏 HTTP 请求记录及 Publication UUID 技术详情；见[本轮验证](../contracts/evidence/plugin-publisher-p07-alignment-20261111.json)。本轮已通过同机实验脚本交接并完成受控包页面主链路；环境现已清理，下一轮读取新 handoff。默认关闭不代表未实现；此前 M1/F08 证据保留为历史记录。

2026-11-11／v0.9 联合进展：Framework 独立生成插件及实际 SDK tarball 的开发页面，已连接本轮真实 PostgreSQL/MinIO/Gin/Market 网页环境，完成送审、丢响应与执行器重启恢复、驳回重提、批准仍私有、显式上架、旧链接与只读换发恢复。输入为市场已上传的受控 ZIP，非 Framework 原生完整包。详见[联合记录](plugin-publisher-framework-m2-joint-record.md)和[新证据](../contracts/evidence/plugin-publisher-m2-page-joint-validation-20261111.json)。此前 v0.8 的待交接描述为历史时点，环境现已关闭；完整构建上传、A15/A16 剩余场景、公开下载及宿主安装继续保留。
