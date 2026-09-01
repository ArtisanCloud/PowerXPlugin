# Quickstart: Framework IAM 统一封装（Standalone/Delegated）

## 1. 前置条件

1. 当前分支为 `018-framework-iam-unification`。
2. 本特性的 production Directory adapter 尚未交付；本指南定义实现完成后的接入和验收标准，不能作为“当前已可联调”的证明。
3. 已准备 local 与 delegated 两套配置。
4. 仓库清单路径已迁移到 `skeleton/plugin.yaml`（如需 CLI 验证请在 `skeleton/` 下执行，或显式传 `--manifest ./skeleton/plugin.yaml`）。

## 2. 插件接入指引（迁移到 framework IAM）

1. 业务代码只能从 `IAMRegistry` 获取 `DirectoryService`、`AuthzService`、`IdentityContextService`；不得查本地 IAM 表、调用 Core 内部 service 或解析 Gateway 原始对象。
2. 启动时仅绑定一个 production adapter（local 或 delegated），运行期不切换；未绑定必须明确失败。
3. handler 层不要自行解析 token/tenant，统一走 framework middleware/context。
4. delegated 模式下组织写操作保持插件侧 `405`（只读边界）。
5. 按成员 UUID 查询使用 `GetMember` 或 `BatchGetMembers`。解析不到名称时返回空值或明确错误，绝不显示 UUID。

## 3. 核心验收场景

### 3.1 模式优先级与 fail-fast

1. 配置 `context.provider_mode=local`，同时设置 `POWERX_PROXY=1`。
2. 启动服务，预期启动失败并提示模式冲突。
3. 清理冲突后重启，访问 `GET /admin/iam/mode`，预期返回单一模式与来源。

### 3.2 单选绑定（不可热切换）

1. 以 `local` 启动并访问 `GET /admin/iam/mode`，记录 `mode=local`。
2. 不重启进程，仅修改环境变量为 delegated。
3. 再次访问接口，预期 `mode` 不变。

### 3.3 Directory 单成员与批量查询

1. 以同一 `tenant_uuid` 查询已存在的 `member_uuid`，预期返回相同 `member_uuid` 和人类可读 `display_name`。
2. 批量查询去重后的成员 UUID，预期结果可按 `member_uuid` 关联，不混用 `user_uuid`。
3. 查询不存在成员，预期 `IAM_MEMBER_NOT_FOUND`；不得返回 UUID 作为显示名。
4. 注入 delegated 上游不可用，预期 `424`（`IAM_UPSTREAM_DEPENDENCY`），不得返回空列表伪装成功。

### 3.4 delegated 只读边界

1. 以 delegated 模式启动。
2. `GET /admin/iam/departments` 预期成功。
3. `POST /admin/iam/departments` 预期 `405`，错误码 `IAM_DELEGATED_READ_ONLY`。

### 3.5 token/context 统一解析

1. 构造合法 bearer token（含 `tid/sub/roles/permissions`）。
2. 请求受保护接口，预期 tenant/user/roles/permissions 被统一读取。
3. 构造非法 token 或缺失 tenant claim，预期 `401`（`IAM_UNAUTHORIZED`）。
4. 注入上游不可用场景，预期 `424`（`IAM_UPSTREAM_DEPENDENCY`）。

### 3.6 tenant 多来源优先级

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
