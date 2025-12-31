package unit_test

import (
	"context"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	customersvc "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/customer"
	"github.com/golang-jwt/jwt/v5"
)

func TestCustomerAuthenticator_LocalJWT_Success(t *testing.T) {
	cfg := &config.Config{
		Server: &config.ServerConfig{DevMode: true},
		CustomerAuth: &config.CustomerAuthConfig{
			Mode:      "local",
			JWTSecret: "test-secret",
		},
	}
	auth := customersvc.NewAuthenticatorFactory(cfg, nil).Build()

	tenantUUID := "00000000-0000-0000-0000-000000000001"
	customerUUID := "00000000-0000-0000-0000-000000000002"
	token := signCustomerJWT(t, cfg.CustomerAuth.JWTSecret, tenantUUID, customerUUID)

	cc, err := auth.Authenticate(context.Background(), tenantUUID, token)
	if err != nil {
		t.Fatalf("Authenticate() err = %v", err)
	}
	if cc == nil || !cc.Authenticated {
		t.Fatalf("expected authenticated customer context, got %#v", cc)
	}
	if cc.TenantUUID != tenantUUID {
		t.Fatalf("tenant mismatch: got %s", cc.TenantUUID)
	}
	if cc.CustomerUUID != customerUUID {
		t.Fatalf("customer mismatch: got %s", cc.CustomerUUID)
	}
}

func TestCustomerAuthenticator_LocalJWT_InvalidToken(t *testing.T) {
	cfg := &config.Config{
		Server: &config.ServerConfig{DevMode: true},
		CustomerAuth: &config.CustomerAuthConfig{
			Mode:      "local",
			JWTSecret: "test-secret",
		},
	}
	auth := customersvc.NewAuthenticatorFactory(cfg, nil).Build()

	_, err := auth.Authenticate(context.Background(), "00000000-0000-0000-0000-000000000001", "not-a-jwt")
	if err == nil {
		t.Fatalf("expected error for invalid token")
	}
}

func TestCustomerAuthenticator_LocalJWT_NoSecret(t *testing.T) {
	cfg := &config.Config{
		Server: &config.ServerConfig{DevMode: true},
		CustomerAuth: &config.CustomerAuthConfig{
			Mode:      "local",
			JWTSecret: "",
		},
	}
	auth := customersvc.NewAuthenticatorFactory(cfg, nil).Build()
	_, err := auth.Authenticate(context.Background(), "00000000-0000-0000-0000-000000000001", "anything")
	if err == nil {
		t.Fatalf("expected error when secret is missing")
	}
}

func signCustomerJWT(t *testing.T, secret, tenantUUID, customerUUID string) string {
	t.Helper()

	claims := jwt.MapClaims{
		"tenant_uuid":   tenantUUID,
		"customer_uuid": customerUUID,
		"iat":           time.Now().Unix(),
		"exp":           time.Now().Add(1 * time.Hour).Unix(),
		"sub":           customerUUID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	out, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}
	return out
}
