# PowerX Publish Hub · 本地安装（Local Install）指南

本指南聚焦一件事：把**已经构建好的插件产物**安装到 PowerX 环境。

市场发布规则统一见[开发者发布中心方案](../../marketplace/plugin-developer-publishing-center-plan.md)与[实现设计](../../marketplace/plugin-publisher-implementation.md)。本地安装不等于市场上传、审核或公开上架；新工具首期使用 dist/ZIP。Go `px-plugin dist` 当前只提示 experimental，当前 Go CLI 未注册 `pack` 子命令。

插件项目如何实现 `make dist`、`make local-install`、`make local-reinstall`，以及 `dist/<version>` 必须长什么样，请先看实现规范：[Makefile 任务与构建说明](../../standards/powerx-plugin/developer/makefile_tasks.md)。本文只说明如何把已经构建好的 dist 安装到 PowerX。

先区分几个常见命令（避免混淆）：

- `px-plugin init`：初始化插件工程（生成代码骨架）。  
  适用场景：新项目起步、需要快速拉起标准目录。  
  不适用场景：已有项目发布安装。
- `px-plugin dev --watch`：本地开发热更新（不做部署安装）。  
  适用场景：高频改代码、秒级验证页面/接口改动。  
  不适用场景：提测验收、模拟真实部署。
- `make dist` / 手动构建：生成可安装产物目录。  
  适用场景：准备提测包、要交给 PowerX 安装。  
  不适用场景：只做本地热更新调试。
- `make package`：构建 dist 后生成 ZIP，可用于传输已构建产物。
  注意该命令会重新构建，不代表市场上传或已有有效签名。
- `POST /admin/plugins/install/local`：把产物真正安装到 PowerX（本文核心）。  
  适用场景：联调环境/测试环境/预发布环境安装验证。  
  不适用场景：仅在开发机本地跑服务。

本文覆盖两种实际运行产物输入形态：

1. 构建后的 `dist/` 目录，包含完整运行文件。
2. 手工压缩包（`.zip/.tar.gz` 等）。

旧 TypeScript `pack` 生成的 JSON 元数据不是完整运行包，不列为当前安装输入。其他 `.pxp` 容器必须按内容及对应宿主契约单独核对，不能仅凭后缀判断。

## 先选模式：你现在处于哪种场景

| 场景 | 推荐方式 | 为什么 |
| ---- | ---- | ---- |
| 日常开发联调（频繁改代码） | `px-plugin dev --watch` | 重点是“快速反馈”，不是部署；改代码后自动重载更高效。 |
| 提测/联调环境验证（要模拟真实安装） | `make dist` + `/admin/plugins/install/local` | 重点是“验证可安装产物”；与线上安装路径一致，能提前暴露打包问题。 |
| 内网/隔离网络交付 | 手工压缩包或拷贝 `dist/` | 重点是“可传输、可落地”；不依赖外网拉取。 |
| 需要审计/签名留痕 | 保存最终包 SHA256、来源和实际签名证据 | 摘要、后缀或签名占位不代表来源已验证；按真实策略验签。 |

一句话区分：
- `dev --watch`：开发模式（改代码快）。
- `local install`：部署模式（验安装真）。

> ⚠️ 与 `px-plugin dev --watch` 不同，本地安装是“部署动作”，依赖 PowerX Admin API 写入插件运行目录，不会热重载源码。
>
> 当前 Go `px-plugin dist` 只输出 experimental 提示，Go CLI 没有 pack 子命令。构建运行产物请使用本文的 `make dist` 或手动构建步骤。

---

## 适用场景

- 研发/测试团队需要在自建 PowerX 环境快速验证插件，而不希望走 Marketplace 提交流程。
- 运维需要在隔离网络中传输实际 ZIP 或 dist 目录，并在宿主显式安装及启用。
- Marketplace 尚未接入，但 PowerX 的插件管理能力已经上线。

---

## 流程总览

- **方案 A（推荐）**：构建 dist → 将 `dist/<version>` 拷贝到 PowerX → 调 `POST /admin/plugins/install/local` 安装 → 验证。
- **传输安装包**：生成并传输实际 ZIP 等受支持容器，在宿主按对应安装接口处理；不把旧 JSON 元数据包当作可解压运行包。
- URL 模式（`/install/url`）和 `apiPrefix` 说明对两种方案都适用。

