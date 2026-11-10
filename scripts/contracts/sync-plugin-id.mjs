#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { parseDocument } from 'yaml';
import { isDeepStrictEqual } from 'node:util';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const source = 'docs/standards/powerx-plugin/_shared/plugin-id.schema.json';
const cases = 'docs/standards/powerx-plugin/_shared/plugin-id.cases.json';
const schema = JSON.parse(fs.readFileSync(path.join(root, source), 'utf8'));
const constraint = Object.fromEntries(Object.entries(schema).filter(([key]) => !['$schema', 'title', '$comment'].includes(key)));
const check = process.argv.includes('--check');
let drift = false;
function emit(relative, content) {
  const target = path.join(root, relative);
  if (fs.existsSync(target) && fs.readFileSync(target, 'utf8') === content) return;
  if (check) { console.error(`PLUGIN_ID_CONTRACT_DRIFT: ${relative}`); drift = true; return; }
  fs.mkdirSync(path.dirname(target), { recursive: true });
  fs.writeFileSync(target, content);
}
for (const relative of ['framework/backend/go/manifest/schema/manifest.json', 'docs/contracts/manifest.json', 'tools/cli/internal/contracts/data/manifest.json', 'specs/001-powerxplugin-foundation/contracts/manifest.schema.json']) {
  const content = fs.readFileSync(path.join(root, relative), 'utf8');
  const field = relative.startsWith('specs/') ? 'pluginId' : 'id';
  const node = parseDocument(content).getIn(['properties', field], true);
  if (!node?.range) throw new Error(`PLUGIN_ID_CONSTRAINT_MISSING: ${relative}`);
  const indent = content.slice(0, node.range[0]).split('\n').at(-1).match(/^ */)[0];
  const replacement = JSON.stringify(constraint, null, 2).replace(/\n/g, `\n${indent}`);
  emit(relative, content.slice(0, node.range[0]) + replacement + content.slice(node.range[1]));
}
const raw = fs.readFileSync(path.join(root, source), 'utf8');
for (const prefix of ['lifecycle', 'integration/01_plugin_lifecycle']) {
  const file = `docs/standards/powerx-plugin/${prefix}/contracts/manifest.schema.json`;
  if (!isDeepStrictEqual(JSON.parse(fs.readFileSync(path.join(root, file), 'utf8')).properties.id, constraint)) {
    console.error(`PLUGIN_ID_STANDARDS_DISTRIBUTION_REQUIRED: ${file}`);
    drift = true;
  }
}
for (const target of ['framework/backend/go/manifest/schema/plugin-id.json', 'docs/contracts/plugin-id.schema.json', 'tools/cli/internal/contracts/data/plugin-id.schema.json', 'skeleton/backend/python-fastapi/app/manifestx/plugin-id.schema.json']) emit(target, raw);
emit('framework/backend/go/manifest/schema/plugin-id.cases.json', fs.readFileSync(path.join(root, cases), 'utf8'));
emit('skeleton/docs/conventions/plugin-id.md', fs.readFileSync(path.join(root, 'docs/standards/powerx-plugin/_shared/plugin-id.md'), 'utf8'));
if (drift) process.exitCode = 1;
