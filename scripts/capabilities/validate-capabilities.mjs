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

const cwd = process.cwd();
const manifestPath = path.resolve(
  cwd,
  getArgValue("--manifest") ?? "plugin.yaml",
);

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
  if (!cap.schemas.input) {
    cap.schemas.input = toPosix(
      path.relative(manifestDir, path.join(inputSchemaDir, `${fileSafe}.json`)),
    );
    manifestUpdated = true;
  }
  if (!cap.schemas.output) {
    cap.schemas.output = toPosix(
      path.relative(manifestDir, path.join(outputSchemaDir, `${fileSafe}.json`)),
    );
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

  const inputSchemaPath = path.resolve(manifestDir, cap.schemas.input);
  if (!fs.existsSync(inputSchemaPath)) {
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

  const outputSchemaPath = path.resolve(manifestDir, cap.schemas.output);
  if (!fs.existsSync(outputSchemaPath)) {
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
