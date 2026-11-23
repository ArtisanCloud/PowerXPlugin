import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import assert from 'node:assert/strict';
import YAML from 'yaml';
import { createConfig, lint } from '@redocly/openapi-core';
import { readFixture, validate } from './conformance.mjs';

const bundle = path.dirname(fileURLToPath(import.meta.url));
const source = path.resolve(bundle, '../../../backend/api/openapi/plugin-publisher.openapi.yaml');
const portable = path.join(bundle, 'openapi.yaml');
const normalize = text => text.replaceAll('../../../contracts/plugin-publisher/v1/protocol.schema.json', './protocol.schema.json');
const hasAuthority = fs.existsSync(source);
assert.ok(hasAuthority || !process.argv.includes('--require-authority'), 'market authority OpenAPI is required');
if (hasAuthority) assert.equal(normalize(fs.readFileSync(source, 'utf8')), fs.readFileSync(portable, 'utf8'), 'portable OpenAPI differs from the backend authority');
const config = await createConfig({ rules: {
  struct: 'error', 'no-unresolved-refs': 'error', 'operation-operationId-unique': 'error',
  'operation-parameters-unique': 'error', 'path-parameters-defined': 'error', 'security-defined': 'error',
} });
for (const file of hasAuthority ? [source, portable] : [portable]) {
  const problems = await lint({ ref: file, config });
  const errors = problems.filter(problem => problem.severity === 'error');
  assert.equal(errors.length, 0, errors.map(error => `${error.ruleId}: ${error.message}`).join('\n'));
}
const api = YAML.parse(fs.readFileSync(portable, 'utf8'));
const definitions = readFixture('protocol.schema.json').$defs;
const constants = readFixture('constants.json');
for (const key of ['limits', 'timeouts', 'recovery', 'pagination']) {
  assert.deepEqual(Object.fromEntries(Object.entries(definitions.Discovery.properties[key].properties).map(([name, rule]) => [name, rule.const])), constants[key], `discovery ${key} differs from constants`);
}
assert.deepEqual(definitions.Error.properties.error.enum.sort(), Object.keys(readFixture('errors.json')).sort());
const seen = new Set();
function refs(node) {
  if (!node || typeof node !== 'object') return;
  if (typeof node.$ref === 'string' && node.$ref.includes('protocol.schema.json#/$defs/')) {
    const name = node.$ref.split('#/$defs/')[1];
    assert.ok(definitions[name], `missing definition ${name}`); seen.add(name);
  }
  Object.values(node).forEach(refs);
}
refs(api);
for (const row of readFixture('examples.json').cases) validate(row.definition, JSON.stringify(row.data));
assert.equal(Object.keys(api.paths).length, 12);
assert.equal(api['x-runtime-status'], 'partial_foundation');
for (const [route, item] of Object.entries(api.paths)) {
  for (const method of ['get', 'post']) {
    if (!item[method]) continue;
    const op = item[method];
    if (route.startsWith('/publisher/') && route !== '/publisher/discovery') assert.deepEqual(op.security, [{ publisherToken: [] }]);
    if (route.startsWith('/dev/')) {
      assert.deepEqual(op.security, [{ marketSession: [] }]);
      if (method === 'post') assert.ok(op.parameters.some(p => p.name === 'X-Market-CSRF' && p.required));
    }
  }
}
console.log(`OpenAPI 3.1 passed: 12 paths, ${seen.size} referenced wire definitions; ${hasAuthority ? 'market authority checked' : 'portable-only mode'}; synthetic examples agree`);
