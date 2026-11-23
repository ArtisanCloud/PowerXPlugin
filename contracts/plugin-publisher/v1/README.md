# PowerX 插件机器发布协议 v1

契约版本：`1.0.0-rc.2`。协议：`powerx.plugin-publisher/v1`。状态：市场 P01 工件供 Framework Review；本仓库已实现凭据、机器读取和操作恢复的后端基础，迁移及部署未执行，真实上传未接入。Framework 已有 F02/F03 受控基础；两端 M0 还须完成 rc.2 消费 Review 和联调资源登记。安装本包不代表服务具备发布能力，实际能力以 discovery 为准。

本目录可整体交给 Framework，不依赖市场后端、数据库或其余工作区。正式接口源在市场仓库 `backend/api/openapi/plugin-publisher.openapi.yaml`，本包 [openapi.yaml](openapi.yaml) 是仅替换 schema 相对路径的便携副本。字段唯一源为 `protocol.schema.json`，参数唯一源为 `constants.json`，机器错误映射为 `errors.json`。这些文件共同约束实现；JSON Schema 无法表达的字节、状态、归属及流式校验要求以下文为准。

## 1 工件与使用方法

| 工件 | 用途 |
| --- | --- |
| `protocol.schema.json` | 严格 DTO、请求和响应、操作状态与 M2 保留定义 |
| `constants.json`、`errors.json` | 本版固定参数和错误状态、可重试性映射 |
| `plugin-id.schema.json`、`plugin-id.cases.json`、`plugin-id-source.json` | 现有 Framework 正式插件 ID 约束的只读快照及来源摘要，保留第三方和单段 ID，不追加总长度限制 |
| `examples.json` | 明确标记 synthetic 的请求回执示例，不包含真实账号、凭据或可安装产物 |
| `schema-cases.json` | 正反例，含权限字段注入、UTF-8 边界、重复键和非法 Unicode |
| `encoding-cases.json` | 原始字节、BOM、UTF-8 及分片视图边界，Go/Node 共同消费 |
| `canonical-cases.json` | 上传与送审的精确 canonical 字符串和预期 SHA256 |
| `recovery-cases.json`、`link-cases.json` | 授权更换、丢响应、重试预算及可信市场链接的共享预期 |
| `conformance.go`、`conformance.mjs` | Go/Node 一致性参考，只校验输入和给出恢复决策，不实现实际上传或授权 |
| `bundle.sha256.json` | 可交接工件的逐文件摘要和整体清单摘要 |

在本目录执行：

```sh
npm ci --ignore-scripts --no-audit --no-fund
npm run check:openapi
npm test
GOWORK=off go test -race ./... -count=1
GOWORK=off go vet ./...
```

在市场仓库根目录执行 `make publisher-contract-check` 可校验清单及运行两种语言的一致性检查，并强制核对后端 OpenAPI authority。独立接收方的 `npm run check:openapi` 使用 portable-only 模式；不要求存在市场后端。依赖须先按上面的 npm 命令安装；目标不会连接市场或启动服务。

Framework 接收时先保留原文件和清单摘要，再运行上述检查。可用 `python3 scripts/check-plugin-publisher-contracts.py --peer <Framework仓库根目录>` 只读比较双方 `contracts/plugin-publisher/v1/`。市场不代写 Framework 文件。迁移到 Framework 自身模块时仍需消费这些共同 fixture，且不能把参考测试结果记为真实上传或产品授权通过。

## 2 首版能力与身份边界

M1 仅声明 `package-upload`、`idempotent-writes`，仅 ZIP、stable、linux/amd64；M2 字段及路由单独标记 `x-stage: M2`，不得先宣告送审可用。Discovery 只在迁移和对应 handler 就绪后开放能力，不能因为加载了本目录就宣告支持。

所有 `/publisher/*` 私有接口只接受发布专用 Bearer 令牌；登录会话 Bearer、Cookie、Host API Key、实例凭据和旧共享令牌均不接受。每次请求重新核对用户、开发者成员、激活状态、插件范围及 scope，写入前再次核对。范围外插件、版本和其他身份操作返回 404；缺少操作 scope 返回 403；令牌失效统一 401。

用户和开发者 UUID 来自市场解析身份，不能由请求体指定。市场自身 `px` CLI 和 `/internal/*` 仍是独立的治理入口。现有 `/dev/*` 上传、审核 DTO 及 CSRF 行为保持原状；新增 `/dev/publish-tokens` 管理接口使用真实市场 Cookie 会话和写请求 CSRF、Origin 校验，不能由发布令牌管理令牌。

