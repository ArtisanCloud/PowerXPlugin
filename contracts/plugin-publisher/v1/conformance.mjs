import fs from 'node:fs';
import { createHash } from 'node:crypto';
import path from 'node:path/posix';
import { isIP } from 'node:net';
import Ajv2020 from 'ajv/dist/2020.js';
import addFormats from 'ajv-formats';

export const readFixture = name => JSON.parse(fs.readFileSync(new URL(name, import.meta.url), 'utf8'));
export const policy = readFixture('constants.json');
const errors = readFixture('errors.json');
const schema = readFixture('protocol.schema.json');
const ajv = new Ajv2020({ allErrors: true, strict: true });
addFormats(ajv);
ajv.addFormat('binary', true);
ajv.addSchema(schema);

export function compileDefinitions() {
  for (const name of Object.keys(schema.$defs)) {
    if (!ajv.getSchema(`${schema.$id}#/$defs/${name}`)) throw new Error(`unresolved schema ${name}`);
  }
}

// Parse before schema validation so duplicate keys, invalid Unicode and unsafe
// integers cannot be silently normalized by JSON.parse. No caller data is logged.
export function parseStrictJSON(input, maxBytes = policy.limits.max_json_bytes) {
  let raw;
  if (typeof input === 'string') {
    if (Buffer.byteLength(input) > maxBytes || !input.isWellFormed()) throw new Error('JSON byte or Unicode limit');
    raw = input;
  } else {
    if (!(input instanceof Uint8Array) || input.byteLength > maxBytes) throw new Error('JSON byte or input type limit');
    // Measure the actual view before decoding. The default TextDecoder consumes
    // a leading BOM; ignoreBOM:true preserves it so our explicit policy rejects it.
    raw = new TextDecoder('utf-8', { fatal: true, ignoreBOM: true }).decode(input);
  }
  if (raw.startsWith('\ufeff')) throw new Error('UTF-8 BOM is not allowed');
  let i = 0;
  const skip = () => { while (/[\t\r\n ]/.test(raw[i] ?? '\0')) i++; };
  const string = () => {
    const start = i++;
    let closed = false;
    while (i < raw.length) {
      if (raw[i] === '\\') { i += 2; continue; }
      if (raw[i++] === '"') { closed = true; break; }
    }
    if (!closed) throw new Error('invalid JSON string');
    const result = JSON.parse(raw.slice(start, i));
    if (!result.isWellFormed() || result.includes('\0')) throw new Error('invalid Unicode or NUL');
    return result;
  };
  const value = depth => {
    if (depth > 64) throw new Error('JSON nesting limit');
    skip();
    if (raw[i] === '"') return string();
    if (raw[i] === '{') {
      i++; skip();
      const out = Object.create(null);
      const keys = new Set();
      if (raw[i] === '}') { i++; return out; }
      for (;;) {
        skip();
        if (raw[i] !== '"') throw new Error('invalid object member');
        const key = string();
        if (keys.has(key)) throw new Error('duplicate JSON member');
        keys.add(key); skip();
        if (raw[i++] !== ':') throw new Error('missing colon');
        out[key] = value(depth + 1); skip();
        const next = raw[i++];
        if (next === '}') return out;
        if (next !== ',') throw new Error('invalid object separator');
      }
    }
    if (raw[i] === '[') {
      i++; skip(); const out = [];
      if (raw[i] === ']') { i++; return out; }
      for (;;) {
        out.push(value(depth + 1)); skip();
        const next = raw[i++];
        if (next === ']') return out;
        if (next !== ',') throw new Error('invalid array separator');
      }
    }
    for (const [word, result] of [['true', true], ['false', false], ['null', null]]) {
      if (raw.startsWith(word, i)) { i += word.length; return result; }
    }
    const number = /^-?(?:0|[1-9][0-9]*)/.exec(raw.slice(i));
    if (!number) throw new Error('invalid JSON value');
    i += number[0].length;
    if (/[.eE0-9]/.test(raw[i] ?? '\0')) throw new Error('only decimal JSON integers are supported');
    const result = Number(number[0]);
    if (!Number.isSafeInteger(result)) throw new Error('unsafe JSON integer');
    return result;
  };
  const out = value(0); skip();
  if (i !== raw.length) throw new Error('trailing JSON');
  return out;
}

