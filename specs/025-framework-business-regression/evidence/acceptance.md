# 本地实施与分层验收

## 结果页分区与可核对证据（2026-10-17）

本次为用户确认的结果页改造；随后明确授权最小全局窄屏适配。使用当前分支，未提交、推送或发布，未操作用户日常服务。原任务仍为 56/57，T045 的真实 Core 授权验收继续待专用环境。

- 结果顶部保留结论、用例数、必要校验通过数、耗时、清理数量及主要操作；内部桌面左侧纵向分区，窄屏横向滚动分区，默认只显示概览。批量结果只查看当前用例，切换分区、窄屏菜单和同批次 SSE 更新保留选择；失败定位、清理失败及断线提示不依赖当前分区。
- 插件 `regressionPresentation.ts` 显式声明各断言字段、枚举与 locale，Framework 通用渲染通过／失败项的预期与实际。缺少映射、未知枚举、缺失值明确提示，原始 JSON 只在技术证据／快照入口查看。
- 对最新真实 CLI 报告的全部 23 项 controlled 断言执行显示映射核对，所有预期／实际字段均已记录、匹配，所有动态 locale key 存在；`artifacts/regression/result-presentation-coverage.json` 保存逐项字段数。
- 资源登记保存脱敏名称、角色和显式同场景来源，禁止未知来源／自指；清理成功必须提供目标一致的正式读回证据，并记录时间。Template 删除后正式读回 `exists=false`，Metadata 恢复清理读回原集合。后者仍只在本地合同测试验收，未执行真实 Core。
- 实际浏览器暴露公共组件响应式样式未进入 Tailwind 扫描，已提供 Framework CSS source 入口并由 Skeleton／双模板主样式导入。窄屏全局侧栏改为菜单抽屉，顶栏收紧，内容区可缩小；菜单关闭返回当前任务。桌面导航保留，未改权限或宿主桥协议。

| 验证 | 当前结果与证据 |
| --- | --- |
| Framework Go | `GOMAXPROCS=2 GOWORK=off go test -race -p=1 ./runtime/regression/...` 通过，含来源约束、清理证据拒绝与名称脱敏；`artifacts/regression/result-framework-race-final.log` |
| 插件 Go | Service、Scenario、HTTP、完整 integration `-race -p=1 -count=1` 通过，8 条业务场景正序／倒序、独立运行与污染故障清理继续验收；`result-business-race.log` |
| 前端 | 21 个文件、197 项 unit 通过；最新 Nuxt production build 通过；`result-unit.log`、`result-build.log` |
| Chromium | 8 项通过；1600×1000 与 390×844，实际检查导航、默认内容、选择保留、失败定位、抽屉边界及浅／深色；缺映射和清理告警另用明确 UI 合同 fixture，未伪造业务通过；`result-browser.log` |
| 正式业务与 CLI | 7 个实际批次、21 次场景执行，含正常／故障／取消；可见残留 0。CLI 8 用例、23 必要校验、21 资源，均有名称和清理时间／读回证据，5 条显式来源关联；`result-acceptance-summary.json`、`cli-business-report.json`、`browser-business-evidence.json` |
| Client／生成物 | Client 4 项通过；两个独立生成插件后端验收通过。双模板同步、OpenAPI 验证与 `git diff --check` 通过；Admin `npm pack --dry-run` 确认展示契约、组件与 CSS source 文件在发布 allowlist 内。未发布模块、未重跑两生成插件的完整前端安装／浏览器链路；`result-client.log`、`result-generated.log`、`result-template-check.log`、`result-openapi.log`、`result-admin-pack.json` |

截图已实际查看，保存在忽略的产物目录：

- [桌面概览](../../../artifacts/regression/result-overview-desktop.png)、[业务校验](../../../artifacts/regression/result-checks-desktop.png)、[名称、来源与清理](../../../artifacts/regression/result-resources-desktop.png)。
- [窄屏业务校验](../../../artifacts/regression/result-comparison-narrow-dark.png)、[窄屏完整对比抽屉](../../../artifacts/regression/result-drawer-narrow-dark.png)。

