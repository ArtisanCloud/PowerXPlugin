#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import path from "node:path";
import process from "node:process";

const manifest = process.env.CAP_MANIFEST;
if (!manifest) {
  console.log(
    "[capabilities] 跳过验证：未设置 CAP_MANIFEST 环境变量（示例：CAP_MANIFEST=./plugin.yaml npm test）",
  );
  process.exit(0);
}

const scriptPath = path.resolve(
  process.cwd(),
  "scripts/capabilities/validate-capabilities.mjs",
);
const result = spawnSync(process.execPath, [scriptPath, "--manifest", manifest], {
  stdio: "inherit",
});
process.exit(result.status ?? 0);