> 重要：`install/local` 成功不等于插件已可用。`switch_version` 仍会执行进程启动与健康检查；若启动阶段（如 manifest 注册、权限格式、迁移）失败，最终仍会启用失败。发版前请先执行 `docs/guides/publish/preflight-checklist.md`。

### PowerXPlugin 仓库内联调（推荐）

如果你正在 `PowerXPlugin` 主仓库里调 framework/skeleton，不必先 `px-plugin init` 新项目：

```bash
export POWERX_ROOT=/private/var/www/html/ArtisanCloud/X/PowerX/Core/PowerX
export PLUGIN_ROOT=/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin
export DIST_DIR=$PLUGIN_ROOT/skeleton/dist/0.7.1

cd $PLUGIN_ROOT

# 自动同步 plugin.d（capabilities/exposure/rbac）并校验映射
make manifest-align-fix

# CI 严格门禁建议使用（本地可省略）
make manifest-align-check

# 输出 skeleton 可安装包
make skeleton-dist

# 安装到 PowerX 宿主（会先构建 dist）
make skeleton-install \
  API_BASE=http://127.0.0.1:8077/api/v1 \
  TOKEN=<ADMIN_BEARER_TOKEN> \
  ENABLE=true \
  FORCE=true

# 一键重装并切换版本（disable -> force install -> switch_version enable）
make skeleton-reinstall \
  VERSION=0.7.1 \
  API_BASE=http://127.0.0.1:8077/api/v1 \
  TOKEN=<ADMIN_BEARER_TOKEN>
```

说明：
- `make manifest-align-fix` 会先同步 `skeleton/plugin.d/*.yaml`，再校验 capability→exposure/rbac 映射（内部已包含基础 plugin.yaml 校验链路）
- `make skeleton-dist` = `make -C skeleton dist`
- `make skeleton-install` = `make -C skeleton local-install ...`
- `make skeleton-reinstall` = `make -C skeleton local-reinstall ...`
- skeleton 默认插件标识来自 `skeleton/plugin.yaml`（当前示例：`id: com.powerx.plugins.base`）
- `ENABLE=true` 表示安装后立即启用；`FORCE=true` 表示同版本强制覆盖/重装

若你已经有现成的 dist 目录（不想重复构建），可直接走 `local-install-run`：

```bash
export POWERX_ROOT=/private/var/www/html/ArtisanCloud/X/PowerX/Core/PowerX
export PLUGIN_ROOT=/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin
export DIST_DIR=$PLUGIN_ROOT/skeleton/dist/0.7.1

cd $PLUGIN_ROOT/skeleton

make local-install-run \
  LOCAL_INSTALL_SRC=$DIST_DIR \
  API_BASE=http://127.0.0.1:8077/api/v1 \
  TOKEN=<ADMIN_BEARER_TOKEN> \
  FORCE=true
```

说明：`local-install` 会先执行 `dist`；`local-install-run` 只安装指定目录。
注意：仅“安装成功”不等于“插件一定重启为新进程”；请同时观察返回体里的启用状态与宿主 lifecycle 日志。

---

## 前置条件

| 项目 | 说明 |
| ---- | ---- |
| PowerX 版本 | 启用了 `backend/internal/transport/http/admin/plugin` 路由；`config/config.yaml` 里的 `server.apiPrefix` 默认为 `/api`，如改成 `/api/v1`，下面的示例 URL 也要同步。 |
| CLI & Toolchain | `px-plugin`、Node.js 18+/npm 9+、Go 1.24+、GNU Make。 |
| 权限 | 调用 Admin API 的 Token 需要 `platform_ops` / `plugin_admin` 权限，能访问 `/admin/plugins/**`。 |
| 服务器目录 | PowerX 需要能够读取你提供的 `src_dir`。通常把产物放在 `/srv/powerx/plugins/<id>/dist` 或 `/opt/powerx/uploads/<version>/`。 |

> 本文不负责 `px-plugin` 的构建/安装/初始化步骤。请先完成：
> - `docs/guides/develop/cli-plugin/cli-plugin-tutorial.md`

环境变量示例：

