# Package Artifact Report

## 执行信息
- 执行日期: 2026-03-15
- 执行命令:
  - `npm run build`
  - `npm run verify:artifacts`
  - `find .output -maxdepth 3 -print | sort`

## 产物规模
- `.output` 总大小: `59M`
- `.output` 树节点数（maxdepth=3）: `117`

## 顶层关键条目
- `.output/BUILD_ID`
- `.output/build-manifest.json`
- `.output/prerender-manifest.json`
- `.output/routes-manifest.json`
- `.output/server/`
- `.output/static/`

## 产物树摘录
```text
.output
.output/BUILD_ID
.output/app-build-manifest.json
.output/app-path-routes-manifest.json
.output/build-manifest.json
.output/cache
.output/export-marker.json
.output/images-manifest.json
.output/next-minimal-server.js.nft.json
.output/next-server.js.nft.json
.output/package.json
.output/prerender-manifest.js
.output/prerender-manifest.json
.output/react-loadable-manifest.json
.output/required-server-files.json
.output/routes-manifest.json
.output/server
.output/server/app
.output/server/chunks
.output/server/middleware.js
.output/static
.output/static/chunks
.output/types
```

## 结论
- 构建产物目录结构满足发布路径对齐要求。
- 发布前仍需先解除 `contract drift` 阻断项。
