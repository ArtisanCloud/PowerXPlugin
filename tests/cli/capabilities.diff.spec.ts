import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs/promises";
import path from "node:path";
import os from "node:os";
import { runCapabilitiesDiffCommand } from "../../tools/cli/src/commands/capabilities/diff";

const capabilityId = "com.powerx.demo.templates.create";
const descriptorPath = "contracts/capabilities/com.powerx.demo.templates.create.yaml";
const inputSchemaPath =
  "contracts/schema/input/com.powerx.demo.templates.create.json";
const outputSchemaPath =
  "contracts/schema/output/com.powerx.demo.templates.create.json";

test("capabilities diff reports schema and exposure changes", async () => {
  const tmpRoot = await fs.mkdtemp(path.join(os.tmpdir(), "cap-diff-"));
  const fromDir = path.join(tmpRoot, "from");
  const toDir = path.join(tmpRoot, "to");
  await fs.mkdir(fromDir, { recursive: true });
  await fs.mkdir(toDir, { recursive: true });

  await writeCapabilitySnapshot(fromDir, {
    version: "1.0.0",
    fields: ["name"],
    entrypoint: "/api/v1/templates",
  });
  await writeCapabilitySnapshot(toDir, {
    version: "1.1.0",
    fields: ["name", "description"],
    entrypoint: "/api/v2/templates",
  });

  const outputPath = path.join(tmpRoot, "report.md");
  const report = await runCapabilitiesDiffCommand({
    from: path.join(fromDir, "plugin.yaml"),
    to: path.join(toDir, "plugin.yaml"),
    rootDir: tmpRoot,
    outputPath,
  });

  assert.equal(report.changes.length, 1);
  const change = report.changes[0];
  assert.equal(change.id, capabilityId);
  assert.equal(change.type, "modified");
  assert.deepEqual(change.version, { from: "1.0.0", to: "1.1.0" });
  assert.ok(change.inputSchema);
  assert.deepEqual(change.inputSchema?.fieldsAdded, ["description"]);
  assert.deepEqual(change.inputSchema?.fieldsRemoved, []);
  assert.ok(change.channels);
  assert.equal(change.channels?.changed.length, 1);
  const written = await fs.readFile(outputPath, "utf8");
  assert.match(written, /Capability Change Report/);
});

async function writeCapabilitySnapshot(
  baseDir: string,
  options: { version: string; fields: string[]; entrypoint: string },
) {
  const manifest = [
    "id: com.powerx.demo",
    "capabilities:",
    "  provides:",
    `    - id: ${capabilityId}`,
    `      version: ${options.version}`,
    `      descriptor: ${descriptorPath}`,
    "      schemas:",
    `        input: ${inputSchemaPath}`,
    `        output: ${outputSchemaPath}`,
    "agent_tools:",
    `  - id: ${capabilityId}`,
    `    capability: ${capabilityId}`,
    "    handler: backend/internal/handlers/capabilities/template/create_handler.go",
    "exposure:",
    "  channels:",
    "    - type: rest",
    `      capability: ${capabilityId}`,
    `      entrypoint: ${options.entrypoint}`,
    "      method: POST",
    "      auth: jwt",
  ].join("\n");
  const descriptor = [
    `id: ${capabilityId}`,
    `version: ${options.version}`,
    "summary: Capability diff test",
    "rbac:",
    "  permissions:",
    `    - ${capabilityId}`,
  ].join("\n");
  const schema = {
    $schema: "https://json-schema.org/draft/2020-12/schema",
    type: "object",
    properties: Object.fromEntries(
      options.fields.map((field) => [field, { type: "string" }]),
    ),
    required: options.fields.slice(0, 1),
  };

  await fs.writeFile(path.join(baseDir, "plugin.yaml"), manifest);
  const descriptorFile = path.join(baseDir, descriptorPath);
  await fs.mkdir(path.dirname(descriptorFile), { recursive: true });
  await fs.writeFile(descriptorFile, descriptor);
  const inputFile = path.join(baseDir, inputSchemaPath);
  await fs.mkdir(path.dirname(inputFile), { recursive: true });
  await fs.writeFile(inputFile, JSON.stringify(schema, null, 2));
  const outputFile = path.join(baseDir, outputSchemaPath);
  await fs.mkdir(path.dirname(outputFile), { recursive: true });
  await fs.writeFile(outputFile, JSON.stringify(schema, null, 2));
}
