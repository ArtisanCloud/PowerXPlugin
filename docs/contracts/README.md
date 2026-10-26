# Contracts

PowerXPlugin 使用 JSON Schema 和 OpenAPI 描述插件与宿主之间的契约。

- `manifest.json`：约束插件 Manifest 字段、菜单路径与权限命名规则。
- `manifest.yaml`：示例 Manifest 输出，包含所需 IAM Scope。
- `rbac.json`：约束 RBAC 权限声明的格式。
- `rbac.schema.json`：示例 RBAC 报告，声明插件角色与所需宿主权限。
- `openapi.yaml`：记录框架保留端点与 skeleton 示例 API。
- `customer-auth.openapi.yaml`：Customer Identity/Auth 稳定合同，描述 C 端 customer token、tenant、membership、bootstrap 与 delegated auth 错误语义。
- [IAM 连续授权基线跨仓实施指导](../../../../PowerX/docs/guides/features/iam-authorization-baseline/guide.md)：Core 权威正文，定义通用批量读取、严格结果校验、初始化提示下界和三方任务。Framework F01–F04 消费入口与验证边界见 [Framework 接入记录](iam-authorization-baseline-framework.md)；插件 v2、正式授权和安装态验收仍待完成。
- [Media 标准图片预览 Framework 对齐](media-image-derivatives-framework.md)：Go/.NET 可选 `derivatives` 响应映射、原图与预览状态区分、插件读取流程，以及服务态通知和安装验收边界。

## 更新流程

1. 修改 Schema 或 OpenAPI 文件。
2. 更新 framework 代码，使 `manifest.Validate` / `rbac.Validate` 等校验逻辑与 Schema 对齐。
3. 运行 `cd framework/backend/go && go test ./manifest ./framework/backend/go/rbac` 验证校验器通过。
4. 提交 PR 时，CI (`.github/workflows/ci.yml`) 会自动运行 Go 测试与前端构建，确保契约变更未破坏实现。
