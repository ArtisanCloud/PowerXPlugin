# PowerX Publish Hub · 本地安装（Local Install）指南

本指南聚焦一件事：把**已经构建好的插件产物**安装到 PowerX 环境。

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
- `px-plugin pack`：在 `dist` 基础上生成 `.pxp` 元数据包（校验/签名用途）。  
  适用场景：需要签名、审计留痕、离线传输规范化。  
  不适用场景：只追求最快安装验证。
- `POST /admin/plugins/install/local`：把产物真正安装到 PowerX（本文核心）。  
  适用场景：联调环境/测试环境/预发布环境安装验证。  
  不适用场景：仅在开发机本地跑服务。

本文覆盖三种可用于安装的输入形态：

1. 构建后的 `dist/` 目录（解压即可运行）。
2. 手工压缩包（`.zip/.tar.gz` 等）。
3. `px-plugin pack` 生成的 `.pxp` 元数据包。

## 先选模式：你现在处于哪种场景

| 场景 | 推荐方式 | 为什么 |
| ---- | ---- | ---- |
| 日常开发联调（频繁改代码） | `px-plugin dev --watch` | 重点是“快速反馈”，不是部署；改代码后自动重载更高效。 |
| 提测/联调环境验证（要模拟真实安装） | `make dist` + `/admin/plugins/install/local` | 重点是“验证可安装产物”；与线上安装路径一致，能提前暴露打包问题。 |
| 内网/隔离网络交付 | 手工压缩包或拷贝 `dist/` | 重点是“可传输、可落地”；不依赖外网拉取。 |
| 需要审计/签名留痕 | `dist` + `.pxp`（`px-plugin pack`） | 重点是“可追溯”；`.pxp` 提供额外校验元数据。 |

一句话区分：
- `dev --watch`：开发模式（改代码快）。
- `local install`：部署模式（验安装真）。

> ⚠️ 与 `px-plugin dev --watch` 不同，本地安装是“部署动作”，依赖 PowerX Admin API 写入插件运行目录，不会热重载源码。
>
> 🚧 当前 `px-plugin dist/pack` 仍会打印 “experimental” 提示；如果你使用现有 CLI，请优先按本文的 `make dist` 或“手动构建”步骤执行。

---

## 适用场景

- 研发/测试团队需要在自建 PowerX 环境快速验证插件，而不希望走 Marketplace 提交流程。
- 运维需要在隔离网络中手动安装 `.pxp` 包（或 dist 目录）并立即启用。
- Marketplace 尚未接入，但 PowerX 的插件管理能力已经上线。

---

## 流程总览

- **方案 A（推荐）**：构建 dist → 将 `dist/<version>` 拷贝到 PowerX → 调 `POST /admin/plugins/install/local` 安装 → 验证。
- **方案 B（可选）**：在 dist 基础上额外生成 `.pxp` → 传输 `.pxp` → 在 PowerX 上解包成 dist 目录 → 再按方案 A 调用 Admin API。
- URL 模式（`/install/url`）和 `apiPrefix` 说明对两种方案都适用。

---

## 前置条件

| 项目 | 说明 |
| ---- | ---- |
| PowerX 版本 | 启用了 `backend/internal/transport/http/admin/plugin` 路由；`config/config.yaml` 里的 `server.apiPrefix` 默认为 `/api`，如改成 `/api/v1`，下面的示例 URL 也要同步。 |
| CLI & Toolchain | `px-plugin`、Node.js 18+/npm 9+、Go 1.24+、GNU Make。 |
| 权限 | 调用 Admin API 的 Token 需要 `platform_ops` / `plugin_admin` 权限，能访问 `/admin/plugins/**`。 |
| 服务器目录 | PowerX 需要能够读取你提供的 `src_dir`。通常把产物放在 `/srv/powerx/plugins/<id>/dist` 或 `/opt/powerx/uploads/<version>/`。 |

> 本文不负责 `px-plugin` 的构建/安装/初始化步骤。请先完成：
> - `docs/guides/develop/cli-plugin-tutorial.md`

环境变量示例：

