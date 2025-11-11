package services

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"time"
)

// KeyEnvelope holds the encrypted symmetric key and metadata
type KeyEnvelope struct {
	Algorithm  string `json:"algorithm"` // Should be "AES-256-GCM"
	KeyID      string `json:"keyId"`
	WrappedKey string `json:"wrappedKey"` // Base64 ciphertext
	IV         string `json:"iv"`         // Base64 IV used for artefact chunks
	ExpiresAt  string `json:"expiresAt"`  // RFC3339 timestamp
}

// OfflineUploadPayload describes `.pxp` + integrity + signature metadata.
type OfflineUploadPayload struct {
	PublishID       string         `json:"publishId"`
	PluginID        string         `json:"pluginId"`
	Version         string         `json:"version"`
	IntegrityFile   []byte         `json:"integrityFile"`
	ManifestHash    string         `json:"manifestHash"`
	KeyEnvelope     KeyEnvelope    `json:"keyEnvelope"`
	Signature       []byte         `json:"signature"`
	AllowedTenants  []string       `json:"allowedTenants"`
	ReleaseMetadata map[string]any `json:"releaseMetadata,omitempty"`
}

// OfflineValidator performs validation and signature verification for offline packages.
type OfflineValidator struct {
	// Marketplace public key for signature verification
	marketplacePublicKey *rsa.PublicKey
}

// NewOfflineValidator creates a new OfflineValidator with marketplace public key
func NewOfflineValidator(marketplacePublicKeyPem string) (*OfflineValidator, error) {
	if marketplacePublicKeyPem == "" {
		return nil, errors.New("marketplace public key is required")
	}

	// Parse the public key
	block, _ := pem.Decode([]byte(marketplacePublicKeyPem))
	if block == nil {
		return nil, errors.New("failed to decode marketplace public key PEM")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, errors.New("failed to parse public key: " + err.Error())
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("public key is not an RSA key")
	}

	return &OfflineValidator{
		marketplacePublicKey: rsaPublicKey,
	}, nil
}

// Validate performs comprehensive validation including signature verification
func (v *OfflineValidator) Validate(payload OfflineUploadPayload) error {
	// Basic field validation
	if payload.PublishID == "" {
		return errors.New("publishId is required")
	}
	if payload.PluginID == "" {
		return errors.New("pluginId is required")
	}
	if len(payload.IntegrityFile) == 0 {
		return errors.New("integrity file missing")
	}

	// Validate manifest hash
	if err := validateManifestHash(payload.IntegrityFile, payload.ManifestHash); err != nil {
		return err
	}

	// Validate key envelope
	if err := v.validateKeyEnvelope(payload.KeyEnvelope); err != nil {
		return err
	}

	// Validate signature
	if err := v.verifySignature(payload); err != nil {
		return err
	}

	return nil
}

// ValidateMetadata ensures release metadata (channel, notes, rollout) is attached.
func (v *OfflineValidator) ValidateMetadata(metadata map[string]any) error {
	if metadata == nil {
		return errors.New("release metadata missing")
	}
	if _, ok := metadata["channel"]; !ok {
		return errors.New("release metadata missing channel")
	}
	return nil
}

// validateKeyEnvelope validates the key envelope structure and expiration
func (v *OfflineValidator) validateKeyEnvelope(envelope KeyEnvelope) error {
	if envelope.Algorithm == "" {
		return errors.New("key envelope algorithm is required")
	}
	if envelope.KeyID == "" {
		return errors.New("key envelope keyId is required")
	}
	if envelope.WrappedKey == "" {
		return errors.New("key envelope wrappedKey is required")
	}
	if envelope.IV == "" {
		return errors.New("key envelope iv is required")
	}
	if envelope.ExpiresAt == "" {
		return errors.New("key envelope expiresAt is required")
	}

	// Validate algorithm
	if envelope.Algorithm != "AES-256-GCM" {
		return errors.New("unsupported encryption algorithm: " + envelope.Algorithm)
	}

	// Validate expiration
	expiresAt, err := time.Parse(time.RFC3339, envelope.ExpiresAt)
	if err != nil {
		return errors.New("invalid expiration timestamp format")
	}
	if time.Now().After(expiresAt) {
		return errors.New("key envelope has expired")
	}

	return nil
}

// verifySignature verifies the digital signature of the integrity file
func (v *OfflineValidator) verifySignature(payload OfflineUploadPayload) error {
	if len(payload.Signature) == 0 {
		return errors.New("signature is required")
	}

	// Create hash of the integrity file
	hash := sha256.Sum256(payload.IntegrityFile)

	// Verify signature using the marketplace public key
	if err := rsa.VerifyPKCS1v15(
		v.marketplacePublicKey,
		crypto.SHA256,
		hash[:],
		payload.Signature,
	); err != nil {
		return errors.New("signature verification failed: " + err.Error())
	}

	return nil
}

// UnwrapKey decrypts the symmetric key using the marketplace private key
func (v *OfflineValidator) UnwrapKey(envelope KeyEnvelope, marketplacePrivateKeyPem string) ([]byte, error) {
	// Parse the private key
	block, _ := pem.Decode([]byte(marketplacePrivateKeyPem))
	if block == nil {
		return nil, errors.New("failed to decode marketplace private key PEM")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.New("failed to parse private key: " + err.Error())
	}

	rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not an RSA key")
	}

	// Decode the wrapped key
	wrappedKeyBytes, err := hex.DecodeString(envelope.WrappedKey)
	if err != nil {
		return nil, errors.New("failed to decode wrapped key: " + err.Error())
	}

	// Decrypt the symmetric key using RSA-OAEP
	symmetricKey, err := rsa.DecryptPKCS1v15(rand.Reader, rsaPrivateKey, wrappedKeyBytes)
	if err != nil {
		return nil, errors.New("failed to decrypt symmetric key: " + err.Error())
	}

	return symmetricKey, nil
}

// validateManifestHash validates the SHA-256 hash of the integrity file
func validateManifestHash(integrity []byte, manifestHash string) error {
	if manifestHash == "" {
		return errors.New("manifest hash missing")
	}
	sum := sha256.Sum256(integrity)
	if manifestHash != hex.EncodeToString(sum[:]) {
		return errors.New("manifest hash mismatch")
	}
	return nil
}
