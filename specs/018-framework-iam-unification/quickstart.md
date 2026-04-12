# Quickstart: Framework IAM 统一封装（Standalone/Delegated）

## 1. 前置条件

1. 位于分支 `018-framework-iam-unification`。  
2. 已完成依赖安装并可执行 `go test`。  
3. 已有 local 与 delegated 两套运行配置。

## 2. 验证模式解析优先级

1. 设置配置文件 `context.iam_mode=local`，同时设置环境变量 `POWERX_PROXY=1`。  
2. 启动服务，预期：启动失败并提示模式冲突（fail-fast）。  
3. 移除冲突后重新启动，预期：服务正常，`/admin/iam/mode` 返回单一模式。

## 3. 验证 adapter 单选绑定

1. 以 `local` 模式启动。  
2. 请求 `/admin/iam/mode`，记录 `mode=local`。  
3. 在不重启进程情况下修改环境变量为 delegated，再请求接口。  
4. 预期：模式不变（仍为 local），运行期不自动切换。

## 4. 验证 delegated 读写边界

1. 以 delegated 模式启动。  
2. 调用 `GET /admin/iam/departments`，预期成功（只读投影/代理查询）。  
3. 调用 `POST /admin/iam/departments`，预期返回 `405`（写操作被拒绝）。

## 5. 验证 local 最小实体集

1. 以 local 模式启动并初始化 IAM 数据。  
2. 分别调用 tenants/departments/members/roles/permissions 查询接口。  
3. 预期：五类实体均可查询，且 tenant_uuid 语义一致。

## 6. 回归命令

```bash
go test ./framework/backend/go/... ./skeleton/backend/go-gin/... -count=1
```

可选完整回归：

```bash
make test-regression
```

## 7. Phase 1（Setup）回归入口

1. 在仓库根目录执行以下命令，确认 Phase 1 目录骨架已就位：

```bash
find framework/backend/go/iam -maxdepth 2 -type f | sort
```

2. 预期输出至少包含以下 4 个占位文件：
   - `framework/backend/go/iam/contracts/.gitkeep`
   - `framework/backend/go/iam/adapters/.gitkeep`
   - `framework/backend/go/iam/context/.gitkeep`
   - `framework/backend/go/iam/errors/.gitkeep`
