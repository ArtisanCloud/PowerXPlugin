import fs from "node:fs";
import path from "node:path";
import YAML from "yaml";

export interface CapabilityProvideEntry {
  id: string;
  version?: string;
  descriptor?: string;
  schemas?: {
    input?: string;
    output?: string;
  };
}

export interface PluginManifest {
  capabilities?: { provides: CapabilityProvideEntry[] };
  agent_tools?: Array<Record<string, any>>;
  exposure?: { channels: Array<Record<string, any>> };
  tools?: Array<Record<string, any>>;
}

export interface CapabilityInitOptions {
  manifestPath?: string;
  capabilityId: string;
  version?: string;
  descriptorPath?: string;
  inputSchemaPath?: string;
  outputSchemaPath?: string;
  handlerPath?: string;
  restPath?: string;
  method?: string;
  description?: string;
}

export interface CapabilityInitResult {
  manifestPath: string;
  descriptorPath: string;
  inputSchemaPath: string;
  outputSchemaPath: string;
}

export interface CapabilityLintResult {
  manifestPath: string;
  checked: number;
  errors: string[];
}

export function loadManifest(manifestPath: string): PluginManifest {
  const raw = fs.existsSync(manifestPath)
    ? fs.readFileSync(manifestPath, "utf8")
    : "";
  return raw ? (YAML.parse(raw) ?? {}) : {};
}

export function saveManifest(
  manifestPath: string,
  manifest: PluginManifest,
) {
  const serialized = YAML.stringify(manifest);
  fs.writeFileSync(manifestPath, serialized, "utf8");
}

function ensureCapabilityContainer(manifest: PluginManifest) {
  if (!manifest.capabilities) {
    manifest.capabilities = { provides: [] };
  } else if (!manifest.capabilities.provides) {
    manifest.capabilities.provides = [];
  }
}

function sanitizeFileName(capId: string) {
  return capId.replace(/[^A-Za-z0-9._-]/g, "-");
}

function deriveHandlerPath(capId: string) {
  const segments = capId.split(".");
  if (segments.length < 2) {
    return "backend/internal/handlers/capabilities/template/create_handler.go";
  }
  const domain = segments[segments.length - 2];
  const action = segments[segments.length - 1];
  return `backend/internal/handlers/capabilities/${domain}/${action}_handler.go`;
}

function ensureArraySection(manifest: PluginManifest, key: "agent_tools" | "tools") {
  if (!manifest[key]) {
    manifest[key] = [];
  }
}

export function ensureCapabilityEntry(
  options: CapabilityInitOptions,
): CapabilityInitResult {
  const manifestPath = path.resolve(options.manifestPath ?? "plugin.yaml");
  const manifestDir = path.dirname(manifestPath);
  const manifest = loadManifest(manifestPath);
  ensureCapabilityContainer(manifest);
  const provides = manifest.capabilities!.provides;

  if (provides.some((entry) => entry.id === options.capabilityId)) {
    throw new Error(
      `Capability ${options.capabilityId} 已存在于 manifest 中。`,
    );
  }

  const fileSafe = sanitizeFileName(options.capabilityId);
  const descriptorRelative =
    options.descriptorPath ?? `contracts/capabilities/${fileSafe}.yaml`;
  const inputSchemaRelative =
    options.inputSchemaPath ?? `contracts/schema/input/${fileSafe}.json`;
  const outputSchemaRelative =
    options.outputSchemaPath ?? `contracts/schema/output/${fileSafe}.json`;

  provides.push({
    id: options.capabilityId,
    version: options.version ?? "1.0.0",
    descriptor: descriptorRelative,
    schemas: {
      input: inputSchemaRelative,
      output: outputSchemaRelative,
    },
  });

  ensureArraySection(manifest, "agent_tools");
  const handlerPath = options.handlerPath ?? deriveHandlerPath(options.capabilityId);
  if (
    !manifest.agent_tools!.some(
      (tool) => tool?.capability === options.capabilityId,
    )
  ) {
    manifest.agent_tools!.push({
      id: options.capabilityId,
      capability: options.capabilityId,
      handler: handlerPath,
      description: options.description ?? "TODO: 描述该能力的 Agent 用途",
    });
  }

  if (!manifest.exposure) {
    manifest.exposure = { channels: [] };
  } else if (!manifest.exposure.channels) {
    manifest.exposure.channels = [];
  }

  const restPath = options.restPath ?? "/resources";
  const restEntry = `\${POWERX_PLUGIN_HTTP_BASE:-/api/v1}${
    restPath.startsWith("/") ? restPath : `/${restPath}`
  }`;

  const restExists = manifest.exposure.channels.some(
    (ch) => ch.type === "rest" && ch.capability === options.capabilityId,
  );
  if (!restExists) {
    manifest.exposure.channels.push({
      type: "rest",
      capability: options.capabilityId,
      entrypoint: restEntry,
      method: options.method ?? "POST",
      auth: "jwt",
      rbac: options.capabilityId.replace(/\.[^.]+$/, ":template"),
    });
  }
  const agentChannelExists = manifest.exposure.channels.some(
    (ch) => ch.type === "agent_tool" && ch.capability === options.capabilityId,
  );
  if (!agentChannelExists) {
    manifest.exposure.channels.push({
      type: "agent_tool",
      capability: options.capabilityId,
      agent_tool_id: options.capabilityId,
    });
  }

  ensureArraySection(manifest, "tools");
  if (
    !manifest.tools!.some(
      (tool) => tool?.id === options.capabilityId && tool.transport === "http",
    )
  ) {
    manifest.tools!.push({
      id: options.capabilityId,
      plugin_id: manifest?.id ?? "",
      name: options.description ?? "创建示例数据",
      description:
        options.description ??
        "TODO: 描述该能力如何被 REST / Agent Tool 调用",
      transport: "http",
      endpoint: restEntry,
      method: options.method ?? "POST",
      rbac_resource: options.capabilityId.replace(/\.[^.]+$/, ":template"),
    });
  }

  saveManifest(manifestPath, manifest);

  const descriptorPath = path.resolve(manifestDir, descriptorRelative);
  const inputSchemaPath = path.resolve(manifestDir, inputSchemaRelative);
  const outputSchemaPath = path.resolve(manifestDir, outputSchemaRelative);

  ensureDescriptorStub(options.capabilityId, options.version, descriptorPath, {
    input: inputSchemaRelative,
    output: outputSchemaRelative,
  });
  ensureSchemaStub(inputSchemaPath, `${options.capabilityId}Input`, true);
  ensureSchemaStub(outputSchemaPath, `${options.capabilityId}Output`, false);

  return {
    manifestPath,
    descriptorPath,
    inputSchemaPath,
    outputSchemaPath,
  };
}

