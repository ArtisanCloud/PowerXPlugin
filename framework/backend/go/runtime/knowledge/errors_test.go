package knowledge

import (
	"net/http"
	"testing"
)

func TestErrorCodeAndHTTPStatus(t *testing.T) {
	err := NewError(CodeTenantMismatch, "tenant mismatch")
	if CodeOf(err) != CodeTenantMismatch {
		t.Fatalf("unexpected code: %s", CodeOf(err))
	}
	if got := HTTPStatusForCode(CodeTenantMismatch); got != http.StatusForbidden {
		t.Fatalf("status = %d", got)
	}
	if got := HTTPStatusForCode(CodeConflict); got != http.StatusConflict {
		t.Fatalf("conflict status = %d", got)
	}
}

func TestRedaction(t *testing.T) {
	if got := RedactString("Authorization: Bearer abc.def"); got == "Authorization: Bearer abc.def" {
		t.Fatalf("expected bearer redaction")
	}
	fields := RedactMap(map[string]any{"api_token": "abc", "nested": map[string]any{"password": "pw"}, "safe": "ok"})
	if fields["api_token"] != "[redacted]" {
		t.Fatalf("expected token redacted: %+v", fields)
	}
	if fields["safe"] != "ok" {
		t.Fatalf("expected safe field retained")
	}
}
