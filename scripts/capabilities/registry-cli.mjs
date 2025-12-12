#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import YAML from "yaml";

const commands = {
  new: handleCreate,
};

const [, , cmd, ...rest] = process.argv;
if (!cmd || cmd === "--help" || cmd === "-h") {
  printHelp();
  process.exit(0);
}

const run = commands[cmd];
if (!run) {
  console.error(`[capabilities] 未知命令: ${cmd}`);
  printHelp();
  process.exit(1);
}

run(parseArgs(rest)).catch((err) => {
  console.error("[capabilities] 执行失败:", err?.message ?? err);
  process.exit(1);
});

function parseArgs(argv) {
  const args = {
    _: [],
  };
  for (let i = 0; i < argv.length; i += 1) {
    const token = argv[i];
    switch (token) {
      case "--id":
        args.id = argv[++i];
        break;
      case "--namespace":
        args.namespace = argv[++i];
        break;
      case "--resource":
        args.resource = argv[++i];
        break;
      case "--action":
        args.action = argv[++i];
        break;
      case "--root":
        args.root = argv[++i];
        break;
      case "--plugin-id":
        args.pluginId = argv[++i];
        break;
      case "--force":
        args.force = true;
        break;
      default:
        args._.push(token);
        break;
    }
  }
  return args;
}

async function handleCreate(options) {
  const repoRoot = resolveRepoRoot(options.root);
  ensureDir(repoRoot);
  const pluginId = options.pluginId || detectPluginId(repoRoot);
  const manifestPath = path.join(repoRoot, "plugin.yaml");
  const capabilityId = resolveCapabilityId(options, pluginId);
  const fileSafe = capabilityId.replace(/[^A-Za-z0-9._-]/g, "-");
  const descriptorPath = path.join(repoRoot, "capabilities", `${fileSafe}.yaml`);
  const importPath = posixPath(path.relative(repoRoot, descriptorPath));
  const inputSchemaPath = path.join(repoRoot, "contracts", "schema", "input", `${fileSafe}.json`);
  const outputSchemaPath = path.join(repoRoot, "contracts", "schema", "output", `${fileSafe}.json`);

  if (!options.force && fs.existsSync(descriptorPath)) {
    throw new Error(`descriptor 已存在: ${descriptorPath}`);
  }

  ensureDir(path.dirname(descriptorPath));
  ensureDir(path.dirname(inputSchemaPath));
  ensureDir(path.dirname(outputSchemaPath));

  writeDescriptor(descriptorPath, capabilityId, pluginId, options);
  ensureSchemaFile(inputSchemaPath, `${fileSafe}Input`, true);
  ensureSchemaFile(outputSchemaPath, `${fileSafe}Output`, false);
  updateManifest(manifestPath, pluginId, importPath);

  console.log(`[capabilities] 创建成功: ${capabilityId}`);
  console.log(`  descriptor: ${importPath}`);
  console.log(`  schemas: ${posixPath(path.relative(repoRoot, inputSchemaPath))}, ${posixPath(path.relative(repoRoot, outputSchemaPath))}`);
  console.log("请继续实现 handler 与 workflow/agent 协议。");
}

function resolveRepoRoot(rootArg) {
  if (rootArg) {
    return path.resolve(rootArg);
  }
  const cwd = process.cwd();
  if (fs.existsSync(path.join(cwd, "plugin.yaml"))) {
    return cwd;
  }
  return path.resolve(cwd, "..", "..");
}

function resolveCapabilityId(options, pluginId) {
  if (options.id) {
    return options.id.trim();
  }
  const namespace = (options.namespace || pluginId || "").trim();
  const resource = (options.resource || "capability").trim();
  const action = (options.action || "create").trim();
  if (!namespace) {
    throw new Error("请通过 --namespace 或 --id 指定能力命名空间");
  }
  return [namespace, resource, action].filter(Boolean).join(".");
}

function detectPluginId(repoRoot) {
  const fromManifest = readManifestPluginID(path.join(repoRoot, "plugin.yaml"));
  if (fromManifest) {
    return fromManifest;
  }
  const appConst = path.join(repoRoot, "skeleton", "backend", "internal", "shared", "app", "consts.go");
  if (fs.existsSync(appConst)) {
    const content = fs.readFileSync(appConst, "utf8");
    const match = content.match(/const PluginID = "([^"]+)"/);
    if (match) {
      return match[1];
    }
  }
  return "com.powerx.plugins.base";
}

