#!/usr/bin/env node

import { execSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const args = new Set(process.argv.slice(2));
const autoFix = args.has("--fix");
const autoStage = args.has("--stage");

function run(command, options = {}) {
  return execSync(command, {
    cwd: repoRoot,
    stdio: ["ignore", "pipe", "pipe"],
    encoding: "utf8",
    ...options,
  });
}

function fail(message) {
  console.error(`❌ ${message}`);
  process.exit(1);
}

function info(message) {
  console.log(`ℹ️  ${message}`);
}

function ok(message) {
  console.log(`✅ ${message}`);
}

function listCapabilityFiles() {
  const dir = path.join(repoRoot, "contracts", "capabilities");
  if (!fs.existsSync(dir)) return [];
  return fs
    .readdirSync(dir)
    .filter((name) => name.endsWith(".yaml"))
    .sort()
    .map((name) => path.join(dir, name));
}

function parseCapability(content, filePath) {
  const idMatch = content.match(/^id:\s*([^\n]+)$/m);
  if (!idMatch) fail(`capability 缺少 id: ${filePath}`);
  const id = idMatch[1].trim();

  const rbacBlock = content.match(/rbac:\s*([\s\S]*?)(?:\n[a-zA-Z_][^:\n]*:|\n$)/m);
  if (!rbacBlock) fail(`capability 缺少 rbac 段: ${filePath}`);
  const resourceMatch = rbacBlock[1].match(/resource:\s*([^\n]+)$/m);
  let actions = [];
  const actionsInlineMatch = rbacBlock[1].match(/actions:\s*\[([^\]]+)\]/m);
  if (actionsInlineMatch) {
    actions = actionsInlineMatch[1]
      .split(",")
      .map((item) => item.trim())
      .filter(Boolean);
  } else {
    const actionsBlockMatch = rbacBlock[1].match(/actions:\s*\n([\s\S]*?)(?:\n\s*[a-zA-Z_][^:\n]*:|$)/m);
    if (actionsBlockMatch) {
      actions = actionsBlockMatch[1]
        .split("\n")
        .map((line) => line.match(/^\s*-\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*$/)?.[1] || "")
        .filter(Boolean);
    }
  }

  if (!resourceMatch || actions.length === 0) {
    fail(`capability rbac 缺少 resource/actions: ${filePath}`);
  }
  const resource = resourceMatch[1].trim();

  return {
    id,
    resource,
    actions,
    firstAction: actions[0],
  };
}

function parseExposure(content) {
  const channels = [];
  const lines = content.split("\n");
  let current = null;
  for (const line of lines) {
    const cap = line.match(/^\s*capability:\s*(\S+)\s*$/);
    if (cap) {
      current = { capability: cap[1], rbac: "" };
      channels.push(current);
      continue;
    }
    const rbac = line.match(/^\s*rbac:\s*(\S+)\s*$/);
    if (rbac && current) {
      current.rbac = rbac[1];
    }
  }
  return channels;
}

function parseRBACResources(content) {
  const lines = content.split("\n");
  const resources = new Map();
  let inResources = false;
  let currentResource = "";
  let collectingActions = false;
  let pendingActions = [];

  function ensureResource(resource) {
    if (!resources.has(resource)) resources.set(resource, new Set());
  }

  function commitPendingActions() {
    if (!currentResource || pendingActions.length === 0) return;
    ensureResource(currentResource);
    for (const action of pendingActions) {
      resources.get(currentResource).add(action);
    }
    pendingActions = [];
  }

  function parseInlineActions(raw) {
    return raw
      .split(",")
      .map((item) => item.trim())
      .filter(Boolean);
  }

  for (const raw of lines) {
    const line = raw.replace(/\r$/, "");
    if (/^rbac:\s*$/.test(line)) continue;
    if (/^\s*resources:\s*$/.test(line)) {
      inResources = true;
      continue;
    }
    if (inResources && /^\s*routes:\s*$/.test(line)) break;
    if (!inResources) continue;

    const listResourceMatch = line.match(/^\s*-\s*resource:\s*(\S+)\s*$/);
    if (listResourceMatch) {
      commitPendingActions();
      collectingActions = false;
      currentResource = listResourceMatch[1];
      ensureResource(currentResource);
      continue;
    }
    const listActionsInlineMatch = line.match(/^\s*-\s*actions:\s*\[([^\]]+)\]\s*$/);
    if (listActionsInlineMatch) {
      pendingActions = parseInlineActions(listActionsInlineMatch[1]);
      collectingActions = false;
      commitPendingActions();
      continue;
    }
    const listActionsStartMatch = line.match(/^\s*-\s*actions:\s*$/);
    if (listActionsStartMatch) {
      pendingActions = [];
      collectingActions = true;
      continue;
    }

    const resourceMatch = line.match(/^\s*resource:\s*(\S+)\s*$/);
    if (resourceMatch) {
      currentResource = resourceMatch[1];
      ensureResource(currentResource);
      commitPendingActions();
      collectingActions = false;
      continue;
    }
    const actionsInlineMatch = line.match(/^\s*actions:\s*\[([^\]]+)\]\s*$/);
    if (actionsInlineMatch) {
      pendingActions = parseInlineActions(actionsInlineMatch[1]);
      collectingActions = false;
      commitPendingActions();
      continue;
    }
    const actionsStartMatch = line.match(/^\s*actions:\s*$/);
    if (actionsStartMatch) {
      pendingActions = [];
      collectingActions = true;
      continue;
    }

    const actionMatch = line.match(/^\s*-\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*$/);
    if (collectingActions && actionMatch) {
      pendingActions.push(actionMatch[1]);
      continue;
    }
  }
  commitPendingActions();
  return resources;
}

