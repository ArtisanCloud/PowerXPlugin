# Quickstart: Framework IAM 统一封装（Standalone/Delegated）

## 1. 前置条件

1. 当前分支为 `018-framework-iam-unification`。
2. 已完成 US1-US3 实现，且 `go test` 可执行。
3. 已准备 local 与 delegated 两套配置。
4. 仓库清单路径已迁移到 `skeleton/plugin.yaml`（如需 CLI 验证请在 `skeleton/` 下执行，或显式传 `--manifest ./skeleton/plugin.yaml`）。

## 2. 插件接入指引（迁移到 framework IAM）

1. 业务代码只依赖 framework 契约：`DirectoryService`、`AuthzService`、`IdentityContextService`。
2. 启动时仅绑定一个 adapter（local 或 delegated），运行期不切换。
3. handler 层不要自行解析 token/tenant，统一走 framework middleware/context。
4. delegated 模式下组织写操作保持插件侧 `405`（只读边界）。

## 3. 核心验收场景

### 3.1 模式优先级与 fail-fast

1. 配置 `context.provider_mode=local`，同时设置 `POWERX_PROXY=1`。
2. 启动服务，预期启动失败并提示模式冲突。
3. 清理冲突后重启，访问 `GET /admin/iam/mode`，预期返回单一模式与来源。

### 3.2 单选绑定（不可热切换）

1. 以 `local` 启动并访问 `GET /admin/iam/mode`，记录 `mode=local`。
2. 不重启进程，仅修改环境变量为 delegated。
3. 再次访问接口，预期 `mode` 不变。

### 3.3 delegated 只读边界

1. 以 delegated 模式启动。
2. `GET /admin/iam/departments` 预期成功。
3. `POST /admin/iam/departments` 预期 `405`，错误码 `IAM_DELEGATED_READ_ONLY`。

### 3.4 token/context 统一解析

1. 构造合法 bearer token（含 `tid/sub/roles/permissions`）。
2. 请求受保护接口，预期 tenant/user/roles/permissions 被统一读取。
3. 构造非法 token 或缺失 tenant claim，预期 `401`（`IAM_UNAUTHORIZED`）。
4. 注入上游不可用场景，预期 `424`（`IAM_UPSTREAM_DEPENDENCY`）。

### 3.5 tenant 多来源优先级

1. 同时注入 context、Authorization、header、query 的 tenant。
2. 预期按 `context > token > header > query` 解析。
3. 若高优来源合法，低优来源冲突不覆盖最终 tenant。

## 4. 回归命令（018）

```bash
cd framework/backend/go && go test ./...
```

```bash
cd skeleton/backend/go-gin && \
  GOCACHE=$PWD/../../tmp/gocache \
  GOMODCACHE=$PWD/../../tmp/gomodcache \
  go test ./...
```

## 5. 可选发布前校验

```bash
cd skeleton
px-plugin capabilities validate --manifest ./plugin.yaml
```
