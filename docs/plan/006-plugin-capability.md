# 006 — 插件能力多协议暴露与复合任务设计

> 为 PowerXPlugin 提供开发可落地的能力注册/暴露方案，使 PowerX 底座在加载插件时即可识别全部原子能力与复合任务，并可在 Workflow 与智能体两个入口中直接消费这些能力。

---

## 1. 背景与诉求

- PowerX Core 已具备 `backend/internal/service/capability*`、`backend/internal/transport/http/admin/capability*` 等模块，要求插件在安装/注册时提交结构化能力目录。
- 现有 spec 仅提到 REST/GraphQL/Webhook/SDK，需要补齐 gRPC/OpenAPI、Agent（MCP/SSE）、Workflow Step、复合任务等协议描述。
- PowerX 有两大调用场景：Workflow（拖拽节点）与智能体（Agent Hub 意图调度）。两者都必须共享同一份插件能力目录。
- 插件开发者需要明确的分层、目录与 CLI 工具规范，以便快速实现“原子能力 → 能力目录 → 能力管理器 → PowerX 底座”链路。

---

## 2. 分层架构

| 层级 | 目录/位置 | 作用 | 说明 |
|------|-----------|------|------|
| **原子能力层 (Atomic Service Layer)** | `backend/internal/handlers/capabilities/<domain>/<action>_handler.go` | 绑定唯一能力 ID 的 Handler，是最小可调用单元。 | Handler 仅负责输入校验、调用 `skeleton/backend/internal/services/**` 里的业务逻辑、输出转换与审计。 |
| **能力目录层 (Capability Registry Layer)** | `capabilities/*.yaml` + `plugin.yaml` (`capabilities.imports`) | 记录全部原子能力与复合任务的元数据（ID、描述、协议、标签、复合关系）。 | 与 `contracts/capabilities/*.yaml`、`contracts/schema/**` 链接，方便 CLI/运行时解析。 |
| **能力管理器层 (Capabilities Manager Layer)** | CLI + 运行时共享模块（`px-plugin capabilities *`、框架 `framework/capabilities/manager`） | 解析能力目录、生成协议产物、向 PowerX 注册能力。 | 暴露 `ListCapabilities()`、`ExportProtocols()`、`RegisterWithHost()` 等接口，供 PowerX 底座在插件加载时调用。 |

> 流程：插件安装 → Capabilities Manager 解析目录 → 调用 PowerX `/admin/capability_registry` → Workflow & Agent 立即获得插件节点/工具。

---

## 3. 能力文件与协议矩阵

| 通道 | 文件/资产 | 触发方式 | PowerX 消费方 |
|------|-----------|----------|---------------|
| REST / OpenAPI | `contracts/exposure/openapi.yaml` | HTTP | API Gateway、Portal、SDK 生成器 |
| GraphQL | `contracts/exposure/graphql/*.graphql` | HTTP | GraphQL Stitching & Portal |
| gRPC | `contracts/exposure/proto/*.proto` + `buf.yaml` | gRPC | gRPC Gateway、客户端 Stub、Workflow | 
| Agent Tool (MCP) | `contracts/exposure/mcp-tools.json` | MCP Runtime | Agent Hub 工具库 |
| Agent SSE (实时) | `contracts/exposure/agent-streams/*.yaml` | SSE/WebSocket | Agent Hub（意图调用），支持事件推送 |
| Workflow Step | `contracts/exposure/workflow/*.json` | Workflow Builder | Workflow 节点模板（输入/输出槽、回滚策略） |
| Webhook（订阅） | `contracts/exposure/webhook/*.json` | Host Push | PowerX 通知中心 |
| SDK Bundle | `dist/agent-sdk/*` | N/A | 对外 SDK / Postman 包 |