## 3 发布令牌规则

- 默认有效期 604800 秒即 7 天，最短 300 秒，最长 2592000 秒即 30 天，从创建时的服务端时间计算。
- 一个市场用户与开发者绑定最多 20 个有效令牌；单令牌最多绑定 50 个明确且归属本人的已登记插件 ID，不提供通配符。
- 明文为 `pxpub_` 加 32 个密码学随机字节的无 padding base64url，共 49 字符。创建仅返回一次；服务器保存校验摘要，不提供密钥找回、refresh token 或自动续期。
- `plugins:read` 为依赖 scope；`packages:write` 需要它；M2 的 `publications:submit` 还依赖 `publications:read`。不得隐式扩大 scope。未启用的 M2 scope 在创建时返回 `publisher_scope_unsupported`。
- 明文只能由本地执行器保存到系统凭据存储，插件浏览器只取得脱敏 DTO。M1 系统凭据存储不可用时报错，不提供文件明文回退。此规则消除主方案第 8.1 节留下的可选回退措辞，与实现设计一致。
- 创建响应丢失不自动重放明文；在市场令牌列表核对并撤销未绑定的令牌，再显式创建。撤销重复调用保持首次撤销时间与原因。
- 新凭据仅在市场用户、开发者、插件范围和对应权限均相符时恢复旧操作；旧凭据撤销后不能继续查询。撤销不回滚先前已完成的上传，不卸载插件。

## 4 请求与响应

所有已认证响应及令牌管理响应使用 `Cache-Control: no-store`，返回服务端可关联的 `X-Request-ID` UUID。成功 envelope 为 `code=0/message/data`，错误为 `code/error/message/retryable`，`code` 等于 HTTP 状态，详见 schema 和错误表。错误不返回凭据、存储 key、原始请求体或本机路径，字段错误仅包含字段路径及原因码。

JSON 大小先按收到的原始字节计数，再严格 UTF-8 解码；拒绝开头的 UTF-8 BOM，不移除后重新计数。JSON 字符串内的合法 U+FEFF 保留。所有 JSON 拒绝重复键、无效 UTF-8、孤立 surrogate、文本或键中的 NUL、尾随内容、未知字段及超过 64 层的嵌套。数字只使用十进制整数词法，不接受小数或指数，值不得超过 JavaScript 安全整数范围。UTC 时间使用 `Z`，可无小数或恰好三位毫秒；服务端 DTO 必须显式格式化，不能直接泄露数据库模型的任意纳秒精度。新生成操作 ID 使用小写 UUIDv4，已有资源引用使用 schema 规定的小写 UUID，不用时间戳或摘要代替对象 ID。

上传 multipart 恰好包含 `file/version/channel/os/arch/sha256/size/release_notes` 八个字段，字段顺序不限，每项一次，`release_notes` 可空。拒绝未知字段、重复字段、额外文件部分和超预算 part header。`size` 使用无前导零的十进制正整数字符串；框架转换后的 DTO 使用 JSON integer。客户端发送真实 ZIP 字节，不发送本地文件 URI。201 表示原始登记成功，200 表示相同操作和内容的原回执，202 只表示仍处理中；202 及处理中查询附带 `Retry-After` 秒数。

新工具的 manifest 使用正式 `id/version/target.os/target.arch`，不回退旧 `plugin_id` 字段。正式 ID 的命名约束直接消费随包快照；市场既有插件登记或字段容量限制仍以“目标插件是否已登记且归属本人”核对，不能将新工具变成自动注册或改名入口。

正式版本不含 `v` 前缀，stable 不接受预发布版本；版本、声明目标、包内目标和实际运行时一致。当前构建单一 linux/amd64 不表示其他平台运行时已验证。前端原生模块也属于目标校验，不能仅检查 Go 跨编译结果。

## 5 包与请求预算

以下均为 rc.2 的实现参数，Discovery 返回同版值；不是对当前运行市场的保证。

