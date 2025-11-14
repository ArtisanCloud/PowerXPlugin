import fs from "node:fs";
import path from "node:path";
import crypto from "node:crypto";
import { spawnSync } from "node:child_process";
import YAML from "yaml";
import type { PluginManifest } from "../../lib/capabilities/manager";
import { TelemetryEmitter } from "../../lib/telemetry/emitter";

export interface CapabilitiesDiffOptions {
  from?: string;
  to?: string;
  manifestPath?: string;
  outputPath?: string;
  rootDir?: string;
}

export interface CapabilityDiffReport {
  fromLabel: string;
  toLabel: string;
  generatedAt: string;
  reportPath: string;
  changes: CapabilityChange[];
}

interface SnapshotContext {
  kind: "fs" | "git";
  label: string;
  manifestDir: string;
  manifestDirPosix?: string;
  repoRoot?: string;
  ref?: string;
  manifest: PluginManifest;
  manifestPath: string;
}

interface CapabilitySnapshot {
  id: string;
  version?: string;
  descriptorSummary?: string;
  descriptorHash?: string;
  rbacPermissions: string[];
  inputSchema?: SchemaSummary;
  outputSchema?: SchemaSummary;
  channels: CapabilityChannelSummary[];
  agentTools: CapabilityAgentToolSummary[];
}

interface SchemaSummary {
  path?: string;
  hash?: string;
  fields: string[];
}

interface CapabilityChannelSummary {
  type?: string;
  capability?: string;
  agent_tool_id?: string;
  entrypoint?: string;
  method?: string;
  auth?: string;
}

interface CapabilityAgentToolSummary {
  id?: string;
  capability?: string;
  handler?: string;
  description?: string;
}

interface CapabilityChange {
  id: string;
  type: "added" | "removed" | "modified";
  version?: {
    from?: string;
    to?: string;
  };
  descriptorChanged?: boolean;
  descriptorSummary?: {
    from?: string;
    to?: string;
  };
  rbac?: {
    added: string[];
    removed: string[];
  };
  inputSchema?: SchemaDiff;
  outputSchema?: SchemaDiff;
  channels?: ChannelsDiff;
  agentTools?: ChannelsDiff;
}

interface SchemaDiff {
  from?: SchemaSummary;
  to?: SchemaSummary;
  fieldsAdded: string[];
  fieldsRemoved: string[];
}

interface ChannelsDiff {
  added: CapabilityChannelSummary[];
  removed: CapabilityChannelSummary[];
  changed: Array<{
    from: CapabilityChannelSummary;
    to: CapabilityChannelSummary;
  }>;
}

export async function runCapabilitiesDiffCommand(
  options: CapabilitiesDiffOptions,
): Promise<CapabilityDiffReport> {
  const rootDir = options.rootDir ?? process.cwd();
  const repoRoot = detectGitRoot(rootDir);
  const manifestPath = options.manifestPath ?? "plugin.yaml";
  const manifestRelPosix =
    repoRoot && path.isAbsolute(manifestPath)
      ? path.posix.join(
          ...path
            .relative(repoRoot, manifestPath)
            .split(path.sep)
            .filter(Boolean),
        )
      : manifestPath.replace(/\\/g, "/");

  const toIdentifier = options.to ?? manifestPath;
  const fromIdentifier =
    options.from || (repoRoot ? `HEAD~1:${manifestRelPosix}` : undefined);

  if (!fromIdentifier) {
    throw new Error(
      "缺少 --from 选项，且未检测到 Git 仓库。请显式提供旧版 manifest。",
    );
  }

  const toSnapshot = loadSnapshot(toIdentifier, {
    rootDir,
    repoRoot,
  });
  const fromSnapshot = loadSnapshot(fromIdentifier, {
    rootDir,
    repoRoot,
  });

  const diff = diffCapabilities(fromSnapshot, toSnapshot);
  const markdown = renderReportMarkdown({
    from: fromSnapshot.label,
    to: toSnapshot.label,
    generatedAt: new Date().toISOString(),
    changes: diff,
  });

  const reportPath =
    options.outputPath ??
    path.resolve(rootDir, "release", "capabilities-change-report.md");
  fs.mkdirSync(path.dirname(reportPath), { recursive: true });
  fs.writeFileSync(reportPath, markdown, "utf8");

  TelemetryEmitter.emitCapabilityEvent({
    type: "capability.cli.diff_total",
    from: fromSnapshot.label,
    to: toSnapshot.label,
    capabilities: diff.length,
    reportPath,
  });

  return {
    fromLabel: fromSnapshot.label,
    toLabel: toSnapshot.label,
    generatedAt: new Date().toISOString(),
    reportPath,
    changes: diff,
  };
}

