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

func TestParseReasonCodeWithCoreDiagnosticString(t *testing.T) {
	for _, raw := range []string{
		`{"reason_code":"MEDIA_UPLOAD_VALIDATION_FAILED","error":"invalid length"}`,
		`{"error_code":"MEDIA_UPLOAD_VALIDATION_FAILED","error":"invalid length"}`,
	} {
		if got := ParseReasonCode([]byte(raw), "fallback"); got != "MEDIA_UPLOAD_VALIDATION_FAILED" {
			t.Fatalf("reason=%q", got)
		}
	}
}

func TestRegistryPermissionReasonInDetails(t *testing.T) {
	got := ParseReasonCode([]byte(`{"code":403,"message":"registry.capability_forbidden","details":{"code":"registry.capability_forbidden","reason_code":"CAPABILITY_FORBIDDEN"}}`), "CAPABILITY_UPSTREAM_DEPENDENCY")
	if got != "CAPABILITY_FORBIDDEN" {
		t.Fatalf("REGISTRY_REASON_LOST: %s", got)
	}
}

func TestDiagnosticDetailsCannotEraseTopLevelReason(t *testing.T) {
	for _, raw := range []string{`{"reason_code":"RUNTIME_IDENTITY_FORBIDDEN","details":"diagnostic"}`, `{"error_code":"RUNTIME_IDENTITY_UNAVAILABLE","details":[]}`} {
		got := ParseReasonCode([]byte(raw), "CAPABILITY_UPSTREAM_DEPENDENCY")
		if got == "CAPABILITY_UPSTREAM_DEPENDENCY" {
			t.Fatal("TOP_LEVEL_REASON_LOST")
		}
	}
}
