# 插件 ID 合同 v1

本合同仅统一插件身份，不合并不同版本的完整 Manifest 格式。

## 命名

- PowerX 自有插件使用 `com.powerx.plugins.<name>`，例如 `com.powerx.plugins.ai-craft`。
- 明确拒绝旧前缀 `com.powerx.plugin` 及其点分隔子命名空间；生成、启动校验和打包均不会自动改名。
- 第三方继续使用自己的合法命名空间，例如 `com.example.demo`、`org.acme.weather-v2`。保留既有单段 ID 支持，例如 `my-plugin`；命名合法不代表拥有该命名空间，也不构成授权。
- 只允许 ASCII 小写字母、数字、点和中划线。点分隔的段不能为空，每段以字母或数字开头和结束，段内允许中划线。禁止下划线、空白、连续点、首尾点，以及段首尾中划线。
- `com.powerx.plugins` 只是自有命名空间，必须继续提供名称。
- 本轮不额外新增总长度上限；不得以本次对齐静默限制第三方已有命名空间。

机器约束唯一源为同目录 `plugin-id.schema.json`，正反例为 `plugin-id.cases.json`。Framework、CLI 和 Python 的约束从这些文件分发／生成，不各自维护正则。

## 身份一致性

`plugin.yaml` 的 `id` 是发布身份。运行配置的 `POWERX_PLUGIN_ID`、`POWERX_SECURITY_JWT_AUDIENCE=plugin:<id>`、前端 `/_p/<id>/...` 挂载路径及运行时注册身份必须一致。可选设置未提供时不自动写入，显式提供但不一致必须失败。插件 Audience 与 STS 服务 Audience 是不同合同，不能互相替换。

`plugin.yaml` 文件名、`plugin:<id>` 格式、Framework 包名、Go import 和现有仓库目录名分别遵守各自合同。能力 ID、权限资源名等也不等同于插件 ID，不做全局字符串替换。

## 分发和校验

1. 在 PowerXDocs 修订本合同及机器约束，再由 `publish:standards` 分发。
2. 两份标准 Manifest Schema 的 `properties.id` 使用同一个机器约束。分发时可使用 `--json-field properties.id`，仅更新 ID 字段，保留目标仓库完整 Manifest 格式的其他字段。
3. PowerXPlugin 执行 `npm run sync:plugin-id` 生成 Framework、CLI 和 Python 的对应约束，再执行模板同步；CI 用 `--check` 阻断漂移。
4. CLI 初始化前、直接打包／发布读取 manifest 时及 Makefile 的 ID 门禁都复用 Framework 校验；Go 和 Python 启动在连接数据库前检查运行身份。
5. 用新版 CLI 执行 `px-plugin plugin-id check --id com.powerx.plugins.demo` 或 `px-plugin plugin-id check --manifest ./plugin.yaml`；`make plugin-id-check PX_PLUGIN=/path/to/px-plugin` 调用相同入口。旧 CLI 缺少该命令时应升级，不静默切换到另一套规则。

正反例至少覆盖自有和第三方生成、手写旧 ID 打包拒绝、各身份字段不一致、Go／FastAPI 生成物和模板同步。源代码旧值扫描只能作为辅助诊断，不得把历史路径、import 或拒绝测试中的字符串作为身份失败依据。

## 已有插件

这次合同统一不执行现有插件改名、迁移、重装或授权变更。旧 ID 在升级校验链路后会被拒绝；迁移必须另行核对安装记录、授权、数据、缓存和能力绑定，不能按“全局替换后重装”处理。Core 本轮不新增自有前缀限制。
