package iamcontext

import (
	"encoding/base64"
	"fmt"
	"testing"

	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
)

func TestIdentityContextResolver_ResolveIdentity(t *testing.T) {
	resolver := IdentityContextResolver{}
	token := buildUnsignedJWT(`{
		"tid":"11111111-1111-1111-1111-111111111111",
		"sub":"user-42",
		"member_id":"member-7",
		"roles":["admin","ops"],
		"permissions":["iam.tenant.read","iam.member.write"],
		"policy_version":"2026.04",
		"trace_id":"trace-001"
	}`)

	identity, err := resolver.ResolveIdentity(nil, "Bearer "+token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if identity.TenantUUID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("tenant mismatch, got=%s", identity.TenantUUID)
	}
	if identity.UserID != "user-42" {
		t.Fatalf("user mismatch, got=%s", identity.UserID)
	}
	if identity.MemberID != "member-7" {
		t.Fatalf("member mismatch, got=%s", identity.MemberID)
	}
	if len(identity.Roles) != 2 || identity.Roles[0] != "admin" || identity.Roles[1] != "ops" {
		t.Fatalf("roles mismatch: %#v", identity.Roles)
	}
	if len(identity.Permissions) != 2 || identity.Permissions[0] != "iam.tenant.read" || identity.Permissions[1] != "iam.member.write" {
		t.Fatalf("permissions mismatch: %#v", identity.Permissions)
	}
	if identity.PolicyVer != "2026.04" {
		t.Fatalf("policy version mismatch, got=%s", identity.PolicyVer)
	}
	if identity.TraceID != "trace-001" {
		t.Fatalf("trace mismatch, got=%s", identity.TraceID)
	}
}

func TestIdentityContextResolver_InvalidToken(t *testing.T) {
	resolver := IdentityContextResolver{}
	_, err := resolver.ResolveIdentity(nil, "not-a-jwt")
	if err == nil {
		t.Fatalf("expected unauthorized error")
	}
	if !iamerrors.IsCode(err, iamerrors.CodeUnauthorized) {
		t.Fatalf("unexpected error code: %s", iamerrors.CodeOf(err))
	}
}

func TestIdentityContextResolver_MissingTenant(t *testing.T) {
	resolver := IdentityContextResolver{}
	token := buildUnsignedJWT(`{"sub":"user-1"}`)
	_, err := resolver.ResolveIdentity(nil, token)
	if err == nil {
		t.Fatalf("expected unauthorized error")
	}
	if !iamerrors.IsCode(err, iamerrors.CodeUnauthorized) {
		t.Fatalf("unexpected error code: %s", iamerrors.CodeOf(err))
	}
}

func buildUnsignedJWT(payload string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	body := base64.RawURLEncoding.EncodeToString([]byte(payload))
	return fmt.Sprintf("%s.%s.", header, body)
}
