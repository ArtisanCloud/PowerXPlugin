package customerfw

import (
	"errors"
	"net/http"
	"testing"
)

func TestErrorCodeAndHTTPStatus(t *testing.T) {
	err := WrapError(CodeCustomerTenantMismatch, "tenant mismatch", errors.New("inner"))
	if got := CodeOf(err); got != CodeCustomerTenantMismatch {
		t.Fatalf("expected mismatch code, got %s", got)
	}
	if got := HTTPStatusForCode(CodeCustomerTenantMismatch); got != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", got)
	}
}

func TestRedactSecret(t *testing.T) {
	if got := RedactSecret("abc"); got != "[redacted]" {
		t.Fatalf("expected redacted, got %q", got)
	}
	if got := RedactSecret(""); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}
