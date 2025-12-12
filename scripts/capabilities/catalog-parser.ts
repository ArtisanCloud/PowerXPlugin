import { readFileSync, writeFileSync, existsSync, mkdirSync } from "node:fs";
import path from "node:path";
import crypto from "node:crypto";
import process from "node:process";
import YAML from "yaml";

type CapabilityFile = {
  id?: string;
  version?: string;
  descriptor?: string;
  summary?: string;
  description?: string;
  schemas?: Record<string, string>;
  protocols?: Record<string, unknown>;
  tags?: string[];
  execution?: ExecutionConfig;
};

type ExecutionConfig = {
  mode?: string;
  callback_url?: string;
  sse_channel?: string;
  status_endpoint?: string;
  timeout_seconds?: number;
};

interface CatalogSnapshot {
  plugin_id: string;
  manifest_version: string;
  generated_at: string;
  imports: string[];
  entries: CatalogEntry[];
}

interface CatalogEntry {
  id: string;
  version: string;
  descriptor: string;
  schemas: Record<string, string>;
  protocols: Record<string, unknown>;
  tags: string[];
  execution: RequiredExecutionConfig;
  checksum: string;
}

interface RequiredExecutionConfig extends ExecutionConfig {
  mode: string;
}

const args = process.argv.slice(2);
const getArg = (flag: string, fallback?: string): string | undefined => {
  const idx = args.indexOf(flag);
  if (idx === -1) return fallback;
  return args[idx + 1];
};

const manifestPath = path.resolve(
  process.cwd(),
  getArg("--manifest", path.join("..", "..", "plugin.yaml"))!,
);

if (!existsSync(manifestPath)) {
  console.warn(`[capabilities] manifest not found: ${manifestPath}, skipping catalog generation.`);
  process.exit(0);
}

const manifestDir = path.dirname(manifestPath);
const manifestRaw = readFileSync(manifestPath, "utf8");
const manifest = YAML.parse(manifestRaw) ?? {};
const pluginID: string = manifest.id ?? "";
const manifestVersion: string = manifest.version ?? "";

const imports: string[] = Array.isArray(manifest?.capabilities?.imports)
  ? manifest.capabilities.imports.filter((item: unknown): item is string => typeof item === "string")
  : [];

const entries: CatalogEntry[] = [];
const seenIDs = new Set<string>();

for (const importPath of imports) {
  const abs = path.resolve(manifestDir, importPath);
  if (!existsSync(abs)) {
    console.warn(`[capabilities] import missing: ${importPath}`);
    continue;
  }
  const yamlRaw = readFileSync(abs, "utf8");
  const descriptor = (YAML.parse(yamlRaw) ?? {}) as CapabilityFile;
  const entry = buildCatalogEntry(descriptor, {
    descriptorPath: toPosix(path.relative(manifestDir, abs)),
    manifest,
    fileContent: yamlRaw,
  });
  if (seenIDs.has(entry.id)) {
    console.warn(`[capabilities] duplicate capability id detected: ${entry.id}`);
  } else {
    seenIDs.add(entry.id);
    entries.push(entry);
  }
}

// fallback to manifest.capabilities.provides when imports are absent
if (entries.length === 0 && Array.isArray(manifest?.capabilities?.provides)) {
  for (const cap of manifest.capabilities.provides) {
    if (!cap?.id) continue;
    const descriptor = {
      ...cap,
    } as CapabilityFile;
    const entry = buildCatalogEntry(descriptor, {
      descriptorPath: descriptor.descriptor ?? "",
      manifest,
      fileContent: YAML.stringify(descriptor),
    });
    if (seenIDs.has(entry.id)) continue;
    seenIDs.add(entry.id);
    entries.push(entry);
  }
}

const snapshot: CatalogSnapshot = {
  plugin_id: pluginID,
  manifest_version: manifestVersion,
  generated_at: new Date().toISOString(),
  imports,
  entries,
};

const outputPath = path.resolve(
  manifestDir,
  getArg("--output", path.join("capabilities", "catalog.json"))!,
);
mkdirSync(path.dirname(outputPath), { recursive: true });
writeFileSync(outputPath, JSON.stringify(snapshot, null, 2), "utf8");
console.log(`[capabilities] catalog generated: ${toPosix(path.relative(process.cwd(), outputPath))}`);

function buildCatalogEntry(
  descriptor: CapabilityFile,
  options: { descriptorPath: string; manifest: any; fileContent: string },
): CatalogEntry {
  const schemas: Record<string, string> = descriptor.schemas ?? {};
  const execution = normalizeExecution(descriptor.execution ?? {});
  const checksum = checksumContent(options.fileContent);

  return {
    id: descriptor.id ?? "",
    version: descriptor.version ?? options.manifest.version ?? "1.0.0",
    descriptor: options.descriptorPath,
    schemas,
    protocols: descriptor.protocols ?? {},
    tags: descriptor.tags ?? [],
    execution,
    checksum,
  };
}

function normalizeExecution(exec: ExecutionConfig): RequiredExecutionConfig {
  const mode = (exec.mode ?? "sync").toLowerCase();
  return {
    mode,
    callback_url: exec.callback_url ?? "",
    sse_channel: exec.sse_channel ?? "",
    status_endpoint: exec.status_endpoint ?? "",
    timeout_seconds: exec.timeout_seconds ?? 0,
  };
}

function checksumContent(content: string): string {
  return crypto.createHash("sha256").update(content).digest("hex");
}

function toPosix(p: string): string {
  return p.split(path.sep).join("/");
}