```bash
export POWERX_ROOT=/private/var/www/html/ArtisanCloud/X/PowerX/Core/PowerX
export API_BASE="https://dev-api.powerx.local/api/v1"
export ADMIN_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

---

## 方案 A：dist 直装（推荐）

Local install 的核心是让 PowerX 读到一份可直接运行的 `dist/`：包含后端二进制、Nuxt 构建结果、`plugin.yaml`、`config/event_fabric.yaml`、`publish.yml`、`manifest.json` 等。**这一部分是所有流程都必须完成的基础**，可以用 Makefile 或手动方式完成。

## 本地安装时的 Topic 对齐要求

安装包必须同时满足两层：

1. 规范声明层：`plugin.yaml.events.topics[]`
2. 执行层：`config/event_fabric.yaml`（供 PowerX 底座启用插件时扫描播种）

底座行为说明：

- 底座启用插件时会扫描插件安装目录内的 `event_fabric.yaml` 并播种 topic/ACL。
- Topic 真相源是 `event_topics`。
- `POST /api/v1/internal/ws-bus/grant` 只做授权绑定，不创建 topic。

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
topics:
  - key: plugin.demo.order.created
    mode: publish
    description: order created event
  - key: plugin.demo.order.status
    mode: subscribe
    description: order status updates
```

建议：两层 topic 名称保持一一对应（`name`/`key` 一致），避免“声明里有、执行层没有”导致安装后链路不通。

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
```

如需额外生成 `.pxp`，请跳转到「方案 B · 步骤 2」使用 `make pack`。

`make dist` 当前默认执行 `go build`（本机平台）+ `nuxi build`，并将产物归档到 `dist/`；不会自动执行 `npm install`。若依赖未安装，请先按“步骤 1：准备依赖”完成安装。若你的项目尚未集成 Makefile，可先运行下方“手动构建”步骤再回到这里。

如果你的 PowerX 底座运行在 Linux，而你在 macOS/Windows 本地打包，请显式指定后端目标平台后再执行 `make dist`：

```bash
GOOS=linux GOARCH=amd64 make dist
```

### 步骤 3：手动构建（可选）

如果 CLI/Makefile 尚未完善，可直接在插件项目手工构建：

```bash
# 1) 编译后端
pushd backend
go mod tidy
GOOS=linux GOARCH=amd64 go build -o ../dist/backend/bin/plugin ./cmd/plugin
popd

# 2) 构建前端
pushd web-admin
npm install
npm run build
popd

