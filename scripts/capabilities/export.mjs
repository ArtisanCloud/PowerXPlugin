#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import YAML from "yaml";

const cwd = process.cwd();
const repoRoot = path.resolve(cwd, "..", "..");
const exposureRoot = path.join(repoRoot, "contracts", "exposure");

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
    { file: path.join(exposureRoot, "openapi.yaml"), header: "# OpenAPI exposure placeholder\n" },
    { file: path.join(exposureRoot, "proto", "README.md"), header: "# Proto definitions\n" },
    { file: path.join(exposureRoot, "workflow", "README.md"), header: "# Workflow step templates\n" },
    { file: path.join(exposureRoot, "agent-streams", "README.md"), header: "# Agent SSE channels\n" },
    { file: path.join(exposureRoot, "composites", "README.md"), header: "# Workflow composites\n" },
    { file: path.join(exposureRoot, "capability-lifecycle.json"), header: JSON.stringify({ plans: [], updated_at: new Date(0).toISOString() }, null, 2) + "\n" },
    { file: path.join(exposureRoot, "mcp-tools.json"), header: JSON.stringify({ plugin_id: "", version: "", generated_at: new Date(0).toISOString(), tools: [] }, null, 2) + "\n" },
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
  const exposureFile = path.join(exposureRoot, "exposure-packages.json");
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

  const composites = loadComposites();
  writeOpenAPI(pluginId, manifestVersion, packages);
  writeAgentBundle(pluginId, manifestVersion, packages);
  writeCompositeWorkflowAssets(composites, manifestVersion);
  writeAgentStreams(composites, manifestVersion);
  writeMCPManifest(pluginId, manifestVersion, composites);
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

function loadComposites() {
  const dir = path.join(exposureRoot, "composites");
  if (!fs.existsSync(dir)) {
    return [];
  }
  const files = fs.readdirSync(dir).filter((file) => file.endsWith(".json"));
  const composites = [];
  for (const file of files) {
    const abs = path.join(dir, file);
    try {
      const parsed = JSON.parse(fs.readFileSync(abs, "utf8"));
      composites.push(parsed);
    } catch (error) {
      console.warn(`[capabilities] failed to parse composite ${file}:`, error.message);
    }
  }
  return composites;
}

function writeCompositeWorkflowAssets(composites, manifestVersion) {
  for (const composite of composites) {
    if (!composite?.workflow_step) continue;
    const payload = {
      id: composite.id,
      version: composite.version || manifestVersion,
      name: composite.name || composite.id,
      summary: composite.summary || "",
      nodes: composite.graph?.nodes || [],
      edges: composite.graph?.edges || [],
      metadata: composite.metadata || {}
    };
    const target = path.join(repoRoot, composite.workflow_step);
    fs.mkdirSync(path.dirname(target), { recursive: true });
    fs.writeFileSync(target, JSON.stringify(payload, null, 2), "utf8");
    console.log(`[capabilities] workflow template generated → ${path.relative(repoRoot, target)}`);
  }
}

function writeAgentStreams(composites, manifestVersion) {
  for (const composite of composites) {
    const stream = composite?.agent_stream;
    if (!stream || !stream.path) continue;
    const payload = {
      capability_id: composite.id,
      version: composite.version || manifestVersion,
      intent: stream.intent || composite.id,
      summary: stream.summary || composite.summary || "",
      events: stream.events || [],
      generated_at: new Date().toISOString()
    };
    const target = path.join(repoRoot, stream.path);
    fs.mkdirSync(path.dirname(target), { recursive: true });
    fs.writeFileSync(target, YAML.stringify(payload), "utf8");
    console.log(`[capabilities] agent stream generated → ${path.relative(repoRoot, target)}`);
  }
}

function writeMCPManifest(pluginId, manifestVersion, composites) {
  const dedup = new Map();
  for (const composite of composites) {
    for (const tool of composite.agent_tools || []) {
      if (!tool?.id) continue;
      const key = tool.id;
      if (dedup.has(key)) continue;
      dedup.set(key, {
        id: tool.id,
        capability_id: tool.capability_id || composite.id,
        name: tool.name || tool.id,
        description: tool.description || composite.summary || "",
        transport: tool.transport || "http",
        endpoint: tool.endpoint || "",
        method: tool.method || "POST",
        input_schema: tool.input_schema || "",
        output_schema: tool.output_schema || ""
      });
    }
  }

  const manifest = {
    plugin_id: pluginId,
    version: manifestVersion,
    generated_at: new Date().toISOString(),
    tools: Array.from(dedup.values())
  };
  const target = path.join(exposureRoot, "mcp-tools.json");
  fs.mkdirSync(path.dirname(target), { recursive: true });
  fs.writeFileSync(target, JSON.stringify(manifest, null, 2), "utf8");
  console.log(`[capabilities] MCP manifest generated → ${path.relative(repoRoot, target)}`);
}
