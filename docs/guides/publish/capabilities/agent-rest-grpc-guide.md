# Agent 能力（REST/gRPC）联调手册

面向需要在本地完成 REST / gRPC 调试并让 PowerX Agent 调用的开发者。以 `template` 能力为例，介绍如何启动环境、获取 Token、测试 OpenAPI/gRPC，以及如何把同一条能力同步到 PowerX。

## 1. 前置环境

- Go 1.24+、Node.js 18+、npm 9+
- 在仓库根目录执行：
  ```bash
  cd skeleton
  npm install
  ```
- 进入 `skeleton/backend` 并根据需要设置数据库：
  ```bash
  export POWERX_PROXY=0
  export IAM_MODE=local
  export PLUGIN_IAM_TENANT_KEY=00000000-0000-0000-0000-000000000001
  export PLUGIN_IAM_TENANT_NAME="Local Tenant"
  export PLUGIN_IAM_ADMIN_EMAIL=admin@local.test
  export PLUGIN_IAM_ADMIN_PASSWORD='S3cret!!'
  go run ./cmd/database/main.go setup   # 初始化表 + 默认租户/管理员
  ```

## 2. 启动本地服务

```bash
cd skeleton/backend
POWERX_PROXY=0 IAM_MODE=local go run ./cmd/plugin
```

- 默认 HTTP 监听 `:8178`，gRPC 监听 `:9101`（若端口被占用会自动递增）。
- DevSwitch 会自动注入示例租户 `00000000-0000-0000-0000-000000000001`，若需要自定义，覆盖 `PLUGIN_IAM_TENANT_KEY` 后重新 `setup`。

## 3. 获取 Access Token

1. 登录：
   ```bash
   curl -X POST http://127.0.0.1:8178/api/v1/admin/user/auth/login \
     -H 'Content-Type: application/json' \
     -d '{
           "tenant": "Local Tenant",
           "identifier": "admin@local.test",
           "password": "S3cret!!"
         }'
   ```
2. 记录响应中的 `access_token` 与 `refresh_token`。
3. 所有 REST/gRPC 请求都需要 `Authorization: Bearer <access_token>`，并确保 `tenant_uuid` 字段填写为你的租户 ID。

## 4. REST (OpenAPI) 调试

- OpenAPI 文档位于 `contracts/exposure/openapi.yaml`，可导入 Postman/Insomnia。
- 常见测试命令：

  ```bash
  # 查询模板列表
  curl http://127.0.0.1:8178/api/v1/templates \
    -H 'Authorization: Bearer <token>'

  # 创建模板
  curl -X POST http://127.0.0.1:8178/api/v1/templates \
    -H 'Authorization: Bearer <token>' \
    -H 'Content-Type: application/json' \
    -d '{
          "name": "demo",
          "description": "样板",
          "content": "Hello PowerX"
        }'

  # 批量克隆模板
  curl -X POST http://127.0.0.1:8178/api/v1/templates/batch-clone \
    -H 'Authorization: Bearer <token>' \
    -H 'Content-Type: application/json' \
    -d '{
          "source_ids": [1,2],
          "copies": 2,
          "name_prefix": "QA"
        }'

  # 校验模板
  curl -X POST http://127.0.0.1:8178/api/v1/templates/1/validate \
    -H 'Authorization: Bearer <token>' \
    -H 'Content-Type: application/json' \
    -d '{
          "rules": ["name_not_empty", "content_min_length"],
          "strict": true
        }'
  ```

- 以上接口均由 `backend/internal/transport/http/admin/templates` 暴露，且已经通过 `// capability: com.powerx.plugins.base.template.*` 注解写入能力目录。

## 5. gRPC 调试

1. 生成/同步 Proto：
   ```bash
   cd skeleton
   npm --prefix scripts/capabilities run export
   # 会刷新 contracts/exposure/proto/template.proto
   ```
2. 使用 `grpcurl` 调用：
   ```bash
   grpcurl -plaintext -d '{
       "tenant_uuid": "00000000-0000-0000-0000-000000000001",
       "page": 1,
       "page_size": 20
     }' \
     -H "authorization: Bearer <token>" \
     127.0.0.1:9101 powerx.template.TemplateService/ListTemplates
   ```
   - 其他 RPC：`GetTemplate`、`CreateTemplate`、`UpdateTemplate`、`DeleteTemplate`，请求体对应 `contracts/exposure/proto/template.proto`。
   - 新增 RPC：`BatchCloneTemplates`、`ValidateTemplate`，需要携带 `source_ids/copies` 或 `id/rules` 字段，可复用同一个 `tenant_uuid`。
   - 如果你的 gRPC 端口因冲突被自动递增，可在启动日志中查看实际端口，或通过 `POWERX_GRPC_SERVER_PORT` 明确配置。

## 6. 同步至 PowerX Catalog

1. 在 `skeleton` 目录执行：
   ```bash
   node ../scripts/capabilities/discover-handlers.mjs --plugin . --handlers backend/internal/transport/http/admin/templates --output tmp/template-capabilities.json
   npx --yes tsx ../tools/cli/src/commands/capabilities/init.ts --manifest ./plugin.yaml --batch tmp/template-capabilities.json
   npm --prefix scripts/capabilities run export
   ```
2. 将更新后的 `capabilities/`、`contracts/` 提交至仓库。

## 7. PowerX 中的调用路径

1. 插件安装到 PowerX 后，通过 `px-plugin capabilities submit` 与 `px-plugin capabilities quota` 把能力目录与暴露资产同步到宿主：
   ```bash
   cd skeleton
   npx --yes tsx ../tools/cli/src/commands/capabilities/submit.ts --manifest ./plugin.yaml --base-url $PX_DEV_API_BASE --token $PX_DEV_API_TOKEN
   npx --yes tsx ../tools/cli/src/commands/capabilities/quota.ts --capability-id com.powerx.plugins.base.template.create --tenant sandbox --base-url $PX_DEV_API_BASE --token $PX_DEV_API_TOKEN
   ```
2. PowerX Workflow Builder / Agent Hub 会读取 `contracts/exposure/openapi.yaml` 与 `contracts/exposure/proto/*.proto`，自动在 Gateway 中映射：
   - REST 统一走宿主的 `/internal/plugins/<plugin-id>/api/v1/templates`，宿主会注入租户上下文与 JWT。
   - gRPC 通过宿主的 Gateway → 插件 gRPC Server（默认 TLS 内网通信），租户信息同样由宿主注入。
3. 插件侧无需额外改动，只要保持能力 ID 与 `metadata.protocols.rest/grpc` 正确，宿主即可按同样的 URL/Service 名调用。

> 进阶：如需扩展更多模型或在 REST 上新增自定义参数，可重复上述步骤并更新对应的 JSON Schema / Proto，然后重新执行 `export → submit`。