首轮 race 的 20ms／1s fixture 在高负载下误超时，复查后仅调整测试 fixture 时限（普通执行 10s、清理 5s、刻意超时 2s），产品时限不变。首轮布局、既有全局窄屏挤压和抽屉入场动画的测试迭代均保留在 `result-attempts.md` 及对应失败日志；最终通过结果见表。

最终自有 Go fixture 命令 `go test ./internal/transport/http/admin/regression -run '^TestRegressionBrowserFixture$' -count=1 -timeout 5m`，父／子 PID 94152／94258、端口 52362；Nuxt 命令 `node .output/server/index.mjs`，PID 94263、端口 52363。脚本 finally 关闭二者，端口均 released=true，backend_exit=0。失败轮次也正常释放临时端口。

日常 8078 的 PID 74746 为 2026-10-15 启动的旧后端，本次未重启；3131 日常服务也未停止。用户需按原方式重新加载后端并新跑批次，才能在日常实例得到新增资源证据。旧历史未记录字段显示缺失提示，不补造名称或证据。测试身份为明确注入的本地身份，不能替代真实安装态／Core 授权验收。

---

日期：2026-10-13。分支：`025-framework-business-regression`。用户已确认暂时没有专用 Core 环境，本次完成本地实现与验收；T045 保留待环境就绪。没有提交、推送、发布或触发 CI。

## 结果与证据边界

| 层次 | 实际执行 | 结果 |
| --- | --- | --- |
| Framework 源码 | `GOWORK=off go test -race ./runtime/regression/...` | 注册、必要断言／证据、生命周期、报告、GORM 原子事件与隔离通过 |
| 插件源码／业务 | config、HTTP／RBAC、迁移、Profile、Metadata 合同、Template integration | 通过；Metadata 合同使用显式测试 adapter，不是 Core 授权证据 |
| SQLite 正式 Service | 连续三次模板生命周期，再注入业务存储读回故障 | 3 passed、1 failed；四次 cleanup=succeeded，无可见残留模板 |
| PostgreSQL | 临时独立实例、真实三表迁移、实际 RegressionRepository、非超级用户 RLS | 幂等迁移、读写租户隔离、RLS FORCE、回滚通过 |
| Framework Client | Node 原生测试 | 4 项通过：Meta／身份、revision、SSE 中断、必要断言列表 |
| Nuxt | 完整现有 unit suite 与 production build | 20 个文件、189 项测试通过；构建通过 |
| 浏览器与真实业务 API | Chromium 连续三次执行、故障定位、协作取消、刷新恢复历史 | 通过，实际 Go API／SSE／业务 SQLite；身份注入明确只在 test fixture |
| CLI 对实际 API | 相同场景 `--wait`，标准 JSON report | 退出 0，verdict=passed、cleanup=succeeded |
| 生成 consumer | alpha、beta 两个新生成插件，`GOWORK=off` 显式 local Framework | 后端、各自 Nuxt 构建、正式 Service、各自浏览器及 CLI 闭环通过 |
| OpenAPI／模板 | Swagger schema 验证、`.specify` prerequisites、`sync:templates --check` | 通过 |
| 真 Core / 安装态 | 未执行 | 需要专用资源与实际 grant/revocation／宿主会话环境 |
| 正式发布模块 consumer | 未执行 | 本次未授权发布，本地替换不能证明发布可用 |

## 业务事实与报告

运行输出保存在仓库忽略目录 `artifacts/regression/`，不进入模板或发布包：

- `controlled-template-reports.json`：原生正式 Service 三次通过和一次失败，失败仍完成清理。
- `browser-business-evidence.json`：六个批次，含页面三次通过、业务失败、协作取消和 CLI 通过；`active_templates=0`。
- `cli-business-report.json`：schema_version=v1，与页面使用相同导出器和必要断言判定。
- `local-validation.log`、`nuxt-build.log`、`postgres-validation.log`、`browser-validation.log`：实际命令结果。

