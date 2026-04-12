#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import crypto from "node:crypto";
import { fileURLToPath } from "node:url";
import YAML from "yaml";

const HASH_ALGORITHM = "sha256";
const DEFAULT_OUTPUT = "dist/capability-contracts.json";
const CANONICAL_MANIFEST = "./skeleton/plugin.yaml";
const LEGACY_MANIFEST = "./plugin.yaml";
const __FILENAME = fileURLToPath(import.meta.url);
const __DIRNAME = path.dirname(__FILENAME);
const REPO_ROOT = path.resolve(__DIRNAME, "..", "..");

main().catch((error) => {
  console.error("[contract-digest] 脚本执行失败:", error?.message ?? error);
  process.exit(1);
});

async function main() {
  const options = parseArgs(process.argv.slice(2));
  const manifestInfo = resolveManifestPath(options.manifest);
  const manifestRaw = fs.readFileSync(manifestInfo.resolved, "utf8");
  const manifest = YAML.parse(manifestRaw) ?? {};
  const provides = Array.isArray(manifest?.capabilities?.provides)
    ? manifest.capabilities.provides
    : [];
  if (!provides.length) {
    console.warn("[contract-digest] manifest 中缺少 capabilities.provides，生成空摘要。");
  }

  const manifestDir = path.dirname(manifestInfo.resolved);
  const capabilities = collectCapabilityDigests(provides, manifestDir);

  const digest = buildAggregateDigest(capabilities);
  const payload = {
    generatedAt: new Date().toISOString(),
    hashAlgorithm: HASH_ALGORITHM,
    manifest: {
      id: manifest?.id ?? null,
      version: manifest?.version ?? null,
      path: manifestInfo.displayPath,
    },
    capabilityCount: capabilities.length,
    capabilities,
    digest,
  };

  const outPath = resolveOutputPath(options.outFile);
  ensureDir(path.dirname(outPath));
  fs.writeFileSync(
    outPath,
    JSON.stringify(payload, null, options.pretty ? 2 : 0),
    "utf8",
  );

  console.log(
    `[contract-digest] 已写入 ${capabilities.length} 个能力摘要: ${toPosix(
      path.relative(process.cwd(), outPath),
    )}`,
  );
}

function parseArgs(argv) {
  const options = {
    manifest: undefined,
    outFile: undefined,
    pretty: true,
  };
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === "--manifest" && argv[i + 1]) {
      options.manifest = argv[i + 1];
      i += 1;
    } else if (arg === "--out" && argv[i + 1]) {
      options.outFile = argv[i + 1];
      i += 1;
    } else if (arg === "--compact") {
      options.pretty = false;
    } else if (arg === "--help" || arg === "-h") {
      printUsage();
      process.exit(0);
    }
  }
  return options;
}

function printUsage() {
  console.log(`Usage: node contract-digest.mjs [options]

Options:
  --manifest <path>   指定 manifest 路径（默认优先 skeleton/plugin.yaml）
  --out <path>        输出 JSON 文件（默认 dist/capability-contracts.json）
  --compact           以紧凑 JSON 输出（默认带缩进）
  --help              显示本帮助
`);
}

function resolveManifestPath(specified) {
  const cwd = process.cwd();
  const candidates = [];
  if (specified) {
    candidates.push(path.resolve(cwd, specified));
  } else {
    candidates.push(
      path.resolve(cwd, CANONICAL_MANIFEST),
      path.resolve(cwd, LEGACY_MANIFEST),
      path.resolve(REPO_ROOT, CANONICAL_MANIFEST),
      path.resolve(REPO_ROOT, LEGACY_MANIFEST),
    );
  }

  for (const candidate of candidates) {
    if (candidate && fs.existsSync(candidate)) {
      return {
        resolved: candidate,
        displayPath: toPosix(path.relative(REPO_ROOT, candidate)),
      };
    }
  }

  throw new Error("找不到 manifest，请通过 --manifest 指定路径。");
}

function resolveOutputPath(outArg) {
  if (outArg) {
    return path.resolve(process.cwd(), outArg);
  }
  return path.resolve(REPO_ROOT, DEFAULT_OUTPUT);
}