| 项目 | 固定值 |
| --- | --- |
| 压缩文件 | 至多 2147483648 字节，即 2 GiB |
| multipart 额外预算 | 至多 1048576 字节，总请求至多 2148532224 字节 |
| multipart part | 最多 8 个，每个 part header 最多 8192 字节 |
| JSON 请求及响应 | 请求最多 65536 字节，完整响应最多 8388608 字节；既有浏览器接口限制不随之改变 |
| manifest | 最多 262144 字节，唯一有效 manifest，拒绝重复 YAML 键、alias 和自定义 tag |
| 解压资源 | 全包至多 8 GiB、50000 条目、路径最多 1024 UTF-8 字节、总解压/压缩大小比不超过 200 |
| 文本 | 更新说明、描述、安全及隐私说明分别最多 10000 UTF-8 字节；三个说明 normalize 后至少 10 个 Unicode scalar |
| URL | 最多 2000 UTF-8 字节；最低 Core 版本至多 20 字符且来自市场选项 |
| 其他文本 | 名称最多 100 scalar 且 200 UTF-8 字节，简介 2000 字节；错误和撤销原因 1000 字节，审核及上架原因 4000 字节 |
| 分页 | page 从 1 开始且最大 1000000，size 默认 20、最大 100；列表返回 items/total/page/size |

包布局为根 `plugin.yaml` 或单一 `<version>/plugin.yaml` 包裹目录，包裹目录之外不能夹带文件，不接旧 CLI 的另一套 payload 安装布局。目录条目可带结尾 `/`。拒绝绝对路径、`.`、`..`、反斜线、NUL、冒号、百分号、重复路径、目录/文件碰撞及大小写折叠碰撞。新 ZIP 不允许链接、设备文件和 setuid/setgid 等特殊权限；Framework 需要链接的受控依赖应在冻结 staging 中安全展开为普通文件，只保留常规文件/目录和必要的执行位。

先校验目录声明的尺寸与路径，再有界读取每一项以核对实际解压尺寸、CRC 和总预算，不能只信 ZIP header。不得在市场执行任何包内程序。服务端独立计算收到的压缩字节摘要和大小；POST 成功重放也须核对本次完整流，客户端应先 GET 原 operation 避免不必要的整包传输。

支持地址等普通资料 URL 不作为市场抓取指令。URL 主机使用 ASCII/IDNA 形式，不包含 userinfo；IP 使用规范文本。控制台链接还必须由执行器校验配置批准的 console base，同 origin、子路径边界、无 fragment 或路径编码歧义；HTTP 仅允许显式回环地址。Discovery 返回的跨 origin 地址不能自行授权信任；不向重定向的新 origin 转发令牌。

## 6 请求指纹

指纹为 `SHA256(UTF8("powerx.plugin-publisher/v1/request\n") || JCS(payload))`，输出 64 位小写 hex。JCS 对象属性按 RFC 8785 排序；本协议属性名均固定 ASCII，整数处于安全范围，字符串保留 Unicode scalar 原文，不做 NFC 或大小写转换。

上传 payload 精确包含：`protocol/type/plugin_id/version/channel/os/arch/sha256/size/release_notes`，`type=package-upload`。`release_notes` 原样参与，换行不归一化。

送审 payload 精确包含：`protocol/type/plugin_id/version_id/metadata/expected_artifacts`，`type=publication-submit`。metadata 中每个文本只剥离两端 ASCII SPACE、TAB、CR、LF；不剥离 NBSP，不改内部换行或大小写；省略的 icon_url 归一为 `""`。normalize 后校验字段和字节上限。

完整制品集合每项包含 `artifact_id/os/arch/sha256/size`，按 `os`、`arch`、`artifact_id` 的 ASCII 升序排序；重复 UUID 或平台拒绝。送审时集合必须与市场当前全部 Artifact 精确一致，并持久化快照，由送审、审核、包查看和上架共同复核。M2 的实现及历史无快照申请阻断规则由 P07 交付。

请求 ID、operation ID、令牌 ID、multipart boundary、文件名、本地路径和临时上传进度均不进入 payload。身份由操作 namespace 绑定，不能用这些排除项绕过跨身份限制。客户端声明的 sha256 只有与实际文件流相符时才允许登记。

## 7 操作身份与存储恢复

唯一操作键为 `(market_user_id, developer_id, operation_id)`，不包含令牌 ID。首次接收的插件、类型和请求指纹随后锁定；同键更换插件、操作类型或规范化内容均为 409。只有同一稳定身份及仍有资源、操作权限的新令牌可恢复。

服务端可先将完整请求流暂存并校验，再认领 operation；因此上传中查询 404 是合法的，不代表原请求不会到达。操作认领后原子保存身份、指纹及状态；同一键的并发认领只有一个有效写入者。processing 不返回结果，succeeded 绑定可靠原业务回执，failed 返回明确错误与可重试性。