function detectGitRoot(rootDir: string) {
  const result = spawnSync("git", ["rev-parse", "--show-toplevel"], {
    cwd: rootDir,
    encoding: "utf8",
  });
  if (result.status !== 0) {
    return null;
  }
  return result.stdout.trim();
}

function loadSnapshot(
  identifier: string,
  context: { rootDir: string; repoRoot: string | null },
): SnapshotContext {
  const asPath = resolveIfPath(identifier, context.rootDir);
  if (asPath) {
    const manifestRaw = fs.readFileSync(asPath, "utf8");
    return {
      kind: "fs",
      label: asPath,
      manifestPath: asPath,
      manifestDir: path.dirname(asPath),
      manifest: YAML.parse(manifestRaw) ?? {},
    };
  }
  const parsed = parseGitIdentifier(identifier);
  if (!context.repoRoot) {
    throw new Error(`无法解析 ${identifier}：该仓库未初始化 Git。`);
  }
  const manifestRaw = readGitFile(
    context.repoRoot,
    parsed.ref,
    parsed.filePath,
  );
  return {
    kind: "git",
    label: identifier,
    manifestPath: parsed.filePath,
    manifestDir: path.posix.dirname(parsed.filePath),
    manifestDirPosix: path.posix.dirname(parsed.filePath),
    repoRoot: context.repoRoot,
    ref: parsed.ref,
    manifest: YAML.parse(manifestRaw) ?? {},
  };
}

function resolveIfPath(identifier: string, rootDir: string) {
  const candidate = path.isAbsolute(identifier)
    ? identifier
    : path.resolve(rootDir, identifier);
  return fs.existsSync(candidate) ? candidate : null;
}

function parseGitIdentifier(identifier: string) {
  const idx = identifier.indexOf(":");
  if (idx === -1) {
    throw new Error(
      `无法解析 ${identifier}：请使用类似 HEAD~1:plugin.yaml 的 Git 语法或提供文件路径`,
    );
  }
  const ref = identifier.slice(0, idx);
  const filePath = identifier.slice(idx + 1);
  if (!filePath) {
    throw new Error(`Git 标识 ${identifier} 缺少文件路径部分`);
  }
  return { ref, filePath };
}

function readGitFile(repoRoot: string, ref: string, filePath: string) {
  const spec = `${ref}:${filePath}`;
  const result = spawnSync("git", ["show", spec], {
    cwd: repoRoot,
    encoding: "utf8",
  });
  if (result.status !== 0) {
    throw new Error(
      `无法读取 ${spec}：${result.stderr?.trim() || "git show 失败"}`,
    );
  }
  return result.stdout;
}

function readRelativeFile(source: SnapshotContext, relativePath?: string) {
  if (!relativePath) {
    return null;
  }
  const normalized = relativePath.replace(/\\/g, "/");
  if (source.kind === "fs") {
    const target = path.resolve(source.manifestDir, normalized);
    return fs.existsSync(target) ? fs.readFileSync(target, "utf8") : null;
  }
  const manifestDir = source.manifestDirPosix ?? ".";
  const joined = path.posix.join(
    manifestDir,
    normalized.startsWith("./") ? normalized.slice(2) : normalized,
  );
  if (!source.repoRoot || !source.ref) {
    return null;
  }
  try {
    return readGitFile(source.repoRoot, source.ref, joined);
  } catch {
    return null;
  }
}

