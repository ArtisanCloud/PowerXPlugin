package iam

import (
	"context"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/golang-jwt/jwt/v5"
)

func TestSTSServiceMint(t *testing.T) {
	cfg := &config.Config{Context: &config.ContextConfig{HMACSecret: "secret", Issuer: "issuer", Audience: "aud"}}
	svc := NewSTSService(cfg, nil, "com.powerx.test", "local.v1")
	tc := authx.TenantContext{TenantUUID: "0000-tenant", UserID: 42, Roles: []string{"superadmin"}, Permissions: []string{"com.powerx.test:iam.role:read"}}
	token, err := svc.Mint(context.Background(), tc)
	if err != nil {
		t.Fatalf("Mint returned error: %v", err)
	}
	if token.PluginID != "com.powerx.test" {
		t.Fatalf("expected plugin id com.powerx.test, got %s", token.PluginID)
	}
	if token.PolicyVersion != "local.v1" {
		t.Fatalf("expected policy version local.v1, got %s", token.PolicyVersion)
	}
	if token.ExpiresIn != int64(defaultSTSTTL/time.Second) {
		t.Fatalf("unexpected expires_in: %d", token.ExpiresIn)
	}
	claims := &authx.PowerXClaims{}
	parsed, err := jwt.ParseWithClaims(token.AccessToken, claims, func(tok *jwt.Token) (any, error) {
		return []byte("secret"), nil
	})
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}
	if !parsed.Valid {
		t.Fatalf("token was not valid")
	}
	if claims.PluginID != "com.powerx.test" {
		t.Fatalf("claims missing plugin id, got %s", claims.PluginID)
	}
	if claims.TenantUUID.String() != "0000-tenant" {
		t.Fatalf("claims missing tenant uuid, got %s", claims.TenantUUID.String())
	}
	if exp := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time); exp != defaultSTSTTL {
		t.Fatalf("expected ttl %v got %v", defaultSTSTTL, exp)
	}
}

func TestSTSServiceMintValidatesTenant(t *testing.T) {
	svc := NewSTSService(&config.Config{}, nil, "", "")
	if _, err := svc.Mint(context.Background(), authx.TenantContext{}); err == nil {
		t.Fatalf("expected error when tenant uuid missing")
	}
	ctx := authx.TenantContext{TenantUUID: "tenant"}
	if _, err := svc.Mint(context.Background(), ctx); err != nil {
		t.Fatalf("expected mint to succeed, err=%v", err)
	}
}

func TestResolveSTSTTL(t *testing.T) {
	t.Setenv("PLUGIN_IAM_STS_TTL", "90s")
	if got := resolveSTSTTL(); got != 90*time.Second {
		t.Fatalf("expected 90s ttl, got %v", got)
	}
}