function ensureDescriptorStub(
  capabilityId: string,
  version = "1.0.0",
  descriptorPath: string,
  schemas: { input: string; output: string },
) {
  if (fs.existsSync(descriptorPath)) {
    return;
  }
  fs.mkdirSync(path.dirname(descriptorPath), { recursive: true });
  const descriptor = {
    id: capabilityId,
    version,
    summary: "TODO: describe capability",
    description: "TODO: detailed description for consumers.",
    input: schemas.input,
    output: schemas.output,
    errors: [],
    rbac: {
      permissions: [capabilityId],
    },
  };
  fs.writeFileSync(descriptorPath, YAML.stringify(descriptor), "utf8");
}

function ensureSchemaStub(
  schemaPath: string,
  title: string,
  strict: boolean,
) {
  if (fs.existsSync(schemaPath)) {
    return;
  }
  fs.mkdirSync(path.dirname(schemaPath), { recursive: true });
  const schemaStub: Record<string, any> = {
    $schema: "https://json-schema.org/draft/2020-12/schema",
    title,
    type: "object",
    properties: {},
    required: [],
  };
  if (strict) {
    schemaStub.additionalProperties = false;
  }
  fs.writeFileSync(schemaPath, JSON.stringify(schemaStub, null, 2), "utf8");
}

export function lintCapabilities(options: {
  manifestPath?: string;
}): CapabilityLintResult {
  const manifestPath = path.resolve(options.manifestPath ?? "plugin.yaml");
  const manifestDir = path.dirname(manifestPath);
  const manifest = loadManifest(manifestPath);
  const provides = manifest?.capabilities?.provides ?? [];
  const errors: string[] = [];
  const capabilitySet = new Set(provides.map((entry) => entry.id));

  for (const cap of provides) {
    if (!cap?.id) {
      errors.push("存在缺少 id 的 capability 条目");
      continue;
    }
    const descriptorRel =
      cap.descriptor ?? `contracts/capabilities/${sanitizeFileName(cap.id)}.yaml`;
    const descriptorPath = path.resolve(manifestDir, descriptorRel);
    if (!fs.existsSync(descriptorPath)) {
      errors.push(`未找到 descriptor: ${descriptorPath}`);
    } else {
      try {
        const doc = YAML.parse(fs.readFileSync(descriptorPath, "utf8"));
        if (doc?.id && doc.id !== cap.id) {
          errors.push(
            `descriptor ${descriptorRel} 的 id (${doc.id}) 与 manifest (${cap.id}) 不一致`,
          );
        }
      } catch (err) {
        errors.push(`解析 descriptor 失败 ${descriptorRel}: ${(err as Error).message}`);
      }
    }

    const inputSchemaPath = path.resolve(manifestDir, cap.schemas?.input ?? "");
    if (!cap.schemas?.input || !fs.existsSync(inputSchemaPath)) {
      errors.push(`缺少 input schema: ${cap.schemas?.input ?? "(未配置)"}`);
    }
    const outputSchemaPath = path.resolve(manifestDir, cap.schemas?.output ?? "");
    if (!cap.schemas?.output || !fs.existsSync(outputSchemaPath)) {
      errors.push(`缺少 output schema: ${cap.schemas?.output ?? "(未配置)"}`);
    }
  }

  const channels = manifest?.exposure?.channels ?? [];
  for (const channel of channels) {
    if (channel?.capability && !capabilitySet.has(channel.capability)) {
      errors.push(
        `exposure channel (${channel.type ?? "unknown"}) 引用了不存在的 capability ${channel.capability}`,
      );
    }
  }

  const agentTools = manifest?.agent_tools ?? [];
  for (const tool of agentTools) {
    if (tool?.capability && !capabilitySet.has(tool.capability)) {
      errors.push(`agent_tool ${tool.id ?? "unknown"} 引用了不存在的 capability`);
    }
  }

  return {
    manifestPath,
    checked: provides.length,
    errors,
  };
}