模板业务沿用正式软删除，残留检查指 `deleted_at IS NULL` 的可见业务资源为 0；不宣称清空历史软删除行。浏览器测试不是生产 JWT／安装态鉴权验收；HTTP 中另行覆盖可信身份缺失、权限不足和跨租户拒绝。

## 异常路径与恢复

- 必要断言漏报／重复、证据缺失、预期／实际不符均不能通过。
- panic、部分准备失败、超时和协作取消后仍尝试独立时限清理。
- 清理重试验证原绑定摘要与场景版本，成功不会修改原始 verdict。
- Snapshot／Event 同事务；追加事件失败会回滚快照。Store 出错明确结束，不能继续展示成功。
- SQLite Store 事务串行，修复实际浏览器暴露的并发快照读写导致 SSE 中断。
- PostgreSQL 回归 Repository 使用事务级参数化 `set_config`；未改动其他业务 Repository。
- 首次可信租户请求恢复本租户旧运行／清理状态；不会跨租户扫描或自动重执业务。
- 断线恢复读快照并按事件 UUID 补读；无自动轮询。

## 临时服务与释放

所有服务由 `regression-postgres.py`／`regression-browser.py` 创建并在 finally 清理。运行日志记录完整启动命令、父进程／服务 PID 和端口。最近一次本地验证：

| 服务 | PID／端口 | 释放结果 |
| --- | --- | --- |
| PostgreSQL | 编排 40669、服务 40797；端口 54521 | pg_ctl fast stop 成功，端口 released=true |
| Go 浏览器 fixture | 编排与测试进程见 browser-validation.log；端口 54987 | 后端正常退出 0，端口 released=true |
| Nuxt production preview | PID 见 browser-validation.log；端口 54988 | 进程组 SIGTERM 后退出，端口 released=true |

失败迭代同样执行清理并验证端口，不操作用户启动的日常服务。

## 复现

仓库根目录执行：

```bash
bash scripts/testing/regression.sh
python3 scripts/testing/regression-postgres.py
python3 scripts/testing/regression-browser.py
```

先安装 Skeleton Nuxt 依赖、构建 `.output`，准备 Playwright Chromium 和 PostgreSQL 工具。也可设置 `REGRESSION_POSTGRES=1 REGRESSION_BROWSER=1` 使用总脚本。CI 配置已增加受控验证与报告归档步骤，本次没有触发远端 CI。

## 两个生成插件的补充验收

alpha、beta 独立构建 Nuxt 并连接各自生成后端，页面与 CLI 验收均通过。事件 source_plugin 分别为 `com.powerx.regression.alpha`、`com.powerx.regression.beta`，每个 consumer 六个批次、active_templates=0。

- alpha：后端 51495、UI 51496，CLI exit=0，端口均 released=true。
- beta：后端 51963、UI 51964，CLI exit=0，端口均 released=true。

前端使用本地发布 allowlist 打包的 tgz；npm peer resolver 的环境问题与共享工具链边界见 [adoption.md](adoption.md)。本地验收未升级或发布正式 Framework 版本。

## 本地交付补强

用户再次确认无专用 Core 环境后，继续补齐可重复交付：

- `regression-generated.py` 自动完成两个生成插件的独立依赖安装、构建、页面和 CLI 验收，不再依赖手工拷贝或共享 Node 工具链；实际结果 passed。
- 最新 alpha 端口 54435／54436、beta 54925／54926 均 released=true，CLI exit=0，两个临时工程自动清理。
- Pinia 直接依赖与双模板已同步；`artifacts/regression/` 已由 Git 忽略，避免运行报告／生成工程进入版本控制。
- CLI 首次连接／最终快照读取失败保留 run_uuid；拒绝无效事件发生时间、跨租户运行快照和零 revision；对应 race 测试通过。
- 新增三项只读 live 前置检查测试，缺配置时不发请求，目录就绪也不声称 Core grant 或业务已通过。
- CI business-regression job 已配置 Chromium 安装和 Skeleton 浏览器闭环；本次仍未触发远端 CI。

