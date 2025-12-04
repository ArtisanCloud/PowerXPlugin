#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const cwd = process.cwd();
const repoRoot = path.resolve(cwd, "..", "..");

runCatalog();
ensureExposurePlaceholders();
console.log("[capabilities] export pipeline completed.");

function runCatalog() {
  const tsxBin = getTsxBinary();
  const parserPath = path.resolve(cwd, "catalog-parser.ts");
  if (!fs.existsSync(tsxBin)) {
    console.error("[capabilities] tsx not found. Did you run `npm --prefix scripts/capabilities install`?");
    process.exit(1);
  }
  const result = spawnSync(tsxBin, [parserPath], { stdio: "inherit" });
  if (result.status !== 0) {
    console.error("[capabilities] catalog-parser failed");
    process.exit(result.status ?? 1);
  }
}

function getTsxBinary() {
  const bin = path.resolve(cwd, "node_modules", ".bin", process.platform === "win32" ? "tsx.cmd" : "tsx");
  return bin;
}

function ensureExposurePlaceholders() {
  const files = [
    { file: path.join(repoRoot, "contracts", "exposure", "openapi.yaml"), header: "# OpenAPI exposure placeholder\n" },
    { file: path.join(repoRoot, "contracts", "exposure", "proto", "README.md"), header: "# Proto definitions\n" },
    { file: path.join(repoRoot, "contracts", "exposure", "workflow", "README.md"), header: "# Workflow step templates\n" },
    { file: path.join(repoRoot, "contracts", "exposure", "mcp-tools", "README.md"), header: "# MCP manifests\n" },
    { file: path.join(repoRoot, "contracts", "exposure", "agent-streams", "README.md"), header: "# Agent SSE channels\n" },
    { file: path.join(repoRoot, "dist", "agent-sdk", "README.md"), header: "# SDK bundle output\n" }
  ];

  for (const { file, header } of files) {
    if (fs.existsSync(file)) continue;
    fs.mkdirSync(path.dirname(file), { recursive: true });
    fs.writeFileSync(file, header, "utf8");
    console.log(`[capabilities] created ${path.relative(repoRoot, file)}`);
  }
}
