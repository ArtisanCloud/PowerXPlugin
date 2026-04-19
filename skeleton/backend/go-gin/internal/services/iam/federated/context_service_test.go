package federated

import (
	"errors"
	"testing"
)

func TestNormalizeUnavailableErrorDelegatedAndStandaloneConsistency(t *testing.T) {
	svc := NewContextService()
	delegated := svc.NormalizeUnavailableError("delegated", errors.New("host down"))
	standalone := svc.NormalizeUnavailableError("standalone", errors.New("local down"))
	if delegated.Code != standalone.Code {
		t.Fatalf("code mismatch delegated=%s standalone=%s", delegated.Code, standalone.Code)
	}
	if delegated.Message != "登录失败，请稍后重试" || standalone.Message != "登录失败，请稍后重试" {
		t.Fatalf("message mismatch delegated=%+v standalone=%+v", delegated, standalone)
	}
}

func TestNormalizeContextDelegatedHostAuthority(t *testing.T) {
	svc := NewContextService()
	ctx := svc.NormalizeContext("delegated", IdentityContext{TenantUUID: "tenant-a", Provider: "wecom"})
	if ctx.PolicySource != "host-authoritative" {
		t.Fatalf("policy_source=%s, want host-authoritative", ctx.PolicySource)
	}
}