function listDriftFiles(targets) {
  const output = run(`git diff --name-only -- ${targets.join(" ")}`).trim();
  if (!output) return [];
  return output.split("\n").map((item) => item.trim()).filter(Boolean);
}

info("执行 plugin catalog 重建检查…");
try {
  const output = run("make plugin-yaml-check");
  process.stdout.write(output);
} catch (error) {
  process.stdout.write(error.stdout || "");
  process.stderr.write(error.stderr || "");
  fail("make plugin-yaml-check 执行失败");
}

const driftTargets = [
  "plugin.d/capabilities.yaml",
  "plugin.d/exposure.yaml",
  "plugin.d/rbac.yaml",
];
const driftFiles = listDriftFiles(driftTargets);
if (driftFiles.length > 0 && !autoFix) {
  fail(
    `检测到 catalog 漂移（请提交同步结果）:\n${driftFiles.join("\n")}\n` +
      "建议：追加 --fix 自动接受同步结果，或手工确认后提交"
  );
}
if (driftFiles.length > 0 && autoFix) {
  ok(`已自动同步 catalog：\n${driftFiles.join("\n")}`);
  if (autoStage) {
    run(`git add -- ${driftFiles.join(" ")}`);
    ok("已自动 git add 同步产物");
  } else {
    info("已写入最新产物；如需提交请执行 git add")
  }
} else {
  ok("plugin.d 产物无漂移");
}

const capabilityFiles = listCapabilityFiles();
if (capabilityFiles.length === 0) {
  fail("未找到 contracts/capabilities/*.yaml");
}
const capabilities = capabilityFiles.map((filePath) =>
  parseCapability(fs.readFileSync(filePath, "utf8"), filePath)
);

const exposureContent = fs.readFileSync(
  path.join(repoRoot, "skeleton", "plugin.d", "exposure.yaml"),
  "utf8"
);
const exposureChannels = parseExposure(exposureContent);
let exposureNoRBACCount = 0;

for (const capability of capabilities) {
  const matchedChannels = exposureChannels.filter(
    (item) => item.capability === capability.id
  );
  if (matchedChannels.length === 0) {
    fail(`exposure 缺少 capability: ${capability.id}`);
  }
  const matched = matchedChannels.find((item) => item.rbac);
  if (!matched) {
    exposureNoRBACCount += 1;
    continue;
  }
  const expectedRBAC = `${capability.resource}:${capability.firstAction}`;
  if (matched.rbac !== expectedRBAC) {
    fail(
      `exposure rbac 不匹配: ${capability.id}\n` +
        `  expected=${expectedRBAC}\n  actual=${matched.rbac || "<empty>"}`
    );
  }
}
ok("capability -> exposure.rbac 映射通过");
if (exposureNoRBACCount > 0) {
  info(`exposure 中 ${exposureNoRBACCount} 个 capability 无 rbac 字段（按非 HTTP 通道跳过）`);
}

const rbacContent = fs.readFileSync(
  path.join(repoRoot, "skeleton", "plugin.d", "rbac.yaml"),
  "utf8"
);
const rbacResources = parseRBACResources(rbacContent);

for (const capability of capabilities) {
  const actions = rbacResources.get(capability.resource);
  if (!actions) {
    fail(`rbac.resources 缺少 resource: ${capability.resource}`);
  }
  for (const action of capability.actions) {
    if (!actions.has(action)) {
      fail(
        `rbac.resources 缺少 action: resource=${capability.resource}, action=${action}`
      );
    }
  }
}
ok("capability -> rbac.resources 覆盖通过");

ok("Manifest 对齐检查全部通过");