```bash
export POWERX_ROOT=/private/var/www/html/ArtisanCloud/X/PowerX/Core/PowerX
export API_BASE="https://dev-api.powerx.local/api/v1"
export ADMIN_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

---

## 方案 A：dist 直装（推荐）

Local install 的核心是让 PowerX 读到一份可直接运行的 `dist/<version>/`：包含后端二进制、Nuxt 构建结果、`plugin.yaml`、`plugin.d/*.yaml`、`config/event_fabric.yaml` 等。**这一部分是所有流程都必须完成的基础**，可以用 Makefile 或手动方式完成。

## 本地安装时的 Topic 对齐要求

安装包必须同时满足两层：

1. 规范声明层：`plugin.d/events.yaml` 的 `events.topics[]`（由 `plugin.yaml.catalogs.events` 引用）
2. 执行层：`config/event_fabric.yaml`（供 PowerX 底座启用插件时扫描播种）

底座行为说明：

- 底座启用插件时会扫描插件安装目录内的 `event_fabric.yaml` 并播种 topic/ACL。
- Topic 真相源是 `event_topics`。
- `POST /api/v1/admin/runtime/ws-bus/grant` 只做授权绑定，不创建 topic。

联调顺序（Standalone + Proxy）：

1. ensure topic 已注册到 `event_topics`
2. 配置 API Key Profile 权限并保存
3. 轮换/新建 API Key
4. 调用 `ws-bus/grant`
5. 调用 `publish/subscribe`

### 最小配置示例（必须两层都写）

`plugin.yaml`（声明层，给审核/注册与能力语义使用）：

```yaml
events:
  topics:
    - name: plugin.demo.order.created
      direction: publish
      desc: order created event
    - name: plugin.demo.order.status
      direction: subscribe
      desc: order status updates
```

`config/event_fabric.yaml`（执行层，给底座启用时播种使用）：

```yaml
version: v1
topics:
  - topic: plugin.demo.order.created
    description: order created event
    acl:
      - actions: [publish]
  - topic: plugin.demo.order.status
    description: order status updates
    acl:
      - actions: [subscribe]
```

建议：两层 topic 名称保持一一对应（`name`/`topic` 一致），避免“声明里有、执行层没有”导致安装后链路不通。

### 联调动作拆解（可直接照做）

1) **确认 topic 已存在于底座**

- 通过底座管理接口或后台确认 `event_topics` 中已有目标 topic。
- 若不存在，先创建 topic，再做后续授权。

2) **配置 API Key Profile 权限**

- 在底座将调用方主体（API Key 对应 profile）授权到目标 topic（publish/subscribe）。
- 保存后轮换或新建 API Key，确保权限快照生效。

3) **调用 grant（只绑定，不创建）**

```bash
curl -X POST "$API_BASE/internal/ws-bus/grant" \
  -H "Authorization: ApiKey $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "topics": ["plugin.demo.order.created", "plugin.demo.order.status"]
  }'
```

4) **验证 publish / subscribe**

- publish 端发送消息到 `plugin.demo.order.created`
- subscribe 端订阅 `plugin.demo.order.status` 并检查是否收到

### 典型误区

- 只配 `plugin.yaml.events.topics[]`，没配 `config/event_fabric.yaml`：安装后底座不会播种执行层 topic。
- `config/event_fabric.yaml` 缺少顶层 `version: v1`：启用阶段会失败，错误通常是 `manifest version must be positive`。
- 只调 `ws-bus/grant` 就以为会自动建 topic：不会，grant 仅做授权绑定。
- topic 名称两层不一致：表现为授权成功但消息链路不通。

### 步骤 1：准备依赖

在执行 `make dist` 之前，请先在插件项目根目录安装好后端与前端依赖，否则 Nuxt 构建阶段会尝试下载临时包，导致版本不一致或出现 “Cannot find module 'nuxt/config'” 之类的错误。

```bash
cd /path/to/your-plugin        # 例如 plugins/com.powerx.helloworld

# Go 依赖
# 若项目未启用 go.work，可忽略 go work sync 的提示
go work sync 2>/dev/null || true
cd backend
go mod tidy
cd ..

# Web Admin 依赖
cd web-admin
npm install
cd ..
```

若之前在 web-admin 目录执行过 `npm install` 但后来清理了 `node_modules/`，请重新安装，确保 `web-admin/node_modules/.bin/nuxi` 与 `nuxt/config` 均来自模板声明的 Nuxt 4.2.x。

### 步骤 2：使用脚手架 Makefile 构建 dist

Phase 14 合并后，`px-plugin init` 会自动把 `skeleton/Makefile` 与 `make-files/` 带入新项目。完成上一步依赖安装后，即可直接运行；若需要覆盖默认版本号，可一并在命令前设置 `VERSION=`：

```bash
# 构建后端 + 前端，输出到 dist/
make dist

# 构建特定版本
VERSION=0.2.0 make dist

# 本地 macOS 调试常用：输出到固定目录，便于 PowerX install/local 选择
make dist VERSION=0.1.0 DIST_DIR=dist/mac
```

如需压缩传输，可使用 `make package` 生成 ZIP；该命令会重新构建。新发布工具对冻结目录压缩一次的能力尚待实现。

`make dist` 当前默认执行 `go build`（本机平台）+ `nuxi build`，并将产物归档到 `dist/`；不会自动执行 `npm install`。若依赖未安装，请先按“步骤 1：准备依赖”完成安装。若你的项目尚未集成 Makefile，可先运行下方“手动构建”步骤再回到这里。

如果你的 PowerX 底座运行在 Linux，而你在 macOS/Windows 本地打包，请显式指定后端目标平台后再执行 `make dist`：

```bash
make dist PLATFORM=linux TARGET_ARCH=amd64
```

也可使用兼容别名：

```bash
make dist-linux
```

### 步骤 3：手动构建（可选）

如果 CLI/Makefile 尚未完善，可直接在插件项目手工构建：

```bash
# 1) 编译后端
pushd backend
go mod tidy
GOOS=linux GOARCH=amd64 go build -o ../dist/backend/bin/plugin ./cmd/plugin
if [ -d cmd/database ]; then
  GOOS=linux GOARCH=amd64 go build -o ../dist/backend/bin/migrate ./cmd/database
fi
popd

# 2) 构建前端
pushd web-admin
npm install
npm run build
popd

# 3) 整理 dist 目录（包含 plugin.yaml、plugin.d、config/event_fabric.yaml 等）
mkdir -p dist/web-admin/.output dist/backend/bin
cp backend/bin/plugin dist/backend/bin/
if [ -f backend/bin/migrate ]; then
  cp backend/bin/migrate dist/backend/bin/
fi
cp -R web-admin/.output/. dist/web-admin/.output/
cp plugin.yaml dist/
cp -R plugin.d dist/
mkdir -p dist/config
cp config/event_fabric.yaml dist/config/
```

完成后即可进入“传输与安装”步骤。

---

## 传输包与旧元数据示例边界

旧文档中的 `px-plugin pack`、PEM 封装及 JSON `.pxp` 示例不作为当前 Go CLI 操作步骤。Go CLI 未注册 pack 命令，Go dist 仅有 experimental 提示；Makefile 的旧 pack 规则不能代替真实文件容器、正式签名或市场上传。

本指南使用已经构建好的 dist 目录，或对应宿主明确支持的实际 ZIP/TAR 文件。包必须包含运行所需文件，接收方复算摘要并核对 manifest 入口；不能对 JSON 元数据执行 unzip 并宣称得到运行时。

## 已构建产物的传输与宿主安装

### 步骤 4：传输 dist 目录（可选）

如果 PowerX Admin API 与打包机部署在同一节点，可直接在本地路径上执行安装，跳过本步骤。只有在 PowerX 运行于远程服务器、需要把 dist 搬过去时再执行以下操作。

dist 输出位于 `dist/<version>/...`，可将整个版本目录同步到 PowerX 服务器，例如：

```bash
DIST_VERSION=0.1.0
rsync -avz dist/$DIST_VERSION \
  powerx-admin:/srv/powerx/plugins/com.powerx.helloworld/dist/$DIST_VERSION
```

也可以先压缩再上传：

```bash
cd dist && zip -r ../com.powerx.helloworld-$DIST_VERSION.zip $DIST_VERSION
aws s3 cp ../com.powerx.helloworld-$DIST_VERSION.zip s3://internal-artifacts/
```

到服务器后解压至目标目录（例如 `/srv/powerx/plugins/com.powerx.helloworld/dist/0.1.0`）。

### 步骤 5：调用 `/admin/plugins/install/local`

不论 dist 在本机还是远程服务器，最终都需要调用 Admin API 告诉 PowerX 去读取那个目录。

- **本地环境**：`src_dir` 填写绝对路径（例如 `$(pwd)/dist/0.1.0`）。
- **远程环境**：`src_dir` 填远程服务器上的同步路径（配合上方步骤 4）。

本地示例：

```bash
DIST_VERSION=0.1.0
curl -X POST "$API_BASE/admin/plugins/install/local" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
        \"src_dir\": \"$(pwd)/dist/$DIST_VERSION\",
        \"enable\": true,
        \"force\": false
      }"
```

远程示例：

```bash
curl -X POST "$API_BASE/admin/plugins/install/local" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
        "src_dir": "/srv/powerx/plugins/com.powerx.helloworld/dist/0.1.0",
        "enable": true,
        "force": false
      }'
```

- `src_dir`：PowerX 服务器可读的绝对路径。应包含 `backend/bin/plugin`, `web-admin/.output`, `plugin.yaml`, `plugin.d/`, `config/event_fabric.yaml` 等文件；若声明 migrations，还应包含 `backend/bin/migrate`。
- `enable`：安装完成后是否立即启用并切换成当前版本。
- `force`：若已有相同版本，是否强制覆盖。

> 关键语义：`install/local` 的核心是“安装产物”。若需要立即切换运行态，请显式传 `enable=true`；若要覆盖同版本并触发重装，请再加 `force=true`。

成功响应示例：

```json
{
  "installed": {
    "id": "com.powerx.helloworld",
    "version": "0.1.0",
    "state": "installed"
  }
}
```

### 步骤 6：URL 模式（可选）

用于 zip/pxp 已上传到可访问的 HTTP/S 位置：

```bash
curl -X POST "$API_BASE/admin/plugins/install/url" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
        "url": "https://upload.local/com.powerx.helloworld-0.1.0.zip",
        "sha256": "4f83c4a8...",
        "enable": false
      }'
```

- `sha256` 可选，提供时 PowerX 会校验包体完整性。
- 容器格式及 URL 安装能力以目标宿主实际支持为准；旧 JSON 元数据不能作为可解压安装包。

### 步骤 7：关于 `apiPrefix`

`backend/etc/config.yaml`（或相应环境的配置）中的 `server.apiPrefix` 决定了最终 URL：

- 默认：`server.apiPrefix: /api`，API 地址为 `https://<host>/api/admin/plugins/install/local`。
- 如果设为 `/api/v1`，需要改成 `https://<host>/api/v1/admin/plugins/install/local`。`px-plugin dev --watch` 的 Dev API 也需保持一致前缀。

---

## 安装后的验证

```bash
# 查看插件列表
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$API_BASE/admin/plugins" | jq

# 查看单个插件状态
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$API_BASE/admin/plugins/com.powerx.helloworld/status" | jq

# 启动/停止
curl -X POST -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$API_BASE/admin/plugins/com.powerx.helloworld/enable"

# 查看日志
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$API_BASE/admin/plugins/com.powerx.helloworld/logs?tail=200"
```

> 若需要切换版本，可使用 `POST /admin/plugins/:id/switch_version`，请求体 `{"version": "0.1.0", "enable": true}`。

---

## 6. 常见问题 / 故障排查

| 问题 | 说明与解决办法 |
| ---- | --------------- |
| `install/local` 返回 404 | 检查 `server.apiPrefix` 是否与请求一致；确认 Admin API 路由已注册。 |
| `src_dir` 找不到 | 目标目录必须存在于 PowerX 服务器本地磁盘；如果你在开发机上执行 curl，需要通过 `ssh` 或 CI 让命令运行在服务器上。 |
| 版本已存在 | 若只是想覆盖当前版本，传 `force=true`；否则建议 bump `plugin.yaml` 里的 `version`。 |
| `lifecycle_error: enable: already_enabled` | 说明旧版本仍是 enabled。先用 `ENABLE=false FORCE=true` 覆盖安装，再调用版本切换/启用接口。 |
| 想“一条命令重装并启用新版本” | 使用 `make skeleton-reinstall VERSION=<target>`（内部执行 disable -> force install(enable=false) -> switch_version(enable=true)）。 |
| 包无法解压或缺少运行文件 | 核对实际容器和 manifest 入口；旧 JSON 元数据不能代替完整 dist/ZIP，后缀不作为格式证明。 |
| 与 `px-plugin dev --watch` 混淆 | `dev --watch` 通过 Dev API 注册临时会话，关闭终端即销毁。Local install 则写入正式的插件版本，需显式卸载或切换版本。 |

---

## 7. 下一步

- 离线产物和市场发布边界见[离线产物指南](offline.md)，真实上传及审核见[在线发布指南](online.md)。
- 如需自动化，可参考即将上线的 `make local-install`、`make local-install-pxp` 目标，把上述 curl 调用写入 CI/CD。完成后请在 `docs/guides/publish/online.md` 记录你的实践经验。 
