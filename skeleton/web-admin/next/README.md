# Web Admin (Next)

## 目标
- 作为 Nuxt 管理端迁移目标实现。
- 行为以 Nuxt 基线为准，后端契约以 Gin 为准。
- 禁止引入仅 Next 可用的私有后端接口。

## 本地运行

```bash
cd /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/skeleton/web-admin/next
npm install
npm run dev
```

默认访问地址：`http://localhost:3231`

## 核心脚本
- `npm run lint`: 静态检查
- `npm run build`: 生成构建产物（输出到 `.output/`）
- `npm run e2e`: 运行 Playwright 全量回归
- `npm run verify:artifacts`: 校验构建产物路径与关键文件

## 构建产物与发布约定
- Next 构建输出目录固定为：`skeleton/web-admin/next/.output/`
- 发布前必须执行：

```bash
npm run build
npm run verify:artifacts
```

- 产物校验结果需同步到：
  - `specs/012-next-nuxt-align/verification-report.md`
  - `specs/012-next-nuxt-align/package-artifact-report.md`

## 迁移与联调约束
- 优先修正 Next 偏差，不直接改 Gin。
- Gin 仅允许“确认缺陷后的最小修复”，见：
  - `specs/012-next-nuxt-align/gin-defect-policy.md`
- 差异归因必须 2 个工作日闭环：
  - `specs/012-next-nuxt-align/parity-triage-sop.md`
