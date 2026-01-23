---
name: crud
description: PowerXPlugin CRUD 规范总入口技能，用于在实现/修改 CRUD 时选择正确子技能（模型/DTO/Service/Repository/HTTP/gRPC/测试/前端 Nuxt），并按 rulesets 执行检查与生成。
---

# CRUD 总入口

## 使用方式

1) 识别需求落点（后端/前端/测试/迁移/SDK/路由）。
2) 调用对应子技能并遵循其规则文件。
3) 需要跨层时可并行使用多个子技能，但避免把所有规则一次性加载。

## 子技能路由（按需选择）

- CRUD over HTTP：`crud-http` -> `.codex/skills/crud/http/SKILL.md`
- CRUD over gRPC：`crud-grpc` -> `.codex/skills/crud/grpc/SKILL.md`
- Frontend Admin 总规：`nuxt-admin` -> `.codex/skills/frontend/nuxt/admin/SKILL.md`
- STS 出站访问：`sts` -> `.codex/skills/sts/SKILL.md`

- 后端模型：`crud-model` -> `.codex/skills/crud/model/SKILL.md`
- DTO 与校验：`crud-dto` -> `.codex/skills/crud/dto/SKILL.md`
- Repository：`crud-repository` -> `.codex/skills/crud/repository/SKILL.md`
- Service：`crud-service` -> `.codex/skills/crud/service/SKILL.md`
- DI/容器：`crud-di` -> `.codex/skills/crud/di/SKILL.md`
- HTTP Handler：`crud-handler-http` -> `.codex/skills/crud/handler-http/SKILL.md`
- REST 约定：`crud-api-rest` -> `.codex/skills/crud/api-rest/SKILL.md`
- gRPC 传输：`crud-transport-grpc` -> `.codex/skills/crud/transport-grpc/SKILL.md`
- gRPC SDK 依赖：`crud-sdk-go` -> `.codex/skills/crud/sdk-go/SKILL.md`
- Migration：`crud-migration` -> `.codex/skills/crud/migration/SKILL.md`
- 测试：`crud-test` -> `.codex/skills/crud/test/SKILL.md`

- Nuxt API Client：`nuxt-api-client` -> `.codex/skills/frontend/nuxt/api-client/SKILL.md`
- Nuxt Components：`nuxt-components` -> `.codex/skills/frontend/nuxt/components/SKILL.md`
- Nuxt i18n：`nuxt-i18n` -> `.codex/skills/frontend/nuxt/i18n/SKILL.md`
- Nuxt Layout：`nuxt-layout` -> `.codex/skills/frontend/nuxt/layout/SKILL.md`
- Nuxt Pages：`nuxt-pages` -> `.codex/skills/frontend/nuxt/pages/SKILL.md`
- Nuxt Stores：`nuxt-stores` -> `.codex/skills/frontend/nuxt/stores/SKILL.md`
- Nuxt Tests：`nuxt-tests` -> `.codex/skills/frontend/nuxt/tests/SKILL.md`

## 约束

- 只加载需要的子技能，保持上下文最小化。
- 实现完成后按需运行 `npm test && npm run lint`。
