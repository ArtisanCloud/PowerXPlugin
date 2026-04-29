package integration

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	pxlog "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
)

// SecretMaterial represents the issued secret result.
type SecretMaterial struct {
	Reference string
	Secret    string
	ExpiresAt *time.Time
	Metadata  map[string]any
}

// SecretProvider defines operations to interact with the host secrets manager.
type SecretProvider interface {
	Issue(ctx context.Context, tenantID, integrationType string) (*SecretMaterial, error)
	Revoke(ctx context.Context, tenantID, reference string) error
}

// RandomSecretProvider issues random secrets for development/testing.
type RandomSecretProvider struct {
	logger *pxlog.Entry
}

// NewRandomSecretProvider constructs a default provider.
func NewRandomSecretProvider(logger *pxlog.Entry) *RandomSecretProvider {
	if logger == nil {
		logger = pxlog.WithComponent("integration.secret_provider.random")
	}
	return &RandomSecretProvider{logger: logger}
}

// Issue returns a random secret payload with a generated reference.
func (p *RandomSecretProvider) Issue(ctx context.Context, tenantID, integrationType string) (*SecretMaterial, error) {
	ref := fmt.Sprintf("secret:%s:%s:%d", tenantID, integrationType, time.Now().UnixNano())
	secret, err := generateRandomSecret(48)
	if err != nil {
		return nil, err
	}
	pxlog.DebugCtx(pxlog.WithLogFields(ctx, map[string]interface{}{
		"module":           "integration",
		"biz_scene":        "secret_provider_issue",
		"biz_domain":       "integration",
		"component":        "integration.secret_provider.random",
		"tenant_uuid":      tenantID,
		"integration_type": integrationType,
	}), "issued random secret material")
	return &SecretMaterial{
		Reference: ref,
		Secret:    secret,
	}, nil
}

// Revoke logs the revocation request—it is a no-op for local provider.
func (p *RandomSecretProvider) Revoke(ctx context.Context, tenantID, reference string) error {
	pxlog.DebugCtx(pxlog.WithLogFields(ctx, map[string]interface{}{
		"module":      "integration",
		"biz_scene":   "secret_provider_revoke",
		"biz_domain":  "integration",
		"component":   "integration.secret_provider.random",
		"tenant_uuid": tenantID,
		"reference":   reference,
	}), "revoked secret reference")
	return nil
}

func generateRandomSecret(length int) (string, error) {
	if length <= 0 {
		length = 32
	}
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
