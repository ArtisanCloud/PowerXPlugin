#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import YAML from "yaml";

const cwd = process.cwd();
const repoRoot = path.resolve(cwd, "..", "..");

runCatalog();
ensureExposurePlaceholders();
generateExposureArtifacts();
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
    { file: path.join(repoRoot, "contracts", "exposure", "capability-lifecycle.json"), header: JSON.stringify({ plans: [], updated_at: new Date(0).toISOString() }, null, 2) + "\n" },
    { file: path.join(repoRoot, "dist", "agent-sdk", "README.md"), header: "# SDK bundle output\n" }
  ];

  for (const { file, header } of files) {
    if (fs.existsSync(file)) continue;
    fs.mkdirSync(path.dirname(file), { recursive: true });
    fs.writeFileSync(file, header, "utf8");
    console.log(`[capabilities] created ${path.relative(repoRoot, file)}`);
  }
}

function generateExposureArtifacts() {
  const exposureFile = path.join(repoRoot, "contracts", "exposure", "exposure-packages.json");
  const catalogFile = path.join(repoRoot, "capabilities", "catalog.json");
  let packages = [];
  if (fs.existsSync(exposureFile)) {
    try {
      const snapshot = JSON.parse(fs.readFileSync(exposureFile, "utf8"));
      packages = snapshot?.packages ?? [];
    } catch (error) {
      console.warn("[capabilities] failed to parse exposure packages:", error.message);
    }
  } else {
    console.warn("[capabilities] exposure package file missing, skipping openapi generation");
  }

  let pluginId = "com.powerx.plugins.base";
  let manifestVersion = "1.0.0";
  if (fs.existsSync(catalogFile)) {
    try {
      const catalog = JSON.parse(fs.readFileSync(catalogFile, "utf8"));
      pluginId = catalog?.plugin_id || pluginId;
      manifestVersion = catalog?.manifest_version || manifestVersion;
    } catch (error) {
      console.warn("[capabilities] failed to parse catalog:", error.message);
    }
  }

  writeOpenAPI(pluginId, manifestVersion, packages);
  writeAgentBundle(pluginId, manifestVersion, packages);
}

function writeOpenAPI(pluginId, manifestVersion, packages) {
  const doc = {
    openapi: "3.0.3",
    info: {
      title: `${pluginId} Capability Exposure`,
      version: manifestVersion,
    },
    paths: {},
  };

  for (const pkg of packages) {
    const capID = pkg?.capability_id;
    if (!capID || !Array.isArray(pkg.channels)) continue;
    for (const channel of pkg.channels) {
      if (!channel?.enabled) continue;
      if (channel.type !== "rest" && channel.type !== "webhook") continue;
      const method = (channel.method || "POST").toLowerCase();
      const pathKey = channel.path || `/plugins/${capID}`;
      if (!doc.paths[pathKey]) {
        doc.paths[pathKey] = {};
      }
      doc.paths[pathKey][method] = {
        tags: [capID],
        summary: channel.name || capID,
        description: channel.description || "",
        responses: {
          "200": {
            description: "Success",
          },
        },
      };
    }
  }

  const output = YAML.stringify(doc);
  const target = path.join(repoRoot, "contracts", "exposure", "openapi.yaml");
  fs.writeFileSync(target, output, "utf8");
  console.log(`[capabilities] exposure OpenAPI generated → ${path.relative(repoRoot, target)}`);
}

function writeAgentBundle(pluginId, manifestVersion, packages) {
  const tools = [];
  for (const pkg of packages) {
    if (!pkg?.capability_id || !Array.isArray(pkg.channels)) continue;
    const asyncChannels = pkg.channels.filter((channel) =>
      ["workflow", "agent", "agent_sse"].includes(channel.type),
    );
    for (const channel of asyncChannels) {
      if (!channel.enabled) continue;
      tools.push({
        capability_id: pkg.capability_id,
        type: channel.type,
        name: channel.name || channel.type,
        description: channel.description || "",
        target: channel.target || channel.path || "",
      });
    }
  }
  const manifest = {
    plugin_id: pluginId,
    version: manifestVersion,
    generated_at: new Date().toISOString(),
    tools,
  };
  const target = path.join(repoRoot, "dist", "agent-sdk", "manifest.json");
  fs.writeFileSync(target, JSON.stringify(manifest, null, 2), "utf8");
  console.log(`[capabilities] agent SDK manifest generated → ${path.relative(repoRoot, target)}`);
}