const trimASCII = value => value.replace(/^[ \t\r\n]+|[ \t\r\n]+$/g, '');
function webURL(raw) {
  if (/[\\\x00\r\n\t ]/.test(raw) || !/^https?:\/\//.test(raw)) throw new Error('invalid web URL');
  const authority = /^https?:\/\/([^/?#]+)/.exec(raw)?.[1];
  if (!authority || /[^\x00-\x7f]/.test(authority)) throw new Error('web URL hosts must use ASCII/IDNA form');
  const url = new URL(raw);
  if (url.username || url.password || !url.hostname || url.port === '0') throw new Error('invalid web URL');
  const rawHost = authority.startsWith('[') ? authority.slice(1, authority.indexOf(']')) : authority.split(':')[0];
  if (rawHost.toLowerCase() !== url.hostname.replace(/^\[|\]$/g, '')) throw new Error('URL host must be canonical ASCII/IDNA');
  return url;
}

function semantic(value) {
  if (Array.isArray(value)) { value.forEach(semantic); return; }
  if (!value || typeof value !== 'object') return;
  const byteLimits = {
    release_notes: 10000, description: 10000, security_statement: 10000, privacy_statement: 10000,
    support_url: 2000, icon_url: 2000, source: 2000, console_base_url: 2000,
    market_version_url: 2000, market_publication_url: 2000, min_core: 20,
    name: 200, summary: 2000, reason: 1000, message: 1000, review_reason: 4000, publish_reason: 4000,
  };
  for (const [key, child] of Object.entries(value)) {
    if (typeof child === 'string') {
      if (byteLimits[key] && Buffer.byteLength(child) > byteLimits[key]) throw new Error(`${key} exceeds UTF-8 byte budget`);
      if (['description', 'security_statement', 'privacy_statement'].includes(key) && Array.from(trimASCII(child)).length < 10) throw new Error('metadata text is too short after normalization');
      if (['support_url', 'icon_url', 'source', 'console_base_url', 'market_version_url', 'market_publication_url'].includes(key) && child) webURL(child);
    }
    semantic(child);
  }
  for (const key of ['artifacts', 'expected_artifacts']) {
    if (!Array.isArray(value[key])) continue;
    const ids = new Set(); const targets = new Set();
    for (const artifact of value[key]) {
      const target = `${artifact.os}/${artifact.arch}`;
      if (ids.has(artifact.artifact_id) || targets.has(target)) throw new Error('duplicate artifact UUID or target');
      ids.add(artifact.artifact_id); targets.add(target);
    }
  }
  if (value.scopes) {
    const scopes = new Set(value.scopes);
    if (!scopes.has('plugins:read') || scopes.has('publications:submit') && !scopes.has('publications:read')) throw new Error('missing scope dependency');
    if (typeof value.name === 'string' && Array.from(trimASCII(value.name)).length < 2) throw new Error('credential name is too short after trimming');
  }
  if (typeof value.reason === 'string' && !Object.hasOwn(value, 'field') && Array.from(trimASCII(value.reason)).length < 2) throw new Error('reason is too short after trimming');
  if (typeof value.error === 'string' && (errors[value.error].http_status !== value.code || errors[value.error].retryable !== value.retryable)) throw new Error('error status or retry policy mismatch');
  if (value.items && (value.items.length > value.size || value.items.length > value.total)) throw new Error('inconsistent page counts');
  if (value.token_id) {
    if (value.created_at) {
      const duration = Date.parse(value.expires_at) - Date.parse(value.created_at);
      if (duration <= 0 || duration > 30 * 86400000) throw new Error('invalid credential lifetime');
    }
    if (value.status && (value.status === 'revoked') !== (value.revoked_at !== null)) throw new Error('inconsistent credential revocation');
  }
  if (value.request_sha256) {
    if (Date.parse(value.updated_at) < Date.parse(value.created_at)) throw new Error('operation time moved backwards');
    if (value.status !== 'processing' && Date.parse(value.retain_until) < Date.parse(value.updated_at) + 7 * 86400000) throw new Error('operation retention is too short');
    if (value.result) {
      const upload = Boolean(value.result.artifact);
      if (upload !== (value.type === 'package-upload') || value.result.operation_id !== value.operation_id || upload && value.result.plugin_id !== value.plugin_id) throw new Error('operation receipt mismatch');
    }
  }
  if (value.features) {
    const features = new Set(value.features);
    if (features.has('package-upload') && !features.has('idempotent-writes') || features.has('publication-submit') && !features.has('package-upload')) throw new Error('missing feature dependency');
  }
}

export function validate(definition, raw) {
  const responseDefinitions = new Set(['Discovery', 'Me', 'Plugin', 'MarketVersion', 'Artifact', 'Artifacts', 'Publication', 'PublicationOptions', 'PublishToken', 'CreatedToken', 'UploadReceipt', 'SubmissionReceipt', 'Operation', 'OperationProcessing', 'OperationSucceeded', 'OperationFailed', 'VersionsPage', 'TokensPage', 'PublicationsPage', 'Error']);
  const limit = definition.endsWith('Response') || responseDefinitions.has(definition) ? policy.limits.max_response_bytes : policy.limits.max_json_bytes;
  const value = parseStrictJSON(raw, limit);
  const check = ajv.getSchema(`${schema.$id}#/$defs/${definition}`);
  if (!check || !check(value)) throw new Error(`${definition} schema rejected value`);
  semantic(value);
  return value;
}

// All allowed object property names are fixed ASCII. Sorting them and using
// JSON.stringify for Unicode scalar strings/safe integers implements JCS here.
const canonical = value => {
  if (Array.isArray(value)) return `[${value.map(canonical).join(',')}]`;
  if (value && typeof value === 'object') return `{${Object.keys(value).sort().map(key => `${JSON.stringify(key)}:${canonical(value[key])}`).join(',')}}`;
  return JSON.stringify(value);
};

export function fingerprint(kind, raw) {
  const value = parseStrictJSON(raw);
  let definition = 'UploadFingerprintInput';
  if (kind === 'publication-submit') {
    definition = 'SubmitFingerprintInput';
    if (value.metadata && typeof value.metadata === 'object') {
      for (const [key, text] of Object.entries(value.metadata)) if (typeof text === 'string') value.metadata[key] = trimASCII(text);
      if (!Object.hasOwn(value.metadata, 'icon_url')) value.metadata.icon_url = '';
    }
  } else if (kind !== 'package-upload') throw new Error('unknown operation type');
  validate(definition, JSON.stringify(value));
  if (kind === 'publication-submit') {
    value.expected_artifacts.sort((a, b) => {
      for (const key of ['os', 'arch', 'artifact_id']) if (a[key] !== b[key]) return a[key] < b[key] ? -1 : 1;
      return 0;
    });
  }
  value.protocol = policy.protocol;
  value.type = kind;
  const payload = canonical(value);
  return { canonical: payload, sha256: createHash('sha256').update(policy.fingerprint_domain + payload).digest('hex') };
}

export function trustedMarketLink(base, target) {
  try {
    const b = webURL(base); const t = webURL(target);
    if (b.search || b.hash || t.hash || b.origin !== t.origin) return false;
    const host = b.hostname.replace(/^\[|\]$/g, '');
    if (b.protocol === 'http:' && host !== 'localhost' && !(isIP(host) === 4 && host.startsWith('127.')) && host !== '::1') return false;
    for (const raw of [base, target]) {
      const p = /^https?:\/\/[^/?#]+([^?#]*)/.exec(raw)[1] || '/';
      if (p.includes('%') || /[^\x00-\x7f]/.test(p) || path.normalize(p) !== p || p.split('/').some(segment => segment === '.' || segment === '..')) return false;
    }
    const root = b.pathname.replace(/\/$/, '');
    return t.pathname === root || t.pathname.startsWith(`${root}/`);
  } catch { return false; }
}

// Reference policy only: this function never sends a request or authorizes a
// credential. Runtime observations must come from the actual client/server.
export function recoveryAction(raw) {
  const v = validate('RecoveryContext', raw);
  if (v.credential_status !== 'active') return 'reauthorize';
  if (!v.identity_matches || !v.plugin_allowed || !v.scope_allowed) return 'deny';
  if (!v.request_matches) return 'conflict';
  if (v.operation_status === 'succeeded') return 'read_current_state';
  if (v.operation_status === 'failed_permanent') return 'stop_permanent';
  const remaining = policy.recovery.max_recovery_elapsed_seconds - v.elapsed_seconds;
  const wait = v.operation_status === 'processing' ? Math.max(v.retry_after_seconds, policy.recovery.operation_poll_seconds) : v.retry_after_seconds;
  if (remaining <= 0 || v.retry_after_seconds > policy.recovery.max_retry_after_seconds || wait >= remaining) return 'manual_review';
  if (v.operation_status === 'processing') return wait >= policy.recovery.max_operation_poll_seconds - v.poll_elapsed_seconds ? 'keep_pending' : 'poll_operation';
  if (v.operation_status === 'unknown') return 'query_operation';
  if (v.retention_expired) return 'reconcile_version';
  if (!v.feature_available) return 'wait_capability';
  if (v.attempts >= policy.recovery.max_attempts) return 'manual_review';
  return 'retry_same_operation';
}
