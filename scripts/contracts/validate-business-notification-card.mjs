import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const contracts = path.join(root, 'docs/contracts');
const snapshot = JSON.parse(fs.readFileSync(path.join(contracts, 'business-notification-card-source.json'), 'utf8'));
for (const [file, expected] of Object.entries(snapshot.sha256)) {
  const actual = crypto.createHash('sha256').update(fs.readFileSync(path.join(contracts, file))).digest('hex');
  if (actual !== expected) throw new Error(`NOTIFICATION_CARD_SNAPSHOT_MISMATCH:${file}`);
}
for (const version of ['v1', 'v2']) {
  const fixture = fs.readFileSync(path.join(contracts, `fixtures/business-notification-card.${version}.json`));
  if (!fixture.equals(fs.readFileSync(path.join(root, `framework/backend/go/runtime/powerx/notifications/testdata/business-notification-card.${version}.json`)))) {
    throw new Error(`NOTIFICATION_CARD_SDK_FIXTURE_MISMATCH:${version}`);
  }
}
const schemas = ['framework/backend/go/manifest/schema/manifest.json', 'docs/contracts/manifest.json', 'tools/cli/internal/contracts/data/manifest.json', 'docs/standards/powerx-plugin/lifecycle/contracts/manifest.schema.json', 'docs/standards/powerx-plugin/integration/01_plugin_lifecycle/contracts/manifest.schema.json'];
let expected;
for (const file of schemas) {
  const schema = JSON.parse(fs.readFileSync(path.join(root, file), 'utf8'));
  const field = schema.properties.notification_presentations;
  if (!field || (expected && JSON.stringify(field) !== expected)) throw new Error(`NOTIFICATION_PRESENTATION_SCHEMA_MISMATCH:${file}`);
  expected = JSON.stringify(field);
  if (JSON.stringify(field.items.properties.contract.enum) !== JSON.stringify(['business_notification_card.v1','business_notification_card.v2'])) throw new Error(`NOTIFICATION_CARD_VERSION_SCHEMA_INVALID:${file}`);
}
console.log('BUSINESS_NOTIFICATION_CARD_CONTRACT_VALID');
