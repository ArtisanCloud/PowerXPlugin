# Implementation Plan: 插件能力注册与暴露治理闭环

**Branch**: `006-plugin-capability` | **Date**: 2025-12-04 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/006-plugin-capability/spec.md`

## Summary

为插件生态交付统一的能力注册与多协议暴露管线：开发者在插件侧声明原子能力、协议矩阵与复合 Workflow/Agent 模板，Capabilities Manager 负责解析 `capabilities/*.yaml`、生成 OpenAPI/Proto/MCP/Workflow 契约，并在插件安装或升级时 3 分钟内同步至 PowerX Core（`capability_registry`、Workflow Builder、Agent Hub）。方案要求所有能力默认同步执行，必要时显式声明 `async`，并在目录同步失败时立即阻断安装回滚，确保宿主获取到一致的节点/工具目录。

此外，本迭代新增“插件本地调试模式”，要求 `/capabilities/register` 的调试面板在插件独立运行时即可直接调用本地 REST/gRPC/Workflow 端点，不依赖 PowerX Gateway 或租户授权；实现需复用 `useCapabilityLab` 的 UI/状态管理，但调用适配器必须支持两种模式（本地 / 宿主代理），默认启用本地模式。

## Technical Context

**Language/Version**: Backend Go 1.24、Frontend/CLI Node.js 20 + TypeScript 5 + Nuxt 4.2、脚本 Bash/Node 18 兼容。  
**Primary Dependencies**: Gin HTTP、Gorm ORM、PowerX Plugin Framework（admin/client layer）、px-plugin CLI、OpenAPI/Buf（proto）、Nuxt UI 3.3.x。  
**Storage**: PostgreSQL ≥13，单 schema（`powerx_plugin_base`）+ RLS；目录/契约文件存放在仓库内。  
**Testing**: Go `make test` + service/unit tests、Playwright/NUXT E2E、CLI 集成测试、Schema/contract lint（OpenAPI/Buf/capabilities lint）。  
**Target Platform**: PowerX 插件容器（Linux amd64）+ Web Admin（SSR/SPA via Nuxt）、PowerX Workflow/Agent Runtime。  
**Project Type**: Multi-surface（backend services + CLI tooling + web-admin 文档）。  
**Performance Goals**: 能力注册自动校验 ≤5 分钟、能力目录同步 ≤3 分钟、审核 SLA ≤2 工作日、暴露同步 100% 成功、Workflow/Agent 工具加载失败率 <1%。  
**Constraints**: Go/Nuxt 栈、STS 凭证、租户隔离（UUID/RLS）、manifest 与运行态保持一致、禁止未声明的通道、默认同步执行；能力目录或协议同步失败必须阻断回滚。  
**Scale/Scope**: 100+ 原子能力、几十个复合 Workflow/Agent 节点，支持千级租户并发查询，目标集成 API Gateway + Agent Hub + Workflow Builder。

## Constitution Check

- ✅ 语言/框架符合宪章（Go 1.24、Nuxt 4、Node 20）。
- ✅ 目录分层沿用 `internal/{services,transport}/`、`capabilities/*.yaml`，manifest 宣告符合 Host Contract。
- ✅ 数据存储限 Postgres + RLS；无跨 schema 行为。
- ✅ 依赖容器注入、Handler 轻薄、Service 中心化；无裸 `*gorm.DB` 暴露。
- ✅ 可观测 / 测试要求（审计、指标、Automation）在 spec 中已有指标/回滚策略。

> Gate 通过，可进入 Phase 0。

## Project Structure

```text
specs/006-plugin-capability/
├── spec.md
├── plan.md
├── research.md              # 本次 /speckit.plan 生成
├── data-model.md            # 本次 /speckit.plan 生成
├── quickstart.md            # 本次 /speckit.plan 生成
├── contracts/               # 本次 /speckit.plan 生成 (OpenAPI/Proto/MCP/Workflow 摘要)
└── tasks.md                 # /speckit.tasks 阶段生成
```

```text
skeleton/backend/
├── internal/
│   ├── transport/
│   │   ├── http/
│   │   │   └── admin/capability/**
│   │   └── grpc/capability/**
│   ├── services/
│   │   ├── capability/**
│   │   └── capability_registry/**
│   ├── handlers/
│   │   └── capabilities/<domain>/<action>_handler.go
│   ├── domain/
│   │   ├── models/
│   │   └── repository/
│   └── observability/**
├── cmd/
│   └── server
skeleton/scripts/
└── capabilities/** (lint/export/compose)

skeleton/web-admin/
├── app/
│   ├── pages/capabilities/**
│   ├── components/capabilities/**
│   ├── stores/
│   └── types/
└── tests/

contracts/
├── exposure/openapi.yaml
├── exposure/proto/
├── exposure/workflow/
└── exposure/mcp-tools.json
```

**Structure Decision**: 采用「后端（Go）+ CLI（Node）+ Web-admin（Nuxt）」三层结构，全部落在 `skeleton/` 目录树：`skeleton/backend/internal/{transport,services,domain}`、`skeleton/scripts/capabilities`、`skeleton/web-admin/app/**`，并结合仓库根下的 `contracts/` 与 `capabilities/` 目录用于声明、生成与同步。

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| _None_ | | |

## Add-on Scope：模板基础能力扩展

为保证 demo 模型具备更贴近真实业务的“模板治理”能力，本迭代新增两条原子能力 + 一个 Workflow：

1. **模板批量克隆/导入（`com.powerx.plugins.base.template.batch_clone`）**
   - HTTP（能力出口）：`POST /api/v1/templates/batch-clone`，Body 包含源模板 ID 列表、克隆数量、目标标签/租户；返回 `created_ids`、`failed` 列表。
   - HTTP（Admin 业务面）：在 `skeleton/backend/internal/transport/http/admin/templates` 下新增 `/admin/api/templates/batch-clone`（或同等路径）供 Web Admin 调用，协议字段可更丰富（含草稿备注/审批人），但内部仍委托 `TemplateService.BatchClone`。
   - gRPC：`TemplateService/BatchCloneTemplates`，复用相同请求/响应结构。
   - Handler 位置：
     - 能力接口 handler：`.../transport/http/admin/templates/template_capability_handler.go`（新增文件）或同目录下新增方法，通过 `contracts/exposure/openapi.yaml` 暴露。
     - Admin 接口 handler：继续位于 `template_handler.go`，挂在 `/admin` 路由组。
     - 两者都依赖服务 `TemplateService.BatchClone`（内部封装事务、批量 insert、克隆标签/内容）。
   - 契约：新增 `contracts/schema/input|output/com.powerx.plugins.base.template.batch_clone.json` 与能力描述 `contracts/capabilities/com.powerx.plugins.base.template.batch_clone.yaml`，并在 `plugin.yaml` + `capabilities/catalog.json` 注册。
   - 暴露：REST/gRPC 必选，同时在 `contracts/exposure/workflow/`、`agent-streams/` 中声明，方便 Workflow/Agent 节点批量拉起模板。

2. **模板检测/校验（`com.powerx.plugins.base.template.validate`）**
   - HTTP（能力出口）：`POST /api/v1/templates/{id}/validate`，请求可指定规则集（lint/profile），响应返回 `violations[]`、`severity`、建议修复字段。
   - HTTP（Admin 业务）：在 `/admin/api/templates/{id}/validate` 追加管理端 API，供后台直接发起巡检任务或批量触发，接口形态可附带“保存记录/备注”字段。
   - gRPC：`TemplateService/ValidateTemplate`。
   - Handler：与批量克隆相同，拆分能力接口 handler（走开放 OpenAPI）与 Admin handler（内部业务）；Service：`TemplateService.Validate`（加载模板、运行校验器、返回结果）。
   - 套件：新增 `contracts/schema/input/output/com.powerx.plugins.base.template.validate.*` 与能力文件，SSE 事件用于推送校验完成通知。

3. **Workflow「模板巡检 + 批量分发」**
   - 文件：`contracts/exposure/workflow/template-quality-distribute.json`。
   - DAG：`list → validate (loop over violations) → batch_clone (生成多份修复版) → update/publish`，将巡检与批量克隆串联，展示“质量检测 + 分发”场景。
   - Agent Tool：`base.template.quality_distribute`（transport=workflow），SSE intent `template.quality_distribute`。
   - 依赖：上述两个原子能力完成后再导出 Workflow，并在文档中新增场景章节（`docs/guides/publish/capabilities/workflow-agent-guide.md`、`mcp-guide.md`）。

本 Add-on Scope 需要更新 contracts、handler/service、workflow、mcp 工具与文档，确保模板模型在 demo 中具备“CRUD + 批处理 + 质量巡检”三类能力以支撑 PowerX 意图识别测试。

## Additional Scope：插件本地调试模式

1. **调试模式切换**：在 `useCapabilityLab` 内新增 `invokeLocalCapability` 适配器，默认通过 REST/fetch 或 gRPC proxy 直接命中插件端口，仅在显式选择“宿主代理”时才回退到 `/api/v1/integration/capabilities/invoke`。
2. **协议映射**：根据表单的 REST/gRPC/Workflow 字段构造真实请求（URL、method、service/method、workflow template），无需依赖 Registry/Exposure 配置。
3. **错误处理**：将本地接口返回的 HTTP/gRPC 错误透传到调试面板，同时保留 TraceId/raw response 记录，确保体验与 `/powerx/capability-lab` 一致。
4. **文档与帮助**：更新 `docs/guides/develop/plugin-capability/README.md`、UI 文案与 i18n，强调“本地调试默认直连插件后端”。
