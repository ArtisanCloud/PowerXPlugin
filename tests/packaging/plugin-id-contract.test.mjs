import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const read = (relative) => JSON.parse(fs.readFileSync(path.join(root, relative), 'utf8'));
const cases = read('docs/standards/powerx-plugin/_shared/plugin-id.cases.json');
const source = read('docs/standards/powerx-plugin/_shared/plugin-id.schema.json');
const constraint = Object.fromEntries(Object.entries(source).filter(([key]) => !['$schema', 'title', '$comment'].includes(key)));

test('all Manifest versions share the ID constraint and JavaScript examples', () => {
  // Fail explicitly if the schema evolves beyond these exercised keywords.
  assert.deepEqual(Object.keys(constraint).sort(), ['not', 'pattern', 'type']);
  for (const file of ['framework/backend/go/manifest/schema/manifest.json', 'docs/contracts/manifest.json', 'tools/cli/internal/contracts/data/manifest.json', 'docs/standards/powerx-plugin/lifecycle/contracts/manifest.schema.json', 'docs/standards/powerx-plugin/integration/01_plugin_lifecycle/contracts/manifest.schema.json', 'specs/001-powerxplugin-foundation/contracts/manifest.schema.json']) {
    const id = read(file).properties[file.startsWith('specs/') ? 'pluginId' : 'id'];
    assert.deepEqual(id, constraint, file);
    const accepts = (value) => typeof value === id.type && new RegExp(id.pattern).test(value) && !new RegExp(id.not.pattern).test(value);
    for (const value of cases.valid) assert.equal(accepts(value), true, `REJECTED_VALID_ID: ${file}: ${value}`);
    for (const value of cases.invalid) assert.equal(accepts(value), false, `ACCEPTED_INVALID_ID: ${file}: ${value}`);
  }
});

test('Go, CLI and Python distribute exactly the source contract', () => {
  for (const file of ['framework/backend/go/manifest/schema/plugin-id.json', 'docs/contracts/plugin-id.schema.json', 'tools/cli/internal/contracts/data/plugin-id.schema.json', 'skeleton/backend/python-fastapi/app/manifestx/plugin-id.schema.json']) assert.deepEqual(read(file), source, file);
  const cli = fs.readFileSync(path.join(root,'tools/cli/internal/templates/data/make-files/manifest.mk.tmpl'), 'utf8');
  assert.ok(cli.includes('plugin-id check --manifest'));
  assert.ok(!cli.includes('LEGACY_MATCH'));
});
