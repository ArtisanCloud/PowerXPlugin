# 能力模型概念总览

以 skeleton 中的 `template` CRUD 为例，说明 PowerX 如何识别、同步并调用插件能力。

> 在 PowerXPlugin 仓库中演练时，请进入 `repo-root/skeleton` 再执行文中的 CLI 命令——这里使用的 `./plugin.yaml` 即 `skeleton/plugin.yaml`。若是在你自己的插件仓库，则继续使用其根目录下的 `plugin.yaml`。

## 能力是什么？

- **能力（Capability）**：插件对外提供、可被 PowerX Workflow/Agent 调度的受控接口，拥有全球唯一 `namespace.resource.action` ID，例如 `com.powerx.skeleton.template.create`。
- **能力契约**：`capabilities/catalog.json` + `contracts/capabilities/<id>.yaml` + 输入/输出 JSON Schema，用于描述能力的语义、输入输出和元数据。
- **暴露 Artefacts**：`contracts/exposure/openapi.yaml`、`workflow/*.json`、`agent-streams/*.yaml` 等，用来告诉 PowerX 各协议的访问路径和参数结构。

> 普通 Console/Admin API 不需要进入 catalog；只有带有能力标注的接口才会在安装阶段被 Workflow/Agent 感知。

## 层级关系

| 层 | 目录/组件 | 作用 |
|----|-----------|------|
| 能力接口层 | `backend/internal/handlers/capabilities`（或 `transport/http/exposure`） | 按 ID 暴露 REST/gRPC/MCP 等 handler，通常由 CLI 生成 stub。 |
| 能力目录层 | `capabilities/catalog.json`、`contracts/capabilities/*.yaml` | 声明能力基础信息，供 Capabilities Manager 及 CLI 使用。 |
| 暴露资产层 | `contracts/exposure/*`、`dist/agent-sdk` | 生成 Workflow Step、Agent 工具、OpenAPI/Proto 等 artefacts，提供给 PowerX Builder/Agent Hub。 |
| 调度层 | PowerX Workflow/Agent + Host Gateway | PowerX Loader 在插件安装时读取 catalog，Host Gateway 根据 `protocols.rest`/`protocols.grpc` 等字段建立路由；Workflow Builder / Agent Hub 将 `com.powerx.xxx.*` 节点拖入流程，执行时由宿主统一调度，附带租户上下文与 RBAC，再通过代理调用插件 handler 并回传结果。 |

### Workflow / Agent 如何复用能力

- 每个能力的 `metadata.protocols` 块除了 `rest`/`grpc` 外，还可以包含 `workflow`（引用 `contracts/exposure/workflow/*.json`）、`agent_tool`（指向 `contracts/exposure/mcp-tools.json` 中的条目）、`agent_stream`（流式推送）。  
- 当你运行 `npm --prefix scripts/capabilities run export` 时，这些协议描述会被同步到 `contracts/exposure/*` 和 `dist/agent-sdk/manifest.json` 中；PowerX Workflow Builder 可以直接读取 Workflow JSON 渲染节点，Agent Hub 则使用 MCP manifest + SSE 定义生成工具和订阅。  
- 操作手册：
  - 《[Agent REST/gRPC 联调手册](./agent-rest-grpc-guide.md)》
  - 《[Workflow + Agent 联调手册（模板示例）](./workflow-agent-guide.md)》
  - 《[MCP 会话与流式能力联调手册](./mcp-guide.md)》
- 复合场景的落地示例仍可参考《[Workflow + Agent 联调手册（模板示例）](./workflow-agent-guide.md)》，该文聚焦业务案例；具体调试步骤请阅读上面的三份手册。

## 能力接口与普通开放接口的区分

- **命名空间隔离**：能力 handler 统一挂载在 `backend/internal/handlers/capabilities` 或 `transport/http/exposure`，REST 前缀可设为 `/api/v1/exposure/*`；普通 Admin/Console API 继续使用 `/api/v1/admin/*` 等路径。
- **显式标注**：在 handler 上添加 `// capability: <id>` 或调用包装函数 `CapRoute(group, "<id>", handler)`，扫描器只处理带标注的接口。
- **自动生成契约**：发现标注后自动更新 catalog/descriptor/schema，并在能力目录层写出 artefacts，避免 Console API 被误注册。
- **RBAC 区别**：能力接口使用 Capabilities RBAC（宿主意图），普通 API 继续走 Console/Admin 权限，防止宿主误调内部接口。

## Template CRUD 流水线

1. **标注 handler（写清 ID 与语义）**
   ```go
   // 文件位置：backend/internal/handlers/capabilities/template_create_handler.go
   // 由脚手架或 px-plugin capabilities init 自动生成骨架
   // CreateTemplate 暴露 com.powerx.skeleton.template.create
   // capability: com.powerx.skeleton.template.create
   func (h *TemplateHandler) CreateTemplate(c *gin.Context) {
       if err := c.ShouldBindJSON(&req); err != nil {
           ...
       }
       ...
   }
   ```
   - **生成方式**：运行 `px-plugin capabilities init` 或 Go scaffolder 时，会在 `backend/internal/handlers/capabilities/` 下生成带占位代码的 handler（与 `transport/http/admin/...` 的 Console handler 不同，此目录只处理能力接口）；
   - `capability:` 注解紧贴 handler，并遵循 `namespace.resource.action`；`namespace` 通常取 plugin ID，`resource`=模型；`action`=CRUD/自定义动词；
   - 若沿用 `transport/http/admin/templates` 的 handler，可在原函数上添加注解，但推荐复制到 `handlers/capabilities` 目录以便扫描器定位。
2. **扫描 & 初始化契约（支持批量脚本）**
   ```bash
   # 扫描带注解的 handler，输出 JSON 描述
   node ../scripts/capabilities/discover-handlers.mjs \
     --plugin . \
     --handlers backend/internal/transport/http/admin/templates \
     --output tmp/template-capabilities.json

   # 批量初始化（也可使用 --capability-id 单独运行）
   npx --yes tsx ../tools/cli/src/commands/capabilities/init.ts \
     --manifest ./plugin.yaml \
     --batch tmp/template-capabilities.json
   ```
   CLI 会生成/更新 `capabilities/catalog.json`、`contracts/capabilities/*.yaml`、`contracts/schema/input|output/*.json`。`tmp/template-capabilities.json` 可手工维护，以便一次性处理多个模型。
3. **导出多协议 artefacts**
   ```bash
   npm --prefix scripts/capabilities run export
   ```
4. **提交能力并同步暴露配置**
   ```bash
   px-plugin capabilities submit \
     --manifest ./plugin.yaml \
     --base-url $PX_DEV_API_BASE \
     --token $PX_DEV_API_TOKEN
   ```
   - **提交目的地**：PowerX Publish Hub（能力中心 API）。
   - **凭证来源**：从 PowerX Dev Console 申请插件 Dev API Token，或在宿主安装阶段获取后写入 `.env`/CI Secret。
   - CLI 会调用 `/internal/plugins/capabilities` 与 `/internal/plugins/capabilities/{id}/exposure`，校验 catalog 与暴露资产，并把结果写入 `.px-plugin/capabilities.json`。
5. **PowerX 调度**：Workflow Builder/Agent Hub 读取 catalog + `contracts/exposure/workflow/template-*.json`，将 `com.powerx.skeleton.template.create`/`review`/`publish` 节点拖入流程；执行时宿主通过 catalog 中的 REST/gRPC 路径调用插件 handler，完成模板 CRUD 能力调用。