`plugin.yaml` 示例：
```yaml
capabilities:
  imports:
    - capabilities/com.powerx.demo.template.create.yaml
    - capabilities/com.powerx.demo.template.compose.yaml
```
单个能力文件：
```yaml
id: com.powerx.demo.template.create
description: 创建模板的原子能力
atomic_service: backend/internal/handlers/capabilities/template/create_handler.go
protocols:
  rest:
    openapi: contracts/exposure/openapi.yaml#/paths/~1templates/post
  grpc:
    proto: contracts/exposure/proto/template.proto
    service: powerx.template.TemplateService
  agent_tool:
    mcp_manifest: contracts/exposure/mcp-tools.json#/tools[0]
  agent_stream:
    sse: contracts/exposure/agent-streams/create-template.yaml
  workflow_step:
    template: contracts/exposure/workflow/template-create.json
tags: [integration, atomic]
```

---

## 4. 原子能力层

- **目录规范**：`backend/internal/handlers/capabilities/<domain>/<action>_handler.go`。使用 `*_handler.go` 命名，导出 `Handle` 或 `Execute` 函数，方便框架自动注册。
- **职责**：
  - 解析输入（引用 `contracts/schema/input/<domain>.<action>.json`）。
  - 调用 `skeleton/backend/internal/services/**` 的业务方法。
  - 返回输出（引用 `contracts/schema/output/<...>.json`）。
  - 记录审计事件（例如 `audit.capability.template.create`）。
- **CLI 支撑**：`px-plugin capabilities new` 基于 Handler 路径生成能力文件与 Schema stub；`lint` 确保 Handler/Schema/协议引用一致。

---

## 5. 复合任务层

- 文件位于 `contracts/exposure/composites/*.json`，描述 DAG（节点=原子能力或辅助服务，边=数据流/条件）。
- 每个复合任务必须同时提供：
  1. Workflow Step 模板（便于直接拖拽复用插件原生 Workflow）。
  2. Agent Tool + Agent SSE 协议（便于智能体按意图调用）。
- 支持“导入插件 Workflow”：能力文件可标记 `composite.type: workflow` 并附 DSL；PowerX 加载后即在 Workflow Builder 生成一个可拖拽节点。
- 回滚策略：复合任务需引用可复用的回滚原子能力，并在模板中声明补偿逻辑。

示例：
```json
{
  "id": "com.powerx.demo.template.compose",
  "type": "workflow",
  "graph": {
    "nodes": [
      {"id": "draft", "capability": "template.create"},
      {"id": "review", "capability": "template.review"},
      {"id": "publish", "capability": "template.publish"}
    ],
    "edges": [
      {"from": "draft", "to": "review"},
      {"from": "review", "to": "publish", "condition": "approved"}
    ]
  },
  "workflow_step": "contracts/exposure/workflow/template-compose.json",
  "agent_stream": "contracts/exposure/agent-streams/template-compose.yaml"
}
```

与 PowerX `pkg/corex/flow/schemas/node.go` 对齐的 Workflow Step 模板示例（`contracts/exposure/workflow/template-compose.json`）：
```json
{
  "nodes": [
    {
      "id": "draft",
      "kind": "plugin",
      "use": "com.powerx.demo.template.create",
      "params": {
        "capability_id": "com.powerx.demo.template.create",
        "protocol": "rest"
      },
      "io": {
        "in_map": {
          "body": "{{var.template_draft}}"
        },
        "out_map": {
          "template_id": "output.id"
        }
      }
    },
    {
      "id": "review",
      "kind": "plugin",
      "use": "com.powerx.demo.template.review",
      "params": {
        "capability_id": "com.powerx.demo.template.review"
      },
      "io": {
        "in_map": {
          "template_id": "{{task.draft.output.template_id}}"
        },
        "out_map": {
          "approved": "output.approved"
        }
      }
    },
    {
      "id": "publish",
      "kind": "plugin",
      "use": "com.powerx.demo.template.publish",
      "params": {
        "capability_id": "com.powerx.demo.template.publish"
      },
      "io": {
        "in_map": {
          "template_id": "{{task.draft.output.template_id}}",
          "approved": "{{task.review.output.approved}}"
        },
        "out_map": {
          "publish_status": "output.status"
        }
      }
    }
  ],
  "edges": [
    {
      "from": "draft",
      "to": "review"
    },
    {
      "from": "review",
      "to": "publish",
      "condition": "{{task.review.output.approved}}"
    }
  ]
}
```
> `kind/use/io` 均遵循 `Node` 结构，`in_map/out_map` 中的表达式可使用 `exec/resolve.go` 中支持的 `{{task.xxx.output.field}}`、`{{var.xxx}}` 等绑定语法。

