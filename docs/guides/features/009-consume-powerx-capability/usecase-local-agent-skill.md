# 本地智能体与 Skill 管理

## 入口与边界

菜单「智能体」位于「PowerX 底座能力」上方：

- 智能体管理：`/intelligence/agents`。
- Skill 管理：`/intelligence/skills`。
- 安装路径同时提供 `/_p/com.powerx.plugins.base/admin/intelligence/{agents,skills}`。

页面沿用 Core 智能体卡片、名称/状态筛选、编辑弹窗，以及 Skill 列表的交互组织。
本地管理接口为 `/api/v1/admin/local-intelligence/*`，仅操作插件本地定义。
菜单沿用 root 可见性；接口继续经过现有 Admin JWT/RBAC，并从认证上下文取 tenant。
旧 PowerX 注册/同步页面保留原路由，此处不调用同步，也不自动导入同步表记录。

## Framework 与存储

- `runtime/agent.ManagementService`：ListAgents、SaveAgent、DebugAgent。
- `runtime/skills.ManagementService`：ListSkills、SaveSkill、InvokeSkill。
- 各模块的 `NewManagementRuntime` 以启动模式选择实现；当前本地管理入口明确绑定 Local。
- Skeleton 本地服务：`internal/services/localagent`；HTTP 层持有 Framework 类型化接口。
- 持久化表：`local_agent_definitions`、`local_skill_definitions`。
- 对象主键与 Agent → Skill 引用均为 UUID。列表按租户分页，更新不能修改 key，Skill 也不能修改已保存的 version。
- 名称、模型和提示词属于本地定义；不会修改 PowerX 主数据。其他版本使用新建 Skill 定义。
- Delegated 管理 adapter 未在本次实现；缺失 adapter 不回退。

## 迁移与运行

本次只将新表注册到标准迁移，没有自动执行运行环境迁移或启动后台。
保持现有配置加载方式，在仓库下执行：

```sh
cd skeleton
make migrate
```

然后重启插件后台，刷新前端。没有 Core migration 或 Core 重启要求。
定义会跨重启保留；这里的调试是单次请求，不持久化聊天会话。

## Skill

1. 新增名称、稳定英文标识、版本、描述，选择执行器。
2. 支持：
   - `template`：调用现有 Framework Local Invoker → 模板 Registry → 模板业务服务。
   - `prompt`：使用已保存的提示词，将 JSON 输入交给插件本地模型。
3. 提示词 Skill 启用前必须配置本地模型与非空提示词；模板 Skill 不需要模型。
4. 保存、启用后点击「调试」，模板列表输入 `{"action":"list"}`，结果包含 trace。
5. 模板 create/update/delete 是真实本地业务写入，不是 mock；调用上下文使用认证 member UUID。
6. 停用后直接执行及经 Agent 执行都拒绝。

这里不会执行用户输入的任意 URL、shell 或上传代码。Skill 包导入、目录市场、签名与发布审批不属于这两个本地执行器的功能。

## 智能体

1. 新增名称、稳定 key、描述、模型、角色设定与系统提示词。
2. 在「Skill 管理」页签关联本地 Skill；不要求 PowerX 同步。
3. 无可用本地模型时可以先以停用状态保存；启用时明确校验。
4. 调试输入消息，可明确选择一个已绑定 Skill 并填写 JSON 输入。
5. 选择 Skill 时，先执行 Skill，再将其结果与消息交给模型；返回模型文本、Skill 结果与 trace。
6. 未绑定、已停用、跨租户 Skill 均拒绝；模型失败不改用 Core。
7. 单次调试不自动选择工具，也不等同于多轮 Agent Session、规划器或团队编排。

模型列表与调用使用现有 `deps.LocalAI`（插件 `local_ai` 启动配置），不是 Core 模型目录。
模型驱动可能访问其配置的模型服务；Local 表示运行责任与配置归插件，不代表模型一定运行在同一进程。

## 验证

源码验证：

```sh
go -C skeleton/backend/go-gin test ./internal/services/localagent ./internal/transport/http/admin/localagent ./cmd/database/migrate ./cmd/plugin
go -C framework/backend/go test ./runtime/agent ./runtime/skills
```

覆盖文件数据库重开后的定义与 UUID 关联、模板 Registry 实际列表调用、提示词执行契约、
跨租户查询/更新/关联/调用拒绝、停用拒绝、未绑定 Skill 拒绝、重复 key/version 冲突、
HTTP 禁止 tenant/endpoint 字段与缺本地模型明确失败。

前端 build、模板同步另行检查。实际 PostgreSQL 迁移、浏览器操作和真实本地模型响应仍须在用户运行最新后台后验证。
