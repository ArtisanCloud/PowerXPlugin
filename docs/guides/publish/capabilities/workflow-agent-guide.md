# Workflow + Agent 联调手册（模板示例）

> 提示：原 `workflow-agent-template.md` 与 `workflow-guide.md` 已合并至本手册，后续请统一参考此文档。

以 skeleton 中的 `template` 模型为例，演示如何在已有 REST/gRPC 能力的基础上，继续定义 Workflow/Agent 复合能力，并在 PowerX 中完成调试。

> 前置要求：你已经按照《tooling-template.md》完成 handler 扫描、契约初始化与 `npm test`，并在本地验证 REST/gRPC 接口可用。

## 1. 明确原子能力

确保以下能力已经在 `contracts/capabilities/*.yaml` 中声明：

- `com.powerx.plugins.base.template.list`
- `com.powerx.plugins.base.template.read`
- `com.powerx.plugins.base.template.create`
- `com.powerx.plugins.base.template.update`
- `com.powerx.plugins.base.template.delete`
- `com.powerx.plugins.base.template.batch_clone`
- `com.powerx.plugins.base.template.validate`

每个能力的 `metadata.protocols` 应至少包含 `rest`（必选）与 `grpc`（可选），并指向对应 handler 路径。后续 Workflow/Agent 节点会直接复用这些 ID。

## 2. 定义 Workflow

在 `contracts/exposure/workflow/` 中新增或更新一个流程 JSON，例如 `template-compose.json`：

```json
{
  "id": "com.powerx.plugins.base.template.compose",
  "version": "1.0.0",
  "name": "模板批处理",
  "nodes": [
    {
      "id": "draft-batch",
      "kind": "plugin",
      "use": "com.powerx.plugins.base.template.create",
      "params": { "protocol": "rest", "capability_id": "com.powerx.plugins.base.template.create" },
      "io": {
        "in_map": { "body": "{{var.batch_payload}}" },
        "out_map": { "created_ids": "output.ids" }
      }
    },
    {
      "id": "update-selected",
      "kind": "plugin",
      "use": "com.powerx.plugins.base.template.update",
      "io": {
        "in_map": {
          "id": "{{task.draft-batch.output.created_ids[0]}}",
          "body": "{{var.update_payload}}"
        },
        "out_map": { "updated_id": "output.id" }
      }
    },
    {
      "id": "delete-rest",
      "kind": "plugin",
      "use": "com.powerx.plugins.base.template.delete",
      "io": {
        "in_map": { "id": "{{task.draft-batch.output.created_ids[4]}}" }
      }
    }
  ],
  "edges": [
    { "from": "draft-batch", "to": "update-selected" },
    { "from": "update-selected", "to": "delete-rest" }
  ],
  "metadata": {
    "summary": "批量创建 5 个模板 → 更新前 2 个 → 删除 1 个，最终返回草稿与变更结果"
  }
}
```

要点：

- `nodes[*].use` 与 `params.capability_id` 必须引用已存在的能力 ID。
- 使用 `io.in_map/out_map` 将上游输出映射到下游输入；可结合 `var.*` 变量让 Workflow Builder/Agent Hub 在调用前注入自定义参数。
- `edges` 描述节点执行顺序与条件（可设置 `condition` 表达式）。

### 场景 2：模板巡检与修复（`template-audit.json`）

第二个 Workflow 负责“巡检已存在模板 → 选中第一条命中项 → 自动修复”，对应文件 `contracts/exposure/workflow/template-audit.json`：

```json
{
  "id": "com.powerx.plugins.base.template.audit",
  "name": "模板巡检与修复",
  "nodes": [
    { "id": "collect-stale", "use": "com.powerx.plugins.base.template.list" },
    { "id": "inspect-primary", "use": "com.powerx.plugins.base.template.read" },
    { "id": "update-primary", "use": "com.powerx.plugins.base.template.update" }
  ],
  "edges": [
    { "from": "collect-stale", "to": "inspect-primary", "condition": "{{task.collect-stale.output.total > 0}}" },
    { "from": "inspect-primary", "to": "update-primary" }
  ],
  "summary": "扫描模板库、挑选命中项后自动修订。"
}
```

它与 `template-compose` 的差异：

- 入口变量来自 `contracts/schema/input/com.powerx.plugins.base.template.audit.json`，可配置巡检过滤条件与自定义的修复内容。
- 第一个节点 `collect-stale` 只要没有命中数据（`total = 0`）就会终止后续流程，PowerX Agent Hub 能通过该条件决定是否继续下个节点。
- Workflow 结束于 `update-primary`，可直接把更新结果回传给 Agent 或交由下游 Workflow/Workflow Builder 决定下一步。

### 场景 3：模板巡检 + 批量分发（`template-quality-distribute.json`）

新 Workflow 结合了批量克隆与巡检校验，文件位于 `contracts/exposure/workflow/template-quality-distribute.json`：

- `scan`：调用 `list` 能力，根据 `var.scan_filter`（关键字、分页）拉当前租户的模板列表。
- `validate`：将 `scan` 的首个模板 ID 输入到 `validate` 能力，`var.validate_rules` 可以覆盖默认校验规则，只有通过校验才继续。
- `batch-clone`：引用 `com.powerx.plugins.base.template.batch_clone`，把 `scan` 输出的 ID 列表批量克隆，`var.clone.*` 控制副本数量/前缀。
- `update`：调用 `com.powerx.plugins.base.template.update`，对克隆出来的模板批量填充最新内容或元数据。

