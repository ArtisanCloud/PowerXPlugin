#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import YAML from "yaml";

function getArgValue(flag) {
  const idx = process.argv.indexOf(flag);
  if (idx === -1) return undefined;
  return process.argv[idx + 1];
}

function toPosix(p) {
  return p.split(path.sep).join("/");
}

function ensureDir(dir) {
  fs.mkdirSync(dir, { recursive: true });
}

function sanitizeFileName(capId) {
  return capId.replace(/[^A-Za-z0-9._-]/g, "-");
}

function isNonEmptyString(value) {
  return typeof value === "string" && value.trim() !== "";
}

function dedupeStrings(values) {
  const seen = new Set();
  const result = [];
  for (const value of values) {
    if (!isNonEmptyString(value)) continue;
    if (seen.has(value)) continue;
    seen.add(value);
    result.push(value);
  }
  return result;
}

const cwd = process.cwd();
const manifestArg = getArgValue("--manifest");
let manifestPath = path.resolve(cwd, manifestArg ?? "plugin.yaml");
if (!fs.existsSync(manifestPath) && !manifestArg) {
  const fallbacks = [
    {
      path: path.resolve(cwd, "..", "..", "skeleton", "plugin.yaml"),
      note: "../../skeleton/plugin.yaml",
    },
    {
      path: path.resolve(cwd, "..", "..", "plugin.yaml"),
      note: "../../plugin.yaml",
    },
  ];
  for (const candidate of fallbacks) {
    if (fs.existsSync(candidate.path)) {
      console.log(
        `[capabilities] manifest not found in scripts/capabilities, fallback to ${candidate.note}`,
      );
      manifestPath = candidate.path;
      break;
    }
  }
}

if (!fs.existsSync(manifestPath)) {
  console.error(`[capabilities] manifest not found: ${manifestPath}`);
  process.exit(1);
}

const manifestDir = path.dirname(manifestPath);
const manifestRaw = fs.readFileSync(manifestPath, "utf8");
const manifest = YAML.parse(manifestRaw) ?? {};
const provides = manifest?.capabilities?.provides ?? [];

if (provides.length === 0) {
  console.log("[capabilities] no capability entries detected, nothing to do.");
  process.exit(0);
}

const capabilitiesDir = path.resolve(
  manifestDir,
  getArgValue("--capabilities-dir") ?? "contracts/capabilities",
);
const schemaDir = path.resolve(
  manifestDir,
  getArgValue("--schemas-dir") ?? "contracts/schema",
);
const inputSchemaDir = path.join(schemaDir, "input");
const outputSchemaDir = path.join(schemaDir, "output");
ensureDir(capabilitiesDir);
ensureDir(inputSchemaDir);
ensureDir(outputSchemaDir);

let manifestUpdated = false;
const createdDescriptors = [];
const createdSchemas = [];

for (const cap of provides) {
  if (!cap?.id) {
    console.warn("[capabilities] skipped entry without id.");
    continue;
  }
  const fileSafe = sanitizeFileName(cap.id);

  const descriptorPath = cap.descriptor
    ? path.resolve(manifestDir, cap.descriptor)
    : path.join(capabilitiesDir, `${fileSafe}.yaml`);
  if (!cap.descriptor) {
    cap.descriptor = toPosix(path.relative(manifestDir, descriptorPath));
    manifestUpdated = true;
  }
  ensureDir(path.dirname(descriptorPath));

  if (!cap.schemas) {
    cap.schemas = {};
    manifestUpdated = true;
  }
  const defaultInputRel = toPosix(
    path.relative(manifestDir, path.join(inputSchemaDir, `${fileSafe}.json`)),
  );
  const defaultOutputRel = toPosix(
    path.relative(manifestDir, path.join(outputSchemaDir, `${fileSafe}.json`)),
  );

  let inputRefs = [];
  if (Array.isArray(cap.schemas.input)) {
    inputRefs = dedupeStrings(cap.schemas.input);
    if (inputRefs.length === 0) {
      cap.schemas.input.push(defaultInputRel);
      inputRefs.push(defaultInputRel);
      manifestUpdated = true;
    }
  } else if (isNonEmptyString(cap.schemas.input)) {
    inputRefs = [cap.schemas.input];
  } else {
    cap.schemas.input = defaultInputRel;
    inputRefs = [defaultInputRel];
    manifestUpdated = true;
  }

  let outputRefs = [];
  if (Array.isArray(cap.schemas.output)) {
    outputRefs = dedupeStrings(cap.schemas.output);
    if (outputRefs.length === 0) {
      cap.schemas.output.push(defaultOutputRel);
      outputRefs.push(defaultOutputRel);
      manifestUpdated = true;
    }
  } else if (isNonEmptyString(cap.schemas.output)) {
    outputRefs = [cap.schemas.output];
  } else {
    cap.schemas.output = defaultOutputRel;
    outputRefs = [defaultOutputRel];
    manifestUpdated = true;
  }

  if (!fs.existsSync(descriptorPath)) {
    const descriptorStub = {
      id: cap.id,
      version: cap.version ?? "1.0.0",
      summary: "TODO: describe capability",
      description: "TODO: add details for reviewers & consumers.",
      input: cap.schemas.input,
      output: cap.schemas.output,
      errors: [],
      rbac: {
        permissions: [cap.rbac ?? `${cap.id}`],
      },
    };
    fs.writeFileSync(descriptorPath, YAML.stringify(descriptorStub), "utf8");
    createdDescriptors.push(descriptorPath);
  }

  for (const schemaRel of inputRefs) {
    const inputSchemaPath = path.resolve(manifestDir, schemaRel);
    if (fs.existsSync(inputSchemaPath)) {
      continue;
    }
    const schemaStub = {
      $schema: "https://json-schema.org/draft/2020-12/schema",
      title: `${cap.id}Input`,
      type: "object",
      properties: {},
      required: [],
      additionalProperties: false,
    };
    ensureDir(path.dirname(inputSchemaPath));
    fs.writeFileSync(
      inputSchemaPath,
      JSON.stringify(schemaStub, null, 2),
      "utf8",
    );
    createdSchemas.push(inputSchemaPath);
  }

  for (const schemaRel of outputRefs) {
    const outputSchemaPath = path.resolve(manifestDir, schemaRel);
    if (fs.existsSync(outputSchemaPath)) {
      continue;
    }
    const schemaStub = {
      $schema: "https://json-schema.org/draft/2020-12/schema",
      title: `${cap.id}Output`,
      type: "object",
      properties: {},
      additionalProperties: true,
    };
    ensureDir(path.dirname(outputSchemaPath));
    fs.writeFileSync(
      outputSchemaPath,
      JSON.stringify(schemaStub, null, 2),
      "utf8",
    );
    createdSchemas.push(outputSchemaPath);
  }
}

if (manifestUpdated) {
  fs.writeFileSync(manifestPath, YAML.stringify(manifest), "utf8");
  console.log(`[capabilities] manifest updated: ${manifestPath}`);
}

if (createdDescriptors.length) {
  console.log("[capabilities] created descriptors:");
  for (const file of createdDescriptors) {
    console.log(`  - ${toPosix(path.relative(cwd, file))}`);
  }
}

if (createdSchemas.length) {
  console.log("[capabilities] created schema files:");
  for (const file of createdSchemas) {
    console.log(`  - ${toPosix(path.relative(cwd, file))}`);
  }
}

console.log("[capabilities] validation complete.");
