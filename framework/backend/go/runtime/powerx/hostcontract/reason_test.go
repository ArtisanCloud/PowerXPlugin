package hostcontract

import "testing"

func TestParseReasonCodeSupportsCoreEnvelopes(t *testing.T) {
	if got := ParseReasonCode([]byte(`{"error":{"reason_code":"CORE_FORBIDDEN"}}`), "fallback"); got != "CORE_FORBIDDEN" {
		t.Fatalf("nested reason = %q", got)
	}
	if got := ParseReasonCode([]byte(`{"error_code":"CORE_UNAUTHORIZED"}`), "fallback"); got != "CORE_UNAUTHORIZED" {
		t.Fatalf("root error code = %q", got)
	}
}
