#!/usr/bin/env node

/**
 * Placeholder hook for fullstack-go-nuxt template.
 * CLI 运行 `px-plugin init` 后可调用该脚本执行附加步骤（如写 publish.yml、生成 SBOM、提示 next steps）。
 * 真正的业务逻辑尚未接入 speckit pipeline，这里只输出提示，避免模板索引引用不存在的文件。
 */
async function main() {
  // 目前仅输出说明，避免误认为 Hook 已实现自动化逻辑。
  console.log("[hook] post-render-fullstack: No-op placeholder. Update scripts/hooks/templates/post-render-fullstack.mjs when automation is ready.");
}

main().catch((err) => {
  console.error("[hook] post-render-fullstack error:", err);
  process.exit(1);
});
