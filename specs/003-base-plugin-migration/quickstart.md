# Quickstart: Base Plugin Migration 验证指南

## 前提条件

- Go 1.24+，开启 `GOWORK=on`
- Node.js 18+、npm 9+
- 已完成 Phase 1~3 的实现并执行 `go work sync`
- 当前分支：`003-base-plugin-migration`

## 步骤 1：验证框架增强 (US1)

```bash
go test ./framework/backend/go/... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total
```

- 预期：Router、Response Helper、Middleware 测试全部通过且总语句覆盖率 ≥90%
- 若失败：检查 Path Param 解析、Response Envelope 字段或缺失的测试场景

## 步骤 2：启动 Skeleton 后端并执行 CRUD (US2)

```bash
go run ./skeleton/backend/go-gin/cmd/plugin &
```

在另一终端执行：

```bash
# 创建模板
curl -X POST http://localhost:8078/api/v1/templates \
  -H 'Content-Type: application/json' \
  -H 'X-PowerX-Tenant: 100' \
  -d '{"name":"Demo","description":"From Quickstart","content":"Hello"}'

# 查询当前租户列表
curl -H 'X-PowerX-Tenant: 100' http://localhost:8078/api/v1/templates

# 访问其他租户资源应返回 404
curl -H 'X-PowerX-Tenant: 200' http://localhost:8078/api/v1/templates/1

# 测量延迟（多次执行，确认 ≤1s）
curl -w 'TOTAL_TIME=%{time_total}\n' -o /dev/null -s \
  -H 'X-PowerX-Tenant: 100' http://localhost:8078/api/v1/templates
```

结束后停止后端进程。

## 步骤 3：前端 Skeleton 联调 (US3/US4)

```bash
cd skeleton/web-admin/nuxt
npm install
npm run dev
```

- 访问 `http://localhost:3031/`（首页）与 `http://localhost:3031/_p/com.powerx.sample/admin/templates/crud`
- 验证：创建/编辑/删除模板成功，并看到 Toast 提示，控制台无报错，网络请求命中 `/api/v1/templates`
- 运行 `POWERX_PROXY=1 npm run dev` 时，确认 `app.baseURL`、`runtimeConfig.public.apiBaseUrl` 切换至代理模式；语言包依旧正确加载

## 步骤 4：CLI 模板输出验证

```bash
cd /path/to/repo
px-plugin init com.powerx.demo --ui=starter --from templates/latest
```

然后在生成目录中运行：

```bash
go test ./backend/...
cd web-admin && npm install && npm run lint
```

- 对比生成项目与 Skeleton 差异（可使用 `git diff` 或 `diff -ru`）
- 预期：结构一致，仅插件 ID/名称等占位符不同

## 步骤 5：更新验证结果

- 在 `specs/003-base-plugin-migration/research.md` 追加验证日志（对应任务 T034/T035/T039）
- 如遇异常，记录修复方案并回补文档

> 按以上步骤即可验证 US1~US3 的核心能力，完成后再执行 Polish 阶段的文档更新。