---

## 6. 安装与调用流程

1. **构建阶段**
   - 开发者实现 Handler，并运行 `px-plugin capabilities new`、`px-plugin capabilities lint/compose/export` 生成/校验能力目录、OpenAPI/Proto/MCP/Workflow 文件。
2. **安装阶段**
   - `.pxp` 包包含 `capabilities/` 与全部 `contracts/exposure/*`。
   - PowerX 加载插件 → 调用 Capabilities Manager → 获取能力目录与协议资产 → `capability_registry` 服务写入记录。
3. **调用阶段**
   - **Workflow**：Builder 读取 Workflow Step 模板，用户拖拽节点；执行时 Workflow Engine 根据模板调用插件 Handler 或复合 Workflow。
   - **智能体**：Agent Hub 读取 MCP manifest / Agent SSE 工具，将其注册到工具库；智能体根据意图调度原子能力、复合任务或插件 Workflow。
   - **API/Gateway**：API Gateway 读取 OpenAPI/gRPC 协议并路由请求到 `/_p/<plugin>/...` 或插件 gRPC server，PowerX 在前置层完成鉴权/限流。

---

## 7. CLI 与运行时职责

| 能力 | CLI 子命令 | 运行时行为 |
|------|-------------|------------|
| 能力创建 | `px-plugin capabilities new --id ...` | 在 `capabilities/` 生成模板，注册 Handler stub。 |
| 目录校验 | `px-plugin capabilities lint` | 启动时由 Capabilities Manager 自动执行校验，阻断构建或启动。 |
| 契约导出 | `px-plugin capabilities export --format openapi|proto|workflow|mcp|agent-stream` | 运行时缓存协议产物，供 PowerX 请求。 |
| 复合任务 | `px-plugin capabilities compose` | 运行时向 PowerX 声明复合任务，暴露 Workflow/Agent 双形态。 |
| 状态同步 | `px-plugin capabilities submit` | 安装时向 `capability_registry` 写入能力/协议/审批状态，PowerX 以此生成节点/工具。 |

---

## 8. 实施阶段

| 阶段 | 时间 | 关键交付 |
|------|------|----------|
| Phase A：资产模板 | Week 1 | 脚手架新增 gRPC/OpenAPI/MCP/Workflow/Agent SSE 模板；`plugin.yaml` 加入 `capabilities.imports`；更新标准文档。 |
| Phase B：CLI 支持 | Week 2-3 | `px-plugin capabilities lint/submit/proto/openapi/mcp/compose` 落地；生成复合任务/Workflow 引入模板。 |
| Phase C：运行时 Hook | Week 4-5 | 框架启动 Capabilities Manager，自动注册 gRPC server、OpenAPI entry、Agent Tool、Workflow Step Catalog。 |
| Phase D：测试 & 文档 | Week 6 | E2E 测试（REST vs gRPC vs Agent SSE 一致性、Workflow/Agent 双入口）；更新 `docs/use_cases/...` 与发布指南。 |

---

## 9. 关键任务清单

1. **协议声明**
   - [ ] `plugin.yaml` 支持 `protocols` 数组与 `capabilities.imports`。
   - [ ] 生成 OpenAPI/Proto/Workflow/MCP/Agent SSE 契约文件。
