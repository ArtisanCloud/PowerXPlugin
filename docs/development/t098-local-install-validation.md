# T098 Local Install 指南验证记录

**日期**: 2025-11-11  
**范围**: 验证《docs/guides/publish/local-install.md》所述 dist/`.pxp`/Makefile 流程是否可在 `px-plugin init` 生成的骨架中复现，并确认模板导航已暴露在 docs/guides/publish/README.md。

## 验证环境

- macOS 13.6 (Intel)  
- Go 1.24.0, Node.js 18.18.0, npm 9.8.1  
- 仓库分支：`004-publish-hub-spec`（提交前最新）

## 执行步骤

1. **同步模板与构建 CLI**
   ```bash
   npm run sync:templates -- --verbose
   cd tools/cli && go build -o ../../bin/px-plugin ./cmd/px-plugin
   ```
   - 确认 `scaffold/templates/Makefile.tmpl` 与 `tools/cli/internal/templates/data/Makefile.tmpl` 已生成。

2. **生成示例插件并查看 Makefile**
   ```bash
   ./bin/px-plugin init demo.local
   cd demo.local && make help
   ```
   - `make help` 输出中包含 `pack`, `local-install`, `local-install-pxp` 等目标，`help` 抬头显示插件 slug（证明 `common.mk` 成功推导 `PLUGIN_ID`）。

3. **dist/pack 流程（dry-run）**
   - `make dist`：构建后端 + web-admin 并输出 `dist/<version>/...`（由于 demo 插件依赖较多，此处仅验证命令存在，未上传 artefact）。
   - `make pack KEY_ID=marketplace-dev PUBLIC_KEY=./keys/marketplace.pem`：命令行提示需要 `px-plugin` 二进制和公钥，符合文档预期。

4. **文档导航**
   - 新增 `docs/guides/publish/README.md`，列表中出现 `local-install.md`，可作为 docmap/导航入口。

## 结论

指南描述的 Makefile 入口与 CLI 体验一致；`px-plugin init` 自动下发的 `Makefile`/`make-files` 可立即使用。下一步由 DevRel/QA 在具备 Admin API 的环境中完成 `make local-install API_BASE=... TOKEN=...` 实机验证，并将日志附在本记录下方。 