function hashContent(content?: string | null) {
  if (!content) {
    return undefined;
  }
  return crypto.createHash("sha256").update(content).digest("hex");
}

function summarizeSchema(
  raw?: string | null,
  schemaPath?: string,
): SchemaSummary | undefined {
  if (!raw) {
    return undefined;
  }
  try {
    const doc = JSON.parse(raw);
    const properties = doc?.properties;
    const fields = properties && typeof properties === "object"
      ? Object.keys(properties).sort()
      : [];
    return {
      path: schemaPath,
      hash: hashContent(raw),
      fields,
    };
  } catch {
    return {
      path: schemaPath,
      hash: hashContent(raw),
      fields: [],
    };
  }
}

function snapshotCapabilities(context: SnapshotContext) {
  const provides = context.manifest?.capabilities?.provides ?? [];
  const channels = context.manifest?.exposure?.channels ?? [];
  const agentTools = context.manifest?.agent_tools ?? [];
  const entries: Record<string, CapabilitySnapshot> = {};

  for (const entry of provides) {
    if (!entry?.id) {
      continue;
    }
    const descriptorPath =
      entry.descriptor ??
      `contracts/capabilities/${sanitizeFileName(entry.id)}.yaml`;
    const descriptorRaw = readRelativeFile(context, descriptorPath);
    const descriptor = descriptorRaw ? YAML.parse(descriptorRaw) ?? {} : {};
    const descriptorSummary = descriptor?.summary || descriptor?.description;
    const rbacPermissions = Array.isArray(descriptor?.rbac?.permissions)
      ? [...descriptor.rbac.permissions]
      : [];

    const inputSchemaPath = entry.schemas?.input;
    const outputSchemaPath = entry.schemas?.output;
    const inputSchemaRaw = readRelativeFile(context, inputSchemaPath);
    const outputSchemaRaw = readRelativeFile(context, outputSchemaPath);

    const snapshot: CapabilitySnapshot = {
      id: entry.id,
      version: entry.version,
      descriptorSummary,
      descriptorHash: hashContent(descriptorRaw ?? undefined),
      rbacPermissions: rbacPermissions.sort(),
      inputSchema: summarizeSchema(inputSchemaRaw, inputSchemaPath ?? undefined),
      outputSchema: summarizeSchema(
        outputSchemaRaw,
        outputSchemaPath ?? undefined,
      ),
      channels: channels
        .filter((channel: CapabilityChannelSummary) => {
          return (
            channel?.capability === entry.id ||
            channel?.agent_tool_id === entry.id
          );
        })
        .map(normalizeChannel),
      agentTools: agentTools
        .filter((tool: CapabilityAgentToolSummary) => {
          return tool?.capability === entry.id || tool?.id === entry.id;
        })
        .map(normalizeAgentTool),
    };
    entries[entry.id] = snapshot;
  }
  return entries;
}

function normalizeChannel(channel: CapabilityChannelSummary) {
  return {
    type: channel?.type,
    capability: channel?.capability,
    agent_tool_id: channel?.agent_tool_id,
    entrypoint: channel?.entrypoint,
    method: channel?.method,
    auth: channel?.auth,
  };
}

function normalizeAgentTool(tool: CapabilityAgentToolSummary) {
  return {
    id: tool?.id,
    capability: tool?.capability,
    handler: tool?.handler,
    description: tool?.description,
  };
}

