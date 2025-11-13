#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import fg from "fast-glob";
import YAML from "yaml";

function parseArgs(argv) {
  const args = {
    plugin: process.cwd(),
    manifest: "plugin.yaml",
    handlers: "backend/internal/handlers/capabilities",
  };
  for (let i = 2; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === "--plugin" && argv[i + 1]) {
      args.plugin = argv[++i];
    } else if (arg === "--manifest" && argv[i + 1]) {
      args.manifest = argv[++i];
    } else if (arg === "--handlers" && argv[i + 1]) {
      args.handlers = argv[++i];
    } else if (arg === "--help" || arg === "-h") {
      args.help = true;
    } else {
      args._ = args._ || [];
      args._.push(arg);
    }
  }
  return args;
}

function printHelp() {
  console.log(`Usage: discover-handlers --plugin <path> [--handlers <dir>] [--manifest <file>]

Options:
  --plugin <dir>     插件根目录（默认：当前目录）
  --handlers <dir>   相对 handlers 根路径（默认：backend/internal/handlers/capabilities）
  --manifest <file>  manifest 相对路径（默认：plugin.yaml，可缺省）

脚本会查找 <handlers>/**/_handler.go 的 // capability: <id> 注释，并对照 manifest.capabilities.provides 输出注册状态。`);
}

async function loadManifestCapabilities(manifestPath) {
  if (!fs.existsSync(manifestPath)) {
    return { ids: new Set(), available: false };
  }
  try {
    const doc = YAML.parse(fs.readFileSync(manifestPath, "utf8"));
    const provides = doc?.capabilities?.provides ?? [];
    return {
      ids: new Set(
        provides
          .map((entry) => entry?.id)
          .filter((id) => typeof id === "string" && id.length > 0),
      ),
      available: true,
    };
  } catch (err) {
    console.warn(
      `⚠️ 无法解析 manifest (${manifestPath}): ${(err)?.message ?? err}`,
    );
    return { ids: new Set(), available: false };
  }
}

async function main() {
  const args = parseArgs(process.argv);
  if (args.help) {
    printHelp();
    process.exit(0);
  }

  const pluginDir = path.resolve(args.plugin);
  const manifestPath = path.resolve(pluginDir, args.manifest);
  const handlersDir = path.resolve(pluginDir, args.handlers);

  if (!fs.existsSync(handlersDir)) {
    console.error(`⚠️ handler 目录不存在: ${handlersDir}`);
    process.exit(1);
  }

  const pattern = path
    .join(handlersDir, "**", "*_handler.go")
    .replace(/\\/g, "/");
  const files = await fg(pattern, { onlyFiles: true });
  if (!files.length) {
    console.log("未发现 *_handler.go 文件");
    process.exit(0);
  }

  const manifest = await loadManifestCapabilities(manifestPath);

  console.log(`扫描目录: ${handlersDir}`);
  if (manifest.available) {
    console.log(`已在 manifest 中声明的 capability 数: ${manifest.ids.size}`);
  } else {
    console.log("未提供可解析的 manifest（仅显示 handler 注释，不比对状态）");
  }
  console.log("--------------------------------------------------");

  const missing = [];
  for (const file of files) {
    const rel = path.relative(pluginDir, file);
    const source = fs.readFileSync(file, "utf8");
    const matches = [...source.matchAll(/capability\s*:\s*([A-Za-z0-9._:-]+)/g)];
    if (!matches.length) {
      console.log(`• (未标注)                        ⚠️ 缺少 // capability 注释 ← ${rel}`);
      continue;
    }
    for (const match of matches) {
      const capabilityId = match[1];
      const registered = manifest.ids.has(capabilityId);
      const status = manifest.available
        ? registered
          ? "✅ 已注册"
          : "⚠️ 未在 manifest 中"
        : "•";
      console.log(`• ${capabilityId.padEnd(40)} ${status} ← ${rel}`);
      if (manifest.available && !registered) {
        missing.push({ capabilityId, file: rel });
      }
    }
  }

  if (manifest.available && missing.length) {
    console.log("\n建议补充到 plugin.yaml:");
    for (const entry of missing) {
      console.log(
        `- id: ${entry.capabilityId}  # 来源: ${entry.file}`,
      );
    }
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