总本地脚本、189 项前端测试、客户端／Go 验证通过；T045 继续待专用环境，见 [live-integration.md](live-integration.md)。

## 管理页面与规范复核（2026-10-14）

- 补齐目录中的场景版本、声明步骤／阶段／验证边界、执行与清理时限，以及当前配置在已加载历史中的最近批次入口。最近提示是整批结论，具体场景结论以详情为准。
- 详情展示测试配置、插件／源码版本快照、批次／场景／步骤耗时、实际依赖、必要断言标记、缺失 key 和测试资源技术详情。缺失版本显示未记录，缺完整时间显示尚无完整计时；没有增加轮询或当前值覆盖历史的兜底。
- 新增[管理页面指南](../../../docs/guides/features/025-framework-business-regression/usecase-workbench.md)，包含目录、执行、历史、恢复、API、命令、字段口径与排障；总览与业务计划均已回链。
- CLI 按规范支持重复 `--case`，请求保留全部 UUID；非契约 CSV 输入在 HTTP 前拒绝。命令级请求验证与 Client race 测试通过。
- SDK／quickstart 统一退出码：等待执行通过 0，参数／鉴权／HTTP／文件错误 1，业务非通过 2，SSE 中断 3；不等待时退出 0 仅表示提交被接受。live 指南口径统一为绑定既有测试标签并恢复原标签集合。
- 189 项前端测试、Nuxt build 与双模板同步检查通过。更新后的页面测试验证声明步骤、最近结果、完整耗时、版本快照、实际依赖、必要断言以及刷新后的历史。
- Skeleton、alpha、beta 分别完成独立 Chromium 和 CLI 验收；每个实例六批次，active_templates=0，CLI exit=0。alpha／beta 仍使用未发布的本地 Framework 归档，各自独立安装和构建。

| 实例 | 后端／UI 端口 | 清理结果 |
| --- | --- | --- |
| Skeleton | 62206／62208 | released=true，backend_exit=0 |
| alpha | 63980／63981 | released=true，backend_exit=0 |
| beta | 65227／65228 | released=true，backend_exit=0 |

启动命令与父／子 PID 分别保存在 `workbench-browser-validation.log`、`consumer-alpha-browser.log`、`consumer-beta-browser.log`；构建日志与业务报告位于 `artifacts/regression/`。任务仍为 56/57，T045 等待用户已确认缺失的专用环境，本轮没有进行真实 Core grant 或业务验收。

## 当前开发实例启用与错误修正（2026-10-15）

- 用户页面的四项目录请求均返回无标准 envelope 的 HTTP 404，前端原先误报为输入／身份错误。现已按目录请求状态显示 `REGRESSION_API_UNAVAILABLE`，保留标准业务原因，其他请求失败明确显示 `REGRESSION_REQUEST_FAILED`。补齐标题与正文主题 token。
- 本机插件开发库原有 Template 表，但缺少三张回归表。新增 `database migrate-regression` 入口并完成仅回归三表、索引及 RLS 的迁移；不调用全量模型迁移或 IAM seed。运行配置已准备单一 controlled profile，前端开关已开启。
- 配置预检发现 PostgreSQL 启动时 `HasTable` 对已加 schema 引号的字符串误判缺表。已改为参数化 `to_regclass` 检查，保留 SQLite 检查。真实 PostgreSQL 测试验证带 schema 的服务启动、回滚后缺表拒绝，以及原有 RLS／幂等；迁移测试额外确认其他表及数据保留。
- 当前配置的 Service 构造预检通过：一个 controlled profile、两个已注册定义，其中 live 定义不匹配受控配置；预检未执行业务。不能据此宣称旧运行进程已加载配置。
- 191 项前端测试、Nuxt build、迁移／命令编译、双模板同步通过。Chromium 两条路径通过：目录 404 显示配置指引且不能执行；正常模板回归仍完成页面／CLI 闭环。
- 临时 Go／UI 端口 54658／54659 正常释放，CLI exit=0；临时 PostgreSQL 端口 56734 正常释放。日志为 `runtime-error-browser-validation.log`、`runtime-postgres-validation.log`，位于被忽略的回归产物目录。
- 用户正在运行的 8078 后端仍返回 404；需重新加载配置后再核对真实会话下的目录及执行。遵守 AGENTS.md，不擅自重启用户进程。本轮没有执行真实 Core 授权验收，T045 状态不变。