function diffCapabilities(
  fromCtx: SnapshotContext,
  toCtx: SnapshotContext,
): CapabilityChange[] {
  const fromCaps = snapshotCapabilities(fromCtx);
  const toCaps = snapshotCapabilities(toCtx);
  const ids = new Set([
    ...Object.keys(fromCaps),
    ...Object.keys(toCaps),
  ]);
  const results: CapabilityChange[] = [];

  for (const id of Array.from(ids).sort()) {
    const before = fromCaps[id];
    const after = toCaps[id];
    if (!before && after) {
      results.push({
        id,
        type: "added",
        version: { to: after.version },
        descriptorSummary: { to: after.descriptorSummary },
        inputSchema: after.inputSchema
          ? { to: after.inputSchema, fieldsAdded: after.inputSchema.fields, fieldsRemoved: [] }
          : undefined,
        outputSchema: after.outputSchema
          ? { to: after.outputSchema, fieldsAdded: after.outputSchema.fields, fieldsRemoved: [] }
          : undefined,
        channels: channelsDiff([], after.channels),
        agentTools: channelsDiff([], after.agentTools),
        rbac: {
          added: after.rbacPermissions,
          removed: [],
        },
      });
      continue;
    }
    if (before && !after) {
      results.push({
        id,
        type: "removed",
        version: { from: before.version },
        descriptorSummary: { from: before.descriptorSummary },
        inputSchema: before.inputSchema
          ? { from: before.inputSchema, fieldsAdded: [], fieldsRemoved: before.inputSchema.fields }
          : undefined,
        outputSchema: before.outputSchema
          ? { from: before.outputSchema, fieldsAdded: [], fieldsRemoved: before.outputSchema.fields }
          : undefined,
        channels: channelsDiff(before.channels, []),
        agentTools: channelsDiff(before.agentTools, []),
        rbac: {
          added: [],
          removed: before.rbacPermissions,
        },
      });
      continue;
    }
    if (!before || !after) {
      continue;
    }
    const schemaDiffIn = schemaDiff(before.inputSchema, after.inputSchema);
    const schemaDiffOut = schemaDiff(before.outputSchema, after.outputSchema);
    const channelDiff = channelsDiff(before.channels, after.channels);
    const toolDiff = channelsDiff(before.agentTools, after.agentTools);
    const rbacDiff = diffLists(before.rbacPermissions, after.rbacPermissions);
    const descriptorChanged = before.descriptorHash !== after.descriptorHash;

    const changed =
      before.version !== after.version ||
      descriptorChanged ||
      schemaDiffIn?.fieldsAdded.length ||
      schemaDiffIn?.fieldsRemoved.length ||
      schemaDiffOut?.fieldsAdded.length ||
      schemaDiffOut?.fieldsRemoved.length ||
      channelDiff.added.length ||
      channelDiff.removed.length ||
      channelDiff.changed.length ||
      toolDiff.added.length ||
      toolDiff.removed.length ||
      toolDiff.changed.length ||
      rbacDiff.added.length ||
      rbacDiff.removed.length;

    if (!changed) {
      continue;
    }

    results.push({
      id,
      type: "modified",
      version: { from: before.version, to: after.version },
      descriptorChanged,
      descriptorSummary: {
        from: before.descriptorSummary,
        to: after.descriptorSummary,
      },
      inputSchema: schemaDiffIn,
      outputSchema: schemaDiffOut,
      channels: channelDiff,
      agentTools: toolDiff,
      rbac: rbacDiff,
    });
  }

  return results;
}

function schemaDiff(from?: SchemaSummary, to?: SchemaSummary): SchemaDiff | undefined {
  if (!from && !to) {
    return undefined;
  }
  const fieldDiff = diffLists(from?.fields ?? [], to?.fields ?? []);
  const added = fieldDiff.added;
  const removed = fieldDiff.removed;
  if (
    (from?.hash === to?.hash || (!from && !to)) &&
    added.length === 0 &&
    removed.length === 0
  ) {
    return undefined;
  }
  return {
    from,
    to,
    fieldsAdded: added,
    fieldsRemoved: removed,
  };
}