# 3) 整理 dist 目录（包含 plugin.yaml、config/event_fabric.yaml、publish.yml、manifest.json 等）
mkdir -p dist/web-admin dist/backend
cp -R backend/dist/* dist/backend/
cp -R web-admin/.output/public dist/web-admin/
cp plugin.yaml dist/
mkdir -p dist/config
cp config/event_fabric.yaml dist/config/
cp publish.yml dist/
```

完成后即可进入“传输与安装”步骤。

---

## 方案 B：`.pxp` + dist 解包（可选）

如果你需要在 PowerX Marketplace 或离线审核链路中保留完整的签名/完整性信息，可以在完成 dist 之后再执行这一路径。`.pxp` **不会取代** dist，只是附加的校验材料；如果只想快速安装，可跳回方案 A。

### 步骤 1：使用 CLI 生成 `.pxp`（规划中）

Go 版本 CLI 已提供 `px-plugin dist`/`px-plugin pack` 子命令，但目前仍是 “experimental” 占位（见 `tools/cli/cmd/dist.go`, `package.go`）。Phase 14 将把 TypeScript 的打包逻辑迁移到 Go CLI，届时命令形态如下：

```
px-plugin dist \
  --manifest ./dist/manifest.json \
  --artefact ./dist/backend \
  --artefact ./dist/web-admin \
  --output-dir ./artifacts \
  --marketplace-public-key ./certs/marketplace.pem \
  --key-id marketplace-dev

px-plugin pack --manifest ./dist/manifest.json --artefact ./dist --output-dir ./artifacts
```

生成结果包含：
- `${pluginId}-${version}.pxp`
- `integrity.txt`
- `report.json`
- `manifest.signature`

### 步骤 2：使用 Makefile 生成 `.pxp`

当 Phase 14 的 Makefile 集成完成后，可直接运行：

```bash
make pack KEY_ID=marketplace-dev PUBLIC_KEY=./keys/marketplace.pem
```

脚本内部会调用 `px-plugin pack` 并把 `.pxp` 与配套校验文件放到 `dist/artifacts/`。运行该命令不会动 `dist/<version>/` 里的运行时代码，典型输出包括：

- `dist/artifacts/<pluginId>-<version>.pxp`
- `dist/artifacts/integrity.txt`
- `dist/artifacts/manifest.signature`
- `dist/artifacts/report.json`

> `.pxp` 只是带签名/校验信息的元数据包，不要把它塞进 `dist/<version>` 的运行目录；PowerX 仍然直接读取 `dist/<version>/backend`、`dist/<version>/web-admin` 等子目录。

#### 准备 `./keys/marketplace.pem`

`PUBLIC_KEY` 应指向一份 PEM（`-----BEGIN PUBLIC KEY-----` 或证书）文件，`px-plugin pack` 会使用它封装 `.pxp` 内部的对称密钥。通常把该文件放在仓库根目录 `keys/` 下，并通过 `.gitignore` 排除。

1. **使用官方 Marketplace 公钥（推荐）**
   ```bash
   mkdir -p keys
   cp ~/Downloads/marketplace-dev.pem ./keys/marketplace.pem
   ```
   拿到的是 `.crt` 时，可执行 `openssl x509 -pubkey -noout -in marketplace.crt > ./keys/marketplace.pem` 提取纯公钥。

2. **仅供本地验证的自签名公钥**
   ```bash
   mkdir -p keys
   openssl genrsa -out ./keys/marketplace-dev.key 2048
   openssl req -new -x509 -key ./keys/marketplace-dev.key \
     -subj "/CN=marketplace-dev" -days 365 \
     -out ./keys/marketplace-dev.crt
   openssl x509 -pubkey -noout -in ./keys/marketplace-dev.crt \
     > ./keys/marketplace.pem
   ```
   请勿把测试密钥用于生产；生产环境必须使用官方下发的公钥或受信任 CA 证书。

若本地环境尚未升级，可暂时跳过 `.pxp` 步骤，只保留 dist 即可完成 local install。

> 当前实现中 `.pxp` 仍是 JSON 元数据，不包含真实文件。无论是否生成 `.pxp`，PowerX 都必须能读到 dist 目录或已解包的文件。

---

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

- `src_dir`：PowerX 服务器可读的绝对路径。应包含 `backend/`, `web-admin/`, `plugin.yaml`, `config/event_fabric.yaml`, `publish.yml` 等文件。
- `enable`：安装完成后是否立即启用并切换成当前版本。
- `force`：若已有相同版本，是否强制覆盖。

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
- `.pxp` 目前仍需在服务器侧解包后才能运行；此模式主要方便上传存档，安装逻辑仍复用 `InstallFromFile`。

### 步骤 7：关于 `apiPrefix`

`backend/etc/config.yaml`（或相应环境的配置）中的 `server.apiPrefix` 决定了最终 URL：

- 默认：`server.apiPrefix: /api`，API 地址为 `https://<host>/api/admin/plugins/install/local`。
- 如果设为 `/api/v1`，需要改成 `https://<host>/api/v1/admin/plugins/install/local`。`px-plugin dev --watch` 的 Dev API 也需保持一致前缀。

---

### 步骤 3：传输 `.pxp`

```bash
aws s3 cp dist/artifacts/com.powerx.helloworld-0.1.0.pxp s3://internal-artifacts/
# 或
scp dist/artifacts/com.powerx.helloworld-0.1.0.pxp powerx-admin:/opt/powerx/uploads/
```

### 步骤 4：在 PowerX 解包并调用 local install

```bash
make local-install-pxp \
  PACKAGE=./dist/artifacts/com.powerx.helloworld-0.1.0.pxp \
  API_BASE=https://dev-api.powerx.local/api/v1 \
  TOKEN=eyJhbGciOi...
```

或手工：

```bash
mkdir -p /tmp/pxp && unzip com.powerx.helloworld-0.1.0.pxp -d /tmp/pxp
curl -X POST "$API_BASE/admin/plugins/install/local" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
        "src_dir": "/tmp/pxp/com.powerx.helloworld-0.1.0",
        "enable": false,
        "force": false
      }'
```

> 解包后若目录结构不同，请确保 `src_dir` 指向含有 `plugin.yaml`/`backend`/`web-admin` 的根，随后重复方案 A 的验证/启用步骤。

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
| `.pxp` 无法直接运行 | 目前 `.pxp` 只包含 manifest/integrity/audit 信息，仍需解包后把 dist 目录传给 `install/local`。后续计划由 PowerX 直接解析 `.pxp`。 |
| 与 `px-plugin dev --watch` 混淆 | `dev --watch` 通过 Dev API 注册临时会话，关闭终端即销毁。Local install 则写入正式的插件版本，需显式卸载或切换版本。 |

---

## 7. 下一步

- 完成本地安装后，建议回到《[离线发布指南](./offline.md)》了解 `.pxp` 在 Marketplace 审核链路中的用法。
- 如需自动化，可参考即将上线的 `make local-install`、`make local-install-pxp` 目标，把上述 curl 调用写入 CI/CD。完成后请在 `docs/guides/publish/online.md` 记录你的实践经验。 
