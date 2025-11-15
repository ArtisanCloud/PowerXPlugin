import path from "node:path";
import { promises as fs } from "node:fs";

export interface PrecheckContext {
  manifest: any;
  channel: "stable" | "beta";
  notes: string;
  changelogPath?: string;
}

export async function runPrechecks(context: PrecheckContext): Promise<void> {
  ensureField(context.manifest, "id");
  ensureField(context.manifest, "version");
  ensurePermissions(context.manifest);
  ensureSemanticVersion(context.manifest.version);
  if (context.channel === "stable") {
    ensureStableRequirements(context);
  }
  await ensureChangelogExists(context.changelogPath);
}

function ensureField(manifest: any, field: string) {
  if (!manifest[field]) {
    throw new Error(`manifest missing required field: ${field}`);
  }
}

function ensurePermissions(manifest: any) {
  if (!Array.isArray(manifest.permissions)) {
    throw new Error("manifest.permissions must be an array");
  }
  const duplicates = new Set<string>();
  manifest.permissions.forEach((perm: string) => {
    if (duplicates.has(perm)) {
      throw new Error(`duplicate permission declaration: ${perm}`);
    }
    duplicates.add(perm);
  });
}

function ensureSemanticVersion(version: string) {
  const semverRegex = /^(\d+)\.(\d+)\.(\d+)(-.+)?$/;
  if (!semverRegex.test(version)) {
    throw new Error(`version ${version} is not a valid semantic version`);
  }
}

function ensureStableRequirements(context: PrecheckContext) {
  if (!context.notes.trim()) {
    throw new Error("stable releases must include release notes");
  }
}

async function ensureChangelogExists(changelogPath?: string) {
  if (!changelogPath) {
    return;
  }
  const resolved = path.resolve(changelogPath);
  try {
    await fs.access(resolved);
  } catch {
    throw new Error(`changelog file not found: ${resolved}`);
  }
}
