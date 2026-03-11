#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import fg from "fast-glob";

const root = process.cwd();

const targets = [
  "scaffold/templates/.codex/**/*",
  "scaffold/templates/.specify/**/*",
  "tools/cli/internal/templates/data/.codex/**/*",
  "tools/cli/internal/templates/data/.specify/**/*",
];

const ignore = ["**/.DS_Store", "**/Thumbs.db"];

const forbidden = [
  "skeleton/plugin.yaml",
  "./skeleton/plugin.yaml",
  "skeleton/plugin.d",
  "./skeleton/plugin.d",
  "make -C skeleton ",
  "skeleton/backend/go-gin",
  "skeleton/backend/python-fastapi",
  "skeleton/web-admin/nuxt",
  "skeleton/web-admin/next",
];

const files = fg.sync(targets, {
  cwd: root,
  onlyFiles: true,
  dot: true,
  ignore,
});

let violations = 0;
for (const file of files) {
  const abs = path.join(root, file);
  const raw = fs.readFileSync(abs, "utf8");
  for (const item of forbidden) {
    if (raw.includes(item)) {
      violations++;
      console.error(`[agent-path-check] ${file} contains forbidden path: ${item}`);
    }
  }
}

if (violations > 0) {
  console.error(`[agent-path-check] failed: ${violations} violation(s) found.`);
  process.exit(1);
}

console.log("[agent-path-check] passed: no legacy skeleton paths found.");
