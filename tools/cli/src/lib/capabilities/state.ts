import fs from "node:fs";
import path from "node:path";
import { loadManifest } from "./manager";

export type CapabilityStatus = "approved" | "pending" | "rejected" | "unknown";

export interface CapabilityStateEntry {
  id: string;
  status: CapabilityStatus;
  lastSubmittedAt?: string;
  note?: string;
}

export interface CapabilityState {
  entries: Record<string, CapabilityStateEntry>;
}

const STATE_DIR = ".px-plugin";
const STATE_FILE = "capabilities.json";
const AUDIT_DIR = path.join(STATE_DIR, "audit");

function getStateFilePath(rootDir = process.cwd()) {
  return path.resolve(rootDir, STATE_DIR, STATE_FILE);
}

export function readCapabilityState(rootDir = process.cwd()): CapabilityState {
  const statePath = getStateFilePath(rootDir);
  if (!fs.existsSync(statePath)) {
    return { entries: {} };
  }
  try {
    const raw = fs.readFileSync(statePath, "utf8");
    const parsed = JSON.parse(raw);
    return parsed?.entries ? parsed : { entries: {} };
  } catch {
    return { entries: {} };
  }
}

export function writeCapabilityState(
  state: CapabilityState,
  rootDir = process.cwd(),
) {
  const dir = path.join(rootDir, STATE_DIR);
  fs.mkdirSync(dir, { recursive: true });
  const statePath = getStateFilePath(rootDir);
  fs.writeFileSync(statePath, JSON.stringify(state, null, 2), "utf8");
}

export function appendCapabilityAuditLog(
  capabilityId: string,
  payload: Record<string, any>,
  rootDir = process.cwd(),
) {
  const auditDir = path.resolve(rootDir, AUDIT_DIR);
  fs.mkdirSync(auditDir, { recursive: true });
  const fileName = `${capabilityId.replace(/[^A-Za-z0-9._-]/g, "-")}-${
    Date.now() / 1000
  }.log`;
  const filePath = path.join(auditDir, fileName);
  fs.writeFileSync(
    filePath,
    JSON.stringify(
      {
        capabilityId,
        timestamp: new Date().toISOString(),
        payload,
      },
      null,
      2,
    ),
    "utf8",
  );
}

export function updateCapabilityStateEntry(
  entry: CapabilityStateEntry,
  rootDir = process.cwd(),
) {
  const state = readCapabilityState(rootDir);
  state.entries[entry.id] = entry;
  writeCapabilityState(state, rootDir);
}

export function checkBlockingCapabilities(options: {
  manifestPath?: string;
  rootDir?: string;
}): string[] {
  const manifestPath = path.resolve(options.manifestPath ?? "plugin.yaml");
  const manifest = loadManifest(manifestPath);
  const provides = manifest?.capabilities?.provides ?? [];
  const capabilityIds = provides.map((cap) => cap.id);
  if (capabilityIds.length === 0) {
    return [];
  }
  const state = readCapabilityState(options.rootDir);
  const blocking: string[] = [];
  for (const id of capabilityIds) {
    const status = state.entries[id]?.status ?? "unknown";
    if (status === "pending" || status === "rejected" || status === "unknown") {
      blocking.push(id);
    }
  }
  return blocking;
}

export function assertCapabilitiesApproved(options: {
  manifestPath?: string;
  rootDir?: string;
}) {
  const blocking = checkBlockingCapabilities(options);
  if (blocking.length) {
    throw new Error(
      [
        "检测到未获批准的 capability：",
        ...blocking.map((id) => ` - ${id}`),
        "",
        "请先执行 `px-plugin capabilities submit` 并等待审核通过，再运行该命令。",
      ].join("\n"),
    );
  }
}
