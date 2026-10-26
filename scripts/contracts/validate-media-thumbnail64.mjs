import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const contracts = path.join(root, 'docs/contracts');
const snapshot = JSON.parse(fs.readFileSync(path.join(contracts, 'media-thumbnail64-source.json'), 'utf8'));
for (const [name, expected] of Object.entries(snapshot.sha256)) {
  const actual = crypto.createHash('sha256').update(fs.readFileSync(path.join(contracts, name))).digest('hex');
  if (actual !== expected) throw new Error(`MEDIA_THUMBNAIL64_SNAPSHOT_MISMATCH:${name}`);
}
const source = fs.readFileSync(path.join(contracts, 'fixtures/media-image-derivatives.v2.json'));
const sdk = fs.readFileSync(path.join(root, 'framework/backend/go/runtime/powerx/media/testdata/media-image-derivatives.v2.json'));
if (!source.equals(sdk)) throw new Error('MEDIA_THUMBNAIL64_SDK_FIXTURE_MISMATCH');
for (const [source, target] of Object.entries({
  'business-notification-card.openapi.yaml': 'business-notification-card.openapi.yaml',
  'image-derivatives.openapi.yaml': 'image-derivatives.openapi.yaml',
  'media-host.openapi.yaml': 'media-host.openapi.yaml',
  'fixtures/business-notification-card.v1.json': 'fixtures/business-notification-card.v1.json',
  'fixtures/business-notification-card.v2.json': 'fixtures/business-notification-card.v2.json',
  'fixtures/media-image-derivatives.v2.json': 'fixtures/media-image-derivatives.v2.json',
})) {
  if (!fs.readFileSync(path.join(contracts, source)).equals(fs.readFileSync(path.join(root, 'tools/cli/internal/contracts/data/media-contracts', target)))) throw new Error(`MEDIA_THUMBNAIL64_GENERATOR_SNAPSHOT_MISMATCH:${target}`);
}
console.log('MEDIA_THUMBNAIL64_CONTRACT_VALID');