function diffLists(before: string[], after: string[]) {
  const beforeSet = new Set(before);
  const afterSet = new Set(after);
  const added: string[] = [];
  const removed: string[] = [];
  for (const value of afterSet) {
    if (!beforeSet.has(value)) {
      added.push(value);
    }
  }
  for (const value of beforeSet) {
    if (!afterSet.has(value)) {
      removed.push(value);
    }
  }
  added.sort();
  removed.sort();
  return { added, removed };
}

function channelsDiff<T extends CapabilityChannelSummary | CapabilityAgentToolSummary>(
  before: T[],
  after: T[],
): ChannelsDiff {
  const beforeMap = new Map<string, T>();
  const afterMap = new Map<string, T>();
  for (const entry of before) {
    beforeMap.set(channelKey(entry), entry);
  }
  for (const entry of after) {
    afterMap.set(channelKey(entry), entry);
  }
  const added: T[] = [];
  const removed: T[] = [];
  const changed: Array<{ from: T; to: T }> = [];

  for (const [key, entry] of beforeMap.entries()) {
    if (!afterMap.has(key)) {
      removed.push(entry);
      continue;
    }
    const candidate = afterMap.get(key)!;
    if (JSON.stringify(entry) !== JSON.stringify(candidate)) {
      changed.push({ from: entry, to: candidate });
    }
  }
  for (const [key, entry] of afterMap.entries()) {
    if (!beforeMap.has(key)) {
      added.push(entry);
    }
  }
  return { added, removed, changed };
}

function channelKey(entry: CapabilityChannelSummary | CapabilityAgentToolSummary) {
  const typeKey = "type" in entry ? entry.type ?? "unknown" : "tool";
  const idKey =
    ("capability" in entry && entry.capability) ||
    ("id" in entry && entry.id) ||
    ("agent_tool_id" in entry && entry.agent_tool_id) ||
    "";
  return `${typeKey}:${idKey}`;
}

function renderReportMarkdown(input: {
  from: string;
  to: string;
  generatedAt: string;
  changes: CapabilityChange[];
}) {
  const lines: string[] = [];
  lines.push("# Capability Change Report");
  lines.push("");
  lines.push(`- From: \`${input.from}\``);
  lines.push(`- To: \`${input.to}\``);
  lines.push(`- Generated: ${input.generatedAt}`);
  lines.push("");
  lines.push("## Summary");
  lines.push("");
  lines.push("| Capability | Change | Version (from → to) |");
  lines.push("|------------|--------|---------------------|");
  for (const change of input.changes) {
    const version =
      change.version?.from || change.version?.to
        ? `${change.version?.from ?? "-"} → ${change.version?.to ?? "-"}`
        : "-";
    lines.push(
      `| \`${change.id}\` | ${change.type} | ${version} |`,
    );
  }
  if (input.changes.length === 0) {
    lines.push("| _none_ | - | - |");
  }
  lines.push("");
  lines.push("## Detailed Changes");
  lines.push("");
  for (const change of input.changes) {
    lines.push(`### ${change.id}`);
    lines.push(`- Change type: **${change.type}**`);
    if (change.version) {
      lines.push(
        `- Version: ${change.version.from ?? "-"} → ${change.version.to ?? "-"}`,
      );
    }
    if (change.descriptorSummary) {
      lines.push(
        `- Descriptor summary: ${change.descriptorSummary.from ?? "-"} → ${change.descriptorSummary.to ?? "-"}`,
      );
    }
    if (change.descriptorChanged) {
      lines.push("- Descriptor payload changed (YAML diff detected)");
    }
    if (change.rbac && (change.rbac.added.length || change.rbac.removed.length)) {
      lines.push("  - RBAC permissions:");
      if (change.rbac.added.length) {
        lines.push(`    - Added: ${change.rbac.added.join(", ")}`);
      }
      if (change.rbac.removed.length) {
        lines.push(`    - Removed: ${change.rbac.removed.join(", ")}`);
      }
    }
    appendSchemaSection(lines, "Input schema", change.inputSchema);
    appendSchemaSection(lines, "Output schema", change.outputSchema);
    appendChannelSection(lines, "Exposure channels", change.channels);
    appendChannelSection(lines, "Agent tools", change.agentTools);
    lines.push("");
  }
  lines.push("## 灰度 / 通知计划模板");
  lines.push("");
  lines.push("```yaml");
  lines.push("release:");
  lines.push("  window: TBD");
  lines.push("  owner: plugin-team@powerx.local");
  lines.push("  capabilities:");
  for (const change of input.changes) {
    lines.push(`    - id: ${change.id}`);
    lines.push(`      change: ${change.type}`);
    lines.push("      notify:");
    lines.push("        channels: [email, slack]");
    lines.push("        subscribers:");
    lines.push("          - team: platform");
    lines.push("            contact: platform@powerx.local");
    lines.push("      rollout:");
    lines.push("        - name: canary");
    lines.push("          percentage: 5");
    lines.push("          duration: 2h");
    lines.push("        - name: production");
    lines.push("          percentage: 100");
    lines.push("          duration: 4h");
    lines.push("      rollback:");
    lines.push("        trigger: error_rate > 2%");
    lines.push(
      `        action: revert capability ${change.id} to ${change.version?.from ?? "previous-approved"}`,
    );
  }
  if (!input.changes.length) {
    lines.push("    - id: (none)");
    lines.push("      change: noop");
    lines.push("      notify:");
    lines.push("        channels: []");
  }
  lines.push("```");
  lines.push("");
  return lines.join("\n");
}

