import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const read = name => fs.readFileSync(path.join(root, name));
const hash = raw => crypto.createHash('sha256').update(raw).digest('hex');
const source = JSON.parse(read('docs/contracts/plugin-session-termination-source.json'));
for (const [file, sha] of Object.entries(source.sha256)) assert.equal(hash(read('docs/contracts/'+file)), sha, file);
assert.equal(hash(read(source.core_consumer.fixture)), source.core_consumer.sha256);
const provenance = JSON.parse(read('skeleton/web-admin/nuxt/tests/fixtures/core-bridge-v2.provenance.json'));
assert.equal(provenance.sha256, source.core_consumer.sha256);
for (const name of ['plugin-session-termination-v2.md','plugin-session-termination-source.json','plugin-session-termination-framework.md']) {
  assert.deepEqual(read('docs/contracts/'+name), read('tools/cli/internal/contracts/data/media-contracts/'+name));
  assert(read('tools/cli/internal/contracts/embed.go').includes('name: "docs/contracts/'+name+'"'));
}
console.log('SESSION_TERMINATION_CONTRACT_VALID');
