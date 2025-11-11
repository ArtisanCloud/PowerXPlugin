import crypto from "node:crypto";

/**
 * Role: generate symmetric keys used to encrypt `.pxp` payloads and wrap them
 * using the Marketplace public key so reviewers can decrypt the artefact.
 */
export interface KeyEnvelope {
  algorithm: "AES-256-GCM";
  keyId: string;
  wrappedKey: string; // Base64 ciphertext
  iv: string; // Base64 IV used for artefact chunks
  authTag?: string; // Optional auth tag if encryption occurred up-front
  expiresAt: string; // RFC3339 timestamp
}

export interface EnvelopeOptions {
  marketplacePublicKeyPem: string;
  keyId: string;
  expiresAt?: Date;
}

const SYMMETRIC_KEY_BYTES = 32;
const GCM_IV_BYTES = 12;

export function generateSymmetricKey(): Buffer {
  return crypto.randomBytes(SYMMETRIC_KEY_BYTES);
}

function wrapKey(publicKeyPem: string, key: Buffer): Buffer {
  return crypto.publicEncrypt(
    {
      key: publicKeyPem,
      padding: crypto.constants.RSA_PKCS1_OAEP_PADDING,
      oaepHash: "sha256",
    },
    key,
  );
}

export function createKeyEnvelope(opts: EnvelopeOptions): KeyEnvelope {
  if (!opts.marketplacePublicKeyPem) {
    throw new Error("marketplace public key is required to wrap encryption key");
  }
  if (!opts.keyId) {
    throw new Error("keyId is required to identify the marketplace public key");
  }
  const symmetricKey = generateSymmetricKey();
  const iv = crypto.randomBytes(GCM_IV_BYTES);
  const wrapped = wrapKey(opts.marketplacePublicKeyPem, symmetricKey);
  const expiresAt = opts.expiresAt ?? new Date(Date.now() + 24 * 60 * 60 * 1000);
  return {
    algorithm: "AES-256-GCM",
    keyId: opts.keyId,
    wrappedKey: wrapped.toString("base64"),
    iv: iv.toString("base64"),
    expiresAt: expiresAt.toISOString(),
  };
}

export function unwrapKey(privateKeyPem: string, envelope: KeyEnvelope): Buffer {
  if (envelope.algorithm !== "AES-256-GCM") {
    throw new Error(`unsupported algorithm ${envelope.algorithm}`);
  }
  return crypto.privateDecrypt(
    {
      key: privateKeyPem,
      padding: crypto.constants.RSA_PKCS1_OAEP_PADDING,
      oaepHash: "sha256",
    },
    Buffer.from(envelope.wrappedKey, "base64"),
  );
}