function appendSchemaSection(
  lines: string[],
  title: string,
  diff?: SchemaDiff,
) {
  if (!diff) {
    return;
  }
  lines.push(`  - ${title}:`);
  if (diff.from?.path || diff.to?.path) {
    lines.push(
      `    - Path: ${diff.from?.path ?? "-"} → ${diff.to?.path ?? "-"}`,
    );
  }
  if (diff.from?.hash || diff.to?.hash) {
    lines.push(
      `    - Hash: ${diff.from?.hash ?? "-"} → ${diff.to?.hash ?? "-"}`,
    );
  }
  if (diff.fieldsAdded.length) {
    lines.push(`    - Fields added: ${diff.fieldsAdded.join(", ")}`);
  }
  if (diff.fieldsRemoved.length) {
    lines.push(`    - Fields removed: ${diff.fieldsRemoved.join(", ")}`);
  }
}

function appendChannelSection(
  lines: string[],
  title: string,
  diff?: ChannelsDiff,
) {
  if (!diff) {
    return;
  }
  if (
    !diff.added.length &&
    !diff.removed.length &&
    !diff.changed.length
  ) {
    return;
  }
  lines.push(`  - ${title}:`);
  if (diff.added.length) {
    lines.push(
      `    - Added: ${diff.added.map((c) => describeChannel(c)).join("; ")}`,
    );
  }
  if (diff.removed.length) {
    lines.push(
      `    - Removed: ${diff.removed.map((c) => describeChannel(c)).join("; ")}`,
    );
  }
  if (diff.changed.length) {
    for (const item of diff.changed) {
      lines.push(
        `    - Changed: ${describeChannel(item.from)} → ${describeChannel(item.to)}`,
      );
    }
  }
}

function describeChannel(
  entry: CapabilityChannelSummary | CapabilityAgentToolSummary,
) {
  if ("type" in entry) {
    return [
      entry.type,
      entry.method ?? "",
      entry.entrypoint ?? "",
      entry.capability ?? entry.agent_tool_id ?? "",
    ]
      .filter(Boolean)
      .join(" ");
  }
  return `${entry.id ?? entry.capability ?? "tool"} (${entry.handler ?? "handler"})`;
}

function sanitizeFileName(id: string) {
  return id.replace(/[^A-Za-z0-9._-]/g, "-");
}
