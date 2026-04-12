package contracts

import (
	"errors"
	"strings"
	"testing"
)

func TestHasCode(t *testing.T) {
	err := NewError(ErrorCodeRiskReplay, "request rejected")
	if !HasCode(err, ErrorCodeRiskReplay) {
		t.Fatalf("HasCode() = false, want true")
	}
	if HasCode(err, ErrorCodeRiskSignature) {
		t.Fatalf("HasCode() = true for wrong code")
	}
}

func TestWrapErrorPreservesCause(t *testing.T) {
	root := errors.New("db timeout")
	err := WrapError(ErrorCodeInvalidChallenge, "challenge failed", root)
	if !errors.Is(err, root) {
		t.Fatalf("errors.Is() = false, want true")
	}
	if !HasCode(err, ErrorCodeInvalidChallenge) {
		t.Fatalf("HasCode() = false, want true")
	}
}

func TestErrorMessageCarriesCodeForFrontendRouting(t *testing.T) {
	err := NewError(ErrorCodeRiskTenantBoundary, "request rejected")
	msg := err.Error()
	if !strings.Contains(msg, string(ErrorCodeRiskTenantBoundary)) {
		t.Fatalf("error message = %q, want contains code", msg)
	}
	if strings.Contains(strings.ToLower(msg), "stack") {
		t.Fatalf("error message should stay generic, got %q", msg)
	}
}
