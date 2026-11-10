import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import crypto from 'node:crypto';
import { fileURLToPath } from 'node:url';
import { validateAIMediaV2Contracts } from './validate-ai-media-v2.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const authority = 'fixtures/capability-version-upgrade-errors.v1.json';
function copyContracts(t) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'ai-media-contract-'));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
  const files = ['ai-media-v2.md', 'ai-media-v2.openapi.yaml', 'ai-media-v2-source.json', 'fixtures/ai-media-v2.json', 'fixtures/ai-media-v2-registration.json', 'fixtures/ai-media-model-limits.v1.json', authority];
  for (const base of ['docs/contracts', 'tools/cli/internal/contracts/data/media-contracts']) {
    for (const file of files) {
      const dest = path.join(dir, base, file);
      fs.mkdirSync(path.dirname(dest), { recursive: true });
      fs.copyFileSync(path.join(root, base, file), dest);
    }
  }
  for (const file of ['ai-media-v2.json', 'ai-media-v2-registration.json', 'ai-media-model-limits.v1.json', 'capability-version-upgrade-errors.v1.json']) {
    const base = 'framework/backend/go/runtime/powerx/ai/testdata';
    fs.mkdirSync(path.join(dir, base), { recursive: true });
    fs.copyFileSync(path.join(root, base, file), path.join(dir, base, file));
  }
  return dir;
}

test('accepts all pinned files and synchronized consumers', t => {
  assert.doesNotThrow(() => validateAIMediaV2Contracts(copyContracts(t)));
});
test('rejects a missing fourth source digest even when the fixture exists', t => {
  const dir = copyContracts(t), pin = path.join(dir, 'docs/contracts/ai-media-v2-source.json');
  const source = JSON.parse(fs.readFileSync(pin));
  delete source.sha256[authority];
  fs.writeFileSync(pin, JSON.stringify(source));
  assert.throws(() => validateAIMediaV2Contracts(dir), /AI_MEDIA_CONTRACT_SOURCE_SET_INVALID/);
});
test('rejects a missing authoritative document', t => {
  const dir = copyContracts(t);
  fs.unlinkSync(path.join(dir, 'docs/contracts', authority));
  assert.throws(() => validateAIMediaV2Contracts(dir), { code: 'ENOENT' });
});
test('rejects SDK testdata drift in the new fixture', t => {
  const dir = copyContracts(t);
  fs.appendFileSync(path.join(dir, 'framework/backend/go/runtime/powerx/ai/testdata/capability-version-upgrade-errors.v1.json'), '\n');
  assert.throws(() => validateAIMediaV2Contracts(dir), /AI_MEDIA_UPGRADE_ERROR_FIXTURE_MISMATCH/);
});
test('rejects CLI embedded fixture drift', t => {
  const dir = copyContracts(t);
  fs.appendFileSync(path.join(dir, 'tools/cli/internal/contracts/data/media-contracts', authority), '\n');
  assert.throws(() => validateAIMediaV2Contracts(dir), /AI_MEDIA_CLI_CONTRACT_MISMATCH/);
});
test('rejects synchronized registration copies with obsolete machine-code expectations', t => {
  const dir = copyContracts(t);
  const registration = JSON.parse(fs.readFileSync(path.join(dir, 'docs/contracts/fixtures/ai-media-v2-registration.json')));
  registration.upgrade_confirmation_error.expected_reason_code = 'AI_UPSTREAM_DEPENDENCY';
  registration.upgrade_confirmation_error.machine_reason_published = false;
  for (const dest of ['docs/contracts/fixtures', 'framework/backend/go/runtime/powerx/ai/testdata', 'tools/cli/internal/contracts/data/media-contracts/fixtures']) {
    fs.writeFileSync(path.join(dir, dest, 'ai-media-v2-registration.json'), JSON.stringify(registration));
  }
  assert.throws(() => validateAIMediaV2Contracts(dir), /AI_MEDIA_UPGRADE_ERROR_CONTRACT_INVALID/);
});

test('rejects SDK model-limit fixture drift', t => {
  const dir = copyContracts(t);
  fs.appendFileSync(path.join(dir, 'framework/backend/go/runtime/powerx/ai/testdata/ai-media-model-limits.v1.json'), '\n');
  assert.throws(() => validateAIMediaV2Contracts(dir), /AI_MEDIA_MODEL_LIMITS_SDK_MISMATCH/);
});

test('rejects synchronized fixtures that erase an explicit zero', t => {
  const dir = copyContracts(t), file = 'fixtures/ai-media-model-limits.v1.json';
  const sample = JSON.parse(fs.readFileSync(path.join(dir, 'docs/contracts', file)));
  delete sample.cases.find(item => item.name === 'explicit_zero').limits.max_image_bytes;
  const raw = Buffer.from(JSON.stringify(sample));
  const pin = JSON.parse(fs.readFileSync(path.join(dir, 'docs/contracts/ai-media-v2-source.json')));
  pin.framework_model_limits_fixture.sha256 = crypto.createHash('sha256').update(raw).digest('hex');
  for (const base of ['docs/contracts', 'tools/cli/internal/contracts/data/media-contracts']) {
    fs.writeFileSync(path.join(dir, base, file), raw);
    fs.writeFileSync(path.join(dir, base, 'ai-media-v2-source.json'), JSON.stringify(pin));
  }
  fs.writeFileSync(path.join(dir, 'framework/backend/go/runtime/powerx/ai/testdata/ai-media-model-limits.v1.json'), raw);
  assert.throws(() => validateAIMediaV2Contracts(dir), /AI_MEDIA_MODEL_LIMITS_FIXTURE_INVALID/);
});