## 工作台操作清晰度改进（2026-10-15）

- 用户后续截图已能加载受控配置和模板目录；本轮针对无法辨认用例数量和运行入口的反馈调整公共 Framework 工作台。当前 controlled 配置明确显示一个模板生命周期用例，创建、修改、读回单独显示为三个内部步骤。
- 首屏显示可运行／已选数量及三步指引。卡片可直接“运行此用例”，使用已准备的默认数据；顶部支持带数量的全部、筛选结果或选中执行。配置、输入 JSON、版本和边界默认折叠；切换配置清空筛选与选择，筛选后移除隐藏选择。
- 运行后滚动至结果，显示各类结论数量、步骤状态和清理；失败或阻塞校验自动展开预期／实际。原有版本快照、依赖证据、报告和恢复功能仍可在对应详情查看。
- 191 项前端测试、Nuxt production build、双模板同步及中英 locale key 对齐检查通过。三条 Chromium 路径通过：目录不可用的错误指引；实际 TemplateService／SQLite／SSE 的直接与批量执行、失败、取消、历史；多用例计数、筛选与配置切换。最后一条只注入目录数据验证展示，未执行合成用例。
- 浅色与深色目录及运行结果截图已目视核对。页面三次正常回归、一次读回故障、一次取消，加 CLI 一次正常回归，共六个批次；CLI exit=0，active_templates=0。取消批次 phase=cancelled、verdict=blocked，不视为通过。
- 自有临时 Go 父／子 PID 9403／9589、端口 56792，Nuxt PID 9647、端口 56793；脚本退出 0、backend_exit=0，两端口 released=true。启动命令、PID、报告与截图位于被忽略的 `artifacts/regression/`，主要日志为 `usability-browser-validation.log`、`usability-build.log`、`usability-unit.log`。
- 当前用户 3131 开发服务的公共组件和中文 locale 均返回新版内容，无须重启用户服务。此项仅证明前端资源已加载；本地浏览器采用测试身份，不替代实际安装态身份或 Core 授权验收。T045 保持待专用环境。

## 工作台产品流程重做与验收（2026-10-15）

- 按用户确认的操作路径重组：默认“回归用例”表格显示数量、验证目的、最近批次结果与行内操作；“查看详情”抽屉解释目的、三个内部步骤、通过条件及数据影响；“运行”打开预填业务表单，点击“开始执行”才创建批次。批量入口仅在勾选后出现，独立“执行记录”负责历史及当次详情。
- 公共 Framework 通过 `RegressionCasePresentation` 接收插件的业务说明、字段与 locale 映射；模板页面提供名称、说明、初始内容、修改后内容。Valibot 校验必填、长度和 UUID；未填字段使用本地化业务提示，缺少展示映射明确禁止执行，不回退 JSON 编辑。
- 结果顶部直接定位失败步骤，展示结论数量、校验差异和清理；版本、实际依赖、必要断言、完整快照与资源证据集中在技术详情。取消不显示为普通业务失败；从历史重跑读取原批次保存的输入，重新准备，不用默认业务值替换。
- 最终 191 项前端测试、Nuxt production build、Go 模板解析与 Bridge v2 生成校验、双模板同步检查通过。浅色／深色列表、说明抽屉、业务表单、正常结果、失败定位与执行记录截图已核对。
- 最终五条 Chromium 路径通过：目录不可用；真实 TemplateService／SQLite／SSE 下的完整操作链路；批量选择、环境切换与空字段提示；只读权限；缺少业务表单时明确报错。其中完整链路覆盖返回准备不执行、连续三次正常运行、持久化内容故障、取消、刷新历史和原输入重跑准备；目录扩展数据只测试显示与表单，没有执行 Core 或合成场景。
- 页面五个业务批次加 CLI 一个正常批次，共六批次；CLI exit=0，active_templates=0，取消批次非通过。故障注入可在业务服务的修改读回阶段被提前检测，页面按实际记录定位，未固定假设失败必在最终读取步骤。
- 最终临时 Go 父／子 PID 84175／84881、端口 56594，Nuxt PID 84975、端口 56595；脚本 exit=0、backend_exit=0，两端口 released=true。此前失败迭代的临时服务同样按脚本清理，未操作用户日常服务。
- 日志：`artifacts/regression/interaction-unit.log`、`interaction-build.log`、`interaction-browser-validation.log`、`interaction-template-render.log`；页面截图与报告在同目录的 Playwright 产物中。用户 3131 服务的列表、结果和插件表单模块均已返回新版内容。
- 更新用例操作指南、其他插件接入说明、展示契约、spec FR-025～FR-027 和既有任务的页面验收记录。此轮为本地测试身份与业务存储验收；T045 仍待专用 Core 环境，未宣称安装态授权或正式发布 consumer 已完成。


