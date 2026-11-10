import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const contract = path.join(root, 'docs/contracts');
const pin = JSON.parse(fs.readFileSync(path.join(contract, 'ai-connection-test-source.json'), 'utf8'));
const required = ['ai-settings-connection-test.md', 'ai-connection-test.v1.openapi.yaml', 'fixtures/ai-connection-test.v1.json'];
if (JSON.stringify(Object.keys(pin.sha256).sort()) !== JSON.stringify(required.sort())) throw new Error('AI_CONNECTION_SOURCE_SET_INVALID');
for (const [file, expected] of Object.entries(pin.sha256)) {
  if (crypto.createHash('sha256').update(fs.readFileSync(path.join(contract, file))).digest('hex') !== expected) throw new Error(`AI_CONNECTION_SOURCE_MISMATCH:${file}`);
}
for (const file of [...required, 'ai-connection-test-source.json']) {
  if (!fs.readFileSync(path.join(contract, file)).equals(fs.readFileSync(path.join(root, 'tools/cli/internal/contracts/data/media-contracts', file)))) throw new Error(`AI_CONNECTION_CLI_CONTRACT_MISMATCH:${file}`);
}
if (!fs.readFileSync(path.join(contract, 'fixtures/ai-connection-test.v1.json')).equals(fs.readFileSync(path.join(root, 'framework/backend/go/runtime/aisettings/testdata/ai-connection-test.v1.json')))) throw new Error('AI_CONNECTION_SDK_FIXTURE_MISMATCH');
console.log('AI_CONNECTION_CONTRACT_VALID');