同一 Workflow 会输出 `created_ids`、`violations` 等字段，可在 Agent Hub 中实时观察。对应 artefact：

- Workflow 描述：`contracts/exposure/workflow/template-quality-distribute.json`
- Agent Tool：`contracts/exposure/mcp-tools.json` → `base.template.quality_distribute`
- SSE：`contracts/exposure/agent-streams/template-quality-distribute.yaml`（intent=`template.quality_distribute`）。

> 提示：新增的 `batch_clone` 与 `validate` 能力需要在 `plugin.yaml`、`capabilities/catalog.json` 以及 `contracts/exposure/exposure-packages.json` 中声明，否则 `export` 阶段会报缺失协议。

## 3. 本地验证 Workflow 输入/输出

Workflow 引擎在插件内并不会自动执行，因此本地调试的关键是确认“每个节点所引用的原子能力”能够完成预期：

1. 启动后端（参考《agent-rest-grpc-guide.md》第 2 节）。
2. 使用同一批 REST/gRPC 接口按 Workflow 顺序调用，例如：
   - `POST /api/v1/templates` → 获取 `ids`，对应 `draft-batch` 节点；
   - `PUT /api/v1/templates/{id}` → 验证输出结构满足 `out_map`；
   - `POST /api/v1/templates/batch-clone`、`POST /api/v1/templates/{id}/validate` → 验证批量/巡检节点。
3. 如需校验变量映射是否正确，可将 Workflow JSON 放入 `contracts/exposure/workflow/` 后执行：
   ```bash
   npm --prefix scripts/capabilities run export
   ```
   该命令会校验 JSON Schema 与引用的能力 ID 是否存在。如果报错，可根据日志定位缺失字段或打错的能力 ID。

## 4. 暴露 Agent Tool + 流式能力

在 `contracts/exposure/mcp-tools.json` 中为 Workflow 增加一个工具入口，并且在 `contracts/exposure/agent-streams/*.yaml` 声明 SSE 频道，供 Agent Hub 监听：

```json
{
  "id": "base.template.compose",
  "capability_id": "com.powerx.plugins.base.template.compose",
  "name": "模板批处理 Workflow",
  "description": "批量创建/更新/删除模板并返回成果。",
  "transport": "workflow",
  "endpoint": "contracts/exposure/workflow/template-compose.json",
  "input_schema": "contracts/schema/input/com.powerx.plugins.base.template.compose.json",
  "output_schema": "contracts/schema/output/com.powerx.plugins.base.template.compose.json"
}
```

SSE 示例（`contracts/exposure/agent-streams/template-compose.yaml`）：

```yaml
capability_id: com.powerx.plugins.base.template.compose
version: 1.0.0
intent: template.compose
summary: 推送模板批处理进度
events:
  - name: draft.created
    payload: contracts/schema/output/com.powerx.plugins.base.template.create.json
  - name: template.updated
    payload: contracts/schema/output/com.powerx.plugins.base.template.update.json
  - name: template.deleted
    payload: contracts/schema/output/com.powerx.plugins.base.template.delete.json
```

对于巡检场景，只要在相同文件分别追加：

- `contracts/exposure/mcp-tools.json` → `base.template.audit`（`transport: workflow`，指向 `contracts/exposure/workflow/template-audit.json`）。
- `contracts/exposure/agent-streams/template-audit.yaml` → `intent: template.audit`，事件包含 `audit.scan.completed`、`audit.template.updated` 等。

这样提交后，PowerX Agent Hub 会拿到两个可选 Workflow 工具，宿主可以依据 `intent` 或 `tool_scope` 把调用路由到 `compose` 或 `audit`。

## 5. 导出暴露资产

每次修改 Workflow/Agent 描述后执行：

```bash
cd skeleton
npm --prefix scripts/capabilities run export
```

该命令会同步以下 artefacts：

- `contracts/exposure/openapi.yaml`（REST 端点）
- `contracts/exposure/proto/*.proto`（gRPC）
- `contracts/exposure/workflow/*.json`
- `contracts/exposure/agent-streams/*.yaml`
- `contracts/exposure/mcp-tools.json`
- `dist/agent-sdk/manifest.json`

## 6. 提交并在 PowerX 联调

1. 将最新 `plugin.yaml` + `contracts/*` 提交到仓库。
2. 在 `skeleton/` 运行：
   ```bash
   npx --yes tsx ../tools/cli/src/commands/capabilities/submit.ts --manifest ./plugin.yaml --base-url $PX_DEV_API_BASE --token $PX_DEV_API_TOKEN
   npx --yes tsx ../tools/cli/src/commands/capabilities/quota.ts --capability-id com.powerx.plugins.base.template.compose --tenant sandbox --base-url $PX_DEV_API_BASE --token $PX_DEV_API_TOKEN
   ```
3. 在 PowerX Dev Console → Workflow Builder 选择 `com.powerx.plugins.base.template.compose` 节点，填写 `var.batch_payload` 等参数即可执行流程；Agent Hub 会根据 `mcp-tools.json` 自动生成工具和 SSE 订阅配置。

通过以上步骤，就可以把原子能力串联成可视化 Workflow，并同时暴露为 Agent/MCP 工具，满足“批量创建/更新/删除模板并输出结果”的复合场景。
