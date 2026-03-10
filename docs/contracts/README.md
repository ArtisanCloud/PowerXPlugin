# Contracts

PowerXPlugin 使用 JSON Schema 和 OpenAPI 描述插件与宿主之间的契约。

- `manifest.json`：约束插件 Manifest 字段、菜单路径与权限命名规则。
- `manifest.yaml`：示例 Manifest 输出，包含所需 IAM Scope。
- `rbac.json`：约束 RBAC 权限声明的格式。
- `rbac.schema.json`：示例 RBAC 报告，声明插件角色与所需宿主权限。
- `openapi.yaml`：记录框架保留端点与 skeleton 示例 API。

## 更新流程

1. 修改 Schema 或 OpenAPI 文件。
2. 更新 framework 代码，使 `manifest.Validate` / `rbac.Validate` 等校验逻辑与 Schema 对齐。
3. 运行 `cd framework/backend/go && go test ./manifest ./framework/backend/go/rbac` 验证校验器通过。
4. 提交 PR 时，CI (`.github/workflows/ci.yml`) 会自动运行 Go 测试与前端构建，确保契约变更未破坏实现。
