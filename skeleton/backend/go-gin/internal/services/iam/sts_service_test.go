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
	cfg := &config.Config{Context: &config.ContextConfig{HMACSecret: "secret", Issuer: "issuer", Audience: "powerx:api"}}
	svc := NewSTSService(cfg, nil, "com.powerx.test", "local.v1")
	tc := authx.TenantContext{TenantUUID: "0000-tenant", TenantID: 7, UserID: 42, MemberID: 4201, Roles: []string{"superadmin"}, Permissions: []string{"com.powerx.test:iam.role:read"}}
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
	if claims.TenantID != 7 {
		t.Fatalf("claims missing tenant numeric id, got %d", claims.TenantID)
	}
	if got := claims.Audience; len(got) != 1 || got[0] != "powerx:api" {
		t.Fatalf("unexpected sts audience: %v", got)
	}
	if claims.Subject != "client:com.powerx.test" {
		t.Fatalf("unexpected sts subject: %s", claims.Subject)
	}
	if claims.Scope != "access" {
		t.Fatalf("unexpected sts scope: %s", claims.Scope)
	}
	if claims.UserID != 0 {
		t.Fatalf("sts token must not carry user id, got %d", claims.UserID)
	}
	if claims.MemberID != 0 || claims.MemberIDAlias != 0 {
		t.Fatalf("sts token must not carry member id, got mid_n=%d member_id=%d", claims.MemberID, claims.MemberIDAlias)
	}
	if exp := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time); exp != defaultSTSTTL {
		t.Fatalf("expected ttl %v got %v", defaultSTSTTL, exp)
	}
}

func TestSTSServiceDefaultAudienceIsPowerXAPI(t *testing.T) {
	svc := NewSTSService(&config.Config{Context: &config.ContextConfig{HMACSecret: "secret", Issuer: "issuer"}}, nil, "com.powerx.test", "local.v1")
	token, err := svc.Mint(context.Background(), authx.TenantContext{TenantUUID: "tenant"})
	if err != nil {
		t.Fatalf("Mint returned error: %v", err)
	}
	claims := &authx.PowerXClaims{}
	if _, err := jwt.ParseWithClaims(token.AccessToken, claims, func(tok *jwt.Token) (any, error) {
		return []byte("secret"), nil
	}); err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}
	if got := claims.Audience; len(got) != 1 || got[0] != "powerx:api" {
		t.Fatalf("unexpected default sts audience: %v", got)
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
