import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const docs = path.join(root, 'docs/contracts');
const source = JSON.parse(fs.readFileSync(path.join(docs, 'event-fabric-service-source.json')));
const hash = raw => crypto.createHash('sha256').update(raw).digest('hex');
assert.equal(source.application_contract, 'event_fabric.service.v1');
assert.equal(source.core_source_uncommitted, true);
assert.deepEqual(Object.keys(source.sha256).sort(), ['event-fabric-service-v1.md', 'event-fabric-service-v1.openapi.yaml', 'event-fabric-service-v1.proto', 'plugin-event-fabric-install.md']);
for (const [file, digest] of Object.entries(source.sha256)) assert.equal(hash(fs.readFileSync(path.join(docs, file))), digest, file);
assert.equal(source.framework_fixture.owner, 'Framework');
const fixture = fs.readFileSync(path.join(docs, source.framework_fixture.file));
assert.equal(hash(fixture), source.framework_fixture.sha256);
assert.deepEqual(fs.readFileSync(path.join(root, 'framework/backend/go/runtime/eventfabric/testdata/event-fabric-service.v1.json')), fixture);
for (const file of [...Object.keys(source.sha256), 'event-fabric-service-source.json', 'event-fabric-framework.md', source.framework_fixture.file]) {
  assert.deepEqual(fs.readFileSync(path.join(root, 'tools/cli/internal/contracts/data/media-contracts', file)), fs.readFileSync(path.join(docs, file)), file);
}
const sample = JSON.parse(fixture);
assert.equal(sample.delivery.event_id, sample.publish_result.event_id);
assert.equal(sample.delivery.version, sample.envelope.meta.payload_version);
assert.notEqual(sample.delivery.trace_id, sample.envelope.meta.trace_id);
assert.equal(sample.delivery.payload.topic, sample.business_topic);
assert.equal(sample.delivery.topic, sample.physical_topic);
assert.equal(sample.delivery.publisher_plugin_id, sample.envelope.meta.source_plugin);
assert.equal(sample.ack_result.delivery_id, sample.delivery.delivery_id);
assert.equal(sample.resolve_topics.publish_request.operation, 'resolve_topics');
assert.equal(sample.resolve_topics.consume_request.subscriber_id, sample.resolve_topics.consume_request.plugin_id);
assert.equal(sample.resolve_topics.publish_result.tenant_uuid, sample.tenant_uuid);
assert.equal(sample.resolve_topics.publish_result.topic_bindings[sample.business_topic], sample.physical_topic);
console.log('EVENT_FABRIC_CONTRACT_OK');

assert.equal(sample.envelope.payload.event_uuid, sample.delivery.event_id);
assert.equal(sample.envelope.payload.dedupe_key, sample.delivery.idempotency_key);
assert.equal(sample.envelope.payload.trace_id, sample.envelope.meta.trace_id);
assert.equal(sample.envelope.payload.tenant_uuid, sample.delivery.tenant_uuid);
