# Host Contract Lab 本地验证记录

更新时间：2026-09-04。

该记录只覆盖可重复的源码级验证；安装态 STS grant 与真实 PowerX 调用另见 `specs/009-consume-powerx-capability/tasks.md` 的 T062–T064。

## 后端模块

在仓库根目录执行：

```bash
go test \
  ./framework/backend/go/runtime/customerfw \
  ./framework/backend/go/runtime/module \
  ./framework/backend/go/runtime/powerx/hostcontract \
  ./skeleton/backend/go-gin/internal/transport/http/admin/host_contract \
  ./skeleton/backend/go-gin/internal/services/capability \
  -count=1
```

该命令覆盖 Customer 双模式工厂、通用 module factory、Host Contract 强类型 transport、Host Contract Lab handler，以及 Capability Catalog 的 provider 元数据保留。

## 模板与边界

```bash
cd skeleton
npm run sync:templates -- --check

cd ..
npm run check:host-contract-lab
git diff --check -- framework skeleton docs/contracts specs/009 scripts package.json scaffold tools/cli/internal/templates/data
```

模板同步检查确保 Skeleton 与 scaffold/CLI 嵌入模板一致。边界检查禁止 Host Contract Lab 直拼 Core URL、直读 Core 数据或以历史 Knowledge QA bridge 替代正式 Knowledge client。

## 前端构建

依赖安装完成后，在下列目录执行：

```bash
cd skeleton/web-admin/nuxt
npm run build
node ./node_modules/vitest/vitest.mjs run tests/unit/useHostContractLab.spec.ts
```

已于 2026-09-04 验证通过 Nuxt production build 与 Host Contract Lab 的 2 个 Vitest 用例。当前 Node 24 环境下 `npx vitest` 会错误解析 npm bin 的相对入口；上面的直接 CLI 命令是本工作区可重复的等价调用。