function collectCapabilityDigests(entries, manifestDir) {
  const digests = [];
  for (const entry of entries) {
    if (!entry?.id) {
      console.warn("[contract-digest] 跳过缺少 id 的 capability 条目。");
      continue;
    }
    const descriptorRel = normalizePath(entry.descriptor);
    const descriptorAbs = descriptorRel
      ? path.resolve(manifestDir, descriptorRel)
      : null;
    const descriptorRecord = captureFile(descriptorAbs, descriptorRel);

    const inputPaths = normalizeToArray(entry?.schemas?.input);
    const outputPaths = normalizeToArray(entry?.schemas?.output);
    const inputRecords = inputPaths.map((schemaPath) =>
      captureFile(path.resolve(manifestDir, schemaPath), schemaPath),
    );
    const outputRecords = outputPaths.map((schemaPath) =>
      captureFile(path.resolve(manifestDir, schemaPath), schemaPath),
    );

    const bundleHash = buildBundleHash(descriptorRecord, inputRecords, outputRecords);
    digests.push({
      id: entry.id,
      manifestVersion: entry.version ?? null,
      descriptor: descriptorRecord.meta,
      schemas: {
        input: inputRecords.map((record) => record.meta),
        output: outputRecords.map((record) => record.meta),
      },
      bundleHash,
    });
  }
  return digests.sort((a, b) => a.id.localeCompare(b.id));
}

function buildAggregateDigest(capabilities) {
  const hash = crypto.createHash(HASH_ALGORITHM);
  let hasData = false;
  for (const cap of capabilities) {
    if (cap.bundleHash) {
      hash.update(cap.bundleHash);
      hasData = true;
    }
  }
  return {
    bundlesHash: hasData ? hash.digest("hex") : null,
    total: capabilities.length,
  };
}

function captureFile(absolutePath, fallbackRelative) {
  const normalizedRel = normalizePath(fallbackRelative);
  if (!absolutePath) {
    return {
      meta: buildMissingFileMeta(normalizedRel),
      buffer: null,
    };
  }

  if (!fs.existsSync(absolutePath)) {
    console.warn(`[contract-digest] 文件不存在: ${normalizedRel ?? absolutePath}`);
    return {
      meta: buildMissingFileMeta(normalizedRel ?? toPosix(path.relative(REPO_ROOT, absolutePath))),
      buffer: null,
    };
  }

  const buffer = fs.readFileSync(absolutePath);
  const stat = fs.statSync(absolutePath);
  const meta = {
    path: toPosix(path.relative(REPO_ROOT, absolutePath)),
    exists: true,
    bytes: buffer.length,
    hash: crypto.createHash(HASH_ALGORITHM).update(buffer).digest("hex"),
    modifiedAt: stat.mtime.toISOString(),
    version: null,
  };

  const ext = path.extname(absolutePath).toLowerCase();
  try {
    if (ext === ".yaml" || ext === ".yml") {
      const parsed = YAML.parse(buffer.toString("utf8")) ?? {};
      meta.version = parsed?.version ?? parsed?.info?.version ?? null;
    } else if (ext === ".json") {
      const parsed = JSON.parse(buffer.toString("utf8"));
      meta.version =
        parsed?.version ?? parsed?.info?.version ?? parsed?.$id ?? null;
    }
  } catch (error) {
    console.warn(
      `[contract-digest] 解析 ${meta.path} 失败: ${error?.message ?? error}`,
    );
  }

  return { meta, buffer };
}

function buildBundleHash(descriptorRecord, inputRecords, outputRecords) {
  const hash = crypto.createHash(HASH_ALGORITHM);
  let hasData = false;

  if (descriptorRecord?.buffer?.length) {
    hash.update(descriptorRecord.buffer);
    hasData = true;
  }
  for (const rec of inputRecords) {
    if (rec.buffer?.length) {
      hash.update(rec.buffer);
      hasData = true;
    }
  }
  for (const rec of outputRecords) {
    if (rec.buffer?.length) {
      hash.update(rec.buffer);
      hasData = true;
    }
  }

  return hasData ? hash.digest("hex") : null;
}

function buildMissingFileMeta(relPath) {
  return {
    path: relPath ?? null,
    exists: false,
    bytes: 0,
    hash: null,
    modifiedAt: null,
    version: null,
  };
}

function normalizeToArray(value) {
  if (!value) {
    return [];
  }
  if (Array.isArray(value)) {
    return value.map((item) => normalizePath(item)).filter(Boolean);
  }
  return [normalizePath(value)].filter(Boolean);
}

function normalizePath(value) {
  if (!value) return null;
  return toPosix(value);
}

function toPosix(p) {
  return p ? p.split(path.sep).join("/") : p;
}

function ensureDir(dir) {
  fs.mkdirSync(dir, { recursive: true });
}