对象存储与数据库之间没有分布式事务。租约为 60 秒，每 15 秒续约；超期接管使用递增 fencing，旧持有者不能提交业务记录、改写结果或删除新持有者对象。业务登记和成功 operation 在同一数据库事务提交，制品 key 唯一且不可覆盖。仅将属于本操作、未被任何登记引用的对象进入可重试清理，不能按 plugin 前缀删除。

终态至少保留最新终态时间后的 7 天；processing 及未收敛清理不按此 TTL 删除。可恢复 failed 重试仍使用原键、原指纹，不能并行起第二次业务登记；确定性错误保持原失败。到期后查询可返回 404，不承诺额外 tombstone；客户端先核对版本和摘要，不自动创建新键覆盖。

原操作成功回执不保证版本仍为 draft：恢复后还需读取当前版本或申请。凭据撤销在 commit-time 授权检查前生效时阻断登记；已完成事务不被之后撤销回滚。取消上传停止客户端传输，随后核对操作，不隐含删除市场草稿。首期不提供分片或字节偏移续传。

## 8 超时与客户端恢复

| 项目 | 固定值 |
| --- | --- |
| 连接 / 普通请求响应头 | 10 / 30 秒 |
| 普通读取 / JSON 写入 | 总时限 30 / 60 秒 |
| 上传 | 接收流连续空闲 60 秒，服务端总时限 1800 秒，客户端总时限 1860 秒，上传响应头等待上限 1800 秒 |
| 单次恢复 | 最多 3 次完整写请求，包含第一次；整个自动恢复窗口至多 7200 秒 |
| 重试等待 | full jitter，基数 1 秒，上限 30 秒；等待至少为服务端 Retry-After，不截短其要求 |
| Retry-After | delta-seconds，1 至 300；超过剩余预算或契约上限时停止自动重试，保留可恢复任务 |
| operation 轮询 | 间隔至少 2 秒，每轮主动恢复最多 120 秒；达到上限保留 pending，由用户后续恢复 |

普通请求的 30 秒 response-header 时限不套用到上传。上传响应头等待上限为 1800 秒，从文件请求体发送完成开始计，仍受客户端 1860 秒总时限约束，允许市场在收齐后校验完整 ZIP。代理和 HTTP 服务也要实现相容的总时限及空闲限制，不能只改客户端。上传总时限包含暂存、验证和提交，不另起无上限校验任务。

处理中分支同样受 7200 秒总窗口、120 秒轮询窗口和 Retry-After 上限约束；等待至少取 Retry-After 与 2 秒轮询间隔的较大值。等待达到剩余总窗口时转手工核对，达到剩余轮询窗口时保留 pending，不再发起越界请求。

网络异常、代理 502/504、响应解析错误或没有合法 envelope 时，视为结果未知，先查询 operation。401 要重新绑定授权，403/404 按资源范围核对，不尝试他人身份；相同键内容冲突不自动换键。`recovery-cases.json` 给出确定的客户端动作及优先顺序，参考函数不会实际重试。

收到 succeeded 先使用原回执，再读取当前业务状态；即使新写能力关闭仍允许授权恢复查询。operation 不存在且保留期内需要重传时，保持同一键、同一冻结文件和内容。恢复超预算、保留期不明或凭据不能恢复时，保留证据并提供手工核对，不将结果不明渲染为成功。

## 9 交接及版本维护

市场 P01 输出与 Framework F02/F03 分别 Review；rc.2 参数不能被称为双方 M0 已冻结，也不能称为现网已实施。Framework 需要反馈系统凭据存储、Linux 执行器、上传预算和本地任务恢复是否相容，并在自己的仓库消费同版文件及 fixture。

修改任何 wire 字段、归一化、错误可重试性或预算时更新契约版本、样例、参考实现及清单，通知另一端同步。市场后端的 `internal/publisher/wire` 是本包 Go 参考及必要 JSON 的生成快照，正常检查逐字节拒绝漂移；它避免后端构建依赖仓库外的 replace 路径。修改公共源后显式执行 `python3 scripts/check-plugin-publisher-contracts.py --sync-backend --write-manifest`；正常检查绝不重写清单或其他仓库。

本包没有真实令牌、服务地址、插件资源或 ZIP。fixture 中的回执和 `pxpub_A...` 都是固定合成数据，仅用于测试结构。必须分别验收真实鉴权、上传、审核、下载和宿主安装，不能以 schema/参考函数通过替代运行证据。
