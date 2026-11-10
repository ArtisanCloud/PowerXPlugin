import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import { fileURLToPath } from 'node:url';
import YAML from 'yaml';

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
  const limits = snapshot.framework_model_limits_fixture;
  if (limits?.owner !== 'Framework' || limits.file !== 'fixtures/ai-media-model-limits.v1.json') throw new Error('AI_MEDIA_MODEL_LIMITS_SOURCE_INVALID');
  const limitsRaw = fs.readFileSync(path.join(contract, limits.file));
  if (crypto.createHash('sha256').update(limitsRaw).digest('hex') !== limits.sha256) throw new Error('AI_MEDIA_MODEL_LIMITS_DIGEST_INVALID');
  if (!limitsRaw.equals(fs.readFileSync(path.join(root, 'framework/backend/go/runtime/powerx/ai/testdata/ai-media-model-limits.v1.json')))) throw new Error('AI_MEDIA_MODEL_LIMITS_SDK_MISMATCH');
  const fields = ['max_image_bytes', 'max_total_image_bytes', 'max_image_pixels'];
  const operation = YAML.parse(fs.readFileSync(path.join(contract, 'ai-media-v2.openapi.yaml'), 'utf8')).components.schemas.ModelOperation;
  if (fields.some(field => operation.properties[field]?.type !== 'integer' || operation.properties[field]?.format !== 'int64' || operation.required.includes(field))) throw new Error('AI_MEDIA_MODEL_LIMITS_SCHEMA_INVALID');
  const samples = JSON.parse(limitsRaw);
  const cases = Object.fromEntries(samples.cases.map(sample => [sample.name, sample.limits]));
  if (samples.fixture_owner !== 'Framework' || samples.contract_version !== 2 || !cases.declared || !cases.undeclared || !cases.explicit_zero || !cases.provider_specific || fields.some(field => !(cases.declared[field] > 0) || Object.hasOwn(cases.undeclared, field) || cases.explicit_zero[field] !== 0 || !(cases.provider_specific[field] > 0) || cases.provider_specific[field] === cases.declared[field])) throw new Error('AI_MEDIA_MODEL_LIMITS_FIXTURE_INVALID');
  if (fields.some(field => samples.catalog_result.items[0].operations.image_generation[field] !== cases.declared[field])) throw new Error('AI_MEDIA_MODEL_LIMITS_FIXTURE_INVALID');
  for (const file of [...requiredSources, 'ai-media-v2-source.json', 'fixtures/ai-media-v2-registration.json', limits.file]) {
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