function readManifestPluginID(manifestPath) {
  if (!fs.existsSync(manifestPath)) {
    return "";
  }
  try {
    const doc = YAML.parse(fs.readFileSync(manifestPath, "utf8"));
    return doc?.id ?? "";
  } catch (error) {
    return "";
  }
}

function writeDescriptor(targetPath, capabilityId, pluginId, options) {
  const resource = options.resource || capabilityId.split(".").slice(-2, -1)[0] || "capability";
  const action = options.action || capabilityId.split(".").slice(-1)[0] || "execute";
  const relativeHandler = posixPath(path.join("backend", "internal", "handlers", "capabilities", resource, `${action}_handler.go`));
  const fileSafe = capabilityId.replace(/[^A-Za-z0-9._-]/g, "-");
  const descriptor = {
    id: capabilityId,
    version: "1.0.0",
    atomic_service: relativeHandler,
    summary: {
      zh: "TODO: 填写能力摘要（中文）",
      en: "TODO: describe the capability in English",
    },
    description: {
      zh: "TODO: 填写详细描述、输入输出、注意事项",
      en: "TODO: Describe what this capability does for global tenants.",
    },
    schemas: {
      input: `contracts/schema/input/${fileSafe}.json`,
      output: `contracts/schema/output/${fileSafe}.json`,
    },
    protocols: {
      rest: {
        path: `/api/v1/${resource}`,
        method: "POST",
        auth: "tenant-jwt",
      },
      workflow_step: {
        template: `contracts/exposure/workflow/${fileSafe}.json`,
      },
      agent_tool: {
        manifest: "contracts/exposure/mcp-tools/README.md",
      },
    },
    tags: ["integration", "draft"],
    execution: {
      mode: "sync",
      timeout_seconds: 30,
    },
    metadata: {
      owner: "TODO: owner email or group",
      plugin: pluginId,
    },
  };
  fs.writeFileSync(targetPath, YAML.stringify(descriptor), "utf8");
}

function ensureSchemaFile(targetPath, title, strict) {
  if (fs.existsSync(targetPath)) {
    return;
  }
  const base = {
    $schema: "https://json-schema.org/draft/2020-12/schema",
    title,
    type: "object",
    properties: {},
  };
  if (strict) {
    base.required = [];
    base.additionalProperties = false;
  } else {
    base.additionalProperties = true;
  }
  fs.mkdirSync(path.dirname(targetPath), { recursive: true });
  fs.writeFileSync(targetPath, JSON.stringify(base, null, 2), "utf8");
}

function updateManifest(manifestPath, pluginId, importPath) {
  let manifest = {};
  if (fs.existsSync(manifestPath)) {
    manifest = YAML.parse(fs.readFileSync(manifestPath, "utf8")) ?? {};
  } else {
    console.log(`[capabilities] manifest 不存在，自动创建: ${manifestPath}`);
    manifest = {
      id: pluginId,
      version: "0.1.0",
      name: {
        zh: "PowerX 插件",
        en: "PowerX Plugin",
      },
      capabilities: {
        imports: [],
      },
    };
  }
  manifest.id = manifest.id || pluginId;
  manifest.capabilities = manifest.capabilities || {};
  manifest.capabilities.imports = manifest.capabilities.imports || [];
  if (!manifest.capabilities.imports.includes(importPath)) {
    manifest.capabilities.imports.push(importPath);
  }
  fs.writeFileSync(manifestPath, YAML.stringify(manifest), "utf8");
}

function ensureDir(dir) {
  fs.mkdirSync(dir, { recursive: true });
}

function posixPath(p) {
  return p.split(path.sep).join("/");
}

function printHelp() {
  console.log(`Usage: registry-cli.mjs <command> [options]

Commands:
  new --id <capabilityId>            创建能力描述及 Schema
     [--namespace ns --resource r --action a]  通过片段生成 ID
     [--root <repoRoot>]             指定仓库根目录
     [--plugin-id <pluginId>]        覆盖默认插件 ID
`);
}