## Template 8 条业务用例扩展与验收（2026-10-15）

### 实现与业务范围

- 保留 `template.lifecycle`；新增 `template.search`、`template.pagination`、`template.deletion`、`template.clone`、`template.partialClone`、`template.invalidInput`、`template.validation`。当前 controlled 有 8 条可独立运行的真实业务用例；另有 live 专属 Metadata 场景，未计入当前可运行数。
- 新场景在 `skeleton/backend/go-gin/internal/services/admin/regression/scenarios/template_business.go`；`service.go` 注册完整目录；`profile.go` 装配正式业务依赖。HTTP 校验使用 `template_http.go` 注册的现有 Template 路由、DTO 和 Handler，可信身份由执行器传入，没有网络监听、登录／宿主授权证明，也没有 transport mock 或 ProviderMode 降级。
- 普通／严格内容校验由 Service 创建空内容草稿，分别验证 warning／error；写入修复内容并正式读回后验证通过。HTTP 非法输入场景验证空白名称、超长内容的创建和修改请求，既检查 400 与标准错误码，也核对出错字段，最后验证数据未变。
- 每条场景独立命名空间、即时登记 fixture，复制副本逐条登记，清理限制在同租户、同批次的已登记 UUID。输入名称预留测试标记与多字节字符空间；对象 UUID 集合只放技术证据，默认差异使用业务字段、数量和匹配事实。
- 工作台表头可全选当前列表；准备弹窗逐条说明目的与数据影响，输入表单按需展开，校验错误主动展开对应表单。插件提供完整展示映射和默认值，中英文 locale key 对齐（regression 共 278 个叶子 key）。scaffold 与 CLI 内嵌模板同步。

### 最终验证结果

| 验证 | 结果与产物 |
|---|---|
| 真实业务整批与独立执行 | 正序／倒序各 8 条、逐条独立运行 8 次，共 24 次用例执行通过；`artifacts/regression/template-catalog-evidence.json` |
| 业务故障注入 | HTTP 已拒绝但后台仍写入时，`template.noPollution` 必须 failed，批次 failed、cleanup succeeded；3 条资源清理完成；同上报告内第 11 个批次 |
| 已有数据保护 | 同租户、其他租户各一条既有模板在全部运行后仍保持原值；最终清理测试基线后 active_templates=0 |
| Go race | `GOMAXPROCS=4 REGRESSION_EVIDENCE_DIR=... go test -p=2 -race ./internal/services/admin/regression/... ./internal/transport/http/admin/regression ./tests/integration/regression -run 'Regression\|TemplateRepeated\|MetadataScenario' -count=1` 通过；`template-catalog-backend.log` |
| 浏览器 | 6/6 Chromium 测试通过（32.0 秒），包含真实 8 条批次、全选、批量表单纠错、单条输入、故障、取消、刷新历史、只读与缺展示映射；`template-catalog-browser.log` |
| CLI 批量 | 同一真实业务 API 上重复 8 个 `--case`，exit=0，8 条均 passed、cleanup succeeded；`cli-business-report.json` |
| 浏览器／CLI 存储事实 | 共 7 个批次，含正常、失败、取消与两次真实 8 条批量；最终 active_templates=0；`browser-business-evidence.json` |
| 两个生成插件 | alpha／beta 均运行完整 Template 目录、原生命周期故障和新意外写入故障测试，并通过服务、HTTP 与 cmd/plugin 编译；`template-catalog-generated.log` |
| 前端 | 20 个测试文件、191 项测试通过；Nuxt production build 通过；`template-catalog-unit.log`、`template-catalog-build.log` |
| 同步与格式 | `npm run sync:templates -- --check`、locale key 对齐、`git diff --check` 通过；`template-catalog-sync-check.log` |