2. **CLI & 验证**
   - [ ] `px-plugin capabilities lint` 校验 Handler、Schema、协议引用的一致性。
   - [ ] `px-plugin capabilities export --format ...` 一次产出所有协议资产。
3. **运行时注册**
   - [ ] Go 框架提供 gRPC server + HTTP Gateway，自适应 Handler。
   - [ ] Agent Runtime 读取 MCP manifest、SSE 模板并注入 PowerX Agent Hub。
   - [ ] Workflow SDK 根据模板注册 Step Catalog，可导入插件自带 Workflow。
4. **复合任务编排**
   - [ ] `contracts/exposure/composites` 定义 DAG；
   - [ ] `px-plugin composites simulate` 本地模拟 Workflow/Agent 运行；
   - [ ] 复合任务回滚规范与示例。
5. **文档 & 培训**
   - [ ] `docs/guides/publish/multi-protocol-capabilities.md`；
   - [ ] README / meta 文档链接示例；
   - [ ] Demo（REST vs gRPC vs Agent SSE 调用、Workflow 拖拽示例）。

---

## 10. 依赖与风险

| 风险 | 影响 | 缓解 |
|------|------|------|
| gRPC / SSE 基础设施未启用 | 多协议无法联调 | 提供本地 mock server；加 flag `PX_CAPABILITY_PROTOCOL_MATRIX`。 |
| MCP / Agent 规范快速变化 | 契约频繁更新 | CLI 加入版本字段，文档列兼容策略。 |
| 复合任务回滚复杂 | 项目风险 | 先实现串行+固定补偿模板，后续迭代支持复杂 DAG。 |

---

## 11. 成功标准

- 插件安装后，PowerX 通过 Capabilities Manager 即可获得 OpenAPI、Proto、Workflow Step、MCP manifest、Agent SSE 模板，并在 Portal/Agent Hub/WF Builder 中显示节点。
- Workflow 与智能体都能消费同一份能力目录：
  - Workflow Builder 支持拖拽原子能力或直接导入插件 Workflow。
  - 智能体可基于意图调度原子能力、复合任务或插件 Workflow。
- 配置更新后 3 分钟内，多协议通道全部生效；所有能力调用共享审计指标，复合任务回滚/失败率可追踪到 `capability.lifecycle.*`。

---

## 12. plugin.yaml 模块化引用

1. **目录结构**
   ```
   capabilities/
     com.powerx.demo.template.create.yaml
     com.powerx.demo.template.compose.yaml
   ```
2. **引用方式**
   ```yaml
   capabilities:
     imports:
       - capabilities/com.powerx.demo.template.create.yaml
       - capabilities/com.powerx.demo.template.compose.yaml
   ```
3. **CLI 支持**
   - `px-plugin capabilities new` 默认在 `capabilities/` 生文件。
   - `lint/submit/export` 自动解析 imports 并检测重复 ID、缺失协议。
4. **兼容策略**
   - 仍支持在 `plugin.yaml` 直接写 `capabilities.provides`；若 imports 与 provides 并存，以解析合并后的结果为准。

---

## 13. PowerX 调用插件能力的参考流程

1. **注册**：插件安装 → Capabilities Manager 暴露能力目录 → PowerX `capability_registry` 写入记录。
2. **Workflow**：Workflow Builder 加载插件 Workflow Step 模板或复合任务节点；执行时通过 REST/gRPC 调插件 Handler 或触发插件 Workflow。
3. **智能体**：Agent Hub 加载 MCP manifest + Agent SSE，注册为工具；智能体根据意图调用原子能力或复合任务，也可调度插件 Workflow。
4. **API/Portal**：API Gateway 基于 OpenAPI/gRPC 协议路由到插件，Portal/SDK 自动展示调用方式；Webhook 由插件负责推送事件给 PowerX。

如上设计确保 PowerX 在“智能体 + Workflow”两条主路径上均可直接使用插件能力，同时所有能力都通过统一目录与管理器治理，便于开发、审计与扩展。
