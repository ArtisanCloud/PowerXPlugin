import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import { fileURLToPath } from 'node:url';

export function validateAIMediaV2Contracts(root) {
  const contract = path.join(root, 'docs/contracts');
  const snapshot = JSON.parse(fs.readFileSync(path.join(contract, 'ai-media-v2-source.json'), 'utf8'));
  const requiredSources = ['ai-media-v2.md', 'ai-media-v2.openapi.yaml', 'fixtures/ai-media-v2.json', 'fixtures/capability-version-upgrade-errors.v1.json'];
  if (JSON.stringify(Object.keys(snapshot.sha256).sort()) !== JSON.stringify([...requiredSources].sort())) throw new Error('AI_MEDIA_CONTRACT_SOURCE_SET_INVALID');
  for (const [file, expected] of Object.entries(snapshot.sha256)) {
    const actual = crypto.createHash('sha256').update(fs.readFileSync(path.join(contract, file))).digest('hex');
    if (actual !== expected) throw new Error(`AI_MEDIA_CONTRACT_SNAPSHOT_MISMATCH:${file}`);
  }
  const fixture = fs.readFileSync(path.join(contract, 'fixtures/ai-media-v2.json'));
  const sdkFixture = fs.readFileSync(path.join(root, 'framework/backend/go/runtime/powerx/ai/testdata/ai-media-v2.json'));
  if (!fixture.equals(sdkFixture)) throw new Error('AI_MEDIA_SDK_FIXTURE_MISMATCH');
  for (const file of [...requiredSources, 'ai-media-v2-source.json', 'fixtures/ai-media-v2-registration.json']) {
    const source = fs.readFileSync(path.join(contract, file));
    const embedded = fs.readFileSync(path.join(root, 'tools/cli/internal/contracts/data/media-contracts', file));
    if (!source.equals(embedded)) throw new Error(`AI_MEDIA_CLI_CONTRACT_MISMATCH:${file}`);
  }
  if (!fs.readFileSync(path.join(contract, 'fixtures/ai-media-v2-registration.json')).equals(fs.readFileSync(path.join(root, 'framework/backend/go/runtime/powerx/ai/testdata/ai-media-v2-registration.json')))) throw new Error('AI_MEDIA_REGISTRATION_FIXTURE_MISMATCH');
  const upgradeFixture = fs.readFileSync(path.join(contract, 'fixtures/capability-version-upgrade-errors.v1.json'));
  if (!upgradeFixture.equals(fs.readFileSync(path.join(root, 'framework/backend/go/runtime/powerx/ai/testdata/capability-version-upgrade-errors.v1.json')))) throw new Error('AI_MEDIA_UPGRADE_ERROR_FIXTURE_MISMATCH');
  const authority = JSON.parse(upgradeFixture);
  const registration = JSON.parse(fs.readFileSync(path.join(contract, 'fixtures/ai-media-v2-registration.json'), 'utf8')).upgrade_confirmation_error;
  const binding = authority.cases.find(item => item.name === 'binding_mismatch');
  if (authority.fixture_owner !== 'Core' || authority.machine_reason_published !== true || authority.cases.length !== 10 || !binding || registration.status !== binding.status || registration.expected_reason_code !== binding.body.reason_code || registration.machine_reason_published !== true || JSON.stringify(registration.body) !== JSON.stringify(binding.body)) throw new Error('AI_MEDIA_UPGRADE_ERROR_CONTRACT_INVALID');
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  validateAIMediaV2Contracts(path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..'));
  console.log('AI_MEDIA_V2_CONTRACT_SNAPSHOT_VALID');
}