### 页面截图与操作

截图目录：`artifacts/regression/playwright/regression-workbench-all-e-67c1d-d-and-executed-as-one-batch-chromium/`，包含 `eight-case-catalog.png`、`batch-preparation.png`、`eight-case-results.png`；按真实视口截图并目视检查用例数、业务名称、选择数量、输入纠错、开始按钮和结果总数。单条流程另保存浅色／深色目录、详情、表单及失败差异截图。

操作：插件能力 → 业务回归 → 确认“当前可运行 8 个回归用例” → 表头全选 → 运行选中的 8 个用例 → 核对目的与数据影响、按需展开输入 → 开始执行 → 核对通过 8 个、清理成功。过滤后全选仅包含可见用例；预期拒绝／部分失败得到正确处理时，用例为通过。

### 临时进程、日常实例及边界

最终浏览器验收由脚本启动 Go 父进程 24407／后端子进程 24873（启动命令见 `template-catalog-browser.log` 第一行），Nuxt production server PID 24894 使用独立临时端口。最终端口 54354／54355 均释放，脚本退出 0、backend_exit=0；各次迭代端口的独立检查保存于 `template-catalog-process-cleanup.json`，没有遗留测试服务。

日常后端 PID 37978（8078，11:55 启动）与前端 PID 94786（3131）未停止或重启。直接请求日常目录返回 401，本轮未用测试注入身份访问该实例，也未声称日常实例目录已更新。Go 场景需要用户按原方式重新启动日常后端后加载，再从页面确认 8 条目录。

以上为本地源码、隔离 SQLite、注入测试身份、真实业务服务／进程内 HTTP、浏览器与 CLI 证据。未执行真实 Core grant／撤权验收，T045 仍待专用环境；未重新运行两个生成插件各自的 Nuxt build／浏览器，未发布 Framework 模块、提交推送、切换分支或触发远端 CI。后续审核发布、跨租户业务操作和客户／联系人用例未包含在本次实施。


## 2026-10-19 配套源码收口验证

在当前 `025-framework-business-regression` 分支统一收口 Skeleton、scaffold／CLI 内嵌模板、.NET、回归工具、只读恢复、包裹体积工具及文档规范。已发布 SDK tag 保持不变，本轮提交不等同于重新发布 CLI 或 .NET。

本轮通过 Skeleton 后端全量测试、CLI 全量测试与实际生成插件的 IAM／WS race 检查、.NET 246 项测试、前端 197 项单元测试、打包 10 项测试、合同／模板／agent 路径检查，以及业务回归完整脚本。Nuxt production build 通过；真实浏览器 8 项通过，包含桌面与窄屏导航、只读用户、业务表单与整批 8 用例执行；独立 CLI 8 用例批次退出 0。

临时 Go 父进程 80755／后端子进程 81175、Nuxt PID 81283 的启动命令与完整日志保存于忽略目录 `artifacts/regression/commit-readiness-20261019/`。后端端口 55648、前端端口 55660 均已释放；日常实例未重启。lint 仍为占位脚本，不能作为静态检查通过的证据。

以上验证使用本地源码与受控测试身份／SQLite fixture，未新增真实 Core grant、PostgreSQL 安装迁移或客户现场验收。T045 继续等待专用环境；后续远端 CI 结果另行记录。
