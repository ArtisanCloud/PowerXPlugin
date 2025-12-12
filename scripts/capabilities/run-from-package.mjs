#!/usr/bin/env node

import fs from "node:fs";
import { spawnSync } from "node:child_process";
import path from "node:path";
import process from "node:process";

const defaultManifest = "./skeleton/plugin.yaml";
const fallbackManifest = "./plugin.yaml";

const manifestEnv = process.env.CAP_MANIFEST;
if (!manifestEnv) {
  console.log(`[capabilities] 未设置 CAP_MANIFEST，尝试使用默认 ${defaultManifest}`);
}

let manifest = manifestEnv || defaultManifest;
let resolvedManifest = path.resolve(process.cwd(), manifest);

if (!fs.existsSync(resolvedManifest) && !manifestEnv) {
  const resolvedFallback = path.resolve(process.cwd(), fallbackManifest);
  if (fs.existsSync(resolvedFallback)) {
    console.log(
      `[capabilities] 默认清单 ${defaultManifest} 不存在，回退到 ${fallbackManifest}`,
    );
    manifest = fallbackManifest;
    resolvedManifest = resolvedFallback;
  }
}

if (!fs.existsSync(resolvedManifest)) {
  console.error(`[capabilities] manifest not found: ${resolvedManifest}`);
  process.exit(1);
}

const scriptPath = path.resolve(
  process.cwd(),
  "scripts/capabilities/validate-capabilities.mjs",
);
const result = spawnSync(process.execPath, [scriptPath, "--manifest", manifest], {
  stdio: "inherit",
});
process.exit(result.status ?? 0);
