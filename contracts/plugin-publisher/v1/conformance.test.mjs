import assert from 'node:assert/strict';
import test from 'node:test';
import { validate, fingerprint, trustedMarketLink, parseStrictJSON, readFixture, policy, recoveryAction, compileDefinitions } from './conformance.mjs';

test('every public definition compiles strictly', compileDefinitions);

for (const row of readFixture('schema-cases.json').cases) {
  test(`schema and semantics: ${row.name}`, () => {
    const call = () => validate(row.definition, row.raw ?? JSON.stringify(row.data));
    if (row.valid) assert.doesNotThrow(call);
    else assert.throws(call);
  });
}

const golden = readFixture('canonical-cases.json').vectors;
for (const row of golden) {
  test(`cross-language canonical request: ${row.name}`, () => {
    assert.deepEqual(fingerprint(row.type, JSON.stringify(row.input)), { canonical: row.canonical, sha256: row.sha256 });
    if (row.type === 'package-upload') {
      const changed = fingerprint(row.type, JSON.stringify({ ...row.input, sha256: 'd'.repeat(64) }));
      assert.notEqual(changed.sha256, row.sha256);
    }
  });
}

test('canonical equivalence and changed intent', () => {
  assert.equal(golden[0].sha256, golden[1].sha256);
  assert.equal(golden[4].sha256, golden[5].sha256);
  assert.notEqual(golden[0].sha256, golden[2].sha256);
  assert.notEqual(golden[4].sha256, golden[6].sha256);
});

for (const row of readFixture('link-cases.json').cases) {
  test(`trusted market links: ${row.name}`, () => assert.equal(trustedMarketLink(row.base, row.target), row.valid));
}

for (const row of readFixture('recovery-cases.json').cases) {
  test(`shared recovery decision: ${row.name}`, () => assert.equal(recoveryAction(JSON.stringify(row.context)), row.action));
}

for (const row of readFixture('encoding-cases.json').cases) {
  test(`raw byte budget and BOM: ${row.name}`, () => {
    const bytes = Buffer.from(row.hex, 'hex');
    const backing = Buffer.concat([Buffer.from([1, 2]), bytes, Buffer.from([3])]);
    const view = new Uint8Array(backing.buffer, backing.byteOffset + 2, bytes.byteLength);
    for (const input of [bytes, view]) {
      const call = () => parseStrictJSON(input, row.max_bytes);
      if (row.valid) assert.doesNotThrow(call); else assert.throws(call);
    }
    if (row.name !== 'invalid UTF-8') {
      const call = () => parseStrictJSON(bytes.toString('utf8'), row.max_bytes);
      if (row.valid) assert.doesNotThrow(call); else assert.throws(call);
    }
  });
}

test('invalid encoding and excessive nesting cannot be silently repaired', () => {
  assert.throws(() => parseStrictJSON(Buffer.from([0x22, 0xff, 0x22])));
  assert.throws(() => parseStrictJSON('['.repeat(66) + '0' + ']'.repeat(66)));
  assert.throws(() => parseStrictJSON(' '.repeat(65537)));
  assert.throws(() => parseStrictJSON('"\\udc00"'));
});

test('synthetic responses are explicit, current capabilities never advertise M2', () => {
  const examples = readFixture('examples.json');
  assert.equal(examples.synthetic, true);
  for (const row of examples.cases) validate(row.definition, JSON.stringify(row.data));
  const discovery = examples.cases.find(row => row.definition === 'DiscoveryResponse').data.data;
  assert.deepEqual(discovery.features, ['package-upload', 'idempotent-writes']);
  assert.deepEqual(discovery.package_formats, ['zip']);
  assert.deepEqual(discovery.supported_scopes, policy.token.m1_scopes);
});

test('plugin ID schema is a pinned existing contract, with no extra namespace/length restriction', () => {
  const pinned = readFixture('plugin-id.schema.json');
  const { $schema, ...definition } = pinned;
  assert.deepEqual(readFixture('protocol.schema.json').$defs.PluginID, definition);
  validate('PluginID', JSON.stringify('org.acme.' + 'a'.repeat(200)));
});

test('request/response byte budgets are separate', () => {
  const response = JSON.stringify(readFixture('examples.json').cases[0].data);
  assert.doesNotThrow(() => validate('DiscoveryResponse', response + ' '.repeat(65536)));
  assert.throws(() => validate('DiscoveryResponse', response + ' '.repeat(policy.limits.max_response_bytes)));
  const request = JSON.stringify({ name: 'Local publisher', plugin_ids: ['com.example.demo'], scopes: ['plugins:read'] });
  assert.throws(() => validate('CreateTokenRequest', request + ' '.repeat(65536)));
});
